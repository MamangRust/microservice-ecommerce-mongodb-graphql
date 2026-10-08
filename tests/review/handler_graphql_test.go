package review_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type ReviewGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler    http.Handler
	reviewID   int
	userID     int
	prodID     int
	merchantID int
}

func (s *ReviewGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.ExperienceDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupMerchantService()
	s.SetupCategoryService()
	s.SetupProductService()
	s.SetupOrderItemService()
	s.SetupShippingAddressService()
	s.SetupOrderService()
	s.SetupReviewService()

	// Seed dependencies
	ctx := context.Background()
	s.userID = s.SeedUser(ctx)
	catID := s.SeedCategory(ctx)
	s.merchantID = s.SeedMerchant(ctx, s.userID)
	s.prodID = s.SeedProduct(ctx, s.merchantID, catID)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *ReviewGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *ReviewGraphqlTestSuite) TestReviewGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateReview($input: CreateReviewRequest!) {
			createReview(input: $input) {
				status
				message
				data { id user_id product_id name comment rating }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"user_id":    s.userID,
			"product_id": s.prodID,
			"name":       "Test Reviewer",
			"comment":    "Test Comment",
			"rating":     5,
		},
	})
	created := data["createReview"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Test Comment", created["comment"])
	s.reviewID = int(created["id"].(float64))
	s.Require().NotZero(s.reviewID)

	// 2. FindAll
	data = s.exec(`
		query FindAllReviews($input: FindAllReviewRequest!) {
			findAllReviews(input: $input) { status data { id } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllReviews"].(map[string]interface{})["data"])

	// 3. FindByProduct
	data = s.exec(`
		query FindReviewsByProduct($input: FindAllReviewProductRequest!) {
			findReviewsByProduct(input: $input) { status data { id product_id rating } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"product_id": s.prodID, "page": 1, "page_size": 10}})
	s.NotEmpty(data["findReviewsByProduct"].(map[string]interface{})["data"])

	// 4. FindByMerchant
	data = s.exec(`
		query FindReviewsByMerchant($input: FindAllReviewMerchantRequest!) {
			findReviewsByMerchant(input: $input) { status data { id product_id rating } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"merchant_id": s.merchantID, "page": 1, "page_size": 10}})
	s.NotEmpty(data["findReviewsByMerchant"].(map[string]interface{})["data"])

	// 5. FindByActive
	data = s.exec(`
		query FindActiveReviews($input: FindAllReviewRequest!) {
			findActiveReviews(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.Equal("success", data["findActiveReviews"].(map[string]interface{})["status"])

	// 6. Update
	data = s.exec(`
		mutation UpdateReview($input: UpdateReviewRequest!) {
			updateReview(input: $input) { status data { id comment rating } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"review_id": s.reviewID,
			"name":      "Test Reviewer",
			"comment":   "Updated Comment",
			"rating":    4,
		},
	})
	updated := data["updateReview"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Comment", updated["comment"])

	// 7. Trash
	data = s.exec(`
		mutation TrashedReview($input: FindByIdReviewRequest!) {
			trashedReview(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewID}})
	s.Equal("success", data["trashedReview"].(map[string]interface{})["status"])

	// 8. FindByTrashed
	data = s.exec(`
		query FindTrashedReviews($input: FindAllReviewRequest!) {
			findTrashedReviews(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.Equal("success", data["findTrashedReviews"].(map[string]interface{})["status"])

	// 9. Restore
	data = s.exec(`
		mutation RestoreReview($input: FindByIdReviewRequest!) {
			restoreReview(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewID}})
	s.Equal("success", data["restoreReview"].(map[string]interface{})["status"])

	// 10. DeletePermanent
	s.exec(`
		mutation TrashedReview($input: FindByIdReviewRequest!) {
			trashedReview(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewID}})
	data = s.exec(`
		mutation DeleteReviewPermanent($input: FindByIdReviewRequest!) {
			deleteReviewPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.reviewID}})
	s.Equal("success", data["deleteReviewPermanent"].(map[string]interface{})["status"])

	// 11. RestoreAll
	data = s.exec(`mutation { restoreAllReviews { status } }`, nil)
	s.Equal("success", data["restoreAllReviews"].(map[string]interface{})["status"])

	// 12. DeleteAll
	data = s.exec(`mutation { deleteAllReviewsPermanent { status } }`, nil)
	s.Equal("success", data["deleteAllReviewsPermanent"].(map[string]interface{})["status"])
}

func TestReviewGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(ReviewGraphqlTestSuite))
}
