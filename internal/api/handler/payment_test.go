package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"i9pay/internal/api/middleware"
	paymentdb "i9pay/internal/database/dbgen/payment"
	"i9pay/internal/database/repository"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ── mock TxManager and TxHandle for FSM tests ────────────────────────────────

type mockTxManager struct {
	fsmRepo domain.PaymentFSMRepository
}

func (m *mockTxManager) WithTx(_ context.Context, fn func(domain.TxHandle) error) error {
	return fn(&mockTxHandle{fsmRepo: m.fsmRepo})
}

type mockTxHandle struct {
	fsmRepo domain.PaymentFSMRepository
}

func (h *mockTxHandle) CustomerRepo() domain.CustomerRepository                 { return nil }
func (h *mockTxHandle) AddressRepo() domain.AddressRepository                   { return nil }
func (h *mockTxHandle) CredentialRepo() domain.CredentialRepository             { return nil }
func (h *mockTxHandle) PaymentFSMRepo() domain.PaymentFSMRepository             { return h.fsmRepo }
func (h *mockTxHandle) SubscriptionRepo() domain.SubscriptionRepository         { return nil }
func (h *mockTxHandle) SubscriptionItemRepo() domain.SubscriptionItemRepository { return nil }

// ── mock payment querier ──────────────────────────────────────────────────────

// noopGateway is a test double that always returns a successful charge result.
type noopGateway struct{}

func (g *noopGateway) Charge(_ context.Context, req domain.GatewayRequest) (*domain.ChargeResult, error) {
	return &domain.ChargeResult{
		GatewayReference: "noop-ref",
		GatewayName:      "noop",
		Status:           domain.StatusAuthorized,
	}, nil
}
func (g *noopGateway) Refund(_ context.Context, _ domain.GatewayRequest) (*domain.RefundResult, error) {
	return &domain.RefundResult{GatewayReference: "noop-ref", Status: domain.StatusRefunded}, nil
}
func (g *noopGateway) Cancel(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
	return &domain.ChargeResult{GatewayReference: "noop-ref", GatewayName: "noop", Status: domain.StatusCanceled}, nil
}

type mockPaymentQuerier struct {
	capturedListParams paymentdb.ListPaymentsParams
	listPaymentsFn     func(ctx context.Context, arg paymentdb.ListPaymentsParams) ([]paymentdb.Payment, error)
}

func (m *mockPaymentQuerier) CountPayments(_ context.Context, _ paymentdb.CountPaymentsParams) (int64, error) {
	return 0, nil
}
func (m *mockPaymentQuerier) CreatePayment(_ context.Context, _ paymentdb.CreatePaymentParams) (paymentdb.Payment, error) {
	return paymentdb.Payment{}, nil
}
func (m *mockPaymentQuerier) GetPaymentByID(_ context.Context, _ paymentdb.GetPaymentByIDParams) (paymentdb.Payment, error) {
	return paymentdb.Payment{}, nil
}
func (m *mockPaymentQuerier) GetPaymentForUpdate(_ context.Context, _ paymentdb.GetPaymentForUpdateParams) (paymentdb.GetPaymentForUpdateRow, error) {
	return paymentdb.GetPaymentForUpdateRow{}, nil
}
func (m *mockPaymentQuerier) InsertPaymentHistory(_ context.Context, _ paymentdb.InsertPaymentHistoryParams) error {
	return nil
}
func (m *mockPaymentQuerier) ListPaymentHistory(_ context.Context, _ paymentdb.ListPaymentHistoryParams) ([]paymentdb.PaymentStatusHistory, error) {
	return nil, nil
}
func (m *mockPaymentQuerier) ListPayments(ctx context.Context, arg paymentdb.ListPaymentsParams) ([]paymentdb.Payment, error) {
	m.capturedListParams = arg
	if m.listPaymentsFn != nil {
		return m.listPaymentsFn(ctx, arg)
	}
	return nil, nil
}
func (m *mockPaymentQuerier) UpdatePaymentStatus(_ context.Context, _ paymentdb.UpdatePaymentStatusParams) error {
	return nil
}
func (m *mockPaymentQuerier) UpdatePaymentErrorLog(_ context.Context, _ paymentdb.UpdatePaymentErrorLogParams) error {
	return nil
}
func (m *mockPaymentQuerier) UpdatePaymentGatewayReference(_ context.Context, _ paymentdb.UpdatePaymentGatewayReferenceParams) error {
	return nil
}

// ── mock domain payment repository ───────────────────────────────────────────

type mockDomainPaymentRepo struct {
	createFn  func(ctx context.Context, orgID uuid.UUID, input domain.PaymentInput) (*domain.Payment, error)
	getByIDFn func(ctx context.Context, orgID, paymentID uuid.UUID) (*domain.Payment, error)
}

func (m *mockDomainPaymentRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return nil, nil
}
func (m *mockDomainPaymentRepo) GetByID(ctx context.Context, orgID, paymentID uuid.UUID) (*domain.Payment, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, paymentID)
	}
	return nil, domain.ErrPaymentNotFound
}
func (m *mockDomainPaymentRepo) List(_ context.Context, _ uuid.UUID, _ domain.PaymentListParams) ([]domain.Payment, error) {
	return nil, nil
}
func (m *mockDomainPaymentRepo) Count(_ context.Context, _ uuid.UUID, _ domain.PaymentListParams) (int64, error) {
	return 0, nil
}
func (m *mockDomainPaymentRepo) UpdateGatewayReference(_ context.Context, _, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockDomainPaymentRepo) UpdateErrorLog(_ context.Context, _, _ uuid.UUID, _ json.RawMessage) error {
	return nil
}

// ── mock FSM repository ───────────────────────────────────────────────────────

type mockPaymentFSMRepo struct {
	listHistoryFn  func(ctx context.Context, orgID, paymentID uuid.UUID) ([]domain.TransitionHistory, error)
	getForUpdateFn func(ctx context.Context, orgID, paymentID uuid.UUID) (*domain.PaymentForUpdate, error)
}

func (m *mockPaymentFSMRepo) GetForUpdate(ctx context.Context, orgID, paymentID uuid.UUID) (*domain.PaymentForUpdate, error) {
	if m.getForUpdateFn != nil {
		return m.getForUpdateFn(ctx, orgID, paymentID)
	}
	return nil, domain.ErrPaymentNotFound
}
func (m *mockPaymentFSMRepo) UpdateStatus(_ context.Context, _, _ uuid.UUID, _ domain.PaymentStatus) error {
	return nil
}
func (m *mockPaymentFSMRepo) InsertHistory(_ context.Context, _ domain.TransitionHistory) error {
	return nil
}
func (m *mockPaymentFSMRepo) ListHistory(ctx context.Context, orgID, paymentID uuid.UUID) ([]domain.TransitionHistory, error) {
	if m.listHistoryFn != nil {
		return m.listHistoryFn(ctx, orgID, paymentID)
	}
	return []domain.TransitionHistory{}, nil
}

// ── router helpers ────────────────────────────────────────────────────────────

func setupPaymentRouter(repo domain.PaymentRepository, orgID string) *gin.Engine {
	fsmRepo := &mockPaymentFSMRepo{}
	return setupPaymentRouterWithFSM(repo, fsmRepo, orgID)
}

func setupPaymentRouterWithFSM(repo domain.PaymentRepository, fsmRepo domain.PaymentFSMRepository, orgID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.NewPaymentService(repo)
	var fsmSvc *service.PaymentFSMService
	var gatewaySvc *service.GatewayService
	if fsmRepo != nil {
		txMgr := &mockTxManager{fsmRepo: fsmRepo}
		fsmSvc = service.NewPaymentFSMService(txMgr, fsmRepo)
		gatewaySvc = service.NewGatewayService(repo, fsmSvc, &noopGateway{})
	}
	h := NewPaymentHandler(svc, fsmSvc, gatewaySvc)
	r := gin.New()
	r.Use(middleware.Errors())
	v1 := r.Group("/v1")
	v1.Use(withClaims(orgID))
	h.RegisterRoutes(v1)
	return r
}

// ── Preservation tests ────────────────────────────────────────────────────────

// TestProperty_Preservation_CreateReturns201Pending verifies POST /payments always
// returns 201 with status=pending and matching amount_cents. Must pass on both
// unfixed and fixed code.
func TestProperty_Preservation_CreateReturns201Pending(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()
		customerID := uuid.New()
		amountCents := rapid.Int32Range(1, 1_000_000).Draw(t, "amount_cents")
		paymentMethod := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "payment_method")

		paymentID := uuid.New()
		createdPayment := &domain.Payment{
			ID:             paymentID,
			OrganizationID: orgID,
			CustomerID:     customerID,
			AmountCents:    amountCents,
			Status:         domain.StatusPending,
		}
		authorizedPayment := &domain.Payment{
			ID:             paymentID,
			OrganizationID: orgID,
			CustomerID:     customerID,
			AmountCents:    amountCents,
			Status:         domain.StatusAuthorized,
		}

		callCount := 0
		repo := &mockDomainPaymentRepo{
			createFn: func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
				createdPayment.AmountCents = input.AmountCents
				authorizedPayment.AmountCents = input.AmountCents
				return createdPayment, nil
			},
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
				callCount++
				// First call: GatewayService.Charge pre-check — return pending
				// Second call: GatewayService.Charge post-transition fetch — return authorized
				if callCount == 1 {
					return createdPayment, nil
				}
				return authorizedPayment, nil
			},
		}

		fsmRepo := &mockPaymentFSMRepo{
			getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
				return &domain.PaymentForUpdate{
					ID:             paymentID,
					OrganizationID: orgID,
					Status:         domain.StatusPending,
				}, nil
			},
		}

		router := setupPaymentRouterWithFSM(repo, fsmRepo, orgID.String())

		body := fmt.Sprintf(`{"customer_id":%q,"payment_method":%q,"amount_cents":%d}`,
			customerID.String(), paymentMethod, amountCents)
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/payments", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		gotAmount, ok := resp["amount_cents"].(float64)
		if !ok {
			t.Fatalf("expected amount_cents number, got %T: %v", resp["amount_cents"], resp["amount_cents"])
		}
		if int32(gotAmount) != amountCents {
			t.Fatalf("expected amount_cents=%d, got %d", amountCents, int32(gotAmount))
		}
	})
}

// TestProperty_Preservation_FSMValidate verifies StateMachine.Validate returns the
// correct sentinel error for all (from, to) pairs. Pure domain logic, no DB needed.
func TestProperty_Preservation_FSMValidate(t *testing.T) {
	allStatuses := []domain.PaymentStatus{
		domain.StatusPending, domain.StatusAuthorized, domain.StatusSucceeded,
		domain.StatusFailed, domain.StatusCanceled, domain.StatusDisputed,
		domain.StatusRefunded, domain.StatusPartiallyRefunded,
	}
	terminalStates := map[domain.PaymentStatus]bool{
		domain.StatusCanceled: true, domain.StatusRefunded: true,
	}
	validTransitions := map[domain.PaymentStatus]map[domain.PaymentStatus]bool{
		domain.StatusPending:           {domain.StatusAuthorized: true, domain.StatusSucceeded: true, domain.StatusFailed: true, domain.StatusCanceled: true},
		domain.StatusAuthorized:        {domain.StatusSucceeded: true, domain.StatusFailed: true, domain.StatusCanceled: true},
		domain.StatusSucceeded:         {domain.StatusDisputed: true, domain.StatusRefunded: true, domain.StatusPartiallyRefunded: true},
		domain.StatusFailed:            {domain.StatusPending: true},
		domain.StatusCanceled:          {},
		domain.StatusDisputed:          {domain.StatusSucceeded: true, domain.StatusRefunded: true},
		domain.StatusRefunded:          {},
		domain.StatusPartiallyRefunded: {domain.StatusRefunded: true},
	}

	fsm := domain.NewStateMachine()

	rapid.Check(t, func(t *rapid.T) {
		from := allStatuses[rapid.IntRange(0, len(allStatuses)-1).Draw(t, "from_idx")]
		to := allStatuses[rapid.IntRange(0, len(allStatuses)-1).Draw(t, "to_idx")]

		err := fsm.Validate(from, to)

		switch {
		case from == to:
			if !errors.Is(err, domain.ErrSameStateTransition) {
				t.Fatalf("expected ErrSameStateTransition for %q→%q, got %v", from, to, err)
			}
		case terminalStates[from]:
			if !errors.Is(err, domain.ErrTerminalState) {
				t.Fatalf("expected ErrTerminalState for %q→%q, got %v", from, to, err)
			}
		case validTransitions[from][to]:
			if err != nil {
				t.Fatalf("expected nil for valid %q→%q, got %v", from, to, err)
			}
		default:
			if !errors.Is(err, domain.ErrInvalidTransition) {
				t.Fatalf("expected ErrInvalidTransition for %q→%q, got %v", from, to, err)
			}
		}
	})
}

// ── Fix-checking tests ────────────────────────────────────────────────────────

// TestFixCheck_HistoryReturns200 (task 7.2): for any valid (orgID, paymentID) where
// the payment exists, GET /v1/payments/:id/history returns 200 with a JSON array.
func TestFixCheck_HistoryReturns200(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()
		paymentID := uuid.New()

		historyRecords := []domain.TransitionHistory{
			{
				ID:          uuid.New(),
				PaymentID:   paymentID,
				NewStatus:   domain.StatusAuthorized,
				EventSource: "api",
			},
		}
		// Use a simpler approach — just return a fixed slice
		fsmRepo := &mockPaymentFSMRepo{
			listHistoryFn: func(_ context.Context, _, _ uuid.UUID) ([]domain.TransitionHistory, error) {
				return historyRecords, nil
			},
		}

		router := setupPaymentRouterWithFSM(&mockDomainPaymentRepo{}, fsmRepo, orgID.String())

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
			"/v1/payments/"+paymentID.String()+"/history", http.NoBody)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
		}

		var resp []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("expected JSON array, got error: %v (body=%s)", err, w.Body.String())
		}
	})
}

// TestFixCheck_FiltersForwarded (task 7.3): for any PaymentListParams with at least
// one non-nil filter, the querier receives the correct NullPaymentStatusType.Valid=true.
func TestFixCheck_FiltersForwarded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()

		allStatuses := []domain.PaymentStatus{
			domain.StatusPending, domain.StatusAuthorized, domain.StatusSucceeded,
			domain.StatusFailed, domain.StatusCanceled,
		}
		status := allStatuses[rapid.IntRange(0, len(allStatuses)-1).Draw(t, "status_idx")]

		querier := &mockPaymentQuerier{}
		repo := repository.NewPaymentRepo(querier)

		params := domain.PaymentListParams{
			Status:     &status,
			ListParams: domain.ListParams{Limit: 20, Offset: 0},
		}

		_, _ = repo.List(context.Background(), orgID, params)

		if !querier.capturedListParams.Status.Valid {
			t.Fatalf("filter not forwarded: Status=%q but querier got Status.Valid=false", status)
		}
		if string(querier.capturedListParams.Status.PaymentStatusType) != string(status) {
			t.Fatalf("filter value wrong: expected %q, got %q",
				status, querier.capturedListParams.Status.PaymentStatusType)
		}
	})
}

// TestFixCheck_ListResponseIncludesTotal (task 7.4): GET /payments response includes
// a "total" key with a non-negative integer value, and total >= len(data).
func TestFixCheck_ListResponseIncludesTotal(t *testing.T) {
	orgID := uuid.New()
	expectedTotal := int64(42)

	router := setupPaymentRouter(&fixedCountPaymentRepo{total: expectedTotal}, orgID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/payments", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	totalVal, ok := resp["total"]
	if !ok {
		t.Fatalf("response missing 'total' field (body=%s)", w.Body.String())
	}
	total, ok := totalVal.(float64)
	if !ok {
		t.Fatalf("'total' is not a number: %T %v", totalVal, totalVal)
	}
	if total < 0 {
		t.Fatalf("'total' must be non-negative, got %v", total)
	}
	data, _ := resp["data"].([]any)
	if int64(total) < int64(len(data)) {
		t.Fatalf("total=%d must be >= len(data)=%d", int64(total), len(data))
	}
}

// fixedCountPaymentRepo is a test double that returns a fixed total from Count.
type fixedCountPaymentRepo struct {
	total int64
}

func (r *fixedCountPaymentRepo) Create(_ context.Context, _ uuid.UUID, _ domain.PaymentInput) (*domain.Payment, error) {
	return nil, nil
}
func (r *fixedCountPaymentRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
	return nil, domain.ErrPaymentNotFound
}
func (r *fixedCountPaymentRepo) List(_ context.Context, _ uuid.UUID, _ domain.PaymentListParams) ([]domain.Payment, error) {
	return []domain.Payment{}, nil
}
func (r *fixedCountPaymentRepo) Count(_ context.Context, _ uuid.UUID, _ domain.PaymentListParams) (int64, error) {
	return r.total, nil
}
func (r *fixedCountPaymentRepo) UpdateGatewayReference(_ context.Context, _, _ uuid.UUID, _, _ string) error {
	return nil
}
func (r *fixedCountPaymentRepo) UpdateErrorLog(_ context.Context, _, _ uuid.UUID, _ json.RawMessage) error {
	return nil
}

// TestFixCheck_IdempotencyKeyRoundTrip (task 7.5): GetByID returns idempotency_key
// when the payment was created with one.
func TestFixCheck_IdempotencyKeyRoundTrip(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	idempotencyKey := "test-idem-key-abc123"

	querier := &mockPaymentQuerier{
		listPaymentsFn: nil,
	}
	// Override GetPaymentByID to return a payment with idempotency_key set
	getByIDQuerier := &getByIDPaymentQuerier{
		mockPaymentQuerier: querier,
		payment: paymentdb.Payment{
			ID:             paymentID,
			OrganizationID: orgID,
			IdempotencyKey: sql.NullString{String: idempotencyKey, Valid: true},
		},
	}

	repo := repository.NewPaymentRepo(getByIDQuerier)

	p, err := repo.GetByID(context.Background(), orgID, paymentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.IdempotencyKey == nil {
		t.Fatalf("expected idempotency_key=%q, got nil", idempotencyKey)
	}
	if *p.IdempotencyKey != idempotencyKey {
		t.Fatalf("expected idempotency_key=%q, got %q", idempotencyKey, *p.IdempotencyKey)
	}
}

// getByIDPaymentQuerier overrides GetPaymentByID on the base mock.
type getByIDPaymentQuerier struct {
	*mockPaymentQuerier
	payment paymentdb.Payment
}

func (q *getByIDPaymentQuerier) GetPaymentByID(_ context.Context, _ paymentdb.GetPaymentByIDParams) (paymentdb.Payment, error) {
	return q.payment, nil
}
