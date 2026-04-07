package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	organizationdb "i9pay/internal/database/dbgen/organization"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// OrganizationSecretRepo implements domain.OrganizationSecretReader backed by the organization SQLC queries.
type OrganizationSecretRepo struct {
	q organizationdb.Querier
}

// NewOrganizationSecretReader creates an OrganizationSecretRepo.
func NewOrganizationSecretReader(q organizationdb.Querier) *OrganizationSecretRepo {
	return &OrganizationSecretRepo{q: q}
}

// GetWebhookSecret retrieves the webhook_secret for the given organization.
func (r *OrganizationSecretRepo) GetWebhookSecret(ctx context.Context, orgID uuid.UUID) (string, error) {
	org, err := r.q.GetOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("organization not found")
		}
		slog.Error("failed to get organization webhook secret", "organization_id", orgID, "error", err)
		return "", domain.ErrInternal
	}
	if !org.WebhookSecret.Valid {
		return "", fmt.Errorf("organization has no webhook secret configured")
	}
	return org.WebhookSecret.String, nil
}

var _ domain.OrganizationSecretReader = (*OrganizationSecretRepo)(nil)
