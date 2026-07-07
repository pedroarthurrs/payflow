package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// TestProperty_Credential_RawKeyFormat verifies P1: raw key format.
// Feature: credentials, Property 1: Formato do Raw Key
func TestProperty_Credential_RawKeyFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		env := rapid.SampledFrom([]string{"test", "production"}).Draw(t, "env")
		repo := &mockCredentialRepo{
			createFn: func(_ context.Context, o uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
				return &domain.CredentialRecord{
					ID: uuid.New(), OrganizationID: o,
					Name: input.Name, KeyPrefix: input.KeyPrefix,
					Environment: input.Environment, CreatedAt: time.Now(),
				}, nil
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, "test-pepper")
		_, rawKey, err := svc.Create(context.Background(), uuid.New(), domain.CreateCredentialInput{
			Name: "k", Environment: env,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if env == "production" {
			if len(rawKey) < 7 || rawKey[:7] != "I9LIVE_" {
				t.Fatalf("expected I9LIVE_ prefix for production, got %q", rawKey)
			}
		} else {
			if len(rawKey) < 7 || rawKey[:7] != "I9TEST_" {
				t.Fatalf("expected I9TEST_ prefix for test, got %q", rawKey)
			}
		}
		if len(rawKey) != 71 {
			t.Fatalf("expected len 71, got %d (rawKey=%q)", len(rawKey), rawKey)
		}
	})
}

// TestProperty_Credential_HashRoundTrip verifies P2: hash round-trip.
// Feature: credentials, Property 2: Round-trip de hashing
func TestProperty_Credential_HashRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pepper := rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "pepper")
		env := rapid.SampledFrom([]string{"test", "production"}).Draw(t, "env")
		var capturedHash string
		repo := &mockCredentialRepo{
			createFn: func(_ context.Context, o uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
				capturedHash = input.KeyHash
				return &domain.CredentialRecord{
					ID: uuid.New(), OrganizationID: o,
					Name: input.Name, KeyPrefix: input.KeyPrefix,
					Environment: input.Environment, CreatedAt: time.Now(),
				}, nil
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, pepper)
		_, rawKey, err := svc.Create(context.Background(), uuid.New(), domain.CreateCredentialInput{
			Name: "k", Environment: env,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		h := sha256.Sum256([]byte(pepper + rawKey))
		expectedHash := hex.EncodeToString(h[:])
		if capturedHash != expectedHash {
			t.Fatalf("hash mismatch: captured=%q, expected=%q", capturedHash, expectedHash)
		}
	})
}

// TestProperty_Credential_KeyPrefixIsPrefix verifies P3: key_prefix is prefix of raw key.
// Feature: credentials, Property 3: Key_Prefix é prefixo do Raw Key
func TestProperty_Credential_KeyPrefixIsPrefix(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		env := rapid.SampledFrom([]string{"test", "production"}).Draw(t, "env")
		var capturedPrefix string
		repo := &mockCredentialRepo{
			createFn: func(_ context.Context, o uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
				capturedPrefix = input.KeyPrefix
				return &domain.CredentialRecord{
					ID: uuid.New(), OrganizationID: o,
					Name: input.Name, KeyPrefix: input.KeyPrefix,
					Environment: input.Environment, CreatedAt: time.Now(),
				}, nil
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, "test-pepper")
		_, rawKey, err := svc.Create(context.Background(), uuid.New(), domain.CreateCredentialInput{
			Name: "k", Environment: env,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rawKey) < 10 {
			t.Fatalf("rawKey too short: %q", rawKey)
		}
		if capturedPrefix != rawKey[:10] {
			t.Fatalf("key_prefix %q != rawKey[:10] %q", capturedPrefix, rawKey[:10])
		}
	})
}

// TestProperty_Credential_ExpiresAtOptional verifies P7: expires_at is optional.
// Feature: credentials, Property 7: expires_at opcional
func TestProperty_Credential_ExpiresAtOptional(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasExpiry := rapid.Bool().Draw(t, "has_expiry")
		var capturedExpiresAt *time.Time
		repo := &mockCredentialRepo{
			createFn: func(_ context.Context, o uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
				capturedExpiresAt = input.ExpiresAt
				return &domain.CredentialRecord{
					ID: uuid.New(), OrganizationID: o,
					Name: input.Name, KeyPrefix: input.KeyPrefix,
					Environment: input.Environment, ExpiresAt: input.ExpiresAt,
					CreatedAt: time.Now(),
				}, nil
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, "test-pepper")
		inp := domain.CreateCredentialInput{Name: "k", Environment: "test"}
		var expectedExpiry *time.Time
		if hasExpiry {
			future := time.Now().Add(24 * time.Hour).Truncate(time.Second)
			expectedExpiry = &future
			inp.ExpiresAt = &future
		}
		_, _, err := svc.Create(context.Background(), uuid.New(), inp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasExpiry {
			if capturedExpiresAt == nil {
				t.Fatal("expected non-nil ExpiresAt, got nil")
			}
			if !capturedExpiresAt.Equal(*expectedExpiry) {
				t.Fatalf("ExpiresAt mismatch: got %v, want %v", capturedExpiresAt, expectedExpiry)
			}
		} else if capturedExpiresAt != nil {
			t.Fatalf("expected nil ExpiresAt, got %v", capturedExpiresAt)
		}
	})
}

// TestProperty_Credential_ListReturnsOnlyActive verifies P8: list returns only active credentials.
// Feature: credentials, Property 8: Listagem retorna apenas chaves ativas
func TestProperty_Credential_ListReturnsOnlyActive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 10).Draw(t, "n")
		orgID := uuid.New()
		records := make([]domain.CredentialRecord, n)
		for i := range records {
			records[i] = domain.CredentialRecord{
				ID: uuid.New(), OrganizationID: orgID,
				Name: "key", Environment: "test", CreatedAt: time.Now(),
			}
		}
		repo := &mockCredentialRepo{
			listFn: func(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.CredentialRecord, error) {
				return records, nil
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, "test-pepper")
		keys, err := svc.List(context.Background(), orgID, domain.ListParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != n {
			t.Fatalf("expected %d keys, got %d", n, len(keys))
		}
	})
}

// TestProperty_Credential_SoftDeleteRemovesFromList verifies P10: soft-delete removes credential from list.
// Feature: credentials, Property 10: Soft-delete remove chave da listagem
func TestProperty_Credential_SoftDeleteRemovesFromList(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
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
			listFn: func(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.CredentialRecord, error) {
				if deleted {
					return []domain.CredentialRecord{}, nil
				}
				return []domain.CredentialRecord{
					{ID: credID, OrganizationID: orgID, Name: "key", Environment: "test"},
				}, nil
			},
			getByIDFn: func(_ context.Context, o, k uuid.UUID) (*domain.CredentialRecord, error) {
				if deleted {
					return nil, domain.ErrCredentialNotFound
				}
				return &domain.CredentialRecord{ID: k, OrganizationID: o}, nil
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, "test-pepper")
		if err := svc.Revoke(context.Background(), orgID, credID); err != nil {
			t.Fatalf("unexpected revoke error: %v", err)
		}
		keys, err := svc.List(context.Background(), orgID, domain.ListParams{})
		if err != nil {
			t.Fatalf("unexpected list error: %v", err)
		}
		if len(keys) != 0 {
			t.Fatalf("expected 0 keys after revoke, got %d", len(keys))
		}
		_, err = svc.GetByID(context.Background(), orgID, credID)
		if !errors.Is(err, domain.ErrCredentialNotFound) {
			t.Fatalf("expected ErrCredentialNotFound after revoke, got %v", err)
		}
	})
}

// TestProperty_Credential_MultiTenantIsolation verifies P11: multi-tenant isolation.
// Feature: credentials, Property 11: Isolamento multi-tenant
func TestProperty_Credential_MultiTenantIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgA := uuid.New()
		orgB := uuid.New()
		for orgB == orgA {
			orgB = uuid.New()
		}
		credID := uuid.New()
		repo := &mockCredentialRepo{
			getByIDFn: func(_ context.Context, o, k uuid.UUID) (*domain.CredentialRecord, error) {
				if o == orgA && k == credID {
					return &domain.CredentialRecord{ID: k, OrganizationID: o}, nil
				}
				return nil, domain.ErrCredentialNotFound
			},
			softDeleteFn: func(_ context.Context, o, k uuid.UUID) error {
				if o == orgA && k == credID {
					return nil
				}
				return domain.ErrCredentialNotFound
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, "test-pepper")
		_, err := svc.GetByID(context.Background(), orgB, credID)
		if !errors.Is(err, domain.ErrCredentialNotFound) {
			t.Fatalf("GetByID: expected ErrCredentialNotFound for orgB, got %v", err)
		}
		err = svc.Revoke(context.Background(), orgB, credID)
		if !errors.Is(err, domain.ErrCredentialNotFound) {
			t.Fatalf("Revoke: expected ErrCredentialNotFound for orgB, got %v", err)
		}
	})
}

// TestProperty_Credential_RevokedKeyNotReturned verifies P12: revoked credential not returned.
// Feature: credentials, Property 12: Autenticação falha após revogação
func TestProperty_Credential_RevokedKeyNotReturned(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()
		credID := uuid.New()
		deleted := false
		repo := &mockCredentialRepo{
			softDeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
				deleted = true
				return nil
			},
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialRecord, error) {
				if deleted {
					return nil, domain.ErrCredentialNotFound
				}
				return &domain.CredentialRecord{ID: credID, OrganizationID: orgID}, nil
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, "test-pepper")
		if err := svc.Revoke(context.Background(), orgID, credID); err != nil {
			t.Fatalf("unexpected revoke error: %v", err)
		}
		_, err := svc.GetByID(context.Background(), orgID, credID)
		if !errors.Is(err, domain.ErrCredentialNotFound) {
			t.Fatalf("expected ErrCredentialNotFound after revoke, got %v", err)
		}
	})
}

// TestProperty_Credential_CreateGetRoundTrip verifies P13: create → get round-trip.
// Feature: credentials, Property 13: Round-trip create → get
func TestProperty_Credential_CreateGetRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[A-Za-z0-9 ]{1,50}`).Draw(t, "name")
		env := rapid.SampledFrom([]string{"test", "production"}).Draw(t, "env")
		hasExpiry := rapid.Bool().Draw(t, "has_expiry")
		orgID := uuid.New()
		credID := uuid.New()
		var storedRecord *domain.CredentialRecord
		repo := &mockCredentialRepo{
			createFn: func(_ context.Context, o uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
				storedRecord = &domain.CredentialRecord{
					ID: credID, OrganizationID: o,
					Name: input.Name, KeyPrefix: input.KeyPrefix,
					Environment: input.Environment, ExpiresAt: input.ExpiresAt,
					CreatedAt: time.Now(),
				}
				return storedRecord, nil
			},
			getByIDFn: func(_ context.Context, o, k uuid.UUID) (*domain.CredentialRecord, error) {
				if storedRecord != nil && o == orgID && k == credID {
					return storedRecord, nil
				}
				return nil, domain.ErrCredentialNotFound
			},
		}
		svc := NewCredentialService(repo, nil, &mockTxManager{}, "test-pepper")
		inp := domain.CreateCredentialInput{Name: name, Environment: env}
		var expectedExpiry *time.Time
		if hasExpiry {
			future := time.Now().Add(24 * time.Hour).Truncate(time.Second)
			expectedExpiry = &future
			inp.ExpiresAt = &future
		}
		_, _, err := svc.Create(context.Background(), orgID, inp)
		if err != nil {
			t.Fatalf("create error: %v", err)
		}
		got, err := svc.GetByID(context.Background(), orgID, credID)
		if err != nil {
			t.Fatalf("get error: %v", err)
		}
		if got.Name != name {
			t.Fatalf("name mismatch: got %q, want %q", got.Name, name)
		}
		if got.Environment != env {
			t.Fatalf("environment mismatch: got %q, want %q", got.Environment, env)
		}
		if hasExpiry {
			if got.ExpiresAt == nil || !got.ExpiresAt.Equal(*expectedExpiry) {
				t.Fatalf("expires_at mismatch: got %v, want %v", got.ExpiresAt, expectedExpiry)
			}
		} else {
			if got.ExpiresAt != nil {
				t.Fatalf("expected nil expires_at, got %v", got.ExpiresAt)
			}
		}
	})
}

// TestProperty_Credential_RotationInvariant verifies P14: rotation invariant.
// Feature: credentials, Property 14: Invariante de rotação
func TestProperty_Credential_RotationInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		env := rapid.SampledFrom([]string{"test", "production"}).Draw(t, "env")
		name := rapid.StringMatching(`[A-Za-z0-9]{1,50}`).Draw(t, "name")
		orgID := uuid.New()
		credID := uuid.New()
		newID := uuid.New()
		existing := &domain.CredentialRecord{
			ID: credID, OrganizationID: orgID,
			Name: name, Environment: env, CreatedAt: time.Now(),
		}
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
								Environment: input.Environment, CreatedAt: time.Now(),
							}, nil
						},
					},
				})
			},
		}
		svc := NewCredentialService(repo, nil, txMgr, "test-pepper")
		newRec, rawKey, err := svc.Rotate(context.Background(), orgID, credID)
		if err != nil {
			t.Fatalf("rotate error: %v", err)
		}
		if newRec.Name != name {
			t.Fatalf("name mismatch after rotation: got %q, want %q", newRec.Name, name)
		}
		if newRec.Environment != env {
			t.Fatalf("environment mismatch after rotation: got %q, want %q", newRec.Environment, env)
		}
		if env == "production" {
			if len(rawKey) < 7 || rawKey[:7] != "I9LIVE_" {
				t.Fatalf("expected I9LIVE_ prefix after rotation, got %q", rawKey)
			}
		} else {
			if len(rawKey) < 7 || rawKey[:7] != "I9TEST_" {
				t.Fatalf("expected I9TEST_ prefix after rotation, got %q", rawKey)
			}
		}
	})
}
