package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	i9token "i9pay/internal/auth/token"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
)

func newTestTokenService(secret string) *i9token.Service {
	return i9token.NewService(secret, 15*time.Minute, 7*24*time.Hour)
}

func newAuthEngine(svc *i9token.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Errors())
	r.GET("/protected", Auth(svc), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestAuth_MissingHeader_Returns401(t *testing.T) {
	svc := newTestTokenService("secret")
	r := newAuthEngine(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if body := w.Body.String(); body != `{"error":"authorization header required"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestAuth_InvalidFormat_NoBearer_Returns401(t *testing.T) {
	svc := newTestTokenService("secret")
	r := newAuthEngine(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Token somevalue")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidFormat_BearerOnly_Returns401(t *testing.T) {
	svc := newTestTokenService("secret")
	r := newAuthEngine(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer ")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken_Returns401(t *testing.T) {
	svc := newTestTokenService("secret")
	r := newAuthEngine(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer notavalidtoken")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if body := w.Body.String(); body != `{"error":"invalid or expired token"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestAuth_ExpiredToken_Returns401(t *testing.T) {
	// Use a large negative TTL so the token is well outside the 5s leeway window
	expiredSvc := i9token.NewService("secret", -10*time.Second, 7*24*time.Hour)
	pair, err := expiredSvc.NewPair("org-id")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	svc := newTestTokenService("secret")
	r := newAuthEngine(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuth_ValidAccessToken_StoresClaims(t *testing.T) {
	svc := newTestTokenService("secret")
	pair, err := svc.NewPairWithClaims(i9token.ClaimsInput{
		OrganizationID: "org-123",
		UserID:         "user-456",
		Role:           domain.RoleAdmin,
		AuthType:       domain.AuthTypeUser,
	})
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	var capturedClaims *i9token.Claims
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Errors())
	r.GET("/protected", Auth(svc), func(c *gin.Context) {
		if v, ok := c.Get(ClaimsKey); ok {
			capturedClaims, _ = v.(*i9token.Claims)
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	if capturedClaims == nil {
		t.Fatal("expected claims in context, got nil")
	}
	if capturedClaims.OrganizationID != "org-123" {
		t.Errorf("expected OrganizationID=org-123, got %s", capturedClaims.OrganizationID)
	}
}

func TestAuth_RefreshTokenRejected_Returns401(t *testing.T) {
	svc := newTestTokenService("secret")
	pair, err := svc.NewPair("org-id")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	r := newAuthEngine(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for refresh token used as access, got %d", w.Code)
	}
}

func TestAuth_WrongSecret_Returns401(t *testing.T) {
	wrongSvc := newTestTokenService("wrong-secret")
	pair, err := wrongSvc.NewPair("org-id")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	svc := newTestTokenService("correct-secret")
	r := newAuthEngine(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong secret, got %d", w.Code)
	}
}
