package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDKey is the Gin context key for the request ID.
const RequestIDKey = "request_id"

// RequestID returns a middleware that generates a UUID v4 per request,
// stores it in the Gin context, and sets the X-Request-ID response header.
// If the incoming request already has X-Request-ID, it is reused.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}

		c.Set(RequestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
