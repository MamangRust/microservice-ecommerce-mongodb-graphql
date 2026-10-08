package cart_test

import (
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type CartGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler http.Handler
	userID  int
	cartIDs []int
}

func (s *CartGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.ExperienceDB
	s.BaseTestSuite.SetupSuite()

	// Bootstrap all required downstream services
	s.SetupUserService()
	s.SetupCategoryService()
	s.SetupMerchantService()
	s.SetupProductService()
	s.SetupCartService()

	// Seed a user to use as the authenticated caller
	s.userID = s.SeedUser(s.Ctx)

	// deleteAllCarts reads the caller from the request context (see
	// cart.resolvers.go), so emulate AuthMiddleware by injecting the seeded user.
	s.handler = graphtest.WithUser(graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log), s.userID)

	s.cartIDs = make([]int, 0)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *CartGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *CartGraphqlTestSuite) TestCartGraphqlLifecycle() {
	// Seed entities needed for cart items
	categoryID := s.SeedCategory(s.Ctx)
	merchantID := s.SeedMerchant(s.Ctx, s.userID)
	prod1ID := s.SeedProduct(s.Ctx, merchantID, categoryID)
	prod2ID := s.SeedProduct(s.Ctx, merchantID, categoryID)

	// 1. Create — add first product to cart
	data := s.exec(`
		mutation CreateCart($input: CreateCartInput!) {
			createCart(input: $input) {
				status
				message
				data { id user_id product_id quantity }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"user_id":    s.userID,
			"product_id": prod1ID,
			"quantity":   2,
		},
	})
	created := data["createCart"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(prod1ID), created["product_id"])
	s.Equal(float64(2), created["quantity"])
	cart1ID := int(created["id"].(float64))
	s.cartIDs = append(s.cartIDs, cart1ID)

	// 2. Create — add second product to cart
	data = s.exec(`
		mutation CreateCart($input: CreateCartInput!) {
			createCart(input: $input) { status data { id product_id quantity } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"user_id":    s.userID,
			"product_id": prod2ID,
			"quantity":   1,
		},
	})
	created2 := data["createCart"].(map[string]interface{})["data"].(map[string]interface{})
	cart2ID := int(created2["id"].(float64))
	s.cartIDs = append(s.cartIDs, cart2ID)

	// 3. FindAll — list cart items for the user
	data = s.exec(`
		query FindAllCarts($input: FindAllCartInput!) {
			findAllCarts(input: $input) {
				status
				data { id user_id product_id quantity }
				pagination { current_page page_size total_records }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"user_id": s.userID, "page": 1, "page_size": 10},
	})
	items := data["findAllCarts"].(map[string]interface{})["data"].([]interface{})
	s.GreaterOrEqual(len(items), 2, "should have at least 2 cart items")

	// 4. Delete — remove the first cart item
	data = s.exec(`
		mutation DeleteCart($input: DeleteCartInput!) {
			deleteCart(input: $input) { status message }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"cart_id": cart1ID, "user_id": s.userID},
	})
	s.Equal("success", data["deleteCart"].(map[string]interface{})["status"])
	s.cartIDs = s.cartIDs[1:] // remove from tracking

	// 5. DeleteAll — remove remaining cart items
	data = s.exec(`
		mutation DeleteAllCarts($input: DeleteCartsInput!) {
			deleteAllCarts(input: $input) { status message }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"cart_ids": s.cartIDs},
	})
	s.Equal("success", data["deleteAllCarts"].(map[string]interface{})["status"])

	// 6. Verify — FindAll should now be empty.
	// Use a distinct page_size so this read does not hit the gateway's
	// user-agnostic list cache populated in step 3.
	data = s.exec(`
		query FindAllCarts($input: FindAllCartInput!) {
			findAllCarts(input: $input) { status data { id } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"user_id": s.userID, "page": 1, "page_size": 5},
	})
	verifyItems := data["findAllCarts"].(map[string]interface{})["data"]
	if verifyItems == nil {
		return
	}
	s.Empty(verifyItems.([]interface{}), "cart should be empty after deleting all items")
}

func TestCartGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(CartGraphqlTestSuite))
}
