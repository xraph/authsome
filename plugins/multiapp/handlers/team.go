package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// TeamHandler handles team-related HTTP requests.
type TeamHandler struct {
	appService *app.ServiceImpl
}

// NewTeamHandler creates a new team handler.
func NewTeamHandler(appService *app.ServiceImpl) *TeamHandler {
	return &TeamHandler{
		appService: appService,
	}
}

// CreateTeam handles team creation requests.
func (h *TeamHandler) CreateTeam(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	var req app.CreateTeamRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Create team
	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	team := &app.Team{
		ID:          xid.New(),
		AppID:       appID,
		Name:        req.Name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.appService.CreateTeam(c.Request().Context(), team); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusCreated, team)
}

// GetTeam handles team retrieval requests.
func (h *TeamHandler) GetTeam(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TEAM_ID", "Team ID parameter is required", http.StatusBadRequest))
	}

	teamID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_TEAM_ID", "Invalid team ID format", http.StatusBadRequest))
	}

	team, err := h.appService.FindTeamByID(c.Request().Context(), teamID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.New("TEAM_NOT_FOUND", "Team not found", http.StatusNotFound))
	}

	return c.JSON(http.StatusOK, team)
}

// UpdateTeam handles team update requests.
func (h *TeamHandler) UpdateTeam(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TEAM_ID", "Team ID parameter is required", http.StatusBadRequest))
	}

	var req app.UpdateTeamRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	teamID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_TEAM_ID", "Invalid team ID format", http.StatusBadRequest))
	}

	// Fetch existing team
	team, err := h.appService.FindTeamByID(c.Request().Context(), teamID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.New("TEAM_NOT_FOUND", "Team not found", http.StatusNotFound))
	}

	// Update fields from request
	if req.Name != nil {
		team.Name = *req.Name
	}

	if req.Description != nil {
		team.Description = *req.Description
	}

	team.UpdatedAt = time.Now()

	// Save changes
	if err := h.appService.UpdateTeam(c.Request().Context(), team); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, team)
}

// DeleteTeam handles team deletion requests.
func (h *TeamHandler) DeleteTeam(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TEAM_ID", "Team ID parameter is required", http.StatusBadRequest))
	}

	teamID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_TEAM_ID", "Invalid team ID format", http.StatusBadRequest))
	}

	err = h.appService.DeleteTeam(c.Request().Context(), teamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusNoContent, nil)
}

// ListTeams handles team listing requests.
func (h *TeamHandler) ListTeams(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
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
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	filter := &app.ListTeamsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  (offset / limit) + 1,
			Limit: limit,
		},
		AppID: appID,
	}

	response, err := h.appService.ListTeams(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":       response.Data,
		"pagination": response.Pagination,
	})
}

// AddTeamMember handles adding a member to a team.
func (h *TeamHandler) AddTeamMember(c forge.Context) error {
	teamIDStr := c.Param("teamId")
	if teamIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TEAM_ID", "Team ID parameter is required", http.StatusBadRequest))
	}

	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_TEAM_ID", "Invalid team ID format", http.StatusBadRequest))
	}

	var req struct {
		MemberID xid.ID `json:"member_id"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Create TeamMember
	teamMember := &app.TeamMember{
		ID:        xid.New(),
		TeamID:    teamID,
		MemberID:  req.MemberID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	addedMember, err := h.appService.AddTeamMember(c.Request().Context(), teamMember)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusCreated, addedMember)
}

// RemoveTeamMember handles removing a member from a team.
func (h *TeamHandler) RemoveTeamMember(c forge.Context) error {
	teamIDStr := c.Param("teamId")
	if teamIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TEAM_ID", "Team ID parameter is required", http.StatusBadRequest))
	}

	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_TEAM_ID", "Invalid team ID format", http.StatusBadRequest))
	}

	memberIDStr := c.Param("memberId")
	if memberIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_MEMBER_ID", "Member ID parameter is required", http.StatusBadRequest))
	}

	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_MEMBER_ID", "Invalid member ID format", http.StatusBadRequest))
	}

	if teamID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TEAM_ID", "Team ID parameter is required", http.StatusBadRequest))
	}

	if memberID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_MEMBER_ID", "Member ID parameter is required", http.StatusBadRequest))
	}

	err = h.appService.RemoveTeamMember(c.Request().Context(), teamID, memberID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusNoContent, nil)
}
