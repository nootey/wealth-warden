package config

import (
	"github.com/go-playground/validator/v10"
	"wealth-warden/internal/models"
)

func ValidateConfig(cfg *models.Config) error {
	validate := validator.New()

	// Register custom validators here if needed.

	return validate.Struct(cfg)
}
