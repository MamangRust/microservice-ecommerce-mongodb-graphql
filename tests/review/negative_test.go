package review_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbreview "github.com/MamangRust/microservice-ecommerce-grpc-pb/review"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: review has no FindById query RPC; Update on a non-existent review must
// map to codes.NotFound (404), not Internal.
func (s *ReviewGapiTestSuite) TestReviewGapiNotFound() {
	ctx := context.Background()
	_, err := s.commandClient.Update(ctx, &pbreview.UpdateReviewRequest{
		ReviewId: 999999,
		Rating:   5,
		Comment:  "not found",
	})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "update on non-existent review must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: update on a non-existent review must surface a not-found error.
func (s *ReviewGraphqlTestSuite) TestReviewGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		mutation UpdateReview($input: UpdateReviewRequest!) {
			updateReview(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{
		"review_id": 999999,
		"name":      "not found",
		"comment":   "not found",
		"rating":    5,
	}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "update on non-existent review must return an error")
}

// graphql: zero review ID must be rejected as a bad request.
func (s *ReviewGraphqlTestSuite) TestReviewGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		mutation UpdateReview($input: UpdateReviewRequest!) {
			updateReview(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{
		"review_id": 0,
		"name":      "invalid",
		"comment":   "invalid",
		"rating":    5,
	}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero review ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *ReviewRepositoryTestSuite) TestReviewFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.ReviewQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
