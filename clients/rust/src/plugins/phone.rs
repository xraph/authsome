// Auto-generated phone plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct PhonePlugin {{
    client: Option<AuthsomeClient>,
}

impl PhonePlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct SendCodeRequest {
        #[serde(rename = "phone")]
        pub phone: String,
    }

    /// SendCode handles sending of verification code via SMS
    pub async fn send_code(
        &self,
        _request: SendCodeRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyRequest {
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "phone")]
        pub phone: String,
        #[serde(rename = "remember")]
        pub remember: bool,
    }

    #[derive(Debug, Deserialize)]
    pub struct VerifyResponse {
        #[serde(rename = "session")]
        pub session: *session.Session,
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "user")]
        pub user: *user.User,
    }

    /// Verify checks the code and creates a session on success
    pub async fn verify(
        &self,
        _request: VerifyRequest,
    ) -> Result<VerifyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct SignInRequest {
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "phone")]
        pub phone: String,
        #[serde(rename = "remember")]
        pub remember: bool,
        #[serde(rename = "code")]
        pub code: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct SignInResponse {
        #[serde(rename = "session")]
        pub session: *session.Session,
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "user")]
        pub user: *user.User,
    }

    /// SignIn aliases to Verify for convenience
    pub async fn sign_in(
        &self,
        _request: SignInRequest,
    ) -> Result<SignInResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for PhonePlugin {{
    fn id(&self) -> &str {
        "phone"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
