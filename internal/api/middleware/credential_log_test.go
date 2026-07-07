package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	i9token "payflow/internal/auth/token"
	"payflow/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockWriter is a test double for domain.CredentialLogWriter.
type mockWriter struct {
	mu    sync.Mutex
	calls []domain.CreateCredentialLogInput
	err   error
}

func (m *mockWriter) Create(_ context.Context, input domain.CreateCredentialLogInput) error {
	m.mu.Lock()
	m.calls = append(m.calls, input)
	m.mu.Unlock()
	return m.err
}

func (m *mockWriter) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockWriter) lastCall() domain.CreateCredentialLogInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

// buildRouter creates a test Gin engine with the CredentialLog middleware and a
// handler that writes the given status + body.
func buildRouter(w domain.CredentialLogWriter, claims *i9token.Claims, status int, respBody, respContentType string) *gin.Engine {
	r := gin.New()
	if claims != nil {
		r.Use(func(c *gin.Context) {
			c.Set(ClaimsKey, claims)
			c.Next()
		})
	}
	r.Use(CredentialLog(w))
	r.Any("/test", func(c *gin.Context) {
		if respContentType != "" {
			c.Header("Content-Type", respContentType)
		}
		c.String(status, respBody)
	})
	return r
}

func systemClaims(orgID, credID string) *i9token.Claims {
	return &i9token.Claims{
		OrganizationID: orgID,
		UserID:         credID,
		AuthType:       domain.AuthTypeSystem,
	}
}

func waitForWrite(m *mockWriter, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.callCount() > 0 {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return false
}

// ── No claims → writer not called ────────────────────────────────────────────

func TestCredentialLog_NoClaims_WriterNotCalled(t *testing.T) {
	w := &mockWriter{}
	r := buildRouter(w, nil, http.StatusOK, `{}`, "application/json")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	r.ServeHTTP(rec, req)

	time.Sleep(20 * time.Millisecond)
	if w.callCount() != 0 {
		t.Fatalf("expected 0 writer calls, got %d", w.callCount())
	}
}

// ── auth_type = "user" → writer not called ───────────────────────────────────

func TestCredentialLog_UserAuthType_WriterNotCalled(t *testing.T) {
	w := &mockWriter{}
	claims := &i9token.Claims{
		OrganizationID: uuid.New().String(),
		AuthType:       domain.AuthTypeUser,
	}
	r := buildRouter(w, claims, http.StatusOK, `{}`, "application/json")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	r.ServeHTTP(rec, req)

	time.Sleep(20 * time.Millisecond)
	if w.callCount() != 0 {
		t.Fatalf("expected 0 writer calls, got %d", w.callCount())
	}
}

// ── invalid organization_id → writer not called ───────────────────────────────

func TestCredentialLog_InvalidOrgID_WriterNotCalled(t *testing.T) {
	w := &mockWriter{}
	claims := &i9token.Claims{
		OrganizationID: "not-a-uuid",
		AuthType:       domain.AuthTypeSystem,
	}
	r := buildRouter(w, claims, http.StatusOK, `{}`, "application/json")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	r.ServeHTTP(rec, req)

	time.Sleep(20 * time.Millisecond)
	if w.callCount() != 0 {
		t.Fatalf("expected 0 writer calls, got %d", w.callCount())
	}
}

// ── invalid credential_id → CredentialID = nil ───────────────────────────────

func TestCredentialLog_InvalidCredentialID_NilInInput(t *testing.T) {
	w := &mockWriter{}
	claims := systemClaims(uuid.New().String(), "not-a-uuid")
	r := buildRouter(w, claims, http.StatusOK, `{}`, "application/json")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	r.ServeHTTP(rec, req)

	if !waitForWrite(w, 200*time.Millisecond) {
		t.Fatal("writer was not called")
	}
	if w.lastCall().CredentialID != nil {
		t.Errorf("expected CredentialID nil, got %v", w.lastCall().CredentialID)
	}
}

// ── non-JSON request body → RequestBody = nil ────────────────────────────────

func TestCredentialLog_NonJSONRequestBody_NilRequestBody(t *testing.T) {
	w := &mockWriter{}
	claims := systemClaims(uuid.New().String(), uuid.New().String())
	r := buildRouter(w, claims, http.StatusOK, `{}`, "application/json")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		strings.NewReader("plain text body"))
	req.Header.Set("Content-Type", "text/plain")
	r.ServeHTTP(rec, req)

	if !waitForWrite(w, 200*time.Millisecond) {
		t.Fatal("writer was not called")
	}
	if w.lastCall().RequestBody != nil {
		t.Errorf("expected RequestBody nil, got %s", w.lastCall().RequestBody)
	}
}

// ── absent request body → RequestBody = nil ──────────────────────────────────

func TestCredentialLog_AbsentRequestBody_NilRequestBody(t *testing.T) {
	w := &mockWriter{}
	claims := systemClaims(uuid.New().String(), uuid.New().String())
	r := buildRouter(w, claims, http.StatusOK, `{}`, "application/json")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	r.ServeHTTP(rec, req)

	if !waitForWrite(w, 200*time.Millisecond) {
		t.Fatal("writer was not called")
	}
	if w.lastCall().RequestBody != nil {
		t.Errorf("expected RequestBody nil, got %s", w.lastCall().RequestBody)
	}
}

// ── writer error → HTTP response unaffected ──────────────────────────────────

func TestCredentialLog_WriterError_ResponseUnaffected(t *testing.T) {
	w := &mockWriter{err: errors.New("db down")}
	claims := systemClaims(uuid.New().String(), uuid.New().String())
	r := buildRouter(w, claims, http.StatusOK, `{"ok":true}`, "application/json")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

// ── system request → all fields captured correctly ───────────────────────────

func TestCredentialLog_SystemRequest_FieldsCaptured(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()
	w := &mockWriter{}
	claims := systemClaims(orgID.String(), credID.String())

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(ClaimsKey, claims); c.Next() })
	r.Use(CredentialLog(w))
	r.POST("/v1/customers", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusCreated, `{"id":"new"}`)
	})

	body := bytes.NewBufferString(`{"name":"acme"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-client/1.0")
	r.ServeHTTP(rec, req)

	if !waitForWrite(w, 200*time.Millisecond) {
		t.Fatal("writer was not called")
	}

	got := w.lastCall()
	if got.OrganizationID != orgID {
		t.Errorf("OrganizationID: want %v, got %v", orgID, got.OrganizationID)
	}
	if got.CredentialID == nil || *got.CredentialID != credID {
		t.Errorf("CredentialID: want %v, got %v", credID, got.CredentialID)
	}
	if got.Method == nil || *got.Method != http.MethodPost {
		t.Errorf("Method: want POST, got %v", got.Method)
	}
	if got.Path == nil || *got.Path != "/v1/customers" {
		t.Errorf("Path: want /v1/customers, got %v", got.Path)
	}
	if got.UserAgent == nil || *got.UserAgent != "test-client/1.0" {
		t.Errorf("UserAgent: want test-client/1.0, got %v", got.UserAgent)
	}
	if got.StatusCode == nil || *got.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode: want 201, got %v", got.StatusCode)
	}
	if got.LatencyMs == nil || *got.LatencyMs < 0 {
		t.Errorf("LatencyMs: want >= 0, got %v", got.LatencyMs)
	}
	if got.RequestBody == nil {
		t.Error("RequestBody: want non-nil JSON")
	}
	if got.ResponseBody == nil {
		t.Error("ResponseBody: want non-nil JSON")
	}
}

// ── request body > 64KB → RequestBody = nil ──────────────────────────────────

func TestCredentialLog_LargeRequestBody_NilRequestBody(t *testing.T) {
	w := &mockWriter{}
	claims := systemClaims(uuid.New().String(), uuid.New().String())

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(ClaimsKey, claims); c.Next() })
	r.Use(CredentialLog(w))
	r.POST("/test", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, `{}`)
	})

	// Build a JSON body just over 64KB: `{"a":"<65KB of 'x'>"}`.
	padding := strings.Repeat("x", maxBodySize+10)
	bigBody := `{"a":"` + padding + `"}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		strings.NewReader(bigBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if !waitForWrite(w, 200*time.Millisecond) {
		t.Fatal("writer was not called")
	}
	if w.lastCall().RequestBody != nil {
		t.Errorf("expected RequestBody nil for oversized body, got %d bytes", len(w.lastCall().RequestBody))
	}
}

// ── response body > 64KB → ResponseBody = nil ────────────────────────────────

func TestCredentialLog_LargeResponseBody_NilResponseBody(t *testing.T) {
	w := &mockWriter{}
	claims := systemClaims(uuid.New().String(), uuid.New().String())

	padding := strings.Repeat("x", maxBodySize+10)
	bigResp := `{"data":"` + padding + `"}`

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(ClaimsKey, claims); c.Next() })
	r.Use(CredentialLog(w))
	r.GET("/test", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, bigResp)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	r.ServeHTTP(rec, req)

	if !waitForWrite(w, 200*time.Millisecond) {
		t.Fatal("writer was not called")
	}
	if w.lastCall().ResponseBody != nil {
		t.Errorf("expected ResponseBody nil for oversized response, got %d bytes", len(w.lastCall().ResponseBody))
	}
}

// ── middleware transparency: response delivered unchanged ─────────────────────

func TestCredentialLog_Transparency_ResponseUnchanged(t *testing.T) {
	w := &mockWriter{}
	claims := systemClaims(uuid.New().String(), uuid.New().String())
	expectedBody := `{"result":"ok"}`

	r := buildRouter(w, claims, http.StatusAccepted, expectedBody, "application/json")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", rec.Code)
	}
	if rec.Body.String() != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, rec.Body.String())
	}
}

// ── body restored for downstream handlers ────────────────────────────────────

func TestCredentialLog_BodyRestored_DownstreamCanRead(t *testing.T) {
	w := &mockWriter{}
	claims := systemClaims(uuid.New().String(), uuid.New().String())

	var downstreamBody []byte
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(ClaimsKey, claims); c.Next() })
	r.Use(CredentialLog(w))
	r.POST("/test", func(c *gin.Context) {
		var err error
		downstreamBody, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "read error")
			return
		}
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, `{}`)
	})

	payload := `{"key":"value"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if string(downstreamBody) != payload {
		t.Errorf("downstream body: want %q, got %q", payload, string(downstreamBody))
	}
}
