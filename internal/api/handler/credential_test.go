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

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// credentialServicer is a test-only interface matching CredentialService methods.
type credentialServicer interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, string, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.CredentialRecord, error)
	GetByID(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, error)
	Revoke(ctx context.Context, orgID, credentialID uuid.UUID) error
	Rotate(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, string, error)
}

// mockCredentialService is a test double for credentialServicer.
type mockCredentialService struct {
	createFn  func(ctx context.Context, orgID uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, string, error)
	listFn    func(ctx context.Context, orgID uuid.UUID) ([]domain.CredentialRecord, error)
	getByIDFn func(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, error)
	revokeFn  func(ctx context.Context, orgID, credentialID uuid.UUID) error
	rotateFn  func(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, string, error)
}

func (m *mockCredentialService) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return nil, "", nil
}
func (m *mockCredentialService) List(ctx context.Context, orgID uuid.UUID) ([]domain.CredentialRecord, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID)
	}
	return nil, nil
}
func (m *mockCredentialService) GetByID(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, credentialID)
	}
	return nil, domain.ErrCredentialNotFound
}
func (m *mockCredentialService) Revoke(ctx context.Context, orgID, credentialID uuid.UUID) error {
	if m.revokeFn != nil {
		return m.revokeFn(ctx, orgID, credentialID)
	}
	return nil
}
func (m *mockCredentialService) Rotate(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, string, error) {
	if m.rotateFn != nil {
		return m.rotateFn(ctx, orgID, credentialID)
	}
	return nil, "", domain.ErrCredentialNotFound
}

// testCredentialHandler wraps a mock service for testing.
type testCredentialHandler struct {
	svc credentialServicer
}

func newTestCredentialHandler(svc credentialServicer) *testCredentialHandler {
	return &testCredentialHandler{svc: svc}
}

func (h *testCredentialHandler) RegisterRoutes(rg *gin.RouterGroup) {
	creds := rg.Group("/credentials")
	creds.POST("", h.Create)
	creds.GET("", h.List)
	creds.GET("/:credential_id", h.GetByID)
	creds.DELETE("/:credential_id", h.Revoke)
	creds.POST("/:credential_id/rotate", h.Rotate)
}

func (h *testCredentialHandler) Create(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	var req createCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}
	record, rawKey, err := h.svc.Create(c.Request.Context(), orgID, domain.CreateCredentialInput{
		Name:        req.Name,
		Environment: req.Environment,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, createCredentialResponse{
		credentialResponse: toCredentialResponse(record),
		RawKey:             rawKey,
		Warning:            credentialWarning,
	})
}

func (h *testCredentialHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	creds, err := h.svc.List(c.Request.Context(), orgID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	resp := make([]credentialResponse, len(creds))
	for i := range creds {
		resp[i] = toCredentialResponse(&creds[i])
	}
	c.JSON(http.StatusOK, resp)
}

func (h *testCredentialHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	credID, ok := dto.CredentialID(c)
	if !ok {
		return
	}
	record, err := h.svc.GetByID(c.Request.Context(), orgID, credID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			_ = c.Error(apierr.NotFound("credential not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toCredentialResponse(record))
}

func (h *testCredentialHandler) Revoke(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	credID, ok := dto.CredentialID(c)
	if !ok {
		return
	}
	err := h.svc.Revoke(c.Request.Context(), orgID, credID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			_ = c.Error(apierr.NotFound("credential not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *testCredentialHandler) Rotate(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	credID, ok := dto.CredentialID(c)
	if !ok {
		return
	}
	record, rawKey, err := h.svc.Rotate(c.Request.Context(), orgID, credID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			_ = c.Error(apierr.NotFound("credential not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, createCredentialResponse{
		credentialResponse: toCredentialResponse(record),
		RawKey:             rawKey,
		Warning:            credentialWarning,
	})
}

func setupCredentialRouter(svc credentialServicer, organizationID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := newTestCredentialHandler(svc)
	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaims(organizationID))
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

// ============================================================
// Unit Tests
// ============================================================

func TestCredentialCreate_ValidPayload_Returns201(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().Truncate(time.Second)
	credID := uuid.New()
	svc := &mockCredentialService{
		createFn: func(_ context.Context, _ uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, string, error) {
			return &domain.CredentialRecord{
				ID: credID, OrganizationID: orgID,
				Name: input.Name, KeyPrefix: "I9LIVE_abc",
				Environment: input.Environment, CreatedAt: now,
			}, "I9LIVE_abcdefghij1234567890", nil
		},
	}
	router := setupCredentialRouter(svc, orgID.String())

	body := `{"name":"My Key","environment":"production"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/credentials", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	rawKey, ok := resp["raw_key"].(string)
	if !ok || rawKey == "" {
		t.Fatalf("expected non-empty raw_key, got %v", resp["raw_key"])
	}
	warning, ok := resp["warning"].(string)
	if !ok || warning == "" {
		t.Fatalf("expected non-empty warning, got %v", resp["warning"])
	}
}

func TestCredentialCreate_InvalidName_Returns400(t *testing.T) {
	svc := &mockCredentialService{}
	router := setupCredentialRouter(svc, uuid.New().String())

	body := `{"name":"","environment":"production"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/credentials", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialCreate_InvalidEnvironment_Returns400(t *testing.T) {
	svc := &mockCredentialService{}
	router := setupCredentialRouter(svc, uuid.New().String())

	body := `{"name":"My Key","environment":"invalid"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/credentials", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialList_Returns200(t *testing.T) {
	orgID := uuid.New()
	svc := &mockCredentialService{
		listFn: func(_ context.Context, _ uuid.UUID) ([]domain.CredentialRecord, error) {
			return []domain.CredentialRecord{
				{ID: uuid.New(), OrganizationID: orgID, Name: "Key 1", Environment: "test"},
				{ID: uuid.New(), OrganizationID: orgID, Name: "Key 2", Environment: "production"},
			}, nil
		},
	}
	router := setupCredentialRouter(svc, orgID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(resp))
	}
}

func TestCredentialList_Empty_Returns200(t *testing.T) {
	svc := &mockCredentialService{
		listFn: func(_ context.Context, _ uuid.UUID) ([]domain.CredentialRecord, error) {
			return []domain.CredentialRecord{}, nil
		},
	}
	router := setupCredentialRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array, got %d items", len(resp))
	}
}

func TestCredentialGetByID_Found_Returns200(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()
	now := time.Now().Truncate(time.Second)
	svc := &mockCredentialService{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialRecord, error) {
			return &domain.CredentialRecord{
				ID: credID, OrganizationID: orgID,
				Name: "My Key", KeyPrefix: "I9LIVE_abc",
				Environment: "production", CreatedAt: now,
			}, nil
		},
	}
	router := setupCredentialRouter(svc, orgID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/"+credID.String(), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["id"] != credID.String() {
		t.Fatalf("expected id %v, got %v", credID.String(), resp["id"])
	}
}

func TestCredentialGetByID_NotFound_Returns404(t *testing.T) {
	svc := &mockCredentialService{}
	router := setupCredentialRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/"+uuid.New().String(), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "credential not found" {
		t.Fatalf("expected 'credential not found', got %q", resp["error"])
	}
}

func TestCredentialGetByID_InvalidUUID_Returns400(t *testing.T) {
	svc := &mockCredentialService{}
	router := setupCredentialRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/not-a-uuid", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialRevoke_Success_Returns204(t *testing.T) {
	svc := &mockCredentialService{
		revokeFn: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	router := setupCredentialRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/credentials/"+uuid.New().String(), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialRevoke_NotFound_Returns404(t *testing.T) {
	svc := &mockCredentialService{
		revokeFn: func(_ context.Context, _, _ uuid.UUID) error { return domain.ErrCredentialNotFound },
	}
	router := setupCredentialRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/credentials/"+uuid.New().String(), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "credential not found" {
		t.Fatalf("expected 'credential not found', got %q", resp["error"])
	}
}

func TestCredentialRotate_Success_Returns201(t *testing.T) {
	orgID := uuid.New()
	newID := uuid.New()
	now := time.Now().Truncate(time.Second)
	svc := &mockCredentialService{
		rotateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialRecord, string, error) {
			return &domain.CredentialRecord{
				ID: newID, OrganizationID: orgID,
				Name: "Rotated Key", KeyPrefix: "I9LIVE_xyz",
				Environment: "production", CreatedAt: now,
			}, "I9LIVE_xyzabcdefgh1234567890", nil
		},
	}
	router := setupCredentialRouter(svc, orgID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/credentials/"+uuid.New().String()+"/rotate", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	rawKey, ok := resp["raw_key"].(string)
	if !ok || rawKey == "" {
		t.Fatalf("expected non-empty raw_key, got %v", resp["raw_key"])
	}
}

func TestCredentialRotate_NotFound_Returns404(t *testing.T) {
	svc := &mockCredentialService{}
	router := setupCredentialRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/credentials/"+uuid.New().String()+"/rotate", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "credential not found" {
		t.Fatalf("expected 'credential not found', got %q", resp["error"])
	}
}
