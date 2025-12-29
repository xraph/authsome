// Auto-generated stepup plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct StepupPlugin {{
    client: Option<AuthsomeClient>,
}

impl StepupPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct EvaluateRequest {
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "method")]
        pub method: String,
        #[serde(rename = "resource_type")]
        pub resource_type: String,
        #[serde(rename = "route")]
        pub route: String,
        #[serde(rename = "action")]
        pub action: String,
        #[serde(rename = "amount")]
        pub amount: f64,
        #[serde(rename = "currency")]
        pub currency: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct EvaluateResponse {
        #[serde(rename = "reason")]
        pub reason: String,
        #[serde(rename = "required")]
        pub required: bool,
    }

    /// Evaluate handles POST /stepup/evaluate
    pub async fn evaluate(
        &self,
        _request: EvaluateRequest,
    ) -> Result<EvaluateResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyRequest {
        #[serde(rename = "device_name")]
        pub device_name: String,
        #[serde(rename = "method")]
        pub method: VerificationMethod,
        #[serde(rename = "remember_device")]
        pub remember_device: bool,
        #[serde(rename = "user_agent")]
        pub user_agent: String,
        #[serde(rename = "challenge_token")]
        pub challenge_token: String,
        #[serde(rename = "device_id")]
        pub device_id: String,
        #[serde(rename = "ip")]
        pub ip: String,
        #[serde(rename = "requirement_id")]
        pub requirement_id: String,
        #[serde(rename = "credential")]
        pub credential: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct VerifyResponse {
        #[serde(rename = "expires_at")]
        pub expires_at: String,
        #[serde(rename = "verified")]
        pub verified: bool,
    }

    /// Verify handles POST /stepup/verify
    pub async fn verify(
        &self,
        _request: VerifyRequest,
    ) -> Result<VerifyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetRequirementResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetRequirement handles GET /stepup/requirements/:id
    pub async fn get_requirement(
        &self,
    ) -> Result<GetRequirementResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListPendingRequirementsResponse {
        #[serde(rename = "requirements")]
        pub requirements: Vec<>,
    }

    /// ListPendingRequirements handles GET /stepup/requirements/pending
    pub async fn list_pending_requirements(
        &self,
    ) -> Result<ListPendingRequirementsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListVerificationsResponse {
        #[serde(rename = "verifications")]
        pub verifications: Vec<>,
    }

    /// ListVerifications handles GET /stepup/verifications
    pub async fn list_verifications(
        &self,
    ) -> Result<ListVerificationsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListRememberedDevicesResponse {
        #[serde(rename = "count")]
        pub count: i32,
        #[serde(rename = "devices")]
        pub devices: ,
    }

    /// ListRememberedDevices handles GET /stepup/devices
    pub async fn list_remembered_devices(
        &self,
    ) -> Result<ListRememberedDevicesResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ForgetDeviceResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// ForgetDevice handles DELETE /stepup/devices/:id
    pub async fn forget_device(
        &self,
    ) -> Result<ForgetDeviceResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreatePolicyRequest {
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "priority")]
        pub priority: i32,
        #[serde(rename = "rules")]
        pub rules: ,
        #[serde(rename = "updated_at")]
        pub updated_at: time.Time,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "org_id")]
        pub org_id: String,
        #[serde(rename = "user_id")]
        pub user_id: String,
        #[serde(rename = "created_at")]
        pub created_at: time.Time,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "id")]
        pub id: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreatePolicyResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// CreatePolicy handles POST /stepup/policies
    pub async fn create_policy(
        &self,
        _request: CreatePolicyRequest,
    ) -> Result<CreatePolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListPoliciesResponse {
        #[serde(rename = "policies")]
        pub policies: Vec<>,
    }

    /// ListPolicies handles GET /stepup/policies
    pub async fn list_policies(
        &self,
    ) -> Result<ListPoliciesResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetPolicyResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetPolicy handles GET /stepup/policies/:id
    pub async fn get_policy(
        &self,
    ) -> Result<GetPolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdatePolicyRequest {
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "org_id")]
        pub org_id: String,
        #[serde(rename = "priority")]
        pub priority: i32,
        #[serde(rename = "rules")]
        pub rules: ,
        #[serde(rename = "updated_at")]
        pub updated_at: time.Time,
        #[serde(rename = "user_id")]
        pub user_id: String,
        #[serde(rename = "created_at")]
        pub created_at: time.Time,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdatePolicyResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// UpdatePolicy handles PUT /stepup/policies/:id
    pub async fn update_policy(
        &self,
        _request: UpdatePolicyRequest,
    ) -> Result<UpdatePolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DeletePolicyResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// DeletePolicy handles DELETE /stepup/policies/:id
    pub async fn delete_policy(
        &self,
    ) -> Result<DeletePolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetAuditLogsResponse {
        #[serde(rename = "audit_logs")]
        pub audit_logs: Vec<>,
    }

    /// GetAuditLogs handles GET /stepup/audit
    pub async fn get_audit_logs(
        &self,
    ) -> Result<GetAuditLogsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct StatusResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// Status handles GET /stepup/status
    pub async fn status(
        &self,
    ) -> Result<StatusResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for StepupPlugin {{
    fn id(&self) -> &str {
        "stepup"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
