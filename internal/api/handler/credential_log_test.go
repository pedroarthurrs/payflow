package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── mock service ──────────────────────────────────────────────────────────────

type mockCredentialLogService struct {
	listFn    func(ctx context.Context, orgID uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, *string, error)
	getByIDFn func(ctx context.Context, orgID, logID uuid.UUID) (*domain.CredentialLogRecord, error)
}

func (m *mockCredentialLogService) List(ctx context.Context, orgID uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, *string, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, input)
	}
	return nil, nil, nil
}

func (m *mockCredentialLogService) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.CredentialLogRecord, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, logID)
	}
	return nil, domain.ErrCredentialLogNotFound
}

func setupCredentialLogRouter(svc *mockCredentialLogService, orgID string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := &credentialLogRepoAdapter{svc: svc}
	realSvc := service.NewCredentialService(nil, repo, nil, "test-pepper")

	h := NewCredentialHandler(realSvc)
	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaimsAndRole(orgID, "admin"))

	logs := r.Group("/credentials/logs")
	logs.GET("", h.ListLogs)
	logs.GET("/:log_id", h.GetLogByID)
	return r
}

// credentialLogRepoAdapter adapts mockCredentialLogService to domain.CredentialLogRepository
// so we can use the real service (and its cursor logic) in handler tests.
type credentialLogRepoAdapter struct {
	svc *mockCredentialLogService
}

func (a *credentialLogRepoAdapter) List(ctx context.Context, orgID uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
	if a.svc.listFn != nil {
		records, _, err := a.svc.listFn(ctx, orgID, input)
		return records, err
	}
	return nil, nil
}

func (a *credentialLogRepoAdapter) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.CredentialLogRecord, error) {
	if a.svc.getByIDFn != nil {
		return a.svc.getByIDFn(ctx, orgID, logID)
	}
	return nil, domain.ErrCredentialLogNotFound
}

// ── List tests ────────────────────────────────────────────────────────────────

func TestCredentialLogList_WithData_Returns200(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().UTC()

	svc := &mockCredentialLogService{
		listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, *string, error) {
			return []domain.CredentialLogRecord{
				{ID: uuid.New(), OrganizationID: orgID, CreatedAt: now},
			}, nil, nil
		},
	}
	r := setupCredentialLogRouter(svc, orgID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", resp["data"])
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 record, got %d", len(data))
	}
}

func TestCredentialLogList_Empty_Returns200(t *testing.T) {
	svc := &mockCredentialLogService{}
	r := setupCredentialLogRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data, ok := resp["data"].([]any)
	if !ok || data == nil {
		// empty slice may marshal as null or []
		if resp["data"] != nil {
			t.Fatalf("expected empty data, got %v", resp["data"])
		}
	}
}

func TestCredentialLogList_InvalidLimit_Returns400(t *testing.T) {
	for _, bad := range []string{"0", "101", "-1", "abc"} {
		svc := &mockCredentialLogService{}
		r := setupCredentialLogRouter(svc, uuid.New().String())

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs?limit="+bad, http.NoBody)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("limit=%q: expected 400, got %d (body=%s)", bad, w.Code, w.Body.String())
		}
	}
}

func TestCredentialLogList_InvalidCursor_Returns400(t *testing.T) {
	svc := &mockCredentialLogService{}
	r := setupCredentialLogRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs?cursor=!!!invalid!!!", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialLogList_InvalidFrom_Returns400(t *testing.T) {
	svc := &mockCredentialLogService{}
	r := setupCredentialLogRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs?from=not-a-date", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialLogList_InvalidTo_Returns400(t *testing.T) {
	svc := &mockCredentialLogService{}
	r := setupCredentialLogRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs?to=not-a-date", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialLogList_FromAfterTo_Returns400(t *testing.T) {
	svc := &mockCredentialLogService{}
	r := setupCredentialLogRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		"/credentials/logs?from=2026-02-01T00:00:00Z&to=2026-01-01T00:00:00Z", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialLogList_InvalidStatusCode_Returns400(t *testing.T) {
	svc := &mockCredentialLogService{}
	r := setupCredentialLogRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs?status_code=abc", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// ── GetByID tests ─────────────────────────────────────────────────────────────

func TestCredentialLogGetByID_Found_Returns200(t *testing.T) {
	orgID := uuid.New()
	logID := uuid.New()
	now := time.Now().UTC()

	svc := &mockCredentialLogService{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialLogRecord, error) {
			return &domain.CredentialLogRecord{
				ID:             logID,
				OrganizationID: orgID,
				CreatedAt:      now,
			}, nil
		},
	}
	r := setupCredentialLogRouter(svc, orgID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs/"+logID.String(), http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["id"] != logID.String() {
		t.Fatalf("expected id %v, got %v", logID.String(), resp["id"])
	}
}

func TestCredentialLogGetByID_NotFound_Returns404(t *testing.T) {
	svc := &mockCredentialLogService{}
	r := setupCredentialLogRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs/"+uuid.New().String(), http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestCredentialLogGetByID_InvalidUUID_Returns400(t *testing.T) {
	svc := &mockCredentialLogService{}
	r := setupCredentialLogRouter(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs/not-a-uuid", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}
