package order_item_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pborder_item "github.com/MamangRust/microservice-ecommerce-grpc-pb/order_item"
)

// order_item has no single-record lookup: FindOrderItemByOrder returns an empty
// list (success) for a non-existent order, so there is no NotFound path.
//
// gapi: a non-existent order must return an empty result, not an error.
func (s *OrderItemGapiTestSuite) TestOrderItemGapiEmptyResult() {
	ctx := context.Background()
	res, err := s.queryClient.FindOrderItemByOrder(ctx, &pborder_item.FindByIdOrderItemRequest{Id: 999999})
	s.NoError(err)
	s.NotNil(res)
	s.Empty(res.Data)
}

// graphql: a non-existent order returns empty data without errors.
func (s *OrderItemGraphqlTestSuite) TestOrderItemGraphqlEmptyResult() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindOrderItemsByOrder($input: FindByIdOrderItemInput!) {
			findOrderItemsByOrder(input: $input) { status data { id } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "non-existent order must not error: %+v", resp.Errors)
	items := resp.Data["findOrderItemsByOrder"].(map[string]interface{})["data"]
	if items != nil {
		s.Empty(items.([]interface{}))
	}
}

// graphql: zero order ID must be rejected as a bad request.
func (s *OrderItemGraphqlTestSuite) TestOrderItemGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindOrderItemsByOrder($input: FindByIdOrderItemInput!) {
			findOrderItemsByOrder(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero order ID must return an error")
}

// repository: FindOrderItemByOrder on a non-existent order must return an empty
// result without error.
func (s *OrderItemRepositoryTestSuite) TestOrderItemFindByOrderEmpty() {
	ctx := context.Background()
	items, err := s.repo.OrderItemQuery.FindOrderItemByOrder(ctx, 999999)
	s.NoError(err)
	s.Empty(items)
}
