package service

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ── Cursor encode/decode ──────────────────────────────────────────────────────

func TestCredentialLog_EncodeCursor_RoundTrip(t *testing.T) {
	ts := time.Now().UTC().Truncate(time.Nanosecond)
	id := uuid.New()

	encoded := EncodeCursor(ts, id)
	if encoded == "" {
		t.Fatal("expected non-empty cursor")
	}

	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if !decoded.CreatedAt.Equal(ts) {
		t.Fatalf("timestamp mismatch: got %v, want %v", decoded.CreatedAt, ts)
	}
	if decoded.ID != id {
		t.Fatalf("id mismatch: got %v, want %v", decoded.ID, id)
	}
}

func TestCredentialLog_DecodeCursor_InvalidBase64(t *testing.T) {
	_, err := DecodeCursor("!!!not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestCredentialLog_DecodeCursor_InvalidJSON(t *testing.T) {
	s := base64.RawURLEncoding.EncodeToString([]byte("invalid-json"))
	_, err := DecodeCursor(s)
	if err == nil {
		t.Fatal("expected error for invalid JSON payload")
	}
}

func TestCredentialLog_DecodeCursor_InvalidTimestamp(t *testing.T) {
	id := uuid.New()
	s := base64.RawURLEncoding.EncodeToString([]byte(`{"t":"not-a-time","i":"` + id.String() + `"}`))
	_, err := DecodeCursor(s)
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}

func TestCredentialLog_DecodeCursor_InvalidUUID(t *testing.T) {
	s := base64.RawURLEncoding.EncodeToString([]byte(`{"t":"2026-01-01T00:00:00Z","i":"not-a-uuid"}`))
	_, err := DecodeCursor(s)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

// ── List pagination ───────────────────────────────────────────────────────────

func TestCredentialLogService_List_NextCursorPresent(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().UTC()

	// Repo returns limit+1 rows to trigger next_cursor generation.
	repo := &mockCredentialLogRepo{
		listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
			rows := make([]domain.CredentialLogRecord, input.Limit)
			for i := range rows {
				rows[i] = domain.CredentialLogRecord{
					ID:        uuid.New(),
					CreatedAt: now.Add(-time.Duration(i) * time.Second),
				}
			}
			return rows, nil
		},
	}

	svc := NewCredentialService(nil, repo, nil, "test-pepper")
	records, nextCursor, err := svc.ListLogs(context.Background(), orgID, domain.ListCredentialLogsInput{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if nextCursor == nil {
		t.Fatal("expected next_cursor to be set")
	}
}

func TestCredentialLogService_List_NoNextCursor(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().UTC()

	// Repo returns fewer rows than limit+1 — no next page.
	repo := &mockCredentialLogRepo{
		listFn: func(_ context.Context, _ uuid.UUID, _ domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
			return []domain.CredentialLogRecord{
				{ID: uuid.New(), CreatedAt: now},
			}, nil
		},
	}

	svc := NewCredentialService(nil, repo, nil, "test-pepper")
	records, nextCursor, err := svc.ListLogs(context.Background(), orgID, domain.ListCredentialLogsInput{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if nextCursor != nil {
		t.Fatalf("expected nil next_cursor, got %v", *nextCursor)
	}
}

func TestCredentialLogService_List_Empty(t *testing.T) {
	repo := &mockCredentialLogRepo{
		listFn: func(_ context.Context, _ uuid.UUID, _ domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
			return nil, nil
		},
	}
	svc := NewCredentialService(nil, repo, nil, "test-pepper")
	records, nextCursor, err := svc.ListLogs(context.Background(), uuid.New(), domain.ListCredentialLogsInput{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
	if nextCursor != nil {
		t.Fatal("expected nil next_cursor for empty result")
	}
}

func TestCredentialLogService_List_FilterPassthrough(t *testing.T) {
	orgID := uuid.New()
	method := "POST"
	sc := int32(201)
	path := "/v1/customers"

	var capturedInput domain.ListCredentialLogsInput
	repo := &mockCredentialLogRepo{
		listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
			capturedInput = input
			return nil, nil
		},
	}

	svc := NewCredentialService(nil, repo, nil, "test-pepper")
	_, _, err := svc.ListLogs(context.Background(), orgID, domain.ListCredentialLogsInput{
		Limit:      10,
		Method:     &method,
		StatusCode: &sc,
		PathPrefix: &path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Service fetches limit+1.
	if capturedInput.Limit != 11 {
		t.Fatalf("expected captured limit 11, got %d", capturedInput.Limit)
	}
	if capturedInput.Method == nil || *capturedInput.Method != method {
		t.Fatalf("expected method %q passed through", method)
	}
	if capturedInput.StatusCode == nil || *capturedInput.StatusCode != sc {
		t.Fatalf("expected status_code %d passed through", sc)
	}
	if capturedInput.PathPrefix == nil || *capturedInput.PathPrefix != path {
		t.Fatalf("expected path_prefix %q passed through", path)
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestCredentialLogService_GetByID_Found(t *testing.T) {
	orgID := uuid.New()
	logID := uuid.New()

	repo := &mockCredentialLogRepo{
		getByIDFn: func(_ context.Context, o, l uuid.UUID) (*domain.CredentialLogRecord, error) {
			if o == orgID && l == logID {
				return &domain.CredentialLogRecord{ID: logID, OrganizationID: orgID}, nil
			}
			return nil, domain.ErrCredentialLogNotFound
		},
	}

	svc := NewCredentialService(nil, repo, nil, "test-pepper")
	rec, err := svc.GetLogByID(context.Background(), orgID, logID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ID != logID {
		t.Fatalf("expected ID %v, got %v", logID, rec.ID)
	}
}

func TestCredentialLogService_GetByID_NotFound(t *testing.T) {
	repo := &mockCredentialLogRepo{}
	svc := NewCredentialService(nil, repo, nil, "test-pepper")
	_, err := svc.GetLogByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCredentialLogNotFound) {
		t.Fatalf("expected ErrCredentialLogNotFound, got %v", err)
	}
}
