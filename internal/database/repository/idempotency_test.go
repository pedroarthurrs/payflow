package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	idempotencydb "i9pay/internal/database/dbgen/idempotency"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// ── Fake Querier ──────────────────────────────────────────────────────────────

type fakeIdempotencyQ struct {
	getFn  func(ctx context.Context, arg idempotencydb.GetIdempotencyKeyParams) (idempotencydb.GetIdempotencyKeyRow, error)
	saveFn func(ctx context.Context, arg idempotencydb.SaveIdempotencyKeyParams) error
}

var _ idempotencydb.Querier = (*fakeIdempotencyQ)(nil)

func (f *fakeIdempotencyQ) GetIdempotencyKey(ctx context.Context, arg idempotencydb.GetIdempotencyKeyParams) (idempotencydb.GetIdempotencyKeyRow, error) {
	if f.getFn != nil {
		return f.getFn(ctx, arg)
	}
	return idempotencydb.GetIdempotencyKeyRow{}, sql.ErrNoRows
}

func (f *fakeIdempotencyQ) SaveIdempotencyKey(ctx context.Context, arg idempotencydb.SaveIdempotencyKeyParams) error {
	if f.saveFn != nil {
		return f.saveFn(ctx, arg)
	}
	return nil
}

// ── Check ─────────────────────────────────────────────────────────────────────

func TestIdempotencyRepo_Check_Found(t *testing.T) {
	orgID := uuid.New()

	q := &fakeIdempotencyQ{
		getFn: func(_ context.Context, arg idempotencydb.GetIdempotencyKeyParams) (idempotencydb.GetIdempotencyKeyRow, error) {
			if arg.OrganizationID != orgID || arg.IdempotencyKey != "key-1" {
				t.Fatalf("unexpected args: %+v", arg)
			}
			return idempotencydb.GetIdempotencyKeyRow{
				ResponseCode: sql.NullInt32{Int32: 201, Valid: true},
				ResponseBody: pqtype.NullRawMessage{RawMessage: []byte(`{"id":"abc"}`), Valid: true},
			}, nil
		},
	}

	repo := NewIdempotencyRepo(q)
	record, err := repo.Check(context.Background(), "key-1", orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record == nil {
		t.Fatal("expected record, got nil")
	}
	if record.ResponseCode != 201 {
		t.Fatalf("expected code 201, got %d", record.ResponseCode)
	}
	if string(record.ResponseBody) != `{"id":"abc"}` {
		t.Fatalf("unexpected body: %s", record.ResponseBody)
	}
}

func TestIdempotencyRepo_Check_NotFound(t *testing.T) {
	q := &fakeIdempotencyQ{
		getFn: func(_ context.Context, _ idempotencydb.GetIdempotencyKeyParams) (idempotencydb.GetIdempotencyKeyRow, error) {
			return idempotencydb.GetIdempotencyKeyRow{}, sql.ErrNoRows
		},
	}

	repo := NewIdempotencyRepo(q)
	record, err := repo.Check(context.Background(), "missing", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil, got %+v", record)
	}
}

func TestIdempotencyRepo_Check_NullResponseCode(t *testing.T) {
	q := &fakeIdempotencyQ{
		getFn: func(_ context.Context, _ idempotencydb.GetIdempotencyKeyParams) (idempotencydb.GetIdempotencyKeyRow, error) {
			return idempotencydb.GetIdempotencyKeyRow{
				ResponseCode: sql.NullInt32{Valid: false},
			}, nil
		},
	}

	repo := NewIdempotencyRepo(q)
	record, err := repo.Check(context.Background(), "key", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil for null response_code, got %+v", record)
	}
}

func TestIdempotencyRepo_Check_DBError(t *testing.T) {
	q := &fakeIdempotencyQ{
		getFn: func(_ context.Context, _ idempotencydb.GetIdempotencyKeyParams) (idempotencydb.GetIdempotencyKeyRow, error) {
			return idempotencydb.GetIdempotencyKeyRow{}, sql.ErrConnDone
		},
	}

	repo := NewIdempotencyRepo(q)
	_, err := repo.Check(context.Background(), "key", uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestIdempotencyRepo_Save_Success(t *testing.T) {
	orgID := uuid.New()
	var savedParams idempotencydb.SaveIdempotencyKeyParams

	q := &fakeIdempotencyQ{
		saveFn: func(_ context.Context, arg idempotencydb.SaveIdempotencyKeyParams) error {
			savedParams = arg
			return nil
		},
	}

	repo := NewIdempotencyRepo(q)
	err := repo.Save(context.Background(), "key-2", orgID, "/v1/payments", domain.IdempotencyRecord{
		ResponseCode: 201,
		ResponseBody: []byte(`{"ok":true}`),
	}, 24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedParams.OrganizationID != orgID {
		t.Fatalf("expected org %v, got %v", orgID, savedParams.OrganizationID)
	}
	if savedParams.IdempotencyKey != "key-2" {
		t.Fatalf("expected key %q, got %q", "key-2", savedParams.IdempotencyKey)
	}
	if savedParams.RequestPath != "/v1/payments" {
		t.Fatalf("expected path %q, got %q", "/v1/payments", savedParams.RequestPath)
	}
	if savedParams.ResponseCode.Int32 != 201 {
		t.Fatalf("expected code 201, got %d", savedParams.ResponseCode.Int32)
	}
	if !savedParams.ResponseBody.Valid {
		t.Fatal("expected valid response body")
	}
	if savedParams.ExpiresAt.Before(time.Now()) {
		t.Fatal("expires_at should be in the future")
	}
}

func TestIdempotencyRepo_Save_NilBody(t *testing.T) {
	q := &fakeIdempotencyQ{
		saveFn: func(_ context.Context, arg idempotencydb.SaveIdempotencyKeyParams) error {
			if arg.ResponseBody.Valid {
				t.Fatal("expected invalid (NULL) response body for nil input")
			}
			return nil
		},
	}

	repo := NewIdempotencyRepo(q)
	err := repo.Save(context.Background(), "key", uuid.New(), "/v1/test", domain.IdempotencyRecord{
		ResponseCode: 204,
		ResponseBody: nil,
	}, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIdempotencyRepo_Save_DBError(t *testing.T) {
	q := &fakeIdempotencyQ{
		saveFn: func(_ context.Context, _ idempotencydb.SaveIdempotencyKeyParams) error {
			return sql.ErrConnDone
		},
	}

	repo := NewIdempotencyRepo(q)
	err := repo.Save(context.Background(), "key", uuid.New(), "/v1/test", domain.IdempotencyRecord{
		ResponseCode: 200,
	}, time.Hour)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
