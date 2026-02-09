package app

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/internal/utils"
)

// InvitationService handles invitation aggregate operations.
type InvitationService struct {
	repo       InvitationRepository
	memberRepo MemberRepository // For validation
	memberSvc  *MemberService   // For creating members with RBAC sync
	appRepo    AppRepository    // For validation
	config     Config
	rbacSvc    *rbac.Service
}

// NewInvitationService creates a new invitation service.
func NewInvitationService(
	repo InvitationRepository,
	memberRepo MemberRepository,
	memberSvc *MemberService, // NEW: For RBAC-synced member creation
	appRepo AppRepository,
	cfg Config,
	rbacSvc *rbac.Service,
) *InvitationService {
	return &InvitationService{
		repo:       repo,
		memberRepo: memberRepo,
		memberSvc:  memberSvc,
		appRepo:    appRepo,
		config:     cfg,
		rbacSvc:    rbacSvc,
	}
}

// CreateInvitation creates an app invitation.
func (s *InvitationService) CreateInvitation(ctx context.Context, inv *Invitation) error {
	if inv.Status == "" {
		inv.Status = InvitationStatusPending
	}

	inv.CreatedAt = time.Now()

	return s.repo.CreateInvitation(ctx, inv.ToSchema())
}

// FindInvitationByID finds an invitation by ID.
func (s *InvitationService) FindInvitationByID(ctx context.Context, id xid.ID) (*Invitation, error) {
	schemaInv, err := s.repo.FindInvitationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find invitation: %w", err)
	}

	return FromSchemaInvitation(schemaInv), nil
}

// FindInvitationByToken finds an invitation by token.
func (s *InvitationService) FindInvitationByToken(ctx context.Context, token string) (*Invitation, error) {
	schemaInv, err := s.repo.FindInvitationByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to find invitation by token: %w", err)
	}

	return FromSchemaInvitation(schemaInv), nil
}

// ListInvitations lists invitations for an app with pagination.
func (s *InvitationService) ListInvitations(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*Invitation], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	response, err := s.repo.ListInvitations(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}

	// Convert schema invitations to DTOs
	invitations := FromSchemaInvitations(response.Data)

	return &pagination.PageResponse[*Invitation]{
		Data:       invitations,
		Pagination: response.Pagination,
	}, nil
}

// AcceptInvitation accepts an invitation and creates a member.
func (s *InvitationService) AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*Member, error) {
	// Find invitation by token
	schemaInv, err := s.repo.FindInvitationByToken(ctx, token)
	if err != nil {
		return nil, InvitationNotFound().WithError(err)
	}

	// Validate invitation is not expired
	if err := validateInvitationExpiry(schemaInv); err != nil {
		// Update status to expired
		schemaInv.Status = InvitationStatusExpired
		_ = s.repo.UpdateInvitation(ctx, schemaInv)

		return nil, err
	}

	// Validate invitation is pending
	if err := validateInvitationStatus(schemaInv, InvitationStatusPending); err != nil {
		return nil, err
	}

	// Check if member already exists
	if err := s.validateMemberDoesNotExist(ctx, schemaInv.AppID, userID); err != nil {
		return nil, err
	}

	// Check member limit
	if err := s.validateMemberLimit(ctx, schemaInv.AppID); err != nil {
		return nil, err
	}

	// Create member using MemberService to ensure RBAC sync
	member := &Member{
		ID:       xid.New(),
		AppID:    schemaInv.AppID,
		UserID:   userID,
		Role:     schemaInv.Role,
		Status:   MemberStatusActive,
		JoinedAt: time.Now(),
	}

	// Use MemberService.CreateMember to ensure RBAC UserRole entry is created
	createdMember, err := s.memberSvc.CreateMember(ctx, member)
	if err != nil {
		return nil, fmt.Errorf("failed to create member: %w", err)
	}

	// Update invitation status
	schemaInv.Status = InvitationStatusAccepted
	now := time.Now()
	schemaInv.AcceptedAt = &now
	schemaInv.UpdatedAt = now

	if err := s.repo.UpdateInvitation(ctx, schemaInv); err != nil {
		// Log error but don't fail the operation
		// Member has already been created
		return createdMember, nil
	}

	return createdMember, nil
}

// DeclineInvitation declines an invitation.
func (s *InvitationService) DeclineInvitation(ctx context.Context, token string) error {
	// Find invitation by token
	schemaInv, err := s.repo.FindInvitationByToken(ctx, token)
	if err != nil {
		return InvitationNotFound().WithError(err)
	}

	// Validate invitation is not expired
	if err := validateInvitationExpiry(schemaInv); err != nil {
		// Update status to expired
		schemaInv.Status = InvitationStatusExpired
		_ = s.repo.UpdateInvitation(ctx, schemaInv)

		return err
	}

	// Validate invitation is pending
	if err := validateInvitationStatus(schemaInv, InvitationStatusPending); err != nil {
		return err
	}

	// Update invitation status
	schemaInv.Status = InvitationStatusDeclined
	schemaInv.UpdatedAt = time.Now()

	if err := s.repo.UpdateInvitation(ctx, schemaInv); err != nil {
		return fmt.Errorf("failed to decline invitation: %w", err)
	}

	return nil
}

// CancelInvitation cancels an invitation (only admins/owners).
func (s *InvitationService) CancelInvitation(ctx context.Context, id, cancellerUserID xid.ID) error {
	// Find invitation
	schemaInv, err := s.repo.FindInvitationByID(ctx, id)
	if err != nil {
		return InvitationNotFound().WithError(err)
	}

	// Check authorization - only admins can cancel
	if err := s.requireAdmin(ctx, schemaInv.AppID, cancellerUserID); err != nil {
		return fmt.Errorf("unauthorized to cancel invitation: %w", err)
	}

	// Validate invitation is pending
	if err := validateInvitationStatus(schemaInv, InvitationStatusPending); err != nil {
		return err
	}

	// Update invitation status
	schemaInv.Status = InvitationStatusCancelled
	schemaInv.UpdatedAt = time.Now()

	if err := s.repo.UpdateInvitation(ctx, schemaInv); err != nil {
		return fmt.Errorf("failed to cancel invitation: %w", err)
	}

	return nil
}

// ResendInvitation resends an invitation by creating a new token and updating expiry.
func (s *InvitationService) ResendInvitation(ctx context.Context, id, resenderUserID xid.ID) (*Invitation, error) {
	// Find invitation
	schemaInv, err := s.repo.FindInvitationByID(ctx, id)
	if err != nil {
		return nil, InvitationNotFound().WithError(err)
	}

	// Check authorization - only admins can resend
	if err := s.requireAdmin(ctx, schemaInv.AppID, resenderUserID); err != nil {
		return nil, fmt.Errorf("unauthorized to resend invitation: %w", err)
	}

	// Validate invitation is pending or expired
	if schemaInv.Status != InvitationStatusPending && schemaInv.Status != InvitationStatusExpired {
		return nil, fmt.Errorf("invitation cannot be resent in status: %s", schemaInv.Status)
	}

	// Generate new token
	token, err := utils.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Update invitation
	schemaInv.Token = token
	schemaInv.Status = InvitationStatusPending
	schemaInv.ExpiresAt = time.Now().Add(time.Duration(s.config.InvitationExpiryHours) * time.Hour)
	schemaInv.UpdatedAt = time.Now()
	schemaInv.AcceptedAt = nil // Reset accepted timestamp

	if err := s.repo.UpdateInvitation(ctx, schemaInv); err != nil {
		return nil, fmt.Errorf("failed to resend invitation: %w", err)
	}

	return FromSchemaInvitation(schemaInv), nil
}

// CleanupExpiredInvitations removes expired invitations.
func (s *InvitationService) CleanupExpiredInvitations(ctx context.Context) (int, error) {
	count, err := s.repo.DeleteExpiredInvitations(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired invitations: %w", err)
	}

	return count, nil
}

// validateMemberDoesNotExist checks if a member already exists for an app and user.
func (s *InvitationService) validateMemberDoesNotExist(ctx context.Context, appID, userID xid.ID) error {
	member, err := s.memberRepo.FindMember(ctx, appID, userID)
	if err == nil && member != nil {
		return MemberAlreadyExists(userID.String())
	}

	return nil
}

// validateMemberLimit checks if the app has reached the maximum number of members.
func (s *InvitationService) validateMemberLimit(ctx context.Context, appID xid.ID) error {
	if s.config.MaxMembersPerApp <= 0 {
		return nil // No limit
	}

	count, err := s.memberRepo.CountMembers(ctx, appID)
	if err != nil {
		return fmt.Errorf("failed to count members: %w", err)
	}

	if count >= s.config.MaxMembersPerApp {
		return MaxMembersReached(s.config.MaxMembersPerApp)
	}

	return nil
}

// requireAdmin checks if a user is an admin or owner and returns an error if not.
func (s *InvitationService) requireAdmin(ctx context.Context, appID, userID xid.ID) error {
	member, err := s.memberRepo.FindMember(ctx, appID, userID)
	if err != nil {
		return NotAdmin()
	}

	if member == nil {
		return NotAdmin()
	}

	isAdminOrOwner := (member.Role == MemberRoleOwner || member.Role == MemberRoleAdmin) &&
		member.Status == MemberStatusActive
	if !isAdminOrOwner {
		return NotAdmin()
	}

	return nil
}

// Type assertion to ensure InvitationService implements InvitationOperations.
var _ InvitationOperations = (*InvitationService)(nil)
