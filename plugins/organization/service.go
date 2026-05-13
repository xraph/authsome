package organization

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/store"
)

// ──────────────────────────────────────────────────
// Organization Management
// ──────────────────────────────────────────────────

// CreateOrganization creates a new organization and adds the creator as owner.
func (p *Plugin) CreateOrganization(ctx context.Context, o *organization.Organization) error {
	if o.CreatedAt.IsZero() {
		now := time.Now()
		o.CreatedAt = now
		o.UpdatedAt = now
	}

	if err := p.store.CreateOrganization(ctx, o); err != nil {
		if p.logger != nil {
			p.logger.Error("organization store: create failed",
				log.String("org_id", o.ID.String()),
				log.String("app_id", o.AppID.String()),
				log.String("env_id", o.EnvID.String()),
				log.String("slug", o.Slug),
				log.String("error", err.Error()),
			)
		}
		return fmt.Errorf("organization: create organization: %w", err)
	}

	member := &organization.Member{
		ID:        id.NewMemberID(),
		OrgID:     o.ID,
		UserID:    o.CreatedBy,
		Role:      organization.RoleOwner,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
	if err := p.store.CreateMember(ctx, member); err != nil {
		if p.logger != nil {
			p.logger.Error("organization store: add owner member failed",
				log.String("org_id", o.ID.String()),
				log.String("user_id", o.CreatedBy.String()),
				log.String("error", err.Error()),
			)
		}
		return fmt.Errorf("organization: add owner member: %w", err)
	}

	p.plugins.EmitAfterOrgCreate(ctx, o)
	p.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionOrgCreate,
		Resource:   hook.ResourceOrganization,
		ResourceID: o.ID.String(),
		ActorID:    o.CreatedBy.String(),
		Tenant:     o.AppID.String(),
	})
	p.relayEvent(ctx, "org.created", o.AppID.String(), map[string]string{
		"org_id":   o.ID.String(),
		"org_slug": o.Slug,
	})

	return nil
}

// GetOrganization returns an organization by ID.
func (p *Plugin) GetOrganization(ctx context.Context, orgID id.OrgID) (*organization.Organization, error) {
	return p.store.GetOrganization(ctx, orgID)
}

// UpdateOrganization updates an existing organization.
func (p *Plugin) UpdateOrganization(ctx context.Context, o *organization.Organization) error {
	o.UpdatedAt = time.Now()
	if err := p.store.UpdateOrganization(ctx, o); err != nil {
		return fmt.Errorf("organization: update organization: %w", err)
	}
	p.plugins.EmitAfterOrgUpdate(ctx, o)
	return nil
}

// DeleteOrganization deletes an organization and all of its dependent records
// (members, teams, invitations) before emitting the AfterOrgDelete hook so
// other plugins (subscription, SCIM, …) can clean up their own org-scoped data.
//
// The cascade is delegated to Store.DeleteOrganizationCascade, which each
// backend implements with its native transaction primitive (PgTx / SqliteTx /
// MongoTx) so a midway failure rolls back atomically. EmitAfterOrgDelete only
// fires once the cascade returns nil, so downstream plugins never see an event
// for an org that wasn't actually deleted.
func (p *Plugin) DeleteOrganization(ctx context.Context, orgID id.OrgID) error {
	if err := p.store.DeleteOrganizationCascade(ctx, orgID); err != nil {
		return fmt.Errorf("organization: delete organization: %w", err)
	}
	p.plugins.EmitAfterOrgDelete(ctx, orgID)
	return nil
}

// ListUserOrganizations returns all organizations a user belongs to.
func (p *Plugin) ListUserOrganizations(ctx context.Context, userID id.UserID) ([]*organization.Organization, error) {
	return p.store.ListUserOrganizations(ctx, userID)
}

// AdminListOrganizations returns all organizations for the given app.
func (p *Plugin) AdminListOrganizations(ctx context.Context, appID id.AppID) ([]*organization.Organization, error) {
	return p.store.ListOrganizations(ctx, appID)
}

// ──────────────────────────────────────────────────
// Member Management
// ──────────────────────────────────────────────────

// AddMember adds a member to an organization.
func (p *Plugin) AddMember(ctx context.Context, m *organization.Member) error {
	if m.CreatedAt.IsZero() {
		now := time.Now()
		m.CreatedAt = now
		m.UpdatedAt = now
	}
	if err := p.store.CreateMember(ctx, m); err != nil {
		return fmt.Errorf("organization: add member: %w", err)
	}
	p.plugins.EmitAfterMemberAdd(ctx, m)
	return nil
}

// RemoveMember removes a member from an organization.
func (p *Plugin) RemoveMember(ctx context.Context, memberID id.MemberID) error {
	if err := p.store.DeleteMember(ctx, memberID); err != nil {
		return fmt.Errorf("organization: remove member: %w", err)
	}
	p.plugins.EmitAfterMemberRemove(ctx, memberID)
	return nil
}

// ListMembers returns all members of an organization.
func (p *Plugin) ListMembers(ctx context.Context, orgID id.OrgID) ([]*organization.Member, error) {
	return p.store.ListMembers(ctx, orgID)
}

// UpdateMemberRole updates a member's role within an organization.
func (p *Plugin) UpdateMemberRole(ctx context.Context, memberID id.MemberID, role organization.MemberRole) (*organization.Member, error) {
	member, err := p.store.GetMember(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("organization: update member role: %w", err)
	}

	member.Role = role
	member.UpdatedAt = time.Now()
	if err := p.store.UpdateMember(ctx, member); err != nil {
		return nil, fmt.Errorf("organization: update member role: %w", err)
	}

	p.plugins.EmitAfterMemberRoleChange(ctx, member)

	// Resolve names for notification template variables (best-effort).
	hookMeta := map[string]string{
		"new_role": string(role),
	}
	if u, err := p.store.GetUser(ctx, member.UserID); err == nil {
		hookMeta["user_name"] = u.Name()
		hookMeta["email"] = u.Email
	}
	if org, err := p.store.GetOrganization(ctx, member.OrgID); err == nil {
		hookMeta["org_name"] = org.Name
	}

	p.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionMemberRoleChange,
		Resource:   hook.ResourceMember,
		ResourceID: member.ID.String(),
		ActorID:    member.UserID.String(),
		Tenant:     member.OrgID.String(),
		Metadata:   hookMeta,
	})
	p.relayEvent(ctx, "org.member.role_changed", member.OrgID.String(), map[string]string{
		"member_id": member.ID.String(),
		"role":      string(role),
	})

	return member, nil
}

// ──────────────────────────────────────────────────
// Invitation Management
// ──────────────────────────────────────────────────

// CreateInvitation creates an organization invitation.
func (p *Plugin) CreateInvitation(ctx context.Context, inv *organization.Invitation) error {
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now()
	}
	if err := p.store.CreateInvitation(ctx, inv); err != nil {
		return fmt.Errorf("organization: create invitation: %w", err)
	}
	return nil
}

// ListInvitations lists invitations for an organization.
func (p *Plugin) ListInvitations(ctx context.Context, orgID id.OrgID) ([]*organization.Invitation, error) {
	return p.store.ListInvitations(ctx, orgID)
}

// AcceptInvitation accepts a pending invitation by token and creates a member.
func (p *Plugin) AcceptInvitation(ctx context.Context, token string) (*organization.Member, error) {
	inv, err := p.store.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("organization: accept invitation: %w", err)
	}

	if inv.Status != organization.InvitationPending {
		return nil, fmt.Errorf("organization: invitation is not pending (status: %s)", inv.Status)
	}

	if !inv.ExpiresAt.IsZero() && time.Now().After(inv.ExpiresAt) {
		inv.Status = organization.InvitationExpired
		_ = p.store.UpdateInvitation(ctx, inv) //nolint:errcheck // best-effort revoke
		return nil, fmt.Errorf("organization: invitation has expired")
	}

	// Mark invitation as accepted
	inv.Status = organization.InvitationAccepted
	if updateErr := p.store.UpdateInvitation(ctx, inv); updateErr != nil {
		return nil, fmt.Errorf("organization: accept invitation: %w", updateErr)
	}

	// Get organization to resolve AppID for the user lookup
	org, err := p.store.GetOrganization(ctx, inv.OrgID)
	if err != nil {
		return nil, fmt.Errorf("organization: accept invitation: org lookup: %w", err)
	}

	// Look up user by email to get their user ID
	u, err := p.store.GetUserByEmail(ctx, org.AppID, inv.Email)
	if err != nil {
		return nil, fmt.Errorf("organization: accept invitation: user not found for email: %w", err)
	}

	// Create member
	now := time.Now()
	member := &organization.Member{
		ID:        id.NewMemberID(),
		OrgID:     inv.OrgID,
		UserID:    u.ID,
		Role:      inv.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := p.store.CreateMember(ctx, member); err != nil {
		return nil, fmt.Errorf("organization: accept invitation: create member: %w", err)
	}

	p.plugins.EmitAfterMemberAdd(ctx, member)
	p.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionInvitationAccept,
		Resource:   hook.ResourceInvitation,
		ResourceID: inv.ID.String(),
		ActorID:    u.ID.String(),
		Tenant:     inv.OrgID.String(),
	})
	p.relayEvent(ctx, "org.invitation.accepted", inv.OrgID.String(), map[string]string{
		"invitation_id": inv.ID.String(),
		"member_id":     member.ID.String(),
		"email":         inv.Email,
	})

	return member, nil
}

// DeclineInvitation declines a pending invitation by token.
func (p *Plugin) DeclineInvitation(ctx context.Context, token string) error {
	inv, err := p.store.GetInvitationByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("organization: decline invitation: %w", err)
	}

	if inv.Status != organization.InvitationPending {
		return fmt.Errorf("organization: invitation is not pending (status: %s)", inv.Status)
	}

	inv.Status = organization.InvitationDeclined
	if err := p.store.UpdateInvitation(ctx, inv); err != nil {
		return fmt.Errorf("organization: decline invitation: %w", err)
	}

	p.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionInvitationDecline,
		Resource:   hook.ResourceInvitation,
		ResourceID: inv.ID.String(),
		Tenant:     inv.OrgID.String(),
	})
	p.relayEvent(ctx, "org.invitation.declined", inv.OrgID.String(), map[string]string{
		"invitation_id": inv.ID.String(),
		"email":         inv.Email,
	})

	return nil
}

// ──────────────────────────────────────────────────
// Team Management
// ──────────────────────────────────────────────────

// CreateTeam creates a new team within an organization.
func (p *Plugin) CreateTeam(ctx context.Context, t *organization.Team) error {
	if t.CreatedAt.IsZero() {
		now := time.Now()
		t.CreatedAt = now
		t.UpdatedAt = now
	}

	if err := p.store.CreateTeam(ctx, t); err != nil {
		return fmt.Errorf("organization: create team: %w", err)
	}

	p.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionTeamCreate,
		Resource:   hook.ResourceTeam,
		ResourceID: t.ID.String(),
		Tenant:     t.OrgID.String(),
	})
	p.relayEvent(ctx, "org.team.created", t.OrgID.String(), map[string]string{
		"team_id":   t.ID.String(),
		"team_slug": t.Slug,
	})

	return nil
}

// GetTeam returns a team by ID.
func (p *Plugin) GetTeam(ctx context.Context, teamID id.TeamID) (*organization.Team, error) {
	return p.store.GetTeam(ctx, teamID)
}

// UpdateTeam updates an existing team.
func (p *Plugin) UpdateTeam(ctx context.Context, t *organization.Team) error {
	t.UpdatedAt = time.Now()
	if err := p.store.UpdateTeam(ctx, t); err != nil {
		return fmt.Errorf("organization: update team: %w", err)
	}

	p.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionTeamUpdate,
		Resource:   hook.ResourceTeam,
		ResourceID: t.ID.String(),
		Tenant:     t.OrgID.String(),
	})

	return nil
}

// DeleteTeam deletes a team.
func (p *Plugin) DeleteTeam(ctx context.Context, teamID id.TeamID) error {
	if err := p.store.DeleteTeam(ctx, teamID); err != nil {
		return fmt.Errorf("organization: delete team: %w", err)
	}

	p.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionTeamDelete,
		Resource:   hook.ResourceTeam,
		ResourceID: teamID.String(),
	})

	return nil
}

// ListTeams returns all teams in an organization.
func (p *Plugin) ListTeams(ctx context.Context, orgID id.OrgID) ([]*organization.Team, error) {
	return p.store.ListTeams(ctx, orgID)
}

// IsOrgSlugAvailable checks whether a slug is available for an app.
func (p *Plugin) IsOrgSlugAvailable(ctx context.Context, appID id.AppID, slug string) (bool, error) {
	_, err := p.store.GetOrganizationBySlug(ctx, appID, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return true, nil
		}
		return false, fmt.Errorf("organization: check slug: %w", err)
	}
	return false, nil
}

// canDeleteOrg returns true when actor is allowed to delete org. The creator
// (org.CreatedBy) always passes as a convenience — note "creator" is not
// "owner"; the permission-check path is the canonical authority for non-
// creator actors. Otherwise the actor must hold the engine-level
// "org.delete" permission on the "org" resource type.
//
// RBAC convention: PermissionChecker.HasPermission's third arg is the
// resource TYPE (e.g. "org"), not an instance ID. rbac/warden_store.go
// forwards this directly as warden.Resource.Type, and middleware/rbac.go
// follows the same shape (e.g. HasPermission(..., "manage", "app")).
// Passing an instance ID here would mean no admin role grant could ever
// match.
func (p *Plugin) canDeleteOrg(ctx context.Context, actor id.UserID, org *organization.Organization) bool {
	if org == nil {
		return false
	}
	// id.UserID is a typed ULID; treat the zero value as "no actor".
	var zero id.UserID
	if actor == zero {
		return false
	}
	if actor == org.CreatedBy {
		return true
	}
	if p.permChecker == nil {
		return false
	}
	ok, err := p.permChecker.HasPermission(ctx, actor, "org.delete", "org")
	return err == nil && ok
}

// chronicleOrNil returns the cached chronicle, falling back to the engine's
// current chronicle (so tests that swap the chronicle via Engine.SetChronicle
// after OnInit are still observed).
func (p *Plugin) chronicleOrNil() bridge.Chronicle {
	if p.engine != nil {
		if ch := p.engine.Chronicle(); ch != nil {
			return ch
		}
	}
	return p.chronicle
}

// ──────────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────────

func (p *Plugin) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if p.relay == nil {
		return
	}
	if err := p.relay.Send(ctx, &bridge.WebhookEvent{
		Type:     eventType,
		TenantID: tenantID,
		Data:     data,
	}); err != nil {
		p.logger.Warn("organization: relay event failed",
			log.String("type", eventType),
			log.String("error", err.Error()),
		)
	}
}

func (p *Plugin) resolveAppID(raw string) (id.AppID, error) {
	if raw != "" {
		return id.ParseAppID(raw)
	}
	return id.ParseAppID(p.defaultAppID)
}

// resolveDefaultEnv looks up the default environment for an app. Used
// when an org-create call comes in without env_id resolved on the
// request context (e.g. a client that doesn't pass X-Environment-ID
// and skipped the EnvironmentMiddleware). Surfaces a non-nil error
// when the app has no default env configured — the caller should
// translate that into a 4xx rather than letting the store's NOT NULL
// constraint blow up with a 500.
func (p *Plugin) resolveDefaultEnv(ctx context.Context, appID id.AppID) (id.EnvironmentID, error) {
	env, err := p.store.GetDefaultEnvironment(ctx, appID)
	if err != nil {
		return id.EnvironmentID{}, fmt.Errorf("get default env: %w", err)
	}
	if env == nil {
		return id.EnvironmentID{}, fmt.Errorf("app %s has no default environment", appID)
	}
	return env.ID, nil
}
