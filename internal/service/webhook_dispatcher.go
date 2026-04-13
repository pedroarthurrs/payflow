package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// WebhookDispatcher handles asynchronous webhook delivery by enqueuing jobs to a Queue.
type WebhookDispatcher struct {
	endpointRepo domain.WebhookEndpointRepository
	secretReader domain.OrganizationSecretReader
	queue        domain.Queue
}

// NewWebhookDispatcher creates a WebhookDispatcher.
func NewWebhookDispatcher(
	endpointRepo domain.WebhookEndpointRepository,
	secretReader domain.OrganizationSecretReader,
	q domain.Queue,
) *WebhookDispatcher {
	return &WebhookDispatcher{
		endpointRepo: endpointRepo,
		secretReader: secretReader,
		queue:        q,
	}
}

// Dispatch finds all active endpoints matching the event and environment, then enqueues
// a WebhookJob for each one. Enqueue failures are logged and do not stop processing.
func (d *WebhookDispatcher) Dispatch(ctx context.Context, orgID uuid.UUID, eventType, environment string, payload []byte) error {
	secret, err := d.secretReader.GetWebhookSecret(ctx, orgID)
	if err != nil {
		slog.Error("failed to get webhook secret", "organization_id", orgID, "error", err)
		return err
	}

	endpoints, err := d.endpointRepo.ListActiveByEvent(ctx, orgID, eventType, environment)
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
		if err := d.queue.Enqueue(ctx, job); err != nil {
			slog.Error("failed to enqueue webhook job", "endpoint_id", endpoints[i].ID, "error", err)
			continue
		}
	}

	return nil
}
