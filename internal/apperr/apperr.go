package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

type Kind int

const (
	GenericMessage = "Something went wrong"

	Internal Kind = iota
	Invalid
	Validation
	Unauthorized
	Forbidden
	NotFound
	Conflict
)

var statuses = map[Kind]int{
	Internal:     http.StatusInternalServerError,
	Invalid:      http.StatusBadRequest,
	Validation:   http.StatusUnprocessableEntity,
	Unauthorized: http.StatusUnauthorized,
	Forbidden:    http.StatusForbidden,
	NotFound:     http.StatusNotFound,
	Conflict:     http.StatusConflict,
}

type Error struct {
	Kind    Kind
	Message string
	Err     error
}

func New(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, err error) *Error {
	return &Error{Kind: kind, Message: message, Err: err}
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) Status() int {
	if status, ok := statuses[e.Kind]; ok {
		return status
	}
	return http.StatusInternalServerError
}

func Resolve(err error) (int, string) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Status(), appErr.Message
	}
	return http.StatusInternalServerError, GenericMessage
}
