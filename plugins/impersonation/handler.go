package impersonation

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles impersonation HTTP requests
// Updated for V2 architecture: App → Environment → Organization
type Handler struct {
	service *impersonation.Service
	config  Config
}

// Response types - use shared responses from core
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

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
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	var reqBody struct {
		TargetUserID    string `json:"target_user_id"`
		Reason          string `json:"reason"`
		TicketNumber    string `json:"ticket_number,omitempty"`
		DurationMinutes int    `json:"duration_minutes,omitempty"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Parse target user ID
	targetUserID, err := xid.FromString(reqBody.TargetUserID)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_USER_ID", "Invalid target user ID", 400))
	}

	// Build service request with V2 context
	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &impersonation.StartRequest{
		AppID:           appID,
		EnvironmentID:   envIDPtr,
		OrganizationID:  orgIDPtr,
		ImpersonatorID:  userID,
		TargetUserID:    targetUserID,
		Reason:          reqBody.Reason,
		TicketNumber:    reqBody.TicketNumber,
		DurationMinutes: reqBody.DurationMinutes,
		IPAddress:       c.Request().RemoteAddr,
		UserAgent:       c.Request().Header.Get("User-Agent"),
	}

	// Start impersonation
	resp, err := h.service.Start(c.Request().Context(), req)
	if err != nil {
		// Handle structured errors
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		// Fallback for unexpected errors
		return c.JSON(500, errs.New("IMPERSONATION_START_FAILED", err.Error(), 500))
	}

	return c.JSON(200, resp)
}

// EndImpersonation handles POST /impersonation/end
func (h *Handler) EndImpersonation(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	var reqBody struct {
		ImpersonationID string `json:"impersonation_id"`
		Reason          string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(400, errs.New("INVALID_REQUEST", "Invalid request body", 400))
	}

	// Parse impersonation ID
	impersonationID, err := xid.FromString(reqBody.ImpersonationID)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_IMPERSONATION_ID", "Invalid impersonation ID", 400))
	}

	// Build service request with V2 context
	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &impersonation.EndRequest{
		ImpersonationID: impersonationID,
		AppID:           appID,
		EnvironmentID:   envIDPtr,
		OrganizationID:  orgIDPtr,
		ImpersonatorID:  userID,
		Reason:          reqBody.Reason,
	}

	resp, err := h.service.End(c.Request().Context(), req)
	if err != nil {
		// Handle structured errors
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		// Fallback for unexpected errors
		return c.JSON(500, errs.New("IMPERSONATION_END_FAILED", err.Error(), 500))
	}

	return c.JSON(200, resp)
}

// GetImpersonation handles GET /impersonation/:id
func (h *Handler) GetImpersonation(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	idParam := c.Param("id")
	if idParam == "" {
		return c.JSON(400, errs.New("IMPERSONATION_ID_REQUIRED", "Impersonation ID is required", 400))
	}

	id, err := xid.FromString(idParam)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_IMPERSONATION_ID", "Invalid impersonation ID", 400))
	}

	// Build service request with V2 context
	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &impersonation.GetRequest{
		ImpersonationID: id,
		AppID:           appID,
		EnvironmentID:   envIDPtr,
		OrganizationID:  orgIDPtr,
	}

	session, err := h.service.Get(c.Request().Context(), req)
	if err != nil {
		// Handle structured errors
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		// Fallback for unexpected errors
		return c.JSON(500, errs.New("GET_IMPERSONATION_FAILED", err.Error(), 500))
	}

	return c.JSON(200, session)
}

// ListImpersonations handles GET /impersonation
func (h *Handler) ListImpersonations(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.Request().URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(c.Request().URL.Query().Get("offset"))

	activeOnlyStr := c.Request().URL.Query().Get("active_only")
	var activeOnly *bool
	if activeOnlyStr != "" {
		val := activeOnlyStr == "true"
		activeOnly = &val
	}

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

	// Build service filter with V2 context
	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	filter := &impersonation.ListSessionsFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
		AppID:          appID,
		EnvironmentID:  envIDPtr,
		OrganizationID: orgIDPtr,
		ImpersonatorID: impersonatorID,
		TargetUserID:   targetUserID,
		ActiveOnly:     activeOnly,
	}

	resp, err := h.service.List(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(500, errs.New("LIST_IMPERSONATIONS_FAILED", err.Error(), 500))
	}

	return c.JSON(200, resp)
}

// ListAuditEvents handles GET /impersonation/audit
func (h *Handler) ListAuditEvents(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
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

	eventType := c.Request().URL.Query().Get("event_type")
	var eventTypePtr *string
	if eventType != "" {
		eventTypePtr = &eventType
	}

	// Build service filter with V2 context
	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	filter := &impersonation.ListAuditEventsFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
		AppID:           appID,
		EnvironmentID:   envIDPtr,
		OrganizationID:  orgIDPtr,
		ImpersonationID: impersonationID,
		EventType:       eventTypePtr,
	}

	resp, err := h.service.ListAuditEvents(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(500, errs.New("LIST_AUDIT_EVENTS_FAILED", err.Error(), 500))
	}

	return c.JSON(200, resp)
}

// VerifyImpersonation handles GET /impersonation/verify/:sessionId
func (h *Handler) VerifyImpersonation(c forge.Context) error {
	sessionIDParam := c.Param("sessionId")
	if sessionIDParam == "" {
		return c.JSON(400, errs.New("SESSION_ID_REQUIRED", "Session ID is required", 400))
	}

	sessionID, err := xid.FromString(sessionIDParam)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_SESSION_ID", "Invalid session ID", 400))
	}

	req := &impersonation.VerifyRequest{
		SessionID: sessionID,
	}

	resp, err := h.service.Verify(c.Request().Context(), req)
	if err != nil {
		return c.JSON(500, errs.New("VERIFY_IMPERSONATION_FAILED", err.Error(), 500))
	}

	return c.JSON(200, resp)
}
