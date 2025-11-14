package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/xid"
	coreapp "github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// AppHandler handles app-related HTTP requests
type AppHandler struct {
	appService *coreapp.ServiceImpl
}

// NewAppHandler creates a new app handler
func NewAppHandler(appService *coreapp.ServiceImpl) *AppHandler {
	return &AppHandler{
		appService: appService,
	}
}

// CreateApp handles app creation requests
func (h *AppHandler) CreateApp(c forge.Context) error {
	var req coreapp.CreateAppRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// TODO: Get creator user ID from context/session
	// creatorUserID := "system" // placeholder

	a, err := h.appService.CreateApp(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusCreated, a)
}

// GetApp handles get app requests
func (h *AppHandler) GetApp(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	a, err := h.appService.FindAppByID(c.Request().Context(), appID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.New("APP_NOT_FOUND", "App not found", http.StatusNotFound))
	}

	return c.JSON(http.StatusOK, a)
}

// UpdateApp handles app update requests
func (h *AppHandler) UpdateApp(c forge.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	var req coreapp.UpdateAppRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	a, err := h.appService.UpdateApp(c.Request().Context(), appID, &coreapp.UpdateAppRequest{
		Name:     req.Name,
		Logo:     req.Logo,
		Metadata: req.Metadata,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, a)
}

// DeleteApp handles app deletion requests
func (h *AppHandler) DeleteApp(c forge.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID parameter is required", http.StatusBadRequest))
	}

	appID, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	err = h.appService.DeleteApp(c.Request().Context(), appID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusNoContent, nil)
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

	filter := &coreapp.ListAppsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  (offset / limit) + 1,
			Limit: limit,
		},
	}

	response, err := h.appService.ListApps(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":       response.Data,
		"pagination": response.Pagination,
	})
}
