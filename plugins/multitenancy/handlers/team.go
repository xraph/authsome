package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/forge"
)

// TeamHandler handles team-related HTTP requests
type TeamHandler struct {
	orgService *organization.Service
}

// NewTeamHandler creates a new team handler
func NewTeamHandler(orgService *organization.Service) *TeamHandler {
	return &TeamHandler{
		orgService: orgService,
	}
}

// CreateTeam handles team creation requests
func (h *TeamHandler) CreateTeam(c forge.Context) error {
	orgIDStr := c.Param("orgId")
	if orgIDStr == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	var req organization.CreateTeamRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	team, err := h.orgService.CreateTeam(c.Request().Context(), orgID, &req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, team)
}

// GetTeam handles team retrieval requests
func (h *TeamHandler) GetTeam(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "team ID is required"})
	}

	teamID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team ID"})
	}

	team, err := h.orgService.GetTeam(c.Request().Context(), teamID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "team not found"})
	}

	return c.JSON(200, team)
}

// UpdateTeam handles team update requests
func (h *TeamHandler) UpdateTeam(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "team ID is required"})
	}

	var req organization.UpdateTeamRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	teamID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team ID"})
	}

	team, err := h.orgService.UpdateTeam(c.Request().Context(), teamID, &req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, team)
}

// DeleteTeam handles team deletion requests
func (h *TeamHandler) DeleteTeam(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "team ID is required"})
	}

	teamID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team ID"})
	}

	err = h.orgService.DeleteTeam(c.Request().Context(), teamID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// ListTeams handles team listing requests
func (h *TeamHandler) ListTeams(c forge.Context) error {
	orgIDStr := c.Param("orgId")
	if orgIDStr == "" {
		return c.JSON(400, map[string]string{"error": "organization ID is required"})
	}

	// Parse pagination parameters
	limitStr := c.Request().URL.Query().Get("limit")
	offsetStr := c.Request().URL.Query().Get("offset")

	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid organization ID"})
	}

	teams, err := h.orgService.ListTeams(c.Request().Context(), orgID, limit, offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"teams":  teams,
		"limit":  limit,
		"offset": offset,
	})
}

// AddTeamMember handles adding a member to a team
func (h *TeamHandler) AddTeamMember(c forge.Context) error {
	teamIDStr := c.Param("teamId")
	if teamIDStr == "" {
		return c.JSON(400, map[string]string{"error": "team ID is required"})
	}

	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team ID"})
	}

	var req struct {
		MemberID xid.ID `json:"member_id"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	err = h.orgService.AddTeamMember(c.Request().Context(), teamID, req.MemberID, req.Role)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// RemoveTeamMember handles removing a member from a team
func (h *TeamHandler) RemoveTeamMember(c forge.Context) error {
	teamIDStr := c.Param("teamId")
	if teamIDStr == "" {
		return c.JSON(400, map[string]string{"error": "team ID is required"})
	}

	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team ID"})
	}

	memberIDStr := c.Param("memberId")
	if memberIDStr == "" {
		return c.JSON(400, map[string]string{"error": "member ID is required"})
	}

	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid member ID"})
	}

	if teamID.IsNil() {
		return c.JSON(400, map[string]string{"error": "team ID is required"})
	}
	if memberID.IsNil() {
		return c.JSON(400, map[string]string{"error": "member ID is required"})
	}

	err = h.orgService.RemoveTeamMember(c.Request().Context(), teamID, memberID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}
