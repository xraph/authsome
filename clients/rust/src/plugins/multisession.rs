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

    #[derive(Debug, Serialize)]
    pub struct ListRequest {
        #[serde(rename = "limit")]
        pub limit: i32,
        #[serde(rename = "sortBy")]
        pub sort_by: *string,
        #[serde(rename = "userAgent")]
        pub user_agent: *string,
        #[serde(rename = "createdFrom")]
        pub created_from: *string,
        #[serde(rename = "ipAddress")]
        pub ip_address: *string,
        #[serde(rename = "offset")]
        pub offset: i32,
        #[serde(rename = "sortOrder")]
        pub sort_order: *string,
        #[serde(rename = "active")]
        pub active: *bool,
        #[serde(rename = "createdTo")]
        pub created_to: *string,
    }

    /// List returns sessions for the current user with optional filtering
    pub async fn list(
        &self,
        _request: ListRequest,
    ) -> Result<()> {
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
        pub session: *session.Session,
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
        pub session: *session.Session,
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
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "session")]
        pub session: *session.Session,
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

    #[derive(Debug, Deserialize)]
    pub struct RevokeAllResponse {
        #[serde(rename = "revokedCount")]
        pub revoked_count: i32,
        #[serde(rename = "status")]
        pub status: String,
    }

    /// RevokeAll revokes all sessions for the current user
    pub async fn revoke_all(
        &self,
        _request: RevokeAllRequest,
    ) -> Result<RevokeAllResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct RevokeOthersResponse {
        #[serde(rename = "revokedCount")]
        pub revoked_count: i32,
        #[serde(rename = "status")]
        pub status: String,
    }

    /// RevokeOthers revokes all sessions except the current one
    pub async fn revoke_others(
        &self,
    ) -> Result<RevokeOthersResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct RefreshResponse {
        #[serde(rename = "session")]
        pub session: *session.Session,
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

    #[derive(Debug, Deserialize)]
    pub struct GetStatsResponse {
        #[serde(rename = "deviceCount")]
        pub device_count: i32,
        #[serde(rename = "locationCount")]
        pub location_count: i32,
        #[serde(rename = "newestSession")]
        pub newest_session: *string,
        #[serde(rename = "oldestSession")]
        pub oldest_session: *string,
        #[serde(rename = "totalSessions")]
        pub total_sessions: i32,
        #[serde(rename = "activeSessions")]
        pub active_sessions: i32,
    }

    /// GetStats returns aggregated session statistics for the current user
    pub async fn get_stats(
        &self,
    ) -> Result<GetStatsResponse> {{
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
