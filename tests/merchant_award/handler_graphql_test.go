package merchant_award_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type MerchantAwardGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler    http.Handler
	awardID    int
	merchantID int
}

func (s *MerchantAwardGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.MerchantDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupMerchantService()
	s.SetupMerchantAwardService()

	// Seed dependencies
	ctx := context.Background()
	userID := s.SeedUser(ctx)
	s.merchantID = s.SeedMerchant(ctx, userID)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *MerchantAwardGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *MerchantAwardGraphqlTestSuite) TestMerchantAwardGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateMerchantAward($input: CreateMerchantAwardInput!) {
			createMerchantAward(input: $input) {
				status
				message
				data { id merchant_id title description issued_by issue_date expiry_date certificate_url }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_id":     s.merchantID,
			"title":           "Best Merchant 2024",
			"description":     "Award for excellence",
			"issued_by":       "E-commerce Platform",
			"issue_date":      "2024-01-01",
			"expiry_date":     "2025-01-01",
			"certificate_url": "http://example.com/cert.pdf",
		},
	})
	created := data["createMerchantAward"].(map[string]interface{})
	award := created["data"].(map[string]interface{})
	s.Equal("Best Merchant 2024", award["title"])
	s.awardID = int(award["id"].(float64))

	// 2. FindById
	data = s.exec(`
		query FindMerchantAwardById($input: FindByIdMerchantAwardInput!) {
			findMerchantAwardById(input: $input) { status data { id title } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.awardID}})
	found := data["findMerchantAwardById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.awardID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllMerchantAwards($input: FindAllMerchantAwardInput!) {
			findAllMerchantAwards(input: $input) { status data { id title } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllMerchantAwards"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveMerchantAwards($input: FindAllMerchantAwardInput!) {
			findActiveMerchantAwards(input: $input) { status data { id title } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveMerchantAwards"].(map[string]interface{})["data"])

	// 5. Update
	data = s.exec(`
		mutation UpdateMerchantAward($input: UpdateMerchantAwardInput!) {
			updateMerchantAward(input: $input) { status data { id title description } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_certification_id": s.awardID,
			"title":                     "Updated Award Title",
			"description":               "Updated Description",
			"issued_by":                 "Updated Issuer",
			"issue_date":                "2024-02-01",
			"expiry_date":               "2025-02-01",
			"certificate_url":           "http://example.com/updated.pdf",
		},
	})
	updated := data["updateMerchantAward"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Award Title", updated["title"])

	// 6. Trash
	data = s.exec(`
		mutation TrashMerchantAward($input: FindByIdMerchantAwardInput!) {
			trashMerchantAward(input: $input) { status data { id title deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.awardID}})
	s.Equal("success", data["trashMerchantAward"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedMerchantAwards($input: FindAllMerchantAwardInput!) {
			findTrashedMerchantAwards(input: $input) { status data { id title } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedMerchantAwards"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreMerchantAward($input: FindByIdMerchantAwardInput!) {
			restoreMerchantAward(input: $input) { status data { id title } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.awardID}})
	s.Equal("success", data["restoreMerchantAward"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashMerchantAward($input: FindByIdMerchantAwardInput!) {
			trashMerchantAward(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.awardID}})
	data = s.exec(`
		mutation DeleteMerchantAwardPermanent($input: FindByIdMerchantAwardInput!) {
			deleteMerchantAwardPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.awardID}})
	s.Equal("success", data["deleteMerchantAwardPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllMerchantAwards { status } }
	`, nil)
	s.Equal("success", data["restoreAllMerchantAwards"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllMerchantAwardsPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllMerchantAwardsPermanent"].(map[string]interface{})["status"])
}

func TestMerchantAwardGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantAwardGraphqlTestSuite))
}
