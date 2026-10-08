package order_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pborder "github.com/MamangRust/microservice-ecommerce-grpc-pb/order"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: non-existent order must map to codes.NotFound (404), not Internal.
func (s *OrderGapiTestSuite) TestOrderGapiNotFound() {
	ctx := context.Background()
	_, err := s.queryClient.FindById(ctx, &pborder.FindByIdOrderRequest{Id: 999999})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "non-existent order must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: non-existent order must surface a not-found error.
func (s *OrderGraphqlTestSuite) TestOrderGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindOrderById($input: FindByIdOrderInput!) {
			findOrderById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent order must return an error")
}

// graphql: zero order ID must be rejected as a bad request.
func (s *OrderGraphqlTestSuite) TestOrderGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindOrderById($input: FindByIdOrderInput!) {
			findOrderById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero order ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *OrderRepositoryTestSuite) TestOrderFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.OrderQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
