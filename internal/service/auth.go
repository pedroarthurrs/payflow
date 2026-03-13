package service

import (
	"i9pay/internal/domain"
	"i9pay/internal/token"
)

// AuthService handles authentication logic.
type AuthService struct {
	tokenSvc *token.Service
}

// NewAuthService creates an AuthService.
func NewAuthService(tokenSvc *token.Service) *AuthService {
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
