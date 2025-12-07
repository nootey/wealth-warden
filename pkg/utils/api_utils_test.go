package utils_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestError_Error(t *testing.T) {
	err := &utils.Error{
		StatusCode: 500,
		Message:    "something went wrong",
	}

	assert.Equal(t, "something went wrong", err.Error())
}

func TestSuccessMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	utils.SuccessMessage(c, "Resource created", "Success", http.StatusCreated)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.JSONEq(t, `{
        "title": "Success",
        "message": "Resource created",
        "code": 201
    }`, w.Body.String())
}

func TestErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		title   string
		message string
		code    int
		err     error
		wantErr bool
	}{
		{
			name:    "with error",
			title:   "Error",
			message: "Something failed",
			code:    http.StatusInternalServerError,
			err:     errors.New("database connection failed"),
			wantErr: true,
		},
		{
			name:    "without error",
			title:   "Not Found",
			message: "Resource not found",
			code:    http.StatusNotFound,
			err:     nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			utils.ErrorMessage(c, tt.title, tt.message, tt.code, tt.err)

			assert.Equal(t, tt.code, w.Code)
			assert.Contains(t, w.Body.String(), tt.message)
			assert.Contains(t, w.Body.String(), tt.title)

			if tt.wantErr {
				assert.Len(t, c.Errors, 1)
				assert.Equal(t, tt.err, c.Errors[0].Err)
			} else {
				assert.Len(t, c.Errors, 0)
			}
		})
	}
}

func TestValidationFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	validationErr := errors.New("email is required")
	utils.ValidationFailed(c, "Invalid input", validationErr)

	assert.Equal(t, 422, w.Code)
	assert.JSONEq(t, `{
        "title": "Validation Failed",
        "message": "Invalid input",
        "code": 422
    }`, w.Body.String())
	assert.Len(t, c.Errors, 1)
}
