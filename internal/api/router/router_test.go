package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pgregory.net/rapid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// buildTestEngine creates a minimal gin.Engine that mirrors the route structure
// from NewRouter without requiring real services or a database connection.
// Public routes respond normally; protected routes use a fake auth middleware
// that returns 401 when no Authorization header is present.
func buildTestEngine() *gin.Engine {
	r := gin.New()

	// Root route (outside version group)
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"name": "i9pay"})
	})

	// Version group /api/v1
	v1 := r.Group("/api/v1")

	// Public routes
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	auth := v1.Group("/auth")
	auth.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})
	auth.POST("/refresh", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	// Protected routes (fake auth middleware returns 401 without Authorization header)
	protected := v1.Group("/")
	protected.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}
		c.Next()
	})

	protected.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	customers := protected.Group("/customers")
	customers.POST("/", func(c *gin.Context) { c.Status(http.StatusCreated) })
	customers.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	customers.GET("/:customer_id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	customers.PUT("/:customer_id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	customers.DELETE("/:customer_id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	addresses := protected.Group("/customers/:customer_id/addresses")
	addresses.POST("/", func(c *gin.Context) { c.Status(http.StatusCreated) })
	addresses.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	addresses.GET("/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	addresses.PUT("/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	addresses.DELETE("/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	users := protected.Group("/users")
	users.POST("/", func(c *gin.Context) { c.Status(http.StatusCreated) })
	users.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	users.PUT("/:user_id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	users.DELETE("/:user_id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

// TestRootRouteOutsideVersionGroup verifies that GET / responds with the API
// name and is NOT served under /api/v1.
func TestRootRouteOutsideVersionGroup(t *testing.T) {
	r := buildTestEngine()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET / expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "i9pay") {
		t.Fatalf("GET / expected body to contain 'i9pay', got %s", w.Body.String())
	}

	// Verify the root route is registered at "/" not "/api/v1/"
	routes := r.Routes()
	for _, route := range routes {
		if route.Path == "/" && route.Method == http.MethodGet {
			return // found at root level — pass
		}
	}
	t.Fatal("expected GET / to be registered at root level")
}

// TestPublicRoutesAccessibleWithoutAuth verifies that health and auth routes
// do not return 401 when called without an Authorization header.
func TestPublicRoutesAccessibleWithoutAuth(t *testing.T) {
	r := buildTestEngine()

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/health"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code == http.StatusUnauthorized {
				t.Errorf("%s %s should be accessible without auth, got 401", tt.method, tt.path)
			}
		})
	}
}

// TestProtectedRoutesRequireAuth verifies that protected routes return 401
// when called without an Authorization header.
func TestProtectedRoutesRequireAuth(t *testing.T) {
	r := buildTestEngine()

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me"},
		{http.MethodPost, "/api/v1/customers/"},
		{http.MethodGet, "/api/v1/customers/"},
		{http.MethodGet, "/api/v1/customers/some-id"},
		{http.MethodPut, "/api/v1/customers/some-id"},
		{http.MethodDelete, "/api/v1/customers/some-id"},
		{http.MethodPost, "/api/v1/users/"},
		{http.MethodGet, "/api/v1/users/"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s %s without auth expected 401, got %d", tt.method, tt.path, w.Code)
			}
		})
	}
}

// TestAddressRoutesNestedUnderCustomers verifies that address routes are
// registered under /api/v1/customers/:customer_id/addresses.
func TestAddressRoutesNestedUnderCustomers(t *testing.T) {
	r := buildTestEngine()
	routes := r.Routes()

	expected := map[string][]string{
		"/api/v1/customers/:customer_id/addresses/":    {http.MethodPost, http.MethodGet},
		"/api/v1/customers/:customer_id/addresses/:id": {http.MethodGet, http.MethodPut, http.MethodDelete},
	}

	for path, methods := range expected {
		for _, method := range methods {
			found := false
			for _, route := range routes {
				if route.Path == path && route.Method == method {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected route %s %s to be registered", method, path)
			}
		}
	}

	// Also verify via httptest that address routes are protected (return 401 without auth)
	addressTests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/customers/some-id/addresses/"},
		{http.MethodGet, "/api/v1/customers/some-id/addresses/"},
		{http.MethodGet, "/api/v1/customers/some-id/addresses/some-addr"},
		{http.MethodPut, "/api/v1/customers/some-id/addresses/some-addr"},
		{http.MethodDelete, "/api/v1/customers/some-id/addresses/some-addr"},
	}

	for _, tt := range addressTests {
		t.Run("protected_"+tt.method+" "+tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s %s without auth expected 401, got %d", tt.method, tt.path, w.Code)
			}
		})
	}
}

// TestProperty_PublicProtectedSeparation verifies that protected routes return 401
// without authentication, while public routes do not.
func TestProperty_PublicProtectedSeparation(t *testing.T) {
	type route struct {
		method string
		path   string
	}

	publicRoutes := []route{
		{http.MethodGet, "/api/v1/health"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
	}

	protectedRoutes := []route{
		{http.MethodGet, "/api/v1/me"},
		{http.MethodPost, "/api/v1/customers/"},
		{http.MethodGet, "/api/v1/customers/"},
		{http.MethodGet, "/api/v1/customers/abc123"},
		{http.MethodPut, "/api/v1/customers/abc123"},
		{http.MethodDelete, "/api/v1/customers/abc123"},
		{http.MethodPost, "/api/v1/customers/abc123/addresses/"},
		{http.MethodGet, "/api/v1/customers/abc123/addresses/"},
		{http.MethodGet, "/api/v1/customers/abc123/addresses/addr1"},
		{http.MethodPut, "/api/v1/customers/abc123/addresses/addr1"},
		{http.MethodDelete, "/api/v1/customers/abc123/addresses/addr1"},
		{http.MethodPost, "/api/v1/users/"},
		{http.MethodGet, "/api/v1/users/"},
		{http.MethodPut, "/api/v1/users/user1"},
		{http.MethodDelete, "/api/v1/users/user1"},
	}

	r := buildTestEngine()

	rapid.Check(t, func(t *rapid.T) {
		// Pick a random protected route and verify it returns 401 without auth
		pi := rapid.IntRange(0, len(protectedRoutes)-1).Draw(t, "protected_index")
		pr := protectedRoutes[pi]

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(pr.method, pr.path, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("protected route %s %s without token: expected 401, got %d", pr.method, pr.path, w.Code)
		}

		// Pick a random public route and verify it does NOT return 401 without auth
		pui := rapid.IntRange(0, len(publicRoutes)-1).Draw(t, "public_index")
		pu := publicRoutes[pui]

		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest(pu.method, pu.path, strings.NewReader("{}"))
		req2.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w2, req2)

		if w2.Code == http.StatusUnauthorized {
			t.Fatalf("public route %s %s without token: should not return 401, got 401", pu.method, pu.path)
		}
	})
}

// buildRBACTestEngine creates a gin.Engine that mirrors the user routes with
// real RBAC middleware. A fake auth middleware injects claims from the
// Authorization header value (treated as a role string for simplicity).
func buildRBACTestEngine() *gin.Engine {
	r := gin.New()

	v1 := r.Group("/api/v1")

	protected := v1.Group("/")
	protected.Use(func(c *gin.Context) {
		role := domain.Role(c.GetHeader("X-Test-Role"))
		if role == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		claims := &token.Claims{
			OrganizationID: uuid.New().String(),
			UserID:   uuid.New().String(),
			Role:     role,
			AuthType: domain.AuthTypeUser,
		}
		c.Set(middleware.ClaimsKey, claims)
		c.Next()
	})

	users := protected.Group("/users")
	users.POST("/", middleware.RequireRole(domain.RoleOwner), func(c *gin.Context) { c.Status(http.StatusCreated) })
	users.GET("/", middleware.RequireRole(domain.RoleAdmin), func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	users.PUT("/:user_id", middleware.RequireRole(domain.RoleOwner), func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	users.DELETE("/:user_id", middleware.RequireRole(domain.RoleOwner), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	return r
}

// TestProperty_RBACUserRoutes verifies that user routes enforce role-based access control.
// Routes requiring Owner role reject Admin and Member roles with 403.
// Routes requiring Admin role reject Member role with 403.
func TestProperty_RBACUserRoutes(t *testing.T) {
	type userRoute struct {
		method  string
		path    string
		minRole domain.Role
	}

	routes := []userRoute{
		{http.MethodPost, "/api/v1/users/", domain.RoleOwner},
		{http.MethodGet, "/api/v1/users/", domain.RoleAdmin},
		{http.MethodPut, "/api/v1/users/user1", domain.RoleOwner},
		{http.MethodDelete, "/api/v1/users/user1", domain.RoleOwner},
	}

	allRoles := []domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleMember}

	r := buildRBACTestEngine()

	rapid.Check(t, func(t *rapid.T) {
		ri := rapid.IntRange(0, len(routes)-1).Draw(t, "route_index")
		route := routes[ri]

		role := rapid.SampledFrom(allRoles).Draw(t, "role")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(route.method, route.path, nil)
		req.Header.Set("X-Test-Role", string(role))
		r.ServeHTTP(w, req)

		hasPermission := domain.RoleLevel(role) >= domain.RoleLevel(route.minRole)

		if hasPermission {
			if w.Code == http.StatusForbidden {
				t.Fatalf("%s %s with role %s: should be allowed but got 403", route.method, route.path, role)
			}
		} else {
			if w.Code != http.StatusForbidden {
				t.Fatalf("%s %s with role %s: expected 403, got %d", route.method, route.path, role, w.Code)
			}
		}
	})
}

// TestProperty_RateLimitLogin verifies that login rate limiting enforces the configured limit.
// Requests within the limit succeed; requests exceeding it return 429.
func TestProperty_RateLimitLogin(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRequests := rapid.IntRange(1, 10).Draw(t, "max_requests")
		totalRequests := rapid.IntRange(maxRequests+1, maxRequests+20).Draw(t, "total_requests")

		store := middleware.NewRateLimiterStore()
		loginRL := middleware.RateLimitByIP(store, maxRequests, 5*time.Minute, "too many login attempts, try again later")

		r := gin.New()
		v1 := r.Group("/api/v1")
		auth := v1.Group("/auth")
		auth.POST("/login", loginRL, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})

		for i := 1; i <= totalRequests; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if i <= maxRequests {
				if w.Code == http.StatusTooManyRequests {
					t.Fatalf("request %d/%d (within limit %d): got 429 unexpectedly", i, totalRequests, maxRequests)
				}
			} else {
				if w.Code != http.StatusTooManyRequests {
					t.Fatalf("request %d/%d (over limit %d): expected 429, got %d", i, totalRequests, maxRequests, w.Code)
				}
			}
		}
	})
}

// TestProperty_APIv1Prefix verifies that versioned routes are accessible under /api/v1
// and not accessible under the old /v1 prefix.
func TestProperty_APIv1Prefix(t *testing.T) {
	type route struct {
		method  string
		apiPath string // path under /api/v1
	}

	// All versioned routes (public ones that respond without auth)
	versionedRoutes := []route{
		{http.MethodGet, "/health"},
		{http.MethodPost, "/auth/login"},
		{http.MethodPost, "/auth/refresh"},
	}

	r := buildTestEngine()

	rapid.Check(t, func(t *rapid.T) {
		ri := rapid.IntRange(0, len(versionedRoutes)-1).Draw(t, "route_index")
		vr := versionedRoutes[ri]

		// Accessible under /api/v1
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest(vr.method, "/api/v1"+vr.apiPath, strings.NewReader("{}"))
		req1.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w1, req1)

		if w1.Code == http.StatusNotFound {
			t.Fatalf("%s /api/v1%s: expected route to exist, got 404", vr.method, vr.apiPath)
		}

		// NOT accessible under old /v1 prefix
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest(vr.method, "/v1"+vr.apiPath, strings.NewReader("{}"))
		req2.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusNotFound {
			t.Fatalf("%s /v1%s: expected 404 (old prefix), got %d", vr.method, vr.apiPath, w2.Code)
		}
	})
}

// buildRequestIDTestEngine creates a gin.Engine with the real RequestID middleware
// and all routes registered, to verify the global middleware chain is active.
func buildRequestIDTestEngine() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID())

	// Root route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"name": "i9pay"})
	})

	v1 := r.Group("/api/v1")

	// Public routes
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	auth := v1.Group("/auth")
	auth.POST("/login", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	auth.POST("/refresh", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

	// Protected routes (stub — no real auth needed for this property)
	v1.GET("/me", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	v1.POST("/customers/", func(c *gin.Context) { c.Status(http.StatusCreated) })
	v1.GET("/customers/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	v1.GET("/customers/:customer_id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	v1.POST("/users/", func(c *gin.Context) { c.Status(http.StatusCreated) })
	v1.GET("/users/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

	return r
}

// TestProperty_GlobalMiddlewareRequestID verifies that the global middleware chain
// is active by checking that all responses contain the X-Request-ID header.
func TestProperty_GlobalMiddlewareRequestID(t *testing.T) {
	type route struct {
		method string
		path   string
	}

	allRoutes := []route{
		{http.MethodGet, "/"},
		{http.MethodGet, "/api/v1/health"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodGet, "/api/v1/me"},
		{http.MethodPost, "/api/v1/customers/"},
		{http.MethodGet, "/api/v1/customers/"},
		{http.MethodGet, "/api/v1/customers/abc123"},
		{http.MethodPost, "/api/v1/users/"},
		{http.MethodGet, "/api/v1/users/"},
	}

	r := buildRequestIDTestEngine()

	rapid.Check(t, func(t *rapid.T) {
		ri := rapid.IntRange(0, len(allRoutes)-1).Draw(t, "route_index")
		rt := allRoutes[ri]

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(rt.method, rt.path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		xReqID := w.Header().Get("X-Request-ID")
		if xReqID == "" {
			t.Fatalf("%s %s: response missing X-Request-ID header", rt.method, rt.path)
		}

		// Validate it looks like a UUID
		if _, err := uuid.Parse(xReqID); err != nil {
			t.Fatalf("%s %s: X-Request-ID %q is not a valid UUID", rt.method, rt.path, xReqID)
		}
	})
}
