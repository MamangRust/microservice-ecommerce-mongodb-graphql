package merchant_detail_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type MerchantDetailGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler    http.Handler
	detailID   int
	merchantID int
	userID     int
}

func (s *MerchantDetailGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.MerchantDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupMerchantService()
	s.SetupMerchantDetailService()

	// Seed dependencies
	ctx := context.Background()
	s.userID = s.SeedUser(ctx)
	s.merchantID = s.SeedMerchant(ctx, s.userID)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *MerchantDetailGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *MerchantDetailGraphqlTestSuite) TestMerchantDetailGraphqlLifecycle() {
	// 1. Create (cover_image and logo are required Upload! fields, so use multipart)
	createResp, err := graphtest.ExecuteGraphQLMultipart(s.handler, `
		mutation CreateMerchantDetail($input: CreateMerchantDetailInput!) {
			createMerchantDetail(input: $input) {
				status
				message
				data { id merchant_id display_name }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_id":       s.merchantID,
			"display_name":      "Test Merchant",
			"cover_image":       nil,
			"logo":              nil,
			"short_description": "A test merchant",
			"website_url":       "http://example.com",
		},
	}, map[string]graphtest.UploadFile{
		"variables.input.cover_image": {
			Filename:    "cover.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy cover content"),
		},
		"variables.input.logo": {
			Filename:    "logo.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy logo content"),
		},
	}, "")
	s.Require().NoError(err)
	s.Require().Empty(createResp.Errors, "graphql errors: %+v", createResp.Errors)

	created := createResp.Data["createMerchantDetail"].(map[string]interface{})
	detail := created["data"].(map[string]interface{})
	s.Equal("Test Merchant", detail["display_name"])
	s.detailID = int(detail["id"].(float64))

	// 2. FindById
	data := s.exec(`
		query FindMerchantDetailById($input: FindByIdMerchantDetailInput!) {
			findMerchantDetailById(input: $input) { status data { id display_name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.detailID}})
	found := data["findMerchantDetailById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.detailID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllMerchantDetails($input: FindAllMerchantDetailInput!) {
			findAllMerchantDetails(input: $input) { status data { id display_name } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllMerchantDetails"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveMerchantDetails($input: FindAllMerchantDetailInput!) {
			findActiveMerchantDetails(input: $input) { status data { id display_name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveMerchantDetails"].(map[string]interface{})["data"])

	// 5. Update
	updateResp, err := graphtest.ExecuteGraphQLMultipart(s.handler, `
		mutation UpdateMerchantDetail($input: UpdateMerchantDetailInput!) {
			updateMerchantDetail(input: $input) { status data { id display_name } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_detail_id": s.detailID,
			"display_name":       "Updated Merchant",
			"cover_image":        nil,
			"logo":               nil,
			"short_description":  "Updated short description",
			"website_url":        "http://updated.com",
		},
	}, map[string]graphtest.UploadFile{
		"variables.input.cover_image": {
			Filename:    "updated_cover.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy cover content"),
		},
		"variables.input.logo": {
			Filename:    "updated_logo.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy logo content"),
		},
	}, "")
	s.Require().NoError(err)
	s.Require().Empty(updateResp.Errors, "graphql errors: %+v", updateResp.Errors)
	updated := updateResp.Data["updateMerchantDetail"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Merchant", updated["display_name"])

	// 6. Trash
	data = s.exec(`
		mutation TrashMerchantDetail($input: FindByIdMerchantDetailInput!) {
			trashMerchantDetail(input: $input) { status data { id display_name deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.detailID}})
	s.Equal("success", data["trashMerchantDetail"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedMerchantDetails($input: FindAllMerchantDetailInput!) {
			findTrashedMerchantDetails(input: $input) { status data { id display_name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedMerchantDetails"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreMerchantDetail($input: FindByIdMerchantDetailInput!) {
			restoreMerchantDetail(input: $input) { status data { id display_name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.detailID}})
	s.Equal("success", data["restoreMerchantDetail"].(map[string]interface{})["status"])

	// 9. DeletePermanent (trash first)
	s.exec(`
		mutation TrashMerchantDetail($input: FindByIdMerchantDetailInput!) {
			trashMerchantDetail(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.detailID}})
	data = s.exec(`
		mutation DeleteMerchantDetailPermanent($input: FindByIdMerchantDetailInput!) {
			deleteMerchantDetailPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.detailID}})
	s.Equal("success", data["deleteMerchantDetailPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllMerchantDetails { status } }
	`, nil)
	s.Equal("success", data["restoreAllMerchantDetails"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllMerchantDetailsPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllMerchantDetailsPermanent"].(map[string]interface{})["status"])
}

func TestMerchantDetailGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantDetailGraphqlTestSuite))
}
