package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func LoadConfig(configPath *string) (*Config, error) {

	v := viper.New()
	cfgName := "dev"

	// Default config search paths
	if configPath != nil && *configPath != "" {
		v.SetConfigFile(filepath.Join(*configPath, cfgName+".yaml"))
	} else {
		overrideFile := filepath.Join("pkg", "config", "override", cfgName+".yaml")
		defaultFile := filepath.Join("pkg", "config", cfgName+".yaml")

		// Prefer override if present
		if _, err := os.Stat(overrideFile); err == nil {
			v.SetConfigFile(overrideFile)
		} else {
			v.SetConfigFile(defaultFile)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
