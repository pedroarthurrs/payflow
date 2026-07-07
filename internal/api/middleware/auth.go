package middleware

import (
	"net/http"
	"strings"

	i9token "payflow/internal/auth/token"

	"github.com/gin-gonic/gin"
)

// ClaimsKey is the gin context key under which *token.Claims is stored after
// successful authentication. Handlers retrieve claims via c.MustGet(ClaimsKey).
const ClaimsKey = "claims"

// Auth returns a middleware that validates a Bearer access token on every
// request. It accepts the token from two sources, tried in order:
//  1. Authorization: Bearer <token>  — used by API key / programmatic clients.
//  2. "payflow_access_token" HttpOnly cookie — used by browser clients.
//
// On success the parsed *token.Claims are stored in the context under ClaimsKey
// and the next handler in the chain is called. On failure the request is
// aborted with 401 Unauthorized.
func Auth(svc *i9token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := tokenFromBearer(c)
		if tokenStr == "" {
			tokenStr = tokenFromCookie(c)
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}

		claims, err := svc.Verify(tokenStr, i9token.TypeAccess)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

// tokenFromBearer extracts a token from the "Authorization: Bearer <token>" header.
// Returns an empty string if the header is absent or malformed.
func tokenFromBearer(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

// tokenFromCookie extracts a token from the "payflow_access_token" HttpOnly cookie.
// Returns an empty string if the cookie is absent.
func tokenFromCookie(c *gin.Context) string {
	cookie, err := c.Cookie("payflow_access_token")
	if err != nil {
		return ""
	}
	return cookie
}
