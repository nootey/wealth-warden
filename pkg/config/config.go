package config

import (
	"github.com/spf13/viper"
	"path/filepath"
	"wealth-warden/internal/models"
)

func LoadConfig(configPath *string) (*models.Config, error) {

	// Default config path
	if configPath == nil {
		path := filepath.Join("pkg", "config")
		configPath = &path
	}

	// Load YAML config via Viper
	viper.SetConfigName("environment")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(*configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg models.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
