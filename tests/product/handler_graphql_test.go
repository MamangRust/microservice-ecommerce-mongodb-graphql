package product_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type ProductGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler      http.Handler
	productID    int
	merchantID   int
	categoryID   int
	categoryName string
}

func (s *ProductGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.CatalogDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupCategoryService()
	s.SetupMerchantService()
	s.SetupProductService()

	// Seed dependencies
	ctx := context.Background()
	userID := s.SeedUser(ctx)
	s.categoryID = s.SeedCategory(ctx)
	s.merchantID = s.SeedMerchant(ctx, userID)
	s.categoryName = "Seed Category"

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *ProductGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *ProductGraphqlTestSuite) TestProductGraphqlLifecycle() {
	// 1. Create (the product service requires an image path, so upload one)
	createResp, err := graphtest.ExecuteGraphQLMultipart(s.handler, `
		mutation CreateProduct($input: CreateProductInput!) {
			createProduct(input: $input) {
				status
				message
				data { id name merchantId categoryId }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchantId":   s.merchantID,
			"categoryId":   s.categoryID,
			"name":         "Test Product",
			"description":  "Test Description",
			"price":        1000,
			"countInStock": 10,
			"brand":        "Test Brand",
			"weight":       1,
			"imageProduct": nil,
		},
	}, map[string]graphtest.UploadFile{
		"variables.input.imageProduct": {
			Filename:    "product.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy image content"),
		},
	}, "")
	s.Require().NoError(err)
	s.Require().Empty(createResp.Errors, "graphql errors: %+v", createResp.Errors)

	created := createResp.Data["createProduct"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Test Product", created["name"])
	s.productID = int(created["id"].(float64))

	// 2. FindById
	data := s.exec(`
		query FindProductById($input: FindByIdProductInput!) {
			findProductById(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	found := data["findProductById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.productID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllProducts($input: FindAllProductInput!) {
			findAllProducts(input: $input) { status data { id name } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "pageSize": 10}})
	s.NotEmpty(data["findAllProducts"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveProducts($input: FindAllProductInput!) {
			findActiveProducts(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "pageSize": 10}})
	s.NotEmpty(data["findActiveProducts"].(map[string]interface{})["data"])

	// 5. FindByMerchant
	data = s.exec(`
		query FindProductsByMerchant($input: FindAllProductMerchantInput!) {
			findProductsByMerchant(input: $input) { status data { id name merchantId } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"merchantId": s.merchantID, "page": 1, "pageSize": 10, "minPrice": 1, "maxPrice": 1000000}})
	s.NotEmpty(data["findProductsByMerchant"].(map[string]interface{})["data"])

	// 6. FindByCategory
	data = s.exec(`
		query FindProductsByCategory($input: FindAllProductCategoryInput!) {
			findProductsByCategory(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"categoryName": s.categoryName, "page": 1, "pageSize": 10, "minPrice": 1, "maxPrice": 1000000}})
	s.NotEmpty(data["findProductsByCategory"].(map[string]interface{})["data"])

	// 7. Update
	updateResp, err := graphtest.ExecuteGraphQLMultipart(s.handler, `
		mutation UpdateProduct($input: UpdateProductInput!) {
			updateProduct(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"productId":    s.productID,
			"merchantId":   s.merchantID,
			"categoryId":   s.categoryID,
			"name":         "Updated Product",
			"description":  "Updated Description",
			"price":        2000,
			"countInStock": 20,
			"brand":        "Updated Brand",
			"weight":       2,
			"imageProduct": nil,
		},
	}, map[string]graphtest.UploadFile{
		"variables.input.imageProduct": {
			Filename:    "updated.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy image content"),
		},
	}, "")
	s.Require().NoError(err)
	s.Require().Empty(updateResp.Errors, "graphql errors: %+v", updateResp.Errors)
	updated := updateResp.Data["updateProduct"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Product", updated["name"])

	// 8. Trash
	data = s.exec(`
		mutation TrashedProduct($input: FindByIdProductInput!) {
			trashedProduct(input: $input) { status data { id name deletedAt } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	s.Equal("success", data["trashedProduct"].(map[string]interface{})["status"])

	// 9. FindByTrashed
	data = s.exec(`
		query FindTrashedProducts($input: FindAllProductInput!) {
			findTrashedProducts(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "pageSize": 10}})
	s.NotEmpty(data["findTrashedProducts"].(map[string]interface{})["data"])

	// 10. Restore
	data = s.exec(`
		mutation RestoreProduct($input: FindByIdProductInput!) {
			restoreProduct(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	s.Equal("success", data["restoreProduct"].(map[string]interface{})["status"])

	// 11. DeletePermanent (trash first)
	s.exec(`
		mutation TrashedProduct($input: FindByIdProductInput!) {
			trashedProduct(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	data = s.exec(`
		mutation DeleteProductPermanent($input: FindByIdProductInput!) {
			deleteProductPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.productID}})
	s.Equal("success", data["deleteProductPermanent"].(map[string]interface{})["status"])

	// 12. RestoreAll
	data = s.exec(`
		mutation { restoreAllProducts { status } }
	`, nil)
	s.Equal("success", data["restoreAllProducts"].(map[string]interface{})["status"])

	// 13. DeleteAll
	data = s.exec(`
		mutation { deleteAllProductsPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllProductsPermanent"].(map[string]interface{})["status"])
}

// graphql: non-existent product must surface a not-found error.
func (s *ProductGraphqlTestSuite) TestProductGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindProductById($input: FindByIdProductInput!) {
			findProductById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent product must return an error")
}

// graphql: zero product ID must be rejected as a bad request.
func (s *ProductGraphqlTestSuite) TestProductGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindProductById($input: FindByIdProductInput!) {
			findProductById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero product ID must return an error")
}

func TestProductGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(ProductGraphqlTestSuite))
}
