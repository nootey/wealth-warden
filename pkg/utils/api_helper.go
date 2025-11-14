package utils

import (
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
