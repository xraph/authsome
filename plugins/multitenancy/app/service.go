package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/xid"
)

// Service handles app-related business logic
type Service struct {
	appRepo        AppRepository
	memberRepo     MemberRepository
	teamRepo       TeamRepository
	invitationRepo InvitationRepository
	config         Config
}

// Config holds the app service configuration
type Config struct {
	PlatformAppID         string `json:"platformAppId"`
	DefaultAppName        string `json:"defaultAppName"`
	EnableAppCreation     bool   `json:"enableAppCreation"`
	MaxMembersPerApp      int    `json:"maxMembersPerApp"`
	MaxTeamsPerApp        int    `json:"maxTeamsPerApp"`
	RequireInvitation     bool   `json:"requireInvitation"`
	InvitationExpiryHours int    `json:"invitationExpiryHours"`
}

// Repository interfaces

// AppRepository defines the interface for app data access
type AppRepository interface {
	Create(ctx context.Context, app *App) error
	FindByID(ctx context.Context, id xid.ID) (*App, error)
	FindBySlug(ctx context.Context, slug string) (*App, error)
	List(ctx context.Context, limit, offset int) ([]*App, error)
	Update(ctx context.Context, app *App) error
	Delete(ctx context.Context, id xid.ID) error
	Count(ctx context.Context) (int, error)
}

// MemberRepository defines the interface for member data access
type MemberRepository interface {
	Create(ctx context.Context, member *Member) error
	FindByID(ctx context.Context, id xid.ID) (*Member, error)
	FindByUserAndApp(ctx context.Context, userID, appID xid.ID) (*Member, error)
	ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*Member, error)
	ListByUser(ctx context.Context, userID xid.ID) ([]*Member, error)
	Update(ctx context.Context, member *Member) error
	Delete(ctx context.Context, id xid.ID) error
	DeleteByUserID(ctx context.Context, userID xid.ID) error
	CountByApp(ctx context.Context, appID xid.ID) (int, error)
}

// TeamRepository defines the interface for team data access
type TeamRepository interface {
	Create(ctx context.Context, team *Team) error
	FindByID(ctx context.Context, id xid.ID) (*Team, error)
	ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*Team, error)
	Update(ctx context.Context, team *Team) error
	Delete(ctx context.Context, id xid.ID) error
	CountByApp(ctx context.Context, appID xid.ID) (int, error)
	AddMember(ctx context.Context, teamID, memberID xid.ID, role string) error
	RemoveMember(ctx context.Context, teamID, memberID xid.ID) error
	ListMembers(ctx context.Context, teamID xid.ID) ([]*Member, error)
}

// InvitationRepository defines the interface for invitation data access
type InvitationRepository interface {
	Create(ctx context.Context, invitation *Invitation) error
	FindByID(ctx context.Context, id xid.ID) (*Invitation, error)
	FindByToken(ctx context.Context, token string) (*Invitation, error)
	ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*Invitation, error)
	Update(ctx context.Context, invitation *Invitation) error
	Delete(ctx context.Context, id xid.ID) error
	DeleteExpired(ctx context.Context) error
}

// NewService creates a new app service
func NewService(
	config Config,
	appRepo AppRepository,
	memberRepo MemberRepository,
	teamRepo TeamRepository,
	inviteRepo InvitationRepository,
) *Service {
	return &Service{
		config:         config,
		appRepo:        appRepo,
		memberRepo:     memberRepo,
		teamRepo:       teamRepo,
		invitationRepo: inviteRepo,
	}
}

// App management

// CreateApp creates a new app
func (s *Service) CreateApp(ctx context.Context, req *CreateAppRequest, creatorUserID xid.ID) (*App, error) {
	// Check if there are any apps yet
	apps, err := s.appRepo.List(ctx, 1, 0)
	isFirstApp := (err != nil || len(apps) == 0)

	// Allow creation if this is the first app (platform app for system owner)
	// Otherwise check if app creation is enabled
	if !isFirstApp && !s.config.EnableAppCreation {
		return nil, fmt.Errorf("app creation is disabled")
	}

	// Check if slug is already taken
	existing, err := s.appRepo.FindBySlug(ctx, req.Slug)
	if err == nil && existing != nil {
		return nil, ErrSlugAlreadyExists
	}

	// Create app
	app := &App{
		ID:        xid.New(),
		Name:      req.Name,
		Slug:      req.Slug,
		Logo:      req.Logo,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	// Add creator as owner
	_, err = s.AddMember(ctx, app.ID, creatorUserID, RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("failed to add creator as owner: %w", err)
	}

	return app, nil
}

// GetApp retrieves an app by ID
func (s *Service) GetApp(ctx context.Context, id xid.ID) (*App, error) {
	return s.appRepo.FindByID(ctx, id)
}

// GetAppBySlug retrieves an app by slug
func (s *Service) GetAppBySlug(ctx context.Context, slug string) (*App, error) {
	return s.appRepo.FindBySlug(ctx, slug)
}

// ListApps lists apps with pagination
func (s *Service) ListApps(ctx context.Context, limit, offset int) ([]*App, error) {
	return s.appRepo.List(ctx, limit, offset)
}

// UpdateApp updates an app
func (s *Service) UpdateApp(ctx context.Context, id xid.ID, req *UpdateAppRequest) (*App, error) {
	app, err := s.appRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}

	// Update fields
	if req.Name != nil {
		app.Name = *req.Name
	}
	if req.Logo != nil {
		app.Logo = req.Logo
	}
	if req.Metadata != nil {
		app.Metadata = req.Metadata
	}
	app.UpdatedAt = time.Now()

	if err := s.appRepo.Update(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to update app: %w", err)
	}

	return app, nil
}

// DeleteApp deletes an app
func (s *Service) DeleteApp(ctx context.Context, id xid.ID) error {
	return s.appRepo.Delete(ctx, id)
}

// GetDefaultApp returns the default app for standalone mode
func (s *Service) GetDefaultApp(ctx context.Context) (*App, error) {
	if s.config.PlatformAppID != "" {
		// Parse the platform app ID from config
		platformID, err := xid.FromString(s.config.PlatformAppID)
		if err != nil {
			return nil, fmt.Errorf("invalid platform app ID in config: %w", err)
		}
		return s.appRepo.FindByID(ctx, platformID)
	}

	// Find or create default app
	apps, err := s.appRepo.List(ctx, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	if len(apps) > 0 {
		return apps[0], nil
	}

	// Create default app
	defaultApp := &App{
		ID:        xid.New(),
		Name:      s.config.DefaultAppName,
		Slug:      "default",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.appRepo.Create(ctx, defaultApp); err != nil {
		return nil, fmt.Errorf("failed to create default app: %w", err)
	}

	return defaultApp, nil
}

// Member management

// AddMember adds a user as a member of an app
func (s *Service) AddMember(ctx context.Context, appID, userID xid.ID, role string) (*Member, error) {
	// Check if user is already a member
	existing, err := s.memberRepo.FindByUserAndApp(ctx, userID, appID)
	if err == nil && existing != nil {
		return nil, ErrMemberAlreadyExists
	}

	// Check member limit
	count, err := s.memberRepo.CountByApp(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}
	if count >= s.config.MaxMembersPerApp {
		return nil, fmt.Errorf("app has reached maximum member limit")
	}

	member := &Member{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    userID,
		Role:      role,
		Status:    StatusActive,
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return member, nil
}

// GetMember retrieves a member by ID
func (s *Service) GetMember(ctx context.Context, id xid.ID) (*Member, error) {
	return s.memberRepo.FindByID(ctx, id)
}

// IsMember checks if a user is a member of an app
func (s *Service) IsMember(ctx context.Context, appID, userID xid.ID) (bool, error) {
	member, err := s.memberRepo.FindByUserAndApp(ctx, userID, appID)
	if err != nil {
		return false, nil // User is not a member (or error occurred)
	}
	return member != nil && member.Status == StatusActive, nil
}

// ListMembers lists members of an app
func (s *Service) ListMembers(ctx context.Context, appID xid.ID, limit, offset int) ([]*Member, error) {
	return s.memberRepo.ListByApp(ctx, appID, limit, offset)
}

// UpdateMember updates a member
func (s *Service) UpdateMember(ctx context.Context, id xid.ID, req *UpdateMemberRequest) (*Member, error) {
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

// RemoveMember removes a member from an app
func (s *Service) RemoveMember(ctx context.Context, id xid.ID) error {
	return s.memberRepo.Delete(ctx, id)
}

// IsUserMember checks if a user is a member of an app
func (s *Service) IsUserMember(ctx context.Context, appID, userID xid.ID) (bool, error) {
	member, err := s.memberRepo.FindByUserAndApp(ctx, userID, appID)
	if err != nil {
		return false, nil
	}
	return member != nil && member.Status == StatusActive, nil
}

// GetUserMemberships returns all apps a user is a member of
func (s *Service) GetUserMemberships(ctx context.Context, userID xid.ID) ([]*Member, error) {
	return s.memberRepo.ListByUser(ctx, userID)
}

// RemoveUserFromAllApps removes a user from all apps
func (s *Service) RemoveUserFromAllApps(ctx context.Context, userID xid.ID) error {
	return s.memberRepo.DeleteByUserID(ctx, userID)
}

// Team management

// CreateTeam creates a new team in an app
func (s *Service) CreateTeam(ctx context.Context, appID xid.ID, req *CreateTeamRequest) (*Team, error) {
	// Check team limit
	count, err := s.teamRepo.CountByApp(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to count teams: %w", err)
	}
	if count >= s.config.MaxTeamsPerApp {
		return nil, fmt.Errorf("app has reached maximum team limit")
	}

	team := &Team{
		ID:          xid.New(),
		AppID:       appID,
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.teamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	return team, nil
}

// GetTeam retrieves a team by ID
func (s *Service) GetTeam(ctx context.Context, id xid.ID) (*Team, error) {
	return s.teamRepo.FindByID(ctx, id)
}

// ListTeams lists teams in an app
func (s *Service) ListTeams(ctx context.Context, appID xid.ID, limit, offset int) ([]*Team, error) {
	return s.teamRepo.ListByApp(ctx, appID, limit, offset)
}

// UpdateTeam updates a team
func (s *Service) UpdateTeam(ctx context.Context, id xid.ID, req *UpdateTeamRequest) (*Team, error) {
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
func (s *Service) DeleteTeam(ctx context.Context, id xid.ID) error {
	return s.teamRepo.Delete(ctx, id)
}

// AddTeamMember adds a member to a team
func (s *Service) AddTeamMember(ctx context.Context, teamID, memberID xid.ID, role string) error {
	return s.teamRepo.AddMember(ctx, teamID, memberID, role)
}

// RemoveTeamMember removes a member from a team
func (s *Service) RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error {
	return s.teamRepo.RemoveMember(ctx, teamID, memberID)
}

// ListTeamMembers lists members of a team
func (s *Service) ListTeamMembers(ctx context.Context, teamID xid.ID) ([]*Member, error) {
	return s.teamRepo.ListMembers(ctx, teamID)
}

// Invitation management

// InviteMember creates an invitation for a user to join an app
func (s *Service) InviteMember(ctx context.Context, appID xid.ID, req *InviteMemberRequest, inviterUserID xid.ID) (*Invitation, error) {
	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	invitation := &Invitation{
		ID:        xid.New(),
		AppID:     appID,
		Email:     req.Email,
		Role:      req.Role,
		Token:     token,
		Status:    InvitationStatusPending,
		InvitedBy: inviterUserID,
		Metadata:  req.Metadata,
		ExpiresAt: time.Now().Add(time.Duration(s.config.InvitationExpiryHours) * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
		return nil, ErrInvitationExpired
	}

	return invitation, nil
}

// AcceptInvitation accepts an invitation and adds the user to the app
func (s *Service) AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*Member, error) {
	invitation, err := s.GetInvitation(ctx, token)
	if err != nil {
		return nil, err
	}

	if invitation.Status != InvitationStatusPending {
		return nil, fmt.Errorf("invitation is not pending")
	}

	// Add user as member
	member, err := s.AddMember(ctx, invitation.AppID, userID, invitation.Role)
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

// ListInvitations lists invitations for an app
func (s *Service) ListInvitations(ctx context.Context, appID xid.ID, limit, offset int) ([]*Invitation, error) {
	return s.invitationRepo.ListByApp(ctx, appID, limit, offset)
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

// Request/Response types

// CreateAppRequest represents the request to create an app
type CreateAppRequest struct {
	Name     string                 `json:"name"`
	Slug     string                 `json:"slug"`
	Logo     *string                `json:"logo,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateAppRequest represents the request to update an app
type UpdateAppRequest struct {
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
