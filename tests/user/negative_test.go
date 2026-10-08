package user_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbuser "github.com/MamangRust/microservice-ecommerce-grpc-pb/user"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: non-existent user must map to codes.NotFound (404), not Internal.
func (s *UserGapiTestSuite) TestUserGapiNotFound() {
	ctx := context.Background()
	_, err := s.queryClient.FindById(ctx, &pbuser.FindByIdUserRequest{Id: 999999})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "non-existent user must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: non-existent user must surface a not-found error.
func (s *UserGraphqlTestSuite) TestUserGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindUserById($input: FindByIdUserInput!) {
			findByIdUser(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent user must return an error")
}

// graphql: zero user ID must be rejected as a bad request.
func (s *UserGraphqlTestSuite) TestUserGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindUserById($input: FindByIdUserInput!) {
			findByIdUser(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero user ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *UserRepositoryTestSuite) TestUserFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.UserQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
