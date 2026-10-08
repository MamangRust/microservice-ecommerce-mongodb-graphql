package merchant_policy_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type MerchantPolicyGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler    http.Handler
	policyID   int
	merchantID int
	userID     int
}

func (s *MerchantPolicyGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.MerchantDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupMerchantService()
	s.SetupMerchantPolicyService()

	// Seed dependencies
	ctx := context.Background()
	s.userID = s.SeedUser(ctx)
	s.merchantID = s.SeedMerchant(ctx, s.userID)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *MerchantPolicyGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *MerchantPolicyGraphqlTestSuite) TestMerchantPolicyGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateMerchantPolicy($input: CreateMerchantPoliciesInput!) {
			createMerchantPolicy(input: $input) {
				status
				message
				data { id merchant_id policy_type title description }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_id": s.merchantID,
			"policy_type": "Return",
			"title":       "Return Policy",
			"description": "30-day returns accepted",
		},
	})
	created := data["createMerchantPolicy"].(map[string]interface{})
	policy := created["data"].(map[string]interface{})
	s.Equal("Return Policy", policy["title"])
	s.policyID = int(policy["id"].(float64))

	// 2. FindById
	data = s.exec(`
		query FindMerchantPolicyById($input: FindByIdMerchantPoliciesInput!) {
			findMerchantPolicyById(input: $input) { status data { id title } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.policyID}})
	found := data["findMerchantPolicyById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.policyID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllMerchantPolicies($input: FindAllMerchantPoliciesInput!) {
			findAllMerchantPolicies(input: $input) { status data { id title } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllMerchantPolicies"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveMerchantPolicies($input: FindAllMerchantPoliciesInput!) {
			findActiveMerchantPolicies(input: $input) { status data { id title } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveMerchantPolicies"].(map[string]interface{})["data"])

	// 5. Update
	data = s.exec(`
		mutation UpdateMerchantPolicy($input: UpdateMerchantPoliciesInput!) {
			updateMerchantPolicy(input: $input) { status data { id title description } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_policy_id": s.policyID,
			"policy_type":        "Shipping",
			"title":              "Shipping Policy",
			"description":        "Free shipping over $50",
		},
	})
	updated := data["updateMerchantPolicy"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Shipping Policy", updated["title"])

	// 6. Trash
	data = s.exec(`
		mutation TrashMerchantPolicy($input: FindByIdMerchantPoliciesInput!) {
			trashMerchantPolicy(input: $input) { status data { id title deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.policyID}})
	s.Equal("success", data["trashMerchantPolicy"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedMerchantPolicies($input: FindAllMerchantPoliciesInput!) {
			findTrashedMerchantPolicies(input: $input) { status data { id title } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedMerchantPolicies"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreMerchantPolicy($input: FindByIdMerchantPoliciesInput!) {
			restoreMerchantPolicy(input: $input) { status data { id title } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.policyID}})
	s.Equal("success", data["restoreMerchantPolicy"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashMerchantPolicy($input: FindByIdMerchantPoliciesInput!) {
			trashMerchantPolicy(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.policyID}})
	data = s.exec(`
		mutation DeleteMerchantPolicyPermanent($input: FindByIdMerchantPoliciesInput!) {
			deleteMerchantPolicyPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.policyID}})
	s.Equal("success", data["deleteMerchantPolicyPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllMerchantPolicies { status } }
	`, nil)
	s.Equal("success", data["restoreAllMerchantPolicies"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllMerchantPoliciesPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllMerchantPoliciesPermanent"].(map[string]interface{})["status"])
}

func TestMerchantPolicyGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantPolicyGraphqlTestSuite))
}
