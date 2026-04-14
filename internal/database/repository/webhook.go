package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	webhookdb "i9pay/internal/database/dbgen/webhook"
	webhooklogdb "i9pay/internal/database/dbgen/webhooklog"
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

// WebhookLogRepo implements domain.WebhookLogRepository backed by PostgreSQL via sqlc.
type WebhookLogRepo struct {
	q webhooklogdb.Querier
}

// NewWebhookLogRepo creates a WebhookLogRepo.
func NewWebhookLogRepo(q webhooklogdb.Querier) *WebhookLogRepo {
	return &WebhookLogRepo{q: q}
}

// Create persists a new webhook log entry.
func (r *WebhookLogRepo) Create(ctx context.Context, input domain.CreateWebhookLogInput) error {
	params := webhooklogdb.CreateWebhookLogParams{
		OrganizationID: input.OrganizationID,
		EndpointID:     input.EndpointID,
		EventType:      input.EventType,
		Payload:        input.Payload,
		AttemptNumber:  sql.NullInt32{Int32: input.AttemptNumber, Valid: true},
		Success:        sql.NullBool{Bool: input.Success, Valid: true},
	}
	if input.ResponseBody != nil {
		params.ResponseBody = sql.NullString{String: *input.ResponseBody, Valid: true}
	}
	if input.ResponseStatus != nil {
		params.ResponseStatus = sql.NullInt32{Int32: *input.ResponseStatus, Valid: true}
	}
	if input.DurationMs != nil {
		params.DurationMs = sql.NullInt32{Int32: *input.DurationMs, Valid: true}
	}
	if err := r.q.CreateWebhookLog(ctx, params); err != nil {
		slog.Error("failed to create webhook log", "organization_id", input.OrganizationID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// GetByID retrieves a single webhook log by ID scoped to an organization.
func (r *WebhookLogRepo) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.WebhookLog, error) {
	row, err := r.q.GetWebhookLogByID(ctx, webhooklogdb.GetWebhookLogByIDParams{
		OrganizationID: orgID,
		ID:             logID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWebhookLogNotFound
		}
		slog.Error("failed to get webhook log", "organization_id", orgID, "log_id", logID, "error", err)
		return nil, domain.ErrInternal
	}
	rec := toDomainWebhookLogFromGetRow(row)
	return &rec, nil
}

// List returns a page of webhook logs for an organization, applying optional filters.
func (r *WebhookLogRepo) List(ctx context.Context, orgID uuid.UUID, input domain.ListWebhookLogsInput) ([]domain.WebhookLog, error) {
	params := webhooklogdb.ListWebhookLogsParams{
		OrganizationID: orgID,
		PageLimit:      int32(input.Limit),
	}

	if input.Cursor != nil {
		params.CursorTs = sql.NullTime{Time: input.Cursor.CreatedAt, Valid: true}
		params.CursorID = uuid.NullUUID{UUID: input.Cursor.ID, Valid: true}
	}
	if input.From != nil {
		params.FromTs = sql.NullTime{Time: *input.From, Valid: true}
	}
	if input.To != nil {
		params.ToTs = sql.NullTime{Time: *input.To, Valid: true}
	}
	if input.EndpointID != nil {
		params.EndpointID = uuid.NullUUID{UUID: *input.EndpointID, Valid: true}
	}
	if input.EventType != nil {
		params.EventType = sql.NullString{String: *input.EventType, Valid: true}
	}
	if input.Success != nil {
		params.Success = sql.NullBool{Bool: *input.Success, Valid: true}
	}

	rows, err := r.q.ListWebhookLogs(ctx, params)
	if err != nil {
		slog.Error("failed to list webhook logs", "organization_id", orgID, "error", err)
		return nil, domain.ErrInternal
	}

	records := make([]domain.WebhookLog, len(rows))
	for i := range rows {
		records[i] = toDomainWebhookLogFromListRow(rows[i])
	}
	return records, nil
}

func toDomainWebhookLogFromGetRow(row webhooklogdb.GetWebhookLogByIDRow) domain.WebhookLog {
	rec := domain.WebhookLog{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		EndpointID:     row.EndpointID,
		EventType:      row.EventType,
		Payload:        row.Payload,
	}
	if row.ResponseBody.Valid {
		rec.ResponseBody = &row.ResponseBody.String
	}
	if row.ResponseStatus.Valid {
		rec.ResponseStatus = &row.ResponseStatus.Int32
	}
	if row.DurationMs.Valid {
		rec.DurationMs = &row.DurationMs.Int32
	}
	if row.AttemptNumber.Valid {
		rec.AttemptNumber = row.AttemptNumber.Int32
	}
	if row.Success.Valid {
		rec.Success = row.Success.Bool
	}
	if row.CreatedAt.Valid {
		rec.CreatedAt = row.CreatedAt.Time
	}
	return rec
}

func toDomainWebhookLogFromListRow(row webhooklogdb.ListWebhookLogsRow) domain.WebhookLog {
	rec := domain.WebhookLog{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		EndpointID:     row.EndpointID,
		EventType:      row.EventType,
		Payload:        row.Payload,
	}
	if row.ResponseBody.Valid {
		rec.ResponseBody = &row.ResponseBody.String
	}
	if row.ResponseStatus.Valid {
		rec.ResponseStatus = &row.ResponseStatus.Int32
	}
	if row.DurationMs.Valid {
		rec.DurationMs = &row.DurationMs.Int32
	}
	if row.AttemptNumber.Valid {
		rec.AttemptNumber = row.AttemptNumber.Int32
	}
	if row.Success.Valid {
		rec.Success = row.Success.Bool
	}
	if row.CreatedAt.Valid {
		rec.CreatedAt = row.CreatedAt.Time
	}
	return rec
}

var _ domain.WebhookLogRepository = (*WebhookLogRepo)(nil)
