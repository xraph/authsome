package routes

import (
	"github.com/xraph/authsome/core/jwt"
	jwtPlugin "github.com/xraph/authsome/plugins/jwt"
	"github.com/xraph/forge"
)

// RegisterJWTRoutes registers JWT-related routes
func RegisterJWTRoutes(router forge.Router, handler *jwtPlugin.Handler) {
	// JWT key management routes
	jwtKeys := router.Group("/jwt/keys")
	{
		jwtKeys.POST("", handler.CreateJWTKey,
			forge.WithName("jwt.keys.create"),
			forge.WithSummary("Create JWT key"),
			forge.WithDescription("Create a new JWT signing key for token generation and verification"),
			forge.WithRequestSchema(jwt.CreateJWTKeyRequest{}),
			forge.WithResponseSchema(200, "JWT key created", jwt.JWTKey{}),
			forge.WithResponseSchema(400, "Invalid request", JWTErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", JWTErrorResponse{}),
			forge.WithTags("JWT", "Keys"),
			forge.WithValidation(true),
		)
		
		jwtKeys.GET("", handler.ListJWTKeys,
			forge.WithName("jwt.keys.list"),
			forge.WithSummary("List JWT keys"),
			forge.WithDescription("List all JWT signing keys for the organization"),
			forge.WithResponseSchema(200, "JWT keys retrieved", jwt.ListJWTKeysResponse{}),
			forge.WithResponseSchema(500, "Internal server error", JWTErrorResponse{}),
			forge.WithTags("JWT", "Keys"),
		)
	}

	// JWT token routes
	jwtTokens := router.Group("/jwt")
	{
		jwtTokens.POST("/generate", handler.GenerateToken,
			forge.WithName("jwt.generate"),
			forge.WithSummary("Generate JWT token"),
			forge.WithDescription("Generate a new JWT token for authenticated access. Requires valid session or API key."),
			forge.WithRequestSchema(jwt.GenerateTokenRequest{}),
			forge.WithResponseSchema(200, "Token generated", jwt.GenerateTokenResponse{}),
			forge.WithResponseSchema(400, "Invalid request", JWTErrorResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", JWTErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", JWTErrorResponse{}),
			forge.WithTags("JWT", "Tokens"),
			forge.WithValidation(true),
		)
		
		jwtTokens.POST("/verify", handler.VerifyToken,
			forge.WithName("jwt.verify"),
			forge.WithSummary("Verify JWT token"),
			forge.WithDescription("Verify the validity and signature of a JWT token"),
			forge.WithRequestSchema(jwt.VerifyTokenRequest{}),
			forge.WithResponseSchema(200, "Token verified", jwt.VerifyTokenResponse{}),
			forge.WithResponseSchema(400, "Invalid request", JWTErrorResponse{}),
			forge.WithResponseSchema(401, "Invalid or expired token", JWTErrorResponse{}),
			forge.WithTags("JWT", "Tokens"),
			forge.WithValidation(true),
		)
		
		jwtTokens.GET("/jwks", handler.GetJWKS,
			forge.WithName("jwt.jwks"),
			forge.WithSummary("Get JSON Web Key Set (JWKS)"),
			forge.WithDescription("Retrieve the public keys used for JWT signature verification in JWKS format (RFC 7517)"),
			forge.WithResponseSchema(200, "JWKS retrieved", jwt.JWKSResponse{}),
			forge.WithResponseSchema(500, "Internal server error", JWTErrorResponse{}),
			forge.WithTags("JWT", "Keys"),
		)
	}
}

// DTOs for JWT routes

// JWTErrorResponse represents an error response for JWT operations
type JWTErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}