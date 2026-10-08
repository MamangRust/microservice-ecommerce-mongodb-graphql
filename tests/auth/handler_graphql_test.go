package auth_test

import (
	"net/http"
	"strings"
	"testing"

	graphtest "github.com/MamangRust/microservice-ecommerce-grpc-apigateway/graphtest"
	"github.com/MamangRust/microservice-ecommerce-pkg/database"
	tests "github.com/MamangRust/microservice-ecommerce-test"
	"github.com/stretchr/testify/suite"
)

type AuthGraphqlTestSuite struct {
	tests.BaseTestSuite
	handler     http.Handler
	email       string
	password    string
	userID      int
	accessToken string
	refreshToken string
}

func (s *AuthGraphqlTestSuite) SetupSuite() {
	s.PrimaryContext = database.IdentityDB
	s.BaseTestSuite.SetupSuite()
	s.SetupRoleService()
	s.SetupUserService()
	s.SetupAuthService()

	s.handler = graphtest.NewTestHandler(s.Conns, s.GetCacheStore(), s.Log)

	s.email = "auth.graphql.test@example.com"
	s.password = "password123"
}

// exec runs a JSON GraphQL request and fails the test on transport or GraphQL errors.
func (s *AuthGraphqlTestSuite) exec(query string, variables map[string]interface{}) map[string]interface{} {
	resp, err := graphtest.ExecuteGraphQL(s.handler, query, variables, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	return resp.Data
}

func (s *AuthGraphqlTestSuite) Test1_Register() {
	data := s.exec(`
		mutation RegisterUser($input: RegisterInput!) {
			registerUser(input: $input) {
				status
				message
				data { id firstname lastname email }
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"firstname":        "Auth",
			"lastname":         "Graphql",
			"email":            s.email,
			"password":         s.password,
			"confirm_password": s.password,
		},
	})
	registered := data["registerUser"].(map[string]interface{})
	s.Equal("success", registered["status"])
	user := registered["data"].(map[string]interface{})
	s.Equal(s.email, user["email"])
	s.userID = int(user["id"].(float64))
	s.Require().NotZero(s.userID)
}

func (s *AuthGraphqlTestSuite) Test2_Login() {
	data := s.exec(`
		mutation LoginUser($input: LoginInput!) {
			loginUser(input: $input) { status message data { access_token refresh_token } }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"email": s.email, "password": s.password},
	})
	login := data["loginUser"].(map[string]interface{})
	s.Equal("success", login["status"])
	tokens := login["data"].(map[string]interface{})
	s.NotEmpty(tokens["access_token"])
	s.accessToken = tokens["access_token"].(string)
	s.refreshToken = tokens["refresh_token"].(string)
}

func (s *AuthGraphqlTestSuite) Test3_GetMe() {
	s.Require().NotZero(s.userID)
	s.Require().NotEmpty(s.accessToken)

	// getMe resolves the caller from the request context, exactly like the
	// gateway's AuthMiddleware does for authenticated traffic.
	authed := graphtest.WithUser(s.handler, s.userID)
	resp, err := graphtest.ExecuteGraphQL(authed, `
		query GetMe($input: GetMeInput!) {
			getMe(input: $input) { status data { id firstname lastname email } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"access_token": s.accessToken}}, "")
	s.Require().NoError(err)
	s.Require().Empty(resp.Errors, "graphql errors: %+v", resp.Errors)
	me := resp.Data["getMe"].(map[string]interface{})
	s.Equal("success", me["status"])
	s.Equal(s.email, me["data"].(map[string]interface{})["email"])
}

func (s *AuthGraphqlTestSuite) Test4_RefreshToken() {
	s.Require().NotEmpty(s.refreshToken)
	data := s.exec(`
		mutation RefreshToken($input: RefreshTokenInput!) {
			refreshToken(input: $input) { status data { access_token refresh_token } }
		}
	`, map[string]interface{}{"input": map[string]interface{}{"refresh_token": s.refreshToken}})
	refreshed := data["refreshToken"].(map[string]interface{})
	s.Equal("success", refreshed["status"])
	tokens := refreshed["data"].(map[string]interface{})
	s.NotEmpty(tokens["access_token"])
	s.NotEmpty(tokens["refresh_token"])
}

func (s *AuthGraphqlTestSuite) Test5_LoginLockout() {
	email := "locked.graphql@example.com"
	password := "correctpassword"

	s.exec(`
		mutation RegisterUser($input: RegisterInput!) {
			registerUser(input: $input) { status }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"firstname":        "Locked",
			"lastname":         "Graphql",
			"email":            email,
			"password":         password,
			"confirm_password": password,
		},
	})

	loginQuery := `
		mutation LoginUser($input: LoginInput!) {
			loginUser(input: $input) { status }
		}
	`
	wrong := map[string]interface{}{
		"input": map[string]interface{}{"email": email, "password": "wrongpassword"},
	}

	// Fail login 5 times.
	for i := 0; i < 5; i++ {
		resp, err := graphtest.ExecuteGraphQL(s.handler, loginQuery, wrong, "")
		s.Require().NoError(err)
		s.Require().NotEmpty(resp.Errors, "failed login attempt %d must return an error", i+1)
	}

	// 6th attempt must report the account is locked.
	resp, err := graphtest.ExecuteGraphQL(s.handler, loginQuery, wrong, "")
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.Errors, "locked account must return an error")
	s.Contains(strings.ToLower(resp.Errors[0].Message), "locked")
}

func TestAuthGraphqlSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(AuthGraphqlTestSuite))
}
