package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"i9pay/internal/domain"

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

// CredentialLogService handles credential log query business logic.
type CredentialLogService struct {
	repo domain.CredentialLogRepository
}

// NewCredentialLogService creates a CredentialLogService.
func NewCredentialLogService(repo domain.CredentialLogRepository) *CredentialLogService {
	return &CredentialLogService{repo: repo}
}

// List returns a page of credential logs and an optional next_cursor string.
// It fetches limit+1 rows to determine whether a next page exists.
func (s *CredentialLogService) List(ctx context.Context, orgID uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, *string, error) {
	// Fetch one extra to detect next page.
	fetchInput := input
	fetchInput.Limit = input.Limit + 1

	rows, err := s.repo.List(ctx, orgID, fetchInput)
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

// GetByID returns a single credential log by ID scoped to an organization.
func (s *CredentialLogService) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.CredentialLogRecord, error) {
	return s.repo.GetByID(ctx, orgID, logID)
}
