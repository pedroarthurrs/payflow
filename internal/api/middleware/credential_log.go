package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"time"

	i9token "i9pay/internal/auth/token"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxBodySize = 64 * 1024 // 64 KB

// responseBodyWriter wraps gin.ResponseWriter to capture the response body.
type responseBodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

// CredentialLog returns a Gin middleware that intercepts requests authenticated
// via API Key (auth_type = "system") and asynchronously persists a log entry.
func CredentialLog(writer domain.CredentialLogWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 4.2 — filter by auth_type
		raw, exists := c.Get(ClaimsKey)
		if !exists {
			c.Next()
			return
		}
		claims, ok := raw.(*i9token.Claims)
		if !ok || claims.AuthType != domain.AuthTypeSystem {
			c.Next()
			return
		}

		// 4.4 — validate organization_id from claims
		orgID, err := uuid.Parse(claims.OrganizationID)
		if err != nil {
			slog.Error("credential_log: invalid organization_id in claims",
				"organization_id", claims.OrganizationID, "error", err)
			c.Next()
			return
		}

		// 4.3 — capture request fields
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()
		ua := c.GetHeader("User-Agent")

		var reqBody json.RawMessage
		if isJSON(c.GetHeader("Content-Type")) && c.Request.Body != nil {
			data, readErr := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize+1))
			if readErr == nil && len(data) <= maxBodySize && json.Valid(data) {
				reqBody = data
			}
			// 4.4 — restore body for downstream handlers
			c.Request.Body = io.NopCloser(bytes.NewReader(data))
		}

		// optional credential_id from UserID claim
		var credID *uuid.UUID
		if id, parseErr := uuid.Parse(claims.UserID); parseErr == nil {
			credID = &id
		}

		// 4.5 — wrap ResponseWriter to capture response body
		rbw := &responseBodyWriter{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = rbw

		start := time.Now()

		// execute the rest of the chain
		c.Next()

		latencyMs := int32(time.Since(start).Milliseconds())
		statusCode := int32(c.Writer.Status())

		var respBody json.RawMessage
		if isJSON(rbw.Header().Get("Content-Type")) {
			data := rbw.buf.Bytes()
			if len(data) <= maxBodySize && json.Valid(data) {
				respBody = make(json.RawMessage, len(data))
				copy(respBody, data)
			}
		}

		// 4.6 — build input and dispatch asynchronously
		input := domain.CreateCredentialLogInput{
			OrganizationID: orgID,
			CredentialID:   credID,
			Method:         &method,
			Path:           &path,
			RequestBody:    reqBody,
			ResponseBody:   respBody,
			StatusCode:     &statusCode,
			LatencyMs:      &latencyMs,
			IPAddress:      &ip,
			UserAgent:      &ua,
		}

		go func() {
			if writeErr := writer.Create(context.Background(), input); writeErr != nil {
				slog.Error("credential_log: failed to persist log",
					"organization_id", orgID, "error", writeErr)
			}
		}()
	}
}

// isJSON reports whether the Content-Type header indicates JSON.
func isJSON(contentType string) bool {
	return len(contentType) >= 16 && contentType[:16] == "application/json"
}
