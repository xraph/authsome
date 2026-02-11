package oidcprovider

import "github.com/xraph/authsome/core/responses"

// =============================================================================
// AUTHORIZATION & CONSENT
// =============================================================================

// AuthorizeRequest represents an OAuth2/OIDC authorization request.
type AuthorizeRequest struct {
	ClientID            string `form:"client_id"             json:"client_id"             validate:"required"`
	RedirectURI         string `form:"redirect_uri"          json:"redirect_uri"          validate:"required,url"`
	ResponseType        string `form:"response_type"         json:"response_type"         validate:"required"`
	Scope               string `form:"scope"                 json:"scope"`
	State               string `form:"state"                 json:"state"`
	Nonce               string `form:"nonce"                 json:"nonce"`
	CodeChallenge       string `form:"code_challenge"        json:"code_challenge"`
	CodeChallengeMethod string `form:"code_challenge_method" json:"code_challenge_method"`
	Prompt              string `form:"prompt"                json:"prompt"` // none, login, consent, select_account
	MaxAge              *int   `form:"max_age"               json:"max_age"`
	UILocales           string `form:"ui_locales"            json:"ui_locales"`
	IDTokenHint         string `form:"id_token_hint"         json:"id_token_hint"`
	LoginHint           string `form:"login_hint"            json:"login_hint"`
	ACRValues           string `form:"acr_values"            json:"acr_values"`
}

// ConsentRequest represents the consent form submission.
type ConsentRequest struct {
	Action              string `form:"action"                json:"action"                validate:"required,oneof=allow deny"`
	ClientID            string `form:"client_id"             json:"client_id"             validate:"required"`
	RedirectURI         string `form:"redirect_uri"          json:"redirect_uri"          validate:"required"`
	ResponseType        string `form:"response_type"         json:"response_type"         validate:"required"`
	Scope               string `form:"scope"                 json:"scope"`
	State               string `form:"state"                 json:"state"`
	CodeChallenge       string `form:"code_challenge"        json:"code_challenge"`
	CodeChallengeMethod string `form:"code_challenge_method" json:"code_challenge_method"`
}

// =============================================================================
// TOKEN ENDPOINT
// =============================================================================

// TokenRequest represents the token endpoint request.
type TokenRequest struct {
	GrantType    string `form:"grant_type"    json:"grant_type"    validate:"required"`
	Code         string `form:"code"          json:"code"`
	RedirectURI  string `form:"redirect_uri"  json:"redirect_uri"`
	ClientID     string `form:"client_id"     json:"client_id"`
	ClientSecret string `form:"client_secret" json:"client_secret"`
	CodeVerifier string `form:"code_verifier" json:"code_verifier"`
	RefreshToken string `form:"refresh_token" json:"refresh_token"`
	Scope        string `form:"scope"         json:"scope"`
	// Client Credentials grant
	Audience string `form:"audience" json:"audience"`
	// Device Code grant (RFC 8628)
	DeviceCode string `form:"device_code" json:"device_code"`
}

// TokenResponse represents the OAuth2/OIDC token response.
type TokenResponse struct {
	AccessToken  string `example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." json:"access_token"`
	TokenType    string `example:"Bearer"                                  json:"token_type"`
	ExpiresIn    int    `example:"3600"                                    json:"expires_in"`
	RefreshToken string `example:"def50200..."                             json:"refresh_token,omitempty"`
	IDToken      string `example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." json:"id_token,omitempty"`
	Scope        string `example:"openid profile email"                    json:"scope,omitempty"`
}

// =============================================================================
// USERINFO ENDPOINT
// =============================================================================

// UserInfoResponse represents the OIDC userinfo endpoint response.
type UserInfoResponse struct {
	Sub               string `example:"01HZ..."                        json:"sub"`
	Email             string `example:"user@example.com"               json:"email,omitempty"`
	EmailVerified     bool   `example:"true"                           json:"email_verified,omitempty"`
	Name              string `example:"John Doe"                       json:"name,omitempty"`
	GivenName         string `example:"John"                           json:"given_name,omitempty"`
	FamilyName        string `example:"Doe"                            json:"family_name,omitempty"`
	MiddleName        string `json:"middle_name,omitempty"`
	Nickname          string `json:"nickname,omitempty"`
	PreferredUsername string `example:"johndoe"                        json:"preferred_username,omitempty"`
	Profile           string `json:"profile,omitempty"`
	Picture           string `example:"https://example.com/avatar.jpg" json:"picture,omitempty"`
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

// ClientRegistrationRequest represents a dynamic client registration request (RFC 7591).
type ClientRegistrationRequest struct {
	ClientName              string   `json:"client_name"                          validate:"required"`
	RedirectURIs            []string `json:"redirect_uris"                        validate:"required,min=1,dive,url"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty"  validate:"omitempty,dive,url"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	ApplicationType         string   `json:"application_type,omitempty"           validate:"omitempty,oneof=web native spa"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty" validate:"omitempty,oneof=client_secret_basic client_secret_post none"`
	LogoURI                 string   `json:"logo_uri,omitempty"                   validate:"omitempty,url"`
	PolicyURI               string   `json:"policy_uri,omitempty"                 validate:"omitempty,url"`
	TosURI                  string   `json:"tos_uri,omitempty"                    validate:"omitempty,url"`
	Contacts                []string `json:"contacts,omitempty"                   validate:"omitempty,dive,email"`
	Scope                   string   `json:"scope,omitempty"`
	RequirePKCE             bool     `json:"require_pkce,omitempty"`
	RequireConsent          bool     `json:"require_consent,omitempty"`
	TrustedClient           bool     `json:"trusted_client,omitempty"`
}

// ClientRegistrationResponse represents a successful client registration response (RFC 7591).
type ClientRegistrationResponse struct {
	ClientID                string   `example:"client_01HZ..."                   json:"client_id"`
	ClientSecret            string   `example:"secret_01HZ..."                   json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `example:"1609459200"                       json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `example:"0"                                json:"client_secret_expires_at"` // 0 = never expires
	ClientName              string   `example:"My Application"                   json:"client_name"`
	RedirectURIs            []string `example:"https://example.com/callback"     json:"redirect_uris"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty"`
	GrantTypes              []string `example:"authorization_code,refresh_token" json:"grant_types"`
	ResponseTypes           []string `example:"code"                             json:"response_types"`
	ApplicationType         string   `example:"web"                              json:"application_type"`
	TokenEndpointAuthMethod string   `example:"client_secret_basic"              json:"token_endpoint_auth_method"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	TosURI                  string   `json:"tos_uri,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
}

// =============================================================================
// TOKEN INTROSPECTION (RFC 7662)
// =============================================================================

// TokenIntrospectionRequest represents a token introspection request (RFC 7662).
type TokenIntrospectionRequest struct {
	Token         string `form:"token"           json:"token"           validate:"required"`
	TokenTypeHint string `form:"token_type_hint" json:"token_type_hint"` // access_token, refresh_token
	ClientID      string `form:"client_id"       json:"client_id"`
	ClientSecret  string `form:"client_secret"   json:"client_secret"`
}

// TokenIntrospectionResponse represents a token introspection response (RFC 7662).
type TokenIntrospectionResponse struct {
	Active    bool     `example:"true"                     json:"active"`
	Scope     string   `example:"openid profile email"     json:"scope,omitempty"`
	ClientID  string   `example:"client_123"               json:"client_id,omitempty"`
	Username  string   `example:"johndoe"                  json:"username,omitempty"`
	TokenType string   `example:"Bearer"                   json:"token_type,omitempty"`
	Exp       int64    `example:"1609459200"               json:"exp,omitempty"`
	Iat       int64    `example:"1609455600"               json:"iat,omitempty"`
	Nbf       int64    `example:"1609455600"               json:"nbf,omitempty"`
	Sub       string   `example:"01HZ..."                  json:"sub,omitempty"`
	Aud       []string `example:"https://api.example.com"  json:"aud,omitempty"`
	Iss       string   `example:"https://auth.example.com" json:"iss,omitempty"`
	Jti       string   `example:"token_01HZ..."            json:"jti,omitempty"`
}

// =============================================================================
// TOKEN REVOCATION (RFC 7009)
// =============================================================================

// TokenRevocationRequest represents a token revocation request (RFC 7009).
type TokenRevocationRequest struct {
	Token         string `form:"token"           json:"token"           validate:"required"`
	TokenTypeHint string `form:"token_type_hint" json:"token_type_hint"` // access_token, refresh_token
	ClientID      string `form:"client_id"       json:"client_id"`
	ClientSecret  string `form:"client_secret"   json:"client_secret"`
}

// =============================================================================
// JWKS ENDPOINT
// =============================================================================

// JWKSResponse represents the JSON Web Key Set response.
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWKResponse is an alias for the JWKS key structure defined in jwks.go.
type JWKResponse = JWK

// =============================================================================
// DISCOVERY ENDPOINT (.well-known/openid-configuration)
// =============================================================================

// DiscoveryResponse represents the OIDC discovery document.
type DiscoveryResponse struct {
	Issuer                                    string   `example:"https://auth.example.com"                             json:"issuer"`
	AuthorizationEndpoint                     string   `example:"https://auth.example.com/oauth2/authorize"            json:"authorization_endpoint"`
	TokenEndpoint                             string   `example:"https://auth.example.com/oauth2/token"                json:"token_endpoint"`
	UserInfoEndpoint                          string   `example:"https://auth.example.com/oauth2/userinfo"             json:"userinfo_endpoint"`
	JwksURI                                   string   `example:"https://auth.example.com/oauth2/jwks"                 json:"jwks_uri"`
	RegistrationEndpoint                      string   `example:"https://auth.example.com/oauth2/register"             json:"registration_endpoint,omitempty"`
	IntrospectionEndpoint                     string   `example:"https://auth.example.com/oauth2/introspect"           json:"introspection_endpoint,omitempty"`
	RevocationEndpoint                        string   `example:"https://auth.example.com/oauth2/revoke"               json:"revocation_endpoint,omitempty"`
	DeviceAuthorizationEndpoint               string   `example:"https://auth.example.com/oauth2/device/authorize"     json:"device_authorization_endpoint,omitempty"`
	ResponseTypesSupported                    []string `example:"code,token,id_token"                                  json:"response_types_supported"`
	ResponseModesSupported                    []string `example:"query,fragment,form_post"                             json:"response_modes_supported,omitempty"`
	GrantTypesSupported                       []string `example:"authorization_code,refresh_token,client_credentials"  json:"grant_types_supported"`
	SubjectTypesSupported                     []string `example:"public"                                               json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported          []string `example:"RS256"                                                json:"id_token_signing_alg_values_supported"`
	ScopesSupported                           []string `example:"openid,profile,email"                                 json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported         []string `example:"client_secret_basic,client_secret_post"               json:"token_endpoint_auth_methods_supported"`
	ClaimsSupported                           []string `example:"sub,name,email,picture"                               json:"claims_supported"`
	CodeChallengeMethodsSupported             []string `example:"S256,plain"                                           json:"code_challenge_methods_supported,omitempty"`
	IntrospectionEndpointAuthMethodsSupported []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`
	RevocationEndpointAuthMethodsSupported    []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`
	RequestParameterSupported                 bool     `json:"request_parameter_supported,omitempty"`
	RequestURIParameterSupported              bool     `json:"request_uri_parameter_supported,omitempty"`
	RequireRequestURIRegistration             bool     `json:"require_request_uri_registration,omitempty"`
	ClaimsParameterSupported                  bool     `json:"claims_parameter_supported,omitempty"`
}

// =============================================================================
// CLIENT MANAGEMENT (Admin Endpoints)
// =============================================================================

// ClientsListResponse represents a list of OAuth clients.
type ClientsListResponse struct {
	Clients    []ClientSummary `json:"clients"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}

// ClientSummary represents a summary of an OAuth client.
type ClientSummary struct {
	ClientID        string `json:"clientID"`
	Name            string `json:"name"`
	ApplicationType string `json:"applicationType"`
	CreatedAt       string `json:"createdAt"`
	IsOrgLevel      bool   `json:"isOrgLevel"`
}

// ClientDetailsResponse represents detailed information about an OAuth client.
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

// ClientUpdateRequest represents a client update request.
type ClientUpdateRequest struct {
	Name                    string   `json:"name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris,omitempty"              validate:"omitempty,dive,url"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty"  validate:"omitempty,dive,url"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	AllowedScopes           []string `json:"allowed_scopes,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty" validate:"omitempty,oneof=client_secret_basic client_secret_post none"`
	RequirePKCE             *bool    `json:"require_pkce,omitempty"`
	RequireConsent          *bool    `json:"require_consent,omitempty"`
	TrustedClient           *bool    `json:"trusted_client,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"                   validate:"omitempty,url"`
	PolicyURI               string   `json:"policy_uri,omitempty"                 validate:"omitempty,url"`
	TosURI                  string   `json:"tos_uri,omitempty"                    validate:"omitempty,url"`
	Contacts                []string `json:"contacts,omitempty"                   validate:"omitempty,dive,email"`
}

// =============================================================================
// ERROR RESPONSES
// =============================================================================

// ErrorResponse is the standard OAuth2/OIDC error response.
//
//nolint:errname // HTTP response DTO, not a Go error type
type ErrorResponse = responses.ErrorResponse

// OAuthErrorResponse represents an OAuth2-specific error response.
type OAuthErrorResponse struct {
	Error            string `example:"invalid_request"                                 json:"error"`
	ErrorDescription string `example:"The request is missing a required parameter"     json:"error_description,omitempty"`
	ErrorURI         string `example:"https://docs.example.com/errors/invalid_request" json:"error_uri,omitempty"`
	State            string `json:"state,omitempty"`
}

// =============================================================================
// INTERNAL TYPES
// =============================================================================

// ScopeInfo represents a scope with its description for consent screens.
type ScopeInfo struct {
	Name        string
	Description string
}

// ConsentDecision represents a user's consent decision.
type ConsentDecision struct {
	Approved bool
	Scopes   []string
}

// ClientAuthResult represents the result of client authentication.
type ClientAuthResult struct {
	ClientID      string
	Authenticated bool
	Method        string // basic, post, none
}

// =============================================================================
// DEVICE FLOW (RFC 8628)
// =============================================================================

// DeviceAuthorizationRequest represents a device authorization request.
type DeviceAuthorizationRequest struct {
	ClientID string `form:"client_id" json:"client_id" validate:"required"`
	Scope    string `form:"scope"     json:"scope"`
}

// DeviceAuthorizationResponse represents the device authorization response.
type DeviceAuthorizationResponse struct {
	DeviceCode              string `example:"GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS"      json:"device_code"`
	UserCode                string `example:"WDJB-MJHT"                                      json:"user_code"`
	VerificationURI         string `example:"https://example.com/device"                     json:"verification_uri"`
	VerificationURIComplete string `example:"https://example.com/device?user_code=WDJB-MJHT" json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `example:"600"                                            json:"expires_in"` // seconds
	Interval                int    `example:"5"                                              json:"interval"`   // polling interval in seconds
}

// DeviceVerificationRequest represents the user code verification request.
type DeviceVerificationRequest struct {
	UserCode string `form:"user_code" json:"user_code" validate:"required"`
}

// DeviceAuthorizationDecisionRequest represents the user's authorization decision.
type DeviceAuthorizationDecisionRequest struct {
	UserCode string `form:"user_code" json:"user_code" validate:"required"`
	Action   string `form:"action"    json:"action"    validate:"required,oneof=approve deny"`
}

// DeviceVerificationInfo contains info to display during verification.
type DeviceVerificationInfo struct {
	UserCode   string
	ClientName string
	Scopes     []ScopeInfo
}

// DeviceCodeEntryResponse is returned for the device code entry endpoint (API mode).
type DeviceCodeEntryResponse struct {
	FormAction  string `example:"/oauth2/device/verify" json:"formAction"`
	Placeholder string `example:"XXXX-XXXX"             json:"placeholder"`
	BasePath    string `example:"/api/identity/oauth2"  json:"basePath"`
}

// DeviceVerifyResponse is returned after verifying a device code (API mode).
type DeviceVerifyResponse struct {
	UserCode          string      `example:"WDJB-MJHT"                    json:"userCode"`
	UserCodeFormatted string      `example:"WDJB-MJHT"                    json:"userCodeFormatted"`
	ClientName        string      `example:"My Application"               json:"clientName"`
	ClientID          string      `example:"client_01HZ..."               json:"clientId"`
	LogoURI           string      `example:"https://example.com/logo.png" json:"logoUri,omitempty"`
	Scopes            []ScopeInfo `json:"scopes"`
	AuthorizeURL      string      `example:"/oauth2/device/authorize"     json:"authorizeUrl"`
}

// DeviceDecisionResponse is returned after the authorization decision (API mode).
type DeviceDecisionResponse struct {
	Success  bool   `example:"true"                           json:"success"`
	Approved bool   `example:"true"                           json:"approved"`
	Message  string `example:"Device authorized successfully" json:"message"`
}
