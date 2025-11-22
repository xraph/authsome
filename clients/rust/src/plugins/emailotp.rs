// Auto-generated emailotp plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct EmailotpPlugin {{
    client: Option<AuthsomeClient>,
}

impl EmailotpPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct SendRequest {
        #[serde(rename = "email")]
        pub email: String,
    }

    /// Send handles sending of OTP to email
    pub async fn send(
        &self,
        _request: SendRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyRequest {
        #[serde(rename = "otp")]
        pub otp: String,
        #[serde(rename = "remember")]
        pub remember: bool,
        #[serde(rename = "email")]
        pub email: String,
    }

    /// Verify checks the OTP and creates a session on success
    pub async fn verify(
        &self,
        _request: VerifyRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for EmailotpPlugin {{
    fn id(&self) -> &str {
        "emailotp"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
