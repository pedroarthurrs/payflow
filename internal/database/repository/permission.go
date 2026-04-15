package repository

import (
	"context"
	"database/sql"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// PermissionRepo implements domain.PermissionRepository.
type PermissionRepo struct {
	db *sql.DB
}

// NewPermissionRepo creates a PermissionRepo.
func NewPermissionRepo(db *sql.DB) *PermissionRepo {
	return &PermissionRepo{db: db}
}

var _ domain.PermissionRepository = (*PermissionRepo)(nil)

// Upsert inserts or updates a permission override for an organization member.
func (r *PermissionRepo) Upsert(ctx context.Context, memberID uuid.UUID, permission domain.Permission, granted bool) error {
	const query = `
		INSERT INTO member_permissions (organization_id, member_id, permission, granted)
		VALUES ((SELECT organization_id FROM organization_members WHERE id = $1), $1, $2, $3)
		ON CONFLICT (member_id, permission)
		DO UPDATE SET granted = EXCLUDED.granted, updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, memberID, string(permission), granted)
	return err
}

// Delete removes a permission override, reverting that permission to its role default.
func (r *PermissionRepo) Delete(ctx context.Context, memberID uuid.UUID, permission domain.Permission) error {
	const query = `
		DELETE FROM member_permissions
		WHERE member_id = $1 AND permission = $2`
	res, err := r.db.ExecContext(ctx, query, memberID, string(permission))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrPermissionNotFound
	}
	return nil
}

// ListForMember returns all permission overrides stored for an organization member.
func (r *PermissionRepo) ListForMember(ctx context.Context, memberID uuid.UUID) ([]domain.MemberPermission, error) {
	const query = `
		SELECT id, organization_id, member_id, permission, granted, created_at, updated_at
		FROM member_permissions
		WHERE member_id = $1`

	rows, err := r.db.QueryContext(ctx, query, memberID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var perms []domain.MemberPermission
	for rows.Next() {
		var p domain.MemberPermission
		var perm string
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.MemberID, &perm, &p.Granted, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Permission = domain.Permission(perm)
		perms = append(perms, p)
	}
	return perms, rows.Err()
}
