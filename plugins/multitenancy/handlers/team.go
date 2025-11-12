package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/multitenancy/app"
	"github.com/xraph/forge"
)

// TeamHandler handles team-related HTTP requests
type TeamHandler struct {
	appService *app.Service
}

// NewTeamHandler creates a new team handler
func NewTeamHandler(appService *app.Service) *TeamHandler {
	return &TeamHandler{
		appService: appService,
	}
}

// CreateTeam handles team creation requests
func (h *TeamHandler) CreateTeam(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(400, map[string]string{"error": "app ID is required"})
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid app ID"})
	}

	var req app.CreateTeamRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	team, err := h.appService.CreateTeam(c.Request().Context(), appID, &req)
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

	team, err := h.appService.GetTeam(c.Request().Context(), teamID)
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

	var req app.UpdateTeamRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	teamID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid team ID"})
	}

	team, err := h.appService.UpdateTeam(c.Request().Context(), teamID, &req)
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

	err = h.appService.DeleteTeam(c.Request().Context(), teamID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// ListTeams handles team listing requests
func (h *TeamHandler) ListTeams(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(400, map[string]string{"error": "app ID is required"})
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

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid app ID"})
	}

	teams, err := h.appService.ListTeams(c.Request().Context(), appID, limit, offset)
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

	err = h.appService.AddTeamMember(c.Request().Context(), teamID, req.MemberID, req.Role)
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

	err = h.appService.RemoveTeamMember(c.Request().Context(), teamID, memberID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}
