// Auto-generated jwt plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct JwtPlugin {{
    client: Option<AuthsomeClient>,
}

impl JwtPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct CreateJWTKeyRequest {
        #[serde(rename = "curve")]
        pub curve: String,
        #[serde(rename = "expiresAt")]
        pub expires_at: *time.Time,
        #[serde(rename = "isPlatformKey")]
        pub is_platform_key: bool,
        #[serde(rename = "keyType")]
        pub key_type: String,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "algorithm")]
        pub algorithm: String,
    }

    /// CreateJWTKey creates a new JWT signing key
    pub async fn create_j_w_t_key(
        &self,
        _request: CreateJWTKeyRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct ListJWTKeysRequest {
        #[serde(rename = "", skip_serializing_if = "Option::is_none")]
        pub : Option<*bool>,
    }

    /// ListJWTKeys lists JWT signing keys
    pub async fn list_j_w_t_keys(
        &self,
        _request: ListJWTKeysRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetJWKS returns the JSON Web Key Set
    pub async fn get_j_w_k_s(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct GenerateTokenRequest {
        #[serde(rename = "userId")]
        pub user_id: String,
        #[serde(rename = "audience")]
        pub audience: []string,
        #[serde(rename = "expiresIn")]
        pub expires_in: time.Duration,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "permissions")]
        pub permissions: []string,
        #[serde(rename = "scopes")]
        pub scopes: []string,
        #[serde(rename = "sessionId")]
        pub session_id: String,
        #[serde(rename = "tokenType")]
        pub token_type: String,
    }

    /// GenerateToken generates a new JWT token
    pub async fn generate_token(
        &self,
        _request: GenerateTokenRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyTokenRequest {
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "tokenType")]
        pub token_type: String,
        #[serde(rename = "audience")]
        pub audience: []string,
    }

    /// VerifyToken verifies a JWT token
    pub async fn verify_token(
        &self,
        _request: VerifyTokenRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for JwtPlugin {{
    fn id(&self) -> &str {
        "jwt"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
