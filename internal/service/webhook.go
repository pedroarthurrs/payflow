package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// cursorPayload is the JSON structure encoded inside a cursor token.
type cursorPayload struct {
	T string `json:"t"` // RFC3339Nano timestamp
	I string `json:"i"` // UUID string
}

// EncodeCursor encodes a (created_at, id) pair into an opaque base64url cursor string.
func EncodeCursor(t time.Time, id uuid.UUID) string {
	p := cursorPayload{T: t.UTC().Format(time.RFC3339Nano), I: id.String()}
	b, _ := json.Marshal(p)
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor decodes an opaque cursor string back into a CursorToken.
// Returns an error if the string is not valid base64url, not valid JSON,
// or contains an invalid timestamp or UUID.
func DecodeCursor(s string) (*domain.CursorToken, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var p cursorPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("invalid cursor payload: %w", err)
	}
	t, err := time.Parse(time.RFC3339Nano, p.T)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	id, err := uuid.Parse(p.I)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor id: %w", err)
	}
	return &domain.CursorToken{CreatedAt: t, ID: id}, nil
}

// WebhookService handles webhook endpoint CRUD and log query business logic.
type WebhookService struct {
	endpointRepo domain.WebhookEndpointRepository
	logRepo      domain.WebhookLogRepository
}

// NewWebhookService creates a WebhookService.
func NewWebhookService(endpointRepo domain.WebhookEndpointRepository, logRepo domain.WebhookLogRepository) *WebhookService {
	return &WebhookService{endpointRepo: endpointRepo, logRepo: logRepo}
}

// CreateEndpoint creates a new webhook endpoint for the given organization.
func (s *WebhookService) CreateEndpoint(ctx context.Context, orgID uuid.UUID, input domain.CreateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	return s.endpointRepo.Create(ctx, orgID, input)
}

// GetEndpointByID returns a single webhook endpoint by ID scoped to an organization.
func (s *WebhookService) GetEndpointByID(ctx context.Context, orgID, endpointID uuid.UUID) (*domain.WebhookEndpoint, error) {
	return s.endpointRepo.GetByID(ctx, orgID, endpointID)
}

// ListEndpoints returns a page of webhook endpoints for the given organization.
func (s *WebhookService) ListEndpoints(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.WebhookEndpoint, error) {
	return s.endpointRepo.List(ctx, orgID, params)
}

// UpdateEndpoint updates a webhook endpoint for the given organization.
func (s *WebhookService) UpdateEndpoint(ctx context.Context, orgID, endpointID uuid.UUID, input domain.UpdateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	return s.endpointRepo.Update(ctx, orgID, endpointID, input)
}

// DeleteEndpoint soft-deletes a webhook endpoint for the given organization.
func (s *WebhookService) DeleteEndpoint(ctx context.Context, orgID, endpointID uuid.UUID) error {
	return s.endpointRepo.SoftDelete(ctx, orgID, endpointID)
}

// ListLogs returns a page of webhook logs and an optional next_cursor string.
// It fetches limit+1 rows to determine whether a next page exists.
func (s *WebhookService) ListLogs(ctx context.Context, orgID uuid.UUID, input domain.ListWebhookLogsInput) ([]domain.WebhookLog, *string, error) {
	// Fetch one extra to detect next page.
	fetchInput := input
	fetchInput.Limit = input.Limit + 1

	rows, err := s.logRepo.List(ctx, orgID, fetchInput)
	if err != nil {
		return nil, nil, err
	}

	var nextCursor *string
	if len(rows) > input.Limit {
		// There is a next page; generate cursor from the last item in the current page.
		last := rows[input.Limit-1]
		c := EncodeCursor(last.CreatedAt, last.ID)
		nextCursor = &c
		rows = rows[:input.Limit]
	}

	return rows, nextCursor, nil
}

// GetLogByID returns a single webhook log by ID scoped to an organization.
func (s *WebhookService) GetLogByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.WebhookLog, error) {
	return s.logRepo.GetByID(ctx, orgID, logID)
}

// ListEventTypes returns all supported webhook event types.
func (s *WebhookService) ListEventTypes() []string {
	return domain.ValidEventTypes()
}
