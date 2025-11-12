package organization

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Service handles organization-related business logic
type Service struct {
	orgRepo        OrganizationRepository
	memberRepo     OrganizationMemberRepository
	teamRepo       OrganizationTeamRepository
	invitationRepo OrganizationInvitationRepository
	config         Config
}

// Config holds the organization service configuration
type Config struct {
	MaxOrganizationsPerUser   int  `json:"maxOrganizationsPerUser"`
	MaxMembersPerOrganization int  `json:"maxMembersPerOrganization"`
	MaxTeamsPerOrganization   int  `json:"maxTeamsPerOrganization"`
	EnableUserCreation        bool `json:"enableUserCreation"`
	RequireInvitation         bool `json:"requireInvitation"`
	InvitationExpiryHours     int  `json:"invitationExpiryHours"`
}

// NewService creates a new organization service
func NewService(
	config Config,
	orgRepo OrganizationRepository,
	memberRepo OrganizationMemberRepository,
	teamRepo OrganizationTeamRepository,
	inviteRepo OrganizationInvitationRepository,
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

// CreateOrganization creates a new user-created organization
func (s *Service) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, creatorUserID, appID, environmentID xid.ID) (*schema.Organization, error) {
	// Check if user creation is enabled
	if !s.config.EnableUserCreation {
		return nil, fmt.Errorf("organization creation is disabled")
	}

	// Check user's organization limit
	count, err := s.orgRepo.CountByUser(ctx, creatorUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user organizations: %w", err)
	}
	if count >= s.config.MaxOrganizationsPerUser {
		return nil, ErrMaxOrganizationsReached
	}

	// Check if slug is already taken within this app+environment
	existing, err := s.orgRepo.FindBySlug(ctx, appID, environmentID, req.Slug)
	if err == nil && existing != nil {
		return nil, ErrSlugAlreadyExists
	}

	// Handle logo (convert pointer to string)
	logo := ""
	if req.Logo != nil {
		logo = *req.Logo
	}

	// Create organization
	org := &schema.Organization{
		ID:            xid.New(),
		AppID:         appID,
		EnvironmentID: environmentID,
		Name:          req.Name,
		Slug:          req.Slug,
		Logo:          logo,
		Metadata:      req.Metadata,
		CreatedBy:     creatorUserID,
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
func (s *Service) GetOrganization(ctx context.Context, id xid.ID) (*schema.Organization, error) {
	return s.orgRepo.FindByID(ctx, id)
}

// GetOrganizationBySlug retrieves an organization by slug
func (s *Service) GetOrganizationBySlug(ctx context.Context, appID, environmentID xid.ID, slug string) (*schema.Organization, error) {
	return s.orgRepo.FindBySlug(ctx, appID, environmentID, slug)
}

// ListOrganizations lists organizations with pagination
func (s *Service) ListOrganizations(ctx context.Context, appID, environmentID xid.ID, limit, offset int) ([]*schema.Organization, error) {
	return s.orgRepo.ListByApp(ctx, appID, environmentID, limit, offset)
}

// ListUserOrganizations lists organizations a user is a member of
func (s *Service) ListUserOrganizations(ctx context.Context, userID xid.ID, limit, offset int) ([]*schema.Organization, error) {
	return s.orgRepo.ListByUser(ctx, userID, limit, offset)
}

// UpdateOrganization updates an organization
func (s *Service) UpdateOrganization(ctx context.Context, id xid.ID, req *UpdateOrganizationRequest) (*schema.Organization, error) {
	org, err := s.orgRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	// Update fields
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Logo != nil {
		org.Logo = *req.Logo
	}
	if req.Metadata != nil {
		org.Metadata = req.Metadata
	}

	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// DeleteOrganization deletes an organization
func (s *Service) DeleteOrganization(ctx context.Context, id, userID xid.ID) error {
	// Verify user is the owner
	member, err := s.memberRepo.FindByUserAndOrg(ctx, userID, id)
	if err != nil || member == nil || member.Role != RoleOwner {
		return ErrNotOrganizationOwner
	}

	return s.orgRepo.Delete(ctx, id)
}

// Member management

// AddMember adds a user as a member of an organization
func (s *Service) AddMember(ctx context.Context, orgID, userID xid.ID, role string) (*schema.OrganizationMember, error) {
	// Check if user is already a member
	existing, err := s.memberRepo.FindByUserAndOrg(ctx, userID, orgID)
	if err == nil && existing != nil {
		return nil, ErrMemberAlreadyExists
	}

	// Check member limit
	count, err := s.memberRepo.CountByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}
	if count >= s.config.MaxMembersPerOrganization {
		return nil, fmt.Errorf("organization has reached maximum member limit")
	}

	member := &schema.OrganizationMember{
		ID:             xid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		Status:         StatusActive,
		JoinedAt:       time.Now(),
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return member, nil
}

// GetMember retrieves a member by ID
func (s *Service) GetMember(ctx context.Context, id xid.ID) (*schema.OrganizationMember, error) {
	return s.memberRepo.FindByID(ctx, id)
}

// IsMember checks if a user is a member of an organization
func (s *Service) IsMember(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	member, err := s.memberRepo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return false, nil // User is not a member (or error occurred)
	}
	return member != nil && member.Status == StatusActive, nil
}

// IsOwner checks if a user is the owner of an organization
func (s *Service) IsOwner(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	member, err := s.memberRepo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return false, nil
	}
	return member != nil && member.Role == RoleOwner && member.Status == StatusActive, nil
}

// IsAdmin checks if a user is an admin or owner of an organization
func (s *Service) IsAdmin(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	member, err := s.memberRepo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return false, nil
	}
	return member != nil && (member.Role == RoleOwner || member.Role == RoleAdmin) && member.Status == StatusActive, nil
}

// ListMembers lists members of an organization
func (s *Service) ListMembers(ctx context.Context, orgID xid.ID, limit, offset int) ([]*schema.OrganizationMember, error) {
	return s.memberRepo.ListByOrganization(ctx, orgID, limit, offset)
}

// UpdateMember updates a member
func (s *Service) UpdateMember(ctx context.Context, id xid.ID, req *UpdateMemberRequest, updaterUserID xid.ID) (*schema.OrganizationMember, error) {
	member, err := s.memberRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("member not found: %w", err)
	}

	// Verify updater is admin or owner
	isAdmin, err := s.IsAdmin(ctx, member.OrganizationID, updaterUserID)
	if err != nil || !isAdmin {
		return nil, fmt.Errorf("only admins and owners can update members")
	}

	// Cannot change owner role
	if member.Role == RoleOwner {
		return nil, ErrCannotRemoveOwner
	}

	if req.Role != nil {
		member.Role = *req.Role
	}
	if req.Status != nil {
		member.Status = *req.Status
	}

	if err := s.memberRepo.Update(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	return member, nil
}

// RemoveMember removes a member from an organization
func (s *Service) RemoveMember(ctx context.Context, id, removerUserID xid.ID) error {
	member, err := s.memberRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}

	// Cannot remove owner
	if member.Role == RoleOwner {
		return ErrCannotRemoveOwner
	}

	// Verify remover is admin or owner
	isAdmin, err := s.IsAdmin(ctx, member.OrganizationID, removerUserID)
	if err != nil || !isAdmin {
		return fmt.Errorf("only admins and owners can remove members")
	}

	return s.memberRepo.Delete(ctx, id)
}

// GetUserMemberships returns all organizations a user is a member of
func (s *Service) GetUserMemberships(ctx context.Context, userID xid.ID, limit, offset int) ([]*schema.OrganizationMember, error) {
	return s.memberRepo.ListByUser(ctx, userID, limit, offset)
}

// RemoveUserFromAllOrganizations removes a user from all organizations they belong to
func (s *Service) RemoveUserFromAllOrganizations(ctx context.Context, userID xid.ID) error {
	// Get all memberships
	memberships, err := s.memberRepo.ListByUser(ctx, userID, 1000, 0) // Get all
	if err != nil {
		return fmt.Errorf("failed to get user memberships: %w", err)
	}

	// Delete each membership
	for _, membership := range memberships {
		if err := s.memberRepo.DeleteByUserAndOrg(ctx, userID, membership.OrganizationID); err != nil {
			return fmt.Errorf("failed to remove membership: %w", err)
		}
	}

	return nil
}

// Team management

// CreateTeam creates a new team in an organization
func (s *Service) CreateTeam(ctx context.Context, orgID xid.ID, req *CreateTeamRequest, creatorUserID xid.ID) (*schema.OrganizationTeam, error) {
	// Verify creator is member
	isMember, err := s.IsMember(ctx, orgID, creatorUserID)
	if err != nil || !isMember {
		return nil, fmt.Errorf("only organization members can create teams")
	}

	// Check team limit
	count, err := s.teamRepo.CountByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count teams: %w", err)
	}
	if count >= s.config.MaxTeamsPerOrganization {
		return nil, fmt.Errorf("organization has reached maximum team limit")
	}

	// Handle description pointer
	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	team := &schema.OrganizationTeam{
		ID:             xid.New(),
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    description,
		Metadata:       req.Metadata,
	}

	if err := s.teamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	return team, nil
}

// GetTeam retrieves a team by ID
func (s *Service) GetTeam(ctx context.Context, id xid.ID) (*schema.OrganizationTeam, error) {
	return s.teamRepo.FindByID(ctx, id)
}

// ListTeams lists teams in an organization
func (s *Service) ListTeams(ctx context.Context, orgID xid.ID, limit, offset int) ([]*schema.OrganizationTeam, error) {
	return s.teamRepo.ListByOrganization(ctx, orgID, limit, offset)
}

// UpdateTeam updates a team
func (s *Service) UpdateTeam(ctx context.Context, id xid.ID, req *UpdateTeamRequest, updaterUserID xid.ID) (*schema.OrganizationTeam, error) {
	team, err := s.teamRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	// Verify updater is admin
	isAdmin, err := s.IsAdmin(ctx, team.OrganizationID, updaterUserID)
	if err != nil || !isAdmin {
		return nil, fmt.Errorf("only admins and owners can update teams")
	}

	if req.Name != nil {
		team.Name = *req.Name
	}
	if req.Description != nil {
		team.Description = *req.Description
	}
	if req.Metadata != nil {
		team.Metadata = req.Metadata
	}

	if err := s.teamRepo.Update(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	return team, nil
}

// DeleteTeam deletes a team
func (s *Service) DeleteTeam(ctx context.Context, id, deleterUserID xid.ID) error {
	team, err := s.teamRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	// Verify deleter is admin
	isAdmin, err := s.IsAdmin(ctx, team.OrganizationID, deleterUserID)
	if err != nil || !isAdmin {
		return fmt.Errorf("only admins and owners can delete teams")
	}

	return s.teamRepo.Delete(ctx, id)
}

// AddTeamMember adds a member to a team
func (s *Service) AddTeamMember(ctx context.Context, teamID, memberID xid.ID, adderUserID xid.ID) error {
	team, err := s.teamRepo.FindByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	// Verify adder is admin
	isAdmin, err := s.IsAdmin(ctx, team.OrganizationID, adderUserID)
	if err != nil || !isAdmin {
		return fmt.Errorf("only admins and owners can add team members")
	}

	teamMember := &schema.OrganizationTeamMember{
		ID:       xid.New(),
		TeamID:   teamID,
		MemberID: memberID,
		JoinedAt: time.Now(),
	}
	return s.teamRepo.AddMember(ctx, teamMember)
}

// RemoveTeamMember removes a member from a team
func (s *Service) RemoveTeamMember(ctx context.Context, teamID, memberID, removerUserID xid.ID) error {
	team, err := s.teamRepo.FindByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	// Verify remover is admin
	isAdmin, err := s.IsAdmin(ctx, team.OrganizationID, removerUserID)
	if err != nil || !isAdmin {
		return fmt.Errorf("only admins and owners can remove team members")
	}

	return s.teamRepo.RemoveMember(ctx, teamID, memberID)
}

// ListTeamMembers lists members of a team
func (s *Service) ListTeamMembers(ctx context.Context, teamID xid.ID, limit, offset int) ([]*schema.OrganizationTeamMember, error) {
	return s.teamRepo.ListMembers(ctx, teamID, limit, offset)
}

// Invitation management

// InviteMember creates an invitation for a user to join an organization
func (s *Service) InviteMember(ctx context.Context, orgID xid.ID, req *InviteMemberRequest, inviterUserID xid.ID) (*schema.OrganizationInvitation, error) {
	// Verify inviter is admin or owner
	isAdmin, err := s.IsAdmin(ctx, orgID, inviterUserID)
	if err != nil || !isAdmin {
		return nil, fmt.Errorf("only admins and owners can invite members")
	}

	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	invitation := &schema.OrganizationInvitation{
		ID:             xid.New(),
		OrganizationID: orgID,
		Email:          req.Email,
		Role:           req.Role,
		Token:          token,
		Status:         InvitationStatusPending,
		InviterID:      inviterUserID,
		ExpiresAt:      time.Now().Add(time.Duration(s.config.InvitationExpiryHours) * time.Hour),
	}

	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	return invitation, nil
}

// GetInvitation retrieves an invitation by token
func (s *Service) GetInvitation(ctx context.Context, token string) (*schema.OrganizationInvitation, error) {
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
func (s *Service) AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*schema.OrganizationMember, error) {
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

	return s.invitationRepo.Update(ctx, invitation)
}

// ListInvitations lists invitations for an organization
func (s *Service) ListInvitations(ctx context.Context, orgID xid.ID, limit, offset int) ([]*schema.OrganizationInvitation, error) {
	return s.invitationRepo.ListByOrganization(ctx, orgID, "", limit, offset)
}

// CleanupExpiredInvitations removes expired invitations
func (s *Service) CleanupExpiredInvitations(ctx context.Context) (int, error) {
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

// Request/Response types

// CreateOrganizationRequest represents the request to create an organization
type CreateOrganizationRequest struct {
	Name     string                 `json:"name"`
	Slug     string                 `json:"slug"`
	Logo     *string                `json:"logo,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateOrganizationRequest represents the request to update an organization
type UpdateOrganizationRequest struct {
	Name     *string                `json:"name,omitempty"`
	Logo     *string                `json:"logo,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CreateTeamRequest represents the request to create a team
type CreateTeamRequest struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTeamRequest represents the request to update a team
type UpdateTeamRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateMemberRequest represents the request to update a member
type UpdateMemberRequest struct {
	Role   *string `json:"role,omitempty"`
	Status *string `json:"status,omitempty"`
}

// InviteMemberRequest represents the request to invite a member
type InviteMemberRequest struct {
	Email    string                 `json:"email"`
	Role     string                 `json:"role"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
