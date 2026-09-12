package utils_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"wealth-warden/internal/apperr"
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

func TestParseID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name  string
		value string
		want  int64
		ok    bool
	}{
		{"valid", "42", 42, true},
		{"negative", "-1", -1, true},
		{"not a number", "abc", 0, false},
		{"empty", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.value}}

			id, err := utils.ParseID(c, "id")

			if !tt.ok {
				status, message := apperr.Resolve(err)
				assert.Equal(t, http.StatusBadRequest, status)
				assert.Equal(t, "id must be a valid integer", message)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, id)
		})
	}
}
