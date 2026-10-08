package slider_test

import (
	"net/http"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type SliderGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler  http.Handler
	sliderID int
}

func (s *SliderGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.ExperienceDB
	s.BaseTestSuite.SetupSuite()
	s.SetupSliderService()

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *SliderGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *SliderGraphqlTestSuite) TestSliderGraphqlLifecycle() {
	// 1. Create (image: Upload! is required, so use a multipart request)
	createResp, err := graphtest.ExecuteGraphQLMultipart(s.handler, `
		mutation CreateSlider($input: CreateSliderRequest!) {
			createSlider(input: $input) {
				status
				message
				data { id name image }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":  "Test Slider",
			"image": nil,
		},
	}, map[string]graphtest.UploadFile{
		"variables.input.image": {
			Filename:    "slider.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy image content"),
		},
	}, "")
	s.Require().NoError(err)
	s.Require().Empty(createResp.Errors, "graphql errors: %+v", createResp.Errors)

	created := createResp.Data["createSlider"].(map[string]interface{})
	slider := created["data"].(map[string]interface{})
	s.Equal("Test Slider", slider["name"])
	s.sliderID = int(slider["id"].(float64))

	// 2. FindAll
	data := s.exec(`
		query FindAllSliders($input: FindAllSliderRequest!) {
			findAllSliders(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findAllSliders"].(map[string]interface{})["data"])

	// 3. FindByActive
	data = s.exec(`
		query FindActiveSliders($input: FindAllSliderRequest!) {
			findActiveSliders(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findActiveSliders"].(map[string]interface{})["data"])

	// 4. Update
	s.Require().NotZero(s.sliderID)
	updateResp, err := graphtest.ExecuteGraphQLMultipart(s.handler, `
		mutation UpdateSlider($input: UpdateSliderRequest!) {
			updateSlider(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"id":    s.sliderID,
			"name":  "Updated Test Slider",
			"image": nil,
		},
	}, map[string]graphtest.UploadFile{
		"variables.input.image": {
			Filename:    "slider_updated.jpg",
			ContentType: "image/jpeg",
			Content:     []byte("dummy image content"),
		},
	}, "")
	s.Require().NoError(err)
	s.Require().Empty(updateResp.Errors, "graphql errors: %+v", updateResp.Errors)
	updated := updateResp.Data["updateSlider"].(map[string]interface{})["data"].(map[string]interface{})
	s.Equal("Updated Test Slider", updated["name"])

	// 5. Trash
	data = s.exec(`
		mutation TrashedSlider($input: FindByIdSliderRequest!) {
			trashedSlider(input: $input) { status data { id name deleted_at } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.sliderID}})
	s.Equal("success", data["trashedSlider"].(map[string]interface{})["status"])

	// 6. FindByTrashed
	data = s.exec(`
		query FindTrashedSliders($input: FindAllSliderRequest!) {
			findTrashedSliders(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"page": 1, "page_size": 10}})
	s.NotEmpty(data["findTrashedSliders"].(map[string]interface{})["data"])

	// 7. Restore
	data = s.exec(`
		mutation RestoreSlider($input: FindByIdSliderRequest!) {
			restoreSlider(input: $input) { status data { id name } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.sliderID}})
	s.Equal("success", data["restoreSlider"].(map[string]interface{})["status"])

	// 8. DeletePermanent (trash first)
	s.exec(`
		mutation TrashedSlider($input: FindByIdSliderRequest!) {
			trashedSlider(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.sliderID}})
	data = s.exec(`
		mutation DeleteSliderPermanent($input: FindByIdSliderRequest!) {
			deleteSliderPermanent(input: $input) { status }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"id": s.sliderID}})
	s.Equal("success", data["deleteSliderPermanent"].(map[string]interface{})["status"])

	// 9. RestoreAll
	data = s.exec(`
		mutation { restoreAllSliders { status } }
	`, nil)
	s.Equal("success", data["restoreAllSliders"].(map[string]interface{})["status"])

	// 10. DeleteAll
	data = s.exec(`
		mutation { deleteAllSlidersPermanent { status } }
	`, nil)
	s.Equal("success", data["deleteAllSlidersPermanent"].(map[string]interface{})["status"])
}

func TestSliderGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(SliderGraphqlTestSuite))
}
