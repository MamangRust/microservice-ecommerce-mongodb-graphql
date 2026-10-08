package merchant_business_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbmerchant_business "github.com/MamangRust/microservice-ecommerce-grpc-pb/merchant_business"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: non-existent merchant business must map to codes.NotFound (404), not Internal.
func (s *MerchantBusinessGapiTestSuite) TestMerchantBusinessGapiNotFound() {
	ctx := context.Background()
	_, err := s.queryClient.FindById(ctx, &pbmerchant_business.FindByIdMerchantBusinessRequest{Id: 999999})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "non-existent merchant business must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: non-existent merchant business must surface a not-found error.
func (s *MerchantBusinessGraphqlTestSuite) TestMerchantBusinessGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindMerchantBusinessById($input: FindByIdMerchantBusinessInput!) {
			findMerchantBusinessById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent merchant business must return an error")
}

// graphql: zero merchant business ID must be rejected as a bad request.
func (s *MerchantBusinessGraphqlTestSuite) TestMerchantBusinessGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindMerchantBusinessById($input: FindByIdMerchantBusinessInput!) {
			findMerchantBusinessById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero merchant business ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *MerchantBusinessRepositoryTestSuite) TestMerchantBusinessFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.MerchantBusinessQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
