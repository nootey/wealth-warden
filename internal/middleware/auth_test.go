package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type AuthMiddlewareTestSuite struct {
	tests.ServiceIntegrationSuite
	router     *gin.Engine
	middleware *middleware.WebClientMiddleware
	cfg        *config.Config
}

func (suite *AuthMiddlewareTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}
	suite.cfg = cfg

	logger, _ := zap.NewDevelopment()
	suite.middleware = middleware.NewWebClientMiddleware(cfg, logger, 2*time.Second, 5*time.Second, 1*time.Minute)

	suite.router = gin.New()
}

func TestAuthMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(AuthMiddlewareTestSuite))
}

func (suite *AuthMiddlewareTestSuite) decodeToken(token, tokenType string) (*middleware.WebClientUserClaim, error) {
	var secret string
	switch tokenType {
	case "access":
		secret = suite.cfg.JWT.WebClientAccess
	case "refresh":
		secret = suite.cfg.JWT.WebClientRefresh
	default:
		return nil, fmt.Errorf("unknown token type")
	}

	claims := &middleware.WebClientUserClaim{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	return claims, err
}

func (suite *AuthMiddlewareTestSuite) TestGenerateLoginTokens_Success() {
	userID := int64(123)

	accessToken, refreshToken, err := suite.middleware.GenerateLoginTokens(userID, false)

	suite.NoError(err)
	suite.NotEmpty(accessToken)
	suite.NotEmpty(refreshToken)
	suite.NotEqual(accessToken, refreshToken)

	// Verify access token expiration (2 seconds - from SetupTest)
	accessClaims, err := suite.decodeToken(accessToken, "access")
	suite.NoError(err)
	suite.WithinDuration(time.Now().Add(2*time.Second), accessClaims.ExpiresAt.Time, 2*time.Second)

	// Verify refresh token expiration (5 seconds for rememberMe=false - from SetupTest)
	refreshClaims, err := suite.decodeToken(refreshToken, "refresh")
	suite.NoError(err)
	suite.WithinDuration(time.Now().Add(5*time.Second), refreshClaims.ExpiresAt.Time, 2*time.Second)
}

func (suite *AuthMiddlewareTestSuite) TestGenerateLoginTokens_RememberMe() {
	userID := int64(123)

	accessToken, refreshToken, err := suite.middleware.GenerateLoginTokens(userID, true)

	suite.NoError(err)
	suite.NotEmpty(accessToken)
	suite.NotEmpty(refreshToken)

	// Verify refresh token expiration (1 minute for rememberMe=true - from SetupTest)
	refreshClaims, err := suite.decodeToken(refreshToken, "refresh")
	suite.NoError(err)
	suite.WithinDuration(time.Now().Add(1*time.Minute), refreshClaims.ExpiresAt.Time, 2*time.Second)
}

func (suite *AuthMiddlewareTestSuite) TestWebClientAuthentication_AccessTokenRotation() {
	authSvc := suite.TC.App.AuthService
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}

	// Login to get a real user
	user, err := authSvc.ValidateLogin(
		context.Background(),
		cfg.Seed.SuperAdminEmail,
		cfg.Seed.SuperAdminPassword,
		"test-agent",
		"127.0.0.1",
	)
	suite.NoError(err)
	suite.NotNil(user)

	// Generate tokens with short TTLs
	accessToken, refreshToken, err := suite.middleware.GenerateLoginTokens(user.ID, false)
	suite.NoError(err)

	// Setup test route with authentication middleware
	suite.router.Use(suite.middleware.WebClientAuthentication())
	suite.router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"user_id": c.GetInt64("user_id")})
	})

	fmt.Println("sleeping 4s to test access token rotation ...")
	time.Sleep(4 * time.Second)

	// Verify token is expired after sleep
	_, err2 := suite.decodeToken(accessToken, "access")
	suite.Error(err2, "Token should be expired")

	// Make request with expired access token but valid refresh token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access", Value: accessToken})
	req.AddCookie(&http.Cookie{Name: "refresh", Value: refreshToken})
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// Should succeed
	suite.Equal(http.StatusOK, w.Code)

	// Check Set-Cookie header directly
	setCookieHeaders := w.Header()["Set-Cookie"]
	suite.NotEmpty(setCookieHeaders, "Should have Set-Cookie headers")

	// Verify access cookie was rotated
	var foundAccessCookie bool
	for _, header := range setCookieHeaders {
		if strings.Contains(header, "access=") {
			foundAccessCookie = true
			suite.NotContains(header, accessToken, "Should be a new access token")
			break
		}
	}
	suite.True(foundAccessCookie, "New access cookie should be issued")
}

func (suite *AuthMiddlewareTestSuite) TestWebClientAuthentication_RefreshTokenExpired() {
	authSvc := suite.TC.App.AuthService
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}

	user, err := authSvc.ValidateLogin(
		context.Background(),
		cfg.Seed.SuperAdminEmail,
		cfg.Seed.SuperAdminPassword,
		"test-agent",
		"127.0.0.1",
	)
	suite.NoError(err)

	// Generate tokens with short TTLs
	accessToken, refreshToken, err := suite.middleware.GenerateLoginTokens(user.ID, false)
	suite.NoError(err)

	// Setup test route
	suite.router.Use(suite.middleware.WebClientAuthentication())
	suite.router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"user_id": c.GetInt64("user_id")})
	})

	fmt.Println("sleeping 7s to test refresh token rotation ...")
	time.Sleep(7 * time.Second)

	// Make request with both expired tokens
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access", Value: accessToken})
	req.AddCookie(&http.Cookie{Name: "refresh", Value: refreshToken})
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// Should return unauthorized
	suite.Equal(http.StatusUnauthorized, w.Code)
}
