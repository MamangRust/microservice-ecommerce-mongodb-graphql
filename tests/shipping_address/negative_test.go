package shipping_address_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbshipping_address "github.com/MamangRust/microservice-ecommerce-grpc-pb/shipping_address"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: non-existent shipping address must map to codes.NotFound (404), not Internal.
func (s *ShippingAddressGapiTestSuite) TestShippingAddressGapiNotFound() {
	ctx := context.Background()
	_, err := s.queryClient.FindById(ctx, &pbshipping_address.FindByIdShippingRequest{Id: 999999})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "non-existent shipping address must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: non-existent shipping address must surface a not-found error.
func (s *ShippingAddressGraphqlTestSuite) TestShippingAddressGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindShippingById($input: FindByIdShippingRequest!) {
			findShippingById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent shipping address must return an error")
}

// graphql: zero shipping address ID must be rejected as a bad request.
func (s *ShippingAddressGraphqlTestSuite) TestShippingAddressGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindShippingById($input: FindByIdShippingRequest!) {
			findShippingById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero shipping address ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *ShippingAddressRepositoryTestSuite) TestShippingAddressFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.ShippingAddressQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
