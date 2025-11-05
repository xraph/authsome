package routes

import (
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/forge"
)

// RegisterWebhookRoutes registers webhook-related routes
func RegisterWebhookRoutes(router forge.Router, handler *handlers.WebhookHandler) {
	// Webhook management routes
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("", handler.CreateWebhook,
			forge.WithName("webhooks.create"),
			forge.WithSummary("Create webhook"),
			forge.WithDescription("Create a new webhook to receive event notifications. Webhooks are called when specified events occur."),
			forge.WithRequestSchema(webhook.CreateWebhookRequest{}),
			forge.WithResponseSchema(200, "Webhook created", webhook.Webhook{}),
			forge.WithResponseSchema(400, "Invalid request", WebhookErrorResponse{}),
			forge.WithTags("Webhooks"),
			forge.WithValidation(true),
		)
		
		webhooks.GET("", handler.ListWebhooks,
			forge.WithName("webhooks.list"),
			forge.WithSummary("List webhooks"),
			forge.WithDescription("List all webhooks for the organization. Supports pagination via page and page_size query parameters."),
			forge.WithResponseSchema(200, "Webhooks retrieved", webhook.ListWebhooksResponse{}),
			forge.WithResponseSchema(500, "Internal server error", WebhookErrorResponse{}),
			forge.WithTags("Webhooks"),
		)
		
		webhooks.GET("/:id", handler.GetWebhook,
			forge.WithName("webhooks.get"),
			forge.WithSummary("Get webhook"),
			forge.WithDescription("Retrieve a specific webhook by ID"),
			forge.WithResponseSchema(200, "Webhook retrieved", webhook.Webhook{}),
			forge.WithResponseSchema(400, "Invalid webhook ID", WebhookErrorResponse{}),
			forge.WithResponseSchema(404, "Webhook not found", WebhookErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", WebhookErrorResponse{}),
			forge.WithTags("Webhooks"),
		)
		
		webhooks.PUT("/:id", handler.UpdateWebhook,
			forge.WithName("webhooks.update"),
			forge.WithSummary("Update webhook"),
			forge.WithDescription("Update an existing webhook's URL, events, or settings"),
			forge.WithRequestSchema(webhook.UpdateWebhookRequest{}),
			forge.WithResponseSchema(200, "Webhook updated", webhook.Webhook{}),
			forge.WithResponseSchema(400, "Invalid request", WebhookErrorResponse{}),
			forge.WithTags("Webhooks"),
			forge.WithValidation(true),
		)
		
		webhooks.DELETE("/:id", handler.DeleteWebhook,
			forge.WithName("webhooks.delete"),
			forge.WithSummary("Delete webhook"),
			forge.WithDescription("Delete a webhook. This action is irreversible."),
			forge.WithResponseSchema(200, "Webhook deleted", WebhookDeleteResponse{}),
			forge.WithResponseSchema(400, "Invalid webhook ID or deletion failed", WebhookErrorResponse{}),
			forge.WithTags("Webhooks"),
		)
		
		webhooks.GET("/:id/deliveries", handler.GetWebhookDeliveries,
			forge.WithName("webhooks.deliveries.list"),
			forge.WithSummary("List webhook deliveries"),
			forge.WithDescription("Retrieve delivery logs for a specific webhook. Shows delivery attempts, responses, and retry information."),
			forge.WithResponseSchema(200, "Deliveries retrieved", webhook.ListDeliveriesResponse{}),
			forge.WithResponseSchema(400, "Invalid webhook ID", WebhookErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", WebhookErrorResponse{}),
			forge.WithTags("Webhooks"),
		)
	}
}

// DTOs for webhook routes

// WebhookErrorResponse represents an error response
type WebhookErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

// WebhookDeleteResponse represents a successful webhook deletion
type WebhookDeleteResponse struct {
	Message string `json:"message" example:"webhook deleted successfully"`
}