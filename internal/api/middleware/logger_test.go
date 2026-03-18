package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"i9pay/internal/token"

	"github.com/gin-gonic/gin"
)

// logEntry represents a parsed JSON log line from slog.
type logEntry struct {
	Level     string  `json:"level"`
	Msg       string  `json:"msg"`
	RequestID string  `json:"request_id"`
	Method    string  `json:"method"`
	Path      string  `json:"path"`
	Status    float64 `json:"status"`
	LatencyMs float64 `json:"latency_ms"`
	ClientIP  string  `json:"client_ip"`
	UserID    string  `json:"user_id"`
}

// setupLoggerTest creates a gin engine with RequestID + StructuredLogger middleware
// and a handler that responds with the given status code. It returns the recorder
// and a buffer containing the JSON log output.
func setupLoggerTest(statusCode int, setupCtx func(c *gin.Context)) (*httptest.ResponseRecorder, *bytes.Buffer) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", RequestID(), StructuredLogger(logger), func(c *gin.Context) {
		if setupCtx != nil {
			setupCtx(c)
		}
		c.Status(statusCode)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	return w, &buf
}

func parseLogEntry(t *testing.T, buf *bytes.Buffer) logEntry {
	t.Helper()
	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log JSON: %v\nraw: %s", err, buf.String())
	}
	return entry
}

func TestStructuredLogger_200_InfoWithAllFields(t *testing.T) {
	w, buf := setupLoggerTest(http.StatusOK, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	entry := parseLogEntry(t, buf)

	// Log level should be INFO for 2xx
	if entry.Level != "INFO" {
		t.Errorf("expected level INFO, got %q", entry.Level)
	}

	// All required fields must be present
	if entry.RequestID == "" {
		t.Error("expected request_id to be non-empty")
	}
	if entry.Method != "GET" {
		t.Errorf("expected method=GET, got %q", entry.Method)
	}
	if entry.Path != "/test" {
		t.Errorf("expected path=/test, got %q", entry.Path)
	}
	if entry.Status != 200 {
		t.Errorf("expected status=200, got %v", entry.Status)
	}
	if entry.ClientIP == "" {
		t.Error("expected client_ip to be non-empty")
	}

	// user_id should be empty string when no claims
	// Verify the field exists in raw JSON
	var raw map[string]any
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("failed to parse raw JSON: %v", err)
	}
	if _, ok := raw["user_id"]; !ok {
		t.Error("expected user_id field to be present in log entry")
	}
	if _, ok := raw["latency_ms"]; !ok {
		t.Error("expected latency_ms field to be present in log entry")
	}
}

func TestStructuredLogger_400_WarnLevel(t *testing.T) {
	_, buf := setupLoggerTest(http.StatusBadRequest, nil)

	entry := parseLogEntry(t, buf)

	if entry.Level != "WARN" {
		t.Errorf("expected level WARN for 400, got %q", entry.Level)
	}
	if entry.Status != 400 {
		t.Errorf("expected status=400, got %v", entry.Status)
	}
}

func TestStructuredLogger_500_ErrorLevel(t *testing.T) {
	_, buf := setupLoggerTest(http.StatusInternalServerError, nil)

	entry := parseLogEntry(t, buf)

	if entry.Level != "ERROR" {
		t.Errorf("expected level ERROR for 500, got %q", entry.Level)
	}
	if entry.Status != 500 {
		t.Errorf("expected status=500, got %v", entry.Status)
	}
}

func TestStructuredLogger_WithClaims_UserIDPopulated(t *testing.T) {
	_, buf := setupLoggerTest(http.StatusOK, func(c *gin.Context) {
		c.Set(ClaimsKey, &token.Claims{TenantID: "tenant-abc-123"})
	})

	entry := parseLogEntry(t, buf)

	if entry.UserID != "tenant-abc-123" {
		t.Errorf("expected user_id=tenant-abc-123, got %q", entry.UserID)
	}
}

func TestStructuredLogger_NoClaims_UserIDEmpty(t *testing.T) {
	_, buf := setupLoggerTest(http.StatusOK, nil)

	entry := parseLogEntry(t, buf)

	if entry.UserID != "" {
		t.Errorf("expected user_id to be empty, got %q", entry.UserID)
	}
}
