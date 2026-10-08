package order_item_test

import (
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type OrderItemGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler http.Handler
	orderID int
}

func (s *OrderItemGraphqlTestSuite) SetupSuite() {
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
	userID := s.SeedUser(s.Ctx)
	catID := s.SeedCategory(s.Ctx)
	merchID := s.SeedMerchant(s.Ctx, userID)
	prodID := s.SeedProduct(s.Ctx, merchID, catID)
	s.orderID = s.SeedOrder(s.Ctx, userID, merchID, prodID)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *OrderItemGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *OrderItemGraphqlTestSuite) TestOrderItemGraphqlLifecycle() {
	// 1. FindAll
	data := s.exec(`
		query FindAllOrderItems($input: FindAllOrderItemInput!) {
			findAllOrderItems(input: $input) {
				status
				data { id order_id product_id quantity price }
				pagination { current_page page_size total_records }
			}
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllOrderItems"].(map[string]interface{})["data"])

	// 2. FindOrderItemsByOrder
	s.Require().NotZero(s.orderID)
	data = s.exec(`
		query FindOrderItemsByOrder($input: FindByIdOrderItemInput!) {
			findOrderItemsByOrder(input: $input) { status data { id order_id product_id quantity price } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	items := data["findOrderItemsByOrder"].(map[string]interface{})["data"].([]interface{})
	s.Require().NotEmpty(items, "seeded order must have at least one item")

	// 3. FindByActive
	data = s.exec(`
		query FindActiveOrderItems($input: FindAllOrderItemInput!) {
			findActiveOrderItems(input: $input) { status data { id order_id product_id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveOrderItems"].(map[string]interface{})["data"])

	// 4. FindByTrashed
	data = s.exec(`
		query FindTrashedOrderItems($input: FindAllOrderItemInput!) {
			findTrashedOrderItems(input: $input) { status data { id order_id product_id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotNil(data["findTrashedOrderItems"].(map[string]interface{})["data"])
}

func TestOrderItemGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(OrderItemGraphqlTestSuite))
}
