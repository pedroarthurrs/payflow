package service

import (
	"context"
	"crypto/sha256"
	"fmt"

	"i9pay/internal/auth/token"
	"i9pay/internal/database"
	"i9pay/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// TokenProvider defines the token operations required by AuthService.
type TokenProvider interface {
	NewPair(tenantID string) (*token.Pair, error)
	NewPairWithClaims(input token.ClaimsInput) (*token.Pair, error)
	Refresh(refreshToken string) (*token.Pair, error)
	RefreshWithClaims(refreshToken string) (*token.Pair, error)
}

// AuthService handles authentication logic.
type AuthService struct {
	tokenSvc TokenProvider
	db       *database.DB
}

// NewAuthService creates an AuthService with token provider and database access.
func NewAuthService(tokenSvc TokenProvider, db *database.DB) *AuthService {
	return &AuthService{tokenSvc: tokenSvc, db: db}
}

// Login authenticates via dynamic mode detection:
//   - If req.APIKey is set → system mode (API Key)
//   - If req.Email + req.Password are set → user mode (tenant resolved from user record)
//   - Otherwise → ErrBadRequest
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*token.Pair, error) {
	if req.APIKey != "" {
		return s.loginByAPIKey(ctx, req.APIKey)
	}
	if req.Email != "" && req.Password != "" {
		return s.loginByUser(ctx, req.Email, req.Password)
	}
	return nil, domain.ErrBadRequest
}

// RefreshToken validates a refresh token and issues a new pair, preserving claims.
func (s *AuthService) RefreshToken(refreshToken string) (*token.Pair, error) {
	return s.tokenSvc.RefreshWithClaims(refreshToken)
}

// loginByAPIKey authenticates via SHA-256 hash of the API key.
func (s *AuthService) loginByAPIKey(ctx context.Context, apiKey string) (*token.Pair, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))

	key, err := s.db.Queries.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Update last_used_at — best effort, don't fail auth on error
	_ = s.db.Queries.UpdateAPIKeyLastUsed(ctx, key.ID)

	pair, err := s.tokenSvc.NewPairWithClaims(token.ClaimsInput{
		TenantID: key.TenantID.String(),
		AuthType: domain.AuthTypeSystem,
	})
	if err != nil {
		return nil, err
	}
	return pair, nil
}

// loginByUser authenticates a human user with email + password; tenant is resolved from the user record.
func (s *AuthService) loginByUser(ctx context.Context, email, password string) (*token.Pair, error) {
	user, err := s.db.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	pair, err := s.tokenSvc.NewPairWithClaims(token.ClaimsInput{
		TenantID: user.TenantID.String(),
		UserID:   user.ID.String(),
		Role:     domain.Role(user.Role),
		AuthType: domain.AuthTypeUser,
	})
	if err != nil {
		return nil, err
	}
	return pair, nil
}
