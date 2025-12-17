// Auto-generated mfa plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct MfaPlugin {{
    client: Option<AuthsomeClient>,
}

impl MfaPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct EnrollFactorRequest {
        #[serde(rename = "priority")]
        pub priority: FactorPriority,
        #[serde(rename = "type")]
        pub type: FactorType,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
    }

    /// EnrollFactor handles POST /mfa/factors/enroll
    pub async fn enroll_factor(
        &self,
        _request: EnrollFactorRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListFactorsResponse {
        #[serde(rename = "count")]
        pub count: i32,
        #[serde(rename = "factors")]
        pub factors: ,
    }

    /// ListFactors handles GET /mfa/factors
    pub async fn list_factors(
        &self,
    ) -> Result<ListFactorsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetFactor handles GET /mfa/factors/:id
    pub async fn get_factor(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// UpdateFactor handles PUT /mfa/factors/:id
    pub async fn update_factor(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// DeleteFactor handles DELETE /mfa/factors/:id
    pub async fn delete_factor(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyFactorRequest {
        #[serde(rename = "code")]
        pub code: String,
    }

    /// VerifyFactor handles POST /mfa/factors/:id/verify
    pub async fn verify_factor(
        &self,
        _request: VerifyFactorRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct InitiateChallengeRequest {
        #[serde(rename = "userId")]
        pub user_id: xid.ID,
        #[serde(rename = "context")]
        pub context: String,
        #[serde(rename = "factorTypes")]
        pub factor_types: []FactorType,
        #[serde(rename = "metadata")]
        pub metadata: ,
    }

    /// InitiateChallenge handles POST /mfa/challenge
    pub async fn initiate_challenge(
        &self,
        _request: InitiateChallengeRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyChallengeRequest {
        #[serde(rename = "rememberDevice")]
        pub remember_device: bool,
        #[serde(rename = "challengeId")]
        pub challenge_id: xid.ID,
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "data")]
        pub data: ,
        #[serde(rename = "deviceInfo")]
        pub device_info: *DeviceInfo,
        #[serde(rename = "factorId")]
        pub factor_id: xid.ID,
    }

    /// VerifyChallenge handles POST /mfa/verify
    pub async fn verify_challenge(
        &self,
        _request: VerifyChallengeRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetChallengeStatus handles GET /mfa/challenge/:id
    pub async fn get_challenge_status(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct TrustDeviceRequest {
        #[serde(rename = "deviceId")]
        pub device_id: String,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
    }

    /// TrustDevice handles POST /mfa/devices/trust
    pub async fn trust_device(
        &self,
        _request: TrustDeviceRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListTrustedDevicesResponse {
        #[serde(rename = "devices")]
        pub devices: ,
        #[serde(rename = "count")]
        pub count: i32,
    }

    /// ListTrustedDevices handles GET /mfa/devices
    pub async fn list_trusted_devices(
        &self,
    ) -> Result<ListTrustedDevicesResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// RevokeTrustedDevice handles DELETE /mfa/devices/:id
    pub async fn revoke_trusted_device(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetStatus handles GET /mfa/status
    pub async fn get_status(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetPolicyResponse {
        #[serde(rename = "allowed_factor_types")]
        pub allowed_factor_types: []string,
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "required_factor_count")]
        pub required_factor_count: i32,
    }

    /// GetPolicy handles GET /mfa/policy
    pub async fn get_policy(
        &self,
    ) -> Result<GetPolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct AdminUpdatePolicyRequest {
        #[serde(rename = "allowedTypes")]
        pub allowed_types: []string,
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "gracePeriod")]
        pub grace_period: i32,
        #[serde(rename = "requiredFactors")]
        pub required_factors: i32,
    }

    /// AdminUpdatePolicy handles PUT /mfa/admin/policy
Updates the MFA policy for an app (admin only)
    pub async fn admin_update_policy(
        &self,
        _request: AdminUpdatePolicyRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// AdminResetUserMFA handles POST /mfa/admin/users/:id/reset
Resets all MFA factors for a user (admin only)
    pub async fn admin_reset_user_m_f_a(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for MfaPlugin {{
    fn id(&self) -> &str {
        "mfa"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
