package cart_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbcart "github.com/MamangRust/microservice-ecommerce-grpc-pb/cart"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: cart has no FindById query; the valid negative path is validation of
// create payload — quantity 0 must be rejected as InvalidArgument.
func (s *CartGapiTestSuite) TestCartGapiInvalidQuantity() {
	ctx := context.Background()
	_, err := s.commandClient.Create(ctx, &pbcart.CreateCartRequest{
		UserId:    1,
		ProductId: 1,
		Quantity:  0,
	})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.InvalidArgument, st.Code(), "cart create with quantity 0 must be InvalidArgument, got %v: %s", st.Code(), st.Message())
}

// graphql: cart create with quantity 0 must surface a validation error.
func (s *CartGraphqlTestSuite) TestCartGraphqlInvalidQuantity() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		mutation CreateCart($input: CreateCartInput!) {
			createCart(input: $input) { status }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"user_id":    1,
			"product_id": 1,
			"quantity":   0,
		},
	}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "cart create with quantity 0 must return an error")
}
