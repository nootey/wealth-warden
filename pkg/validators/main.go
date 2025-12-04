package validators

import (
	"github.com/go-playground/validator/v10"
)

type Validator interface {
	ValidateStruct(data interface{}) error
}

type GoValidator struct {
	Validator *validator.Validate
}

func NewValidator() *GoValidator {
	return &GoValidator{Validator: validator.New()}
}

var _ Validator = (*GoValidator)(nil)

func (cv *GoValidator) ValidateStruct(data interface{}) error {
	return cv.Validator.Struct(data)
}
