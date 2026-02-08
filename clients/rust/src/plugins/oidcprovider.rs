// Auto-generated oidcprovider plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct OidcproviderPlugin {{
    client: Option<AuthsomeClient>,
}

impl OidcproviderPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct RegisterClientRequest {
        #[serde(rename = "require_pkce")]
        pub require_pkce: bool,
        #[serde(rename = "tos_uri")]
        pub tos_uri: String,
        #[serde(rename = "trusted_client")]
        pub trusted_client: bool,
        #[serde(rename = "grant_types")]
        pub grant_types: []string,
        #[serde(rename = "policy_uri")]
        pub policy_uri: String,
        #[serde(rename = "post_logout_redirect_uris")]
        pub post_logout_redirect_uris: []string,
        #[serde(rename = "redirect_uris")]
        pub redirect_uris: []string,
        #[serde(rename = "response_types")]
        pub response_types: []string,
        #[serde(rename = "client_name")]
        pub client_name: String,
        #[serde(rename = "scope")]
        pub scope: String,
        #[serde(rename = "token_endpoint_auth_method")]
        pub token_endpoint_auth_method: String,
        #[serde(rename = "contacts")]
        pub contacts: []string,
        #[serde(rename = "logo_uri")]
        pub logo_uri: String,
        #[serde(rename = "require_consent")]
        pub require_consent: bool,
        #[serde(rename = "application_type")]
        pub application_type: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct RegisterClientResponse {
        #[serde(rename = "post_logout_redirect_uris")]
        pub post_logout_redirect_uris: []string,
        #[serde(rename = "token_endpoint_auth_method")]
        pub token_endpoint_auth_method: String,
        #[serde(rename = "grant_types")]
        pub grant_types: []string,
        #[serde(rename = "logo_uri")]
        pub logo_uri: String,
        #[serde(rename = "redirect_uris")]
        pub redirect_uris: []string,
        #[serde(rename = "scope")]
        pub scope: String,
        #[serde(rename = "tos_uri")]
        pub tos_uri: String,
        #[serde(rename = "application_type")]
        pub application_type: String,
        #[serde(rename = "contacts")]
        pub contacts: []string,
        #[serde(rename = "response_types")]
        pub response_types: []string,
        #[serde(rename = "client_id_issued_at")]
        pub client_id_issued_at: i64,
        #[serde(rename = "policy_uri")]
        pub policy_uri: String,
        #[serde(rename = "client_id")]
        pub client_id: String,
        #[serde(rename = "client_name")]
        pub client_name: String,
        #[serde(rename = "client_secret")]
        pub client_secret: String,
        #[serde(rename = "client_secret_expires_at")]
        pub client_secret_expires_at: i64,
    }

    /// RegisterClient handles dynamic client registration (admin only)
    pub async fn register_client(
        &self,
        _request: RegisterClientRequest,
    ) -> Result<RegisterClientResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListClientsResponse {
        #[serde(rename = "clients")]
        pub clients: []ClientSummary,
        #[serde(rename = "page")]
        pub page: i32,
        #[serde(rename = "pageSize")]
        pub page_size: i32,
        #[serde(rename = "total")]
        pub total: i32,
        #[serde(rename = "totalPages")]
        pub total_pages: i32,
    }

    /// ListClients lists all OAuth clients for the current app/env/org
    pub async fn list_clients(
        &self,
    ) -> Result<ListClientsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetClientResponse {
        #[serde(rename = "requirePKCE")]
        pub require_p_k_c_e: bool,
        #[serde(rename = "tokenEndpointAuthMethod")]
        pub token_endpoint_auth_method: String,
        #[serde(rename = "createdAt")]
        pub created_at: String,
        #[serde(rename = "organizationID")]
        pub organization_i_d: String,
        #[serde(rename = "grantTypes")]
        pub grant_types: []string,
        #[serde(rename = "logoURI")]
        pub logo_u_r_i: String,
        #[serde(rename = "responseTypes")]
        pub response_types: []string,
        #[serde(rename = "tosURI")]
        pub tos_u_r_i: String,
        #[serde(rename = "trustedClient")]
        pub trusted_client: bool,
        #[serde(rename = "updatedAt")]
        pub updated_at: String,
        #[serde(rename = "allowedScopes")]
        pub allowed_scopes: []string,
        #[serde(rename = "applicationType")]
        pub application_type: String,
        #[serde(rename = "clientID")]
        pub client_i_d: String,
        #[serde(rename = "isOrgLevel")]
        pub is_org_level: bool,
        #[serde(rename = "postLogoutRedirectURIs")]
        pub post_logout_redirect_u_r_is: []string,
        #[serde(rename = "redirectURIs")]
        pub redirect_u_r_is: []string,
        #[serde(rename = "requireConsent")]
        pub require_consent: bool,
        #[serde(rename = "contacts")]
        pub contacts: []string,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "policyURI")]
        pub policy_u_r_i: String,
    }

    /// GetClient retrieves detailed information about an OAuth client
    pub async fn get_client(
        &self,
    ) -> Result<GetClientResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdateClientRequest {
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "policy_uri")]
        pub policy_uri: String,
        #[serde(rename = "post_logout_redirect_uris")]
        pub post_logout_redirect_uris: []string,
        #[serde(rename = "redirect_uris")]
        pub redirect_uris: []string,
        #[serde(rename = "require_consent")]
        pub require_consent: *bool,
        #[serde(rename = "allowed_scopes")]
        pub allowed_scopes: []string,
        #[serde(rename = "require_pkce")]
        pub require_pkce: *bool,
        #[serde(rename = "response_types")]
        pub response_types: []string,
        #[serde(rename = "token_endpoint_auth_method")]
        pub token_endpoint_auth_method: String,
        #[serde(rename = "tos_uri")]
        pub tos_uri: String,
        #[serde(rename = "trusted_client")]
        pub trusted_client: *bool,
        #[serde(rename = "contacts")]
        pub contacts: []string,
        #[serde(rename = "grant_types")]
        pub grant_types: []string,
        #[serde(rename = "logo_uri")]
        pub logo_uri: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdateClientResponse {
        #[serde(rename = "updatedAt")]
        pub updated_at: String,
        #[serde(rename = "clientID")]
        pub client_i_d: String,
        #[serde(rename = "contacts")]
        pub contacts: []string,
        #[serde(rename = "isOrgLevel")]
        pub is_org_level: bool,
        #[serde(rename = "tosURI")]
        pub tos_u_r_i: String,
        #[serde(rename = "trustedClient")]
        pub trusted_client: bool,
        #[serde(rename = "allowedScopes")]
        pub allowed_scopes: []string,
        #[serde(rename = "organizationID")]
        pub organization_i_d: String,
        #[serde(rename = "requireConsent")]
        pub require_consent: bool,
        #[serde(rename = "requirePKCE")]
        pub require_p_k_c_e: bool,
        #[serde(rename = "tokenEndpointAuthMethod")]
        pub token_endpoint_auth_method: String,
        #[serde(rename = "applicationType")]
        pub application_type: String,
        #[serde(rename = "createdAt")]
        pub created_at: String,
        #[serde(rename = "grantTypes")]
        pub grant_types: []string,
        #[serde(rename = "logoURI")]
        pub logo_u_r_i: String,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "policyURI")]
        pub policy_u_r_i: String,
        #[serde(rename = "postLogoutRedirectURIs")]
        pub post_logout_redirect_u_r_is: []string,
        #[serde(rename = "responseTypes")]
        pub response_types: []string,
        #[serde(rename = "redirectURIs")]
        pub redirect_u_r_is: []string,
    }

    /// UpdateClient updates an existing OAuth client
    pub async fn update_client(
        &self,
        _request: UpdateClientRequest,
    ) -> Result<UpdateClientResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// DeleteClient deletes an OAuth client
    pub async fn delete_client(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DiscoveryResponse {
        #[serde(rename = "require_request_uri_registration")]
        pub require_request_uri_registration: bool,
        #[serde(rename = "response_modes_supported")]
        pub response_modes_supported: []string,
        #[serde(rename = "scopes_supported")]
        pub scopes_supported: []string,
        #[serde(rename = "subject_types_supported")]
        pub subject_types_supported: []string,
        #[serde(rename = "claims_supported")]
        pub claims_supported: []string,
        #[serde(rename = "device_authorization_endpoint")]
        pub device_authorization_endpoint: String,
        #[serde(rename = "jwks_uri")]
        pub jwks_uri: String,
        #[serde(rename = "response_types_supported")]
        pub response_types_supported: []string,
        #[serde(rename = "revocation_endpoint")]
        pub revocation_endpoint: String,
        #[serde(rename = "userinfo_endpoint")]
        pub userinfo_endpoint: String,
        #[serde(rename = "grant_types_supported")]
        pub grant_types_supported: []string,
        #[serde(rename = "id_token_signing_alg_values_supported")]
        pub id_token_signing_alg_values_supported: []string,
        #[serde(rename = "issuer")]
        pub issuer: String,
        #[serde(rename = "request_uri_parameter_supported")]
        pub request_uri_parameter_supported: bool,
        #[serde(rename = "revocation_endpoint_auth_methods_supported")]
        pub revocation_endpoint_auth_methods_supported: []string,
        #[serde(rename = "token_endpoint")]
        pub token_endpoint: String,
        #[serde(rename = "authorization_endpoint")]
        pub authorization_endpoint: String,
        #[serde(rename = "claims_parameter_supported")]
        pub claims_parameter_supported: bool,
        #[serde(rename = "code_challenge_methods_supported")]
        pub code_challenge_methods_supported: []string,
        #[serde(rename = "introspection_endpoint")]
        pub introspection_endpoint: String,
        #[serde(rename = "token_endpoint_auth_methods_supported")]
        pub token_endpoint_auth_methods_supported: []string,
        #[serde(rename = "introspection_endpoint_auth_methods_supported")]
        pub introspection_endpoint_auth_methods_supported: []string,
        #[serde(rename = "registration_endpoint")]
        pub registration_endpoint: String,
        #[serde(rename = "request_parameter_supported")]
        pub request_parameter_supported: bool,
    }

    /// Discovery handles the OIDC discovery endpoint (.well-known/openid-configuration)
    pub async fn discovery(
        &self,
    ) -> Result<DiscoveryResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct JWKSResponse {
        #[serde(rename = "keys")]
        pub keys: []JWK,
    }

    /// JWKS returns the JSON Web Key Set
    pub async fn j_w_k_s(
        &self,
    ) -> Result<JWKSResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// Authorize handles OAuth2/OIDC authorization requests
    pub async fn authorize(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct HandleConsentRequest {
        #[serde(rename = "response_type")]
        pub response_type: String,
        #[serde(rename = "scope")]
        pub scope: String,
        #[serde(rename = "state")]
        pub state: String,
        #[serde(rename = "action")]
        pub action: String,
        #[serde(rename = "client_id")]
        pub client_id: String,
        #[serde(rename = "code_challenge")]
        pub code_challenge: String,
        #[serde(rename = "code_challenge_method")]
        pub code_challenge_method: String,
        #[serde(rename = "redirect_uri")]
        pub redirect_uri: String,
    }

    /// HandleConsent processes the consent form submission
    pub async fn handle_consent(
        &self,
        _request: HandleConsentRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct TokenRequest {
        #[serde(rename = "redirect_uri")]
        pub redirect_uri: String,
        #[serde(rename = "scope")]
        pub scope: String,
        #[serde(rename = "client_secret")]
        pub client_secret: String,
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "code_verifier")]
        pub code_verifier: String,
        #[serde(rename = "device_code")]
        pub device_code: String,
        #[serde(rename = "refresh_token")]
        pub refresh_token: String,
        #[serde(rename = "audience")]
        pub audience: String,
        #[serde(rename = "client_id")]
        pub client_id: String,
        #[serde(rename = "grant_type")]
        pub grant_type: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct TokenResponse {
        #[serde(rename = "id_token")]
        pub id_token: String,
        #[serde(rename = "refresh_token")]
        pub refresh_token: String,
        #[serde(rename = "scope")]
        pub scope: String,
        #[serde(rename = "token_type")]
        pub token_type: String,
        #[serde(rename = "access_token")]
        pub access_token: String,
        #[serde(rename = "expires_in")]
        pub expires_in: i32,
    }

    /// Token handles the token endpoint
    pub async fn token(
        &self,
        _request: TokenRequest,
    ) -> Result<TokenResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct UserInfoResponse {
        #[serde(rename = "family_name")]
        pub family_name: String,
        #[serde(rename = "nickname")]
        pub nickname: String,
        #[serde(rename = "phone_number")]
        pub phone_number: String,
        #[serde(rename = "phone_number_verified")]
        pub phone_number_verified: bool,
        #[serde(rename = "picture")]
        pub picture: String,
        #[serde(rename = "sub")]
        pub sub: String,
        #[serde(rename = "updated_at")]
        pub updated_at: i64,
        #[serde(rename = "given_name")]
        pub given_name: String,
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "email_verified")]
        pub email_verified: bool,
        #[serde(rename = "gender")]
        pub gender: String,
        #[serde(rename = "locale")]
        pub locale: String,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "website")]
        pub website: String,
        #[serde(rename = "birthdate")]
        pub birthdate: String,
        #[serde(rename = "middle_name")]
        pub middle_name: String,
        #[serde(rename = "preferred_username")]
        pub preferred_username: String,
        #[serde(rename = "profile")]
        pub profile: String,
        #[serde(rename = "zoneinfo")]
        pub zoneinfo: String,
    }

    /// UserInfo returns user information based on the access token
    pub async fn user_info(
        &self,
    ) -> Result<UserInfoResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct IntrospectTokenRequest {
        #[serde(rename = "client_id")]
        pub client_id: String,
        #[serde(rename = "client_secret")]
        pub client_secret: String,
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "token_type_hint")]
        pub token_type_hint: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct IntrospectTokenResponse {
        #[serde(rename = "username")]
        pub username: String,
        #[serde(rename = "active")]
        pub active: bool,
        #[serde(rename = "aud")]
        pub aud: []string,
        #[serde(rename = "client_id")]
        pub client_id: String,
        #[serde(rename = "exp")]
        pub exp: i64,
        #[serde(rename = "iat")]
        pub iat: i64,
        #[serde(rename = "iss")]
        pub iss: String,
        #[serde(rename = "jti")]
        pub jti: String,
        #[serde(rename = "scope")]
        pub scope: String,
        #[serde(rename = "nbf")]
        pub nbf: i64,
        #[serde(rename = "sub")]
        pub sub: String,
        #[serde(rename = "token_type")]
        pub token_type: String,
    }

    /// IntrospectToken handles token introspection requests
    pub async fn introspect_token(
        &self,
        _request: IntrospectTokenRequest,
    ) -> Result<IntrospectTokenResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RevokeTokenRequest {
        #[serde(rename = "client_id")]
        pub client_id: String,
        #[serde(rename = "client_secret")]
        pub client_secret: String,
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "token_type_hint")]
        pub token_type_hint: String,
    }

    /// RevokeToken handles token revocation requests
    pub async fn revoke_token(
        &self,
        _request: RevokeTokenRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct DeviceAuthorizeRequest {
        #[serde(rename = "scope")]
        pub scope: String,
        #[serde(rename = "client_id")]
        pub client_id: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct DeviceAuthorizeResponse {
        #[serde(rename = "user_code")]
        pub user_code: String,
        #[serde(rename = "verification_uri")]
        pub verification_uri: String,
        #[serde(rename = "verification_uri_complete")]
        pub verification_uri_complete: String,
        #[serde(rename = "device_code")]
        pub device_code: String,
        #[serde(rename = "expires_in")]
        pub expires_in: i32,
        #[serde(rename = "interval")]
        pub interval: i32,
    }

    /// DeviceAuthorize initiates the device authorization flow
    pub async fn device_authorize(
        &self,
        _request: DeviceAuthorizeRequest,
    ) -> Result<DeviceAuthorizeResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DeviceCodeEntryResponse {
        #[serde(rename = "basePath")]
        pub base_path: String,
        #[serde(rename = "formAction")]
        pub form_action: String,
        #[serde(rename = "placeholder")]
        pub placeholder: String,
    }

    /// DeviceCodeEntry shows the device code entry form
    pub async fn device_code_entry(
        &self,
    ) -> Result<DeviceCodeEntryResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct DeviceVerifyRequest {
        #[serde(rename = "user_code")]
        pub user_code: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct DeviceVerifyResponse {
        #[serde(rename = "authorizeUrl")]
        pub authorize_url: String,
        #[serde(rename = "clientId")]
        pub client_id: String,
        #[serde(rename = "clientName")]
        pub client_name: String,
        #[serde(rename = "logoUri")]
        pub logo_uri: String,
        #[serde(rename = "scopes")]
        pub scopes: []ScopeInfo,
        #[serde(rename = "userCode")]
        pub user_code: String,
        #[serde(rename = "userCodeFormatted")]
        pub user_code_formatted: String,
    }

    /// DeviceVerify verifies the user code and shows the consent screen
    pub async fn device_verify(
        &self,
        _request: DeviceVerifyRequest,
    ) -> Result<DeviceVerifyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct DeviceAuthorizeDecisionRequest {
        #[serde(rename = "action")]
        pub action: String,
        #[serde(rename = "user_code")]
        pub user_code: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct DeviceAuthorizeDecisionResponse {
        #[serde(rename = "approved")]
        pub approved: bool,
        #[serde(rename = "message")]
        pub message: String,
        #[serde(rename = "success")]
        pub success: bool,
    }

    /// DeviceAuthorizeDecision handles the user's authorization decision
    pub async fn device_authorize_decision(
        &self,
        _request: DeviceAuthorizeDecisionRequest,
    ) -> Result<DeviceAuthorizeDecisionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for OidcproviderPlugin {{
    fn id(&self) -> &str {
        "oidcprovider"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
