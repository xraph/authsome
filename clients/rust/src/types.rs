// Auto-generated Rust types

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Plugin {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*DashboardExtension>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateUserRequestDTO {
    #[serde(rename = "username")]
    pub username: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "email_verified")]
    pub email_verified: bool,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "password")]
    pub password: String,
    #[serde(rename = "role")]
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecretsInput {
    #[serde(rename = "search")]
    pub search: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetInvitationsResult {
    #[serde(rename = "data")]
    pub data: []InvitationDTO,
    #[serde(rename = "pagination")]
    pub pagination: PaginationInfo,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetNotificationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTemplateInput {
    #[serde(rename = "subject")]
    pub subject: String,
    #[serde(rename = "templateKey")]
    pub template_key: String,
    #[serde(rename = "type")]
    pub type: String,
    #[serde(rename = "variables")]
    pub variables: []string,
    #[serde(rename = "body")]
    pub body: String,
    #[serde(rename = "language")]
    pub language: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BatchEvaluateResponse {
    #[serde(rename = "successCount")]
    pub success_count: i32,
    #[serde(rename = "totalEvaluations")]
    pub total_evaluations: i32,
    #[serde(rename = "totalTimeMs")]
    pub total_time_ms: f64,
    #[serde(rename = "failureCount")]
    pub failure_count: i32,
    #[serde(rename = "results")]
    pub results: []*BatchEvaluationResult,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTemplate {
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "auditFrequencyDays")]
    pub audit_frequency_days: i32,
    #[serde(rename = "dataResidency")]
    pub data_residency: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "requiredPolicies")]
    pub required_policies: []string,
    #[serde(rename = "requiredTraining")]
    pub required_training: []string,
    #[serde(rename = "sessionMaxAge")]
    pub session_max_age: i32,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "mfaRequired")]
    pub mfa_required: bool,
    #[serde(rename = "passwordMinLength")]
    pub password_min_length: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyRequest {
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "otp")]
    pub otp: String,
    #[serde(rename = "remember")]
    pub remember: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminAddProviderRequest {
    #[serde(rename = "clientId")]
    pub client_id: String,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResendResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataExportConfig {
    #[serde(rename = "expiryHours")]
    pub expiry_hours: i32,
    #[serde(rename = "includeSections")]
    pub include_sections: []string,
    #[serde(rename = "maxExportSize")]
    pub max_export_size: i64,
    #[serde(rename = "maxRequests")]
    pub max_requests: i32,
    #[serde(rename = "requestPeriod")]
    pub request_period: time.Duration,
    #[serde(rename = "autoCleanup")]
    pub auto_cleanup: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "storagePath")]
    pub storage_path: String,
    #[serde(rename = "allowedFormats")]
    pub allowed_formats: []string,
    #[serde(rename = "cleanupInterval")]
    pub cleanup_interval: time.Duration,
    #[serde(rename = "defaultFormat")]
    pub default_format: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorEnrollmentResponse {
    #[serde(rename = "factorId")]
    pub factor_id: xid.ID,
    #[serde(rename = "provisioningData")]
    pub provisioning_data: ,
    #[serde(rename = "status")]
    pub status: FactorStatus,
    #[serde(rename = "type")]
    pub type: FactorType,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderDetailResponse {
    #[serde(rename = "updatedAt")]
    pub updated_at: String,
    #[serde(rename = "hasSamlCert")]
    pub has_saml_cert: bool,
    #[serde(rename = "oidcClientID")]
    pub oidc_client_i_d: String,
    #[serde(rename = "samlEntryPoint")]
    pub saml_entry_point: String,
    #[serde(rename = "type")]
    pub type: String,
    #[serde(rename = "attributeMapping")]
    pub attribute_mapping: ,
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "domain")]
    pub domain: String,
    #[serde(rename = "oidcIssuer")]
    pub oidc_issuer: String,
    #[serde(rename = "oidcRedirectURI")]
    pub oidc_redirect_u_r_i: String,
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "samlIssuer")]
    pub saml_issuer: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnpublishContentTypeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartRecoveryResponse {
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
    #[serde(rename = "status")]
    pub status: RecoveryStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentRecordResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataDeletionConfig {
    #[serde(rename = "archivePath")]
    pub archive_path: String,
    #[serde(rename = "autoProcessAfterGrace")]
    pub auto_process_after_grace: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "preserveLegalData")]
    pub preserve_legal_data: bool,
    #[serde(rename = "requireAdminApproval")]
    pub require_admin_approval: bool,
    #[serde(rename = "archiveBeforeDeletion")]
    pub archive_before_deletion: bool,
    #[serde(rename = "gracePeriodDays")]
    pub grace_period_days: i32,
    #[serde(rename = "notifyBeforeDeletion")]
    pub notify_before_deletion: bool,
    #[serde(rename = "retentionExemptions")]
    pub retention_exemptions: []string,
    #[serde(rename = "allowPartialDeletion")]
    pub allow_partial_deletion: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateReport_req {
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "period")]
    pub period: String,
    #[serde(rename = "reportType")]
    pub report_type: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListProfilesFilter {
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "standard")]
    pub standard: *ComplianceStandard,
    #[serde(rename = "status")]
    pub status: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebAuthnWrapper {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<Config>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiStepRecoveryConfig {
    #[serde(rename = "allowUserChoice")]
    pub allow_user_choice: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "requireAdminApproval")]
    pub require_admin_approval: bool,
    #[serde(rename = "sessionExpiry")]
    pub session_expiry: time.Duration,
    #[serde(rename = "allowStepSkip")]
    pub allow_step_skip: bool,
    #[serde(rename = "highRiskSteps")]
    pub high_risk_steps: []RecoveryMethod,
    #[serde(rename = "lowRiskSteps")]
    pub low_risk_steps: []RecoveryMethod,
    #[serde(rename = "mediumRiskSteps")]
    pub medium_risk_steps: []RecoveryMethod,
    #[serde(rename = "minimumSteps")]
    pub minimum_steps: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListSessionsResponse {
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "sessions")]
    pub sessions: []*session.Session,
    #[serde(rename = "total")]
    pub total: i32,
    #[serde(rename = "total_pages")]
    pub total_pages: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteRule {
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "method")]
    pub method: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "pattern")]
    pub pattern: String,
}

/// Simple message response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageResponse {
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResetUserMFAResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "devicesRevoked")]
    pub devices_revoked: i32,
    #[serde(rename = "factorsReset")]
    pub factors_reset: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InviteMemberHandlerRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionTokenResponse {
    #[serde(rename = "session")]
    pub session: *session.Session,
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentManager {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*ConsentService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenIntrospectionRequest {
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "client_secret")]
    pub client_secret: String,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "token_type_hint")]
    pub token_type_hint: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRecoveryStatsRequest {
    #[serde(rename = "endDate")]
    pub end_date: time.Time,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "startDate")]
    pub start_date: time.Time,
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
pub struct VerifyRequest2FA {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "device_id")]
    pub device_id: String,
    #[serde(rename = "remember_device")]
    pub remember_device: bool,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetStatusRequest {
    #[serde(rename = "device_id")]
    pub device_id: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OTPSentResponse {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "status")]
    pub status: String,
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
pub struct SendVerificationCodeRequest {
    #[serde(rename = "method")]
    pub method: RecoveryMethod,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
    #[serde(rename = "target")]
    pub target: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimiter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
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
pub struct Factor {
    #[serde(rename = "metadata")]
    pub metadata: ,
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
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "priority")]
    pub priority: FactorPriority,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "lastUsedAt")]
    pub last_used_at: *time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTeamHandlerRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCLoginRequest {
    #[serde(rename = "nonce")]
    pub nonce: String,
    #[serde(rename = "redirectUri")]
    pub redirect_uri: String,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "state")]
    pub state: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrateRBACRequest {
    #[serde(rename = "dryRun")]
    pub dry_run: bool,
    #[serde(rename = "keepRbacPolicies")]
    pub keep_rbac_policies: bool,
    #[serde(rename = "namespaceId")]
    pub namespace_id: String,
    #[serde(rename = "validateEquivalence")]
    pub validate_equivalence: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*migration.RBACMigrationService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthConfigResponse {
    #[serde(rename = "config")]
    pub config: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpVerification {
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "verified_at")]
    pub verified_at: time.Time,
    #[serde(rename = "device_id")]
    pub device_id: String,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "session_id")]
    pub session_id: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "expires_at")]
    pub expires_at: time.Time,
    #[serde(rename = "method")]
    pub method: VerificationMethod,
    #[serde(rename = "rule_name")]
    pub rule_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpPoliciesResponse {
    #[serde(rename = "policies")]
    pub policies: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateRoleTemplateInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "permissions")]
    pub permissions: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrganizationSummaryDTO {
    #[serde(rename = "teamCount")]
    pub team_count: i64,
    #[serde(rename = "userRole")]
    pub user_role: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "logo")]
    pub logo: String,
    #[serde(rename = "memberCount")]
    pub member_count: i64,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "slug")]
    pub slug: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailProviderDTO {
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "fromEmail")]
    pub from_email: String,
    #[serde(rename = "fromName")]
    pub from_name: String,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Adapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrateAllRequest {
    #[serde(rename = "dryRun")]
    pub dry_run: bool,
    #[serde(rename = "preserveOriginal")]
    pub preserve_original: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentPolicyResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyChallengeRequest {
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
pub struct AsyncConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "max_retries")]
    pub max_retries: i32,
    #[serde(rename = "persist_failures")]
    pub persist_failures: bool,
    #[serde(rename = "queue_size")]
    pub queue_size: i32,
    #[serde(rename = "retry_backoff")]
    pub retry_backoff: []string,
    #[serde(rename = "retry_enabled")]
    pub retry_enabled: bool,
    #[serde(rename = "worker_pool_size")]
    pub worker_pool_size: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateProvider_req {
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "isDefault")]
    pub is_default: bool,
    #[serde(rename = "organizationId", skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<*string>,
    #[serde(rename = "providerName")]
    pub provider_name: String,
    #[serde(rename = "providerType")]
    pub provider_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockNotificationProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]mockSentNotification>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateProfileRequest {
    #[serde(rename = "mfaRequired")]
    pub mfa_required: *bool,
    #[serde(rename = "name")]
    pub name: *string,
    #[serde(rename = "retentionDays")]
    pub retention_days: *int,
    #[serde(rename = "status")]
    pub status: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunCheckRequest {
    #[serde(rename = "checkType")]
    pub check_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScopeResolver {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<Repository>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientsListResponse {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "total")]
    pub total: i32,
    #[serde(rename = "totalPages")]
    pub total_pages: i32,
    #[serde(rename = "clients")]
    pub clients: []ClientSummary,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListFactorsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "factors")]
    pub factors: []Factor,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignUpResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CancelInvitationResult {
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetExtensionDataResult {
    #[serde(rename = "actions")]
    pub actions: []ActionDataDTO,
    #[serde(rename = "quickLinks")]
    pub quick_links: []QuickLinkDataDTO,
    #[serde(rename = "tabs")]
    pub tabs: []TabDataDTO,
    #[serde(rename = "widgets")]
    pub widgets: []WidgetDataDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockSessionService {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAPIKeyResponse {
    #[serde(rename = "api_key")]
    pub api_key: *apikey.APIKey,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationsConfig {
    #[serde(rename = "notifyOnRecoveryFailed")]
    pub notify_on_recovery_failed: bool,
    #[serde(rename = "notifyOnRecoveryStart")]
    pub notify_on_recovery_start: bool,
    #[serde(rename = "securityOfficerEmail")]
    pub security_officer_email: String,
    #[serde(rename = "channels")]
    pub channels: []string,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "notifyAdminOnHighRisk")]
    pub notify_admin_on_high_risk: bool,
    #[serde(rename = "notifyAdminOnReviewNeeded")]
    pub notify_admin_on_review_needed: bool,
    #[serde(rename = "notifyOnRecoveryComplete")]
    pub notify_on_recovery_complete: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionsResponse {
    #[serde(rename = "connections")]
    pub connections: []*schema.SocialAccount,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentRecord {
    #[serde(rename = "consentType")]
    pub consent_type: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "granted")]
    pub granted: bool,
    #[serde(rename = "grantedAt")]
    pub granted_at: time.Time,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "metadata")]
    pub metadata: JSONBMap,
    #[serde(rename = "purpose")]
    pub purpose: String,
    #[serde(rename = "revokedAt")]
    pub revoked_at: *time.Time,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCState {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StripeIdentityProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTemplateResult {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "template")]
    pub template: TemplateDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateDefault {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LinkRequest {
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "password")]
    pub password: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegenerateCodesRequest {
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "count")]
    pub count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateStorageConfig {
    #[serde(rename = "redisPassword")]
    pub redis_password: String,
    #[serde(rename = "stateTtl")]
    pub state_ttl: time.Duration,
    #[serde(rename = "useRedis")]
    pub use_redis: bool,
    #[serde(rename = "redisAddr")]
    pub redis_addr: String,
    #[serde(rename = "redisDb")]
    pub redis_db: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentSettingsResponse {
    #[serde(rename = "settings")]
    pub settings: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrivacySettingsRequest {
    #[serde(rename = "consentRequired")]
    pub consent_required: *bool,
    #[serde(rename = "cookieConsentStyle")]
    pub cookie_consent_style: String,
    #[serde(rename = "deletionGracePeriodDays")]
    pub deletion_grace_period_days: *int,
    #[serde(rename = "requireAdminApprovalForDeletion")]
    pub require_admin_approval_for_deletion: *bool,
    #[serde(rename = "requireExplicitConsent")]
    pub require_explicit_consent: *bool,
    #[serde(rename = "anonymousConsentEnabled")]
    pub anonymous_consent_enabled: *bool,
    #[serde(rename = "autoDeleteAfterDays")]
    pub auto_delete_after_days: *int,
    #[serde(rename = "dpoEmail")]
    pub dpo_email: String,
    #[serde(rename = "allowDataPortability")]
    pub allow_data_portability: *bool,
    #[serde(rename = "contactPhone")]
    pub contact_phone: String,
    #[serde(rename = "cookieConsentEnabled")]
    pub cookie_consent_enabled: *bool,
    #[serde(rename = "dataRetentionDays")]
    pub data_retention_days: *int,
    #[serde(rename = "gdprMode")]
    pub gdpr_mode: *bool,
    #[serde(rename = "ccpaMode")]
    pub ccpa_mode: *bool,
    #[serde(rename = "contactEmail")]
    pub contact_email: String,
    #[serde(rename = "dataExportExpiryHours")]
    pub data_export_expiry_hours: *int,
    #[serde(rename = "exportFormat")]
    pub export_format: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationContext {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemberDTO {
    #[serde(rename = "joinedAt")]
    pub joined_at: time.Time,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "userEmail")]
    pub user_email: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "userName")]
    pub user_name: String,
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SMSProviderDTO {
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeSessionResult {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetOrganizationBySlugRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RoleTemplateDTO {
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "permissions")]
    pub permissions: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthAutoSendDTO {
    #[serde(rename = "magicLink")]
    pub magic_link: bool,
    #[serde(rename = "mfaCode")]
    pub mfa_code: bool,
    #[serde(rename = "passwordReset")]
    pub password_reset: bool,
    #[serde(rename = "verificationEmail")]
    pub verification_email: bool,
    #[serde(rename = "welcome")]
    pub welcome: bool,
    #[serde(rename = "emailOtp")]
    pub email_otp: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationStatusResponse {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "environmentId")]
    pub environment_id: String,
    #[serde(rename = "failedCount")]
    pub failed_count: i32,
    #[serde(rename = "migratedCount")]
    pub migrated_count: i32,
    #[serde(rename = "progress")]
    pub progress: f64,
    #[serde(rename = "userOrganizationId")]
    pub user_organization_id: *string,
    #[serde(rename = "errors")]
    pub errors: []string,
    #[serde(rename = "startedAt")]
    pub started_at: time.Time,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "totalPolicies")]
    pub total_policies: i32,
    #[serde(rename = "validationPassed")]
    pub validation_passed: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourcesListResponse {
    #[serde(rename = "resources")]
    pub resources: []*ResourceResponse,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IPWhitelistConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "strict_mode")]
    pub strict_mode: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompliancePolicyResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddTeamMember_req {
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "member_id")]
    pub member_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IntrospectionService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*repo.OAuthTokenRepository>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecretItem {
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "path")]
    pub path: String,
    #[serde(rename = "tags")]
    pub tags: []string,
    #[serde(rename = "updatedAt")]
    pub updated_at: String,
    #[serde(rename = "valueType")]
    pub value_type: String,
    #[serde(rename = "key")]
    pub key: String,
    #[serde(rename = "version")]
    pub version: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSecretInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "path")]
    pub path: String,
    #[serde(rename = "tags")]
    pub tags: []string,
    #[serde(rename = "value")]
    pub value: ,
    #[serde(rename = "valueType")]
    pub value_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateMemberRoleInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "memberId")]
    pub member_id: String,
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "role")]
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NamespacesListResponse {
    #[serde(rename = "namespaces")]
    pub namespaces: []*NamespaceResponse,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RotateAPIKeyResponse {
    #[serde(rename = "api_key")]
    pub api_key: *apikey.APIKey,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditEvent {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListContentTypesRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
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
pub struct AdminUpdateProviderRequest {
    #[serde(rename = "scopes")]
    pub scopes: []string,
    #[serde(rename = "clientId")]
    pub client_id: *string,
    #[serde(rename = "clientSecret")]
    pub client_secret: *string,
    #[serde(rename = "enabled")]
    pub enabled: *bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceInfo {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "deviceId")]
    pub device_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceEvidencesResponse {
    #[serde(rename = "evidence")]
    pub evidence: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetImpersonationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateContentTypeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetEntryRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CancelRecoveryRequest {
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
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
pub struct UpdateProviderRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceReportResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetPasskeyRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RestoreEntryRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContentTypeHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*service.ContentFieldService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetContentTypeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientDetailsResponse {
    #[serde(rename = "applicationType")]
    pub application_type: String,
    #[serde(rename = "contacts")]
    pub contacts: []string,
    #[serde(rename = "grantTypes")]
    pub grant_types: []string,
    #[serde(rename = "isOrgLevel")]
    pub is_org_level: bool,
    #[serde(rename = "organizationID")]
    pub organization_i_d: String,
    #[serde(rename = "redirectURIs")]
    pub redirect_u_r_is: []string,
    #[serde(rename = "requireConsent")]
    pub require_consent: bool,
    #[serde(rename = "tosURI")]
    pub tos_u_r_i: String,
    #[serde(rename = "responseTypes")]
    pub response_types: []string,
    #[serde(rename = "tokenEndpointAuthMethod")]
    pub token_endpoint_auth_method: String,
    #[serde(rename = "clientID")]
    pub client_i_d: String,
    #[serde(rename = "policyURI")]
    pub policy_u_r_i: String,
    #[serde(rename = "requirePKCE")]
    pub require_p_k_c_e: bool,
    #[serde(rename = "updatedAt")]
    pub updated_at: String,
    #[serde(rename = "allowedScopes")]
    pub allowed_scopes: []string,
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "logoURI")]
    pub logo_u_r_i: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "postLogoutRedirectURIs")]
    pub post_logout_redirect_u_r_is: []string,
    #[serde(rename = "trustedClient")]
    pub trusted_client: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VideoSessionInfo {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRoleTemplateInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetOrganizationsInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "search")]
    pub search: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<error>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteProviderRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionStatsResponse {
    #[serde(rename = "activeSessions")]
    pub active_sessions: i32,
    #[serde(rename = "deviceCount")]
    pub device_count: i32,
    #[serde(rename = "locationCount")]
    pub location_count: i32,
    #[serde(rename = "newestSession")]
    pub newest_session: *string,
    #[serde(rename = "oldestSession")]
    pub oldest_session: *string,
    #[serde(rename = "totalSessions")]
    pub total_sessions: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NamespaceResponse {
    #[serde(rename = "inheritPlatform")]
    pub inherit_platform: bool,
    #[serde(rename = "policyCount")]
    pub policy_count: i32,
    #[serde(rename = "resourceCount")]
    pub resource_count: i32,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "templateId")]
    pub template_id: *string,
    #[serde(rename = "userOrganizationId")]
    pub user_organization_id: *string,
    #[serde(rename = "actionCount")]
    pub action_count: i32,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "environmentId")]
    pub environment_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Handler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*IntrospectionService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyRecoveryCodeRequest {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupCodeFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*twofa.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookConfig {
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
    #[serde(rename = "expiry_warning_days")]
    pub expiry_warning_days: i32,
    #[serde(rename = "notify_on_created")]
    pub notify_on_created: bool,
    #[serde(rename = "notify_on_deleted")]
    pub notify_on_deleted: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockAuditService {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListPasskeysRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentDecision {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]string>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceCodeEntryResponse {
    #[serde(rename = "basePath")]
    pub base_path: String,
    #[serde(rename = "formAction")]
    pub form_action: String,
    #[serde(rename = "placeholder")]
    pub placeholder: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthContactsResponse {
    #[serde(rename = "contacts")]
    pub contacts: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetStatsRequestDTO {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSessionsInput {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "search")]
    pub search: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "device")]
    pub device: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnalyticsResponse {
    #[serde(rename = "generatedAt")]
    pub generated_at: time.Time,
    #[serde(rename = "summary")]
    pub summary: AnalyticsSummary,
    #[serde(rename = "timeRange")]
    pub time_range: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentVerificationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]byte>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataExportRequestInput {
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "includeSections")]
    pub include_sections: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContextRule {
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "condition")]
    pub condition: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFAPolicy {
    #[serde(rename = "gracePeriodDays")]
    pub grace_period_days: i32,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "stepUpRequired")]
    pub step_up_required: bool,
    #[serde(rename = "trustedDeviceDays")]
    pub trusted_device_days: i32,
    #[serde(rename = "adaptiveMfaEnabled")]
    pub adaptive_mfa_enabled: bool,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "lockoutDurationMinutes")]
    pub lockout_duration_minutes: i32,
    #[serde(rename = "maxFailedAttempts")]
    pub max_failed_attempts: i32,
    #[serde(rename = "organizationId")]
    pub organization_id: xid.ID,
    #[serde(rename = "requiredFactorCount")]
    pub required_factor_count: i32,
    #[serde(rename = "requiredFactorTypes")]
    pub required_factor_types: []FactorType,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "allowedFactorTypes")]
    pub allowed_factor_types: []FactorType,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustedDevice {
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "lastUsedAt")]
    pub last_used_at: *time.Time,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
    #[serde(rename = "deviceId")]
    pub device_id: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrganizationDetailDTO {
    #[serde(rename = "logo")]
    pub logo: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "slug")]
    pub slug: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct sessionStats {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartVideoSessionRequest {
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecretRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetOverviewStatsResult {
    #[serde(rename = "stats")]
    pub stats: OverviewStatsDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestCase {
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "expected")]
    pub expected: bool,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "principal")]
    pub principal: ,
    #[serde(rename = "request")]
    pub request: ,
    #[serde(rename = "resource")]
    pub resource: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddFieldRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReorderFieldsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWTService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpNotificationProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetDocumentVerificationRequest {
    #[serde(rename = "documentId")]
    pub document_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRecoveryStatsResponse {
    #[serde(rename = "successRate")]
    pub success_rate: f64,
    #[serde(rename = "adminReviewsRequired")]
    pub admin_reviews_required: i32,
    #[serde(rename = "failedRecoveries")]
    pub failed_recoveries: i32,
    #[serde(rename = "highRiskAttempts")]
    pub high_risk_attempts: i32,
    #[serde(rename = "pendingRecoveries")]
    pub pending_recoveries: i32,
    #[serde(rename = "successfulRecoveries")]
    pub successful_recoveries: i32,
    #[serde(rename = "totalAttempts")]
    pub total_attempts: i32,
    #[serde(rename = "averageRiskScore")]
    pub average_risk_score: f64,
    #[serde(rename = "methodStats")]
    pub method_stats: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryCodeUsage {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyRecoveryCodeResponse {
    #[serde(rename = "remainingCodes")]
    pub remaining_codes: i32,
    #[serde(rename = "valid")]
    pub valid: bool,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BanUserRequest {
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
    #[serde(rename = "expires_at")]
    pub expires_at: *time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecretInput {
    #[serde(rename = "secretId")]
    pub secret_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VideoVerificationSession {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpVerificationsResponse {
    #[serde(rename = "verifications")]
    pub verifications: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DashboardConfig {
    #[serde(rename = "enableExport")]
    pub enable_export: bool,
    #[serde(rename = "enableImport")]
    pub enable_import: bool,
    #[serde(rename = "enableReveal")]
    pub enable_reveal: bool,
    #[serde(rename = "enableTreeView")]
    pub enable_tree_view: bool,
    #[serde(rename = "revealTimeout")]
    pub reveal_timeout: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InvitationResponse {
    #[serde(rename = "invitation")]
    pub invitation: *organization.Invitation,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnfidoConfig {
    #[serde(rename = "facialCheck")]
    pub facial_check: FacialCheckConfig,
    #[serde(rename = "includeDocumentReport")]
    pub include_document_report: bool,
    #[serde(rename = "includeWatchlistReport")]
    pub include_watchlist_report: bool,
    #[serde(rename = "region")]
    pub region: String,
    #[serde(rename = "workflowId")]
    pub workflow_id: String,
    #[serde(rename = "documentCheck")]
    pub document_check: DocumentCheckConfig,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "includeFacialReport")]
    pub include_facial_report: bool,
    #[serde(rename = "webhookToken")]
    pub webhook_token: String,
    #[serde(rename = "apiToken")]
    pub api_token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SaveBuilderTemplateInput {
    #[serde(rename = "builderJson")]
    pub builder_json: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "subject")]
    pub subject: String,
    #[serde(rename = "templateId")]
    pub template_id: String,
    #[serde(rename = "templateKey")]
    pub template_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockSentNotification {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReportsConfig {
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
    #[serde(rename = "includeEvidence")]
    pub include_evidence: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthSessionsResponse {
    #[serde(rename = "sessions")]
    pub sessions: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Status {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateSettingsInput {
    #[serde(rename = "settings")]
    pub settings: OrganizationSettingsDTO,
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BulkUnpublishRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "ids")]
    pub ids: []string,
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
pub struct ConsentAuditConfig {
    #[serde(rename = "archiveInterval")]
    pub archive_interval: time.Duration,
    #[serde(rename = "archiveOldLogs")]
    pub archive_old_logs: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "exportFormat")]
    pub export_format: String,
    #[serde(rename = "immutable")]
    pub immutable: bool,
    #[serde(rename = "logIpAddress")]
    pub log_ip_address: bool,
    #[serde(rename = "logUserAgent")]
    pub log_user_agent: bool,
    #[serde(rename = "signLogs")]
    pub sign_logs: bool,
    #[serde(rename = "logAllChanges")]
    pub log_all_changes: bool,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentReport {
    #[serde(rename = "completedDeletions")]
    pub completed_deletions: i32,
    #[serde(rename = "consentRate")]
    pub consent_rate: f64,
    #[serde(rename = "dpasActive")]
    pub dpas_active: i32,
    #[serde(rename = "dpasExpiringSoon")]
    pub dpas_expiring_soon: i32,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "reportPeriodEnd")]
    pub report_period_end: time.Time,
    #[serde(rename = "usersWithConsent")]
    pub users_with_consent: i32,
    #[serde(rename = "consentsByType")]
    pub consents_by_type: ,
    #[serde(rename = "dataExportsThisPeriod")]
    pub data_exports_this_period: i32,
    #[serde(rename = "pendingDeletions")]
    pub pending_deletions: i32,
    #[serde(rename = "reportPeriodStart")]
    pub report_period_start: time.Time,
    #[serde(rename = "totalUsers")]
    pub total_users: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTeamInput {
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateProfileFromTemplate_req {
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetAppRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasskeyInfo {
    #[serde(rename = "aaguid")]
    pub aaguid: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "signCount")]
    pub sign_count: i32,
    #[serde(rename = "authenticatorType")]
    pub authenticator_type: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "credentialId")]
    pub credential_id: String,
    #[serde(rename = "isResidentKey")]
    pub is_resident_key: bool,
    #[serde(rename = "lastUsedAt")]
    pub last_used_at: *time.Time,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDTokenClaims {
    #[serde(rename = "email_verified")]
    pub email_verified: bool,
    #[serde(rename = "family_name")]
    pub family_name: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "nonce")]
    pub nonce: String,
    #[serde(rename = "preferred_username")]
    pub preferred_username: String,
    #[serde(rename = "session_state")]
    pub session_state: String,
    #[serde(rename = "auth_time")]
    pub auth_time: i64,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "given_name")]
    pub given_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CookieConsentConfig {
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
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "requireExplicit")]
    pub require_explicit: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateSecretInput {
    #[serde(rename = "secretId")]
    pub secret_id: String,
    #[serde(rename = "tags")]
    pub tags: []string,
    #[serde(rename = "value")]
    pub value: ,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "changeReason")]
    pub change_reason: String,
    #[serde(rename = "description")]
    pub description: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListAPIKeysRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTemplateResponse {
    #[serde(rename = "standard")]
    pub standard: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimeBasedRule {
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "max_age")]
    pub max_age: time.Duration,
    #[serde(rename = "operation")]
    pub operation: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VersioningConfig {
    #[serde(rename = "maxVersions")]
    pub max_versions: i32,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "autoCleanup")]
    pub auto_cleanup: bool,
    #[serde(rename = "cleanupInterval")]
    pub cleanup_interval: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSecretRequest {
    #[serde(rename = "value")]
    pub value: ,
    #[serde(rename = "valueType")]
    pub value_type: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "path")]
    pub path: String,
    #[serde(rename = "tags")]
    pub tags: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateNamespaceRequest {
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "inheritPlatform")]
    pub inherit_platform: bool,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*RegistrationService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccessConfig {
    #[serde(rename = "allowApiAccess")]
    pub allow_api_access: bool,
    #[serde(rename = "allowDashboardAccess")]
    pub allow_dashboard_access: bool,
    #[serde(rename = "rateLimitPerMinute")]
    pub rate_limit_per_minute: i32,
    #[serde(rename = "requireAuthentication")]
    pub require_authentication: bool,
    #[serde(rename = "requireRbac")]
    pub require_rbac: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateOrganizationHandlerRequest {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteRoleTemplateResult {
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompliancePoliciesResponse {
    #[serde(rename = "policies")]
    pub policies: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationChannels {
    #[serde(rename = "webhook")]
    pub webhook: bool,
    #[serde(rename = "email")]
    pub email: bool,
    #[serde(rename = "slack")]
    pub slack: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceReport {
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "generatedBy")]
    pub generated_by: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "reportType")]
    pub report_type: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "fileUrl")]
    pub file_url: String,
    #[serde(rename = "period")]
    pub period: String,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "summary")]
    pub summary: ,
    #[serde(rename = "fileSize")]
    pub file_size: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListSessionsRequestDTO {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserVerificationStatusResponse {
    #[serde(rename = "status")]
    pub status: *base.UserVerificationStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTemplateInput {
    #[serde(rename = "variables")]
    pub variables: []string,
    #[serde(rename = "active")]
    pub active: *bool,
    #[serde(rename = "body")]
    pub body: *string,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: *string,
    #[serde(rename = "subject")]
    pub subject: *string,
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceStatus {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "checksPassed")]
    pub checks_passed: i32,
    #[serde(rename = "checksWarning")]
    pub checks_warning: i32,
    #[serde(rename = "lastChecked")]
    pub last_checked: time.Time,
    #[serde(rename = "violations")]
    pub violations: i32,
    #[serde(rename = "checksFailed")]
    pub checks_failed: i32,
    #[serde(rename = "nextAudit")]
    pub next_audit: time.Time,
    #[serde(rename = "overallStatus")]
    pub overall_status: String,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "score")]
    pub score: i32,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevisionHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*service.ContentTypeService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientAuthenticator {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*repo.OAuthClientRepository>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecurityQuestionsRequest {
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevealSecretOutput {
    #[serde(rename = "valueType")]
    pub value_type: String,
    #[serde(rename = "value")]
    pub value: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateAnalyticsDTO {
    #[serde(rename = "clickRate")]
    pub click_rate: f64,
    #[serde(rename = "openRate")]
    pub open_rate: f64,
    #[serde(rename = "totalClicked")]
    pub total_clicked: i64,
    #[serde(rename = "totalDelivered")]
    pub total_delivered: i64,
    #[serde(rename = "totalOpened")]
    pub total_opened: i64,
    #[serde(rename = "totalSent")]
    pub total_sent: i64,
    #[serde(rename = "deliveryRate")]
    pub delivery_rate: f64,
    #[serde(rename = "templateId")]
    pub template_id: String,
    #[serde(rename = "templateName")]
    pub template_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct userServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*user.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListSessionsRequest {
    #[serde(rename = "-")]
    pub -: xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "user_id")]
    pub user_id: *xid.ID,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonateUserRequestDTO {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "duration")]
    pub duration: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataExportRequest {
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "errorMessage")]
    pub error_message: String,
    #[serde(rename = "exportSize")]
    pub export_size: i64,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "includeSections")]
    pub include_sections: []string,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "exportUrl")]
    pub export_url: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "exportPath")]
    pub export_path: String,
    #[serde(rename = "format")]
    pub format: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiscoverProviderRequest {
    #[serde(rename = "email")]
    pub email: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResetTemplateRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendWithTemplateRequest {
    #[serde(rename = "type")]
    pub type: notification.NotificationType,
    #[serde(rename = "variables")]
    pub variables: ,
    #[serde(rename = "appId")]
    pub app_id: xid.ID,
    #[serde(rename = "language")]
    pub language: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "recipient")]
    pub recipient: String,
    #[serde(rename = "templateKey")]
    pub template_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceTypeStats {
    #[serde(rename = "allowRate")]
    pub allow_rate: f64,
    #[serde(rename = "avgLatencyMs")]
    pub avg_latency_ms: f64,
    #[serde(rename = "evaluationCount")]
    pub evaluation_count: i64,
    #[serde(rename = "resourceType")]
    pub resource_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteFactorRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterProviderRequest {
    #[serde(rename = "domain")]
    pub domain: String,
    #[serde(rename = "oidcClientID")]
    pub oidc_client_i_d: String,
    #[serde(rename = "oidcClientSecret")]
    pub oidc_client_secret: String,
    #[serde(rename = "oidcIssuer")]
    pub oidc_issuer: String,
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "oidcRedirectURI")]
    pub oidc_redirect_u_r_i: String,
    #[serde(rename = "samlCert")]
    pub saml_cert: String,
    #[serde(rename = "samlEntryPoint")]
    pub saml_entry_point: String,
    #[serde(rename = "samlIssuer")]
    pub saml_issuer: String,
    #[serde(rename = "type")]
    pub type: String,
    #[serde(rename = "attributeMapping")]
    pub attribute_mapping: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateVerificationSession_req {
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
    #[serde(rename = "cancelUrl")]
    pub cancel_url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSessionRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationSessionResponse {
    #[serde(rename = "session")]
    pub session: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationResponse {
    #[serde(rename = "notification")]
    pub notification: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeAllUserSessionsInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "userId")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockSessionService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*bun.DB>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetInvitationsInput {
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "page")]
    pub page: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSettingsInput {
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateProvidersInput {
    #[serde(rename = "emailProvider")]
    pub email_provider: *EmailProviderDTO,
    #[serde(rename = "smsProvider")]
    pub sms_provider: *SMSProviderDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResendNotificationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListReportsFilter {
    #[serde(rename = "standard")]
    pub standard: *ComplianceStandard,
    #[serde(rename = "status")]
    pub status: *string,
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "format")]
    pub format: *string,
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "reportType")]
    pub report_type: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteTrainingRequest {
    #[serde(rename = "score")]
    pub score: i32,
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
pub struct MemberHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*coreapp.ServiceImpl>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenRequest {
    #[serde(rename = "refresh_token")]
    pub refresh_token: String,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "audience")]
    pub audience: String,
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "client_secret")]
    pub client_secret: String,
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "code_verifier")]
    pub code_verifier: String,
    #[serde(rename = "device_code")]
    pub device_code: String,
    #[serde(rename = "grant_type")]
    pub grant_type: String,
    #[serde(rename = "redirect_uri")]
    pub redirect_uri: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentCookieResponse {
    #[serde(rename = "preferences")]
    pub preferences: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskFactor {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFAStatus {
    #[serde(rename = "enrolledFactors")]
    pub enrolled_factors: []FactorInfo,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetValueRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderRegisteredResponse {
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentCheckConfig {
    #[serde(rename = "validateExpiry")]
    pub validate_expiry: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "extractData")]
    pub extract_data: bool,
    #[serde(rename = "validateDataConsistency")]
    pub validate_data_consistency: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplatesListResponse {
    #[serde(rename = "categories")]
    pub categories: []string,
    #[serde(rename = "templates")]
    pub templates: []*TemplateResponse,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProvidersResponse {
    #[serde(rename = "providers")]
    pub providers: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryStateStore {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentExpiryConfig {
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
    #[serde(rename = "allowRenewal")]
    pub allow_renewal: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecretsOutput {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "secrets")]
    pub secrets: []SecretItem,
    #[serde(rename = "total")]
    pub total: i64,
    #[serde(rename = "totalPages")]
    pub total_pages: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestPolicyResponse {
    #[serde(rename = "total")]
    pub total: i32,
    #[serde(rename = "error")]
    pub error: String,
    #[serde(rename = "failedCount")]
    pub failed_count: i32,
    #[serde(rename = "passed")]
    pub passed: bool,
    #[serde(rename = "passedCount")]
    pub passed_count: i32,
    #[serde(rename = "results")]
    pub results: []TestCaseResult,
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
pub struct CreatePolicy_req {
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutomatedChecksConfig {
    #[serde(rename = "checkInterval")]
    pub check_interval: time.Duration,
    #[serde(rename = "inactiveUsers")]
    pub inactive_users: bool,
    #[serde(rename = "passwordPolicy")]
    pub password_policy: bool,
    #[serde(rename = "sessionPolicy")]
    pub session_policy: bool,
    #[serde(rename = "suspiciousActivity")]
    pub suspicious_activity: bool,
    #[serde(rename = "dataRetention")]
    pub data_retention: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "mfaCoverage")]
    pub mfa_coverage: bool,
    #[serde(rename = "accessReview")]
    pub access_review: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientAuthResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Service {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<Config>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecretsConfigSource {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateDTO {
    #[serde(rename = "body")]
    pub body: String,
    #[serde(rename = "isDefault")]
    pub is_default: bool,
    #[serde(rename = "templateKey")]
    pub template_key: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "isModified")]
    pub is_modified: bool,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "active")]
    pub active: bool,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "subject")]
    pub subject: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: String,
    #[serde(rename = "language")]
    pub language: String,
    #[serde(rename = "type")]
    pub type: String,
    #[serde(rename = "variables")]
    pub variables: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PoliciesListResponse {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "policies")]
    pub policies: []*PolicyResponse,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdatePasskeyResponse {
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "passkeyId")]
    pub passkey_id: String,
}

/// Webhook configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Webhook {
    #[serde(rename = "secret")]
    pub secret: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PublishContentTypeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartVideoSessionResponse {
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "sessionUrl")]
    pub session_url: String,
    #[serde(rename = "startedAt")]
    pub started_at: time.Time,
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateRoleTemplateResult {
    #[serde(rename = "template")]
    pub template: RoleTemplateDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListEvidenceFilter {
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "controlId")]
    pub control_id: *string,
    #[serde(rename = "evidenceType")]
    pub evidence_type: *string,
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "standard")]
    pub standard: *ComplianceStandard,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockAppService {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetEntryStatsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListUsersRequest {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "search")]
    pub search: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "-")]
    pub -: xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "limit")]
    pub limit: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StatusResponse {
    #[serde(rename = "emailVerified")]
    pub email_verified: bool,
    #[serde(rename = "emailVerifiedAt")]
    pub email_verified_at: *time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InitiateChallengeRequest {
    #[serde(rename = "context")]
    pub context: String,
    #[serde(rename = "factorTypes")]
    pub factor_types: []FactorType,
    #[serde(rename = "metadata")]
    pub metadata: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SMSConfig {
    #[serde(rename = "rate_limit")]
    pub rate_limit: *RateLimitConfig,
    #[serde(rename = "template_id")]
    pub template_id: String,
    #[serde(rename = "code_expiry_minutes")]
    pub code_expiry_minutes: i32,
    #[serde(rename = "code_length")]
    pub code_length: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "provider")]
    pub provider: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountLockoutError {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminBlockUser_req {
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplatesResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "templates")]
    pub templates: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProvidersConfigDTO {
    #[serde(rename = "emailProvider")]
    pub email_provider: EmailProviderDTO,
    #[serde(rename = "smsProvider")]
    pub sms_provider: SMSProviderDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityQuestionsConfig {
    #[serde(rename = "allowCustomQuestions")]
    pub allow_custom_questions: bool,
    #[serde(rename = "caseSensitive")]
    pub case_sensitive: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "forbidCommonAnswers")]
    pub forbid_common_answers: bool,
    #[serde(rename = "lockoutDuration")]
    pub lockout_duration: time.Duration,
    #[serde(rename = "minimumQuestions")]
    pub minimum_questions: i32,
    #[serde(rename = "requiredToRecover")]
    pub required_to_recover: i32,
    #[serde(rename = "maxAnswerLength")]
    pub max_answer_length: i32,
    #[serde(rename = "maxAttempts")]
    pub max_attempts: i32,
    #[serde(rename = "predefinedQuestions")]
    pub predefined_questions: []string,
    #[serde(rename = "requireMinLength")]
    pub require_min_length: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailConfig {
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "rate_limit")]
    pub rate_limit: *RateLimitConfig,
    #[serde(rename = "template_id")]
    pub template_id: String,
    #[serde(rename = "code_expiry_minutes")]
    pub code_expiry_minutes: i32,
    #[serde(rename = "code_length")]
    pub code_length: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetExtensionDataInput {
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*notification.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SettingsDTO {
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
pub struct AppServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateAppRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CallbackDataResponse {
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "isNewUser")]
    pub is_new_user: bool,
    #[serde(rename = "user")]
    pub user: *schema.User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResetUserMFARequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTreeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetOrganizationsResult {
    #[serde(rename = "data")]
    pub data: []OrganizationSummaryDTO,
    #[serde(rename = "pagination")]
    pub pagination: PaginationInfo,
    #[serde(rename = "stats")]
    pub stats: OrganizationStatsDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockOrganizationUIExtension {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]ui.OrganizationQuickLink>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateStore {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatePolicyRequest {
    #[serde(rename = "expression")]
    pub expression: String,
    #[serde(rename = "resourceType")]
    pub resource_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListViolationsFilter {
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "severity")]
    pub severity: *string,
    #[serde(rename = "status")]
    pub status: *string,
    #[serde(rename = "userId")]
    pub user_id: *string,
    #[serde(rename = "violationType")]
    pub violation_type: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegistrationService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<Config>,
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
pub struct RequirementsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "requirements")]
    pub requirements: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFAConfigResponse {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "required_factor_count")]
    pub required_factor_count: i32,
    #[serde(rename = "allowed_factor_types")]
    pub allowed_factor_types: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderInfo {
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "domain")]
    pub domain: String,
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StripeIdentityConfig {
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
    #[serde(rename = "returnUrl")]
    pub return_url: String,
    #[serde(rename = "useMock")]
    pub use_mock: bool,
    #[serde(rename = "webhookSecret")]
    pub webhook_secret: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionAutoSendConfig {
    #[serde(rename = "suspicious_login")]
    pub suspicious_login: bool,
    #[serde(rename = "all_revoked")]
    pub all_revoked: bool,
    #[serde(rename = "device_removed")]
    pub device_removed: bool,
    #[serde(rename = "new_device")]
    pub new_device: bool,
    #[serde(rename = "new_location")]
    pub new_location: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockImpersonationRepository {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct deviceFlowServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*deviceflow.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceDecisionResponse {
    #[serde(rename = "approved")]
    pub approved: bool,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnableRequest2FA {
    #[serde(rename = "method")]
    pub method: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockRepository {
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
pub struct ListTeamsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationListResponse {
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "offset")]
    pub offset: i32,
    #[serde(rename = "total")]
    pub total: i32,
    #[serde(rename = "verifications")]
    pub verifications: []*base.IdentityVerification,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionsListResponse {
    #[serde(rename = "actions")]
    pub actions: []*ActionResponse,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthContactResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnbanUserRequestDTO {
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*notificationPlugin.Adapter>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "factors")]
    pub factors: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetadataResponse {
    #[serde(rename = "metadata")]
    pub metadata: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "notifications")]
    pub notifications: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAppRequest {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditConfig {
    #[serde(rename = "logSuccessful")]
    pub log_successful: bool,
    #[serde(rename = "logAllAttempts")]
    pub log_all_attempts: bool,
    #[serde(rename = "logFailed")]
    pub log_failed: bool,
    #[serde(rename = "logIpAddress")]
    pub log_ip_address: bool,
    #[serde(rename = "logUserAgent")]
    pub log_user_agent: bool,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "archiveInterval")]
    pub archive_interval: time.Duration,
    #[serde(rename = "archiveOldLogs")]
    pub archive_old_logs: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "immutableLogs")]
    pub immutable_logs: bool,
    #[serde(rename = "logDeviceInfo")]
    pub log_device_info: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RequestTrustedContactVerificationRequest {
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
    #[serde(rename = "contactId")]
    pub contact_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SaveBuilderTemplateResult {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTrainingsResponse {
    #[serde(rename = "training")]
    pub training: Vec<>,
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
pub struct AuthURLResponse {
    #[serde(rename = "url")]
    pub url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelsResponse {
    #[serde(rename = "channels")]
    pub channels: ,
    #[serde(rename = "count")]
    pub count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RestoreTemplateVersionRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionStatsDTO {
    #[serde(rename = "tabletCount")]
    pub tablet_count: i32,
    #[serde(rename = "totalSessions")]
    pub total_sessions: i64,
    #[serde(rename = "uniqueUsers")]
    pub unique_users: i32,
    #[serde(rename = "activeCount")]
    pub active_count: i32,
    #[serde(rename = "desktopCount")]
    pub desktop_count: i32,
    #[serde(rename = "expiredCount")]
    pub expired_count: i32,
    #[serde(rename = "expiringCount")]
    pub expiring_count: i32,
    #[serde(rename = "mobileCount")]
    pub mobile_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*coreapp.ServiceImpl>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BulkDeleteRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "ids")]
    pub ids: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteEntryRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryAttemptLog {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthVideoResponse {
    #[serde(rename = "session_id")]
    pub session_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpEmailProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFAPolicyResponse {
    #[serde(rename = "allowedFactorTypes")]
    pub allowed_factor_types: []string,
    #[serde(rename = "appId")]
    pub app_id: xid.ID,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "gracePeriodDays")]
    pub grace_period_days: i32,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "organizationId")]
    pub organization_id: *xid.ID,
    #[serde(rename = "requiredFactorCount")]
    pub required_factor_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrgDetailStatsDTO {
    #[serde(rename = "teamCount")]
    pub team_count: i64,
    #[serde(rename = "invitationCount")]
    pub invitation_count: i64,
    #[serde(rename = "memberCount")]
    pub member_count: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginationInfoDTO {
    #[serde(rename = "currentPage")]
    pub current_page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "totalItems")]
    pub total_items: i64,
    #[serde(rename = "totalPages")]
    pub total_pages: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUserSessionsInput {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyStats {
    #[serde(rename = "policyName")]
    pub policy_name: String,
    #[serde(rename = "allowCount")]
    pub allow_count: i64,
    #[serde(rename = "avgLatencyMs")]
    pub avg_latency_ms: f64,
    #[serde(rename = "denyCount")]
    pub deny_count: i64,
    #[serde(rename = "evaluationCount")]
    pub evaluation_count: i64,
    #[serde(rename = "policyId")]
    pub policy_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentVerificationResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteVideoSessionResponse {
    #[serde(rename = "result")]
    pub result: String,
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
    #[serde(rename = "completedAt")]
    pub completed_at: time.Time,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthCodesResponse {
    #[serde(rename = "codes")]
    pub codes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreatePolicyRequest {
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "renewable")]
    pub renewable: bool,
    #[serde(rename = "validityPeriod")]
    pub validity_period: *int,
    #[serde(rename = "consentType")]
    pub consent_type: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "required")]
    pub required: bool,
    #[serde(rename = "version")]
    pub version: String,
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
pub struct CreateSecretOutput {
    #[serde(rename = "secret")]
    pub secret: SecretItem,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnalyticsDTO {
    #[serde(rename = "byDay")]
    pub by_day: []DailyAnalyticsDTO,
    #[serde(rename = "byTemplate")]
    pub by_template: []TemplateAnalyticsDTO,
    #[serde(rename = "overview")]
    pub overview: OverviewStatsDTO,
    #[serde(rename = "topTemplates")]
    pub top_templates: []TemplatePerformanceDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RolesResponse {
    #[serde(rename = "roles")]
    pub roles: []*apikey.Role,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyPair {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthQuestionsResponse {
    #[serde(rename = "questions")]
    pub questions: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BunRepository {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*bun.DB>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnableRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct stateEntry {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeTrustedDeviceRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct navItem {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContinueRecoveryRequest {
    #[serde(rename = "method")]
    pub method: RecoveryMethod,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProvidersConfig {
    #[serde(rename = "linkedin")]
    pub linkedin: *providers.ProviderConfig,
    #[serde(rename = "microsoft")]
    pub microsoft: *providers.ProviderConfig,
    #[serde(rename = "slack")]
    pub slack: *providers.ProviderConfig,
    #[serde(rename = "twitch")]
    pub twitch: *providers.ProviderConfig,
    #[serde(rename = "twitter")]
    pub twitter: *providers.ProviderConfig,
    #[serde(rename = "bitbucket")]
    pub bitbucket: *providers.ProviderConfig,
    #[serde(rename = "discord")]
    pub discord: *providers.ProviderConfig,
    #[serde(rename = "github")]
    pub github: *providers.ProviderConfig,
    #[serde(rename = "reddit")]
    pub reddit: *providers.ProviderConfig,
    #[serde(rename = "notion")]
    pub notion: *providers.ProviderConfig,
    #[serde(rename = "gitlab")]
    pub gitlab: *providers.ProviderConfig,
    #[serde(rename = "google")]
    pub google: *providers.ProviderConfig,
    #[serde(rename = "line")]
    pub line: *providers.ProviderConfig,
    #[serde(rename = "spotify")]
    pub spotify: *providers.ProviderConfig,
    #[serde(rename = "apple")]
    pub apple: *providers.ProviderConfig,
    #[serde(rename = "dropbox")]
    pub dropbox: *providers.ProviderConfig,
    #[serde(rename = "facebook")]
    pub facebook: *providers.ProviderConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TeamsResponse {
    #[serde(rename = "teams")]
    pub teams: []*organization.Team,
    #[serde(rename = "total")]
    pub total: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginationDTO {
    #[serde(rename = "hasPrev")]
    pub has_prev: bool,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "totalCount")]
    pub total_count: i64,
    #[serde(rename = "totalPages")]
    pub total_pages: i32,
    #[serde(rename = "currentPage")]
    pub current_page: i32,
    #[serde(rename = "hasNext")]
    pub has_next: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestProviderResult {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceAttributeRequest {
    #[serde(rename = "default")]
    pub default: ,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "required")]
    pub required: bool,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetMigrationStatusResponse {
    #[serde(rename = "hasMigratedPolicies")]
    pub has_migrated_policies: bool,
    #[serde(rename = "lastMigrationAt")]
    pub last_migration_at: String,
    #[serde(rename = "migratedCount")]
    pub migrated_count: i32,
    #[serde(rename = "pendingRbacPolicies")]
    pub pending_rbac_policies: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyAPIKeyRequest {
    #[serde(rename = "key")]
    pub key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyCodeRequest {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
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
pub struct OrganizationSettingsDTO {
    #[serde(rename = "requireInvitation")]
    pub require_invitation: bool,
    #[serde(rename = "allowMultipleMemberships")]
    pub allow_multiple_memberships: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "invitationExpiryDays")]
    pub invitation_expiry_days: i32,
    #[serde(rename = "maxOrgsPerUser")]
    pub max_orgs_per_user: i32,
    #[serde(rename = "maxTeamsPerOrg")]
    pub max_teams_per_org: i32,
    #[serde(rename = "allowUserCreation")]
    pub allow_user_creation: bool,
    #[serde(rename = "defaultRole")]
    pub default_role: String,
    #[serde(rename = "maxMembersPerOrg")]
    pub max_members_per_org: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateOrganizationInput {
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "logo")]
    pub logo: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateABTestVariant_req {
    #[serde(rename = "body")]
    pub body: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "subject")]
    pub subject: String,
    #[serde(rename = "weight")]
    pub weight: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyPreviewResponse {
    #[serde(rename = "actions")]
    pub actions: []string,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "expression")]
    pub expression: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "resourceType")]
    pub resource_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetAPIKeyRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AssignRoleRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "roleID")]
    pub role_i_d: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpDocumentProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Middleware {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*Config>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChallengeResponse {
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrganizationHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*Plugin>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationTemplateResponse {
    #[serde(rename = "template")]
    pub template: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionStats {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*session.Session>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTrainingRequest {
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "trainingType")]
    pub training_type: String,
    #[serde(rename = "userId")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyStore {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NoOpVideoProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustedContact {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeSessionRequestDTO {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceRule {
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "sensitivity")]
    pub sensitivity: String,
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "resource_type")]
    pub resource_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecretOutput {
    #[serde(rename = "secret")]
    pub secret: SecretItem,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TeamDTO {
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "memberCount")]
    pub member_count: i64,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrateAllResponse {
    #[serde(rename = "dryRun")]
    pub dry_run: bool,
    #[serde(rename = "failedPolicies")]
    pub failed_policies: i32,
    #[serde(rename = "migratedPolicies")]
    pub migrated_policies: i32,
    #[serde(rename = "startedAt")]
    pub started_at: String,
    #[serde(rename = "completedAt")]
    pub completed_at: String,
    #[serde(rename = "convertedPolicies")]
    pub converted_policies: []PolicyPreviewResponse,
    #[serde(rename = "errors")]
    pub errors: []MigrationErrorResponse,
    #[serde(rename = "skippedPolicies")]
    pub skipped_policies: i32,
    #[serde(rename = "totalPolicies")]
    pub total_policies: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityQuestion {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*xid.ID>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorInfo {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "type")]
    pub type: FactorType,
    #[serde(rename = "factorId")]
    pub factor_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceEvidence {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "evidenceType")]
    pub evidence_type: String,
    #[serde(rename = "fileHash")]
    pub file_hash: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "title")]
    pub title: String,
    #[serde(rename = "collectedBy")]
    pub collected_by: String,
    #[serde(rename = "controlId")]
    pub control_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "fileUrl")]
    pub file_url: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateMemberRequest {
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiscoveryResponse {
    #[serde(rename = "userinfo_endpoint")]
    pub userinfo_endpoint: String,
    #[serde(rename = "authorization_endpoint")]
    pub authorization_endpoint: String,
    #[serde(rename = "device_authorization_endpoint")]
    pub device_authorization_endpoint: String,
    #[serde(rename = "id_token_signing_alg_values_supported")]
    pub id_token_signing_alg_values_supported: []string,
    #[serde(rename = "request_parameter_supported")]
    pub request_parameter_supported: bool,
    #[serde(rename = "response_types_supported")]
    pub response_types_supported: []string,
    #[serde(rename = "token_endpoint_auth_methods_supported")]
    pub token_endpoint_auth_methods_supported: []string,
    #[serde(rename = "code_challenge_methods_supported")]
    pub code_challenge_methods_supported: []string,
    #[serde(rename = "grant_types_supported")]
    pub grant_types_supported: []string,
    #[serde(rename = "introspection_endpoint_auth_methods_supported")]
    pub introspection_endpoint_auth_methods_supported: []string,
    #[serde(rename = "issuer")]
    pub issuer: String,
    #[serde(rename = "claims_supported")]
    pub claims_supported: []string,
    #[serde(rename = "require_request_uri_registration")]
    pub require_request_uri_registration: bool,
    #[serde(rename = "revocation_endpoint")]
    pub revocation_endpoint: String,
    #[serde(rename = "revocation_endpoint_auth_methods_supported")]
    pub revocation_endpoint_auth_methods_supported: []string,
    #[serde(rename = "scopes_supported")]
    pub scopes_supported: []string,
    #[serde(rename = "subject_types_supported")]
    pub subject_types_supported: []string,
    #[serde(rename = "token_endpoint")]
    pub token_endpoint: String,
    #[serde(rename = "claims_parameter_supported")]
    pub claims_parameter_supported: bool,
    #[serde(rename = "introspection_endpoint")]
    pub introspection_endpoint: String,
    #[serde(rename = "jwks_uri")]
    pub jwks_uri: String,
    #[serde(rename = "registration_endpoint")]
    pub registration_endpoint: String,
    #[serde(rename = "request_uri_parameter_supported")]
    pub request_uri_parameter_supported: bool,
    #[serde(rename = "response_modes_supported")]
    pub response_modes_supported: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdatePolicyRequest {
    #[serde(rename = "validityPeriod")]
    pub validity_period: *int,
    #[serde(rename = "active")]
    pub active: *bool,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "renewable")]
    pub renewable: *bool,
    #[serde(rename = "required")]
    pub required: *bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationResult {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "required")]
    pub required: bool,
    #[serde(rename = "requirement_id")]
    pub requirement_id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "can_remember")]
    pub can_remember: bool,
    #[serde(rename = "challenge_token")]
    pub challenge_token: String,
    #[serde(rename = "expires_at")]
    pub expires_at: time.Time,
    #[serde(rename = "grace_period_ends_at")]
    pub grace_period_ends_at: time.Time,
    #[serde(rename = "allowed_methods")]
    pub allowed_methods: []VerificationMethod,
    #[serde(rename = "current_level")]
    pub current_level: SecurityLevel,
    #[serde(rename = "matched_rules")]
    pub matched_rules: []string,
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
pub struct TokenIntrospectionResponse {
    #[serde(rename = "sub")]
    pub sub: String,
    #[serde(rename = "jti")]
    pub jti: String,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "token_type")]
    pub token_type: String,
    #[serde(rename = "username")]
    pub username: String,
    #[serde(rename = "active")]
    pub active: bool,
    #[serde(rename = "aud")]
    pub aud: []string,
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "exp")]
    pub exp: i64,
    #[serde(rename = "iat")]
    pub iat: i64,
    #[serde(rename = "iss")]
    pub iss: String,
    #[serde(rename = "nbf")]
    pub nbf: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CookieConsentRequest {
    #[serde(rename = "functional")]
    pub functional: bool,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateMemberRoleResult {
    #[serde(rename = "member")]
    pub member: MemberDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeSessionInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "sessionId")]
    pub session_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyStats {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceStatusDetailsResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateProfileRequest {
    #[serde(rename = "mfaRequired")]
    pub mfa_required: bool,
    #[serde(rename = "passwordRequireLower")]
    pub password_require_lower: bool,
    #[serde(rename = "passwordRequireNumber")]
    pub password_require_number: bool,
    #[serde(rename = "rbacRequired")]
    pub rbac_required: bool,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "auditLogExport")]
    pub audit_log_export: bool,
    #[serde(rename = "complianceContact")]
    pub compliance_contact: String,
    #[serde(rename = "dataResidency")]
    pub data_residency: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "passwordRequireSymbol")]
    pub password_require_symbol: bool,
    #[serde(rename = "regularAccessReview")]
    pub regular_access_review: bool,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "dpoContact")]
    pub dpo_contact: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "sessionIpBinding")]
    pub session_ip_binding: bool,
    #[serde(rename = "sessionMaxAge")]
    pub session_max_age: i32,
    #[serde(rename = "standards")]
    pub standards: []ComplianceStandard,
    #[serde(rename = "detailedAuditTrail")]
    pub detailed_audit_trail: bool,
    #[serde(rename = "passwordExpiryDays")]
    pub password_expiry_days: i32,
    #[serde(rename = "passwordMinLength")]
    pub password_min_length: i32,
    #[serde(rename = "passwordRequireUpper")]
    pub password_require_upper: bool,
    #[serde(rename = "sessionIdleTimeout")]
    pub session_idle_timeout: i32,
    #[serde(rename = "encryptionAtRest")]
    pub encryption_at_rest: bool,
    #[serde(rename = "encryptionInTransit")]
    pub encryption_in_transit: bool,
    #[serde(rename = "leastPrivilege")]
    pub least_privilege: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceReportsResponse {
    #[serde(rename = "reports")]
    pub reports: Vec<>,
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
pub struct CallbackResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteOrganizationInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "orgId")]
    pub org_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PreviewTemplateRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetActiveRequest {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceCheck {
    #[serde(rename = "checkType")]
    pub check_type: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "evidence")]
    pub evidence: []string,
    #[serde(rename = "lastCheckedAt")]
    pub last_checked_at: time.Time,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "nextCheckAt")]
    pub next_check_at: time.Time,
    #[serde(rename = "result")]
    pub result: ,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLog {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginResponse {
    #[serde(rename = "passkeyUsed")]
    pub passkey_used: String,
    #[serde(rename = "session")]
    pub session: ,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifySecurityAnswersRequest {
    #[serde(rename = "answers")]
    pub answers: ,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentVerification {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AmountRule {
    #[serde(rename = "min_amount")]
    pub min_amount: f64,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "currency")]
    pub currency: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "max_amount")]
    pub max_amount: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<sync.RWMutex>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockUserService {
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
pub struct DeletePasskeyRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TOTPSecret {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
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
pub struct StepUpRequirementResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListSecretsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]string>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateProvidersResult {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "providers")]
    pub providers: ProvidersConfigDTO,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeAllUserSessionsResult {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "revokedCount")]
    pub revoked_count: i32,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLogEntry {
    #[serde(rename = "oldValue")]
    pub old_value: ,
    #[serde(rename = "resourceId")]
    pub resource_id: String,
    #[serde(rename = "actorId")]
    pub actor_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "resourceType")]
    pub resource_type: String,
    #[serde(rename = "timestamp")]
    pub timestamp: time.Time,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "userOrganizationId")]
    pub user_organization_id: *string,
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "environmentId")]
    pub environment_id: String,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "newValue")]
    pub new_value: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteTraining_req {
    #[serde(rename = "score")]
    pub score: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListRecoverySessionsResponse {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "sessions")]
    pub sessions: []RecoverySessionInfo,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApproveRecoveryRequest {
    #[serde(rename = "notes")]
    pub notes: String,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetChallengeStatusResponse {
    #[serde(rename = "factorsRequired")]
    pub factors_required: i32,
    #[serde(rename = "factorsVerified")]
    pub factors_verified: i32,
    #[serde(rename = "maxAttempts")]
    pub max_attempts: i32,
    #[serde(rename = "status")]
    pub status: ChallengeStatus,
    #[serde(rename = "attempts")]
    pub attempts: i32,
    #[serde(rename = "availableFactors")]
    pub available_factors: []FactorInfo,
    #[serde(rename = "challengeId")]
    pub challenge_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListOrganizationsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceUserTrainingResponse {
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FinishLoginRequest {
    #[serde(rename = "response")]
    pub response: ,
    #[serde(rename = "remember")]
    pub remember: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContentEntryHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*service.ContentTypeService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientRegistrationResponse {
    #[serde(rename = "application_type")]
    pub application_type: String,
    #[serde(rename = "client_id_issued_at")]
    pub client_id_issued_at: i64,
    #[serde(rename = "grant_types")]
    pub grant_types: []string,
    #[serde(rename = "response_types")]
    pub response_types: []string,
    #[serde(rename = "tos_uri")]
    pub tos_uri: String,
    #[serde(rename = "client_secret_expires_at")]
    pub client_secret_expires_at: i64,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "contacts")]
    pub contacts: []string,
    #[serde(rename = "policy_uri")]
    pub policy_uri: String,
    #[serde(rename = "token_endpoint_auth_method")]
    pub token_endpoint_auth_method: String,
    #[serde(rename = "client_name")]
    pub client_name: String,
    #[serde(rename = "client_secret")]
    pub client_secret: String,
    #[serde(rename = "logo_uri")]
    pub logo_uri: String,
    #[serde(rename = "post_logout_redirect_uris")]
    pub post_logout_redirect_uris: []string,
    #[serde(rename = "redirect_uris")]
    pub redirect_uris: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenResponse {
    #[serde(rename = "access_token")]
    pub access_token: String,
    #[serde(rename = "expires_in")]
    pub expires_in: i32,
    #[serde(rename = "id_token")]
    pub id_token: String,
    #[serde(rename = "refresh_token")]
    pub refresh_token: String,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "token_type")]
    pub token_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteRecoveryRequest {
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailVerificationConfig {
    #[serde(rename = "fromAddress")]
    pub from_address: String,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrustedContactInfo {
    #[serde(rename = "id")]
    pub id: xid.ID,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SMSFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*phone.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetByPathRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteSecretInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "secretId")]
    pub secret_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetDocumentVerificationResponse {
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
    #[serde(rename = "rejectionReason")]
    pub rejection_reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorResponse {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "details")]
    pub details: ,
    #[serde(rename = "error")]
    pub error: String,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentExportResponse {
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateRoleTemplateResult {
    #[serde(rename = "template")]
    pub template: RoleTemplateDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InviteMemberResult {
    #[serde(rename = "invitation")]
    pub invitation: InvitationDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PreviewTemplateInput {
    #[serde(rename = "templateId")]
    pub template_id: String,
    #[serde(rename = "variables")]
    pub variables: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RotateAPIKeyRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckResult {
    #[serde(rename = "result")]
    pub result: ,
    #[serde(rename = "score")]
    pub score: f64,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "checkType")]
    pub check_type: String,
    #[serde(rename = "error")]
    pub error: error,
    #[serde(rename = "evidence")]
    pub evidence: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct serviceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientSummary {
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "applicationType")]
    pub application_type: String,
    #[serde(rename = "clientID")]
    pub client_i_d: String,
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "isOrgLevel")]
    pub is_org_level: bool,
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
pub struct ConsentExportFileResponse {
    #[serde(rename = "content_type")]
    pub content_type: String,
    #[serde(rename = "data")]
    pub data: []byte,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateDPARequest {
    #[serde(rename = "effectiveDate")]
    pub effective_date: time.Time,
    #[serde(rename = "expiryDate")]
    pub expiry_date: *time.Time,
    #[serde(rename = "signedByName")]
    pub signed_by_name: String,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "signedByEmail")]
    pub signed_by_email: String,
    #[serde(rename = "signedByTitle")]
    pub signed_by_title: String,
    #[serde(rename = "agreementType")]
    pub agreement_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFASession {
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "factorsRequired")]
    pub factors_required: i32,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
    #[serde(rename = "verifiedFactors")]
    pub verified_factors: []xid.ID,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "factorsVerified")]
    pub factors_verified: i32,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "riskLevel")]
    pub risk_level: RiskLevel,
    #[serde(rename = "sessionToken")]
    pub session_token: String,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessment {
    #[serde(rename = "recommended")]
    pub recommended: []FactorType,
    #[serde(rename = "score")]
    pub score: f64,
    #[serde(rename = "factors")]
    pub factors: []string,
    #[serde(rename = "level")]
    pub level: RiskLevel,
    #[serde(rename = "metadata")]
    pub metadata: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimitConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "lockout_minutes")]
    pub lockout_minutes: i32,
    #[serde(rename = "max_attempts")]
    pub max_attempts: i32,
    #[serde(rename = "window_minutes")]
    pub window_minutes: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceVerificationInfo {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]ScopeInfo>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendOTPRequest {
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BanUserRequestDTO {
    #[serde(rename = "expires_at")]
    pub expires_at: *time.Time,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockRepository {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentSummary {
    #[serde(rename = "expiredConsents")]
    pub expired_consents: i32,
    #[serde(rename = "hasPendingDeletion")]
    pub has_pending_deletion: bool,
    #[serde(rename = "pendingRenewals")]
    pub pending_renewals: i32,
    #[serde(rename = "grantedConsents")]
    pub granted_consents: i32,
    #[serde(rename = "hasPendingExport")]
    pub has_pending_export: bool,
    #[serde(rename = "lastConsentUpdate")]
    pub last_consent_update: *time.Time,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "revokedConsents")]
    pub revoked_consents: i32,
    #[serde(rename = "totalConsents")]
    pub total_consents: i32,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "consentsByType")]
    pub consents_by_type: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationSessionResponse {
    #[serde(rename = "session")]
    pub session: *base.IdentityVerificationSession,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteTemplateResult {
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationListResponse {
    #[serde(rename = "notifications")]
    pub notifications: Vec<>,
    #[serde(rename = "total")]
    pub total: i32,
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
pub struct SignUpRequest {
    #[serde(rename = "password")]
    pub password: String,
    #[serde(rename = "username")]
    pub username: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteTeamInput {
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "teamId")]
    pub team_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationWebhookResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AMLMatch {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JumioProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<JumioConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTemplateRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateResponse {
    #[serde(rename = "parameters")]
    pub parameters: []core.TemplateParameter,
    #[serde(rename = "category")]
    pub category: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "examples")]
    pub examples: []string,
    #[serde(rename = "expression")]
    pub expression: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpAttempt {
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "method")]
    pub method: VerificationMethod,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "requirement_id")]
    pub requirement_id: String,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "failure_reason")]
    pub failure_reason: String,
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
}

/// User session
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Session {
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
    #[serde(rename = "userAgent", skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    #[serde(rename = "createdAt")]
    pub created_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTeamsResult {
    #[serde(rename = "canManage")]
    pub can_manage: bool,
    #[serde(rename = "data")]
    pub data: []TeamDTO,
    #[serde(rename = "pagination")]
    pub pagination: PaginationInfo,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateMemberHandlerRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BatchEvaluateRequest {
    #[serde(rename = "requests")]
    pub requests: []EvaluateRequest,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLogResponse {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
    #[serde(rename = "entries")]
    pub entries: []*AuditLogEntry,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*notification.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddTeamMemberRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "member_id")]
    pub member_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTeamInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "teamId")]
    pub team_id: String,
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
pub struct NoOpSMSProvider {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpEvaluationResponse {
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "required")]
    pub required: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetProviderRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluateResponse {
    #[serde(rename = "allowed")]
    pub allowed: bool,
    #[serde(rename = "cacheHit")]
    pub cache_hit: bool,
    #[serde(rename = "error")]
    pub error: String,
    #[serde(rename = "evaluatedPolicies")]
    pub evaluated_policies: i32,
    #[serde(rename = "evaluationTimeMs")]
    pub evaluation_time_ms: f64,
    #[serde(rename = "matchedPolicies")]
    pub matched_policies: []string,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateAPIKeyRequest {
    #[serde(rename = "permissions")]
    pub permissions: ,
    #[serde(rename = "rate_limit")]
    pub rate_limit: *int,
    #[serde(rename = "scopes")]
    pub scopes: []string,
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "allowed_ips")]
    pub allowed_ips: []string,
    #[serde(rename = "description")]
    pub description: *string,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoveTeamMemberRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OAuthErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
    #[serde(rename = "error_description")]
    pub error_description: String,
    #[serde(rename = "error_uri")]
    pub error_uri: String,
    #[serde(rename = "state")]
    pub state: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthRecoveryResponse {
    #[serde(rename = "session_id")]
    pub session_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddTrustedContactRequest {
    #[serde(rename = "phone")]
    pub phone: String,
    #[serde(rename = "relationship")]
    pub relationship: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyResponse {
    #[serde(rename = "session")]
    pub session: *session.Session,
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: *user.User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateConsentRequest {
    #[serde(rename = "purpose")]
    pub purpose: String,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationsResponse {
    #[serde(rename = "verifications")]
    pub verifications: ,
    #[serde(rename = "count")]
    pub count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaValidator {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTeamResult {
    #[serde(rename = "team")]
    pub team: TeamDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VideoSessionResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListUsersRequestDTO {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockUserRequest {
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrackNotificationEvent_req {
    #[serde(rename = "organizationId", skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<*string>,
    #[serde(rename = "templateId")]
    pub template_id: String,
    #[serde(rename = "event")]
    pub event: String,
    #[serde(rename = "eventData", skip_serializing_if = "Option::is_none")]
    pub event_data: Option<>,
    #[serde(rename = "notificationId")]
    pub notification_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAPIKeyRequest {
    #[serde(rename = "rate_limit")]
    pub rate_limit: i32,
    #[serde(rename = "scopes")]
    pub scopes: []string,
    #[serde(rename = "allowed_ips")]
    pub allowed_ips: []string,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "permissions")]
    pub permissions: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceViolation {
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "resolvedBy")]
    pub resolved_by: String,
    #[serde(rename = "severity")]
    pub severity: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "violationType")]
    pub violation_type: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "resolvedAt")]
    pub resolved_at: *time.Time,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartImpersonationRequest {
    #[serde(rename = "target_user_id")]
    pub target_user_id: String,
    #[serde(rename = "ticket_number")]
    pub ticket_number: String,
    #[serde(rename = "duration_minutes")]
    pub duration_minutes: i32,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ArchiveEntryRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
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
pub struct SignInResponse {
    #[serde(rename = "session")]
    pub session: *session.Session,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: *user.User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRoleTemplateResult {
    #[serde(rename = "template")]
    pub template: RoleTemplateDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InvitationDTO {
    #[serde(rename = "invitedBy")]
    pub invited_by: String,
    #[serde(rename = "inviterName")]
    pub inviter_name: String,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
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
pub struct GetSessionInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "sessionId")]
    pub session_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PreviewConversionResponse {
    #[serde(rename = "resourceType")]
    pub resource_type: String,
    #[serde(rename = "success")]
    pub success: bool,
    #[serde(rename = "celExpression")]
    pub cel_expression: String,
    #[serde(rename = "error")]
    pub error: String,
    #[serde(rename = "policyName")]
    pub policy_name: String,
    #[serde(rename = "resourceId")]
    pub resource_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChallengeSession {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Time>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    #[serde(rename = "issuer")]
    pub issuer: String,
    #[serde(rename = "keys")]
    pub keys: ,
    #[serde(rename = "tokens")]
    pub tokens: ,
    #[serde(rename = "deviceFlow")]
    pub device_flow: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryCodesConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "regenerateCount")]
    pub regenerate_count: i32,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFASendOTPResponse {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockSocialAccountRepository {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteOrganizationResult {
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteTemplateRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutoSendConfig {
    #[serde(rename = "auth")]
    pub auth: AuthAutoSendConfig,
    #[serde(rename = "organization")]
    pub organization: OrganizationAutoSendConfig,
    #[serde(rename = "session")]
    pub session: SessionAutoSendConfig,
    #[serde(rename = "account")]
    pub account: AccountAutoSendConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestPolicyRequest {
    #[serde(rename = "actions")]
    pub actions: []string,
    #[serde(rename = "expression")]
    pub expression: String,
    #[serde(rename = "resourceType")]
    pub resource_type: String,
    #[serde(rename = "testCases")]
    pub test_cases: []TestCase,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebAuthnConfig {
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
    #[serde(rename = "rp_display_name")]
    pub rp_display_name: String,
    #[serde(rename = "rp_id")]
    pub rp_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateOrganizationInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "logo")]
    pub logo: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "slug")]
    pub slug: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoveMemberRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeclineInvitationRequest {
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SAMLLoginResponse {
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "redirectUrl")]
    pub redirect_url: String,
    #[serde(rename = "requestId")]
    pub request_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplateEngine {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<template.FuncMap>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<[]webauthn.Credential>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryChallengeStore {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifySecurityAnswersResponse {
    #[serde(rename = "attemptsLeft")]
    pub attempts_left: i32,
    #[serde(rename = "correctAnswers")]
    pub correct_answers: i32,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "requiredAnswers")]
    pub required_answers: i32,
    #[serde(rename = "valid")]
    pub valid: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResendRequest {
    #[serde(rename = "email")]
    pub email: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpAuditLogsResponse {
    #[serde(rename = "audit_logs")]
    pub audit_logs: Vec<>,
}

/// User device
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Device {
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "name", skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(rename = "type", skip_serializing_if = "Option::is_none")]
    pub type: Option<String>,
    #[serde(rename = "lastUsedAt")]
    pub last_used_at: String,
    #[serde(rename = "ipAddress", skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(rename = "userAgent", skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcceptInvitationRequest {
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuickLinkDataDTO {
    #[serde(rename = "url")]
    pub url: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "icon")]
    pub icon: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "order")]
    pub order: i32,
    #[serde(rename = "requireAdmin")]
    pub require_admin: bool,
    #[serde(rename = "title")]
    pub title: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTemplateAnalyticsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TeamHandler {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*app.ServiceImpl>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthDocumentResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpPolicy {
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "priority")]
    pub priority: i32,
    #[serde(rename = "rules")]
    pub rules: ,
    #[serde(rename = "updated_at")]
    pub updated_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChallengeStatusResponse {
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
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TOTPFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*twofa.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorAdapterRegistry {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SAMLLoginRequest {
    #[serde(rename = "relayState")]
    pub relay_state: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnblockUserRequest {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetNotificationDetailResult {
    #[serde(rename = "notification")]
    pub notification: NotificationHistoryDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScheduleVideoSessionResponse {
    #[serde(rename = "joinUrl")]
    pub join_url: String,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "scheduledAt")]
    pub scheduled_at: time.Time,
    #[serde(rename = "videoSessionId")]
    pub video_session_id: xid.ID,
    #[serde(rename = "instructions")]
    pub instructions: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrivacySettings {
    #[serde(rename = "dataExportExpiryHours")]
    pub data_export_expiry_hours: i32,
    #[serde(rename = "deletionGracePeriodDays")]
    pub deletion_grace_period_days: i32,
    #[serde(rename = "exportFormat")]
    pub export_format: []string,
    #[serde(rename = "requireExplicitConsent")]
    pub require_explicit_consent: bool,
    #[serde(rename = "anonymousConsentEnabled")]
    pub anonymous_consent_enabled: bool,
    #[serde(rename = "cookieConsentEnabled")]
    pub cookie_consent_enabled: bool,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "dataRetentionDays")]
    pub data_retention_days: i32,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "metadata")]
    pub metadata: JSONBMap,
    #[serde(rename = "requireAdminApprovalForDeletion")]
    pub require_admin_approval_for_deletion: bool,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "allowDataPortability")]
    pub allow_data_portability: bool,
    #[serde(rename = "autoDeleteAfterDays")]
    pub auto_delete_after_days: i32,
    #[serde(rename = "contactPhone")]
    pub contact_phone: String,
    #[serde(rename = "dpoEmail")]
    pub dpo_email: String,
    #[serde(rename = "gdprMode")]
    pub gdpr_mode: bool,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "ccpaMode")]
    pub ccpa_mode: bool,
    #[serde(rename = "consentRequired")]
    pub consent_required: bool,
    #[serde(rename = "contactEmail")]
    pub contact_email: String,
    #[serde(rename = "cookieConsentStyle")]
    pub cookie_consent_style: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluateRequest {
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
    #[serde(rename = "resource_type")]
    pub resource_type: String,
    #[serde(rename = "route")]
    pub route: String,
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
pub struct VerifyEnrolledFactorRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "data")]
    pub data: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetAnalyticsResult {
    #[serde(rename = "analytics")]
    pub analytics: AnalyticsDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUserSessionsResult {
    #[serde(rename = "activeCount")]
    pub active_count: i32,
    #[serde(rename = "pagination")]
    pub pagination: PaginationInfoDTO,
    #[serde(rename = "sessions")]
    pub sessions: []SessionDTO,
    #[serde(rename = "totalCount")]
    pub total_count: i32,
    #[serde(rename = "userId")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BatchEvaluationResult {
    #[serde(rename = "evaluationTimeMs")]
    pub evaluation_time_ms: f64,
    #[serde(rename = "index")]
    pub index: i32,
    #[serde(rename = "policies")]
    pub policies: []string,
    #[serde(rename = "resourceId")]
    pub resource_id: String,
    #[serde(rename = "resourceType")]
    pub resource_type: String,
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "allowed")]
    pub allowed: bool,
    #[serde(rename = "error")]
    pub error: String,
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetUserRoleRequestDTO {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "role")]
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimitRule {
    #[serde(rename = "max")]
    pub max: i32,
    #[serde(rename = "window")]
    pub window: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstantiateTemplateRequest {
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "namespaceId")]
    pub namespace_id: String,
    #[serde(rename = "parameters")]
    pub parameters: ,
    #[serde(rename = "priority")]
    pub priority: i32,
    #[serde(rename = "resourceType")]
    pub resource_type: String,
    #[serde(rename = "actions")]
    pub actions: []string,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BeginRegisterRequest {
    #[serde(rename = "userVerification")]
    pub user_verification: String,
    #[serde(rename = "authenticatorType")]
    pub authenticator_type: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "requireResidentKey")]
    pub require_resident_key: bool,
    #[serde(rename = "userId")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RestoreRevisionRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListEntriesRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WidgetDataDTO {
    #[serde(rename = "requireAdmin")]
    pub require_admin: bool,
    #[serde(rename = "size")]
    pub size: i32,
    #[serde(rename = "title")]
    pub title: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "icon")]
    pub icon: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "order")]
    pub order: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrganizationUIRegistry {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TabDataDTO {
    #[serde(rename = "label")]
    pub label: String,
    #[serde(rename = "order")]
    pub order: i32,
    #[serde(rename = "path")]
    pub path: String,
    #[serde(rename = "requireAdmin")]
    pub require_admin: bool,
    #[serde(rename = "icon")]
    pub icon: String,
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListNotificationsHistoryInput {
    #[serde(rename = "type")]
    pub type: *string,
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "recipient")]
    pub recipient: *string,
    #[serde(rename = "status")]
    pub status: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetInvitationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddMemberRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InviteMemberRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "role")]
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentRequest {
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "state")]
    pub state: String,
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "code_challenge")]
    pub code_challenge: String,
    #[serde(rename = "code_challenge_method")]
    pub code_challenge_method: String,
    #[serde(rename = "redirect_uri")]
    pub redirect_uri: String,
    #[serde(rename = "response_type")]
    pub response_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateRecoveryConfigRequest {
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
pub struct ListUsersResponse {
    #[serde(rename = "users")]
    pub users: []*user.User,
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "total")]
    pub total: i32,
    #[serde(rename = "total_pages")]
    pub total_pages: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpRequirement {
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "expires_at")]
    pub expires_at: time.Time,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "risk_score")]
    pub risk_score: f64,
    #[serde(rename = "rule_name")]
    pub rule_name: String,
    #[serde(rename = "session_id")]
    pub session_id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "currency")]
    pub currency: String,
    #[serde(rename = "current_level")]
    pub current_level: SecurityLevel,
    #[serde(rename = "resource_type")]
    pub resource_type: String,
    #[serde(rename = "amount")]
    pub amount: f64,
    #[serde(rename = "challenge_token")]
    pub challenge_token: String,
    #[serde(rename = "required_level")]
    pub required_level: SecurityLevel,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "fulfilled_at")]
    pub fulfilled_at: *time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "method")]
    pub method: String,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "resource_action")]
    pub resource_action: String,
    #[serde(rename = "route")]
    pub route: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptionConfig {
    #[serde(rename = "masterKey")]
    pub master_key: String,
    #[serde(rename = "rotateKeyAfter")]
    pub rotate_key_after: time.Duration,
    #[serde(rename = "testOnStartup")]
    pub test_on_startup: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteOrganizationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "migrationId")]
    pub migration_id: String,
    #[serde(rename = "startedAt")]
    pub started_at: time.Time,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTraining {
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "score")]
    pub score: i32,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "trainingType")]
    pub training_type: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonateUserRequest {
    #[serde(rename = "-")]
    pub -: xid.ID,
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
pub struct StepUpRequirementsResponse {
    #[serde(rename = "requirements")]
    pub requirements: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetMembersInput {
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "search")]
    pub search: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "limit")]
    pub limit: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateOrganizationHandlerRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookPayload {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateProvider_req {
    #[serde(rename = "isDefault")]
    pub is_default: bool,
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "isActive")]
    pub is_active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceProfile {
    #[serde(rename = "sessionIdleTimeout")]
    pub session_idle_timeout: i32,
    #[serde(rename = "passwordRequireNumber")]
    pub password_require_number: bool,
    #[serde(rename = "standards")]
    pub standards: []ComplianceStandard,
    #[serde(rename = "mfaRequired")]
    pub mfa_required: bool,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "encryptionInTransit")]
    pub encryption_in_transit: bool,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "complianceContact")]
    pub compliance_contact: String,
    #[serde(rename = "passwordMinLength")]
    pub password_min_length: i32,
    #[serde(rename = "passwordRequireLower")]
    pub password_require_lower: bool,
    #[serde(rename = "rbacRequired")]
    pub rbac_required: bool,
    #[serde(rename = "regularAccessReview")]
    pub regular_access_review: bool,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "auditLogExport")]
    pub audit_log_export: bool,
    #[serde(rename = "passwordRequireUpper")]
    pub password_require_upper: bool,
    #[serde(rename = "passwordRequireSymbol")]
    pub password_require_symbol: bool,
    #[serde(rename = "retentionDays")]
    pub retention_days: i32,
    #[serde(rename = "sessionIpBinding")]
    pub session_ip_binding: bool,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "dataResidency")]
    pub data_residency: String,
    #[serde(rename = "encryptionAtRest")]
    pub encryption_at_rest: bool,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "passwordExpiryDays")]
    pub password_expiry_days: i32,
    #[serde(rename = "detailedAuditTrail")]
    pub detailed_audit_trail: bool,
    #[serde(rename = "dpoContact")]
    pub dpo_contact: String,
    #[serde(rename = "sessionMaxAge")]
    pub session_max_age: i32,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "leastPrivilege")]
    pub least_privilege: bool,
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
pub struct DeviceVerifyResponse {
    #[serde(rename = "authorizeUrl")]
    pub authorize_url: String,
    #[serde(rename = "clientId")]
    pub client_id: String,
    #[serde(rename = "clientName")]
    pub client_name: String,
    #[serde(rename = "logoUri")]
    pub logo_uri: String,
    #[serde(rename = "scopes")]
    pub scopes: []ScopeInfo,
    #[serde(rename = "userCode")]
    pub user_code: String,
    #[serde(rename = "userCodeFormatted")]
    pub user_code_formatted: String,
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
pub struct TrustedContactsConfig {
    #[serde(rename = "maxNotificationsPerDay")]
    pub max_notifications_per_day: i32,
    #[serde(rename = "verificationExpiry")]
    pub verification_expiry: time.Duration,
    #[serde(rename = "allowPhoneContacts")]
    pub allow_phone_contacts: bool,
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
    #[serde(rename = "cooldownPeriod")]
    pub cooldown_period: time.Duration,
    #[serde(rename = "enabled")]
    pub enabled: bool,
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
pub struct ConsentPolicy {
    #[serde(rename = "consentType")]
    pub consent_type: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "metadata")]
    pub metadata: JSONBMap,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "required")]
    pub required: bool,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "createdBy")]
    pub created_by: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "renewable")]
    pub renewable: bool,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "publishedAt")]
    pub published_at: *time.Time,
    #[serde(rename = "validityPeriod")]
    pub validity_period: *int,
    #[serde(rename = "active")]
    pub active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpPolicyResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateSettingsResult {
    #[serde(rename = "success")]
    pub success: bool,
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
pub struct RecoverySession {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<RecoveryStatus>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentReportResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderListResponse {
    #[serde(rename = "providers")]
    pub providers: []ProviderInfo,
    #[serde(rename = "total")]
    pub total: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderSessionRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DailyAnalyticsDTO {
    #[serde(rename = "totalDelivered")]
    pub total_delivered: i64,
    #[serde(rename = "totalOpened")]
    pub total_opened: i64,
    #[serde(rename = "totalSent")]
    pub total_sent: i64,
    #[serde(rename = "date")]
    pub date: String,
    #[serde(rename = "deliveryRate")]
    pub delivery_rate: f64,
    #[serde(rename = "openRate")]
    pub open_rate: f64,
    #[serde(rename = "totalClicked")]
    pub total_clicked: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScopeDefinition {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckDependencies {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*ScopeResolver>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompliancePolicy {
    #[serde(rename = "title")]
    pub title: String,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "approvedBy")]
    pub approved_by: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "reviewDate")]
    pub review_date: time.Time,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "effectiveDate")]
    pub effective_date: time.Time,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "approvedAt")]
    pub approved_at: *time.Time,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "profileId")]
    pub profile_id: String,
    #[serde(rename = "policyType")]
    pub policy_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentAuditLogsResponse {
    #[serde(rename = "audit_logs")]
    pub audit_logs: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateSecretRequest {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "tags")]
    pub tags: []string,
    #[serde(rename = "value")]
    pub value: ,
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "description")]
    pub description: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MembersResponse {
    #[serde(rename = "members")]
    pub members: []*organization.Member,
    #[serde(rename = "total")]
    pub total: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestProviderInput {
    #[serde(rename = "providerType")]
    pub provider_type: String,
    #[serde(rename = "recipient")]
    pub recipient: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteAPIKeyRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendCodeRequest {
    #[serde(rename = "phone")]
    pub phone: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListAuditEventsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct mockUserService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
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
pub struct ScheduleVideoSessionRequest {
    #[serde(rename = "scheduledAt")]
    pub scheduled_at: time.Time,
    #[serde(rename = "sessionId")]
    pub session_id: xid.ID,
    #[serde(rename = "timeZone")]
    pub time_zone: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MFABypassResponse {
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrganizationStatsDTO {
    #[serde(rename = "totalOrganizations")]
    pub total_organizations: i64,
    #[serde(rename = "totalTeams")]
    pub total_teams: i64,
    #[serde(rename = "totalMembers")]
    pub total_members: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Email {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdatePasskeyRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteFieldRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryEntriesRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendRequest {
    #[serde(rename = "email")]
    pub email: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OAuthState {
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "redirect_url")]
    pub redirect_url: String,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "extra_scopes")]
    pub extra_scopes: []string,
    #[serde(rename = "link_user_id")]
    pub link_user_id: *xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CallbackResponse {
    #[serde(rename = "session")]
    pub session: *session.Session,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: *user.User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataProcessingAgreement {
    #[serde(rename = "agreementType")]
    pub agreement_type: String,
    #[serde(rename = "effectiveDate")]
    pub effective_date: time.Time,
    #[serde(rename = "expiryDate")]
    pub expiry_date: *time.Time,
    #[serde(rename = "signedBy")]
    pub signed_by: String,
    #[serde(rename = "signedByEmail")]
    pub signed_by_email: String,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "version")]
    pub version: String,
    #[serde(rename = "digitalSignature")]
    pub digital_signature: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "content")]
    pub content: String,
    #[serde(rename = "signedByTitle")]
    pub signed_by_title: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "metadata")]
    pub metadata: JSONBMap,
    #[serde(rename = "signedByName")]
    pub signed_by_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListJWTKeysRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetFactorRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RollbackRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceViolationsResponse {
    #[serde(rename = "violations")]
    pub violations: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFARequiredResponse {
    #[serde(rename = "device_id")]
    pub device_id: String,
    #[serde(rename = "require_twofa")]
    pub require_twofa: bool,
    #[serde(rename = "user")]
    pub user: *user.User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteRoleTemplateInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AsyncAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*notification.RetryService>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateActionRequest {
    #[serde(rename = "namespaceId")]
    pub namespace_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateMemberRoleRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "role")]
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FinishRegisterResponse {
    #[serde(rename = "credentialId")]
    pub credential_id: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "passkeyId")]
    pub passkey_id: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BulkRequest {
    #[serde(rename = "ids")]
    pub ids: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateFieldRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRoleTemplatesResult {
    #[serde(rename = "templates")]
    pub templates: []RoleTemplateDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SSOAuthResponse {
    #[serde(rename = "session")]
    pub session: *session.Session,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: *user.User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeclareABTestWinner_req {
    #[serde(rename = "abTestGroup")]
    pub ab_test_group: String,
    #[serde(rename = "winnerId")]
    pub winner_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenderTemplateRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhoneVerifyResponse {
    #[serde(rename = "session")]
    pub session: *session.Session,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "user")]
    pub user: *user.User,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListImpersonationsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompareRevisionsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRoleTemplatesInput {
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetMigrationStatusRequest {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdatePolicy_req {
    #[serde(rename = "title")]
    pub title: *string,
    #[serde(rename = "version")]
    pub version: *string,
    #[serde(rename = "content")]
    pub content: *string,
    #[serde(rename = "status")]
    pub status: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTeamRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BeginRegisterResponse {
    #[serde(rename = "challenge")]
    pub challenge: String,
    #[serde(rename = "options")]
    pub options: ,
    #[serde(rename = "timeout")]
    pub timeout: time.Duration,
    #[serde(rename = "userId")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthStatsResponse {
    #[serde(rename = "stats")]
    pub stats: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoveTrustedContactRequest {
    #[serde(rename = "contactId")]
    pub contact_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTrustedContactsResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "contacts")]
    pub contacts: []TrustedContactInfo,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BaseFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTemplateRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRolesRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnpublishEntryRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateEntryRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessmentConfig {
    #[serde(rename = "newIpWeight")]
    pub new_ip_weight: f64,
    #[serde(rename = "requireReviewAbove")]
    pub require_review_above: f64,
    #[serde(rename = "velocityWeight")]
    pub velocity_weight: f64,
    #[serde(rename = "blockHighRisk")]
    pub block_high_risk: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "highRiskThreshold")]
    pub high_risk_threshold: f64,
    #[serde(rename = "historyWeight")]
    pub history_weight: f64,
    #[serde(rename = "lowRiskThreshold")]
    pub low_risk_threshold: f64,
    #[serde(rename = "newDeviceWeight")]
    pub new_device_weight: f64,
    #[serde(rename = "newLocationWeight")]
    pub new_location_weight: f64,
    #[serde(rename = "mediumRiskThreshold")]
    pub medium_risk_threshold: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateRoleTemplateInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "permissions")]
    pub permissions: []string,
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceResponse {
    #[serde(rename = "attributes")]
    pub attributes: []core.ResourceAttribute,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "namespaceId")]
    pub namespace_id: String,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateEvidenceRequest {
    #[serde(rename = "fileUrl")]
    pub file_url: String,
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
    #[serde(rename = "title")]
    pub title: String,
    #[serde(rename = "controlId")]
    pub control_id: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "evidenceType")]
    pub evidence_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListAppsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListPasskeysResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "passkeys")]
    pub passkeys: []PasskeyInfo,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFAStatusResponse {
    #[serde(rename = "method")]
    pub method: String,
    #[serde(rename = "trusted")]
    pub trusted: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpAuditLog {
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "org_id")]
    pub org_id: String,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "user_id")]
    pub user_id: String,
    #[serde(rename = "event_data")]
    pub event_data: ,
    #[serde(rename = "event_type")]
    pub event_type: String,
    #[serde(rename = "ip")]
    pub ip: String,
    #[serde(rename = "severity")]
    pub severity: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateOrganizationResult {
    #[serde(rename = "organization")]
    pub organization: OrganizationDetailDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationResponse {
    #[serde(rename = "verification")]
    pub verification: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWKSResponse {
    #[serde(rename = "keys")]
    pub keys: []JWK,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationRepository {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*bun.DB>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteTeamRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationTemplateListResponse {
    #[serde(rename = "templates")]
    pub templates: Vec<>,
    #[serde(rename = "total")]
    pub total: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTemplateVersionRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateEntryRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DisableRequest {
    #[serde(rename = "user_id")]
    pub user_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTeamHandlerRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionDataDTO {
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "label")]
    pub label: String,
    #[serde(rename = "order")]
    pub order: i32,
    #[serde(rename = "requireAdmin")]
    pub require_admin: bool,
    #[serde(rename = "style")]
    pub style: String,
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "icon")]
    pub icon: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSettingsResult {
    #[serde(rename = "settings")]
    pub settings: OrganizationSettingsDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeTokenService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*repo.OAuthTokenRepository>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentNotificationsConfig {
    #[serde(rename = "notifyDeletionApproved")]
    pub notify_deletion_approved: bool,
    #[serde(rename = "notifyExportReady")]
    pub notify_export_ready: bool,
    #[serde(rename = "notifyOnGrant")]
    pub notify_on_grant: bool,
    #[serde(rename = "channels")]
    pub channels: []string,
    #[serde(rename = "notifyDeletionComplete")]
    pub notify_deletion_complete: bool,
    #[serde(rename = "notifyDpoEmail")]
    pub notify_dpo_email: String,
    #[serde(rename = "notifyOnExpiry")]
    pub notify_on_expiry: bool,
    #[serde(rename = "notifyOnRevoke")]
    pub notify_on_revoke: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestSendTemplateResult {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateProfileFromTemplateRequest {
    #[serde(rename = "standard")]
    pub standard: ComplianceStandard,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSecurityQuestionsResponse {
    #[serde(rename = "questions")]
    pub questions: []SecurityQuestionInfo,
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
pub struct CreateTeamResult {
    #[serde(rename = "team")]
    pub team: TeamDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OIDCLoginResponse {
    #[serde(rename = "authUrl")]
    pub auth_url: String,
    #[serde(rename = "nonce")]
    pub nonce: String,
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "state")]
    pub state: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetNotificationDetailInput {
    #[serde(rename = "notificationId")]
    pub notification_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemplatePerformanceDTO {
    #[serde(rename = "totalSent")]
    pub total_sent: i64,
    #[serde(rename = "clickRate")]
    pub click_rate: f64,
    #[serde(rename = "openRate")]
    pub open_rate: f64,
    #[serde(rename = "templateId")]
    pub template_id: String,
    #[serde(rename = "templateName")]
    pub template_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckRegistry {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenRevocationRequest {
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "client_secret")]
    pub client_secret: String,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "token_type_hint")]
    pub token_type_hint: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWK {
    #[serde(rename = "kid")]
    pub kid: String,
    #[serde(rename = "kty")]
    pub kty: String,
    #[serde(rename = "n")]
    pub n: String,
    #[serde(rename = "use")]
    pub use: String,
    #[serde(rename = "alg")]
    pub alg: String,
    #[serde(rename = "e")]
    pub e: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWKSService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*KeyStore>,
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteSecretRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnfidoProvider {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<OnfidoConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnassignRoleRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateRecoveryCodesResponse {
    #[serde(rename = "codes")]
    pub codes: []string,
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "generatedAt")]
    pub generated_at: time.Time,
    #[serde(rename = "warning")]
    pub warning: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateOrganizationResult {
    #[serde(rename = "organization")]
    pub organization: OrganizationDetailDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionDetailDTO {
    #[serde(rename = "os")]
    pub os: String,
    #[serde(rename = "createdAtFormatted")]
    pub created_at_formatted: String,
    #[serde(rename = "deviceType")]
    pub device_type: String,
    #[serde(rename = "isActive")]
    pub is_active: bool,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "userEmail")]
    pub user_email: String,
    #[serde(rename = "browser")]
    pub browser: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "updatedAtFormatted")]
    pub updated_at_formatted: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "environmentId")]
    pub environment_id: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "isExpiring")]
    pub is_expiring: bool,
    #[serde(rename = "lastRefreshedAt")]
    pub last_refreshed_at: *time.Time,
    #[serde(rename = "osVersion")]
    pub os_version: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "browserVersion")]
    pub browser_version: String,
    #[serde(rename = "deviceInfo")]
    pub device_info: String,
    #[serde(rename = "expiresAtFormatted")]
    pub expires_at_formatted: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "lastRefreshedFormatted")]
    pub last_refreshed_formatted: String,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceReportFileResponse {
    #[serde(rename = "content_type")]
    pub content_type: String,
    #[serde(rename = "data")]
    pub data: []byte,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateRecoveryCodesRequest {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "format")]
    pub format: String,
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
pub struct OrganizationAutoSendDTO {
    #[serde(rename = "invite")]
    pub invite: bool,
    #[serde(rename = "memberAdded")]
    pub member_added: bool,
    #[serde(rename = "memberLeft")]
    pub member_left: bool,
    #[serde(rename = "memberRemoved")]
    pub member_removed: bool,
    #[serde(rename = "roleChanged")]
    pub role_changed: bool,
    #[serde(rename = "transfer")]
    pub transfer: bool,
    #[serde(rename = "deleted")]
    pub deleted: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationWebhookResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EndImpersonationRequest {
    #[serde(rename = "impersonation_id")]
    pub impersonation_id: String,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddTrustedContactResponse {
    #[serde(rename = "message")]
    pub message: String,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupAuthStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RedisStateStore {
    #[serde(rename = "client", skip_serializing_if = "Option::is_none")]
    pub client: Option<*redis.Client>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentAuditLog {
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "newValue")]
    pub new_value: JSONBMap,
    #[serde(rename = "purpose")]
    pub purpose: String,
    #[serde(rename = "reason")]
    pub reason: String,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "consentId")]
    pub consent_id: String,
    #[serde(rename = "consentType")]
    pub consent_type: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "previousValue")]
    pub previous_value: JSONBMap,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpDevicesResponse {
    #[serde(rename = "devices")]
    pub devices: ,
    #[serde(rename = "count")]
    pub count: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Challenge {
    #[serde(rename = "attempts")]
    pub attempts: i32,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "factorId")]
    pub factor_id: xid.ID,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "type")]
    pub type: FactorType,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "verifiedAt")]
    pub verified_at: *time.Time,
    #[serde(rename = "-")]
    pub -: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "maxAttempts")]
    pub max_attempts: i32,
    #[serde(rename = "status")]
    pub status: ChallengeStatus,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookResponse {
    #[serde(rename = "received")]
    pub received: bool,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationStatusResponse {
    #[serde(rename = "status")]
    pub status: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateSecretOutput {
    #[serde(rename = "secret")]
    pub secret: SecretItem,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSessionHTTPRequest {
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
pub struct GetSessionStatsResult {
    #[serde(rename = "stats")]
    pub stats: SessionStatsDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LinkResponse {
    #[serde(rename = "user")]
    pub user: ,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListChecksFilter {
    #[serde(rename = "appId")]
    pub app_id: *string,
    #[serde(rename = "checkType")]
    pub check_type: *string,
    #[serde(rename = "profileId")]
    pub profile_id: *string,
    #[serde(rename = "sinceBefore")]
    pub since_before: *time.Time,
    #[serde(rename = "status")]
    pub status: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationSession {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteContentTypeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfoResponse {
    #[serde(rename = "nickname")]
    pub nickname: String,
    #[serde(rename = "preferred_username")]
    pub preferred_username: String,
    #[serde(rename = "profile")]
    pub profile: String,
    #[serde(rename = "website")]
    pub website: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "locale")]
    pub locale: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "updated_at")]
    pub updated_at: i64,
    #[serde(rename = "birthdate")]
    pub birthdate: String,
    #[serde(rename = "email_verified")]
    pub email_verified: bool,
    #[serde(rename = "phone_number")]
    pub phone_number: String,
    #[serde(rename = "picture")]
    pub picture: String,
    #[serde(rename = "sub")]
    pub sub: String,
    #[serde(rename = "zoneinfo")]
    pub zoneinfo: String,
    #[serde(rename = "gender")]
    pub gender: String,
    #[serde(rename = "middle_name")]
    pub middle_name: String,
    #[serde(rename = "phone_number_verified")]
    pub phone_number_verified: bool,
    #[serde(rename = "family_name")]
    pub family_name: String,
    #[serde(rename = "given_name")]
    pub given_name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetOrganizationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateVerificationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*xid.ID>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IDVerificationListResponse {
    #[serde(rename = "verifications")]
    pub verifications: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RequestReverification_req {
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationHistoryDTO {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "body")]
    pub body: String,
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "recipient")]
    pub recipient: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "type")]
    pub type: String,
    #[serde(rename = "error")]
    pub error: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "subject")]
    pub subject: String,
    #[serde(rename = "createdAt")]
    pub created_at: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "sentAt")]
    pub sent_at: *string,
    #[serde(rename = "templateId")]
    pub template_id: *string,
    #[serde(rename = "deliveredAt")]
    pub delivered_at: *string,
    #[serde(rename = "updatedAt")]
    pub updated_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockEmailService {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JWKS {
    #[serde(rename = "keys")]
    pub keys: []JWK,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VideoVerificationConfig {
    #[serde(rename = "minScheduleAdvance")]
    pub min_schedule_advance: time.Duration,
    #[serde(rename = "recordingRetention")]
    pub recording_retention: time.Duration,
    #[serde(rename = "requireAdminReview")]
    pub require_admin_review: bool,
    #[serde(rename = "requireLivenessCheck")]
    pub require_liveness_check: bool,
    #[serde(rename = "sessionDuration")]
    pub session_duration: time.Duration,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "livenessThreshold")]
    pub liveness_threshold: f64,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "recordSessions")]
    pub record_sessions: bool,
    #[serde(rename = "requireScheduling")]
    pub require_scheduling: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoverySessionInfo {
    #[serde(rename = "totalSteps")]
    pub total_steps: i32,
    #[serde(rename = "userEmail")]
    pub user_email: String,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "currentStep")]
    pub current_step: i32,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "method")]
    pub method: RecoveryMethod,
    #[serde(rename = "requiresReview")]
    pub requires_review: bool,
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "riskScore")]
    pub risk_score: f64,
    #[serde(rename = "status")]
    pub status: RecoveryStatus,
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
pub struct LinkAccountRequest {
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "scopes")]
    pub scopes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackupCodesConfig {
    #[serde(rename = "format")]
    pub format: String,
    #[serde(rename = "length")]
    pub length: i32,
    #[serde(rename = "allow_reuse")]
    pub allow_reuse: bool,
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct bunRepository {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*bun.DB>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestSendTemplateInput {
    #[serde(rename = "recipient")]
    pub recipient: String,
    #[serde(rename = "templateId")]
    pub template_id: String,
    #[serde(rename = "variables")]
    pub variables: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFAErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpRememberedDevice {
    #[serde(rename = "created_at")]
    pub created_at: time.Time,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "security_level")]
    pub security_level: SecurityLevel,
    #[serde(rename = "user_agent")]
    pub user_agent: String,
    #[serde(rename = "device_id")]
    pub device_id: String,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateJWTKeyRequest {
    #[serde(rename = "algorithm")]
    pub algorithm: String,
    #[serde(rename = "curve")]
    pub curve: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: *time.Time,
    #[serde(rename = "isPlatformKey")]
    pub is_platform_key: bool,
    #[serde(rename = "keyType")]
    pub key_type: String,
    #[serde(rename = "metadata")]
    pub metadata: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LimitResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTeamsInput {
    #[serde(rename = "search")]
    pub search: String,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "limit")]
    pub limit: i32,
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "page")]
    pub page: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteTeamResult {
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationStatusResponse {
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DefaultProviderRegistry {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<EmailProvider>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetChallengeStatusRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskEngine {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*repository.MFARepository>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTemplatesInput {
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "type")]
    pub type: *string,
    #[serde(rename = "active")]
    pub active: *bool,
    #[serde(rename = "language")]
    pub language: *string,
    #[serde(rename = "limit")]
    pub limit: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTemplateInput {
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSessionsResult {
    #[serde(rename = "stats")]
    pub stats: SessionStatsDTO,
    #[serde(rename = "pagination")]
    pub pagination: PaginationInfoDTO,
    #[serde(rename = "sessions")]
    pub sessions: []SessionDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetEffectivePermissionsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CloneContentTypeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionResponse {
    #[serde(rename = "connection")]
    pub connection: *schema.SocialAccount,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CancelInvitationInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "inviteId")]
    pub invite_id: String,
    #[serde(rename = "orgId")]
    pub org_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthAutoSendConfig {
    #[serde(rename = "email_otp")]
    pub email_otp: bool,
    #[serde(rename = "magic_link")]
    pub magic_link: bool,
    #[serde(rename = "mfa_code")]
    pub mfa_code: bool,
    #[serde(rename = "password_reset")]
    pub password_reset: bool,
    #[serde(rename = "verification_email")]
    pub verification_email: bool,
    #[serde(rename = "welcome")]
    pub welcome: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PreviewTemplateResult {
    #[serde(rename = "body")]
    pub body: String,
    #[serde(rename = "renderedAt")]
    pub rendered_at: String,
    #[serde(rename = "subject")]
    pub subject: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionResponse {
    #[serde(rename = "namespaceId")]
    pub namespace_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnalyticsSummary {
    #[serde(rename = "activePolicies")]
    pub active_policies: i32,
    #[serde(rename = "allowedCount")]
    pub allowed_count: i64,
    #[serde(rename = "cacheHitRate")]
    pub cache_hit_rate: f64,
    #[serde(rename = "deniedCount")]
    pub denied_count: i64,
    #[serde(rename = "topPolicies")]
    pub top_policies: []PolicyStats,
    #[serde(rename = "topResourceTypes")]
    pub top_resource_types: []ResourceTypeStats,
    #[serde(rename = "totalEvaluations")]
    pub total_evaluations: i64,
    #[serde(rename = "avgLatencyMs")]
    pub avg_latency_ms: f64,
    #[serde(rename = "totalPolicies")]
    pub total_policies: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRevisionRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SMSVerificationConfig {
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
    #[serde(rename = "codeLength")]
    pub code_length: i32,
    #[serde(rename = "cooldownPeriod")]
    pub cooldown_period: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockUserRepository {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateTokenRequest {
    #[serde(rename = "tokenType")]
    pub token_type: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "audience")]
    pub audience: []string,
    #[serde(rename = "expiresIn")]
    pub expires_in: time.Duration,
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "permissions")]
    pub permissions: []string,
    #[serde(rename = "scopes")]
    pub scopes: []string,
    #[serde(rename = "sessionId")]
    pub session_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSessionResult {
    #[serde(rename = "session")]
    pub session: SessionDetailDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationErrorResponse {
    #[serde(rename = "policyIndex")]
    pub policy_index: i32,
    #[serde(rename = "resource")]
    pub resource: String,
    #[serde(rename = "subject")]
    pub subject: String,
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthorizeRequest {
    #[serde(rename = "code_challenge")]
    pub code_challenge: String,
    #[serde(rename = "max_age")]
    pub max_age: *int,
    #[serde(rename = "response_type")]
    pub response_type: String,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "state")]
    pub state: String,
    #[serde(rename = "ui_locales")]
    pub ui_locales: String,
    #[serde(rename = "acr_values")]
    pub acr_values: String,
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "code_challenge_method")]
    pub code_challenge_method: String,
    #[serde(rename = "id_token_hint")]
    pub id_token_hint: String,
    #[serde(rename = "login_hint")]
    pub login_hint: String,
    #[serde(rename = "nonce")]
    pub nonce: String,
    #[serde(rename = "prompt")]
    pub prompt: String,
    #[serde(rename = "redirect_uri")]
    pub redirect_uri: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentDeletionResponse {
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTemplateVersion_req {
    #[serde(rename = "changes")]
    pub changes: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*audit.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetAuditLogsRequestDTO {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetOrganizationResult {
    #[serde(rename = "organization")]
    pub organization: OrganizationDetailDTO,
    #[serde(rename = "stats")]
    pub stats: OrgDetailStatsDTO,
    #[serde(rename = "userRole")]
    pub user_role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct settingField {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<user.ServiceInterface>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiscoveryService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<Config>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderSession {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetAnalyticsInput {
    #[serde(rename = "days")]
    pub days: *int,
    #[serde(rename = "endDate")]
    pub end_date: *string,
    #[serde(rename = "startDate")]
    pub start_date: *string,
    #[serde(rename = "templateId")]
    pub template_id: *string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceCheckResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListRevisionsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientRegistrationRequest {
    #[serde(rename = "response_types")]
    pub response_types: []string,
    #[serde(rename = "tos_uri")]
    pub tos_uri: String,
    #[serde(rename = "client_name")]
    pub client_name: String,
    #[serde(rename = "require_consent")]
    pub require_consent: bool,
    #[serde(rename = "require_pkce")]
    pub require_pkce: bool,
    #[serde(rename = "trusted_client")]
    pub trusted_client: bool,
    #[serde(rename = "contacts")]
    pub contacts: []string,
    #[serde(rename = "grant_types")]
    pub grant_types: []string,
    #[serde(rename = "policy_uri")]
    pub policy_uri: String,
    #[serde(rename = "post_logout_redirect_uris")]
    pub post_logout_redirect_uris: []string,
    #[serde(rename = "scope")]
    pub scope: String,
    #[serde(rename = "application_type")]
    pub application_type: String,
    #[serde(rename = "logo_uri")]
    pub logo_uri: String,
    #[serde(rename = "redirect_uris")]
    pub redirect_uris: []string,
    #[serde(rename = "token_endpoint_auth_method")]
    pub token_endpoint_auth_method: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentVerificationConfig {
    #[serde(rename = "minConfidenceScore")]
    pub min_confidence_score: f64,
    #[serde(rename = "requireBothSides")]
    pub require_both_sides: bool,
    #[serde(rename = "requireSelfie")]
    pub require_selfie: bool,
    #[serde(rename = "storagePath")]
    pub storage_path: String,
    #[serde(rename = "acceptedDocuments")]
    pub accepted_documents: []string,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "requireManualReview")]
    pub require_manual_review: bool,
    #[serde(rename = "retentionPeriod")]
    pub retention_period: time.Duration,
    #[serde(rename = "storageProvider")]
    pub storage_provider: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "encryptAtRest")]
    pub encrypt_at_rest: bool,
    #[serde(rename = "encryptionKey")]
    pub encryption_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CookieConsent {
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "personalization")]
    pub personalization: bool,
    #[serde(rename = "sessionId")]
    pub session_id: String,
    #[serde(rename = "thirdParty")]
    pub third_party: bool,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "essential")]
    pub essential: bool,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "marketing")]
    pub marketing: bool,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "consentBannerVersion")]
    pub consent_banner_version: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "functional")]
    pub functional: bool,
    #[serde(rename = "analytics")]
    pub analytics: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrganizationAutoSendConfig {
    #[serde(rename = "deleted")]
    pub deleted: bool,
    #[serde(rename = "invite")]
    pub invite: bool,
    #[serde(rename = "member_added")]
    pub member_added: bool,
    #[serde(rename = "member_left")]
    pub member_left: bool,
    #[serde(rename = "member_removed")]
    pub member_removed: bool,
    #[serde(rename = "role_changed")]
    pub role_changed: bool,
    #[serde(rename = "transfer")]
    pub transfer: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTemplatesResponse {
    #[serde(rename = "templates")]
    pub templates: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PublishEntryRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateFactorRequest {
    #[serde(rename = "name")]
    pub name: *string,
    #[serde(rename = "priority")]
    pub priority: *FactorPriority,
    #[serde(rename = "status")]
    pub status: *FactorStatus,
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "metadata")]
    pub metadata: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionAutoSendDTO {
    #[serde(rename = "suspiciousLogin")]
    pub suspicious_login: bool,
    #[serde(rename = "allRevoked")]
    pub all_revoked: bool,
    #[serde(rename = "deviceRemoved")]
    pub device_removed: bool,
    #[serde(rename = "newDevice")]
    pub new_device: bool,
    #[serde(rename = "newLocation")]
    pub new_location: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatePolicyResponse {
    #[serde(rename = "warnings")]
    pub warnings: []string,
    #[serde(rename = "complexity")]
    pub complexity: i32,
    #[serde(rename = "error")]
    pub error: String,
    #[serde(rename = "errors")]
    pub errors: []string,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "valid")]
    pub valid: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceChecksResponse {
    #[serde(rename = "checks")]
    pub checks: Vec<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskContext {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfigSourceConfig {
    #[serde(rename = "priority")]
    pub priority: i32,
    #[serde(rename = "refreshInterval")]
    pub refresh_interval: time.Duration,
    #[serde(rename = "autoRefresh")]
    pub auto_refresh: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "prefix")]
    pub prefix: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginationInfo {
    #[serde(rename = "totalItems")]
    pub total_items: i64,
    #[serde(rename = "totalPages")]
    pub total_pages: i32,
    #[serde(rename = "currentPage")]
    pub current_page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PreviewTemplate_req {
    #[serde(rename = "variables")]
    pub variables: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct App {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTeamRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyImpersonationRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientUpdateRequest {
    #[serde(rename = "logo_uri")]
    pub logo_uri: String,
    #[serde(rename = "redirect_uris")]
    pub redirect_uris: []string,
    #[serde(rename = "response_types")]
    pub response_types: []string,
    #[serde(rename = "tos_uri")]
    pub tos_uri: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "policy_uri")]
    pub policy_uri: String,
    #[serde(rename = "post_logout_redirect_uris")]
    pub post_logout_redirect_uris: []string,
    #[serde(rename = "require_consent")]
    pub require_consent: *bool,
    #[serde(rename = "require_pkce")]
    pub require_pkce: *bool,
    #[serde(rename = "token_endpoint_auth_method")]
    pub token_endpoint_auth_method: String,
    #[serde(rename = "trusted_client")]
    pub trusted_client: *bool,
    #[serde(rename = "allowed_scopes")]
    pub allowed_scopes: []string,
    #[serde(rename = "contacts")]
    pub contacts: []string,
    #[serde(rename = "grant_types")]
    pub grant_types: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendResponse {
    #[serde(rename = "dev_otp")]
    pub dev_otp: String,
    #[serde(rename = "status")]
    pub status: String,
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
pub struct DeleteUserRequestDTO {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTemplatesResult {
    #[serde(rename = "pagination")]
    pub pagination: PaginationDTO,
    #[serde(rename = "templates")]
    pub templates: []TemplateDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteTemplateInput {
    #[serde(rename = "templateId")]
    pub template_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StepUpVerificationResponse {
    #[serde(rename = "expires_at")]
    pub expires_at: String,
    #[serde(rename = "verified")]
    pub verified: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmailProviderConfig {
    #[serde(rename = "config")]
    pub config: ,
    #[serde(rename = "from")]
    pub from: String,
    #[serde(rename = "from_name")]
    pub from_name: String,
    #[serde(rename = "provider")]
    pub provider: String,
    #[serde(rename = "reply_to")]
    pub reply_to: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetProvidersResult {
    #[serde(rename = "providers")]
    pub providers: ProvidersConfigDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTemplateResult {
    #[serde(rename = "template")]
    pub template: TemplateDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeAllRequest {
    #[serde(rename = "includeCurrentSession")]
    pub include_current_session: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceTrainingResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceDashboardResponse {
    #[serde(rename = "metrics")]
    pub metrics: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountAutoSendConfig {
    #[serde(rename = "reactivated")]
    pub reactivated: bool,
    #[serde(rename = "suspended")]
    pub suspended: bool,
    #[serde(rename = "username_changed")]
    pub username_changed: bool,
    #[serde(rename = "deleted")]
    pub deleted: bool,
    #[serde(rename = "email_change_request")]
    pub email_change_request: bool,
    #[serde(rename = "email_changed")]
    pub email_changed: bool,
    #[serde(rename = "password_changed")]
    pub password_changed: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateResourceRequest {
    #[serde(rename = "attributes")]
    pub attributes: []ResourceAttributeRequest,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "namespaceId")]
    pub namespace_id: String,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PreviewConversionRequest {
    #[serde(rename = "actions")]
    pub actions: []string,
    #[serde(rename = "condition")]
    pub condition: String,
    #[serde(rename = "resource")]
    pub resource: String,
    #[serde(rename = "subject")]
    pub subject: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutoCleanupConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "interval")]
    pub interval: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BeginLoginRequest {
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "userVerification")]
    pub user_verification: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationEndResponse {
    #[serde(rename = "ended_at")]
    pub ended_at: String,
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
pub struct DevicesResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "devices")]
    pub devices: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdminBypassRequest {
    #[serde(rename = "userId")]
    pub user_id: xid.ID,
    #[serde(rename = "duration")]
    pub duration: i32,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderDiscoveredResponse {
    #[serde(rename = "found")]
    pub found: bool,
    #[serde(rename = "providerId")]
    pub provider_id: String,
    #[serde(rename = "type")]
    pub type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetProvidersInput {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnableResponse {
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "totp_uri")]
    pub totp_uri: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetUserRoleRequest {
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "-")]
    pub -: xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "user_id")]
    pub user_id: xid.ID,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenderTemplate_req {
    #[serde(rename = "template")]
    pub template: String,
    #[serde(rename = "variables")]
    pub variables: ,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoveMemberInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "memberId")]
    pub member_id: String,
    #[serde(rename = "orgId")]
    pub org_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetMembersResult {
    #[serde(rename = "canManage")]
    pub can_manage: bool,
    #[serde(rename = "data")]
    pub data: []MemberDTO,
    #[serde(rename = "pagination")]
    pub pagination: PaginationInfo,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InviteMemberInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "role")]
    pub role: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationSettingsDTO {
    #[serde(rename = "account")]
    pub account: AccountAutoSendDTO,
    #[serde(rename = "appName")]
    pub app_name: String,
    #[serde(rename = "auth")]
    pub auth: AuthAutoSendDTO,
    #[serde(rename = "organization")]
    pub organization: OrganizationAutoSendDTO,
    #[serde(rename = "session")]
    pub session: SessionAutoSendDTO,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyEngine {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTrainingFilter {
    #[serde(rename = "appId")]
    pub app_id: *string,
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
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceEvidenceResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyCodeResponse {
    #[serde(rename = "attemptsLeft")]
    pub attempts_left: i32,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "valid")]
    pub valid: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListRecoverySessionsRequest {
    #[serde(rename = "status")]
    pub status: RecoveryStatus,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "page")]
    pub page: i32,
    #[serde(rename = "pageSize")]
    pub page_size: i32,
    #[serde(rename = "requiresReview")]
    pub requires_review: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetOrganizationInput {
    #[serde(rename = "orgId")]
    pub org_id: String,
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderCheckResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceVerificationRequest {
    #[serde(rename = "user_code")]
    pub user_code: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFAEnableResponse {
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "totp_uri")]
    pub totp_uri: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebAuthnFactorAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*passkey.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountLockedResponse {
    #[serde(rename = "code")]
    pub code: String,
    #[serde(rename = "locked_minutes")]
    pub locked_minutes: i32,
    #[serde(rename = "locked_until")]
    pub locked_until: time.Time,
    #[serde(rename = "message")]
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListMembersRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JumioConfig {
    #[serde(rename = "dataCenter")]
    pub data_center: String,
    #[serde(rename = "enableAMLScreening")]
    pub enable_a_m_l_screening: bool,
    #[serde(rename = "enableExtraction")]
    pub enable_extraction: bool,
    #[serde(rename = "enabledDocumentTypes")]
    pub enabled_document_types: []string,
    #[serde(rename = "presetId")]
    pub preset_id: String,
    #[serde(rename = "apiSecret")]
    pub api_secret: String,
    #[serde(rename = "enableLiveness")]
    pub enable_liveness: bool,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "enabledCountries")]
    pub enabled_countries: []string,
    #[serde(rename = "verificationType")]
    pub verification_type: String,
    #[serde(rename = "apiToken")]
    pub api_token: String,
    #[serde(rename = "callbackUrl")]
    pub callback_url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetSessionStatsInput {
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceProfileResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryConfiguration {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<xid.ID>,
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
pub struct TOTPConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "issuer")]
    pub issuer: String,
    #[serde(rename = "period")]
    pub period: i32,
    #[serde(rename = "window_size")]
    pub window_size: i32,
    #[serde(rename = "algorithm")]
    pub algorithm: String,
    #[serde(rename = "digits")]
    pub digits: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevealSecretInput {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "secretId")]
    pub secret_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetVersionsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OverviewStatsDTO {
    #[serde(rename = "deliveryRate")]
    pub delivery_rate: f64,
    #[serde(rename = "totalOpened")]
    pub total_opened: i64,
    #[serde(rename = "openRate")]
    pub open_rate: f64,
    #[serde(rename = "totalBounced")]
    pub total_bounced: i64,
    #[serde(rename = "totalClicked")]
    pub total_clicked: i64,
    #[serde(rename = "totalDelivered")]
    pub total_delivered: i64,
    #[serde(rename = "totalFailed")]
    pub total_failed: i64,
    #[serde(rename = "totalSent")]
    pub total_sent: i64,
    #[serde(rename = "bounceRate")]
    pub bounce_rate: f64,
    #[serde(rename = "clickRate")]
    pub click_rate: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunCheck_req {
    #[serde(rename = "checkType")]
    pub check_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTeamRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFABackupCodesResponse {
    #[serde(rename = "codes")]
    pub codes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateUserRequest {
    #[serde(rename = "metadata")]
    pub metadata: ,
    #[serde(rename = "password")]
    pub password: String,
    #[serde(rename = "role")]
    pub role: String,
    #[serde(rename = "user_organization_id")]
    pub user_organization_id: *xid.ID,
    #[serde(rename = "username")]
    pub username: String,
    #[serde(rename = "-")]
    pub -: xid.ID,
    #[serde(rename = "app_id")]
    pub app_id: xid.ID,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "email")]
    pub email: String,
    #[serde(rename = "email_verified")]
    pub email_verified: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ForgetDeviceResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTrustedDevicesResponse {
    #[serde(rename = "count")]
    pub count: i32,
    #[serde(rename = "devices")]
    pub devices: []TrustedDevice,
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
pub struct BulkPublishRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
    #[serde(rename = "ids")]
    pub ids: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceAuthorizationRequest {
    #[serde(rename = "client_id")]
    pub client_id: String,
    #[serde(rename = "scope")]
    pub scope: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockStateStore {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyResponse {
    #[serde(rename = "userOrganizationId")]
    pub user_organization_id: *string,
    #[serde(rename = "version")]
    pub version: i32,
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "createdBy")]
    pub created_by: String,
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "expression")]
    pub expression: String,
    #[serde(rename = "namespaceId")]
    pub namespace_id: String,
    #[serde(rename = "priority")]
    pub priority: i32,
    #[serde(rename = "resourceType")]
    pub resource_type: String,
    #[serde(rename = "actions")]
    pub actions: []string,
    #[serde(rename = "environmentId")]
    pub environment_id: String,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComplianceViolationResponse {
    #[serde(rename = "id")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsentService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*repo.OAuthClientRepository>,
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
pub struct DataDeletionRequestInput {
    #[serde(rename = "deleteSections")]
    pub delete_sections: []string,
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListFactorsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTemplateResult {
    #[serde(rename = "template")]
    pub template: TemplateDTO,
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RedisChallengeStore {
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceAuthorizationResponse {
    #[serde(rename = "expires_in")]
    pub expires_in: i32,
    #[serde(rename = "interval")]
    pub interval: i32,
    #[serde(rename = "user_code")]
    pub user_code: String,
    #[serde(rename = "verification_uri")]
    pub verification_uri: String,
    #[serde(rename = "verification_uri_complete")]
    pub verification_uri_complete: String,
    #[serde(rename = "device_code")]
    pub device_code: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataDeletionRequest {
    #[serde(rename = "retentionExempt")]
    pub retention_exempt: bool,
    #[serde(rename = "updatedAt")]
    pub updated_at: time.Time,
    #[serde(rename = "approvedAt")]
    pub approved_at: *time.Time,
    #[serde(rename = "approvedBy")]
    pub approved_by: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "requestReason")]
    pub request_reason: String,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "archivePath")]
    pub archive_path: String,
    #[serde(rename = "deleteSections")]
    pub delete_sections: []string,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "exemptionReason")]
    pub exemption_reason: String,
    #[serde(rename = "id")]
    pub id: xid.ID,
    #[serde(rename = "completedAt")]
    pub completed_at: *time.Time,
    #[serde(rename = "errorMessage")]
    pub error_message: String,
    #[serde(rename = "organizationId")]
    pub organization_id: String,
    #[serde(rename = "rejectedAt")]
    pub rejected_at: *time.Time,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FactorVerificationRequest {
    #[serde(rename = "data")]
    pub data: ,
    #[serde(rename = "factorId")]
    pub factor_id: xid.ID,
    #[serde(rename = "code")]
    pub code: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckSubResult {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationPreviewResponse {
    #[serde(rename = "body")]
    pub body: String,
    #[serde(rename = "subject")]
    pub subject: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiSessionErrorResponse {
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResolveViolationRequest {
    #[serde(rename = "notes")]
    pub notes: String,
    #[serde(rename = "resolution")]
    pub resolution: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FinishRegisterRequest {
    #[serde(rename = "response")]
    pub response: ,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetABTestResultsRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountAutoSendDTO {
    #[serde(rename = "usernameChanged")]
    pub username_changed: bool,
    #[serde(rename = "deleted")]
    pub deleted: bool,
    #[serde(rename = "emailChangeRequest")]
    pub email_change_request: bool,
    #[serde(rename = "emailChanged")]
    pub email_changed: bool,
    #[serde(rename = "passwordChanged")]
    pub password_changed: bool,
    #[serde(rename = "reactivated")]
    pub reactivated: bool,
    #[serde(rename = "suspended")]
    pub suspended: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteAppRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationMiddleware {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<Config>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImpersonationStartResponse {
    #[serde(rename = "target_user_id")]
    pub target_user_id: String,
    #[serde(rename = "impersonator_id")]
    pub impersonator_id: String,
    #[serde(rename = "session_id")]
    pub session_id: String,
    #[serde(rename = "started_at")]
    pub started_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidateContentTypeRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceAuthorizationDecisionRequest {
    #[serde(rename = "action")]
    pub action: String,
    #[serde(rename = "user_code")]
    pub user_code: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyTrustedContactRequest {
    #[serde(rename = "token")]
    pub token: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SuccessResponse {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
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
pub struct GetOverviewStatsInput {
    #[serde(rename = "endDate")]
    pub end_date: *string,
    #[serde(rename = "startDate")]
    pub start_date: *string,
    #[serde(rename = "days")]
    pub days: *int,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrateRolesRequest {
    #[serde(rename = "dryRun")]
    pub dry_run: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CodesResponse {
    #[serde(rename = "codes")]
    pub codes: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimit {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<time.Duration>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct auditServiceAdapter {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*audit.Service>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdaptiveMFAConfig {
    #[serde(rename = "enabled")]
    pub enabled: bool,
    #[serde(rename = "factor_ip_reputation")]
    pub factor_ip_reputation: bool,
    #[serde(rename = "factor_location_change")]
    pub factor_location_change: bool,
    #[serde(rename = "new_device_risk")]
    pub new_device_risk: f64,
    #[serde(rename = "require_step_up_threshold")]
    pub require_step_up_threshold: f64,
    #[serde(rename = "velocity_risk")]
    pub velocity_risk: f64,
    #[serde(rename = "factor_new_device")]
    pub factor_new_device: bool,
    #[serde(rename = "factor_velocity")]
    pub factor_velocity: bool,
    #[serde(rename = "location_change_risk")]
    pub location_change_risk: f64,
    #[serde(rename = "risk_threshold")]
    pub risk_threshold: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptionService {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<sync.RWMutex>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RevokeResponse {
    #[serde(rename = "revokedCount")]
    pub revoked_count: i32,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckMetadata {
    #[serde(rename = "severity")]
    pub severity: String,
    #[serde(rename = "standards")]
    pub standards: []string,
    #[serde(rename = "autoRun")]
    pub auto_run: bool,
    #[serde(rename = "category")]
    pub category: String,
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendCodeResponse {
    #[serde(rename = "dev_code")]
    pub dev_code: String,
    #[serde(rename = "status")]
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScopeInfo {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
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
pub struct GetRecoveryConfigResponse {
    #[serde(rename = "enabledMethods")]
    pub enabled_methods: []RecoveryMethod,
    #[serde(rename = "minimumStepsRequired")]
    pub minimum_steps_required: i32,
    #[serde(rename = "requireAdminReview")]
    pub require_admin_review: bool,
    #[serde(rename = "requireMultipleSteps")]
    pub require_multiple_steps: bool,
    #[serde(rename = "riskScoreThreshold")]
    pub risk_score_threshold: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProvidersAppResponse {
    #[serde(rename = "appId")]
    pub app_id: String,
    #[serde(rename = "providers")]
    pub providers: []string,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReverifyRequest {
    #[serde(rename = "reason")]
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeclareABTestWinnerRequest {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestCaseResult {
    #[serde(rename = "evaluationTimeMs")]
    pub evaluation_time_ms: f64,
    #[serde(rename = "expected")]
    pub expected: bool,
    #[serde(rename = "name")]
    pub name: String,
    #[serde(rename = "passed")]
    pub passed: bool,
    #[serde(rename = "actual")]
    pub actual: bool,
    #[serde(rename = "error")]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateNamespaceRequest {
    #[serde(rename = "description")]
    pub description: String,
    #[serde(rename = "inheritPlatform")]
    pub inherit_platform: *bool,
    #[serde(rename = "name")]
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DashboardExtension {
    #[serde(rename = "", skip_serializing_if = "Option::is_none")]
    pub : Option<*pages.PagesManager>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetupSecurityQuestionsRequest {
    #[serde(rename = "questions")]
    pub questions: []SetupSecurityQuestionRequest,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyTokenRequest {
    #[serde(rename = "audience")]
    pub audience: []string,
    #[serde(rename = "token")]
    pub token: String,
    #[serde(rename = "tokenType")]
    pub token_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeAppInput {
    #[serde(rename = "appId")]
    pub app_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionDTO {
    #[serde(rename = "deviceInfo")]
    pub device_info: String,
    #[serde(rename = "ipAddress")]
    pub ip_address: String,
    #[serde(rename = "os")]
    pub os: String,
    #[serde(rename = "osVersion")]
    pub os_version: String,
    #[serde(rename = "expiresIn")]
    pub expires_in: String,
    #[serde(rename = "isActive")]
    pub is_active: bool,
    #[serde(rename = "isExpiring")]
    pub is_expiring: bool,
    #[serde(rename = "status")]
    pub status: String,
    #[serde(rename = "userAgent")]
    pub user_agent: String,
    #[serde(rename = "createdAt")]
    pub created_at: time.Time,
    #[serde(rename = "deviceType")]
    pub device_type: String,
    #[serde(rename = "expiresAt")]
    pub expires_at: time.Time,
    #[serde(rename = "userEmail")]
    pub user_email: String,
    #[serde(rename = "userId")]
    pub user_id: String,
    #[serde(rename = "browser")]
    pub browser: String,
    #[serde(rename = "id")]
    pub id: String,
    #[serde(rename = "lastUsed")]
    pub last_used: String,
    #[serde(rename = "browserVersion")]
    pub browser_version: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BeginLoginResponse {
    #[serde(rename = "challenge")]
    pub challenge: String,
    #[serde(rename = "options")]
    pub options: ,
    #[serde(rename = "timeout")]
    pub timeout: time.Duration,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteSecretOutput {
    #[serde(rename = "message")]
    pub message: String,
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoveMemberResult {
    #[serde(rename = "success")]
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListNotificationsHistoryResult {
    #[serde(rename = "notifications")]
    pub notifications: []NotificationHistoryDTO,
    #[serde(rename = "pagination")]
    pub pagination: PaginationDTO,
}

