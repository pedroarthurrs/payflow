package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"payflow/internal/crypto"
	"payflow/internal/domain"

	"github.com/google/uuid"
)

// MemberService handles organization membership operations.
type MemberService struct {
	members  domain.MemberRepository
	accounts domain.UserAccountRepository
	logs     domain.MemberLogRepository
	ph       *crypto.PasswordHasher
}

// NewMemberService creates a MemberService.
func NewMemberService(members domain.MemberRepository) *MemberService {
	return &MemberService{members: members, ph: crypto.NewPasswordHasher(12)}
}

// NewMemberServiceWithAccounts creates a MemberService with user account creation support.
func NewMemberServiceWithAccounts(members domain.MemberRepository, accounts domain.UserAccountRepository) *MemberService {
	return &MemberService{members: members, accounts: accounts, ph: crypto.NewPasswordHasher(12)}
}

// NewMemberServiceFull creates a MemberService with user account creation and activity log support.
func NewMemberServiceFull(members domain.MemberRepository, accounts domain.UserAccountRepository, logs domain.MemberLogRepository) *MemberService {
	return &MemberService{members: members, accounts: accounts, logs: logs, ph: crypto.NewPasswordHasher(12)}
}

// InviteMember finds or creates a user account by email, then adds them as a member
// of the organization. Returns the membership and an optional temporary password —
// the temp password is only non-nil when a new user account was created.
// Returns ErrForbidden if trying to add a second owner.
// Returns ErrAlreadyMember if the user is already a member.
func (s *MemberService) InviteMember(ctx context.Context, orgID uuid.UUID, email, name string, role domain.Role, invitedBy *uuid.UUID) (*domain.OrganizationMember, *string, error) {
	if role == domain.RoleOwner {
		existing, _, err := s.members.ListMembers(ctx, orgID, domain.ListParams{Limit: 1000, Offset: 0})
		if err != nil {
			return nil, nil, err
		}
		for i := range existing {
			if existing[i].Role == domain.RoleOwner {
				return nil, nil, domain.ErrForbidden
			}
		}
	}

	var (
		userAccountID uuid.UUID
		tempPassword  *string
	)

	if s.accounts != nil {
		account, err := s.accounts.GetByEmail(ctx, email)
		if errors.Is(err, domain.ErrNotFound) {
			// Create a new account with a temporary password.
			tmp, err := generateTempPassword()
			if err != nil {
				return nil, nil, err
			}
			hash, err := s.ph.Hash(tmp)
			if err != nil {
				return nil, nil, err
			}
			account, err = s.accounts.Create(ctx, domain.CreateUserAccountInput{
				Email:        email,
				Name:         name,
				PasswordHash: hash,
			})
			if err != nil {
				return nil, nil, err
			}
			tempPassword = &tmp
		} else if err != nil {
			return nil, nil, err
		}
		userAccountID = account.ID
	}

	member, err := s.members.AddMember(ctx, orgID, userAccountID, role, invitedBy)
	if err != nil {
		return nil, nil, err
	}
	return member, tempPassword, nil
}

// AddMember adds a user account as a member of an organization by UUID.
// Returns ErrForbidden if trying to add a second owner.
func (s *MemberService) AddMember(ctx context.Context, orgID, userAccountID uuid.UUID, role domain.Role, invitedBy *uuid.UUID) (*domain.OrganizationMember, error) {
	if role == domain.RoleOwner {
		existing, _, err := s.members.ListMembers(ctx, orgID, domain.ListParams{Limit: 1000, Offset: 0})
		if err != nil {
			return nil, err
		}
		for i := range existing {
			if existing[i].Role == domain.RoleOwner {
				return nil, domain.ErrForbidden
			}
		}
	}
	return s.members.AddMember(ctx, orgID, userAccountID, role, invitedBy)
}

// GetMember returns the membership record for a user in an organization.
func (s *MemberService) GetMember(ctx context.Context, orgID, userAccountID uuid.UUID) (*domain.OrganizationMember, error) {
	return s.members.GetMember(ctx, orgID, userAccountID)
}

// ListMembers returns all active members of an organization.
func (s *MemberService) ListMembers(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.OrganizationMember, int32, error) {
	return s.members.ListMembers(ctx, orgID, params)
}

// UpdateMemberRole changes the role of a member within an organization.
// Returns ErrForbidden if trying to assign owner when one already exists.
func (s *MemberService) UpdateMemberRole(ctx context.Context, orgID, userAccountID uuid.UUID, role domain.Role) (*domain.OrganizationMember, error) {
	if role == domain.RoleOwner {
		existing, _, err := s.members.ListMembers(ctx, orgID, domain.ListParams{Limit: 1000, Offset: 0})
		if err != nil {
			return nil, err
		}
		for i := range existing {
			if existing[i].Role == domain.RoleOwner && existing[i].UserAccountID != userAccountID {
				return nil, domain.ErrForbidden
			}
		}
	}
	return s.members.UpdateMemberRole(ctx, orgID, userAccountID, role)
}

// RemoveMember removes a user account from an organization (soft delete).
func (s *MemberService) RemoveMember(ctx context.Context, orgID, userAccountID uuid.UUID) error {
	return s.members.RemoveMember(ctx, orgID, userAccountID)
}

// ListOrganizationsForUser returns all organizations the user account belongs to.
func (s *MemberService) ListOrganizationsForUser(ctx context.Context, userAccountID uuid.UUID) ([]domain.OrganizationRef, error) {
	return s.members.ListOrganizationsForUser(ctx, userAccountID)
}

// generateTempPassword returns a random 16-character hexadecimal string.
func generateTempPassword() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ListActivityLogs lists member activity logs for an organization, keyset-paginated.
// Returns (records, nextCursor, error). nextCursor is nil when no more pages.
func (s *MemberService) ListActivityLogs(ctx context.Context, orgID uuid.UUID, input domain.ListMemberLogsInput) ([]domain.MemberLogRecord, *string, error) {
	records, err := s.logs.List(ctx, orgID, input)
	if err != nil {
		return nil, nil, err
	}

	var nextCursor *string
	if len(records) == input.Limit {
		last := records[len(records)-1]
		cur := EncodeCursor(last.CreatedAt, last.ID)
		nextCursor = &cur
	}

	return records, nextCursor, nil
}

// GetActivityLogByID fetches a single member activity log by ID, scoped to the organization.
func (s *MemberService) GetActivityLogByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.MemberLogRecord, error) {
	return s.logs.GetByID(ctx, orgID, logID)
}
