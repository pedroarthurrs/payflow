package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"i9pay/internal/database"

	"github.com/gin-gonic/gin"
	"pgregory.net/rapid"
)

// Property 2: Readiness response is deterministic from service state.
// For any boolean value of dbStatus, the readiness handler must return
// 200 {"status":"ready"} when true, and 503 {"status":"unavailable"} when false.
func TestProperty_ReadinessResponseDeterministicFromState(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ok := rapid.Bool().Draw(t, "dbStatus")
		svc := database.NewHealthChecker(nil)
		svc.SetDatabaseOKForTest(ok)

		gin.SetMode(gin.TestMode)
		h := NewHealthHandler(svc)
		r := gin.New()
		h.RegisterRoutes(&r.RouterGroup)

		w := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", http.NoBody)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		r.ServeHTTP(w, req)

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response body: %v", err)
		}

		if ok {
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 when dbStatus=true, got %d", w.Code)
			}
			if resp["status"] != "ready" {
				t.Fatalf("expected status 'ready', got %q", resp["status"])
			}
		} else {
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected 503 when dbStatus=false, got %d", w.Code)
			}
			if resp["status"] != "unavailable" {
				t.Fatalf("expected status 'unavailable', got %q", resp["status"])
			}
		}
	})
}
