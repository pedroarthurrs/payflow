package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ============================================================
// Unit tests — EffectivePermissions
// ============================================================

func TestEffectivePermissions_RoleDefaults(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	member := &domain.OrganizationMember{
		OrganizationID: orgID,
		UserAccountID:  userID,
		Role:           domain.RoleDeveloper,
	}

	svc := NewPermissionService(
		&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
			return member, nil
		}},
		&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
			return nil, nil
		}},
	)

	perms, err := svc.EffectivePermissions(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults := domain.DefaultPermissionsFor(domain.RoleDeveloper)
	if len(perms) != len(defaults) {
		t.Fatalf("expected %d permissions (role defaults), got %d", len(defaults), len(perms))
	}
}

func TestEffectivePermissions_GrantOverride(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	// support role does not have payments:charge by default
	member := &domain.OrganizationMember{Role: domain.RoleSupport}

	grantedPerm := domain.PermPaymentsCharge

	svc := NewPermissionService(
		&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
			return member, nil
		}},
		&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
			return []domain.MemberPermission{{Permission: grantedPerm, Granted: true}}, nil
		}},
	)

	perms, err := svc.EffectivePermissions(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, p := range perms {
		if p == grantedPerm {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected granted permission %q to be in effective set", grantedPerm)
	}
}

func TestEffectivePermissions_RevokeOverride(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	// admin has customers:delete by default
	member := &domain.OrganizationMember{Role: domain.RoleAdmin}

	revokedPerm := domain.PermCustomersDelete

	svc := NewPermissionService(
		&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
			return member, nil
		}},
		&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
			return []domain.MemberPermission{{Permission: revokedPerm, Granted: false}}, nil
		}},
	)

	perms, err := svc.EffectivePermissions(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, p := range perms {
		if p == revokedPerm {
			t.Fatalf("revoked permission %q should not be in effective set", revokedPerm)
		}
	}
}

// ============================================================
// Unit tests — HasPermission
// ============================================================

func TestHasPermission_RoleDefault_True(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	svc := NewPermissionService(
		&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
			return &domain.OrganizationMember{Role: domain.RoleAdmin}, nil
		}},
		&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
			return nil, nil
		}},
	)

	ok, err := svc.HasPermission(context.Background(), orgID, userID, domain.PermCustomersDelete)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected admin to have customers:delete by default")
	}
}

func TestHasPermission_RoleDefault_False(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	svc := NewPermissionService(
		&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
			return &domain.OrganizationMember{Role: domain.RoleSupport}, nil
		}},
		&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
			return nil, nil
		}},
	)

	ok, err := svc.HasPermission(context.Background(), orgID, userID, domain.PermCustomersDelete)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected support to NOT have customers:delete by default")
	}
}

func TestHasPermission_GrantOverride(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	svc := NewPermissionService(
		&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
			return &domain.OrganizationMember{Role: domain.RoleSupport}, nil
		}},
		&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
			return []domain.MemberPermission{{Permission: domain.PermPaymentsCharge, Granted: true}}, nil
		}},
	)

	ok, err := svc.HasPermission(context.Background(), orgID, userID, domain.PermPaymentsCharge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected grant override to give payments:charge to support")
	}
}

func TestHasPermission_RevokeOverride(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	svc := NewPermissionService(
		&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
			return &domain.OrganizationMember{Role: domain.RoleAdmin}, nil
		}},
		&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
			return []domain.MemberPermission{{Permission: domain.PermCustomersDelete, Granted: false}}, nil
		}},
	)

	ok, err := svc.HasPermission(context.Background(), orgID, userID, domain.PermCustomersDelete)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected revoke override to remove customers:delete from admin")
	}
}

func TestHasPermission_MemberNotFound(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	svc := NewPermissionService(
		&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
			return nil, errors.New("not found")
		}},
		&mockPermissionRepo{},
	)

	_, err := svc.HasPermission(context.Background(), orgID, userID, domain.PermCustomersRead)
	if err == nil {
		t.Fatal("expected error when member not found")
	}
}

// ============================================================
// Property-based tests
// ============================================================

// Feature: user-permissions, Property 1:
// For any member, EffectivePermissions returns exactly the role defaults ± overrides.
func TestProperty_EffectivePermissions_MatchesFormula(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()
		userID := uuid.New()
		role := rapid.SampledFrom([]domain.Role{
			domain.RoleOwner, domain.RoleAdmin, domain.RoleDeveloper, domain.RoleSupport,
		}).Draw(t, "role")

		allPerms := []domain.Permission{
			domain.PermUsersRead, domain.PermUsersInvite, domain.PermUsersManage,
			domain.PermCustomersRead, domain.PermCustomersWrite, domain.PermCustomersDelete,
			domain.PermPaymentsRead, domain.PermPaymentsCharge, domain.PermPaymentsRefund,
		}

		// Generate a random set of overrides
		overrideCount := rapid.IntRange(0, 4).Draw(t, "overrideCount")
		overrides := make([]domain.MemberPermission, overrideCount)
		usedPerms := map[domain.Permission]bool{}
		for i := range overrides {
			var perm domain.Permission
			for {
				perm = rapid.SampledFrom(allPerms).Draw(t, "overridePerm")
				if !usedPerms[perm] {
					break
				}
			}
			usedPerms[perm] = true
			overrides[i] = domain.MemberPermission{
				Permission: perm,
				Granted:    rapid.Bool().Draw(t, "granted"),
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
		}

		svc := NewPermissionService(
			&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
				return &domain.OrganizationMember{Role: role}, nil
			}},
			&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
				return overrides, nil
			}},
		)

		// Compute expected set manually
		expected := make(map[domain.Permission]bool)
		for _, p := range domain.DefaultPermissionsFor(role) {
			expected[p] = true
		}
		for _, o := range overrides {
			expected[o.Permission] = o.Granted
		}

		got, err := svc.EffectivePermissions(context.Background(), orgID, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify got matches expected
		gotSet := make(map[domain.Permission]bool, len(got))
		for _, p := range got {
			gotSet[p] = true
		}

		for p, want := range expected {
			if want && !gotSet[p] {
				t.Fatalf("permission %q should be in effective set but is missing", p)
			}
			if !want && gotSet[p] {
				t.Fatalf("permission %q should NOT be in effective set but is present", p)
			}
		}
		for p := range gotSet {
			if !expected[p] {
				t.Fatalf("unexpected permission %q in effective set", p)
			}
		}
	})
}

// Feature: user-permissions, Property 2:
// A grant override for a permission not in the role default results in HasPermission=true.
func TestProperty_GrantOverride_AddsPermission(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		role := rapid.SampledFrom([]domain.Role{domain.RoleSupport, domain.RoleDeveloper}).Draw(t, "role")

		// Find a permission NOT in the role's defaults to grant
		allPerms := []domain.Permission{
			domain.PermUsersManage, domain.PermPaymentsCharge, domain.PermPaymentsRefund,
			domain.PermCredentialsManage, domain.PermCustomersDelete,
		}
		var targetPerm domain.Permission
		for _, p := range allPerms {
			if !domain.HasDefaultPermission(role, p) {
				targetPerm = p
				break
			}
		}
		if targetPerm == "" {
			return // no non-default permission found for this role, skip
		}

		orgID := uuid.New()
		userID := uuid.New()

		svc := NewPermissionService(
			&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
				return &domain.OrganizationMember{Role: role}, nil
			}},
			&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
				return []domain.MemberPermission{{Permission: targetPerm, Granted: true}}, nil
			}},
		)

		ok, err := svc.HasPermission(context.Background(), orgID, userID, targetPerm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatalf("grant override must result in HasPermission=true for %q (role %s)", targetPerm, role)
		}
	})
}

// Feature: user-permissions, Property 3:
// A revoke override for a permission in the role default results in HasPermission=false.
func TestProperty_RevokeOverride_RemovesPermission(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		role := rapid.SampledFrom([]domain.Role{domain.RoleOwner, domain.RoleAdmin}).Draw(t, "role")

		// Find a permission in the role's defaults to revoke
		defaults := domain.DefaultPermissionsFor(role)
		if len(defaults) == 0 {
			return
		}
		targetPerm := defaults[0]

		orgID := uuid.New()
		userID := uuid.New()

		svc := NewPermissionService(
			&mockMemberRepo{getMemberFn: func(_ context.Context, _, _ uuid.UUID) (*domain.OrganizationMember, error) {
				return &domain.OrganizationMember{Role: role}, nil
			}},
			&mockPermissionRepo{listForMemberFn: func(_ context.Context, _ uuid.UUID) ([]domain.MemberPermission, error) {
				return []domain.MemberPermission{{Permission: targetPerm, Granted: false}}, nil
			}},
		)

		ok, err := svc.HasPermission(context.Background(), orgID, userID, targetPerm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatalf("revoke override must result in HasPermission=false for %q (role %s)", targetPerm, role)
		}
	})
}

// Feature: user-permissions, Property 7:
// EffectivePermissions of org A never mixes overrides from org B.
func TestProperty_PermissionOrgIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgA := uuid.New()
		orgB := uuid.New()
		userID := uuid.New()

		role := domain.RoleAdmin

		// Each org has a distinct member ID so the mock can discriminate.
		memberA := uuid.New()
		memberB := uuid.New()

		// org B has a revoke override for customers:read
		svc := NewPermissionService(
			&mockMemberRepo{getMemberFn: func(_ context.Context, orgID uuid.UUID, _ uuid.UUID) (*domain.OrganizationMember, error) {
				id := memberA
				if orgID == orgB {
					id = memberB
				}
				return &domain.OrganizationMember{ID: id, OrganizationID: orgID, Role: role}, nil
			}},
			&mockPermissionRepo{listForMemberFn: func(_ context.Context, memberID uuid.UUID) ([]domain.MemberPermission, error) {
				if memberID == memberB {
					return []domain.MemberPermission{{Permission: domain.PermCustomersRead, Granted: false}}, nil
				}
				return nil, nil
			}},
		)

		permsA, err := svc.EffectivePermissions(context.Background(), orgA, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, p := range permsA {
			_ = p // just verify no panic and no revoke from org B leaked
		}

		// org A should still have customers:read (admin default)
		ok, err := svc.HasPermission(context.Background(), orgA, userID, domain.PermCustomersRead)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("org B revoke should not affect org A permissions (isolation violation)")
		}
	})
}
