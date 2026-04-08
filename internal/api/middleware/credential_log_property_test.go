package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	i9token "i9pay/internal/auth/token"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// buildPropRouter creates a minimal test router with injected claims and a
// handler that writes the given status, body, and content-type.
func buildPropRouter(
	w domain.CredentialLogWriter,
	claims *i9token.Claims,
	status int,
	respBody string,
	respCT string,
) *gin.Engine {
	r := gin.New()
	if claims != nil {
		r.Use(func(c *gin.Context) { c.Set(ClaimsKey, claims); c.Next() })
	}
	r.Use(CredentialLog(w))
	r.Any("/test", func(c *gin.Context) {
		if respCT != "" {
			c.Header("Content-Type", respCT)
		}
		c.String(status, respBody)
	})
	return r
}

func waitWrite(m *mockWriter, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if m.callCount() > 0 {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return false
}

// ── Property 1: auth_type filtering ──────────────────────────────────────────
// Feature: credential-log-middleware, Property 1: auth_type filtering
// For any request, writer.Create is called iff auth_type = "system".

func TestProperty_CredentialLogMiddleware_AuthTypeFiltering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		authType := rapid.SampledFrom([]domain.AuthType{
			domain.AuthTypeSystem,
			domain.AuthTypeUser,
		}).Draw(t, "auth_type")

		orgID := uuid.New().String()
		claims := &i9token.Claims{
			OrganizationID: orgID,
			UserID:         uuid.New().String(),
			AuthType:       authType,
		}

		w := &mockWriter{}
		r := buildPropRouter(w, claims, http.StatusOK, `{}`, "application/json")

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
		r.ServeHTTP(rec, req)

		if authType == domain.AuthTypeSystem {
			if !waitWrite(w, 200*time.Millisecond) {
				t.Fatalf("auth_type=system: expected writer to be called")
			}
		} else {
			time.Sleep(20 * time.Millisecond)
			if w.callCount() != 0 {
				t.Fatalf("auth_type=%s: expected writer NOT to be called, got %d calls", authType, w.callCount())
			}
		}
	})
}

// ── Property 1b: no claims → writer never called ─────────────────────────────

func TestProperty_CredentialLogMiddleware_NoClaims_WriterNeverCalled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		_ = rapid.String().Draw(t, "ignored")

		w := &mockWriter{}
		r := buildPropRouter(w, nil, http.StatusOK, `{}`, "application/json")

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
		r.ServeHTTP(rec, req)

		time.Sleep(20 * time.Millisecond)
		if w.callCount() != 0 {
			t.Fatalf("no claims: expected 0 writer calls, got %d", w.callCount())
		}
	})
}

// ── Property 2: log field completeness ───────────────────────────────────────
// Feature: credential-log-middleware, Property 2: log field completeness
// For any system request, all captured fields match the request/response data.

func TestProperty_CredentialLogMiddleware_FieldCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()
		credID := uuid.New()
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}).Draw(t, "method")
		statusCode := rapid.IntRange(200, 599).Draw(t, "status_code")
		userAgent := rapid.StringMatching(`[a-zA-Z0-9/\.\-]{1,30}`).Draw(t, "user_agent")

		claims := &i9token.Claims{
			OrganizationID: orgID.String(),
			UserID:         credID.String(),
			AuthType:       domain.AuthTypeSystem,
		}

		w := &mockWriter{}
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set(ClaimsKey, claims); c.Next() })
		r.Use(CredentialLog(w))
		r.Any("/test", func(c *gin.Context) {
			c.Header("Content-Type", "application/json")
			c.String(statusCode, `{}`)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), method, "/test", http.NoBody)
		req.Header.Set("User-Agent", userAgent)
		r.ServeHTTP(rec, req)

		if !waitWrite(w, 200*time.Millisecond) {
			t.Fatal("writer was not called")
		}

		got := w.lastCall()

		if got.OrganizationID != orgID {
			t.Errorf("OrganizationID: want %v, got %v", orgID, got.OrganizationID)
		}
		if got.CredentialID == nil || *got.CredentialID != credID {
			t.Errorf("CredentialID: want %v, got %v", credID, got.CredentialID)
		}
		if got.Method == nil || *got.Method != method {
			t.Errorf("Method: want %q, got %v", method, got.Method)
		}
		if got.Path == nil || *got.Path != "/test" {
			t.Errorf("Path: want /test, got %v", got.Path)
		}
		if got.UserAgent == nil || *got.UserAgent != userAgent {
			t.Errorf("UserAgent: want %q, got %v", userAgent, got.UserAgent)
		}
		if got.StatusCode == nil || *got.StatusCode != int32(statusCode) {
			t.Errorf("StatusCode: want %d, got %v", statusCode, got.StatusCode)
		}
		if got.LatencyMs == nil || *got.LatencyMs < 0 {
			t.Errorf("LatencyMs: want >= 0, got %v", got.LatencyMs)
		}
	})
}

// ── Property 3: conditional JSON body capture ─────────────────────────────────
// Feature: credential-log-middleware, Property 3: conditional JSON body capture
// RequestBody is non-nil iff Content-Type is application/json and body is valid JSON.

func TestProperty_CredentialLogMiddleware_ConditionalJSONBodyCapture(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		isJSON := rapid.Bool().Draw(t, "is_json")
		hasBody := rapid.Bool().Draw(t, "has_body")

		claims := &i9token.Claims{
			OrganizationID: uuid.New().String(),
			UserID:         uuid.New().String(),
			AuthType:       domain.AuthTypeSystem,
		}

		w := &mockWriter{}
		r := buildPropRouter(w, claims, http.StatusOK, `{}`, "application/json")

		var reqBody *strings.Reader
		ct := "text/plain"
		if isJSON {
			ct = "application/json"
		}

		var bodyStr string
		if hasBody && isJSON {
			bodyStr = `{"key":"value"}`
			reqBody = strings.NewReader(bodyStr)
		} else if hasBody {
			bodyStr = "plain text"
			reqBody = strings.NewReader(bodyStr)
		}

		rec := httptest.NewRecorder()
		var req *http.Request
		if reqBody != nil {
			req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", reqBody)
		} else {
			req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", http.NoBody)
		}
		req.Header.Set("Content-Type", ct)
		r.ServeHTTP(rec, req)

		if !waitWrite(w, 200*time.Millisecond) {
			t.Fatal("writer was not called")
		}

		got := w.lastCall()
		shouldCapture := isJSON && hasBody && json.Valid([]byte(bodyStr))

		if shouldCapture && got.RequestBody == nil {
			t.Errorf("expected non-nil RequestBody for JSON body, got nil")
		}
		if !shouldCapture && got.RequestBody != nil {
			t.Errorf("expected nil RequestBody for non-JSON/absent body, got %s", got.RequestBody)
		}
	})
}

// ── Property 4: body size limit ───────────────────────────────────────────────
// Feature: credential-log-middleware, Property 4: body size limit
// For any body > 64KB, the corresponding field in the input is nil.

func TestProperty_CredentialLogMiddleware_BodySizeLimit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate sizes: either within limit or over limit.
		overLimit := rapid.Bool().Draw(t, "over_limit")
		var size int
		if overLimit {
			size = rapid.IntRange(maxBodySize+1, maxBodySize+10240).Draw(t, "size")
		} else {
			// Small valid JSON body well within limit.
			size = 0
		}

		claims := &i9token.Claims{
			OrganizationID: uuid.New().String(),
			UserID:         uuid.New().String(),
			AuthType:       domain.AuthTypeSystem,
		}

		w := &mockWriter{}
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set(ClaimsKey, claims); c.Next() })
		r.Use(CredentialLog(w))
		r.POST("/test", func(c *gin.Context) {
			c.Header("Content-Type", "application/json")
			if overLimit {
				// Write a large response body too.
				padding := strings.Repeat("x", size)
				c.String(http.StatusOK, `{"d":"`+padding+`"}`)
			} else {
				c.String(http.StatusOK, `{}`)
			}
		})

		var reqBodyStr string
		if overLimit {
			padding := strings.Repeat("x", size)
			reqBodyStr = `{"d":"` + padding + `"}`
		} else {
			reqBodyStr = `{"small":true}`
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
			strings.NewReader(reqBodyStr))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(rec, req)

		if !waitWrite(w, 300*time.Millisecond) {
			t.Fatal("writer was not called")
		}

		got := w.lastCall()
		if overLimit {
			if got.RequestBody != nil {
				t.Errorf("oversize request body: expected nil RequestBody, got %d bytes", len(got.RequestBody))
			}
			if got.ResponseBody != nil {
				t.Errorf("oversize response body: expected nil ResponseBody, got %d bytes", len(got.ResponseBody))
			}
		} else if got.RequestBody == nil {
			t.Error("small request body: expected non-nil RequestBody")
		}
	})
}

// ── Property 5: middleware transparency ───────────────────────────────────────
// Feature: credential-log-middleware, Property 5: middleware transparency
// For any request, the response delivered to the client is identical to what the handler wrote.

func TestProperty_CredentialLogMiddleware_Transparency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		statusCode := rapid.IntRange(200, 599).Draw(t, "status_code")
		body := rapid.StringMatching(`[a-zA-Z0-9]{1,50}`).Draw(t, "body")

		claims := &i9token.Claims{
			OrganizationID: uuid.New().String(),
			UserID:         uuid.New().String(),
			AuthType:       domain.AuthTypeSystem,
		}

		w := &mockWriter{}
		r := buildPropRouter(w, claims, statusCode, body, "text/plain")

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
		r.ServeHTTP(rec, req)

		if rec.Code != statusCode {
			t.Errorf("status: want %d, got %d", statusCode, rec.Code)
		}
		// Some status codes legitimately have no body per HTTP spec:
		// 1xx informational, 204 No Content, 304 Not Modified.
		noBody := statusCode < 200 || statusCode == http.StatusNoContent || statusCode == http.StatusNotModified
		if !noBody && rec.Body.String() != body {
			t.Errorf("body: want %q, got %q", body, rec.Body.String())
		}
	})
}

// ── Property 6: persistence error isolation ───────────────────────────────────
// Feature: credential-log-middleware, Property 6: persistence error isolation
// For any system request where writer returns error, the HTTP response is unaffected.

func TestProperty_CredentialLogMiddleware_PersistenceErrorIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		statusCode := rapid.IntRange(200, 299).Draw(t, "status_code")

		claims := &i9token.Claims{
			OrganizationID: uuid.New().String(),
			UserID:         uuid.New().String(),
			AuthType:       domain.AuthTypeSystem,
		}

		w := &mockWriter{err: domain.ErrInternal}
		r := buildPropRouter(w, claims, statusCode, `{"ok":true}`, "application/json")

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
		r.ServeHTTP(rec, req)

		if rec.Code != statusCode {
			t.Errorf("status: want %d, got %d (writer error should not affect response)", statusCode, rec.Code)
		}
		// 204 No Content legitimately has no body; only check body for other codes.
		if statusCode != http.StatusNoContent && !strings.Contains(rec.Body.String(), `"ok":true`) {
			t.Errorf("body should be unaffected by writer error, got %q", rec.Body.String())
		}
	})
}
