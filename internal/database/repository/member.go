package repository

import (
	"context"
	"database/sql"
	"errors"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// MemberRepo implements domain.MemberRepository backed by PostgreSQL.
type MemberRepo struct {
	db *sql.DB
}

// NewMemberRepo creates a MemberRepo.
func NewMemberRepo(db *sql.DB) *MemberRepo {
	return &MemberRepo{db: db}
}

var _ domain.MemberRepository = (*MemberRepo)(nil)

// AddMember inserts a new organization_members row and returns the created membership.
func (r *MemberRepo) AddMember(ctx context.Context, orgID, userAccountID uuid.UUID, role domain.Role, invitedBy *uuid.UUID) (*domain.OrganizationMember, error) {
	const query = `
		INSERT INTO organization_members (organization_id, user_id, role, invited_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, organization_id, user_id, role, joined_at, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, orgID, userAccountID, string(role), invitedBy)
	var m domain.OrganizationMember
	if err := row.Scan(&m.ID, &m.OrganizationID, &m.UserAccountID, &m.Role, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	// Populate denormalized fields.
	_ = r.db.QueryRowContext(ctx,
		`SELECT email, name FROM "user" WHERE id = $1`, userAccountID,
	).Scan(&m.Email, &m.Name)
	return &m, nil
}

// GetMember returns a single active membership by org + user account IDs.
func (r *MemberRepo) GetMember(ctx context.Context, orgID, userAccountID uuid.UUID) (*domain.OrganizationMember, error) {
	const query = `
		SELECT om.id, om.organization_id, om.user_id, ua.email, ua.name,
		       om.role, om.joined_at, om.created_at, om.updated_at
		FROM organization_members om
		JOIN "user" ua ON ua.id = om.user_id
		WHERE om.organization_id = $1 AND om.user_id = $2 AND om.deleted_at IS NULL`

	row := r.db.QueryRowContext(ctx, query, orgID, userAccountID)
	var m domain.OrganizationMember
	if err := row.Scan(&m.ID, &m.OrganizationID, &m.UserAccountID, &m.Email, &m.Name,
		&m.Role, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// ListMembers returns a paginated list of active members for the given organization
// along with the total count.
func (r *MemberRepo) ListMembers(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.OrganizationMember, int32, error) {
	const countQuery = `SELECT COUNT(*) FROM organization_members WHERE organization_id = $1 AND deleted_at IS NULL`
	var total int32
	if err := r.db.QueryRowContext(ctx, countQuery, orgID).Scan(&total); err != nil {
		return nil, 0, err
	}

	const query = `
		SELECT om.id, om.organization_id, om.user_id, ua.email, ua.name,
		       om.role, om.joined_at, om.created_at, om.updated_at
		FROM organization_members om
		JOIN "user" ua ON ua.id = om.user_id
		WHERE om.organization_id = $1 AND om.deleted_at IS NULL
		ORDER BY om.joined_at ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, orgID, params.Limit, params.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var members []domain.OrganizationMember
	for rows.Next() {
		var m domain.OrganizationMember
		if err := rows.Scan(&m.ID, &m.OrganizationID, &m.UserAccountID, &m.Email, &m.Name,
			&m.Role, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, err
		}
		members = append(members, m)
	}
	return members, total, rows.Err()
}

// UpdateMemberRole changes the role of an existing member.
func (r *MemberRepo) UpdateMemberRole(ctx context.Context, orgID, userAccountID uuid.UUID, role domain.Role) (*domain.OrganizationMember, error) {
	const query = `
		UPDATE organization_members SET role = $1, updated_at = NOW()
		WHERE organization_id = $2 AND user_id = $3 AND deleted_at IS NULL
		RETURNING id, organization_id, user_id, role, joined_at, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, string(role), orgID, userAccountID)
	var m domain.OrganizationMember
	if err := row.Scan(&m.ID, &m.OrganizationID, &m.UserAccountID, &m.Role, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	_ = r.db.QueryRowContext(ctx,
		`SELECT email, name FROM "user" WHERE id = $1`, userAccountID,
	).Scan(&m.Email, &m.Name)
	return &m, nil
}

// RemoveMember soft-deletes a membership by setting deleted_at.
func (r *MemberRepo) RemoveMember(ctx context.Context, orgID, userAccountID uuid.UUID) error {
	const query = `
		UPDATE organization_members SET deleted_at = NOW(), updated_at = NOW()
		WHERE organization_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	res, err := r.db.ExecContext(ctx, query, orgID, userAccountID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListOrganizationsForUser returns lightweight org references for all active memberships.
func (r *MemberRepo) ListOrganizationsForUser(ctx context.Context, userAccountID uuid.UUID) ([]domain.OrganizationRef, error) {
	const query = `
		SELECT o.id, o.name
		FROM organization_members om
		JOIN organizations o ON o.id = om.organization_id
		WHERE om.user_id = $1 AND om.deleted_at IS NULL AND o.deleted_at IS NULL`

	rows, err := r.db.QueryContext(ctx, query, userAccountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var orgs []domain.OrganizationRef
	for rows.Next() {
		var o domain.OrganizationRef
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// UserAccountRepo implements domain.UserAccountRepository.
type UserAccountRepo struct {
	db *sql.DB
}

// NewUserAccountRepo creates a UserAccountRepo.
func NewUserAccountRepo(db *sql.DB) *UserAccountRepo {
	return &UserAccountRepo{db: db}
}

var _ domain.UserAccountRepository = (*UserAccountRepo)(nil)

// Create inserts a new global user account and returns it.
func (r *UserAccountRepo) Create(ctx context.Context, input domain.CreateUserAccountInput) (*domain.UserAccount, error) {
	const query = `
		INSERT INTO "user" (email, name, password)
		VALUES ($1, $2, $3)
		RETURNING id, email, name, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, input.Email, input.Name, input.PasswordHash)
	var a domain.UserAccount
	if err := row.Scan(&a.ID, &a.Email, &a.Name, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

// GetByID returns a user account by its UUID.
func (r *UserAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.UserAccount, error) {
	const query = `
		SELECT id, email, name, created_at, updated_at
		FROM "user" WHERE id = $1 AND deleted_at IS NULL`

	row := r.db.QueryRowContext(ctx, query, id)
	var a domain.UserAccount
	if err := row.Scan(&a.ID, &a.Email, &a.Name, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// GetByEmail returns a user account by email address.
func (r *UserAccountRepo) GetByEmail(ctx context.Context, email string) (*domain.UserAccount, error) {
	const query = `
		SELECT id, email, name, created_at, updated_at
		FROM "user" WHERE email = $1 AND deleted_at IS NULL`

	row := r.db.QueryRowContext(ctx, query, email)
	var a domain.UserAccount
	if err := row.Scan(&a.ID, &a.Email, &a.Name, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// Update sets the display name of a user account.
func (r *UserAccountRepo) Update(ctx context.Context, id uuid.UUID, input domain.UpdateUserAccountInput) (*domain.UserAccount, error) {
	const query = `
		UPDATE "user" SET name = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, email, name, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, input.Name, id)
	var a domain.UserAccount
	if err := row.Scan(&a.ID, &a.Email, &a.Name, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// SoftDelete sets deleted_at on a user account.
func (r *UserAccountRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE "user" SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
