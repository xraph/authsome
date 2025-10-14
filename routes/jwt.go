package routes

import (
	"github.com/xraph/authsome/plugins/jwt"
	"github.com/xraph/forge"
)

// RegisterJWTRoutes registers JWT-related routes
func RegisterJWTRoutes(router forge.Router, handler *jwt.Handler) {
	// JWT key management routes
	jwtKeys := router.Group("/jwt/keys")
	{
		jwtKeys.POST("", handler.CreateJWTKey)
		jwtKeys.GET("", handler.ListJWTKeys)
	}

	// JWT token routes
	jwtTokens := router.Group("/jwt")
	{
		jwtTokens.POST("/generate", handler.GenerateToken)
		jwtTokens.POST("/verify", handler.VerifyToken)
		jwtTokens.GET("/jwks", handler.GetJWKS)
	}
}