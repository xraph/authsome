package secrets

import (
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/secrets/core"
)

// Handler handles HTTP requests for the secrets API
type Handler struct {
	service *Service
	logger  forge.Logger
}

// NewHandler creates a new secrets handler
func NewHandler(service *Service, logger forge.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// =============================================================================
// Request/Response DTOs
// =============================================================================

// Request types
type ListSecretsRequest struct {
	Prefix    string   `query:"prefix"`
	ValueType string   `query:"valueType"`
	Search    string   `query:"search"`
	SortBy    string   `query:"sortBy"`
	SortOrder string   `query:"sortOrder"`
	Recursive bool     `query:"recursive"`
	Page      int      `query:"page"`
	PageSize  int      `query:"pageSize"`
	Tags      []string `query:"tags"`
}

type CreateSecretRequest struct {
	Path        string                 `json:"path" validate:"required"`
	Value       interface{}            `json:"value" validate:"required"`
	ValueType   string                 `json:"valueType" validate:"required"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type GetSecretRequest struct {
	ID string `path:"id" validate:"required"`
}

type UpdateSecretRequest struct {
	ID          string                 `path:"id" validate:"required"`
	Value       interface{}            `json:"value"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type DeleteSecretRequest struct {
	ID string `path:"id" validate:"required"`
}

type GetValueRequest struct {
	ID string `path:"id" validate:"required"`
}

type GetByPathRequest struct {
	Path string `query:"path" validate:"required"`
}

type GetVersionsRequest struct {
	ID       string `path:"id" validate:"required"`
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
}

type RollbackRequest struct {
	ID      string `path:"id" validate:"required"`
	Version string `path:"version" validate:"required"`
	Reason  string `json:"reason"`
}

type GetTreeRequest struct {
	Prefix string `query:"prefix"`
}

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse is a generic success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// =============================================================================
// Secret CRUD Handlers
// =============================================================================

// List handles GET /secrets
func (h *Handler) List(c forge.Context) error {
	var req ListSecretsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
	}

	// Set defaults
	if req.SortBy == "" {
		req.SortBy = "path"
	}
	if req.SortOrder == "" {
		req.SortOrder = "asc"
	}

	query := &core.ListSecretsQuery{
		Prefix:    req.Prefix,
		ValueType: req.ValueType,
		Search:    req.Search,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Recursive: req.Recursive,
		Page:      req.Page,
		PageSize:  req.PageSize,
		Tags:      req.Tags,
	}

	secrets, pagination, err := h.service.List(c.Request().Context(), query)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, core.ListSecretsResponse{
		Secrets:    secrets,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalItems: pagination.TotalItems,
		TotalPages: pagination.TotalPages,
	})
}

// Create handles POST /secrets
func (h *Handler) Create(c forge.Context) error {
	var req CreateSecretRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
	}

	serviceReq := &core.CreateSecretRequest{
		Path:        req.Path,
		Value:       req.Value,
		ValueType:   req.ValueType,
		Description: req.Description,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
	}

	secret, err := h.service.Create(c.Request().Context(), serviceReq)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusCreated, secret)
}

// Get handles GET /secrets/:id
func (h *Handler) Get(c forge.Context) error {
	var req GetSecretRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
	}

	id, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid secret ID format",
		})
	}

	secret, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, secret)
}

// GetValue handles GET /secrets/:id/value
func (h *Handler) GetValue(c forge.Context) error {
	var req GetValueRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
	}

	id, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid secret ID format",
		})
	}

	value, err := h.service.GetValue(c.Request().Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	// Get secret for type info
	secret, _ := h.service.Get(c.Request().Context(), id)

	return c.JSON(http.StatusOK, core.RevealValueResponse{
		Value:     value,
		ValueType: secret.ValueType,
	})
}

// Update handles PUT /secrets/:id
func (h *Handler) Update(c forge.Context) error {
	var req UpdateSecretRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
	}

	id, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid secret ID format",
		})
	}

	serviceReq := &core.UpdateSecretRequest{
		Value:       req.Value,
		Description: req.Description,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
	}

	secret, err := h.service.Update(c.Request().Context(), id, serviceReq)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, secret)
}

// Delete handles DELETE /secrets/:id
func (h *Handler) Delete(c forge.Context) error {
	var req DeleteSecretRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
	}

	id, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid secret ID format",
		})
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Secret deleted successfully",
	})
}

// =============================================================================
// Path-based Handlers
// =============================================================================

// GetByPath handles GET /secrets/path/*path
func (h *Handler) GetByPath(c forge.Context) error {
	path := c.Param("path")
	if path == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_path",
			Message: "Path is required",
		})
	}

	secret, err := h.service.GetByPath(c.Request().Context(), path)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, secret)
}

// GetValueByPath handles GET /secrets/path/*path/value
func (h *Handler) GetValueByPath(c forge.Context) error {
	path := c.Param("path")
	if path == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_path",
			Message: "Path is required",
		})
	}

	value, err := h.service.GetValueByPath(c.Request().Context(), path)
	if err != nil {
		return h.handleError(c, err)
	}

	// Get secret for type info
	secret, _ := h.service.GetByPath(c.Request().Context(), path)

	return c.JSON(http.StatusOK, core.RevealValueResponse{
		Value:     value,
		ValueType: secret.ValueType,
	})
}

// =============================================================================
// Version Handlers
// =============================================================================

// GetVersions handles GET /secrets/:id/versions
func (h *Handler) GetVersions(c forge.Context) error {
	var req GetVersionsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
	}

	id, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid secret ID format",
		})
	}

	// Set defaults
	page := req.Page
	pageSize := req.PageSize
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}

	versions, pagination, err := h.service.GetVersions(c.Request().Context(), id, page, pageSize)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, core.ListVersionsResponse{
		Versions:   versions,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalItems: pagination.TotalItems,
		TotalPages: pagination.TotalPages,
	})
}

// Rollback handles POST /secrets/:id/rollback/:version
func (h *Handler) Rollback(c forge.Context) error {
	var req RollbackRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
	}

	id, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid secret ID format",
		})
	}

	version, err := strconv.Atoi(req.Version)
	if err != nil || version < 1 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_version",
			Message: "Invalid version number",
		})
	}

	secret, err := h.service.Rollback(c.Request().Context(), id, version, req.Reason)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, secret)
}

// =============================================================================
// Stats and Tree Handlers
// =============================================================================

// GetStats handles GET /secrets/stats
func (h *Handler) GetStats(c forge.Context) error {
	stats, err := h.service.GetStats(c.Request().Context())
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, stats)
}

// GetTree handles GET /secrets/tree
func (h *Handler) GetTree(c forge.Context) error {
	var req GetTreeRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
	}

	tree, err := h.service.GetTree(c.Request().Context(), req.Prefix)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, tree)
}

// =============================================================================
// Helper Methods
// =============================================================================

// parseID parses an xid from a URL parameter
func (h *Handler) parseID(c forge.Context, param string) (xid.ID, error) {
	idStr := c.Param(param)
	return xid.FromString(idStr)
}

// handleError converts service errors to HTTP responses
func (h *Handler) handleError(c forge.Context, err error) error {
	if err == nil {
		return nil
	}

	// Log the error
	if h.logger != nil {
		h.logger.Error("secrets handler error", forge.F("error", err.Error()))
	}

	// Determine status code based on error message
	errMsg := err.Error()

	switch {
	case contains(errMsg, "not found"):
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: errMsg,
		})
	case contains(errMsg, "already exists") || contains(errMsg, "conflict"):
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: errMsg,
		})
	case contains(errMsg, "invalid") || contains(errMsg, "bad request") || contains(errMsg, "required"):
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: errMsg,
		})
	case contains(errMsg, "expired"):
		return c.JSON(http.StatusGone, ErrorResponse{
			Error:   "gone",
			Message: errMsg,
		})
	case contains(errMsg, "forbidden") || contains(errMsg, "access denied"):
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: errMsg,
		})
	case contains(errMsg, "unauthorized"):
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: errMsg,
		})
	default:
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "An internal error occurred",
		})
	}
}

// splitTags splits a comma-separated tags string into a slice
func splitTags(tags string) []string {
	if tags == "" {
		return nil
	}

	result := make([]string, 0)
	current := ""
	for _, c := range tags {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else if c != ' ' {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
