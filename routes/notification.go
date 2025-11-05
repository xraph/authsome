package routes

import (
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/forge"
)

// RegisterNotificationRoutes registers notification-related routes
func RegisterNotificationRoutes(router forge.Router, handler *handlers.NotificationHandler) {
	// Notification routes
	notifications := router.Group("/notifications")
	{
		notifications.POST("/send", handler.SendNotification,
			forge.WithName("notifications.send"),
			forge.WithSummary("Send notification"),
			forge.WithDescription("Send a notification via email, SMS, or push. Requires organization ID header."),
			forge.WithRequestSchema(notification.SendRequest{}),
			forge.WithResponseSchema(201, "Notification sent", notification.Notification{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications"),
			forge.WithValidation(true),
		)

		notifications.GET("", handler.ListNotifications,
			forge.WithName("notifications.list"),
			forge.WithSummary("List notifications"),
			forge.WithDescription("List notifications with optional filtering by type, status, and recipient. Supports pagination."),
			forge.WithResponseSchema(200, "Notifications retrieved", NotificationsListResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications"),
		)

		notifications.GET("/:id", handler.GetNotification,
			forge.WithName("notifications.get"),
			forge.WithSummary("Get notification"),
			forge.WithDescription("Retrieve a specific notification by ID"),
			forge.WithResponseSchema(200, "Notification retrieved", notification.Notification{}),
			forge.WithResponseSchema(400, "Invalid notification ID", NotificationErrorResponse{}),
			forge.WithResponseSchema(404, "Notification not found", NotificationErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications"),
		)

		notifications.PUT("/:id/status", handler.UpdateDeliveryStatus,
			forge.WithName("notifications.status.update"),
			forge.WithSummary("Update notification status"),
			forge.WithDescription("Update the delivery status of a notification (pending, sent, delivered, failed)"),
			forge.WithRequestSchema(NotificationStatusUpdateRequest{}),
			forge.WithResponseSchema(200, "Status updated", NotificationStatusUpdateResponse{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications"),
			forge.WithValidation(true),
		)
	}

	// Template routes
	templates := router.Group("/notifications/templates")
	{
		templates.POST("", handler.CreateTemplate,
			forge.WithName("notifications.templates.create"),
			forge.WithSummary("Create notification template"),
			forge.WithDescription("Create a new notification template for email, SMS, or push notifications"),
			forge.WithRequestSchema(notification.CreateTemplateRequest{}),
			forge.WithResponseSchema(201, "Template created", notification.Template{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications", "Templates"),
			forge.WithValidation(true),
		)

		templates.GET("", handler.ListTemplates,
			forge.WithName("notifications.templates.list"),
			forge.WithSummary("List notification templates"),
			forge.WithDescription("List all notification templates with optional filtering by type and active status"),
			forge.WithResponseSchema(200, "Templates retrieved", TemplatesListResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications", "Templates"),
		)

		templates.GET("/:id", handler.GetTemplate,
			forge.WithName("notifications.templates.get"),
			forge.WithSummary("Get notification template"),
			forge.WithDescription("Retrieve a specific notification template by ID"),
			forge.WithResponseSchema(200, "Template retrieved", notification.Template{}),
			forge.WithResponseSchema(400, "Invalid template ID", NotificationErrorResponse{}),
			forge.WithResponseSchema(404, "Template not found", NotificationErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications", "Templates"),
		)

		templates.PUT("/:id", handler.UpdateTemplate,
			forge.WithName("notifications.templates.update"),
			forge.WithSummary("Update notification template"),
			forge.WithDescription("Update an existing notification template"),
			forge.WithRequestSchema(notification.UpdateTemplateRequest{}),
			forge.WithResponseSchema(200, "Template updated", NotificationUpdateResponse{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications", "Templates"),
			forge.WithValidation(true),
		)

		templates.DELETE("/:id", handler.DeleteTemplate,
			forge.WithName("notifications.templates.delete"),
			forge.WithSummary("Delete notification template"),
			forge.WithDescription("Delete a notification template. This action is irreversible."),
			forge.WithResponseSchema(200, "Template deleted", NotificationDeleteResponse{}),
			forge.WithResponseSchema(400, "Invalid template ID", NotificationErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", NotificationErrorResponse{}),
			forge.WithTags("Notifications", "Templates"),
		)
	}
}

// DTOs for notification routes

// NotificationErrorResponse represents an error response
type NotificationErrorResponse struct {
	Message string `json:"message" example:"Error message"`
	Code    string `json:"code,omitempty" example:"ERROR_CODE"`
}

// NotificationsListResponse represents a list of notifications with metadata
type NotificationsListResponse struct {
	Notifications []notification.Notification `json:"notifications"`
	Total         int                         `json:"total"`
	Limit         int                         `json:"limit"`
	Offset        int                         `json:"offset"`
}

// NotificationStatusUpdateRequest represents a request to update notification status
type NotificationStatusUpdateRequest struct {
	Status string `json:"status" validate:"required" example:"delivered"`
}

// NotificationStatusUpdateResponse represents a successful status update
type NotificationStatusUpdateResponse struct {
	Message string `json:"message" example:"Status updated successfully"`
}

// TemplatesListResponse represents a list of templates with metadata
type TemplatesListResponse struct {
	Templates []notification.Template `json:"templates"`
	Total     int                     `json:"total"`
	Limit     int                     `json:"limit"`
	Offset    int                     `json:"offset"`
}

// NotificationUpdateResponse represents a successful template update
type NotificationUpdateResponse struct {
	Message string `json:"message" example:"Template updated successfully"`
}

// NotificationDeleteResponse represents a successful template deletion
type NotificationDeleteResponse struct {
	Message string `json:"message" example:"Template deleted successfully"`
}
