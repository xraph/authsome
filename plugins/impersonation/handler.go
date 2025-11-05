package impersonation

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/forge"
)

// Handler handles impersonation HTTP requests
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
	var req impersonation.StartRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Extract IP and User Agent from request
	if req.IPAddress == "" {
		req.IPAddress = c.Request().RemoteAddr
	}
	if req.UserAgent == "" {
		req.UserAgent = c.Request().Header.Get("User-Agent")
	}

	// Start impersonation
	resp, err := h.service.Start(c.Request().Context(), &req)
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
	var req impersonation.EndRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	resp, err := h.service.End(c.Request().Context(), &req)
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

	orgIDParam := c.Request().URL.Query().Get("org_id")
	if orgIDParam == "" {
		return c.JSON(400, map[string]string{
			"error": "Organization ID is required",
		})
	}

	orgID, err := xid.FromString(orgIDParam)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid organization ID",
		})
	}

	req := &impersonation.GetRequest{
		ImpersonationID: id,
		OrganizationID:  orgID,
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
	orgIDParam := c.Request().URL.Query().Get("org_id")
	if orgIDParam == "" {
		return c.JSON(400, map[string]string{
			"error": "Organization ID is required",
		})
	}

	orgID, err := xid.FromString(orgIDParam)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid organization ID",
		})
	}

	req := &impersonation.ListRequest{
		OrganizationID: orgID,
	}

	// Parse optional filters
	if impersonatorIDParam := c.Request().URL.Query().Get("impersonator_id"); impersonatorIDParam != "" {
		if impersonatorID, err := xid.FromString(impersonatorIDParam); err == nil {
			req.ImpersonatorID = &impersonatorID
		}
	}

	if targetUserIDParam := c.Request().URL.Query().Get("target_user_id"); targetUserIDParam != "" {
		if targetUserID, err := xid.FromString(targetUserIDParam); err == nil {
			req.TargetUserID = &targetUserID
		}
	}

	if activeOnlyParam := c.Request().URL.Query().Get("active_only"); activeOnlyParam == "true" {
		req.ActiveOnly = true
	}

	// Parse pagination
	limitStr := c.Request().URL.Query().Get("limit")
	offsetStr := c.Request().URL.Query().Get("offset")

	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	req.Limit = limit

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	req.Offset = offset

	resp, err := h.service.List(c.Request().Context(), req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, resp)
}

// VerifyImpersonation handles POST /impersonation/verify
func (h *Handler) VerifyImpersonation(c forge.Context) error {
	var req impersonation.VerifyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	resp, err := h.service.Verify(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, resp)
}

// ListAuditEvents handles GET /impersonation/audit
func (h *Handler) ListAuditEvents(c forge.Context) error {
	orgIDParam := c.Request().URL.Query().Get("org_id")
	if orgIDParam == "" {
		return c.JSON(400, map[string]string{
			"error": "Organization ID is required",
		})
	}

	orgID, err := xid.FromString(orgIDParam)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid organization ID",
		})
	}

	req := &impersonation.AuditListRequest{
		OrganizationID: orgID,
	}

	// Parse optional filters
	if impersonationIDParam := c.Request().URL.Query().Get("impersonation_id"); impersonationIDParam != "" {
		if impersonationID, err := xid.FromString(impersonationIDParam); err == nil {
			req.ImpersonationID = &impersonationID
		}
	}

	if eventTypeParam := c.Request().URL.Query().Get("event_type"); eventTypeParam != "" {
		req.EventType = eventTypeParam
	}

	// Parse pagination
	limitStr := c.Request().URL.Query().Get("limit")
	offsetStr := c.Request().URL.Query().Get("offset")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	req.Limit = limit

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	req.Offset = offset

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
