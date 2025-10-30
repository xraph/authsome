package organization

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service handles organization-related business logic
type Service struct {
	orgRepo        OrganizationRepository
	memberRepo     MemberRepository
	teamRepo       TeamRepository
	invitationRepo InvitationRepository
	config         Config
}

// Config holds the organization service configuration
type Config struct {
	PlatformOrganizationID     string `json:"platformOrganizationId"`
	DefaultOrganizationName    string `json:"defaultOrganizationName"`
	EnableOrganizationCreation bool   `json:"enableOrganizationCreation"`
	MaxMembersPerOrganization  int    `json:"maxMembersPerOrganization"`
	MaxTeamsPerOrganization    int    `json:"maxTeamsPerOrganization"`
	RequireInvitation          bool   `json:"requireInvitation"`
	InvitationExpiryHours      int    `json:"invitationExpiryHours"`
}

// Repository interfaces

// OrganizationRepository defines the interface for organization data access
type OrganizationRepository interface {
	Create(ctx context.Context, org *Organization) error
	FindByID(ctx context.Context, id string) (*Organization, error)
	FindBySlug(ctx context.Context, slug string) (*Organization, error)
	List(ctx context.Context, limit, offset int) ([]*Organization, error)
	Update(ctx context.Context, org *Organization) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

// MemberRepository defines the interface for member data access
type MemberRepository interface {
	Create(ctx context.Context, member *Member) error
	FindByID(ctx context.Context, id string) (*Member, error)
	FindByUserAndOrg(ctx context.Context, userID, orgID string) (*Member, error)
	ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*Member, error)
	ListByUser(ctx context.Context, userID string) ([]*Member, error)
	Update(ctx context.Context, member *Member) error
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	CountByOrganization(ctx context.Context, orgID string) (int, error)
}

// TeamRepository defines the interface for team data access
type TeamRepository interface {
	Create(ctx context.Context, team *Team) error
	FindByID(ctx context.Context, id string) (*Team, error)
	ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*Team, error)
	Update(ctx context.Context, team *Team) error
	Delete(ctx context.Context, id string) error
	CountByOrganization(ctx context.Context, orgID string) (int, error)
	AddMember(ctx context.Context, teamID, memberID, role string) error
	RemoveMember(ctx context.Context, teamID, memberID string) error
	ListMembers(ctx context.Context, teamID string) ([]*Member, error)
}

// InvitationRepository defines the interface for invitation data access
type InvitationRepository interface {
	Create(ctx context.Context, invitation *Invitation) error
	FindByID(ctx context.Context, id string) (*Invitation, error)
	FindByToken(ctx context.Context, token string) (*Invitation, error)
	ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*Invitation, error)
	Update(ctx context.Context, invitation *Invitation) error
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
}

// NewService creates a new organization service
func NewService(
	config Config,
	orgRepo OrganizationRepository,
	memberRepo MemberRepository,
	teamRepo TeamRepository,
	inviteRepo InvitationRepository,
) *Service {
	return &Service{
		config:         config,
		orgRepo:        orgRepo,
		memberRepo:     memberRepo,
		teamRepo:       teamRepo,
		invitationRepo: inviteRepo,
	}
}

// Organization management

// CreateOrganization creates a new organization
func (s *Service) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, creatorUserID string) (*Organization, error) {
	if !s.config.EnableOrganizationCreation {
		return nil, fmt.Errorf("organization creation is disabled")
	}

	// Check if slug is already taken
	existing, err := s.orgRepo.FindBySlug(ctx, req.Slug)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("organization slug already exists")
	}

	// Create organization
	org := &Organization{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Slug:      req.Slug,
		Logo:      req.Logo,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Add creator as owner
	_, err = s.AddMember(ctx, org.ID, creatorUserID, RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("failed to add creator as owner: %w", err)
	}

	return org, nil
}

// GetOrganization retrieves an organization by ID
func (s *Service) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	return s.orgRepo.FindByID(ctx, id)
}

// GetOrganizationBySlug retrieves an organization by slug
func (s *Service) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	return s.orgRepo.FindBySlug(ctx, slug)
}

// ListOrganizations lists organizations with pagination
func (s *Service) ListOrganizations(ctx context.Context, limit, offset int) ([]*Organization, error) {
	return s.orgRepo.List(ctx, limit, offset)
}

// UpdateOrganization updates an organization
func (s *Service) UpdateOrganization(ctx context.Context, id string, req *UpdateOrganizationRequest) (*Organization, error) {
	org, err := s.orgRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	// Update fields
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Logo != nil {
		org.Logo = req.Logo
	}
	if req.Metadata != nil {
		org.Metadata = req.Metadata
	}
	org.UpdatedAt = time.Now()

	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// DeleteOrganization deletes an organization
func (s *Service) DeleteOrganization(ctx context.Context, id string) error {
	return s.orgRepo.Delete(ctx, id)
}

// GetDefaultOrganization returns the default organization for standalone mode
func (s *Service) GetDefaultOrganization(ctx context.Context) (*Organization, error) {
	if s.config.PlatformOrganizationID != "" {
		return s.orgRepo.FindByID(ctx, s.config.PlatformOrganizationID)
	}

	// Find or create default organization
	orgs, err := s.orgRepo.List(ctx, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	if len(orgs) > 0 {
		return orgs[0], nil
	}

	// Create default organization
	defaultOrg := &Organization{
		ID:        uuid.New().String(),
		Name:      s.config.DefaultOrganizationName,
		Slug:      "default",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.orgRepo.Create(ctx, defaultOrg); err != nil {
		return nil, fmt.Errorf("failed to create default organization: %w", err)
	}

	return defaultOrg, nil
}

// Member management

// AddMember adds a user as a member of an organization
func (s *Service) AddMember(ctx context.Context, orgID, userID, role string) (*Member, error) {
	// Check if user is already a member
	existing, err := s.memberRepo.FindByUserAndOrg(ctx, userID, orgID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("user is already a member of this organization")
	}

	// Check member limit
	count, err := s.memberRepo.CountByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}
	if count >= s.config.MaxMembersPerOrganization {
		return nil, fmt.Errorf("organization has reached maximum member limit")
	}

	member := &Member{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		Status:         StatusActive,
		JoinedAt:       time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return member, nil
}

// GetMember retrieves a member by ID
func (s *Service) GetMember(ctx context.Context, id string) (*Member, error) {
	return s.memberRepo.FindByID(ctx, id)
}

// ListMembers lists members of an organization
func (s *Service) ListMembers(ctx context.Context, orgID string, limit, offset int) ([]*Member, error) {
	return s.memberRepo.ListByOrganization(ctx, orgID, limit, offset)
}

// UpdateMember updates a member
func (s *Service) UpdateMember(ctx context.Context, id string, req *UpdateMemberRequest) (*Member, error) {
	member, err := s.memberRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("member not found: %w", err)
	}

	if req.Role != nil {
		member.Role = *req.Role
	}
	if req.Status != nil {
		member.Status = *req.Status
	}
	member.UpdatedAt = time.Now()

	if err := s.memberRepo.Update(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	return member, nil
}

// RemoveMember removes a member from an organization
func (s *Service) RemoveMember(ctx context.Context, id string) error {
	return s.memberRepo.Delete(ctx, id)
}

// IsUserMember checks if a user is a member of an organization
func (s *Service) IsUserMember(ctx context.Context, orgID, userID string) (bool, error) {
	member, err := s.memberRepo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return false, nil
	}
	return member != nil && member.Status == StatusActive, nil
}

// GetUserMemberships returns all organizations a user is a member of
func (s *Service) GetUserMemberships(ctx context.Context, userID string) ([]*Member, error) {
	return s.memberRepo.ListByUser(ctx, userID)
}

// RemoveUserFromAllOrganizations removes a user from all organizations
func (s *Service) RemoveUserFromAllOrganizations(ctx context.Context, userID string) error {
	return s.memberRepo.DeleteByUserID(ctx, userID)
}

// Team management

// CreateTeam creates a new team in an organization
func (s *Service) CreateTeam(ctx context.Context, orgID string, req *CreateTeamRequest) (*Team, error) {
	// Check team limit
	count, err := s.teamRepo.CountByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count teams: %w", err)
	}
	if count >= s.config.MaxTeamsPerOrganization {
		return nil, fmt.Errorf("organization has reached maximum team limit")
	}

	team := &Team{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.teamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	return team, nil
}

// GetTeam retrieves a team by ID
func (s *Service) GetTeam(ctx context.Context, id string) (*Team, error) {
	return s.teamRepo.FindByID(ctx, id)
}

// ListTeams lists teams in an organization
func (s *Service) ListTeams(ctx context.Context, orgID string, limit, offset int) ([]*Team, error) {
	return s.teamRepo.ListByOrganization(ctx, orgID, limit, offset)
}

// UpdateTeam updates a team
func (s *Service) UpdateTeam(ctx context.Context, id string, req *UpdateTeamRequest) (*Team, error) {
	team, err := s.teamRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	if req.Name != nil {
		team.Name = *req.Name
	}
	if req.Description != nil {
		team.Description = req.Description
	}
	if req.Metadata != nil {
		team.Metadata = req.Metadata
	}
	team.UpdatedAt = time.Now()

	if err := s.teamRepo.Update(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	return team, nil
}

// DeleteTeam deletes a team
func (s *Service) DeleteTeam(ctx context.Context, id string) error {
	return s.teamRepo.Delete(ctx, id)
}

// AddTeamMember adds a member to a team
func (s *Service) AddTeamMember(ctx context.Context, teamID, memberID, role string) error {
	return s.teamRepo.AddMember(ctx, teamID, memberID, role)
}

// RemoveTeamMember removes a member from a team
func (s *Service) RemoveTeamMember(ctx context.Context, teamID, memberID string) error {
	return s.teamRepo.RemoveMember(ctx, teamID, memberID)
}

// ListTeamMembers lists members of a team
func (s *Service) ListTeamMembers(ctx context.Context, teamID string) ([]*Member, error) {
	return s.teamRepo.ListMembers(ctx, teamID)
}

// Invitation management

// InviteMember creates an invitation for a user to join an organization
func (s *Service) InviteMember(ctx context.Context, orgID string, req *InviteMemberRequest, inviterUserID string) (*Invitation, error) {
	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	invitation := &Invitation{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Email:          req.Email,
		Role:           req.Role,
		Token:          token,
		Status:         InvitationStatusPending,
		InvitedBy:      inviterUserID,
		Metadata:       req.Metadata,
		ExpiresAt:      time.Now().Add(time.Duration(s.config.InvitationExpiryHours) * time.Hour),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	return invitation, nil
}

// GetInvitation retrieves an invitation by token
func (s *Service) GetInvitation(ctx context.Context, token string) (*Invitation, error) {
	invitation, err := s.invitationRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invitation not found: %w", err)
	}

	// Check if expired
	if time.Now().After(invitation.ExpiresAt) {
		invitation.Status = InvitationStatusExpired
		s.invitationRepo.Update(ctx, invitation)
		return nil, fmt.Errorf("invitation has expired")
	}

	return invitation, nil
}

// AcceptInvitation accepts an invitation and adds the user to the organization
func (s *Service) AcceptInvitation(ctx context.Context, token, userID string) (*Member, error) {
	invitation, err := s.GetInvitation(ctx, token)
	if err != nil {
		return nil, err
	}

	if invitation.Status != InvitationStatusPending {
		return nil, fmt.Errorf("invitation is not pending")
	}

	// Add user as member
	member, err := s.AddMember(ctx, invitation.OrganizationID, userID, invitation.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	// Update invitation status
	invitation.Status = InvitationStatusAccepted
	invitation.UpdatedAt = time.Now()
	s.invitationRepo.Update(ctx, invitation)

	return member, nil
}

// DeclineInvitation declines an invitation
func (s *Service) DeclineInvitation(ctx context.Context, token string) error {
	invitation, err := s.GetInvitation(ctx, token)
	if err != nil {
		return err
	}

	invitation.Status = InvitationStatusDeclined
	invitation.UpdatedAt = time.Now()
	return s.invitationRepo.Update(ctx, invitation)
}

// ListInvitations lists invitations for an organization
func (s *Service) ListInvitations(ctx context.Context, orgID string, limit, offset int) ([]*Invitation, error) {
	return s.invitationRepo.ListByOrganization(ctx, orgID, limit, offset)
}

// CleanupExpiredInvitations removes expired invitations
func (s *Service) CleanupExpiredInvitations(ctx context.Context) error {
	return s.invitationRepo.DeleteExpired(ctx)
}

// Helper functions

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
