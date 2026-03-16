package service

import (
	"i9pay/internal/domain"
	"i9pay/internal/token"
)

// TokenProvider defines the token operations required by AuthService.
type TokenProvider interface {
	NewPair(tenantID string) (*token.Pair, error)
	Refresh(refreshToken string) (*token.Pair, error)
}

// AuthService handles authentication logic.
type AuthService struct {
	tokenSvc TokenProvider
}

// NewAuthService creates an AuthService.
func NewAuthService(tokenSvc TokenProvider) *AuthService {
	return &AuthService{tokenSvc: tokenSvc}
}

// Login authenticates a tenant and returns a token pair.
func (s *AuthService) Login(creds domain.Credentials) (*token.Pair, error) {
	tenantID, err := s.authenticate(creds)
	if err != nil {
		return nil, err
	}

	return s.tokenSvc.NewPair(tenantID)
}

// RefreshToken validates a refresh token and issues a new pair.
func (s *AuthService) RefreshToken(refreshToken string) (*token.Pair, error) {
	return s.tokenSvc.Refresh(refreshToken)
}

// authenticate verifies tenant credentials.
// TODO: Replace with actual database lookup and password hashing.
func (s *AuthService) authenticate(creds domain.Credentials) (string, error) {
	return creds.Email + creds.Password, nil
}
