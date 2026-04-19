package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/google/uuid"
)

var recurrenceBackoffSchedule = []time.Duration{
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

// RecurrenceWorker consumes recurring billing jobs from a dedicated queue.
type RecurrenceWorker struct {
	queue       domain.Queue
	svc         *service.RecurrenceService
	workerCount int
	maxAttempts int
}

// NewRecurrenceWorker creates a RecurrenceWorker.
func NewRecurrenceWorker(q domain.Queue, svc *service.RecurrenceService, workerCount, maxAttempts int) *RecurrenceWorker {
	return &RecurrenceWorker{
		queue:       q,
		svc:         svc,
		workerCount: workerCount,
		maxAttempts: maxAttempts,
	}
}

// Start launches workerCount consumer goroutines and blocks until the context is canceled.
func (w *RecurrenceWorker) Start(ctx context.Context) {
	slog.Info("recurrence worker started", "worker_count", w.workerCount)
	var wg sync.WaitGroup
	for i := 0; i < w.workerCount; i++ {
		wg.Add(1)
		go w.consume(ctx, &wg)
	}
	wg.Wait()
	slog.Info("recurrence worker stopped")
}

func (w *RecurrenceWorker) consume(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		job, err := w.queue.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("failed to dequeue recurrence job", "error", err)
			continue
		}
		w.process(ctx, job)
	}
}

func (w *RecurrenceWorker) process(ctx context.Context, job *domain.Job) {
	if job.Type != domain.JobTypeRecurringBilling {
		slog.Warn("unexpected job type in recurrence queue, discarding", "type", job.Type, "job_id", job.ID)
		return
	}

	var billingJob domain.RecurringBillingJob
	if err := json.Unmarshal(job.Payload, &billingJob); err != nil {
		slog.Error("failed to unmarshal recurring billing job", "job_id", job.ID, "error", err)
		return
	}

	if delay := time.Until(job.ScheduledAt); delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
	}

	attemptNum := int(job.AttemptNumber)
	err := w.svc.ProcessBillingCycle(ctx, billingJob.OrganizationID, billingJob.SubscriptionID, attemptNum)
	if err == nil {
		slog.Info("recurring billing cycle completed",
			"subscription_id", billingJob.SubscriptionID,
			"organization_id", billingJob.OrganizationID,
			"attempt", attemptNum)
		return
	}

	if attemptNum >= w.maxAttempts {
		slog.Error("recurring billing dead letter — max attempts reached",
			"subscription_id", billingJob.SubscriptionID,
			"organization_id", billingJob.OrganizationID,
			"attempts", attemptNum,
			"error", err)
		// Release the lock so the scheduler can re-evaluate this subscription later.
		if relErr := w.svc.ReleaseBillingLock(ctx, billingJob.OrganizationID, billingJob.SubscriptionID); relErr != nil {
			slog.Error("failed to release billing lock after dead-letter",
				"subscription_id", billingJob.SubscriptionID, "error", relErr)
		}
		return
	}

	backoffIdx := attemptNum - 1
	if backoffIdx >= len(recurrenceBackoffSchedule) {
		backoffIdx = len(recurrenceBackoffSchedule) - 1
	}
	nextRetryAt := time.Now().Add(recurrenceBackoffSchedule[backoffIdx])

	slog.Warn("recurring billing failed, re-enqueuing",
		"subscription_id", billingJob.SubscriptionID,
		"attempt", attemptNum,
		"next_retry_at", nextRetryAt,
		"error", err)

	nextJob := domain.Job{
		ID:            uuid.New(),
		Type:          domain.JobTypeRecurringBilling,
		Payload:       job.Payload,
		AttemptNumber: int32(attemptNum + 1),
		ScheduledAt:   nextRetryAt,
	}
	if enqErr := w.queue.Enqueue(ctx, nextJob); enqErr != nil {
		slog.Error("failed to re-enqueue recurring billing job",
			"subscription_id", billingJob.SubscriptionID, "error", enqErr)
	}

	// Extend the billing lock to cover the retry window so the scheduler does
	// not re-enqueue a fresh attempt=1 before the worker retry fires.
	lockUntil := nextRetryAt.Add(5 * time.Minute)
	if lockErr := w.svc.ExtendBillingLock(ctx, billingJob.OrganizationID, billingJob.SubscriptionID, lockUntil); lockErr != nil {
		slog.Error("failed to extend billing lock for retry",
			"subscription_id", billingJob.SubscriptionID, "error", lockErr)
	}
}
