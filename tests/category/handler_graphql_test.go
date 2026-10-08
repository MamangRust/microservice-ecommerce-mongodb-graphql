package category_test

import (
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type CategoryGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler    http.Handler
	categoryID int
}

func (s *CategoryGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.CatalogDB
	s.BaseTestSuite.SetupSuite()
	s.SetupCategoryService()

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *CategoryGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *CategoryGraphqlTestSuite) TestCategoryGraphqlLifecycle() {
	// 1. Create
	createResp, err := graphtest.ExecuteGraphQLMultipart(s.handler, `
		mutation CreateCategory($input: CreateCategoryInput!) {
			createCategory(input: $input) {
				status
				message
				data { id name description slug_category }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":           "Test Category",
			"description":    "Test Description",
			"slug_category":  "test-category",
			"image_category": nil,
		},
	}, map[string]graphtest.UploadFile{
		"variables.input.image_category": {
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy image content"),
		},
	}, "")
	s.Require().NoError(err)
	s.Require().Empty(createResp.Errors, "graphql errors: %+v", createResp.Errors)

	created := createResp.Data["createCategory"].(map[string]interface{})
	category := created["data"].(map[string]interface{})
	s.Equal("Test Category", category["name"])
	s.categoryID = int(category["id"].(float64))

	// 2. FindById
	data := s.exec(`
		query FindCategoryById($input: FindByIdCategoryInput!) {
			findCategoryById(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	found := data["findCategoryById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.categoryID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllCategories($input: FindAllCategoryInput!) {
			findAllCategories(input: $input) { status data { id name } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllCategories"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveCategories($input: FindAllCategoryInput!) {
			findActiveCategories(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveCategories"].(map[string]interface{})["data"])

	// 5. Update
	updateResp, err := graphtest.ExecuteGraphQLMultipart(s.handler, `
		mutation UpdateCategory($input: UpdateCategoryInput!) {
			updateCategory(input: $input) { status data { id name description } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"category_id":    s.categoryID,
			"name":           "Updated Category",
			"description":    "Updated Description",
			"slug_category":  "updated-category",
			"image_category": nil,
		},
	}, map[string]graphtest.UploadFile{
		"variables.input.image_category": {
			Filename:    "updated.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy image content"),
		},
	}, "")
	s.Require().NoError(err)
	s.Require().Empty(updateResp.Errors, "graphql errors: %+v", updateResp.Errors)
	updated := updateResp.Data["updateCategory"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Category", updated["name"])

	// 6. Trash
	data = s.exec(`
		mutation TrashCategory($input: FindByIdCategoryInput!) {
			trashCategory(input: $input) { status data { id name deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	s.Equal("success", data["trashCategory"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedCategories($input: FindAllCategoryInput!) {
			findTrashedCategories(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedCategories"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreCategory($input: FindByIdCategoryInput!) {
			restoreCategory(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	s.Equal("success", data["restoreCategory"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashCategory($input: FindByIdCategoryInput!) {
			trashCategory(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	data = s.exec(`
		mutation DeleteCategoryPermanent($input: FindByIdCategoryInput!) {
			deleteCategoryPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.categoryID}})
	s.Equal("success", data["deleteCategoryPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllCategories { status } }
	`, nil)
	s.Equal("success", data["restoreAllCategories"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllCategoriesPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllCategoriesPermanent"].(map[string]interface{})["status"])
}

func TestCategoryGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(CategoryGraphqlTestSuite))
}
