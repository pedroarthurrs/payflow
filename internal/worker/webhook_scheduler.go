package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// WebhookScheduler enqueues webhook delivery jobs when domain events occur.
// It resolves the active endpoints for an event and places a WebhookJob into
// the queue for each one, leaving delivery to WebhookWorker.
type WebhookScheduler struct {
	endpointRepo domain.WebhookEndpointRepository
	secretReader domain.OrganizationSecretReader
	queue        domain.Queue
}

// NewWebhookScheduler creates a WebhookScheduler.
func NewWebhookScheduler(
	endpointRepo domain.WebhookEndpointRepository,
	secretReader domain.OrganizationSecretReader,
	q domain.Queue,
) *WebhookScheduler {
	return &WebhookScheduler{
		endpointRepo: endpointRepo,
		secretReader: secretReader,
		queue:        q,
	}
}

// Dispatch finds all active endpoints matching the event and environment, then enqueues
// a WebhookJob for each one. Enqueue failures are logged and do not stop processing.
func (s *WebhookScheduler) Dispatch(ctx context.Context, orgID uuid.UUID, eventType, environment string, payload []byte) error {
	secret, err := s.secretReader.GetWebhookSecret(ctx, orgID)
	if err != nil {
		slog.Error("failed to get webhook secret", "organization_id", orgID, "error", err)
		return err
	}

	endpoints, err := s.endpointRepo.ListActiveByEvent(ctx, orgID, eventType, environment)
	if err != nil {
		slog.Error("failed to list active endpoints", "organization_id", orgID, "event_type", eventType, "error", err)
		return err
	}

	for i := range endpoints {
		webhookJob := domain.WebhookJob{
			OrganizationID: endpoints[i].OrganizationID,
			EndpointID:     endpoints[i].ID,
			EndpointURL:    endpoints[i].URL,
			Secret:         secret,
			EventType:      eventType,
			Environment:    environment,
			Payload:        payload,
		}
		bytes, err := json.Marshal(webhookJob)
		if err != nil {
			slog.Error("failed to serialize webhook job", "endpoint_id", endpoints[i].ID, "error", err)
			continue
		}
		job := domain.Job{
			ID:            uuid.New(),
			Type:          domain.JobTypeWebhook,
			Payload:       bytes,
			ScheduledAt:   time.Now(),
			AttemptNumber: 1,
		}
		if err := s.queue.Enqueue(ctx, job); err != nil {
			slog.Error("failed to enqueue webhook job", "endpoint_id", endpoints[i].ID, "error", err)
			continue
		}
	}

	return nil
}
