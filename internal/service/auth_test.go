package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

	"payflow/internal/auth/token"
	"payflow/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"
)

// ============================================================
// Mock TokenProvider
// ============================================================

type mockTokenProvider struct {
	newPairFn           func(organizationID string) (*token.Pair, error)
	newPairWithClaimsFn func(input token.ClaimsInput) (*token.Pair, error)
	refreshFn           func(refreshToken string) (*token.Pair, error)
	refreshWithClaimsFn func(refreshToken string) (*token.Pair, error)
	newOrgSelectTokenFn func(userAccountID string) (string, error)
	verifyFn            func(tokenString string, expectedType token.TokenType) (*token.Claims, error)
}

func (m *mockTokenProvider) NewPair(organizationID string) (*token.Pair, error) {
	if m.newPairFn != nil {
		return m.newPairFn(organizationID)
	}
	return nil, nil
}

func (m *mockTokenProvider) NewPairWithClaims(input token.ClaimsInput) (*token.Pair, error) {
	if m.newPairWithClaimsFn != nil {
		return m.newPairWithClaimsFn(input)
	}
	return &token.Pair{
		AccessToken:           "access-" + input.OrganizationID,
		AccessTokenExpiresAt:  time.Now().Add(15 * time.Minute),
		RefreshToken:          "refresh-" + input.OrganizationID,
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

func (m *mockTokenProvider) NewOrgSelectToken(userAccountID string) (string, error) {
	if m.newOrgSelectTokenFn != nil {
		return m.newOrgSelectTokenFn(userAccountID)
	}
	return "session-token-" + userAccountID, nil
}

func (m *mockTokenProvider) Verify(tokenString string, expectedType token.TokenType) (*token.Claims, error) {
	if m.verifyFn != nil {
		return m.verifyFn(tokenString, expectedType)
	}
	return nil, errors.New("not implemented")
}

// ============================================================
// Helpers
// ============================================================

// singleMembershipRepo returns a mockAuthRepo wired for a user with one org membership.
func singleMembershipRepo(account *domain.AuthUserAccount, membership domain.AuthMembership) *mockAuthRepo {
	return &mockAuthRepo{
		getUserAccountByEmailFn: func(_ context.Context, _ string) (*domain.AuthUserAccount, error) {
			return account, nil
		},
		listMembershipsForUserFn: func(_ context.Context, _ uuid.UUID) ([]domain.AuthMembership, error) {
			return []domain.AuthMembership{membership}, nil
		},
	}
}

// ============================================================
// Unit tests - Login (user mode)
// ============================================================

func TestLogin_UserMode_Success(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	password := "s3cret"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)

	auth := singleMembershipRepo(
		&domain.AuthUserAccount{ID: userID, Password: string(hash)},
		domain.AuthMembership{OrganizationID: organizationID, Role: domain.RoleAdmin},
	)

	var capturedInput token.ClaimsInput
	mock := &mockTokenProvider{
		newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
			capturedInput = input
			return &token.Pair{AccessToken: "tok"}, nil
		},
	}

	svc := NewAuthService(mock, auth)
	result, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: password,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || result.TokenPair == nil {
		t.Fatal("expected token pair in result")
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

func TestLogin_UserMode_MultiOrg(t *testing.T) {
	userID := uuid.New()
	password := "s3cret"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)

	orgs := []domain.AuthMembership{
		{OrganizationID: uuid.New(), Role: domain.RoleAdmin},
		{OrganizationID: uuid.New(), Role: domain.RoleAdmin},
	}

	auth := &mockAuthRepo{
		getUserAccountByEmailFn: func(_ context.Context, _ string) (*domain.AuthUserAccount, error) {
			return &domain.AuthUserAccount{ID: userID, Password: string(hash)}, nil
		},
		listMembershipsForUserFn: func(_ context.Context, _ uuid.UUID) ([]domain.AuthMembership, error) {
			return orgs, nil
		},
	}

	var sessionTokenIssued string
	mock := &mockTokenProvider{
		newOrgSelectTokenFn: func(userAccountID string) (string, error) {
			sessionTokenIssued = "session-" + userAccountID
			return sessionTokenIssued, nil
		},
	}

	svc := NewAuthService(mock, auth)
	result, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: password,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.TokenPair != nil {
		t.Fatal("expected no token pair for multi-org login")
	}
	if result.SessionToken == nil || *result.SessionToken != sessionTokenIssued {
		t.Fatalf("expected session token %q", sessionTokenIssued)
	}
	if len(result.Organizations) != 2 {
		t.Fatalf("expected 2 organizations, got %d", len(result.Organizations))
	}
}

func TestLogin_UserMode_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)

	auth := &mockAuthRepo{
		getUserAccountByEmailFn: func(_ context.Context, _ string) (*domain.AuthUserAccount, error) {
			return &domain.AuthUserAccount{Password: string(hash)}, nil
		},
	}

	svc := NewAuthService(&mockTokenProvider{}, auth)
	_, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: "wrong",
	})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLogin_UserMode_NoMemberships(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)

	auth := &mockAuthRepo{
		getUserAccountByEmailFn: func(_ context.Context, _ string) (*domain.AuthUserAccount, error) {
			return &domain.AuthUserAccount{ID: uuid.New(), Password: string(hash)}, nil
		},
		listMembershipsForUserFn: func(_ context.Context, _ uuid.UUID) ([]domain.AuthMembership, error) {
			return nil, nil // no memberships
		},
	}

	svc := NewAuthService(&mockTokenProvider{}, auth)
	_, err := svc.Login(context.Background(), domain.LoginRequest{Email: "x@x.com", Password: "pass"})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for user with no memberships, got %v", err)
	}
}

func TestLogin_APIKeyMode_Success(t *testing.T) {
	organizationID := uuid.New()
	keyID := uuid.New()
	apiKey := "sk_live_testkey123"
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

	auth := &mockAuthRepo{
		getAPIKeyByHashFn: func(_ context.Context, keyHash string) (*domain.APIKey, error) {
			if keyHash != expectedHash {
				return nil, errors.New("not found")
			}
			return &domain.APIKey{ID: keyID, OrganizationID: organizationID}, nil
		},
	}

	var capturedInput token.ClaimsInput
	mock := &mockTokenProvider{
		newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
			capturedInput = input
			return &token.Pair{AccessToken: "tok"}, nil
		},
	}

	svc := NewAuthService(mock, auth)
	result, err := svc.Login(context.Background(), domain.LoginRequest{APIKey: apiKey})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || result.TokenPair == nil {
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
	if !auth.lastUsedUpdated {
		t.Fatal("expected UpdateAPIKeyLastUsed to be called")
	}
}

func TestLogin_BadRequest(t *testing.T) {
	svc := NewAuthService(&mockTokenProvider{}, &mockAuthRepo{})
	_, err := svc.Login(context.Background(), domain.LoginRequest{})
	if !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

// ============================================================
// Unit tests - SelectOrganization
// ============================================================

func TestSelectOrganization_Success(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	auth := &mockAuthRepo{
		listMembershipsForUserFn: func(_ context.Context, id uuid.UUID) ([]domain.AuthMembership, error) {
			if id != userID {
				return nil, errors.New("not found")
			}
			return []domain.AuthMembership{
				{OrganizationID: orgID, Role: domain.RoleAdmin},
			}, nil
		},
	}

	var capturedInput token.ClaimsInput
	mock := &mockTokenProvider{
		verifyFn: func(tokenString string, expectedType token.TokenType) (*token.Claims, error) {
			return &token.Claims{OrganizationID: userID.String()}, nil
		},
		newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
			capturedInput = input
			return &token.Pair{AccessToken: "tok"}, nil
		},
	}

	svc := NewAuthService(mock, auth)
	pair, err := svc.SelectOrganization(context.Background(), "session-tok", orgID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair == nil || pair.AccessToken != "tok" {
		t.Fatal("expected token pair")
	}
	if capturedInput.OrganizationID != orgID.String() {
		t.Fatalf("expected org_id=%s, got %s", orgID, capturedInput.OrganizationID)
	}
	if capturedInput.UserID != userID.String() {
		t.Fatalf("expected user_id=%s, got %s", userID, capturedInput.UserID)
	}
}

func TestSelectOrganization_InvalidToken(t *testing.T) {
	mock := &mockTokenProvider{
		verifyFn: func(_ string, _ token.TokenType) (*token.Claims, error) {
			return nil, errors.New("token expired")
		},
	}

	svc := NewAuthService(mock, &mockAuthRepo{})
	_, err := svc.SelectOrganization(context.Background(), "bad-token", uuid.New())

	if !errors.Is(err, domain.ErrInvalidSessionToken) {
		t.Fatalf("expected ErrInvalidSessionToken, got %v", err)
	}
}

func TestSelectOrganization_OrgNotInMemberships(t *testing.T) {
	userID := uuid.New()

	auth := &mockAuthRepo{
		listMembershipsForUserFn: func(_ context.Context, _ uuid.UUID) ([]domain.AuthMembership, error) {
			return []domain.AuthMembership{
				{OrganizationID: uuid.New(), Role: domain.RoleAdmin},
			}, nil
		},
	}

	mock := &mockTokenProvider{
		verifyFn: func(_ string, _ token.TokenType) (*token.Claims, error) {
			return &token.Claims{OrganizationID: userID.String()}, nil
		},
	}

	svc := NewAuthService(mock, auth)
	_, err := svc.SelectOrganization(context.Background(), "tok", uuid.New())

	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// ============================================================
// Unit tests - RefreshToken
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

	svc := NewAuthService(mock, &mockAuthRepo{})
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

	svc := NewAuthService(mock, &mockAuthRepo{})
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

// For any valid single-org user login, the emitted access token must have
// non-empty user_id, non-empty role, and auth_type == "user".
func TestProperty_UserTokenClaimsComplete(t *testing.T) {
	const fixedPassword = "testpassword1"
	fixedHash, _ := bcrypt.GenerateFromPassword([]byte(fixedPassword), bcrypt.MinCost)

	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		userID := uuid.New()
		role := rapid.SampledFrom([]string{"owner", "admin", "member"}).Draw(t, "role")

		auth := singleMembershipRepo(
			&domain.AuthUserAccount{ID: userID, Password: string(fixedHash)},
			domain.AuthMembership{OrganizationID: organizationID, Role: domain.Role(role)},
		)

		var capturedInput token.ClaimsInput
		mock := &mockTokenProvider{
			newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
				capturedInput = input
				return &token.Pair{AccessToken: "tok"}, nil
			},
		}

		svc := NewAuthService(mock, auth)
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
		organizationID := uuid.New()
		keyID := uuid.New()
		apiKey := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "apiKey")
		expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

		auth := &mockAuthRepo{
			getAPIKeyByHashFn: func(_ context.Context, keyHash string) (*domain.APIKey, error) {
				if keyHash != expectedHash {
					return nil, errors.New("not found")
				}
				return &domain.APIKey{ID: keyID, OrganizationID: organizationID}, nil
			},
		}

		var capturedInput token.ClaimsInput
		mock := &mockTokenProvider{
			newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
				capturedInput = input
				return &token.Pair{AccessToken: "tok"}, nil
			},
		}

		svc := NewAuthService(mock, auth)
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
// For any payload with email+password and no api_key, mode must be user.
func TestProperty_LoginModeDetection(t *testing.T) {
	const fixedPassword = "testpassword1"
	fixedHash, _ := bcrypt.GenerateFromPassword([]byte(fixedPassword), bcrypt.MinCost)

	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		userID := uuid.New()
		apiKey := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "apiKey")
		apiKeyHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

		auth := &mockAuthRepo{
			getAPIKeyByHashFn: func(_ context.Context, keyHash string) (*domain.APIKey, error) {
				if keyHash == apiKeyHash {
					return &domain.APIKey{ID: uuid.New(), OrganizationID: organizationID}, nil
				}
				return nil, errors.New("not found")
			},
			getUserAccountByEmailFn: func(_ context.Context, _ string) (*domain.AuthUserAccount, error) {
				return &domain.AuthUserAccount{ID: userID, Password: string(fixedHash)}, nil
			},
			listMembershipsForUserFn: func(_ context.Context, _ uuid.UUID) ([]domain.AuthMembership, error) {
				return []domain.AuthMembership{
					{OrganizationID: organizationID, Role: domain.RoleDeveloper},
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

		svc := NewAuthService(mock, auth)

		// System mode: api_key present
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
		organizationID := uuid.New()
		keyID := uuid.New()
		apiKey := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "apiKey")
		expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

		auth := &mockAuthRepo{
			getAPIKeyByHashFn: func(_ context.Context, keyHash string) (*domain.APIKey, error) {
				if keyHash == expectedHash {
					return &domain.APIKey{ID: keyID, OrganizationID: organizationID}, nil
				}
				return nil, errors.New("not found")
			},
		}

		svc := NewAuthService(&mockTokenProvider{}, auth)
		_, err := svc.Login(context.Background(), domain.LoginRequest{APIKey: apiKey})
		if err != nil {
			t.Fatalf("unexpected login error: %v", err)
		}

		if !auth.lastUsedUpdated {
			t.Fatal("UpdateAPIKeyLastUsed must be called after successful API key login")
		}
	})
}

// For any user token (auth_type=user), after refresh, the new token must have
// the same user_id and role as the original.
func TestProperty_RefreshPreservesUserClaims(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := rapid.StringMatching(`[a-f0-9\-]{36}`).Draw(t, "organizationID")
		userID := rapid.StringMatching(`[a-f0-9\-]{36}`).Draw(t, "userID")
		role := rapid.SampledFrom([]domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleDeveloper, domain.RoleSupport}).Draw(t, "role")

		// Use the real token.Service so refresh actually parses and re-emits claims.
		tokenSvc := token.NewService("test-secret-key-for-property-tests", 15*time.Minute, 7*24*time.Hour)

		// Issue an original pair with user claims.
		original, err := tokenSvc.NewPairWithClaims(token.ClaimsInput{
			OrganizationID: organizationID,
			UserID:         userID,
			Role:           role,
			AuthType:       domain.AuthTypeUser,
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

// TestProperty_RefreshTokenErrorPropagation verifies that RefreshToken propagates errors.
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

		svc := NewAuthService(mock, &mockAuthRepo{})
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

// TestProperty_Preservation_NonBuggyInputs verifies non-bug-condition inputs
// produce the same results before and after the fix.
func TestProperty_Preservation_NonBuggyInputs(t *testing.T) {
	// Sub-property A: API key preservation
	t.Run("A_APIKeyPreservation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			organizationID := uuid.New()
			keyID := uuid.New()
			apiKey := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "apiKey")
			expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

			auth := &mockAuthRepo{
				getAPIKeyByHashFn: func(_ context.Context, keyHash string) (*domain.APIKey, error) {
					if keyHash == expectedHash {
						return &domain.APIKey{ID: keyID, OrganizationID: organizationID}, nil
					}
					return nil, errors.New("not found")
				},
			}

			var capturedInput token.ClaimsInput
			mock := &mockTokenProvider{
				newPairWithClaimsFn: func(input token.ClaimsInput) (*token.Pair, error) {
					capturedInput = input
					return &token.Pair{AccessToken: "tok"}, nil
				},
			}

			svc := NewAuthService(mock, auth)
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
	t.Run("B_EmptyRequestPreservation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			svc := NewAuthService(&mockTokenProvider{}, &mockAuthRepo{})
			_, err := svc.Login(context.Background(), domain.LoginRequest{})

			if !errors.Is(err, domain.ErrBadRequest) {
				t.Fatalf("expected ErrBadRequest for empty request, got: %v", err)
			}
		})
	})

	// Sub-property C: Wrong password preservation
	t.Run("C_WrongPasswordPreservation", func(t *testing.T) {
		knownHash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)

		rapid.Check(t, func(t *rapid.T) {
			email := rapid.StringMatching(`[a-zA-Z0-9]{3,10}@example\.com`).Draw(t, "email")

			auth := &mockAuthRepo{
				getUserAccountByEmailFn: func(_ context.Context, _ string) (*domain.AuthUserAccount, error) {
					return &domain.AuthUserAccount{ID: uuid.New(), Password: string(knownHash)}, nil
				},
			}

			svc := NewAuthService(&mockTokenProvider{}, auth)
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
	t.Run("D_UnknownEmailPreservation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			email := rapid.StringMatching(`[a-zA-Z0-9]{3,10}@example\.com`).Draw(t, "email")
			password := rapid.StringMatching(`[a-zA-Z0-9]{8,16}`).Draw(t, "password")

			auth := &mockAuthRepo{
				getUserAccountByEmailFn: func(_ context.Context, _ string) (*domain.AuthUserAccount, error) {
					return nil, errors.New("not found")
				},
			}

			svc := NewAuthService(&mockTokenProvider{}, auth)
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
