package utils

import (
	"strconv"
	"wealth-warden/internal/apperr"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	return e.Message
}

func SuccessMessage(c *gin.Context, message, title string, code int) {
	response := APIResponse{
		Title:   title,
		Message: message,
		Code:    code,
	}
	c.JSON(code, response)
}

func ErrorMessage(c *gin.Context, title, message string, code int, err error) {

	// Append the error to the context and let the gin middleware log it.
	if err != nil {
		_ = c.Error(err)
	}

	response := APIResponse{
		Title:   title,
		Message: message,
		Code:    code,
	}
	c.JSON(code, response)
}

func ValidationFailed(c *gin.Context, message string, err error) {
	ErrorMessage(c, "Validation Failed", message, 422, err)
}

func ParseID(c *gin.Context, param string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil {
		return 0, apperr.Wrap(apperr.Invalid, "id must be a valid integer", err)
	}
	return id, nil
}
