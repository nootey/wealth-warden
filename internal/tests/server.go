package tests

import (
	"testing"
	wwHttp "wealth-warden/internal/http"

	"github.com/gin-gonic/gin"
)

func NewTestServer(t *testing.T) *wwHttp.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	return wwHttp.NewServer(Container, Logger)
}
