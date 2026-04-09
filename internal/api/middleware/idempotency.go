package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"

	"i9pay/internal/auth/token"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// IdempotencyHeader is the HTTP header used by clients to supply an idempotency key.
	IdempotencyHeader = "X-Idempotency-Key"

	// DefaultIdempotencyTTL is the default time-to-live for cached responses.
	DefaultIdempotencyTTL = 24 * time.Hour
)

// idempotencyResponseWriter wraps gin.ResponseWriter to capture the response body
// written by the handler chain so it can be persisted in the idempotency store.
type idempotencyResponseWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *idempotencyResponseWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

// Idempotency returns a Gin middleware that prevents duplicate write operations
// (POST, PATCH) by caching the response keyed on (organization_id, X-Idempotency-Key).
//
// Flow:
//  1. Extract X-Idempotency-Key header — skip if absent.
//  2. Identify the Organization from the auth claims.
//  3. storage.Check — on hit, replay the cached response immediately.
//  4. On miss, wrap the ResponseWriter, call c.Next(), then storage.Save.
func Idempotency(storage domain.IdempotencyStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only intercept write methods.
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		key := c.GetHeader(IdempotencyHeader)
		if key == "" {
			c.Next()
			return
		}

		// Extract organization_id from JWT claims set by the Auth middleware.
		raw, exists := c.Get(ClaimsKey)
		if !exists {
			c.Next()
			return
		}
		claims, ok := raw.(*token.Claims)
		if !ok {
			c.Next()
			return
		}
		orgID, err := uuid.Parse(claims.OrganizationID)
		if err != nil {
			c.Next()
			return
		}

		// --- Cache HIT ---
		record, err := storage.Check(c.Request.Context(), key, orgID)
		if err != nil {
			slog.Error("idempotency: failed to check key", "key", key, "organization_id", orgID, "error", err)
			// On storage errors proceed normally instead of blocking.
			c.Next()
			return
		}
		if record != nil {
			c.Data(record.ResponseCode, "application/json; charset=utf-8", record.ResponseBody)
			c.Abort()
			return
		}

		// --- Cache MISS — capture the handler's response ---
		irw := &idempotencyResponseWriter{
			ResponseWriter: c.Writer,
			buf:            &bytes.Buffer{},
		}
		c.Writer = irw

		c.Next()

		// Persist the result after the handler chain completes.
		statusCode := c.Writer.Status()
		body := make([]byte, irw.buf.Len())
		copy(body, irw.buf.Bytes())

		saveRecord := domain.IdempotencyRecord{
			ResponseCode: statusCode,
			ResponseBody: body,
		}

		if saveErr := storage.Save(c.Request.Context(), key, orgID, c.Request.URL.Path, saveRecord, DefaultIdempotencyTTL); saveErr != nil {
			slog.Error("idempotency: failed to save key", "key", key, "organization_id", orgID, "error", saveErr)
		}
	}
}
