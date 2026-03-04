package organization

import (
	"fmt"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
)

// RegisterRoutes registers organization management routes on a forge.Router.
func (p *Plugin) RegisterRoutes(r any) error {
	router, ok := r.(forge.Router)
	if !ok {
		return fmt.Errorf("organization: expected forge.Router, got %T", r)
	}

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
		forge.WithDescription("Accepts a pending organization invitation using the invitation token."),
		forge.WithOperationID("acceptInvitation"),
		forge.WithRequestSchema(AcceptInvitationRequest{}),
		forge.WithCreatedResponse(organization.Member{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/orgs/invitations/decline", p.handleDeclineInvitation,
		forge.WithSummary("Decline invitation"),
		forge.WithDescription("Declines a pending organization invitation using the invitation token."),
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
		forge.WithGroupMiddleware(
			middleware.RequireAuth(),
			middleware.RequireAnyRole(p.roleChecker, "admin", "super_admin"),
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

	appID, err := p.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	o := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      req.Name,
		Slug:      req.Slug,
		Logo:      req.Logo,
		CreatedBy: userID,
	}

	if err := p.CreateOrganization(ctx.Context(), o); err != nil {
		return nil, mapError(err)
	}

	return o, ctx.JSON(http.StatusCreated, o)
}

func (p *Plugin) handleListOrgs(ctx forge.Context, _ *ListOrgsRequest) (*OrgListResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	orgs, err := p.ListUserOrganizations(ctx.Context(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	if orgs == nil {
		orgs = []*organization.Organization{}
	}
	resp := &OrgListResponse{Organizations: orgs}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleGetOrg(ctx forge.Context, _ *GetOrgRequest) (*organization.Organization, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	o, err := p.GetOrganization(ctx.Context(), orgID)
	if err != nil {
		return nil, mapError(err)
	}

	return o, ctx.JSON(http.StatusOK, o)
}

func (p *Plugin) handleUpdateOrg(ctx forge.Context, req *UpdateOrgRequest) (*organization.Organization, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	o, err := p.GetOrganization(ctx.Context(), orgID)
	if err != nil {
		return nil, mapError(err)
	}

	if req.Name != nil {
		o.Name = *req.Name
	}
	if req.Logo != nil {
		o.Logo = *req.Logo
	}

	if err := p.UpdateOrganization(ctx.Context(), o); err != nil {
		return nil, mapError(err)
	}

	return o, ctx.JSON(http.StatusOK, o)
}

func (p *Plugin) handleDeleteOrg(ctx forge.Context, _ *DeleteOrgRequest) (*StatusResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	if err := p.DeleteOrganization(ctx.Context(), orgID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Member handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleListMembers(ctx forge.Context, _ *ListMembersRequest) (*MemberListResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	members, err := p.ListMembers(ctx.Context(), orgID)
	if err != nil {
		return nil, mapError(err)
	}

	if members == nil {
		members = []*organization.Member{}
	}
	resp := &MemberListResponse{Members: members}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleAddMember(ctx forge.Context, req *AddMemberRequest) (*organization.Member, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
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
		role = organization.MemberRole(req.Role)
	}

	m := &organization.Member{
		ID:     id.NewMemberID(),
		OrgID:  orgID,
		UserID: userID,
		Role:   role,
	}

	if err := p.AddMember(ctx.Context(), m); err != nil {
		return nil, mapError(err)
	}

	return m, ctx.JSON(http.StatusCreated, m)
}

func (p *Plugin) handleRemoveMember(ctx forge.Context, _ *RemoveMemberRequest) (*StatusResponse, error) {
	memberID, err := id.ParseMemberID(ctx.Param("memberId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid member id: %v", err))
	}

	if err := p.RemoveMember(ctx.Context(), memberID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "removed"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleUpdateMember(ctx forge.Context, req *UpdateMemberRequest) (*organization.Member, error) {
	memberID, err := id.ParseMemberID(ctx.Param("memberId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid member id: %v", err))
	}

	if req.Role == "" {
		return nil, forge.BadRequest("role is required")
	}

	member, err := p.UpdateMemberRole(ctx.Context(), memberID, organization.MemberRole(req.Role))
	if err != nil {
		return nil, mapError(err)
	}

	return member, ctx.JSON(http.StatusOK, member)
}

// ──────────────────────────────────────────────────
// Invitation handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleCreateInvitation(ctx forge.Context, req *CreateInvitationRequest) (*organization.Invitation, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	inviterID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Email == "" {
		return nil, forge.BadRequest("email is required")
	}

	role := organization.RoleMember
	if req.Role != "" {
		role = organization.MemberRole(req.Role)
	}

	token, err := account.GenerateVerificationToken()
	if err != nil {
		return nil, forge.InternalError(err)
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
		return nil, mapError(err)
	}

	return inv, ctx.JSON(http.StatusCreated, inv)
}

func (p *Plugin) handleListInvitations(ctx forge.Context, _ *ListInvitationsRequest) (*InvitationListResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	invitations, err := p.ListInvitations(ctx.Context(), orgID)
	if err != nil {
		return nil, mapError(err)
	}

	if invitations == nil {
		invitations = []*organization.Invitation{}
	}
	resp := &InvitationListResponse{Invitations: invitations}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleAcceptInvitation(ctx forge.Context, req *AcceptInvitationRequest) (*organization.Member, error) {
	if req.Token == "" {
		return nil, forge.BadRequest("token is required")
	}

	member, err := p.AcceptInvitation(ctx.Context(), req.Token)
	if err != nil {
		return nil, mapError(err)
	}

	return member, ctx.JSON(http.StatusCreated, member)
}

func (p *Plugin) handleDeclineInvitation(ctx forge.Context, req *DeclineInvitationRequest) (*StatusResponse, error) {
	if req.Token == "" {
		return nil, forge.BadRequest("token is required")
	}

	if err := p.DeclineInvitation(ctx.Context(), req.Token); err != nil {
		return nil, mapError(err)
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
		return nil, mapError(err)
	}

	return t, ctx.JSON(http.StatusCreated, t)
}

func (p *Plugin) handleListTeams(ctx forge.Context, _ *ListTeamsRequest) (*TeamListResponse, error) {
	orgID, err := id.ParseOrgID(ctx.Param("orgId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", err))
	}

	teams, err := p.ListTeams(ctx.Context(), orgID)
	if err != nil {
		return nil, mapError(err)
	}

	if teams == nil {
		teams = []*organization.Team{}
	}
	resp := &TeamListResponse{Teams: teams}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleGetTeam(ctx forge.Context, _ *GetTeamRequest) (*organization.Team, error) {
	teamID, err := id.ParseTeamID(ctx.Param("teamId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid team id: %v", err))
	}

	t, err := p.GetTeam(ctx.Context(), teamID)
	if err != nil {
		return nil, mapError(err)
	}

	return t, ctx.JSON(http.StatusOK, t)
}

func (p *Plugin) handleUpdateTeam(ctx forge.Context, req *UpdateTeamRequest) (*organization.Team, error) {
	teamID, err := id.ParseTeamID(ctx.Param("teamId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid team id: %v", err))
	}

	t, err := p.GetTeam(ctx.Context(), teamID)
	if err != nil {
		return nil, mapError(err)
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Slug != nil {
		t.Slug = *req.Slug
	}

	if err := p.UpdateTeam(ctx.Context(), t); err != nil {
		return nil, mapError(err)
	}

	return t, ctx.JSON(http.StatusOK, t)
}

func (p *Plugin) handleDeleteTeam(ctx forge.Context, _ *DeleteTeamRequest) (*StatusResponse, error) {
	teamID, err := id.ParseTeamID(ctx.Param("teamId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid team id: %v", err))
	}

	if err := p.DeleteTeam(ctx.Context(), teamID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Slug check handler
// ──────────────────────────────────────────────────

func (p *Plugin) handleCheckSlug(ctx forge.Context, req *CheckSlugRequest) (*SlugAvailableResponse, error) {
	if req.Slug == "" {
		return nil, forge.BadRequest("slug query parameter is required")
	}

	appID, err := p.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	available, err := p.IsOrgSlugAvailable(ctx.Context(), appID, req.Slug)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &SlugAvailableResponse{Available: available}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Admin handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleAdminListOrgs(ctx forge.Context, req *AdminListOrgsRequest) (*OrgListResponse, error) {
	appID, err := p.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	orgs, err := p.AdminListOrganizations(ctx.Context(), appID)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &OrgListResponse{Organizations: orgs}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Error mapping
// ──────────────────────────────────────────────────

func mapError(err error) error {
	if err == nil {
		return nil
	}
	return forge.InternalError(err)
}
