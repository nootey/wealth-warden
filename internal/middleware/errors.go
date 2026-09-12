package middleware

import (
	"net/http"
	"wealth-warden/internal/apperr"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	RequestIDHeader = "X-Request-Id"
	RequestIDKey    = "request_id"
)

func PropagateRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.NewString()
		if sc := trace.SpanContextFromContext(c.Request.Context()); sc.HasTraceID() {
			id = sc.TraceID().String()
		}

		c.Set(RequestIDKey, id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		if !c.Writer.Written() {
			status, message := apperr.Resolve(c.Errors.Last().Err)
			c.JSON(status, utils.APIResponse{Message: message, Code: status})
		}

		log := logger.Warn
		if c.Writer.Status() >= http.StatusInternalServerError {
			log = logger.Error
		}

		for _, err := range c.Errors {
			log("HTTP error",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("client_ip", c.ClientIP()),
				zap.Int("status_code", c.Writer.Status()),
				zap.String("request_id", c.GetString(RequestIDKey)),
				zap.Error(err),
			)
		}
	}
}
