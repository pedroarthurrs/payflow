package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// testAuthHandler mirrors AuthHandler but accepts the authServicer interface.
type testAuthHandler struct {
	svc authServicer
}

func newTestAuthHandler(svc authServicer) *testAuthHandler {
	return &testAuthHandler{svc: svc}
}

func (h *testAuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/select-organization", h.SelectOrganization)
}

func (h *testAuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(apierr.BadRequest(e.Error()))
		return
	}
	if req.Email == "" && req.Password == "" && req.APIKey == "" {
		_ = c.Error(apierr.BadRequest("email and password or api_key required"))
		return
	}
	result, e := h.svc.Login(c.Request.Context(), domain.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
		APIKey:   req.APIKey,
	})
	if e != nil {
		if errors.Is(e, domain.ErrBadRequest) {
			_ = c.Error(apierr.BadRequest(e.Error()))
			return
		}
		_ = c.Error(apierr.Unauthorized("invalid credentials"))
		return
	}
	if result.TokenPair != nil {
		c.JSON(http.StatusOK, result.TokenPair)
		return
	}
	orgs := make([]orgRefResponse, len(result.Organizations))
	for i, m := range result.Organizations {
		orgs[i] = orgRefResponse{
			OrganizationID: m.OrganizationID.String(),
			Name:           m.Name,
			Role:           m.Role,
		}
	}
	c.JSON(http.StatusOK, multiOrgLoginResponse{
		SessionToken:  *result.SessionToken,
		Organizations: orgs,
	})
}

func (h *testAuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(apierr.BadRequest(e.Error()))
		return
	}
	pair, e := h.svc.RefreshToken(req.RefreshToken)
	if e != nil {
		_ = c.Error(apierr.Unauthorized("invalid or expired refresh token"))
		return
	}
	c.JSON(http.StatusOK, pair)
}

func (h *testAuthHandler) SelectOrganization(c *gin.Context) {
	var req selectOrgRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(apierr.BadRequest(e.Error()))
		return
	}
	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid organization_id"))
		return
	}
	pair, e := h.svc.SelectOrganization(c.Request.Context(), req.SessionToken, orgID)
	if e != nil {
		if errors.Is(e, domain.ErrInvalidSessionToken) {
			_ = c.Error(apierr.Unauthorized("invalid or expired session token"))
			return
		}
		if errors.Is(e, domain.ErrForbidden) {
			_ = c.Error(apierr.Forbidden("not a member of that organization"))
			return
		}
		_ = c.Error(apierr.Unauthorized("invalid credentials"))
		return
	}
	c.JSON(http.StatusOK, pair)
}

func setupAuthRouter(svc authServicer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Errors())
	h := newTestAuthHandler(svc)
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func validTokenPair() *token.Pair {
	now := time.Now()
	return &token.Pair{
		AccessToken:           "access-token",
		AccessTokenExpiresAt:  now.Add(15 * time.Minute),
		RefreshToken:          "refresh-token",
		RefreshTokenExpiresAt: now.Add(7 * 24 * time.Hour),
	}
}

// ── Unit Tests ────────────────────────────────────────────────────────────────

func TestAuthLogin_SingleOrg_Returns200WithTokenPair(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ domain.LoginRequest) (*service.LoginResult, error) {
			return &service.LoginResult{TokenPair: validTokenPair()}, nil
		},
	}
	r := setupAuthRouter(svc)

	body := `{"email":"user@example.com","password":"secret"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["access_token"]; !ok {
		t.Error("expected access_token in response")
	}
	if _, ok := resp["refresh_token"]; !ok {
		t.Error("expected refresh_token in response")
	}
}

func TestAuthLogin_MultiOrg_Returns200WithSessionToken(t *testing.T) {
	sessionToken := "session-tok"
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ domain.LoginRequest) (*service.LoginResult, error) {
			return &service.LoginResult{
				SessionToken: &sessionToken,
				Organizations: []domain.AuthMembership{
					{OrganizationID: uuid.New(), Name: "Org A", Role: domain.RoleAdmin},
				},
			}, nil
		},
	}
	r := setupAuthRouter(svc)

	body := `{"email":"user@example.com","password":"secret"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["session_token"] != sessionToken {
		t.Errorf("expected session_token=%s, got %v", sessionToken, resp["session_token"])
	}
	orgs, ok := resp["organizations"].([]any)
	if !ok || len(orgs) == 0 {
		t.Error("expected non-empty organizations array")
	}
}

func TestAuthLogin_InvalidCredentials_Returns401(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ domain.LoginRequest) (*service.LoginResult, error) {
			return nil, domain.ErrUnauthorized
		},
	}
	r := setupAuthRouter(svc)

	body := `{"email":"user@example.com","password":"wrong"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "invalid credentials" {
		t.Errorf("expected 'invalid credentials', got %q", resp["error"])
	}
}

func TestAuthLogin_MissingFields_Returns400(t *testing.T) {
	svc := &mockAuthService{}
	r := setupAuthRouter(svc)

	body := `{}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestAuthLogin_MalformedJSON_Returns400(t *testing.T) {
	svc := &mockAuthService{}
	r := setupAuthRouter(svc)

	body := `{invalid`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthLogin_APIKey_Returns200WithSystemToken(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, req domain.LoginRequest) (*service.LoginResult, error) {
			if req.APIKey == "" {
				return nil, domain.ErrUnauthorized
			}
			return &service.LoginResult{TokenPair: validTokenPair()}, nil
		},
	}
	r := setupAuthRouter(svc)

	body := `{"api_key":"I9LIVE_validkey123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestAuthLogin_InvalidAPIKey_Returns401(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ domain.LoginRequest) (*service.LoginResult, error) {
			return nil, domain.ErrUnauthorized
		},
	}
	r := setupAuthRouter(svc)

	body := `{"api_key":"invalid-key"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthRefresh_ValidToken_Returns200(t *testing.T) {
	svc := &mockAuthService{
		refreshTokenFn: func(_ string) (*token.Pair, error) {
			return validTokenPair(), nil
		},
	}
	r := setupAuthRouter(svc)

	body := `{"refresh_token":"valid-refresh"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["access_token"]; !ok {
		t.Error("expected access_token in response")
	}
}

func TestAuthRefresh_InvalidToken_Returns401(t *testing.T) {
	svc := &mockAuthService{
		refreshTokenFn: func(_ string) (*token.Pair, error) {
			return nil, errors.New("invalid token")
		},
	}
	r := setupAuthRouter(svc)

	body := `{"refresh_token":"bad-token"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "invalid or expired refresh token" {
		t.Errorf("expected 'invalid or expired refresh token', got %q", resp["error"])
	}
}

func TestAuthRefresh_MissingField_Returns400(t *testing.T) {
	svc := &mockAuthService{}
	r := setupAuthRouter(svc)

	body := `{}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestAuthSelectOrg_ValidToken_MemberOrg_Returns200(t *testing.T) {
	orgID := uuid.New()
	svc := &mockAuthService{
		selectOrganizationFn: func(_ context.Context, _ string, _ uuid.UUID) (*token.Pair, error) {
			return validTokenPair(), nil
		},
	}
	r := setupAuthRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"session_token":   "valid-session",
		"organization_id": orgID.String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/select-organization", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestAuthSelectOrg_InvalidSessionToken_Returns401(t *testing.T) {
	orgID := uuid.New()
	svc := &mockAuthService{
		selectOrganizationFn: func(_ context.Context, _ string, _ uuid.UUID) (*token.Pair, error) {
			return nil, domain.ErrInvalidSessionToken
		},
	}
	r := setupAuthRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"session_token":   "bad-session",
		"organization_id": orgID.String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/select-organization", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "invalid or expired session token" {
		t.Errorf("expected 'invalid or expired session token', got %q", resp["error"])
	}
}

func TestAuthSelectOrg_NotMember_Returns403(t *testing.T) {
	orgID := uuid.New()
	svc := &mockAuthService{
		selectOrganizationFn: func(_ context.Context, _ string, _ uuid.UUID) (*token.Pair, error) {
			return nil, domain.ErrForbidden
		},
	}
	r := setupAuthRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"session_token":   "valid-session",
		"organization_id": orgID.String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/select-organization", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "not a member of that organization" {
		t.Errorf("expected 'not a member of that organization', got %q", resp["error"])
	}
}

func TestAuthSelectOrg_InvalidUUID_Returns400(t *testing.T) {
	svc := &mockAuthService{}
	r := setupAuthRouter(svc)

	body, _ := json.Marshal(map[string]string{
		"session_token":   "valid-session",
		"organization_id": "not-a-uuid",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/select-organization", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}
