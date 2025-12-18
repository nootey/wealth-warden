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

	// Create an already-expired access token and valid refresh token
	expiredAccessToken := suite.createWebClientTokenWithExpiry(user.ID, "access", -5*time.Second)
	validRefreshToken := suite.createWebClientTokenWithExpiry(user.ID, "refresh", 10*time.Minute)

	// Setup test route with authentication middleware
	suite.router.Use(suite.middleware.WebClientAuthentication())
	suite.router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"user_id": c.GetInt64("user_id")})
	})

	// Make request with expired access token but valid refresh token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access", Value: expiredAccessToken})
	req.AddCookie(&http.Cookie{Name: "refresh", Value: validRefreshToken})
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
			suite.NotContains(header, expiredAccessToken, "Should be a new access token")
			break
		}
	}
	suite.True(foundAccessCookie, "New access cookie should be issued")
}

func (suite *AuthMiddlewareTestSuite) createWebClientTokenWithExpiry(userID int64, tokenType string, expiryOffset time.Duration) string {
	cfg := suite.TC.App.Config

	var jwtKey []byte
	switch tokenType {
	case "access":
		jwtKey = []byte(cfg.JWT.WebClientAccess)
	case "refresh":
		jwtKey = []byte(cfg.JWT.WebClientRefresh)
	default:
		suite.T().Fatalf("unsupported token type: %s", tokenType)
	}

	encryptedUserID, err := suite.middleware.EncodeWebClientUserID(userID)
	if err != nil {
		suite.T().Fatal(err)
	}

	claims := &middleware.WebClientUserClaim{
		UserID: encryptedUserID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryOffset)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "wealth-warden-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		suite.T().Fatal(err)
	}

	return tokenString
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

	// Create both tokens as already expired
	expiredAccessToken := suite.createWebClientTokenWithExpiry(user.ID, "access", -5*time.Second)
	expiredRefreshToken := suite.createWebClientTokenWithExpiry(user.ID, "refresh", -1*time.Second)

	// Setup test route
	suite.router.Use(suite.middleware.WebClientAuthentication())
	suite.router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"user_id": c.GetInt64("user_id")})
	})

	// Make request with both expired tokens
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access", Value: expiredAccessToken})
	req.AddCookie(&http.Cookie{Name: "refresh", Value: expiredRefreshToken})
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// Should return unauthorized
	suite.Equal(http.StatusUnauthorized, w.Code)
}
