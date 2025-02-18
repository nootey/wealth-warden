package validators

import (
	"github.com/go-playground/validator/v10"
)

type GoValidator struct {
	Validator *validator.Validate
}

func NewValidator() *GoValidator {
	return &GoValidator{Validator: validator.New()}
}

func (cv *GoValidator) ValidateStruct(data interface{}) error {
	return cv.Validator.Struct(data)
}
