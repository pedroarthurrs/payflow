package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"i9pay/internal/auth/token"
	"i9pay/internal/database"
	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"
)

// ============================================================
// Mock TokenProvider
// ============================================================

type mockTokenProvider struct {
	newPairFn           func(tenantID string) (*token.Pair, error)
	newPairWithClaimsFn func(input token.ClaimsInput) (*token.Pair, error)
	refreshFn           func(refreshToken string) (*token.Pair, error)
	refreshWithClaimsFn func(refreshToken string) (*token.Pair, error)
}

func (m *mockTokenProvider) NewPair(tenantID string) (*token.Pair, error) {
	if m.newPairFn != nil {
		return m.newPairFn(tenantID)
	}
	return nil, nil
}

func (m *mockTokenProvider) NewPairWithClaims(input token.ClaimsInput) (*token.Pair, error) {
	if m.newPairWithClaimsFn != nil {
		return m.newPairWithClaimsFn(input)
	}
	return &token.Pair{
		AccessToken:           "access-" + input.TenantID,
		AccessTokenExpiresAt:  time.Now().Add(15 * time.Minute),
		RefreshToken:          "refresh-" + input.TenantID,
		RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}, nil
}

func (m *mockTokenProvider) Refresh(refreshToken string) (*token.Pair, error) {
	if m.refreshFn != nil {
		return m.refreshFn(refreshToken)
	}
	return nil, nil
}

func (m *mockTokenProvider) RefreshWithClaims(refreshToken string) (*token.Pair, error) {
	if m.refreshWithClaimsFn != nil {
		return m.refreshWithClaimsFn(refreshToken)
	}
	return nil, nil
}

// ============================================================
// Mock DB Querier
// ============================================================

// authMockQuerier is a minimal implementation of dbgen.Querier for auth service testing.
type authMockQuerier struct {
	getAPIKeyByHashFn      func(ctx context.Context, keyHash string) (dbgen.ApiKey, error)
	updateAPIKeyLastUsedFn func(ctx context.Context, id uuid.UUID) error
	getUserByEmailFn       func(ctx context.Context, email string) (dbgen.User, error)
	lastUsedUpdated        bool
}

func (m *authMockQuerier) GetAPIKeyByHash(ctx context.Context, keyHash string) (dbgen.ApiKey, error) {
	if m.getAPIKeyByHashFn != nil {
		return m.getAPIKeyByHashFn(ctx, keyHash)
	}
	return dbgen.ApiKey{}, errors.New("not found")
}

func (m *authMockQuerier) UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error {
	m.lastUsedUpdated = true
	if m.updateAPIKeyLastUsedFn != nil {
		return m.updateAPIKeyLastUsedFn(ctx, id)
	}
	return nil
}

func (m *authMockQuerier) GetUserByEmail(ctx context.Context, email string) (dbgen.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return dbgen.User{}, errors.New("not found")
}

func (m *authMockQuerier) GetUserByEmailAndTenant(ctx context.Context, arg dbgen.GetUserByEmailAndTenantParams) (dbgen.User, error) {
	return dbgen.User{}, errors.New("not implemented")
}

func (m *authMockQuerier) CreateAddress(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
	return dbgen.Address{}, nil
}
func (m *authMockQuerier) CreateCustomer(ctx context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
	return dbgen.Customer{}, nil
}
func (m *authMockQuerier) CreateUser(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *authMockQuerier) DeleteAddress(ctx context.Context, arg dbgen.DeleteAddressParams) error {
	return nil
}
func (m *authMockQuerier) GetAddressByID(ctx context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
	return dbgen.Address{}, nil
}
func (m *authMockQuerier) GetCustomerByID(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
	return dbgen.Customer{}, nil
}
func (m *authMockQuerier) GetTenant(ctx context.Context, id uuid.UUID) (dbgen.Tenant, error) {
	return dbgen.Tenant{}, nil
}
func (m *authMockQuerier) GetUserByID(ctx context.Context, arg dbgen.GetUserByIDParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *authMockQuerier) HardDeleteCustomer(ctx context.Context, arg dbgen.HardDeleteCustomerParams) error {
	return nil
}
func (m *authMockQuerier) ListAddressesByCustomer(ctx context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
	return nil, nil
}
func (m *authMockQuerier) ListCustomersByTenant(ctx context.Context, tenantID uuid.UUID) ([]dbgen.Customer, error) {
	return nil, nil
}
func (m *authMockQuerier) ListTenants(ctx context.Context) ([]dbgen.Tenant, error) {
	return nil, nil
}
func (m *authMockQuerier) ListUsersByTenant(ctx context.Context, tenantID uuid.UUID) ([]dbgen.User, error) {
	return nil, nil
}
func (m *authMockQuerier) SoftDeleteCustomer(ctx context.Context, arg dbgen.SoftDeleteCustomerParams) error {
	return nil
}
func (m *authMockQuerier) SoftDeleteUser(ctx context.Context, arg dbgen.SoftDeleteUserParams) error {
	return nil
}
func (m *authMockQuerier) UpdateAddress(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
	return dbgen.Address{}, nil
}
func (m *authMockQuerier) UpdateCustomer(ctx context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
	return dbgen.Customer{}, nil
}
func (m *authMockQuerier) UpdateUser(ctx context.Context, arg dbgen.UpdateUserParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}

// newMockDB wraps an authMockQuerier in a *database.DB for use in AuthService.
func newMockDB(q dbgen.Querier) *database.DB {
	return &database.DB{Queries: q}
}

// ============================================================
// Unit tests — Login (user mode)
// ============================================================

func TestLogin_UserMode_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	password := "s3cret"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)

	q := &authMockQuerier{
		getUserByEmailFn: func(_ context.Context, email string) (dbgen.User, error) {
			return dbgen.User{
				ID:       userID,
				TenantID: tenantID,
				Email:    email,
				Role:     "admin",
				Password: string(hash),
			}, nil
		},
	}

	var capturedInput token.ClaimsInput
	mock := &mockTokenProvider{
		newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
			capturedInput = input
			return &token.Pair{AccessToken: "tok"}, nil
		},
	}

	svc := NewAuthService(mock, newMockDB(q))
	pair, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: password,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair == nil {
		t.Fatal("expected token pair, got nil")
	}
	if capturedInput.AuthType != domain.AuthTypeUser {
		t.Fatalf("expected auth_type=user, got %s", capturedInput.AuthType)
	}
	if capturedInput.UserID != userID.String() {
		t.Fatalf("expected user_id=%s, got %s", userID, capturedInput.UserID)
	}
	if capturedInput.Role != domain.RoleAdmin {
		t.Fatalf("expected role=admin, got %s", capturedInput.Role)
	}
}

func TestLogin_UserMode_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)

	q := &authMockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (dbgen.User, error) {
			return dbgen.User{Password: string(hash)}, nil
		},
	}

	svc := NewAuthService(&mockTokenProvider{}, newMockDB(q))
	_, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: "wrong",
	})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLogin_APIKeyMode_Success(t *testing.T) {
	tenantID := uuid.New()
	keyID := uuid.New()
	apiKey := "sk_live_testkey123"
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

	q := &authMockQuerier{
		getAPIKeyByHashFn: func(_ context.Context, keyHash string) (dbgen.ApiKey, error) {
			if keyHash != hash {
				return dbgen.ApiKey{}, errors.New("not found")
			}
			return dbgen.ApiKey{ID: keyID, TenantID: tenantID}, nil
		},
	}

	var capturedInput token.ClaimsInput
	mock := &mockTokenProvider{
		newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
			capturedInput = input
			return &token.Pair{AccessToken: "tok"}, nil
		},
	}

	svc := NewAuthService(mock, newMockDB(q))
	pair, err := svc.Login(context.Background(), domain.LoginRequest{APIKey: apiKey})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair == nil {
		t.Fatal("expected token pair, got nil")
	}
	if capturedInput.AuthType != domain.AuthTypeSystem {
		t.Fatalf("expected auth_type=system, got %s", capturedInput.AuthType)
	}
	if capturedInput.UserID != "" {
		t.Fatalf("expected empty user_id, got %s", capturedInput.UserID)
	}
	if capturedInput.Role != "" {
		t.Fatalf("expected empty role, got %s", capturedInput.Role)
	}
	if !q.lastUsedUpdated {
		t.Fatal("expected UpdateAPIKeyLastUsed to be called")
	}
}

func TestLogin_BadRequest(t *testing.T) {
	svc := NewAuthService(&mockTokenProvider{}, newMockDB(&authMockQuerier{}))
	_, err := svc.Login(context.Background(), domain.LoginRequest{})
	if !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

// ============================================================
// Unit tests — RefreshToken
// ============================================================

func TestRefreshToken_Success(t *testing.T) {
	expectedRefresh := "my-refresh-token-abc"
	expectedPair := &token.Pair{
		AccessToken:           "new-access-token",
		AccessTokenExpiresAt:  time.Now().Add(15 * time.Minute),
		RefreshToken:          "new-refresh-token",
		RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	var calledWith string
	mock := &mockTokenProvider{
		refreshWithClaimsFn: func(refreshToken string) (*token.Pair, error) {
			calledWith = refreshToken
			return expectedPair, nil
		},
	}

	svc := NewAuthService(mock, newMockDB(&authMockQuerier{}))
	pair, err := svc.RefreshToken(expectedRefresh)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair != expectedPair {
		t.Fatal("expected exact pair returned")
	}
	if calledWith != expectedRefresh {
		t.Fatalf("expected RefreshWithClaims called with %q, got %q", expectedRefresh, calledWith)
	}
}

func TestRefreshToken_Error(t *testing.T) {
	providerErr := errors.New("refresh token expired")

	mock := &mockTokenProvider{
		refreshWithClaimsFn: func(_ string) (*token.Pair, error) {
			return nil, providerErr
		},
	}

	svc := NewAuthService(mock, newMockDB(&authMockQuerier{}))
	pair, err := svc.RefreshToken("some-expired-token")

	if pair != nil {
		t.Fatalf("expected nil pair, got %+v", pair)
	}
	if !errors.Is(err, providerErr) {
		t.Fatalf("expected error %v, got %v", providerErr, err)
	}
}

// ============================================================
// Property-based tests (pgregory.net/rapid)
// ============================================================

// For any API key string, hex(SHA-256(key)) computed twice must be identical.
func TestProperty_APIKeyHashRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.String().Draw(t, "apiKey")

		hash1 := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
		hash2 := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))

		if hash1 != hash2 {
			t.Fatalf("SHA-256 is not deterministic: %q != %q", hash1, hash2)
		}
		if len(hash1) != 64 {
			t.Fatalf("expected 64-char hex SHA-256, got len=%d", len(hash1))
		}
	})
}

// For any valid user login, the emitted access token must have non-empty user_id,
// non-empty role, and auth_type == "user".
func TestProperty_UserTokenClaimsComplete(t *testing.T) {
	// Pre-compute hash once — bcrypt is expensive and its correctness isn't what we're testing here.
	const fixedPassword = "testpassword1"
	fixedHash, _ := bcrypt.GenerateFromPassword([]byte(fixedPassword), bcrypt.MinCost)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		userID := uuid.New()
		role := rapid.SampledFrom([]string{"owner", "admin", "member"}).Draw(t, "role")

		q := &authMockQuerier{
			getUserByEmailFn: func(_ context.Context, _ string) (dbgen.User, error) {
				return dbgen.User{
					ID:       userID,
					TenantID: tenantID,
					Email:    "user@example.com",
					Role:     role,
					Password: string(fixedHash),
				}, nil
			},
		}

		var capturedInput token.ClaimsInput
		mock := &mockTokenProvider{
			newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
				capturedInput = input
				return &token.Pair{AccessToken: "tok"}, nil
			},
		}

		svc := NewAuthService(mock, newMockDB(q))
		_, loginErr := svc.Login(context.Background(), domain.LoginRequest{
			Email:    "user@example.com",
			Password: fixedPassword,
		})
		if loginErr != nil {
			t.Fatalf("unexpected login error: %v", loginErr)
		}

		if capturedInput.UserID == "" {
			t.Fatal("user_id must not be empty for user tokens")
		}
		if capturedInput.Role == "" {
			t.Fatal("role must not be empty for user tokens")
		}
		if capturedInput.AuthType != domain.AuthTypeUser {
			t.Fatalf("expected auth_type=user, got %s", capturedInput.AuthType)
		}
	})
}

// For any valid API key login, the emitted access token must have empty user_id,
// empty role, and auth_type == "system".
func TestProperty_SystemTokenClaimsAbsent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		keyID := uuid.New()
		apiKey := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "apiKey")
		expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

		q := &authMockQuerier{
			getAPIKeyByHashFn: func(_ context.Context, keyHash string) (dbgen.ApiKey, error) {
				if keyHash != expectedHash {
					return dbgen.ApiKey{}, errors.New("not found")
				}
				return dbgen.ApiKey{ID: keyID, TenantID: tenantID}, nil
			},
		}

		var capturedInput token.ClaimsInput
		mock := &mockTokenProvider{
			newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
				capturedInput = input
				return &token.Pair{AccessToken: "tok"}, nil
			},
		}

		svc := NewAuthService(mock, newMockDB(q))
		_, loginErr := svc.Login(context.Background(), domain.LoginRequest{APIKey: apiKey})
		if loginErr != nil {
			t.Fatalf("unexpected login error: %v", loginErr)
		}

		if capturedInput.UserID != "" {
			t.Fatalf("user_id must be empty for system tokens, got %q", capturedInput.UserID)
		}
		if capturedInput.Role != "" {
			t.Fatalf("role must be empty for system tokens, got %q", capturedInput.Role)
		}
		if capturedInput.AuthType != domain.AuthTypeSystem {
			t.Fatalf("expected auth_type=system, got %s", capturedInput.AuthType)
		}
	})
}

// For any payload with api_key present, mode must be system.
// For any payload with tenant_id+email+password and no api_key, mode must be user.
func TestProperty_LoginModeDetection(t *testing.T) {
	// Pre-compute hash once — bcrypt cost is not what this property tests.
	const fixedPassword = "testpassword1"
	fixedHash, _ := bcrypt.GenerateFromPassword([]byte(fixedPassword), bcrypt.MinCost)

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		userID := uuid.New()
		apiKey := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "apiKey")
		apiKeyHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

		q := &authMockQuerier{
			getAPIKeyByHashFn: func(_ context.Context, keyHash string) (dbgen.ApiKey, error) {
				if keyHash == apiKeyHash {
					return dbgen.ApiKey{ID: uuid.New(), TenantID: tenantID}, nil
				}
				return dbgen.ApiKey{}, errors.New("not found")
			},
			getUserByEmailFn: func(_ context.Context, _ string) (dbgen.User, error) {
				return dbgen.User{
					ID:       userID,
					TenantID: tenantID,
					Role:     "member",
					Password: string(fixedHash),
				}, nil
			},
		}

		var capturedInput token.ClaimsInput
		mock := &mockTokenProvider{
			newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
				capturedInput = input
				return &token.Pair{AccessToken: "tok"}, nil
			},
		}

		svc := NewAuthService(mock, newMockDB(q))

		// System mode: api_key present (even with other fields)
		_, err := svc.Login(context.Background(), domain.LoginRequest{
			APIKey: apiKey,
		})
		if err != nil {
			t.Fatalf("system mode login failed: %v", err)
		}
		if capturedInput.AuthType != domain.AuthTypeSystem {
			t.Fatalf("expected system mode when api_key present, got %s", capturedInput.AuthType)
		}

		// User mode: no api_key
		_, err = svc.Login(context.Background(), domain.LoginRequest{
			Email:    "user@example.com",
			Password: fixedPassword,
		})
		if err != nil {
			t.Fatalf("user mode login failed: %v", err)
		}
		if capturedInput.AuthType != domain.AuthTypeUser {
			t.Fatalf("expected user mode when no api_key, got %s", capturedInput.AuthType)
		}
	})
}

// After a successful API key login, UpdateAPIKeyLastUsed must have been called.
func TestProperty_APIKeyLastUsedAtUpdated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		keyID := uuid.New()
		apiKey := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "apiKey")
		expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

		q := &authMockQuerier{
			getAPIKeyByHashFn: func(_ context.Context, keyHash string) (dbgen.ApiKey, error) {
				if keyHash == expectedHash {
					return dbgen.ApiKey{ID: keyID, TenantID: tenantID}, nil
				}
				return dbgen.ApiKey{}, errors.New("not found")
			},
		}

		svc := NewAuthService(&mockTokenProvider{}, newMockDB(q))
		_, err := svc.Login(context.Background(), domain.LoginRequest{APIKey: apiKey})
		if err != nil {
			t.Fatalf("unexpected login error: %v", err)
		}

		if !q.lastUsedUpdated {
			t.Fatal("UpdateAPIKeyLastUsed must be called after successful API key login")
		}
	})
}

// For any user token (auth_type=user), after refresh, the new token must have
// the same user_id and role as the original.
func TestProperty_RefreshPreservesUserClaims(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := rapid.StringMatching(`[a-f0-9\-]{36}`).Draw(t, "tenantID")
		userID := rapid.StringMatching(`[a-f0-9\-]{36}`).Draw(t, "userID")
		role := rapid.SampledFrom([]domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleMember}).Draw(t, "role")

		// Use the real token.Service so refresh actually parses and re-emits claims.
		tokenSvc := token.NewService("test-secret-key-for-property-tests", 15*time.Minute, 7*24*time.Hour)

		// Issue an original pair with user claims.
		original, err := tokenSvc.NewPairWithClaims(token.ClaimsInput{
			TenantID: tenantID,
			UserID:   userID,
			Role:     role,
			AuthType: domain.AuthTypeUser,
		})
		if err != nil {
			t.Fatalf("failed to issue original token: %v", err)
		}

		// Refresh using the real service.
		refreshed, err := tokenSvc.RefreshWithClaims(original.RefreshToken)
		if err != nil {
			t.Fatalf("failed to refresh token: %v", err)
		}

		// Verify the new access token preserves user_id and role.
		claims, err := tokenSvc.Verify(refreshed.AccessToken, token.TypeAccess)
		if err != nil {
			t.Fatalf("failed to verify refreshed access token: %v", err)
		}

		if claims.UserID != userID {
			t.Fatalf("expected user_id=%q after refresh, got %q", userID, claims.UserID)
		}
		if claims.Role != role {
			t.Fatalf("expected role=%q after refresh, got %q", role, claims.Role)
		}
		if claims.AuthType != domain.AuthTypeUser {
			t.Fatalf("expected auth_type=user after refresh, got %s", claims.AuthType)
		}
	})
}

// TestProperty_RefreshTokenErrorPropagation verifies that RefreshToken propagates errors from the TokenProvider.
func TestProperty_RefreshTokenErrorPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		refreshInput := rapid.String().Draw(t, "refreshToken")
		errMsg := rapid.StringMatching(`[a-zA-Z ]{5,30}`).Draw(t, "errorMessage")

		providerErr := fmt.Errorf("%s", errMsg)

		mock := &mockTokenProvider{
			refreshWithClaimsFn: func(_ string) (*token.Pair, error) {
				return nil, providerErr
			},
		}

		svc := NewAuthService(mock, newMockDB(&authMockQuerier{}))
		pair, err := svc.RefreshToken(refreshInput)

		if pair != nil {
			t.Fatalf("expected nil pair, got %+v", pair)
		}
		if err != providerErr {
			t.Fatalf("expected error %v, got %v", providerErr, err)
		}
	})
}

// ============================================================
// Bug condition exploration test
// ============================================================

// TestLogin_BugCondition_EmailPasswordWithoutTenantID confirms the bug described in
// .kiro/specs/login-remove-tenant-id/bugfix.md:
//
//	isBugCondition(X) = X.APIKey="" AND X.Email!="" AND X.Password!="" AND X.TenantID=""
//
// The test asserts the CORRECT (post-fix) behavior: err==nil AND pair!=nil.
// On UNFIXED code the guard `req.TenantID != ""` is false, so Login returns
// ErrBadRequest — causing this test to FAIL and confirming the bug.
//
// Counterexample (confirmed on unfixed code):
//
//	Login({Email: "000@example.com", Password: "testpassword1"}) → ErrBadRequest
//	("tenant_id, email and password or api_key required")
//
// Root cause: `req.TenantID != ""` guard in AuthService.Login (auth.go) is false
// when TenantID is empty, so the function falls through to `return nil, ErrBadRequest`.
func TestLogin_BugCondition_EmailPasswordWithoutTenantID(t *testing.T) {
	const fixedPassword = "testpassword1"
	fixedHash, _ := bcrypt.GenerateFromPassword([]byte(fixedPassword), bcrypt.MinCost)

	rapid.Check(t, func(t *rapid.T) {
		email := rapid.StringMatching(`[a-zA-Z0-9]{3,10}@example\.com`).Draw(t, "email")

		q := &authMockQuerier{
			getUserByEmailFn: func(_ context.Context, email string) (dbgen.User, error) {
				// Return a valid user for any email so the bug is isolated to the dispatch guard.
				return dbgen.User{
					ID:       uuid.New(),
					TenantID: uuid.New(),
					Email:    email,
					Role:     "member",
					Password: string(fixedHash),
				}, nil
			},
		}

		mock := &mockTokenProvider{
			newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
				return &token.Pair{AccessToken: "tok"}, nil
			},
		}

		svc := NewAuthService(mock, newMockDB(q))

		// Bug condition: no TenantID supplied.
		pair, err := svc.Login(context.Background(), domain.LoginRequest{
			Email:    email,
			Password: fixedPassword,
			// TenantID intentionally omitted
		})

		// CORRECT behavior: should succeed.
		// On UNFIXED code this fails because `req.TenantID != ""` is false → ErrBadRequest.
		if err != nil {
			t.Fatalf("expected no error for email+password login without TenantID, got: %v", err)
		}
		if pair == nil {
			t.Fatal("expected token pair, got nil")
		}
	})
}

// TestProperty_Preservation_NonBuggyInputs verifies that non-bug-condition inputs
// (where isBugCondition returns FALSE) produce the same results before and after the fix.
func TestProperty_Preservation_NonBuggyInputs(t *testing.T) {
	// Sub-property A: API key preservation
	// For any valid API key, auth_type=system token is returned.
	t.Run("A_APIKeyPreservation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tenantID := uuid.New()
			keyID := uuid.New()
			apiKey := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "apiKey")
			expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

			q := &authMockQuerier{
				getAPIKeyByHashFn: func(_ context.Context, keyHash string) (dbgen.ApiKey, error) {
					if keyHash == expectedHash {
						return dbgen.ApiKey{ID: keyID, TenantID: tenantID}, nil
					}
					return dbgen.ApiKey{}, errors.New("not found")
				},
			}

			var capturedInput token.ClaimsInput
			mock := &mockTokenProvider{
				newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
					capturedInput = input
					return &token.Pair{AccessToken: "tok"}, nil
				},
			}

			svc := NewAuthService(mock, newMockDB(q))
			_, err := svc.Login(context.Background(), domain.LoginRequest{APIKey: apiKey})

			if err != nil {
				t.Fatalf("expected no error for API key login, got: %v", err)
			}
			if capturedInput.AuthType != domain.AuthTypeSystem {
				t.Fatalf("expected auth_type=system, got %s", capturedInput.AuthType)
			}
		})
	})

	// Sub-property B: Empty request preservation
	// For any empty LoginRequest{}, ErrBadRequest is returned.
	t.Run("B_EmptyRequestPreservation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			svc := NewAuthService(&mockTokenProvider{}, newMockDB(&authMockQuerier{}))
			_, err := svc.Login(context.Background(), domain.LoginRequest{})

			if !errors.Is(err, domain.ErrBadRequest) {
				t.Fatalf("expected ErrBadRequest for empty request, got: %v", err)
			}
		})
	})

	// Sub-property C: Wrong password preservation
	// For any {Email, Password: "wrong", TenantID: <valid uuid>} where DB returns a user
	// with a different password hash, ErrUnauthorized is returned.
	t.Run("C_WrongPasswordPreservation", func(t *testing.T) {
		knownHash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)

		rapid.Check(t, func(t *rapid.T) {
			tenantID := uuid.New()
			email := rapid.StringMatching(`[a-zA-Z0-9]{3,10}@example\.com`).Draw(t, "email")

			q := &authMockQuerier{
				getUserByEmailFn: func(_ context.Context, email string) (dbgen.User, error) {
					return dbgen.User{
						ID:       uuid.New(),
						TenantID: tenantID,
						Email:    email,
						Role:     "member",
						Password: string(knownHash),
					}, nil
				},
			}

			svc := NewAuthService(&mockTokenProvider{}, newMockDB(q))
			_, err := svc.Login(context.Background(), domain.LoginRequest{
				Email:    email,
				Password: "wrong",
			})

			if !errors.Is(err, domain.ErrUnauthorized) {
				t.Fatalf("expected ErrUnauthorized for wrong password, got: %v", err)
			}
		})
	})

	// Sub-property D: Unknown email preservation
	// For any {Email: generated, Password: generated, TenantID: <valid uuid>} where DB
	// returns not-found, ErrUnauthorized is returned.
	t.Run("D_UnknownEmailPreservation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			email := rapid.StringMatching(`[a-zA-Z0-9]{3,10}@example\.com`).Draw(t, "email")
			password := rapid.StringMatching(`[a-zA-Z0-9]{8,16}`).Draw(t, "password")

			q := &authMockQuerier{
				getUserByEmailFn: func(_ context.Context, _ string) (dbgen.User, error) {
					return dbgen.User{}, errors.New("not found")
				},
			}

			svc := NewAuthService(&mockTokenProvider{}, newMockDB(q))
			_, err := svc.Login(context.Background(), domain.LoginRequest{
				Email:    email,
				Password: password,
			})

			if !errors.Is(err, domain.ErrUnauthorized) {
				t.Fatalf("expected ErrUnauthorized for unknown email, got: %v", err)
			}
		})
	})
}

// Ensure authMockQuerier satisfies the interface at compile time.
var _ dbgen.Querier = (*authMockQuerier)(nil)

// Ensure sql package is used (for NullTime in stubs).
var _ = sql.NullTime{}
