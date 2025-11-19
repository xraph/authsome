// Auto-generated idverification plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct IdverificationPlugin {{
    client: Option<AuthsomeClient>,
}

impl IdverificationPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct CreateVerificationSessionRequest {
        #[serde(rename = "cancelUrl")]
        pub cancel_url: String,
        #[serde(rename = "config")]
        pub config: ,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "provider")]
        pub provider: String,
        #[serde(rename = "requiredChecks")]
        pub required_checks: []string,
        #[serde(rename = "successUrl")]
        pub success_url: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateVerificationSessionResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// CreateVerificationSession creates a new verification session
POST /verification/sessions
    pub async fn create_verification_session(
        &self,
        _request: CreateVerificationSessionRequest,
    ) -> Result<CreateVerificationSessionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetVerificationSessionResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// GetVerificationSession retrieves a verification session
GET /verification/sessions/:id
    pub async fn get_verification_session(
        &self,
    ) -> Result<GetVerificationSessionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetVerificationResponse {
        #[serde(rename = "verification")]
        pub verification: ,
    }

    /// GetVerification retrieves a verification by ID
GET /verification/:id
    pub async fn get_verification(
        &self,
    ) -> Result<GetVerificationResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetUserVerificationsResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// GetUserVerifications retrieves all verifications for the current user
GET /verification/me
    pub async fn get_user_verifications(
        &self,
    ) -> Result<GetUserVerificationsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetUserVerificationStatusResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// GetUserVerificationStatus retrieves the verification status for the current user
GET /verification/me/status
    pub async fn get_user_verification_status(
        &self,
    ) -> Result<GetUserVerificationStatusResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RequestReverificationRequest {
        #[serde(rename = "reason")]
        pub reason: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct RequestReverificationResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// RequestReverification requests re-verification for the current user
POST /verification/me/reverify
    pub async fn request_reverification(
        &self,
        _request: RequestReverificationRequest,
    ) -> Result<RequestReverificationResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct HandleWebhookResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// HandleWebhook handles provider webhook callbacks
POST /verification/webhook/:provider
    pub async fn handle_webhook(
        &self,
    ) -> Result<HandleWebhookResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct AdminBlockUserRequest {
        #[serde(rename = "reason")]
        pub reason: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct AdminBlockUserResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// AdminBlockUser blocks a user from verification (admin only)
POST /verification/admin/users/:userId/block
    pub async fn admin_block_user(
        &self,
        _request: AdminBlockUserRequest,
    ) -> Result<AdminBlockUserResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct AdminUnblockUserResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// AdminUnblockUser unblocks a user (admin only)
POST /verification/admin/users/:userId/unblock
    pub async fn admin_unblock_user(
        &self,
    ) -> Result<AdminUnblockUserResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct AdminGetUserVerificationStatusResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// AdminGetUserVerificationStatus retrieves verification status for any user (admin only)
GET /verification/admin/users/:userId/status
    pub async fn admin_get_user_verification_status(
        &self,
    ) -> Result<AdminGetUserVerificationStatusResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct AdminGetUserVerificationsResponse {
        #[serde(rename = "error")]
        pub error: String,
    }

    /// AdminGetUserVerifications retrieves all verifications for any user (admin only)
GET /verification/admin/users/:userId/verifications
    pub async fn admin_get_user_verifications(
        &self,
    ) -> Result<AdminGetUserVerificationsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for IdverificationPlugin {{
    fn id(&self) -> &str {
        "idverification"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
