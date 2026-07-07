package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// RecurrenceScheduler periodically scans for subscriptions due for billing
// and enqueues RecurringBillingJob items into the recurrence queue.
type RecurrenceScheduler struct {
	recRepo   domain.RecurrenceRepository
	queue     domain.Queue
	batchSize int
	cronExpr  string
}

// NewRecurrenceScheduler creates a RecurrenceScheduler.
func NewRecurrenceScheduler(recRepo domain.RecurrenceRepository, queue domain.Queue, batchSize int, cronExpr string) *RecurrenceScheduler {
	return &RecurrenceScheduler{
		recRepo:   recRepo,
		queue:     queue,
		batchSize: batchSize,
		cronExpr:  cronExpr,
	}
}

// Start registers the scan function with a cron scheduler and blocks until ctx is canceled.
func (s *RecurrenceScheduler) Start(ctx context.Context) {
	c := cron.New()

	_, err := c.AddFunc(s.cronExpr, func() {
		s.scan(ctx)
	})
	if err != nil {
		slog.Error("failed to register recurrence cron job", "cron_expr", s.cronExpr, "error", err)
		return
	}

	c.Start()
	slog.Info("recurrence scheduler started", "cron_expr", s.cronExpr, "batch_size", s.batchSize)

	<-ctx.Done()
	c.Stop()
	slog.Info("recurrence scheduler stopped")
}

// scan fetches subscriptions due for billing and enqueues a job for each one.
func (s *RecurrenceScheduler) scan(ctx context.Context) {
	now := time.Now().UTC()

	candidates, err := s.recRepo.ListDueForBilling(ctx, now, s.batchSize)
	if err != nil {
		slog.Error("recurrence scan failed", "error", err)
		return
	}

	enqueued := 0
	for _, candidate := range candidates {
		payload, err := json.Marshal(domain.RecurringBillingJob{
			SubscriptionID: candidate.ID,
			OrganizationID: candidate.OrganizationID,
		})
		if err != nil {
			slog.Error("failed to marshal recurring billing job",
				"subscription_id", candidate.ID, "error", err)
			continue
		}

		job := domain.Job{
			ID:            uuid.New(),
			Type:          domain.JobTypeRecurringBilling,
			Payload:       payload,
			ScheduledAt:   time.Now(),
			AttemptNumber: 1,
		}

		if err := s.queue.Enqueue(ctx, job); err != nil {
			slog.Error("failed to enqueue recurring billing job",
				"subscription_id", candidate.ID, "error", err)
			continue
		}
		enqueued++
	}

	slog.Info("recurrence scan completed",
		"candidates", len(candidates),
		"enqueued", enqueued)
}
