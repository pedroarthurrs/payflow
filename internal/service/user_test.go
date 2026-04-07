package service

import (
	"context"
	"errors"
	"testing"

	"i9pay/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"
)

// ============================================================
// Property-based tests
// ============================================================

// TestProperty_UserMultiOrganizationIsolation verifies that for any List call with organizationID_A,
// no user belonging to organizationID_B appears in results.
func TestProperty_UserMultiOrganizationIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationA := uuid.New()
		organizationB := uuid.New()

		// Generate 1-5 users for organization A and 1-5 for organization B
		countA := rapid.IntRange(1, 5).Draw(t, "countA")
		countB := rapid.IntRange(1, 5).Draw(t, "countB")

		usersA := make([]domain.User, countA)
		for i := range usersA {
			usersA[i] = domain.User{
				ID:             uuid.New().String(),
				OrganizationID: organizationA.String(),
				Email:          rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "emailA"),
				Name:           "User A",
				Role:           "member",
			}
		}

		usersB := make([]domain.User, countB)
		for i := range usersB {
			usersB[i] = domain.User{
				ID:             uuid.New().String(),
				OrganizationID: organizationB.String(),
				Email:          rapid.StringMatching(`[a-z]{4,8}@other\.com`).Draw(t, "emailB"),
				Name:           "User B",
				Role:           "member",
			}
		}

		users := &mockUserRepo{
			listFn: func(_ context.Context, organizationID uuid.UUID, _ domain.ListParams) ([]domain.User, error) {
				if organizationID == organizationA {
					return usersA, nil
				}
				if organizationID == organizationB {
					return usersB, nil
				}
				return nil, nil
			},
		}

		svc := NewUserService(users)

		results, err := svc.List(context.Background(), organizationA.String(), domain.ListParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, u := range results {
			if u.OrganizationID != organizationA.String() {
				t.Fatalf("isolation violation: got user with organizationID=%s, expected %s", u.OrganizationID, organizationA)
			}
		}

		if len(results) != countA {
			t.Fatalf("expected %d users for organizationA, got %d", countA, len(results))
		}
	})
}

// TestProperty_UserListNoPassword verifies that for any List call,
// no domain.User in the result has a non-empty Password field.
func TestProperty_UserListNoPassword(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		count := rapid.IntRange(1, 3).Draw(t, "count")

		domainUsers := make([]domain.User, count)
		for i := range domainUsers {
			domainUsers[i] = domain.User{
				ID:             uuid.New().String(),
				OrganizationID: organizationID.String(),
				Email:          rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "email"),
				Name:           rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "name"),
				Role:           "member",
			}
		}

		users := &mockUserRepo{
			listFn: func(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.User, error) {
				return domainUsers, nil
			},
		}

		svc := NewUserService(users)
		results, err := svc.List(context.Background(), organizationID.String(), domain.ListParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// domain.User has no Password field - this verifies the mapping never leaks it.
		// The struct itself enforces this at compile time; we verify count and content.
		if len(results) != count {
			t.Fatalf("expected %d users, got %d", count, len(results))
		}
		for _, u := range results {
			// Verify the returned user is a domain.User (no Password field by design).
			if u.ID == "" {
				t.Fatal("user ID must not be empty")
			}
			if u.Email == "" {
				t.Fatal("user Email must not be empty")
			}
			// domain.User struct has no Password field - confirmed by type system.
		}
	})
}

// TestProperty_UserCreationRoundTrip verifies that for any valid CreateUserRequest
// (role != owner), after Create succeeds, the returned domain.User must have the
// same email and role as the request, and be linked to the correct organizationID.
func TestProperty_UserCreationRoundTrip(t *testing.T) {
	// Use a fixed password - bcrypt cost is not what this property tests.
	const fixedPassword = "testpassword1"

	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		email := rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "email")
		name := rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "name")
		role := rapid.SampledFrom([]domain.Role{domain.RoleAdmin, domain.RoleMember}).Draw(t, "role")

		newUserID := uuid.New()

		users := &mockUserRepo{
			getByEmailAndOrgFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.User, error) {
				return nil, errors.New("not found")
			},
			createFn: func(_ context.Context, orgID uuid.UUID, input domain.CreateUserInput) (*domain.User, error) {
				if err := bcrypt.CompareHashAndPassword([]byte(input.PasswordHash), []byte(fixedPassword)); err != nil {
					return nil, errors.New("invalid bcrypt hash in CreateUser call")
				}
				return &domain.User{
					ID:             newUserID.String(),
					OrganizationID: orgID.String(),
					Email:          input.Email,
					Name:           input.Name,
					Role:           domain.Role(input.Role),
				}, nil
			},
		}

		svc := newUserServiceWithCost(users, bcrypt.MinCost)
		result, err := svc.Create(context.Background(), organizationID.String(), CreateUserRequest{
			Name:     name,
			Email:    email,
			Password: fixedPassword,
			Role:     role,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Email != email {
			t.Fatalf("expected email=%q, got %q", email, result.Email)
		}
		if result.Role != role {
			t.Fatalf("expected role=%q, got %q", role, result.Role)
		}
		if result.OrganizationID != organizationID.String() {
			t.Fatalf("expected organizationID=%q, got %q", organizationID, result.OrganizationID)
		}
		if result.ID == "" {
			t.Fatal("expected non-empty user ID")
		}
	})
}
