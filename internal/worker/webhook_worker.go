package worker

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"i9pay/internal/crypto"
	"i9pay/internal/domain"
	"i9pay/internal/queue"
)

var backoffSchedule = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

// WebhookWorker consome jobs da Queue e executa a entrega HTTP com retry via backoff exponential.
type WebhookWorker struct {
	queue       queue.Queue
	logRepo     domain.WebhookLogRepository
	httpClient  *http.Client
	workerCount int
	hmacGen     *crypto.HMACGenerator
}

// NewWebhookWorker cria um WebhookWorker.
// workerCount define o número de goroutines de consumo paralelas.
func NewWebhookWorker(q queue.Queue, logRepo domain.WebhookLogRepository, httpClient *http.Client, workerCount int) *WebhookWorker {
	return &WebhookWorker{
		queue:       q,
		logRepo:     logRepo,
		httpClient:  httpClient,
		workerCount: workerCount,
		hmacGen:     crypto.NewHMACGenerator(),
	}
}

// Start inicia workerCount goroutines de consumo e bloqueia até que o contexto seja cancelado.
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

// consume é o loop de consumo executado por cada goroutine worker.
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

// process executa a entrega de um único job: respeita ScheduledAt, faz HTTP POST,
// persiste WebhookLog e reenfileira em caso de falha.
func (w *WebhookWorker) process(ctx context.Context, job *queue.WebhookJob) {
	if delay := time.Until(job.ScheduledAt); delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
	}

	logInput := w.deliver(ctx, job)

	if err := w.logRepo.Create(ctx, *logInput); err != nil {
		slog.Error("failed to persist webhook log", "endpoint_id", job.EndpointID, "error", err)
	}

	if logInput.Success {
		slog.Info("webhook delivered", "endpoint_id", job.EndpointID, "event_type", job.EventType,
			"attempt", job.AttemptNumber, "status_code", *logInput.ResponseStatus, "duration_ms", *logInput.DurationMs)
		return
	}

	if job.AttemptNumber >= 5 {
		slog.Warn("webhook dead letter", "endpoint_id", job.EndpointID,
			"event_type", job.EventType, "total_attempts", job.AttemptNumber)
		return
	}

	nextJob := *job
	nextJob.AttemptNumber = job.AttemptNumber + 1
	nextJob.ScheduledAt = time.Now().Add(backoffSchedule[job.AttemptNumber-1])

	slog.Warn("webhook failed, re-enqueuing", "endpoint_id", job.EndpointID,
		"event_type", job.EventType, "attempt", job.AttemptNumber,
		"next_retry_at", nextJob.ScheduledAt)

	if err := w.queue.Enqueue(ctx, queue.WebhookJob(nextJob)); err != nil {
		slog.Error("failed to re-enqueue webhook job", "endpoint_id", job.EndpointID, "error", err)
	}
}

// deliver executa o HTTP POST e retorna o CreateWebhookLogInput preenchido.
func (w *WebhookWorker) deliver(ctx context.Context, job *queue.WebhookJob) *domain.CreateWebhookLogInput {
	logInput := &domain.CreateWebhookLogInput{
		OrganizationID: job.OrganizationID,
		EndpointID:     job.EndpointID,
		EventType:      job.EventType,
		Payload:        job.Payload,
		AttemptNumber:  job.AttemptNumber,
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
		slog.Error("webhook delivery failed", "endpoint_id", job.EndpointID, "attempt", job.AttemptNumber, "error", err)
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

// sign computa HMAC-SHA256 do payload com o secret do job.
func (w *WebhookWorker) sign(secret string, payload []byte) string {
	return w.hmacGen.HMACSHA256(secret, payload)
}
