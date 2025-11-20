package oidcprovider

import "github.com/xraph/authsome/core/responses"

// =============================================================================
// AUTHORIZATION & CONSENT
// =============================================================================

// AuthorizeRequest represents an OAuth2/OIDC authorization request
type AuthorizeRequest struct {
	ClientID            string `json:"client_id" form:"client_id" validate:"required"`
	RedirectURI         string `json:"redirect_uri" form:"redirect_uri" validate:"required,url"`
	ResponseType        string `json:"response_type" form:"response_type" validate:"required"`
	Scope               string `json:"scope" form:"scope"`
	State               string `json:"state" form:"state"`
	Nonce               string `json:"nonce" form:"nonce"`
	CodeChallenge       string `json:"code_challenge" form:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method" form:"code_challenge_method"`
	Prompt              string `json:"prompt" form:"prompt"` // none, login, consent, select_account
	MaxAge              *int   `json:"max_age" form:"max_age"`
	UILocales           string `json:"ui_locales" form:"ui_locales"`
	IDTokenHint         string `json:"id_token_hint" form:"id_token_hint"`
	LoginHint           string `json:"login_hint" form:"login_hint"`
	ACRValues           string `json:"acr_values" form:"acr_values"`
}

// ConsentRequest represents the consent form submission
type ConsentRequest struct {
	Action              string `json:"action" form:"action" validate:"required,oneof=allow deny"`
	ClientID            string `json:"client_id" form:"client_id" validate:"required"`
	RedirectURI         string `json:"redirect_uri" form:"redirect_uri" validate:"required"`
	ResponseType        string `json:"response_type" form:"response_type" validate:"required"`
	Scope               string `json:"scope" form:"scope"`
	State               string `json:"state" form:"state"`
	CodeChallenge       string `json:"code_challenge" form:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method" form:"code_challenge_method"`
}

// =============================================================================
// TOKEN ENDPOINT
// =============================================================================

// TokenRequest represents the token endpoint request
type TokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type" validate:"required"`
	Code         string `json:"code" form:"code"`
	RedirectURI  string `json:"redirect_uri" form:"redirect_uri"`
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
	CodeVerifier string `json:"code_verifier" form:"code_verifier"`
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
	Scope        string `json:"scope" form:"scope"`
	// Client Credentials grant
	Audience string `json:"audience" form:"audience"`
}

// TokenResponse represents the OAuth2/OIDC token response
type TokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int    `json:"expires_in" example:"3600"`
	RefreshToken string `json:"refresh_token,omitempty" example:"def50200..."`
	IDToken      string `json:"id_token,omitempty" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Scope        string `json:"scope,omitempty" example:"openid profile email"`
}

// =============================================================================
// USERINFO ENDPOINT
// =============================================================================

// UserInfoResponse represents the OIDC userinfo endpoint response
type UserInfoResponse struct {
	Sub               string `json:"sub" example:"01HZ..."`
	Email             string `json:"email,omitempty" example:"user@example.com"`
	EmailVerified     bool   `json:"email_verified,omitempty" example:"true"`
	Name              string `json:"name,omitempty" example:"John Doe"`
	GivenName         string `json:"given_name,omitempty" example:"John"`
	FamilyName        string `json:"family_name,omitempty" example:"Doe"`
	MiddleName        string `json:"middle_name,omitempty"`
	Nickname          string `json:"nickname,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty" example:"johndoe"`
	Profile           string `json:"profile,omitempty"`
	Picture           string `json:"picture,omitempty" example:"https://example.com/avatar.jpg"`
	Website           string `json:"website,omitempty"`
	Gender            string `json:"gender,omitempty"`
	Birthdate         string `json:"birthdate,omitempty"`
	Zoneinfo          string `json:"zoneinfo,omitempty"`
	Locale            string `json:"locale,omitempty"`
	UpdatedAt         int64  `json:"updated_at,omitempty"`
	PhoneNumber       string `json:"phone_number,omitempty"`
	PhoneVerified     bool   `json:"phone_number_verified,omitempty"`
}

// =============================================================================
// CLIENT REGISTRATION (RFC 7591)
// =============================================================================

// ClientRegistrationRequest represents a dynamic client registration request (RFC 7591)
type ClientRegistrationRequest struct {
	ClientName              string   `json:"client_name" validate:"required"`
	RedirectURIs            []string `json:"redirect_uris" validate:"required,min=1,dive,url"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty" validate:"omitempty,dive,url"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	ApplicationType         string   `json:"application_type,omitempty" validate:"omitempty,oneof=web native spa"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty" validate:"omitempty,oneof=client_secret_basic client_secret_post none"`
	LogoURI                 string   `json:"logo_uri,omitempty" validate:"omitempty,url"`
	PolicyURI               string   `json:"policy_uri,omitempty" validate:"omitempty,url"`
	TosURI                  string   `json:"tos_uri,omitempty" validate:"omitempty,url"`
	Contacts                []string `json:"contacts,omitempty" validate:"omitempty,dive,email"`
	Scope                   string   `json:"scope,omitempty"`
	RequirePKCE             bool     `json:"require_pkce,omitempty"`
	RequireConsent          bool     `json:"require_consent,omitempty"`
	TrustedClient           bool     `json:"trusted_client,omitempty"`
}

// ClientRegistrationResponse represents a successful client registration response (RFC 7591)
type ClientRegistrationResponse struct {
	ClientID                string   `json:"client_id" example:"client_01HZ..."`
	ClientSecret            string   `json:"client_secret,omitempty" example:"secret_01HZ..."`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at" example:"1609459200"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at" example:"0"` // 0 = never expires
	ClientName              string   `json:"client_name" example:"My Application"`
	RedirectURIs            []string `json:"redirect_uris" example:"https://example.com/callback"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty"`
	GrantTypes              []string `json:"grant_types" example:"authorization_code,refresh_token"`
	ResponseTypes           []string `json:"response_types" example:"code"`
	ApplicationType         string   `json:"application_type" example:"web"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method" example:"client_secret_basic"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	TosURI                  string   `json:"tos_uri,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
}

// =============================================================================
// TOKEN INTROSPECTION (RFC 7662)
// =============================================================================

// TokenIntrospectionRequest represents a token introspection request (RFC 7662)
type TokenIntrospectionRequest struct {
	Token         string `json:"token" form:"token" validate:"required"`
	TokenTypeHint string `json:"token_type_hint" form:"token_type_hint"` // access_token, refresh_token
	ClientID      string `json:"client_id" form:"client_id"`
	ClientSecret  string `json:"client_secret" form:"client_secret"`
}

// TokenIntrospectionResponse represents a token introspection response (RFC 7662)
type TokenIntrospectionResponse struct {
	Active    bool     `json:"active" example:"true"`
	Scope     string   `json:"scope,omitempty" example:"openid profile email"`
	ClientID  string   `json:"client_id,omitempty" example:"client_123"`
	Username  string   `json:"username,omitempty" example:"johndoe"`
	TokenType string   `json:"token_type,omitempty" example:"Bearer"`
	Exp       int64    `json:"exp,omitempty" example:"1609459200"`
	Iat       int64    `json:"iat,omitempty" example:"1609455600"`
	Nbf       int64    `json:"nbf,omitempty" example:"1609455600"`
	Sub       string   `json:"sub,omitempty" example:"01HZ..."`
	Aud       []string `json:"aud,omitempty" example:"https://api.example.com"`
	Iss       string   `json:"iss,omitempty" example:"https://auth.example.com"`
	Jti       string   `json:"jti,omitempty" example:"token_01HZ..."`
}

// =============================================================================
// TOKEN REVOCATION (RFC 7009)
// =============================================================================

// TokenRevocationRequest represents a token revocation request (RFC 7009)
type TokenRevocationRequest struct {
	Token         string `json:"token" form:"token" validate:"required"`
	TokenTypeHint string `json:"token_type_hint" form:"token_type_hint"` // access_token, refresh_token
	ClientID      string `json:"client_id" form:"client_id"`
	ClientSecret  string `json:"client_secret" form:"client_secret"`
}

// =============================================================================
// JWKS ENDPOINT
// =============================================================================

// JWKSResponse represents the JSON Web Key Set response
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWKResponse is an alias for the JWKS key structure defined in jwks.go
type JWKResponse = JWK

// =============================================================================
// DISCOVERY ENDPOINT (.well-known/openid-configuration)
// =============================================================================

// DiscoveryResponse represents the OIDC discovery document
type DiscoveryResponse struct {
	Issuer                                     string   `json:"issuer" example:"https://auth.example.com"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint" example:"https://auth.example.com/oauth2/authorize"`
	TokenEndpoint                              string   `json:"token_endpoint" example:"https://auth.example.com/oauth2/token"`
	UserInfoEndpoint                           string   `json:"userinfo_endpoint" example:"https://auth.example.com/oauth2/userinfo"`
	JwksURI                                    string   `json:"jwks_uri" example:"https://auth.example.com/oauth2/jwks"`
	RegistrationEndpoint                       string   `json:"registration_endpoint,omitempty" example:"https://auth.example.com/oauth2/register"`
	IntrospectionEndpoint                      string   `json:"introspection_endpoint,omitempty" example:"https://auth.example.com/oauth2/introspect"`
	RevocationEndpoint                         string   `json:"revocation_endpoint,omitempty" example:"https://auth.example.com/oauth2/revoke"`
	ResponseTypesSupported                     []string `json:"response_types_supported" example:"code,token,id_token"`
	ResponseModesSupported                     []string `json:"response_modes_supported,omitempty" example:"query,fragment,form_post"`
	GrantTypesSupported                        []string `json:"grant_types_supported" example:"authorization_code,refresh_token,client_credentials"`
	SubjectTypesSupported                      []string `json:"subject_types_supported" example:"public"`
	IDTokenSigningAlgValuesSupported           []string `json:"id_token_signing_alg_values_supported" example:"RS256"`
	ScopesSupported                            []string `json:"scopes_supported" example:"openid,profile,email"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported" example:"client_secret_basic,client_secret_post"`
	ClaimsSupported                            []string `json:"claims_supported" example:"sub,name,email,picture"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported,omitempty" example:"S256,plain"`
	IntrospectionEndpointAuthMethodsSupported  []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`
	RevocationEndpointAuthMethodsSupported     []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`
	RequestParameterSupported                  bool     `json:"request_parameter_supported,omitempty"`
	RequestURIParameterSupported               bool     `json:"request_uri_parameter_supported,omitempty"`
	RequireRequestURIRegistration              bool     `json:"require_request_uri_registration,omitempty"`
	ClaimsParameterSupported                   bool     `json:"claims_parameter_supported,omitempty"`
}

// =============================================================================
// CLIENT MANAGEMENT (Admin Endpoints)
// =============================================================================

// ClientsListResponse represents a list of OAuth clients
type ClientsListResponse struct {
	Clients    []ClientSummary `json:"clients"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}

// ClientSummary represents a summary of an OAuth client
type ClientSummary struct {
	ClientID        string `json:"clientID"`
	Name            string `json:"name"`
	ApplicationType string `json:"applicationType"`
	CreatedAt       string `json:"createdAt"`
	IsOrgLevel      bool   `json:"isOrgLevel"`
}

// ClientDetailsResponse represents detailed information about an OAuth client
type ClientDetailsResponse struct {
	ClientID                string   `json:"clientID"`
	Name                    string   `json:"name"`
	ApplicationType         string   `json:"applicationType"`
	RedirectURIs            []string `json:"redirectURIs"`
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs,omitempty"`
	GrantTypes              []string `json:"grantTypes"`
	ResponseTypes           []string `json:"responseTypes"`
	AllowedScopes           []string `json:"allowedScopes,omitempty"`
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
	RequirePKCE             bool     `json:"requirePKCE"`
	RequireConsent          bool     `json:"requireConsent"`
	TrustedClient           bool     `json:"trustedClient"`
	LogoURI                 string   `json:"logoURI,omitempty"`
	PolicyURI               string   `json:"policyURI,omitempty"`
	TosURI                  string   `json:"tosURI,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
	CreatedAt               string   `json:"createdAt"`
	UpdatedAt               string   `json:"updatedAt"`
	IsOrgLevel              bool     `json:"isOrgLevel"`
	OrganizationID          string   `json:"organizationID,omitempty"`
}

// ClientUpdateRequest represents a client update request
type ClientUpdateRequest struct {
	Name                    string   `json:"name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris,omitempty" validate:"omitempty,dive,url"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty" validate:"omitempty,dive,url"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	AllowedScopes           []string `json:"allowed_scopes,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty" validate:"omitempty,oneof=client_secret_basic client_secret_post none"`
	RequirePKCE             *bool    `json:"require_pkce,omitempty"`
	RequireConsent          *bool    `json:"require_consent,omitempty"`
	TrustedClient           *bool    `json:"trusted_client,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty" validate:"omitempty,url"`
	PolicyURI               string   `json:"policy_uri,omitempty" validate:"omitempty,url"`
	TosURI                  string   `json:"tos_uri,omitempty" validate:"omitempty,url"`
	Contacts                []string `json:"contacts,omitempty" validate:"omitempty,dive,email"`
}

// =============================================================================
// ERROR RESPONSES
// =============================================================================

// ErrorResponse is the standard OAuth2/OIDC error response
type ErrorResponse = responses.ErrorResponse

// OAuthErrorResponse represents an OAuth2-specific error response
type OAuthErrorResponse struct {
	Error            string `json:"error" example:"invalid_request"`
	ErrorDescription string `json:"error_description,omitempty" example:"The request is missing a required parameter"`
	ErrorURI         string `json:"error_uri,omitempty" example:"https://docs.example.com/errors/invalid_request"`
	State            string `json:"state,omitempty"`
}

// =============================================================================
// INTERNAL TYPES
// =============================================================================

// ScopeInfo represents a scope with its description for consent screens
type ScopeInfo struct {
	Name        string
	Description string
}

// ConsentDecision represents a user's consent decision
type ConsentDecision struct {
	Approved bool
	Scopes   []string
}

// ClientAuthResult represents the result of client authentication
type ClientAuthResult struct {
	ClientID     string
	Authenticated bool
	Method       string // basic, post, none
}

