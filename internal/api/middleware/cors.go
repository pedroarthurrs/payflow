package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
// It allows credentials (cookies + Authorization headers) from a strict allowlist
// of origins. The wildcard "*" is intentionally not supported when credentials
// are enabled, as required by the CORS specification.
//
// OPTIONS pre-flight requests are responded to immediately with 204 No Content
// so they do not traverse the rest of the handler chain.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o != "" {
			allowed[o] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Headers",
					"Content-Type, Authorization, X-Request-ID, X-CSRF-Token, X-Idempotency-Key")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Max-Age", "86400")
				// Vary must list Origin so caches handle different origins correctly.
				c.Header("Vary", "Origin")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
