package handlers

import (
	"net/http"

	"github.com/rs/xid"
	coreapp "github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// AppHandler handles app-related HTTP requests.
type AppHandler struct {
	appService *coreapp.ServiceImpl
}

// NewAppHandler creates a new app handler.
func NewAppHandler(appService *coreapp.ServiceImpl) *AppHandler {
	return &AppHandler{
		appService: appService,
	}
}

// CreateApp handles app creation requests.
func (h *AppHandler) CreateApp(c forge.Context) error {
	var req CreateAppRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// TODO: Get creator user ID from context/session
	// creatorUserID := "system" // placeholder

	a, err := h.appService.CreateApp(c.Request().Context(), &req.CreateAppRequest)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusCreated, a)
}

// GetApp handles get app requests.
func (h *AppHandler) GetApp(c forge.Context) error {
	var req GetAppRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	appID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	a, err := h.appService.FindAppByID(c.Request().Context(), appID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.New("APP_NOT_FOUND", "App not found", http.StatusNotFound))
	}

	return c.JSON(http.StatusOK, a)
}

// UpdateApp handles app update requests.
func (h *AppHandler) UpdateApp(c forge.Context) error {
	var req UpdateAppRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	appID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	a, err := h.appService.UpdateApp(c.Request().Context(), appID, &req.UpdateAppRequest)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, a)
}

// DeleteApp handles app deletion requests.
func (h *AppHandler) DeleteApp(c forge.Context) error {
	var req DeleteAppRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	appID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_APP_ID", "Invalid app ID format", http.StatusBadRequest))
	}

	err = h.appService.DeleteApp(c.Request().Context(), appID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusNoContent, nil)
}

// ListApps handles list apps requests.
func (h *AppHandler) ListApps(c forge.Context) error {
	var req ListAppsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Set defaults
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	offset := max(req.Offset, 0)

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

	return c.JSON(http.StatusOK, map[string]any{
		"data":       response.Data,
		"pagination": response.Pagination,
	})
}
