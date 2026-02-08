// Auto-generated compliance plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct CompliancePlugin {{
    client: Option<AuthsomeClient>,
}

impl CompliancePlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct CreateProfileRequest {
        #[serde(rename = "standards")]
        pub standards: []ComplianceStandard,
        #[serde(rename = "complianceContact")]
        pub compliance_contact: String,
        #[serde(rename = "encryptionAtRest")]
        pub encryption_at_rest: bool,
        #[serde(rename = "passwordExpiryDays")]
        pub password_expiry_days: i32,
        #[serde(rename = "passwordRequireSymbol")]
        pub password_require_symbol: bool,
        #[serde(rename = "regularAccessReview")]
        pub regular_access_review: bool,
        #[serde(rename = "sessionIdleTimeout")]
        pub session_idle_timeout: i32,
        #[serde(rename = "sessionMaxAge")]
        pub session_max_age: i32,
        #[serde(rename = "appId")]
        pub app_id: String,
        #[serde(rename = "dataResidency")]
        pub data_residency: String,
        #[serde(rename = "detailedAuditTrail")]
        pub detailed_audit_trail: bool,
        #[serde(rename = "dpoContact")]
        pub dpo_contact: String,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "encryptionInTransit")]
        pub encryption_in_transit: bool,
        #[serde(rename = "leastPrivilege")]
        pub least_privilege: bool,
        #[serde(rename = "mfaRequired")]
        pub mfa_required: bool,
        #[serde(rename = "passwordMinLength")]
        pub password_min_length: i32,
        #[serde(rename = "passwordRequireLower")]
        pub password_require_lower: bool,
        #[serde(rename = "passwordRequireNumber")]
        pub password_require_number: bool,
        #[serde(rename = "passwordRequireUpper")]
        pub password_require_upper: bool,
        #[serde(rename = "sessionIpBinding")]
        pub session_ip_binding: bool,
        #[serde(rename = "auditLogExport")]
        pub audit_log_export: bool,
        #[serde(rename = "metadata")]
        pub metadata: ,
        #[serde(rename = "rbacRequired")]
        pub rbac_required: bool,
        #[serde(rename = "retentionDays")]
        pub retention_days: i32,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateProfileResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// CreateProfile creates a new compliance profile
POST /auth/compliance/profiles
    pub async fn create_profile(
        &self,
        _request: CreateProfileRequest,
    ) -> Result<CreateProfileResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreateProfileFromTemplateRequest {
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateProfileFromTemplateResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// CreateProfileFromTemplate creates a profile from a template
POST /auth/compliance/profiles/from-template
    pub async fn create_profile_from_template(
        &self,
        _request: CreateProfileFromTemplateRequest,
    ) -> Result<CreateProfileFromTemplateResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetProfileResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetProfile retrieves a compliance profile
GET /auth/compliance/profiles/:id
    pub async fn get_profile(
        &self,
    ) -> Result<GetProfileResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetAppProfileResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetAppProfile retrieves the compliance profile for an app
GET /auth/compliance/apps/:appId/profile
    pub async fn get_app_profile(
        &self,
    ) -> Result<GetAppProfileResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdateProfileRequest {
        #[serde(rename = "retentionDays")]
        pub retention_days: *int,
        #[serde(rename = "status")]
        pub status: *string,
        #[serde(rename = "mfaRequired")]
        pub mfa_required: *bool,
        #[serde(rename = "name")]
        pub name: *string,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdateProfileResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// UpdateProfile updates a compliance profile
PUT /auth/compliance/profiles/:id
    pub async fn update_profile(
        &self,
        _request: UpdateProfileRequest,
    ) -> Result<UpdateProfileResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DeleteProfileResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// DeleteProfile deletes a compliance profile
DELETE /auth/compliance/profiles/:id
    pub async fn delete_profile(
        &self,
    ) -> Result<DeleteProfileResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetComplianceStatusResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// GetComplianceStatus gets overall compliance status for an app
GET /auth/compliance/apps/:appId/status
    pub async fn get_compliance_status(
        &self,
    ) -> Result<GetComplianceStatusResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetDashboardResponse {
        #[serde(rename = "metrics")]
        pub metrics: ,
    }

    /// GetDashboard gets compliance dashboard data
GET /auth/compliance/apps/:appId/dashboard
    pub async fn get_dashboard(
        &self,
    ) -> Result<GetDashboardResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RunCheckRequest {
        #[serde(rename = "checkType")]
        pub check_type: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct RunCheckResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// RunCheck executes a compliance check
POST /auth/compliance/profiles/:profileId/checks
    pub async fn run_check(
        &self,
        _request: RunCheckRequest,
    ) -> Result<RunCheckResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListChecksResponse {
        #[serde(rename = "checks")]
        pub checks: Vec<>,
    }

    /// ListChecks lists compliance checks
GET /auth/compliance/profiles/:profileId/checks
    pub async fn list_checks(
        &self,
    ) -> Result<ListChecksResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetCheckResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetCheck retrieves a compliance check
GET /auth/compliance/checks/:id
    pub async fn get_check(
        &self,
    ) -> Result<GetCheckResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListViolationsResponse {
        #[serde(rename = "violations")]
        pub violations: Vec<>,
    }

    /// ListViolations lists compliance violations
GET /auth/compliance/apps/:appId/violations
    pub async fn list_violations(
        &self,
    ) -> Result<ListViolationsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetViolationResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetViolation retrieves a compliance violation
GET /auth/compliance/violations/:id
    pub async fn get_violation(
        &self,
    ) -> Result<GetViolationResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct ResolveViolationRequest {
        #[serde(rename = "notes")]
        pub notes: String,
        #[serde(rename = "resolution")]
        pub resolution: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct ResolveViolationResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// ResolveViolation resolves a compliance violation
PUT /auth/compliance/violations/:id/resolve
    pub async fn resolve_violation(
        &self,
        _request: ResolveViolationRequest,
    ) -> Result<ResolveViolationResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct GenerateReportRequest {
        #[serde(rename = "period")]
        pub period: String,
        #[serde(rename = "reportType")]
        pub report_type: String,
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
        #[serde(rename = "format")]
        pub format: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct GenerateReportResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GenerateReport generates a compliance report
POST /auth/compliance/apps/:appId/reports
    pub async fn generate_report(
        &self,
        _request: GenerateReportRequest,
    ) -> Result<GenerateReportResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListReportsResponse {
        #[serde(rename = "reports")]
        pub reports: Vec<>,
    }

    /// ListReports lists compliance reports
GET /auth/compliance/apps/:appId/reports
    pub async fn list_reports(
        &self,
    ) -> Result<ListReportsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetReportResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetReport retrieves a compliance report
GET /auth/compliance/reports/:id
    pub async fn get_report(
        &self,
    ) -> Result<GetReportResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DownloadReportResponse {
        #[serde(rename = "content_type")]
        pub content_type: String,
        #[serde(rename = "data")]
        pub data: []byte,
    }

    /// DownloadReport downloads a compliance report file
GET /auth/compliance/reports/:id/download
    pub async fn download_report(
        &self,
    ) -> Result<DownloadReportResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreateEvidenceRequest {
        #[serde(rename = "controlId")]
        pub control_id: String,
        #[serde(rename = "description")]
        pub description: String,
        #[serde(rename = "evidenceType")]
        pub evidence_type: String,
        #[serde(rename = "fileUrl")]
        pub file_url: String,
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
        #[serde(rename = "title")]
        pub title: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateEvidenceResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// CreateEvidence creates compliance evidence
POST /auth/compliance/apps/:appId/evidence
    pub async fn create_evidence(
        &self,
        _request: CreateEvidenceRequest,
    ) -> Result<CreateEvidenceResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListEvidenceResponse {
        #[serde(rename = "evidence")]
        pub evidence: Vec<>,
    }

    /// ListEvidence lists compliance evidence
GET /auth/compliance/apps/:appId/evidence
    pub async fn list_evidence(
        &self,
    ) -> Result<ListEvidenceResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetEvidenceResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetEvidence retrieves compliance evidence
GET /auth/compliance/evidence/:id
    pub async fn get_evidence(
        &self,
    ) -> Result<GetEvidenceResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DeleteEvidenceResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// DeleteEvidence deletes compliance evidence
DELETE /auth/compliance/evidence/:id
    pub async fn delete_evidence(
        &self,
    ) -> Result<DeleteEvidenceResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreatePolicyRequest {
        #[serde(rename = "policyType")]
        pub policy_type: String,
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
        #[serde(rename = "title")]
        pub title: String,
        #[serde(rename = "version")]
        pub version: String,
        #[serde(rename = "content")]
        pub content: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreatePolicyResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// CreatePolicy creates a compliance policy
POST /auth/compliance/apps/:appId/policies
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

    /// ListPolicies lists compliance policies
GET /auth/compliance/apps/:appId/policies
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

    /// GetPolicy retrieves a compliance policy
GET /auth/compliance/policies/:id
    pub async fn get_policy(
        &self,
    ) -> Result<GetPolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdatePolicyRequest {
        #[serde(rename = "version")]
        pub version: *string,
        #[serde(rename = "content")]
        pub content: *string,
        #[serde(rename = "status")]
        pub status: *string,
        #[serde(rename = "title")]
        pub title: *string,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdatePolicyResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// UpdatePolicy updates a compliance policy
PUT /auth/compliance/policies/:id
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

    /// DeletePolicy deletes a compliance policy
DELETE /auth/compliance/policies/:id
    pub async fn delete_policy(
        &self,
    ) -> Result<DeletePolicyResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreateTrainingRequest {
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
        #[serde(rename = "trainingType")]
        pub training_type: String,
        #[serde(rename = "userId")]
        pub user_id: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateTrainingResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// CreateTraining creates a training record
POST /auth/compliance/apps/:appId/training
    pub async fn create_training(
        &self,
        _request: CreateTrainingRequest,
    ) -> Result<CreateTrainingResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListTrainingResponse {
        #[serde(rename = "training")]
        pub training: Vec<>,
    }

    /// ListTraining lists training records
GET /auth/compliance/apps/:appId/training
    pub async fn list_training(
        &self,
    ) -> Result<ListTrainingResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetUserTrainingResponse {
        #[serde(rename = "user_id")]
        pub user_id: String,
    }

    /// GetUserTraining gets training status for a user
GET /auth/compliance/users/:userId/training
    pub async fn get_user_training(
        &self,
    ) -> Result<GetUserTrainingResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CompleteTrainingRequest {
        #[serde(rename = "score")]
        pub score: i32,
    }

    #[derive(Debug, Deserialize)]
    pub struct CompleteTrainingResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// CompleteTraining marks training as completed
PUT /auth/compliance/training/:id/complete
    pub async fn complete_training(
        &self,
        _request: CompleteTrainingRequest,
    ) -> Result<CompleteTrainingResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListTemplatesResponse {
        #[serde(rename = "templates")]
        pub templates: Vec<>,
    }

    /// ListTemplates lists available compliance templates
GET /auth/compliance/templates
    pub async fn list_templates(
        &self,
    ) -> Result<ListTemplatesResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetTemplateResponse {
        #[serde(rename = "standard")]
        pub standard: String,
    }

    /// GetTemplate retrieves a compliance template
GET /auth/compliance/templates/:standard
    pub async fn get_template(
        &self,
    ) -> Result<GetTemplateResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for CompliancePlugin {{
    fn id(&self) -> &str {
        "compliance"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
