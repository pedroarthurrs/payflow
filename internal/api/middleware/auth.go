package middleware

import (
	i9token "i9pay/internal/token"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ClaimsKey is the gin context key under which *token.Claims is stored after
// successful authentication. Handlers retrieve claims via c.MustGet(ClaimsKey).
const ClaimsKey = "claims"

// Auth returns a middleware that validates a Bearer access token on every
// request. On success the parsed *token.Claims are stored in the context under
// ClaimsKey and the next handler in the chain is called. On failure the
// request is aborted with 401 Unauthorized.
func Auth(svc *i9token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header format must be: Bearer <token>"})
			return
		}

		claims, err := svc.Verify(parts[1], i9token.TypeAccess)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ClaimsKey, claims)
		c.Next()
	}
}
