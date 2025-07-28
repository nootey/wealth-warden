package config

import (
	"github.com/spf13/viper"
	"path/filepath"
)

func LoadConfig(configPath *string) (*Config, error) {

	// Default config path
	if configPath == nil {
		path := filepath.Join("pkg", "config")
		configPath = &path
	}

	// Load YAML config via Viper
	viper.SetConfigName(filepath.Join("configurable", "environment"))
	viper.SetConfigType("yaml")
	viper.AddConfigPath(*configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
