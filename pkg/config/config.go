package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
)

func setDefaults() *Config {
	return &Config{
		Host:              "0.0.0.0",
		Release:           false,
		FinanceAPIBaseURL: "",
		HttpServer: HttpServerConfig{
			Port:       "2000",
			ReqTimeout: 60,
		},
		WebClient: WebClientConfig{
			Domain: "localhost",
			Port:   "5000",
		},
		Postgres: PostgresConfig{
			Host:     "db",
			User:     "postgres",
			Password: "postgres",
			Port:     5432,
			Database: "wealth_warden",
		},
		JWT: JWTConfig{
			WebClientAccess:   "O7yslMel&nR6",
			WebClientRefresh:  "M2tb,_R!X4w~",
			WebClientEncodeID: "Rjy6E*)Dz'UwWLPPk*47c0||o`-Oy<p/",
		},
		CORS: CorsConfig{
			AllowedOrigins:   []string{"http://localhost:5000", "http://app:5000"},
			WildcardSuffixes: []string{},
			AllowedSchemes:   []string{"http"},
		},
		Mailer: MailerConfig{
			Host:     "",
			Port:     587,
			Username: "",
			Password: "",
		},
		Seed: SeedConfig{
			SuperAdminEmail:    "admin@wealth.warden",
			SuperAdminPassword: "password",
			MemberUserEmail:    "",
			MemberUserPassword: "",
		},
	}
}

func LoadConfig(configPath *string, configName ...string) (*Config, error) {

	// Set all defaults first
	cfg := setDefaults()

	// Try to load override config (optional)
	v := viper.New()

	// Load env variables
	v.AutomaticEnv()

	// Load env variables
	v.AutomaticEnv()
	// Bind environment variables for nested postgres config
	v.BindEnv("postgres.host", "POSTGRES_HOST")
	v.BindEnv("postgres.user", "POSTGRES_USER")
	v.BindEnv("postgres.db", "POSTGRES_DB")
	v.BindEnv("postgres.port", "POSTGRES_PORT")
	v.BindEnv("postgres.password", "POSTGRES_PASSWORD")

	cfgName := "dev"
	if len(configName) > 0 && configName[0] != "" {
		cfgName = configName[0]
	}

	if configPath != nil && *configPath != "" {
		v.SetConfigFile(filepath.Join(*configPath, cfgName+".yaml"))
	} else {
		overrideFile := filepath.Join("pkg", "config", "override", cfgName+".yaml")
		v.SetConfigFile(overrideFile)
	}

	// If config file exists, unmarshal over defaults
	if err := v.ReadInConfig(); err == nil {
		if err := v.Unmarshal(cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables if they exist (env vars take precedence)
	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		cfg.Postgres.Host = host
	}
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		cfg.Postgres.User = user
	}
	if db := os.Getenv("POSTGRES_DB"); db != "" {
		cfg.Postgres.Database = db
	}
	if portStr := os.Getenv("POSTGRES_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Postgres.Port = port
		}
	}
	if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
		cfg.Postgres.Password = password
	}

	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
