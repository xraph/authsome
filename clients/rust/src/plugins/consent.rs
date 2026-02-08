// Auto-generated consent plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct ConsentPlugin {{
    client: Option<AuthsomeClient>,
}

impl ConsentPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct CreateConsentRequest {
        #[serde(rename = "version")]
        pub version: String,
        #[serde(rename = "consentType")]
        pub consent_type: String,
        #[serde(rename = "expiresIn")]
        pub expires_in: *int,
        #[serde(rename = "granted")]
        pub granted: bool,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "purpose")]
        pub purpose: String,
        #[serde(rename = "userId")]
        pub user_id: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateConsentResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// CreateConsent handles POST /consent/records
    pub async fn create_consent(
        &self,
        _request: CreateConsentRequest,
    ) -> Result<CreateConsentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetConsentResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetConsent handles GET /consent/records/:id
    pub async fn get_consent(
        &self,
    ) -> Result<GetConsentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdateConsentRequest {
        #[serde(rename = "granted")]
        pub granted: *bool,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "reason")]
        pub reason: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdateConsentResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// UpdateConsent handles PATCH /consent/records/:id
    pub async fn update_consent(
        &self,
        _request: UpdateConsentRequest,
    ) -> Result<UpdateConsentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RevokeConsentRequest {
        #[serde(rename = "granted")]
        pub granted: *bool,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "reason")]
        pub reason: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct RevokeConsentResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// RevokeConsent handles POST /consent/records/:id/revoke
    pub async fn revoke_consent(
        &self,
        _request: RevokeConsentRequest,
    ) -> Result<RevokeConsentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreateConsentPolicyRequest {
        #[serde(rename = "version")]
        pub version: String,
        #[serde(rename = "consentType")]
        pub consent_type: String,
        #[serde(rename = "content")]
        pub content: String,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "renewable")]
        pub renewable: bool,
        #[serde(rename = "required")]
        pub required: bool,
        #[serde(rename = "validityPeriod")]
        pub validity_period: *int,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateConsentPolicyResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// CreateConsentPolicy handles POST /consent/policies
    pub async fn create_consent_policy(
        &self,
        _request: CreateConsentPolicyRequest,
    ) -> Result<CreateConsentPolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetConsentPolicyResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetConsentPolicy handles GET /consent/policies/:id
    pub async fn get_consent_policy(
        &self,
    ) -> Result<GetConsentPolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RecordCookieConsentRequest {
        #[serde(rename = "sessionId")]
        pub session_id: String,
        #[serde(rename = "thirdParty")]
        pub third_party: bool,
        #[serde(rename = "analytics")]
        pub analytics: bool,
        #[serde(rename = "bannerVersion")]
        pub banner_version: String,
        #[serde(rename = "essential")]
        pub essential: bool,
        #[serde(rename = "functional")]
        pub functional: bool,
        #[serde(rename = "marketing")]
        pub marketing: bool,
        #[serde(rename = "personalization")]
        pub personalization: bool,
    }

    #[derive(Debug, Deserialize)]
    pub struct RecordCookieConsentResponse {
        #[serde(rename = "preferences")]
        pub preferences: ,
    }

    /// RecordCookieConsent handles POST /consent/cookies
    pub async fn record_cookie_consent(
        &self,
        _request: RecordCookieConsentRequest,
    ) -> Result<RecordCookieConsentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetCookieConsentResponse {
        #[serde(rename = "preferences")]
        pub preferences: ,
    }

    /// GetCookieConsent handles GET /consent/cookies
    pub async fn get_cookie_consent(
        &self,
    ) -> Result<GetCookieConsentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RequestDataExportRequest {
        #[serde(rename = "format")]
        pub format: String,
        #[serde(rename = "includeSections")]
        pub include_sections: []string,
    }

    #[derive(Debug, Deserialize)]
    pub struct RequestDataExportResponse {
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "status")]
        pub status: String,
    }

    /// RequestDataExport handles POST /consent/data-exports
    pub async fn request_data_export(
        &self,
        _request: RequestDataExportRequest,
    ) -> Result<RequestDataExportResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetDataExportResponse {
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "status")]
        pub status: String,
    }

    /// GetDataExport handles GET /consent/data-exports/:id
    pub async fn get_data_export(
        &self,
    ) -> Result<GetDataExportResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DownloadDataExportResponse {
        #[serde(rename = "content_type")]
        pub content_type: String,
        #[serde(rename = "data")]
        pub data: []byte,
    }

    /// DownloadDataExport handles GET /consent/data-exports/:id/download
    pub async fn download_data_export(
        &self,
    ) -> Result<DownloadDataExportResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RequestDataDeletionRequest {
        #[serde(rename = "deleteSections")]
        pub delete_sections: []string,
        #[serde(rename = "reason")]
        pub reason: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct RequestDataDeletionResponse {
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "status")]
        pub status: String,
    }

    /// RequestDataDeletion handles POST /consent/data-deletions
    pub async fn request_data_deletion(
        &self,
        _request: RequestDataDeletionRequest,
    ) -> Result<RequestDataDeletionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetDataDeletionResponse {
        #[serde(rename = "id")]
        pub id: String,
        #[serde(rename = "status")]
        pub status: String,
    }

    /// GetDataDeletion handles GET /consent/data-deletions/:id
    pub async fn get_data_deletion(
        &self,
    ) -> Result<GetDataDeletionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ApproveDeletionRequestResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// ApproveDeletionRequest handles POST /consent/data-deletions/:id/approve (Admin only)
    pub async fn approve_deletion_request(
        &self,
    ) -> Result<ApproveDeletionRequestResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetPrivacySettingsResponse {
        #[serde(rename = "settings")]
        pub settings: ,
    }

    /// GetPrivacySettings handles GET /consent/privacy-settings
    pub async fn get_privacy_settings(
        &self,
    ) -> Result<GetPrivacySettingsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdatePrivacySettingsRequest {
        #[serde(rename = "allowDataPortability")]
        pub allow_data_portability: *bool,
        #[serde(rename = "anonymousConsentEnabled")]
        pub anonymous_consent_enabled: *bool,
        #[serde(rename = "contactEmail")]
        pub contact_email: String,
        #[serde(rename = "contactPhone")]
        pub contact_phone: String,
        #[serde(rename = "dataExportExpiryHours")]
        pub data_export_expiry_hours: *int,
        #[serde(rename = "deletionGracePeriodDays")]
        pub deletion_grace_period_days: *int,
        #[serde(rename = "exportFormat")]
        pub export_format: []string,
        #[serde(rename = "gdprMode")]
        pub gdpr_mode: *bool,
        #[serde(rename = "ccpaMode")]
        pub ccpa_mode: *bool,
        #[serde(rename = "dataRetentionDays")]
        pub data_retention_days: *int,
        #[serde(rename = "autoDeleteAfterDays")]
        pub auto_delete_after_days: *int,
        #[serde(rename = "cookieConsentEnabled")]
        pub cookie_consent_enabled: *bool,
        #[serde(rename = "cookieConsentStyle")]
        pub cookie_consent_style: String,
        #[serde(rename = "requireAdminApprovalForDeletion")]
        pub require_admin_approval_for_deletion: *bool,
        #[serde(rename = "requireExplicitConsent")]
        pub require_explicit_consent: *bool,
        #[serde(rename = "consentRequired")]
        pub consent_required: *bool,
        #[serde(rename = "dpoEmail")]
        pub dpo_email: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdatePrivacySettingsResponse {
        #[serde(rename = "settings")]
        pub settings: ,
    }

    /// UpdatePrivacySettings handles PATCH /consent/privacy-settings (Admin only)
    pub async fn update_privacy_settings(
        &self,
        _request: UpdatePrivacySettingsRequest,
    ) -> Result<UpdatePrivacySettingsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetConsentAuditLogsResponse {
        #[serde(rename = "audit_logs")]
        pub audit_logs: Vec<>,
    }

    /// GetConsentAuditLogs handles GET /consent/audit-logs
    pub async fn get_consent_audit_logs(
        &self,
    ) -> Result<GetConsentAuditLogsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GenerateConsentReportResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GenerateConsentReport handles GET /consent/reports
    pub async fn generate_consent_report(
        &self,
    ) -> Result<GenerateConsentReportResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for ConsentPlugin {{
    fn id(&self) -> &str {
        "consent"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
