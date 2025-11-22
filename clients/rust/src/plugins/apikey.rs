// Auto-generated apikey plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct ApikeyPlugin {{
    client: Option<AuthsomeClient>,
}

impl ApikeyPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct CreateAPIKeyRequest {
        #[serde(rename = "permissions", skip_serializing_if = "Option::is_none")]
        pub permissions: Option<>,
        #[serde(rename = "rate_limit", skip_serializing_if = "Option::is_none")]
        pub rate_limit: Option<i32>,
        #[serde(rename = "scopes")]
        pub scopes: []string,
        #[serde(rename = "allowed_ips", skip_serializing_if = "Option::is_none")]
        pub allowed_ips: Option<[]string>,
        #[serde(rename = "description", skip_serializing_if = "Option::is_none")]
        pub description: Option<String>,
        #[serde(rename = "metadata", skip_serializing_if = "Option::is_none")]
        pub metadata: Option<>,
        #[serde(rename = "name")]
        pub name: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateAPIKeyResponse {
        #[serde(rename = "api_key")]
        pub api_key: *apikey.APIKey,
        #[serde(rename = "message")]
        pub message: String,
    }

    /// CreateAPIKey handles POST /api-keys
    pub async fn create_a_p_i_key(
        &self,
        _request: CreateAPIKeyRequest,
    ) -> Result<CreateAPIKeyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// ListAPIKeys handles GET /api-keys
    pub async fn list_a_p_i_keys(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetAPIKey handles GET /api-keys/:id
    pub async fn get_a_p_i_key(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// UpdateAPIKey handles PATCH /api-keys/:id
    pub async fn update_a_p_i_key(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// DeleteAPIKey handles DELETE /api-keys/:id
    pub async fn delete_a_p_i_key(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct RotateAPIKeyResponse {
        #[serde(rename = "api_key")]
        pub api_key: *apikey.APIKey,
        #[serde(rename = "message")]
        pub message: String,
    }

    /// RotateAPIKey handles POST /api-keys/:id/rotate
    pub async fn rotate_a_p_i_key(
        &self,
    ) -> Result<RotateAPIKeyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// VerifyAPIKey handles POST /api-keys/verify
    pub async fn verify_a_p_i_key(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for ApikeyPlugin {{
    fn id(&self) -> &str {
        "apikey"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
