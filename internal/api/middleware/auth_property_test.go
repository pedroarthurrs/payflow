package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	i9token "payflow/internal/auth/token"
	"payflow/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// TestProperty_Auth_RandomStringsAlwaysRejected verifies Property 5: arbitrary strings always return 401.
// Feature: critical-tests-coverage, Property 5: random strings always rejected
func TestProperty_Auth_RandomStringsAlwaysRejected(t *testing.T) {
	svc := newTestTokenService("secret")
	r := newAuthEngine(svc)

	rapid.Check(t, func(t *rapid.T) {
		token := rapid.String().Draw(t, "token")

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for random token %q, got %d", token, w.Code)
		}
	})
}

// TestProperty_Auth_ClaimsPreservation verifies Property 6: all claims fields are preserved exactly.
// Feature: critical-tests-coverage, Property 6: claims preservation
func TestProperty_Auth_ClaimsPreservation(t *testing.T) {
	svc := newTestTokenService("secret")

	validRoles := []domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleDeveloper, domain.RoleSupport}
	validAuthTypes := []domain.AuthType{domain.AuthTypeUser, domain.AuthTypeSystem}

	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New().String()
		userID := uuid.New().String()
		role := rapid.SampledFrom(validRoles).Draw(t, "role")
		authType := rapid.SampledFrom(validAuthTypes).Draw(t, "auth_type")

		pair, err := svc.NewPairWithClaims(i9token.ClaimsInput{
			OrganizationID: orgID,
			UserID:         userID,
			Role:           role,
			AuthType:       authType,
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
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if capturedClaims == nil {
			t.Fatal("expected claims in context, got nil")
		}
		if capturedClaims.OrganizationID != orgID {
			t.Errorf("OrganizationID: expected %s, got %s", orgID, capturedClaims.OrganizationID)
		}
		if capturedClaims.UserID != userID {
			t.Errorf("UserID: expected %s, got %s", userID, capturedClaims.UserID)
		}
		if capturedClaims.Role != role {
			t.Errorf("Role: expected %s, got %s", role, capturedClaims.Role)
		}
		if capturedClaims.AuthType != authType {
			t.Errorf("AuthType: expected %s, got %s", authType, capturedClaims.AuthType)
		}
	})
}
