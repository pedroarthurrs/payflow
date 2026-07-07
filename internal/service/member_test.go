package service

import (
	"context"
	"errors"
	"testing"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ============================================================
// Unit tests — AddMember
// ============================================================

func TestAddMember_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	expected := &domain.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserAccountID:  userID,
		Role:           domain.RoleDeveloper,
	}

	svc := NewMemberService(&mockMemberRepo{
		addMemberFn: func(_ context.Context, _, _ uuid.UUID, _ domain.Role, _ *uuid.UUID) (*domain.OrganizationMember, error) {
			return expected, nil
		},
	})

	result, err := svc.AddMember(context.Background(), orgID, userID, domain.RoleDeveloper, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Role != domain.RoleDeveloper {
		t.Fatalf("expected role developer, got %s", result.Role)
	}
}

func TestAddMember_SecondOwnerForbidden(t *testing.T) {
	orgID := uuid.New()
	existingOwner := uuid.New()
	newUser := uuid.New()

	svc := NewMemberService(&mockMemberRepo{
		listMembersFn: func(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.OrganizationMember, int32, error) {
			return []domain.OrganizationMember{
				{UserAccountID: existingOwner, Role: domain.RoleOwner},
			}, 1, nil
		},
	})

	_, err := svc.AddMember(context.Background(), orgID, newUser, domain.RoleOwner, nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when adding second owner, got %v", err)
	}
}

// ============================================================
// Unit tests — UpdateMemberRole
// ============================================================

func TestUpdateMemberRole_SecondOwnerForbidden(t *testing.T) {
	orgID := uuid.New()
	existingOwnerID := uuid.New()
	memberToPromote := uuid.New()

	svc := NewMemberService(&mockMemberRepo{
		listMembersFn: func(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.OrganizationMember, int32, error) {
			return []domain.OrganizationMember{
				{UserAccountID: existingOwnerID, Role: domain.RoleOwner},
			}, 1, nil
		},
	})

	_, err := svc.UpdateMemberRole(context.Background(), orgID, memberToPromote, domain.RoleOwner)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when promoting to owner with existing owner, got %v", err)
	}
}

func TestUpdateMemberRole_SameUserOwnerAllowed(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	svc := NewMemberService(&mockMemberRepo{
		listMembersFn: func(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.OrganizationMember, int32, error) {
			// The same user is already owner
			return []domain.OrganizationMember{
				{UserAccountID: userID, Role: domain.RoleOwner},
			}, 1, nil
		},
		updateMemberRoleFn: func(_ context.Context, _, _ uuid.UUID, role domain.Role) (*domain.OrganizationMember, error) {
			return &domain.OrganizationMember{UserAccountID: userID, Role: role}, nil
		},
	})

	result, err := svc.UpdateMemberRole(context.Background(), orgID, userID, domain.RoleOwner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Role != domain.RoleOwner {
		t.Fatalf("expected owner role, got %s", result.Role)
	}
}

// ============================================================
// Unit tests — ListOrganizationsForUser
// ============================================================

func TestListOrganizationsForUser_ReturnsAll(t *testing.T) {
	userID := uuid.New()

	expectedOrgs := []domain.OrganizationRef{
		{ID: uuid.New(), Name: "Org A"},
		{ID: uuid.New(), Name: "Org B"},
	}

	svc := NewMemberService(&mockMemberRepo{
		listOrgsForUserFn: func(_ context.Context, _ uuid.UUID) ([]domain.OrganizationRef, error) {
			return expectedOrgs, nil
		},
	})

	orgs, err := svc.ListOrganizationsForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orgs) != 2 {
		t.Fatalf("expected 2 organizations, got %d", len(orgs))
	}
}

// ============================================================
// Property-based tests
// ============================================================

// Feature: multi-organization, Property:
// A user account can belong to multiple organizations — ListOrganizationsForUser
// returns exactly the set of orgs the repository reports.
func TestProperty_MultiOrgMembership(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := uuid.New()
		count := rapid.IntRange(1, 5).Draw(t, "orgCount")

		orgs := make([]domain.OrganizationRef, count)
		for i := range orgs {
			orgs[i] = domain.OrganizationRef{ID: uuid.New(), Name: rapid.StringMatching(`[A-Z][a-z]{3,12}`).Draw(t, "orgName")}
		}

		svc := NewMemberService(&mockMemberRepo{
			listOrgsForUserFn: func(_ context.Context, _ uuid.UUID) ([]domain.OrganizationRef, error) {
				return orgs, nil
			},
		})

		result, err := svc.ListOrganizationsForUser(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != count {
			t.Fatalf("expected %d orgs, got %d", count, len(result))
		}
	})
}
