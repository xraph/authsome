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
        #[serde(rename = "allowed_ips")]
        pub allowed_ips: []string,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "permissions")]
        pub permissions: ,
        #[serde(rename = "rate_limit")]
        pub rate_limit: i32,
        #[serde(rename = "scopes")]
        pub scopes: []string,
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

    #[derive(Debug, Serialize)]
    pub struct ListAPIKeysRequest {
        #[serde(rename = "", skip_serializing_if = "Option::is_none")]
        pub : Option<*bool>,
    }

    /// ListAPIKeys handles GET /api-keys
    pub async fn list_a_p_i_keys(
        &self,
        _request: ListAPIKeysRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct GetAPIKeyRequest {
        #[serde(rename = "", skip_serializing_if = "Option::is_none")]
        pub : Option<String>,
    }

    /// GetAPIKey handles GET /api-keys/:id
    pub async fn get_a_p_i_key(
        &self,
        _request: GetAPIKeyRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdateAPIKeyRequest {
        #[serde(rename = "", skip_serializing_if = "Option::is_none")]
        pub : Option<String>,
        #[serde(rename = "allowed_ips")]
        pub allowed_ips: []string,
        #[serde(rename = "description")]
        pub description: *string,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: *string,
        #[serde(rename = "permissions")]
        pub permissions: ,
        #[serde(rename = "rate_limit")]
        pub rate_limit: *int,
        #[serde(rename = "scopes")]
        pub scopes: []string,
    }

    /// UpdateAPIKey handles PATCH /api-keys/:id
    pub async fn update_a_p_i_key(
        &self,
        _request: UpdateAPIKeyRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct DeleteAPIKeyRequest {
        #[serde(rename = "", skip_serializing_if = "Option::is_none")]
        pub : Option<String>,
    }

    /// DeleteAPIKey handles DELETE /api-keys/:id
    pub async fn delete_a_p_i_key(
        &self,
        _request: DeleteAPIKeyRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RotateAPIKeyRequest {
        #[serde(rename = "", skip_serializing_if = "Option::is_none")]
        pub : Option<String>,
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
        _request: RotateAPIKeyRequest,
    ) -> Result<RotateAPIKeyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyAPIKeyRequest {
        #[serde(rename = "key")]
        pub key: String,
    }

    /// VerifyAPIKey handles POST /api-keys/verify
    pub async fn verify_a_p_i_key(
        &self,
        _request: VerifyAPIKeyRequest,
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
