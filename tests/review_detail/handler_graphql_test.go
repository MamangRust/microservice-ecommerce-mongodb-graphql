package review_detail_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type ReviewDetailGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler        http.Handler
	reviewDetailID int
	reviewID       int
}

func (s *ReviewDetailGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.ExperienceDB
	s.BaseTestSuite.SetupSuite()

	// Setup dependencies
	s.SetupUserService()
	s.SetupCategoryService()
	s.SetupMerchantService()
	s.SetupProductService()
	s.SetupOrderItemService()
	s.SetupShippingAddressService()
	s.SetupOrderService()
	s.SetupReviewService()
	s.SetupReviewDetailService()

	// Seed dependencies
	ctx := context.Background()
	userID := s.SeedUser(ctx)
	catID := s.SeedCategory(ctx)
	merchID := s.SeedMerchant(ctx, userID)
	prodID := s.SeedProduct(ctx, merchID, catID)
	s.SeedOrder(ctx, userID, merchID, prodID)
	s.reviewID = s.SeedReview(ctx, userID, prodID)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *ReviewDetailGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *ReviewDetailGraphqlTestSuite) TestReviewDetailGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateReviewDetail($input: CreateReviewDetailInput!) {
			createReviewDetail(input: $input) {
				status
				message
				data { id reviewId type url caption }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"reviewId": s.reviewID,
			"type":     "photo",
			"url":      "http://example.com/review_photo.jpg",
			"caption":  "API Test Caption",
		},
	})
	created := data["createReviewDetail"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("API Test Caption", created["caption"])
	s.reviewDetailID = int(created["id"].(float64))
	s.Require().NotZero(s.reviewDetailID)

	// 2. FindById
	data = s.exec(`
		query FindReviewDetailById($input: FindByIdReviewDetailInput!) {
			findReviewDetailById(input: $input) { status data { id reviewId caption } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewDetailID}})
	found := data["findReviewDetailById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.reviewDetailID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllReviewDetails($input: FindAllReviewDetailInput!) {
			findAllReviewDetails(input: $input) {
				status
				data { id }
				pagination { current_page page_size total_records }
			}
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "pageSize": 10}})
	s.NotEmpty(data["findAllReviewDetails"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveReviewDetails($input: FindAllReviewDetailInput!) {
			findActiveReviewDetails(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "pageSize": 10}})
	s.Equal("success", data["findActiveReviewDetails"].(map[string]interface{})["status"])

	// 5. Update
	data = s.exec(`
		mutation UpdateReviewDetail($input: UpdateReviewDetailInput!) {
			updateReviewDetail(input: $input) { status data { id type caption } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"reviewDetailId": s.reviewDetailID,
			"type":           "video",
			"url":            "http://example.com/review_video.mp4",
			"caption":        "Updated Caption",
		},
	})
	updated := data["updateReviewDetail"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Caption", updated["caption"])

	// 6. Trash
	data = s.exec(`
		mutation TrashedReviewDetail($input: FindByIdReviewDetailInput!) {
			trashedReviewDetail(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewDetailID}})
	s.Equal("success", data["trashedReviewDetail"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedReviewDetails($input: FindAllReviewDetailInput!) {
			findTrashedReviewDetails(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "pageSize": 10}})
	s.NotEmpty(data["findTrashedReviewDetails"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreReviewDetail($input: FindByIdReviewDetailInput!) {
			restoreReviewDetail(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewDetailID}})
	s.Equal("success", data["restoreReviewDetail"].(map[string]interface{})["status"])

	// 9. DeletePermanent (must be trashed first)
	s.exec(`
		mutation TrashedReviewDetail($input: FindByIdReviewDetailInput!) {
			trashedReviewDetail(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewDetailID}})
	data = s.exec(`
		mutation DeleteReviewDetailPermanent($input: FindByIdReviewDetailInput!) {
			deleteReviewDetailPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewDetailID}})
	s.Equal("success", data["deleteReviewDetailPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`mutation { restoreAllReviewDetails { status } }`, nil)
	s.Equal("success", data["restoreAllReviewDetails"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`mutation { deleteAllReviewDetailsPermanent { status } }`, nil)
	s.Equal("success", data["deleteAllReviewDetailsPermanent"].(map[string]interface{})["status"])
}

func TestReviewDetailGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(ReviewDetailGraphqlTestSuite))
}
