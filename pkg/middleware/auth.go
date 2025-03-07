package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
	"wealth-warden/pkg/config"
)

var (
	ErrTokenExpired = errors.New("token has expired")
)

type WebClientUserClaim struct {
	UserID string `json:"ID"`
	jwt.RegisteredClaims
}

func refreshAccessToken(c *gin.Context, refreshClaims *WebClientUserClaim) error {

	cfg := config.LoadConfig()
	userId, err := DecodeWebClientUserID(refreshClaims.UserID)
	if err != nil {
		return err
	}

	accessToken, err := GenerateToken("access", time.Now().Add(15*time.Minute), userId)
	if err != nil {
		return err
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access", accessToken, 60*15, "/", cfg.WebClientDomain, cfg.Release, true)

	return nil
}

func WebClientAuthentication(releaseMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error

		//if webClientIdentifier := c.GetHeader("wealth-warden-client"); webClientIdentifier != "true" {
		//	c.Next()
		//	return
		//}

		// Skip JWT authentication in development mode
		// TODO: REMOVE THIS AFTER DEVELOPMENT
		if releaseMode == false {
			c.Next()
			return
		}

		accessToken, accessError := c.Cookie("access")
		if accessError != nil {

			refreshToken, refreshError := c.Cookie("refresh")
			if refreshError != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthenticated")
				return
			}

			refreshClaims, err2 := DecodeWebClientToken(refreshToken, "refresh")
			if err2 != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthenticated")
				return
			}

			// Perform token refresh asynchronously
			go func() {
				if err := refreshAccessToken(c, refreshClaims); err != nil {
					fmt.Println("Error refreshing token:", err)
				}
			}()

			c.Next()
			return
		}
		_, err = DecodeWebClientToken(accessToken, "access")
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				refreshToken, _ := c.Cookie("refresh")
				refreshClaims, err2 := DecodeWebClientToken(refreshToken, "refresh")
				if err2 != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthenticated")
					return
				}

				// Perform token refresh asynchronously
				go func() {
					if err := refreshAccessToken(c, refreshClaims); err != nil {
						fmt.Println("Error refreshing token:", err)
					}
				}()
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, "Unauthenticated")
				return
			}
		}

		c.Next()
	}
}

func encodeWebClientUserID(userID uint) (string, error) {
	key := os.Getenv("JWT_WEB_CLIENT_ENCODE_ID")
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

func DecodeWebClientUserID(encodedString string) (uint, error) {
	key := os.Getenv("JWT_WEB_CLIENT_ENCODE_ID")
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

	// Ensure the parsed value is non-negative before converting to uint
	if intUserID < 0 {
		return 0, fmt.Errorf("invalid user ID: negative value")
	}

	// Convert int64 to uint
	return uint(intUserID), nil
}

func GenerateToken(tokenType string, expiration time.Time, userID uint) (string, error) {
	var jwtKey []byte
	issuedAt := time.Now()

	// Select the appropriate JWT secret based on token type
	switch tokenType {
	case "access":
		jwtKey = []byte(os.Getenv("JWT_WEB_CLIENT_ACCESS"))
	case "refresh":
		jwtKey = []byte(os.Getenv("JWT_WEB_CLIENT_REFRESH"))
	default:
		return "", fmt.Errorf("unsupported token type: %s", tokenType)
	}

	// Encrypt the user ID before embedding it into the token
	encryptedUserID, err := encodeWebClientUserID(userID)
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
func GenerateLoginTokens(userID uint, rememberMe bool) (string, string, error) {

	var expiresAt time.Time
	if rememberMe {
		expiresAt = time.Now().Add(1 * 24 * time.Hour) // Token expires in 1 day
	} else {
		expiresAt = time.Now().Add(1 * time.Hour) // Token expires in 1 hour
	}

	accessToken, err := GenerateToken("access", time.Now().Add(15*time.Minute), userID)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := GenerateToken("refresh", expiresAt, userID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func DecodeWebClientToken(tokenString string, cookieType string) (*WebClientUserClaim, error) {
	var secret string

	switch cookieType {
	case "access":
		secret = os.Getenv("JWT_WEB_CLIENT_ACCESS")
	case "refresh":
		secret = os.Getenv("JWT_WEB_CLIENT_REFRESH")
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
