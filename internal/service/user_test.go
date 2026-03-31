package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"
)

// ============================================================
// userMockQuerier — full Querier implementation for UserService tests
// ============================================================

type userMockQuerier struct {
	getUserByEmailFn    func(ctx context.Context, arg dbgen.GetUserByEmailAndOrganizationParams) (dbgen.User, error)
	getUserByIDFn       func(ctx context.Context, arg dbgen.GetUserByIDParams) (dbgen.User, error)
	listUsersByOrganizationFn func(ctx context.Context, organizationID uuid.UUID) ([]dbgen.User, error)
	createUserFn        func(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.User, error)
	softDeleteUserFn    func(ctx context.Context, arg dbgen.SoftDeleteUserParams) error
	updateUserFn        func(ctx context.Context, arg dbgen.UpdateUserParams) (dbgen.User, error)
}

func (m *userMockQuerier) GetUserByEmailAndOrganization(ctx context.Context, arg dbgen.GetUserByEmailAndOrganizationParams) (dbgen.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, arg)
	}
	return dbgen.User{}, errors.New("not found")
}

func (m *userMockQuerier) GetUserByID(ctx context.Context, arg dbgen.GetUserByIDParams) (dbgen.User, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, arg)
	}
	return dbgen.User{}, sql.ErrNoRows
}

func (m *userMockQuerier) ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]dbgen.User, error) {
	if m.listUsersByOrganizationFn != nil {
		return m.listUsersByOrganizationFn(ctx, organizationID)
	}
	return nil, nil
}

func (m *userMockQuerier) CreateUser(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, arg)
	}
	return dbgen.User{}, nil
}

func (m *userMockQuerier) SoftDeleteUser(ctx context.Context, arg dbgen.SoftDeleteUserParams) error {
	if m.softDeleteUserFn != nil {
		return m.softDeleteUserFn(ctx, arg)
	}
	return nil
}

func (m *userMockQuerier) UpdateUser(ctx context.Context, arg dbgen.UpdateUserParams) (dbgen.User, error) {
	if m.updateUserFn != nil {
		return m.updateUserFn(ctx, arg)
	}
	return dbgen.User{}, nil
}

// Stub implementations for the remaining Querier methods.
func (m *userMockQuerier) CreateAddress(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
	return dbgen.Address{}, nil
}
func (m *userMockQuerier) CreateCustomer(ctx context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
	return dbgen.Customer{}, nil
}
func (m *userMockQuerier) DeleteAddress(ctx context.Context, arg dbgen.DeleteAddressParams) error {
	return nil
}
func (m *userMockQuerier) GetAPIKeyByHash(ctx context.Context, keyHash string) (dbgen.ApiKey, error) {
	return dbgen.ApiKey{}, errors.New("not found")
}
func (m *userMockQuerier) GetAddressByID(ctx context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
	return dbgen.Address{}, nil
}
func (m *userMockQuerier) GetCustomerByID(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
	return dbgen.Customer{}, nil
}
func (m *userMockQuerier) GetOrganization(ctx context.Context, id uuid.UUID) (dbgen.Organization, error) {
	return dbgen.Organization{}, nil
}
func (m *userMockQuerier) HardDeleteCustomer(ctx context.Context, arg dbgen.HardDeleteCustomerParams) error {
	return nil
}
func (m *userMockQuerier) ListAddressesByCustomer(ctx context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
	return nil, nil
}
func (m *userMockQuerier) ListCustomersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]dbgen.Customer, error) {
	return nil, nil
}
func (m *userMockQuerier) ListOrganizations(ctx context.Context) ([]dbgen.Organization, error) {
	return nil, nil
}
func (m *userMockQuerier) SoftDeleteCustomer(ctx context.Context, arg dbgen.SoftDeleteCustomerParams) error {
	return nil
}
func (m *userMockQuerier) UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *userMockQuerier) GetUserByEmail(ctx context.Context, email string) (dbgen.User, error) {
	return dbgen.User{}, errors.New("not found")
}
func (m *userMockQuerier) UpdateAddress(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
	return dbgen.Address{}, nil
}
func (m *userMockQuerier) UpdateCustomer(ctx context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
	return dbgen.Customer{}, nil
}

// Ensure userMockQuerier satisfies the interface at compile time.
var _ dbgen.Querier = (*userMockQuerier)(nil)

// ============================================================
// Property-based tests
// ============================================================

// TestProperty_UserMultiOrganizationIsolation verifies that for any List call with organizationID_A,
// no user belonging to organizationID_B appears in results.
func TestProperty_UserMultiOrganizationIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationA := uuid.New()
		organizationB := uuid.New()

		// Generate 1–5 users for organization A and 1–5 for organization B
		countA := rapid.IntRange(1, 5).Draw(t, "countA")
		countB := rapid.IntRange(1, 5).Draw(t, "countB")

		usersA := make([]dbgen.User, countA)
		for i := range usersA {
			usersA[i] = dbgen.User{
				ID:       uuid.New(),
				OrganizationID: organizationA,
				Email:    rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "emailA"),
				Name:     "User A",
				Role:     "member",
			}
		}

		usersB := make([]dbgen.User, countB)
		for i := range usersB {
			usersB[i] = dbgen.User{
				ID:       uuid.New(),
				OrganizationID: organizationB,
				Email:    rapid.StringMatching(`[a-z]{4,8}@other\.com`).Draw(t, "emailB"),
				Name:     "User B",
				Role:     "member",
			}
		}

		q := &userMockQuerier{
			listUsersByOrganizationFn: func(_ context.Context, organizationID uuid.UUID) ([]dbgen.User, error) {
				if organizationID == organizationA {
					return usersA, nil
				}
				if organizationID == organizationB {
					return usersB, nil
				}
				return nil, nil
			},
		}

		svc := NewUserService(newMockDB(q))

		results, err := svc.List(context.Background(), organizationA.String())
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
	// Pre-compute hash once outside the loop — bcrypt cost is not what this property tests.
	const fixedPassword = "testpassword1"
	fixedHash, _ := bcrypt.GenerateFromPassword([]byte(fixedPassword), bcrypt.MinCost)

	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		count := rapid.IntRange(1, 3).Draw(t, "count")

		dbUsers := make([]dbgen.User, count)
		for i := range dbUsers {
			dbUsers[i] = dbgen.User{
				ID:       uuid.New(),
				OrganizationID: organizationID,
				Email:    rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "email"),
				Name:     rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "name"),
				Password: string(fixedHash),
				Role:     "member",
			}
		}

		q := &userMockQuerier{
			listUsersByOrganizationFn: func(_ context.Context, _ uuid.UUID) ([]dbgen.User, error) {
				return dbUsers, nil
			},
		}

		svc := NewUserService(newMockDB(q))
		results, err := svc.List(context.Background(), organizationID.String())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// domain.User has no Password field — this verifies the mapping never leaks it.
		// The struct itself enforces this at compile time; we verify count and content.
		if len(results) != count {
			t.Fatalf("expected %d users, got %d", count, len(results))
		}
		for _, u := range results {
			// Verify the returned user is a domain.User (no Password field by design).
			// We check that the struct only has the expected fields populated.
			if u.ID == "" {
				t.Fatal("user ID must not be empty")
			}
			if u.Email == "" {
				t.Fatal("user Email must not be empty")
			}
			// domain.User struct has no Password field — confirmed by type system.
			// The following line would not compile if Password existed:
			// _ = u.Password
		}
	})
}

// TestProperty_UserCreationRoundTrip verifies that for any valid CreateUserRequest
// (role != owner), after Create succeeds, the returned domain.User must have the
// same email and role as the request, and be linked to the correct organizationID.
func TestProperty_UserCreationRoundTrip(t *testing.T) {
	// Use a fixed password — bcrypt cost is not what this property tests.
	const fixedPassword = "testpassword1"

	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		email := rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "email")
		name := rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "name")
		role := rapid.SampledFrom([]domain.Role{domain.RoleAdmin, domain.RoleMember}).Draw(t, "role")

		newUserID := uuid.New()

		q := &userMockQuerier{
			getUserByEmailFn: func(_ context.Context, _ dbgen.GetUserByEmailAndOrganizationParams) (dbgen.User, error) {
				return dbgen.User{}, errors.New("not found")
			},
			createUserFn: func(_ context.Context, arg dbgen.CreateUserParams) (dbgen.User, error) {
				if err := bcrypt.CompareHashAndPassword([]byte(arg.Password), []byte(fixedPassword)); err != nil {
					return dbgen.User{}, errors.New("invalid bcrypt hash in CreateUser call")
				}
				return dbgen.User{
					ID:       newUserID,
					OrganizationID: arg.OrganizationID,
					Email:    arg.Email,
					Name:     arg.Name,
					Password: arg.Password,
					Role:     arg.Role,
				}, nil
			},
		}

		svc := newUserServiceWithCost(newMockDB(q), bcrypt.MinCost)
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
