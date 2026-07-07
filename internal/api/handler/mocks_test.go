package handler

import (
	"context"

	"payflow/internal/auth/token"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/google/uuid"
)

// authServicer is a test-only interface matching AuthService methods used by AuthHandler.
type authServicer interface {
	Login(ctx context.Context, req domain.LoginRequest) (*service.LoginResult, error)
	RefreshToken(refreshToken string) (*token.Pair, error)
	SelectOrganization(ctx context.Context, sessionToken string, orgID uuid.UUID) (*token.Pair, error)
}

// mockAuthService is a test double for authServicer.
type mockAuthService struct {
	loginFn              func(ctx context.Context, req domain.LoginRequest) (*service.LoginResult, error)
	refreshTokenFn       func(refreshToken string) (*token.Pair, error)
	selectOrganizationFn func(ctx context.Context, sessionToken string, orgID uuid.UUID) (*token.Pair, error)
}

func (m *mockAuthService) Login(ctx context.Context, req domain.LoginRequest) (*service.LoginResult, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, req)
	}
	return nil, domain.ErrUnauthorized
}

func (m *mockAuthService) RefreshToken(refreshToken string) (*token.Pair, error) {
	if m.refreshTokenFn != nil {
		return m.refreshTokenFn(refreshToken)
	}
	return nil, domain.ErrUnauthorized
}

func (m *mockAuthService) SelectOrganization(ctx context.Context, sessionToken string, orgID uuid.UUID) (*token.Pair, error) {
	if m.selectOrganizationFn != nil {
		return m.selectOrganizationFn(ctx, sessionToken, orgID)
	}
	return nil, domain.ErrInvalidSessionToken
}

// NOTE: withClaims is already defined in address_test.go in this package.
// This file only adds the authServicer mock specific to auth tests.
