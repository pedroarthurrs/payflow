package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	memberlogdb "i9pay/internal/database/dbgen/memberlog"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// MemberLogRepo implements domain.MemberLogRepository backed by PostgreSQL via sqlc.
type MemberLogRepo struct {
	q memberlogdb.Querier
}

// NewMemberLogRepo creates a MemberLogRepo.
func NewMemberLogRepo(q memberlogdb.Querier) *MemberLogRepo {
	return &MemberLogRepo{q: q}
}

// List returns a page of member logs for an organization, applying optional filters.
func (r *MemberLogRepo) List(ctx context.Context, orgID uuid.UUID, input domain.ListMemberLogsInput) ([]domain.MemberLogRecord, error) {
	params := memberlogdb.ListMemberLogsParams{
		OrganizationID: orgID,
		PageLimit:      int32(input.Limit),
	}

	if input.Cursor != nil {
		params.CursorTs = sql.NullTime{Time: input.Cursor.CreatedAt, Valid: true}
		params.CursorID = uuid.NullUUID{UUID: input.Cursor.ID, Valid: true}
	}
	if input.MemberID != nil {
		params.MemberID = uuid.NullUUID{UUID: *input.MemberID, Valid: true}
	}
	if input.Action != nil {
		params.Action = sql.NullString{String: *input.Action, Valid: true}
	}
	if input.From != nil {
		params.FromTs = sql.NullTime{Time: *input.From, Valid: true}
	}
	if input.To != nil {
		params.ToTs = sql.NullTime{Time: *input.To, Valid: true}
	}
	if input.StatusCode != nil {
		params.StatusCode = sql.NullInt32{Int32: *input.StatusCode, Valid: true}
	}
	if input.Method != nil {
		params.Method = sql.NullString{String: *input.Method, Valid: true}
	}
	if input.PathPrefix != nil {
		params.PathPrefix = sql.NullString{String: *input.PathPrefix, Valid: true}
	}

	rows, err := r.q.ListMemberLogs(ctx, params)
	if err != nil {
		slog.Error("failed to list member logs", "organization_id", orgID, "error", err)
		return nil, domain.ErrInternal
	}

	records := make([]domain.MemberLogRecord, len(rows))
	for i := range rows {
		records[i] = toDomainMemberLogFromListRow(rows[i])
	}
	return records, nil
}

// GetByID retrieves a single member log by ID scoped to an organization.
func (r *MemberLogRepo) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.MemberLogRecord, error) {
	row, err := r.q.GetMemberLogByID(ctx, memberlogdb.GetMemberLogByIDParams{
		OrganizationID: orgID,
		ID:             logID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMemberLogNotFound
		}
		slog.Error("failed to get member log", "organization_id", orgID, "log_id", logID, "error", err)
		return nil, domain.ErrInternal
	}
	rec := toDomainMemberLogFromGetRow(row)
	return &rec, nil
}

func toDomainMemberLogFromGetRow(row memberlogdb.GetMemberLogByIDRow) domain.MemberLogRecord {
	rec := domain.MemberLogRecord{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		MemberID:       row.MemberID,
		Action:         row.Action,
		CreatedAt:      row.CreatedAt.Time,
	}
	if row.Method.Valid {
		rec.Method = &row.Method.String
	}
	if row.Path.Valid {
		rec.Path = &row.Path.String
	}
	if row.StatusCode.Valid {
		rec.StatusCode = &row.StatusCode.Int32
	}
	if row.LatencyMs.Valid {
		rec.LatencyMs = &row.LatencyMs.Int32
	}
	if row.IpAddress.Valid {
		rec.IPAddress = &row.IpAddress.String
	}
	if row.UserAgent.Valid {
		rec.UserAgent = &row.UserAgent.String
	}
	if row.Metadata.Valid {
		rec.Metadata = row.Metadata.RawMessage
	}
	return rec
}

func toDomainMemberLogFromListRow(row memberlogdb.ListMemberLogsRow) domain.MemberLogRecord {
	rec := domain.MemberLogRecord{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		MemberID:       row.MemberID,
		Action:         row.Action,
		CreatedAt:      row.CreatedAt.Time,
	}
	if row.Method.Valid {
		rec.Method = &row.Method.String
	}
	if row.Path.Valid {
		rec.Path = &row.Path.String
	}
	if row.StatusCode.Valid {
		rec.StatusCode = &row.StatusCode.Int32
	}
	if row.LatencyMs.Valid {
		rec.LatencyMs = &row.LatencyMs.Int32
	}
	if row.IpAddress.Valid {
		rec.IPAddress = &row.IpAddress.String
	}
	if row.UserAgent.Valid {
		rec.UserAgent = &row.UserAgent.String
	}
	if row.Metadata.Valid {
		rec.Metadata = row.Metadata.RawMessage
	}
	return rec
}

// Create persists a new member log entry.
func (r *MemberLogRepo) Create(ctx context.Context, input domain.CreateMemberLogInput) error {
	params := memberlogdb.CreateMemberLogParams{
		OrganizationID: input.OrganizationID,
		MemberID:       input.MemberID,
		Action:         input.Action,
	}
	if input.Method != nil {
		params.Method = sql.NullString{String: *input.Method, Valid: true}
	}
	if input.Path != nil {
		params.Path = sql.NullString{String: *input.Path, Valid: true}
	}
	if input.StatusCode != nil {
		params.StatusCode = sql.NullInt32{Int32: *input.StatusCode, Valid: true}
	}
	if input.LatencyMs != nil {
		params.LatencyMs = sql.NullInt32{Int32: *input.LatencyMs, Valid: true}
	}
	if input.IPAddress != nil {
		params.IpAddress = sql.NullString{String: *input.IPAddress, Valid: true}
	}
	if input.UserAgent != nil {
		params.UserAgent = sql.NullString{String: *input.UserAgent, Valid: true}
	}
	if input.Metadata != nil {
		params.Metadata = pqtype.NullRawMessage{RawMessage: input.Metadata, Valid: true}
	}

	if err := r.q.CreateMemberLog(ctx, params); err != nil {
		slog.Error("failed to create member log", "member_id", input.MemberID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

var _ domain.MemberLogWriter = (*MemberLogRepo)(nil)
var _ domain.MemberLogRepository = (*MemberLogRepo)(nil)
