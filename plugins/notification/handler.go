package notification

import (
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles notification HTTP requests
type Handler struct {
	service     *notification.Service
	templateSvc *TemplateService
	config      Config
}

// NewHandler creates a new notification handler
func NewHandler(service *notification.Service, templateSvc *TemplateService, config Config) *Handler {
	return &Handler{
		service:     service,
		templateSvc: templateSvc,
		config:      config,
	}
}

// =============================================================================
// TEMPLATE HANDLERS
// =============================================================================

// CreateTemplate creates a new notification template
func (h *Handler) CreateTemplate(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	var req notification.CreateTemplateRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	// Set app ID from context
	req.AppID = appID

	template, err := h.service.CreateTemplate(c.Context(), &req)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusCreated, template)
}

// GetTemplate retrieves a template by ID
func (h *Handler) GetTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	template, err := h.service.GetTemplate(c.Context(), id)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, template)
}

// ListTemplates lists all templates with pagination
func (h *Handler) ListTemplates(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Parse filter parameters
	var notifType *notification.NotificationType
	if typeStr := c.Query("type"); typeStr != "" {
		t := notification.NotificationType(typeStr)
		notifType = &t
	}

	var language *string
	if lang := c.Query("language"); lang != "" {
		language = &lang
	}

	var active *bool
	if activeStr := c.Query("active"); activeStr != "" {
		if a, err := strconv.ParseBool(activeStr); err == nil {
			active = &a
		}
	}

	filter := &notification.ListTemplatesFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		AppID:    appID,
		Type:     notifType,
		Language: language,
		Active:   active,
	}

	response, err := h.service.ListTemplates(c.Context(), filter)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateTemplate updates a template
func (h *Handler) UpdateTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	var req notification.UpdateTemplateRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	if err := h.service.UpdateTemplate(c.Context(), id, &req); err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "template updated successfully",
	})
}

// DeleteTemplate deletes a template
func (h *Handler) DeleteTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	if err := h.service.DeleteTemplate(c.Context(), id); err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "template deleted successfully",
	})
}

// ResetTemplate resets a template to default values
func (h *Handler) ResetTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	if err := h.service.ResetTemplate(c.Context(), id); err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "template reset to default successfully",
	})
}

// ResetAllTemplates resets all templates for an app to defaults
func (h *Handler) ResetAllTemplates(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	if err := h.service.ResetAllTemplates(c.Context(), appID); err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "all templates reset to defaults successfully",
	})
}

// GetTemplateDefaults returns default template metadata
func (h *Handler) GetTemplateDefaults(c forge.Context) error {
	// Get default template metadata
	defaults := notification.GetDefaultTemplateMetadata()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"templates": defaults,
		"count":     len(defaults),
	})
}

// PreviewTemplate renders a template with provided variables
func (h *Handler) PreviewTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid template ID format"))
	}

	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	template, err := h.service.GetTemplate(c.Context(), id)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	// Create template engine for rendering
	engine := NewTemplateEngine()

	// Render body
	body, err := engine.Render(template.Body, req.Variables)
	if err != nil {
		renderErr := notification.TemplateRenderFailed(err)
		return c.JSON(renderErr.HTTPStatus, renderErr)
	}

	// Render subject if present
	subject := ""
	if template.Subject != "" {
		subject, err = engine.Render(template.Subject, req.Variables)
		if err != nil {
			renderErr := notification.TemplateRenderFailed(err)
			return c.JSON(renderErr.HTTPStatus, renderErr)
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"subject": subject,
		"body":    body,
	})
}

// RenderTemplate renders a template string with variables (no template ID required)
func (h *Handler) RenderTemplate(c forge.Context) error {
	var req struct {
		Template  string                 `json:"template"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	engine := NewTemplateEngine()
	rendered, err := engine.Render(req.Template, req.Variables)
	if err != nil {
		renderErr := notification.TemplateRenderFailed(err)
		return c.JSON(renderErr.HTTPStatus, renderErr)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"rendered": rendered,
	})
}

// =============================================================================
// NOTIFICATION HANDLERS
// =============================================================================

// SendNotification sends a notification
func (h *Handler) SendNotification(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	var req notification.SendRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request"))
	}

	// Set app ID from context
	req.AppID = appID

	notif, err := h.service.Send(c.Context(), &req)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, notif)
}

// GetNotification retrieves a notification by ID
func (h *Handler) GetNotification(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid notification ID format"))
	}

	notif, err := h.service.GetNotification(c.Context(), id)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, notif)
}

// ListNotifications lists all notifications with pagination
func (h *Handler) ListNotifications(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Parse filter parameters
	var notifType *notification.NotificationType
	if typeStr := c.Query("type"); typeStr != "" {
		t := notification.NotificationType(typeStr)
		notifType = &t
	}

	var status *notification.NotificationStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := notification.NotificationStatus(statusStr)
		status = &s
	}

	var recipient *string
	if rec := c.Query("recipient"); rec != "" {
		recipient = &rec
	}

	filter := &notification.ListNotificationsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		AppID:     appID,
		Type:      notifType,
		Status:    status,
		Recipient: recipient,
	}

	response, err := h.service.ListNotifications(c.Context(), filter)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, response)
}

// ResendNotification resends a notification
func (h *Handler) ResendNotification(c forge.Context) error {
	// Extract app ID from context
	appID, err := contexts.RequireAppID(c.Context())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Unauthorized())
	}

	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid notification ID format"))
	}

	// Get original notification
	originalNotif, err := h.service.GetNotification(c.Context(), id)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	// Send new notification with same details
	newNotif, err := h.service.Send(c.Context(), &notification.SendRequest{
		AppID:     appID,
		Type:      originalNotif.Type,
		Recipient: originalNotif.Recipient,
		Subject:   originalNotif.Subject,
		Body:      originalNotif.Body,
		Metadata:  originalNotif.Metadata,
	})
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, newNotif)
}

// HandleWebhook handles provider webhook callbacks
func (h *Handler) HandleWebhook(c forge.Context) error {
	providerID := c.Param("provider")
	if providerID == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("provider ID required"))
	}

	// Parse webhook payload based on provider
	// This is a placeholder - actual implementation would depend on provider specs
	var payload map[string]interface{}
	if err := c.BindJSON(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid webhook payload"))
	}

	// Process webhook based on provider
	// For now, just acknowledge receipt
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "processed",
	})
}
