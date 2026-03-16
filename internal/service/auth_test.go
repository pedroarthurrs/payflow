package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"i9pay/internal/domain"
	"i9pay/internal/token"

	"pgregory.net/rapid"
)

// ============================================================
// Mock TokenProvider
// ============================================================

type mockTokenProvider struct {
	newPairFn func(tenantID string) (*token.Pair, error)
	refreshFn func(refreshToken string) (*token.Pair, error)
}

func (m *mockTokenProvider) NewPair(tenantID string) (*token.Pair, error) {
	if m.newPairFn != nil {
		return m.newPairFn(tenantID)
	}
	return nil, nil
}

func (m *mockTokenProvider) Refresh(refreshToken string) (*token.Pair, error) {
	if m.refreshFn != nil {
		return m.refreshFn(refreshToken)
	}
	return nil, nil
}

// ============================================================
// Unit tests — Login
// ============================================================

func TestLogin_Success(t *testing.T) {
	expectedEmail := "user@example.com"
	expectedPassword := "s3cret"
	expectedTenantID := expectedEmail + expectedPassword

	now := time.Now().Truncate(time.Second)
	expectedPair := &token.Pair{
		AccessToken:           "access-token-abc",
		AccessTokenExpiresAt:  now.Add(15 * time.Minute),
		RefreshToken:          "refresh-token-xyz",
		RefreshTokenExpiresAt: now.Add(7 * 24 * time.Hour),
	}

	var calledWith string
	mock := &mockTokenProvider{
		newPairFn: func(tenantID string) (*token.Pair, error) {
			calledWith = tenantID
			return expectedPair, nil
		},
	}

	svc := NewAuthService(mock)
	pair, err := svc.Login(domain.Credentials{Email: expectedEmail, Password: expectedPassword})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair == nil {
		t.Fatal("expected token pair, got nil")
	}
	if pair.AccessToken != expectedPair.AccessToken {
		t.Fatalf("expected AccessToken %q, got %q", expectedPair.AccessToken, pair.AccessToken)
	}
	if pair.RefreshToken != expectedPair.RefreshToken {
		t.Fatalf("expected RefreshToken %q, got %q", expectedPair.RefreshToken, pair.RefreshToken)
	}
	if calledWith != expectedTenantID {
		t.Fatalf("expected NewPair called with %q, got %q", expectedTenantID, calledWith)
	}
}

func TestLogin_TokenProviderError(t *testing.T) {
	providerErr := errors.New("token generation failed")

	mock := &mockTokenProvider{
		newPairFn: func(_ string) (*token.Pair, error) {
			return nil, providerErr
		},
	}

	svc := NewAuthService(mock)
	pair, err := svc.Login(domain.Credentials{Email: "a@b.com", Password: "pass"})

	if pair != nil {
		t.Fatalf("expected nil pair, got %+v", pair)
	}
	if !errors.Is(err, providerErr) {
		t.Fatalf("expected error %v, got %v", providerErr, err)
	}
}

// ============================================================
// Unit tests — RefreshToken
// ============================================================

func TestRefreshToken_Success(t *testing.T) {
	expectedRefresh := "my-refresh-token-abc"

	now := time.Now().Truncate(time.Second)
	expectedPair := &token.Pair{
		AccessToken:           "new-access-token",
		AccessTokenExpiresAt:  now.Add(15 * time.Minute),
		RefreshToken:          "new-refresh-token",
		RefreshTokenExpiresAt: now.Add(7 * 24 * time.Hour),
	}

	var calledWith string
	mock := &mockTokenProvider{
		refreshFn: func(refreshToken string) (*token.Pair, error) {
			calledWith = refreshToken
			return expectedPair, nil
		},
	}

	svc := NewAuthService(mock)
	pair, err := svc.RefreshToken(expectedRefresh)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair == nil {
		t.Fatal("expected token pair, got nil")
	}
	if pair.AccessToken != expectedPair.AccessToken {
		t.Fatalf("expected AccessToken %q, got %q", expectedPair.AccessToken, pair.AccessToken)
	}
	if pair.RefreshToken != expectedPair.RefreshToken {
		t.Fatalf("expected RefreshToken %q, got %q", expectedPair.RefreshToken, pair.RefreshToken)
	}
	if calledWith != expectedRefresh {
		t.Fatalf("expected Refresh called with %q, got %q", expectedRefresh, calledWith)
	}
}

func TestRefreshToken_Error(t *testing.T) {
	providerErr := errors.New("refresh token expired")

	mock := &mockTokenProvider{
		refreshFn: func(_ string) (*token.Pair, error) {
			return nil, providerErr
		},
	}

	svc := NewAuthService(mock)
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

// Property 1: Login delega tenantID correto e retorna par de tokens inalterado
func TestProperty_LoginDelegation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapid.StringMatching(`[a-z]{3,10}@[a-z]{3,6}\.[a-z]{2,4}`).Draw(t, "email")
		password := rapid.StringMatching(`[a-zA-Z0-9]{4,16}`).Draw(t, "password")

		expectedPair := &token.Pair{
			AccessToken:           rapid.String().Draw(t, "accessToken"),
			AccessTokenExpiresAt:  time.Now().Add(15 * time.Minute),
			RefreshToken:          rapid.String().Draw(t, "refreshToken"),
			RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		var calledWith string
		mock := &mockTokenProvider{
			newPairFn: func(tenantID string) (*token.Pair, error) {
				calledWith = tenantID
				return expectedPair, nil
			},
		}

		svc := NewAuthService(mock)
		pair, err := svc.Login(domain.Credentials{Email: email, Password: password})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if calledWith != email+password {
			t.Fatalf("expected NewPair called with %q, got %q", email+password, calledWith)
		}
		if pair != expectedPair {
			t.Fatalf("expected returned pair to be exactly the mock pair")
		}
	})
}

// Property 2: Login propaga erros do TokenProvider
func TestProperty_LoginErrorPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapid.StringMatching(`[a-z]{3,10}@[a-z]{3,6}\.[a-z]{2,4}`).Draw(t, "email")
		password := rapid.StringMatching(`[a-zA-Z0-9]{4,16}`).Draw(t, "password")
		errMsg := rapid.StringMatching(`[a-zA-Z ]{5,30}`).Draw(t, "errorMessage")

		providerErr := fmt.Errorf("%s", errMsg)

		mock := &mockTokenProvider{
			newPairFn: func(_ string) (*token.Pair, error) {
				return nil, providerErr
			},
		}

		svc := NewAuthService(mock)
		pair, err := svc.Login(domain.Credentials{Email: email, Password: password})

		if pair != nil {
			t.Fatalf("expected nil pair, got %+v", pair)
		}
		if err != providerErr {
			t.Fatalf("expected error %v, got %v", providerErr, err)
		}
	})
}

// Property 3: RefreshToken delega string exata e retorna par inalterado
func TestProperty_RefreshTokenDelegation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		refreshInput := rapid.String().Draw(t, "refreshToken")

		expectedPair := &token.Pair{
			AccessToken:           rapid.String().Draw(t, "accessToken"),
			AccessTokenExpiresAt:  time.Now().Add(15 * time.Minute),
			RefreshToken:          rapid.String().Draw(t, "newRefreshToken"),
			RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		var calledWith string
		mock := &mockTokenProvider{
			refreshFn: func(refreshToken string) (*token.Pair, error) {
				calledWith = refreshToken
				return expectedPair, nil
			},
		}

		svc := NewAuthService(mock)
		pair, err := svc.RefreshToken(refreshInput)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if calledWith != refreshInput {
			t.Fatalf("expected Refresh called with %q, got %q", refreshInput, calledWith)
		}
		if pair != expectedPair {
			t.Fatalf("expected returned pair to be exactly the mock pair")
		}
	})
}

// Property 4: RefreshToken propaga erros do TokenProvider
func TestProperty_RefreshTokenErrorPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		refreshInput := rapid.String().Draw(t, "refreshToken")
		errMsg := rapid.StringMatching(`[a-zA-Z ]{5,30}`).Draw(t, "errorMessage")

		providerErr := fmt.Errorf("%s", errMsg)

		mock := &mockTokenProvider{
			refreshFn: func(_ string) (*token.Pair, error) {
				return nil, providerErr
			},
		}

		svc := NewAuthService(mock)
		pair, err := svc.RefreshToken(refreshInput)

		if pair != nil {
			t.Fatalf("expected nil pair, got %+v", pair)
		}
		if err != providerErr {
			t.Fatalf("expected error %v, got %v", providerErr, err)
		}
	})
}
