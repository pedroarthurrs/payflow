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
	getUserByEmailFn    func(ctx context.Context, arg dbgen.GetUserByEmailAndTenantParams) (dbgen.User, error)
	getUserByIDFn       func(ctx context.Context, arg dbgen.GetUserByIDParams) (dbgen.User, error)
	listUsersByTenantFn func(ctx context.Context, tenantID uuid.UUID) ([]dbgen.User, error)
	createUserFn        func(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.User, error)
	softDeleteUserFn    func(ctx context.Context, arg dbgen.SoftDeleteUserParams) error
	updateUserFn        func(ctx context.Context, arg dbgen.UpdateUserParams) (dbgen.User, error)
}

func (m *userMockQuerier) GetUserByEmailAndTenant(ctx context.Context, arg dbgen.GetUserByEmailAndTenantParams) (dbgen.User, error) {
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

func (m *userMockQuerier) ListUsersByTenant(ctx context.Context, tenantID uuid.UUID) ([]dbgen.User, error) {
	if m.listUsersByTenantFn != nil {
		return m.listUsersByTenantFn(ctx, tenantID)
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
func (m *userMockQuerier) GetTenant(ctx context.Context, id uuid.UUID) (dbgen.Tenant, error) {
	return dbgen.Tenant{}, nil
}
func (m *userMockQuerier) HardDeleteCustomer(ctx context.Context, arg dbgen.HardDeleteCustomerParams) error {
	return nil
}
func (m *userMockQuerier) ListAddressesByCustomer(ctx context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
	return nil, nil
}
func (m *userMockQuerier) ListCustomersByTenant(ctx context.Context, tenantID uuid.UUID) ([]dbgen.Customer, error) {
	return nil, nil
}
func (m *userMockQuerier) ListTenants(ctx context.Context) ([]dbgen.Tenant, error) {
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

// Feature: rbac-dynamic-auth, Property 9: Isolamento multi-tenant em operações de usuário
// For any List call with tenantID_A, no user belonging to tenantID_B appears in results.
// Validates: Requirements 8.2, 8.3
func TestProperty_UserMultiTenantIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantA := uuid.New()
		tenantB := uuid.New()

		// Generate 1–5 users for tenant A and 1–5 for tenant B
		countA := rapid.IntRange(1, 5).Draw(t, "countA")
		countB := rapid.IntRange(1, 5).Draw(t, "countB")

		usersA := make([]dbgen.User, countA)
		for i := range usersA {
			usersA[i] = dbgen.User{
				ID:       uuid.New(),
				TenantID: tenantA,
				Email:    rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "emailA"),
				Name:     "User A",
				Role:     "member",
			}
		}

		usersB := make([]dbgen.User, countB)
		for i := range usersB {
			usersB[i] = dbgen.User{
				ID:       uuid.New(),
				TenantID: tenantB,
				Email:    rapid.StringMatching(`[a-z]{4,8}@other\.com`).Draw(t, "emailB"),
				Name:     "User B",
				Role:     "member",
			}
		}

		q := &userMockQuerier{
			listUsersByTenantFn: func(_ context.Context, tenantID uuid.UUID) ([]dbgen.User, error) {
				if tenantID == tenantA {
					return usersA, nil
				}
				if tenantID == tenantB {
					return usersB, nil
				}
				return nil, nil
			},
		}

		svc := NewUserService(newMockDB(q))

		results, err := svc.List(context.Background(), tenantA.String())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, u := range results {
			if u.TenantID != tenantA.String() {
				t.Fatalf("isolation violation: got user with tenantID=%s, expected %s", u.TenantID, tenantA)
			}
		}

		if len(results) != countA {
			t.Fatalf("expected %d users for tenantA, got %d", countA, len(results))
		}
	})
}

// Feature: rbac-dynamic-auth, Property 10: Lista de usuários nunca expõe o campo password
// For any List call, no domain.User in the result has a non-empty Password field.
// Validates: Requirements 7.2
func TestProperty_UserListNoPassword(t *testing.T) {
	// Pre-compute hash once outside the loop — bcrypt cost is not what this property tests.
	const fixedPassword = "testpassword1"
	fixedHash, _ := bcrypt.GenerateFromPassword([]byte(fixedPassword), bcrypt.MinCost)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		count := rapid.IntRange(1, 3).Draw(t, "count")

		dbUsers := make([]dbgen.User, count)
		for i := range dbUsers {
			dbUsers[i] = dbgen.User{
				ID:       uuid.New(),
				TenantID: tenantID,
				Email:    rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "email"),
				Name:     rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "name"),
				Password: string(fixedHash),
				Role:     "member",
			}
		}

		q := &userMockQuerier{
			listUsersByTenantFn: func(_ context.Context, _ uuid.UUID) ([]dbgen.User, error) {
				return dbUsers, nil
			},
		}

		svc := NewUserService(newMockDB(q))
		results, err := svc.List(context.Background(), tenantID.String())
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

// Feature: rbac-dynamic-auth, Property 11: Criação de usuário round-trip
// For any valid CreateUserRequest (role != owner), after Create succeeds,
// the returned domain.User must have the same email and role as the request,
// and be linked to the correct tenantID.
// Validates: Requirements 7.1
func TestProperty_UserCreationRoundTrip(t *testing.T) {
	// Use a fixed password — bcrypt cost is not what this property tests.
	const fixedPassword = "testpassword1"

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		email := rapid.StringMatching(`[a-z]{4,8}@example\.com`).Draw(t, "email")
		name := rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "name")
		role := rapid.SampledFrom([]domain.Role{domain.RoleAdmin, domain.RoleMember}).Draw(t, "role")

		newUserID := uuid.New()

		q := &userMockQuerier{
			getUserByEmailFn: func(_ context.Context, _ dbgen.GetUserByEmailAndTenantParams) (dbgen.User, error) {
				return dbgen.User{}, errors.New("not found")
			},
			createUserFn: func(_ context.Context, arg dbgen.CreateUserParams) (dbgen.User, error) {
				if err := bcrypt.CompareHashAndPassword([]byte(arg.Password), []byte(fixedPassword)); err != nil {
					return dbgen.User{}, errors.New("invalid bcrypt hash in CreateUser call")
				}
				return dbgen.User{
					ID:       newUserID,
					TenantID: arg.TenantID,
					Email:    arg.Email,
					Name:     arg.Name,
					Password: arg.Password,
					Role:     arg.Role,
				}, nil
			},
		}

		svc := newUserServiceWithCost(newMockDB(q), bcrypt.MinCost)
		result, err := svc.Create(context.Background(), tenantID.String(), CreateUserRequest{
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
		if result.TenantID != tenantID.String() {
			t.Fatalf("expected tenantID=%q, got %q", tenantID, result.TenantID)
		}
		if result.ID == "" {
			t.Fatal("expected non-empty user ID")
		}
	})
}
