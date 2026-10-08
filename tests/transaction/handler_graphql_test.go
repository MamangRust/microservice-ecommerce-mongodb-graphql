package transaction_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbtransaction "github.com/MamangRust/microservice-ecommerce-grpc-pb/transaction"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type TransactionGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler       http.Handler
	transactionID int
	userID        int
	merchID       int
	orderID       int
}

func (s *TransactionGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.SalesDB
	s.BaseTestSuite.SetupSuite()
	s.SetupRoleService()
	s.SetupUserService()
	s.SetupCategoryService()
	s.SetupMerchantService()
	s.SetupProductService()
	s.SetupShippingAddressService()
	s.SetupOrderItemService()
	s.SetupOrderService()
	s.SetupTransactionService()

	// Seed dependencies
	ctx := context.Background()
	userID := s.SeedUser(ctx)
	merchID := s.SeedMerchant(ctx, userID)
	catID := s.SeedCategory(ctx)
	prodID := s.SeedProduct(ctx, merchID, catID)
	orderID := s.SeedOrder(ctx, userID, merchID, prodID)
	s.SeedShippingAddress(ctx, orderID)
	s.SeedOrderItem(ctx, orderID, prodID)

	s.userID = userID
	s.merchID = merchID
	s.orderID = orderID

	// The GraphQL createTransaction resolver does not forward a user_id to the
	// transaction service, which requires a valid user, so the transaction is
	// seeded through the gRPC command client (same connection the gateway uses).
	// The remaining lifecycle is exercised through GraphQL.
	seedRes, err := pbtransaction.NewTransactionCommandServiceClient(s.Conns["transaction"]).Create(ctx, &pbtransaction.CreateTransactionRequest{
		UserId:        int32(userID),
		MerchantId:    int32(merchID),
		OrderId:       int32(orderID),
		PaymentMethod: "credit_card",
		Amount:        100000,
	})
	s.Require().NoError(err)
	s.transactionID = int(seedRes.Data.Id)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *TransactionGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *TransactionGraphqlTestSuite) TestTransactionGraphqlLifecycle() {
	s.Require().NotZero(s.transactionID)

	// 1. FindById
	data := s.exec(`
		query FindTransactionById($input: FindByIdTransactionRequest!) {
			findTransactionById(input: $input) { status data { id order_id merchant_id payment_method amount payment_status } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.transactionID}})
	found := data["findTransactionById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.transactionID), found["id"])

	// 2. FindAll
	data = s.exec(`
		query FindAllTransaction($input: FindAllTransactionRequest!) {
			findAllTransaction(input: $input) {
				status
				data { id }
				pagination { current_page page_size total_records }
			}
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllTransaction"].(map[string]interface{})["data"])

	// 3. FindByMerchant
	data = s.exec(`
		query FindTransactionByMerchant($input: FindAllTransactionMerchantRequest!) {
			findTransactionByMerchant(input: $input) { status data { id merchant_id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"merchant_id": s.merchID, "page": 1, "page_size": 10}})
	s.NotEmpty(data["findTransactionByMerchant"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindByActive($input: FindAllTransactionRequest!) {
			findByActive(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findByActive"].(map[string]interface{})["data"])

	// 5. Update
	data = s.exec(`
		mutation UpdateTransaction($input: UpdateTransactionRequest!) {
			updateTransaction(input: $input) { status data { id payment_method } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"transaction_id": s.transactionID,
			"order_id":       s.orderID,
			"merchant_id":    s.merchID,
			"payment_method": "credit_card",
			"amount":         100000,
		},
	})
	s.Equal("success", data["updateTransaction"].(map[string]interface{})["status"])

	// 6. Trash
	data = s.exec(`
		mutation TrashedTransaction($input: FindByIdTransactionRequest!) {
			trashedTransaction(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.transactionID}})
	s.Equal("success", data["trashedTransaction"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindByTrashed($input: FindAllTransactionRequest!) {
			findByTrashed(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findByTrashed"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreTransaction($input: FindByIdTransactionRequest!) {
			restoreTransaction(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.transactionID}})
	s.Equal("success", data["restoreTransaction"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashedTransaction($input: FindByIdTransactionRequest!) {
			trashedTransaction(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.transactionID}})
	data = s.exec(`
		mutation DeleteTransactionPermanent($input: FindByIdTransactionRequest!) {
			deleteTransactionPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.transactionID}})
	s.Equal("success", data["deleteTransactionPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`mutation { restoreAllTransaction { status } }`, nil)
	s.Equal("success", data["restoreAllTransaction"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`mutation { deleteAllTransactionPermanent { status } }`, nil)
	s.Equal("success", data["deleteAllTransactionPermanent"].(map[string]interface{})["status"])
}

func TestTransactionGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransactionGraphqlTestSuite))
}
