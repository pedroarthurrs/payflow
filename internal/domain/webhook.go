package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType constants for supported webhook events.
const (
	EventCustomerCreated      = "customer.created"
	EventCustomerUpdated      = "customer.updated"
	EventCustomerDeleted      = "customer.deleted"
	EventInvoiceCreated       = "invoice.created"
	EventInvoicePaid          = "invoice.paid"
	EventInvoicePaymentFailed = "invoice.payment_failed"
	EventSubscriptionCreated  = "subscription.created"
	EventSubscriptionUpdated  = "subscription.updated"
	EventSubscriptionCanceled = "subscription.canceled"
)

// ValidEventTypes returns all supported event types.
func ValidEventTypes() []string {
	return []string{
		EventCustomerCreated, EventCustomerUpdated, EventCustomerDeleted,
		EventInvoiceCreated, EventInvoicePaid, EventInvoicePaymentFailed,
		EventSubscriptionCreated, EventSubscriptionUpdated, EventSubscriptionCanceled,
	}
}

// IsValidEventType checks whether the given event type is supported.
func IsValidEventType(et string) bool {
	for _, v := range ValidEventTypes() {
		if et == v {
			return true
		}
	}
	return false
}

// WebhookEndpoint represents an HTTP endpoint configured by an organization to receive event notifications.
type WebhookEndpoint struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	URL            string
	Description    *string
	EnabledEvents  []string
	Environment    string // "test" | "production"
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateWebhookEndpointInput carries the fields needed to create a new webhook endpoint.
type CreateWebhookEndpointInput struct {
	URL           string
	Description   *string
	EnabledEvents []string
	Environment   string
}

// UpdateWebhookEndpointInput carries the fields that can be updated on a webhook endpoint.
type UpdateWebhookEndpointInput struct {
	URL           *string
	Description   *string
	EnabledEvents []string
	IsActive      *bool
}

// WebhookLog represents an immutable record of a webhook delivery attempt.
type WebhookLog struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	EndpointID     uuid.UUID
	EventType      string
	Payload        json.RawMessage
	ResponseBody   *string
	ResponseStatus *int32
	DurationMs     *int32
	AttemptNumber  int32
	Success        bool
	CreatedAt      time.Time
}

// CreateWebhookLogInput carries all fields needed to persist a new webhook log entry.
type CreateWebhookLogInput struct {
	OrganizationID uuid.UUID
	EndpointID     uuid.UUID
	EventType      string
	Payload        json.RawMessage
	ResponseBody   *string
	ResponseStatus *int32
	DurationMs     *int32
	AttemptNumber  int32
	Success        bool
}

// ListWebhookLogsInput carries all filter and pagination parameters for listing webhook logs.
type ListWebhookLogsInput struct {
	Limit      int
	Cursor     *CursorToken
	From       *time.Time
	To         *time.Time
	EndpointID *uuid.UUID
	EventType  *string
	Success    *bool
}

// WebhookJob holds all data required for a single webhook delivery attempt.
// It is serialized as JSON by the WebhookDispatcher and stored in Job.Payload.
type WebhookJob struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	EndpointID     uuid.UUID `json:"endpoint_id"`
	EndpointURL    string    `json:"endpoint_url"`
	Secret         string    `json:"secret"`
	EventType      string    `json:"event_type"`
	Environment    string    `json:"environment"`
	Payload        []byte    `json:"payload"`
}

// WebhookEndpointRepository defines persistence operations for webhook endpoints.
type WebhookEndpointRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input CreateWebhookEndpointInput) (*WebhookEndpoint, error)
	GetByID(ctx context.Context, orgID, endpointID uuid.UUID) (*WebhookEndpoint, error)
	List(ctx context.Context, orgID uuid.UUID, params ListParams) ([]WebhookEndpoint, error)
	Update(ctx context.Context, orgID, endpointID uuid.UUID, input UpdateWebhookEndpointInput) (*WebhookEndpoint, error)
	SoftDelete(ctx context.Context, orgID, endpointID uuid.UUID) error
	ListActiveByEvent(ctx context.Context, orgID uuid.UUID, eventType string, env string) ([]WebhookEndpoint, error)
}

// WebhookLogRepository defines persistence operations for webhook logs.
type WebhookLogRepository interface {
	Create(ctx context.Context, input CreateWebhookLogInput) error
	GetByID(ctx context.Context, orgID, logID uuid.UUID) (*WebhookLog, error)
	List(ctx context.Context, orgID uuid.UUID, input ListWebhookLogsInput) ([]WebhookLog, error)
}

// OrganizationSecretReader defines read access to an organization's webhook secret.
type OrganizationSecretReader interface {
	GetWebhookSecret(ctx context.Context, orgID uuid.UUID) (string, error)
}
