package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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
	v, ok := c.Get("user_id")
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

func GenerateHttpReleaseLink(cfg *config.Config) string {
	domain := cfg.WebClient.Domain
	port := cfg.HttpServer.Port
	version := cfg.Api.Version
	production := cfg.Release
	prefix := "http://"

	if production {
		prefix = "https://"
		port = ""
	}

	base := fmt.Sprintf("%s%s", prefix, domain)
	if port != "" {
		base += ":" + port
	}

	return fmt.Sprintf("%s/api/v%s/", base, version)
}

func GenerateWebClientReleaseLink(cfg *config.Config, subdomain string) string {
	domain := cfg.WebClient.Domain
	port := cfg.WebClient.Port
	production := cfg.Release
	prefix := "http://"

	if production {
		prefix = "https://"
		port = ""
	}

	if subdomain != "" {
		subdomain += "/"
	}

	return fmt.Sprintf("%s%s:%s/%s", prefix, domain, port, subdomain)
}

func ValidatePasswordStrength(password string) (string, error) {

	const minLength = 8
	var (
		hasUpper   bool
		hasNumber  bool
		hasSpecial bool
	)

	sanitized := strings.TrimSpace(password)

	if len(sanitized) < minLength {
		return "", fmt.Errorf("password must be at least %d characters long", minLength)
	}

	for _, char := range sanitized {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char), unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	var missing []string

	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasNumber {
		missing = append(missing, "number")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return "", errors.New("password must contain: " + strings.Join(missing, ", "))
	}

	return sanitized, nil
}

func GenerateRandomToken(length int) (string, error) {
	if length > 32 {
		return "", errors.New("max supported length is 32 characters")
	}
	randomBytes := make([]byte, length*2)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode the random bytes to base64
	token := base64.URLEncoding.EncodeToString(randomBytes)

	// Trim padding characters '=' from the end
	token = strings.TrimRight(token, "=")

	// Trim to the desired length
	return token[:length], nil
}

func UnwrapToken(token *models.Token, key string) (interface{}, error) {
	if token.Data == nil {
		return nil, fmt.Errorf("token data is nil")
	}
	val, ok := token.Data[key]
	if !ok {
		return nil, fmt.Errorf("missing key %q in token data", key)
	}
	return val, nil
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func EmailToName(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 0 {
		return ""
	}
	local := parts[0]

	// cut off at the first digit
	re := regexp.MustCompile(`[0-9].*`)
	local = re.ReplaceAllString(local, "")

	// split by . or - and take the first token
	separators := regexp.MustCompile(`[.\-]`)
	tokens := separators.Split(local, -1)
	if len(tokens) > 0 {
		local = tokens[0]
	}

	return capitalize(local)
}
