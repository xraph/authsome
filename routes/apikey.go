package routes

import (
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/forge"
)

// RegisterAPIKeyRoutes registers API key management routes
func RegisterAPIKeyRoutes(router forge.Router, handler *handlers.APIKeyHandler) {
	// API key management routes
	apikeys := router.Group("/api-keys")
	{
		apikeys.POST("", handler.CreateAPIKey,
			forge.WithName("apikey.create"),
			forge.WithSummary("Create API key"),
			forge.WithDescription("Create a new API key for programmatic access to the API"),
			forge.WithRequestSchema(apikey.CreateAPIKeyRequest{}),
			forge.WithResponseSchema(201, "API key created", apikey.APIKey{}),
			forge.WithResponseSchema(400, "Invalid request", APIKeyErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", APIKeyErrorResponse{}),
			forge.WithTags("API Keys"),
			forge.WithValidation(true),
		)
		
		apikeys.GET("", handler.ListAPIKeys,
			forge.WithName("apikey.list"),
			forge.WithSummary("List API keys"),
			forge.WithDescription("List all API keys for the specified organization or user. Supports pagination via query parameters (limit, offset)."),
			forge.WithResponseSchema(200, "API keys retrieved", apikey.ListAPIKeysResponse{}),
			forge.WithResponseSchema(500, "Internal server error", APIKeyErrorResponse{}),
			forge.WithTags("API Keys"),
		)
		
		apikeys.GET("/:id", handler.GetAPIKey,
			forge.WithName("apikey.get"),
			forge.WithSummary("Get API key"),
			forge.WithDescription("Retrieve a specific API key by ID. Requires user_id and org_id query parameters."),
			forge.WithResponseSchema(200, "API key retrieved", apikey.APIKey{}),
			forge.WithResponseSchema(400, "Invalid request", APIKeyErrorResponse{}),
			forge.WithResponseSchema(404, "API key not found", APIKeyErrorResponse{}),
			forge.WithTags("API Keys"),
		)
		
		apikeys.PUT("/:id", handler.UpdateAPIKey,
			forge.WithName("apikey.update"),
			forge.WithSummary("Update API key"),
			forge.WithDescription("Update an existing API key's properties (name, scopes, permissions, rate limits, etc.)"),
			forge.WithRequestSchema(apikey.UpdateAPIKeyRequest{}),
			forge.WithResponseSchema(200, "API key updated", apikey.APIKey{}),
			forge.WithResponseSchema(400, "Invalid request", APIKeyErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", APIKeyErrorResponse{}),
			forge.WithTags("API Keys"),
			forge.WithValidation(true),
		)
		
		apikeys.DELETE("/:id", handler.DeleteAPIKey,
			forge.WithName("apikey.delete"),
			forge.WithSummary("Delete API key"),
			forge.WithDescription("Delete an API key. This action is irreversible."),
			forge.WithResponseSchema(200, "API key deleted", APIKeyDeleteResponse{}),
			forge.WithResponseSchema(400, "Invalid request", APIKeyErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", APIKeyErrorResponse{}),
			forge.WithTags("API Keys"),
		)
		
		apikeys.POST("/verify", handler.VerifyAPIKey,
			forge.WithName("apikey.verify"),
			forge.WithSummary("Verify API key"),
			forge.WithDescription("Verify the validity of an API key and retrieve its permissions and metadata"),
			forge.WithRequestSchema(apikey.VerifyAPIKeyRequest{}),
			forge.WithResponseSchema(200, "API key verified", apikey.VerifyAPIKeyResponse{}),
			forge.WithResponseSchema(400, "Invalid request", APIKeyErrorResponse{}),
			forge.WithResponseSchema(401, "Invalid or expired API key", APIKeyErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", APIKeyErrorResponse{}),
			forge.WithTags("API Keys"),
			forge.WithValidation(true),
		)
	}
}

// DTOs for API key routes

// APIKeyErrorResponse represents an error response for API key operations
type APIKeyErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

// APIKeyDeleteResponse represents a successful API key deletion
type APIKeyDeleteResponse struct {
	Message string `json:"message" example:"API key deleted successfully"`
}