package organization

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// OrganizationHandler handles organization-related HTTP requests.
type OrganizationHandler struct {
	orgService *organization.Service
	plugin     *Plugin // Reference to plugin for notification sending
}

// Handler-level request types with path parameters.
type CreateOrganizationHandlerRequest struct {
	organization.CreateOrganizationRequest
}

type GetOrganizationRequest struct {
	ID string `path:"id" validate:"required"`
}

type ListOrganizationsRequest struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

type UpdateOrganizationHandlerRequest struct {
	organization.UpdateOrganizationRequest

	ID string `path:"id" validate:"required"`
}

type DeleteOrganizationRequest struct {
	ID string `path:"id" validate:"required"`
}

type GetOrganizationBySlugRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type ListMembersRequest struct {
	ID    string `path:"id"     validate:"required"`
	Page  int    `query:"page"`
	Limit int    `query:"limit"`
	Role  string `query:"role"`
}

type InviteMemberHandlerRequest struct {
	organization.InviteMemberRequest

	ID string `path:"id" validate:"required"`
}

type UpdateMemberHandlerRequest struct {
	organization.UpdateMemberRequest

	ID       string `path:"id"       validate:"required"`
	MemberID string `path:"memberId" validate:"required"`
}

type RemoveMemberRequest struct {
	ID       string `path:"id"       validate:"required"`
	MemberID string `path:"memberId" validate:"required"`
}

type AcceptInvitationRequest struct {
	Token string `json:"token" validate:"required"`
}

type DeclineInvitationRequest struct {
	Token string `json:"token" validate:"required"`
}

type ListTeamsRequest struct {
	ID    string `path:"id"     validate:"required"`
	Page  int    `query:"page"`
	Limit int    `query:"limit"`
}

type CreateTeamHandlerRequest struct {
	organization.CreateTeamRequest

	ID string `path:"id" validate:"required"`
}

type UpdateTeamHandlerRequest struct {
	organization.UpdateTeamRequest

	ID     string `path:"id"     validate:"required"`
	TeamID string `path:"teamId" validate:"required"`
}

type DeleteTeamRequest struct {
	ID     string `path:"id"     validate:"required"`
	TeamID string `path:"teamId" validate:"required"`
}

// Response types
// Use shared response type.
type MessageResponse = responses.MessageResponse

type MembersResponse struct {
	Members []*organization.Member `json:"members"`
	Total   int                    `json:"total,omitempty"`
}

type InvitationResponse struct {
	Invitation *organization.Invitation `json:"invitation"`
	Message    string                   `json:"message,omitempty"`
}

type TeamsResponse struct {
	Teams []*organization.Team `json:"teams"`
	Total int                  `json:"total,omitempty"`
}

// NewOrganizationHandler creates a new organization handler.
func NewOrganizationHandler(orgService *organization.Service) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
	}
}

// CreateOrganization handles organization creation requests.
func (h *OrganizationHandler) CreateOrganization(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(400, errs.BadRequest("app context required"))
	}

	// Get environment ID from context
	environmentID, ok := contexts.GetEnvironmentID(ctx)
	if !ok || environmentID.IsNil() {
		return c.JSON(400, errs.BadRequest("environment context required"))
	}

	var req CreateOrganizationHandlerRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	org, err := h.orgService.CreateOrganization(ctx, &req.CreateOrganizationRequest, userID, appID, environmentID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(201, org)
}

// GetOrganization handles get organization requests.
func (h *OrganizationHandler) GetOrganization(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req GetOrganizationRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	orgID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid organization ID"))
	}

	// Check if user is a member of the organization (view permission)
	isMember, _ := h.orgService.IsMember(ctx, orgID, userID)
	if !isMember {
		return c.JSON(403, errs.PermissionDenied("view", "organization"))
	}

	org, err := h.orgService.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return c.JSON(404, errs.OrganizationNotFound())
	}

	return c.JSON(200, org)
}

// ListOrganizations handles list organizations requests (user's organizations).
func (h *OrganizationHandler) ListOrganizations(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req ListOrganizationsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Set defaults
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	filter := &pagination.PaginationParams{
		Page:  page,
		Limit: limit,
	}

	orgs, err := h.orgService.ListUserOrganizations(ctx, userID, filter)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, orgs)
}

// UpdateOrganization handles organization update requests.
func (h *OrganizationHandler) UpdateOrganization(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req UpdateOrganizationHandlerRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	orgID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid organization ID"))
	}

	// Check permission to edit organization using RBAC
	if err := h.orgService.RequirePermission(ctx, orgID, userID, "edit", "organization"); err != nil {
		// Fallback to admin check if RBAC denies (policies may not be configured)
		if err := h.orgService.RequireAdmin(ctx, orgID, userID); err != nil {
			return c.JSON(403, err)
		}
	}

	org, err := h.orgService.UpdateOrganization(ctx, orgID, &req.UpdateOrganizationRequest)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, org)
}

// DeleteOrganization handles organization deletion requests.
func (h *OrganizationHandler) DeleteOrganization(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req DeleteOrganizationRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	orgID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid organization ID"))
	}

	// Check permission to delete organization using RBAC
	if err := h.orgService.RequirePermission(ctx, orgID, userID, "delete", "organization"); err != nil {
		// Fallback to owner check - only owners can delete
		if err := h.orgService.RequireOwner(ctx, orgID, userID); err != nil {
			return c.JSON(403, err)
		}
	}

	err = h.orgService.DeleteOrganization(ctx, orgID, userID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(204, nil)
}

// GetOrganizationBySlug handles get organization by slug requests.
func (h *OrganizationHandler) GetOrganizationBySlug(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req GetOrganizationBySlugRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(400, errs.BadRequest("app context required"))
	}

	// Get environment ID from context
	environmentID, ok := contexts.GetEnvironmentID(ctx)
	if !ok || environmentID.IsNil() {
		return c.JSON(400, errs.BadRequest("environment context required"))
	}

	org, err := h.orgService.FindOrganizationBySlug(ctx, appID, environmentID, req.Slug)
	if err != nil {
		return c.JSON(404, errs.OrganizationNotFound())
	}

	// Check if user is a member of the organization
	isMember, _ := h.orgService.IsMember(ctx, org.ID, userID)
	if !isMember {
		return c.JSON(403, errs.PermissionDenied("view", "organization"))
	}

	return c.JSON(200, org)
}

// ListMembers handles list organization members requests.
func (h *OrganizationHandler) ListMembers(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req ListMembersRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	orgID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid organization ID"))
	}

	// Check if user is a member of the organization (view permission)
	isMember, _ := h.orgService.IsMember(ctx, orgID, userID)
	if !isMember {
		return c.JSON(403, errs.PermissionDenied("view", "organization"))
	}

	// Set defaults
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	filter := &organization.ListMembersFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		OrganizationID: orgID,
		Role:           &req.Role,
	}

	members, err := h.orgService.ListMembers(ctx, filter)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, members)
}

// InviteMember handles member invitation requests.
func (h *OrganizationHandler) InviteMember(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req InviteMemberHandlerRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	orgID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid organization ID"))
	}

	// Check permission to create members using RBAC
	if err := h.orgService.RequirePermission(ctx, orgID, userID, "create", "members"); err != nil {
		// Fallback to admin check
		if err := h.orgService.RequireAdmin(ctx, orgID, userID); err != nil {
			return c.JSON(403, err)
		}
	}

	invitation, err := h.orgService.InviteMember(ctx, orgID, &req.InviteMemberRequest, userID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	// Send invitation notification if plugin reference is available
	if h.plugin != nil && h.plugin.notifAdapter != nil {
		// Get organization details
		org, err := h.orgService.FindOrganizationByID(ctx, orgID)
		if err == nil {
			// Get inviter user details
			userSvc := h.plugin.authInst.GetServiceRegistry().UserService()
			if userSvc != nil {
				inviter, err := userSvc.FindByID(ctx, userID)
				if err == nil {
					// Send invitation notification (errors are logged, not returned)
					_ = h.plugin.SendInvitationNotification(ctx, invitation, inviter, org)
				}
			}
		}
	}

	return c.JSON(201, invitation)
}

// UpdateMember handles member update requests.
func (h *OrganizationHandler) UpdateMember(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req UpdateMemberHandlerRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	memberID, err := xid.FromString(req.MemberID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid member ID"))
	}

	// Permission check is handled in the service layer (UpdateMember checks admin/owner)
	member, err := h.orgService.UpdateMember(ctx, memberID, &req.UpdateMemberRequest, userID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, member)
}

// RemoveMember handles member removal requests.
func (h *OrganizationHandler) RemoveMember(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req RemoveMemberRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	memberID, err := xid.FromString(req.MemberID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid member ID"))
	}

	// Permission check is handled in the service layer (RemoveMember checks admin/owner)
	err = h.orgService.RemoveMember(ctx, memberID, userID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(204, nil)
}

// AcceptInvitation handles invitation acceptance requests.
func (h *OrganizationHandler) AcceptInvitation(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req AcceptInvitationRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	member, err := h.orgService.AcceptInvitation(ctx, req.Token, userID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, member)
}

// DeclineInvitation handles invitation decline requests.
func (h *OrganizationHandler) DeclineInvitation(c forge.Context) error {
	ctx := c.Request().Context()

	// User doesn't need to be authenticated to decline an invitation
	// (they might not have an account yet)

	var req DeclineInvitationRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	err := h.orgService.DeclineInvitation(ctx, req.Token)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, &StatusResponse{Status: "declined"})
}

// ListTeams handles list teams requests.
func (h *OrganizationHandler) ListTeams(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req ListTeamsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	orgID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid organization ID"))
	}

	// Check if user is a member of the organization
	isMember, _ := h.orgService.IsMember(ctx, orgID, userID)
	if !isMember {
		return c.JSON(403, errs.PermissionDenied("view", "organization"))
	}

	// Set defaults
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	filter := &organization.ListTeamsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		OrganizationID: orgID,
	}

	teams, err := h.orgService.ListTeams(ctx, filter)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, teams)
}

// CreateTeam handles team creation requests.
func (h *OrganizationHandler) CreateTeam(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req CreateTeamHandlerRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	orgID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid organization ID"))
	}

	// Check permission to create teams using RBAC
	if err := h.orgService.RequirePermission(ctx, orgID, userID, "create", "teams"); err != nil {
		// Fallback to admin check
		if err := h.orgService.RequireAdmin(ctx, orgID, userID); err != nil {
			return c.JSON(403, err)
		}
	}

	team, err := h.orgService.CreateTeam(ctx, orgID, &req.CreateTeamRequest, userID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(201, team)
}

// UpdateTeam handles team update requests.
func (h *OrganizationHandler) UpdateTeam(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req UpdateTeamHandlerRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	teamID, err := xid.FromString(req.TeamID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid team ID"))
	}

	// Permission check is handled in the service layer (UpdateTeam checks admin/owner)
	team, err := h.orgService.UpdateTeam(ctx, teamID, &req.UpdateTeamRequest, userID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	// Check if team is SCIM-managed and return with warning if so
	if team.ProvisionedBy != nil && *team.ProvisionedBy == "scim" {
		response := responses.NewResponseWithWarnings(team)
		response.AddWarning("scim_managed_team", "This team is managed via SCIM provisioning. Manual changes may be overwritten by the identity provider.")

		return c.JSON(200, response)
	}

	return c.JSON(200, team)
}

// DeleteTeam handles team deletion requests.
func (h *OrganizationHandler) DeleteTeam(c forge.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user ID
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	var req DeleteTeamRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	teamID, err := xid.FromString(req.TeamID)
	if err != nil {
		return c.JSON(400, errs.BadRequest("invalid team ID"))
	}

	// Check if team is SCIM-managed before deletion
	team, err := h.orgService.FindTeamByID(ctx, teamID)
	if err != nil {
		return c.JSON(404, errs.NotFound("team not found"))
	}

	// Permission check is handled in the service layer (DeleteTeam checks admin/owner)
	err = h.orgService.DeleteTeam(ctx, teamID, userID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	// If team was SCIM-managed, return warning in response
	if team.ProvisionedBy != nil && *team.ProvisionedBy == "scim" {
		response := responses.NewResponseWithWarnings(map[string]string{"message": "Team deleted"})
		response.AddWarning("scim_managed_team", "This team was managed via SCIM provisioning. The deletion may be reversed by the identity provider.")

		return c.JSON(200, response)
	}

	return c.JSON(204, nil)
}
