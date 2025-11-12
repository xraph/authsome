package impersonation

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/internal/interfaces"
	"github.com/xraph/forge"
)

// Handler handles impersonation HTTP requests
// Updated for V2 architecture: App → Environment → Organization
type Handler struct {
	service *impersonation.Service
	config  Config
}

// NewHandler creates a new impersonation handler
func NewHandler(service *impersonation.Service, config Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// StartImpersonation handles POST /impersonation/start
func (h *Handler) StartImpersonation(c forge.Context) error {
	// Extract V2 context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	userID := interfaces.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, map[string]string{
			"error": "App context required",
		})
	}

	var reqBody struct {
		TargetUserID    string `json:"target_user_id"`
		Reason          string `json:"reason"`
		TicketNumber    string `json:"ticket_number,omitempty"`
		DurationMinutes int    `json:"duration_minutes,omitempty"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Parse target user ID
	targetUserID, err := xid.FromString(reqBody.TargetUserID)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid target user ID",
		})
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &impersonation.StartRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		ImpersonatorID:     userID,
		TargetUserID:       targetUserID,
		Reason:             reqBody.Reason,
		TicketNumber:       reqBody.TicketNumber,
		DurationMinutes:    reqBody.DurationMinutes,
		IPAddress:          c.Request().RemoteAddr,
		UserAgent:          c.Request().Header.Get("User-Agent"),
	}

	// Start impersonation
	resp, err := h.service.Start(c.Request().Context(), req)
	if err != nil {
		statusCode := 500
		if err == impersonation.ErrPermissionDenied {
			statusCode = 403
		} else if err == impersonation.ErrCannotImpersonateSelf {
			statusCode = 400
		} else if err == impersonation.ErrAlreadyImpersonating {
			statusCode = 409
		} else if err == impersonation.ErrInvalidReason || err == impersonation.ErrInvalidDuration {
			statusCode = 400
		}

		return c.JSON(statusCode, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, resp)
}

// EndImpersonation handles POST /impersonation/end
func (h *Handler) EndImpersonation(c forge.Context) error {
	// Extract V2 context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	userID := interfaces.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, map[string]string{
			"error": "App context required",
		})
	}

	var reqBody struct {
		ImpersonationID string `json:"impersonation_id"`
		Reason          string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Parse impersonation ID
	impersonationID, err := xid.FromString(reqBody.ImpersonationID)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid impersonation ID",
		})
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &impersonation.EndRequest{
		ImpersonationID:    impersonationID,
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		ImpersonatorID:     userID,
		Reason:             reqBody.Reason,
	}

	resp, err := h.service.End(c.Request().Context(), req)
	if err != nil {
		statusCode := 500
		if err == impersonation.ErrPermissionDenied {
			statusCode = 403
		} else if err == impersonation.ErrImpersonationNotFound {
			statusCode = 404
		}

		return c.JSON(statusCode, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, resp)
}

// GetImpersonation handles GET /impersonation/:id
func (h *Handler) GetImpersonation(c forge.Context) error {
	// Extract V2 context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, map[string]string{
			"error": "App context required",
		})
	}

	idParam := c.Param("id")
	if idParam == "" {
		return c.JSON(400, map[string]string{
			"error": "Impersonation ID is required",
		})
	}

	id, err := xid.FromString(idParam)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid impersonation ID",
		})
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &impersonation.GetRequest{
		ImpersonationID:    id,
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
	}

	session, err := h.service.Get(c.Request().Context(), req)
	if err != nil {
		statusCode := 500
		if err == impersonation.ErrImpersonationNotFound {
			statusCode = 404
		}

		return c.JSON(statusCode, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, session)
}

// ListImpersonations handles GET /impersonation
func (h *Handler) ListImpersonations(c forge.Context) error {
	// Extract V2 context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, map[string]string{
			"error": "App context required",
		})
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.Request().URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(c.Request().URL.Query().Get("offset"))

	activeOnly := c.Request().URL.Query().Get("active_only") == "true"

	// Optional filters from query params
	var impersonatorID *xid.ID
	if impersonatorIDStr := c.Request().URL.Query().Get("impersonator_id"); impersonatorIDStr != "" {
		id, err := xid.FromString(impersonatorIDStr)
		if err == nil {
			impersonatorID = &id
		}
	}

	var targetUserID *xid.ID
	if targetUserIDStr := c.Request().URL.Query().Get("target_user_id"); targetUserIDStr != "" {
		id, err := xid.FromString(targetUserIDStr)
		if err == nil {
			targetUserID = &id
		}
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &impersonation.ListRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		ImpersonatorID:     impersonatorID,
		TargetUserID:       targetUserID,
		ActiveOnly:         activeOnly,
		Limit:              limit,
		Offset:             offset,
	}

	resp, err := h.service.List(c.Request().Context(), req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, resp)
}

// ListAuditEvents handles GET /impersonation/audit
func (h *Handler) ListAuditEvents(c forge.Context) error {
	// Extract V2 context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, map[string]string{
			"error": "App context required",
		})
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.Request().URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(c.Request().URL.Query().Get("offset"))

	// Optional filters
	var impersonationID *xid.ID
	if impersonationIDStr := c.Request().URL.Query().Get("impersonation_id"); impersonationIDStr != "" {
		id, err := xid.FromString(impersonationIDStr)
		if err == nil {
			impersonationID = &id
		}
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &impersonation.AuditListRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		ImpersonationID:    impersonationID,
		EventType:          c.Request().URL.Query().Get("event_type"),
		Limit:              limit,
		Offset:             offset,
	}

	events, total, err := h.service.ListAuditEvents(c.Request().Context(), req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// VerifyImpersonation handles GET /impersonation/verify/:sessionId
func (h *Handler) VerifyImpersonation(c forge.Context) error {
	sessionIDParam := c.Param("sessionId")
	if sessionIDParam == "" {
		return c.JSON(400, map[string]string{
			"error": "Session ID is required",
		})
	}

	sessionID, err := xid.FromString(sessionIDParam)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid session ID",
		})
	}

	req := &impersonation.VerifyRequest{
		SessionID: sessionID,
	}

	resp, err := h.service.Verify(c.Request().Context(), req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, resp)
}
