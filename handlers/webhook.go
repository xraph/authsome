package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/webhook"
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

// CreateWebhookRequest represents a webhook creation request
type CreateWebhookRequest struct {
	URL          string            `json:"url"`
	Events       []string          `json:"events"`
	MaxRetries   int               `json:"max_retries"`
	RetryBackoff string            `json:"retry_backoff"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// UpdateWebhookRequest represents a webhook update request
type UpdateWebhookRequest struct {
	URL          *string           `json:"url,omitempty"`
	Events       []string          `json:"events,omitempty"`
	Enabled      *bool             `json:"enabled,omitempty"`
	MaxRetries   *int              `json:"max_retries,omitempty"`
	RetryBackoff *string           `json:"retry_backoff,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// CreateWebhook creates a new webhook
func (h *WebhookHandler) CreateWebhook(c forge.Context) error {
	var req CreateWebhookRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get organization ID from header or context
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default" // fallback for standalone mode
	}

	// Set defaults
	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}
	if req.RetryBackoff == "" {
		req.RetryBackoff = "exponential"
	}

	// Create webhook
	createReq := &webhook.CreateWebhookRequest{
		OrganizationID: orgID,
		URL:            req.URL,
		Events:         req.Events,
		MaxRetries:     req.MaxRetries,
		RetryBackoff:   req.RetryBackoff,
		Headers:        req.Headers,
	}

	webhookObj, err := h.service.CreateWebhook(c.Request().Context(), createReq)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, webhookObj)
}

// GetWebhook retrieves a webhook by ID
func (h *WebhookHandler) GetWebhook(c forge.Context) error {
	webhookID := c.Param("id")
	if webhookID == "" {
		return c.JSON(400, map[string]string{"error": "webhook ID is required"})
	}

	id, err := xid.FromString(webhookID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid webhook ID format"})
	}

	webhookObj, err := h.service.GetWebhook(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	if webhookObj == nil {
		return c.JSON(404, map[string]string{"error": "webhook not found"})
	}

	return c.JSON(200, webhookObj)
}

// ListWebhooks lists webhooks for an organization
func (h *WebhookHandler) ListWebhooks(c forge.Context) error {
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20

	if p := c.Request().URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Request().URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	req := &webhook.ListWebhooksRequest{
		OrganizationID: orgID,
		Page:           page,
		PageSize:       pageSize,
	}

	response, err := h.service.ListWebhooks(c.Request().Context(), req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, response)
}

// UpdateWebhook updates a webhook
func (h *WebhookHandler) UpdateWebhook(c forge.Context) error {
	webhookID := c.Param("id")
	if webhookID == "" {
		return c.JSON(400, map[string]string{"error": "webhook ID is required"})
	}

	id, err := xid.FromString(webhookID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid webhook ID format"})
	}

	var req UpdateWebhookRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
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
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, webhookObj)
}

// DeleteWebhook deletes a webhook
func (h *WebhookHandler) DeleteWebhook(c forge.Context) error {
	webhookID := c.Param("id")
	if webhookID == "" {
		return c.JSON(400, map[string]string{"error": "webhook ID is required"})
	}

	id, err := xid.FromString(webhookID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid webhook ID format"})
	}

	err = h.service.DeleteWebhook(c.Request().Context(), id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "webhook deleted successfully"})
}

// GetWebhookDeliveries retrieves delivery logs for a webhook
func (h *WebhookHandler) GetWebhookDeliveries(c forge.Context) error {
	webhookID := c.Param("id")
	if webhookID == "" {
		return c.JSON(400, map[string]string{"error": "webhook ID is required"})
	}

	id, err := xid.FromString(webhookID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid webhook ID format"})
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20

	if p := c.Request().URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Request().URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	req := &webhook.ListDeliveriesRequest{
		WebhookID: id,
		Page:      page,
		PageSize:  pageSize,
	}

	response, err := h.service.ListDeliveries(c.Request().Context(), req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, response)
}