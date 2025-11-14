package organization

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/forge"
)

// OrganizationHandler handles organization-related HTTP requests
type OrganizationHandler struct {
	orgService *organization.Service
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(orgService *organization.Service) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
	}
}

// CreateOrganization handles organization creation requests
func (h *OrganizationHandler) CreateOrganization(c forge.Context) error {
	var req CreateOrganizationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	// TODO: Get app ID from context/middleware
	appID := xid.New() // Placeholder

	// TODO: Get environment ID from context/middleware
	environmentID := xid.New() // Placeholder

	org, err := h.orgService.CreateOrganization(c.Request().Context(), &req, userID, appID, environmentID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, org)
}

// GetOrganization handles get organization requests
func (h *OrganizationHandler) GetOrganization(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}

	orgID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	org, err := h.orgService.FindOrganizationByID(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "organization not found"})
	}

	return c.JSON(200, org)
}

// ListOrganizations handles list organizations requests (user's organizations)
func (h *OrganizationHandler) ListOrganizations(c forge.Context) error {
	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	// Get pagination parameters
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 10
	}
	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}

	filter := &pagination.PaginationParams{
		Page:  page,
		Limit: limit,
	}

	orgs, err := h.orgService.ListUserOrganizations(c.Request().Context(), userID, filter)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"organizations": orgs,
	})
}

// UpdateOrganization handles organization update requests
func (h *OrganizationHandler) UpdateOrganization(c forge.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}

	orgID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	var req UpdateOrganizationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	org, err := h.orgService.UpdateOrganization(c.Request().Context(), orgID, &req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, org)
}

// DeleteOrganization handles organization deletion requests
func (h *OrganizationHandler) DeleteOrganization(c forge.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}

	orgID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	err = h.orgService.DeleteOrganization(c.Request().Context(), orgID, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// GetOrganizationBySlug handles get organization by slug requests
func (h *OrganizationHandler) GetOrganizationBySlug(c forge.Context) error {
	slug := c.Param("slug")
	if slug == "" {
		return c.JSON(400, map[string]string{"error": "organization slug is required"})
	}

	// TODO: Get app ID from context/middleware
	appID := xid.New() // Placeholder

	// TODO: Get environment ID from context/middleware
	environmentID := xid.New() // Placeholder

	org, err := h.orgService.FindOrganizationBySlug(c.Request().Context(), appID, environmentID, slug)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "organization not found"})
	}

	return c.JSON(200, org)
}

// ListMembers handles list organization members requests
func (h *OrganizationHandler) ListMembers(c forge.Context) error {
	orgIDStr := c.Param("id")
	if orgIDStr == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	// Parse pagination parameters
	limitStr := c.Request().URL.Query().Get("limit")
	pageStr := c.Request().URL.Query().Get("page")

	limit := 10 // default
	page := 1   // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	filter := &organization.ListMembersFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		OrganizationID: orgID,
	}

	members, err := h.orgService.ListMembers(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, members)
}

// InviteMember handles member invitation requests
func (h *OrganizationHandler) InviteMember(c forge.Context) error {
	orgIDStr := c.Param("id")
	if orgIDStr == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	var req InviteMemberRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	invitation, err := h.orgService.InviteMember(c.Request().Context(), orgID, &req, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, invitation)
}

// UpdateMember handles member update requests
func (h *OrganizationHandler) UpdateMember(c forge.Context) error {
	memberIDStr := c.Param("memberId")
	if memberIDStr == "" {
		return c.JSON(400, map[string]string{"error": "member ID is required"})
	}
	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid member ID"})
	}

	var req UpdateMemberRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	member, err := h.orgService.UpdateMember(c.Request().Context(), memberID, &req, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, member)
}

// RemoveMember handles member removal requests
func (h *OrganizationHandler) RemoveMember(c forge.Context) error {
	memberIDStr := c.Param("memberId")
	if memberIDStr == "" {
		return c.JSON(400, map[string]string{"error": "member ID is required"})
	}
	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid member ID"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	err = h.orgService.RemoveMember(c.Request().Context(), memberID, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// AcceptInvitation handles invitation acceptance requests
func (h *OrganizationHandler) AcceptInvitation(c forge.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(400, map[string]string{"error": "invitation token is required"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	member, err := h.orgService.AcceptInvitation(c.Request().Context(), token, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, member)
}

// DeclineInvitation handles invitation decline requests
func (h *OrganizationHandler) DeclineInvitation(c forge.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(400, map[string]string{"error": "invitation token is required"})
	}

	err := h.orgService.DeclineInvitation(c.Request().Context(), token)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "declined"})
}

// ListTeams handles list teams requests
func (h *OrganizationHandler) ListTeams(c forge.Context) error {
	orgIDStr := c.Param("id")
	if orgIDStr == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	// Parse pagination parameters
	limitStr := c.Request().URL.Query().Get("limit")
	pageStr := c.Request().URL.Query().Get("page")

	limit := 10 // default
	page := 1   // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	filter := &organization.ListTeamsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		OrganizationID: orgID,
	}

	teams, err := h.orgService.ListTeams(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, teams)
}

// CreateTeam handles team creation requests
func (h *OrganizationHandler) CreateTeam(c forge.Context) error {
	orgIDStr := c.Param("id")
	if orgIDStr == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	var req CreateTeamRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	team, err := h.orgService.CreateTeam(c.Request().Context(), orgID, &req, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, team)
}

// UpdateTeam handles team update requests
func (h *OrganizationHandler) UpdateTeam(c forge.Context) error {
	teamIDStr := c.Param("teamId")
	if teamIDStr == "" {
		return c.JSON(400, map[string]string{"error": "team ID is required"})
	}
	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team ID"})
	}

	var req UpdateTeamRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	team, err := h.orgService.UpdateTeam(c.Request().Context(), teamID, &req, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, team)
}

// DeleteTeam handles team deletion requests
func (h *OrganizationHandler) DeleteTeam(c forge.Context) error {
	teamIDStr := c.Param("teamId")
	if teamIDStr == "" {
		return c.JSON(400, map[string]string{"error": "team ID is required"})
	}
	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team ID"})
	}

	// TODO: Get user ID from session/context
	userID := xid.New() // Placeholder

	err = h.orgService.DeleteTeam(c.Request().Context(), teamID, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}
