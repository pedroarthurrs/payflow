package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

func setupHealthRouter(svc *service.HealthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(svc)
	r := gin.New()
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func TestLive_Returns200Alive(t *testing.T) {
	svc := service.NewHealthService(nil)
	router := setupHealthRouter(svc)

	w := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/live", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "alive" {
		t.Fatalf("expected status 'alive', got %q", resp["status"])
	}
}

func TestReady_ServiceTrue_Returns200Ready(t *testing.T) {
	svc := service.NewHealthService(nil)
	svc.SetDatabaseOKForTest(true)
	router := setupHealthRouter(svc)

	w := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "ready" {
		t.Fatalf("expected status 'ready', got %q", resp["status"])
	}
}

func TestReady_ServiceFalse_Returns503Unavailable(t *testing.T) {
	svc := service.NewHealthService(nil)
	// NewHealthService defaults to false
	router := setupHealthRouter(svc)

	w := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "unavailable" {
		t.Fatalf("expected status 'unavailable', got %q", resp["status"])
	}
}
