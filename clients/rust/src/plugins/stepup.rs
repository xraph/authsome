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
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "method")]
        pub method: String,
    }

    /// Evaluate handles POST /stepup/evaluate
    pub async fn evaluate(
        &self,
        _request: EvaluateRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyRequest {
        #[serde(rename = "challenge_token")]
        pub challenge_token: String,
        #[serde(rename = "device_name")]
        pub device_name: String,
        #[serde(rename = "ip")]
        pub ip: String,
        #[serde(rename = "requirement_id")]
        pub requirement_id: String,
        #[serde(rename = "user_agent")]
        pub user_agent: String,
        #[serde(rename = "credential")]
        pub credential: String,
        #[serde(rename = "device_id")]
        pub device_id: String,
        #[serde(rename = "method")]
        pub method: VerificationMethod,
        #[serde(rename = "remember_device")]
        pub remember_device: bool,
    }

    /// Verify handles POST /stepup/verify
    pub async fn verify(
        &self,
        _request: VerifyRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetRequirement handles GET /stepup/requirements/:id
    pub async fn get_requirement(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// ListPendingRequirements handles GET /stepup/requirements/pending
    pub async fn list_pending_requirements(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// ListVerifications handles GET /stepup/verifications
    pub async fn list_verifications(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// ListRememberedDevices handles GET /stepup/devices
    pub async fn list_remembered_devices(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// ForgetDevice handles DELETE /stepup/devices/:id
    pub async fn forget_device(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreatePolicyRequest {
        #[serde(rename = "created_at")]
        pub created_at: time.Time,
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "org_id")]
        pub org_id: String,
        #[serde(rename = "priority")]
        pub priority: i32,
        #[serde(rename = "rules")]
        pub rules: ,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "updated_at")]
        pub updated_at: time.Time,
        #[serde(rename = "user_id")]
        pub user_id: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreatePolicyResponse {
        #[serde(rename = "rules")]
        pub rules: ,
        #[serde(rename = "user_id")]
        pub user_id: String,
        #[serde(rename = "created_at")]
        pub created_at: time.Time,
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "priority")]
        pub priority: i32,
        #[serde(rename = "updated_at")]
        pub updated_at: time.Time,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "org_id")]
        pub org_id: String,
    }

    /// CreatePolicy handles POST /stepup/policies
    pub async fn create_policy(
        &self,
        _request: CreatePolicyRequest,
    ) -> Result<CreatePolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// ListPolicies handles GET /stepup/policies
    pub async fn list_policies(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetPolicy handles GET /stepup/policies/:id
    pub async fn get_policy(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdatePolicyRequest {
        #[serde(rename = "updated_at")]
        pub updated_at: time.Time,
        #[serde(rename = "created_at")]
        pub created_at: time.Time,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "priority")]
        pub priority: i32,
        #[serde(rename = "rules")]
        pub rules: ,
        #[serde(rename = "user_id")]
        pub user_id: String,
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "org_id")]
        pub org_id: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdatePolicyResponse {
        #[serde(rename = "created_at")]
        pub created_at: time.Time,
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "org_id")]
        pub org_id: String,
        #[serde(rename = "rules")]
        pub rules: ,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "priority")]
        pub priority: i32,
        #[serde(rename = "updated_at")]
        pub updated_at: time.Time,
        #[serde(rename = "user_id")]
        pub user_id: String,
    }

    /// UpdatePolicy handles PUT /stepup/policies/:id
    pub async fn update_policy(
        &self,
        _request: UpdatePolicyRequest,
    ) -> Result<UpdatePolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// DeletePolicy handles DELETE /stepup/policies/:id
    pub async fn delete_policy(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetAuditLogs handles GET /stepup/audit
    pub async fn get_audit_logs(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// Status handles GET /stepup/status
    pub async fn status(
        &self,
    ) -> Result<()> {
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
