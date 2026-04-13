package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"i9pay/internal/crypto"
	"i9pay/internal/domain"
)

var backoffSchedule = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

// WebhookWorker consumes jobs from the Queue and performs HTTP delivery with exponential backoff retry.
type WebhookWorker struct {
	queue       domain.Queue
	logRepo     domain.WebhookLogRepository
	httpClient  *http.Client
	workerCount int
	hmacGen     *crypto.HMACGenerator
}

// NewWebhookWorker creates a WebhookWorker. workerCount sets the number of concurrent consumer goroutines.
func NewWebhookWorker(q domain.Queue, logRepo domain.WebhookLogRepository, httpClient *http.Client, workerCount int) *WebhookWorker {
	return &WebhookWorker{
		queue:       q,
		logRepo:     logRepo,
		httpClient:  httpClient,
		workerCount: workerCount,
		hmacGen:     crypto.NewHMACGenerator(),
	}
}

// Start launches workerCount consumer goroutines and blocks until the context is canceled.
func (w *WebhookWorker) Start(ctx context.Context) {
	slog.Info("webhook worker started", "worker_count", w.workerCount)
	var wg sync.WaitGroup
	for i := 0; i < w.workerCount; i++ {
		wg.Add(1)
		go w.consume(ctx, &wg)
	}
	wg.Wait()
	slog.Info("webhook worker stopped")
}

// consume is the processing loop run by each worker goroutine.
func (w *WebhookWorker) consume(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		job, err := w.queue.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("failed to dequeue webhook job", "error", err)
			continue
		}
		w.process(ctx, job)
	}
}

// process handles a single job: validates the type, deserializes the payload,
// respects ScheduledAt, performs the HTTP POST, persists the WebhookLog, and re-enqueues on failure.
func (w *WebhookWorker) process(ctx context.Context, job *domain.Job) {
	if job.Type != domain.JobTypeWebhook {
		slog.Warn("unknown job type, discarding", "type", job.Type, "job_id", job.ID)
		return
	}

	var webhookJob domain.WebhookJob
	if err := json.Unmarshal(job.Payload, &webhookJob); err != nil {
		slog.Error("failed to unmarshal webhook job payload", "job_id", job.ID, "error", err)
		return
	}

	if delay := time.Until(job.ScheduledAt); delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
	}

	logInput := w.deliver(ctx, &webhookJob, job.AttemptNumber)

	if err := w.logRepo.Create(ctx, *logInput); err != nil {
		slog.Error("failed to persist webhook log", "endpoint_id", webhookJob.EndpointID, "error", err)
	}

	if logInput.Success {
		slog.Info("webhook delivered", "endpoint_id", webhookJob.EndpointID, "event_type", webhookJob.EventType,
			"attempt", job.AttemptNumber, "status_code", *logInput.ResponseStatus, "duration_ms", *logInput.DurationMs)
		return
	}

	if job.AttemptNumber >= 5 {
		slog.Warn("webhook dead letter", "endpoint_id", webhookJob.EndpointID,
			"event_type", webhookJob.EventType, "total_attempts", job.AttemptNumber)
		return
	}

	nextJob := domain.Job{
		ID:            job.ID,
		Type:          job.Type,
		Payload:       job.Payload,
		AttemptNumber: job.AttemptNumber + 1,
		ScheduledAt:   time.Now().Add(backoffSchedule[job.AttemptNumber-1]),
	}

	slog.Warn("webhook failed, re-enqueuing", "endpoint_id", webhookJob.EndpointID,
		"event_type", webhookJob.EventType, "attempt", job.AttemptNumber,
		"next_retry_at", nextJob.ScheduledAt)

	if err := w.queue.Enqueue(ctx, nextJob); err != nil {
		slog.Error("failed to re-enqueue webhook job", "endpoint_id", webhookJob.EndpointID, "error", err)
	}
}

// deliver performs the HTTP POST and returns a populated CreateWebhookLogInput.
func (w *WebhookWorker) deliver(ctx context.Context, job *domain.WebhookJob, attemptNumber int32) *domain.CreateWebhookLogInput {
	logInput := &domain.CreateWebhookLogInput{
		OrganizationID: job.OrganizationID,
		EndpointID:     job.EndpointID,
		EventType:      job.EventType,
		Payload:        job.Payload,
		AttemptNumber:  attemptNumber,
		Success:        false,
	}

	signature := w.sign(job.Secret, job.Payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, job.EndpointURL, bytes.NewReader(job.Payload))
	if err != nil {
		slog.Error("failed to create webhook request", "endpoint_id", job.EndpointID, "error", err)
		return logInput
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", job.EventType)

	start := time.Now()
	resp, err := w.httpClient.Do(req)
	durationMs := int32(time.Since(start).Milliseconds())
	logInput.DurationMs = &durationMs

	if err != nil {
		slog.Error("webhook delivery failed", "endpoint_id", job.EndpointID, "attempt", attemptNumber, "error", err)
		return logInput
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close webhook response body", "endpoint_id", job.EndpointID, "error", err)
		}
	}()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	respBody := string(body)
	respStatus := int32(resp.StatusCode)

	logInput.ResponseBody = &respBody
	logInput.ResponseStatus = &respStatus
	logInput.Success = resp.StatusCode >= 200 && resp.StatusCode <= 299

	return logInput
}

// sign computes the HMAC-SHA256 signature of the payload using the job's secret.
func (w *WebhookWorker) sign(secret string, payload []byte) string {
	return w.hmacGen.HMACSHA256(secret, payload)
}
