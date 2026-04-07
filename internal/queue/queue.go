package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// WebhookJob contém todos os dados necessários para uma tentativa de entrega de webhook.
// O job é auto-suficiente: carrega Secret e EndpointURL para que o worker não precise
// acessar o banco durante a entrega.
type WebhookJob struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	EndpointID     uuid.UUID `json:"endpoint_id"`
	EndpointURL    string    `json:"endpoint_url"`
	Secret         string    `json:"secret"`
	EventType      string    `json:"event_type"`
	Environment    string    `json:"environment"`
	Payload        []byte    `json:"payload"`
	AttemptNumber  int32     `json:"attempt_number"`
	ScheduledAt    time.Time `json:"scheduled_at"`
}

// Queue abstrai um sistema de filas de mensagens para entrega de webhooks.
// Implementações concretas podem usar InMemoryQueue, SQSQueue, Redis, etc.
type Queue interface {
	// Enqueue adiciona um job à fila. Retorna erro se a fila estiver cheia ou indisponível.
	Enqueue(ctx context.Context, job WebhookJob) error
	// Dequeue retira o próximo job da fila. Bloqueia até que um job esteja disponível
	// ou o contexto seja cancelado. Retorna nil, ctx.Err() quando o contexto é cancelado.
	Dequeue(ctx context.Context) (*WebhookJob, error)
}
