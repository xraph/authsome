package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/multitenancy/app"
	"github.com/xraph/forge"
)

// AppHandler handles app-related HTTP requests
type AppHandler struct {
	appService *app.Service
}

// NewAppHandler creates a new app handler
func NewAppHandler(appService *app.Service) *AppHandler {
	return &AppHandler{
		appService: appService,
	}
}

// CreateApp handles app creation requests
func (h *AppHandler) CreateApp(c forge.Context) error {
	var req app.CreateAppRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// TODO: Get creator user ID from context/session
	// creatorUserID := "system" // placeholder

	a, err := h.appService.CreateApp(c.Request().Context(), &req, xid.New())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, a)
}

// GetApp handles get app requests
func (h *AppHandler) GetApp(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, map[string]string{"error": "app ID is required"})
	}

	appID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid app ID"})
	}

	a, err := h.appService.GetApp(c.Request().Context(), appID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	return c.JSON(200, a)
}

// UpdateApp handles app update requests
func (h *AppHandler) UpdateApp(c forge.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(400, map[string]string{"error": "app ID is required"})
	}

	appID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid app ID"})
	}

	var req app.UpdateAppRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	a, err := h.appService.UpdateApp(c.Request().Context(), appID, &req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, a)
}

// DeleteApp handles app deletion requests
func (h *AppHandler) DeleteApp(c forge.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(400, map[string]string{"error": "app ID is required"})
	}

	appID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid app ID"})
	}

	err = h.appService.DeleteApp(c.Request().Context(), appID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(204, nil)
}

// ListApps handles list apps requests
func (h *AppHandler) ListApps(c forge.Context) error {
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

	apps, err := h.appService.ListApps(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"apps":   apps,
		"limit":  limit,
		"offset": offset,
	})
}
