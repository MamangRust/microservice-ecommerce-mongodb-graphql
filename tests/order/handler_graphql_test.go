package order_test

import (
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type OrderGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler http.Handler
	orderID int
	userID  int
	merchID int
	prodID  int
}

func (s *OrderGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.SalesDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupMerchantService()
	s.SetupCategoryService()
	s.SetupProductService()
	s.SetupOrderItemService()
	s.SetupShippingAddressService()
	s.SetupTransactionService()
	s.SetupOrderService()

	// Seed dependencies
	userID := s.SeedUser(s.Ctx)
	catID := s.SeedCategory(s.Ctx)
	merchID := s.SeedMerchant(s.Ctx, userID)
	prodID := s.SeedProduct(s.Ctx, merchID, catID)

	s.userID = userID
	s.merchID = merchID
	s.prodID = prodID

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *OrderGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *OrderGraphqlTestSuite) TestOrderGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateOrder($input: CreateOrderInput!) {
			createOrder(input: $input) {
				status
				message
				data { id merchant_id user_id total_price }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_id": s.merchID,
			"user_id":     s.userID,
			"total_price": 1000,
			"items": []interface{}{
				map[string]interface{}{"product_id": s.prodID, "quantity": 1, "price": 1000},
			},
			"shipping": map[string]interface{}{
				"alamat":          "Test Alamat",
				"provinsi":        "Test Provinsi",
				"kota":            "Test Kota",
				"courier":         "Test Courier",
				"shipping_method": "Test Method",
				"shipping_cost":   100,
				"negara":          "Test Negara",
			},
		},
	})
	created := data["createOrder"].(map[string]interface{})["data"].(map[string]interface{})
	s.orderID = int(created["id"].(float64))

	// 2. FindById
	data = s.exec(`
		query FindOrderById($input: FindByIdOrderInput!) {
			findOrderById(input: $input) { status data { id merchant_id user_id total_price } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	found := data["findOrderById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.orderID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllOrders($input: FindAllOrderInput!) {
			findAllOrders(input: $input) {
				status
				data { id user_id total_price }
				pagination { current_page page_size total_records }
			}
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllOrders"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveOrders($input: FindAllOrderInput!) {
			findActiveOrders(input: $input) { status data { id user_id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveOrders"].(map[string]interface{})["data"])

	// 5. Update — the update input needs the line-item id and shipping id, which
	// are not part of OrderResponse, so resolve them through the sibling queries.
	itemData := s.exec(`
		query FindOrderItemsByOrder($input: FindByIdOrderItemInput!) {
			findOrderItemsByOrder(input: $input) { status data { id order_id product_id quantity price } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	items := itemData["findOrderItemsByOrder"].(map[string]interface{})["data"].([]interface{})
	s.Require().NotEmpty(items, "order must have at least one item")
	orderItemID := int(items[0].(map[string]interface{})["id"].(float64))

	shipData := s.exec(`
		query FindShippingByOrder($input: FindByIdShippingRequest) {
			findShippingByOrder(input: $input) { status data { id order_id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	shippingID := int(shipData["findShippingByOrder"].(map[string]interface{})["data"].(map[string]interface{})["id"].(float64))

	data = s.exec(`
		mutation UpdateOrder($input: UpdateOrderInput!) {
			updateOrder(input: $input) { status message data { id total_price } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"order_id":    s.orderID,
			"user_id":     s.userID,
			"total_price": 1500,
			"items": []interface{}{
				map[string]interface{}{"order_item_id": orderItemID, "product_id": s.prodID, "quantity": 1, "price": 1500},
			},
			"shipping": map[string]interface{}{
				"shipping_id":     shippingID,
				"alamat":          "Updated Alamat",
				"provinsi":        "Updated Provinsi",
				"kota":            "Updated Kota",
				"courier":         "Updated Courier",
				"shipping_method": "Updated Method",
				"shipping_cost":   200,
				"negara":          "Updated Negara",
			},
		},
	})
	s.Equal("success", data["updateOrder"].(map[string]interface{})["status"])

	// 6. Trash
	data = s.exec(`
		mutation TrashOrder($input: FindByIdOrderInput!) {
			trashOrder(input: $input) { status data { id deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	s.Equal("success", data["trashOrder"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedOrders($input: FindAllOrderInput!) {
			findTrashedOrders(input: $input) { status data { id user_id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedOrders"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreOrder($input: FindByIdOrderInput!) {
			restoreOrder(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	s.Equal("success", data["restoreOrder"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashOrder($input: FindByIdOrderInput!) {
			trashOrder(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	data = s.exec(`
		mutation DeleteOrderPermanent($input: FindByIdOrderInput!) {
			deleteOrderPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.orderID}})
	s.Equal("success", data["deleteOrderPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllOrders { status } }
	`, nil)
	s.Equal("success", data["restoreAllOrders"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllOrdersPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllOrdersPermanent"].(map[string]interface{})["status"])
}

func TestOrderGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(OrderGraphqlTestSuite))
}
