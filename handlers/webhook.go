package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// WebhookHandler handles webhook-related HTTP requests
type WebhookHandler struct {
	service *webhook.Service
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(service *webhook.Service) *WebhookHandler {
	return &WebhookHandler{
		service: service,
	}
}

// CreateWebhook creates a new webhook
func (h *WebhookHandler) CreateWebhook(c forge.Context) error {
	var req struct {
		URL          string            `json:"url"`
		Events       []string          `json:"events"`
		MaxRetries   int               `json:"maxRetries"`
		RetryBackoff string            `json:"retryBackoff"`
		Headers      map[string]string `json:"headers,omitempty"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Extract required context values
	appID, err := contexts.RequireAppID(c.Request().Context())
	if err != nil {
		authErr := webhook.MissingAppContext()
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	envID, err := contexts.RequireEnvironmentID(c.Request().Context())
	if err != nil {
		authErr := webhook.MissingEnvironmentContext()
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	// Set defaults
	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}
	if req.RetryBackoff == "" {
		req.RetryBackoff = "exponential"
	}

	// Create webhook with app-centric context
	createReq := &webhook.CreateWebhookRequest{
		AppID:         appID,
		EnvironmentID: envID,
		URL:           req.URL,
		Events:        req.Events,
		MaxRetries:    req.MaxRetries,
		RetryBackoff:  req.RetryBackoff,
		Headers:       req.Headers,
	}

	webhookObj, err := h.service.CreateWebhook(c.Request().Context(), createReq)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, webhookObj)
}

// GetWebhook retrieves a webhook by ID
func (h *WebhookHandler) GetWebhook(c forge.Context) error {
	webhookID := c.Param("id")
	if webhookID == "" {
		authErr := errs.New("INVALID_REQUEST", "webhook ID is required", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	id, err := xid.FromString(webhookID)
	if err != nil {
		authErr := errs.New("INVALID_REQUEST", "invalid webhook ID format", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	webhookObj, err := h.service.GetWebhook(c.Request().Context(), id)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, webhookObj)
}

// ListWebhooks lists webhooks for an app environment
func (h *WebhookHandler) ListWebhooks(c forge.Context) error {
	// Extract required context values
	appID, err := contexts.RequireAppID(c.Request().Context())
	if err != nil {
		authErr := webhook.MissingAppContext()
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	envID, err := contexts.RequireEnvironmentID(c.Request().Context())
	if err != nil {
		authErr := webhook.MissingEnvironmentContext()
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	// Parse pagination and filter parameters
	filter := &webhook.ListWebhooksFilter{
		AppID:         appID,
		EnvironmentID: envID,
	}

	if p := c.Request().URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			filter.Page = parsed
		}
	}

	if ps := c.Request().URL.Query().Get("limit"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			filter.Limit = parsed
		}
	}

	if enabled := c.Request().URL.Query().Get("enabled"); enabled != "" {
		if parsed, err := strconv.ParseBool(enabled); err == nil {
			filter.Enabled = &parsed
		}
	}

	if event := c.Request().URL.Query().Get("event"); event != "" {
		filter.Event = &event
	}

	response, err := h.service.ListWebhooks(c.Request().Context(), filter)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateWebhook updates a webhook
func (h *WebhookHandler) UpdateWebhook(c forge.Context) error {
	webhookID := c.Param("id")
	if webhookID == "" {
		authErr := errs.New("INVALID_REQUEST", "webhook ID is required", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	id, err := xid.FromString(webhookID)
	if err != nil {
		authErr := errs.New("INVALID_REQUEST", "invalid webhook ID format", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var req struct {
		URL          *string           `json:"url,omitempty"`
		Events       []string          `json:"events,omitempty"`
		Enabled      *bool             `json:"enabled,omitempty"`
		MaxRetries   *int              `json:"maxRetries,omitempty"`
		RetryBackoff *string           `json:"retryBackoff,omitempty"`
		Headers      map[string]string `json:"headers,omitempty"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		authErr := errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	updateReq := &webhook.UpdateWebhookRequest{
		URL:          req.URL,
		Events:       req.Events,
		Enabled:      req.Enabled,
		MaxRetries:   req.MaxRetries,
		RetryBackoff: req.RetryBackoff,
		Headers:      req.Headers,
	}

	webhookObj, err := h.service.UpdateWebhook(c.Request().Context(), id, updateReq)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, webhookObj)
}

// DeleteWebhook deletes a webhook
func (h *WebhookHandler) DeleteWebhook(c forge.Context) error {
	webhookID := c.Param("id")
	if webhookID == "" {
		authErr := errs.New("INVALID_REQUEST", "webhook ID is required", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	id, err := xid.FromString(webhookID)
	if err != nil {
		authErr := errs.New("INVALID_REQUEST", "invalid webhook ID format", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	err = h.service.DeleteWebhook(c.Request().Context(), id)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "webhook deleted successfully"})
}

// GetWebhookDeliveries retrieves delivery logs for a webhook
func (h *WebhookHandler) GetWebhookDeliveries(c forge.Context) error {
	webhookID := c.Param("id")
	if webhookID == "" {
		authErr := errs.New("INVALID_REQUEST", "webhook ID is required", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	id, err := xid.FromString(webhookID)
	if err != nil {
		authErr := errs.New("INVALID_REQUEST", "invalid webhook ID format", http.StatusBadRequest)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	// Parse pagination and filter parameters
	filter := &webhook.ListDeliveriesFilter{
		WebhookID: id,
	}

	if p := c.Request().URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			filter.Page = parsed
		}
	}

	if ps := c.Request().URL.Query().Get("limit"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			filter.Limit = parsed
		}
	}

	if status := c.Request().URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	response, err := h.service.ListDeliveries(c.Request().Context(), filter)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, response)
}
