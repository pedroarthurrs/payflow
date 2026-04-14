# Implementação: Subscriptions

Guia passo a passo para implementar o módulo de Subscriptions. Siga a ordem exata — cada etapa depende da anterior. **Plans devem estar implementados antes de iniciar este módulo.**

## Sumário de arquivos

| Ação | Arquivo |
|------|---------|
| Criar | `internal/domain/subscription.go` |
| Criar | `internal/domain/subscription_fsm.go` |
| Modificar | `internal/domain/errors.go` |
| Modificar | `internal/domain/tx.go` |
| Criar | `internal/database/queries/subscription.sql` |
| Modificar | `sqlc.yml` |
| Executar | `make sqlc` |
| Criar | `internal/database/repository/subscription.go` |
| Modificar | `internal/database/repository/tx.go` |
| Modificar | `internal/database/connection.go` |
| Criar | `internal/service/subscription.go` |
| Modificar | `internal/api/dto/context.go` |
| Criar | `internal/api/handler/subscription.go` |
| Modificar | `internal/api/router/router.go` |

---

## 1. Domain

### `internal/domain/subscription.go` (criar)

```go
package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the lifecycle state of a subscription.
type SubscriptionStatus string

const (
	SubStatusTrialing   SubscriptionStatus = "trialing"
	SubStatusActive     SubscriptionStatus = "active"
	SubStatusPastDue    SubscriptionStatus = "past_due"
	SubStatusUnpaid     SubscriptionStatus = "unpaid"
	SubStatusCanceled   SubscriptionStatus = "canceled"
	SubStatusPaused     SubscriptionStatus = "paused"
	SubStatusIncomplete SubscriptionStatus = "incomplete"
)

// SubscriptionRepository defines persistence for subscriptions.
type SubscriptionRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input CreateSubscriptionInput) (*Subscription, error)
	GetByID(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*Subscription, error)
	List(ctx context.Context, organizationID uuid.UUID, params SubscriptionListParams) ([]Subscription, error)
	Count(ctx context.Context, organizationID uuid.UUID, params SubscriptionListParams) (int64, error)
	UpdateStatus(ctx context.Context, organizationID, subscriptionID uuid.UUID, newStatus SubscriptionStatus, reason string) (*Subscription, error)
	SetCancelAtPeriodEnd(ctx context.Context, organizationID, subscriptionID uuid.UUID, value bool) (*Subscription, error)
}

// SubscriptionItemRepository defines persistence for subscription items.
type SubscriptionItemRepository interface {
	CreateItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []SubscriptionItemInput) ([]SubscriptionItem, error)
	ReplaceItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []SubscriptionItemInput) ([]SubscriptionItem, error)
	ListBySubscription(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]SubscriptionItem, error)
}

// SubscriptionScheduleRepository defines persistence for subscription schedules.
type SubscriptionScheduleRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input CreateScheduleInput) (*SubscriptionSchedule, error)
	GetPending(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*SubscriptionSchedule, error)
	Cancel(ctx context.Context, organizationID, scheduleID uuid.UUID) error
}

// SubscriptionHistoryRepository defines read access to subscription status history.
type SubscriptionHistoryRepository interface {
	List(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]SubscriptionStatusHistory, error)
}

// Subscription represents a recurring billing agreement.
type Subscription struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	CustomerID         uuid.UUID
	Status             SubscriptionStatus
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
	Metadata           json.RawMessage
	Items              []SubscriptionItem
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SubscriptionItem links a plan to a subscription.
type SubscriptionItem struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	PlanID         uuid.UUID
	Quantity       int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SubscriptionStatusHistory records every status transition.
type SubscriptionStatusHistory struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	OldStatus      *SubscriptionStatus
	NewStatus      SubscriptionStatus
	Reason         string
	CreatedAt      time.Time
}

// SubscriptionSchedule represents a future change to a subscription.
type SubscriptionSchedule struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	ScheduledAt    time.Time
	TargetStatus   *SubscriptionStatus
	TargetPlanID   *uuid.UUID
	AppliedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateSubscriptionInput carries the data required to create a subscription.
type CreateSubscriptionInput struct {
	CustomerID uuid.UUID
	TrialEnd   *time.Time
	Metadata   json.RawMessage
	Items      []SubscriptionItemInput
}

// SubscriptionItemInput carries the data for a single subscription item.
type SubscriptionItemInput struct {
	PlanID   uuid.UUID
	Quantity int32
}

// CreateScheduleInput carries the data required to create a schedule.
type CreateScheduleInput struct {
	SubscriptionID uuid.UUID
	ScheduledAt    time.Time
	TargetStatus   *SubscriptionStatus
	TargetPlanID   *uuid.UUID
}

// SubscriptionListParams carries filter and pagination parameters for listing subscriptions.
type SubscriptionListParams struct {
	ListParams
	CustomerID  *uuid.UUID
	Status      []SubscriptionStatus
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}
```

### `internal/domain/subscription_fsm.go` (criar)

Segue o mesmo padrão de `internal/domain/payment_fsm.go`:

```go
package domain

import "fmt"

// SubscriptionStateMachine is the pure FSM for subscriptions — no infrastructure dependencies.
type SubscriptionStateMachine struct{}

// subscriptionTransitionRules is the immutable graph of valid transitions.
var subscriptionTransitionRules = map[SubscriptionStatus][]SubscriptionStatus{
	SubStatusTrialing:   {SubStatusActive, SubStatusCanceled},
	SubStatusActive:     {SubStatusPastDue, SubStatusCanceled, SubStatusPaused},
	SubStatusPastDue:    {SubStatusActive, SubStatusUnpaid, SubStatusCanceled},
	SubStatusUnpaid:     {SubStatusActive, SubStatusCanceled},
	SubStatusPaused:     {SubStatusActive, SubStatusCanceled},
	SubStatusIncomplete: {SubStatusActive, SubStatusCanceled},
	SubStatusCanceled:   {},
}

// subscriptionTerminalStates are irreversible statuses.
var subscriptionTerminalStates = map[SubscriptionStatus]bool{
	SubStatusCanceled: true,
}

// NewSubscriptionStateMachine creates a SubscriptionStateMachine.
func NewSubscriptionStateMachine() *SubscriptionStateMachine {
	return &SubscriptionStateMachine{}
}

// Validate returns nil if the transition from → to is valid, or an error otherwise.
func (sm *SubscriptionStateMachine) Validate(from, to SubscriptionStatus) error {
	if from == to {
		return ErrSameStateTransition
	}
	if subscriptionTerminalStates[from] {
		return ErrSubscriptionCanceled
	}
	for _, allowed := range subscriptionTransitionRules[from] {
		if allowed == to {
			return nil
		}
	}
	return fmt.Errorf("%w: %s → %s", ErrInvalidSubscriptionTransition, from, to)
}

// IsTerminal returns true if status is a terminal state.
func (sm *SubscriptionStateMachine) IsTerminal(status SubscriptionStatus) bool {
	return subscriptionTerminalStates[status]
}
```

> **Nota:** `ErrSameStateTransition` já deve existir em `domain/errors.go` (adicionado junto com os erros de payment). Se não existir, adicionar junto com os erros new abaixo.

### `internal/domain/errors.go` (modificar)

Adicionar ao bloco `var (...)`:

```go
	// Subscription errors
	ErrSubscriptionNotFound              = errors.New("subscription not found")
	ErrInvalidSubscriptionTransition     = errors.New("invalid subscription status transition")
	ErrSubscriptionCanceled              = errors.New("subscription is already canceled")
	ErrNoSubscriptionItems               = errors.New("subscription must have at least one item")
	ErrScheduleAlreadyPending            = errors.New("subscription already has a pending schedule")
	ErrScheduleNotFound                  = errors.New("subscription schedule not found")
```

Verificar se `ErrSameStateTransition` já existe (foi adicionado para payments). Se não existir, adicionar:

```go
	ErrSameStateTransition = errors.New("cannot transition to the same state")
```

### `internal/domain/tx.go` (modificar)

Adicionar `SubscriptionRepo()` e `SubscriptionItemRepo()` à interface `TxHandle` para suportar criação atômica de subscription + items:

```go
// TxHandle is passed to transaction callbacks, providing scoped repositories.
type TxHandle interface {
	CustomerRepo() CustomerRepository
	AddressRepo() AddressRepository
	CredentialRepo() CredentialRepository
	PaymentFSMRepo() PaymentFSMRepository
	// Subscription
	SubscriptionRepo() SubscriptionRepository
	SubscriptionItemRepo() SubscriptionItemRepository
}
```

---

## 2. SQL Queries

### `internal/database/queries/subscription.sql` (criar)

```sql
-- =============================================================================
-- Subscriptions
-- =============================================================================

-- name: CreateSubscription :one
INSERT INTO subscriptions (
    organization_id,
    customer_id,
    status,
    current_period_start,
    current_period_end,
    cancel_at_period_end,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions
WHERE organization_id = $1 AND id = $2
LIMIT 1;

-- name: GetSubscriptionForUpdate :one
SELECT id, organization_id, status
FROM subscriptions
WHERE organization_id = $1 AND id = $2
FOR UPDATE;

-- name: ListSubscriptions :many
SELECT * FROM subscriptions
WHERE organization_id = $1
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSubscriptions :one
SELECT count(*) FROM subscriptions
WHERE organization_id = $1
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'));

-- name: UpdateSubscriptionStatus :one
UPDATE subscriptions
SET status = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: SetCancelAtPeriodEnd :one
UPDATE subscriptions
SET cancel_at_period_end = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- =============================================================================
-- Subscription Items
-- =============================================================================

-- name: CreateSubscriptionItem :one
INSERT INTO subscription_items (organization_id, subscription_id, plan_id, quantity)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListSubscriptionItems :many
SELECT * FROM subscription_items
WHERE organization_id = $1 AND subscription_id = $2
ORDER BY created_at ASC;

-- name: DeleteSubscriptionItems :exec
DELETE FROM subscription_items
WHERE organization_id = $1 AND subscription_id = $2;

-- =============================================================================
-- Subscription Status History
-- =============================================================================

-- name: ListSubscriptionHistory :many
SELECT * FROM subscription_status_history
WHERE organization_id = $1 AND subscription_id = $2
ORDER BY created_at ASC;

-- =============================================================================
-- Subscription Schedules
-- =============================================================================

-- name: CreateSubscriptionSchedule :one
INSERT INTO subscription_schedules (
    organization_id,
    subscription_id,
    scheduled_at,
    target_status,
    target_plan_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPendingSchedule :one
SELECT * FROM subscription_schedules
WHERE organization_id = $1
  AND subscription_id = $2
  AND applied_at IS NULL
LIMIT 1;

-- name: CancelSchedule :exec
UPDATE subscription_schedules
SET applied_at = NOW(), updated_at = NOW()
WHERE organization_id = $1 AND id = $2 AND applied_at IS NULL;
```

> **Nota:** O filtro de múltiplos `status` (`?status=active&status=past_due`) **não** é incluído diretamente no SQL aqui, pois SQLC não suporta arrays de enum diretamente como parâmetro de forma simples. A estratégia adotada é filtrar no service/repository em memória após buscar por `organization_id`. Se a performance exigir, usar `sqlc.slice()` com o parâmetro de tipo correto em versão futura. Por ora, listar sem filtro de status e filtrar na camada Go.

---

## 3. Configurar SQLC

### `sqlc.yml` (modificar)

Adicionar como último item da lista `sql:`:

```yaml
  - schema: "deploy/migrations"
    queries: "internal/database/queries/subscription.sql"
    engine: "postgresql"
    gen:
      go:
        package: "subscriptiondb"
        out: "internal/database/dbgen/subscription"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
```

### Gerar código

```bash
make sqlc
```

Isso cria `internal/database/dbgen/subscription/` com os arquivos gerados. Os structs principais gerados serão:

```go
// Subscription — tabela subscriptions
type Subscription struct {
    ID                 uuid.UUID                       `json:"id"`
    OrganizationID     uuid.UUID                       `json:"organization_id"`
    CustomerID         uuid.UUID                       `json:"customer_id"`
    Status             SubscriptionStatusType          `json:"status"`
    CurrentPeriodStart time.Time                       `json:"current_period_start"`
    CurrentPeriodEnd   time.Time                       `json:"current_period_end"`
    CancelAtPeriodEnd  sql.NullBool                    `json:"cancel_at_period_end"`
    Metadata           pqtype.NullRawMessage           `json:"metadata"`
    CreatedAt          sql.NullTime                    `json:"created_at"`
    UpdatedAt          sql.NullTime                    `json:"updated_at"`
}

// SubscriptionStatusType — enum gerado do tipo subscription_status_type do Postgres
type SubscriptionStatusType string

const (
    SubscriptionStatusTypeTrialing   SubscriptionStatusType = "trialing"
    SubscriptionStatusTypeActive     SubscriptionStatusType = "active"
    SubscriptionStatusTypePastDue    SubscriptionStatusType = "past_due"
    SubscriptionStatusTypeUnpaid     SubscriptionStatusType = "unpaid"
    SubscriptionStatusTypeCanceled   SubscriptionStatusType = "canceled"
    SubscriptionStatusTypePaused     SubscriptionStatusType = "paused"
    SubscriptionStatusTypeIncomplete SubscriptionStatusType = "incomplete"
)
```

> Os nomes exatos dos campos e constantes devem ser verificados em `internal/database/dbgen/subscription/models.go` após executar `make sqlc`.

---

## 4. Repository

### `internal/database/repository/subscription.go` (criar)

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	subscriptiondb "i9pay/internal/database/dbgen/subscription"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// =============================================================================
// SubscriptionRepo
// =============================================================================

// SubscriptionRepo implements domain.SubscriptionRepository.
type SubscriptionRepo struct {
	q subscriptiondb.Querier
}

// NewSubscriptionRepo creates a SubscriptionRepo.
func NewSubscriptionRepo(q subscriptiondb.Querier) *SubscriptionRepo {
	return &SubscriptionRepo{q: q}
}

// Create inserts a new subscription (without items — items are inserted separately inside a transaction).
func (r *SubscriptionRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateSubscriptionInput) (*domain.Subscription, error) {
	status := subscriptiondb.SubscriptionStatusTypeIncomplete
	periodStart := time.Now().UTC()
	// period_end must be set by the caller via input or calculated in service before persisting
	// We default to periodStart + 1 month here as a safety net — the service sets the real value
	periodEnd := periodStart.AddDate(0, 1, 0)

	row, err := r.q.CreateSubscription(ctx, subscriptiondb.CreateSubscriptionParams{
		OrganizationID:     organizationID,
		CustomerID:         input.CustomerID,
		Status:             status,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		CancelAtPeriodEnd:  sql.NullBool{Bool: false, Valid: true},
		Metadata:           toNullRawMessage(input.Metadata),
	})
	if err != nil {
		slog.Error("failed to create subscription", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSubscription(row)
	return &s, nil
}

// GetByID retrieves a subscription by ID.
func (r *SubscriptionRepo) GetByID(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	row, err := r.q.GetSubscriptionByID(ctx, subscriptiondb.GetSubscriptionByIDParams{
		OrganizationID: organizationID,
		ID:             subscriptionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		slog.Error("failed to get subscription", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSubscription(row)
	return &s, nil
}

// List returns subscriptions for an organization with optional filters.
func (r *SubscriptionRepo) List(ctx context.Context, organizationID uuid.UUID, params domain.SubscriptionListParams) ([]domain.Subscription, error) {
	rows, err := r.q.ListSubscriptions(ctx, subscriptiondb.ListSubscriptionsParams{
		OrganizationID: organizationID,
		CustomerID:     toNullUUID(params.CustomerID),
		CreatedFrom:    toNullTime(params.CreatedFrom),
		CreatedTo:      toNullTime(params.CreatedTo),
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list subscriptions", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}

	subs := make([]domain.Subscription, 0, len(rows))
	for _, row := range rows {
		s := toDomainSubscription(row)
		// Apply in-memory status filter when provided
		if len(params.Status) > 0 && !containsStatus(params.Status, s.Status) {
			continue
		}
		subs = append(subs, s)
	}
	return subs, nil
}

// Count returns the total count matching the filters (without status filtering — done in-memory).
func (r *SubscriptionRepo) Count(ctx context.Context, organizationID uuid.UUID, params domain.SubscriptionListParams) (int64, error) {
	count, err := r.q.CountSubscriptions(ctx, subscriptiondb.CountSubscriptionsParams{
		OrganizationID: organizationID,
		CustomerID:     toNullUUID(params.CustomerID),
		CreatedFrom:    toNullTime(params.CreatedFrom),
		CreatedTo:      toNullTime(params.CreatedTo),
	})
	if err != nil {
		slog.Error("failed to count subscriptions", "organization_id", organizationID, "error", err)
		return 0, domain.ErrInternal
	}
	return count, nil
}

// UpdateStatus transitions the subscription to a new status.
func (r *SubscriptionRepo) UpdateStatus(ctx context.Context, organizationID, subscriptionID uuid.UUID, newStatus domain.SubscriptionStatus, reason string) (*domain.Subscription, error) {
	row, err := r.q.UpdateSubscriptionStatus(ctx, subscriptiondb.UpdateSubscriptionStatusParams{
		OrganizationID: organizationID,
		ID:             subscriptionID,
		Status:         subscriptiondb.SubscriptionStatusType(newStatus),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		slog.Error("failed to update subscription status", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSubscription(row)
	return &s, nil
}

// SetCancelAtPeriodEnd updates the cancel_at_period_end flag.
func (r *SubscriptionRepo) SetCancelAtPeriodEnd(ctx context.Context, organizationID, subscriptionID uuid.UUID, value bool) (*domain.Subscription, error) {
	row, err := r.q.SetCancelAtPeriodEnd(ctx, subscriptiondb.SetCancelAtPeriodEndParams{
		OrganizationID:    organizationID,
		ID:                subscriptionID,
		CancelAtPeriodEnd: sql.NullBool{Bool: value, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		slog.Error("failed to set cancel_at_period_end", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSubscription(row)
	return &s, nil
}

// =============================================================================
// SubscriptionItemRepo
// =============================================================================

// SubscriptionItemRepo implements domain.SubscriptionItemRepository.
type SubscriptionItemRepo struct {
	q subscriptiondb.Querier
}

// NewSubscriptionItemRepo creates a SubscriptionItemRepo.
func NewSubscriptionItemRepo(q subscriptiondb.Querier) *SubscriptionItemRepo {
	return &SubscriptionItemRepo{q: q}
}

// CreateItems inserts multiple items for a subscription.
func (r *SubscriptionItemRepo) CreateItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	result := make([]domain.SubscriptionItem, 0, len(items))
	for _, item := range items {
		row, err := r.q.CreateSubscriptionItem(ctx, subscriptiondb.CreateSubscriptionItemParams{
			OrganizationID: organizationID,
			SubscriptionID: subscriptionID,
			PlanID:         item.PlanID,
			Quantity:       sql.NullInt32{Int32: item.Quantity, Valid: true},
		})
		if err != nil {
			slog.Error("failed to create subscription item", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
			return nil, domain.ErrInternal
		}
		result = append(result, toDomainSubscriptionItem(row))
	}
	return result, nil
}

// ReplaceItems deletes existing items and inserts the new set atomically.
// Must be called inside a transaction (txHandle.SubscriptionItemRepo()).
func (r *SubscriptionItemRepo) ReplaceItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	if err := r.q.DeleteSubscriptionItems(ctx, subscriptiondb.DeleteSubscriptionItemsParams{
		OrganizationID: organizationID,
		SubscriptionID: subscriptionID,
	}); err != nil {
		slog.Error("failed to delete subscription items", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	return r.CreateItems(ctx, organizationID, subscriptionID, items)
}

// ListBySubscription returns all items for a subscription.
func (r *SubscriptionItemRepo) ListBySubscription(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]domain.SubscriptionItem, error) {
	rows, err := r.q.ListSubscriptionItems(ctx, subscriptiondb.ListSubscriptionItemsParams{
		OrganizationID: organizationID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		slog.Error("failed to list subscription items", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	result := make([]domain.SubscriptionItem, len(rows))
	for i, row := range rows {
		result[i] = toDomainSubscriptionItem(row)
	}
	return result, nil
}

// =============================================================================
// SubscriptionScheduleRepo
// =============================================================================

// SubscriptionScheduleRepo implements domain.SubscriptionScheduleRepository.
type SubscriptionScheduleRepo struct {
	q subscriptiondb.Querier
}

// NewSubscriptionScheduleRepo creates a SubscriptionScheduleRepo.
func NewSubscriptionScheduleRepo(q subscriptiondb.Querier) *SubscriptionScheduleRepo {
	return &SubscriptionScheduleRepo{q: q}
}

// Create inserts a new schedule.
func (r *SubscriptionScheduleRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateScheduleInput) (*domain.SubscriptionSchedule, error) {
	params := subscriptiondb.CreateSubscriptionScheduleParams{
		OrganizationID: organizationID,
		SubscriptionID: input.SubscriptionID,
		ScheduledAt:    input.ScheduledAt,
	}
	if input.TargetStatus != nil {
		params.TargetStatus = subscriptiondb.NullSubscriptionStatusType{
			SubscriptionStatusType: subscriptiondb.SubscriptionStatusType(*input.TargetStatus),
			Valid:                  true,
		}
	}
	if input.TargetPlanID != nil {
		params.TargetPlanID = uuid.NullUUID{UUID: *input.TargetPlanID, Valid: true}
	}
	row, err := r.q.CreateSubscriptionSchedule(ctx, params)
	if err != nil {
		slog.Error("failed to create subscription schedule", "organization_id", organizationID, "subscription_id", input.SubscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSchedule(row)
	return &s, nil
}

// GetPending retrieves the pending (unapplied) schedule for a subscription.
func (r *SubscriptionScheduleRepo) GetPending(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*domain.SubscriptionSchedule, error) {
	row, err := r.q.GetPendingSchedule(ctx, subscriptiondb.GetPendingScheduleParams{
		OrganizationID: organizationID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrScheduleNotFound
		}
		slog.Error("failed to get pending schedule", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSchedule(row)
	return &s, nil
}

// Cancel marks a schedule as applied (effectively canceling it without deleting).
func (r *SubscriptionScheduleRepo) Cancel(ctx context.Context, organizationID, scheduleID uuid.UUID) error {
	if err := r.q.CancelSchedule(ctx, subscriptiondb.CancelScheduleParams{
		OrganizationID: organizationID,
		ID:             scheduleID,
	}); err != nil {
		slog.Error("failed to cancel schedule", "organization_id", organizationID, "schedule_id", scheduleID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// =============================================================================
// SubscriptionHistoryRepo
// =============================================================================

// SubscriptionHistoryRepo implements domain.SubscriptionHistoryRepository.
type SubscriptionHistoryRepo struct {
	q subscriptiondb.Querier
}

// NewSubscriptionHistoryRepo creates a SubscriptionHistoryRepo.
func NewSubscriptionHistoryRepo(q subscriptiondb.Querier) *SubscriptionHistoryRepo {
	return &SubscriptionHistoryRepo{q: q}
}

// List returns all history entries for a subscription, ordered by created_at ASC.
func (r *SubscriptionHistoryRepo) List(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]domain.SubscriptionStatusHistory, error) {
	rows, err := r.q.ListSubscriptionHistory(ctx, subscriptiondb.ListSubscriptionHistoryParams{
		OrganizationID: organizationID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		slog.Error("failed to list subscription history", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	result := make([]domain.SubscriptionStatusHistory, len(rows))
	for i, row := range rows {
		result[i] = toDomainHistory(row)
	}
	return result, nil
}

// =============================================================================
// Mapping helpers
// =============================================================================

func toDomainSubscription(row subscriptiondb.Subscription) domain.Subscription {
	s := domain.Subscription{
		ID:                 row.ID,
		OrganizationID:     row.OrganizationID,
		CustomerID:         row.CustomerID,
		Status:             domain.SubscriptionStatus(row.Status),
		CurrentPeriodStart: row.CurrentPeriodStart,
		CurrentPeriodEnd:   row.CurrentPeriodEnd,
	}
	if row.CancelAtPeriodEnd.Valid {
		s.CancelAtPeriodEnd = row.CancelAtPeriodEnd.Bool
	}
	if row.Metadata.Valid {
		s.Metadata = row.Metadata.RawMessage
	}
	if row.CreatedAt.Valid {
		s.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		s.UpdatedAt = row.UpdatedAt.Time
	}
	return s
}

func toDomainSubscriptionItem(row subscriptiondb.SubscriptionItem) domain.SubscriptionItem {
	item := domain.SubscriptionItem{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		SubscriptionID: row.SubscriptionID,
		PlanID:         row.PlanID,
		Quantity:       1,
	}
	if row.Quantity.Valid {
		item.Quantity = row.Quantity.Int32
	}
	if row.CreatedAt.Valid {
		item.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = row.UpdatedAt.Time
	}
	return item
}

func toDomainHistory(row subscriptiondb.SubscriptionStatusHistory) domain.SubscriptionStatusHistory {
	h := domain.SubscriptionStatusHistory{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		SubscriptionID: row.SubscriptionID,
		NewStatus:      domain.SubscriptionStatus(row.NewStatus),
	}
	if row.OldStatus.Valid {
		s := domain.SubscriptionStatus(row.OldStatus.SubscriptionStatusType)
		h.OldStatus = &s
	}
	if row.Reason.Valid {
		h.Reason = row.Reason.String
	}
	if row.CreatedAt.Valid {
		h.CreatedAt = row.CreatedAt.Time
	}
	return h
}

func toDomainSchedule(row subscriptiondb.SubscriptionSchedule) domain.SubscriptionSchedule {
	s := domain.SubscriptionSchedule{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		SubscriptionID: row.SubscriptionID,
		ScheduledAt:    row.ScheduledAt,
	}
	if row.TargetStatus.Valid {
		st := domain.SubscriptionStatus(row.TargetStatus.SubscriptionStatusType)
		s.TargetStatus = &st
	}
	if row.TargetPlanID.Valid {
		s.TargetPlanID = &row.TargetPlanID.UUID
	}
	if row.AppliedAt.Valid {
		s.AppliedAt = &row.AppliedAt.Time
	}
	if row.CreatedAt.Valid {
		s.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		s.UpdatedAt = row.UpdatedAt.Time
	}
	return s
}

func containsStatus(statuses []domain.SubscriptionStatus, s domain.SubscriptionStatus) bool {
	for _, v := range statuses {
		if v == s {
			return true
		}
	}
	return false
}

func toNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// Ensure interfaces are satisfied at compile time.
var _ domain.SubscriptionRepository       = (*SubscriptionRepo)(nil)
var _ domain.SubscriptionItemRepository   = (*SubscriptionItemRepo)(nil)
var _ domain.SubscriptionScheduleRepository = (*SubscriptionScheduleRepo)(nil)
var _ domain.SubscriptionHistoryRepository = (*SubscriptionHistoryRepo)(nil)
```

> **Nota:** Verificar os nomes exatos dos campos gerados em `dbgen/subscription/models.go` após `make sqlc`. Especialmente `NullSubscriptionStatusType` — pode precisar de ajuste no nome do campo `.SubscriptionStatusType`.

### `internal/database/repository/tx.go` (modificar)

Adicionar o import do pacote `subscriptiondb`, o campo no `txHandle`, e os dois métodos:

**Adicionar import:**
```go
subscriptiondb "i9pay/internal/database/dbgen/subscription"
```

**Adicionar campos no `txHandle`:**
```go
type txHandle struct {
	customerQ      *customerdb.Queries
	addressQ       *addressdb.Queries
	credentialQ    *credentialdb.Queries
	paymentQ       *paymentdb.Queries
	subscriptionQ  *subscriptiondb.Queries  // adicionar
}
```

**Adicionar na inicialização do `handle` dentro de `WithTx`:**
```go
handle := &txHandle{
	customerQ:      customerdb.New(sqlTx),
	addressQ:       addressdb.New(sqlTx),
	credentialQ:    credentialdb.New(sqlTx),
	paymentQ:       paymentdb.New(sqlTx),
	subscriptionQ:  subscriptiondb.New(sqlTx),  // adicionar
}
```

**Adicionar os dois métodos:**
```go
func (h *txHandle) SubscriptionRepo() domain.SubscriptionRepository {
	return NewSubscriptionRepo(h.subscriptionQ)
}

func (h *txHandle) SubscriptionItemRepo() domain.SubscriptionItemRepository {
	return NewSubscriptionItemRepo(h.subscriptionQ)
}
```

---

## 5. Connection

### `internal/database/connection.go` (modificar)

**Adicionar import:**
```go
subscriptiondb "i9pay/internal/database/dbgen/subscription"
```

**Adicionar campo na struct `DB`:**
```go
SubscriptionQ subscriptiondb.Querier
```

**Adicionar inicialização em `New`:**
```go
SubscriptionQ: subscriptiondb.New(db),
```

---

## 6. Service

### `internal/service/subscription.go` (criar)

```go
package service

import (
	"context"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// SubscriptionService handles subscription business logic.
type SubscriptionService struct {
	fsm       *domain.SubscriptionStateMachine
	subs      domain.SubscriptionRepository
	items     domain.SubscriptionItemRepository
	schedules domain.SubscriptionScheduleRepository
	history   domain.SubscriptionHistoryRepository
	plans     domain.PlanRepository
	customers domain.CustomerRepository
	txMgr     domain.TxManager
}

// NewSubscriptionService creates a SubscriptionService.
func NewSubscriptionService(
	subs domain.SubscriptionRepository,
	items domain.SubscriptionItemRepository,
	schedules domain.SubscriptionScheduleRepository,
	history domain.SubscriptionHistoryRepository,
	plans domain.PlanRepository,
	customers domain.CustomerRepository,
	txMgr domain.TxManager,
) *SubscriptionService {
	return &SubscriptionService{
		fsm:       domain.NewSubscriptionStateMachine(),
		subs:      subs,
		items:     items,
		schedules: schedules,
		history:   history,
		plans:     plans,
		customers: customers,
		txMgr:     txMgr,
	}
}

// Create creates a subscription with its items atomically inside a transaction.
func (s *SubscriptionService) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateSubscriptionInput) (*domain.Subscription, error) {
	if len(input.Items) == 0 {
		return nil, domain.ErrNoSubscriptionItems
	}

	// Validate customer
	if _, err := s.customers.GetByID(ctx, organizationID, input.CustomerID); err != nil {
		return nil, err
	}

	// Validate all plans exist, are active, and belong to the org
	for _, item := range input.Items {
		plan, err := s.plans.GetByID(ctx, organizationID, item.PlanID)
		if err != nil {
			return nil, err
		}
		if !plan.Active {
			return nil, domain.ErrPlanNotFound // plan inactive — treat as not found for this context
		}
	}

	// Determine initial status and period
	now := time.Now().UTC()
	initialStatus := domain.SubStatusIncomplete
	if input.TrialEnd != nil && input.TrialEnd.After(now) {
		initialStatus = domain.SubStatusTrialing
	}

	// Calculate period from first item's plan
	firstPlan, _ := s.plans.GetByID(ctx, organizationID, input.Items[0].PlanID)
	periodEnd := calculatePeriodEnd(now, firstPlan.Interval, int(firstPlan.IntervalCount))

	var sub *domain.Subscription
	err := s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		subRepo := tx.SubscriptionRepo()
		itemRepo := tx.SubscriptionItemRepo()

		created, err := subRepo.Create(ctx, organizationID, input)
		if err != nil {
			return err
		}

		// Override status and period (Create sets defaults; we update here)
		updated, err := subRepo.UpdateStatus(ctx, organizationID, created.ID, initialStatus, "")
		if err != nil {
			return err
		}
		_ = periodEnd // period update can be done via a dedicated SQL later if needed
		sub = updated

		insertedItems, err := itemRepo.CreateItems(ctx, organizationID, created.ID, input.Items)
		if err != nil {
			return err
		}
		sub.Items = insertedItems
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// GetByID returns a subscription with its items populated.
func (s *SubscriptionService) GetByID(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subs.GetByID(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}
	items, err := s.items.ListBySubscription(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}
	sub.Items = items
	return sub, nil
}

// List returns subscriptions with pagination.
func (s *SubscriptionService) List(ctx context.Context, organizationID uuid.UUID, params domain.SubscriptionListParams) ([]domain.Subscription, int64, error) {
	subs, err := s.subs.List(ctx, organizationID, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.subs.Count(ctx, organizationID, params)
	if err != nil {
		return nil, 0, err
	}
	return subs, total, nil
}

// Cancel cancels a subscription immediately or at period end.
func (s *SubscriptionService) Cancel(ctx context.Context, organizationID, subscriptionID uuid.UUID, reason string, cancelAtPeriodEnd bool) (*domain.Subscription, error) {
	sub, err := s.subs.GetByID(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}

	if cancelAtPeriodEnd {
		return s.subs.SetCancelAtPeriodEnd(ctx, organizationID, subscriptionID, true)
	}

	if err := s.fsm.Validate(sub.Status, domain.SubStatusCanceled); err != nil {
		return nil, err
	}
	return s.subs.UpdateStatus(ctx, organizationID, subscriptionID, domain.SubStatusCanceled, reason)
}

// Pause pauses an active subscription.
func (s *SubscriptionService) Pause(ctx context.Context, organizationID, subscriptionID uuid.UUID, reason string) (*domain.Subscription, error) {
	sub, err := s.subs.GetByID(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}
	if err := s.fsm.Validate(sub.Status, domain.SubStatusPaused); err != nil {
		return nil, err
	}
	return s.subs.UpdateStatus(ctx, organizationID, subscriptionID, domain.SubStatusPaused, reason)
}

// Resume resumes a paused subscription.
func (s *SubscriptionService) Resume(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subs.GetByID(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}
	if err := s.fsm.Validate(sub.Status, domain.SubStatusActive); err != nil {
		return nil, err
	}
	return s.subs.UpdateStatus(ctx, organizationID, subscriptionID, domain.SubStatusActive, "resumed")
}

// ReplaceItems replaces all items of a subscription atomically.
func (s *SubscriptionService) ReplaceItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, itemInputs []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	if len(itemInputs) == 0 {
		return nil, domain.ErrNoSubscriptionItems
	}

	// Validate subscription exists
	if _, err := s.subs.GetByID(ctx, organizationID, subscriptionID); err != nil {
		return nil, err
	}

	// Validate all plans
	for _, item := range itemInputs {
		plan, err := s.plans.GetByID(ctx, organizationID, item.PlanID)
		if err != nil {
			return nil, err
		}
		if !plan.Active {
			return nil, domain.ErrPlanNotFound
		}
	}

	var insertedItems []domain.SubscriptionItem
	err := s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		itemRepo := tx.SubscriptionItemRepo()
		var err error
		insertedItems, err = itemRepo.ReplaceItems(ctx, organizationID, subscriptionID, itemInputs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return insertedItems, nil
}

// ListItems returns items for a subscription.
func (s *SubscriptionService) ListItems(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]domain.SubscriptionItem, error) {
	if _, err := s.subs.GetByID(ctx, organizationID, subscriptionID); err != nil {
		return nil, err
	}
	return s.items.ListBySubscription(ctx, organizationID, subscriptionID)
}

// ListHistory returns the status change history for a subscription.
func (s *SubscriptionService) ListHistory(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]domain.SubscriptionStatusHistory, error) {
	if _, err := s.subs.GetByID(ctx, organizationID, subscriptionID); err != nil {
		return nil, err
	}
	return s.history.List(ctx, organizationID, subscriptionID)
}

// CreateSchedule creates a future status or plan change schedule.
func (s *SubscriptionService) CreateSchedule(ctx context.Context, organizationID, subscriptionID uuid.UUID, input domain.CreateScheduleInput) (*domain.SubscriptionSchedule, error) {
	// Subscription must exist and not be canceled
	sub, err := s.subs.GetByID(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}
	if s.fsm.IsTerminal(sub.Status) {
		return nil, domain.ErrSubscriptionCanceled
	}

	// Only one pending schedule at a time
	if _, err := s.schedules.GetPending(ctx, organizationID, subscriptionID); err == nil {
		return nil, domain.ErrScheduleAlreadyPending
	}

	input.SubscriptionID = subscriptionID
	return s.schedules.Create(ctx, organizationID, input)
}

// CancelSchedule cancels the pending schedule for a subscription.
func (s *SubscriptionService) CancelSchedule(ctx context.Context, organizationID, subscriptionID uuid.UUID) error {
	schedule, err := s.schedules.GetPending(ctx, organizationID, subscriptionID)
	if err != nil {
		return err
	}
	return s.schedules.Cancel(ctx, organizationID, schedule.ID)
}

// calculatePeriodEnd computes the end of the billing period from the start date.
func calculatePeriodEnd(start time.Time, interval string, intervalCount int) time.Time {
	switch interval {
	case "day":
		return start.AddDate(0, 0, intervalCount)
	case "week":
		return start.AddDate(0, 0, intervalCount*7)
	case "month":
		return start.AddDate(0, intervalCount, 0)
	case "year":
		return start.AddDate(intervalCount, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}
```

> **Nota sobre `Create`:** A query `CreateSubscription` define `period_start` e `period_end` de forma fixa. Para que o service controle os valores corretos de status e período, a abordagem mais limpa é adicionar uma query `UpdateSubscriptionPeriod :exec` e chamar logo após o `Create` dentro da mesma transação. Alternativa: passar status, period_start e period_end como campos do `CreateSubscriptionInput` e receber no SQL — preferível para remover a dependência do status padrão do banco.

---

## 7. HTTP

### `internal/api/dto/context.go` (modificar)

Adicionar após a função `PlanID` (adicionada na implementação de Plans):

```go
// SubscriptionID parses the ":subscription_id" path parameter as a UUID.
func SubscriptionID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("subscription_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid subscription id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// ScheduleID parses the ":schedule_id" path parameter as a UUID.
func ScheduleID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("schedule_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid schedule id"))
		return uuid.UUID{}, false
	}
	return id, true
}
```

### `internal/api/handler/subscription.go` (criar)

```go
package handler

import (
	"errors"
	"net/http"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscriptionHandler handles subscription-related HTTP endpoints.
type SubscriptionHandler struct {
	svc *service.SubscriptionService
}

// NewSubscriptionHandler creates a SubscriptionHandler.
func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

// RegisterRoutes registers all subscription routes.
func (h *SubscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	subs := rg.Group("/subscriptions")
	subs.POST("", h.Create)
	subs.GET("", h.List)
	subs.GET("/:subscription_id", h.GetByID)
	subs.POST("/:subscription_id/cancel", middleware.RequireRole("member"), h.Cancel)
	subs.POST("/:subscription_id/pause", middleware.RequireRole("member"), h.Pause)
	subs.POST("/:subscription_id/resume", middleware.RequireRole("member"), h.Resume)
	subs.PUT("/:subscription_id/items", middleware.RequireRole(domain.RoleAdmin), h.ReplaceItems)
	subs.GET("/:subscription_id/items", h.ListItems)
	subs.GET("/:subscription_id/history", middleware.RequireRole(domain.RoleAdmin), h.ListHistory)
	subs.POST("/:subscription_id/schedules", middleware.RequireRole(domain.RoleAdmin), h.CreateSchedule)
	subs.DELETE("/:subscription_id/schedules/pending", middleware.RequireRole(domain.RoleAdmin), h.CancelSchedule)
}

// =============================================================================
// Request / Response DTOs
// =============================================================================

type subscriptionItemRequest struct {
	PlanID   uuid.UUID `json:"plan_id"  binding:"required"`
	Quantity int32     `json:"quantity" binding:"omitempty,min=1"`
}

type createSubscriptionRequest struct {
	CustomerID uuid.UUID                  `json:"customer_id" binding:"required"`
	TrialEnd   *time.Time                 `json:"trial_end"   binding:"omitempty"`
	Metadata   map[string]any             `json:"metadata"    binding:"omitempty"`
	Items      []subscriptionItemRequest  `json:"items"       binding:"required,min=1"`
}

type cancelSubscriptionRequest struct {
	Reason            string `json:"reason"              binding:"required"`
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
}

type pauseResumeRequest struct {
	Reason string `json:"reason" binding:"omitempty"`
}

type createScheduleRequest struct {
	ScheduledAt  time.Time  `json:"scheduled_at"  binding:"required"`
	TargetStatus *string    `json:"target_status" binding:"omitempty"`
	TargetPlanID *uuid.UUID `json:"target_plan_id" binding:"omitempty"`
}

type subscriptionItemResponse struct {
	ID        uuid.UUID `json:"id"`
	PlanID    uuid.UUID `json:"plan_id"`
	Quantity  int32     `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type subscriptionResponse struct {
	ID                 uuid.UUID                  `json:"id"`
	OrganizationID     uuid.UUID                  `json:"organization_id"`
	CustomerID         uuid.UUID                  `json:"customer_id"`
	Status             string                     `json:"status"`
	CurrentPeriodStart time.Time                  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                  `json:"current_period_end"`
	CancelAtPeriodEnd  bool                       `json:"cancel_at_period_end"`
	Metadata           any                        `json:"metadata,omitempty"`
	Items              []subscriptionItemResponse `json:"items,omitempty"`
	CreatedAt          time.Time                  `json:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at"`
}

type subscriptionHistoryResponse struct {
	ID        uuid.UUID `json:"id"`
	OldStatus *string   `json:"old_status,omitempty"`
	NewStatus string    `json:"new_status"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type scheduleResponse struct {
	ID             uuid.UUID  `json:"id"`
	SubscriptionID uuid.UUID  `json:"subscription_id"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	TargetStatus   *string    `json:"target_status,omitempty"`
	TargetPlanID   *uuid.UUID `json:"target_plan_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func toSubResponse(s *domain.Subscription) subscriptionResponse {
	resp := subscriptionResponse{
		ID:                 s.ID,
		OrganizationID:     s.OrganizationID,
		CustomerID:         s.CustomerID,
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		CancelAtPeriodEnd:  s.CancelAtPeriodEnd,
		Metadata:           s.Metadata,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
	if len(s.Items) > 0 {
		resp.Items = make([]subscriptionItemResponse, len(s.Items))
		for i, item := range s.Items {
			resp.Items[i] = subscriptionItemResponse{
				ID:        item.ID,
				PlanID:    item.PlanID,
				Quantity:  item.Quantity,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			}
		}
	}
	return resp
}

func toScheduleResponse(s *domain.SubscriptionSchedule) scheduleResponse {
	r := scheduleResponse{
		ID:             s.ID,
		SubscriptionID: s.SubscriptionID,
		ScheduledAt:    s.ScheduledAt,
		TargetPlanID:   s.TargetPlanID,
		CreatedAt:      s.CreatedAt,
	}
	if s.TargetStatus != nil {
		str := string(*s.TargetStatus)
		r.TargetStatus = &str
	}
	return r
}

// =============================================================================
// Handlers
// =============================================================================

// Create handles POST /v1/subscriptions.
func (h *SubscriptionHandler) Create(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var req createSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]domain.SubscriptionItemInput, len(req.Items))
	for i, item := range req.Items {
		q := item.Quantity
		if q == 0 {
			q = 1
		}
		items[i] = domain.SubscriptionItemInput{PlanID: item.PlanID, Quantity: q}
	}

	input := domain.CreateSubscriptionInput{
		CustomerID: req.CustomerID,
		TrialEnd:   req.TrialEnd,
		Items:      items,
	}

	sub, err := h.svc.Create(c.Request.Context(), orgID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, toSubResponse(sub))
}

// List handles GET /v1/subscriptions.
func (h *SubscriptionHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	dateRange, ok := dto.ParseDateRange(c)
	if !ok {
		return
	}

	params := domain.SubscriptionListParams{
		ListParams:  pag.ToListParams(),
		CreatedFrom: dateRange.From,
		CreatedTo:   dateRange.To,
	}

	if raw := c.Query("customer_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid customer_id: must be a valid UUID"))
			return
		}
		params.CustomerID = &id
	}

	// Multiple status values: ?status=active&status=past_due
	for _, s := range c.QueryArray("status") {
		params.Status = append(params.Status, domain.SubscriptionStatus(s))
	}

	subs, total, err := h.svc.List(c.Request.Context(), orgID, params)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]subscriptionResponse, len(subs))
	for i := range subs {
		resp[i] = toSubResponse(&subs[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[subscriptionResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  total,
	})
}

// GetByID handles GET /v1/subscriptions/:subscription_id.
func (h *SubscriptionHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	sub, err := h.svc.GetByID(c.Request.Context(), orgID, subID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			_ = c.Error(apierr.NotFound("subscription not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toSubResponse(sub))
}

// Cancel handles POST /v1/subscriptions/:subscription_id/cancel.
func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	var req cancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	sub, err := h.svc.Cancel(c.Request.Context(), orgID, subID, req.Reason, req.CancelAtPeriodEnd)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		case errors.Is(err, domain.ErrSubscriptionCanceled):
			_ = c.Error(apierr.UnprocessableEntity("subscription is already canceled"))
		case errors.Is(err, domain.ErrInvalidSubscriptionTransition):
			_ = c.Error(apierr.Conflict(err.Error()))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, toSubResponse(sub))
}

// Pause handles POST /v1/subscriptions/:subscription_id/pause.
func (h *SubscriptionHandler) Pause(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	var req pauseResumeRequest
	_ = c.ShouldBindJSON(&req) // optional body

	sub, err := h.svc.Pause(c.Request.Context(), orgID, subID, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		case errors.Is(err, domain.ErrSubscriptionCanceled):
			_ = c.Error(apierr.UnprocessableEntity("subscription is already canceled"))
		case errors.Is(err, domain.ErrInvalidSubscriptionTransition):
			_ = c.Error(apierr.Conflict(err.Error()))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, toSubResponse(sub))
}

// Resume handles POST /v1/subscriptions/:subscription_id/resume.
func (h *SubscriptionHandler) Resume(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	sub, err := h.svc.Resume(c.Request.Context(), orgID, subID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		case errors.Is(err, domain.ErrInvalidSubscriptionTransition):
			_ = c.Error(apierr.Conflict(err.Error()))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, toSubResponse(sub))
}

// ReplaceItems handles PUT /v1/subscriptions/:subscription_id/items.
func (h *SubscriptionHandler) ReplaceItems(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	var req []subscriptionItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}
	if len(req) == 0 {
		_ = c.Error(apierr.BadRequest("items must not be empty"))
		return
	}

	items := make([]domain.SubscriptionItemInput, len(req))
	for i, item := range req {
		q := item.Quantity
		if q == 0 {
			q = 1
		}
		items[i] = domain.SubscriptionItemInput{PlanID: item.PlanID, Quantity: q}
	}

	inserted, err := h.svc.ReplaceItems(c.Request.Context(), orgID, subID, items)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			_ = c.Error(apierr.NotFound("subscription not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]subscriptionItemResponse, len(inserted))
	for i, item := range inserted {
		resp[i] = subscriptionItemResponse{
			ID:        item.ID,
			PlanID:    item.PlanID,
			Quantity:  item.Quantity,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// ListItems handles GET /v1/subscriptions/:subscription_id/items.
func (h *SubscriptionHandler) ListItems(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	items, err := h.svc.ListItems(c.Request.Context(), orgID, subID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			_ = c.Error(apierr.NotFound("subscription not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]subscriptionItemResponse, len(items))
	for i, item := range items {
		resp[i] = subscriptionItemResponse{
			ID:        item.ID,
			PlanID:    item.PlanID,
			Quantity:  item.Quantity,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// ListHistory handles GET /v1/subscriptions/:subscription_id/history.
func (h *SubscriptionHandler) ListHistory(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	history, err := h.svc.ListHistory(c.Request.Context(), orgID, subID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			_ = c.Error(apierr.NotFound("subscription not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]subscriptionHistoryResponse, len(history))
	for i, h := range history {
		r := subscriptionHistoryResponse{
			ID:        h.ID,
			NewStatus: string(h.NewStatus),
			Reason:    h.Reason,
			CreatedAt: h.CreatedAt,
		}
		if h.OldStatus != nil {
			s := string(*h.OldStatus)
			r.OldStatus = &s
		}
		resp[i] = r
	}
	c.JSON(http.StatusOK, resp)
}

// CreateSchedule handles POST /v1/subscriptions/:subscription_id/schedules.
func (h *SubscriptionHandler) CreateSchedule(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	var req createScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}
	if req.TargetStatus == nil && req.TargetPlanID == nil {
		_ = c.Error(apierr.BadRequest("one of target_status or target_plan_id is required"))
		return
	}
	if req.TargetStatus != nil && req.TargetPlanID != nil {
		_ = c.Error(apierr.BadRequest("only one of target_status or target_plan_id may be provided"))
		return
	}

	input := domain.CreateScheduleInput{
		ScheduledAt:  req.ScheduledAt,
		TargetPlanID: req.TargetPlanID,
	}
	if req.TargetStatus != nil {
		s := domain.SubscriptionStatus(*req.TargetStatus)
		input.TargetStatus = &s
	}

	schedule, err := h.svc.CreateSchedule(c.Request.Context(), orgID, subID, input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		case errors.Is(err, domain.ErrSubscriptionCanceled):
			_ = c.Error(apierr.UnprocessableEntity("subscription is already canceled"))
		case errors.Is(err, domain.ErrScheduleAlreadyPending):
			_ = c.Error(apierr.Conflict("subscription already has a pending schedule"))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.JSON(http.StatusCreated, toScheduleResponse(schedule))
}

// CancelSchedule handles DELETE /v1/subscriptions/:subscription_id/schedules/pending.
func (h *SubscriptionHandler) CancelSchedule(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	if err := h.svc.CancelSchedule(c.Request.Context(), orgID, subID); err != nil {
		switch {
		case errors.Is(err, domain.ErrScheduleNotFound):
			_ = c.Error(apierr.NotFound("no pending schedule found"))
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.Status(http.StatusNoContent)
}
```

---

## 8. Router

### `internal/api/router/router.go` (modificar)

**Adicionar na seção de Repositories** (após `planRepo`):

```go
subscriptionRepo     := repository.NewSubscriptionRepo(adapters.DB.SubscriptionQ)
subscriptionItemRepo := repository.NewSubscriptionItemRepo(adapters.DB.SubscriptionQ)
subscriptionSchRepo  := repository.NewSubscriptionScheduleRepo(adapters.DB.SubscriptionQ)
subscriptionHistRepo := repository.NewSubscriptionHistoryRepo(adapters.DB.SubscriptionQ)
```

**Adicionar na seção de Services** (após `planSvc`):

```go
subscriptionSvc := service.NewSubscriptionService(
    subscriptionRepo,
    subscriptionItemRepo,
    subscriptionSchRepo,
    subscriptionHistRepo,
    planRepo,
    customerRepo,
    txManager,
)
```

**Adicionar na seção de Handlers** (após `planHandler`):

```go
subscriptionHandler := handler.NewSubscriptionHandler(subscriptionSvc)
```

**Adicionar no registro de rotas protegidas** (após `planHandler.RegisterRoutes(protected)`):

```go
subscriptionHandler.RegisterRoutes(protected)
```

---

## 9. Ajuste SQL de criação com status e período corretos

A query `CreateSubscription` conforme escrita não permite injetar `status` e `current_period_end` dinamicamente. A solução mais limpa é adicionar essas colunas como parâmetros. Alterar o SQL para:

```sql
-- name: CreateSubscription :one
INSERT INTO subscriptions (
    organization_id,
    customer_id,
    status,
    current_period_start,
    current_period_end,
    cancel_at_period_end,
    metadata
) VALUES ($1, $2, $3, $4, $5, false, $6)
RETURNING *;
```

E passar `status`, `current_period_start` e `current_period_end` calculados pelo service. O repository deve receber esses valores de fora:

```go
// CreateSubscriptionParams (gerado pelo SQLC após ajuste)
type CreateSubscriptionParams struct {
    OrganizationID     uuid.UUID
    CustomerID         uuid.UUID
    Status             SubscriptionStatusType
    CurrentPeriodStart time.Time
    CurrentPeriodEnd   time.Time
    Metadata           pqtype.NullRawMessage
}
```

O service calcula `status` (`trialing` ou `incomplete`) e `periodEnd` via `calculatePeriodEnd` antes de chamar o repo. O repositório não precisa mais chamar `UpdateStatus` logo após o `Create`.

---

## 10. Verificação

### Compilar

```bash
go build ./...
```

### Testar manualmente

```bash
# Criar subscription
curl -s -X POST http://localhost:8080/v1/subscriptions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "<customer_uuid>",
    "items": [{"plan_id": "<plan_uuid>", "quantity": 1}]
  }' | jq

# Cancelar subscription
curl -s -X POST http://localhost:8080/v1/subscriptions/<id>/cancel \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"reason": "customer request", "cancel_at_period_end": false}' | jq

# Listar com filtros
curl -s "http://localhost:8080/v1/subscriptions?status=active&status=past_due&customer_id=<uuid>" \
  -H "Authorization: Bearer <token>" | jq
```

---

## Observações

- **Trigger de histórico:** o trigger `tr_subscriptions_status_history` já grava cada UPDATE no status automaticamente. O service **não** precisa inserir no histórico manualmente.
- **FSM:** `SubscriptionStateMachine.Validate` usa `ErrSameStateTransition` (herdado de payments) e `ErrSubscriptionCanceled` para estados terminais. O erro `ErrInvalidSubscriptionTransition` usa `fmt.Errorf` com `%w` para incluir o caminho `from → to` na mensagem.
- **Status múltiplos no filtro:** o `List` aceita múltiplos `?status=` via `c.QueryArray`. O filtro é aplicado in-memory no repositório pois SQLC não suporta `ANY($1::subscription_status_type[])` diretamente com enums gerados. Performance é aceitável para volumes típicos; otimizar com `sqlc.slice()` em versão futura se necessário.
- **TxManager.WithTx:** as interfaces `SubscriptionRepo()` e `SubscriptionItemRepo()` são adicionadas ao `TxHandle`. Isso mantém o padrão existente do projeto sem quebrar o `var _ domain.TxHandle = (*txHandle)(nil)`.
- **`CreateSubscription` com status e período:** a query final deve receber `status`, `current_period_start` e `current_period_end` como parâmetros para evitar o duplo UPDATE. Ver seção 9 acima.
