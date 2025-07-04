package config

import (
	"errors"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"wealth-warden/internal/models"
)

func LoadConfig(configPath *string) (*models.Config, error) {

	// Default config path
	if configPath == nil {
		path := filepath.Join("pkg", "config")
		configPath = &path
	}

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		return nil, errors.New("no .env file found")
	}

	// Load YAML config via Viper
	viper.SetConfigName("settings")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(*configPath)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg models.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Load sensitive info from environment
	loadEnvSecrets(&cfg)

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadEnvSecrets(cfg *models.Config) {
	cfg.MySQL.User = os.Getenv("MYSQL_USER")
	cfg.MySQL.Password = os.Getenv("MYSQL_PASSWORD")
	cfg.MySQL.Database = os.Getenv("MYSQL_DATABASE")

	if host := os.Getenv("HOST"); host != "" {
		cfg.MySQL.Host = host
	}

	if portStr := os.Getenv("MYSQL_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.MySQL.Port = port
		}
	}

	cfg.JWT = models.JWTConfig{
		WebClientAccess:   os.Getenv("JWT_WEB_CLIENT_ACCESS"),
		WebClientRefresh:  os.Getenv("JWT_WEB_CLIENT_REFRESH"),
		WebClientEncodeID: os.Getenv("JWT_WEB_CLIENT_ENCODE_ID"),
	}
}
