package service

import (
	"context"
	"log/slog"

	"i9pay/internal/crypto"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// CredentialService handles credential business logic.
type CredentialService struct {
	repo         domain.CredentialRepository
	txMgr        domain.TxManager
	keyGenerator *crypto.KeyGenerator
	hasher       *crypto.Hasher
	pepper       string
}

// NewCredentialService creates a CredentialService.
// Panics if pepper is empty — the application must not start without this security secret.
func NewCredentialService(repo domain.CredentialRepository, txMgr domain.TxManager, pepper string) *CredentialService {
	if pepper == "" {
		panic("CredentialService: pepper must not be empty")
	}
	return &CredentialService{
		repo:         repo,
		txMgr:        txMgr,
		keyGenerator: crypto.NewKeyGenerator(),
		hasher:       crypto.NewHasher(),
		pepper:       pepper,
	}
}

// Create generates a new credential for the given organization.
// Returns the persisted record and the raw key (only available in memory, never stored).
func (s *CredentialService) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, string, error) {
	rawKey, err := s.keyGenerator.GenerateAPIKey(input.Environment)
	if err != nil {
		return nil, "", err
	}
	keyHash := s.hasher.SHA256WithPepper(s.pepper, rawKey)
	keyPrefix := rawKey[:10]

	record, err := s.repo.Create(ctx, orgID, domain.CreateCredentialInput{
		Name:        input.Name,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Environment: input.Environment,
		ExpiresAt:   input.ExpiresAt,
	})
	if err != nil {
		return nil, "", err
	}
	return record, rawKey, nil
}

// List returns all active credentials for the given organization.
func (s *CredentialService) List(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.CredentialRecord, error) {
	return s.repo.List(ctx, orgID, params)
}

// GetByID returns a single credential by ID for the given organization.
func (s *CredentialService) GetByID(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, error) {
	return s.repo.GetByID(ctx, orgID, credentialID)
}

// Revoke soft-deletes a credential for the given organization.
func (s *CredentialService) Revoke(ctx context.Context, orgID, credentialID uuid.UUID) error {
	return s.repo.SoftDelete(ctx, orgID, credentialID)
}

// Rotate atomically revokes the existing credential and creates a new one with the same metadata.
// Returns the new record and the new raw key (only available in memory, never stored).
func (s *CredentialService) Rotate(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, string, error) {
	existing, err := s.repo.GetByID(ctx, orgID, credentialID)
	if err != nil {
		return nil, "", err
	}

	var newRecord *domain.CredentialRecord
	var newRawKey string

	if err := s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		if err := tx.CredentialRepo().SoftDelete(ctx, orgID, credentialID); err != nil {
			slog.Error("rotate: failed to soft-delete old credential", "credential_id", credentialID, "error", err)
			return err
		}

		rawKey, err := s.keyGenerator.GenerateAPIKey(existing.Environment)
		if err != nil {
			return err
		}

		keyHash := s.hasher.SHA256WithPepper(s.pepper, rawKey)
		keyPrefix := rawKey[:10]

		record, err := tx.CredentialRepo().Create(ctx, orgID, domain.CreateCredentialInput{
			Name:        existing.Name,
			KeyHash:     keyHash,
			KeyPrefix:   keyPrefix,
			Environment: existing.Environment,
			ExpiresAt:   existing.ExpiresAt,
		})
		if err != nil {
			slog.Error("rotate: failed to create new credential", "org_id", orgID, "error", err)
			return err
		}

		newRecord = record
		newRawKey = rawKey
		return nil
	}); err != nil {
		return nil, "", err
	}

	return newRecord, newRawKey, nil
}
