package role_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbrole "github.com/MamangRust/microservice-ecommerce-grpc-pb/role"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: non-existent role must map to codes.NotFound (404), not Internal.
func (s *RoleGapiTestSuite) TestRoleGapiNotFound() {
	ctx := context.Background()
	_, err := s.queryClient.FindByIdRole(ctx, &pbrole.FindByIdRoleRequest{RoleId: 999999})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "non-existent role must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: non-existent role must surface a not-found error.
func (s *RoleGraphqlTestSuite) TestRoleGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindRoleById($input: FindByIdRoleInput!) {
			findByIdRole(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"role_id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent role must return an error")
}

// graphql: zero role ID must be rejected as a bad request.
func (s *RoleGraphqlTestSuite) TestRoleGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindRoleById($input: FindByIdRoleInput!) {
			findByIdRole(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"role_id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero role ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *RoleRepositoryTestSuite) TestRoleFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.RoleQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
