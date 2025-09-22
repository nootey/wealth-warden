package authz

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func permsFromCtx(c *gin.Context) map[string]struct{} {
	if v, ok := c.Get(CtxPermsKey); ok {
		if m, ok := v.(map[string]struct{}); ok {
			return m
		}
	}
	return nil
}

func hasAll(perms map[string]struct{}, required ...string) bool {
	if len(required) == 0 {
		return true
	}
	if perms == nil {
		return false
	}
	// super-permission bypass
	if _, ok := perms["root_access"]; ok {
		return true
	}
	for _, r := range required {
		if _, ok := perms[r]; !ok {
			return false
		}
	}
	return true
}

func hasAny(perms map[string]struct{}, required ...string) bool {
	if len(required) == 0 {
		return true
	}
	if perms == nil {
		return false
	}
	if _, ok := perms["root_access"]; ok {
		return true
	}
	for _, r := range required {
		if _, ok := perms[r]; ok {
			return true
		}
	}
	return false
}

func RequireAllMW(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasAll(permsFromCtx(c), required...) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"message": "Forbidden",
		})
	}
}

func RequireAnyMW(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasAny(permsFromCtx(c), required...) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"message": "Forbidden",
		})
	}
}
