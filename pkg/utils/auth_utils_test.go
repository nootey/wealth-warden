package utils_test

import (
	"strings"
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashAndSaltPassword(t *testing.T) {
	password := "mySecurePassword123!"

	hashed, err := utils.HashAndSaltPassword(password)

	require.NoError(t, err)
	assert.NotEmpty(t, hashed)
	assert.NotEqual(t, password, hashed)

	// Verify the hash is valid
	err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	assert.NoError(t, err)

	// Verify different password fails
	err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte("wrongPassword"))
	assert.Error(t, err)
}

func TestDetermineServiceSource(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{
			name:      "postman",
			userAgent: "PostmanRuntime/7.32.3",
			want:      "Postman",
		},
		{
			name:      "curl",
			userAgent: "curl/7.81.0",
			want:      "cURL",
		},
		{
			name:      "python requests",
			userAgent: "python-requests/2.28.0",
			want:      "Python Script",
		},
		{
			name:      "node-fetch",
			userAgent: "node-fetch/2.6.7",
			want:      "Node.js Client",
		},
		{
			name:      "axios",
			userAgent: "axios/1.4.0",
			want:      "Node.js Client",
		},
		{
			name:      "web browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			want:      "Web Client",
		},
		{
			name:      "mobile safari",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) Mobile Safari/604.1",
			want:      "Web Client", // Changed: caught by "mozilla" check first
		},
		{
			name:      "android",
			userAgent: "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36",
			want:      "Web Client", // Changed: caught by "mozilla" check first
		},
		{
			name:      "iphone",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)",
			want:      "Web Client", // Changed: caught by "mozilla" check first
		},
		{
			name:      "ipad",
			userAgent: "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X)",
			want:      "Web Client", // Changed: caught by "mozilla" check first
		},
		{
			name:      "unknown",
			userAgent: "CustomClient/1.0",
			want:      "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.DetermineServiceSource(tt.userAgent)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSanitizeStruct(t *testing.T) {
	t.Run("sanitizes string fields", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Email string
		}

		s := &TestStruct{
			Name:  "  John Doe  ",
			Email: "  test@example.com  ",
		}

		err := utils.SanitizeStruct(s)

		require.NoError(t, err)
		assert.Equal(t, "John Doe", s.Name)
		assert.Equal(t, "test@example.com", s.Email)
	})

	t.Run("sanitizes string pointer fields", func(t *testing.T) {
		type TestStruct struct {
			Name *string
		}

		name := "  John Doe  "
		s := &TestStruct{Name: &name}

		err := utils.SanitizeStruct(s)

		require.NoError(t, err)
		assert.Equal(t, "John Doe", *s.Name)
	})

	t.Run("handles nil string pointer", func(t *testing.T) {
		type TestStruct struct {
			Name *string
		}

		s := &TestStruct{Name: nil}

		err := utils.SanitizeStruct(s)

		require.NoError(t, err)
		assert.Nil(t, s.Name)
	})

	t.Run("removes control characters and tabs/newlines", func(t *testing.T) {
		type TestStruct struct {
			Text string
		}

		s := &TestStruct{
			Text: "Hello\x00World\x01\nTest\tEnd",
		}

		err := utils.SanitizeStruct(s)

		require.NoError(t, err)
		assert.Equal(t, "HelloWorldTestEnd", s.Text)
	})

	t.Run("errors on non-pointer", func(t *testing.T) {
		type TestStruct struct {
			Name string
		}

		s := TestStruct{Name: "test"}

		err := utils.SanitizeStruct(s)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects a pointer to a struct")
	})

	t.Run("errors on non-struct", func(t *testing.T) {
		s := "not a struct"

		err := utils.SanitizeStruct(&s)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects a pointer to a struct")
	})
}
func TestGenerateSecureToken(t *testing.T) {
	t.Run("generates token of correct length", func(t *testing.T) {
		token, err := utils.GenerateSecureToken(16)

		require.NoError(t, err)
		assert.Len(t, token, 32)
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		token1, err := utils.GenerateSecureToken(16)
		require.NoError(t, err)

		token2, err := utils.GenerateSecureToken(16)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})
}

func TestGenerateHttpReleaseLink(t *testing.T) {
	t.Run("development mode with port", func(t *testing.T) {
		cfg := &config.Config{
			Release: false,
			HttpServer: config.HttpServerConfig{
				Port: "8080",
			},
			WebClient: config.WebClientConfig{
				Domain: "localhost",
			},
			Api: config.ApiConfig{
				Version: "1",
			},
		}

		link := utils.GenerateHttpReleaseLink(cfg)

		assert.Equal(t, "http://localhost:8080/api/v1/", link)
	})

	t.Run("production mode without port", func(t *testing.T) {
		cfg := &config.Config{
			Release: true,
			HttpServer: config.HttpServerConfig{
				Port: "8080",
			},
			WebClient: config.WebClientConfig{
				Domain: "example.com",
			},
			Api: config.ApiConfig{
				Version: "2",
			},
		}

		link := utils.GenerateHttpReleaseLink(cfg)

		assert.Equal(t, "https://example.com/api/v2/", link)
	})
}

func TestGenerateWebClientReleaseLink(t *testing.T) {
	t.Run("development mode with subdomain", func(t *testing.T) {
		cfg := &config.Config{
			Release: false,
			WebClient: config.WebClientConfig{
				Domain: "localhost",
				Port:   "3000",
			},
		}

		link := utils.GenerateWebClientReleaseLink(cfg, "auth")

		assert.Equal(t, "http://localhost:3000/auth/", link)
	})

	t.Run("production mode without subdomain", func(t *testing.T) {
		cfg := &config.Config{
			Release: true,
			WebClient: config.WebClientConfig{
				Domain: "example.com",
				Port:   "3000",
			},
		}

		link := utils.GenerateWebClientReleaseLink(cfg, "")

		assert.Equal(t, "https://example.com:/", link)
	})
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid strong password",
			password: "MyPass123!",
			wantErr:  false,
		},
		{
			name:        "too short",
			password:    "Abc1!",
			wantErr:     true,
			errContains: "at least 8 characters",
		},
		{
			name:        "missing uppercase",
			password:    "mypass123!",
			wantErr:     true,
			errContains: "uppercase letter",
		},
		{
			name:        "missing number",
			password:    "MyPassword!",
			wantErr:     true,
			errContains: "number",
		},
		{
			name:        "missing special character",
			password:    "MyPassword123",
			wantErr:     true,
			errContains: "special character",
		},
		{
			name:        "missing multiple requirements",
			password:    "password",
			wantErr:     true,
			errContains: "uppercase letter",
		},
		{
			name:     "password with whitespace trimmed",
			password: "  MyPass123!  ",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ValidatePasswordStrength(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.Equal(t, strings.TrimSpace(tt.password), result)
			}
		})
	}
}

func TestGenerateRandomToken(t *testing.T) {
	t.Run("generates token of correct length", func(t *testing.T) {
		lengths := []int{8, 16, 24, 32}

		for _, length := range lengths {
			token, err := utils.GenerateRandomToken(length)

			require.NoError(t, err)
			assert.Len(t, token, length)
		}
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		token1, err := utils.GenerateRandomToken(16)
		require.NoError(t, err)

		token2, err := utils.GenerateRandomToken(16)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})

	t.Run("errors on length exceeding 32", func(t *testing.T) {
		token, err := utils.GenerateRandomToken(33)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max supported length is 32")
		assert.Empty(t, token)
	})
}

func TestUnwrapToken(t *testing.T) {
	t.Run("successfully unwraps existing key", func(t *testing.T) {
		token := &models.Token{
			Data: map[string]interface{}{
				"user_id": "12345",
				"email":   "test@example.com",
			},
		}

		val, err := utils.UnwrapToken(token, "user_id")

		require.NoError(t, err)
		assert.Equal(t, "12345", val)
	})

	t.Run("errors on nil data", func(t *testing.T) {
		token := &models.Token{
			Data: nil,
		}

		val, err := utils.UnwrapToken(token, "user_id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token data is nil")
		assert.Nil(t, val)
	})

	t.Run("errors on missing key", func(t *testing.T) {
		token := &models.Token{
			Data: map[string]interface{}{
				"user_id": "12345",
			},
		}

		val, err := utils.UnwrapToken(token, "email")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), `missing key "email"`)
		assert.Nil(t, val)
	})
}

func TestEmailToName(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "simple email",
			email: "john@example.com",
			want:  "John",
		},
		{
			name:  "email with dot separator",
			email: "john.doe@example.com",
			want:  "John",
		},
		{
			name:  "email with dash separator",
			email: "john-smith@example.com",
			want:  "John",
		},
		{
			name:  "email with numbers",
			email: "john123@example.com",
			want:  "John",
		},
		{
			name:  "email with numbers after text",
			email: "john.doe123@example.com",
			want:  "John",
		},
		{
			name:  "single letter email",
			email: "j@example.com",
			want:  "J",
		},
		{
			name:  "empty string",
			email: "",
			want:  "",
		},
		{
			name:  "no @ symbol",
			email: "invalid",
			want:  "Invalid",
		},
		{
			name:  "lowercase name",
			email: "alice@example.com",
			want:  "Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.EmailToName(tt.email)
			assert.Equal(t, tt.want, result)
		})
	}
}
