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

func ParseID(c *gin.Context, param string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil {
		return 0, apperr.Wrap(apperr.Invalid, "id must be a valid integer", err)
	}
	return id, nil
}
