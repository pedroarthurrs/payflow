# Plan: Motor de Recorrência (Recurrence Engine)

## Decisões Tomadas
- PaymentMethod pertence à Subscription (não ao Customer)
- TxHandle expande com SubscriptionScheduleRepo() — OK
- Fila dedicada para recorrência (separada da fila de webhooks)
- Alterações de schema nas migrations existentes (projeto não está em produção)

---

## Fase 1 — Schema (editar migrations existentes)

**`deploy/migrations/000001_create_base_tables.up.sql`**:
- Na tabela `subscriptions`: adicionar `payment_method VARCHAR(50) NOT NULL` e `billing_locked_until TIMESTAMPTZ NULL`
- Na tabela `payments`: adicionar `subscription_id UUID NULL` + FK `FOREIGN KEY (organization_id, subscription_id) REFERENCES subscriptions(organization_id, id) ON DELETE SET NULL`

**`deploy/migrations/000002_create_indexes.up.sql`**:
- Adicionar índice parcial: `CREATE INDEX idx_subscriptions_billing ON subscriptions(current_period_end) WHERE status IN ('active', 'past_due', 'trialing') AND billing_locked_until IS NULL`

---

## Fase 2 — Domínio (parallel com Fase 1)

**`internal/domain/job.go`**: adicionar `JobTypeRecurringBilling JobType = "recurring_billing"`

**`internal/domain/recurrence.go`** (NOVO):
- Struct `RecurringBillingJob { SubscriptionID, OrganizationID uuid.UUID }`
- Struct `SubscriptionBillingCandidate { ID, OrganizationID, CustomerID uuid.UUID; Status SubscriptionStatus; CancelAtPeriodEnd bool; CurrentPeriodEnd time.Time; PaymentMethod string }`
- Interface `RecurrenceRepository`:
  - `ListDueForBilling(ctx, now time.Time, batchSize int) ([]SubscriptionBillingCandidate, error)` — cross-org, usa FOR UPDATE SKIP LOCKED
  - `LockForBilling(ctx, orgID, subID uuid.UUID, until time.Time) error`
  - `UnlockBilling(ctx, orgID, subID uuid.UUID) error`

**`internal/domain/subscription.go`** (MODIFICAR):
- Adicionar campos `PaymentMethod string` e `BillingLockedUntil *time.Time` à struct `Subscription`
- Adicionar `PaymentMethod string` ao `CreateSubscriptionInput`
- Adicionar método `AdvancePeriod(ctx, orgID, subID uuid.UUID, newStart, newEnd time.Time) (*Subscription, error)` à interface `SubscriptionRepository`

**`internal/domain/payment.go`** (MODIFICAR):
- Adicionar `SubscriptionID *uuid.UUID` à struct `Payment` e a `PaymentInput`

**`internal/domain/tx.go`** (MODIFICAR):
- Adicionar `SubscriptionScheduleRepo() SubscriptionScheduleRepository` ao `TxHandle`

**`internal/domain/errors.go`** (MODIFICAR):
- Adicionar `ErrSubscriptionAlreadyLocked`, `ErrBillingAmountZero`

---

## Fase 3 — Database (depends on Fases 1+2)

**`internal/database/queries/recurrence.sql`** (NOVO):
```sql
-- name: ListSubscriptionsDueForBilling :many
SELECT id, organization_id, customer_id, status, cancel_at_period_end, current_period_end, payment_method
FROM subscriptions
WHERE current_period_end <= $1
  AND status IN ('active', 'past_due', 'trialing')
  AND (billing_locked_until IS NULL OR billing_locked_until < NOW())
ORDER BY current_period_end ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;

-- name: LockSubscriptionForBilling :exec
UPDATE subscriptions SET billing_locked_until = $3 WHERE organization_id = $1 AND id = $2;

-- name: UnlockSubscriptionBilling :exec
UPDATE subscriptions SET billing_locked_until = NULL WHERE organization_id = $1 AND id = $2;

-- name: AdvanceSubscriptionPeriod :one
UPDATE subscriptions
SET current_period_start = $3, current_period_end = $4, billing_locked_until = NULL, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;
```

**`sqlc.yml`**: adicionar seção para `recurrence.sql` → package `recurrencedb`; rodar `make sqlc` (ou equivalente)

**`internal/database/repository/recurrence.go`** (NOVO): `RecurrenceRepo` implementando `domain.RecurrenceRepository`

**`internal/database/repository/subscription.go`** (MODIFICAR): implementar `AdvancePeriod` chamando query `AdvanceSubscriptionPeriod`

**`internal/database/repository/payment.go`** (MODIFICAR): incluir `SubscriptionID` no insert

**`internal/database/connection.go`** (MODIFICAR): adicionar `RecurrenceQ` ao struct `DB`

**Implementação de `TxHandle`**: adicionar `SubscriptionScheduleRepo()` em todos os repos que implementam essa interface (verificar em `internal/database/repository/`)

---

## Fase 4 — Service (depends on Fase 3)

**`internal/service/recurrence.go`** (NOVO):

`RecurrenceService` deps: `subs domain.SubscriptionRepository`, `items domain.SubscriptionItemRepository`, `schedules domain.SubscriptionScheduleRepository`, `plans domain.PlanRepository`, `recRepo domain.RecurrenceRepository`, `payFSMRepo domain.PaymentFSMRepository`, `gateway domain.Gateway`, `txMgr domain.TxManager`, `fsm *domain.StateMachine`, `subFSM *domain.SubscriptionStateMachine`

Método `ProcessBillingCycle(ctx, orgID, subID uuid.UUID) error`:
1. `subs.GetByID()` para carregar dados completos incluindo `PaymentMethod`
2. Checar `subFSM.IsTerminal(status)` → skip se terminal
3. Se `cancel_at_period_end` && `period_end <= now` → `subs.UpdateStatus(canceled)` + `recRepo.UnlockBilling()` → retornar
4. Se `trialing` && `period_end <= now` → `subs.UpdateStatus(active)`; recarregar status
5. `items.ListBySubscription()` → para cada item: `plans.GetByID()` → acumular `totalAmountCents = sum(qty * amountCents)`, capturar `currency` e `interval/intervalCount` do primeiro plano
6. Verificar `totalAmountCents > 0` → senão `ErrBillingAmountZero`
7. Construir `idempotencyKey = fmt.Sprintf("sub_%s_%s", subID, periodStart.Format("2006-01-02"))`
8. `payments.Create(ctx, orgID, PaymentInput{CustomerID, PaymentMethod: sub.PaymentMethod, AmountCents: total, Currency, SubscriptionID: &subID})`
9. `gateway.Charge(ctx, GatewayRequest{PaymentID, AmountCents, Currency, PaymentMethod: sub.PaymentMethod, CustomerID})`
10. **Sucesso**:
    - `txMgr.WithTx()`: `fsmRepo.GetForUpdate` → `fsmRepo.UpdateStatus(succeeded)` → `fsmRepo.InsertHistory(source="system")` → `payments.UpdateGatewayReference`
    - Se `status == past_due`: `subs.UpdateStatus(active)`
    - `subs.AdvancePeriod(orgID, subID, oldEnd, calculatePeriodEnd(oldEnd, interval, count))` — limpa lock automaticamente
    - Verificar pending schedule via `schedules.GetPending()` e aplicar se existir
11. **Falha**:
    - `payments.UpdateErrorLog(ctx, orgID, paymentID, errorJSON)`
    - `txMgr.WithTx()`: FSM payment `→ failed`
    - Se `status == active`: `subs.UpdateStatus(past_due)`
    - Se `status == past_due` e tentativa >= maxAttempts: `subs.UpdateStatus(unpaid)`
    - `recRepo.UnlockBilling()` para permitir futura retentativa via re-enqueue

---

## Fase 5 — Workers (depends on Fase 4, parallel entre si)

**`internal/worker/recurrence_worker.go`** (NOVO):
```
RecurrenceWorker:
  - deps: queue domain.Queue, svc *service.RecurrenceService, workerCount int, maxAttempts int
  - Start(ctx): spawn N goroutines → consume loop
  - consume(ctx, wg): loop Dequeue → process
  - process(ctx, job):
      - validar job.Type == JobTypeRecurringBilling
      - unmarshal RecurringBillingJob
      - respeitar ScheduledAt (time.After + ctx.Done)
      - svc.ProcessBillingCycle(ctx, orgID, subID)
      - erro: re-enqueue com backoff [2m, 10m, 30m, 2h] até maxAttempts; depois dead-letter log
```

**`internal/worker/recurrence_scheduler.go`** (NOVO):
```
RecurrenceScheduler:
  - deps: recRepo domain.RecurrenceRepository, queue domain.Queue, batchSize int, cronExpr string
  - Start(ctx): cron.New() → AddFunc(cronExpr, scan) → cron.Start() → <-ctx.Done() → cron.Stop()
  - scan(ctx):
      1. ListDueForBilling(ctx, time.Now(), batchSize)
      2. para cada candidato:
         a. LockForBilling(ctx, orgID, subID, time.Now().Add(1h))
         b. Enqueue(ctx, Job{Type: JobTypeRecurringBilling, Payload: marshal(RecurringBillingJob)})
      3. slog.Info("recurrence scan", "enqueued", count)
```

---

## Fase 6 — Config + Wiring (depends on Fase 5)

**`internal/config/config.go`** (MODIFICAR): adicionar campos:
- `RecurrenceWorkerCount int`
- `RecurrenceQueueBufferSize int`
- `RecurrenceScanCron string`
- `RecurrenceMaxAttempts int`

**`internal/config/flags_env.go`** (MODIFICAR): mapear envvars:
- `RECURRENCE_WORKER_COUNT` (default 5)
- `RECURRENCE_QUEUE_BUFFER_SIZE` (default 500)
- `RECURRENCE_SCAN_CRON` (default `"*/5 * * * *"`)
- `RECURRENCE_MAX_ATTEMPTS` (default 4)

**`cmd/server/main.go`** (MODIFICAR):
- Em `setupInfrastructure()`: criar `recurrenceQueue = queue.New("memory", cfg.RecurrenceQueueBufferSize)`; retornar junto com webhookQueue e gateway
- Em `startWorkers()`: instanciar `RecurrenceRepo`, `RecurrenceService`, `RecurrenceWorker`, `RecurrenceScheduler`; `go recurrenceWorker.Start(ctx)` e `go recurrenceScheduler.Start(ctx)`

---

## Fluxo Completo (Happy Path)

```
[robfig/cron a cada 5min]
  RecurrenceScheduler.scan()
  → SELECT FOR UPDATE SKIP LOCKED WHERE period_end <= NOW() AND lock IS NULL
  → UPDATE billing_locked_until = NOW() + 1h  (previne double-enqueue)
  → queue.Enqueue(RecurringBillingJob{OrgID, SubID})

[goroutina RecurrenceWorker]
  → queue.Dequeue()
  → RecurrenceService.ProcessBillingCycle(orgID, subID)
       → load subscription (inclui PaymentMethod)
       → sum(items × plan.AmountCents × quantity)
       → payments.Create(pending, subscriptionID, idempotencyKey)
       → gateway.Charge(PaymentMethod da subscription)
       → [sucesso] FSM pending→succeeded + AdvancePeriod (limpa lock)
       → [falha]   FSM pending→failed + sub active→past_due + re-queue com backoff
```

---

## Arquivos a Modificar/Criar

- `deploy/migrations/000001_create_base_tables.up.sql` — schema subscriptions + payments
- `deploy/migrations/000002_create_indexes.up.sql` — índice parcial billing
- `internal/domain/job.go` — JobTypeRecurringBilling
- `internal/domain/recurrence.go` — NOVO
- `internal/domain/subscription.go` — PaymentMethod, BillingLockedUntil, AdvancePeriod
- `internal/domain/payment.go` — SubscriptionID
- `internal/domain/tx.go` — SubscriptionScheduleRepo()
- `internal/domain/errors.go` — novos sentinelas
- `internal/database/queries/recurrence.sql` — NOVO
- `sqlc.yml` — nova seção recurrencedb
- `internal/database/repository/recurrence.go` — NOVO
- `internal/database/repository/subscription.go` — AdvancePeriod
- `internal/database/repository/payment.go` — SubscriptionID
- `internal/database/connection.go` — RecurrenceQ
- `internal/database/repository/` — TxHandle: SubscriptionScheduleRepo()
- `internal/service/recurrence.go` — NOVO (ProcessBillingCycle)
- `internal/worker/recurrence_worker.go` — NOVO
- `internal/worker/recurrence_scheduler.go` — NOVO
- `internal/config/config.go` — novas vars
- `internal/config/flags_env.go` — envvars
- `cmd/server/main.go` — wiring fila dedicada + workers

---

## Verificação

1. `internal/service/recurrence_test.go`: `ProcessBillingCycle` — success, failure, trialing→active, cancel_at_period_end, idempotência (mesmo job duas vezes não duplica charge)
2. `internal/worker/recurrence_scheduler_test.go`: `scan()` enfileira os candidatos corretos; job com lock ativo é ignorado
3. Teste de propriedade: lock + idempotencyKey garantem exatamente 1 charge por subscription por período
4. Smoke test manual: subscription com `current_period_end` no passado → job enfileirado → payment criado + status succeeded → `current_period_end` avançado → `billing_locked_until = NULL`
