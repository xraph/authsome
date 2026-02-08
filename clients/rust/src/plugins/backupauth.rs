// Auto-generated backupauth plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct BackupauthPlugin {{
    client: Option<AuthsomeClient>,
}

impl BackupauthPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct StartRecoveryRequest {
        #[serde(rename = "deviceId")]
        pub device_id: String,
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "preferredMethod")]
        pub preferred_method: RecoveryMethod,
        #[serde(rename = "userId")]
        pub user_id: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct StartRecoveryResponse {
        #[serde(rename = "session_id")]
        pub session_id: String,
    }

    /// StartRecovery handles POST /recovery/start
    pub async fn start_recovery(
        &self,
        _request: StartRecoveryRequest,
    ) -> Result<StartRecoveryResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct ContinueRecoveryRequest {
        #[serde(rename = "method")]
        pub method: RecoveryMethod,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct ContinueRecoveryResponse {
        #[serde(rename = "session_id")]
        pub session_id: String,
    }

    /// ContinueRecovery handles POST /recovery/continue
    pub async fn continue_recovery(
        &self,
        _request: ContinueRecoveryRequest,
    ) -> Result<ContinueRecoveryResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CompleteRecoveryRequest {
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct CompleteRecoveryResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// CompleteRecovery handles POST /recovery/complete
    pub async fn complete_recovery(
        &self,
        _request: CompleteRecoveryRequest,
    ) -> Result<CompleteRecoveryResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CancelRecoveryRequest {
        #[serde(rename = "reason")]
        pub reason: String,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct CancelRecoveryResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// CancelRecovery handles POST /recovery/cancel
    pub async fn cancel_recovery(
        &self,
        _request: CancelRecoveryRequest,
    ) -> Result<CancelRecoveryResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct GenerateRecoveryCodesRequest {
        #[serde(rename = "count")]
        pub count: i32,
        #[serde(rename = "format")]
        pub format: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct GenerateRecoveryCodesResponse {
        #[serde(rename = "codes")]
        pub codes: []string,
    }

    /// GenerateRecoveryCodes handles POST /recovery-codes/generate
    pub async fn generate_recovery_codes(
        &self,
        _request: GenerateRecoveryCodesRequest,
    ) -> Result<GenerateRecoveryCodesResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyRecoveryCodeRequest {
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct VerifyRecoveryCodeResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// VerifyRecoveryCode handles POST /recovery-codes/verify
    pub async fn verify_recovery_code(
        &self,
        _request: VerifyRecoveryCodeRequest,
    ) -> Result<VerifyRecoveryCodeResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct SetupSecurityQuestionsRequest {
        #[serde(rename = "questions")]
        pub questions: []SetupSecurityQuestionRequest,
    }

    #[derive(Debug, Deserialize)]
    pub struct SetupSecurityQuestionsResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// SetupSecurityQuestions handles POST /security-questions/setup
    pub async fn setup_security_questions(
        &self,
        _request: SetupSecurityQuestionsRequest,
    ) -> Result<SetupSecurityQuestionsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct GetSecurityQuestionsRequest {
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct GetSecurityQuestionsResponse {
        #[serde(rename = "questions")]
        pub questions: []string,
    }

    /// GetSecurityQuestions handles POST /security-questions/get
    pub async fn get_security_questions(
        &self,
        _request: GetSecurityQuestionsRequest,
    ) -> Result<GetSecurityQuestionsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifySecurityAnswersRequest {
        #[serde(rename = "answers")]
        pub answers: ,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct VerifySecurityAnswersResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// VerifySecurityAnswers handles POST /security-questions/verify
    pub async fn verify_security_answers(
        &self,
        _request: VerifySecurityAnswersRequest,
    ) -> Result<VerifySecurityAnswersResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct AddTrustedContactRequest {
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "name")]
        pub name: String,
        #[serde(rename = "phone")]
        pub phone: String,
        #[serde(rename = "relationship")]
        pub relationship: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct AddTrustedContactResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// AddTrustedContact handles POST /trusted-contacts/add
    pub async fn add_trusted_contact(
        &self,
        _request: AddTrustedContactRequest,
    ) -> Result<AddTrustedContactResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListTrustedContactsResponse {
        #[serde(rename = "contacts")]
        pub contacts: Vec<>,
    }

    /// ListTrustedContacts handles GET /trusted-contacts
    pub async fn list_trusted_contacts(
        &self,
    ) -> Result<ListTrustedContactsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyTrustedContactRequest {
        #[serde(rename = "token")]
        pub token: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct VerifyTrustedContactResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// VerifyTrustedContact handles POST /trusted-contacts/verify
    pub async fn verify_trusted_contact(
        &self,
        _request: VerifyTrustedContactRequest,
    ) -> Result<VerifyTrustedContactResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RequestTrustedContactVerificationRequest {
        #[serde(rename = "contactId")]
        pub contact_id: xid.ID,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct RequestTrustedContactVerificationResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// RequestTrustedContactVerification handles POST /trusted-contacts/request-verification
    pub async fn request_trusted_contact_verification(
        &self,
        _request: RequestTrustedContactVerificationRequest,
    ) -> Result<RequestTrustedContactVerificationResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct RemoveTrustedContactResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// RemoveTrustedContact handles DELETE /trusted-contacts/:id
    pub async fn remove_trusted_contact(
        &self,
    ) -> Result<RemoveTrustedContactResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct SendVerificationCodeRequest {
        #[serde(rename = "method")]
        pub method: RecoveryMethod,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
        #[serde(rename = "target")]
        pub target: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct SendVerificationCodeResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// SendVerificationCode handles POST /verification/send
    pub async fn send_verification_code(
        &self,
        _request: SendVerificationCodeRequest,
    ) -> Result<SendVerificationCodeResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct VerifyCodeRequest {
        #[serde(rename = "code")]
        pub code: String,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct VerifyCodeResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// VerifyCode handles POST /verification/verify
    pub async fn verify_code(
        &self,
        _request: VerifyCodeRequest,
    ) -> Result<VerifyCodeResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct ScheduleVideoSessionRequest {
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
        #[serde(rename = "timeZone")]
        pub time_zone: String,
        #[serde(rename = "scheduledAt")]
        pub scheduled_at: time.Time,
    }

    #[derive(Debug, Deserialize)]
    pub struct ScheduleVideoSessionResponse {
        #[serde(rename = "session_id")]
        pub session_id: String,
    }

    /// ScheduleVideoSession handles POST /video/schedule
    pub async fn schedule_video_session(
        &self,
        _request: ScheduleVideoSessionRequest,
    ) -> Result<ScheduleVideoSessionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct StartVideoSessionRequest {
        #[serde(rename = "videoSessionId")]
        pub video_session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct StartVideoSessionResponse {
        #[serde(rename = "session_id")]
        pub session_id: String,
    }

    /// StartVideoSession handles POST /video/start
    pub async fn start_video_session(
        &self,
        _request: StartVideoSessionRequest,
    ) -> Result<StartVideoSessionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct CompleteVideoSessionRequest {
        #[serde(rename = "livenessScore")]
        pub liveness_score: f64,
        #[serde(rename = "notes")]
        pub notes: String,
        #[serde(rename = "verificationResult")]
        pub verification_result: String,
        #[serde(rename = "videoSessionId")]
        pub video_session_id: xid.ID,
        #[serde(rename = "livenessPassed")]
        pub liveness_passed: bool,
    }

    #[derive(Debug, Deserialize)]
    pub struct CompleteVideoSessionResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// CompleteVideoSession handles POST /video/complete (admin)
    pub async fn complete_video_session(
        &self,
        _request: CompleteVideoSessionRequest,
    ) -> Result<CompleteVideoSessionResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UploadDocumentRequest {
        #[serde(rename = "backImage")]
        pub back_image: String,
        #[serde(rename = "documentType")]
        pub document_type: String,
        #[serde(rename = "frontImage")]
        pub front_image: String,
        #[serde(rename = "selfie")]
        pub selfie: String,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct UploadDocumentResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// UploadDocument handles POST /documents/upload
    pub async fn upload_document(
        &self,
        _request: UploadDocumentRequest,
    ) -> Result<UploadDocumentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetDocumentVerificationResponse {
        #[serde(rename = "id")]
        pub id: String,
    }

    /// GetDocumentVerification handles GET /documents/:id
    pub async fn get_document_verification(
        &self,
    ) -> Result<GetDocumentVerificationResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct ReviewDocumentRequest {
        #[serde(rename = "rejectionReason")]
        pub rejection_reason: String,
        #[serde(rename = "approved")]
        pub approved: bool,
        #[serde(rename = "documentId")]
        pub document_id: xid.ID,
        #[serde(rename = "notes")]
        pub notes: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct ReviewDocumentResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// ReviewDocument handles POST /documents/:id/review (admin)
    pub async fn review_document(
        &self,
        _request: ReviewDocumentRequest,
    ) -> Result<ReviewDocumentResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct ListRecoverySessionsResponse {
        #[serde(rename = "sessions")]
        pub sessions: Vec<>,
    }

    /// ListRecoverySessions handles GET /admin/sessions (admin)
    pub async fn list_recovery_sessions(
        &self,
    ) -> Result<ListRecoverySessionsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct ApproveRecoveryRequest {
        #[serde(rename = "notes")]
        pub notes: String,
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
    }

    #[derive(Debug, Deserialize)]
    pub struct ApproveRecoveryResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// ApproveRecovery handles POST /admin/sessions/:id/approve (admin)
    pub async fn approve_recovery(
        &self,
        _request: ApproveRecoveryRequest,
    ) -> Result<ApproveRecoveryResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct RejectRecoveryRequest {
        #[serde(rename = "sessionId")]
        pub session_id: xid.ID,
        #[serde(rename = "notes")]
        pub notes: String,
        #[serde(rename = "reason")]
        pub reason: String,
    }

    #[derive(Debug, Deserialize)]
    pub struct RejectRecoveryResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// RejectRecovery handles POST /admin/sessions/:id/reject (admin)
    pub async fn reject_recovery(
        &self,
        _request: RejectRecoveryRequest,
    ) -> Result<RejectRecoveryResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetRecoveryStatsResponse {
        #[serde(rename = "stats")]
        pub stats: ,
    }

    /// GetRecoveryStats handles GET /admin/stats (admin)
    pub async fn get_recovery_stats(
        &self,
    ) -> Result<GetRecoveryStatsResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Deserialize)]
    pub struct GetRecoveryConfigResponse {
        #[serde(rename = "config")]
        pub config: ,
    }

    /// GetRecoveryConfig handles GET /admin/config (admin)
    pub async fn get_recovery_config(
        &self,
    ) -> Result<GetRecoveryConfigResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    #[derive(Debug, Serialize)]
    pub struct UpdateRecoveryConfigRequest {
        #[serde(rename = "riskScoreThreshold")]
        pub risk_score_threshold: f64,
        #[serde(rename = "enabledMethods")]
        pub enabled_methods: []RecoveryMethod,
        #[serde(rename = "minimumStepsRequired")]
        pub minimum_steps_required: i32,
        #[serde(rename = "requireAdminReview")]
        pub require_admin_review: bool,
        #[serde(rename = "requireMultipleSteps")]
        pub require_multiple_steps: bool,
    }

    #[derive(Debug, Deserialize)]
    pub struct UpdateRecoveryConfigResponse {
        #[serde(rename = "config")]
        pub config: ,
    }

    /// UpdateRecoveryConfig handles PUT /admin/config (admin)
    pub async fn update_recovery_config(
        &self,
        _request: UpdateRecoveryConfigRequest,
    ) -> Result<UpdateRecoveryConfigResponse> {{
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    /// HealthCheck handles GET /health
    pub async fn health_check(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for BackupauthPlugin {{
    fn id(&self) -> &str {
        "backupauth"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
