package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"payflow/internal/api/middleware"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── mock repositories ─────────────────────────────────────────────────────────

type mockWebhookEndpointRepo struct {
	createFn         func(ctx context.Context, orgID uuid.UUID, input domain.CreateWebhookEndpointInput) (*domain.WebhookEndpoint, error)
	getByIDFn        func(ctx context.Context, orgID, endpointID uuid.UUID) (*domain.WebhookEndpoint, error)
	listFn           func(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.WebhookEndpoint, error)
	updateFn         func(ctx context.Context, orgID, endpointID uuid.UUID, input domain.UpdateWebhookEndpointInput) (*domain.WebhookEndpoint, error)
	softDeleteFn     func(ctx context.Context, orgID, endpointID uuid.UUID) error
	listActiveByEvFn func(ctx context.Context, orgID uuid.UUID, eventType, env string) ([]domain.WebhookEndpoint, error)
}

func (m *mockWebhookEndpointRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return nil, nil
}

func (m *mockWebhookEndpointRepo) GetByID(ctx context.Context, orgID, endpointID uuid.UUID) (*domain.WebhookEndpoint, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, endpointID)
	}
	return nil, domain.ErrWebhookEndpointNotFound
}

func (m *mockWebhookEndpointRepo) List(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.WebhookEndpoint, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, params)
	}
	return nil, nil
}

func (m *mockWebhookEndpointRepo) Update(ctx context.Context, orgID, endpointID uuid.UUID, input domain.UpdateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, orgID, endpointID, input)
	}
	return nil, domain.ErrWebhookEndpointNotFound
}

func (m *mockWebhookEndpointRepo) SoftDelete(ctx context.Context, orgID, endpointID uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, orgID, endpointID)
	}
	return domain.ErrWebhookEndpointNotFound
}
func (m *mockWebhookEndpointRepo) ListActiveByEvent(ctx context.Context, orgID uuid.UUID, eventType, env string) ([]domain.WebhookEndpoint, error) {
	if m.listActiveByEvFn != nil {
		return m.listActiveByEvFn(ctx, orgID, eventType, env)
	}
	return nil, nil
}

type mockWebhookLogRepo struct {
	createFn  func(ctx context.Context, input domain.CreateWebhookLogInput) error
	getByIDFn func(ctx context.Context, orgID, logID uuid.UUID) (*domain.WebhookLog, error)
	listFn    func(ctx context.Context, orgID uuid.UUID, input domain.ListWebhookLogsInput) ([]domain.WebhookLog, error)
}

func (m *mockWebhookLogRepo) Create(ctx context.Context, input domain.CreateWebhookLogInput) error {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	return nil
}

func (m *mockWebhookLogRepo) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.WebhookLog, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, logID)
	}
	return nil, domain.ErrWebhookLogNotFound
}

func (m *mockWebhookLogRepo) List(ctx context.Context, orgID uuid.UUID, input domain.ListWebhookLogsInput) ([]domain.WebhookLog, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, input)
	}
	return nil, nil
}

// ── router setup ──────────────────────────────────────────────────────────────

func setupWebhookRouter(epRepo *mockWebhookEndpointRepo, logRepo *mockWebhookLogRepo, orgID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.NewWebhookService(epRepo, logRepo)
	h := NewWebhookHandler(svc)
	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaimsAndRole(orgID, "admin"))
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func setupWebhookRouterWithRole(epRepo *mockWebhookEndpointRepo, logRepo *mockWebhookLogRepo, orgID, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.NewWebhookService(epRepo, logRepo)
	h := NewWebhookHandler(svc)
	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaimsAndRole(orgID, role))
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

// ── Create tests ──────────────────────────────────────────────────────────────

func TestWebhookCreate_ValidPayload_Returns201(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	whID := uuid.New()

	epRepo := &mockWebhookEndpointRepo{
		createFn: func(_ context.Context, oID uuid.UUID, input domain.CreateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
			return &domain.WebhookEndpoint{
				ID:             whID,
				OrganizationID: oID,
				URL:            input.URL,
				Description:    input.Description,
				EnabledEvents:  input.EnabledEvents,
				Environment:    input.Environment,
				IsActive:       true,
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	}
	r := setupWebhookRouter(epRepo, &mockWebhookLogRepo{}, orgID.String())

	body := `{"url":"https://example.com/hook","enabled_events":["customer.created"]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["id"] != whID.String() {
		t.Fatalf("expected id %s, got %v", whID.String(), resp["id"])
	}
	if resp["is_active"] != true {
		t.Fatalf("expected is_active true, got %v", resp["is_active"])
	}
	if resp["environment"] != "test" {
		t.Fatalf("expected environment test, got %v", resp["environment"])
	}
}

func TestWebhookCreate_MissingURL_Returns400(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	body := `{"enabled_events":["customer.created"]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookCreate_NonHTTPS_Returns400(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	body := `{"url":"http://example.com/hook","enabled_events":["customer.created"]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookCreate_EmptyEnabledEvents_Returns400(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	body := `{"url":"https://example.com/hook","enabled_events":[]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookCreate_InvalidEventType_Returns400(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{
		createFn: func(_ context.Context, _ uuid.UUID, _ domain.CreateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
			t.Fatal("create should not be called for invalid event types")
			return nil, nil
		},
	}, &mockWebhookLogRepo{}, uuid.New().String())

	body := `{"url":"https://example.com/hook","enabled_events":["invalid.event"]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookCreate_DescriptionOver255_Returns400(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	longDesc := strings.Repeat("a", 256)
	body := `{"url":"https://example.com/hook","enabled_events":["customer.created"],"description":"` + longDesc + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookCreate_InvalidEnvironment_Returns400(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	body := `{"url":"https://example.com/hook","enabled_events":["customer.created"],"environment":"staging"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// ── ListEvents tests ──────────────────────────────────────────────────────────

func TestWebhookListEvents_Returns9EventTypes(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/webhooks/events", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string][]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	events := resp["events"]
	if len(events) != 9 {
		t.Fatalf("expected 9 event types, got %d: %v", len(events), events)
	}

	expected := map[string]bool{
		"customer.created":       true,
		"customer.updated":       true,
		"customer.deleted":       true,
		"invoice.created":        true,
		"invoice.paid":           true,
		"invoice.payment_failed": true,
		"subscription.created":   true,
		"subscription.updated":   true,
		"subscription.canceled":  true,
	}
	for _, e := range events {
		if !expected[e] {
			t.Fatalf("unexpected event type: %s", e)
		}
	}
}

// ── List tests ────────────────────────────────────────────────────────────────

func TestWebhookList_Returns200(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().UTC()

	epRepo := &mockWebhookEndpointRepo{
		listFn: func(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.WebhookEndpoint, error) {
			return []domain.WebhookEndpoint{
				{ID: uuid.New(), OrganizationID: orgID, URL: "https://a.com", EnabledEvents: []string{"customer.created"}, Environment: "test", IsActive: true, CreatedAt: now, UpdatedAt: now},
				{ID: uuid.New(), OrganizationID: orgID, URL: "https://b.com", EnabledEvents: []string{"invoice.paid"}, Environment: "production", IsActive: true, CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}
	r := setupWebhookRouter(epRepo, &mockWebhookLogRepo{}, orgID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/webhooks", http.NoBody)
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
	if len(data) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(data))
	}
}

// ── GetByID tests ─────────────────────────────────────────────────────────────

func TestWebhookGetByID_Found_Returns200(t *testing.T) {
	orgID := uuid.New()
	whID := uuid.New()
	now := time.Now().UTC()

	epRepo := &mockWebhookEndpointRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.WebhookEndpoint, error) {
			return &domain.WebhookEndpoint{
				ID: whID, OrganizationID: orgID, URL: "https://example.com",
				EnabledEvents: []string{"customer.created"}, Environment: "test",
				IsActive: true, CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}
	r := setupWebhookRouter(epRepo, &mockWebhookLogRepo{}, orgID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/webhooks/"+whID.String(), http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["id"] != whID.String() {
		t.Fatalf("expected id %s, got %v", whID.String(), resp["id"])
	}
}

func TestWebhookGetByID_NotFound_Returns404(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/webhooks/"+uuid.New().String(), http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookGetByID_InvalidUUID_Returns400(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/webhooks/not-a-uuid", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// ── Update tests ──────────────────────────────────────────────────────────────

func TestWebhookUpdate_ValidPayload_Returns200(t *testing.T) {
	orgID := uuid.New()
	whID := uuid.New()
	now := time.Now().UTC()
	newURL := "https://updated.com/hook"

	epRepo := &mockWebhookEndpointRepo{
		updateFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, input domain.UpdateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
			url := "https://example.com"
			if input.URL != nil {
				url = *input.URL
			}
			return &domain.WebhookEndpoint{
				ID: whID, OrganizationID: orgID, URL: url,
				EnabledEvents: []string{"customer.created"}, Environment: "test",
				IsActive: true, CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}
	r := setupWebhookRouter(epRepo, &mockWebhookLogRepo{}, orgID.String())

	body := `{"url":"` + newURL + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/webhooks/"+whID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookUpdate_NotFound_Returns404(t *testing.T) {
	r := setupWebhookRouter(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String())

	body := `{"url":"https://updated.com/hook"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/webhooks/"+uuid.New().String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// ── Delete tests ──────────────────────────────────────────────────────────────

func TestWebhookDelete_Success_Returns204(t *testing.T) {
	orgID := uuid.New()
	epRepo := &mockWebhookEndpointRepo{
		softDeleteFn: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	r := setupWebhookRouterWithRole(epRepo, &mockWebhookLogRepo{}, orgID.String(), "owner")

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/webhooks/"+uuid.New().String(), http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookDelete_NotFound_Returns404(t *testing.T) {
	r := setupWebhookRouterWithRole(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String(), "owner")

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/webhooks/"+uuid.New().String(), http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestWebhookDelete_InvalidWebhookID_Returns400(t *testing.T) {
	r := setupWebhookRouterWithRole(&mockWebhookEndpointRepo{}, &mockWebhookLogRepo{}, uuid.New().String(), "owner")

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/webhooks/not-a-uuid", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}
