package config

import (
	"github.com/spf13/viper"
	"path/filepath"
)

func LoadConfig(configPath *string) (*Config, error) {

	// Default config search paths
	if configPath == nil {
		overridePath := filepath.Join("pkg", "config", "override")
		defaultPath := filepath.Join("pkg", "config")

		viper.AddConfigPath(overridePath)
		viper.AddConfigPath(defaultPath)
	} else {
		viper.AddConfigPath(*configPath)
	}

	viper.SetConfigName("dev")
	viper.SetConfigType("yaml")

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
