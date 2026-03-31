package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"i9pay/internal/auth/token"

	"github.com/gin-gonic/gin"
)

// StructuredLogger returns a Gin middleware that logs each request using slog.
// Fields: request_id, method, path, status, latency_ms, client_ip, user_id.
// Log level is selected based on response status: Info for 2xx/3xx, Warn for
// 4xx, Error for 5xx.
func StructuredLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		requestID, _ := c.Get(RequestIDKey)

		userID := ""
		if claims, exists := c.Get(ClaimsKey); exists {
			if tc, ok := claims.(*token.Claims); ok {
				userID = tc.OrganizationID
			}
		}

		attrs := []slog.Attr{
			slog.String("request_id", fmt.Sprintf("%v", requestID)),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Float64("latency_ms", float64(latency.Milliseconds())),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_id", userID),
		}

		msg := c.Request.Method + " " + c.Request.URL.Path

		switch {
		case status >= 500:
			logger.LogAttrs(c.Request.Context(), slog.LevelError, msg, attrs...)
		case status >= 400:
			logger.LogAttrs(c.Request.Context(), slog.LevelWarn, msg, attrs...)
		default:
			logger.LogAttrs(c.Request.Context(), slog.LevelInfo, msg, attrs...)
		}
	}
}
