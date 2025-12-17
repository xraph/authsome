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

    #[derive(Debug, Deserialize)]
    pub struct GetCurrentResponse {
        #[serde(rename = "session")]
        pub session: ,
        #[serde(rename = "token")]
        pub token: String,
    }

    /// GetCurrent returns details about the currently active session
    pub async fn get_current(
        &self,
    ) -> Result<GetCurrentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetByIDResponse {
        #[serde(rename = "session")]
        pub session: ,
        #[serde(rename = "token")]
        pub token: String,
    }

    /// GetByID returns details about a specific session by ID
    pub async fn get_by_i_d(
        &self,
    ) -> Result<GetByIDResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RevokeAllRequest {
        #[serde(rename = "includeCurrentSession")]
        pub include_current_session: bool,
    }

    /// RevokeAll revokes all sessions for the current user
    pub async fn revoke_all(
        &self,
        _request: RevokeAllRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// RevokeOthers revokes all sessions except the current one
    pub async fn revoke_others(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct RefreshResponse {
        #[serde(rename = "session")]
        pub session: ,
        #[serde(rename = "token")]
        pub token: String,
    }

    /// Refresh extends the current session's expiry time
    pub async fn refresh(
        &self,
    ) -> Result<RefreshResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetStats returns aggregated session statistics for the current user
    pub async fn get_stats(
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
