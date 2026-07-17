package config

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Defaults double as viper's key registry: AutomaticEnv only resolves keys viper knows about.
func setDefaults(v *viper.Viper) {
	v.SetDefault("host", "0.0.0.0")
	v.SetDefault("release", false)
	v.SetDefault("finance_api_base_url", "")

	v.SetDefault("http_server.port", "2000")
	v.SetDefault("http_server.request_timeout", 60)

	v.SetDefault("web_client.domain", "localhost")
	v.SetDefault("web_client.port", "5000")

	v.SetDefault("postgres.host", "db")
	v.SetDefault("postgres.user", "postgres")
	v.SetDefault("postgres.password", "postgres")
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.db", "wealth_warden")

	v.SetDefault("redis.host", "redis")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("session.ttl_hours", 24)
	v.SetDefault("session.remember_me_ttl_hours", 720)
	v.SetDefault("session.max_lifetime_hours", 2160)

	v.SetDefault("cors.allowed_origins", []string{"http://localhost:5000", "http://app:5000"})
	v.SetDefault("cors.wildcard_suffixes", []string{})
	v.SetDefault("cors.allowed_schemes", []string{"http"})

	v.SetDefault("mailer.host", "")
	v.SetDefault("mailer.port", 587)
	v.SetDefault("mailer.username", "")
	v.SetDefault("mailer.password", "")

	v.SetDefault("seed.super_admin_email", "admin@wealth.warden")
	v.SetDefault("seed.super_admin_password", "password")
	v.SetDefault("seed.member_user_email", "")
	v.SetDefault("seed.member_user_password", "")

	v.SetDefault("scheduler.concurrent_workers", 5)
	v.SetDefault("scheduler.immediate_jobs", []string{})

	v.SetDefault("otel.otlp_endpoint", "tempo:4317")
	v.SetDefault("otel.service_name", "wealth-warden")

	v.SetDefault("queue.workers", 1)
	v.SetDefault("queue.max_attempts", 5)
	v.SetDefault("queue.poll_interval_ms", 1000)
	v.SetDefault("queue.retry_initial_backoff_sec", 60)
	v.SetDefault("queue.retry_subsequent_backoff_sec", 120)
	v.SetDefault("queue.visibility_timeout_sec", 900)
}

func LoadConfig(configPath *string, configName ...string) (*Config, error) {
	v := viper.New()
	setDefaults(v)

	// Every key is overridable via env: dots become underscores (postgres.host -> POSTGRES_HOST)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("otel.otlp_endpoint", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_OTLP_ENDPOINT")

	cfgName := "dev"
	if len(configName) > 0 && configName[0] != "" {
		cfgName = configName[0]
	}

	explicitPath := configPath != nil && *configPath != ""
	if explicitPath {
		v.SetConfigFile(filepath.Join(*configPath, cfgName+".yaml"))
	} else {
		v.SetConfigFile(filepath.Join("pkg", "config", "override", cfgName+".yaml"))
	}

	if err := v.ReadInConfig(); err != nil {
		// The default override file is optional, but an explicitly requested or malformed one is not
		if explicitPath || !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
