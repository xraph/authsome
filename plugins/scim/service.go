package scim

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugin"
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
	plugins     *plugin.Registry
	// apiKeys and hooks back enforceDeactivation: a deactivated user loses
	// their sessions and keys, and the trail must record it.
	apiKeys apikey.Store
	hooks   *hook.Bus
}

// enforceDeactivation ends every live credential of a user the IdP has just
// deactivated and records the deactivation as a critical trail entry. The
// entry is written through EmitCritical: a deactivation the trail cannot
// hold is reported as failed so the IdP retries it.
func (s *Service) enforceDeactivation(ctx context.Context, cfg *SCIMConfig, u *user.User) error {
	if err := s.authStore.DeleteUserSessions(ctx, u.ID); err != nil {
		return fmt.Errorf("scim: deactivate: sessions: %w", err)
	}
	revoked := 0
	if s.apiKeys != nil {
		keys, err := s.apiKeys.ListAPIKeysByUser(ctx, u.AppID, u.ID)
		if err != nil {
			return fmt.Errorf("scim: deactivate: list keys: %w", err)
		}
		now := time.Now()
		for _, k := range keys {
			if k.Revoked {
				continue
			}
			k.Revoked = true
			k.UpdatedAt = now
			if err := s.apiKeys.UpdateAPIKey(ctx, k); err != nil {
				return fmt.Errorf("scim: deactivate: key %s: %w", k.ID, err)
			}
			revoked++
		}
	}
	if s.hooks == nil {
		return nil
	}
	return s.hooks.EmitCritical(ctx, &hook.Event{
		Action:     hook.ActionAdminBanUser,
		Resource:   hook.ResourceUser,
		ResourceID: u.ID.String(),
		ActorID:    "scim:" + cfg.ID.String(),
		Tenant:     u.AppID.String(),
		Category:   "scim",
		Severity:   hook.SeverityWarning,
		Reason:     "deactivated by identity provider",
		Metadata: map[string]string{
			"source":       "scim",
			"config_id":    cfg.ID.String(),
			"keys_revoked": strconv.Itoa(revoked),
		},
		Private: map[string]string{
			"email": u.Email,
		},
	})
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
func (s *Service) GenerateToken(ctx context.Context, configID id.SCIMConfigID, name string, expiresAt *time.Time) (string, *Token, error) {
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

	token := &Token{
		ID:          id.NewSCIMTokenID(),
		ConfigID:    configID,
		Name:        name,
		TokenHash:   string(hash),
		TokenLookup: authStore.HashToken(plaintext),
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	if err := s.store.CreateToken(ctx, token); err != nil {
		return "", nil, err
	}

	return plaintext, token, nil
}

// ListTokens returns all tokens for a SCIM configuration.
func (s *Service) ListTokens(ctx context.Context, configID id.SCIMConfigID) ([]*Token, error) {
	return s.store.ListTokens(ctx, configID)
}

// RevokeToken deletes a SCIM bearer token.
func (s *Service) RevokeToken(ctx context.Context, tokenID id.SCIMTokenID) error {
	return s.store.DeleteToken(ctx, tokenID)
}

// RotateToken issues a fresh bearer token for the same SCIM config and
// schedules the old token for expiry after a grace window. During the
// grace window both tokens authenticate so in-flight provisioning runs
// don't fail; once the grace expires the old token is rejected by
// ValidateToken (via Token.IsExpired). The new token's plaintext is
// returned to the caller exactly once — operators must persist it
// before discarding the response.
//
// Grace must be ≥ 1 minute and ≤ 30 days; values outside that range
// are clamped to the bound, so a misconfigured rotation can't either
// instantly invalidate the old token (locking out a long-running job)
// nor leave it valid indefinitely.
//
// If the old token's existing ExpiresAt is already sooner than
// now + grace, it is left unchanged — rotation never extends the
// lifetime of a token that was already on a tighter expiry.
func (s *Service) RotateToken(ctx context.Context, oldTokenID id.SCIMTokenID, newName string, grace time.Duration) (string, *Token, error) {
	const minGrace = time.Minute
	const maxGrace = 30 * 24 * time.Hour
	if grace < minGrace {
		grace = minGrace
	}
	if grace > maxGrace {
		grace = maxGrace
	}

	old, err := s.store.GetToken(ctx, oldTokenID)
	if err != nil {
		return "", nil, fmt.Errorf("scim: rotate token: %w", err)
	}

	graceExpiry := time.Now().Add(grace)
	if old.ExpiresAt == nil || old.ExpiresAt.After(graceExpiry) {
		old.ExpiresAt = &graceExpiry
		if updErr := s.store.UpdateToken(ctx, old); updErr != nil {
			return "", nil, fmt.Errorf("scim: rotate token: shorten old expiry: %w", updErr)
		}
	}

	if newName == "" {
		newName = old.Name + " (rotated)"
	}
	plaintext, newToken, err := s.GenerateToken(ctx, old.ConfigID, newName, nil)
	if err != nil {
		return "", nil, fmt.Errorf("scim: rotate token: issue new: %w", err)
	}
	return plaintext, newToken, nil
}

// ValidateToken checks a bearer token against stored hashes.
// Returns the matching token and its associated config, or an error.
func (s *Service) ValidateToken(ctx context.Context, plaintext string) (*Token, *SCIMConfig, error) {
	// We can't reverse a salted bcrypt digest, so resolution is delegated to the
	// store, which scans candidates and bcrypt-compares. For production, consider
	// a token prefix lookup table to narrow that scan.
	t, cfg, err := s.store.FindTokenByPlaintext(ctx, plaintext)
	if err != nil {
		return nil, nil, fmt.Errorf("scim: invalid token")
	}

	if t.IsExpired() {
		return nil, nil, fmt.Errorf("scim: token expired")
	}

	// Record last use. Best-effort: a failed write must not fail the request.
	now := time.Now()
	t.LastUsedAt = &now
	// A token from before the lookup column gets its digest on first use, so
	// the next presentation is an indexed lookup rather than a scan.
	if t.TokenLookup == "" {
		t.TokenLookup = authStore.HashToken(plaintext)
	}
	_ = s.store.UpdateToken(ctx, t) //nolint:errcheck // best-effort usage tracking

	return t, cfg, nil
}

// ──────────────────────────────────────────────────
// Provisioning operations
// ──────────────────────────────────────────────────

// ErrOutOfScope marks a row that exists but lies outside the SCIM
// configuration's app or organization. Handlers answer 404 so a token for
// one tenant cannot probe another.
var ErrOutOfScope = errors.New("scim: resource outside configuration scope")

// userInScope reports whether u belongs to cfg's app and, for an org-scoped
// configuration, to that organization.
func (s *Service) userInScope(ctx context.Context, cfg *SCIMConfig, u *user.User) error {
	if u == nil || u.AppID != cfg.AppID {
		return ErrOutOfScope
	}
	if cfg.OrgID.IsNil() {
		return nil
	}
	if _, err := s.authStore.GetMemberByUserAndOrg(ctx, u.ID, cfg.OrgID); err != nil {
		return ErrOutOfScope
	}
	return nil
}

func teamInScope(cfg *SCIMConfig, t *organization.Team) error {
	if t == nil || cfg.OrgID.IsNil() || t.OrgID != cfg.OrgID {
		return ErrOutOfScope
	}
	return nil
}

// LoadUser resolves a SCIM user id to a user inside cfg's scope. Any miss,
// including a well-formed id in another tenant, is ErrOutOfScope.
func (s *Service) LoadUser(ctx context.Context, cfg *SCIMConfig, raw string) (*user.User, error) {
	if s.authStore == nil {
		return nil, fmt.Errorf("scim: auth store not available")
	}
	userID, err := id.ParseUserID(raw)
	if err != nil {
		return nil, ErrOutOfScope
	}
	u, err := s.authStore.GetUser(ctx, userID)
	if err != nil {
		return nil, ErrOutOfScope
	}
	if err := s.userInScope(ctx, cfg, u); err != nil {
		return nil, err
	}
	return u, nil
}

// LoadTeam resolves a SCIM group id to a team inside cfg's organization.
func (s *Service) LoadTeam(ctx context.Context, cfg *SCIMConfig, raw string) (*organization.Team, error) {
	if s.authStore == nil {
		return nil, fmt.Errorf("scim: auth store not available")
	}
	teamID, err := id.ParseTeamID(raw)
	if err != nil {
		return nil, ErrOutOfScope
	}
	t, err := s.authStore.GetTeam(ctx, teamID)
	if err != nil {
		return nil, ErrOutOfScope
	}
	if err := teamInScope(cfg, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ReplaceUser applies a SCIM PUT to target, which the caller resolved by
// path id. The body's email must be the target's own address: an address
// that resolves to another account is out of scope, so a PUT can never
// re-point one user's record at another.
func (s *Service) ReplaceUser(ctx context.Context, cfg *SCIMConfig, target *user.User, scimUser *UserResource) error {
	if s.authStore == nil {
		return fmt.Errorf("scim: auth store not available")
	}
	if email := scimUser.PrimaryEmail(); email != "" {
		var envID id.EnvironmentID
		if env, _ := s.authStore.GetDefaultEnvironment(ctx, cfg.AppID); env != nil { //nolint:errcheck // best-effort env lookup
			envID = env.ID
		}
		other, err := s.authStore.GetUserByAnyEmail(ctx, cfg.AppID, envID, email)
		if err == nil && other != nil && other.ID != target.ID {
			return ErrOutOfScope
		}
	}
	wasBanned := target.Banned
	target.FirstName = scimUser.Name.GivenName
	target.LastName = scimUser.Name.FamilyName
	target.Banned = !scimUser.Active
	if email := scimUser.PrimaryEmail(); email != "" {
		if err := s.ChangeUserName(ctx, cfg, target, email); err != nil {
			return err
		}
	}
	target.UpdatedAt = time.Now()
	if err := s.authStore.UpdateUser(ctx, target); err != nil {
		return err
	}
	// PUT can flip Banned, and plugins watching AfterUserUpdate (agent grant
	// revocation among them) must see every path that does.
	if s.plugins != nil {
		s.plugins.EmitAfterUserUpdate(ctx, target)
	}
	if target.Banned && !wasBanned {
		return s.enforceDeactivation(ctx, cfg, target)
	}
	return nil
}

// ProvisionUser creates or updates a user from SCIM data.
func (s *Service) ProvisionUser(ctx context.Context, cfg *SCIMConfig, scimUser *UserResource) (*user.User, string, error) {
	if s.authStore == nil {
		return nil, ActionCreateUser, fmt.Errorf("scim: auth store not available")
	}

	// Resolve the app's default environment so the user and its email row are
	// scoped consistently with the other sign-up paths.
	var envID id.EnvironmentID
	if env, _ := s.authStore.GetDefaultEnvironment(ctx, cfg.AppID); env != nil { //nolint:errcheck // best-effort env lookup
		envID = env.ID
	}

	// Try to find an existing user by any of their emails.
	existing, err := s.authStore.GetUserByAnyEmail(ctx, cfg.AppID, envID, scimUser.PrimaryEmail())
	if err == nil && existing != nil {
		// An org-scoped configuration may update only its own members; an
		// app user outside the org is invisible to it.
		if scopeErr := s.userInScope(ctx, cfg, existing); scopeErr != nil {
			return nil, ActionUpdateUser, scopeErr
		}
		// Update existing user.
		wasBanned := existing.Banned
		existing.FirstName = scimUser.Name.GivenName
		existing.LastName = scimUser.Name.FamilyName
		existing.Banned = !scimUser.Active
		if err := s.authStore.UpdateUser(ctx, existing); err != nil {
			return nil, ActionUpdateUser, err
		}
		if existing.Banned && !wasBanned {
			if err := s.enforceDeactivation(ctx, cfg, existing); err != nil {
				return nil, ActionUpdateUser, err
			}
		}
		// ProvisionUser backs both PUT /Users/:id (handleReplaceUser) and a
		// POST that resolves to an existing user, and either can flip Banned.
		// handleCreateUser forces Active = true before calling in, so the
		// POST case never actually bans here, but PUT can and must still be
		// seen by plugins watching AfterUserUpdate.
		if s.plugins != nil {
			s.plugins.EmitAfterUserUpdate(ctx, existing)
		}
		return existing, ActionUpdateUser, nil
	}

	if !cfg.AutoCreate {
		return nil, ActionCreateUser, fmt.Errorf("scim: auto-create disabled for config %s", cfg.ID)
	}

	// Create new user, scoped to the SCIM config's app + default env, seeding
	// its primary email row so it participates in cross-account matching.
	newUser := &user.User{
		ID:            id.NewUserID(),
		AppID:         cfg.AppID,
		EnvID:         envID,
		Email:         scimUser.PrimaryEmail(),
		FirstName:     scimUser.Name.GivenName,
		LastName:      scimUser.Name.FamilyName,
		EmailVerified: true, // SCIM-provisioned users are pre-verified.
		Banned:        !scimUser.Active,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.authStore.CreateUserWithPrimaryEmail(ctx, newUser, user.NewPrimaryEmail(newUser, "scim")); err != nil {
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
		return ErrOutOfScope
	}
	if err := s.userInScope(ctx, cfg, u); err != nil {
		return err
	}

	u.Banned = true
	u.UpdatedAt = time.Now()
	if err := s.authStore.UpdateUser(ctx, u); err != nil {
		return err
	}

	// Same reasoning as AdminBanUser in service.go: a SCIM deactivation is a
	// ban by another name, and plugins watching AfterUserUpdate (agentauth's
	// grant revocation among them) need to see it fire.
	if s.plugins != nil {
		s.plugins.EmitAfterUserUpdate(ctx, u)
	}
	return s.enforceDeactivation(ctx, cfg, u)
}

// ProvisionGroup creates or updates a team from SCIM Group data.
func (s *Service) ProvisionGroup(ctx context.Context, cfg *SCIMConfig, scimGroup *GroupResource) (*organization.Team, string, error) {
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
	l := &ProvisionLog{
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
func (s *Service) ListLogs(ctx context.Context, configID id.SCIMConfigID, limit int) ([]*ProvisionLog, error) {
	return s.store.ListLogs(ctx, configID, limit)
}

// ListAllLogs returns provision logs across all configs for an app.
func (s *Service) ListAllLogs(ctx context.Context, appID string, limit int) ([]*ProvisionLog, error) {
	return s.store.ListAllLogs(ctx, appID, limit)
}

// CountLogsByStatus returns log counts grouped by status for a config.
func (s *Service) CountLogsByStatus(ctx context.Context, configID id.SCIMConfigID) (success, failed, skipped int, err error) {
	return s.store.CountLogsByStatus(ctx, configID)
}

// CountAllLogsByStatus returns log counts across all configs for an app.
func (s *Service) CountAllLogsByStatus(ctx context.Context, appID string) (success, failed, skipped int, err error) {
	return s.store.CountAllLogsByStatus(ctx, appID)
}

// ChangeUserName moves a provisioned user to a new primary address through
// the email-record API rather than by overwriting the user's email field:
// the address becomes a verified email record owned by the user and is then
// made primary, so uniqueness inside the environment holds and every place
// that reads email records sees the change. An address another account
// owns is refused as out of scope; the user's own current address is a
// no-op.
func (s *Service) ChangeUserName(ctx context.Context, cfg *SCIMConfig, u *user.User, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || strings.EqualFold(email, u.Email) {
		return nil
	}
	if s.authStore == nil {
		return fmt.Errorf("scim: auth store not available")
	}
	var envID id.EnvironmentID
	if env, _ := s.authStore.GetDefaultEnvironment(ctx, cfg.AppID); env != nil { //nolint:errcheck // best-effort env lookup
		envID = env.ID
	}
	if !u.EnvID.IsNil() {
		envID = u.EnvID
	}
	owner, err := s.authStore.GetUserByAnyEmail(ctx, cfg.AppID, envID, email)
	switch {
	case err == nil && owner != nil && owner.ID != u.ID:
		return ErrOutOfScope
	case err == nil && owner != nil:
		// Already one of this user's addresses; make sure it counts as
		// verified before it becomes primary.
		if verr := s.authStore.MarkUserEmailVerified(ctx, u.ID, email); verr != nil && !errors.Is(verr, authStore.ErrNotFound) {
			return verr
		}
	default:
		now := time.Now()
		rec := &user.UserEmail{
			ID: id.NewUserEmailID(), UserID: u.ID, AppID: cfg.AppID, EnvID: envID, Email: email,
			Verified: true, Source: "scim", CreatedAt: now, UpdatedAt: now,
		}
		if aerr := s.authStore.AddUserEmail(ctx, rec); aerr != nil && !errors.Is(aerr, account.ErrEmailTaken) {
			return aerr
		}
	}
	if perr := s.authStore.SetPrimaryEmail(ctx, u.ID, email); perr != nil {
		return perr
	}
	u.Email = email
	return nil
}
