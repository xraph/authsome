package organization

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/page"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// RegisterRoutes registers organization management routes on a forge.Router.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if err := p.registerOrgRoutes(router); err != nil {
		return err
	}
	return p.registerAdminOrgRoutes(router)
}

// ──────────────────────────────────────────────────
// Organization route registration
// ──────────────────────────────────────────────────

func (p *Plugin) registerOrgRoutes(router forge.Router) error {
	g := router.Group(p.config.PathPrefix, forge.WithGroupTags("organizations"))

	// Organization CRUD
	if err := g.POST("/orgs", p.handleCreateOrg,
		forge.WithSummary("Create organization"),
		forge.WithDescription("Creates a new organization. The authenticated user becomes the owner."),
		forge.WithOperationID("createOrganization"),
		forge.WithRequestSchema(CreateOrgRequest{}),
		forge.WithCreatedResponse(organization.Organization{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/orgs", p.handleListOrgs,
		forge.WithSummary("List organizations"),
		forge.WithDescription("Returns all organizations the authenticated user belongs to."),
		forge.WithOperationID("listOrganizations"),
		forge.WithResponseSchema(http.StatusOK, "Organization list", OrgListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/orgs/:orgId", p.handleGetOrg,
		forge.WithSummary("Get organization"),
		forge.WithDescription("Returns details of a specific organization."),
		forge.WithOperationID("getOrganization"),
		forge.WithResponseSchema(http.StatusOK, "Organization details", organization.Organization{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/orgs/:orgId", p.handleUpdateOrg,
		forge.WithSummary("Update organization"),
		forge.WithDescription("Updates an organization's name or logo."),
		forge.WithOperationID("updateOrganization"),
		forge.WithRequestSchema(UpdateOrgRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated organization", organization.Organization{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/orgs/:orgId", p.handleDeleteOrg,
		forge.WithSummary("Delete organization"),
		forge.WithDescription("Deletes an organization and all its members."),
		forge.WithOperationID("deleteOrganization"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Members
	if err := g.GET("/orgs/:orgId/members", p.handleListMembers,
		forge.WithSummary("List members"),
		forge.WithDescription("Returns all members of an organization."),
		forge.WithOperationID("listMembers"),
		forge.WithResponseSchema(http.StatusOK, "Member list", MemberListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/orgs/:orgId/members", p.handleAddMember,
		forge.WithSummary("Add member"),
		forge.WithDescription("Adds a user as a member of an organization."),
		forge.WithOperationID("addMember"),
		forge.WithRequestSchema(AddMemberRequest{}),
		forge.WithCreatedResponse(organization.Member{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/orgs/:orgId/members/:memberId", p.handleRemoveMember,
		forge.WithSummary("Remove member"),
		forge.WithDescription("Removes a member from an organization."),
		forge.WithOperationID("removeMember"),
		forge.WithResponseSchema(http.StatusOK, "Removed", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/orgs/:orgId/members/:memberId", p.handleUpdateMember,
		forge.WithSummary("Update member role"),
		forge.WithDescription("Updates a member's role within an organization."),
		forge.WithOperationID("updateMember"),
		forge.WithRequestSchema(UpdateMemberRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated member", organization.Member{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Invitations
	if err := g.POST("/orgs/invitations/accept", p.handleAcceptInvitation,
		forge.WithSummary("Accept invitation"),
		forge.WithDescription("Accepts a pending organization invitation using the invitation token. The signed-in user must own, and have verified, the email the invitation was sent to. Answers 401 when signed out and 404 for an unknown token or for anyone but the invitee, 409 for a token already used or withdrawn and 410 for an expired one."),
		forge.WithOperationID("acceptInvitation"),
		forge.WithRequestSchema(AcceptInvitationRequest{}),
		forge.WithCreatedResponse(organization.Member{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Unauthenticated on purpose: whoever is deciding whether to accept has
	// usually not signed in, and often has no account at all.
	if err := g.GET("/orgs/invitations/:inviteToken", p.handleGetInvitation,
		forge.WithSummary("Get invitation"),
		forge.WithDescription("Reads a pending invitation by its token so the person holding it can see who invited them, to which organization, and as what, before accepting. No session required."),
		forge.WithOperationID("getInvitation"),
		forge.WithResponseSchema(http.StatusOK, "Invitation preview", InvitationPreview{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/orgs/invitations/decline", p.handleDeclineInvitation,
		forge.WithSummary("Decline invitation"),
		forge.WithDescription("Declines a pending organization invitation using the invitation token. The signed-in user must own, and have verified, the email the invitation was sent to. Answers 401 when signed out and 404 for an unknown token or for anyone but the invitee, 409 for a token already used or withdrawn and 410 for an expired one."),
		forge.WithOperationID("declineInvitation"),
		forge.WithRequestSchema(DeclineInvitationRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Declined", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/orgs/:orgId/invitations", p.handleCreateInvitation,
		forge.WithSummary("Create invitation"),
		forge.WithDescription("Creates an invitation to join an organization."),
		forge.WithOperationID("createInvitation"),
		forge.WithRequestSchema(CreateInvitationRequest{}),
		forge.WithCreatedResponse(organization.Invitation{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/orgs/:orgId/invitations", p.handleListInvitations,
		forge.WithSummary("List invitations"),
		forge.WithDescription("Returns all pending invitations for an organization."),
		forge.WithOperationID("listInvitations"),
		forge.WithResponseSchema(http.StatusOK, "Invitation list", InvitationListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/orgs/:orgId/invitations/:invitationId/resend", p.handleResendInvitation,
		forge.WithSummary("Resend invitation"),
		forge.WithDescription("Gives a pending or expired invitation a new token and a fresh expiry, so the link already sent stops working. Answers 409 for an accepted, declined or revoked invitation and 429 within a minute of the last send."),
		forge.WithOperationID("resendInvitation"),
		forge.WithResponseSchema(http.StatusOK, "Resent invitation", organization.Invitation{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/orgs/:orgId/invitations/:invitationId", p.handleRevokeInvitation,
		forge.WithSummary("Revoke invitation"),
		forge.WithDescription("Revokes a pending or expired invitation. The record is kept with status revoked and its token stops working. Answers 409 for any other status."),
		forge.WithOperationID("revokeInvitation"),
		forge.WithResponseSchema(http.StatusOK, "Revoked invitation", organization.Invitation{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Teams
	if err := g.POST("/orgs/:orgId/teams", p.handleCreateTeam,
		forge.WithSummary("Create team"),
		forge.WithDescription("Creates a new team within an organization."),
		forge.WithOperationID("createTeam"),
		forge.WithRequestSchema(CreateTeamRequest{}),
		forge.WithCreatedResponse(organization.Team{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/orgs/:orgId/teams", p.handleListTeams,
		forge.WithSummary("List teams"),
		forge.WithDescription("Returns all teams in an organization."),
		forge.WithOperationID("listTeams"),
		forge.WithResponseSchema(http.StatusOK, "Team list", TeamListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/orgs/:orgId/teams/:teamId", p.handleGetTeam,
		forge.WithSummary("Get team"),
		forge.WithDescription("Returns details of a specific team."),
		forge.WithOperationID("getTeam"),
		forge.WithResponseSchema(http.StatusOK, "Team details", organization.Team{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/orgs/:orgId/teams/:teamId", p.handleUpdateTeam,
		forge.WithSummary("Update team"),
		forge.WithDescription("Updates a team's name or slug."),
		forge.WithOperationID("updateTeam"),
		forge.WithRequestSchema(UpdateTeamRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated team", organization.Team{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/orgs/:orgId/teams/:teamId", p.handleDeleteTeam,
		forge.WithSummary("Delete team"),
		forge.WithDescription("Deletes a team from an organization."),
		forge.WithOperationID("deleteTeam"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Slug check
	return g.GET("/orgs/check-slug", p.handleCheckSlug,
		forge.WithSummary("Check slug availability"),
		forge.WithDescription("Checks whether an organization slug is available."),
		forge.WithOperationID("checkOrgSlug"),
		forge.WithResponseSchema(http.StatusOK, "Slug availability", SlugAvailableResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Admin route registration
// ──────────────────────────────────────────────────

func (p *Plugin) registerAdminOrgRoutes(router forge.Router) error {
	g := router.Group(p.config.PathPrefix+"/admin",
		forge.WithGroupTags("admin"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(
			authprovider.RegistryMiddleware(p.engine.AuthRegistry(), "session"),
			// An agent's authority is its grant, not its user's admin role.
			// After the session middleware, which resolves the caller, and
			// before the permission check, so the refusal doesn't depend on
			// what the delegating user may do.
			middleware.DenyDelegatedSession(),
			middleware.RequirePermission(p.permChecker, "manage", "organization"),
		),
	)

	return g.GET("/orgs", p.handleAdminListOrgs,
		forge.WithSummary("List organizations (admin)"),
		forge.WithDescription("Returns all organizations for an app. Requires admin role."),
		forge.WithOperationID("adminListOrgs"),
		forge.WithResponseSchema(http.StatusOK, "Organization list", OrgListResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Organization handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleCreateOrg(ctx forge.Context, req *CreateOrgRequest) (*organization.Organization, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Name == "" || req.Slug == "" {
		return nil, forge.BadRequest("name and slug are required")
	}

	// The org lives in the caller's app. A body app_id may only restate it.
	appID, err := plugin.ScopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	// Resolve the env to stamp on the new org. Order:
	//   1. Active env from the environment middleware (header / detection
	//      / app default) — most callers go through this path.
	//   2. App's default environment as a final fallback. This keeps
	//      legacy clients (and tests that don't wire the middleware)
	//      working without surfacing a confusing FK violation from the
	//      store on a NOT NULL env_id column.
	envID, ok := middleware.EnvIDFrom(ctx.Context())
	if !ok || envID.IsNil() {
		resolved, rerr := p.resolveDefaultEnv(ctx.Context(), appID)
		if rerr != nil {
			return nil, forge.BadRequest("no environment for app")
		}
		envID = resolved
	}

	o := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		EnvID:     envID,
		Name:      req.Name,
		Slug:      req.Slug,
		Logo:      req.Logo,
		CreatedBy: userID,
	}

	if err := p.CreateOrganization(ctx.Context(), o); err != nil {
		if errors.Is(err, ErrSlugTaken) {
			return nil, forge.NewHTTPError(http.StatusConflict, "organization slug is already taken")
		}
		// Structured log so the next 500 carries the org context
		// (id, app, env, slug, creator) regardless of which layer
		// blew up. Inner store errors are also logged in service.go.
		if p.logger != nil {
			p.logger.Error("organization handler: create failed",
				log.String("org_id", o.ID.String()),
				log.String("app_id", o.AppID.String()),
				log.String("env_id", o.EnvID.String()),
				log.String("slug", o.Slug),
				log.String("created_by", o.CreatedBy.String()),
				log.String("error", err.Error()),
			)
		}
		return nil, p.mapError(ctx, err)
	}

	return nil, ctx.JSON(http.StatusCreated, o)
}

func (p *Plugin) handleListOrgs(ctx forge.Context, req *ListOrgsRequest) (*OrgListResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	pg, err := p.ListUserOrganizationsPage(ctx.Context(), userID, page.Opts{Limit: req.Limit, Cursor: req.Cursor})
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}
	resp := &OrgListResponse{Organizations: pg.Items, NextCursor: pg.NextCursor}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleGetOrg(ctx forge.Context, _ *GetOrgRequest) (*organization.Organization, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleMember); err != nil {
		return nil, err
	}

	o, err := p.GetOrganization(ctx.Context(), orgID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusOK, o)
}

func (p *Plugin) handleUpdateOrg(ctx forge.Context, req *UpdateOrgRequest) (*organization.Organization, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleAdmin); err != nil {
		return nil, err
	}

	o, err := p.GetOrganization(ctx.Context(), orgID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	if req.Name != nil {
		o.Name = *req.Name
	}
	if req.Logo != nil {
		o.Logo = *req.Logo
	}

	if err := p.UpdateOrganization(ctx.Context(), o); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusOK, o)
}

func (p *Plugin) handleDeleteOrg(ctx forge.Context, _ *DeleteOrgRequest) (*StatusResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleOwner); err != nil {
		return nil, err
	}

	if err := p.DeleteOrganization(ctx.Context(), orgID); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Member handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleListMembers(ctx forge.Context, req *ListMembersRequest) (*MemberListResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleMember); err != nil {
		return nil, err
	}
	pg, err := p.ListMembersPage(ctx.Context(), orgID, page.Opts{Limit: req.Limit, Cursor: req.Cursor})
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}
	resp := &MemberListResponse{Members: pg.Items, NextCursor: pg.NextCursor}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleAddMember(ctx forge.Context, req *AddMemberRequest) (*organization.Member, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	caller, err := p.requireOrgRole(ctx, orgID, organization.RoleAdmin)
	if err != nil {
		return nil, err
	}

	if req.UserID == "" {
		return nil, forge.BadRequest("user_id is required")
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid user_id: %v", err))
	}

	role := organization.RoleMember
	if req.Role != "" {
		role = normalizeRole(req.Role)
	}
	if gerr := p.mayGrantRole(caller.Role, role); gerr != nil {
		return nil, gerr
	}

	m := &organization.Member{
		ID:     id.NewMemberID(),
		OrgID:  orgID,
		UserID: userID,
		Role:   role,
	}

	if err := p.AddMember(ctx.Context(), m); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusCreated, m)
}

func (p *Plugin) handleRemoveMember(ctx forge.Context, _ *RemoveMemberRequest) (*StatusResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	memberID, err := id.ParseMemberID(ctx.Param("memberId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid member id: %v", err))
	}

	caller, err := p.requireOrgRole(ctx, orgID, organization.RoleAdmin)
	if err != nil {
		return nil, err
	}
	if merr := p.assertMemberInOrg(ctx, memberID, orgID); merr != nil {
		return nil, merr
	}
	if err := p.requireRemovable(ctx, caller, memberID, orgID); err != nil {
		return nil, err
	}

	if err := p.RemoveMember(ctx.Context(), memberID); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &StatusResponse{Status: "removed"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleUpdateMember(ctx forge.Context, req *UpdateMemberRequest) (*organization.Member, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	memberID, err := id.ParseMemberID(ctx.Param("memberId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid member id: %v", err))
	}

	if req.Role == "" {
		return nil, forge.BadRequest("role is required")
	}

	// Only an owner may change member roles (including granting ownership),
	// which prevents members/admins from escalating themselves or others.
	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleOwner); err != nil {
		return nil, err
	}
	if merr := p.assertMemberInOrg(ctx, memberID, orgID); merr != nil {
		return nil, merr
	}

	member, err := p.UpdateMemberRole(ctx.Context(), memberID, organization.MemberRole(req.Role))
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusOK, member)
}

// ──────────────────────────────────────────────────
// Invitation handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleCreateInvitation(ctx forge.Context, req *CreateInvitationRequest) (*organization.Invitation, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	inviter, err := p.requireOrgRole(ctx, orgID, organization.RoleAdmin)
	if err != nil {
		return nil, err
	}
	inviterID := inviter.UserID

	if req.Email == "" {
		return nil, forge.BadRequest("email is required")
	}

	role := organization.RoleMember
	if req.Role != "" {
		role = normalizeRole(req.Role)
	}
	if gerr := p.mayGrantRole(inviter.Role, role); gerr != nil {
		return nil, gerr
	}

	token, err := account.GenerateVerificationToken()
	if err != nil {
		return nil, middleware.InternalError(ctx, err)
	}

	inv := &organization.Invitation{
		ID:        id.NewInvitationID(),
		OrgID:     orgID,
		Email:     req.Email,
		Role:      role,
		InviterID: inviterID,
		Status:    organization.InvitationPending,
		Token:     token,
	}

	if err := p.CreateInvitation(ctx.Context(), inv); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusCreated, inv)
}

func (p *Plugin) handleListInvitations(ctx forge.Context, req *ListInvitationsRequest) (*InvitationListResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleAdmin); err != nil {
		return nil, err
	}
	pg, err := p.ListInvitationsPage(ctx.Context(), orgID, page.Opts{Limit: req.Limit, Cursor: req.Cursor})
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}
	resp := &InvitationListResponse{Invitations: pg.Items, NextCursor: pg.NextCursor}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// authorizeInvitationChange is what resending and revoking share with creating
// an invitation: the caller is an admin of the org, and may grant the role the
// invitation carries (an admin cannot touch an owner's invitation). It returns
// the invitation's org and ID.
func (p *Plugin) authorizeInvitationChange(ctx forge.Context) (id.OrgID, id.InvitationID, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return id.OrgID{}, id.InvitationID{}, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	invID, err := id.ParseInvitationID(ctx.Param("invitationId"))
	if err != nil {
		return id.OrgID{}, id.InvitationID{}, forge.BadRequest(fmt.Sprintf("invalid invitation id: %v", err))
	}
	caller, err := p.requireOrgRole(ctx, orgID, organization.RoleAdmin)
	if err != nil {
		return id.OrgID{}, id.InvitationID{}, err
	}
	inv, err := p.GetInvitation(ctx.Context(), orgID, invID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return id.OrgID{}, id.InvitationID{}, forge.NotFound("invitation not found")
		}
		return id.OrgID{}, id.InvitationID{}, p.mapError(ctx, err)
	}
	if gerr := p.mayGrantRole(caller.Role, inv.Role); gerr != nil {
		return id.OrgID{}, id.InvitationID{}, gerr
	}
	return orgID, invID, nil
}

// mapInvitationChangeError turns the resend and revoke sentinels into their
// status codes.
func (p *Plugin) mapInvitationChangeError(ctx forge.Context, err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return forge.NotFound("invitation not found")
	case errors.Is(err, ErrInvitationNotActionable):
		return forge.NewHTTPError(http.StatusConflict, "only a pending or expired invitation can be changed")
	case errors.Is(err, ErrInvitationResendTooSoon):
		return forge.NewHTTPError(http.StatusTooManyRequests, "this invitation was sent a moment ago; try again in a minute")
	}
	return p.mapError(ctx, err)
}

func (p *Plugin) handleResendInvitation(ctx forge.Context, _ *ResendInvitationRequest) (*organization.Invitation, error) {
	orgID, invID, err := p.authorizeInvitationChange(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := p.ResendInvitation(ctx.Context(), orgID, invID)
	if err != nil {
		return nil, p.mapInvitationChangeError(ctx, err)
	}
	return nil, ctx.JSON(http.StatusOK, inv)
}

func (p *Plugin) handleRevokeInvitation(ctx forge.Context, _ *RevokeInvitationRequest) (*organization.Invitation, error) {
	orgID, invID, err := p.authorizeInvitationChange(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := p.RevokeInvitation(ctx.Context(), orgID, invID)
	if err != nil {
		return nil, p.mapInvitationChangeError(ctx, err)
	}
	return nil, ctx.JSON(http.StatusOK, inv)
}

// invitationTokenError maps what accepting or declining by token can return.
// An unknown token and a token that is someone else's are the same 404; only
// the invitee can see 409 (used or withdrawn) or 410 (expired).
func invitationTokenError(ctx forge.Context, err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return forge.NotFound("invitation not found")
	case errors.Is(err, ErrInvitationExpired):
		return forge.NewHTTPError(http.StatusGone, "invitation has expired")
	case errors.Is(err, ErrInvitationNotPending):
		return forge.NewHTTPError(http.StatusConflict, "this invitation was already used or withdrawn")
	}
	return mapErrorCtx(ctx, err)
}

func (p *Plugin) handleAcceptInvitation(ctx forge.Context, req *AcceptInvitationRequest) (*organization.Member, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	if req.Token == "" {
		return nil, forge.BadRequest("token is required")
	}

	member, err := p.AcceptInvitationAs(ctx.Context(), req.Token, userID)
	if err != nil {
		return nil, invitationTokenError(ctx, err)
	}

	return nil, ctx.JSON(http.StatusCreated, member)
}

// maskEmail turns "rex.raphael@acme.com" into "re***@acme.com".
//
// Enough for someone to recognise their own address, not enough for whoever
// forwarded or intercepted the link to learn a new one. The domain stays
// whole because it is the part that tells an invitee which of their accounts
// to sign in with.
func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	local, domain := email[:at], email[at:]
	keep := 2
	if len(local) < keep {
		keep = len(local)
	}
	return local[:keep] + "***" + domain
}

// inviterName is the best human-readable name available for a user, falling
// back through the fields until something is non-empty. An inviter whose
// account was deleted leaves "Someone" rather than an empty line.
func inviterName(u *user.User) string {
	if u == nil {
		return "Someone"
	}
	if name := strings.TrimSpace(u.FirstName + " " + u.LastName); name != "" {
		return name
	}
	if u.DisplayUsername != "" {
		return u.DisplayUsername
	}
	if u.Username != "" {
		return u.Username
	}
	if u.Email != "" {
		return u.Email
	}
	return "Someone"
}

func (p *Plugin) handleGetInvitation(ctx forge.Context, req *GetInvitationRequest) (*InvitationPreview, error) {
	token := req.Token
	if token == "" {
		token = ctx.Param("inviteToken")
	}
	if token == "" {
		return nil, forge.BadRequest("token is required")
	}

	inv, err := p.store.GetInvitationByToken(ctx.Context(), token)
	// One answer for every reason an invitation cannot be shown: never
	// existed, already used, declined, expired. Telling them apart would let
	// anyone holding a list of guesses learn which tokens are real.
	if err != nil || inv == nil || inv.Status != organization.InvitationPending {
		return nil, forge.NotFound("invitation not found")
	}
	if !inv.ExpiresAt.IsZero() && time.Now().After(inv.ExpiresAt) {
		return nil, forge.NotFound("invitation not found")
	}

	org, err := p.store.GetOrganization(ctx.Context(), inv.OrgID)
	if err != nil {
		return nil, forge.NotFound("invitation not found")
	}

	// A missing inviter is not a reason to refuse the invitation. The account
	// may have been deleted since it was sent, and the seat is still valid.
	var invitedBy *user.User
	if u, uErr := p.store.GetUser(ctx.Context(), inv.InviterID); uErr == nil {
		invitedBy = u
	}

	resp := &InvitationPreview{
		OrgID:       inv.OrgID.String(),
		OrgName:     org.Name,
		OrgSlug:     org.Slug,
		InviterName: inviterName(invitedBy),
		Role:        string(inv.Role),
		EmailMasked: maskEmail(inv.Email),
		ExpiresAt:   inv.ExpiresAt,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleDeclineInvitation(ctx forge.Context, req *DeclineInvitationRequest) (*StatusResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	if req.Token == "" {
		return nil, forge.BadRequest("token is required")
	}

	if err := p.DeclineInvitationAs(ctx.Context(), req.Token, userID); err != nil {
		return nil, invitationTokenError(ctx, err)
	}

	resp := &StatusResponse{Status: "declined"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Team handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleCreateTeam(ctx forge.Context, req *CreateTeamRequest) (*organization.Team, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleAdmin); err != nil {
		return nil, err
	}

	if req.Name == "" || req.Slug == "" {
		return nil, forge.BadRequest("name and slug are required")
	}

	t := &organization.Team{
		ID:    id.NewTeamID(),
		OrgID: orgID,
		Name:  req.Name,
		Slug:  req.Slug,
	}

	if err := p.CreateTeam(ctx.Context(), t); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusCreated, t)
}

func (p *Plugin) handleListTeams(ctx forge.Context, req *ListTeamsRequest) (*TeamListResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleMember); err != nil {
		return nil, err
	}
	pg, err := p.ListTeamsPage(ctx.Context(), orgID, page.Opts{Limit: req.Limit, Cursor: req.Cursor})
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}
	resp := &TeamListResponse{Teams: pg.Items, NextCursor: pg.NextCursor}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleGetTeam(ctx forge.Context, _ *GetTeamRequest) (*organization.Team, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	teamID, err := id.ParseTeamID(ctx.Param("teamId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid team id: %v", err))
	}

	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleMember); err != nil {
		return nil, err
	}
	if terr := p.assertTeamInOrg(ctx, teamID, orgID); terr != nil {
		return nil, terr
	}

	t, err := p.GetTeam(ctx.Context(), teamID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusOK, t)
}

func (p *Plugin) handleUpdateTeam(ctx forge.Context, req *UpdateTeamRequest) (*organization.Team, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	teamID, err := id.ParseTeamID(ctx.Param("teamId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid team id: %v", err))
	}

	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleAdmin); err != nil {
		return nil, err
	}
	if terr := p.assertTeamInOrg(ctx, teamID, orgID); terr != nil {
		return nil, terr
	}

	t, err := p.GetTeam(ctx.Context(), teamID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Slug != nil {
		t.Slug = *req.Slug
	}

	if err := p.UpdateTeam(ctx.Context(), t); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	return nil, ctx.JSON(http.StatusOK, t)
}

func (p *Plugin) handleDeleteTeam(ctx forge.Context, _ *DeleteTeamRequest) (*StatusResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}
	teamID, err := id.ParseTeamID(ctx.Param("teamId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid team id: %v", err))
	}

	if _, err = p.requireOrgRole(ctx, orgID, organization.RoleAdmin); err != nil {
		return nil, err
	}
	if terr := p.assertTeamInOrg(ctx, teamID, orgID); terr != nil {
		return nil, terr
	}

	if err := p.DeleteTeam(ctx.Context(), teamID); err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Slug check handler
// ──────────────────────────────────────────────────

func (p *Plugin) handleCheckSlug(ctx forge.Context, req *CheckSlugRequest) (*SlugAvailableResponse, error) {
	if _, ok := middleware.UserIDFrom(ctx.Context()); !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Slug == "" {
		return nil, forge.BadRequest("slug query parameter is required")
	}

	appID, err := p.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	available, err := p.IsOrgSlugAvailable(ctx.Context(), appID, req.Slug)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &SlugAvailableResponse{Available: available}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Admin handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleAdminListOrgs(ctx forge.Context, req *AdminListOrgsRequest) (*OrgListResponse, error) {
	appID, err := plugin.ScopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	orgs, err := p.AdminListOrganizations(ctx.Context(), appID)
	if err != nil {
		return nil, mapErrorCtx(ctx, err)
	}

	resp := &OrgListResponse{Organizations: orgs}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Error mapping
// ──────────────────────────────────────────────────

// mapError converts a service-layer error into a forge HTTP error.
// Free function form retained for handlers that don't have plugin
// receiver access; new call sites prefer (*Plugin).mapError so the
// error gets logged with structured context before being collapsed
// to a generic 500.
// mapErrorCtx maps domain errors to HTTP errors; an unknown error is ours and
// answers a generic 500 with the request id.
func mapErrorCtx(ctx forge.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrConflict) {
		return forge.NewHTTPError(http.StatusConflict, "conflicts with an existing record")
	}
	return middleware.InternalError(ctx, err)
}

// mapError on *Plugin logs the underlying error before mapping it
// to forge.InternalError. Handlers that have already logged the
// error with rich field context (org_id, app_id, etc.) can still
// call this — the duplicate `error` line is cheap and keeps the
// final cause adjacent to the HTTP-conversion boundary.
func (p *Plugin) mapError(ctx forge.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrConflict) {
		return forge.NewHTTPError(http.StatusConflict, "conflicts with an existing record")
	}
	if p != nil && p.logger != nil {
		p.logger.Error("organization plugin: handler error",
			log.String("error", err.Error()),
		)
	}
	return middleware.InternalError(ctx, err)
}
