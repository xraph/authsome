// Auto-generated anonymous plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct AnonymousPlugin {{
    client: Option<AuthsomeClient>,
}

impl AnonymousPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Deserialize)]
    pub struct SignInResponse {
        #[serde(rename = "session")]
        pub session: ,
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "user")]
        pub user: ,
    }

    /// SignIn creates a guest user and session
    pub async fn sign_in(
        &self,
    ) -> Result<SignInResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct LinkRequest {
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "password")]
        pub password: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct LinkResponse {
        #[serde(rename = "message")]
        pub message: String,
        #[serde(rename = "user")]
        pub user: ,
    }

    /// Link upgrades an anonymous session to a real account
    pub async fn link(
        &self,
        _request: LinkRequest,
    ) -> Result<LinkResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for AnonymousPlugin {{
    fn id(&self) -> &str {
        "anonymous"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
