package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	webhookdb "i9pay/internal/database/dbgen/webhook"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// WebhookRepo implements domain.WebhookEndpointRepository backed by PostgreSQL via sqlc.
type WebhookRepo struct {
	q webhookdb.Querier
}

// NewWebhookRepo creates a WebhookRepo.
func NewWebhookRepo(q webhookdb.Querier) *WebhookRepo {
	return &WebhookRepo{q: q}
}

// Create persists a new webhook endpoint for an organization.
func (r *WebhookRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	row, err := r.q.CreateWebhookEndpoint(ctx, webhookdb.CreateWebhookEndpointParams{
		OrganizationID: orgID,
		Url:            input.URL,
		Description:    toNullString(input.Description),
		EnabledEvents:  input.EnabledEvents,
		Environment:    webhookdb.NullEnvironmentType{EnvironmentType: webhookdb.EnvironmentType(input.Environment), Valid: input.Environment != ""},
	})
	if err != nil {
		slog.Error("failed to create webhook endpoint", "organization_id", orgID, "error", err)
		return nil, domain.ErrInternal
	}
	ep := toDomainWebhookEndpoint(row.ID, row.OrganizationID, row.Url, row.Description, row.EnabledEvents, row.Environment, row.IsActive, row.CreatedAt, row.UpdatedAt)
	return &ep, nil
}

// GetByID retrieves a webhook endpoint by ID scoped to an organization.
func (r *WebhookRepo) GetByID(ctx context.Context, orgID, endpointID uuid.UUID) (*domain.WebhookEndpoint, error) {
	row, err := r.q.GetWebhookEndpointByID(ctx, webhookdb.GetWebhookEndpointByIDParams{
		OrganizationID: orgID,
		ID:             endpointID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWebhookEndpointNotFound
		}
		slog.Error("failed to get webhook endpoint", "organization_id", orgID, "endpoint_id", endpointID, "error", err)
		return nil, domain.ErrInternal
	}
	ep := toDomainWebhookEndpoint(row.ID, row.OrganizationID, row.Url, row.Description, row.EnabledEvents, row.Environment, row.IsActive, row.CreatedAt, row.UpdatedAt)
	return &ep, nil
}

// List returns a page of webhook endpoints for an organization.
func (r *WebhookRepo) List(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.WebhookEndpoint, error) {
	rows, err := r.q.ListWebhookEndpoints(ctx, webhookdb.ListWebhookEndpointsParams{
		OrganizationID: orgID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list webhook endpoints", "organization_id", orgID, "error", err)
		return nil, domain.ErrInternal
	}
	endpoints := make([]domain.WebhookEndpoint, len(rows))
	for i := range rows {
		endpoints[i] = toDomainWebhookEndpoint(rows[i].ID, rows[i].OrganizationID, rows[i].Url, rows[i].Description, rows[i].EnabledEvents, rows[i].Environment, rows[i].IsActive, rows[i].CreatedAt, rows[i].UpdatedAt)
	}
	return endpoints, nil
}

// Update modifies a webhook endpoint.
func (r *WebhookRepo) Update(ctx context.Context, orgID, endpointID uuid.UUID, input domain.UpdateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	params := webhookdb.UpdateWebhookEndpointParams{
		OrganizationID: orgID,
		EndpointID:     endpointID,
		EnabledEvents:  input.EnabledEvents,
	}
	if input.URL != nil {
		params.Url = sql.NullString{String: *input.URL, Valid: true}
	}
	if input.Description != nil {
		params.Description = sql.NullString{String: *input.Description, Valid: true}
	}
	if input.IsActive != nil {
		params.IsActive = sql.NullBool{Bool: *input.IsActive, Valid: true}
	}

	row, err := r.q.UpdateWebhookEndpoint(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWebhookEndpointNotFound
		}
		slog.Error("failed to update webhook endpoint", "organization_id", orgID, "endpoint_id", endpointID, "error", err)
		return nil, domain.ErrInternal
	}
	ep := toDomainWebhookEndpoint(row.ID, row.OrganizationID, row.Url, row.Description, row.EnabledEvents, row.Environment, row.IsActive, row.CreatedAt, row.UpdatedAt)
	return &ep, nil
}

// SoftDelete marks a webhook endpoint as deleted.
func (r *WebhookRepo) SoftDelete(ctx context.Context, orgID, endpointID uuid.UUID) error {
	// Verify existence first
	_, err := r.q.GetWebhookEndpointByID(ctx, webhookdb.GetWebhookEndpointByIDParams{
		OrganizationID: orgID,
		ID:             endpointID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrWebhookEndpointNotFound
		}
		slog.Error("failed to get webhook endpoint for delete", "organization_id", orgID, "endpoint_id", endpointID, "error", err)
		return domain.ErrInternal
	}

	if err := r.q.SoftDeleteWebhookEndpoint(ctx, webhookdb.SoftDeleteWebhookEndpointParams{
		OrganizationID: orgID,
		ID:             endpointID,
	}); err != nil {
		slog.Error("failed to soft delete webhook endpoint", "organization_id", orgID, "endpoint_id", endpointID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// ListActiveByEvent returns active endpoints matching the given event type and environment.
func (r *WebhookRepo) ListActiveByEvent(ctx context.Context, orgID uuid.UUID, eventType, env string) ([]domain.WebhookEndpoint, error) {
	rows, err := r.q.ListActiveWebhookEndpointsByEvent(ctx, webhookdb.ListActiveWebhookEndpointsByEventParams{
		OrganizationID: orgID,
		Column2:        webhookdb.EnvironmentType(env),
		EnabledEvents:  []string{eventType},
	})
	if err != nil {
		slog.Error("failed to list active webhook endpoints by event", "organization_id", orgID, "event_type", eventType, "error", err)
		return nil, domain.ErrInternal
	}
	endpoints := make([]domain.WebhookEndpoint, len(rows))
	for i := range rows {
		endpoints[i] = toDomainWebhookEndpoint(rows[i].ID, rows[i].OrganizationID, rows[i].Url, rows[i].Description, rows[i].EnabledEvents, rows[i].Environment, rows[i].IsActive, rows[i].CreatedAt, rows[i].UpdatedAt)
	}
	return endpoints, nil
}

func toDomainWebhookEndpoint(
	id, orgID uuid.UUID,
	url string,
	description sql.NullString,
	enabledEvents []string,
	environment webhookdb.NullEnvironmentType,
	isActive sql.NullBool,
	createdAt, updatedAt sql.NullTime,
) domain.WebhookEndpoint {
	ep := domain.WebhookEndpoint{
		ID:             id,
		OrganizationID: orgID,
		URL:            url,
		EnabledEvents:  enabledEvents,
	}
	if description.Valid {
		ep.Description = &description.String
	}
	if environment.Valid {
		ep.Environment = string(environment.EnvironmentType)
	}
	if isActive.Valid {
		ep.IsActive = isActive.Bool
	}
	if createdAt.Valid {
		ep.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		ep.UpdatedAt = updatedAt.Time
	}
	return ep
}

var _ domain.WebhookEndpointRepository = (*WebhookRepo)(nil)
