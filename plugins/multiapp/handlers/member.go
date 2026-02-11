package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/xid"
	app "github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// MemberHandler handles app member-related HTTP requests.
type MemberHandler struct {
	appService *app.ServiceImpl
}

// NewMemberHandler creates a new member handler.
func NewMemberHandler(appService *app.ServiceImpl) *MemberHandler {
	return &MemberHandler{
		appService: appService,
	}
}

// AddMember handles adding a member to an organization.
func (h *MemberHandler) AddMember(c forge.Context) error {
	var req AddMemberRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	appID, err := xid.FromString(req.AppID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	userID, err := xid.FromString(req.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest))
	}

	member, err := h.appService.CreateMember(c.Request().Context(), &app.Member{
		AppID:     appID,
		UserID:    userID,
		Role:      app.MemberRole(req.Role),
		Status:    app.MemberStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusCreated, member)
}

// RemoveMember handles removing a member from an organization.
func (h *MemberHandler) RemoveMember(c forge.Context) error {
	memberIDStr := c.Param("memberId")

	if memberIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_MEMBER_ID", "Member ID parameter is required", http.StatusBadRequest))
	}

	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_MEMBER_ID", "Invalid member ID format", http.StatusBadRequest))
	}

	err = h.appService.DeleteMember(c.Request().Context(), memberID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusNoContent, nil)
}

// ListMembers handles listing app members.
func (h *MemberHandler) ListMembers(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
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

	filter := &app.ListMembersFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  (offset / limit) + 1,
			Limit: limit,
		},
		AppID: appID,
	}

	response, err := h.appService.ListMembers(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":       response.Data,
		"pagination": response.Pagination,
	})
}

// UpdateMemberRole handles updating a member's role in an organization.
func (h *MemberHandler) UpdateMemberRole(c forge.Context) error {
	memberIDStr := c.Param("memberId")

	if memberIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_MEMBER_ID", "Member ID parameter is required", http.StatusBadRequest))
	}

	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_MEMBER_ID", "Invalid member ID format", http.StatusBadRequest))
	}

	var req app.UpdateMemberRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Fetch existing member
	member, err := h.appService.FindMemberByID(c.Request().Context(), memberID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.New("MEMBER_NOT_FOUND", "Member not found", http.StatusNotFound))
	}

	// Update fields from request
	if req.Role != nil {
		member.Role = app.MemberRole(*req.Role)
	}

	member.UpdatedAt = time.Now()

	// Save changes
	if err := h.appService.UpdateMember(c.Request().Context(), member); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, member)
}

// InviteMember handles inviting a member to an organization.
func (h *MemberHandler) InviteMember(c forge.Context) error {
	appIDStr := c.Param("orgId")
	if appIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	var req app.InviteMemberRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get inviter user ID from context (this would typically come from auth middleware)
	inviterUserIDStr := c.Request().Header.Get("X-User-Id") // placeholder
	if inviterUserIDStr == "" {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Unauthorized access", http.StatusUnauthorized))
	}

	inviterUserID, err := xid.FromString(inviterUserIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_INVITER_ID", "Invalid inviter user ID format", http.StatusBadRequest))
	}

	if inviterUserID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Unauthorized access", http.StatusUnauthorized))
	}

	// Create invitation
	invitation := &app.Invitation{
		ID:        xid.New(),
		AppID:     appID,
		Email:     req.Email,
		Role:      app.MemberRole(req.Role),
		InviterID: inviterUserID,
		Status:    "pending",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.appService.CreateInvitation(c.Request().Context(), invitation); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusCreated, invitation)
}

// UpdateMember handles updating a member in an organization.
func (h *MemberHandler) UpdateMember(c forge.Context) error {
	memberIDStr := c.Param("memberId")
	if memberIDStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_MEMBER_ID", "Member ID parameter is required", http.StatusBadRequest))
	}

	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_MEMBER_ID", "Invalid member ID format", http.StatusBadRequest))
	}

	var req app.UpdateMemberRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Fetch existing member
	member, err := h.appService.FindMemberByID(c.Request().Context(), memberID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.New("MEMBER_NOT_FOUND", "Member not found", http.StatusNotFound))
	}

	// Update fields from request
	if req.Role != nil {
		member.Role = app.MemberRole(*req.Role)
	}

	member.UpdatedAt = time.Now()

	// Save changes
	if err := h.appService.UpdateMember(c.Request().Context(), member); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, member)
}

// GetInvitation handles getting an invitation by token.
func (h *MemberHandler) GetInvitation(c forge.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TOKEN", "Invitation token parameter is required", http.StatusBadRequest))
	}

	invitation, err := h.appService.FindInvitationByToken(c.Request().Context(), token)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.New("INVITATION_NOT_FOUND", "Invitation not found", http.StatusNotFound))
	}

	return c.JSON(http.StatusOK, invitation)
}

// AcceptInvitation handles accepting an invitation.
func (h *MemberHandler) AcceptInvitation(c forge.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TOKEN", "Invitation token parameter is required", http.StatusBadRequest))
	}

	// Get user ID from context (this would typically come from auth middleware)
	userIDStr := c.Request().Header.Get("X-User-Id") // placeholder
	if userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Unauthorized access", http.StatusUnauthorized))
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest))
	}

	member, err := h.appService.AcceptInvitation(c.Request().Context(), token, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, member)
}

// DeclineInvitation handles declining an invitation.
func (h *MemberHandler) DeclineInvitation(c forge.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TOKEN", "Invitation token parameter is required", http.StatusBadRequest))
	}

	err := h.appService.DeclineInvitation(c.Request().Context(), token)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusNoContent, nil)
}
