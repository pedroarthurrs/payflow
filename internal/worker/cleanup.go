package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"payflow/internal/domain"
)

// CleanupWorker executa tarefas periódicas de limpeza de dados antigos.
//   - credential_logs e webhook_logs: limpos diariamente à meia-noite (retenção 90 dias).
//   - idempotency_keys expiradas: limpas diariamente às 03:00.
type CleanupWorker struct {
	repo         domain.CleanupRepository
	logRetention time.Duration // retenção máxima dos logs (padrão 90 dias)
	now          func() time.Time
}

// NewCleanupWorker cria um CleanupWorker com as dependências necessárias.
func NewCleanupWorker(repo domain.CleanupRepository) *CleanupWorker {
	return &CleanupWorker{
		repo:         repo,
		logRetention: 90 * 24 * time.Hour,
		now:          time.Now,
	}
}

// Start registra os jobs no scheduler cron e bloqueia até o contexto ser cancelado.
func (w *CleanupWorker) Start(ctx context.Context) {
	slog.Info("cleanup worker started",
		"log_retention_days", int(w.logRetention.Hours()/24),
	)

	c := cron.New()

	// Limpeza de logs: diariamente às 02:00.
	if _, err := c.AddFunc("0 2 * * *", func() { w.cleanLogs(ctx) }); err != nil {
		slog.Error("cleanup: failed to register logs cleanup job", "error", err)
	}

	// Limpeza de idempotency keys expiradas: diariamente às 03:00.
	if _, err := c.AddFunc("0 3 * * *", func() { w.cleanIdempotencyKeys(ctx) }); err != nil {
		slog.Error("cleanup: failed to register idempotency cleanup job", "error", err)
	}

	c.Start()
	<-ctx.Done()
	c.Stop()

	slog.Info("cleanup worker stopped")
}

// cleanLogs remove credential_logs e webhook_logs mais antigos que logRetention.
func (w *CleanupWorker) cleanLogs(ctx context.Context) {
	cutoff := w.now().Add(-w.logRetention)

	deleted, err := w.repo.DeleteCredentialLogsOlderThan(ctx, cutoff)
	if err != nil {
		slog.Error("cleanup: failed to delete old credential logs", "error", err)
	} else {
		slog.Info("cleanup: credential logs purged", "deleted", deleted, "cutoff", cutoff)
	}

	deleted, err = w.repo.DeleteWebhookLogsOlderThan(ctx, cutoff)
	if err != nil {
		slog.Error("cleanup: failed to delete old webhook logs", "error", err)
	} else {
		slog.Info("cleanup: webhook logs purged", "deleted", deleted, "cutoff", cutoff)
	}
}

// cleanIdempotencyKeys remove chaves de idempotência expiradas.
func (w *CleanupWorker) cleanIdempotencyKeys(ctx context.Context) {
	deleted, err := w.repo.DeleteExpiredIdempotencyKeys(ctx)
	if err != nil {
		slog.Error("cleanup: failed to delete expired idempotency keys", "error", err)
	} else {
		slog.Info("cleanup: expired idempotency keys purged", "deleted", deleted)
	}
}
