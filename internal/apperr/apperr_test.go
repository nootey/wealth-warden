package apperr_test

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"wealth-warden/internal/apperr"

	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	notFound := apperr.New(apperr.NotFound, "session not found")

	tests := []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"known kind", notFound, http.StatusNotFound, "session not found"},
		{"wrapped sentinel", fmt.Errorf("lookup: %w", notFound), http.StatusNotFound, "session not found"},
		{"wrapped cause", apperr.Wrap(apperr.Conflict, "account in use", sql.ErrNoRows), http.StatusConflict, "account in use"},
		{"plain error", errors.New("pq: relation \"users\" does not exist"), http.StatusInternalServerError, apperr.GenericMessage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, message := apperr.Resolve(tt.err)
			assert.Equal(t, tt.status, status)
			assert.Equal(t, tt.message, message)
		})
	}
}

func TestErrorKeepsCause(t *testing.T) {
	err := apperr.Wrap(apperr.Invalid, "bad account", sql.ErrNoRows)

	assert.True(t, errors.Is(err, sql.ErrNoRows))
	assert.Contains(t, err.Error(), sql.ErrNoRows.Error())
}
