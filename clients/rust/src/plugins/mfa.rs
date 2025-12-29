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
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "priority")]
        pub priority: FactorPriority,
        #[serde(rename = "type")]
        pub type: FactorType,
    }

    #[derive(Debug, Deserialize)]
    pub struct EnrollFactorResponse {
        #[serde(rename = "provisioningData")]
        pub provisioning_data: ,
        #[serde(rename = "status")]
        pub status: FactorStatus,
        #[serde(rename = "type")]
        pub type: FactorType,
        #[serde(rename = "factorId")]
        pub factor_id: xid.ID,
    }

    /// EnrollFactor handles POST /mfa/factors/enroll
    pub async fn enroll_factor(
        &self,
        _request: EnrollFactorRequest,
    ) -> Result<EnrollFactorResponse> {{
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

    #[derive(Debug, Deserialize)]
    pub struct GetFactorResponse {
        #[serde(rename = "expiresAt")]
        pub expires_at: *time.Time,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "status")]
        pub status: FactorStatus,
        #[serde(rename = "type")]
        pub type: FactorType,
        #[serde(rename = "verifiedAt")]
        pub verified_at: *time.Time,
        #[serde(rename = "-")]
        pub -: String,
        #[serde(rename = "createdAt")]
        pub created_at: time.Time,
        #[serde(rename = "id")]
        pub id: xid.ID,
        #[serde(rename = "lastUsedAt")]
        pub last_used_at: *time.Time,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "priority")]
        pub priority: FactorPriority,
        #[serde(rename = "updatedAt")]
        pub updated_at: time.Time,
        #[serde(rename = "userId")]
        pub user_id: xid.ID,
    }

    /// GetFactor handles GET /mfa/factors/:id
    pub async fn get_factor(
        &self,
    ) -> Result<GetFactorResponse> {{
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

    /// VerifyFactor handles POST /mfa/factors/:id/verify
    pub async fn verify_factor(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct InitiateChallengeRequest {
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "userId")]
        pub user_id: xid.ID,
        #[serde(rename = "context")]
        pub context: String,
        #[serde(rename = "factorTypes")]
        pub factor_types: []FactorType,
    }

    #[derive(Debug, Deserialize)]
    pub struct InitiateChallengeResponse {
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
        #[serde(rename = "availableFactors")]
        pub available_factors: []FactorInfo,
        #[serde(rename = "challengeId")]
        pub challenge_id: xid.ID,
        #[serde(rename = "expiresAt")]
        pub expires_at: time.Time,
        #[serde(rename = "factorsRequired")]
        pub factors_required: i32,
    }

    /// InitiateChallenge handles POST /mfa/challenge
    pub async fn initiate_challenge(
        &self,
        _request: InitiateChallengeRequest,
    ) -> Result<InitiateChallengeResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyChallengeRequest {
        #[serde(rename = "factorId")]
        pub factor_id: xid.ID,
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
    }

    #[derive(Debug, Deserialize)]
    pub struct VerifyChallengeResponse {
        #[serde(rename = "success")]
        pub success: bool,
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "expiresAt")]
        pub expires_at: *time.Time,
        #[serde(rename = "factorsRemaining")]
        pub factors_remaining: i32,
        #[serde(rename = "sessionComplete")]
        pub session_complete: bool,
    }

    /// VerifyChallenge handles POST /mfa/verify
    pub async fn verify_challenge(
        &self,
        _request: VerifyChallengeRequest,
    ) -> Result<VerifyChallengeResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetChallengeStatusResponse {
        #[serde(rename = "status")]
        pub status: String,
        #[serde(rename = "completedAt")]
        pub completed_at: *time.Time,
        #[serde(rename = "expiresAt")]
        pub expires_at: time.Time,
        #[serde(rename = "factorsRemaining")]
        pub factors_remaining: i32,
        #[serde(rename = "factorsRequired")]
        pub factors_required: i32,
        #[serde(rename = "factorsVerified")]
        pub factors_verified: i32,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    /// GetChallengeStatus handles GET /mfa/challenge/:id
    pub async fn get_challenge_status(
        &self,
    ) -> Result<GetChallengeStatusResponse> {{
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
        #[serde(rename = "count")]
        pub count: i32,
        #[serde(rename = "devices")]
        pub devices: ,
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

    #[derive(Debug, Deserialize)]
    pub struct GetStatusResponse {
        #[serde(rename = "gracePeriod")]
        pub grace_period: *time.Time,
        #[serde(rename = "policyActive")]
        pub policy_active: bool,
        #[serde(rename = "requiredCount")]
        pub required_count: i32,
        #[serde(rename = "trustedDevice")]
        pub trusted_device: bool,
        #[serde(rename = "enabled")]
        pub enabled: bool,
        #[serde(rename = "enrolledFactors")]
        pub enrolled_factors: []FactorInfo,
    }

    /// GetStatus handles GET /mfa/status
    pub async fn get_status(
        &self,
    ) -> Result<GetStatusResponse> {{
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

    /// AdminUpdatePolicy handles PUT /mfa/admin/policy
Updates the MFA policy for an app (admin only)
    pub async fn admin_update_policy(
        &self,
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
