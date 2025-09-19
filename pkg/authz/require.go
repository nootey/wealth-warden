package authz

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func RequireAll(c *gin.Context, perms ...string) bool {
	p := PrincipalFromCtx(c)
	if p == nil || !p.HasAll(perms...) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"message": "Forbidden",
			"missing": perms,
		})
		return false
	}
	return true
}

func RequireRole(c *gin.Context, role string) bool {
	p := PrincipalFromCtx(c)
	if p == nil || !p.HasRole(role) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Forbidden"})
		return false
	}
	return true
}
