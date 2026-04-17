package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"i9pay/internal/api/middleware"
	i9token "i9pay/internal/auth/token"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── permission resolver stubs ─────────────────────────────────────────────────

type alwaysGrantResolver struct{}

func (r *alwaysGrantResolver) HasPermission(_ context.Context, _, _ uuid.UUID, _ domain.Permission) (bool, error) {
	return true, nil
}

type alwaysDenyResolver struct{}

func (r *alwaysDenyResolver) HasPermission(_ context.Context, _, _ uuid.UUID, _ domain.Permission) (bool, error) {
	return false, nil
}

// ── mock wallet service ───────────────────────────────────────────────────────

type mockWalletService struct {
	getOrgWalletFn  func(ctx context.Context, orgID uuid.UUID) (*domain.Wallet, error)
	getByIDFn       func(ctx context.Context, orgID, walletID uuid.UUID) (*domain.Wallet, error)
	listEntriesFn   func(ctx context.Context, orgID, walletID uuid.UUID, params domain.LedgerListParams) ([]domain.LedgerEntry, int64, error)
	requestPayoutFn func(ctx context.Context, orgID uuid.UUID, input domain.CreatePayoutInput) (*domain.Payout, error)
	listPayoutsFn   func(ctx context.Context, orgID, walletID uuid.UUID, params domain.ListParams) ([]domain.Payout, error)
	adjustFn        func(ctx context.Context, orgID, walletID uuid.UUID, input domain.AdjustWalletInput) (*domain.Wallet, error)
}

func (m *mockWalletService) GetOrgWallet(ctx context.Context, orgID uuid.UUID) (*domain.Wallet, error) {
	if m.getOrgWalletFn != nil {
		return m.getOrgWalletFn(ctx, orgID)
	}
	return &domain.Wallet{ID: uuid.New(), OrganizationID: orgID, Status: domain.WalletStatusActive}, nil
}

func (m *mockWalletService) GetByID(ctx context.Context, orgID, walletID uuid.UUID) (*domain.Wallet, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, walletID)
	}
	return &domain.Wallet{ID: walletID, OrganizationID: orgID, Status: domain.WalletStatusActive}, nil
}

func (m *mockWalletService) ListEntries(ctx context.Context, orgID, walletID uuid.UUID, params domain.LedgerListParams) ([]domain.LedgerEntry, int64, error) {
	if m.listEntriesFn != nil {
		return m.listEntriesFn(ctx, orgID, walletID, params)
	}
	return nil, 0, nil
}

func (m *mockWalletService) RequestPayout(ctx context.Context, orgID uuid.UUID, input domain.CreatePayoutInput) (*domain.Payout, error) {
	if m.requestPayoutFn != nil {
		return m.requestPayoutFn(ctx, orgID, input)
	}
	return &domain.Payout{ID: uuid.New(), WalletID: input.WalletID, AmountCents: input.AmountCents, Status: domain.PayoutPending}, nil
}

func (m *mockWalletService) ListPayouts(ctx context.Context, orgID, walletID uuid.UUID, params domain.ListParams) ([]domain.Payout, error) {
	if m.listPayoutsFn != nil {
		return m.listPayoutsFn(ctx, orgID, walletID, params)
	}
	return nil, nil
}

func (m *mockWalletService) Adjust(ctx context.Context, orgID, walletID uuid.UUID, input domain.AdjustWalletInput) (*domain.Wallet, error) {
	if m.adjustFn != nil {
		return m.adjustFn(ctx, orgID, walletID, input)
	}
	return &domain.Wallet{ID: walletID, OrganizationID: orgID, Balance: 1000, Status: domain.WalletStatusActive}, nil
}

// ── router setup ──────────────────────────────────────────────────────────────

// handlerTxManager is a TxManager stub for handler tests.
type handlerTxManager struct {
	repo domain.WalletRepository
}

func (m *handlerTxManager) WithTx(ctx context.Context, fn func(domain.TxHandle) error) error {
	return fn(&handlerTxHandle{walletRepo: m.repo})
}

// handlerTxHandle is a TxHandle stub that only provides WalletRepo.
type handlerTxHandle struct {
	walletRepo domain.WalletRepository
}

func (h *handlerTxHandle) CustomerRepo() domain.CustomerRepository                 { return nil }
func (h *handlerTxHandle) AddressRepo() domain.AddressRepository                   { return nil }
func (h *handlerTxHandle) CredentialRepo() domain.CredentialRepository             { return nil }
func (h *handlerTxHandle) PaymentFSMRepo() domain.PaymentFSMRepository             { return nil }
func (h *handlerTxHandle) SubscriptionRepo() domain.SubscriptionRepository         { return nil }
func (h *handlerTxHandle) SubscriptionItemRepo() domain.SubscriptionItemRepository { return nil }
func (h *handlerTxHandle) SubscriptionScheduleRepo() domain.SubscriptionScheduleRepository {
	return nil
}
func (h *handlerTxHandle) WalletRepo() domain.WalletRepository           { return h.walletRepo }
func (h *handlerTxHandle) InvoiceRepo() domain.InvoiceRepository         { return nil }
func (h *handlerTxHandle) InvoiceItemRepo() domain.InvoiceItemRepository { return nil }

func setupWalletRouterDirect(svcMock *mockWalletService, resolver middleware.PermissionResolver, orgID, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := &mockDelegatingRepo{mock: svcMock}
	txMgr := &handlerTxManager{repo: repo}
	realSvc := service.NewWalletService(repo, txMgr)
	h := NewWalletHandler(realSvc, resolver)

	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaimsUserID(orgID, userID))
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func withClaimsUserID(orgID, userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &i9token.Claims{
			OrganizationID: orgID,
			UserID:         userID,
		})
		c.Next()
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestWalletGetOrgWallet_200(t *testing.T) {
	orgID := uuid.New().String()
	userID := uuid.New().String()
	r := setupWalletRouterDirect(&mockWalletService{}, &alwaysGrantResolver{}, orgID, userID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/wallets/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := resp["locked_balance"]; !ok {
		t.Error("response missing locked_balance field")
	}
	if _, ok := resp["available_balance"]; !ok {
		t.Error("response missing available_balance field")
	}
}

func TestWalletGetOrgWallet_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockDelegatingRepo{mock: &mockWalletService{}}
	realSvc := service.NewWalletService(repo, &handlerTxManager{repo: repo})
	h := NewWalletHandler(realSvc, &alwaysGrantResolver{})

	r := gin.New()
	r.Use(middleware.Errors())
	h.RegisterRoutes(&r.RouterGroup)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/wallets/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWalletGetOrgWallet_403(t *testing.T) {
	orgID := uuid.New().String()
	userID := uuid.New().String()
	r := setupWalletRouterDirect(&mockWalletService{}, &alwaysDenyResolver{}, orgID, userID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/wallets/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestWalletGetByID_404(t *testing.T) {
	orgID := uuid.New().String()
	userID := uuid.New().String()
	mock := &mockWalletService{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Wallet, error) {
			return nil, domain.ErrWalletNotFound
		},
	}
	r := setupWalletRouterDirect(mock, &alwaysGrantResolver{}, orgID, userID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/wallets/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWalletRequestPayout_201(t *testing.T) {
	orgID := uuid.New().String()
	userID := uuid.New().String()
	r := setupWalletRouterDirect(&mockWalletService{}, &alwaysGrantResolver{}, orgID, userID)

	body := `{"amount_cents":1000,"fee_cents":50}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/wallets/"+uuid.New().String()+"/payouts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.IdempotencyHeader, uuid.New().String())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWalletRequestPayout_422_InsufficientBalance(t *testing.T) {
	orgID := uuid.New().String()
	userID := uuid.New().String()
	mock := &mockWalletService{
		requestPayoutFn: func(_ context.Context, _ uuid.UUID, _ domain.CreatePayoutInput) (*domain.Payout, error) {
			return nil, domain.ErrInsufficientBalance
		},
	}
	r := setupWalletRouterDirect(mock, &alwaysGrantResolver{}, orgID, userID)

	body := `{"amount_cents":999999}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/wallets/"+uuid.New().String()+"/payouts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.IdempotencyHeader, uuid.New().String())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWalletAdjust_200(t *testing.T) {
	orgID := uuid.New().String()
	userID := uuid.New().String()
	r := setupWalletRouterDirect(&mockWalletService{}, &alwaysGrantResolver{}, orgID, userID)

	body := `{"direction":"credit","amount_cents":500,"reason":"test"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/wallets/"+uuid.New().String()+"/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.IdempotencyHeader, uuid.New().String())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWalletAdjust_422_InvalidInput(t *testing.T) {
	orgID := uuid.New().String()
	userID := uuid.New().String()
	// Return a wallet with 0 balance so debit of 99999 triggers ErrInsufficientBalance
	mock := &mockWalletService{
		getByIDFn: func(_ context.Context, orgID, wID uuid.UUID) (*domain.Wallet, error) {
			return &domain.Wallet{ID: wID, OrganizationID: orgID, Balance: 0, Status: domain.WalletStatusActive}, nil
		},
	}
	r := setupWalletRouterDirect(mock, &alwaysGrantResolver{}, orgID, userID)

	body := `{"direction":"debit","amount_cents":99999,"reason":"too much"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/wallets/"+uuid.New().String()+"/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.IdempotencyHeader, uuid.New().String())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

// ── mockDelegatingRepo ────────────────────────────────────────────────────────

type mockDelegatingRepo struct {
	mock *mockWalletService
}

func (r *mockDelegatingRepo) GetWalletByOrganizationOwner(ctx context.Context, orgID uuid.UUID) (*domain.Wallet, error) {
	return r.mock.GetOrgWallet(ctx, orgID)
}

func (r *mockDelegatingRepo) GetWalletByID(ctx context.Context, orgID, walletID uuid.UUID) (*domain.Wallet, error) {
	if r.mock.getByIDFn != nil {
		return r.mock.getByIDFn(ctx, orgID, walletID)
	}
	return &domain.Wallet{ID: walletID, OrganizationID: orgID, Balance: 100000, LockedBalance: 0, Status: domain.WalletStatusActive}, nil
}

func (r *mockDelegatingRepo) GetWalletForUpdate(ctx context.Context, orgID, walletID uuid.UUID) (*domain.Wallet, error) {
	if r.mock.getByIDFn != nil {
		return r.mock.getByIDFn(ctx, orgID, walletID)
	}
	return &domain.Wallet{ID: walletID, OrganizationID: orgID, Balance: 100000, LockedBalance: 0, Status: domain.WalletStatusActive}, nil
}

func (r *mockDelegatingRepo) CreateWallet(_ context.Context, orgID uuid.UUID, _ domain.CreateWalletInput) (*domain.Wallet, error) {
	return &domain.Wallet{ID: uuid.New(), OrganizationID: orgID, Status: domain.WalletStatusActive, Currency: "BRL"}, nil
}

func (r *mockDelegatingRepo) UpdateWalletBalance(_ context.Context, _, walletID uuid.UUID, newBalance int32) (*domain.Wallet, error) {
	return &domain.Wallet{ID: walletID, Balance: newBalance, Status: domain.WalletStatusActive}, nil
}

func (r *mockDelegatingRepo) UpdateWalletLockedBalance(_ context.Context, _, walletID uuid.UUID, newLocked int32) (*domain.Wallet, error) {
	return &domain.Wallet{ID: walletID, LockedBalance: newLocked, Status: domain.WalletStatusActive}, nil
}

func (r *mockDelegatingRepo) InsertLedgerEntry(_ context.Context, _ uuid.UUID, _ domain.LedgerEntryInput) (*domain.LedgerEntry, error) {
	return &domain.LedgerEntry{ID: uuid.New()}, nil
}

func (r *mockDelegatingRepo) ListLedgerEntries(ctx context.Context, orgID, walletID uuid.UUID, params domain.LedgerListParams) ([]domain.LedgerEntry, error) {
	entries, _, _ := r.mock.ListEntries(ctx, orgID, walletID, params)
	return entries, nil
}

func (r *mockDelegatingRepo) CountLedgerEntries(ctx context.Context, orgID, walletID uuid.UUID, params domain.LedgerListParams) (int64, error) {
	_, count, _ := r.mock.ListEntries(ctx, orgID, walletID, params)
	return count, nil
}

func (r *mockDelegatingRepo) CreatePayout(_ context.Context, _ uuid.UUID, input domain.CreatePayoutInput) (*domain.Payout, error) {
	return &domain.Payout{ID: uuid.New(), WalletID: input.WalletID, AmountCents: input.AmountCents, Status: domain.PayoutPending}, nil
}

func (r *mockDelegatingRepo) GetPayoutByID(_ context.Context, _, payoutID uuid.UUID) (*domain.Payout, error) {
	return &domain.Payout{ID: payoutID, Status: domain.PayoutPending}, nil
}

func (r *mockDelegatingRepo) UpdatePayoutStatus(_ context.Context, _, payoutID uuid.UUID, status domain.PayoutStatus) (*domain.Payout, error) {
	return &domain.Payout{ID: payoutID, Status: status}, nil
}

func (r *mockDelegatingRepo) ListPayouts(ctx context.Context, orgID, walletID uuid.UUID, params domain.ListParams) ([]domain.Payout, error) {
	return r.mock.ListPayouts(ctx, orgID, walletID, params)
}

func (r *mockDelegatingRepo) UpsertCommissionSetting(_ context.Context, _ uuid.UUID, input domain.CommissionSetting) (*domain.CommissionSetting, error) {
	return &input, nil
}

func (r *mockDelegatingRepo) GetCommissionSettingByCustomer(_ context.Context, _, _ uuid.UUID) (*domain.CommissionSetting, error) {
	return nil, domain.ErrCommissionNotFound
}

func (r *mockDelegatingRepo) ListCommissionSettings(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.CommissionSetting, error) {
	return nil, nil
}

func (r *mockDelegatingRepo) SoftDeleteCommissionSetting(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (r *mockDelegatingRepo) CreateWalletLock(_ context.Context, _ uuid.UUID, input domain.CreateWalletLockInput) (*domain.WalletLock, error) {
	return &domain.WalletLock{ID: uuid.New(), WalletID: input.WalletID, AmountCents: input.AmountCents}, nil
}

func (r *mockDelegatingRepo) ReleaseWalletLock(_ context.Context, _, _ uuid.UUID) error { return nil }

func (r *mockDelegatingRepo) GetWalletLockByReference(_ context.Context, _, _ uuid.UUID, _ domain.WalletLockType) (*domain.WalletLock, error) {
	return nil, domain.ErrWalletLockNotFound
}

func (r *mockDelegatingRepo) GetWalletByCustomerOwner(_ context.Context, _, _ uuid.UUID) (*domain.Wallet, error) {
	return nil, domain.ErrWalletNotFound
}
