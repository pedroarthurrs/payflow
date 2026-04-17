package service

import (
	"context"
	"errors"
	"testing"

	"i9pay/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// allInvoiceStatuses is the complete set of InvoiceStatus values.
var allInvoiceStatuses = []domain.InvoiceStatus{
	domain.InvStatusPending,
	domain.InvStatusSent,
	domain.InvStatusPaid,
	domain.InvStatusError,
	domain.InvStatusCanceled,
}

// validTransitionGraph mirrors the FSM rules for assertion.
var validTransitionGraph = map[domain.InvoiceStatus]map[domain.InvoiceStatus]bool{
	domain.InvStatusPending: {
		domain.InvStatusSent:     true,
		domain.InvStatusCanceled: true,
	},
	domain.InvStatusSent: {
		domain.InvStatusPaid:     true,
		domain.InvStatusError:    true,
		domain.InvStatusCanceled: true,
	},
	domain.InvStatusError: {
		domain.InvStatusSent:     true,
		domain.InvStatusCanceled: true,
	},
	domain.InvStatusPaid:     {},
	domain.InvStatusCanceled: {},
}

// terminalStatesSet defines which states are terminal.
var terminalStatesSet = map[domain.InvoiceStatus]bool{
	domain.InvStatusPaid:     true,
	domain.InvStatusCanceled: true,
}

// ── Property 1: FSM comprehensive validation ─────────────────────────────────
// Feature: invoices, Property 1: Validação abrangente da FSM de Invoice
// Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 11.4a, 11.4b

func TestProperty_InvoiceFSM_ComprehensiveValidation(t *testing.T) {
	fsm := domain.NewInvoiceStateMachine()

	rapid.Check(t, func(t *rapid.T) {
		from := rapid.SampledFrom(allInvoiceStatuses).Draw(t, "from")
		to := rapid.SampledFrom(allInvoiceStatuses).Draw(t, "to")

		err := fsm.Validate(from, to)

		switch {
		case from == to:
			// Same state → ErrSameStateTransition
			if !errors.Is(err, domain.ErrSameStateTransition) {
				t.Fatalf("from=%s to=%s: expected ErrSameStateTransition, got %v", from, to, err)
			}

		case terminalStatesSet[from]:
			// Terminal state → appropriate error
			if from == domain.InvStatusCanceled {
				if !errors.Is(err, domain.ErrInvoiceAlreadyCanceled) {
					t.Fatalf("from=canceled to=%s: expected ErrInvoiceAlreadyCanceled, got %v", to, err)
				}
			} else {
				if !errors.Is(err, domain.ErrTerminalState) {
					t.Fatalf("from=%s to=%s: expected ErrTerminalState, got %v", from, to, err)
				}
			}

		case validTransitionGraph[from][to]:
			// Valid transition → nil
			if err != nil {
				t.Fatalf("from=%s to=%s: expected nil (valid transition), got %v", from, to, err)
			}

		default:
			// Invalid transition → ErrInvalidInvoiceTransition
			if !errors.Is(err, domain.ErrInvalidInvoiceTransition) {
				t.Fatalf("from=%s to=%s: expected ErrInvalidInvoiceTransition, got %v", from, to, err)
			}
		}

		// Additional invariant: no valid transition results in same state
		if err == nil && from == to {
			t.Fatalf("from=%s to=%s: valid transition must not have from == to", from, to)
		}
	})
}

// ── Property 2: Items required ────────────────────────────────────────────────
// Feature: invoices, Property 2: Rejeição de invoice sem itens
// Validates: Requirements 4.1a

func TestProperty_InvoiceCreate_ItemsRequired(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		deps := newInvoiceTestDeps()
		deps.customerRepo.getByIDFn = customerExistsMock()
		svc := deps.service()

		orgID := uuid.New()
		customerID := uuid.New()

		// Generate random invoice type
		invType := rapid.SampledFrom([]domain.InvoiceType{
			domain.InvTypeNational,
			domain.InvTypeInternational,
		}).Draw(t, "invoice_type")

		// Generate random currency
		currency := rapid.SampledFrom([]string{"BRL", "USD", "EUR", ""}).Draw(t, "currency")

		input := domain.CreateInvoiceInput{
			CustomerID:  customerID,
			InvoiceType: invType,
			Currency:    currency,
			Items:       []domain.CreateInvoiceItemInput{}, // empty items
		}

		_, err := svc.Create(context.Background(), orgID, input)
		if !errors.Is(err, domain.ErrInvoiceItemsRequired) {
			t.Fatalf("invoice_type=%s currency=%s: expected ErrInvoiceItemsRequired, got %v", invType, currency, err)
		}
	})
}

// ── Property 3: amount_cents invariant ────────────────────────────────────────
// Feature: invoices, Property 3: Invariante de cálculo do amount_cents
// Validates: Requirements 4.1d, 7.6

func TestProperty_InvoiceCreate_AmountCentsInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		deps := newInvoiceTestDeps()
		deps.customerRepo.getByIDFn = customerExistsMock()

		// Track the amount_cents passed to the repo
		var capturedAmountCents int32
		deps.txMgr.withTxFn = func(_ context.Context, fn func(tx domain.TxHandle) error) error {
			invoiceRepo := &mockInvoiceRepoStub{
				createFn: func(_ context.Context, orgID uuid.UUID, input domain.CreateInvoiceInput) (*domain.Invoice, error) {
					capturedAmountCents = input.AmountCents
					return &domain.Invoice{
						ID:             uuid.New(),
						OrganizationID: orgID,
						CustomerID:     input.CustomerID,
						InvoiceType:    input.InvoiceType,
						AmountCents:    input.AmountCents,
						Currency:       input.Currency,
						Status:         domain.InvStatusPending,
					}, nil
				},
			}
			itemRepo := &mockInvoiceItemRepoStub{}
			return fn(&mockTxHandle{invoices: invoiceRepo, invoiceItems: itemRepo})
		}

		svc := deps.service()
		orgID := uuid.New()

		// Generate random items (1 to 10) with random total_price_cents >= 0
		numItems := rapid.IntRange(1, 10).Draw(t, "num_items")
		items := make([]domain.CreateInvoiceItemInput, numItems)
		var expectedSum int32

		for i := 0; i < numItems; i++ {
			totalPrice := rapid.Int32Range(0, 100000).Draw(t, "total_price_cents")
			items[i] = domain.CreateInvoiceItemInput{
				Description:     "Item",
				Quantity:        "1",
				UnitPriceCents:  totalPrice,
				TotalPriceCents: totalPrice,
			}
			expectedSum += totalPrice
		}

		input := domain.CreateInvoiceInput{
			CustomerID:  uuid.New(),
			InvoiceType: domain.InvTypeNational,
			Items:       items,
		}

		invoice, err := svc.Create(context.Background(), orgID, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedAmountCents != expectedSum {
			t.Fatalf("amount_cents mismatch: got %d, expected sum %d", capturedAmountCents, expectedSum)
		}
		if invoice.AmountCents != expectedSum {
			t.Fatalf("invoice.AmountCents mismatch: got %d, expected sum %d", invoice.AmountCents, expectedSum)
		}
	})
}

// ── Property 5: Cancel behavior ───────────────────────────────────────────────
// Feature: invoices, Property 5: Cancel sempre transita para canceled
// Validates: Requirements 4.5

func TestProperty_InvoiceCancel_Behavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		status := rapid.SampledFrom(allInvoiceStatuses).Draw(t, "status")

		deps := newInvoiceTestDeps()
		orgID := uuid.New()
		invID := uuid.New()

		deps.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return &domain.Invoice{ID: invID, OrganizationID: orgID, Status: status}, nil
		}
		deps.invoiceRepo.updateStatusFn = func(_ context.Context, _, _ uuid.UUID, newStatus domain.InvoiceStatus) (*domain.Invoice, error) {
			return &domain.Invoice{ID: invID, OrganizationID: orgID, Status: newStatus}, nil
		}

		svc := deps.service()
		invoice, err := svc.Cancel(context.Background(), orgID, invID)

		nonTerminal := !terminalStatesSet[status]

		if nonTerminal {
			// Non-terminal states: Cancel succeeds, status = canceled
			if err != nil {
				t.Fatalf("status=%s: expected Cancel to succeed, got error: %v", status, err)
			}
			if invoice.Status != domain.InvStatusCanceled {
				t.Fatalf("status=%s: expected canceled, got %s", status, invoice.Status)
			}
		} else {
			// Terminal states: Cancel returns error
			if err == nil {
				t.Fatalf("status=%s (terminal): expected Cancel to return error, got nil", status)
			}
			// canceled→canceled is same-state, paid→canceled is terminal-state error
			if status == domain.InvStatusCanceled {
				if !errors.Is(err, domain.ErrSameStateTransition) {
					t.Fatalf("status=canceled: expected ErrSameStateTransition, got %v", err)
				}
			} else {
				if !errors.Is(err, domain.ErrTerminalState) {
					t.Fatalf("status=%s: expected ErrTerminalState, got %v", status, err)
				}
			}
		}
	})
}

// ── Property 8: Upsert idempotency ───────────────────────────────────────────
// Feature: invoices, Property 8: Idempotência do upsert de InvoiceSetting
// Validates: Requirements 4.8

func TestProperty_InvoiceUpsertSetting_Idempotency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()

		// Generate random setting input
		invType := rapid.SampledFrom([]domain.InvoiceType{
			domain.InvTypeNational,
			domain.InvTypeInternational,
		}).Draw(t, "type")

		issueOn := rapid.SampledFrom([]string{"payment_confirmed", "manual", "subscription_created"}).Draw(t, "issue_on")
		isActive := rapid.Bool().Draw(t, "is_active")

		input := domain.UpsertInvoiceSettingInput{
			Type:     invType,
			IssueOn:  issueOn,
			IsActive: isActive,
		}

		// Track call count
		callCount := 0
		settingID := uuid.New()

		deps := newInvoiceTestDeps()
		deps.settingRepo.upsertFn = func(_ context.Context, _ uuid.UUID, inp domain.UpsertInvoiceSettingInput) (*domain.InvoiceSetting, error) {
			callCount++
			return &domain.InvoiceSetting{
				ID:             settingID,
				OrganizationID: orgID,
				Type:           inp.Type,
				IssueOn:        inp.IssueOn,
				IsActive:       inp.IsActive,
			}, nil
		}

		svc := deps.service()

		// Call UpsertSetting twice with same (org, type)
		s1, err1 := svc.UpsertSetting(context.Background(), orgID, input)
		s2, err2 := svc.UpsertSetting(context.Background(), orgID, input)

		// Both calls must succeed
		if err1 != nil {
			t.Fatalf("first upsert failed: %v", err1)
		}
		if err2 != nil {
			t.Fatalf("second upsert failed: %v", err2)
		}

		// Both return the same setting ID (same logical record)
		if s1.ID != s2.ID {
			t.Fatalf("expected same setting ID, got %s and %s", s1.ID, s2.ID)
		}

		// Mock was called exactly twice
		if callCount != 2 {
			t.Fatalf("expected 2 upsert calls, got %d", callCount)
		}

		// Values match the input
		if s2.Type != invType {
			t.Fatalf("expected type=%s, got %s", invType, s2.Type)
		}
		if s2.IssueOn != issueOn {
			t.Fatalf("expected issue_on=%s, got %s", issueOn, s2.IssueOn)
		}
		if s2.IsActive != isActive {
			t.Fatalf("expected is_active=%v, got %v", isActive, s2.IsActive)
		}
	})
}
