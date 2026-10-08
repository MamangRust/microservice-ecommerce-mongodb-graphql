package banner_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbbanner "github.com/MamangRust/microservice-ecommerce-grpc-pb/banner"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: non-existent banner must map to codes.NotFound (404), not Internal.
func (s *BannerGapiTestSuite) TestBannerGapiNotFound() {
	ctx := context.Background()
	_, err := s.queryClient.FindById(ctx, &pbbanner.FindByIdBannerRequest{Id: 999999})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "non-existent banner must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: non-existent banner must surface a not-found error.
func (s *BannerGraphqlTestSuite) TestBannerGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindBannerById($input: FindByIdBannerInput!) {
			findBannerById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent banner must return an error")
}

// graphql: zero banner ID must be rejected as a bad request.
func (s *BannerGraphqlTestSuite) TestBannerGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindBannerById($input: FindByIdBannerInput!) {
			findBannerById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero banner ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *BannerRepositoryTestSuite) TestBannerFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.BannerQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
