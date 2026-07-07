package middleware

import (
	"context"
	"log/slog"
	"time"

	i9token "payflow/internal/auth/token"
	"payflow/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MemberLog returns a Gin middleware that intercepts requests authenticated
// via JWT (auth_type = "user") and asynchronously persists a member log entry.
// The action uses the format "{METHOD} {path}", e.g. "GET /v1/customers".
func MemberLog(writer domain.MemberLogWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Filter by auth_type
		raw, exists := c.Get(ClaimsKey)
		if !exists {
			c.Next()
			return
		}
		claims, ok := raw.(*i9token.Claims)
		if !ok || claims.AuthType != domain.AuthTypeUser {
			c.Next()
			return
		}

		// Extract member_id from claims — identifies the organization_members row
		memberID, err := uuid.Parse(claims.MemberID)
		if err != nil {
			slog.Error("member_log: invalid member_id in claims",
				"member_id", claims.MemberID, "error", err)
			c.Next()
			return
		}

		// Extract organization_id from claims — required (member_logs.organization_id is NOT NULL)
		orgID, orgParseErr := uuid.Parse(claims.OrganizationID)
		if orgParseErr != nil {
			c.Next()
			return
		}

		// Capture request fields
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()
		ua := c.GetHeader("User-Agent")

		// Wrap ResponseWriter to capture response body
		rbw := &responseBodyWriter{ResponseWriter: c.Writer, buf: nil}
		c.Writer = rbw

		start := time.Now()

		// Execute the rest of the chain
		c.Next()

		latencyMs := int32(time.Since(start).Milliseconds())
		statusCode := int32(c.Writer.Status())

		// Build action string
		action := method + " " + path

		// Build input and dispatch asynchronously
		input := domain.CreateMemberLogInput{
			OrganizationID: orgID,
			MemberID:       memberID,
			Action:         action,
			Method:         &method,
			Path:           &path,
			StatusCode:     &statusCode,
			LatencyMs:      &latencyMs,
			IPAddress:      &ip,
			UserAgent:      &ua,
		}

		go func() {
			if writeErr := writer.Create(context.Background(), input); writeErr != nil {
				slog.Error("member_log: failed to persist log",
					"member_id", memberID, "error", writeErr)
			}
		}()
	}
}
