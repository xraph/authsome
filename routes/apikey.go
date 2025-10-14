package routes

import (
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/forge"
)

// RegisterAPIKeyRoutes registers API key management routes
func RegisterAPIKeyRoutes(router forge.Router, handler *handlers.APIKeyHandler) {
	// API key management routes
	apikeys := router.Group("/api-keys")
	{
		apikeys.POST("", handler.CreateAPIKey)
		apikeys.GET("", handler.ListAPIKeys)
		apikeys.GET("/:id", handler.GetAPIKey)
		apikeys.PUT("/:id", handler.UpdateAPIKey)
		apikeys.DELETE("/:id", handler.DeleteAPIKey)
		apikeys.POST("/verify", handler.VerifyAPIKey)
	}
}