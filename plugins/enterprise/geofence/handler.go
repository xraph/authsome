package geofence

import (
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for geofencing
type Handler struct {
	service *Service
	config  *Config
}

// Response types - use shared responses from core
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

type RulesResponse struct {
	Rules interface{} `json:"rules"`
	Count int         `json:"count"`
}

// NewHandler creates a new geofencing handler
func NewHandler(service *Service, config *Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// CreateRule creates a new geofence rule
func (h *Handler) CreateRule(c forge.Context) error {
	var req GeofenceRule
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	// Get app ID from context (set by auth middleware)
	appID := c.Get("app_id").(xid.ID)
	req.AppID = appID

	// Get user ID from context (creator)
	userID := c.Get("user_id").(xid.ID)
	req.CreatedBy = userID

	if err := h.service.repo.CreateRule(c.Context(), &req); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("failed to create rule"))
	}

	return c.JSON(http.StatusCreated, req)
}

// ListRules lists all geofence rules for an organization
func (h *Handler) ListRules(c forge.Context) error {
	orgID := c.Get("organization_id").(xid.ID)

	rules, err := h.service.repo.GetRulesByApp(c.Context(), orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, rules)
}

// GetRule gets a specific geofence rule
func (h *Handler) GetRule(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid rule ID"))
	}

	rule, err := h.service.repo.GetRule(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("rule not found"))
	}

	return c.JSON(http.StatusOK, rule)
}

// UpdateRule updates a geofence rule
func (h *Handler) UpdateRule(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid rule ID"))
	}

	var req GeofenceRule
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	req.ID = id

	// Get user ID from context (updater)
	userID := c.Get("user_id").(xid.ID)
	req.UpdatedBy = &userID

	if err := h.service.repo.UpdateRule(c.Context(), &req); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, req)
}

// DeleteRule deletes a geofence rule
func (h *Handler) DeleteRule(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid rule ID"))
	}

	if err := h.service.repo.DeleteRule(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "rule deleted successfully"})
}

// CheckLocation performs a geofence check
func (h *Handler) CheckLocation(c forge.Context) error {
	var req struct {
		IPAddress string   `json:"ipAddress"`
		UserID    string   `json:"userId,omitempty"`
		EventType string   `json:"eventType"`
		GPS       *GPSData `json:"gps,omitempty"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	var userID xid.ID
	if req.UserID != "" {
		id, err := xid.FromString(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid user ID"))
		}
		userID = id
	} else {
		userID = c.Get("user_id").(xid.ID)
	}

	// Get app ID from context
	appID, ok := c.Get("app_id").(xid.ID)
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	checkReq := &LocationCheckRequest{
		UserID:    userID,
		AppID:     appID,
		IPAddress: req.IPAddress,
		EventType: req.EventType,
		GPS:       req.GPS,
	}

	result, err := h.service.CheckLocation(c.Context(), checkReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, result)
}

// LookupIP performs IP geolocation lookup
func (h *Handler) LookupIP(c forge.Context) error {
	ip := c.Param("ip")

	geoData, err := h.service.GetGeolocation(c.Context(), ip)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	detection, _ := h.service.GetDetection(c.Context(), ip)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"geolocation": geoData,
		"detection":   detection,
	})
}

// ListLocationEvents lists location events for the authenticated user
func (h *Handler) ListLocationEvents(c forge.Context) error {
	userID := c.Get("user_id").(xid.ID)

	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	events, err := h.service.repo.GetUserLocationHistory(c.Context(), userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, events)
}

// GetLocationEvent gets a specific location event
func (h *Handler) GetLocationEvent(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid event ID"))
	}

	event, err := h.service.repo.GetLocationEvent(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("event not found"))
	}

	return c.JSON(http.StatusOK, event)
}

// ListTravelAlerts lists travel alerts for the authenticated user
func (h *Handler) ListTravelAlerts(c forge.Context) error {
	userID := c.Get("user_id").(xid.ID)
	status := c.Query("status") // Optional filter

	alerts, err := h.service.repo.GetUserTravelAlerts(c.Context(), userID, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, alerts)
}

// GetTravelAlert gets a specific travel alert
func (h *Handler) GetTravelAlert(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid alert ID"))
	}

	alert, err := h.service.repo.GetTravelAlert(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("alert not found"))
	}

	return c.JSON(http.StatusOK, alert)
}

// ApproveTravelAlert approves a travel alert
func (h *Handler) ApproveTravelAlert(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid alert ID"))
	}

	userID := c.Get("user_id").(xid.ID)

	if err := h.service.repo.ApproveTravel(c.Context(), id, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "travel alert approved"})
}

// DenyTravelAlert denies a travel alert
func (h *Handler) DenyTravelAlert(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid alert ID"))
	}

	userID := c.Get("user_id").(xid.ID)

	if err := h.service.repo.DenyTravel(c.Context(), id, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "travel alert denied"})
}

// CreateTrustedLocation creates a trusted location
func (h *Handler) CreateTrustedLocation(c forge.Context) error {
	var req TrustedLocation
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	appID := c.Get("app_id").(xid.ID)
	userID := c.Get("user_id").(xid.ID)

	req.AppID = appID
	req.UserID = userID

	if err := h.service.repo.CreateTrustedLocation(c.Context(), &req); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusCreated, req)
}

// ListTrustedLocations lists trusted locations for the authenticated user
func (h *Handler) ListTrustedLocations(c forge.Context) error {
	userID := c.Get("user_id").(xid.ID)

	locations, err := h.service.repo.GetUserTrustedLocations(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, locations)
}

// GetTrustedLocation gets a specific trusted location
func (h *Handler) GetTrustedLocation(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid location ID"))
	}

	location, err := h.service.repo.GetTrustedLocation(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("location not found"))
	}

	return c.JSON(http.StatusOK, location)
}

// UpdateTrustedLocation updates a trusted location
func (h *Handler) UpdateTrustedLocation(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid location ID"))
	}

	var req TrustedLocation
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	req.ID = id

	if err := h.service.repo.UpdateTrustedLocation(c.Context(), &req); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, req)
}

// DeleteTrustedLocation deletes a trusted location
func (h *Handler) DeleteTrustedLocation(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid location ID"))
	}

	if err := h.service.repo.DeleteTrustedLocation(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "trusted location deleted"})
}

// ListViolations lists geofence violations
func (h *Handler) ListViolations(c forge.Context) error {
	userID := c.Get("user_id").(xid.ID)

	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	violations, err := h.service.repo.GetUserViolations(c.Context(), userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, violations)
}

// GetViolation gets a specific violation
func (h *Handler) GetViolation(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid violation ID"))
	}

	violation, err := h.service.repo.GetViolation(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("violation not found"))
	}

	return c.JSON(http.StatusOK, violation)
}

// ResolveViolation resolves a geofence violation
func (h *Handler) ResolveViolation(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid violation ID"))
	}

	var req struct {
		Resolution string `json:"resolution"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	userID := c.Get("user_id").(xid.ID)

	if err := h.service.repo.ResolveViolation(c.Context(), id, userID, req.Resolution); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "violation resolved"})
}

// GetMetrics returns geofencing metrics
func (h *Handler) GetMetrics(c forge.Context) error {
	// TODO: Implement metrics aggregation
	return c.JSON(http.StatusOK, &MessageResponse{Message: "metrics endpoint - to be implemented"})
}

// GetLocationAnalytics returns location analytics
func (h *Handler) GetLocationAnalytics(c forge.Context) error {
	// TODO: Implement location analytics
	return c.JSON(http.StatusOK, &MessageResponse{Message: "location analytics endpoint - to be implemented"})
}

// GetViolationAnalytics returns violation analytics
func (h *Handler) GetViolationAnalytics(c forge.Context) error {
	// TODO: Implement violation analytics
	return c.JSON(http.StatusOK, &MessageResponse{Message: "violation analytics endpoint - to be implemented"})
}
