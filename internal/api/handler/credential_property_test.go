package handler

import (
	"context"
	"encoding/json"
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

// setupCredentialRouterWithRole creates a Gin test router with RBAC enforcement.
func setupCredentialRouterWithRole(svc credentialServicer, organizationID, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := newTestCredentialHandler(svc)
	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaimsAndRole(organizationID, role))

	creds := r.Group("/credentials")
	creds.POST("", middleware.RequireRole(domain.RoleAdmin), h.Create)
	creds.GET("", middleware.RequireRole(domain.RoleAdmin), h.List)
	creds.GET("/:credential_id", middleware.RequireRole(domain.RoleAdmin), h.GetByID)
	creds.DELETE("/:credential_id", middleware.RequireRole(domain.RoleOwner), h.Revoke)
	creds.POST("/:credential_id/rotate", middleware.RequireRole(domain.RoleOwner), h.Rotate)

	return r
}

// withClaimsAndRole injects JWT claims with both OrganizationID and Role into the Gin context.
func withClaimsAndRole(organizationID, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &token.Claims{
			OrganizationID: organizationID,
			Role:           domain.Role(role),
		})
		c.Next()
	}
}

// validCredentialRecord returns a valid CredentialRecord for use in mock responses.
func validCredentialRecord(orgID uuid.UUID) *domain.CredentialRecord {
	return &domain.CredentialRecord{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "Test Key",
		KeyPrefix:      "I9LIVE_abc",
		Environment:    "production",
		CreatedAt:      time.Now().Truncate(time.Second),
	}
}

// TestProperty_Credential_CreateResponseFields verifies P4: required fields in create response.
// Feature: credentials, Property 4: Campos obrigatórios na resposta de criação e rotação
func TestProperty_Credential_CreateResponseFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[A-Za-z0-9]{1,50}`).Draw(t, "name")
		env := rapid.SampledFrom([]string{"test", "production"}).Draw(t, "env")

		orgID := uuid.New()
		prefix := "I9LIVE_abc"
		if env == "test" {
			prefix = "I9TEST_abc"
		}
		rawKey := prefix + "defghij1234567890abcdefghij1234567890abcdefghij1234567890abcde"

		svc := &mockCredentialService{
			createFn: func(_ context.Context, _ uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, string, error) {
				return &domain.CredentialRecord{
					ID:          uuid.New(),
					Name:        input.Name,
					KeyPrefix:   prefix,
					Environment: input.Environment,
					CreatedAt:   time.Now().Truncate(time.Second),
				}, rawKey, nil
			},
		}
		router := setupCredentialRouter(svc, orgID.String())

		body, _ := json.Marshal(map[string]string{"name": name, "environment": env})
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/credentials", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		requiredFields := []string{"id", "name", "key_prefix", "environment", "raw_key", "warning"}
		for _, field := range requiredFields {
			val, ok := resp[field]
			if !ok {
				t.Fatalf("response missing field %q", field)
			}
			s, ok := val.(string)
			if !ok || s == "" {
				t.Fatalf("field %q is empty or not a string: %v", field, val)
			}
		}
	})
}

// TestProperty_Credential_NameValidation verifies P5: name validation.
// Feature: credentials, Property 5: Validação de name
func TestProperty_Credential_NameValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invalidName := rapid.OneOf(
			rapid.Just(""),
			rapid.StringMatching(`[A-Za-z]{51,100}`),
		).Draw(t, "invalid_name")

		svc := &mockCredentialService{}
		router := setupCredentialRouter(svc, uuid.New().String())

		body, _ := json.Marshal(map[string]string{"name": invalidName, "environment": "production"})
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/credentials", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid name %q, got %d (body=%s)", invalidName, w.Code, w.Body.String())
		}
	})
}

// TestProperty_Credential_EnvironmentValidation verifies P6: environment validation.
// Feature: credentials, Property 6: Validação de environment
func TestProperty_Credential_EnvironmentValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invalidEnv := rapid.StringMatching(`[a-z]{1,20}`).Draw(t, "invalid_env")
		if invalidEnv == "test" || invalidEnv == "production" {
			return
		}

		svc := &mockCredentialService{}
		router := setupCredentialRouter(svc, uuid.New().String())

		body, _ := json.Marshal(map[string]string{"name": "My Key", "environment": invalidEnv})
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/credentials", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid environment %q, got %d (body=%s)", invalidEnv, w.Code, w.Body.String())
		}
	})
}

// TestProperty_Credential_ResponseNeverExposesSecrets verifies P9: responses never expose key_hash or raw_key.
// Feature: credentials, Property 9: Respostas nunca expõem key_hash ou raw_key
func TestProperty_Credential_ResponseNeverExposesSecrets(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 5).Draw(t, "n")
		orgID := uuid.New()
		credID := uuid.New()

		records := make([]domain.CredentialRecord, n)
		for i := range records {
			records[i] = domain.CredentialRecord{
				ID:          uuid.New(),
				Name:        "key",
				KeyPrefix:   "I9TEST_abc",
				Environment: "test",
				CreatedAt:   time.Now().Truncate(time.Second),
			}
		}

		svc := &mockCredentialService{
			listFn: func(_ context.Context, _ uuid.UUID) ([]domain.CredentialRecord, error) {
				return records, nil
			},
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialRecord, error) {
				return &domain.CredentialRecord{
					ID:          credID,
					Name:        "key",
					KeyPrefix:   "I9TEST_abc",
					Environment: "test",
					CreatedAt:   time.Now().Truncate(time.Second),
				}, nil
			},
		}
		router := setupCredentialRouter(svc, orgID.String())

		wList := httptest.NewRecorder()
		reqList := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials", http.NoBody)
		router.ServeHTTP(wList, reqList)

		if wList.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d", wList.Code)
		}
		var listResp []map[string]any
		if err := json.Unmarshal(wList.Body.Bytes(), &listResp); err != nil {
			t.Fatalf("list: failed to parse response: %v", err)
		}
		for i, item := range listResp {
			if _, ok := item["key_hash"]; ok {
				t.Fatalf("list item[%d] exposes key_hash", i)
			}
			if _, ok := item["raw_key"]; ok {
				t.Fatalf("list item[%d] exposes raw_key", i)
			}
		}

		wGet := httptest.NewRecorder()
		reqGet := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/"+credID.String(), http.NoBody)
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", wGet.Code)
		}
		var getResp map[string]any
		if err := json.Unmarshal(wGet.Body.Bytes(), &getResp); err != nil {
			t.Fatalf("get: failed to parse response: %v", err)
		}
		if _, ok := getResp["key_hash"]; ok {
			t.Fatal("get response exposes key_hash")
		}
		if _, ok := getResp["raw_key"]; ok {
			t.Fatal("get response exposes raw_key")
		}
	})
}

// TestProperty_Credential_RBACEnforcement verifies P15: RBAC enforcement per endpoint.
// Feature: credentials, Property 15: RBAC por endpoint
func TestProperty_Credential_RBACEnforcement(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()

	svc := &mockCredentialService{
		createFn: func(_ context.Context, _ uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, string, error) {
			return validCredentialRecord(orgID), "I9LIVE_abcdefghij1234567890abcdefghij1234567890abcdefghij1234567890ab", nil
		},
		listFn: func(_ context.Context, _ uuid.UUID) ([]domain.CredentialRecord, error) {
			return []domain.CredentialRecord{}, nil
		},
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialRecord, error) {
			return validCredentialRecord(orgID), nil
		},
		revokeFn: func(_ context.Context, _, _ uuid.UUID) error { return nil },
		rotateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialRecord, string, error) {
			return validCredentialRecord(orgID), "I9LIVE_abcdefghij1234567890abcdefghij1234567890abcdefghij1234567890ab", nil
		},
	}

	type routeCase struct {
		method string
		path   string
	}
	routes := []routeCase{
		{http.MethodPost, "/credentials"},
		{http.MethodGet, "/credentials"},
		{http.MethodGet, "/credentials/" + credID.String()},
		{http.MethodDelete, "/credentials/" + credID.String()},
		{http.MethodPost, "/credentials/" + credID.String() + "/rotate"},
	}

	rapid.Check(t, func(t *rapid.T) {
		role := rapid.SampledFrom([]string{"member", "admin", "owner"}).Draw(t, "role")
		router := setupCredentialRouterWithRole(svc, orgID.String(), role)

		for _, route := range routes {
			var body string
			if route.method == http.MethodPost && route.path == "/credentials" {
				body = `{"name":"Test","environment":"production"}`
			}
			reqBody := strings.NewReader(body)

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), route.method, route.path, reqBody)
			if body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			router.ServeHTTP(w, req)

			switch role {
			case "member":
				if w.Code != http.StatusForbidden {
					t.Fatalf("member: %s %s expected 403, got %d", route.method, route.path, w.Code)
				}
			case "admin":
				isOwnerOnly := (route.method == http.MethodDelete) ||
					(route.method == http.MethodPost && strings.HasSuffix(route.path, "/rotate"))
				if isOwnerOnly {
					if w.Code != http.StatusForbidden {
						t.Fatalf("admin: %s %s expected 403, got %d", route.method, route.path, w.Code)
					}
				} else {
					if w.Code == http.StatusForbidden {
						t.Fatalf("admin: %s %s expected accessible (not 403), got %d", route.method, route.path, w.Code)
					}
				}
			case "owner":
				if w.Code == http.StatusForbidden {
					t.Fatalf("owner: %s %s expected accessible (not 403), got %d", route.method, route.path, w.Code)
				}
			}
		}
	})
}
