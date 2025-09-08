package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

func HashAndSaltPassword(password string) (string, error) {
	// Generate a hashed and salted password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func DetermineServiceSource(userAgent string) string {
	ua := strings.ToLower(userAgent)

	switch {
	case strings.Contains(ua, "postman"):
		return "Postman"
	case strings.Contains(ua, "curl"):
		return "cURL"
	case strings.Contains(ua, "python-requests"):
		return "Python Script"
	case strings.Contains(ua, "node-fetch"), strings.Contains(ua, "axios"):
		return "Node.js Client"
	case strings.Contains(ua, "mozilla"):
		return "Web Client"
	case strings.Contains(ua, "mobile") && strings.Contains(ua, "safari"):
		return "Mobile Browser"
	case strings.Contains(ua, "android"):
		return "Android App"
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"):
		return "iOS App"
	default:
		return "Unknown"
	}
}

func UserIDFromCtx(c *gin.Context) (int64, error) {
	v, ok := c.Get("userID")
	if !ok {
		return 0, errors.New("unauthenticated")
	}
	id, ok := v.(int64)
	if !ok {
		return 0, errors.New("invalid user id type")
	}
	return id, nil
}
