package impersonation

import (
	"errors"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles impersonation HTTP requests
// Handler for V2 architecture: App → Environment → Organization.
type Handler struct {
	service *impersonation.Service
	config  Config
}

// StartImpersonationRequest represents request types.
type StartImpersonationRequest struct {
	TargetUserID    string `json:"target_user_id"   validate:"required"`
	Reason          string `json:"reason"           validate:"required"`
	TicketNumber    string `json:"ticket_number"`
	DurationMinutes int    `json:"duration_minutes"`
}

type EndImpersonationRequest struct {
	ImpersonationID string `json:"impersonation_id" validate:"required"`
	Reason          string `json:"reason"`
}

type GetImpersonationRequest struct {
	ID string `path:"id" validate:"required"`
}

type ListImpersonationsRequest struct {
	Page           int    `query:"page"`
	Limit          int    `query:"limit"`
	Offset         int    `query:"offset"`
	ActiveOnly     *bool  `query:"active_only"`
	ImpersonatorID string `query:"impersonator_id"`
	TargetUserID   string `query:"target_user_id"`
}

type ListAuditEventsRequest struct {
	Page            int    `query:"page"`
	Limit           int    `query:"limit"`
	Offset          int    `query:"offset"`
	ImpersonationID string `query:"impersonation_id"`
	EventType       string `query:"event_type"`
}

type VerifyImpersonationRequest struct {
	SessionID string `path:"sessionId" validate:"required"`
}

// ErrorResponse types - use shared responses from core.
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

// NewHandler creates a new impersonation handler.
func NewHandler(service *impersonation.Service, config Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// StartImpersonation handles POST /impersonation/start.
func (h *Handler) StartImpersonation(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	var reqBody StartImpersonationRequest
	if err := c.BindRequest(&reqBody); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Parse target user ID
	targetUserID, err := xid.FromString(reqBody.TargetUserID)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_USER_ID", "Invalid target user ID", 400))
	}

	// envIDPtr service request with V2 context
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
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		// Fallback for unexpected errors
		return c.JSON(500, errs.New("IMPERSONATION_START_FAILED", err.Error(), 500))
	}

	return c.JSON(200, resp)
}

// EndImpersonation handles POST /impersonation/end.
func (h *Handler) EndImpersonation(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	var reqBody EndImpersonationRequest
	if err := c.BindRequest(&reqBody); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Parse impersonation ID
	impersonationID, err := xid.FromString(reqBody.ImpersonationID)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_IMPERSONATION_ID", "Invalid impersonation ID", 400))
	}

	// envIDPtr service request with V2 context
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
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		// Fallback for unexpected errors
		return c.JSON(500, errs.New("IMPERSONATION_END_FAILED", err.Error(), 500))
	}

	return c.JSON(200, resp)
}

// GetImpersonation handles GET /impersonation/:id.
func (h *Handler) GetImpersonation(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	var reqParams GetImpersonationRequest
	if err := c.BindRequest(&reqParams); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	id, err := xid.FromString(reqParams.ID)
	if err != nil {
		return c.JSON(400, errs.New("INVALID_IMPERSONATION_ID", "Invalid impersonation ID", 400))
	}

	// envIDPtr service request with V2 context
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
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		// Fallback for unexpected errors
		return c.JSON(500, errs.New("GET_IMPERSONATION_FAILED", err.Error(), 500))
	}

	return c.JSON(200, session)
}

// ListImpersonations handles GET /impersonation.
func (h *Handler) ListImpersonations(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	var reqParams ListImpersonationsRequest
	if err := c.BindRequest(&reqParams); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Set defaults
	limit := reqParams.Limit
	if limit == 0 {
		limit = 50
	}

	// impersonatorID filters from query params
	var impersonatorID *xid.ID

	if reqParams.ImpersonatorID != "" {
		id, err := xid.FromString(reqParams.ImpersonatorID)
		if err == nil {
			impersonatorID = &id
		}
	}

	var targetUserID *xid.ID

	if reqParams.TargetUserID != "" {
		id, err := xid.FromString(reqParams.TargetUserID)
		if err == nil {
			targetUserID = &id
		}
	}

	// envIDPtr service filter with V2 context
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
			Offset: reqParams.Offset,
		},
		AppID:          appID,
		EnvironmentID:  envIDPtr,
		OrganizationID: orgIDPtr,
		ImpersonatorID: impersonatorID,
		TargetUserID:   targetUserID,
		ActiveOnly:     reqParams.ActiveOnly,
	}

	resp, err := h.service.List(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(500, errs.New("LIST_IMPERSONATIONS_FAILED", err.Error(), 500))
	}

	return c.JSON(200, resp)
}

// ListAuditEvents handles GET /impersonation/audit.
func (h *Handler) ListAuditEvents(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, errs.New("APP_CONTEXT_REQUIRED", "App context required", 400))
	}

	var reqParams ListAuditEventsRequest
	if err := c.BindRequest(&reqParams); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Set defaults
	limit := reqParams.Limit
	if limit == 0 {
		limit = 50
	}

	// impersonationID filters
	var impersonationID *xid.ID

	if reqParams.ImpersonationID != "" {
		id, err := xid.FromString(reqParams.ImpersonationID)
		if err == nil {
			impersonationID = &id
		}
	}

	var eventTypePtr *string
	if reqParams.EventType != "" {
		eventTypePtr = &reqParams.EventType
	}

	// envIDPtr service filter with V2 context
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
			Offset: reqParams.Offset,
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

// VerifyImpersonation handles GET /impersonation/verify/:sessionId.
func (h *Handler) VerifyImpersonation(c forge.Context) error {
	var reqParams VerifyImpersonationRequest
	if err := c.BindRequest(&reqParams); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	sessionID, err := xid.FromString(reqParams.SessionID)
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
