package config

import (
	"github.com/go-playground/validator/v10"
)

func ValidateConfig(cfg *Config) error {
	validate := validator.New()

	return validate.Struct(cfg)
}
