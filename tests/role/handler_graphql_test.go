package role_test

import (
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type RoleGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler http.Handler
	roleID  int
}

func (s *RoleGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.IdentityDB
	s.BaseTestSuite.SetupSuite()
	s.SetupRoleService()

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *RoleGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *RoleGraphqlTestSuite) TestRoleGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateRole($input: CreateRoleInput!) {
			createRole(input: $input) {
				status
				message
				data { id name }
			}
		}
	`, map[string]interface{}{"input": map[string]interface{}{"name": "Graphql Role"}})
	created := data["createRole"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Graphql Role", created["name"])
	s.roleID = int(created["id"].(float64))
	s.Require().NotZero(s.roleID)

	// 2. FindById
	data = s.exec(`
		query FindRoleById($input: FindByIdRoleInput!) {
			findByIdRole(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	found := data["findByIdRole"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.roleID), found["id"])
	s.Equal("Graphql Role", found["name"])

	// 3. FindAll
	data = s.exec(`
		query FindAllRole($input: FindAllRoleInput!) {
			findAllRole(input: $input) { status data { id name } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllRole"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveRole($input: FindAllRoleInput!) {
			findByActiveRole(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findByActiveRole"].(map[string]interface{})["data"])

	// 5. FindByUserId
	data = s.exec(`
		query FindRoleByUser($input: FindByIdUserRoleInput!) {
			findByUserIdRole(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"user_id": 1}})
	s.Equal("success", data["findByUserIdRole"].(map[string]interface{})["status"])

	// 6. Update
	data = s.exec(`
		mutation UpdateRole($input: UpdateRoleInput!) {
			updateRole(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.roleID, "name": "Updated Graphql Role"}})
	updated := data["updateRole"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Graphql Role", updated["name"])

	// 7. Trash
	data = s.exec(`
		mutation TrashRole($input: FindByIdRoleInput!) {
			trashedRole(input: $input) { status data { id name deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	s.Equal("success", data["trashedRole"].(map[string]interface{})["status"])

	// 8. FindByTrashed
	data = s.exec(`
		query FindTrashedRole($input: FindAllRoleInput!) {
			findByTrashedRole(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findByTrashedRole"].(map[string]interface{})["data"])

	// 9. Restore
	data = s.exec(`
		mutation RestoreRole($input: FindByIdRoleInput!) {
			restoreRole(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	s.Equal("success", data["restoreRole"].(map[string]interface{})["status"])

	// 10. DeletePermanent
	s.exec(`
		mutation TrashRole($input: FindByIdRoleInput!) {
			trashedRole(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	data = s.exec(`
		mutation DeleteRolePermanent($input: FindByIdRoleInput!) {
			deleteRolePermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"role_id": s.roleID}})
	s.Equal("success", data["deleteRolePermanent"].(map[string]interface{})["status"])

	// 11. RestoreAll
	data = s.exec(`
		mutation { restoreAllRole { status } }
	`, nil)
	s.Equal("success", data["restoreAllRole"].(map[string]interface{})["status"])

	// 12. DeleteAll
	data = s.exec(`
		mutation { deleteAllRolePermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllRolePermanent"].(map[string]interface{})["status"])
}

func TestRoleGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(RoleGraphqlTestSuite))
}
