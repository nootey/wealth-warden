package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"wealth-warden/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(content), 0o600))
	return dir
}

func TestLoadConfig_MissingDefaultFileFallsBackToDefaults(t *testing.T) {
	cfg, err := config.LoadConfig(nil)

	require.NoError(t, err)
	assert.Equal(t, 5, cfg.Queue.MaxAttempts)
	assert.Equal(t, "wealth-warden", cfg.Otel.ServiceName)
}

func TestLoadConfig_MissingExplicitFileErrors(t *testing.T) {
	dir := t.TempDir()

	_, err := config.LoadConfig(&dir, "test")

	assert.Error(t, err)
}

func TestLoadConfig_MalformedYamlErrors(t *testing.T) {
	dir := writeConfig(t, "postgres: [not: valid")

	_, err := config.LoadConfig(&dir, "test")

	assert.Error(t, err)
}

func TestLoadConfig_EnvOverridesYamlAndDefaults(t *testing.T) {
	dir := writeConfig(t, `
postgres:
  host: "yamlhost"
  port: 5432
`)
	t.Setenv("POSTGRES_HOST", "envhost")
	t.Setenv("POSTGRES_PORT", "6543")
	t.Setenv("JWT_WEB_CLIENT_ACCESS", "env-access-secret")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example,https://b.example")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")

	cfg, err := config.LoadConfig(&dir, "test")

	require.NoError(t, err)
	assert.Equal(t, "envhost", cfg.Postgres.Host)
	assert.Equal(t, 6543, cfg.Postgres.Port)
	assert.Equal(t, "env-access-secret", cfg.JWT.WebClientAccess)
	assert.Equal(t, []string{"https://a.example", "https://b.example"}, cfg.CORS.AllowedOrigins)
	assert.Equal(t, "collector:4317", cfg.Otel.OTLPEndpoint)
}

func TestLoadConfig_YamlOverridesDefaults(t *testing.T) {
	dir := writeConfig(t, `
postgres:
  host: "yamlhost"
cors:
  allowed_origins:
    - "https://app.example"
`)

	cfg, err := config.LoadConfig(&dir, "test")

	require.NoError(t, err)
	assert.Equal(t, "yamlhost", cfg.Postgres.Host)
	assert.Equal(t, []string{"https://app.example"}, cfg.CORS.AllowedOrigins)
	assert.Equal(t, "postgres", cfg.Postgres.User)
}

func TestLoadConfig_ReleaseRejectsDefaultJWTSecrets(t *testing.T) {
	dir := writeConfig(t, "release: true")

	_, err := config.LoadConfig(&dir, "test")

	assert.ErrorContains(t, err, "non-default jwt secrets")
}

func TestLoadConfig_ReleaseAcceptsCustomJWTSecrets(t *testing.T) {
	dir := writeConfig(t, `
release: true
jwt:
  web_client_access: "custom-access"
  web_client_refresh: "custom-refresh"
  web_client_encode_id: "custom-encode-id-32-chars-long!!"
`)

	_, err := config.LoadConfig(&dir, "test")

	assert.NoError(t, err)
}

func TestLoadConfig_RejectsWrongLengthEncodeID(t *testing.T) {
	dir := writeConfig(t, `
jwt:
  web_client_encode_id: "too-short"
`)

	_, err := config.LoadConfig(&dir, "test")

	assert.Error(t, err)
}
