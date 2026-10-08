package banner_test

import (
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type BannerGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler  http.Handler
	bannerID int
}

func (s *BannerGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.ExperienceDB
	s.BaseTestSuite.SetupSuite()
	s.SetupBannerService()

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *BannerGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *BannerGraphqlTestSuite) TestBannerGraphqlLifecycle() {
	// 1. Create
	data := s.exec(`
		mutation CreateBanner($input: CreateBannerInput!) {
			createBanner(input: $input) {
				status
				message
				data { banner_id name start_date end_date start_time end_time is_active }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":       "Test Banner",
			"start_date": "2024-01-01",
			"end_date":   "2024-12-31",
			"start_time": "00:00:00",
			"end_time":   "23:59:59",
			"is_active":  true,
		},
	})
	created := data["createBanner"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Test Banner", created["name"])
	s.bannerID = int(created["banner_id"].(float64))

	// 2. FindById
	data = s.exec(`
		query FindBannerById($input: FindByIdBannerInput!) {
			findBannerById(input: $input) { status data { banner_id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.bannerID}})
	found := data["findBannerById"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal(float64(s.bannerID), found["banner_id"])

	// 3. FindAll
	data = s.exec(`
		query FindAllBanners($input: FindAllBannerInput!) {
			findAllBanners(input: $input) { status data { banner_id name } pagination { current_page page_size total_records } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllBanners"].(map[string]interface{})["data"])

	// 4. FindByActive
	data = s.exec(`
		query FindActiveBanners($input: FindAllBannerInput!) {
			findActiveBanners(input: $input) { status data { banner_id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveBanners"].(map[string]interface{})["data"])

	// 5. Update
	data = s.exec(`
		mutation UpdateBanner($input: UpdateBannerInput!) {
			updateBanner(input: $input) {
				status
				data { banner_id name is_active }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"banner_id":  s.bannerID,
			"name":       "Updated Banner",
			"start_date": "2024-01-01",
			"end_date":   "2024-12-31",
			"start_time": "00:00:00",
			"end_time":   "23:59:59",
			"is_active":  false,
		},
	})
	updated := data["updateBanner"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Banner", updated["name"])

	// 6. Trash
	data = s.exec(`
		mutation TrashBanner($input: FindByIdBannerInput!) {
			trashBanner(input: $input) { status data { banner_id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.bannerID}})
	s.Equal("success", data["trashBanner"].(map[string]interface{})["status"])

	// 7. FindByTrashed
	data = s.exec(`
		query FindTrashedBanners($input: FindAllBannerInput!) {
			findTrashedBanners(input: $input) { status data { banner_id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedBanners"].(map[string]interface{})["data"])

	// 8. Restore
	data = s.exec(`
		mutation RestoreBanner($input: FindByIdBannerInput!) {
			restoreBanner(input: $input) { status data { banner_id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.bannerID}})
	s.Equal("success", data["restoreBanner"].(map[string]interface{})["status"])

	// 9. DeletePermanent
	s.exec(`
		mutation TrashBanner($input: FindByIdBannerInput!) {
			trashBanner(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.bannerID}})
	data = s.exec(`
		mutation DeleteBannerPermanent($input: FindByIdBannerInput!) {
			deleteBannerPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.bannerID}})
	s.Equal("success", data["deleteBannerPermanent"].(map[string]interface{})["status"])

	// 10. RestoreAll
	data = s.exec(`
		mutation { restoreAllBanners { status } }
	`, nil)
	s.Equal("success", data["restoreAllBanners"].(map[string]interface{})["status"])

	// 11. DeleteAll
	data = s.exec(`
		mutation { deleteAllBannersPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllBannersPermanent"].(map[string]interface{})["status"])
}

func TestBannerGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(BannerGraphqlTestSuite))
}
