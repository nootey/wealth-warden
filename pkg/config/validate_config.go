package config

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

func ValidateConfig(cfg *Config) error {
	validate := validator.New()

	if err := validate.Struct(cfg); err != nil {
		return err
	}

	if cfg.Release &&
		(cfg.JWT.WebClientAccess == defaultJWTAccess ||
			cfg.JWT.WebClientRefresh == defaultJWTRefresh ||
			cfg.JWT.WebClientEncodeID == defaultJWTEncodeID) {
		return errors.New("release mode requires non-default jwt secrets")
	}

	return nil
}
