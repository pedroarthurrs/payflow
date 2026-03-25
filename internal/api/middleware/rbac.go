package middleware

import (
	"net/http"

	"i9pay/internal/domain"
	i9token "i9pay/internal/token"

	"github.com/gin-gonic/gin"
)

// RequireRole returns a Gin middleware that enforces a minimum role requirement.
// - If Claims are absent from context → 401
// - If claims.Role is empty (system token, auth_type=system) → 403 "user role required"
// - If domain.RoleLevel(claims.Role) == 0 (invalid/unrecognized role) → 403
// - If domain.RoleLevel(claims.Role) < domain.RoleLevel(minRole) → 403 "insufficient permissions"
// - Otherwise → c.Next()
func RequireRole(minRole domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get(ClaimsKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}

		claims, ok := val.(*i9token.Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}

		if claims.Role == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user role required"})
			return
		}

		if domain.RoleLevel(claims.Role) == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		if domain.RoleLevel(claims.Role) < domain.RoleLevel(minRole) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}
