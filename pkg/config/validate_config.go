package config

import (
	"github.com/go-playground/validator/v10"
)

func ValidateConfig(cfg *Config) error {
	validate := validator.New()

	// Register custom validators here if needed.

	return validate.Struct(cfg)
}
