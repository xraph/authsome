package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/multitenancy/app"
	"github.com/xraph/forge"
)

// MemberHandler handles app member-related HTTP requests
type MemberHandler struct {
	appService *app.Service
}

// NewMemberHandler creates a new member handler
func NewMemberHandler(appService *app.Service) *MemberHandler {
	return &MemberHandler{
		appService: appService,
	}
}

// AddMember handles adding a member to an organization
func (h *MemberHandler) AddMember(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(400, map[string]string{"error": "app ID is required"})
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid app ID"})
	}

	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	userID, err := xid.FromString(req.UserID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid user ID"})
	}

	member, err := h.appService.AddMember(c.Request().Context(), appID, userID, req.Role)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, member)
}

// RemoveMember handles removing a member from an organization
func (h *MemberHandler) RemoveMember(c forge.Context) error {
	memberIDStr := c.Param("memberId")

	if memberIDStr == "" {
		return c.JSON(400, map[string]string{"error": "member ID is required"})
	}
	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid member ID"})
	}

	err = h.appService.RemoveMember(c.Request().Context(), memberID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// ListMembers handles listing app members
func (h *MemberHandler) ListMembers(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(400, map[string]string{"error": "app ID is required"})
	}
	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid app ID"})
	}

	// Parse pagination parameters
	limitStr := c.Request().URL.Query().Get("limit")
	offsetStr := c.Request().URL.Query().Get("offset")

	limit := 10 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	members, err := h.appService.ListMembers(c.Request().Context(), appID, limit, offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"members": members,
		"limit":   limit,
		"offset":  offset,
	})
}

// UpdateMemberRole handles updating a member's role in an organization
func (h *MemberHandler) UpdateMemberRole(c forge.Context) error {
	memberIDStr := c.Param("memberId")

	if memberIDStr == "" {
		return c.JSON(400, map[string]string{"error": "member ID is required"})
	}

	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid member ID"})
	}

	var req app.UpdateMemberRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	member, err := h.appService.UpdateMember(c.Request().Context(), memberID, &req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, member)
}

// InviteMember handles inviting a member to an organization
func (h *MemberHandler) InviteMember(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(400, map[string]string{"error": "app ID is required"})
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid app ID"})
	}

	var req app.InviteMemberRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get inviter user ID from context (this would typically come from auth middleware)
	inviterUserIDStr := c.Request().Header.Get("X-User-ID") // placeholder
	if inviterUserIDStr == "" {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}
	inviterUserID, err := xid.FromString(inviterUserIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid inviter user ID"})
	}
	if inviterUserID.IsNil() {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	invitation, err := h.appService.InviteMember(c.Request().Context(), appID, &req, inviterUserID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, invitation)
}

// UpdateMember handles updating a member in an organization
func (h *MemberHandler) UpdateMember(c forge.Context) error {
	memberIDStr := c.Param("memberId")
	if memberIDStr == "" {
		return c.JSON(400, map[string]string{"error": "member ID is required"})
	}

	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid member ID"})
	}

	var req app.UpdateMemberRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	member, err := h.appService.UpdateMember(c.Request().Context(), memberID, &req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, member)
}

// GetInvitation handles getting an invitation by token
func (h *MemberHandler) GetInvitation(c forge.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(400, map[string]string{"error": "invitation token is required"})
	}

	invitation, err := h.appService.GetInvitation(c.Request().Context(), token)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, invitation)
}

// AcceptInvitation handles accepting an invitation
func (h *MemberHandler) AcceptInvitation(c forge.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(400, map[string]string{"error": "invitation token is required"})
	}

	// Get user ID from context (this would typically come from auth middleware)
	userIDStr := c.Request().Header.Get("X-User-ID") // placeholder
	if userIDStr == "" {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid user ID"})
	}

	member, err := h.appService.AcceptInvitation(c.Request().Context(), token, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, member)
}

// DeclineInvitation handles declining an invitation
func (h *MemberHandler) DeclineInvitation(c forge.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(400, map[string]string{"error": "invitation token is required"})
	}

	err := h.appService.DeclineInvitation(c.Request().Context(), token)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}
