package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

func TestCredentialCreate_Success(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().Truncate(time.Second)
	repo := &mockCredentialRepo{
		createFn: func(_ context.Context, o uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
			return &domain.CredentialRecord{
				ID: uuid.New(), OrganizationID: o,
				Name: input.Name, KeyPrefix: input.KeyPrefix,
				Environment: input.Environment, CreatedAt: now,
			}, nil
		},
	}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	rec, rawKey, err := svc.Create(context.Background(), orgID, domain.CreateCredentialInput{
		Name: "My Key", Environment: "production",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rawKey == "" {
		t.Fatal("expected raw key, got empty string")
	}
	if len(rawKey) < 7 || rawKey[:7] != "I9LIVE_" {
		t.Fatalf("expected I9LIVE_ prefix, got %q", rawKey)
	}
	if rec.KeyPrefix != rawKey[:10] {
		t.Fatalf("expected key_prefix %q, got %q", rawKey[:10], rec.KeyPrefix)
	}
}

func TestCredentialCreate_TestEnvironment_Prefix(t *testing.T) {
	orgID := uuid.New()
	repo := &mockCredentialRepo{
		createFn: func(_ context.Context, o uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
			return &domain.CredentialRecord{
				ID: uuid.New(), OrganizationID: o,
				Name: input.Name, KeyPrefix: input.KeyPrefix,
				Environment: input.Environment,
			}, nil
		},
	}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	_, rawKey, err := svc.Create(context.Background(), orgID, domain.CreateCredentialInput{
		Name: "Test Key", Environment: "test",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(rawKey) < 7 || rawKey[:7] != "I9TEST_" {
		t.Fatalf("expected I9TEST_ prefix, got %q", rawKey)
	}
}

func TestCredentialList_Success(t *testing.T) {
	orgID := uuid.New()
	repo := &mockCredentialRepo{
		listFn: func(_ context.Context, o uuid.UUID) ([]domain.CredentialRecord, error) {
			return []domain.CredentialRecord{
				{ID: uuid.New(), OrganizationID: o, Name: "Key 1", Environment: "test"},
				{ID: uuid.New(), OrganizationID: o, Name: "Key 2", Environment: "production"},
			}, nil
		},
	}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	keys, err := svc.List(context.Background(), orgID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestCredentialList_Empty(t *testing.T) {
	orgID := uuid.New()
	repo := &mockCredentialRepo{
		listFn: func(_ context.Context, _ uuid.UUID) ([]domain.CredentialRecord, error) {
			return []domain.CredentialRecord{}, nil
		},
	}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	keys, err := svc.List(context.Background(), orgID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected empty list, got %d", len(keys))
	}
}

func TestCredentialGetByID_Found(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()
	repo := &mockCredentialRepo{
		getByIDFn: func(_ context.Context, o, k uuid.UUID) (*domain.CredentialRecord, error) {
			if o == orgID && k == credID {
				return &domain.CredentialRecord{ID: k, OrganizationID: o, Name: "My Key"}, nil
			}
			return nil, domain.ErrCredentialNotFound
		},
	}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	rec, err := svc.GetByID(context.Background(), orgID, credID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.ID != credID {
		t.Fatalf("expected ID %v, got %v", credID, rec.ID)
	}
}

func TestCredentialGetByID_NotFound(t *testing.T) {
	repo := &mockCredentialRepo{}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCredentialNotFound) {
		t.Fatalf("expected ErrCredentialNotFound, got %v", err)
	}
}

func TestCredentialRevoke_Success(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()
	deleted := false
	repo := &mockCredentialRepo{
		softDeleteFn: func(_ context.Context, o, k uuid.UUID) error {
			if o == orgID && k == credID {
				deleted = true
			}
			return nil
		},
	}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	err := svc.Revoke(context.Background(), orgID, credID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleted {
		t.Fatal("expected SoftDelete to be called")
	}
}

func TestCredentialRevoke_NotFound(t *testing.T) {
	repo := &mockCredentialRepo{
		softDeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return domain.ErrCredentialNotFound
		},
	}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	err := svc.Revoke(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCredentialNotFound) {
		t.Fatalf("expected ErrCredentialNotFound, got %v", err)
	}
}

func TestCredentialRotate_Success(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()
	now := time.Now().Truncate(time.Second)
	existing := &domain.CredentialRecord{
		ID: credID, OrganizationID: orgID,
		Name: "Old Key", Environment: "production",
		CreatedAt: now,
	}
	newID := uuid.New()
	repo := &mockCredentialRepo{
		getByIDFn: func(_ context.Context, o, k uuid.UUID) (*domain.CredentialRecord, error) {
			if o == orgID && k == credID {
				return existing, nil
			}
			return nil, domain.ErrCredentialNotFound
		},
	}
	txMgr := &mockTxManager{
		withTxFn: func(_ context.Context, fn func(tx domain.TxHandle) error) error {
			return fn(&mockTxHandle{
				credentials: &mockCredentialRepo{
					softDeleteFn: func(_ context.Context, _, _ uuid.UUID) error { return nil },
					createFn: func(_ context.Context, o uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
						return &domain.CredentialRecord{
							ID: newID, OrganizationID: o,
							Name: input.Name, KeyPrefix: input.KeyPrefix,
							Environment: input.Environment, CreatedAt: now,
						}, nil
					},
				},
			})
		},
	}
	svc := NewCredentialService(repo, txMgr, "test-pepper")
	newRec, rawKey, err := svc.Rotate(context.Background(), orgID, credID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if newRec == nil {
		t.Fatal("expected new record, got nil")
	}
	if newRec.ID == credID {
		t.Fatal("expected new ID, got same as old")
	}
	if rawKey == "" {
		t.Fatal("expected raw key, got empty")
	}
	if newRec.Name != existing.Name {
		t.Fatalf("expected same name %q, got %q", existing.Name, newRec.Name)
	}
	if newRec.Environment != existing.Environment {
		t.Fatalf("expected same environment %q, got %q", existing.Environment, newRec.Environment)
	}
}

func TestCredentialRotate_NotFound(t *testing.T) {
	repo := &mockCredentialRepo{}
	svc := NewCredentialService(repo, &mockTxManager{}, "test-pepper")
	_, _, err := svc.Rotate(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCredentialNotFound) {
		t.Fatalf("expected ErrCredentialNotFound, got %v", err)
	}
}

func TestNewCredentialService_PanicOnEmptyPepper(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty pepper")
		}
	}()
	NewCredentialService(&mockCredentialRepo{}, &mockTxManager{}, "")
}
