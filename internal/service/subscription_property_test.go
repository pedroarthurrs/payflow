package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// allSubStatuses lists every subscription status for random generation.
var allSubStatuses = []domain.SubscriptionStatus{
	domain.SubStatusTrialing,
	domain.SubStatusActive,
	domain.SubStatusPastDue,
	domain.SubStatusUnpaid,
	domain.SubStatusCanceled,
	domain.SubStatusPaused,
	domain.SubStatusIncomplete,
}

// nonTerminalStatuses are statuses from which at least one transition is valid.
var nonTerminalStatuses = []domain.SubscriptionStatus{
	domain.SubStatusTrialing,
	domain.SubStatusActive,
	domain.SubStatusPastDue,
	domain.SubStatusUnpaid,
	domain.SubStatusPaused,
	domain.SubStatusIncomplete,
}

// validTransitionsMap mirrors the FSM graph for test assertions.
var validTransitionsMap = map[domain.SubscriptionStatus][]domain.SubscriptionStatus{
	domain.SubStatusTrialing:   {domain.SubStatusActive, domain.SubStatusCanceled},
	domain.SubStatusActive:     {domain.SubStatusPastDue, domain.SubStatusCanceled, domain.SubStatusPaused},
	domain.SubStatusPastDue:    {domain.SubStatusActive, domain.SubStatusUnpaid, domain.SubStatusCanceled},
	domain.SubStatusUnpaid:     {domain.SubStatusActive, domain.SubStatusCanceled},
	domain.SubStatusPaused:     {domain.SubStatusActive, domain.SubStatusCanceled},
	domain.SubStatusIncomplete: {domain.SubStatusActive, domain.SubStatusCanceled},
	domain.SubStatusCanceled:   {},
}

// drawValidTransition picks a random valid (from, to) pair.
func drawValidTransition(t *rapid.T) (from, to domain.SubscriptionStatus) {
	fromIdx := rapid.IntRange(0, len(nonTerminalStatuses)-1).Draw(t, "from_idx")
	from = nonTerminalStatuses[fromIdx]
	targets := validTransitionsMap[from]
	if len(targets) == 0 {
		t.Skip("no valid transitions from this status")
	}
	toIdx := rapid.IntRange(0, len(targets)-1).Draw(t, "to_idx")
	to = targets[toIdx]
	return
}

// inMemoryFSMRepo is a thread-safe in-memory SubscriptionFSMRepository for tests.
type inMemoryFSMRepo struct {
	mu      sync.Mutex
	status  domain.SubscriptionStatus
	history []domain.SubscriptionStatusHistory
}

func newInMemoryFSMRepo(initial domain.SubscriptionStatus) *inMemoryFSMRepo {
	return &inMemoryFSMRepo{status: initial}
}

func (r *inMemoryFSMRepo) GetForUpdate(_ context.Context, orgID, subID uuid.UUID) (*domain.SubscriptionForUpdate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &domain.SubscriptionForUpdate{
		ID:             subID,
		OrganizationID: orgID,
		CustomerID:     uuid.New(),
		Status:         r.status,
	}, nil
}

func (r *inMemoryFSMRepo) UpdateStatus(_ context.Context, _, _ uuid.UUID, newStatus domain.SubscriptionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = newStatus
	return nil
}

func (r *inMemoryFSMRepo) InsertHistory(_ context.Context, entry domain.SubscriptionStatusHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history = append(r.history, entry)
	return nil
}

func (r *inMemoryFSMRepo) ListHistory(_ context.Context, _, _ uuid.UUID, limit int, cursor *domain.CursorToken) ([]domain.SubscriptionStatusHistory, *domain.CursorToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.history, nil, nil
}

func (r *inMemoryFSMRepo) historyLen() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.history)
}

func (r *inMemoryFSMRepo) currentStatus() domain.SubscriptionStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (r *inMemoryFSMRepo) lastHistory() domain.SubscriptionStatusHistory {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.history[len(r.history)-1]
}

// buildTransitionSvc creates a SubscriptionService wired to the given fsmRepo.
func buildTransitionSvc(fsmRepo *inMemoryFSMRepo) *SubscriptionService {
	txMgr := &mockTxManager{
		withTxFn: func(_ context.Context, fn func(domain.TxHandle) error) error {
			return fn(&mockTxHandle{subFSM: fsmRepo})
		},
	}
	return NewSubscriptionService(
		&mockSubscriptionRepo{},
		&mockSubscriptionItemRepo{},
		&mockScheduleRepo{},
		&mockSubscriptionHistoryRepo{},
		&mockPlanRepo{},
		&mockCustomerRepo{},
		txMgr,
		fsmRepo,
	)
}

// ── Property 1: Transition atômica — status e histórico ──────────────────────

// Feature: subscription-status-log-backend, Property 1: Transition atômica — status e histórico
// Para qualquer subscription em estado não-terminal e qualquer toStatus válido,
// após Transition: status == toStatus E len(history) aumentou em 1.
func TestProperty_SubTransition_AtomicStatusAndHistory(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		from, to := drawValidTransition(t)

		orgID := uuid.New()
		subID := uuid.New()
		fsmRepo := newInMemoryFSMRepo(from)
		svc := buildTransitionSvc(fsmRepo)

		histBefore := fsmRepo.historyLen()

		err := svc.Transition(context.Background(), orgID, subID, to, domain.TransitionEvent{
			Source: "api",
			Reason: "test",
		})
		if err != nil {
			t.Fatalf("expected no error for valid transition %s→%s, got %v", from, to, err)
		}

		if fsmRepo.currentStatus() != to {
			t.Fatalf("expected status=%s after transition, got %s", to, fsmRepo.currentStatus())
		}
		if fsmRepo.historyLen() != histBefore+1 {
			t.Fatalf("expected exactly 1 new history record, got %d new", fsmRepo.historyLen()-histBefore)
		}
	})
}

// ── Property 2: FSM inválida não persiste nada ────────────────────────────────

// Feature: subscription-status-log-backend, Property 2: FSM inválida não persiste nada
// Para qualquer subscription e toStatus inválido gerado pelo rapid,
// Transition retorna erro E status inalterado E zero novos registros em history.
func TestProperty_SubTransition_InvalidFSMNoPersist(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fromIdx := rapid.IntRange(0, len(allSubStatuses)-1).Draw(t, "from_idx")
		from := allSubStatuses[fromIdx]

		// Build a set of valid targets for this from-status.
		validTargets := map[domain.SubscriptionStatus]bool{}
		for _, v := range validTransitionsMap[from] {
			validTargets[v] = true
		}

		// Pick a to-status that is NOT valid (different from from, not in valid targets).
		var to domain.SubscriptionStatus
		found := false
		for _, s := range allSubStatuses {
			if s != from && !validTargets[s] {
				to = s
				found = true
				break
			}
		}
		if !found {
			t.Skip("no invalid transition available from this status")
		}

		orgID := uuid.New()
		subID := uuid.New()
		fsmRepo := newInMemoryFSMRepo(from)
		svc := buildTransitionSvc(fsmRepo)

		histBefore := fsmRepo.historyLen()

		err := svc.Transition(context.Background(), orgID, subID, to, domain.TransitionEvent{
			Source: "api",
			Reason: "invalid_test",
		})
		if err == nil {
			t.Fatalf("expected error for invalid transition %s→%s, got nil", from, to)
		}
		if fsmRepo.currentStatus() != from {
			t.Fatalf("expected status unchanged (%s), got %s", from, fsmRepo.currentStatus())
		}
		if fsmRepo.historyLen() != histBefore {
			t.Fatalf("expected no new history records, got %d new", fsmRepo.historyLen()-histBefore)
		}
	})
}

// ── Property 3: Campos do TransitionEvent preservados no histórico ────────────

// Feature: subscription-status-log-backend, Property 3: Campos do TransitionEvent preservados no histórico
// Para qualquer TransitionEvent com Source, SourceID e Reason arbitrários gerados pelo rapid,
// o registro de history contém exatamente esses valores.
func TestProperty_SubTransition_EventFieldsPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		from, to := drawValidTransition(t)

		source := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "source")
		reason := rapid.StringMatching(`[a-z_]{3,20}`).Draw(t, "reason")

		// Optionally include a SourceID.
		var sourceID *uuid.UUID
		if rapid.Bool().Draw(t, "has_source_id") {
			id := uuid.New()
			sourceID = &id
		}

		orgID := uuid.New()
		subID := uuid.New()
		fsmRepo := newInMemoryFSMRepo(from)
		svc := buildTransitionSvc(fsmRepo)

		err := svc.Transition(context.Background(), orgID, subID, to, domain.TransitionEvent{
			Source:   source,
			SourceID: sourceID,
			Reason:   reason,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		h := fsmRepo.lastHistory()

		if h.EventSource != source {
			t.Fatalf("expected EventSource=%q, got %q", source, h.EventSource)
		}
		if h.Reason != reason {
			t.Fatalf("expected Reason=%q, got %q", reason, h.Reason)
		}
		if sourceID == nil && h.EventSourceID != nil {
			t.Fatalf("expected nil EventSourceID, got %v", h.EventSourceID)
		}
		if sourceID != nil {
			if h.EventSourceID == nil {
				t.Fatalf("expected EventSourceID=%v, got nil", *sourceID)
			}
			if *h.EventSourceID != *sourceID {
				t.Fatalf("expected EventSourceID=%v, got %v", *sourceID, *h.EventSourceID)
			}
		}
		if h.NewStatus != to {
			t.Fatalf("expected NewStatus=%s, got %s", to, h.NewStatus)
		}
		if h.OldStatus == nil || *h.OldStatus != from {
			t.Fatalf("expected OldStatus=%s, got %v", from, h.OldStatus)
		}

		// Timestamps should be zero-value since inMemoryFSMRepo doesn't set them,
		// but the struct fields must be present.
		_ = h.CreatedAt
	})
}

// drawValidTransitionForWorker returns a (from, to) pair where from is a status
// the worker can encounter and to is a valid system-driven transition.
func drawValidTransitionForWorker(t *rapid.T) (from, to domain.SubscriptionStatus, reason string) {
	type workerTransition struct {
		from   domain.SubscriptionStatus
		to     domain.SubscriptionStatus
		reason string
	}
	transitions := []workerTransition{
		{domain.SubStatusActive, domain.SubStatusCanceled, "cancel_at_period_end"},
		{domain.SubStatusTrialing, domain.SubStatusActive, "trial_ended"},
		{domain.SubStatusActive, domain.SubStatusPastDue, "payment_failed"},
		{domain.SubStatusPastDue, domain.SubStatusUnpaid, "max_retries_exceeded"},
		{domain.SubStatusPastDue, domain.SubStatusActive, "payment_succeeded"},
	}
	idx := rapid.IntRange(0, len(transitions)-1).Draw(t, "transition_idx")
	tr := transitions[idx]
	return tr.from, tr.to, tr.reason
}

// workerTransitionSvc builds a SubscriptionService that captures the TransitionEvent
// passed to InsertHistory, so we can assert on event_source.
type captureInsertFSMRepo struct {
	*inMemoryFSMRepo
	captured []domain.SubscriptionStatusHistory
}

func (r *captureInsertFSMRepo) InsertHistory(ctx context.Context, entry domain.SubscriptionStatusHistory) error {
	r.captured = append(r.captured, entry)
	return r.inMemoryFSMRepo.InsertHistory(ctx, entry)
}

func buildWorkerTransitionSvc(fsmRepo *captureInsertFSMRepo) *SubscriptionService {
	txMgr := &mockTxManager{
		withTxFn: func(_ context.Context, fn func(domain.TxHandle) error) error {
			return fn(&mockTxHandle{subFSM: fsmRepo})
		},
	}
	return NewSubscriptionService(
		&mockSubscriptionRepo{},
		&mockSubscriptionItemRepo{},
		&mockScheduleRepo{},
		&mockSubscriptionHistoryRepo{},
		&mockPlanRepo{},
		&mockCustomerRepo{},
		txMgr,
		fsmRepo,
	)
}

// ── Property 4 helper: verify worker always uses EventSource "system" ─────────
// (The actual Property 4 test lives in internal/worker/recurrence_property_test.go)
// This helper is used by the service-level assertion below.

func TestProperty_SubTransition_WorkerEventSourceIsSystem(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		from, to, reason := drawValidTransitionForWorker(t)

		inner := newInMemoryFSMRepo(from)
		fsmRepo := &captureInsertFSMRepo{inMemoryFSMRepo: inner}
		svc := buildWorkerTransitionSvc(fsmRepo)

		orgID := uuid.New()
		subID := uuid.New()

		err := svc.Transition(context.Background(), orgID, subID, to, domain.TransitionEvent{
			Source: "system",
			Reason: reason,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(fsmRepo.captured) == 0 {
			t.Fatal("expected at least one history record")
		}
		last := fsmRepo.captured[len(fsmRepo.captured)-1]
		if last.EventSource != "system" {
			t.Fatalf("expected EventSource=system, got %q", last.EventSource)
		}
		if last.Reason == "" {
			t.Fatal("expected non-empty Reason")
		}
		_ = time.Now() // ensure time import is used
	})
}
