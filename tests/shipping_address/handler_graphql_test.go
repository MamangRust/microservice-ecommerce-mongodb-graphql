package shipping_address_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type ShippingAddressGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler           http.Handler
	orderID           int
	shippingAddressID int
}

func (s *ShippingAddressGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.ExperienceDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupCategoryService()
	s.SetupMerchantService()
	s.SetupProductService()
	s.SetupOrderItemService()
	s.SetupShippingAddressService()
	s.SetupOrderService()

	// Seed dependencies (the order creates its shipping address).
	ctx := context.Background()
	userID := s.SeedUser(ctx)
	catID := s.SeedCategory(ctx)
	merchID := s.SeedMerchant(ctx, userID)
	prodID := s.SeedProduct(ctx, merchID, catID)
	s.orderID = s.SeedOrder(ctx, userID, merchID, prodID)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *ShippingAddressGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *ShippingAddressGraphqlTestSuite) TestShippingAddressGraphqlLifecycle() {
	// 1. FindAll
	data := s.exec(`
		query FindAllShipping($input: FindAllShippingRequest!) {
			findAllShipping(input: $input) {
				status
				data { id order_id }
				pagination { current_page page_size total_records }
			}
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllShipping"].(map[string]interface{})["data"])

	// 2. FindByOrder
	s.Require().NotZero(s.orderID)
	data = s.exec(`
		query FindShippingByOrder($input: FindByIdShippingRequest!) {
			findShippingByOrder(input: $input) { status data { id order_id alamat } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	byOrder := data["findShippingByOrder"].(map[string]interface{})["data"].(map[string]interface{})
	s.shippingAddressID = int(byOrder["id"].(float64))
	s.Require().NotZero(s.shippingAddressID)

	// 3. FindById
	data = s.exec(`
		query FindShippingById($input: FindByIdShippingRequest!) {
			findShippingById(input: $input) { status data { id order_id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.shippingAddressID}})
	found := data["findShippingById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.shippingAddressID), found["id"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveShipping($input: FindAllShippingRequest!) {
			findActiveShipping(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveShipping"].(map[string]interface{})["data"])

	// 5. Trash
	data = s.exec(`
		mutation TrashedShipping($input: FindByIdShippingRequest!) {
			trashedShipping(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.shippingAddressID}})
	s.Equal("success", data["trashedShipping"].(map[string]interface{})["status"])

	// 6. FindByTrashed
	data = s.exec(`
		query FindTrashedShipping($input: FindAllShippingRequest!) {
			findTrashedShipping(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedShipping"].(map[string]interface{})["data"])

	// 7. Restore
	data = s.exec(`
		mutation RestoreShipping($input: FindByIdShippingRequest!) {
			restoreShipping(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.shippingAddressID}})
	s.Equal("success", data["restoreShipping"].(map[string]interface{})["status"])

	// 8. DeletePermanent
	s.exec(`
		mutation TrashedShipping($input: FindByIdShippingRequest!) {
			trashedShipping(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.shippingAddressID}})
	data = s.exec(`
		mutation DeleteShippingPermanent($input: FindByIdShippingRequest!) {
			deleteShippingPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.shippingAddressID}})
	s.Equal("success", data["deleteShippingPermanent"].(map[string]interface{})["status"])

	// 9. RestoreAll
	data = s.exec(`mutation { restoreAllShipping { status } }`, nil)
	s.Equal("success", data["restoreAllShipping"].(map[string]interface{})["status"])

	// 10. DeleteAll
	data = s.exec(`mutation { deleteAllShippingPermanent { status } }`, nil)
	s.Equal("success", data["deleteAllShippingPermanent"].(map[string]interface{})["status"])
}

func TestShippingAddressGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(ShippingAddressGraphqlTestSuite))
}
