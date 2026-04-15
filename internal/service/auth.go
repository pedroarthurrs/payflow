package service

import (
	"context"

	"i9pay/internal/auth/token"
	"i9pay/internal/crypto"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// TokenProvider defines the token operations required by AuthService.
type TokenProvider interface {
	NewPair(organizationID string) (*token.Pair, error)
	NewPairWithClaims(input token.ClaimsInput) (*token.Pair, error)
	Refresh(refreshToken string) (*token.Pair, error)
	RefreshWithClaims(refreshToken string) (*token.Pair, error)
	// NewOrgSelectToken issues a 5-min org_selection JWT for multi-org login.
	NewOrgSelectToken(userAccountID string) (string, error)
	// Verify validates a JWT and checks its TokenType.
	Verify(tokenString string, expectedType token.TokenType) (*token.Claims, error)
}

// LoginResult is the discriminated union returned by AuthService.Login.
// Exactly one branch is populated:
//   - Single-org:  TokenPair is set, Organizations is nil.
//   - Multi-org:   SessionToken + Organizations are set, TokenPair is nil.
type LoginResult struct {
	// Single-org: a ready-to-use access/refresh token pair.
	TokenPair *token.Pair

	// Multi-org: a short-lived org_selection JWT (5 min). Client must POST
	// /v1/auth/select-organization with this token to get a full pair.
	SessionToken  *string
	Organizations []domain.AuthMembership
}

// AuthService handles authentication logic.
type AuthService struct {
	tokenSvc       TokenProvider
	auth           domain.AuthRepository
	hasher         *crypto.Hasher
	passwordHasher *crypto.PasswordHasher
}

// NewAuthService creates an AuthService with token provider and auth repository.
func NewAuthService(tokenSvc TokenProvider, auth domain.AuthRepository) *AuthService {
	return &AuthService{
		tokenSvc:       tokenSvc,
		auth:           auth,
		hasher:         crypto.NewHasher(),
		passwordHasher: crypto.NewPasswordHasher(12),
	}
}

// Login authenticates via dynamic mode detection:
//   - If req.APIKey is set → system mode (API Key) → always returns a TokenPair.
//   - If req.Email + req.Password are set → user mode:
//     0 orgs:  ErrUnauthorized
//     1 org:   returns LoginResult{TokenPair}
//     N orgs:  returns LoginResult{SessionToken, Organizations}
//   - Otherwise → ErrBadRequest
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*LoginResult, error) {
	if req.APIKey != "" {
		pair, err := s.loginByAPIKey(ctx, req.APIKey)
		if err != nil {
			return nil, err
		}
		return &LoginResult{TokenPair: pair}, nil
	}
	if req.Email != "" && req.Password != "" {
		return s.loginByUser(ctx, req.Email, req.Password)
	}
	return nil, domain.ErrBadRequest
}

// SelectOrganization validates an org_selection session token and issues a full
// access/refresh pair for the chosen organization.
func (s *AuthService) SelectOrganization(ctx context.Context, sessionToken string, orgID uuid.UUID) (*token.Pair, error) {
	claims, err := s.tokenSvc.Verify(sessionToken, token.TypeOrgSelect)
	if err != nil {
		return nil, domain.ErrInvalidSessionToken
	}

	userAccountID, err := uuid.Parse(claims.OrganizationID)
	if err != nil {
		return nil, domain.ErrInvalidSessionToken
	}

	memberships, err := s.auth.ListMembershipsForUser(ctx, userAccountID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	var matched *domain.AuthMembership
	for i := range memberships {
		if memberships[i].OrganizationID == orgID {
			matched = &memberships[i]
			break
		}
	}
	if matched == nil {
		return nil, domain.ErrForbidden
	}

	return s.tokenSvc.NewPairWithClaims(token.ClaimsInput{
		OrganizationID: orgID.String(),
		UserID:         userAccountID.String(),
		MemberID:       matched.MemberID.String(),
		Role:           matched.Role,
		AuthType:       domain.AuthTypeUser,
	})
}

// RefreshToken validates a refresh token and issues a new pair, preserving claims.
func (s *AuthService) RefreshToken(refreshToken string) (*token.Pair, error) {
	return s.tokenSvc.RefreshWithClaims(refreshToken)
}

// loginByAPIKey authenticates via SHA-256 hash of the API key.
func (s *AuthService) loginByAPIKey(ctx context.Context, apiKey string) (*token.Pair, error) {
	hash := s.hasher.SHA256(apiKey)

	key, err := s.auth.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Update last_used_at — best effort, don't fail auth on error
	_ = s.auth.UpdateAPIKeyLastUsed(ctx, key.ID)

	pair, err := s.tokenSvc.NewPairWithClaims(token.ClaimsInput{
		OrganizationID: key.OrganizationID.String(),
		AuthType:       domain.AuthTypeSystem,
	})
	if err != nil {
		return nil, err
	}
	return pair, nil
}

// loginByUser authenticates a human user with email + password, then branches on
// the number of org memberships.
func (s *AuthService) loginByUser(ctx context.Context, email, password string) (*LoginResult, error) {
	account, err := s.auth.GetUserAccountByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if err := s.passwordHasher.Compare(account.Password, password); err != nil {
		return nil, domain.ErrUnauthorized
	}

	memberships, err := s.auth.ListMembershipsForUser(ctx, account.ID)
	if err != nil || len(memberships) == 0 {
		return nil, domain.ErrUnauthorized
	}

	if len(memberships) == 1 {
		m := memberships[0]
		pair, err := s.tokenSvc.NewPairWithClaims(token.ClaimsInput{
			OrganizationID: m.OrganizationID.String(),
			UserID:         account.ID.String(),
			MemberID:       m.MemberID.String(),
			Role:           m.Role,
			AuthType:       domain.AuthTypeUser,
		})
		if err != nil {
			return nil, err
		}
		return &LoginResult{TokenPair: pair}, nil
	}

	// Multi-org: issue a short-lived session token.
	sessionToken, err := s.tokenSvc.NewOrgSelectToken(account.ID.String())
	if err != nil {
		return nil, err
	}
	return &LoginResult{SessionToken: &sessionToken, Organizations: memberships}, nil
}
