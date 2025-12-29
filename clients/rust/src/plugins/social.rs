// Auto-generated social plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct SocialPlugin {{
    client: Option<AuthsomeClient>,
}

impl SocialPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct SignInRequest {
        #[serde(rename = "scopes")]
        pub scopes: []string,
        #[serde(rename = "provider")]
        pub provider: String,
        #[serde(rename = "redirectUrl")]
        pub redirect_url: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct SignInResponse {
        #[serde(rename = "url")]
        pub url: String,
    }

    /// SignIn initiates OAuth flow for sign-in
POST /api/auth/signin/social
    pub async fn sign_in(
        &self,
        _request: SignInRequest,
    ) -> Result<SignInResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct CallbackResponse {
        #[serde(rename = "user")]
        pub user: *schema.User,
        #[serde(rename = "action")]
        pub action: String,
        #[serde(rename = "isNewUser")]
        pub is_new_user: bool,
    }

    /// Callback handles OAuth provider callback
GET /api/auth/callback/:provider
    pub async fn callback(
        &self,
    ) -> Result<CallbackResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct LinkAccountRequest {
        #[serde(rename = "provider")]
        pub provider: String,
        #[serde(rename = "scopes")]
        pub scopes: []string,
    }

    #[derive(Debug, Deserialize)]
    pub struct LinkAccountResponse {
        #[serde(rename = "url")]
        pub url: String,
    }

    /// LinkAccount links a social provider to the current user
POST /api/auth/account/link
    pub async fn link_account(
        &self,
        _request: LinkAccountRequest,
    ) -> Result<LinkAccountResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// UnlinkAccount unlinks a social provider from the current user
DELETE /api/auth/account/unlink/:provider
    pub async fn unlink_account(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListProvidersResponse {
        #[serde(rename = "providers")]
        pub providers: []string,
    }

    /// ListProviders returns available OAuth providers
GET /api/auth/providers
    pub async fn list_providers(
        &self,
    ) -> Result<ListProvidersResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for SocialPlugin {{
    fn id(&self) -> &str {
        "social"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
