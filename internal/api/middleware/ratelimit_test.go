package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"i9pay/internal/auth/token"

	"github.com/gin-gonic/gin"
)

func TestAllow_FirstRequestAllowed(t *testing.T) {
	store := NewRateLimiterStore()
	maxReqs := 5
	window := time.Minute

	allowed, remaining, resetAt := store.Allow("192.168.1.1", maxReqs, window)

	if !allowed {
		t.Fatal("expected first request to be allowed")
	}
	if remaining != maxReqs-1 {
		t.Fatalf("expected remaining=%d, got %d", maxReqs-1, remaining)
	}
	if resetAt.Before(time.Now()) {
		t.Fatal("expected resetAt to be in the future")
	}
}

func TestAllow_RequestsUpToLimitAllowed(t *testing.T) {
	store := NewRateLimiterStore()
	maxReqs := 3
	window := time.Minute
	key := "10.0.0.1"

	for i := 1; i <= maxReqs; i++ {
		allowed, remaining, _ := store.Allow(key, maxReqs, window)
		if !allowed {
			t.Fatalf("request %d should be allowed", i)
		}
		wantRemaining := maxReqs - i
		if remaining != wantRemaining {
			t.Fatalf("request %d: expected remaining=%d, got %d", i, wantRemaining, remaining)
		}
	}
}

func TestAllow_ExceedingLimitRejected(t *testing.T) {
	store := NewRateLimiterStore()
	maxReqs := 3
	window := time.Minute
	key := "10.0.0.2"

	// Exhaust the limit
	for i := 0; i < maxReqs; i++ {
		store.Allow(key, maxReqs, window)
	}

	// Next request should be rejected
	allowed, remaining, _ := store.Allow(key, maxReqs, window)
	if allowed {
		t.Fatal("expected request exceeding limit to be rejected")
	}
	if remaining != 0 {
		t.Fatalf("expected remaining=0 on rejection, got %d", remaining)
	}
}

func TestAllow_ExpiredWindowResetsCounter(t *testing.T) {
	store := NewRateLimiterStore()
	maxReqs := 2
	window := 10 * time.Millisecond
	key := "10.0.0.3"

	// Exhaust the limit
	for i := 0; i < maxReqs; i++ {
		store.Allow(key, maxReqs, window)
	}

	// Confirm rejected
	allowed, _, _ := store.Allow(key, maxReqs, window)
	if allowed {
		t.Fatal("expected rejection before window expires")
	}

	// Wait for window to expire
	time.Sleep(15 * time.Millisecond)

	// Should be allowed again with a fresh counter
	allowed, remaining, _ := store.Allow(key, maxReqs, window)
	if !allowed {
		t.Fatal("expected request to be allowed after window expired")
	}
	if remaining != maxReqs-1 {
		t.Fatalf("expected remaining=%d after reset, got %d", maxReqs-1, remaining)
	}
}

func TestAllow_DifferentKeysIndependent(t *testing.T) {
	store := NewRateLimiterStore()
	maxReqs := 1
	window := time.Minute

	// Exhaust limit for key A
	store.Allow("keyA", maxReqs, window)
	allowed, _, _ := store.Allow("keyA", maxReqs, window)
	if allowed {
		t.Fatal("keyA should be rejected after exceeding limit")
	}

	// Key B should still be allowed
	allowed, remaining, _ := store.Allow("keyB", maxReqs, window)
	if !allowed {
		t.Fatal("keyB should be allowed independently of keyA")
	}
	if remaining != 0 {
		t.Fatalf("expected remaining=0 for keyB (maxReqs=1), got %d", remaining)
	}
}

func TestRateLimitByIP_AllowedRequestReturnsHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewRateLimiterStore()
	maxReqs := 5
	window := time.Minute

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", RateLimitByIP(store, maxReqs, window, "too many requests"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Fatalf("expected X-RateLimit-Limit=5, got %q", got)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "4" {
		t.Fatalf("expected X-RateLimit-Remaining=4, got %q", got)
	}
}

func TestRateLimitByIP_RejectedRequestReturns429WithBodyAndHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewRateLimiterStore()
	maxReqs := 1
	window := time.Minute

	handler := RateLimitByIP(store, maxReqs, window, "too many requests")

	// First request — allowed, exhausts the limit
	w1 := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w1)
	r.GET("/test", handler, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.ServeHTTP(w1, httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody))

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}

	// Second request — should be rejected
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody))

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", w2.Code)
	}

	// Check JSON body
	body := w2.Body.String()
	expected := `{"error":"too many requests"}`
	if body != expected {
		t.Fatalf("expected body %s, got %s", expected, body)
	}

	// Check rejection headers
	if got := w2.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header on rejection")
	}
	if got := w2.Header().Get("X-RateLimit-Reset"); got == "" {
		t.Fatal("expected X-RateLimit-Reset header on rejection")
	}
	if got := w2.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("expected X-RateLimit-Remaining=0 on rejection, got %q", got)
	}
}

func TestRateLimitByIP_RemainingDecreasesWithEachRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewRateLimiterStore()
	maxReqs := 3
	window := time.Minute

	_, r := gin.CreateTestContext(httptest.NewRecorder())
	handler := RateLimitByIP(store, maxReqs, window, "too many requests")
	r.GET("/test", handler, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := range maxReqs {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody))

		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}

		wantRemaining := strconv.Itoa(maxReqs - 1 - i)
		if got := w.Header().Get("X-RateLimit-Remaining"); got != wantRemaining {
			t.Fatalf("request %d: expected X-RateLimit-Remaining=%s, got %s", i+1, wantRemaining, got)
		}
	}
}

func TestRateLimitByOrganization_AllowedRequestReturnsHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewRateLimiterStore()
	maxReqs := 5
	window := time.Minute

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", func(c *gin.Context) {
		c.Set(ClaimsKey, &token.Claims{OrganizationID: "organization-123"})
		c.Next()
	}, RateLimitByOrganization(store, maxReqs, window), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Fatalf("expected X-RateLimit-Limit=5, got %q", got)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "4" {
		t.Fatalf("expected X-RateLimit-Remaining=4, got %q", got)
	}
}

func TestRateLimitByOrganization_RejectedRequestReturns429WithBodyAndHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewRateLimiterStore()
	maxReqs := 1
	window := time.Minute

	setClaims := func(c *gin.Context) {
		c.Set(ClaimsKey, &token.Claims{OrganizationID: "organization-456"})
		c.Next()
	}
	handler := RateLimitByOrganization(store, maxReqs, window)

	// First request — allowed, exhausts the limit
	w1 := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w1)
	r.GET("/test", setClaims, handler, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.ServeHTTP(w1, httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody))

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}

	// Second request — should be rejected
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody))

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", w2.Code)
	}

	// Check JSON body
	body := w2.Body.String()
	expected := `{"error":"too many requests, try again later"}`
	if body != expected {
		t.Fatalf("expected body %s, got %s", expected, body)
	}

	// Check rejection headers
	if got := w2.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header on rejection")
	}
	if got := w2.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("expected X-RateLimit-Remaining=0 on rejection, got %q", got)
	}
}

func TestRateLimitByOrganization_NoClaims_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewRateLimiterStore()
	maxReqs := 5
	window := time.Minute

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	// No claims set in context — middleware should return 401
	r.GET("/test", RateLimitByOrganization(store, maxReqs, window), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	body := w.Body.String()
	expected := `{"error":"unauthorized"}`
	if body != expected {
		t.Fatalf("expected body %s, got %s", expected, body)
	}
}
