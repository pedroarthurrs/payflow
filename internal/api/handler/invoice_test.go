package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── mock invoice service dependencies ─────────────────────────────────────────

type mockInvoiceRepo struct {
	createFn       func(ctx context.Context, orgID uuid.UUID, input domain.CreateInvoiceInput) (*domain.Invoice, error)
	getByIDFn      func(ctx context.Context, orgID, invoiceID uuid.UUID) (*domain.Invoice, error)
	listFn         func(ctx context.Context, orgID uuid.UUID, params domain.InvoiceListParams) ([]domain.Invoice, error)
	countFn        func(ctx context.Context, orgID uuid.UUID, params domain.InvoiceListParams) (int64, error)
	updateStatusFn func(ctx context.Context, orgID, invoiceID uuid.UUID, newStatus domain.InvoiceStatus) (*domain.Invoice, error)
}

func (m *mockInvoiceRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateInvoiceInput) (*domain.Invoice, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return &domain.Invoice{
		ID: uuid.New(), OrganizationID: orgID, CustomerID: input.CustomerID,
		InvoiceType: input.InvoiceType, AmountCents: input.AmountCents,
		Currency: "BRL", Status: domain.InvStatusPending,
	}, nil
}
func (m *mockInvoiceRepo) GetByID(ctx context.Context, orgID, invoiceID uuid.UUID) (*domain.Invoice, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, invoiceID)
	}
	return &domain.Invoice{ID: invoiceID, OrganizationID: orgID, Status: domain.InvStatusPending}, nil
}
func (m *mockInvoiceRepo) List(ctx context.Context, orgID uuid.UUID, params domain.InvoiceListParams) ([]domain.Invoice, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, params)
	}
	return []domain.Invoice{}, nil
}
func (m *mockInvoiceRepo) Count(ctx context.Context, orgID uuid.UUID, params domain.InvoiceListParams) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, orgID, params)
	}
	return 0, nil
}
func (m *mockInvoiceRepo) UpdateStatus(ctx context.Context, orgID, invoiceID uuid.UUID, newStatus domain.InvoiceStatus) (*domain.Invoice, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, orgID, invoiceID, newStatus)
	}
	return &domain.Invoice{ID: invoiceID, OrganizationID: orgID, Status: newStatus}, nil
}
func (m *mockInvoiceRepo) UpdateERPReference(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockInvoiceRepo) UpdateLastSyncLog(_ context.Context, _, _ uuid.UUID, _ json.RawMessage) error {
	return nil
}

type mockInvoiceItemRepo struct {
	createFn        func(ctx context.Context, orgID, invoiceID uuid.UUID, input domain.CreateInvoiceItemInput) (*domain.InvoiceItem, error)
	listByInvoiceFn func(ctx context.Context, orgID, invoiceID uuid.UUID) ([]domain.InvoiceItem, error)
}

func (m *mockInvoiceItemRepo) Create(ctx context.Context, orgID, invoiceID uuid.UUID, input domain.CreateInvoiceItemInput) (*domain.InvoiceItem, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, invoiceID, input)
	}
	return &domain.InvoiceItem{
		ID: uuid.New(), OrganizationID: orgID, InvoiceID: invoiceID,
		Description: input.Description, Quantity: input.Quantity,
		UnitPriceCents: input.UnitPriceCents, TotalPriceCents: input.TotalPriceCents,
	}, nil
}
func (m *mockInvoiceItemRepo) ListByInvoice(ctx context.Context, orgID, invoiceID uuid.UUID) ([]domain.InvoiceItem, error) {
	if m.listByInvoiceFn != nil {
		return m.listByInvoiceFn(ctx, orgID, invoiceID)
	}
	return nil, nil
}
func (m *mockInvoiceItemRepo) DeleteByInvoice(_ context.Context, _, _ uuid.UUID) error { return nil }

type mockInvoiceLogRepo struct {
	createFn func(ctx context.Context, orgID uuid.UUID, input domain.CreateInvoiceLogInput) (*domain.InvoiceLog, error)
}

func (m *mockInvoiceLogRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateInvoiceLogInput) (*domain.InvoiceLog, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return &domain.InvoiceLog{ID: uuid.New()}, nil
}
func (m *mockInvoiceLogRepo) ListByInvoice(_ context.Context, _, _ uuid.UUID, _ domain.ListParams) ([]domain.InvoiceLog, error) {
	return nil, nil
}
func (m *mockInvoiceLogRepo) CountByInvoice(_ context.Context, _, _ uuid.UUID) (int64, error) {
	return 0, nil
}

type mockInvoiceSettingRepo struct {
	upsertFn          func(ctx context.Context, orgID uuid.UUID, input domain.UpsertInvoiceSettingInput) (*domain.InvoiceSetting, error)
	getByOrgAndTypeFn func(ctx context.Context, orgID uuid.UUID, invType domain.InvoiceType) (*domain.InvoiceSetting, error)
	listFn            func(ctx context.Context, orgID uuid.UUID) ([]domain.InvoiceSetting, error)
	deleteFn          func(ctx context.Context, orgID, settingID uuid.UUID) error
}

func (m *mockInvoiceSettingRepo) Upsert(ctx context.Context, orgID uuid.UUID, input domain.UpsertInvoiceSettingInput) (*domain.InvoiceSetting, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, orgID, input)
	}
	return &domain.InvoiceSetting{ID: uuid.New(), OrganizationID: orgID, Type: input.Type, IssueOn: input.IssueOn, IsActive: input.IsActive}, nil
}
func (m *mockInvoiceSettingRepo) GetByOrgAndType(ctx context.Context, orgID uuid.UUID, invType domain.InvoiceType) (*domain.InvoiceSetting, error) {
	if m.getByOrgAndTypeFn != nil {
		return m.getByOrgAndTypeFn(ctx, orgID, invType)
	}
	return nil, domain.ErrInvoiceSettingNotFound
}
func (m *mockInvoiceSettingRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.InvoiceSetting, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID)
	}
	return []domain.InvoiceSetting{}, nil
}
func (m *mockInvoiceSettingRepo) Delete(ctx context.Context, orgID, settingID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, orgID, settingID)
	}
	return nil
}

type mockCustomerRepoHandler struct {
	getByIDFn func(ctx context.Context, orgID, customerID uuid.UUID) (*domain.Customer, error)
}

func (m *mockCustomerRepoHandler) Create(_ context.Context, _ uuid.UUID, _ domain.CustomerInput) (*domain.Customer, error) {
	return nil, nil
}
func (m *mockCustomerRepoHandler) GetByID(ctx context.Context, orgID, customerID uuid.UUID) (*domain.Customer, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, customerID)
	}
	return &domain.Customer{ID: customerID, OrganizationID: orgID}, nil
}
func (m *mockCustomerRepoHandler) List(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.Customer, error) {
	return nil, nil
}
func (m *mockCustomerRepoHandler) Update(_ context.Context, _, _ uuid.UUID, _ domain.CustomerInput) (*domain.Customer, error) {
	return nil, nil
}
func (m *mockCustomerRepoHandler) SoftDelete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockCustomerRepoHandler) HardDelete(_ context.Context, _, _ uuid.UUID) error { return nil }

// ── invoice TxManager stub ────────────────────────────────────────────────────

type invoiceTxManager struct {
	invoiceRepo domain.InvoiceRepository
	itemRepo    domain.InvoiceItemRepository
}

func (m *invoiceTxManager) WithTx(ctx context.Context, fn func(domain.TxHandle) error) error {
	return fn(&invoiceTxHandle{invoiceRepo: m.invoiceRepo, itemRepo: m.itemRepo})
}

type invoiceTxHandle struct {
	invoiceRepo domain.InvoiceRepository
	itemRepo    domain.InvoiceItemRepository
}

func (h *invoiceTxHandle) CustomerRepo() domain.CustomerRepository                 { return nil }
func (h *invoiceTxHandle) AddressRepo() domain.AddressRepository                   { return nil }
func (h *invoiceTxHandle) CredentialRepo() domain.CredentialRepository             { return nil }
func (h *invoiceTxHandle) PaymentFSMRepo() domain.PaymentFSMRepository             { return nil }
func (h *invoiceTxHandle) SubscriptionRepo() domain.SubscriptionRepository         { return nil }
func (h *invoiceTxHandle) SubscriptionItemRepo() domain.SubscriptionItemRepository { return nil }
func (h *invoiceTxHandle) SubscriptionScheduleRepo() domain.SubscriptionScheduleRepository {
	return nil
}
func (h *invoiceTxHandle) WalletRepo() domain.WalletRepository           { return nil }
func (h *invoiceTxHandle) InvoiceRepo() domain.InvoiceRepository         { return h.invoiceRepo }
func (h *invoiceTxHandle) InvoiceItemRepo() domain.InvoiceItemRepository { return h.itemRepo }

// ── mock webhook dispatcher ───────────────────────────────────────────────────

type noopDispatcher struct{}

func (d *noopDispatcher) Dispatch(_ context.Context, _ uuid.UUID, _, _ string, _ []byte) error {
	return nil
}

// ── router setup ──────────────────────────────────────────────────────────────

type invoiceTestSetup struct {
	invoiceRepo  *mockInvoiceRepo
	itemRepo     *mockInvoiceItemRepo
	logRepo      *mockInvoiceLogRepo
	settingRepo  *mockInvoiceSettingRepo
	customerRepo *mockCustomerRepoHandler
}

func newInvoiceTestSetup() *invoiceTestSetup {
	return &invoiceTestSetup{
		invoiceRepo:  &mockInvoiceRepo{},
		itemRepo:     &mockInvoiceItemRepo{},
		logRepo:      &mockInvoiceLogRepo{},
		settingRepo:  &mockInvoiceSettingRepo{},
		customerRepo: &mockCustomerRepoHandler{},
	}
}

func (s *invoiceTestSetup) router(resolver middleware.PermissionResolver, orgID, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	txMgr := &invoiceTxManager{invoiceRepo: s.invoiceRepo, itemRepo: s.itemRepo}
	svc := service.NewInvoiceService(
		s.invoiceRepo, s.itemRepo, s.logRepo, s.settingRepo,
		s.customerRepo, nil, nil, nil,
		txMgr, &noopDispatcher{},
	)
	h := NewInvoiceHandler(svc, resolver)

	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaimsUserID(orgID, userID))
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func (s *invoiceTestSetup) routerNoAuth(resolver middleware.PermissionResolver) *gin.Engine {
	gin.SetMode(gin.TestMode)
	txMgr := &invoiceTxManager{invoiceRepo: s.invoiceRepo, itemRepo: s.itemRepo}
	svc := service.NewInvoiceService(
		s.invoiceRepo, s.itemRepo, s.logRepo, s.settingRepo,
		s.customerRepo, nil, nil, nil,
		txMgr, &noopDispatcher{},
	)
	h := NewInvoiceHandler(svc, resolver)

	r := gin.New()
	r.Use(middleware.Errors())
	// No claims middleware — simulates unauthenticated request
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

// ============================================================
// Invoice Route Tests
// ============================================================

func TestInvoiceCreate(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*invoiceTestSetup)
		resolver   middleware.PermissionResolver
		noAuth     bool
		body       string
		wantStatus int
	}{
		{
			name:       "401_no_auth",
			noAuth:     true,
			resolver:   &alwaysGrantResolver{},
			body:       `{"customer_id":"` + uuid.New().String() + `","invoice_type":"national","items":[{"description":"x","quantity":"1","unit_price_cents":100}]}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "403_no_permission",
			resolver:   &alwaysDenyResolver{},
			body:       `{"customer_id":"` + uuid.New().String() + `","invoice_type":"national","items":[{"description":"x","quantity":"1","unit_price_cents":100}]}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "422_no_items",
			resolver:   &alwaysGrantResolver{},
			body:       `{"customer_id":"` + uuid.New().String() + `","invoice_type":"national","items":[]}`,
			wantStatus: http.StatusBadRequest, // binding validation: min=1
		},
		{
			name:       "201_success",
			resolver:   &alwaysGrantResolver{},
			body:       `{"customer_id":"` + uuid.New().String() + `","invoice_type":"national","items":[{"description":"Item","quantity":"1","unit_price_cents":500}]}`,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			if tc.setup != nil {
				tc.setup(s)
			}

			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/invoices", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(middleware.IdempotencyHeader, uuid.New().String())
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestInvoiceList(t *testing.T) {
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, wantStatus: http.StatusForbidden},
		{name: "200_success", resolver: &alwaysGrantResolver{}, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/invoices", nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestInvoiceGetByID(t *testing.T) {
	invID := uuid.New()
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		setup      func(*invoiceTestSetup)
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, wantStatus: http.StatusForbidden},
		{
			name:     "404_not_found",
			resolver: &alwaysGrantResolver{},
			setup: func(s *invoiceTestSetup) {
				s.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
					return nil, domain.ErrInvoiceNotFound
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{name: "200_success", resolver: &alwaysGrantResolver{}, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			if tc.setup != nil {
				tc.setup(s)
			}
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/invoices/"+invID.String(), nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestInvoiceCancel(t *testing.T) {
	invID := uuid.New()
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		setup      func(*invoiceTestSetup)
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, wantStatus: http.StatusForbidden},
		{
			name:     "422_already_canceled",
			resolver: &alwaysGrantResolver{},
			setup: func(s *invoiceTestSetup) {
				s.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
					return &domain.Invoice{ID: invID, Status: domain.InvStatusCanceled}, nil
				}
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "200_success",
			resolver: &alwaysGrantResolver{},
			setup: func(s *invoiceTestSetup) {
				s.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
					return &domain.Invoice{ID: invID, Status: domain.InvStatusPending}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			if tc.setup != nil {
				tc.setup(s)
			}
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/invoices/"+invID.String()+"/cancel", nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestInvoiceTransition(t *testing.T) {
	invID := uuid.New()
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		setup      func(*invoiceTestSetup)
		body       string
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, body: `{"status":"sent"}`, wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, body: `{"status":"sent"}`, wantStatus: http.StatusForbidden},
		{
			name:     "422_invalid_transition",
			resolver: &alwaysGrantResolver{},
			setup: func(s *invoiceTestSetup) {
				s.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
					return &domain.Invoice{ID: invID, Status: domain.InvStatusPaid}, nil
				}
			},
			body:       `{"status":"sent"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "200_success",
			resolver: &alwaysGrantResolver{},
			setup: func(s *invoiceTestSetup) {
				s.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
					return &domain.Invoice{ID: invID, Status: domain.InvStatusPending}, nil
				}
			},
			body:       `{"status":"sent"}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			if tc.setup != nil {
				tc.setup(s)
			}
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/invoices/"+invID.String()+"/transition", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestInvoiceListLogs(t *testing.T) {
	invID := uuid.New()
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, wantStatus: http.StatusForbidden},
		{name: "200_success", resolver: &alwaysGrantResolver{}, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/invoices/"+invID.String()+"/logs", nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================
// Invoice Settings Route Tests
// ============================================================

func TestInvoiceSettingUpsert(t *testing.T) {
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		path       string
		body       string
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, path: "/invoice-settings/national", body: `{}`, wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, path: "/invoice-settings/national", body: `{}`, wantStatus: http.StatusForbidden},
		{name: "400_invalid_type", resolver: &alwaysGrantResolver{}, path: "/invoice-settings/invalid", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "200_success", resolver: &alwaysGrantResolver{}, path: "/invoice-settings/national", body: `{"issue_on":"payment_confirmed","is_active":true}`, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestInvoiceSettingList(t *testing.T) {
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, wantStatus: http.StatusForbidden},
		{name: "200_success", resolver: &alwaysGrantResolver{}, wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/invoice-settings", nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestInvoiceSettingGetByType(t *testing.T) {
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		setup      func(*invoiceTestSetup)
		path       string
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, path: "/invoice-settings/national", wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, path: "/invoice-settings/national", wantStatus: http.StatusForbidden},
		{name: "400_invalid_type", resolver: &alwaysGrantResolver{}, path: "/invoice-settings/invalid", wantStatus: http.StatusBadRequest},
		{
			name:       "404_not_found",
			resolver:   &alwaysGrantResolver{},
			path:       "/invoice-settings/national",
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "200_success",
			resolver: &alwaysGrantResolver{},
			path:     "/invoice-settings/national",
			setup: func(s *invoiceTestSetup) {
				s.settingRepo.getByOrgAndTypeFn = func(_ context.Context, orgID uuid.UUID, _ domain.InvoiceType) (*domain.InvoiceSetting, error) {
					return &domain.InvoiceSetting{ID: uuid.New(), OrganizationID: orgID, Type: domain.InvTypeNational, IssueOn: "payment_confirmed"}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			if tc.setup != nil {
				tc.setup(s)
			}
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, tc.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestInvoiceSettingDelete(t *testing.T) {
	tests := []struct {
		name       string
		noAuth     bool
		resolver   middleware.PermissionResolver
		setup      func(*invoiceTestSetup)
		path       string
		wantStatus int
	}{
		{name: "401_no_auth", noAuth: true, resolver: &alwaysGrantResolver{}, path: "/invoice-settings/national", wantStatus: http.StatusUnauthorized},
		{name: "403_no_permission", resolver: &alwaysDenyResolver{}, path: "/invoice-settings/national", wantStatus: http.StatusForbidden},
		{name: "400_invalid_type", resolver: &alwaysGrantResolver{}, path: "/invoice-settings/invalid", wantStatus: http.StatusBadRequest},
		{
			name:       "404_not_found",
			resolver:   &alwaysGrantResolver{},
			path:       "/invoice-settings/national",
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "204_success",
			resolver: &alwaysGrantResolver{},
			path:     "/invoice-settings/national",
			setup: func(s *invoiceTestSetup) {
				s.settingRepo.getByOrgAndTypeFn = func(_ context.Context, orgID uuid.UUID, _ domain.InvoiceType) (*domain.InvoiceSetting, error) {
					return &domain.InvoiceSetting{ID: uuid.New(), OrganizationID: orgID, Type: domain.InvTypeNational}, nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newInvoiceTestSetup()
			if tc.setup != nil {
				tc.setup(s)
			}
			var r *gin.Engine
			if tc.noAuth {
				r = s.routerNoAuth(tc.resolver)
			} else {
				r = s.router(tc.resolver, uuid.New().String(), uuid.New().String())
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, tc.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}
