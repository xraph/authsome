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
        #[serde(rename = "", skip_serializing_if = "Option::is_none")]
        pub : Option<>,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateProfileResponse {
        #[serde(rename = "error")]
        pub error: String,
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
        #[serde(rename = "appId")]
        pub app_id: String,
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateProfileFromTemplateResponse {
        #[serde(rename = "error")]
        pub error: String,
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

    /// GetProfile retrieves a compliance profile
GET /auth/compliance/profiles/:id
    pub async fn get_profile(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetAppProfile retrieves the compliance profile for an app
GET /auth/compliance/apps/:appId/profile
    pub async fn get_app_profile(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdateProfileRequest {
        #[serde(rename = "", skip_serializing_if = "Option::is_none")]
        pub : Option<*int>,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdateProfileResponse {
        #[serde(rename = "error")]
        pub error: String,
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

    /// DeleteProfile deletes a compliance profile
DELETE /auth/compliance/profiles/:id
    pub async fn delete_profile(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetComplianceStatus gets overall compliance status for an app
GET /auth/compliance/apps/:appId/status
    pub async fn get_compliance_status(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetDashboard gets compliance dashboard data
GET /auth/compliance/apps/:appId/dashboard
    pub async fn get_dashboard(
        &self,
    ) -> Result<()> {
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
        #[serde(rename = "error")]
        pub error: String,
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

    /// ListChecks lists compliance checks
GET /auth/compliance/profiles/:profileId/checks
    pub async fn list_checks(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetCheck retrieves a compliance check
GET /auth/compliance/checks/:id
    pub async fn get_check(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// ListViolations lists compliance violations
GET /auth/compliance/apps/:appId/violations
    pub async fn list_violations(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetViolation retrieves a compliance violation
GET /auth/compliance/violations/:id
    pub async fn get_violation(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// ResolveViolation resolves a compliance violation
PUT /auth/compliance/violations/:id/resolve
    pub async fn resolve_violation(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct GenerateReportRequest {
        #[serde(rename = "format")]
        pub format: String,
        #[serde(rename = "period")]
        pub period: String,
        #[serde(rename = "reportType")]
        pub report_type: String,
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
    }

    #[derive(Debug, Deserialize)]
    pub struct GenerateReportResponse {
        #[serde(rename = "error")]
        pub error: String,
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

    /// ListReports lists compliance reports
GET /auth/compliance/apps/:appId/reports
    pub async fn list_reports(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetReport retrieves a compliance report
GET /auth/compliance/reports/:id
    pub async fn get_report(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct DownloadReportResponse {
        #[serde(rename = "error")]
        pub error: String,
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
        #[serde(rename = "error")]
        pub error: String,
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

    /// ListEvidence lists compliance evidence
GET /auth/compliance/apps/:appId/evidence
    pub async fn list_evidence(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetEvidence retrieves compliance evidence
GET /auth/compliance/evidence/:id
    pub async fn get_evidence(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// DeleteEvidence deletes compliance evidence
DELETE /auth/compliance/evidence/:id
    pub async fn delete_evidence(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreatePolicyRequest {
        #[serde(rename = "content")]
        pub content: String,
        #[serde(rename = "policyType")]
        pub policy_type: String,
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
        #[serde(rename = "title")]
        pub title: String,
        #[serde(rename = "version")]
        pub version: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreatePolicyResponse {
        #[serde(rename = "error")]
        pub error: String,
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

    /// ListPolicies lists compliance policies
GET /auth/compliance/apps/:appId/policies
    pub async fn list_policies(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetPolicy retrieves a compliance policy
GET /auth/compliance/policies/:id
    pub async fn get_policy(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdatePolicyRequest {
        #[serde(rename = "content")]
        pub content: *string,
        #[serde(rename = "status")]
        pub status: *string,
        #[serde(rename = "title")]
        pub title: *string,
        #[serde(rename = "version")]
        pub version: *string,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdatePolicyResponse {
        #[serde(rename = "error")]
        pub error: String,
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

    /// DeletePolicy deletes a compliance policy
DELETE /auth/compliance/policies/:id
    pub async fn delete_policy(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CreateTrainingRequest {
        #[serde(rename = "trainingType")]
        pub training_type: String,
        #[serde(rename = "userId")]
        pub user_id: String,
        #[serde(rename = "standard")]
        pub standard: ComplianceStandard,
    }

    #[derive(Debug, Deserialize)]
    pub struct CreateTrainingResponse {
        #[serde(rename = "error")]
        pub error: String,
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

    /// ListTraining lists training records
GET /auth/compliance/apps/:appId/training
    pub async fn list_training(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// GetUserTraining gets training status for a user
GET /auth/compliance/users/:userId/training
    pub async fn get_user_training(
        &self,
    ) -> Result<()> {
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
        #[serde(rename = "error")]
        pub error: String,
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

    /// ListTemplates lists available compliance templates
GET /auth/compliance/templates
    pub async fn list_templates(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetTemplateResponse {
        #[serde(rename = "error")]
        pub error: String,
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
