package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRequestID_GeneratesUUIDWhenNoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var ctxRequestID string
	r.GET("/test", RequestID(), func(c *gin.Context) {
		ctxRequestID = c.GetString(RequestIDKey)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Response header should be set
	got := w.Header().Get("X-Request-ID")
	if got == "" {
		t.Fatal("expected X-Request-ID response header to be set")
	}

	// Should be a valid UUID
	if _, err := uuid.Parse(got); err != nil {
		t.Fatalf("expected valid UUID in X-Request-ID header, got %q: %v", got, err)
	}

	// Context value should match the response header
	if ctxRequestID != got {
		t.Fatalf("expected context request_id=%q, got %q", got, ctxRequestID)
	}
}

func TestRequestID_ReusesExistingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var ctxRequestID string
	r.GET("/test", RequestID(), func(c *gin.Context) {
		ctxRequestID = c.GetString(RequestIDKey)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody)
	req.Header.Set("X-Request-ID", "existing-id-123")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Response header should echo the incoming value
	got := w.Header().Get("X-Request-ID")
	if got != "existing-id-123" {
		t.Fatalf("expected X-Request-ID=existing-id-123, got %q", got)
	}

	// Context value should match
	if ctxRequestID != "existing-id-123" {
		t.Fatalf("expected context request_id=existing-id-123, got %q", ctxRequestID)
	}
}

func TestRequestID_ResponseHeaderAlwaysSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, r := gin.CreateTestContext(httptest.NewRecorder())
	r.GET("/test", RequestID(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Send multiple requests and verify each has the header
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody))

		if got := w.Header().Get("X-Request-ID"); got == "" {
			t.Fatalf("request %d: expected X-Request-ID header to be set", i+1)
		}
	}
}

func TestRequestID_StoredInGinContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var ctxValue string
	r.GET("/test", RequestID(), func(c *gin.Context) {
		ctxValue = c.GetString(RequestIDKey)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	if ctxValue == "" {
		t.Fatal("expected request_id to be stored in Gin context, got empty string")
	}

	// The context value should match the response header
	if ctxValue != w.Header().Get("X-Request-ID") {
		t.Fatalf("context value %q does not match response header %q", ctxValue, w.Header().Get("X-Request-ID"))
	}
}
