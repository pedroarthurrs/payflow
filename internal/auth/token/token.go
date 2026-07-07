// Package token provides JWT creation and validation for payflow.
//
// It defines a Service that issues short-lived access tokens and long-lived
// refresh tokens, and validates each according to its expected type. Keeping
// this logic in its own package avoids coupling transport (middleware/handlers)
// to JWT internals and makes the behavior easy to test in isolation.
package token

import (
	"errors"
	"fmt"
	"time"

	"payflow/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType distinguishes access tokens from refresh tokens.
// Embedding the type inside the JWT payload prevents a refresh token from
// being accepted where only an access token is expected, and vice-versa.
type TokenType string

const (
	// TypeAccess is a short-lived token used to authenticate API requests.
	TypeAccess TokenType = "access"

	// TypeRefresh is a long-lived token used only to obtain a new token pair.
	TypeRefresh TokenType = "refresh"

	// TypeOrgSelect is a short-lived token (5 min) issued when a user belongs to
	// multiple organizations. The client must POST /v1/auth/select-organization
	// with this token to obtain a full access/refresh pair for the chosen org.
	TypeOrgSelect TokenType = "org_selection"

	issuer = "payflow"
)

// Claims is the JWT payload shared by both access and refresh tokens.
type Claims struct {
	// OrganizationID identifies the authenticated organization (stored in the "sub" field).
	OrganizationID string `json:"sub"`

	// TokenType indicates whether this is an access or refresh token.
	TokenType TokenType `json:"token_type"`

	// UserID is the authenticated user's UUID. Empty for system tokens.
	UserID string `json:"user_id,omitempty"`
	// MemberID is the organization_members.id UUID for the active membership. Empty for system tokens.
	MemberID string `json:"member_id,omitempty"`
	// Role is the user's role within the organization. Empty for system tokens.
	Role domain.Role `json:"role,omitempty"`
	// AuthType indicates the authentication mode (user or system). Always present.
	AuthType domain.AuthType `json:"auth_type"`

	jwt.RegisteredClaims
}

// ClaimsInput holds the data needed to issue a token pair with extended claims.
type ClaimsInput struct {
	OrganizationID string          // organization UUID
	UserID         string          // empty for system tokens
	MemberID       string          // organization_members.id UUID; empty for system tokens
	Role           domain.Role     // empty for system tokens
	AuthType       domain.AuthType // user or system
}

// Pair is returned on a successful login or token refresh.
// Callers must store the RefreshToken securely; it is long-lived and allows
// re-issuing access tokens without re-authenticating.
type Pair struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// Service handles JWT creation and validation. It is safe for concurrent use.
type Service struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewService creates a Service with the provided HMAC secret and token TTLs.
// Recommended values: accessTTL = 15 minutes, refreshTTL = 7 days.
func NewService(secret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// NewPair issues a new access/refresh token pair for the given organizationID.
// NewPair is kept for backward compatibility; it emits tokens without user/role claims.
func (s *Service) NewPair(organizationID string) (*Pair, error) {
	return s.NewPairWithClaims(ClaimsInput{OrganizationID: organizationID})
}

// NewPairWithClaims issues a new access/refresh token pair with extended claims.
func (s *Service) NewPairWithClaims(input ClaimsInput) (*Pair, error) {
	accessToken, accessExpiry, err := s.sign(input, TypeAccess, s.accessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, refreshExpiry, err := s.sign(input, TypeRefresh, s.refreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &Pair{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiry,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiry,
	}, nil
}

// Refresh validates a refresh token and issues a fresh token pair.
// Returns an error if the token is invalid, expired, or not of TypeRefresh.
func (s *Service) Refresh(refreshToken string) (*Pair, error) {
	claims, err := s.Verify(refreshToken, TypeRefresh)
	if err != nil {
		return nil, err
	}

	return s.NewPair(claims.OrganizationID)
}

// RefreshWithClaims validates a refresh token and issues a fresh token pair,
// preserving the UserID, Role, and AuthType from the original token.
func (s *Service) RefreshWithClaims(refreshToken string) (*Pair, error) {
	claims, err := s.Verify(refreshToken, TypeRefresh)
	if err != nil {
		return nil, err
	}

	return s.NewPairWithClaims(ClaimsInput{
		OrganizationID: claims.OrganizationID,
		UserID:         claims.UserID,
		Role:           claims.Role,
		AuthType:       claims.AuthType,
	})
}

// Verify parses and validates a JWT string, ensuring it matches the expected
// TokenType. Returns the parsed Claims on success.
func (s *Service) Verify(tokenString string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return s.secret, nil
		},
		jwt.WithIssuer(issuer),
		jwt.WithLeeway(5*time.Second),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("expected %s token, got %s", expectedType, claims.TokenType)
	}

	return claims, nil
}

// NewOrgSelectToken issues a 5-minute org_selection token for a user who
// authenticated successfully but belongs to multiple organizations.
// The JWT sub field is set to userAccountID; the client must present this token
// via POST /v1/auth/select-organization to obtain a full access/refresh pair.
func (s *Service) NewOrgSelectToken(userAccountID string) (string, error) {
	tok, _, err := s.sign(ClaimsInput{
		OrganizationID: userAccountID,
		AuthType:       domain.AuthTypeUser,
	}, TypeOrgSelect, 5*time.Minute)
	return tok, err
}

// sign creates and signs a JWT for the given ClaimsInput and TokenType.
func (s *Service) sign(input ClaimsInput, tokenType TokenType, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	expiry := now.Add(ttl)

	claims := Claims{
		OrganizationID: input.OrganizationID,
		TokenType:      tokenType,
		UserID:         input.UserID,
		MemberID:       input.MemberID,
		Role:           input.Role,
		AuthType:       input.AuthType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiry, nil
}
