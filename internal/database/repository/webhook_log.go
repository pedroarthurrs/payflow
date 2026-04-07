package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	webhooklogdb "i9pay/internal/database/dbgen/webhooklog"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

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
	rec := toDomainWebhookLog(row)
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
		records[i] = toDomainWebhookLog(rows[i])
	}
	return records, nil
}

func toDomainWebhookLog(row webhooklogdb.WebhookLog) domain.WebhookLog {
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
