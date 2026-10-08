package merchant_business_test

import (
	"context"
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type MerchantBusinessGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler    http.Handler
	businessID int
	merchantID int
	userID     int
}

func (s *MerchantBusinessGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.MerchantDB
	s.BaseTestSuite.SetupSuite()
	s.SetupUserService()
	s.SetupMerchantService()
	s.SetupMerchantBusinessService()

	// Seed dependencies
	ctx := context.Background()
	s.userID = s.SeedUser(ctx)
	s.merchantID = s.SeedMerchant(ctx, s.userID)

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *MerchantBusinessGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *MerchantBusinessGraphqlTestSuite) TestMerchantBusinessGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateMerchantBusiness($input: CreateMerchantBusinessInput!) {
			createMerchantBusiness(input: $input) {
				status
				message
				data { id merchant_id business_type tax_id established_year number_of_employees website_url merchant_name }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_id":         s.merchantID,
			"business_type":       "Retail",
			"tax_id":              "123-456-789",
			"established_year":    2020,
			"number_of_employees": 10,
			"website_url":         "http://example.com",
		},
	})
	created := data["createMerchantBusiness"].(map[string]interface{})
	business := created["data"].(map[string]interface{})
	s.Equal("Retail", business["business_type"])
	s.businessID = int(business["id"].(float64))

	// 2. FindById
	data = s.exec(`
		query FindMerchantBusinessById($input: FindByIdMerchantBusinessInput!) {
			findMerchantBusinessById(input: $input) { status data { id business_type } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.businessID}})
	found := data["findMerchantBusinessById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.businessID), found["id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllMerchantBusinesses($input: FindAllMerchantBusinessInput!) {
			findAllMerchantBusinesses(input: $input) { status data { id business_type } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllMerchantBusinesses"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveMerchantBusinesses($input: FindAllMerchantBusinessInput!) {
			findActiveMerchantBusinesses(input: $input) { status data { id business_type } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveMerchantBusinesses"].(map[string]interface{})["data"])

	// 5. Update
	data = s.exec(`
		mutation UpdateMerchantBusiness($input: UpdateMerchantBusinessInput!) {
			updateMerchantBusiness(input: $input) { status data { id business_type } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"merchant_business_info_id": s.businessID,
			"business_type":             "Wholesale",
			"tax_id":                    "987-654-321",
			"established_year":          2021,
			"number_of_employees":       20,
			"website_url":               "http://updated.com",
		},
	})
	updated := data["updateMerchantBusiness"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Wholesale", updated["business_type"])

	// 6. Trash
	data = s.exec(`
		mutation TrashMerchantBusiness($input: FindByIdMerchantBusinessInput!) {
			trashMerchantBusiness(input: $input) { status data { id business_type deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.businessID}})
	s.Equal("success", data["trashMerchantBusiness"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedMerchantBusinesses($input: FindAllMerchantBusinessInput!) {
			findTrashedMerchantBusinesses(input: $input) { status data { id business_type } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedMerchantBusinesses"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreMerchantBusiness($input: FindByIdMerchantBusinessInput!) {
			restoreMerchantBusiness(input: $input) { status data { id business_type } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.businessID}})
	s.Equal("success", data["restoreMerchantBusiness"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashMerchantBusiness($input: FindByIdMerchantBusinessInput!) {
			trashMerchantBusiness(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.businessID}})
	data = s.exec(`
		mutation DeleteMerchantBusinessPermanent($input: FindByIdMerchantBusinessInput!) {
			deleteMerchantBusinessPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.businessID}})
	s.Equal("success", data["deleteMerchantBusinessPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllMerchantBusinesses { status } }
	`, nil)
	s.Equal("success", data["restoreAllMerchantBusinesses"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllMerchantBusinessesPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllMerchantBusinessesPermanent"].(map[string]interface{})["status"])
}

func TestMerchantBusinessGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantBusinessGraphqlTestSuite))
}
