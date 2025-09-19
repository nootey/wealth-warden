package authz

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type UserIDExtractor func(*gin.Context) (int64, error)

func Middleware(s *Service, extract UserIDExtractor) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, err := extract(c)
		if err != nil || uid == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			return
		}
		pr, err := s.LoadPrincipal(c, uid)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Auth load error"})
			return
		}
		c.Set("principal", pr)
		c.Next()
	}
}

func PrincipalFromCtx(c *gin.Context) *Principal {
	if v, ok := c.Get("principal"); ok {
		if p, ok := v.(*Principal); ok {
			return p
		}
	}
	return nil
}

func RequireAllMW(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if p := PrincipalFromCtx(c); p != nil && p.HasAll(perms...) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Forbidden"})
	}
}
func RequireRoleMW(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if p := PrincipalFromCtx(c); p != nil && p.HasRole(role) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Forbidden"})
	}
}
