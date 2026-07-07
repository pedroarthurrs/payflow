package service

import (
	"context"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// PermissionService resolves effective permissions for organization members,
// combining role defaults with individual per-user overrides.
type PermissionService struct {
	members     domain.MemberRepository
	permissions domain.PermissionRepository
}

// NewPermissionService creates a PermissionService.
func NewPermissionService(members domain.MemberRepository, permissions domain.PermissionRepository) *PermissionService {
	return &PermissionService{members: members, permissions: permissions}
}

// EffectivePermissions resolves the full permission set for a user in an org.
// Result = DefaultPermissionsFor(role) ± individual overrides from member_permissions.
func (s *PermissionService) EffectivePermissions(ctx context.Context, orgID, userAccountID uuid.UUID) ([]domain.Permission, error) {
	member, err := s.members.GetMember(ctx, orgID, userAccountID)
	if err != nil {
		return nil, err
	}

	// Start with role defaults.
	defaults := domain.DefaultPermissionsFor(member.Role)
	effective := make(map[domain.Permission]bool, len(defaults))
	for _, p := range defaults {
		effective[p] = true
	}

	// Apply individual overrides.
	overrides, err := s.permissions.ListForMember(ctx, member.ID)
	if err != nil {
		return nil, err
	}
	for _, o := range overrides {
		effective[o.Permission] = o.Granted
	}

	result := make([]domain.Permission, 0, len(effective))
	for p, granted := range effective {
		if granted {
			result = append(result, p)
		}
	}
	return result, nil
}

// HasPermission checks if a user has a specific effective permission in an org.
// Overrides take precedence over role defaults.
func (s *PermissionService) HasPermission(ctx context.Context, orgID, userAccountID uuid.UUID, perm domain.Permission) (bool, error) {
	member, err := s.members.GetMember(ctx, orgID, userAccountID)
	if err != nil {
		return false, err
	}

	// Check for an individual override first (highest precedence).
	overrides, err := s.permissions.ListForMember(ctx, member.ID)
	if err != nil {
		return false, err
	}
	for _, o := range overrides {
		if o.Permission == perm {
			return o.Granted, nil
		}
	}

	// Fall back to role default.
	return domain.HasDefaultPermission(member.Role, perm), nil
}

// SetPermission creates or updates a permission override for a member.
func (s *PermissionService) SetPermission(ctx context.Context, orgID, userAccountID uuid.UUID, perm domain.Permission, granted bool) error {
	member, err := s.members.GetMember(ctx, orgID, userAccountID)
	if err != nil {
		return err
	}
	return s.permissions.Upsert(ctx, member.ID, perm, granted)
}

// RemovePermission deletes a permission override, reverting the member to the role default.
func (s *PermissionService) RemovePermission(ctx context.Context, orgID, userAccountID uuid.UUID, perm domain.Permission) error {
	member, err := s.members.GetMember(ctx, orgID, userAccountID)
	if err != nil {
		return err
	}
	return s.permissions.Delete(ctx, member.ID, perm)
}
