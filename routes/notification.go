package routes

import (
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/forge"
)

// RegisterNotificationRoutes registers notification-related routes
func RegisterNotificationRoutes(router forge.Router, handler *handlers.NotificationHandler) {
	// Notification routes
	notifications := router.Group("/notifications")
	{
		notifications.POST("/send", handler.SendNotification)
		notifications.GET("", handler.ListNotifications)
		notifications.GET("/:id", handler.GetNotification)
		notifications.PUT("/:id/status", handler.UpdateDeliveryStatus)
	}

	// Template routes
	templates := router.Group("/notifications/templates")
	{
		templates.POST("", handler.CreateTemplate)
		templates.GET("", handler.ListTemplates)
		templates.GET("/:id", handler.GetTemplate)
		templates.PUT("/:id", handler.UpdateTemplate)
		templates.DELETE("/:id", handler.DeleteTemplate)
	}
}