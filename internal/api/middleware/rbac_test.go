package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	i9token "i9pay/internal/auth/token"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
	"pgregory.net/rapid"
)

// validRoles holds the three recognized roles for use in generators.
var validRoles = []domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleMember}

// newTestEngine builds a minimal Gin engine (test mode) with a route that
// applies RequireRole(minRole) before returning 200.
func newTestEngine(minRole domain.Role, claims *i9token.Claims) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", func(c *gin.Context) {
		if claims != nil {
			c.Set(ClaimsKey, claims)
		}
	}, RequireRole(minRole), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

// doRequest fires a GET /protected against the engine and returns the status code.
func doRequest(engine *gin.Engine) int {
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	engine.ServeHTTP(w, req)
	return w.Code
}

// TestProperty_RBACHierarchy verifies the RBAC hierarchy:
// For any pair (R1, R2) where RoleLevel(R1) >= RoleLevel(R2), RequireRole(R2) accepts R1 (200).
// For any pair where RoleLevel(R1) < RoleLevel(R2), RequireRole(R2) rejects R1 (403).
func TestProperty_RBACHierarchy(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Draw two roles from the valid set.
		r1 := rapid.SampledFrom(validRoles).Draw(rt, "r1")
		r2 := rapid.SampledFrom(validRoles).Draw(rt, "r2")

		claims := &i9token.Claims{
			Role:     r1,
			AuthType: domain.AuthTypeUser,
		}

		engine := newTestEngine(r2, claims)
		status := doRequest(engine)

		if domain.RoleLevel(r1) >= domain.RoleLevel(r2) {
			if status != http.StatusOK {
				rt.Fatalf("RequireRole(%s) rejected role %s (level %d >= %d): got %d, want 200",
					r2, r1, domain.RoleLevel(r1), domain.RoleLevel(r2), status)
			}
		} else {
			if status != http.StatusForbidden {
				rt.Fatalf("RequireRole(%s) accepted role %s (level %d < %d): got %d, want 403",
					r2, r1, domain.RoleLevel(r1), domain.RoleLevel(r2), status)
			}
		}
	})
}

// TestRequireRole_MissingClaims verifies that a request without claims in context → 401.
func TestRequireRole_MissingClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", RequireRole(domain.RoleMember), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing claims, got %d", w.Code)
	}
}

// TestRequireRole_SystemToken verifies that a system token (empty role) → 403 "user role required".
func TestRequireRole_SystemToken(t *testing.T) {
	claims := &i9token.Claims{
		Role:     "",
		AuthType: domain.AuthTypeSystem,
	}
	engine := newTestEngine(domain.RoleMember, claims)
	status := doRequest(engine)

	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for system token, got %d", status)
	}
}
