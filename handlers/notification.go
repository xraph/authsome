package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/forge"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	service *notification.Service
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(service *notification.Service) *NotificationHandler {
	return &NotificationHandler{
		service: service,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// SendNotification sends a notification
func (h *NotificationHandler) SendNotification(c forge.Context) error {
	var req notification.SendRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}

	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}
	req.OrganizationID = orgID

	notification, err := h.service.Send(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, notification)
}

// GetNotification gets a notification by ID
func (h *NotificationHandler) GetNotification(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid notification ID",
		})
	}

	notification, err := h.service.GetNotification(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	if notification == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "Notification not found",
		})
	}

	return c.JSON(http.StatusOK, notification)
}

// ListNotifications lists notifications
func (h *NotificationHandler) ListNotifications(c forge.Context) error {
	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}

	// Parse query parameters
	query := c.Request().URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	notificationType := query.Get("type")
	status := query.Get("status")
	recipient := query.Get("recipient")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	req := &notification.ListNotificationsRequest{
		OrganizationID: orgID,
		Type:           notification.NotificationType(notificationType),
		Status:         notification.NotificationStatus(status),
		Recipient:      recipient,
		Limit:          limit,
		Offset:         offset,
	}

	notifications, total, err := h.service.ListNotifications(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	response := map[string]interface{}{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateDeliveryStatus updates the delivery status of a notification
func (h *NotificationHandler) UpdateDeliveryStatus(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid notification ID",
		})
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}

	status := notification.NotificationStatus(req.Status)
	if err := h.service.UpdateDeliveryStatus(c.Request().Context(), id, status); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Status updated successfully"})
}

// CreateTemplate creates a new notification template
func (h *NotificationHandler) CreateTemplate(c forge.Context) error {
	var req notification.CreateTemplateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}

	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}
	req.OrganizationID = orgID

	template, err := h.service.CreateTemplate(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, template)
}

// GetTemplate gets a template by ID
func (h *NotificationHandler) GetTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid template ID",
		})
	}

	template, err := h.service.GetTemplate(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	if template == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "Template not found",
		})
	}

	return c.JSON(http.StatusOK, template)
}

// UpdateTemplate updates a template
func (h *NotificationHandler) UpdateTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid template ID",
		})
	}

	var req notification.UpdateTemplateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}

	if err := h.service.UpdateTemplate(c.Request().Context(), id, &req); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Template updated successfully"})
}

// DeleteTemplate deletes a template
func (h *NotificationHandler) DeleteTemplate(c forge.Context) error {
	idStr := c.Param("id")
	id, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid template ID",
		})
	}

	if err := h.service.DeleteTemplate(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Template deleted successfully"})
}

// ListTemplates lists notification templates
func (h *NotificationHandler) ListTemplates(c forge.Context) error {
	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}

	// Parse query parameters
	query := c.Request().URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	notificationType := query.Get("type")
	activeStr := query.Get("active")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var active *bool
	if activeStr != "" {
		if a, err := strconv.ParseBool(activeStr); err == nil {
			active = &a
		}
	}

	req := &notification.ListTemplatesRequest{
		OrganizationID: orgID,
		Type:           notification.NotificationType(notificationType),
		Active:         active,
		Limit:          limit,
		Offset:         offset,
	}

	templates, total, err := h.service.ListTemplates(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	response := map[string]interface{}{
		"templates": templates,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	}

	return c.JSON(http.StatusOK, response)
}