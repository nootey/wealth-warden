package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/constants"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var (
	ErrTokenExpired = errors.New("token has expired")
)

type WebClientUserClaim struct {
	UserID string `json:"ID"`
	jwt.RegisteredClaims
}

type WebClientMiddleware struct {
	config *config.Config
	logger *zap.Logger
}

func NewWebClientMiddleware(cfg *config.Config, logger *zap.Logger) *WebClientMiddleware {
	return &WebClientMiddleware{
		config: cfg,
		logger: logger,
	}
}

func (m *WebClientMiddleware) WebClientAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try access
		access, _ := c.Cookie("access")
		if access != "" {
			claims, err := m.DecodeWebClientToken(access, "access")
			if err == nil {
				userID, err := m.DecodeWebClientUserID(claims.UserID)
				if err == nil {
					c.Set("user_id", userID)
					c.Next()
					return
				}
			}
		}

		// If access missing/expired, try refresh
		refresh, _ := c.Cookie("refresh")
		if refresh == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthenticated")
			return
		}
		rClaims, err := m.DecodeWebClientToken(refresh, "refresh")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthenticated")
			return
		}

		// Check refresh token server-side  and rotation state here

		userID, err := m.DecodeWebClientUserID(rClaims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthenticated")
			return
		}

		// 4) Issue new access (and rotate refresh if you implement rotation)
		if err := m.issueAccessCookie(c, userID); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthenticated")
			return
		}

		// Optionally rotate refresh cookie here as well
		// if rotated:
		//     m.issueRefreshCookie(c, userID, newRefreshToken)

		c.Set("user_id", userID)
		c.Next()
	}
}

func (m *WebClientMiddleware) issueAccessCookie(c *gin.Context, userID int64) error {

	accessExp := time.Now().Add(constants.AccessCookieTTL)
	token, err := m.GenerateToken("access", accessExp, userID)
	if err != nil {
		return err
	}

	secure := m.config.Release
	c.SetSameSite(http.SameSiteLaxMode)
	maxAge := int(time.Until(accessExp).Seconds())
	c.SetCookie("access", token, maxAge, "/", m.config.WebClient.Domain, secure, true)
	return nil
}

func (m *WebClientMiddleware) issueRefreshCookie(c *gin.Context, userID int64, remember bool) error {

	refreshExp := time.Now().Add(map[bool]time.Duration{
		true:  constants.RefreshCookieTTLLong,
		false: constants.RefreshCookieTTLShort,
	}[remember])

	token, err := m.GenerateToken("refresh", refreshExp, userID)
	if err != nil {
		return err
	}

	secure := m.config.Release
	c.SetSameSite(http.SameSiteLaxMode)
	maxAge := int(time.Until(refreshExp).Seconds())
	c.SetCookie("refresh", token, maxAge, "/", m.config.WebClient.Domain, secure, true)
	return nil
}

func (m *WebClientMiddleware) encodeWebClientUserID(userID int64) (string, error) {
	key := m.config.JWT.WebClientEncodeID
	if len(key) != 32 {
		return "", fmt.Errorf("encryption key must be 32 bytes long for AES-256")
	}

	userIDString := strconv.FormatInt(int64(userID), 10)

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(userIDString), nil)
	ciphertext = append(nonce, ciphertext...) // Prepend nonce to ciphertext

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encoded, nil
}

func (m *WebClientMiddleware) DecodeWebClientUserID(encodedString string) (int64, error) {
	key := m.config.JWT.WebClientEncodeID
	if len(key) != 32 {
		return 0, fmt.Errorf("encryption key must be 32 bytes long for AES-256")
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(encodedString)
	if err != nil {
		return 0, err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return 0, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return 0, err
	}

	nonceSize := gcm.NonceSize()
	if len(decodedBytes) < nonceSize {
		return 0, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := decodedBytes[:nonceSize], decodedBytes[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, err
	}

	intUserID, err := strconv.ParseInt(string(plaintext), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user ID: %v", err)
	}

	return intUserID, nil
}

func (m *WebClientMiddleware) GenerateToken(tokenType string, expiration time.Time, userID int64) (string, error) {
	var jwtKey []byte
	issuedAt := time.Now()

	// Select the appropriate JWT secret based on token type
	switch tokenType {
	case "access":
		jwtKey = []byte(m.config.JWT.WebClientAccess)
	case "refresh":
		jwtKey = []byte(m.config.JWT.WebClientRefresh)
	default:
		return "", fmt.Errorf("unsupported token type: %s", tokenType)
	}

	// Encrypt the user ID before embedding it into the token
	encryptedUserID, err := m.encodeWebClientUserID(userID)
	if err != nil {
		return "", err
	}

	// Define the JWT claims
	claims := WebClientUserClaim{
		UserID: encryptedUserID, // Store the encrypted user ID
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			Issuer:    "wealth-warden-server",
		},
	}

	// Create the token and sign it
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (m *WebClientMiddleware) GenerateLoginTokens(userID int64, rememberMe bool) (string, string, error) {

	var expiresAt time.Time
	if rememberMe {
		expiresAt = time.Now().Add(1 * 24 * time.Hour) // Token expires in 1 day
	} else {
		expiresAt = time.Now().Add(1 * time.Hour) // Token expires in 1 hour
	}

	accessToken, err := m.GenerateToken("access", time.Now().Add(15*time.Minute), userID)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := m.GenerateToken("refresh", expiresAt, userID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (m *WebClientMiddleware) DecodeWebClientToken(tokenString string, cookieType string) (*WebClientUserClaim, error) {
	var secret string

	switch cookieType {
	case "access":
		secret = m.config.JWT.WebClientAccess
	case "refresh":
		secret = m.config.JWT.WebClientRefresh
	default:
		return nil, fmt.Errorf("unknown cookieType: %s", cookieType)
	}

	secretKey := []byte(secret)

	claims := &WebClientUserClaim{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	switch {
	case token.Valid:
		return claims, nil
	case errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet):
		return nil, ErrTokenExpired
	default:
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (m *WebClientMiddleware) ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process request

		// After request
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				m.logger.Info("HTTP error",
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("client_ip", c.ClientIP()),
					zap.Int("status_code", c.Writer.Status()),
					zap.Error(err),
				)
			}
		}
	}
}
