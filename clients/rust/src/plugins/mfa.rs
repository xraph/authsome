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

    #[derive(Debug, Deserialize)]
    pub struct EnrollFactorResponse {
        #[serde(rename = "error")]
        pub error: String,
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "details")]
        pub details: ,
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
        #[serde(rename = "error")]
        pub error: String,
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "details")]
        pub details: ,
    }

    /// GetFactor handles GET /mfa/factors/:id
    pub async fn get_factor(
        &self,
    ) -> Result<GetFactorResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdateFactorResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// UpdateFactor handles PUT /mfa/factors/:id
    pub async fn update_factor(
        &self,
    ) -> Result<UpdateFactorResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DeleteFactorResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// DeleteFactor handles DELETE /mfa/factors/:id
    pub async fn delete_factor(
        &self,
    ) -> Result<DeleteFactorResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyFactorRequest {
        #[serde(rename = "code")]
        pub code: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct VerifyFactorResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// VerifyFactor handles POST /mfa/factors/:id/verify
    pub async fn verify_factor(
        &self,
        _request: VerifyFactorRequest,
    ) -> Result<VerifyFactorResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct InitiateChallengeRequest {
        #[serde(rename = "context")]
        pub context: String,
        #[serde(rename = "factorTypes")]
        pub factor_types: []FactorType,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "userId")]
        pub user_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct InitiateChallengeResponse {
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "details")]
        pub details: ,
        #[serde(rename = "error")]
        pub error: String,
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

    #[derive(Debug, Deserialize)]
    pub struct VerifyChallengeResponse {
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "details")]
        pub details: ,
        #[serde(rename = "error")]
        pub error: String,
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
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "details")]
        pub details: ,
        #[serde(rename = "error")]
        pub error: String,
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
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "deviceId")]
        pub device_id: String,
        #[serde(rename = "metadata")]
        pub metadata: ,
    }

    #[derive(Debug, Deserialize)]
    pub struct TrustDeviceResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// TrustDevice handles POST /mfa/devices/trust
    pub async fn trust_device(
        &self,
        _request: TrustDeviceRequest,
    ) -> Result<TrustDeviceResponse> {{
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

    #[derive(Debug, Deserialize)]
    pub struct RevokeTrustedDeviceResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// RevokeTrustedDevice handles DELETE /mfa/devices/:id
    pub async fn revoke_trusted_device(
        &self,
    ) -> Result<RevokeTrustedDeviceResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetStatusResponse {
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "details")]
        pub details: ,
        #[serde(rename = "error")]
        pub error: String,
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
        #[serde(rename = "required_factor_count")]
        pub required_factor_count: i32,
        #[serde(rename = "allowed_factor_types")]
        pub allowed_factor_types: []string,
        #[serde(rename = "enabled")]
        pub enabled: bool,
    }

    /// GetPolicy handles GET /mfa/policy
    pub async fn get_policy(
        &self,
    ) -> Result<GetPolicyResponse> {{
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
