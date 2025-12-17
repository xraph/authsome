// Auto-generated emailverification plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct EmailverificationPlugin {{
    client: Option<AuthsomeClient>,
}

impl EmailverificationPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct SendRequest {
        #[serde(rename = "email")]
        pub email: String,
    }

    /// Send handles manual verification email sending
POST /email-verification/send
    pub async fn send(
        &self,
        _request: SendRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// Verify handles email verification via token
GET /email-verification/verify?token=xyz
    pub async fn verify(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct ResendRequest {
        #[serde(rename = "email")]
        pub email: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct ResendResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// Resend handles resending verification email
POST /email-verification/resend
    pub async fn resend(
        &self,
        _request: ResendRequest,
    ) -> Result<ResendResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for EmailverificationPlugin {{
    fn id(&self) -> &str {
        "emailverification"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
