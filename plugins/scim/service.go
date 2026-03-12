package scim

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/settings"
	authStore "github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// roleEnsurer assigns a default Warden role to a newly created user.
type roleEnsurer interface {
	EnsureDefaultRole(ctx context.Context, appID id.AppID, userID id.UserID)
}

// Service encapsulates the SCIM business logic.
type Service struct {
	store       Store
	authStore   authStore.Store
	settings    *settings.Manager
	logger      log.Logger
	roleEnsurer roleEnsurer
}

// ──────────────────────────────────────────────────
// Config management
// ──────────────────────────────────────────────────

// CreateConfig creates a new SCIM configuration.
func (s *Service) CreateConfig(ctx context.Context, c *SCIMConfig) error {
	now := time.Now()
	c.ID = id.NewSCIMConfigID()
	c.CreatedAt = now
	c.UpdatedAt = now
	return s.store.CreateConfig(ctx, c)
}

// GetConfig returns a SCIM configuration by ID.
func (s *Service) GetConfig(ctx context.Context, configID id.SCIMConfigID) (*SCIMConfig, error) {
	return s.store.GetConfig(ctx, configID)
}

// UpdateConfig updates a SCIM configuration.
func (s *Service) UpdateConfig(ctx context.Context, c *SCIMConfig) error {
	c.UpdatedAt = time.Now()
	return s.store.UpdateConfig(ctx, c)
}

// DeleteConfig deletes a SCIM configuration.
func (s *Service) DeleteConfig(ctx context.Context, configID id.SCIMConfigID) error {
	return s.store.DeleteConfig(ctx, configID)
}

// ListConfigs returns all SCIM configurations for an app.
func (s *Service) ListConfigs(ctx context.Context, appID string) ([]*SCIMConfig, error) {
	return s.store.ListConfigs(ctx, appID)
}

// ListConfigsByOrg returns all SCIM configurations for an organization.
func (s *Service) ListConfigsByOrg(ctx context.Context, orgID id.OrgID) ([]*SCIMConfig, error) {
	return s.store.ListConfigsByOrg(ctx, orgID)
}

// ──────────────────────────────────────────────────
// Token management
// ──────────────────────────────────────────────────

// GenerateToken creates a new bearer token for a SCIM config.
// Returns the plaintext token (shown once) and the stored token record.
func (s *Service) GenerateToken(ctx context.Context, configID id.SCIMConfigID, name string, expiresAt *time.Time) (string, *SCIMToken, error) {
	// Generate random token.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("scim: generate token: %w", err)
	}
	plaintext := "scim_" + hex.EncodeToString(tokenBytes)

	// Hash for storage.
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("scim: hash token: %w", err)
	}

	token := &SCIMToken{
		ID:        id.NewSCIMTokenID(),
		ConfigID:  configID,
		Name:      name,
		TokenHash: string(hash),
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateToken(ctx, token); err != nil {
		return "", nil, err
	}

	return plaintext, token, nil
}

// ListTokens returns all tokens for a SCIM configuration.
func (s *Service) ListTokens(ctx context.Context, configID id.SCIMConfigID) ([]*SCIMToken, error) {
	return s.store.ListTokens(ctx, configID)
}

// RevokeToken deletes a SCIM bearer token.
func (s *Service) RevokeToken(ctx context.Context, tokenID id.SCIMTokenID) error {
	return s.store.DeleteToken(ctx, tokenID)
}

// ValidateToken checks a bearer token against stored hashes.
// Returns the matching token and its associated config, or an error.
func (s *Service) ValidateToken(ctx context.Context, plaintext string) (*SCIMToken, *SCIMConfig, error) {
	// We need to iterate and bcrypt-compare since we can't reverse the hash.
	// For the in-memory store, FindTokenByHash does a linear scan with bcrypt.Compare.
	// For production, consider a token prefix lookup table.
	configs, err := s.store.ListConfigs(ctx, "")
	if err != nil {
		return nil, nil, err
	}

	for _, cfg := range configs {
		tokens, err := s.store.ListTokens(ctx, cfg.ID)
		if err != nil {
			continue
		}
		for _, t := range tokens {
			if err := bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(plaintext)); err == nil {
				if t.IsExpired() {
					return nil, nil, fmt.Errorf("scim: token expired")
				}
				// Update last used.
				now := time.Now()
				t.LastUsedAt = &now
				return t, cfg, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("scim: invalid token")
}

// ──────────────────────────────────────────────────
// Provisioning operations
// ──────────────────────────────────────────────────

// ProvisionUser creates or updates a user from SCIM data.
func (s *Service) ProvisionUser(ctx context.Context, cfg *SCIMConfig, scimUser *SCIMUserResource) (*user.User, string, error) {
	if s.authStore == nil {
		return nil, ActionCreateUser, fmt.Errorf("scim: auth store not available")
	}

	// Try to find existing user by email.
	existing, err := s.authStore.GetUserByEmail(ctx, cfg.AppID, scimUser.PrimaryEmail())
	if err == nil && existing != nil {
		// Update existing user.
		existing.FirstName = scimUser.Name.GivenName
		existing.LastName = scimUser.Name.FamilyName
		existing.Banned = !scimUser.Active
		if err := s.authStore.UpdateUser(ctx, existing); err != nil {
			return nil, ActionUpdateUser, err
		}
		return existing, ActionUpdateUser, nil
	}

	if !cfg.AutoCreate {
		return nil, ActionCreateUser, fmt.Errorf("scim: auto-create disabled for config %s", cfg.ID)
	}

	// Create new user.
	newUser := &user.User{
		ID:            id.NewUserID(),
		Email:         scimUser.PrimaryEmail(),
		FirstName:     scimUser.Name.GivenName,
		LastName:      scimUser.Name.FamilyName,
		EmailVerified: true, // SCIM-provisioned users are pre-verified.
		Banned:        !scimUser.Active,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.authStore.CreateUser(ctx, newUser); err != nil {
		return nil, ActionCreateUser, err
	}
	if s.roleEnsurer != nil {
		s.roleEnsurer.EnsureDefaultRole(ctx, cfg.AppID, newUser.ID)
	}

	// If org-scoped, add user as member.
	if !cfg.OrgID.IsNil() {
		role := organization.MemberRole(cfg.DefaultRole)
		if role == "" {
			role = organization.RoleMember
		}
		member := &organization.Member{
			ID:        id.NewMemberID(),
			OrgID:     cfg.OrgID,
			UserID:    newUser.ID,
			Role:      role,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.authStore.CreateMember(ctx, member); err != nil {
			if s.logger != nil {
				s.logger.Warn("scim: failed to add member to org",
					log.String("user_id", newUser.ID.String()),
					log.String("org_id", cfg.OrgID.String()),
					log.Error(err),
				)
			}
		}
	}

	return newUser, ActionCreateUser, nil
}

// DeactivateUser suspends a user by setting Active=false.
func (s *Service) DeactivateUser(ctx context.Context, cfg *SCIMConfig, userID id.UserID) error {
	if s.authStore == nil {
		return fmt.Errorf("scim: auth store not available")
	}

	if !cfg.AutoSuspend {
		return fmt.Errorf("scim: auto-suspend disabled")
	}

	u, err := s.authStore.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	u.Banned = true
	u.UpdatedAt = time.Now()
	return s.authStore.UpdateUser(ctx, u)
}

// ProvisionGroup creates or updates a team from SCIM Group data.
func (s *Service) ProvisionGroup(ctx context.Context, cfg *SCIMConfig, scimGroup *SCIMGroupResource) (*organization.Team, string, error) {
	if s.authStore == nil {
		return nil, ActionCreateGroup, fmt.Errorf("scim: auth store not available")
	}

	if !cfg.GroupSync {
		return nil, ActionCreateGroup, fmt.Errorf("scim: group sync disabled")
	}

	if cfg.OrgID.IsNil() {
		return nil, ActionCreateGroup, fmt.Errorf("scim: group sync requires org-scoped config")
	}

	// Try to find existing team by name in the org.
	teams, err := s.authStore.ListTeams(ctx, cfg.OrgID)
	if err != nil {
		return nil, ActionCreateGroup, err
	}

	for _, t := range teams {
		if t.Name == scimGroup.DisplayName {
			// Update existing team.
			t.Name = scimGroup.DisplayName
			t.UpdatedAt = time.Now()
			if err := s.authStore.UpdateTeam(ctx, t); err != nil {
				return nil, ActionUpdateGroup, err
			}
			return t, ActionUpdateGroup, nil
		}
	}

	// Create new team.
	team := &organization.Team{
		ID:        id.NewTeamID(),
		OrgID:     cfg.OrgID,
		Name:      scimGroup.DisplayName,
		Slug:      scimGroup.DisplayName, // Will be slugified by store.
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.authStore.CreateTeam(ctx, team); err != nil {
		return nil, ActionCreateGroup, err
	}

	return team, ActionCreateGroup, nil
}

// ──────────────────────────────────────────────────
// Provision logging
// ──────────────────────────────────────────────────

// RecordLog records a SCIM provisioning action.
func (s *Service) RecordLog(ctx context.Context, configID id.SCIMConfigID, action, resourceType, externalID, internalID, status, detail string) {
	l := &SCIMProvisionLog{
		ID:           id.NewSCIMLogID(),
		ConfigID:     configID,
		Action:       action,
		ResourceType: resourceType,
		ExternalID:   externalID,
		InternalID:   internalID,
		Status:       status,
		Detail:       detail,
		CreatedAt:    time.Now(),
	}
	if err := s.store.CreateLog(ctx, l); err != nil && s.logger != nil {
		s.logger.Warn("scim: failed to create provision log", log.Error(err))
	}
}

// ListLogs returns provision logs for a config.
func (s *Service) ListLogs(ctx context.Context, configID id.SCIMConfigID, limit int) ([]*SCIMProvisionLog, error) {
	return s.store.ListLogs(ctx, configID, limit)
}

// ListAllLogs returns provision logs across all configs for an app.
func (s *Service) ListAllLogs(ctx context.Context, appID string, limit int) ([]*SCIMProvisionLog, error) {
	return s.store.ListAllLogs(ctx, appID, limit)
}

// CountLogsByStatus returns log counts grouped by status for a config.
func (s *Service) CountLogsByStatus(ctx context.Context, configID id.SCIMConfigID) (success, errors, skipped int, err error) {
	return s.store.CountLogsByStatus(ctx, configID)
}

// CountAllLogsByStatus returns log counts across all configs for an app.
func (s *Service) CountAllLogsByStatus(ctx context.Context, appID string) (success, errors, skipped int, err error) {
	return s.store.CountAllLogsByStatus(ctx, appID)
}
