package merchant_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type MerchantGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler    http.Handler
	merchantID int
	userID     int
}

func (s *MerchantGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.MerchantDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupMerchantService()

	// Seed user
	s.userID = s.SeedUser(context.Background())

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *MerchantGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *MerchantGraphqlTestSuite) TestMerchantGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateMerchant($input: CreateMerchantInput!) {
			createMerchant(input: $input) {
				status
				message
				data { id user_id name description address contact_email contact_phone status }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"user_id":       s.userID,
			"name":          "Test Merchant",
			"description":   "Test Description",
			"address":       "Test Address",
			"contact_email": "merchant@example.com",
			"contact_phone": "123456789",
			"status":        "active",
		},
	})
	created := data["createMerchant"].(map[string]interface{})
	merchant := created["data"].(map[string]interface{})
	s.Equal("Test Merchant", merchant["name"])
	s.merchantID = int(merchant["id"].(float64))

	// 2. FindById
	data = s.exec(`
		query FindMerchantById($input: FindByIdMerchantInput!) {
			findMerchantById(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	found := data["findMerchantById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.merchantID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllMerchants($input: FindAllMerchantInput!) {
			findAllMerchants(input: $input) { status data { id name } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllMerchants"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveMerchants($input: FindAllMerchantInput!) {
			findActiveMerchants(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveMerchants"].(map[string]interface{})["data"])

	// 5. Update
	data = s.exec(`
		mutation UpdateMerchant($input: UpdateMerchantInput!) {
			updateMerchant(input: $input) { status data { id name description } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_id":   s.merchantID,
			"user_id":       s.userID,
			"name":          "Updated Merchant",
			"description":   "Updated Description",
			"address":       "Updated Address",
			"contact_email": "updated@example.com",
			"contact_phone": "987654321",
			"status":        "active",
		},
	})
	updated := data["updateMerchant"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Merchant", updated["name"])

	// 6. Trash
	data = s.exec(`
		mutation TrashMerchant($input: FindByIdMerchantInput!) {
			trashMerchant(input: $input) { status data { id name deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	s.Equal("success", data["trashMerchant"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedMerchants($input: FindAllMerchantInput!) {
			findTrashedMerchants(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedMerchants"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreMerchant($input: FindByIdMerchantInput!) {
			restoreMerchant(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	s.Equal("success", data["restoreMerchant"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashMerchant($input: FindByIdMerchantInput!) {
			trashMerchant(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	data = s.exec(`
		mutation DeleteMerchantPermanent($input: FindByIdMerchantInput!) {
			deleteMerchantPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.merchantID}})
	s.Equal("success", data["deleteMerchantPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllMerchants { status } }
	`, nil)
	s.Equal("success", data["restoreAllMerchants"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllMerchantsPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllMerchantsPermanent"].(map[string]interface{})["status"])
}

func TestMerchantGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantGraphqlTestSuite))
}
