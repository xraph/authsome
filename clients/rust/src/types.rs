// Auto-generated Rust types

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InitiateChallengeRequest {
    #[serde(rename = "factorTypes")]
    pub factor_types: []FactorType,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "context")]
    pub context: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebAuthnConfig {
    #[serde(rename = "rp_display_name")]
    pub rp_display_name: String,
    #[serde(rename = "rp_id")]
    pub rp_id: String,
    #[serde(rename = "rp_origins")]
    pub rp_origins: []string,
    #[serde(rename = "timeout")]
    pub timeout: i32,
    #[serde(rename = "attestation_preference")]
    pub attestation_preference: String,
    #[serde(rename = "authenticator_selection")]
    pub authenticator_selection: ,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CodesResponse {
    #[serde(rename = "codes")]
    pub codes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UploadDocumentRequest {
    #[serde(rename = "selfie")]
    pub selfie: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
    #[serde(rename = "backImage")]
    pub back_image: String,
    #[serde(rename = "documentType")]
    pub document_type: String,
    #[serde(rename = "frontImage")]
    pub front_image: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderCheckResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreatePolicy_req {
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "title")]
    pub title: String,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "policyType")]
    pub policy_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VideoSessionResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataExportConfig {
    #[serde(rename = "allowedFormats")]
    pub allowed_formats: []string,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "expiryHours")]
    pub expiry_hours: i32,
    #[serde(rename = "includeSections")]
    pub include_sections: []string,
    #[serde(rename = "maxExportSize")]
    pub max_export_size: i64,
    #[serde(rename = "maxRequests")]
    pub max_requests: i32,
    #[serde(rename = "storagePath")]
    pub storage_path: String,
    #[serde(rename = "autoCleanup")]
    pub auto_cleanup: bool,
    #[serde(rename = "cleanupInterval")]
    pub cleanup_interval: time.Duration,
    #[serde(rename = "defaultFormat")]
    pub default_format: String,
    #[serde(rename = "requestPeriod")]
    pub request_period: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Status {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddTeamMember_req {
    #[serde(rename = "member_id")]
    pub member_id: xid.ID,
    #[serde(rename = "role")]
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Plugin {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*DashboardExtension>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLog {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTrainingResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetDocumentVerificationResponse {
    #[serde(rename = "rejectionReason")]
    pub rejection_reason: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "verifiedAt")]
    pub verified_at: *time.Time,
    #[serde(rename = "confidenceScore")]
    pub confidence_score: f64,
    #[serde(rename = "documentId")]
    pub document_id: xid.ID,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpRequirement {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "resource_type")]
    pub resource_type: String,
    #[serde(rename = "session_id")]
    pub session_id: String,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "amount")]
    pub amount: f64,
    #[serde(rename = "fulfilled_at")]
    pub fulfilled_at: *time.Time,
    #[serde(rename = "required_level")]
    pub required_level: SecurityLevel,
    #[serde(rename = "resource_action")]
    pub resource_action: String,
    #[serde(rename = "challenge_token")]
    pub challenge_token: String,
    #[serde(rename = "currency")]
    pub currency: String,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "risk_score")]
    pub risk_score: f64,
    #[serde(rename = "rule_name")]
    pub rule_name: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "current_level")]
    pub current_level: SecurityLevel,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "method")]
    pub method: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "route")]
    pub route: String,
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "expires_at")]
    pub expires_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminAddProviderRequest {
    #[serde(rename = "clientSecret")]
    pub client_secret: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "scopes")]
    pub scopes: []string,
    #[serde(rename = "appId")]
    pub app_id: xid.ID,
    #[serde(rename = "clientId")]
    pub client_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentPolicyResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiStepRecoveryConfig {
    #[serde(rename = "highRiskSteps")]
    pub high_risk_steps: []RecoveryMethod,
    #[serde(rename = "lowRiskSteps")]
    pub low_risk_steps: []RecoveryMethod,
    #[serde(rename = "mediumRiskSteps")]
    pub medium_risk_steps: []RecoveryMethod,
    #[serde(rename = "requireAdminApproval")]
    pub require_admin_approval: bool,
    #[serde(rename = "allowStepSkip")]
    pub allow_step_skip: bool,
    #[serde(rename = "allowUserChoice")]
    pub allow_user_choice: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "minimumSteps")]
    pub minimum_steps: i32,
    #[serde(rename = "sessionExpiry")]
    pub session_expiry: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentRecord {
    #[serde(rename = "granted")]
    pub granted: bool,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "grantedAt")]
    pub granted_at: time.Time,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "metadata")]
    pub metadata: JSONBMap,
    #[serde(rename = "revokedAt")]
    pub revoked_at: *time.Time,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "purpose")]
    pub purpose: String,
    #[serde(rename = "consentType")]
    pub consent_type: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationContext {
    #[serde(rename = "impersonation_id")]
    pub impersonation_id: *xid.ID,
    #[serde(rename = "impersonator_id")]
    pub impersonator_id: *xid.ID,
    #[serde(rename = "indicator_message")]
    pub indicator_message: String,
    #[serde(rename = "is_impersonating")]
    pub is_impersonating: bool,
    #[serde(rename = "target_user_id")]
    pub target_user_id: *xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProvidersResponse {
    #[serde(rename = "providers")]
    pub providers: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    #[serde(rename = "allowCrossPlatform")]
    pub allow_cross_platform: bool,
    #[serde(rename = "enableDeviceTracking")]
    pub enable_device_tracking: bool,
    #[serde(rename = "maxSessionsPerUser")]
    pub max_sessions_per_user: i32,
    #[serde(rename = "sessionExpiryHours")]
    pub session_expiry_hours: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionTokenResponse {
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrackNotificationEvent_req {
    #[serde(rename = "event")]
    pub event: String,
    #[serde(rename = "eventData", skip_serializing_if = "Option::is_none")]
    pub event_data: Option<>,
    #[serde(rename = "notificationId")]
    pub notification_id: String,
    #[serde(rename = "organizationId", skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<*string>,
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentPolicy {
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "consentType")]
    pub consent_type: String,
    #[serde(rename = "active")]
    pub active: bool,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "renewable")]
    pub renewable: bool,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "createdBy")]
    pub created_by: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "publishedAt")]
    pub published_at: *time.Time,
    #[serde(rename = "required")]
    pub required: bool,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "validityPeriod")]
    pub validity_period: *int,
    #[serde(rename = "metadata")]
    pub metadata: JSONBMap,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterClient_req {
    #[serde(rename = "redirect_uri")]
    pub redirect_uri: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LimitResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*time.Duration>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorVerificationRequest {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "data")]
    pub data: ,
    #[serde(rename = "factorId")]
    pub factor_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetDocumentVerificationRequest {
    #[serde(rename = "documentId")]
    pub document_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetupSecurityQuestionsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "setupAt")]
    pub setup_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentAuditLog {
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "consentId")]
    pub consent_id: String,
    #[serde(rename = "newValue")]
    pub new_value: JSONBMap,
    #[serde(rename = "previousValue")]
    pub previous_value: JSONBMap,
    #[serde(rename = "purpose")]
    pub purpose: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "consentType")]
    pub consent_type: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FinishRegister_body {
    #[serde(rename = "credential_id")]
    pub credential_id: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyEnrolledFactorRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "data")]
    pub data: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TOTPFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*twofa.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationStartResponse {
    #[serde(rename = "started_at")]
    pub started_at: String,
    #[serde(rename = "target_user_id")]
    pub target_user_id: String,
    #[serde(rename = "impersonator_id")]
    pub impersonator_id: String,
    #[serde(rename = "session_id")]
    pub session_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationChannels {
    #[serde(rename = "email")]
    pub email: bool,
    #[serde(rename = "slack")]
    pub slack: bool,
    #[serde(rename = "webhook")]
    pub webhook: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpEmailProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceViolation {
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "resolvedAt")]
    pub resolved_at: *time.Time,
    #[serde(rename = "resolvedBy")]
    pub resolved_by: String,
    #[serde(rename = "severity")]
    pub severity: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "violationType")]
    pub violation_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustedContactInfo {
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "phone")]
    pub phone: String,
    #[serde(rename = "relationship")]
    pub relationship: String,
    #[serde(rename = "verified")]
    pub verified: bool,
    #[serde(rename = "verifiedAt")]
    pub verified_at: *time.Time,
    #[serde(rename = "active")]
    pub active: bool,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentVerificationConfig {
    #[serde(rename = "acceptedDocuments")]
    pub accepted_documents: []string,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "encryptAtRest")]
    pub encrypt_at_rest: bool,
    #[serde(rename = "minConfidenceScore")]
    pub min_confidence_score: f64,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "requireSelfie")]
    pub require_selfie: bool,
    #[serde(rename = "retentionPeriod")]
    pub retention_period: time.Duration,
    #[serde(rename = "storageProvider")]
    pub storage_provider: String,
    #[serde(rename = "encryptionKey")]
    pub encryption_key: String,
    #[serde(rename = "requireBothSides")]
    pub require_both_sides: bool,
    #[serde(rename = "requireManualReview")]
    pub require_manual_review: bool,
    #[serde(rename = "storagePath")]
    pub storage_path: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationTemplateListResponse {
    #[serde(rename = "templates")]
    pub templates: Vec<>,
    #[serde(rename = "total")]
    pub total: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PreviewTemplate_req {
    #[serde(rename = "variables")]
    pub variables: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustedDevice {
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "deviceId")]
    pub device_id: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "lastUsedAt")]
    pub last_used_at: *time.Time,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCConfigResponse {
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "issuer")]
    pub issuer: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCErrorResponse {
    #[serde(rename = "error_description")]
    pub error_description: String,
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialLinkResponse {
    #[serde(rename = "linked")]
    pub linked: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListUsersRequest {
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "-")]
    pub -: xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "search")]
    pub search: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BeginLogin_body {
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationListResponse {
    #[serde(rename = "verifications")]
    pub verifications: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DashboardExtension {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*dashboard.ExtensionRegistry>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Status_body {
    #[serde(rename = "device_id")]
    pub device_id: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationResponse {
    #[serde(rename = "verification")]
    pub verification: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceReport {
    #[serde(rename = "period")]
    pub period: String,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "fileSize")]
    pub file_size: i64,
    #[serde(rename = "generatedBy")]
    pub generated_by: String,
    #[serde(rename = "reportType")]
    pub report_type: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "summary")]
    pub summary: ,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "fileUrl")]
    pub file_url: String,
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryConfiguration {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]RecoveryMethod>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpVerificationResponse {
    #[serde(rename = "expires_at")]
    pub expires_at: String,
    #[serde(rename = "verified")]
    pub verified: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimeBasedRule {
    #[serde(rename = "operation")]
    pub operation: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "max_age")]
    pub max_age: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*audit.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyEngine {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendVerificationCodeResponse {
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "maskedTarget")]
    pub masked_target: String,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "sent")]
    pub sent: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnfidoProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<OnfidoConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct App {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceDashboardResponse {
    #[serde(rename = "metrics")]
    pub metrics: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTraining_req {
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "trainingType")]
    pub training_type: String,
    #[serde(rename = "userId")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustedContact {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SSOErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SSOInitResponse {
    #[serde(rename = "redirect_url")]
    pub redirect_url: String,
    #[serde(rename = "request_id")]
    pub request_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockUserService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyStore {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceViolationResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*coreapp.ServiceImpl>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorEnrollmentRequest {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "priority")]
    pub priority: FactorPriority,
    #[serde(rename = "type")]
    pub type: FactorType,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRecoveryStatsRequest {
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "startDate")]
    pub start_date: time.Time,
    #[serde(rename = "endDate")]
    pub end_date: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteRecoveryRequest {
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetadataResponse {
    #[serde(rename = "metadata")]
    pub metadata: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpVerificationsResponse {
    #[serde(rename = "verifications")]
    pub verifications: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenResponse {
    #[serde(rename = "refresh_token")]
    pub refresh_token: String,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "token_type")]
    pub token_type: String,
    #[serde(rename = "access_token")]
    pub access_token: String,
    #[serde(rename = "expires_in")]
    pub expires_in: i32,
    #[serde(rename = "id_token")]
    pub id_token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutoCleanupConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "interval")]
    pub interval: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationTemplateResponse {
    #[serde(rename = "template")]
    pub template: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListUsersResponse {
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "total")]
    pub total: i32,
    #[serde(rename = "total_pages")]
    pub total_pages: i32,
    #[serde(rename = "users")]
    pub users: []*user.User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<error>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentExpiryConfig {
    #[serde(rename = "allowRenewal")]
    pub allow_renewal: bool,
    #[serde(rename = "autoExpireCheck")]
    pub auto_expire_check: bool,
    #[serde(rename = "defaultValidityDays")]
    pub default_validity_days: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "expireCheckInterval")]
    pub expire_check_interval: time.Duration,
    #[serde(rename = "renewalReminderDays")]
    pub renewal_reminder_days: i32,
    #[serde(rename = "requireReConsent")]
    pub require_re_consent: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CallbackResponse {
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RequestTrustedContactVerificationRequest {
    #[serde(rename = "contactId")]
    pub contact_id: xid.ID,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetupSecurityQuestionRequest {
    #[serde(rename = "answer")]
    pub answer: String,
    #[serde(rename = "customText")]
    pub custom_text: String,
    #[serde(rename = "questionId")]
    pub question_id: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockImpersonationRepository {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JumioConfig {
    #[serde(rename = "callbackUrl")]
    pub callback_url: String,
    #[serde(rename = "dataCenter")]
    pub data_center: String,
    #[serde(rename = "enableExtraction")]
    pub enable_extraction: bool,
    #[serde(rename = "enableLiveness")]
    pub enable_liveness: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "apiSecret")]
    pub api_secret: String,
    #[serde(rename = "enableAMLScreening")]
    pub enable_a_m_l_screening: bool,
    #[serde(rename = "enabledCountries")]
    pub enabled_countries: []string,
    #[serde(rename = "enabledDocumentTypes")]
    pub enabled_document_types: []string,
    #[serde(rename = "presetId")]
    pub preset_id: String,
    #[serde(rename = "verificationType")]
    pub verification_type: String,
    #[serde(rename = "apiToken")]
    pub api_token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskContext {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateConsentRequest {
    #[serde(rename = "userId")]
    pub user_id: String,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct testContext {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*http.Request>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateProfileRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*string>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResetUserMFAResponse {
    #[serde(rename = "devicesRevoked")]
    pub devices_revoked: i32,
    #[serde(rename = "factorsReset")]
    pub factors_reset: i32,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthContactResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StripeIdentityConfig {
    #[serde(rename = "returnUrl")]
    pub return_url: String,
    #[serde(rename = "useMock")]
    pub use_mock: bool,
    #[serde(rename = "webhookSecret")]
    pub webhook_secret: String,
    #[serde(rename = "allowedTypes")]
    pub allowed_types: []string,
    #[serde(rename = "apiKey")]
    pub api_key: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "requireLiveCapture")]
    pub require_live_capture: bool,
    #[serde(rename = "requireMatchingSelfie")]
    pub require_matching_selfie: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSessionErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceChecksResponse {
    #[serde(rename = "checks")]
    pub checks: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnonymousSignInResponse {
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthQuestionsResponse {
    #[serde(rename = "questions")]
    pub questions: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StripeIdentityProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateProvider_req {
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "isActive")]
    pub is_active: bool,
    #[serde(rename = "isDefault")]
    pub is_default: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteTraining_req {
    #[serde(rename = "score")]
    pub score: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpVideoProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContextRule {
    #[serde(rename = "condition")]
    pub condition: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentTypesResponse {
    #[serde(rename = "document_types")]
    pub document_types: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFAStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateBackupCodes_body {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessment {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "recommended")]
    pub recommended: []FactorType,
    #[serde(rename = "score")]
    pub score: f64,
    #[serde(rename = "factors")]
    pub factors: []string,
    #[serde(rename = "level")]
    pub level: RiskLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookConfig {
    #[serde(rename = "expiry_warning_days")]
    pub expiry_warning_days: i32,
    #[serde(rename = "notify_on_created")]
    pub notify_on_created: bool,
    #[serde(rename = "notify_on_deleted")]
    pub notify_on_deleted: bool,
    #[serde(rename = "notify_on_expiring")]
    pub notify_on_expiring: bool,
    #[serde(rename = "notify_on_rate_limit")]
    pub notify_on_rate_limit: bool,
    #[serde(rename = "notify_on_rotated")]
    pub notify_on_rotated: bool,
    #[serde(rename = "webhook_urls")]
    pub webhook_urls: []string,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryAttemptLog {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Disable_body {
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityQuestion {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartVideoSessionRequest {
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationContext {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "notifications")]
    pub notifications: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SMSFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*phone.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DefaultProviderRegistry {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<NotificationProvider>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthCodesResponse {
    #[serde(rename = "codes")]
    pub codes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UsernameErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteFactorRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyCodeRequest {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetupSecurityQuestionsRequest {
    #[serde(rename = "questions")]
    pub questions: []SetupSecurityQuestionRequest,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentSummary {
    #[serde(rename = "hasPendingExport")]
    pub has_pending_export: bool,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "consentsByType")]
    pub consents_by_type: ,
    #[serde(rename = "expiredConsents")]
    pub expired_consents: i32,
    #[serde(rename = "hasPendingDeletion")]
    pub has_pending_deletion: bool,
    #[serde(rename = "lastConsentUpdate")]
    pub last_consent_update: *time.Time,
    #[serde(rename = "pendingRenewals")]
    pub pending_renewals: i32,
    #[serde(rename = "revokedConsents")]
    pub revoked_consents: i32,
    #[serde(rename = "totalConsents")]
    pub total_consents: i32,
    #[serde(rename = "grantedConsents")]
    pub granted_consents: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationsConfig {
    #[serde(rename = "auditReminders")]
    pub audit_reminders: bool,
    #[serde(rename = "channels")]
    pub channels: NotificationChannels,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "failedChecks")]
    pub failed_checks: bool,
    #[serde(rename = "notifyComplianceContact")]
    pub notify_compliance_contact: bool,
    #[serde(rename = "notifyOwners")]
    pub notify_owners: bool,
    #[serde(rename = "violations")]
    pub violations: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "factors")]
    pub factors: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartRecoveryRequest {
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "deviceId")]
    pub device_id: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "preferredMethod")]
    pub preferred_method: RecoveryMethod,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Adapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*TemplateService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialCallbackResponse {
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignInRequest {
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "redirectUrl")]
    pub redirect_url: String,
    #[serde(rename = "scopes")]
    pub scopes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentVerificationResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasskeyErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyPair {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*rsa.PublicKey>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MagicLinkVerifyResponse {
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpAuditLog {
    #[serde(rename = "event_type")]
    pub event_type: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "event_data")]
    pub event_data: ,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "severity")]
    pub severity: String,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelsResponse {
    #[serde(rename = "channels")]
    pub channels: ,
    #[serde(rename = "count")]
    pub count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignIn_body {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "phone")]
    pub phone: String,
    #[serde(rename = "remember")]
    pub remember: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetActive_body {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpRequirementsResponse {
    #[serde(rename = "requirements")]
    pub requirements: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationStatusResponse {
    #[serde(rename = "status")]
    pub status: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockEmailService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]*Email>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTrustedDevicesResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "devices")]
    pub devices: []TrustedDevice,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimitConfig {
    #[serde(rename = "window_minutes")]
    pub window_minutes: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "lockout_minutes")]
    pub lockout_minutes: i32,
    #[serde(rename = "max_attempts")]
    pub max_attempts: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyCodeResponse {
    #[serde(rename = "valid")]
    pub valid: bool,
    #[serde(rename = "attemptsLeft")]
    pub attempts_left: i32,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UsernameStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailOTPVerifyResponse {
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceReportResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdatePolicy_req {
    #[serde(rename = "content")]
    pub content: *string,
    #[serde(rename = "status")]
    pub status: *string,
    #[serde(rename = "title")]
    pub title: *string,
    #[serde(rename = "version")]
    pub version: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateFactorRequest {
    #[serde(rename = "priority")]
    pub priority: *FactorPriority,
    #[serde(rename = "status")]
    pub status: *FactorStatus,
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContinueRecoveryRequest {
    #[serde(rename = "method")]
    pub method: RecoveryMethod,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CookieConsent {
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "marketing")]
    pub marketing: bool,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "analytics")]
    pub analytics: bool,
    #[serde(rename = "essential")]
    pub essential: bool,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "consentBannerVersion")]
    pub consent_banner_version: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "personalization")]
    pub personalization: bool,
    #[serde(rename = "sessionId")]
    pub session_id: String,
    #[serde(rename = "thirdParty")]
    pub third_party: bool,
    #[serde(rename = "functional")]
    pub functional: bool,
    #[serde(rename = "id")]
    pub id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTrustedContactsResponse {
    #[serde(rename = "contacts")]
    pub contacts: []TrustedContactInfo,
    #[serde(rename = "count")]
    pub count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthDocumentResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentAuditConfig {
    #[serde(rename = "signLogs")]
    pub sign_logs: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "immutable")]
    pub immutable: bool,
    #[serde(rename = "logAllChanges")]
    pub log_all_changes: bool,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "archiveInterval")]
    pub archive_interval: time.Duration,
    #[serde(rename = "archiveOldLogs")]
    pub archive_old_logs: bool,
    #[serde(rename = "exportFormat")]
    pub export_format: String,
    #[serde(rename = "logIpAddress")]
    pub log_ip_address: bool,
    #[serde(rename = "logUserAgent")]
    pub log_user_agent: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpAuditLogsResponse {
    #[serde(rename = "audit_logs")]
    pub audit_logs: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutomatedChecksConfig {
    #[serde(rename = "dataRetention")]
    pub data_retention: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "inactiveUsers")]
    pub inactive_users: bool,
    #[serde(rename = "mfaCoverage")]
    pub mfa_coverage: bool,
    #[serde(rename = "passwordPolicy")]
    pub password_policy: bool,
    #[serde(rename = "sessionPolicy")]
    pub session_policy: bool,
    #[serde(rename = "accessReview")]
    pub access_review: bool,
    #[serde(rename = "checkInterval")]
    pub check_interval: time.Duration,
    #[serde(rename = "suspiciousActivity")]
    pub suspicious_activity: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CallbackResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthVideoResponse {
    #[serde(rename = "session_id")]
    pub session_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookPayload {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteVideoSessionRequest {
    #[serde(rename = "livenessPassed")]
    pub liveness_passed: bool,
    #[serde(rename = "livenessScore")]
    pub liveness_score: f64,
    #[serde(rename = "notes")]
    pub notes: String,
    #[serde(rename = "verificationResult")]
    pub verification_result: String,
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SMSVerificationConfig {
    #[serde(rename = "codeLength")]
    pub code_length: i32,
    #[serde(rename = "cooldownPeriod")]
    pub cooldown_period: time.Duration,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "maxAttempts")]
    pub max_attempts: i32,
    #[serde(rename = "maxSmsPerDay")]
    pub max_sms_per_day: i32,
    #[serde(rename = "messageTemplate")]
    pub message_template: String,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "codeExpiry")]
    pub code_expiry: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFAEnableResponse {
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "totp_uri")]
    pub totp_uri: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailProviderConfig {
    #[serde(rename = "reply_to")]
    pub reply_to: String,
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "from")]
    pub from: String,
    #[serde(rename = "from_name")]
    pub from_name: String,
    #[serde(rename = "provider")]
    pub provider: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SaveNotificationSettings_req {
    #[serde(rename = "retryDelay")]
    pub retry_delay: String,
    #[serde(rename = "autoSendWelcome")]
    pub auto_send_welcome: bool,
    #[serde(rename = "cleanupAfter")]
    pub cleanup_after: String,
    #[serde(rename = "retryAttempts")]
    pub retry_attempts: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminListProvidersRequest {
    #[serde(rename = "appId")]
    pub app_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScopeInfo {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTemplatesResponse {
    #[serde(rename = "templates")]
    pub templates: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockUserService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]*User>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListFactorsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "factors")]
    pub factors: []Factor,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluateRequest {
    #[serde(rename = "amount")]
    pub amount: f64,
    #[serde(rename = "currency")]
    pub currency: String,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTemplateVersion_req {
    #[serde(rename = "changes")]
    pub changes: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SuccessResponse {
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Email {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeTrustedDeviceRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CookieConsentConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "requireExplicit")]
    pub require_explicit: bool,
    #[serde(rename = "validityPeriod")]
    pub validity_period: time.Duration,
    #[serde(rename = "allowAnonymous")]
    pub allow_anonymous: bool,
    #[serde(rename = "bannerVersion")]
    pub banner_version: String,
    #[serde(rename = "categories")]
    pub categories: []string,
    #[serde(rename = "defaultStyle")]
    pub default_style: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateConsentRequest {
    #[serde(rename = "granted")]
    pub granted: *bool,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWTService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockSessionService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RolesResponse {
    #[serde(rename = "roles")]
    pub roles: []*apikey.Role,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AMLMatch {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateVerificationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BunRepository {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*bun.DB>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessmentConfig {
    #[serde(rename = "mediumRiskThreshold")]
    pub medium_risk_threshold: f64,
    #[serde(rename = "newDeviceWeight")]
    pub new_device_weight: f64,
    #[serde(rename = "velocityWeight")]
    pub velocity_weight: f64,
    #[serde(rename = "blockHighRisk")]
    pub block_high_risk: bool,
    #[serde(rename = "historyWeight")]
    pub history_weight: f64,
    #[serde(rename = "newIpWeight")]
    pub new_ip_weight: f64,
    #[serde(rename = "newLocationWeight")]
    pub new_location_weight: f64,
    #[serde(rename = "requireReviewAbove")]
    pub require_review_above: f64,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "highRiskThreshold")]
    pub high_risk_threshold: f64,
    #[serde(rename = "lowRiskThreshold")]
    pub low_risk_threshold: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentExportResponse {
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceStatus {
    #[serde(rename = "lastChecked")]
    pub last_checked: time.Time,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "violations")]
    pub violations: i32,
    #[serde(rename = "checksFailed")]
    pub checks_failed: i32,
    #[serde(rename = "checksPassed")]
    pub checks_passed: i32,
    #[serde(rename = "checksWarning")]
    pub checks_warning: i32,
    #[serde(rename = "nextAudit")]
    pub next_audit: time.Time,
    #[serde(rename = "overallStatus")]
    pub overall_status: String,
    #[serde(rename = "score")]
    pub score: i32,
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityQuestionInfo {
    #[serde(rename = "questionText")]
    pub question_text: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "isCustom")]
    pub is_custom: bool,
    #[serde(rename = "questionId")]
    pub question_id: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceProfileResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskEngine {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*repository.MFARepository>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryCodesConfig {
    #[serde(rename = "allowDownload")]
    pub allow_download: bool,
    #[serde(rename = "allowPrint")]
    pub allow_print: bool,
    #[serde(rename = "autoRegenerate")]
    pub auto_regenerate: bool,
    #[serde(rename = "codeCount")]
    pub code_count: i32,
    #[serde(rename = "codeLength")]
    pub code_length: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "regenerateCount")]
    pub regenerate_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct auditServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*audit.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminUpdateProviderRequest {
    #[serde(rename = "clientId")]
    pub client_id: *string,
    #[serde(rename = "clientSecret")]
    pub client_secret: *string,
    #[serde(rename = "enabled")]
    pub enabled: *bool,
    #[serde(rename = "scopes")]
    pub scopes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnbanUser_reqBody {
    #[serde(rename = "reason", skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteVideoSessionResponse {
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
    #[serde(rename = "completedAt")]
    pub completed_at: time.Time,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "result")]
    pub result: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteRecoveryResponse {
    #[serde(rename = "completedAt")]
    pub completed_at: time.Time,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
    #[serde(rename = "status")]
    pub status: RecoveryStatus,
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRecoveryConfigResponse {
    #[serde(rename = "requireMultipleSteps")]
    pub require_multiple_steps: bool,
    #[serde(rename = "riskScoreThreshold")]
    pub risk_score_threshold: f64,
    #[serde(rename = "enabledMethods")]
    pub enabled_methods: []RecoveryMethod,
    #[serde(rename = "minimumStepsRequired")]
    pub minimum_steps_required: i32,
    #[serde(rename = "requireAdminReview")]
    pub require_admin_review: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpRememberedDevice {
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "device_name")]
    pub device_name: String,
    #[serde(rename = "expires_at")]
    pub expires_at: time.Time,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "last_used_at")]
    pub last_used_at: time.Time,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "remembered_at")]
    pub remembered_at: time.Time,
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "device_id")]
    pub device_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyRecoveryCodeRequest {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListRecoverySessionsRequest {
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "requiresReview")]
    pub requires_review: bool,
    #[serde(rename = "status")]
    pub status: RecoveryStatus,
}

/// Webhook configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Webhook {
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "url")]
    pub url: String,
    #[serde(rename = "events")]
    pub events: Vec<String>,
    #[serde(rename = "secret")]
    pub secret: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockOrganizationService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTemplate {
    #[serde(rename = "dataResidency")]
    pub data_residency: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "passwordMinLength")]
    pub password_min_length: i32,
    #[serde(rename = "requiredTraining")]
    pub required_training: []string,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "auditFrequencyDays")]
    pub audit_frequency_days: i32,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "mfaRequired")]
    pub mfa_required: bool,
    #[serde(rename = "requiredPolicies")]
    pub required_policies: []string,
    #[serde(rename = "sessionMaxAge")]
    pub session_max_age: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*emailotp.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminBypassRequest {
    #[serde(rename = "duration")]
    pub duration: i32,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthConfigResponse {
    #[serde(rename = "config")]
    pub config: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentStats {
    #[serde(rename = "averageLifetime")]
    pub average_lifetime: i32,
    #[serde(rename = "expiredCount")]
    pub expired_count: i32,
    #[serde(rename = "grantRate")]
    pub grant_rate: f64,
    #[serde(rename = "grantedCount")]
    pub granted_count: i32,
    #[serde(rename = "revokedCount")]
    pub revoked_count: i32,
    #[serde(rename = "totalConsents")]
    pub total_consents: i32,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailOTPSendResponse {
    #[serde(rename = "dev_otp")]
    pub dev_otp: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentDecision {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnonymousAuthResponse {
    #[serde(rename = "user")]
    pub user: ,
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SSOProviderResponse {
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TeamsResponse {
    #[serde(rename = "teams")]
    pub teams: []*organization.Team,
    #[serde(rename = "total")]
    pub total: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationWebhookResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceEvidencesResponse {
    #[serde(rename = "evidence")]
    pub evidence: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LinkAccountRequest {
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "scopes")]
    pub scopes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApproveRecoveryResponse {
    #[serde(rename = "approved")]
    pub approved: bool,
    #[serde(rename = "approvedAt")]
    pub approved_at: time.Time,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthCheckResponse {
    #[serde(rename = "enabledMethods")]
    pub enabled_methods: []RecoveryMethod,
    #[serde(rename = "healthy")]
    pub healthy: bool,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "providersStatus")]
    pub providers_status: ,
    #[serde(rename = "version")]
    pub version: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataDeletionRequest {
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "deleteSections")]
    pub delete_sections: []string,
    #[serde(rename = "retentionExempt")]
    pub retention_exempt: bool,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "approvedAt")]
    pub approved_at: *time.Time,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "rejectedAt")]
    pub rejected_at: *time.Time,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "archivePath")]
    pub archive_path: String,
    #[serde(rename = "requestReason")]
    pub request_reason: String,
    #[serde(rename = "approvedBy")]
    pub approved_by: String,
    #[serde(rename = "errorMessage")]
    pub error_message: String,
    #[serde(rename = "exemptionReason")]
    pub exemption_reason: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateProfileRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFASession {
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "riskLevel")]
    pub risk_level: RiskLevel,
    #[serde(rename = "sessionToken")]
    pub session_token: String,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "factorsRequired")]
    pub factors_required: i32,
    #[serde(rename = "factorsVerified")]
    pub factors_verified: i32,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "verifiedFactors")]
    pub verified_factors: []xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Link_body {
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "password")]
    pub password: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BeginRegister_body {
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteRule {
    #[serde(rename = "pattern")]
    pub pattern: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "method")]
    pub method: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateVerificationSession_req {
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskFactor {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationRequest {
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
    #[serde(rename = "rememberDevice")]
    pub remember_device: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChallengeRequest {
    #[serde(rename = "context")]
    pub context: String,
    #[serde(rename = "factorTypes")]
    pub factor_types: []FactorType,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BanUserRequest {
    #[serde(rename = "expires_at")]
    pub expires_at: *time.Time,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "user_id")]
    pub user_id: xid.ID,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "-")]
    pub -: xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifySecurityAnswersResponse {
    #[serde(rename = "valid")]
    pub valid: bool,
    #[serde(rename = "attemptsLeft")]
    pub attempts_left: i32,
    #[serde(rename = "correctAnswers")]
    pub correct_answers: i32,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "requiredAnswers")]
    pub required_answers: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentAuditLogsResponse {
    #[serde(rename = "audit_logs")]
    pub audit_logs: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDTokenClaims {
    #[serde(rename = "email_verified")]
    pub email_verified: bool,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "nonce")]
    pub nonce: String,
    #[serde(rename = "session_state")]
    pub session_state: String,
    #[serde(rename = "auth_time")]
    pub auth_time: i64,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "family_name")]
    pub family_name: String,
    #[serde(rename = "given_name")]
    pub given_name: String,
    #[serde(rename = "preferred_username")]
    pub preferred_username: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendCode_body {
    #[serde(rename = "phone")]
    pub phone: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnableRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthResponse {
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
    #[serde(rename = "session")]
    pub session: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceUserTrainingResponse {
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpPolicyResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCTokenResponse {
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "token_type")]
    pub token_type: String,
    #[serde(rename = "access_token")]
    pub access_token: String,
    #[serde(rename = "expires_in")]
    pub expires_in: i32,
    #[serde(rename = "id_token")]
    pub id_token: String,
    #[serde(rename = "refresh_token")]
    pub refresh_token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrganizationHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*organization.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScheduleVideoSessionRequest {
    #[serde(rename = "timeZone")]
    pub time_zone: String,
    #[serde(rename = "scheduledAt")]
    pub scheduled_at: time.Time,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentTypeStatus {
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "granted")]
    pub granted: bool,
    #[serde(rename = "grantedAt")]
    pub granted_at: time.Time,
    #[serde(rename = "needsRenewal")]
    pub needs_renewal: bool,
    #[serde(rename = "type")]
    pub type: String,
    #[serde(rename = "version")]
    pub version: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustedDevicesConfig {
    #[serde(rename = "default_expiry_days")]
    pub default_expiry_days: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "max_devices_per_user")]
    pub max_devices_per_user: i32,
    #[serde(rename = "max_expiry_days")]
    pub max_expiry_days: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationMiddleware {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<Config>,
}

/// User account
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    #[serde(rename = "emailVerified")]
    pub email_verified: bool,
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: String,
    #[serde(rename = "organizationId", skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<String>,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "name", skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnrollFactorRequest {
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "priority")]
    pub priority: FactorPriority,
    #[serde(rename = "type")]
    pub type: FactorType,
    #[serde(rename = "metadata")]
    pub metadata: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateDPARequest {
    #[serde(rename = "agreementType")]
    pub agreement_type: String,
    #[serde(rename = "expiryDate")]
    pub expiry_date: *time.Time,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "signedByEmail")]
    pub signed_by_email: String,
    #[serde(rename = "signedByTitle")]
    pub signed_by_title: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "effectiveDate")]
    pub effective_date: time.Time,
    #[serde(rename = "signedByName")]
    pub signed_by_name: String,
    #[serde(rename = "version")]
    pub version: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentCookieResponse {
    #[serde(rename = "preferences")]
    pub preferences: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TeamHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*app.ServiceImpl>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetFactorRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListSessionsRequest {
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "user_id")]
    pub user_id: *xid.ID,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "-")]
    pub -: xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataDeletionRequestInput {
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "deleteSections")]
    pub delete_sections: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UsernameSignInResponse {
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWK {
    #[serde(rename = "alg")]
    pub alg: String,
    #[serde(rename = "e")]
    pub e: String,
    #[serde(rename = "kid")]
    pub kid: String,
    #[serde(rename = "kty")]
    pub kty: String,
    #[serde(rename = "n")]
    pub n: String,
    #[serde(rename = "use")]
    pub use: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyResponse {
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
    #[serde(rename = "session")]
    pub session: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhoneErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhoneVerifyResponse {
    #[serde(rename = "user")]
    pub user: ,
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetUserRoleRequest {
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "user_id")]
    pub user_id: xid.ID,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "-")]
    pub -: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecurityQuestionsResponse {
    #[serde(rename = "questions")]
    pub questions: []SecurityQuestionInfo,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScheduleVideoSessionResponse {
    #[serde(rename = "instructions")]
    pub instructions: String,
    #[serde(rename = "joinUrl")]
    pub join_url: String,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "scheduledAt")]
    pub scheduled_at: time.Time,
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<error>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FacialCheckConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "motionCapture")]
    pub motion_capture: bool,
    #[serde(rename = "variant")]
    pub variant: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionsResponse {
    #[serde(rename = "sessions")]
    pub sessions: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BaseFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustedContactsConfig {
    #[serde(rename = "allowPhoneContacts")]
    pub allow_phone_contacts: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "verificationExpiry")]
    pub verification_expiry: time.Duration,
    #[serde(rename = "cooldownPeriod")]
    pub cooldown_period: time.Duration,
    #[serde(rename = "maxNotificationsPerDay")]
    pub max_notifications_per_day: i32,
    #[serde(rename = "maximumContacts")]
    pub maximum_contacts: i32,
    #[serde(rename = "minimumContacts")]
    pub minimum_contacts: i32,
    #[serde(rename = "requireVerification")]
    pub require_verification: bool,
    #[serde(rename = "requiredToRecover")]
    pub required_to_recover: i32,
    #[serde(rename = "allowEmailContacts")]
    pub allow_email_contacts: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpDocumentProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentNotificationsConfig {
    #[serde(rename = "notifyDeletionApproved")]
    pub notify_deletion_approved: bool,
    #[serde(rename = "notifyDeletionComplete")]
    pub notify_deletion_complete: bool,
    #[serde(rename = "notifyDpoEmail")]
    pub notify_dpo_email: String,
    #[serde(rename = "notifyExportReady")]
    pub notify_export_ready: bool,
    #[serde(rename = "notifyOnExpiry")]
    pub notify_on_expiry: bool,
    #[serde(rename = "channels")]
    pub channels: []string,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "notifyOnGrant")]
    pub notify_on_grant: bool,
    #[serde(rename = "notifyOnRevoke")]
    pub notify_on_revoke: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationEndResponse {
    #[serde(rename = "ended_at")]
    pub ended_at: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListChecksFilter {
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "sinceBefore")]
    pub since_before: *time.Time,
    #[serde(rename = "status")]
    pub status: *string,
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "checkType")]
    pub check_type: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentReport {
    #[serde(rename = "usersWithConsent")]
    pub users_with_consent: i32,
    #[serde(rename = "consentsByType")]
    pub consents_by_type: ,
    #[serde(rename = "dpasActive")]
    pub dpas_active: i32,
    #[serde(rename = "pendingDeletions")]
    pub pending_deletions: i32,
    #[serde(rename = "reportPeriodEnd")]
    pub report_period_end: time.Time,
    #[serde(rename = "totalUsers")]
    pub total_users: i32,
    #[serde(rename = "completedDeletions")]
    pub completed_deletions: i32,
    #[serde(rename = "consentRate")]
    pub consent_rate: f64,
    #[serde(rename = "dataExportsThisPeriod")]
    pub data_exports_this_period: i32,
    #[serde(rename = "dpasExpiringSoon")]
    pub dpas_expiring_soon: i32,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "reportPeriodStart")]
    pub report_period_start: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationWebhookResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChallengeResponse {
    #[serde(rename = "availableFactors")]
    pub available_factors: []FactorInfo,
    #[serde(rename = "challengeId")]
    pub challenge_id: xid.ID,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "factorsRequired")]
    pub factors_required: i32,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonateUser_reqBody {
    #[serde(rename = "duration", skip_serializing_if = "Option::is_none")]
    pub duration: Option<time.Duration>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RequestTrustedContactVerificationResponse {
    #[serde(rename = "contactId")]
    pub contact_id: xid.ID,
    #[serde(rename = "contactName")]
    pub contact_name: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "notifiedAt")]
    pub notified_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceEvidenceResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminPolicyRequest {
    #[serde(rename = "allowedTypes")]
    pub allowed_types: []string,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "gracePeriod")]
    pub grace_period: i32,
    #[serde(rename = "requiredFactors")]
    pub required_factors: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorAdapterRegistry {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RequestReverification_req {
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunCheck_req {
    #[serde(rename = "checkType")]
    pub check_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFAStatus {
    #[serde(rename = "requiredCount")]
    pub required_count: i32,
    #[serde(rename = "trustedDevice")]
    pub trusted_device: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "enrolledFactors")]
    pub enrolled_factors: []FactorInfo,
    #[serde(rename = "gracePeriod")]
    pub grace_period: *time.Time,
    #[serde(rename = "policyActive")]
    pub policy_active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyTrustedContactRequest {
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpSMSProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RetentionConfig {
    #[serde(rename = "gracePeriodDays")]
    pub grace_period_days: i32,
    #[serde(rename = "purgeSchedule")]
    pub purge_schedule: String,
    #[serde(rename = "archiveBeforePurge")]
    pub archive_before_purge: bool,
    #[serde(rename = "archivePath")]
    pub archive_path: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdaptiveMFAConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "factor_ip_reputation")]
    pub factor_ip_reputation: bool,
    #[serde(rename = "factor_location_change")]
    pub factor_location_change: bool,
    #[serde(rename = "factor_new_device")]
    pub factor_new_device: bool,
    #[serde(rename = "factor_velocity")]
    pub factor_velocity: bool,
    #[serde(rename = "location_change_risk")]
    pub location_change_risk: f64,
    #[serde(rename = "require_step_up_threshold")]
    pub require_step_up_threshold: f64,
    #[serde(rename = "velocity_risk")]
    pub velocity_risk: f64,
    #[serde(rename = "new_device_risk")]
    pub new_device_risk: f64,
    #[serde(rename = "risk_threshold")]
    pub risk_threshold: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartVideoSessionResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "sessionUrl")]
    pub session_url: String,
    #[serde(rename = "startedAt")]
    pub started_at: time.Time,
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VideoVerificationConfig {
    #[serde(rename = "minScheduleAdvance")]
    pub min_schedule_advance: time.Duration,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "sessionDuration")]
    pub session_duration: time.Duration,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "livenessThreshold")]
    pub liveness_threshold: f64,
    #[serde(rename = "recordSessions")]
    pub record_sessions: bool,
    #[serde(rename = "recordingRetention")]
    pub recording_retention: time.Duration,
    #[serde(rename = "requireAdminReview")]
    pub require_admin_review: bool,
    #[serde(rename = "requireLivenessCheck")]
    pub require_liveness_check: bool,
    #[serde(rename = "requireScheduling")]
    pub require_scheduling: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateEvidence_req {
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyFactor_req {
    #[serde(rename = "code")]
    pub code: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasskeyLoginOptionsResponse {
    #[serde(rename = "options")]
    pub options: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationResult {
    #[serde(rename = "current_level")]
    pub current_level: SecurityLevel,
    #[serde(rename = "expires_at")]
    pub expires_at: time.Time,
    #[serde(rename = "grace_period_ends_at")]
    pub grace_period_ends_at: time.Time,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "allowed_methods")]
    pub allowed_methods: []VerificationMethod,
    #[serde(rename = "can_remember")]
    pub can_remember: bool,
    #[serde(rename = "challenge_token")]
    pub challenge_token: String,
    #[serde(rename = "matched_rules")]
    pub matched_rules: []string,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "required")]
    pub required: bool,
    #[serde(rename = "requirement_id")]
    pub requirement_id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSessionSetActiveResponse {
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyTrustedContactResponse {
    #[serde(rename = "contactId")]
    pub contact_id: xid.ID,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "verified")]
    pub verified: bool,
    #[serde(rename = "verifiedAt")]
    pub verified_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailConfig {
    #[serde(rename = "code_length")]
    pub code_length: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "rate_limit")]
    pub rate_limit: *RateLimitConfig,
    #[serde(rename = "template_id")]
    pub template_id: String,
    #[serde(rename = "code_expiry_minutes")]
    pub code_expiry_minutes: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendVerificationCodeRequest {
    #[serde(rename = "method")]
    pub method: RecoveryMethod,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
    #[serde(rename = "target")]
    pub target: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignUp_body {
    #[serde(rename = "password")]
    pub password: String,
    #[serde(rename = "username")]
    pub username: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpEvaluationResponse {
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "required")]
    pub required: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCUserInfoResponse {
    #[serde(rename = "email_verified")]
    pub email_verified: bool,
    #[serde(rename = "family_name")]
    pub family_name: String,
    #[serde(rename = "given_name")]
    pub given_name: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "picture")]
    pub picture: String,
    #[serde(rename = "sub")]
    pub sub: String,
    #[serde(rename = "email")]
    pub email: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTrainingFilter {
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "standard")]
    pub standard: *ComplianceStandard,
    #[serde(rename = "status")]
    pub status: *string,
    #[serde(rename = "trainingType")]
    pub training_type: *string,
    #[serde(rename = "userId")]
    pub user_id: *string,
    #[serde(rename = "appId")]
    pub app_id: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimiter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*repository.MFARepository>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Factor {
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "verifiedAt")]
    pub verified_at: *time.Time,
    #[serde(rename = "-")]
    pub -: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "lastUsedAt")]
    pub last_used_at: *time.Time,
    #[serde(rename = "priority")]
    pub priority: FactorPriority,
    #[serde(rename = "status")]
    pub status: FactorStatus,
    #[serde(rename = "type")]
    pub type: FactorType,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupCodeFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*twofa.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterProvider_req {
    #[serde(rename = "type")]
    pub type: String,
    #[serde(rename = "OIDCClientSecret")]
    pub o_i_d_c_client_secret: String,
    #[serde(rename = "OIDCIssuer")]
    pub o_i_d_c_issuer: String,
    #[serde(rename = "OIDCRedirectURI")]
    pub o_i_d_c_redirect_u_r_i: String,
    #[serde(rename = "SAMLCert")]
    pub s_a_m_l_cert: String,
    #[serde(rename = "SAMLEntryPoint")]
    pub s_a_m_l_entry_point: String,
    #[serde(rename = "SAMLIssuer")]
    pub s_a_m_l_issuer: String,
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "OIDCClientID")]
    pub o_i_d_c_client_i_d: String,
    #[serde(rename = "domain")]
    pub domain: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenRequest {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "code_verifier")]
    pub code_verifier: String,
    #[serde(rename = "grant_type")]
    pub grant_type: String,
    #[serde(rename = "redirect_uri")]
    pub redirect_uri: String,
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "client_secret")]
    pub client_secret: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFABackupCodesResponse {
    #[serde(rename = "codes")]
    pub codes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceCheckResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceViolationsResponse {
    #[serde(rename = "violations")]
    pub violations: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TOTPConfig {
    #[serde(rename = "algorithm")]
    pub algorithm: String,
    #[serde(rename = "digits")]
    pub digits: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "issuer")]
    pub issuer: String,
    #[serde(rename = "period")]
    pub period: i32,
    #[serde(rename = "window_size")]
    pub window_size: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OAuthState {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoverySession {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Middleware {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*Config>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationVerifyResponse {
    #[serde(rename = "impersonator_id")]
    pub impersonator_id: String,
    #[serde(rename = "is_impersonating")]
    pub is_impersonating: bool,
    #[serde(rename = "target_user_id")]
    pub target_user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddTrustedContactResponse {
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "phone")]
    pub phone: String,
    #[serde(rename = "verified")]
    pub verified: bool,
    #[serde(rename = "addedAt")]
    pub added_at: time.Time,
    #[serde(rename = "contactId")]
    pub contact_id: xid.ID,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentExportFileResponse {
    #[serde(rename = "content_type")]
    pub content_type: String,
    #[serde(rename = "data")]
    pub data: []byte,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCClientResponse {
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "client_secret")]
    pub client_secret: String,
    #[serde(rename = "redirect_uris")]
    pub redirect_uris: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpNotificationProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasskeyLoginResponse {
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderConfigResponse {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "provider")]
    pub provider: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateProvider_req {
    #[serde(rename = "providerType")]
    pub provider_type: String,
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "isDefault")]
    pub is_default: bool,
    #[serde(rename = "organizationId", skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<*string>,
    #[serde(rename = "providerName")]
    pub provider_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditConfig {
    #[serde(rename = "signLogs")]
    pub sign_logs: bool,
    #[serde(rename = "detailedTrail")]
    pub detailed_trail: bool,
    #[serde(rename = "exportFormat")]
    pub export_format: String,
    #[serde(rename = "immutable")]
    pub immutable: bool,
    #[serde(rename = "maxRetentionDays")]
    pub max_retention_days: i32,
    #[serde(rename = "minRetentionDays")]
    pub min_retention_days: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListReportsFilter {
    #[serde(rename = "format")]
    pub format: *string,
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "reportType")]
    pub report_type: *string,
    #[serde(rename = "standard")]
    pub standard: *ComplianceStandard,
    #[serde(rename = "status")]
    pub status: *string,
    #[serde(rename = "appId")]
    pub app_id: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RotateAPIKeyResponse {
    #[serde(rename = "api_key")]
    pub api_key: *apikey.APIKey,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProvidersConfig {
    #[serde(rename = "email")]
    pub email: EmailProviderConfig,
    #[serde(rename = "sms")]
    pub sms: *SMSProviderConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetChallengeStatusResponse {
    #[serde(rename = "attempts")]
    pub attempts: i32,
    #[serde(rename = "availableFactors")]
    pub available_factors: []FactorInfo,
    #[serde(rename = "challengeId")]
    pub challenge_id: xid.ID,
    #[serde(rename = "factorsRequired")]
    pub factors_required: i32,
    #[serde(rename = "factorsVerified")]
    pub factors_verified: i32,
    #[serde(rename = "maxAttempts")]
    pub max_attempts: i32,
    #[serde(rename = "status")]
    pub status: ChallengeStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhoneSendCodeResponse {
    #[serde(rename = "dev_code")]
    pub dev_code: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentsResponse {
    #[serde(rename = "consents")]
    pub consents: ,
    #[serde(rename = "count")]
    pub count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateEngine {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<template.FuncMap>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReportsConfig {
    #[serde(rename = "includeEvidence")]
    pub include_evidence: bool,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "schedule")]
    pub schedule: String,
    #[serde(rename = "storagePath")]
    pub storage_path: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "formats")]
    pub formats: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateDefault {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendOTP_body {
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonateUserRequest {
    #[serde(rename = "-")]
    pub -: String,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "duration")]
    pub duration: time.Duration,
    #[serde(rename = "user_id")]
    pub user_id: xid.ID,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentRecordResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AmountRule {
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "max_amount")]
    pub max_amount: f64,
    #[serde(rename = "min_amount")]
    pub min_amount: f64,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "currency")]
    pub currency: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccessTokenClaims {
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "token_type")]
    pub token_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWKSService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*KeyStore>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DashboardConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "path")]
    pub path: String,
    #[serde(rename = "showRecentChecks")]
    pub show_recent_checks: bool,
    #[serde(rename = "showReports")]
    pub show_reports: bool,
    #[serde(rename = "showScore")]
    pub show_score: bool,
    #[serde(rename = "showViolations")]
    pub show_violations: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BanUser_reqBody {
    #[serde(rename = "expires_at", skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<*time.Time>,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InvitationResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "invitation")]
    pub invitation: *organization.Invitation,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminBlockUser_req {
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Handler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionResponse {
    #[serde(rename = "connection")]
    pub connection: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifySecurityAnswersRequest {
    #[serde(rename = "answers")]
    pub answers: ,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentSettingsResponse {
    #[serde(rename = "settings")]
    pub settings: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailOTPErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Service {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*auth.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockAuditService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]*AuditEvent>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorInfo {
    #[serde(rename = "factorId")]
    pub factor_id: xid.ID,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "type")]
    pub type: FactorType,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoveTrustedContactRequest {
    #[serde(rename = "contactId")]
    pub contact_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FinishLogin_body {
    #[serde(rename = "remember")]
    pub remember: bool,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceEvidence {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "title")]
    pub title: String,
    #[serde(rename = "collectedBy")]
    pub collected_by: String,
    #[serde(rename = "controlId")]
    pub control_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "fileHash")]
    pub file_hash: String,
    #[serde(rename = "fileUrl")]
    pub file_url: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "evidenceType")]
    pub evidence_type: String,
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimitingConfig {
    #[serde(rename = "maxAttemptsPerHour")]
    pub max_attempts_per_hour: i32,
    #[serde(rename = "maxAttemptsPerIp")]
    pub max_attempts_per_ip: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "exponentialBackoff")]
    pub exponential_backoff: bool,
    #[serde(rename = "ipCooldownPeriod")]
    pub ip_cooldown_period: time.Duration,
    #[serde(rename = "lockoutAfterAttempts")]
    pub lockout_after_attempts: i32,
    #[serde(rename = "lockoutDuration")]
    pub lockout_duration: time.Duration,
    #[serde(rename = "maxAttemptsPerDay")]
    pub max_attempts_per_day: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataExportRequest {
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "exportPath")]
    pub export_path: String,
    #[serde(rename = "exportSize")]
    pub export_size: i64,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "errorMessage")]
    pub error_message: String,
    #[serde(rename = "exportUrl")]
    pub export_url: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "includeSections")]
    pub include_sections: []string,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MagicLinkSendResponse {
    #[serde(rename = "dev_url")]
    pub dev_url: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*notification.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateReport_req {
    #[serde(rename = "period")]
    pub period: String,
    #[serde(rename = "reportType")]
    pub report_type: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "format")]
    pub format: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFAPolicy {
    #[serde(rename = "requiredFactorTypes")]
    pub required_factor_types: []FactorType,
    #[serde(rename = "trustedDeviceDays")]
    pub trusted_device_days: i32,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "gracePeriodDays")]
    pub grace_period_days: i32,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "lockoutDurationMinutes")]
    pub lockout_duration_minutes: i32,
    #[serde(rename = "organizationId")]
    pub organization_id: xid.ID,
    #[serde(rename = "stepUpRequired")]
    pub step_up_required: bool,
    #[serde(rename = "adaptiveMfaEnabled")]
    pub adaptive_mfa_enabled: bool,
    #[serde(rename = "allowedFactorTypes")]
    pub allowed_factor_types: []FactorType,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "maxFailedAttempts")]
    pub max_failed_attempts: i32,
    #[serde(rename = "requiredFactorCount")]
    pub required_factor_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VideoSessionInfo {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthSessionsResponse {
    #[serde(rename = "sessions")]
    pub sessions: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthorizeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateProfileFromTemplate_req {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SMSConfig {
    #[serde(rename = "code_expiry_minutes")]
    pub code_expiry_minutes: i32,
    #[serde(rename = "code_length")]
    pub code_length: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "rate_limit")]
    pub rate_limit: *RateLimitConfig,
    #[serde(rename = "template_id")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Enable_body {
    #[serde(rename = "method")]
    pub method: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasskeyRegistrationOptionsResponse {
    #[serde(rename = "options")]
    pub options: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SMSProviderConfig {
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "from")]
    pub from: String,
    #[serde(rename = "provider")]
    pub provider: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustDeviceRequest {
    #[serde(rename = "deviceId")]
    pub device_id: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SSOSAMLMetadataResponse {
    #[serde(rename = "metadata")]
    pub metadata: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceRule {
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "resource_type")]
    pub resource_type: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "sensitivity")]
    pub sensitivity: String,
    #[serde(rename = "action")]
    pub action: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockForgeContext {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*http.Request>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditEvent {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderSession {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplatesResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "templates")]
    pub templates: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<user.ServiceInterface>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateRecoveryCodesResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "generatedAt")]
    pub generated_at: time.Time,
    #[serde(rename = "warning")]
    pub warning: String,
    #[serde(rename = "codes")]
    pub codes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartImpersonation_reqBody {
    #[serde(rename = "duration_minutes", skip_serializing_if = "Option::is_none")]
    pub duration_minutes: Option<i32>,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "target_user_id")]
    pub target_user_id: String,
    #[serde(rename = "ticket_number", skip_serializing_if = "Option::is_none")]
    pub ticket_number: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAPIKey_reqBody {
    #[serde(rename = "allowed_ips", skip_serializing_if = "Option::is_none")]
    pub allowed_ips: Option<[]string>,
    #[serde(rename = "description", skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(rename = "metadata", skip_serializing_if = "Option::is_none")]
    pub metadata: Option<>,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "permissions", skip_serializing_if = "Option::is_none")]
    pub permissions: Option<>,
    #[serde(rename = "rate_limit", skip_serializing_if = "Option::is_none")]
    pub rate_limit: Option<i32>,
    #[serde(rename = "scopes")]
    pub scopes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckSubResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MagicLinkErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TOTPSecret {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentVerification {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFARequiredResponse {
    #[serde(rename = "require_twofa")]
    pub require_twofa: bool,
    #[serde(rename = "user")]
    pub user: ,
    #[serde(rename = "device_id")]
    pub device_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockRepository {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationSession {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFAConfigResponse {
    #[serde(rename = "allowed_factor_types")]
    pub allowed_factor_types: []string,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "required_factor_count")]
    pub required_factor_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CallbackDataResponse {
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "isNewUser")]
    pub is_new_user: bool,
    #[serde(rename = "user")]
    pub user: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RejectRecoveryRequest {
    #[serde(rename = "notes")]
    pub notes: String,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceProfile {
    #[serde(rename = "detailedAuditTrail")]
    pub detailed_audit_trail: bool,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "passwordMinLength")]
    pub password_min_length: i32,
    #[serde(rename = "rbacRequired")]
    pub rbac_required: bool,
    #[serde(rename = "standards")]
    pub standards: []ComplianceStandard,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "passwordRequireSymbol")]
    pub password_require_symbol: bool,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "passwordExpiryDays")]
    pub password_expiry_days: i32,
    #[serde(rename = "regularAccessReview")]
    pub regular_access_review: bool,
    #[serde(rename = "sessionIpBinding")]
    pub session_ip_binding: bool,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "encryptionInTransit")]
    pub encryption_in_transit: bool,
    #[serde(rename = "passwordRequireLower")]
    pub password_require_lower: bool,
    #[serde(rename = "passwordRequireNumber")]
    pub password_require_number: bool,
    #[serde(rename = "sessionIdleTimeout")]
    pub session_idle_timeout: i32,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "complianceContact")]
    pub compliance_contact: String,
    #[serde(rename = "encryptionAtRest")]
    pub encryption_at_rest: bool,
    #[serde(rename = "sessionMaxAge")]
    pub session_max_age: i32,
    #[serde(rename = "auditLogExport")]
    pub audit_log_export: bool,
    #[serde(rename = "dpoContact")]
    pub dpo_contact: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "passwordRequireUpper")]
    pub password_require_upper: bool,
    #[serde(rename = "leastPrivilege")]
    pub least_privilege: bool,
    #[serde(rename = "mfaRequired")]
    pub mfa_required: bool,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "dataResidency")]
    pub data_residency: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceReportFileResponse {
    #[serde(rename = "data")]
    pub data: []byte,
    #[serde(rename = "content_type")]
    pub content_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetChallengeStatusRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RejectRecoveryResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "rejected")]
    pub rejected: bool,
    #[serde(rename = "rejectedAt")]
    pub rejected_at: time.Time,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataExportRequestInput {
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "includeSections")]
    pub include_sections: []string,
}

/// User device
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Device {
    #[serde(rename = "lastUsedAt")]
    pub last_used_at: String,
    #[serde(rename = "ipAddress", skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(rename = "userAgent", skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "name", skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(rename = "type", skip_serializing_if = "Option::is_none")]
    pub type: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceReportsResponse {
    #[serde(rename = "reports")]
    pub reports: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialProvidersResponse {
    #[serde(rename = "providers")]
    pub providers: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoverySessionInfo {
    #[serde(rename = "userEmail")]
    pub user_email: String,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "currentStep")]
    pub current_step: i32,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "status")]
    pub status: RecoveryStatus,
    #[serde(rename = "totalSteps")]
    pub total_steps: i32,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "method")]
    pub method: RecoveryMethod,
    #[serde(rename = "requiresReview")]
    pub requires_review: bool,
    #[serde(rename = "riskScore")]
    pub risk_score: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListPoliciesFilter {
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "policyType")]
    pub policy_type: *string,
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "standard")]
    pub standard: *ComplianceStandard,
    #[serde(rename = "status")]
    pub status: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRecoveryStatsResponse {
    #[serde(rename = "failedRecoveries")]
    pub failed_recoveries: i32,
    #[serde(rename = "successRate")]
    pub success_rate: f64,
    #[serde(rename = "totalAttempts")]
    pub total_attempts: i32,
    #[serde(rename = "averageRiskScore")]
    pub average_risk_score: f64,
    #[serde(rename = "highRiskAttempts")]
    pub high_risk_attempts: i32,
    #[serde(rename = "methodStats")]
    pub method_stats: ,
    #[serde(rename = "pendingRecoveries")]
    pub pending_recoveries: i32,
    #[serde(rename = "successfulRecoveries")]
    pub successful_recoveries: i32,
    #[serde(rename = "adminReviewsRequired")]
    pub admin_reviews_required: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemberHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*coreapp.ServiceImpl>,
}

/// User session
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Session {
    #[serde(rename = "userAgent", skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: String,
    #[serde(rename = "ipAddress", skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DevicesResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "devices")]
    pub devices: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProvidersAppResponse {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "providers")]
    pub providers: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentVerificationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]byte>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SSOSAMLCallbackResponse {
    #[serde(rename = "attributes")]
    pub attributes: ,
    #[serde(rename = "issuer")]
    pub issuer: String,
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "subject")]
    pub subject: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpPolicy {
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "priority")]
    pub priority: i32,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "rules")]
    pub rules: ,
    #[serde(rename = "updated_at")]
    pub updated_at: time.Time,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationPreviewResponse {
    #[serde(rename = "body")]
    pub body: String,
    #[serde(rename = "subject")]
    pub subject: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateABTestVariant_req {
    #[serde(rename = "weight")]
    pub weight: i32,
    #[serde(rename = "body")]
    pub body: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "subject")]
    pub subject: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompliancePolicyResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebAuthnFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*passkey.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignInResponse {
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
    #[serde(rename = "session")]
    pub session: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAPIKeyResponse {
    #[serde(rename = "api_key")]
    pub api_key: *apikey.APIKey,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationListResponse {
    #[serde(rename = "total")]
    pub total: i32,
    #[serde(rename = "notifications")]
    pub notifications: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListProfilesFilter {
    #[serde(rename = "standard")]
    pub standard: *ComplianceStandard,
    #[serde(rename = "status")]
    pub status: *string,
    #[serde(rename = "appId")]
    pub app_id: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StatsResponse {
    #[serde(rename = "active_sessions")]
    pub active_sessions: i32,
    #[serde(rename = "active_users")]
    pub active_users: i32,
    #[serde(rename = "banned_users")]
    pub banned_users: i32,
    #[serde(rename = "timestamp")]
    pub timestamp: String,
    #[serde(rename = "total_sessions")]
    pub total_sessions: i32,
    #[serde(rename = "total_users")]
    pub total_users: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecurityQuestionsRequest {
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateRecoveryCodesRequest {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "format")]
    pub format: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorEnrollmentResponse {
    #[serde(rename = "provisioningData")]
    pub provisioning_data: ,
    #[serde(rename = "status")]
    pub status: FactorStatus,
    #[serde(rename = "type")]
    pub type: FactorType,
    #[serde(rename = "factorId")]
    pub factor_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OTPSentResponse {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentReportResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Username2FARequiredResponse {
    #[serde(rename = "user")]
    pub user: ,
    #[serde(rename = "device_id")]
    pub device_id: String,
    #[serde(rename = "require_twofa")]
    pub require_twofa: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnfidoConfig {
    #[serde(rename = "includeFacialReport")]
    pub include_facial_report: bool,
    #[serde(rename = "facialCheck")]
    pub facial_check: FacialCheckConfig,
    #[serde(rename = "includeWatchlistReport")]
    pub include_watchlist_report: bool,
    #[serde(rename = "region")]
    pub region: String,
    #[serde(rename = "webhookToken")]
    pub webhook_token: String,
    #[serde(rename = "workflowId")]
    pub workflow_id: String,
    #[serde(rename = "apiToken")]
    pub api_token: String,
    #[serde(rename = "documentCheck")]
    pub document_check: DocumentCheckConfig,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "includeDocumentReport")]
    pub include_document_report: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimit {
    #[serde(rename = "max_requests")]
    pub max_requests: i32,
    #[serde(rename = "window")]
    pub window: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceInfo {
    #[serde(rename = "deviceId")]
    pub device_id: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataProcessingAgreement {
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "metadata")]
    pub metadata: JSONBMap,
    #[serde(rename = "signedByEmail")]
    pub signed_by_email: String,
    #[serde(rename = "signedByTitle")]
    pub signed_by_title: String,
    #[serde(rename = "expiryDate")]
    pub expiry_date: *time.Time,
    #[serde(rename = "signedByName")]
    pub signed_by_name: String,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "digitalSignature")]
    pub digital_signature: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "agreementType")]
    pub agreement_type: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "effectiveDate")]
    pub effective_date: time.Time,
    #[serde(rename = "signedBy")]
    pub signed_by: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockRepository {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResetUserMFARequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateRecoveryConfigRequest {
    #[serde(rename = "minimumStepsRequired")]
    pub minimum_steps_required: i32,
    #[serde(rename = "requireAdminReview")]
    pub require_admin_review: bool,
    #[serde(rename = "requireMultipleSteps")]
    pub require_multiple_steps: bool,
    #[serde(rename = "riskScoreThreshold")]
    pub risk_score_threshold: f64,
    #[serde(rename = "enabledMethods")]
    pub enabled_methods: []RecoveryMethod,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpDevicesResponse {
    #[serde(rename = "devices")]
    pub devices: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CancelRecoveryRequest {
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreatePolicyRequest {
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "required")]
    pub required: bool,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "consentType")]
    pub consent_type: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "renewable")]
    pub renewable: bool,
    #[serde(rename = "validityPeriod")]
    pub validity_period: *int,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendWithTemplateRequest {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "recipient")]
    pub recipient: String,
    #[serde(rename = "templateKey")]
    pub template_key: String,
    #[serde(rename = "type")]
    pub type: notification.NotificationType,
    #[serde(rename = "variables")]
    pub variables: ,
    #[serde(rename = "appId")]
    pub app_id: xid.ID,
    #[serde(rename = "language")]
    pub language: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageResponse {
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UploadDocumentResponse {
    #[serde(rename = "uploadedAt")]
    pub uploaded_at: time.Time,
    #[serde(rename = "documentId")]
    pub document_id: xid.ID,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "processingTime")]
    pub processing_time: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrivacySettingsRequest {
    #[serde(rename = "allowDataPortability")]
    pub allow_data_portability: *bool,
    #[serde(rename = "autoDeleteAfterDays")]
    pub auto_delete_after_days: *int,
    #[serde(rename = "consentRequired")]
    pub consent_required: *bool,
    #[serde(rename = "contactEmail")]
    pub contact_email: String,
    #[serde(rename = "cookieConsentEnabled")]
    pub cookie_consent_enabled: *bool,
    #[serde(rename = "cookieConsentStyle")]
    pub cookie_consent_style: String,
    #[serde(rename = "anonymousConsentEnabled")]
    pub anonymous_consent_enabled: *bool,
    #[serde(rename = "ccpaMode")]
    pub ccpa_mode: *bool,
    #[serde(rename = "contactPhone")]
    pub contact_phone: String,
    #[serde(rename = "dataExportExpiryHours")]
    pub data_export_expiry_hours: *int,
    #[serde(rename = "dataRetentionDays")]
    pub data_retention_days: *int,
    #[serde(rename = "exportFormat")]
    pub export_format: []string,
    #[serde(rename = "deletionGracePeriodDays")]
    pub deletion_grace_period_days: *int,
    #[serde(rename = "dpoEmail")]
    pub dpo_email: String,
    #[serde(rename = "gdprMode")]
    pub gdpr_mode: *bool,
    #[serde(rename = "requireAdminApprovalForDeletion")]
    pub require_admin_approval_for_deletion: *bool,
    #[serde(rename = "requireExplicitConsent")]
    pub require_explicit_consent: *bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasskeyListResponse {
    #[serde(rename = "passkeys")]
    pub passkeys: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<Config>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTraining {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "score")]
    pub score: i32,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "trainingType")]
    pub training_type: String,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "userId")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyRecoveryCodeResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "remainingCodes")]
    pub remaining_codes: i32,
    #[serde(rename = "valid")]
    pub valid: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpVerification {
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "method")]
    pub method: VerificationMethod,
    #[serde(rename = "rule_name")]
    pub rule_name: String,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "expires_at")]
    pub expires_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "device_id")]
    pub device_id: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "session_id")]
    pub session_id: String,
    #[serde(rename = "verified_at")]
    pub verified_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddMember_req {
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetStatusRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Verify_body {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "phone")]
    pub phone: String,
    #[serde(rename = "remember")]
    pub remember: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IPWhitelistConfig {
    #[serde(rename = "strict_mode")]
    pub strict_mode: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionsResponse {
    #[serde(rename = "connections")]
    pub connections: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestSendTemplate_req {
    #[serde(rename = "recipient")]
    pub recipient: String,
    #[serde(rename = "variables")]
    pub variables: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentDashboardConfig {
    #[serde(rename = "showConsentHistory")]
    pub show_consent_history: bool,
    #[serde(rename = "showCookiePreferences")]
    pub show_cookie_preferences: bool,
    #[serde(rename = "showDataDeletion")]
    pub show_data_deletion: bool,
    #[serde(rename = "showDataExport")]
    pub show_data_export: bool,
    #[serde(rename = "showPolicies")]
    pub show_policies: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "path")]
    pub path: String,
    #[serde(rename = "showAuditLog")]
    pub show_audit_log: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AssignRole_reqBody {
    #[serde(rename = "roleID")]
    pub role_i_d: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryCodeUsage {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrivacySettings {
    #[serde(rename = "consentRequired")]
    pub consent_required: bool,
    #[serde(rename = "contactPhone")]
    pub contact_phone: String,
    #[serde(rename = "cookieConsentStyle")]
    pub cookie_consent_style: String,
    #[serde(rename = "deletionGracePeriodDays")]
    pub deletion_grace_period_days: i32,
    #[serde(rename = "exportFormat")]
    pub export_format: []string,
    #[serde(rename = "gdprMode")]
    pub gdpr_mode: bool,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "metadata")]
    pub metadata: JSONBMap,
    #[serde(rename = "allowDataPortability")]
    pub allow_data_portability: bool,
    #[serde(rename = "autoDeleteAfterDays")]
    pub auto_delete_after_days: i32,
    #[serde(rename = "ccpaMode")]
    pub ccpa_mode: bool,
    #[serde(rename = "dpoEmail")]
    pub dpo_email: String,
    #[serde(rename = "requireExplicitConsent")]
    pub require_explicit_consent: bool,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "anonymousConsentEnabled")]
    pub anonymous_consent_enabled: bool,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "dataExportExpiryHours")]
    pub data_export_expiry_hours: i32,
    #[serde(rename = "dataRetentionDays")]
    pub data_retention_days: i32,
    #[serde(rename = "contactEmail")]
    pub contact_email: String,
    #[serde(rename = "cookieConsentEnabled")]
    pub cookie_consent_enabled: bool,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "requireAdminApprovalForDeletion")]
    pub require_admin_approval_for_deletion: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpAttempt {
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "requirement_id")]
    pub requirement_id: String,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "failure_reason")]
    pub failure_reason: String,
    #[serde(rename = "method")]
    pub method: VerificationMethod,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MembersResponse {
    #[serde(rename = "members")]
    pub members: []*organization.Member,
    #[serde(rename = "total")]
    pub total: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocialSignInResponse {
    #[serde(rename = "redirect_url")]
    pub redirect_url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthContactsResponse {
    #[serde(rename = "contacts")]
    pub contacts: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthStatsResponse {
    #[serde(rename = "stats")]
    pub stats: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSessionDeleteResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenderTemplate_req {
    #[serde(rename = "template")]
    pub template: String,
    #[serde(rename = "variables")]
    pub variables: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationResponse {
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "factorsRemaining")]
    pub factors_remaining: i32,
    #[serde(rename = "sessionComplete")]
    pub session_complete: bool,
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthRecoveryResponse {
    #[serde(rename = "session_id")]
    pub session_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CookieConsentRequest {
    #[serde(rename = "marketing")]
    pub marketing: bool,
    #[serde(rename = "personalization")]
    pub personalization: bool,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationSessionResponse {
    #[serde(rename = "session")]
    pub session: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Challenge {
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
    #[serde(rename = "factorId")]
    pub factor_id: xid.ID,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "maxAttempts")]
    pub max_attempts: i32,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "type")]
    pub type: FactorType,
    #[serde(rename = "verifiedAt")]
    pub verified_at: *time.Time,
    #[serde(rename = "-")]
    pub -: String,
    #[serde(rename = "attempts")]
    pub attempts: i32,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "status")]
    pub status: ChallengeStatus,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFAStatusDetailResponse {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "method")]
    pub method: String,
    #[serde(rename = "trusted")]
    pub trusted: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnbanUserRequest {
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "user_id")]
    pub user_id: xid.ID,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "-")]
    pub -: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateUser_reqBody {
    #[serde(rename = "password", skip_serializing_if = "Option::is_none")]
    pub password: Option<String>,
    #[serde(rename = "role", skip_serializing_if = "Option::is_none")]
    pub role: Option<String>,
    #[serde(rename = "username", skip_serializing_if = "Option::is_none")]
    pub username: Option<String>,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "email_verified")]
    pub email_verified: bool,
    #[serde(rename = "metadata", skip_serializing_if = "Option::is_none")]
    pub metadata: Option<>,
    #[serde(rename = "name", skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VideoVerificationSession {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasskeyStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestProvider_req {
    #[serde(rename = "providerName")]
    pub provider_name: String,
    #[serde(rename = "providerType")]
    pub provider_type: String,
    #[serde(rename = "testRecipient")]
    pub test_recipient: String,
    #[serde(rename = "config")]
    pub config: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnonymousErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EndImpersonation_reqBody {
    #[serde(rename = "impersonation_id")]
    pub impersonation_id: String,
    #[serde(rename = "reason", skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JumioProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<JumioConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "verifications")]
    pub verifications: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct testRouter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListEvidenceFilter {
    #[serde(rename = "standard")]
    pub standard: *ComplianceStandard,
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "controlId")]
    pub control_id: *string,
    #[serde(rename = "evidenceType")]
    pub evidence_type: *string,
    #[serde(rename = "profileId")]
    pub profile_id: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTrainingsResponse {
    #[serde(rename = "training")]
    pub training: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceStatusDetailsResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupCodesConfig {
    #[serde(rename = "allow_reuse")]
    pub allow_reuse: bool,
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "length")]
    pub length: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApproveRecoveryRequest {
    #[serde(rename = "notes")]
    pub notes: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataDeletionConfig {
    #[serde(rename = "archivePath")]
    pub archive_path: String,
    #[serde(rename = "autoProcessAfterGrace")]
    pub auto_process_after_grace: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "notifyBeforeDeletion")]
    pub notify_before_deletion: bool,
    #[serde(rename = "requireAdminApproval")]
    pub require_admin_approval: bool,
    #[serde(rename = "archiveBeforeDeletion")]
    pub archive_before_deletion: bool,
    #[serde(rename = "gracePeriodDays")]
    pub grace_period_days: i32,
    #[serde(rename = "preserveLegalData")]
    pub preserve_legal_data: bool,
    #[serde(rename = "retentionExemptions")]
    pub retention_exemptions: []string,
    #[serde(rename = "allowPartialDeletion")]
    pub allow_partial_deletion: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCJWKSResponse {
    #[serde(rename = "keys")]
    pub keys: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompliancePoliciesResponse {
    #[serde(rename = "policies")]
    pub policies: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentDeletionResponse {
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpPoliciesResponse {
    #[serde(rename = "policies")]
    pub policies: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddCustomPermission_req {
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "category")]
    pub category: String,
    #[serde(rename = "description")]
    pub description: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderSessionRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListSessionsResponse {
    #[serde(rename = "total")]
    pub total: i32,
    #[serde(rename = "total_pages")]
    pub total_pages: i32,
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "sessions")]
    pub sessions: []*session.Session,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetUserRole_reqBody {
    #[serde(rename = "role")]
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWKS {
    #[serde(rename = "keys")]
    pub keys: []JWK,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFASendOTPResponse {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSessionRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeclareABTestWinner_req {
    #[serde(rename = "abTestGroup")]
    pub ab_test_group: String,
    #[serde(rename = "winnerId")]
    pub winner_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTemplateResponse {
    #[serde(rename = "standard")]
    pub standard: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthURLResponse {
    #[serde(rename = "url")]
    pub url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFAErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailVerificationConfig {
    #[serde(rename = "fromName")]
    pub from_name: String,
    #[serde(rename = "maxAttempts")]
    pub max_attempts: i32,
    #[serde(rename = "requireEmailProof")]
    pub require_email_proof: bool,
    #[serde(rename = "codeExpiry")]
    pub code_expiry: time.Duration,
    #[serde(rename = "codeLength")]
    pub code_length: i32,
    #[serde(rename = "emailTemplate")]
    pub email_template: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "fromAddress")]
    pub from_address: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentCheckConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "extractData")]
    pub extract_data: bool,
    #[serde(rename = "validateDataConsistency")]
    pub validate_data_consistency: bool,
    #[serde(rename = "validateExpiry")]
    pub validate_expiry: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceCheck {
    #[serde(rename = "checkType")]
    pub check_type: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "evidence")]
    pub evidence: []string,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "nextCheckAt")]
    pub next_check_at: time.Time,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "result")]
    pub result: ,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "lastCheckedAt")]
    pub last_checked_at: time.Time,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdatePolicyRequest {
    #[serde(rename = "adaptiveMfaEnabled")]
    pub adaptive_mfa_enabled: *bool,
    #[serde(rename = "gracePeriodDays")]
    pub grace_period_days: *int,
    #[serde(rename = "lockoutDurationMinutes")]
    pub lockout_duration_minutes: *int,
    #[serde(rename = "requiredFactorCount")]
    pub required_factor_count: *int,
    #[serde(rename = "requiredFactorTypes")]
    pub required_factor_types: []FactorType,
    #[serde(rename = "trustedDeviceDays")]
    pub trusted_device_days: *int,
    #[serde(rename = "allowedFactorTypes")]
    pub allowed_factor_types: []FactorType,
    #[serde(rename = "maxFailedAttempts")]
    pub max_failed_attempts: *int,
    #[serde(rename = "stepUpRequired")]
    pub step_up_required: *bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListFactorsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContinueRecoveryResponse {
    #[serde(rename = "data")]
    pub data: ,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "instructions")]
    pub instructions: String,
    #[serde(rename = "method")]
    pub method: RecoveryMethod,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
    #[serde(rename = "totalSteps")]
    pub total_steps: i32,
    #[serde(rename = "currentStep")]
    pub current_step: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListRecoverySessionsResponse {
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "sessions")]
    pub sessions: []RecoverySessionInfo,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
    #[serde(rename = "page")]
    pub page: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSessionListResponse {
    #[serde(rename = "sessions")]
    pub sessions: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateUserRequest {
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "email_verified")]
    pub email_verified: bool,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "password")]
    pub password: String,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "username")]
    pub username: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "-")]
    pub -: xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReviewDocumentRequest {
    #[serde(rename = "documentId")]
    pub document_id: xid.ID,
    #[serde(rename = "notes")]
    pub notes: String,
    #[serde(rename = "rejectionReason")]
    pub rejection_reason: String,
    #[serde(rename = "approved")]
    pub approved: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Send_body {
    #[serde(rename = "email")]
    pub email: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpRequirementResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationResponse {
    #[serde(rename = "notification")]
    pub notification: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListViolationsFilter {
    #[serde(rename = "status")]
    pub status: *string,
    #[serde(rename = "userId")]
    pub user_id: *string,
    #[serde(rename = "violationType")]
    pub violation_type: *string,
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "severity")]
    pub severity: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompliancePolicy {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "approvedAt")]
    pub approved_at: *time.Time,
    #[serde(rename = "approvedBy")]
    pub approved_by: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "effectiveDate")]
    pub effective_date: time.Time,
    #[serde(rename = "reviewDate")]
    pub review_date: time.Time,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "title")]
    pub title: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "policyType")]
    pub policy_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityQuestionsConfig {
    #[serde(rename = "caseSensitive")]
    pub case_sensitive: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "maxAnswerLength")]
    pub max_answer_length: i32,
    #[serde(rename = "minimumQuestions")]
    pub minimum_questions: i32,
    #[serde(rename = "allowCustomQuestions")]
    pub allow_custom_questions: bool,
    #[serde(rename = "forbidCommonAnswers")]
    pub forbid_common_answers: bool,
    #[serde(rename = "lockoutDuration")]
    pub lockout_duration: time.Duration,
    #[serde(rename = "maxAttempts")]
    pub max_attempts: i32,
    #[serde(rename = "predefinedQuestions")]
    pub predefined_questions: []string,
    #[serde(rename = "requireMinLength")]
    pub require_min_length: i32,
    #[serde(rename = "requiredToRecover")]
    pub required_to_recover: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartRecoveryResponse {
    #[serde(rename = "status")]
    pub status: RecoveryStatus,
    #[serde(rename = "availableMethods")]
    pub available_methods: []RecoveryMethod,
    #[serde(rename = "completedSteps")]
    pub completed_steps: i32,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "requiredSteps")]
    pub required_steps: i32,
    #[serde(rename = "requiresReview")]
    pub requires_review: bool,
    #[serde(rename = "riskScore")]
    pub risk_score: f64,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

