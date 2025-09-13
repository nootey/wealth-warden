package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"reflect"
	"strings"
	"unicode"
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

func cleanString(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsControl(r) && !unicode.IsSpace(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func SanitizeStruct(s interface{}) error {
	v := reflect.ValueOf(s)

	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("SanitizeStruct expects a pointer to a struct")
	}

	v = v.Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		if !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			trimmed := cleanString(field.String())
			field.SetString(trimmed)

		case reflect.Ptr:
			if field.Type().Elem().Kind() == reflect.String && !field.IsNil() {
				trimmed := cleanString(field.Elem().String())
				field.Elem().SetString(trimmed)
			}
		}
	}

	return nil
}

func GenerateSecureToken(nBytes int) (string, error) {
	bytes := make([]byte, nBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
