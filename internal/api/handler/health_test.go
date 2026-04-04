package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"i9pay/internal/api/middleware"
	"i9pay/internal/database"

	"github.com/gin-gonic/gin"
)

func setupHealthRouter(svc *database.HealthChecker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(svc, middleware.NewMaintenanceProvider())
	r := gin.New()
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func TestLive_Returns200Alive(t *testing.T) {
	svc := database.NewHealthChecker(nil)
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
	svc := database.NewHealthChecker(nil)
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
	svc := database.NewHealthChecker(nil)
	// NewHealthChecker defaults to false
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

func TestStatus_ReturnsMaintenance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := database.NewHealthChecker(nil)
	mp := middleware.NewMaintenanceProvider()
	h := NewHealthHandler(svc, mp)
	r := gin.New()
	h.RegisterRoutes(&r.RouterGroup)

	// default: maintenance disabled
	w := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/status", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["maintenance"] != false {
		t.Fatalf("expected maintenance false, got %v", resp["maintenance"])
	}

	// enable maintenance and check again
	mp.Set(middleware.MaintenanceConfig{Enabled: true, Message: "em manutenção"})
	w2 := httptest.NewRecorder()
	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/status", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w2.Code)
	}
	var resp2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp2["maintenance"] != true {
		t.Fatalf("expected maintenance true, got %v", resp2["maintenance"])
	}
}
