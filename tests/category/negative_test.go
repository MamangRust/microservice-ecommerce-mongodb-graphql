package category_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbcategory "github.com/MamangRust/microservice-ecommerce-grpc-pb/category"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: non-existent category must map to codes.NotFound (404), not Internal.
func (s *CategoryGapiTestSuite) TestCategoryGapiNotFound() {
	ctx := context.Background()
	_, err := s.queryClient.FindById(ctx, &pbcategory.FindByIdCategoryRequest{Id: 999999})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "non-existent category must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: non-existent category must surface a not-found error.
func (s *CategoryGraphqlTestSuite) TestCategoryGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindCategoryById($input: FindByIdCategoryInput!) {
			findCategoryById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent category must return an error")
}

// graphql: zero category ID must be rejected as a bad request.
func (s *CategoryGraphqlTestSuite) TestCategoryGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindCategoryById($input: FindByIdCategoryInput!) {
			findCategoryById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero category ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *CategoryRepositoryTestSuite) TestCategoryFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.CategoryQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
