package oidcprovider

import (
	"context"
	"slices"
)

// DiscoveryService handles OIDC discovery document generation.
type DiscoveryService struct {
	config Config
}

// NewDiscoveryService creates a new discovery service.
func NewDiscoveryService(config Config) *DiscoveryService {
	return &DiscoveryService{
		config: config,
	}
}

// GetDiscoveryDocument generates the OIDC discovery document (.well-known/openid-configuration).
func (s *DiscoveryService) GetDiscoveryDocument(ctx context.Context, baseURL, basePath string) *DiscoveryResponse {
	// Ensure baseURL doesn't end with slash
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	// Use configured issuer or fall back to baseURL
	issuer := s.config.Issuer
	if issuer == "" {
		issuer = baseURL
	}

	// Build grant types list
	grantTypes := []string{
		"authorization_code", // ✅ Implemented
		"refresh_token",      // ✅ Implemented
		"client_credentials", // ✅ Implemented
	}

	// Add device code grant if enabled
	deviceAuthEndpoint := ""

	if s.config.DeviceFlow.Enabled {
		grantTypes = append(grantTypes, "urn:ietf:params:oauth:grant-type:device_code")
		deviceAuthEndpoint = baseURL + basePath + "/device_authorization"
	}

	return &DiscoveryResponse{
		Issuer:                      issuer,
		AuthorizationEndpoint:       baseURL + basePath + "/authorize",
		TokenEndpoint:               baseURL + basePath + "/token",
		UserInfoEndpoint:            baseURL + basePath + "/userinfo",
		JwksURI:                     baseURL + basePath + "/jwks",
		RegistrationEndpoint:        baseURL + basePath + "/register",
		IntrospectionEndpoint:       baseURL + basePath + "/introspect",
		RevocationEndpoint:          baseURL + basePath + "/revoke",
		DeviceAuthorizationEndpoint: deviceAuthEndpoint,

		// Supported response types (only authorization code flow for now)
		ResponseTypesSupported: []string{
			"code", // Authorization code flow (primary)
		},

		// Supported response modes
		ResponseModesSupported: []string{
			"query", // Standard for authorization code
		},

		// Supported grant types (accurately reflects implementation)
		GrantTypesSupported: grantTypes,

		// Subject types
		SubjectTypesSupported: []string{
			"public",
		},

		// ID Token signing algorithms
		IDTokenSigningAlgValuesSupported: []string{
			"RS256",
			"RS384",
			"RS512",
		},

		// Supported scopes
		ScopesSupported: []string{
			"openid",
			"profile",
			"email",
			"phone",
			"address",
			"offline_access",
		},

		// Token endpoint auth methods
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_basic",
			"client_secret_post",
			"none",
		},

		// Introspection endpoint auth methods
		IntrospectionEndpointAuthMethodsSupported: []string{
			"client_secret_basic",
			"client_secret_post",
		},

		// Revocation endpoint auth methods
		RevocationEndpointAuthMethodsSupported: []string{
			"client_secret_basic",
			"client_secret_post",
		},

		// Supported claims
		ClaimsSupported: []string{
			"sub",
			"iss",
			"aud",
			"exp",
			"iat",
			"nbf",
			"jti",
			"auth_time",
			"acr",
			"amr",
			"nonce",
			"email",
			"email_verified",
			"name",
			"given_name",
			"family_name",
			"middle_name",
			"nickname",
			"preferred_username",
			"profile",
			"picture",
			"website",
			"gender",
			"birthdate",
			"zoneinfo",
			"locale",
			"updated_at",
			"phone_number",
			"phone_number_verified",
		},

		// PKCE support
		CodeChallengeMethodsSupported: []string{
			"S256",
			"plain",
		},

		// Additional capabilities
		RequestParameterSupported:     false,
		RequestURIParameterSupported:  false,
		RequireRequestURIRegistration: false,
		ClaimsParameterSupported:      false,
	}
}

// GetIssuer returns the configured issuer URL.
func (s *DiscoveryService) GetIssuer() string {
	return s.config.Issuer
}

// SupportsGrantType checks if a grant type is supported.
func (s *DiscoveryService) SupportsGrantType(grantType string) bool {
	supported := []string{"authorization_code", "refresh_token", "client_credentials", "implicit"}

	return slices.Contains(supported, grantType)
}

// SupportsResponseType checks if a response type is supported.
func (s *DiscoveryService) SupportsResponseType(responseType string) bool {
	supported := []string{"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token"}

	return slices.Contains(supported, responseType)
}

// SupportsScope checks if a scope is supported.
func (s *DiscoveryService) SupportsScope(scope string) bool {
	supported := []string{"openid", "profile", "email", "phone", "address", "offline_access"}

	return slices.Contains(supported, scope)
}
