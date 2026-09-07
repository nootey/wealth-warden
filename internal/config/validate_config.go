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

	if cfg.Release && cfg.Redis.Password == "" {
		return errors.New("release mode requires a redis password")
	}

	return nil
}
