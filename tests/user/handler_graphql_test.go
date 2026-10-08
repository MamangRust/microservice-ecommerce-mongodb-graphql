package user_test

import (
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type UserGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler   http.Handler
	userID    int
	userEmail string
}

func (s *UserGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.IdentityDB
	s.BaseTestSuite.SetupSuite()
	s.SetupRoleService()
	s.SetupUserService()

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *UserGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *UserGraphqlTestSuite) TestUserGraphqlLifecycle() {
	// 1. Create
	s.userEmail = "graphql.user@example.com"
	data := s.exec(`
		mutation CreateUser($input: CreateUserInput!) {
			createUser(input: $input) {
				status
				message
				data { id firstname lastname email }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"firstname":        "Graphql",
			"lastname":         "User",
			"email":            s.userEmail,
			"password":         "password123",
			"confirm_password": "password123",
		},
	})
	created := data["createUser"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Graphql", created["firstname"])
	s.Equal(s.userEmail, created["email"])
	s.userID = int(created["id"].(float64))
	s.Require().NotZero(s.userID)

	// 2. FindById
	data = s.exec(`
		query FindUserById($input: FindByIdUserInput!) {
			findByIdUser(input: $input) { status data { id firstname email } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	found := data["findByIdUser"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.userID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllUsers($input: FindAllUserInput!) {
			findAllUsers(input: $input) { status data { id email } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllUsers"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveUsers($input: FindAllUserInput!) {
			findByActiveUsers(input: $input) { status data { id email } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findByActiveUsers"].(map[string]interface{})["data"])

	// 5. Update
	data = s.exec(`
		mutation UpdateUser($input: UpdateUserInput!) {
			updateUser(input: $input) { status data { id firstname lastname email } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"id":               s.userID,
			"firstname":        "Updated",
			"lastname":         "User",
			"email":            s.userEmail,
			"password":         "password123",
			"confirm_password": "password123",
		},
	})
	updated := data["updateUser"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated", updated["firstname"])

	// 6. Trash
	data = s.exec(`
		mutation TrashUser($input: FindByIdUserInput!) {
			trashedUser(input: $input) { status data { id email deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	s.Equal("success", data["trashedUser"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedUsers($input: FindAllUserInput!) {
			findByTrashedUsers(input: $input) { status data { id email } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findByTrashedUsers"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreUser($input: FindByIdUserInput!) {
			restoreUser(input: $input) { status data { id email } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	s.Equal("success", data["restoreUser"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashUser($input: FindByIdUserInput!) {
			trashedUser(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	data = s.exec(`
		mutation DeleteUserPermanent($input: FindByIdUserInput!) {
			deleteUserPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.userID}})
	s.Equal("success", data["deleteUserPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllUser { status } }
	`, nil)
	s.Equal("success", data["restoreAllUser"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllUserPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllUserPermanent"].(map[string]interface{})["status"])
}

func TestUserGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(UserGraphqlTestSuite))
}
