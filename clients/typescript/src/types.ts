// Auto-generated TypeScript types

export interface MockService {
}

export interface TemplateParameter {
  [key: string]: any;
}

export interface ListVersionsResponse {
  [key: string]: any;
}

export interface PublishContentTypeRequest {
}

export interface TestSendTemplate_req {
  recipient: string;
  variables: any;
}

export interface AddTrustedContactResponse {
  email: string;
  message: string;
  name: string;
  phone: string;
  verified: boolean;
  addedAt: string;
  contactId: string;
}

export interface AuditLog {
}

export interface ComplianceStatusResponse {
  status: string;
}

export interface ReportsConfig {
  enabled: boolean;
  formats: string[];
  includeEvidence: boolean;
  retentionDays: number;
  schedule: string;
  storagePath: string;
}

export interface LinkRequest {
  email: string;
  name: string;
  password: string;
}

export interface CookieConsent {
  expiresAt: string;
  organizationId: string;
  createdAt: string;
  essential: boolean;
  functional: boolean;
  id: string;
  ipAddress: string;
  personalization: boolean;
  sessionId: string;
  thirdParty: boolean;
  analytics: boolean;
  marketing: boolean;
  updatedAt: string;
  userAgent: string;
  userId: string;
  consentBannerVersion: string;
}

export interface JumioConfig {
  apiSecret: string;
  callbackUrl: string;
  dataCenter: string;
  enableExtraction: boolean;
  enableLiveness: boolean;
  enabledCountries: string[];
  presetId: string;
  verificationType: string;
  apiToken: string;
  enableAMLScreening: boolean;
  enabled: boolean;
  enabledDocumentTypes: string[];
}

export interface VerificationSessionResponse {
  session: IdentityVerificationSession | undefined;
}

export interface GetProviderRequest {
}

export interface ComplianceStatusDetailsResponse {
  status: string;
}

export interface ClientAuthResult {
}

export interface TestPolicyResponse {
  total: number;
  error: string;
  failedCount: number;
  passed: boolean;
  passedCount: number;
  results: TestCaseResult[];
}

export interface AuditServiceAdapter {
}

export interface StepUpRequirementsResponse {
  requirements: any[];
}

export interface AddFieldRequest {
}

export interface SetupSecurityQuestionsResponse {
  count: number;
  message: string;
  setupAt: string;
}

export interface ListViolationsFilter {
  appId: string | undefined;
  profileId: string | undefined;
  severity: string | undefined;
  status: string | undefined;
  userId: string | undefined;
  violationType: string | undefined;
}

export interface MockAuditService {
  [key: string]: any;
}

export interface PreviewTemplate_req {
  variables: any;
}

export interface RateLimitingConfig {
  lockoutAfterAttempts: number;
  lockoutDuration: Duration;
  maxAttemptsPerDay: number;
  maxAttemptsPerHour: number;
  maxAttemptsPerIp: number;
  enabled: boolean;
  exponentialBackoff: boolean;
  ipCooldownPeriod: Duration;
}

export interface MockRepository {
}

export interface ClientRegistrationResponse {
  response_types: string[];
  client_id: string;
  client_secret_expires_at: number;
  policy_uri: string;
  post_logout_redirect_uris: string[];
  contacts: string[];
  grant_types: string[];
  token_endpoint_auth_method: string;
  redirect_uris: string[];
  scope: string;
  application_type: string;
  client_id_issued_at: number;
  client_secret: string;
  logo_uri: string;
  tos_uri: string;
  client_name: string;
}

export interface AccessConfig {
  requireRbac: boolean;
  allowApiAccess: boolean;
  allowDashboardAccess: boolean;
  rateLimitPerMinute: number;
  requireAuthentication: boolean;
}

export interface UnbanUserRequest {
  user_organization_id: string | undefined;
  app_id: string;
  reason: string;
  user_id: string;
}

export interface RateLimit {
  max_requests: number;
  window: Duration;
}

export interface ResetTemplateRequest {
}

export interface NoOpVideoProvider {
  [key: string]: any;
}

export interface GetAppRequest {
}

export interface LoginResponse {
  passkeyUsed: string;
  session: any;
  token: string;
  user: any;
}

export interface OrganizationProvider {
  [key: string]: any;
}

export interface MigrationHandler {
}

export interface StatusResponse {
  status: string;
}

export interface IDVerificationListResponse {
  verifications: any[];
}

export interface KeyStats {
}

export interface DocumentVerificationResult {
}

export interface DevicesResponse {
  count: number;
  devices: any;
}

export interface BackupCodesConfig {
  count: number;
  enabled: boolean;
  format: string;
  length: number;
  allow_reuse: boolean;
}

export interface FinishRegisterResponse {
  name: string;
  passkeyId: string;
  status: string;
  createdAt: string;
  credentialId: string;
}

export interface GetTemplateAnalyticsRequest {
}

export interface CreateNamespaceRequest {
  description: string;
  inheritPlatform: boolean;
  name: string;
  templateId: string;
}

export interface ComplianceReportFileResponse {
  content_type: string;
  data: number[];
}

export interface StepUpVerificationResponse {
  expires_at: string;
  verified: boolean;
}

export interface Challenge {
  userAgent: string;
  userId: string;
  verifiedAt: string | undefined;
  attempts: number;
  createdAt: string;
  expiresAt: string;
  factorId: string;
  ipAddress: string;
  maxAttempts: number;
  metadata: any;
  id: string;
  status: string;
  type: string;
}

export interface CreateTeamRequest {
  description: string;
  name: string;
}

export interface ListAuditEventsRequest {
}

export interface BanUserRequestDTO {
  expires_at: string | undefined;
  reason: string;
}

export interface ListUsersResponse {
  limit: number;
  page: number;
  total: number;
  total_pages: number;
  users: User | undefined[];
}

export interface SendOTPRequest {
  user_id: string;
}

export interface TrustedContactInfo {
  phone: string;
  relationship: string;
  verified: boolean;
  verifiedAt: string | undefined;
  active: boolean;
  email: string;
  id: string;
  name: string;
}

export interface SignUpResponse {
  message: string;
  status: string;
}

export interface ConsentRecordResponse {
  id: string;
}

export interface ListSecretsRequest {
}

export interface ResendRequest {
  email: string;
}

export interface RestoreTemplateVersionRequest {
}

export interface CreateAPIKeyRequest {
  allowed_ips: string[];
  description: string;
  metadata: any;
  name: string;
  permissions: any;
  rate_limit: number;
  scopes: string[];
}

export interface InviteMemberRequest {
  email: string;
  role: string;
}

export interface SendCodeResponse {
  status: string;
  dev_code: string;
}

export interface ClientSummary {
  applicationType: string;
  clientID: string;
  createdAt: string;
  isOrgLevel: boolean;
  name: string;
}

export interface UserService {
  [key: string]: any;
}

export interface MigrateRBACRequest {
  dryRun: boolean;
  keepRbacPolicies: boolean;
  namespaceId: string;
  validateEquivalence: boolean;
}

export interface FacialCheckConfig {
  enabled: boolean;
  motionCapture: boolean;
  variant: string;
}

export interface CheckSubResult {
}

export interface ProviderSession {
}

export interface TwoFAEnableResponse {
  status: string;
  totp_uri: string;
}

export interface StartVideoSessionRequest {
  videoSessionId: string;
}

export interface UploadDocumentResponse {
  processingTime: string;
  status: string;
  uploadedAt: string;
  documentId: string;
  message: string;
}

export interface ScheduleVideoSessionRequest {
  scheduledAt: string;
  sessionId: string;
  timeZone: string;
}

export interface ResourceAttributeRequest {
  description: string;
  name: string;
  required: boolean;
  type: string;
  default: any;
}

export interface StartRecoveryRequest {
  deviceId: string;
  email: string;
  preferredMethod: string;
  userId: string;
}

export interface RecoveryCodesConfig {
  codeCount: number;
  codeLength: number;
  enabled: boolean;
  format: string;
  regenerateCount: number;
  allowDownload: boolean;
  allowPrint: boolean;
  autoRegenerate: boolean;
}

export interface VideoSessionInfo {
}

export interface RetentionConfig {
  archiveBeforePurge: boolean;
  archivePath: string;
  enabled: boolean;
  gracePeriodDays: number;
  purgeSchedule: string;
}

export interface StepUpEvaluationResponse {
  reason: string;
  required: boolean;
}

export interface ComplianceReportsResponse {
  reports: any[];
}

export interface ForgetDeviceResponse {
  message: string;
  success: boolean;
}

export interface DeleteEntryRequest {
}

export interface ProvidersConfig {
  email: EmailProviderConfig;
  sms: SMSProviderConfig | undefined;
}

export interface TemplateEngine {
}

export interface MFAStatus {
  enabled: boolean;
  enrolledFactors: FactorInfo[];
  gracePeriod: string | undefined;
  policyActive: boolean;
  requiredCount: number;
  trustedDevice: boolean;
}

export interface MemberHandler {
}

export interface ConsentAuditLogsResponse {
  audit_logs: any[];
}

export interface PolicyPreviewResponse {
  actions: string[];
  description: string;
  expression: string;
  name: string;
  resourceType: string;
}

export interface DeleteContentTypeRequest {
}

export interface GetAPIKeyRequest {
}

export interface UpdateRecoveryConfigRequest {
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
  riskScoreThreshold: number;
}

export interface UpdatePolicy_req {
  version: string | undefined;
  content: string | undefined;
  status: string | undefined;
  title: string | undefined;
}

export interface StepUpVerification {
  user_agent: string;
  created_at: string;
  rule_name: string;
  verified_at: string;
  ip: string;
  reason: string;
  security_level: string;
  user_id: string;
  device_id: string;
  id: string;
  metadata: any;
  method: string;
  org_id: string;
  session_id: string;
  expires_at: string;
}

export interface CreateAppRequest {
  [key: string]: any;
}

export interface UpdateAppRequest {
}

export interface HealthCheckResponse {
  message: string;
  providersStatus: any;
  version: string;
  enabledMethods: string[];
  healthy: boolean;
}

export interface CancelRecoveryRequest {
  reason: string;
  sessionId: string;
}

export interface ComplianceProfileResponse {
  id: string;
}

export interface TOTPFactorAdapter {
}

export interface BeginRegisterResponse {
  challenge: string;
  options: any;
  timeout: Duration;
  userId: string;
}

export interface AccountLockoutError {
}

export interface DataDeletionRequestInput {
  deleteSections: string[];
  reason: string;
}

export interface DataExportRequest {
  completedAt: string | undefined;
  errorMessage: string;
  format: string;
  createdAt: string;
  exportUrl: string;
  status: string;
  updatedAt: string;
  exportPath: string;
  id: string;
  includeSections: string[];
  userId: string;
  expiresAt: string | undefined;
  exportSize: number;
  ipAddress: string;
  organizationId: string;
}

export interface UpdateNamespaceRequest {
  description: string;
  inheritPlatform: boolean | undefined;
  name: string;
}

export interface ConsentPolicy {
  active: boolean;
  consentType: string;
  content: string;
  createdAt: string;
  description: string;
  metadata: Record<string, any>;
  publishedAt: string | undefined;
  validityPeriod: number | undefined;
  name: string;
  organizationId: string;
  renewable: boolean;
  required: boolean;
  updatedAt: string;
  version: string;
  createdBy: string;
  id: string;
}

export interface ClientRegistrationRequest {
  post_logout_redirect_uris: string[];
  require_pkce: boolean;
  response_types: string[];
  application_type: string;
  client_name: string;
  policy_uri: string;
  redirect_uris: string[];
  trusted_client: boolean;
  contacts: string[];
  grant_types: string[];
  logo_uri: string;
  scope: string;
  require_consent: boolean;
  token_endpoint_auth_method: string;
  tos_uri: string;
}

export interface IntrospectionService {
}

export interface JWKSResponse {
  keys: JWK[];
}

export interface ListOrganizationsRequest {
}

export interface ProviderInfo {
  createdAt: string;
  domain: string;
  providerId: string;
  type: string;
}

export interface StatsResponse {
  active_sessions: number;
  active_users: number;
  banned_users: number;
  timestamp: string;
  total_sessions: number;
  total_users: number;
}

export interface DashboardExtension {
}

export interface IPWhitelistConfig {
  enabled: boolean;
  strict_mode: boolean;
}

export interface GetSecurityQuestionsResponse {
  questions: SecurityQuestionInfo[];
}

export interface UpdatePasskeyResponse {
  passkeyId: string;
  updatedAt: string;
  name: string;
}

export interface IDTokenClaims {
  name: string;
  nonce: string;
  auth_time: number;
  email: string;
  given_name: string;
  preferred_username: string;
  session_state: string;
  email_verified: boolean;
  family_name: string;
}

export interface JWKSService {
}

export interface InviteMemberHandlerRequest {
}

export interface GetImpersonationRequest {
}

export interface ConsentManager {
}

export interface OrganizationHandler {
}

export interface SchemaValidator {
  [key: string]: any;
}

export interface MockSocialAccountRepository {
  [key: string]: any;
}

export interface NotificationType {
  [key: string]: any;
}

export interface ListSecretsResponse {
  [key: string]: any;
}

export interface Repository {
  [key: string]: any;
}

export interface NoOpSMSProvider {
  [key: string]: any;
}

export interface FactorEnrollmentResponse {
  provisioningData: any;
  status: string;
  type: string;
  factorId: string;
}

export interface UpdateConsentRequest {
  granted: boolean | undefined;
  metadata: any;
  reason: string;
}

export interface ResendNotificationRequest {
}

export interface BackupAuthVideoResponse {
  session_id: string;
}

export interface SessionStatsResponse {
  activeSessions: number;
  deviceCount: number;
  locationCount: number;
  newestSession: string | undefined;
  oldestSession: string | undefined;
  totalSessions: number;
}

export interface ConsentDecision {
}

export interface VersioningConfig {
  autoCleanup: boolean;
  cleanupInterval: Duration;
  maxVersions: number;
  retentionDays: number;
}

export interface JWTKey {
  [key: string]: any;
}

export interface CreateAPIKeyResponse {
  message: string;
  api_key: APIKey | undefined;
}

export interface ComplianceTemplate {
  mfaRequired: boolean;
  name: string;
  passwordMinLength: number;
  requiredPolicies: string[];
  standard: string;
  auditFrequencyDays: number;
  dataResidency: string;
  description: string;
  requiredTraining: string[];
  retentionDays: number;
  sessionMaxAge: number;
}

export interface CreateEvidenceRequest {
  controlId: string;
  description: string;
  evidenceType: string;
  fileUrl: string;
  standard: string;
  title: string;
}

export interface LimitResult {
}

export interface LinkResponse {
  message: string;
  user: any;
}

export interface RedisChallengeStore {
  [key: string]: any;
}

export interface MockSessionService {
  [key: string]: any;
}

export interface RedisStateStore {
  [key: string]: any;
}

export interface ActionResponse {
  name: string;
  namespaceId: string;
  createdAt: string;
  description: string;
  id: string;
}

export interface StripeIdentityProvider {
}

export interface TrustedContact {
}

export interface VideoSessionResult {
}

export interface CreatePolicy_req {
  policyType: string;
  standard: string;
  title: string;
  version: string;
  content: string;
}

export interface ListAppsRequest {
}

export interface ClientAuthenticator {
}

export interface OIDCLoginRequest {
  nonce: string;
  redirectUri: string;
  scope: string;
  state: string;
}

export interface ApproveRecoveryResponse {
  sessionId: string;
  approved: boolean;
  approvedAt: string;
  message: string;
}

export interface BackupAuthContactResponse {
  id: string;
}

export interface ComplianceViolation {
  description: string;
  id: string;
  profileId: string;
  resolvedAt: string | undefined;
  status: string;
  violationType: string;
  appId: string;
  createdAt: string;
  metadata: any;
  resolvedBy: string;
  severity: string;
  userId: string;
}

export interface BackupCodeFactorAdapter {
}

export interface MFASession {
  metadata: any;
  verifiedFactors: string[];
  expiresAt: string;
  id: string;
  riskLevel: string;
  sessionToken: string;
  userAgent: string;
  userId: string;
  completedAt: string | undefined;
  createdAt: string;
  factorsRequired: number;
  factorsVerified: number;
  ipAddress: string;
}

export interface ID {
  [key: string]: any;
}

export interface AnalyticsSummary {
  allowedCount: number;
  cacheHitRate: number;
  totalEvaluations: number;
  totalPolicies: number;
  activePolicies: number;
  avgLatencyMs: number;
  deniedCount: number;
  topPolicies: PolicyStats[];
  topResourceTypes: ResourceTypeStats[];
}

export interface ErrorResponse {
  code: string;
  details: any;
  error: string;
}

export interface RecoveryConfiguration {
}

export interface ComplianceEvidencesResponse {
  evidence: any[];
}

export interface DeleteSecretRequest {
}

export interface RBACMigrationService {
  [key: string]: any;
}

export interface AnalyticsResponse {
  generatedAt: string;
  summary: AnalyticsSummary;
  timeRange: any;
}

export interface UpdateContentTypeRequest {
}

export interface GetContentTypeRequest {
}

export interface RegenerateCodesRequest {
  user_id: string;
  count: number;
}

export interface RemoveTrustedContactRequest {
  contactId: string;
}

export interface AppServiceAdapter {
}

export interface ConsentTypeStatus {
  expiresAt: string | undefined;
  granted: boolean;
  grantedAt: string;
  needsRenewal: boolean;
  type: string;
  version: string;
}

export interface ImpersonationVerifyResponse {
  impersonator_id: string;
  is_impersonating: boolean;
  target_user_id: string;
}

export interface TwoFASendOTPResponse {
  code: string;
  status: string;
}

export interface RequestTrustedContactVerificationRequest {
  contactId: string;
  sessionId: string;
}

export interface BunRepository {
}

export interface DeclineInvitationRequest {
}

export interface RateLimitRule {
  max: number;
  window: Duration;
}

export interface ABTestService {
  [key: string]: any;
}

export interface EvaluationContext {
}

export interface Adapter {
}

export interface TwoFAErrorResponse {
  error: string;
}

export interface RecoverySessionInfo {
  totalSteps: number;
  userEmail: string;
  method: string;
  riskScore: number;
  status: string;
  userId: string;
  completedAt: string | undefined;
  createdAt: string;
  currentStep: number;
  expiresAt: string;
  id: string;
  requiresReview: boolean;
}

export interface GenerateReport_req {
  format: string;
  period: string;
  reportType: string;
  standard: string;
}

export interface NotificationProvider {
  [key: string]: any;
}

export interface WebAuthnFactorAdapter {
}

export interface ConsentReport {
  usersWithConsent: number;
  completedDeletions: number;
  consentsByType: any;
  dpasExpiringSoon: number;
  organizationId: string;
  pendingDeletions: number;
  reportPeriodStart: string;
  totalUsers: number;
  consentRate: number;
  dataExportsThisPeriod: number;
  dpasActive: number;
  reportPeriodEnd: string;
}

export interface MigrateAllResponse {
  convertedPolicies: PolicyPreviewResponse[];
  errors: MigrationErrorResponse[];
  failedPolicies: number;
  migratedPolicies: number;
  skippedPolicies: number;
  completedAt: string;
  dryRun: boolean;
  startedAt: string;
  totalPolicies: number;
}

export interface GetMigrationStatusRequest {
  [key: string]: any;
}

export interface GetTemplateVersionRequest {
}

export interface APIKey {
  [key: string]: any;
}

export interface UnpublishContentTypeRequest {
}

export interface ListTrustedContactsResponse {
  contacts: TrustedContactInfo[];
  count: number;
}

export interface ImpersonationEndResponse {
  status: string;
  ended_at: string;
}

export interface DiscoveryService {
}

export interface UpdateProvider_req {
  config: any;
  isActive: boolean;
  isDefault: boolean;
}

export interface CompleteVideoSessionResponse {
  message: string;
  result: string;
  videoSessionId: string;
  completedAt: string;
}

export interface GetRecoveryStatsResponse {
  successRate: number;
  successfulRecoveries: number;
  averageRiskScore: number;
  methodStats: any;
  pendingRecoveries: number;
  totalAttempts: number;
  adminReviewsRequired: number;
  failedRecoveries: number;
  highRiskAttempts: number;
}

export interface CreateSessionHTTPRequest {
  cancelUrl: string;
  config: any;
  metadata: any;
  provider: string;
  requiredChecks: string[];
  successUrl: string;
}

export interface ComplianceReportResponse {
  id: string;
}

export interface GetTeamRequest {
}

export interface ResourceAttribute {
  [key: string]: any;
}

export interface GetTemplateRequest {
}

export interface VerificationResponse {
  verification: IdentityVerification | undefined;
}

export interface IDVerificationWebhookResponse {
  status: string;
}

export interface RemoveTeamMemberRequest {
}

export interface ListPasskeysRequest {
}

export interface AdminHandler {
}

export interface FactorPriority {
  [key: string]: any;
}

export interface CreateTrainingRequest {
  standard: string;
  trainingType: string;
  userId: string;
}

export interface ComplianceTrainingResponse {
  id: string;
}

export interface EmailServiceAdapter {
}

export interface Email {
}

export interface SessionAutoSendConfig {
  suspicious_login: boolean;
  all_revoked: boolean;
  device_removed: boolean;
  new_device: boolean;
  new_location: boolean;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  email_verified: boolean;
  name: string;
  role: string;
  user_organization_id: string | undefined;
  app_id: string;
  metadata: any;
  password: string;
}

export interface Session {
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
  id: string;
  userId: string;
  token: string;
  expiresAt: string;
}

export interface CreateSessionRequest {
}

export interface ListUsersRequest {
  status: string;
  user_organization_id: string | undefined;
  app_id: string;
  limit: number;
  page: number;
  role: string;
  search: string;
}

export interface ResourceResponse {
  attributes: ResourceAttribute[];
  createdAt: string;
  description: string;
  id: string;
  namespaceId: string;
  type: string;
}

export interface UpdateFieldRequest {
}

export interface NotificationErrorResponse {
  error: string;
}

export interface DisableRequest {
  user_id: string;
}

export interface TwoFAStatusDetailResponse {
  enabled: boolean;
  method: string;
  trusted: boolean;
}

export interface VerifyTrustedContactResponse {
  contactId: string;
  message: string;
  verified: boolean;
  verifiedAt: string;
}

export interface DocumentVerification {
}

export interface OnfidoConfig {
  includeWatchlistReport: boolean;
  workflowId: string;
  enabled: boolean;
  facialCheck: FacialCheckConfig;
  includeFacialReport: boolean;
  region: string;
  webhookToken: string;
  apiToken: string;
  documentCheck: DocumentCheckConfig;
  includeDocumentReport: boolean;
}

export interface ComplianceCheck {
  evidence: string[];
  lastCheckedAt: string;
  nextCheckAt: string;
  profileId: string;
  result: any;
  status: string;
  appId: string;
  id: string;
  checkType: string;
  createdAt: string;
}

export interface UpdateMemberRequest {
  role: string;
}

export interface ConsentDashboardConfig {
  enabled: boolean;
  path: string;
  showAuditLog: boolean;
  showConsentHistory: boolean;
  showCookiePreferences: boolean;
  showDataDeletion: boolean;
  showDataExport: boolean;
  showPolicies: boolean;
}

export interface ConfigSourceConfig {
  autoRefresh: boolean;
  enabled: boolean;
  prefix: string;
  priority: number;
  refreshInterval: Duration;
}

export interface ContentTypeService {
  [key: string]: any;
}

export interface TwoFARepository {
  [key: string]: any;
}

export interface ListJWTKeysResponse {
  [key: string]: any;
}

export interface ResourceRule {
  resource_type: string;
  security_level: string;
  sensitivity: string;
  action: string;
  description: string;
  org_id: string;
}

export interface AsyncConfig {
  retry_enabled: boolean;
  worker_pool_size: number;
  enabled: boolean;
  max_retries: number;
  persist_failures: boolean;
  queue_size: number;
  retry_backoff: string[];
}

export interface CompleteRecoveryResponse {
  completedAt: string;
  message: string;
  sessionId: string;
  status: string;
  token: string;
}

export interface ComplianceEvidence {
  collectedBy: string;
  createdAt: string;
  description: string;
  metadata: any;
  controlId: string;
  evidenceType: string;
  fileHash: string;
  fileUrl: string;
  id: string;
  profileId: string;
  standard: string;
  title: string;
  appId: string;
}

export interface CookieConsentRequest {
  marketing: boolean;
  personalization: boolean;
  sessionId: string;
  thirdParty: boolean;
  analytics: boolean;
  bannerVersion: string;
  essential: boolean;
  functional: boolean;
}

export interface EndImpersonationRequest {
  reason: string;
  impersonation_id: string;
}

export interface TokenIntrospectionRequest {
  client_id: string;
  client_secret: string;
  token: string;
  token_type_hint: string;
}

export interface GetOrganizationRequest {
}

export interface CompleteTraining_req {
  score: number;
}

export interface VerifyCodeRequest {
  sessionId: string;
  code: string;
}

export interface RejectRecoveryRequest {
  notes: string;
  reason: string;
  sessionId: string;
}

export interface ListProfilesFilter {
  appId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface AdminAddProviderRequest {
  provider: string;
  scopes: string[];
  appId: string;
  clientId: string;
  clientSecret: string;
  enabled: boolean;
}

export interface MemoryStateStore {
  [key: string]: any;
}

export interface SecretTreeNode {
  [key: string]: any;
}

export interface FactorStatus {
  [key: string]: any;
}

export interface AuthAutoSendConfig {
  password_reset: boolean;
  verification_email: boolean;
  welcome: boolean;
  email_otp: boolean;
  magic_link: boolean;
  mfa_code: boolean;
}

export interface MigrateAllRequest {
  dryRun: boolean;
  preserveOriginal: boolean;
}

export interface ListMembersRequest {
}

export interface ServiceImpl {
  [key: string]: any;
}

export interface CompliancePolicy {
  appId: string;
  approvedAt: string | undefined;
  profileId: string;
  standard: string;
  content: string;
  status: string;
  title: string;
  id: string;
  policyType: string;
  reviewDate: string;
  approvedBy: string;
  createdAt: string;
  effectiveDate: string;
  metadata: any;
  updatedAt: string;
  version: string;
}

export interface Config {
  max_impersonation_duration: Duration;
  required_role: string;
  allow_impersonation: boolean;
  allow_user_creation: boolean;
  allow_user_deletion: boolean;
}

export interface CreateEntryRequest {
}

export interface Middleware {
}

export interface DefaultProviderRegistry {
}

export interface StartRecoveryResponse {
  completedSteps: number;
  expiresAt: string;
  requiredSteps: number;
  requiresReview: boolean;
  riskScore: number;
  sessionId: string;
  status: string;
  availableMethods: string[];
}

export interface MultiStepRecoveryConfig {
  enabled: boolean;
  highRiskSteps: string[];
  lowRiskSteps: string[];
  mediumRiskSteps: string[];
  minimumSteps: number;
  sessionExpiry: Duration;
  allowStepSkip: boolean;
  allowUserChoice: boolean;
  requireAdminApproval: boolean;
}

export interface UpdateMemberHandlerRequest {
}

export interface CloneContentTypeRequest {
}

export interface ContinueRecoveryRequest {
  method: string;
  sessionId: string;
}

export interface MembersResponse {
  members: Member | undefined[];
  total: number;
}

export interface DeleteUserRequestDTO {
}

export interface RevokeSessionRequestDTO {
}

export interface AuditLogEntry {
  action: string;
  appId: string;
  newValue: any;
  resourceType: string;
  userOrganizationId: string | undefined;
  actorId: string;
  environmentId: string;
  id: string;
  ipAddress: string;
  oldValue: any;
  resourceId: string;
  timestamp: string;
  userAgent: string;
}

export interface TwoFABackupCodesResponse {
  codes: string[];
}

export interface ComplianceTemplatesResponse {
  templates: any[];
}

export interface StepUpRememberedDevice {
  created_at: string;
  last_used_at: string;
  security_level: string;
  user_agent: string;
  device_id: string;
  device_name: string;
  expires_at: string;
  id: string;
  ip: string;
  org_id: string;
  remembered_at: string;
  user_id: string;
}

export interface DeletePasskeyRequest {
}

export interface RegisterProviderRequest {
  attributeMapping: any;
  oidcIssuer: string;
  providerId: string;
  samlCert: string;
  samlEntryPoint: string;
  domain: string;
  oidcClientID: string;
  oidcClientSecret: string;
  oidcRedirectURI: string;
  samlIssuer: string;
  type: string;
}

export interface TestPolicyRequest {
  testCases: TestCase[];
  actions: string[];
  expression: string;
  resourceType: string;
}

export interface BulkUnpublishRequest {
  ids: string[];
}

export interface AdminBlockUser_req {
  reason: string;
}

export interface StepUpErrorResponse {
  error: string;
}

export interface MockStateStore {
}

export interface StateStorageConfig {
  redisDb: number;
  redisPassword: string;
  stateTtl: Duration;
  useRedis: boolean;
  redisAddr: string;
}

export interface IdentityVerification {
  [key: string]: any;
}

export interface ExtensionRegistry {
  [key: string]: any;
}

export interface ContinueRecoveryResponse {
  method: string;
  sessionId: string;
  totalSteps: number;
  currentStep: number;
  data: any;
  expiresAt: string;
  instructions: string;
}

export interface MFABypassResponse {
  reason: string;
  userId: string;
  expiresAt: string;
  id: string;
}

export interface TwoFARequiredResponse {
  device_id: string;
  require_twofa: boolean;
  user: User | undefined;
}

export interface OIDCState {
}

export interface CreateTemplateRequest {
  [key: string]: any;
}

export interface GenerateTokenResponse {
  [key: string]: any;
}

export interface BulkRequest {
  ids: string[];
}

export interface CreateTemplateVersion_req {
  changes: string;
}

export interface VerifyAPIKeyRequest {
  key: string;
}

export interface AcceptInvitationRequest {
}

export interface FinishLoginRequest {
  response: any;
  remember: boolean;
}

export interface MockOrganizationUIExtension {
}

export interface SecretDTO {
  [key: string]: any;
}

export interface SendRequest {
  email: string;
}

export interface VerifySecurityAnswersRequest {
  sessionId: string;
  answers: any;
}

export interface RiskAssessmentConfig {
  mediumRiskThreshold: number;
  newDeviceWeight: number;
  requireReviewAbove: number;
  velocityWeight: number;
  highRiskThreshold: number;
  historyWeight: number;
  lowRiskThreshold: number;
  newIpWeight: number;
  newLocationWeight: number;
  blockHighRisk: boolean;
  enabled: boolean;
}

export interface AdminBypassRequest {
  reason: string;
  userId: string;
  duration: number;
}

export interface RiskFactor {
}

export interface DataDeletionRequest {
  rejectedAt: string | undefined;
  retentionExempt: boolean;
  userId: string;
  createdAt: string;
  id: string;
  status: string;
  archivePath: string;
  completedAt: string | undefined;
  exemptionReason: string;
  requestReason: string;
  updatedAt: string;
  approvedAt: string | undefined;
  approvedBy: string;
  errorMessage: string;
  deleteSections: string[];
  ipAddress: string;
  organizationId: string;
}

export interface JWK {
  use: string;
  alg: string;
  e: string;
  kid: string;
  kty: string;
  n: string;
}

export interface RiskLevel {
  [key: string]: any;
}

export interface BatchEvaluationResult {
  resourceId: string;
  resourceType: string;
  action: string;
  allowed: boolean;
  error: string;
  evaluationTimeMs: number;
  index: number;
  policies: string[];
}

export interface RestoreRevisionRequest {
}

export interface BackupAuthStatusResponse {
  status: string;
}

export interface RejectRecoveryResponse {
  message: string;
  reason: string;
  rejected: boolean;
  rejectedAt: string;
  sessionId: string;
}

export interface RecoveryAttemptLog {
}

export interface GetFactorRequest {
}

export interface GetChallengeStatusResponse {
  attempts: number;
  availableFactors: FactorInfo[];
  challengeId: string;
  factorsRequired: number;
  factorsVerified: number;
  maxAttempts: number;
  status: string;
}

export interface BeginLoginRequest {
  userId: string;
  userVerification: string;
}

export interface Invitation {
  [key: string]: any;
}

export interface SocialAccount {
  [key: string]: any;
}

export interface CreateUserRequestDTO {
  email_verified: boolean;
  metadata: any;
  name: string;
  password: string;
  role: string;
  username: string;
  email: string;
}

export interface SessionStats {
}

export interface ConsentStatusResponse {
  status: string;
}

export interface ConsentStats {
  type: string;
  averageLifetime: number;
  expiredCount: number;
  grantRate: number;
  grantedCount: number;
  revokedCount: number;
  totalConsents: number;
}

export interface CreateConsentRequest {
  expiresIn: number | undefined;
  granted: boolean;
  metadata: any;
  purpose: string;
  userId: string;
  version: string;
  consentType: string;
}

export interface ContentEntryService {
  [key: string]: any;
}

export interface ChallengeStatus {
  [key: string]: any;
}

export interface MockAppService {
  [key: string]: any;
}

export interface RiskAssessment {
  factors: string[];
  level: string;
  metadata: any;
  recommended: string[];
  score: number;
}

export interface AppHandler {
}

export interface ChallengeSession {
}

export interface ListSessionsRequestDTO {
}

export interface PhoneVerifyResponse {
  session: Session | undefined;
  token: string;
  user: User | undefined;
}

export interface Duration {
  [key: string]: any;
}

export interface OrganizationAction {
  [key: string]: any;
}

export interface StripeIdentityConfig {
  requireMatchingSelfie: boolean;
  returnUrl: string;
  useMock: boolean;
  webhookSecret: string;
  allowedTypes: string[];
  apiKey: string;
  enabled: boolean;
  requireLiveCapture: boolean;
}

export interface NotificationTemplateResponse {
  template: any;
}

export interface CreateProfileFromTemplate_req {
  standard: string;
}

export interface VerifyEnrolledFactorRequest {
  code: string;
  data: any;
}

export interface ListFactorsRequest {
}

export interface PrivacySettings {
  allowDataPortability: boolean;
  contactEmail: string;
  dataExportExpiryHours: number;
  dataRetentionDays: number;
  metadata: Record<string, any>;
  organizationId: string;
  requireExplicitConsent: boolean;
  anonymousConsentEnabled: boolean;
  autoDeleteAfterDays: number;
  consentRequired: boolean;
  cookieConsentStyle: string;
  dpoEmail: string;
  exportFormat: string[];
  id: string;
  contactPhone: string;
  cookieConsentEnabled: boolean;
  requireAdminApprovalForDeletion: boolean;
  ccpaMode: boolean;
  createdAt: string;
  deletionGracePeriodDays: number;
  gdprMode: boolean;
  updatedAt: string;
}

export interface ImpersonationContext {
  impersonator_id: string | undefined;
  indicator_message: string;
  is_impersonating: boolean;
  target_user_id: string | undefined;
  impersonation_id: string | undefined;
}

export interface UserVerificationStatus {
  [key: string]: any;
}

export interface VerificationListResponse {
  verifications: IdentityVerification | undefined[];
  limit: number;
  offset: number;
  total: number;
}

export interface NotificationResponse {
  notification: any;
}

export interface SecurityQuestionsConfig {
  predefinedQuestions: string[];
  requiredToRecover: number;
  enabled: boolean;
  forbidCommonAnswers: boolean;
  lockoutDuration: Duration;
  maxAnswerLength: number;
  minimumQuestions: number;
  requireMinLength: number;
  allowCustomQuestions: boolean;
  caseSensitive: boolean;
  maxAttempts: number;
}

export interface UpdateProfileRequest {
  mfaRequired: boolean | undefined;
  name: string | undefined;
  retentionDays: number | undefined;
  status: string | undefined;
}

export interface StepUpDevicesResponse {
  devices: any;
  count: number;
}

export interface EmailConfig {
  enabled: boolean;
  provider: string;
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
  code_expiry_minutes: number;
  code_length: number;
}

export interface TeamsResponse {
  teams: Team | undefined[];
  total: number;
}

export interface SetUserRoleRequest {
  app_id: string;
  role: string;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface BatchEvaluateRequest {
  requests: EvaluateRequest[];
}

export interface RenderTemplateRequest {
}

export interface WebAuthnWrapper {
}

export interface MigrationResponse {
  status: string;
  message: string;
  migrationId: string;
  startedAt: string;
}

export interface TemplatesListResponse {
  categories: string[];
  templates: TemplateResponse | undefined[];
  totalCount: number;
}

export interface CompareRevisionsRequest {
}

export interface ContextRule {
  condition: string;
  description: string;
  name: string;
  org_id: string;
  security_level: string;
}

export interface AdminPolicyRequest {
  allowedTypes: string[];
  enabled: boolean;
  gracePeriod: number;
  requiredFactors: number;
}

export interface ConsentReportResponse {
  id: string;
}

export interface ConsentSettingsResponse {
  settings: any;
}

export interface Webhook {
  events: string[];
  secret: string;
  enabled: boolean;
  createdAt: string;
  id: string;
  organizationId: string;
  url: string;
}

export interface ComplianceTraining {
  standard: string;
  userId: string;
  appId: string;
  createdAt: string;
  expiresAt: string | undefined;
  id: string;
  metadata: any;
  score: number;
  status: string;
  trainingType: string;
  completedAt: string | undefined;
  profileId: string;
}

export interface ResetUserMFAResponse {
  message: string;
  success: boolean;
  devicesRevoked: number;
  factorsReset: number;
}

export interface FactorEnrollmentRequest {
  metadata: any;
  name: string;
  priority: string;
  type: string;
}

export interface ClientUpdateRequest {
  trusted_client: boolean | undefined;
  allowed_scopes: string[];
  name: string;
  policy_uri: string;
  post_logout_redirect_uris: string[];
  redirect_uris: string[];
  require_consent: boolean | undefined;
  response_types: string[];
  token_endpoint_auth_method: string;
  contacts: string[];
  grant_types: string[];
  logo_uri: string;
  require_pkce: boolean | undefined;
  tos_uri: string;
}

export interface TeamsListResponse {
  [key: string]: any;
}

export interface EvaluateRequest {
  context: any;
  principal: any;
  request: any;
  resource: any;
  resourceId: string;
  resourceType: string;
  action: string;
}

export interface PoliciesListResponse {
  page: number;
  pageSize: number;
  policies: PolicyResponse | undefined[];
  totalCount: number;
}

export interface TestProvider_req {
  providerType: string;
  testRecipient: string;
  providerName: string;
}

export interface SignUpRequest {
  username: string;
  password: string;
}

export interface CreateOrganizationHandlerRequest {
  [key: string]: any;
}

export interface ProviderListResponse {
  providers: ProviderInfo[];
  total: number;
}

export interface CreateJWTKeyRequest {
  algorithm: string;
  curve: string;
  expiresAt: string | undefined;
  isPlatformKey: boolean;
  keyType: string;
  metadata: any;
}

export interface StatsDTO {
  [key: string]: any;
}

export interface ResourceTypeStats {
  evaluationCount: number;
  resourceType: string;
  allowRate: number;
  avgLatencyMs: number;
}

export interface MigrateRolesRequest {
  dryRun: boolean;
}

export interface CheckRegistry {
}

export interface AuditEvent {
}

export interface StepUpVerificationsResponse {
  verifications: any[];
}

export interface UpdateOrganizationHandlerRequest {
}

export interface CallbackDataResponse {
  user: User | undefined;
  action: string;
  isNewUser: boolean;
}

export interface SuccessResponse {
  data: any;
  message: string;
}

export interface UnassignRoleRequest {
}

export interface ListTrustedDevicesResponse {
  count: number;
  devices: TrustedDevice[];
}

export interface TokenRequest {
  redirect_uri: string;
  refresh_token: string;
  audience: string;
  client_id: string;
  code_verifier: string;
  grant_type: string;
  scope: string;
  client_secret: string;
  code: string;
}

export interface ProviderConfig {
  [key: string]: any;
}

export interface QueryEntriesRequest {
}

export interface PolicyEngine {
}

export interface ComplianceCheckResponse {
  id: string;
}

export interface ListFactorsResponse {
  count: number;
  factors: Factor[];
}

export interface TeamHandler {
}

export interface FinishRegisterRequest {
  name: string;
  response: any;
  userId: string;
}

export interface StartImpersonationRequest {
  target_user_id: string;
  ticket_number: string;
  duration_minutes: number;
  reason: string;
}

export interface VideoVerificationConfig {
  provider: string;
  recordSessions: boolean;
  recordingRetention: Duration;
  requireLivenessCheck: boolean;
  livenessThreshold: number;
  requireAdminReview: boolean;
  requireScheduling: boolean;
  sessionDuration: Duration;
  enabled: boolean;
  minScheduleAdvance: Duration;
}

export interface CompleteTrainingRequest {
  score: number;
}

export interface ComplianceViolationResponse {
  id: string;
}

export interface AddTeamMemberRequest {
  member_id: string;
}

export interface Document {
  [key: string]: any;
}

export interface RecoveryStatus {
  [key: string]: any;
}

export interface MultitenancyStatusResponse {
  [key: string]: any;
}

export interface ChannelsResponse {
  channels: any;
  count: number;
}

export interface CreateProfileFromTemplateRequest {
  standard: string;
}

export interface StepUpPoliciesResponse {
  policies: any[];
}

export interface ChallengeRequest {
  context: string;
  factorTypes: string[];
  metadata: any;
  userId: string;
}

export interface FactorType {
  [key: string]: any;
}

export interface SaveNotificationSettings_req {
  cleanupAfter: string;
  retryAttempts: number;
  retryDelay: string;
  autoSendWelcome: boolean;
}

export interface RecoveryCodeUsage {
}

export interface ApproveRecoveryRequest {
  sessionId: string;
  notes: string;
}

export interface ComplianceUserTrainingResponse {
  user_id: string;
}

export interface StepUpRequirement {
  currency: string;
  id: string;
  org_id: string;
  required_level: string;
  risk_score: number;
  route: string;
  rule_name: string;
  method: string;
  reason: string;
  resource_action: string;
  session_id: string;
  status: string;
  user_agent: string;
  amount: number;
  fulfilled_at: string | undefined;
  challenge_token: string;
  current_level: string;
  expires_at: string;
  ip: string;
  metadata: any;
  resource_type: string;
  user_id: string;
  created_at: string;
}

export interface BeginRegisterRequest {
  userId: string;
  userVerification: string;
  authenticatorType: string;
  name: string;
  requireResidentKey: boolean;
}

export interface KeyPair {
}

export interface ServiceInterface {
  [key: string]: any;
}

export interface UpdatePolicyRequest {
  name: string;
  priority: number;
  resourceType: string;
  actions: string[];
  description: string;
  enabled: boolean | undefined;
  expression: string;
}

export interface MigrationErrorResponse {
  error: string;
  policyIndex: number;
  resource: string;
  subject: string;
}

export interface VerifyRecoveryCodeRequest {
  code: string;
  sessionId: string;
}

export interface ListRecoverySessionsRequest {
  status: string;
  organizationId: string;
  page: number;
  pageSize: number;
  requiresReview: boolean;
}

export interface ListJWTKeysRequest {
}

export interface UserServiceAdapter {
}

export interface AdaptiveMFAConfig {
  new_device_risk: number;
  risk_threshold: number;
  velocity_risk: number;
  enabled: boolean;
  factor_location_change: boolean;
  require_step_up_threshold: number;
  factor_ip_reputation: boolean;
  factor_new_device: boolean;
  factor_velocity: boolean;
  location_change_risk: number;
}

export interface IdentityVerificationSession {
  [key: string]: any;
}

export interface Organization {
  [key: string]: any;
}

export interface SMSProviderConfig {
  config: any;
  from: string;
  provider: string;
}

export interface ComplianceTrainingsResponse {
  training: any[];
}

export interface ComplianceEvidenceResponse {
  id: string;
}

export interface RouteRule {
  description: string;
  method: string;
  org_id: string;
  pattern: string;
  security_level: string;
}

export interface DeleteAppRequest {
}

export interface KeyStore {
}

export interface ImpersonateUserRequest {
  app_id: string;
  duration: Duration;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface ValidatePolicyRequest {
  expression: string;
  resourceType: string;
}

export interface DashboardConfig {
  showReports: boolean;
  showScore: boolean;
  showViolations: boolean;
  enabled: boolean;
  path: string;
  showRecentChecks: boolean;
}

export interface MultiSessionErrorResponse {
  error: string;
}

export interface ConsentPolicyResponse {
  id: string;
}

export interface GetAuditLogsRequestDTO {
}

export interface UnpublishEntryRequest {
}

export interface UnblockUserRequest {
  [key: string]: any;
}

export interface TwoFAStatusResponse {
  enabled: boolean;
  method: string;
  trusted: boolean;
}

export interface EnableRequest2FA {
  method: string;
  user_id: string;
}

export interface FactorInfo {
  type: string;
  factorId: string;
  metadata: any;
  name: string;
}

export interface UserAdapter {
}

export interface TokenIntrospectionResponse {
  active: boolean;
  aud: string[];
  client_id: string;
  exp: number;
  iat: number;
  scope: string;
  token_type: string;
  iss: string;
  jti: string;
  nbf: number;
  sub: string;
  username: string;
}

export interface GetMigrationStatusResponse {
  hasMigratedPolicies: boolean;
  lastMigrationAt: string;
  migratedCount: number;
  pendingRbacPolicies: number;
}

export interface CreateVerificationSession_req {
  config: any;
  metadata: any;
  provider: string;
  requiredChecks: string[];
  successUrl: string;
  cancelUrl: string;
}

export interface SaveBuilderTemplate_req {
  document: Document;
  name: string;
  subject: string;
  templateId?: string;
  templateKey: string;
}

export interface BackupAuthRecoveryResponse {
  session_id: string;
}

export interface TrustDeviceRequest {
  name: string;
  deviceId: string;
  metadata: any;
}

export interface JWTService {
}

export interface GenerateTokenRequest {
  userId: string;
  audience: string[];
  expiresIn: Duration;
  metadata: any;
  permissions: string[];
  scopes: string[];
  sessionId: string;
  tokenType: string;
}

export interface Member {
  [key: string]: any;
}

export interface NamespacesListResponse {
  namespaces: NamespaceResponse | undefined[];
  totalCount: number;
}

export interface DeviceInfo {
  deviceId: string;
  metadata: any;
  name: string;
}

export interface FactorsResponse {
  count: number;
  factors: any;
}

export interface UpdateMemberRoleRequest {
  role: string;
}

export interface FuncMap {
  [key: string]: any;
}

export interface IDVerificationStatusResponse {
  status: any;
}

export interface GetStatusRequest {
  device_id: string;
  user_id: string;
}

export interface EnableRequest {
}

export interface ComplianceReport {
  expiresAt: string;
  fileSize: number;
  fileUrl: string;
  format: string;
  generatedBy: string;
  reportType: string;
  standard: string;
  status: string;
  appId: string;
  createdAt: string;
  id: string;
  period: string;
  profileId: string;
  summary: any;
}

export interface VerificationRequest {
  challengeId: string;
  code: string;
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
}

export interface GetEntryStatsRequest {
}

export interface CreateABTestVariant_req {
  weight: number;
  body: string;
  name: string;
  subject: string;
}

export interface ListContentTypesRequest {
}

export interface BackupAuthSessionsResponse {
  sessions: any[];
}

export interface PasskeyInfo {
  createdAt: string;
  aaguid: string;
  credentialId: string;
  id: string;
  isResidentKey: boolean;
  lastUsedAt: string | undefined;
  name: string;
  signCount: number;
  authenticatorType: string;
}

export interface MemoryChallengeStore {
}

export interface UpdateTeamHandlerRequest {
}

export interface DeleteFieldRequest {
}

export interface VerificationResult {
}

export interface VerifyTrustedContactRequest {
  token: string;
}

export interface SecurityQuestion {
}

export interface BackupAuthQuestionsResponse {
  questions: string[];
}

export interface SessionTokenResponse {
  session: Session | undefined;
  token: string;
}

export interface ProviderRegistry {
  [key: string]: any;
}

export interface IDVerificationSessionResponse {
  session: any;
}

export interface CodesResponse {
  codes: string[];
}

export interface TimeBasedRule {
  org_id: string;
  security_level: string;
  description: string;
  max_age: Duration;
  operation: string;
}

export interface StepUpPolicy {
  created_at: string;
  description: string;
  metadata: any;
  name: string;
  org_id: string;
  priority: number;
  rules: any;
  updated_at: string;
  enabled: boolean;
  id: string;
  user_id: string;
}

export interface WebAuthnConfig {
  attestation_preference: string;
  authenticator_selection: any;
  enabled: boolean;
  rp_display_name: string;
  rp_id: string;
  rp_origins: string[];
  timeout: number;
}

export interface AccountLockedResponse {
  code: string;
  locked_minutes: number;
  locked_until: string;
  message: string;
}

export interface BatchEvaluateResponse {
  failureCount: number;
  results: BatchEvaluationResult | undefined[];
  successCount: number;
  totalEvaluations: number;
  totalTimeMs: number;
}

export interface UserInfoResponse {
  gender: string;
  locale: string;
  nickname: string;
  profile: string;
  zoneinfo: string;
  name: string;
  preferred_username: string;
  website: string;
  phone_number_verified: boolean;
  picture: string;
  email: string;
  family_name: string;
  given_name: string;
  phone_number: string;
  updated_at: number;
  email_verified: boolean;
  middle_name: string;
  sub: string;
  birthdate: string;
}

export interface GetValueRequest {
}

export interface Credential {
  [key: string]: any;
}

export interface CreateFieldRequest {
  [key: string]: any;
}

export interface UpdateEntryRequest {
}

export interface CheckResult {
  error: Error;
  evidence: string[];
  result: any;
  score: number;
  status: string;
  checkType: string;
}

export interface StepUpPolicyResponse {
  id: string;
}

export interface Factor {
  status: string;
  type: string;
  updatedAt: string;
  userId: string;
  createdAt: string;
  expiresAt: string | undefined;
  id: string;
  lastUsedAt: string | undefined;
  name: string;
  priority: string;
  verifiedAt: string | undefined;
  metadata: any;
}

export interface ListTeamsRequest {
}

export interface ProvidersResponse {
  providers: string[];
}

export interface ComplianceStatus {
  checksWarning: number;
  nextAudit: string;
  profileId: string;
  standard: string;
  lastChecked: string;
  overallStatus: string;
  score: number;
  violations: number;
  appId: string;
  checksFailed: number;
  checksPassed: number;
}

export interface SignInResponse {
  session: any;
  token: string;
  user: any;
}

export interface SSOAuthResponse {
  token: string;
  user: User | undefined;
  session: Session | undefined;
}

export interface SendResponse {
  devToken: string;
  status: string;
}

export interface CheckMetadata {
  autoRun: boolean;
  category: string;
  description: string;
  name: string;
  severity: string;
  standards: string[];
}

export interface StepUpAttempt {
  created_at: string;
  id: string;
  ip: string;
  method: string;
  failure_reason: string;
  org_id: string;
  requirement_id: string;
  success: boolean;
  user_agent: string;
  user_id: string;
}

export interface GetSecretRequest {
}

export interface ListEntriesRequest {
}

export interface CompliancePoliciesResponse {
  policies: any[];
}

export interface InitiateChallengeRequest {
  context: string;
  factorTypes: string[];
  metadata: any;
}

export interface AddMemberRequest {
  role: string;
  user_id: string;
}

export interface ConsentNotificationsConfig {
  channels: string[];
  notifyDeletionComplete: boolean;
  notifyOnRevoke: boolean;
  enabled: boolean;
  notifyDeletionApproved: boolean;
  notifyDpoEmail: string;
  notifyExportReady: boolean;
  notifyOnExpiry: boolean;
  notifyOnGrant: boolean;
}

export interface MFARepository {
  [key: string]: any;
}

export interface RevealValueResponse {
  [key: string]: any;
}

export interface DeleteAPIKeyRequest {
}

export interface NoOpEmailProvider {
  [key: string]: any;
}

export interface StepUpRequirementResponse {
  id: string;
}

export interface UpdateFactorRequest {
  metadata: any;
  name: string | undefined;
  priority: string | undefined;
  status: string | undefined;
}

export interface RollbackRequest {
  reason: string;
}

export interface ListSessionsResponse {
  limit: number;
  page: number;
  sessions: Session | undefined[];
  total: number;
  total_pages: number;
}

export interface ArchiveEntryRequest {
}

export interface ListRevisionsRequest {
}

export interface BackupAuthCodesResponse {
  codes: string[];
}

export interface CompliancePolicyResponse {
  id: string;
}

export interface MFAConfigResponse {
  allowed_factor_types: string[];
  enabled: boolean;
  required_factor_count: number;
}

export interface DeleteFactorRequest {
}

export interface RetryService {
  [key: string]: any;
}

export interface ImpersonateUserRequestDTO {
  duration: Duration;
}

export interface User {
  createdAt: string;
  updatedAt: string;
  organizationId?: string;
  id: string;
  email: string;
  name?: string;
  emailVerified: boolean;
}

export interface VerifyResponse {
  user: User | undefined;
  session: Session | undefined;
  success: boolean;
  token: string;
}

export interface RateLimiter {
}

export interface CreateTeamHandlerRequest {
}

export interface OrganizationUIRegistry {
}

export interface MembersListResponse {
  [key: string]: any;
}

export interface RequestTrustedContactVerificationResponse {
  contactId: string;
  contactName: string;
  expiresAt: string;
  message: string;
  notifiedAt: string;
}

export interface NoOpNotificationProvider {
  [key: string]: any;
}

export interface ResetUserMFARequest {
  reason: string;
}

export interface VerifyTokenResponse {
  [key: string]: any;
}

export interface ContentEntryHandler {
}

export interface ProviderCheckResult {
}

export interface CreateVerificationRequest {
}

export interface TrackNotificationEvent_req {
  templateId: string;
  event: string;
  eventData?: any;
  notificationId: string;
  organizationId?: string | undefined;
}

export interface WebhookConfig {
  enabled: boolean;
  expiry_warning_days: number;
  notify_on_created: boolean;
  notify_on_deleted: boolean;
  notify_on_expiring: boolean;
  notify_on_rate_limit: boolean;
  notify_on_rotated: boolean;
  webhook_urls: string[];
}

export interface ListReportsFilter {
  status: string | undefined;
  appId: string | undefined;
  format: string | undefined;
  profileId: string | undefined;
  reportType: string | undefined;
  standard: string | undefined;
}

export interface GetInvitationRequest {
}

export interface SetActiveRequest {
  id: string;
}

export interface ReorderFieldsRequest {
}

export interface EvaluateResponse {
  reason: string;
  allowed: boolean;
  cacheHit: boolean;
  error: string;
  evaluatedPolicies: number;
  evaluationTimeMs: number;
  matchedPolicies: string[];
}

export interface AddTrustedContactRequest {
  email: string;
  name: string;
  phone: string;
  relationship: string;
}

export interface NotificationsConfig {
  securityOfficerEmail: string;
  channels: string[];
  enabled: boolean;
  notifyAdminOnHighRisk: boolean;
  notifyAdminOnReviewNeeded: boolean;
  notifyOnRecoveryComplete: boolean;
  notifyOnRecoveryFailed: boolean;
  notifyOnRecoveryStart: boolean;
}

export interface SecurityQuestionInfo {
  id: string;
  isCustom: boolean;
  questionId: number;
  questionText: string;
}

export interface ConsentAuditLog {
  userId: string;
  consentId: string;
  ipAddress: string;
  newValue: Record<string, any>;
  purpose: string;
  reason: string;
  action: string;
  consentType: string;
  createdAt: string;
  id: string;
  organizationId: string;
  previousValue: Record<string, any>;
  userAgent: string;
}

export interface ConsentExportFileResponse {
  content_type: string;
  data: number[];
}

export interface VerificationRepository {
}

export interface DeclareABTestWinnerRequest {
}

export interface BackupAuthConfigResponse {
  config: any;
}

export interface ListEvidenceFilter {
  controlId: string | undefined;
  evidenceType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  appId: string | undefined;
}

export interface RevokeTrustedDeviceRequest {
}

export interface DeleteOrganizationRequest {
}

export interface LinkAccountRequest {
  provider: string;
  scopes: string[];
}

export interface ListAPIKeysResponse {
  [key: string]: any;
}

export interface ContentTypeHandler {
}

export interface Device {
  name?: string;
  type?: string;
  lastUsedAt: string;
  ipAddress?: string;
  userAgent?: string;
  id: string;
  userId: string;
}

export interface DocumentCheckConfig {
  enabled: boolean;
  extractData: boolean;
  validateDataConsistency: boolean;
  validateExpiry: boolean;
}

export interface DeclareABTestWinner_req {
  abTestGroup: string;
  winnerId: string;
}

export interface ListPoliciesFilter {
  appId: string | undefined;
  policyType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface BaseFactorAdapter {
}

export interface FactorAdapterRegistry {
}

export interface OAuthErrorResponse {
  error: string;
  error_description: string;
  error_uri: string;
  state: string;
}

export interface OnfidoProvider {
}

export interface UpdateTemplateRequest {
}

export interface VerifyCodeResponse {
  attemptsLeft: number;
  message: string;
  valid: boolean;
}

export interface VideoVerificationSession {
}

export interface CreateEvidence_req {
  description: string;
  evidenceType: string;
  fileUrl: string;
  standard: string;
  title: string;
  controlId: string;
}

export interface RemoveMemberRequest {
}

export interface ConsentsResponse {
  consents: any;
  count: number;
}

export interface AuthorizeRequest {
  id_token_hint: string;
  redirect_uri: string;
  response_type: string;
  state: string;
  ui_locales: string;
  client_id: string;
  code_challenge: string;
  code_challenge_method: string;
  login_hint: string;
  max_age: number | undefined;
  nonce: string;
  prompt: string;
  scope: string;
  acr_values: string;
}

export interface PreviewConversionRequest {
  condition: string;
  resource: string;
  subject: string;
  actions: string[];
}

export interface TestCaseResult {
  actual: boolean;
  error: string;
  evaluationTimeMs: number;
  expected: boolean;
  name: string;
  passed: boolean;
}

export interface BulkPublishRequest {
  ids: string[];
}

export interface EmailProviderConfig {
  config: any;
  from: string;
  from_name: string;
  provider: string;
  reply_to: string;
}

export interface AsyncAdapter {
}

export interface UpdateProviderRequest {
}

export interface BackupAuthContactsResponse {
  contacts: any[];
}

export interface ComplianceViolationsResponse {
  violations: any[];
}

export interface StepUpAuditLogsResponse {
  audit_logs: any[];
}

export interface AuditLogResponse {
  entries: AuditLogEntry | undefined[];
  page: number;
  pageSize: number;
  totalCount: number;
}

export interface PreviewTemplateRequest {
}

export interface RolesResponse {
  roles: Role | undefined[];
}

export interface ChallengeStatusResponse {
  sessionId: string;
  status: string;
  completedAt: string | undefined;
  expiresAt: string;
  factorsRemaining: number;
  factorsRequired: number;
  factorsVerified: number;
}

export interface UpdateTeamRequest {
  description: string;
  name: string;
}

export interface ImpersonationStartResponse {
  impersonator_id: string;
  session_id: string;
  started_at: string;
  target_user_id: string;
}

export interface GetStatsRequestDTO {
}

export interface NamespaceResponse {
  appId: string;
  description: string;
  inheritPlatform: boolean;
  policyCount: number;
  templateId: string | undefined;
  userOrganizationId: string | undefined;
  actionCount: number;
  createdAt: string;
  environmentId: string;
  id: string;
  name: string;
  resourceCount: number;
  updatedAt: string;
}

export interface App {
}

export interface ListTrainingFilter {
  appId: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
  trainingType: string | undefined;
  userId: string | undefined;
}

export interface CheckDependencies {
}

export interface EnrollFactorRequest {
  name: string;
  priority: string;
  type: string;
  metadata: any;
}

export interface ConsentExportResponse {
  id: string;
  status: string;
}

export interface PrivacySettingsRequest {
  deletionGracePeriodDays: number | undefined;
  requireAdminApprovalForDeletion: boolean | undefined;
  ccpaMode: boolean | undefined;
  contactEmail: string;
  contactPhone: string;
  dataExportExpiryHours: number | undefined;
  gdprMode: boolean | undefined;
  consentRequired: boolean | undefined;
  dataRetentionDays: number | undefined;
  dpoEmail: string;
  requireExplicitConsent: boolean | undefined;
  allowDataPortability: boolean | undefined;
  anonymousConsentEnabled: boolean | undefined;
  exportFormat: string[];
  autoDeleteAfterDays: number | undefined;
  cookieConsentEnabled: boolean | undefined;
  cookieConsentStyle: string;
}

export interface MigrationStatusResponse {
  environmentId: string;
  migratedCount: number;
  startedAt: string;
  userOrganizationId: string | undefined;
  validationPassed: boolean;
  completedAt: string | undefined;
  errors: string[];
  failedCount: number;
  progress: number;
  status: string;
  totalPolicies: number;
  appId: string;
}

export interface CreateActionRequest {
  description: string;
  name: string;
  namespaceId: string;
}

export interface GetEntryRequest {
}

export interface EmailFactorAdapter {
}

export interface AccessTokenClaims {
  client_id: string;
  scope: string;
  token_type: string;
}

export interface DiscoveryResponse {
  claims_supported: string[];
  jwks_uri: string;
  revocation_endpoint_auth_methods_supported: string[];
  subject_types_supported: string[];
  token_endpoint: string;
  claims_parameter_supported: boolean;
  grant_types_supported: string[];
  issuer: string;
  request_uri_parameter_supported: boolean;
  require_request_uri_registration: boolean;
  response_modes_supported: string[];
  response_types_supported: string[];
  userinfo_endpoint: string;
  authorization_endpoint: string;
  introspection_endpoint: string;
  introspection_endpoint_auth_methods_supported: string[];
  request_parameter_supported: boolean;
  scopes_supported: string[];
  code_challenge_methods_supported: string[];
  id_token_signing_alg_values_supported: string[];
  registration_endpoint: string;
  revocation_endpoint: string;
  token_endpoint_auth_methods_supported: string[];
}

export interface GetVersionsRequest {
}

export interface SecretsConfigSource {
}

export interface AccountAutoSendConfig {
  suspended: boolean;
  username_changed: boolean;
  deleted: boolean;
  email_change_request: boolean;
  email_changed: boolean;
  password_changed: boolean;
  reactivated: boolean;
}

export interface NoOpDocumentProvider {
  [key: string]: any;
}

export interface SendVerificationCodeResponse {
  expiresAt: string;
  maskedTarget: string;
  message: string;
  sent: boolean;
}

export interface MFAPolicy {
  id: string;
  lockoutDurationMinutes: number;
  maxFailedAttempts: number;
  organizationId: string;
  requiredFactorCount: number;
  requiredFactorTypes: string[];
  stepUpRequired: boolean;
  trustedDeviceDays: number;
  adaptiveMfaEnabled: boolean;
  allowedFactorTypes: string[];
  createdAt: string;
  gracePeriodDays: number;
  updatedAt: string;
}

export interface ImpersonationMiddleware {
}

export interface EncryptionConfig {
  masterKey: string;
  rotateKeyAfter: Duration;
  testOnStartup: boolean;
}

export interface AuthURLResponse {
  url: string;
}

export interface ProviderConfigResponse {
  appId: string;
  message: string;
  provider: string;
}

export interface TrustedDevice {
  deviceId: string;
  expiresAt: string;
  id: string;
  ipAddress: string;
  name: string;
  lastUsedAt: string | undefined;
  metadata: any;
  userAgent: string;
  userId: string;
  createdAt: string;
}

export interface StateStore {
}

export interface DiscoverProviderRequest {
  email: string;
}

export interface CreateSecretRequest {
  tags: string[];
  value: any;
  valueType: string;
  description: string;
  metadata: any;
  path: string;
}

export interface VerifyTokenRequest {
  audience: string[];
  token: string;
  tokenType: string;
}

export interface Time {
  [key: string]: any;
}

export interface DB {
  [key: string]: any;
}

export interface RWMutex {
  [key: string]: any;
}

export interface PolicyResponse {
  createdAt: string;
  enabled: boolean;
  environmentId: string;
  priority: number;
  expression: string;
  resourceType: string;
  actions: string[];
  name: string;
  namespaceId: string;
  updatedAt: string;
  userOrganizationId: string | undefined;
  version: number;
  createdBy: string;
  description: string;
  id: string;
  appId: string;
}

export interface PublishEntryRequest {
}

export interface RestoreEntryRequest {
}

export interface IDVerificationResponse {
  verification: any;
}

export interface StartVideoSessionResponse {
  expiresAt: string;
  message: string;
  sessionUrl: string;
  startedAt: string;
  videoSessionId: string;
}

export interface GenerateReportRequest {
  format: string;
  period: string;
  reportType: string;
  standard: string;
}

export interface DataExportConfig {
  cleanupInterval: Duration;
  expiryHours: number;
  maxExportSize: number;
  requestPeriod: Duration;
  allowedFormats: string[];
  autoCleanup: boolean;
  defaultFormat: string;
  enabled: boolean;
  includeSections: string[];
  maxRequests: number;
  storagePath: string;
}

export interface ConnectionsResponse {
  connections: SocialAccount | undefined[];
}

export interface CreateProfileRequest {
  metadata: any;
  passwordRequireLower: boolean;
  sessionIpBinding: boolean;
  standards: string[];
  dpoContact: string;
  encryptionInTransit: boolean;
  leastPrivilege: boolean;
  passwordRequireSymbol: boolean;
  passwordRequireUpper: boolean;
  retentionDays: number;
  sessionMaxAge: number;
  appId: string;
  auditLogExport: boolean;
  dataResidency: string;
  mfaRequired: boolean;
  name: string;
  passwordExpiryDays: number;
  passwordRequireNumber: boolean;
  rbacRequired: boolean;
  complianceContact: string;
  detailedAuditTrail: boolean;
  encryptionAtRest: boolean;
  passwordMinLength: number;
  regularAccessReview: boolean;
  sessionIdleTimeout: number;
}

export interface ComplianceProfile {
  dataResidency: string;
  dpoContact: string;
  encryptionInTransit: boolean;
  sessionMaxAge: number;
  standards: string[];
  detailedAuditTrail: boolean;
  mfaRequired: boolean;
  rbacRequired: boolean;
  appId: string;
  passwordMinLength: number;
  passwordRequireLower: boolean;
  passwordRequireNumber: boolean;
  regularAccessReview: boolean;
  auditLogExport: boolean;
  retentionDays: number;
  sessionIpBinding: boolean;
  name: string;
  complianceContact: string;
  id: string;
  status: string;
  encryptionAtRest: boolean;
  leastPrivilege: boolean;
  passwordExpiryDays: number;
  passwordRequireSymbol: boolean;
  passwordRequireUpper: boolean;
  sessionIdleTimeout: number;
  createdAt: string;
  metadata: any;
  updatedAt: string;
}

export interface UnbanUserRequestDTO {
  reason: string;
}

export interface CreatePolicyRequest {
  priority: number;
  resourceType: string;
  actions: string[];
  description: string;
  enabled: boolean;
  expression: string;
  name: string;
  namespaceId: string;
}

export interface MessageResponse {
  message: string;
}

export interface GetABTestResultsRequest {
}

export interface ListAPIKeysRequest {
}

export interface GetDocumentVerificationRequest {
  documentId: string;
}

export interface GetDocumentVerificationResponse {
  message: string;
  rejectionReason: string;
  status: string;
  verifiedAt: string | undefined;
  confidenceScore: number;
  documentId: string;
}

export interface StepUpStatusResponse {
  status: string;
}

export interface ListImpersonationsRequest {
}

export interface InvitationResponse {
  invitation: Invitation | undefined;
  message: string;
}

export interface ProviderDiscoveredResponse {
  found: boolean;
  providerId: string;
  type: string;
}

export interface AuditServiceInterface {
  [key: string]: any;
}

export interface ValidatePolicyResponse {
  complexity: number;
  error: string;
  errors: string[];
  message: string;
  valid: boolean;
  warnings: string[];
}

export interface JumioProvider {
}

export interface RenderTemplate_req {
  variables: any;
  template: string;
}

export interface VerificationsResponse {
  count: number;
  verifications: any;
}

export interface ConsentService {
}

export interface SecurityLevel {
  [key: string]: any;
}

export interface NotificationTemplateListResponse {
  templates: any[];
  total: number;
}

export interface DeleteProviderRequest {
}

export interface AssignRoleRequest {
  roleID: string;
}

export interface RiskContext {
}

export interface SendCodeRequest {
  phone: string;
}

export interface ProviderRegisteredResponse {
  providerId: string;
  status: string;
  type: string;
}

export interface OAuthState {
  app_id: string;
  created_at: string;
  extra_scopes: string[];
  link_user_id: string | undefined;
  provider: string;
  redirect_url: string;
  user_organization_id: string | undefined;
}

export interface VerificationMethod {
  [key: string]: any;
}

export interface Handler {
}

export interface GenerateRecoveryCodesRequest {
  count: number;
  format: string;
}

export interface SetupSecurityQuestionRequest {
  answer: string;
  customText: string;
  questionId: number;
}

export interface SetupSecurityQuestionsRequest {
  questions: SetupSecurityQuestionRequest[];
}

export interface TrustedDevicesConfig {
  default_expiry_days: number;
  enabled: boolean;
  max_devices_per_user: number;
  max_expiry_days: number;
}

export interface GetChallengeStatusRequest {
}

export interface ConnectionResponse {
  connection: SocialAccount | undefined;
}

export interface CallbackResponse {
  user: User | undefined;
  session: Session | undefined;
  token: string;
}

export interface TemplateService {
}

export interface RevisionHandler {
}

export interface VerifyRequest {
}

export interface HandleUpdateSettings_updates {
  [key: string]: any;
}

export interface NotificationStatusResponse {
  status: string;
}

export interface GetRolesRequest {
}

export interface SMSFactorAdapter {
}

export interface ConsentSummary {
  organizationId: string;
  pendingRenewals: number;
  revokedConsents: number;
  userId: string;
  consentsByType: any;
  hasPendingDeletion: boolean;
  hasPendingExport: boolean;
  lastConsentUpdate: string | undefined;
  totalConsents: number;
  expiredConsents: number;
  grantedConsents: number;
}

export interface RunCheckRequest {
  checkType: string;
}

export interface CreateResourceRequest {
  attributes: ResourceAttributeRequest[];
  description: string;
  namespaceId: string;
  type: string;
}

export interface AmountRule {
  max_amount: number;
  min_amount: number;
  org_id: string;
  security_level: string;
  currency: string;
  description: string;
}

export interface ConsentRecord {
  userAgent: string;
  granted: boolean;
  ipAddress: string;
  createdAt: string;
  expiresAt: string | undefined;
  grantedAt: string;
  metadata: Record<string, any>;
  purpose: string;
  revokedAt: string | undefined;
  updatedAt: string;
  userId: string;
  consentType: string;
  id: string;
  organizationId: string;
  version: string;
}

export interface GetOrganizationBySlugRequest {
}

export interface SAMLLoginRequest {
  relayState: string;
}

export interface GetTreeRequest {
}

export interface OrganizationAutoSendConfig {
  role_changed: boolean;
  transfer: boolean;
  deleted: boolean;
  invite: boolean;
  member_added: boolean;
  member_left: boolean;
  member_removed: boolean;
}

export interface RotateAPIKeyResponse {
  message: string;
  api_key: APIKey | undefined;
}

export interface ComplianceChecksResponse {
  checks: any[];
}

export interface ScopeResolver {
}

export interface EvaluationResult {
  can_remember: boolean;
  grace_period_ends_at: string;
  matched_rules: string[];
  requirement_id: string;
  security_level: string;
  allowed_methods: string[];
  challenge_token: string;
  current_level: string;
  expires_at: string;
  metadata: any;
  reason: string;
  required: boolean;
}

export interface SMSConfig {
  provider: string;
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
  code_expiry_minutes: number;
  code_length: number;
  enabled: boolean;
}

export interface AdminUpdateProviderRequest {
  clientSecret: string | undefined;
  enabled: boolean | undefined;
  scopes: string[];
  clientId: string | undefined;
}

export interface CancelFunc {
  [key: string]: any;
}

export interface BanUserRequest {
  app_id: string;
  expires_at: string | undefined;
  reason: string;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface ScopeDefinition {
}

export interface GenerateRecoveryCodesResponse {
  codes: string[];
  count: number;
  generatedAt: string;
  warning: string;
}

export interface NotificationChannels {
  email: boolean;
  slack: boolean;
  webhook: boolean;
}

export interface ConsentDeletionResponse {
  id: string;
  status: string;
}

export interface ImpersonationAuditResponse {
  [key: string]: any;
}

export interface SetUserRoleRequestDTO {
  role: string;
}

export interface PreviewConversionResponse {
  celExpression: string;
  error: string;
  policyName: string;
  resourceId: string;
  resourceType: string;
  success: boolean;
}

export interface TemplateDefault {
}

export interface SendVerificationCodeRequest {
  method: string;
  sessionId: string;
  target: string;
}

export interface ConsentCookieResponse {
  preferences: any;
}

export interface ImpersonationSession {
  [key: string]: any;
}

export interface Role {
  [key: string]: any;
}

export interface InstantiateTemplateRequest {
  name: string;
  namespaceId: string;
  parameters: any;
  priority: number;
  resourceType: string;
  actions: string[];
  description: string;
  enabled: boolean;
}

export interface GetRevisionRequest {
}

export interface Status {
}

export interface DocumentVerificationConfig {
  encryptionKey: string;
  provider: string;
  requireSelfie: boolean;
  retentionPeriod: Duration;
  storageProvider: string;
  minConfidenceScore: number;
  requireBothSides: boolean;
  requireManualReview: boolean;
  storagePath: string;
  acceptedDocuments: string[];
  enabled: boolean;
  encryptAtRest: boolean;
}

export interface ScheduleVideoSessionResponse {
  instructions: string;
  joinUrl: string;
  message: string;
  scheduledAt: string;
  videoSessionId: string;
}

export interface ComplianceDashboardResponse {
  metrics: any;
}

export interface VerifyChallengeRequest {
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
  challengeId: string;
  code: string;
}

export interface TokenRevocationRequest {
  client_id: string;
  client_secret: string;
  token: string;
  token_type_hint: string;
}

export interface TestCase {
  resource: any;
  action: string;
  expected: boolean;
  name: string;
  principal: any;
  request: any;
}

export interface WebhookResponse {
  status: string;
  received: boolean;
}

export interface UpdateAPIKeyRequest {
  name: string | undefined;
  permissions: any;
  rate_limit: number | undefined;
  scopes: string[];
  allowed_ips: string[];
  description: string | undefined;
  metadata: any;
}

export interface OIDCLoginResponse {
  providerId: string;
  state: string;
  authUrl: string;
  nonce: string;
}

export interface UpdateSecretRequest {
  tags: string[];
  value: any;
  description: string;
  metadata: any;
}

export interface ProvidersAppResponse {
  appId: string;
  providers: string[];
}

export interface ReviewDocumentRequest {
  approved: boolean;
  documentId: string;
  notes: string;
  rejectionReason: string;
}

export interface RecoverySession {
}

export interface RunCheck_req {
  checkType: string;
}

export interface ScopeInfo {
}

export interface SAMLLoginResponse {
  redirectUrl: string;
  requestId: string;
  providerId: string;
}

export interface CallbackResult {
}

export interface JSONBMap {
  [key: string]: any;
}

export interface UserVerificationStatusResponse {
  status: UserVerificationStatus | undefined;
}

export interface VerifyRequest2FA {
  user_id: string;
  code: string;
  device_id: string;
  remember_device: boolean;
}

export interface OTPSentResponse {
  status: string;
  code: string;
}

export interface GetSecurityQuestionsRequest {
  sessionId: string;
}

export interface RevokeTokenService {
}

export interface TokenResponse {
  access_token: string;
  expires_in: number;
  id_token: string;
  refresh_token: string;
  scope: string;
  token_type: string;
}

export interface ListUsersRequestDTO {
}

export interface PolicyStats {
  policyName: string;
  allowCount: number;
  avgLatencyMs: number;
  denyCount: number;
  evaluationCount: number;
  policyId: string;
}

export interface ReverifyRequest {
  reason: string;
}

export interface RequestReverification_req {
  reason: string;
}

export interface NotificationPreviewResponse {
  body: string;
  subject: string;
}

export interface SignInRequest {
  [key: string]: any;
}

export interface ConsentExpiryConfig {
  expireCheckInterval: Duration;
  renewalReminderDays: number;
  requireReConsent: boolean;
  allowRenewal: boolean;
  autoExpireCheck: boolean;
  defaultValidityDays: number;
  enabled: boolean;
}

export interface Logger {
  [key: string]: any;
}

export interface NotificationListResponse {
  notifications: any[];
  total: number;
}

export interface TemplatesResponse {
  count: number;
  templates: any;
}

export interface AutoSendConfig {
  auth: AuthAutoSendConfig;
  organization: OrganizationAutoSendConfig;
  session: SessionAutoSendConfig;
  account: AccountAutoSendConfig;
}

export interface CreateDPARequest {
  metadata: any;
  agreementType: string;
  effectiveDate: string;
  expiryDate: string | undefined;
  signedByEmail: string;
  signedByName: string;
  signedByTitle: string;
  version: string;
  content: string;
}

export interface DataDeletionConfig {
  requireAdminApproval: boolean;
  archivePath: string;
  retentionExemptions: string[];
  allowPartialDeletion: boolean;
  archiveBeforeDeletion: boolean;
  autoProcessAfterGrace: boolean;
  enabled: boolean;
  gracePeriodDays: number;
  notifyBeforeDeletion: boolean;
  preserveLegalData: boolean;
}

export interface VerifyImpersonationRequest {
}

export interface VerifyAPIKeyResponse {
  [key: string]: any;
}

export interface IDVerificationErrorResponse {
  error: string;
}

export interface ProviderSessionRequest {
}

export interface GetEffectivePermissionsRequest {
}

export interface BeginLoginResponse {
  challenge: string;
  options: any;
  timeout: Duration;
}

export interface ValidateContentTypeRequest {
}

export interface BlockUserRequest {
  reason: string;
}

export interface DeleteTemplateRequest {
}

export interface ListChecksFilter {
  appId: string | undefined;
  checkType: string | undefined;
  profileId: string | undefined;
  sinceBefore: string | undefined;
  status: string | undefined;
}

export interface GetByPathRequest {
}

export interface ContentFieldService {
  [key: string]: any;
}

export interface SendWithTemplateRequest {
  templateKey: string;
  type: NotificationType;
  variables: any;
  appId: string;
  language: string;
  metadata: any;
  recipient: string;
}

export interface CompleteRecoveryRequest {
  sessionId: string;
}

export interface ResolveViolationRequest {
  notes: string;
  resolution: string;
}

export interface RegistrationService {
}

export interface ProviderDetailResponse {
  samlEntryPoint: string;
  samlIssuer: string;
  type: string;
  updatedAt: string;
  createdAt: string;
  domain: string;
  hasSamlCert: boolean;
  oidcRedirectURI: string;
  attributeMapping: any;
  oidcClientID: string;
  oidcIssuer: string;
  providerId: string;
}

export interface Team {
  [key: string]: any;
}

export interface TrustedContactsConfig {
  allowEmailContacts: boolean;
  allowPhoneContacts: boolean;
  enabled: boolean;
  maxNotificationsPerDay: number;
  maximumContacts: number;
  minimumContacts: number;
  verificationExpiry: Duration;
  cooldownPeriod: Duration;
  requireVerification: boolean;
  requiredToRecover: number;
}

export interface TOTPConfig {
  window_size: number;
  algorithm: string;
  digits: number;
  enabled: boolean;
  issuer: string;
  period: number;
}

export interface GetPasskeyRequest {
}

export interface ConsentRequest {
  client_id: string;
  code_challenge: string;
  code_challenge_method: string;
  redirect_uri: string;
  response_type: string;
  scope: string;
  state: string;
  action: string;
}

export interface CreateContentTypeRequest {
  [key: string]: any;
}

export interface AppsListResponse {
  [key: string]: any;
}

export interface ImpersonationListResponse {
  [key: string]: any;
}

export interface ListSessionsRequest {
  app_id: string;
  limit: number;
  page: number;
  user_id: string | undefined;
  user_organization_id: string | undefined;
}

export interface ActionsListResponse {
  actions: ActionResponse | undefined[];
  totalCount: number;
}

export interface VerifyRecoveryCodeResponse {
  message: string;
  remainingCodes: number;
  valid: boolean;
}

export interface GetRecoveryStatsRequest {
  endDate: string;
  organizationId: string;
  startDate: string;
}

export interface UploadDocumentRequest {
  backImage: string;
  documentType: string;
  frontImage: string;
  selfie: string;
  sessionId: string;
}

export interface ListRecoverySessionsResponse {
  page: number;
  pageSize: number;
  sessions: RecoverySessionInfo[];
  totalCount: number;
}

export interface RequirementsResponse {
  count: number;
  requirements: any;
}

export interface MFAPolicyResponse {
  id: string;
  organizationId: string | undefined;
  requiredFactorCount: number;
  allowedFactorTypes: string[];
  appId: string;
  enabled: boolean;
  gracePeriodDays: number;
}

export interface NotificationWebhookResponse {
  status: string;
}

export interface BackupAuthStatsResponse {
  stats: any;
}

export interface AutomatedChecksConfig {
  accessReview: boolean;
  checkInterval: Duration;
  dataRetention: boolean;
  enabled: boolean;
  inactiveUsers: boolean;
  mfaCoverage: boolean;
  suspiciousActivity: boolean;
  passwordPolicy: boolean;
  sessionPolicy: boolean;
}

export interface MockEmailService {
  [key: string]: any;
}

export interface ChallengeResponse {
  factorsRequired: number;
  sessionId: string;
  availableFactors: FactorInfo[];
  challengeId: string;
  expiresAt: string;
}

export interface DeleteTeamRequest {
}

export interface UpdatePasskeyRequest {
  name: string;
}

export interface ListPasskeysResponse {
  count: number;
  passkeys: PasskeyInfo[];
}

export interface AutoCleanupConfig {
  enabled: boolean;
  interval: Duration;
}

export interface DocumentVerificationRequest {
}

export interface SMSVerificationConfig {
  codeLength: number;
  cooldownPeriod: Duration;
  enabled: boolean;
  maxAttempts: number;
  maxSmsPerDay: number;
  messageTemplate: string;
  provider: string;
  codeExpiry: Duration;
}

export interface EmailVerificationConfig {
  emailTemplate: string;
  enabled: boolean;
  fromAddress: string;
  fromName: string;
  maxAttempts: number;
  requireEmailProof: boolean;
  codeExpiry: Duration;
  codeLength: number;
}

export interface CreateTraining_req {
  standard: string;
  trainingType: string;
  userId: string;
}

export interface AddTeamMember_req {
  member_id: string;
  role: string;
}

export interface ClientsListResponse {
  clients: ClientSummary[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface ClientDetailsResponse {
  trustedClient: boolean;
  allowedScopes: string[];
  createdAt: string;
  grantTypes: string[];
  logoURI: string;
  tosURI: string;
  name: string;
  organizationID: string;
  policyURI: string;
  updatedAt: string;
  applicationType: string;
  clientID: string;
  requireConsent: boolean;
  responseTypes: string[];
  contacts: string[];
  isOrgLevel: boolean;
  postLogoutRedirectURIs: string[];
  redirectURIs: string[];
  requirePKCE: boolean;
  tokenEndpointAuthMethod: string;
}

export interface Service {
}

export interface NotificationsResponse {
  count: number;
  notifications: any;
}

export interface RotateAPIKeyRequest {
}

export interface GetRecoveryConfigResponse {
  riskScoreThreshold: number;
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
}

export interface FactorVerificationRequest {
  data: any;
  factorId: string;
  code: string;
}

export interface RevokeResponse {
  revokedCount: number;
  status: string;
}

export interface MetadataResponse {
  metadata: string;
}

export interface MockUserRepository {
  [key: string]: any;
}

export interface BulkDeleteRequest {
  ids: string[];
}

export interface ResendResponse {
  status: string;
}

export interface BackupAuthDocumentResponse {
  id: string;
}

export interface DataExportRequestInput {
  format: string;
  includeSections: string[];
}

export interface JWKS {
  keys: JWK[];
}

export interface AddCustomPermission_req {
  category: string;
  description: string;
  name: string;
}

export interface ComplianceStandard {
  [key: string]: any;
}

export interface TemplateResponse {
  category: string;
  description: string;
  examples: string[];
  expression: string;
  id: string;
  name: string;
  parameters: TemplateParameter[];
}

export interface AMLMatch {
}

export interface AuditConfig {
  retentionDays: number;
  archiveInterval: Duration;
  archiveOldLogs: boolean;
  enabled: boolean;
  immutableLogs: boolean;
  logAllAttempts: boolean;
  logSuccessful: boolean;
  logUserAgent: boolean;
  logDeviceInfo: boolean;
  logFailed: boolean;
  logIpAddress: boolean;
}

export interface ImpersonationErrorResponse {
  error: string;
}

export interface Authsome {
  [key: string]: any;
}

export interface OAuthTokenRepository {
  [key: string]: any;
}

export interface RecoveryMethod {
  [key: string]: any;
}

export interface Plugin {
}

export interface ComplianceTemplateResponse {
  standard: string;
}

export interface ResourcesListResponse {
  resources: ResourceResponse | undefined[];
  totalCount: number;
}

export interface WebhookPayload {
}

export interface CreateProvider_req {
  config: any;
  isDefault: boolean;
  organizationId?: string | undefined;
  providerName: string;
  providerType: string;
}

export interface TOTPSecret {
}

export interface RateLimitConfig {
  enabled: boolean;
  window: Duration;
}

export interface MockUserService {
  [key: string]: any;
}

export interface RiskEngine {
}

export interface RevokeAllRequest {
  includeCurrentSession: boolean;
}

export interface EnableResponse {
  status: string;
  totp_uri: string;
}

export interface CompleteVideoSessionRequest {
  livenessPassed: boolean;
  livenessScore: number;
  notes: string;
  verificationResult: string;
  videoSessionId: string;
}

export interface StepUpAuditLog {
  user_agent: string;
  event_data: any;
  event_type: string;
  org_id: string;
  user_id: string;
  created_at: string;
  id: string;
  ip: string;
  severity: string;
}

export interface CookieConsentConfig {
  allowAnonymous: boolean;
  bannerVersion: string;
  categories: string[];
  defaultStyle: string;
  enabled: boolean;
  requireExplicit: boolean;
  validityPeriod: Duration;
}

export interface DataProcessingAgreement {
  content: string;
  digitalSignature: string;
  effectiveDate: string;
  agreementType: string;
  id: string;
  signedBy: string;
  signedByEmail: string;
  status: string;
  updatedAt: string;
  createdAt: string;
  expiryDate: string | undefined;
  organizationId: string;
  signedByTitle: string;
  version: string;
  ipAddress: string;
  metadata: Record<string, any>;
  signedByName: string;
}

export interface EncryptionService {
}

export interface OAuthClientRepository {
  [key: string]: any;
}

export interface Client {
  [key: string]: any;
}

export interface GetNotificationRequest {
}

export interface VerifySecurityAnswersResponse {
  message: string;
  requiredAnswers: number;
  valid: boolean;
  attemptsLeft: number;
  correctAnswers: number;
}

export interface ConsentAuditConfig {
  signLogs: boolean;
  enabled: boolean;
  exportFormat: string;
  immutable: boolean;
  logUserAgent: boolean;
  retentionDays: number;
  archiveInterval: Duration;
  archiveOldLogs: boolean;
  logAllChanges: boolean;
  logIpAddress: boolean;
}

