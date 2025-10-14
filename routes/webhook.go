package routes

import (
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/forge"
)

// RegisterWebhookRoutes registers webhook-related routes
func RegisterWebhookRoutes(router forge.Router, handler *handlers.WebhookHandler) {
	// Webhook management routes
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("", handler.CreateWebhook)
		webhooks.GET("", handler.ListWebhooks)
		webhooks.GET("/:id", handler.GetWebhook)
		webhooks.PUT("/:id", handler.UpdateWebhook)
		webhooks.DELETE("/:id", handler.DeleteWebhook)
		webhooks.GET("/:id/deliveries", handler.GetWebhookDeliveries)
	}
}