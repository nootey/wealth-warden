package middleware

import (
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func InjectPerms(s *authz.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user_id")
		if !ok {
			c.Next()
			return
		}
		uid, ok := v.(int64)
		if !ok || uid <= 0 {
			c.Next()
			return
		}
		if perms, err := s.PermsForUser(c.Request.Context(), uid); err == nil {
			c.Set(authz.CtxPermsKey, perms)
		}
		c.Next()
	}
}
