package notification

import (
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/notification"
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

// CreateTemplate creates a new notification template
func (h *Handler) CreateTemplate(c forge.Context) error {
	var req notification.CreateTemplateRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid request",
		})
	}

	// Set default organization if not provided
	if req.OrganizationID == "" {
		req.OrganizationID = "default"
	}

	template, err := h.service.CreateTemplate(c.Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, template)
}

// GetTemplate retrieves a template by ID
func (h *Handler) GetTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid template ID",
		})
	}

	template, err := h.service.GetTemplate(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "template not found",
		})
	}

	return c.JSON(http.StatusOK, template)
}

// ListTemplates lists all templates
func (h *Handler) ListTemplates(c forge.Context) error {
	orgID := c.Query("organization_id")
	if orgID == "" {
		orgID = "default"
	}

	notifType := c.Query("type")
	language := c.Query("language")

	req := &notification.ListTemplatesRequest{
		OrganizationID: orgID,
		Type:           notification.NotificationType(notifType),
		Language:       language,
		Limit:          50,
		Offset:         0,
	}

	templates, total, err := h.service.ListTemplates(c.Context(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"templates": templates,
		"total":     total,
	})
}

// UpdateTemplate updates a template
func (h *Handler) UpdateTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid template ID",
		})
	}

	var req notification.UpdateTemplateRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid request",
		})
	}

	if err := h.service.UpdateTemplate(c.Context(), id, &req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid template ID",
		})
	}

	if err := h.service.DeleteTemplate(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "template deleted successfully",
	})
}

// PreviewTemplate previews a template with test variables
func (h *Handler) PreviewTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid template ID",
		})
	}

	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid request",
		})
	}

	subject, body, err := h.templateSvc.RenderTemplate(c.Context(), id, req.Variables)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"subject": subject,
		"body":    body,
	})
}

// RenderTemplate renders a template without sending
func (h *Handler) RenderTemplate(c forge.Context) error {
	var req struct {
		TemplateKey    string                 `json:"template_key"`
		OrganizationID string                 `json:"organization_id"`
		Type           string                 `json:"type"`
		Language       string                 `json:"language"`
		Variables      map[string]interface{} `json:"variables"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid request",
		})
	}

	if req.OrganizationID == "" {
		req.OrganizationID = "default"
	}

	// Find template
	template, err := h.templateSvc.findTemplate(c.Context(), req.OrganizationID, req.TemplateKey, req.Type, req.Language)
	if err != nil || template == nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "template not found",
		})
	}

	subject, body, err := h.templateSvc.RenderTemplate(c.Context(), template.ID, req.Variables)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"subject": subject,
		"body":    body,
	})
}

// SendNotification sends a notification
func (h *Handler) SendNotification(c forge.Context) error {
	var req SendWithTemplateRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid request",
		})
	}

	notification, err := h.templateSvc.SendWithTemplate(c.Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, notification)
}

// ListNotifications lists notifications
func (h *Handler) ListNotifications(c forge.Context) error {
	orgID := c.Query("organization_id")
	if orgID == "" {
		orgID = "default"
	}

	req := &notification.ListNotificationsRequest{
		OrganizationID: orgID,
		Limit:          50,
		Offset:         0,
	}

	notifications, total, err := h.service.ListNotifications(c.Context(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"total":         total,
	})
}

// GetNotification retrieves a notification by ID
func (h *Handler) GetNotification(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid notification ID",
		})
	}

	notification, err := h.service.GetNotification(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "notification not found",
		})
	}

	return c.JSON(http.StatusOK, notification)
}

// ResendNotification resends a failed notification
func (h *Handler) ResendNotification(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid notification ID",
		})
	}

	// Get the notification
	notif, err := h.service.GetNotification(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "notification not found",
		})
	}

	// Resend
	newNotif, err := h.service.Send(c.Context(), &notification.SendRequest{
		OrganizationID: notif.OrganizationID,
		Type:           notif.Type,
		Recipient:      notif.Recipient,
		Subject:        notif.Subject,
		Body:           notif.Body,
		Metadata:       notif.Metadata,
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, newNotif)
}

// HandleWebhook handles provider webhooks for delivery status
func (h *Handler) HandleWebhook(c forge.Context) error {
	provider := c.Param("provider")

	// Parse webhook payload based on provider
	var payload map[string]interface{}
	if err := c.BindJSON(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid payload",
		})
	}

	// TODO: Implement provider-specific webhook handling
	// For now, just acknowledge receipt

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "webhook received",
		"provider": provider,
	})
}
