package transaction_test

import (
	"context"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	pbtransaction "github.com/MamangRust/microservice-ecommerce-grpc-pb/transaction"

	"github.com/MamangRust/microservice-ecommerce-shared/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gapi: non-existent transaction must map to codes.NotFound (404), not Internal.
func (s *TransactionGapiTestSuite) TestTransactionGapiNotFound() {
	ctx := context.Background()
	_, err := s.queryClient.FindById(ctx, &pbtransaction.FindByIdTransactionRequest{Id: 999999})
	s.Require().Error(err)
	st, ok := status.FromError(err)
	s.Require().True(ok, "expected a gRPC status error")
	s.Equal(codes.NotFound, st.Code(), "non-existent transaction must be NotFound, got %v: %s", st.Code(), st.Message())
}

// graphql: non-existent transaction must surface a not-found error.
func (s *TransactionGraphqlTestSuite) TestTransactionGraphqlNotFound() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindTransactionById($input: FindByIdTransactionRequest!) {
			findTransactionById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 999999}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "non-existent transaction must return an error")
}

// graphql: zero transaction ID must be rejected as a bad request.
func (s *TransactionGraphqlTestSuite) TestTransactionGraphqlInvalidID() {
	resp, err := graphtest.ExecuteGraphQL(s.handler, `
		query FindTransactionById($input: FindByIdTransactionRequest!) {
			findTransactionById(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": 0}}, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "zero transaction ID must return an error")
}

// repository: FindByID on a non-existent ID must return a typed not-found error.
func (s *TransactionRepositoryTestSuite) TestTransactionFindByIDNotFound() {
	ctx := context.Background()
	_, err := s.repo.TransactionQuery.FindByID(ctx, 999999)
	s.Require().Error(err)
	var appErr *errors.AppError
	s.Require().ErrorAs(err, &appErr)
	s.Equal(errors.ErrorTypeNotFound, appErr.Type, "expected not-found error type, got %s: %v", appErr.Type, err)
}
