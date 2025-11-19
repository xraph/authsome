// Auto-generated admin plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct AdminPlugin {{
    client: Option<AuthsomeClient>,
}

impl AdminPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct CreateUserRequest {
        #[serde(rename = "password", skip_serializing_if = "Option::is_none")]
        pub password: Option<String>,
        #[serde(rename = "role", skip_serializing_if = "Option::is_none")]
        pub role: Option<String>,
        #[serde(rename = "username", skip_serializing_if = "Option::is_none")]
        pub username: Option<String>,
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "email_verified")]
        pub email_verified: bool,
        #[serde(rename = "metadata", skip_serializing_if = "Option::is_none")]
        pub metadata: Option<>,
        #[serde(rename = "name", skip_serializing_if = "Option::is_none")]
        pub name: Option<String>,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateUserResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// CreateUser handles POST /admin/users
    pub async fn create_user(
        &self,
        _request: CreateUserRequest,
    ) -> Result<CreateUserResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListUsersResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// ListUsers handles GET /admin/users
    pub async fn list_users(
        &self,
    ) -> Result<ListUsersResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DeleteUserResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// DeleteUser handles DELETE /admin/users/:id
    pub async fn delete_user(
        &self,
    ) -> Result<DeleteUserResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct BanUserRequest {
        #[serde(rename = "expires_at", skip_serializing_if = "Option::is_none")]
        pub expires_at: Option<*time.Time>,
        #[serde(rename = "reason")]
        pub reason: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct BanUserResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// BanUser handles POST /admin/users/:id/ban
    pub async fn ban_user(
        &self,
        _request: BanUserRequest,
    ) -> Result<BanUserResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UnbanUserRequest {
        #[serde(rename = "reason", skip_serializing_if = "Option::is_none")]
        pub reason: Option<String>,
    }

    #[derive(Debug, Deserialize)]
    pub struct UnbanUserResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// UnbanUser handles POST /admin/users/:id/unban
    pub async fn unban_user(
        &self,
        _request: UnbanUserRequest,
    ) -> Result<UnbanUserResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct ImpersonateUserRequest {
        #[serde(rename = "duration", skip_serializing_if = "Option::is_none")]
        pub duration: Option<time.Duration>,
    }

    #[derive(Debug, Deserialize)]
    pub struct ImpersonateUserResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// ImpersonateUser handles POST /admin/users/:id/impersonate
    pub async fn impersonate_user(
        &self,
        _request: ImpersonateUserRequest,
    ) -> Result<ImpersonateUserResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct SetUserRoleRequest {
        #[serde(rename = "role")]
        pub role: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct SetUserRoleResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// SetUserRole handles POST /admin/users/:id/role
    pub async fn set_user_role(
        &self,
        _request: SetUserRoleRequest,
    ) -> Result<SetUserRoleResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListSessionsResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// ListSessions handles GET /admin/sessions
    pub async fn list_sessions(
        &self,
    ) -> Result<ListSessionsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct RevokeSessionResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// RevokeSession handles DELETE /admin/sessions/:id
    pub async fn revoke_session(
        &self,
    ) -> Result<RevokeSessionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetStatsResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// GetStats handles GET /admin/stats
    pub async fn get_stats(
        &self,
    ) -> Result<GetStatsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetAuditLogsResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// GetAuditLogs handles GET /admin/audit
    pub async fn get_audit_logs(
        &self,
    ) -> Result<GetAuditLogsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for AdminPlugin {{
    fn id(&self) -> &str {
        "admin"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
