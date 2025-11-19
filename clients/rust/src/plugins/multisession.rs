// Auto-generated multisession plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct MultisessionPlugin {{
    client: Option<AuthsomeClient>,
}

impl MultisessionPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Deserialize)]
    pub struct ListResponse {
        #[serde(rename = "sessions")]
        pub sessions: ,
    }

    /// List returns sessions for the current user based on cookie
    pub async fn list(
        &self,
    ) -> Result<ListResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct SetActiveRequest {
        #[serde(rename = "id")]
        pub id: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct SetActiveResponse {
        #[serde(rename = "session")]
        pub session: ,
        #[serde(rename = "token")]
        pub token: String,
    }

    /// SetActive switches the current session cookie to the provided session id
    pub async fn set_active(
        &self,
        _request: SetActiveRequest,
    ) -> Result<SetActiveResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// Delete revokes a session by id for the current user
    pub async fn delete(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for MultisessionPlugin {{
    fn id(&self) -> &str {
        "multisession"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
