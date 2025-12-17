// Auto-generated TypeScript types

export interface UserInfoResponse {
  family_name: string;
  locale: string;
  nickname: string;
  phone_number: string;
  sub: string;
  birthdate: string;
  given_name: string;
  profile: string;
  zoneinfo: string;
  name: string;
  phone_number_verified: boolean;
  picture: string;
  preferred_username: string;
  updated_at: number;
  website: string;
  email: string;
  email_verified: boolean;
  gender: string;
  middle_name: string;
}

export interface CreateVerificationSession_req {
  requiredChecks: string[];
  successUrl: string;
  cancelUrl: string;
  config: any;
  metadata: any;
  provider: string;
}

export interface ConnectionsResponse {
  connections: any | undefined[];
}

export interface RecoveryConfiguration {
}

export interface StepUpRequirementResponse {
  id: string;
}

export interface DataProcessingAgreement {
  agreementType: string;
  content: string;
  expiryDate: string | undefined;
  id: string;
  metadata: Record<string, any>;
  organizationId: string;
  updatedAt: string;
  effectiveDate: string;
  ipAddress: string;
  version: string;
  createdAt: string;
  signedBy: string;
  signedByName: string;
  status: string;
  digitalSignature: string;
  signedByEmail: string;
  signedByTitle: string;
}

export interface MigrateRBACRequest {
  dryRun: boolean;
  keepRbacPolicies: boolean;
  namespaceId: string;
  validateEquivalence: boolean;
}

export interface MigrationStatusResponse {
  migratedCount: number;
  progress: number;
  status: string;
  totalPolicies: number;
  appId: string;
  completedAt: string | undefined;
  environmentId: string;
  errors: string[];
  startedAt: string;
  userOrganizationId: string | undefined;
  validationPassed: boolean;
  failedCount: number;
}

export interface ClientRegistrationRequest {
  require_consent: boolean;
  require_pkce: boolean;
  token_endpoint_auth_method: string;
  logo_uri: string;
  response_types: string[];
  scope: string;
  contacts: string[];
  policy_uri: string;
  post_logout_redirect_uris: string[];
  tos_uri: string;
  application_type: string;
  grant_types: string[];
  redirect_uris: string[];
  trusted_client: boolean;
  client_name: string;
}

export interface BulkRequest {
  ids: string[];
}

export interface Device {
  userAgent?: string;
  id: string;
  userId: string;
  name?: string;
  type?: string;
  lastUsedAt: string;
  ipAddress?: string;
}

export interface PreviewTemplate_req {
  variables: any;
}

export interface ConsentsResponse {
  consents: any;
  count: number;
}

export interface CallbackResponse {
  session: any | undefined;
  token: string;
  user: any | undefined;
}

export interface TwoFARequiredResponse {
  device_id: string;
  require_twofa: boolean;
  user: any | undefined;
}

export interface CreateUser_reqBody {
  email: string;
  email_verified: boolean;
  metadata?: any;
  name?: string;
  password?: string;
  role?: string;
  username?: string;
}

export interface RevokeTokenService {
}

export interface CallbackDataResponse {
  action: string;
  isNewUser: boolean;
  user: any | undefined;
}

export interface SetUserRoleRequest {
  user_organization_id: string | undefined;
  app_id: string;
  role: string;
  user_id: string;
}

export interface ConsentManager {
}

export interface ProviderSessionRequest {
}

export interface CheckSubResult {
}

export interface RequestTrustedContactVerificationRequest {
  contactId: string;
  sessionId: string;
}

export interface ListUsersRequest {
  role: string;
  search: string;
  status: string;
  user_organization_id: string | undefined;
  app_id: string;
  limit: number;
  page: number;
}

export interface ComplianceReportResponse {
  id: string;
}

export interface ComplianceStatusDetailsResponse {
  status: string;
}

export interface MFAConfigResponse {
  allowed_factor_types: string[];
  enabled: boolean;
  required_factor_count: number;
}

export interface SMSProviderConfig {
  from: string;
  provider: string;
  config: any;
}

export interface ComplianceEvidencesResponse {
  evidence: any[];
}

export interface PolicyStats {
  denyCount: number;
  evaluationCount: number;
  policyId: string;
  policyName: string;
  allowCount: number;
  avgLatencyMs: number;
}

export interface Webhook {
  organizationId: string;
  url: string;
  events: string[];
  secret: string;
  enabled: boolean;
  createdAt: string;
  id: string;
}

export interface AdminBlockUser_req {
  reason: string;
}

export interface VerificationSessionResponse {
  session: any | undefined;
}

export interface JumioConfig {
  enableExtraction: boolean;
  enableLiveness: boolean;
  enabled: boolean;
  enabledCountries: string[];
  enabledDocumentTypes: string[];
  presetId: string;
  apiToken: string;
  callbackUrl: string;
  dataCenter: string;
  enableAMLScreening: boolean;
  verificationType: string;
  apiSecret: string;
}

export interface RotateAPIKeyResponse {
  api_key: any | undefined;
  message: string;
}

export interface TemplatesListResponse {
  categories: string[];
  templates: TemplateResponse | undefined[];
  totalCount: number;
}

export interface Rollback_req {
  reason: string;
}

export interface ImpersonationMiddleware {
}

export interface BackupAuthStatsResponse {
  stats: any;
}

export interface sessionStats {
}

export interface ResourceTypeStats {
  allowRate: number;
  avgLatencyMs: number;
  evaluationCount: number;
  resourceType: string;
}

export interface TemplateDefault {
}

export interface BackupAuthContactsResponse {
  contacts: any[];
}

export interface MigrationErrorResponse {
  policyIndex: number;
  resource: string;
  subject: string;
  error: string;
}

export interface ListTrustedDevicesResponse {
  count: number;
  devices: TrustedDevice[];
}

export interface RateLimit {
  max_requests: number;
  window: any;
}

export interface DocumentCheckConfig {
  enabled: boolean;
  extractData: boolean;
  validateDataConsistency: boolean;
  validateExpiry: boolean;
}

export interface TOTPFactorAdapter {
}

export interface ConsentNotificationsConfig {
  notifyOnGrant: boolean;
  channels: string[];
  enabled: boolean;
  notifyDeletionApproved: boolean;
  notifyDeletionComplete: boolean;
  notifyDpoEmail: string;
  notifyExportReady: boolean;
  notifyOnExpiry: boolean;
  notifyOnRevoke: boolean;
}

export interface VerifyRecoveryCodeRequest {
  sessionId: string;
  code: string;
}

export interface ComplianceChecksResponse {
  checks: any[];
}

export interface RevokeAllResponse {
  status: string;
  revokedCount: number;
}

export interface UserAdapter {
}

export interface RiskAssessment {
  factors: string[];
  level: string;
  metadata: any;
  recommended: string[];
  score: number;
}

export interface GetRecoveryConfigResponse {
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
  riskScoreThreshold: number;
}

export interface RedisChallengeStore {
}

export interface Config {
  accessExpirySeconds: number;
  includeAppIDClaim: boolean;
  issuer: string;
  refreshExpirySeconds: number;
  signingAlgorithm: string;
}

export interface OnfidoProvider {
}

export interface navItem {
}

export interface BackupAuthVideoResponse {
  session_id: string;
}

export interface RiskFactor {
}

export interface TwoFAStatusDetailResponse {
  enabled: boolean;
  method: string;
  trusted: boolean;
}

export interface ApproveRecoveryResponse {
  approved: boolean;
  approvedAt: string;
  message: string;
  sessionId: string;
}

export interface BanUser_reqBody {
  reason: string;
  expires_at?: string | undefined;
}

export interface ResourcesListResponse {
  resources: ResourceResponse | undefined[];
  totalCount: number;
}

export interface ResendResponse {
  status: string;
}

export interface MockStateStore {
}

export interface CreateAPIKeyResponse {
  api_key: any | undefined;
  message: string;
}

export interface CreateEvidenceRequest {
  controlId: string;
  description: string;
  evidenceType: string;
  fileUrl: string;
  standard: string;
  title: string;
}

export interface WebhookPayload {
}

export interface RunCheck_req {
  checkType: string;
}

export interface VerifyChallengeRequest {
  code: string;
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
  challengeId: string;
}

export interface ConsentPolicy {
  id: string;
  name: string;
  publishedAt: string | undefined;
  active: boolean;
  metadata: Record<string, any>;
  required: boolean;
  validityPeriod: number | undefined;
  version: string;
  consentType: string;
  createdBy: string;
  description: string;
  organizationId: string;
  renewable: boolean;
  updatedAt: string;
  content: string;
  createdAt: string;
}

export interface CreateConsentRequest {
  userId: string;
  version: string;
  consentType: string;
  expiresIn: number | undefined;
  granted: boolean;
  metadata: any;
  purpose: string;
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

export interface HealthCheckResponse {
  providersStatus: any;
  version: string;
  enabledMethods: string[];
  healthy: boolean;
  message: string;
}

export interface NotificationsConfig {
  notifyOnRecoveryStart: boolean;
  securityOfficerEmail: string;
  channels: string[];
  enabled: boolean;
  notifyAdminOnHighRisk: boolean;
  notifyAdminOnReviewNeeded: boolean;
  notifyOnRecoveryComplete: boolean;
  notifyOnRecoveryFailed: boolean;
}

export interface DeleteFactorRequest {
}

export interface WebAuthnConfig {
  rp_origins: string[];
  timeout: number;
  attestation_preference: string;
  authenticator_selection: any;
  enabled: boolean;
  rp_display_name: string;
  rp_id: string;
}

export interface MFABypassResponse {
  expiresAt: string;
  id: string;
  reason: string;
  userId: string;
}

export interface ReverifyRequest {
  reason: string;
}

export interface CreatePolicy_req {
  content: string;
  policyType: string;
  standard: string;
  title: string;
  version: string;
}

export interface AdminBypassRequest {
  duration: number;
  reason: string;
  userId: string;
}

export interface NotificationTemplateListResponse {
  total: number;
  templates: any[];
}

export interface ContinueRecoveryRequest {
  method: string;
  sessionId: string;
}

export interface GenerateBackupCodes_body {
  user_id: string;
  count: number;
}

export interface NotificationWebhookResponse {
  status: string;
}

export interface ComplianceStatusResponse {
  status: string;
}

export interface EvaluateRequest {
  amount: number;
  currency: string;
  metadata: any;
  method: string;
  resource_type: string;
  route: string;
  action: string;
}

export interface MultiSessionErrorResponse {
  error: string;
}

export interface ResetUserMFARequest {
  reason: string;
}

export interface ConfigSourceConfig {
  autoRefresh: boolean;
  enabled: boolean;
  prefix: string;
  priority: number;
  refreshInterval: any;
}

export interface ContinueRecoveryResponse {
  totalSteps: number;
  currentStep: number;
  data: any;
  expiresAt: string;
  instructions: string;
  method: string;
  sessionId: string;
}

export interface MultiSessionDeleteResponse {
  status: string;
}

export interface GenerateRecoveryCodesRequest {
  count: number;
  format: string;
}

export interface NoOpDocumentProvider {
}

export interface ConsentRecord {
  createdAt: string;
  id: string;
  organizationId: string;
  purpose: string;
  updatedAt: string;
  userAgent: string;
  version: string;
  consentType: string;
  grantedAt: string;
  ipAddress: string;
  userId: string;
  granted: boolean;
  metadata: Record<string, any>;
  revokedAt: string | undefined;
  expiresAt: string | undefined;
}

export interface NotificationTemplateResponse {
  template: any;
}

export interface PolicyResponse {
  environmentId: string;
  version: number;
  createdAt: string;
  createdBy: string;
  enabled: boolean;
  id: string;
  name: string;
  resourceType: string;
  updatedAt: string;
  userOrganizationId: string | undefined;
  appId: string;
  priority: number;
  actions: string[];
  expression: string;
  namespaceId: string;
  description: string;
}

export interface ResetUserMFAResponse {
  devicesRevoked: number;
  factorsReset: number;
  message: string;
  success: boolean;
}

export interface TemplateService {
}

export interface RemoveTrustedContactRequest {
  contactId: string;
}

export interface MembersResponse {
  members: any | undefined[];
  total: number;
}

export interface UpdatePasskeyResponse {
  name: string;
  passkeyId: string;
  updatedAt: string;
}

export interface MigrationResponse {
  startedAt: string;
  status: string;
  message: string;
  migrationId: string;
}

export interface UpdateFactorRequest {
  status: string | undefined;
  metadata: any;
  name: string | undefined;
  priority: string | undefined;
}

export interface StepUpRememberedDevice {
  device_id: string;
  expires_at: string;
  id: string;
  last_used_at: string;
  org_id: string;
  remembered_at: string;
  security_level: string;
  device_name: string;
  ip: string;
  user_agent: string;
  user_id: string;
  created_at: string;
}

export interface SAMLLoginResponse {
  requestId: string;
  providerId: string;
  redirectUrl: string;
}

export interface NoOpSMSProvider {
}

export interface DeviceInfo {
}

export interface ActionsListResponse {
  actions: ActionResponse | undefined[];
  totalCount: number;
}

export interface Status {
}

export interface ResendRequest {
  email: string;
}

export interface userServiceAdapter {
}

export interface ConsentCookieResponse {
  preferences: any;
}

export interface CreateProfileFromTemplateRequest {
  standard: string;
}

export interface SendCodeResponse {
  dev_code: string;
  status: string;
}

export interface LimitResult {
}

export interface Enable_body {
  method: string;
  user_id: string;
}

export interface AddTeamMember_req {
  member_id: string;
  role: string;
}

export interface RecoverySessionInfo {
  userEmail: string;
  completedAt: string | undefined;
  expiresAt: string;
  id: string;
  method: string;
  requiresReview: boolean;
  status: string;
  totalSteps: number;
  userId: string;
  createdAt: string;
  currentStep: number;
  riskScore: number;
}

export interface CreateResourceRequest {
  description: string;
  namespaceId: string;
  type: string;
  attributes: ResourceAttributeRequest[];
}

export interface GetChallengeStatusResponse {
  challengeId: string;
  factorsRequired: number;
  factorsVerified: number;
  maxAttempts: number;
  status: string;
  attempts: number;
  availableFactors: FactorInfo[];
}

export interface TestSendTemplate_req {
  recipient: string;
  variables: any;
}

export interface DataDeletionRequestInput {
  deleteSections: string[];
  reason: string;
}

export interface ConsentRecordResponse {
  id: string;
}

export interface ListViolationsFilter {
  violationType: string | undefined;
  appId: string | undefined;
  profileId: string | undefined;
  severity: string | undefined;
  status: string | undefined;
  userId: string | undefined;
}

export interface CompleteVideoSessionResponse {
  result: string;
  videoSessionId: string;
  completedAt: string;
  message: string;
}

export interface UnbanUser_reqBody {
  reason?: string;
}

export interface FinishRegisterRequest {
  name: string;
  response: any;
  userId: string;
}

export interface NamespaceResponse {
  description: string;
  environmentId: string;
  name: string;
  policyCount: number;
  resourceCount: number;
  appId: string;
  createdAt: string;
  id: string;
  inheritPlatform: boolean;
  templateId: string | undefined;
  updatedAt: string;
  userOrganizationId: string | undefined;
  actionCount: number;
}

export interface RegisterProviderRequest {
  providerId: string;
  samlIssuer: string;
  type: string;
  oidcClientID: string;
  oidcClientSecret: string;
  samlCert: string;
  samlEntryPoint: string;
  attributeMapping: any;
  domain: string;
  oidcIssuer: string;
  oidcRedirectURI: string;
}

export interface ImpersonationContext {
  impersonation_id: string | undefined;
  impersonator_id: string | undefined;
  indicator_message: string;
  is_impersonating: boolean;
  target_user_id: string | undefined;
}

export interface ListChecksFilter {
  appId: string | undefined;
  checkType: string | undefined;
  profileId: string | undefined;
  sinceBefore: string | undefined;
  status: string | undefined;
}

export interface ComplianceStatus {
  score: number;
  standard: string;
  violations: number;
  appId: string;
  checksPassed: number;
  overallStatus: string;
  profileId: string;
  checksFailed: number;
  checksWarning: number;
  lastChecked: string;
  nextAudit: string;
}

export interface UpdateNamespaceRequest {
  description: string;
  inheritPlatform: boolean | undefined;
  name: string;
}

export interface CompleteRecoveryResponse {
  token: string;
  completedAt: string;
  message: string;
  sessionId: string;
  status: string;
}

export interface VideoVerificationConfig {
  minScheduleAdvance: any;
  requireLivenessCheck: boolean;
  requireScheduling: boolean;
  provider: string;
  recordSessions: boolean;
  recordingRetention: any;
  requireAdminReview: boolean;
  sessionDuration: any;
  enabled: boolean;
  livenessThreshold: number;
}

export interface BeginLoginRequest {
  userId: string;
  userVerification: string;
}

export interface ChallengeSession {
}

export interface TwoFAErrorResponse {
  error: string;
}

export interface CreateSessionHTTPRequest {
  cancelUrl: string;
  config: any;
  metadata: any;
  provider: string;
  requiredChecks: string[];
  successUrl: string;
}

export interface UpdatePolicyRequest {
  metadata: any;
  name: string;
  renewable: boolean | undefined;
  required: boolean | undefined;
  validityPeriod: number | undefined;
  active: boolean | undefined;
  content: string;
  description: string;
}

export interface EndImpersonation_reqBody {
  impersonation_id: string;
  reason?: string;
}

export interface ListRecoverySessionsResponse {
  page: number;
  pageSize: number;
  sessions: RecoverySessionInfo[];
  totalCount: number;
}

export interface ListReportsFilter {
  status: string | undefined;
  appId: string | undefined;
  format: string | undefined;
  profileId: string | undefined;
  reportType: string | undefined;
  standard: string | undefined;
}

export interface OIDCState {
}

export interface NotificationPreviewResponse {
  body: string;
  subject: string;
}

export interface FactorsResponse {
  count: number;
  factors: any;
}

export interface NotificationResponse {
  notification: any;
}

export interface GetRecoveryStatsResponse {
  adminReviewsRequired: number;
  failedRecoveries: number;
  methodStats: any;
  pendingRecoveries: number;
  totalAttempts: number;
  averageRiskScore: number;
  highRiskAttempts: number;
  successRate: number;
  successfulRecoveries: number;
}

export interface TwoFASendOTPResponse {
  code: string;
  status: string;
}

export interface TemplatesResponse {
  count: number;
  templates: any;
}

export interface DocumentVerificationConfig {
  encryptAtRest: boolean;
  retentionPeriod: any;
  storageProvider: string;
  acceptedDocuments: string[];
  encryptionKey: string;
  minConfidenceScore: number;
  provider: string;
  requireBothSides: boolean;
  requireManualReview: boolean;
  requireSelfie: boolean;
  storagePath: string;
  enabled: boolean;
}

export interface StepUpAuditLog {
  org_id: string;
  severity: string;
  user_agent: string;
  event_type: string;
  user_id: string;
  created_at: string;
  event_data: any;
  id: string;
  ip: string;
}

export interface ClientAuthenticator {
}

export interface ComplianceReportFileResponse {
  content_type: string;
  data: number[];
}

export interface MockOrganizationUIExtension {
}

export interface ListFactorsResponse {
  count: number;
  factors: Factor[];
}

export interface RevokeTrustedDeviceRequest {
}

export interface TokenIntrospectionRequest {
  token: string;
  token_type_hint: string;
  client_id: string;
  client_secret: string;
}

export interface ResourceRule {
  action: string;
  description: string;
  org_id: string;
  resource_type: string;
  security_level: string;
  sensitivity: string;
}

export interface ChallengeStatusResponse {
  completedAt: string | undefined;
  expiresAt: string;
  factorsRemaining: number;
  factorsRequired: number;
  factorsVerified: number;
  sessionId: string;
  status: string;
}

export interface Session {
  expiresAt: string;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
  id: string;
  userId: string;
  token: string;
}

export interface BlockUserRequest {
  reason: string;
}

export interface CookieConsentRequest {
  bannerVersion: string;
  essential: boolean;
  functional: boolean;
  marketing: boolean;
  personalization: boolean;
  sessionId: string;
  thirdParty: boolean;
  analytics: boolean;
}

export interface CallbackResult {
}

export interface FactorInfo {
  metadata: any;
  name: string;
  type: string;
  factorId: string;
}

export interface VerifyTrustedContactRequest {
  token: string;
}

export interface SessionDetailResponse {
  device: any;
  session: any;
}

export interface IDTokenClaims {
  auth_time: number;
  email: string;
  family_name: string;
  nonce: string;
  email_verified: boolean;
  given_name: string;
  name: string;
  preferred_username: string;
  session_state: string;
}

export interface ListSessionsResponse {
  limit: number;
  page: number;
  sessions: any | undefined[];
  total: number;
  total_pages: number;
}

export interface CompleteTraining_req {
  score: number;
}

export interface SessionStats {
}

export interface TestCase {
  name: string;
  principal: any;
  request: any;
  resource: any;
  action: string;
  expected: boolean;
}

export interface ProviderSession {
}

export interface MockSocialAccountRepository {
}

export interface BanUserRequest {
  expires_at: string | undefined;
  reason: string;
  user_id: string;
  user_organization_id: string | undefined;
  app_id: string;
}

export interface UpdatePasskeyRequest {
  name: string;
}

export interface StartImpersonation_reqBody {
  duration_minutes?: number;
  reason: string;
  target_user_id: string;
  ticket_number?: string;
}

export interface UpdateRecoveryConfigRequest {
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
  riskScoreThreshold: number;
}

export interface ApproveRecoveryRequest {
  notes: string;
  sessionId: string;
}

export interface BackupAuthCodesResponse {
  codes: string[];
}

export interface MockAuditService {
}

export interface CreateEvidence_req {
  standard: string;
  title: string;
  controlId: string;
  description: string;
  evidenceType: string;
  fileUrl: string;
}

export interface RevokeAll_body {
  includeCurrentSession: boolean;
}

export interface TrustDeviceRequest {
  deviceId: string;
  metadata: any;
  name: string;
}

export interface DiscoveryResponse {
  registration_endpoint: string;
  request_uri_parameter_supported: boolean;
  authorization_endpoint: string;
  claims_supported: string[];
  introspection_endpoint: string;
  subject_types_supported: string[];
  token_endpoint: string;
  token_endpoint_auth_methods_supported: string[];
  userinfo_endpoint: string;
  code_challenge_methods_supported: string[];
  grant_types_supported: string[];
  issuer: string;
  require_request_uri_registration: boolean;
  response_modes_supported: string[];
  revocation_endpoint: string;
  scopes_supported: string[];
  request_parameter_supported: boolean;
  response_types_supported: string[];
  revocation_endpoint_auth_methods_supported: string[];
  claims_parameter_supported: boolean;
  id_token_signing_alg_values_supported: string[];
  introspection_endpoint_auth_methods_supported: string[];
  jwks_uri: string;
}

export interface Middleware {
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

export interface VerifyTrustedContactResponse {
  contactId: string;
  message: string;
  verified: boolean;
  verifiedAt: string;
}

export interface AuditEvent {
}

export interface VerificationRequest {
  rememberDevice: boolean;
  challengeId: string;
  code: string;
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
}

export interface BackupCodesConfig {
  allow_reuse: boolean;
  count: number;
  enabled: boolean;
  format: string;
  length: number;
}

export interface SendResponse {
  devToken: string;
  status: string;
}

export interface Status_body {
  device_id: string;
  user_id: string;
}

export interface SendWithTemplateRequest {
  language: string;
  metadata: any;
  recipient: string;
  templateKey: string;
  type: any;
  variables: any;
  appId: string;
}

export interface ReportsConfig {
  storagePath: string;
  enabled: boolean;
  formats: string[];
  includeEvidence: boolean;
  retentionDays: number;
  schedule: string;
}

export interface TimeBasedRule {
  description: string;
  max_age: any;
  operation: string;
  org_id: string;
  security_level: string;
}

export interface FactorAdapterRegistry {
}

export interface SendOTP_body {
  user_id: string;
}

export interface SchemaValidator {
}

export interface AdminUpdateProviderRequest {
  clientSecret: string | undefined;
  enabled: boolean | undefined;
  scopes: string[];
  clientId: string | undefined;
}

export interface ComplianceCheckResponse {
  id: string;
}

export interface CreateSessionRequest {
}

export interface FacialCheckConfig {
  enabled: boolean;
  motionCapture: boolean;
  variant: string;
}

export interface StartRecoveryRequest {
  deviceId: string;
  email: string;
  preferredMethod: string;
  userId: string;
}

export interface mockUserService {
}

export interface ProviderDetailResponse {
  type: string;
  updatedAt: string;
  attributeMapping: any;
  createdAt: string;
  hasSamlCert: boolean;
  oidcClientID: string;
  oidcRedirectURI: string;
  samlIssuer: string;
  domain: string;
  oidcIssuer: string;
  providerId: string;
  samlEntryPoint: string;
}

export interface RunCheckRequest {
  checkType: string;
}

export interface OrganizationHandler {
}

export interface RiskContext {
}

export interface MFAPolicy {
  stepUpRequired: boolean;
  adaptiveMfaEnabled: boolean;
  maxFailedAttempts: number;
  organizationId: string;
  requiredFactorTypes: string[];
  trustedDeviceDays: number;
  updatedAt: string;
  allowedFactorTypes: string[];
  createdAt: string;
  gracePeriodDays: number;
  id: string;
  lockoutDurationMinutes: number;
  requiredFactorCount: number;
}

export interface AccessTokenClaims {
  client_id: string;
  scope: string;
  token_type: string;
}

export interface MockUserRepository {
}

export interface ComplianceReportsResponse {
  reports: any[];
}

export interface ConsentDecision {
}

export interface ClientsListResponse {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
  clients: ClientSummary[];
}

export interface BackupAuthContactResponse {
  id: string;
}

export interface RetentionConfig {
  archiveBeforePurge: boolean;
  archivePath: string;
  enabled: boolean;
  gracePeriodDays: number;
  purgeSchedule: string;
}

export interface ComplianceCheck {
  status: string;
  appId: string;
  checkType: string;
  id: string;
  nextCheckAt: string;
  profileId: string;
  createdAt: string;
  evidence: string[];
  lastCheckedAt: string;
  result: any;
}

export interface AmountRule {
  security_level: string;
  currency: string;
  description: string;
  max_amount: number;
  min_amount: number;
  org_id: string;
}

export interface BackupCodeFactorAdapter {
}

export interface RateLimiter {
}

export interface TrustedContactInfo {
  relationship: string;
  verified: boolean;
  verifiedAt: string | undefined;
  active: boolean;
  email: string;
  id: string;
  name: string;
  phone: string;
}

export interface ListPasskeysRequest {
}

export interface SendCodeRequest {
  phone: string;
}

export interface CreateTemplateVersion_req {
  changes: string;
}

export interface LinkAccountRequest {
  provider: string;
  scopes: string[];
}

export interface CompleteVideoSessionRequest {
  livenessPassed: boolean;
  livenessScore: number;
  notes: string;
  verificationResult: string;
  videoSessionId: string;
}

export interface RateLimitRule {
  max: number;
  window: any;
}

export interface MockAppService {
}

export interface BatchEvaluateRequest {
  requests: EvaluateRequest[];
}

export interface SaveNotificationSettings_req {
  autoSendWelcome: boolean;
  cleanupAfter: string;
  retryAttempts: number;
  retryDelay: string;
}

export interface OAuthState {
  created_at: string;
  extra_scopes: string[];
  link_user_id: string | undefined;
  provider: string;
  redirect_url: string;
  user_organization_id: string | undefined;
  app_id: string;
}

export interface Email {
}

export interface MultiSessionSetActiveResponse {
  session: any;
  token: string;
}

export interface TwoFAEnableResponse {
  status: string;
  totp_uri: string;
}

export interface StripeIdentityProvider {
}

export interface CreateVerificationRequest {
}

export interface MockRepository {
}

export interface OIDCLoginRequest {
  nonce: string;
  redirectUri: string;
  scope: string;
  state: string;
}

export interface FinishRegisterResponse {
  createdAt: string;
  credentialId: string;
  name: string;
  passkeyId: string;
  status: string;
}

export interface Service {
}

export interface JWK {
  kid: string;
  kty: string;
  n: string;
  use: string;
  alg: string;
  e: string;
}

export interface SecretsConfigSource {
}

export interface ProvidersAppResponse {
  appId: string;
  providers: string[];
}

export interface GetDocumentVerificationResponse {
  rejectionReason: string;
  status: string;
  verifiedAt: string | undefined;
  confidenceScore: number;
  documentId: string;
  message: string;
}

export interface ResolveViolationRequest {
  notes: string;
  resolution: string;
}

export interface ComplianceTemplate {
  retentionDays: number;
  sessionMaxAge: number;
  auditFrequencyDays: number;
  dataResidency: string;
  description: string;
  mfaRequired: boolean;
  name: string;
  standard: string;
  passwordMinLength: number;
  requiredPolicies: string[];
  requiredTraining: string[];
}

export interface VerifyFactor_req {
  code: string;
}

export interface DiscoveryService {
}

export interface mockImpersonationRepository {
}

export interface BackupAuthQuestionsResponse {
  questions: string[];
}

export interface RecoverySession {
}

export interface DeletePasskeyRequest {
}

export interface CreateNamespaceRequest {
  description: string;
  inheritPlatform: boolean;
  name: string;
  templateId: string;
}

export interface EmailConfig {
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
  code_expiry_minutes: number;
  code_length: number;
  enabled: boolean;
  provider: string;
}

export interface EncryptionService {
}

export interface PolicyEngine {
}

export interface SaveBuilderTemplate_req {
  subject: string;
  templateId?: string;
  templateKey: string;
  document: any;
  name: string;
}

export interface AdminHandler {
}

export interface IPWhitelistConfig {
  enabled: boolean;
  strict_mode: boolean;
}

export interface ComplianceEvidenceResponse {
  id: string;
}

export interface MockEmailService {
}

export interface ComplianceUserTrainingResponse {
  user_id: string;
}

export interface CreateTrainingRequest {
  standard: string;
  trainingType: string;
  userId: string;
}

export interface FinishLoginRequest {
  response: any;
  remember: boolean;
}

export interface AddCustomPermission_req {
  category: string;
  description: string;
  name: string;
}

export interface AnalyticsResponse {
  timeRange: any;
  generatedAt: string;
  summary: AnalyticsSummary;
}

export interface ProvidersConfig {
  sms: SMSProviderConfig | undefined;
  email: EmailProviderConfig;
}

export interface ClientRegistrationResponse {
  logo_uri: string;
  response_types: string[];
  application_type: string;
  policy_uri: string;
  redirect_uris: string[];
  token_endpoint_auth_method: string;
  client_name: string;
  client_secret: string;
  contacts: string[];
  grant_types: string[];
  post_logout_redirect_uris: string[];
  client_id: string;
  client_id_issued_at: number;
  client_secret_expires_at: number;
  scope: string;
  tos_uri: string;
}

export interface IDVerificationResponse {
  verification: any;
}

export interface ImpersonationErrorResponse {
  error: string;
}

export interface AppHandler {
}

export interface TOTPConfig {
  algorithm: string;
  digits: number;
  enabled: boolean;
  issuer: string;
  period: number;
  window_size: number;
}

export interface SMSFactorAdapter {
}

export interface ClientDetailsResponse {
  postLogoutRedirectURIs: string[];
  updatedAt: string;
  name: string;
  responseTypes: string[];
  allowedScopes: string[];
  contacts: string[];
  createdAt: string;
  grantTypes: string[];
  requireConsent: boolean;
  tokenEndpointAuthMethod: string;
  tosURI: string;
  trustedClient: boolean;
  applicationType: string;
  redirectURIs: string[];
  requirePKCE: boolean;
  clientID: string;
  isOrgLevel: boolean;
  logoURI: string;
  organizationID: string;
  policyURI: string;
}

export interface VerificationListResponse {
  offset: number;
  total: number;
  verifications: any | undefined[];
  limit: number;
}

export interface GetRecoveryStatsRequest {
  endDate: string;
  organizationId: string;
  startDate: string;
}

export interface GenerateReport_req {
  format: string;
  period: string;
  reportType: string;
  standard: string;
}

export interface MFAStatus {
  enrolledFactors: FactorInfo[];
  gracePeriod: string | undefined;
  policyActive: boolean;
  requiredCount: number;
  trustedDevice: boolean;
  enabled: boolean;
}

export interface ErrorResponse {
  details: any;
  error: string;
  code: string;
}

export interface ConsentRequest {
  action: string;
  client_id: string;
  code_challenge: string;
  code_challenge_method: string;
  redirect_uri: string;
  response_type: string;
  scope: string;
  state: string;
}

export interface MockSessionService {
}

export interface StartVideoSessionRequest {
  videoSessionId: string;
}

export interface SendRequest {
  email: string;
}

export interface CreateABTestVariant_req {
  body: string;
  name: string;
  subject: string;
  weight: number;
}

export interface TestPolicyRequest {
  actions: string[];
  expression: string;
  resourceType: string;
  testCases: TestCase[];
}

export interface TeamHandler {
}

export interface SetupSecurityQuestionsRequest {
  questions: SetupSecurityQuestionRequest[];
}

export interface ChallengeRequest {
  metadata: any;
  userId: string;
  context: string;
  factorTypes: string[];
}

export interface NotificationsResponse {
  count: number;
  notifications: any;
}

export interface ClientUpdateRequest {
  require_pkce: boolean | undefined;
  token_endpoint_auth_method: string;
  trusted_client: boolean | undefined;
  allowed_scopes: string[];
  contacts: string[];
  name: string;
  policy_uri: string;
  redirect_uris: string[];
  require_consent: boolean | undefined;
  response_types: string[];
  tos_uri: string;
  grant_types: string[];
  logo_uri: string;
  post_logout_redirect_uris: string[];
}

export interface AppServiceAdapter {
}

export interface ComplianceTemplatesResponse {
  templates: any[];
}

export interface OIDCLoginResponse {
  state: string;
  authUrl: string;
  nonce: string;
  providerId: string;
}

export interface DataExportRequest {
  completedAt: string | undefined;
  createdAt: string;
  exportPath: string;
  userId: string;
  exportUrl: string;
  id: string;
  organizationId: string;
  status: string;
  errorMessage: string;
  expiresAt: string | undefined;
  exportSize: number;
  format: string;
  ipAddress: string;
  includeSections: string[];
  updatedAt: string;
}

export interface RiskAssessmentConfig {
  blockHighRisk: boolean;
  enabled: boolean;
  historyWeight: number;
  newDeviceWeight: number;
  newLocationWeight: number;
  requireReviewAbove: number;
  highRiskThreshold: number;
  lowRiskThreshold: number;
  mediumRiskThreshold: number;
  newIpWeight: number;
  velocityWeight: number;
}

export interface BackupAuthDocumentResponse {
  id: string;
}

export interface NotificationStatusResponse {
  status: string;
}

export interface IDVerificationWebhookResponse {
  status: string;
}

export interface RecoveryCodeUsage {
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

export interface FactorEnrollmentRequest {
  type: string;
  metadata: any;
  name: string;
  priority: string;
}

export interface ScopeInfo {
}

export interface VideoVerificationSession {
}

export interface WebAuthnWrapper {
}

export interface ValidatePolicyResponse {
  message: string;
  valid: boolean;
  warnings: string[];
  complexity: number;
  error: string;
  errors: string[];
}

export interface ConsentExportResponse {
  status: string;
  id: string;
}

export interface SetupSecurityQuestionRequest {
  answer: string;
  customText: string;
  questionId: number;
}

export interface ListUsersResponse {
  limit: number;
  page: number;
  total: number;
  total_pages: number;
  users: any | undefined[];
}

export interface EvaluationResult {
  challenge_token: string;
  current_level: string;
  expires_at: string;
  grace_period_ends_at: string;
  metadata: any;
  reason: string;
  security_level: string;
  can_remember: boolean;
  matched_rules: string[];
  required: boolean;
  requirement_id: string;
  allowed_methods: string[];
}

export interface PoliciesListResponse {
  pageSize: number;
  policies: PolicyResponse | undefined[];
  totalCount: number;
  page: number;
}

export interface ConsentReport {
  reportPeriodStart: string;
  consentRate: number;
  dataExportsThisPeriod: number;
  dpasActive: number;
  dpasExpiringSoon: number;
  totalUsers: number;
  usersWithConsent: number;
  completedDeletions: number;
  consentsByType: any;
  organizationId: string;
  pendingDeletions: number;
  reportPeriodEnd: string;
}

export interface Handler {
}

export interface KeyPair {
}

export interface SAMLLoginRequest {
  relayState: string;
}

export interface OnfidoConfig {
  includeDocumentReport: boolean;
  region: string;
  includeFacialReport: boolean;
  includeWatchlistReport: boolean;
  webhookToken: string;
  workflowId: string;
  apiToken: string;
  documentCheck: DocumentCheckConfig;
  enabled: boolean;
  facialCheck: FacialCheckConfig;
}

export interface ScopeDefinition {
}

export interface CompleteTrainingRequest {
  score: number;
}

export interface ListPasskeysResponse {
  count: number;
  passkeys: PasskeyInfo[];
}

export interface GetMigrationStatusResponse {
  migratedCount: number;
  pendingRbacPolicies: number;
  hasMigratedPolicies: boolean;
  lastMigrationAt: string;
}

export interface TrackNotificationEvent_req {
  event: string;
  eventData?: any;
  notificationId: string;
  organizationId?: string | undefined;
  templateId: string;
}

export interface InstantiateTemplateRequest {
  resourceType: string;
  actions: string[];
  description: string;
  enabled: boolean;
  name: string;
  namespaceId: string;
  parameters: any;
  priority: number;
}

export interface ResourceAttributeRequest {
  default: any;
  description: string;
  name: string;
  required: boolean;
  type: string;
}

export interface ChallengeResponse {
  sessionId: string;
  availableFactors: FactorInfo[];
  challengeId: string;
  expiresAt: string;
  factorsRequired: number;
}

export interface TrustedDevice {
  name: string;
  userAgent: string;
  userId: string;
  id: string;
  ipAddress: string;
  lastUsedAt: string | undefined;
  createdAt: string;
  deviceId: string;
  expiresAt: string;
  metadata: any;
}

export interface JumioProvider {
}

export interface ImpersonationVerifyResponse {
  impersonator_id: string;
  is_impersonating: boolean;
  target_user_id: string;
}

export interface BackupAuthRecoveryResponse {
  session_id: string;
}

export interface CompliancePolicy {
  content: string;
  createdAt: string;
  reviewDate: string;
  standard: string;
  status: string;
  title: string;
  approvedAt: string | undefined;
  effectiveDate: string;
  profileId: string;
  updatedAt: string;
  version: string;
  appId: string;
  approvedBy: string;
  policyType: string;
  id: string;
  metadata: any;
}

export interface AccessConfig {
  requireRbac: boolean;
  allowApiAccess: boolean;
  allowDashboardAccess: boolean;
  rateLimitPerMinute: number;
  requireAuthentication: boolean;
}

export interface LinkResponse {
  message: string;
  user: any;
}

export interface EmailVerificationConfig {
  codeExpiry: any;
  codeLength: number;
  emailTemplate: string;
  enabled: boolean;
  fromAddress: string;
  fromName: string;
  maxAttempts: number;
  requireEmailProof: boolean;
}

export interface GetDocumentVerificationRequest {
  documentId: string;
}

export interface AuditServiceAdapter {
}

export interface TestCaseResult {
  error: string;
  evaluationTimeMs: number;
  expected: boolean;
  name: string;
  passed: boolean;
  actual: boolean;
}

export interface TemplateResponse {
  id: string;
  name: string;
  parameters: any[];
  category: string;
  description: string;
  examples: string[];
  expression: string;
}

export interface RiskEngine {
}

export interface AuditConfig {
  autoCleanup: boolean;
  enableAccessLog: boolean;
  logReads: boolean;
  logWrites: boolean;
  retentionDays: number;
}

export interface ProvidersResponse {
  providers: string[];
}

export interface ScheduleVideoSessionRequest {
  scheduledAt: string;
  sessionId: string;
  timeZone: string;
}

export interface ComplianceTrainingResponse {
  id: string;
}

export interface DiscoverProviderRequest {
  email: string;
}

export interface DataDeletionRequest {
  archivePath: string;
  id: string;
  rejectedAt: string | undefined;
  retentionExempt: boolean;
  updatedAt: string;
  errorMessage: string;
  ipAddress: string;
  organizationId: string;
  exemptionReason: string;
  requestReason: string;
  userId: string;
  approvedAt: string | undefined;
  approvedBy: string;
  completedAt: string | undefined;
  createdAt: string;
  deleteSections: string[];
  status: string;
}

export interface VerifyCodeResponse {
  valid: boolean;
  attemptsLeft: number;
  message: string;
}

export interface SMSVerificationConfig {
  messageTemplate: string;
  provider: string;
  codeExpiry: any;
  codeLength: number;
  cooldownPeriod: any;
  enabled: boolean;
  maxAttempts: number;
  maxSmsPerDay: number;
}

export interface StepUpDevicesResponse {
  count: number;
  devices: any;
}

export interface MFAPolicyResponse {
  requiredFactorCount: number;
  allowedFactorTypes: string[];
  appId: string;
  enabled: boolean;
  gracePeriodDays: number;
  id: string;
  organizationId: string | undefined;
}

export interface EnrollFactorRequest {
  metadata: any;
  name: string;
  priority: string;
  type: string;
}

export interface Factor {
  verifiedAt: string | undefined;
  createdAt: string;
  expiresAt: string | undefined;
  id: string;
  lastUsedAt: string | undefined;
  metadata: any;
  updatedAt: string;
  name: string;
  priority: string;
  status: string;
  type: string;
  userId: string;
}

export interface ClientSummary {
  applicationType: string;
  clientID: string;
  createdAt: string;
  isOrgLevel: boolean;
  name: string;
}

export interface CookieConsent {
  essential: boolean;
  expiresAt: string;
  id: string;
  organizationId: string;
  userAgent: string;
  consentBannerVersion: string;
  functional: boolean;
  sessionId: string;
  thirdParty: boolean;
  createdAt: string;
  ipAddress: string;
  marketing: boolean;
  personalization: boolean;
  userId: string;
  analytics: boolean;
  updatedAt: string;
}

export interface BackupAuthStatusResponse {
  status: string;
}

export interface CancelRecoveryRequest {
  reason: string;
  sessionId: string;
}

export interface ComplianceTrainingsResponse {
  training: any[];
}

export interface StepUpStatusResponse {
  status: string;
}

export interface ProviderDiscoveredResponse {
  found: boolean;
  providerId: string;
  type: string;
}

export interface StateStore {
}

export interface ConsentStatusResponse {
  status: string;
}

export interface ImpersonationEndResponse {
  ended_at: string;
  status: string;
}

export interface CreateAPIKey_reqBody {
  allowed_ips?: string[];
  description?: string;
  metadata?: any;
  name: string;
  permissions?: any;
  rate_limit?: number;
  scopes: string[];
}

export interface MemberHandler {
}

export interface ContentTypeHandler {
}

export interface TokenResponse {
  expires_in: number;
  id_token: string;
  refresh_token: string;
  scope: string;
  token_type: string;
  access_token: string;
}

export interface IDVerificationListResponse {
  verifications: any[];
}

export interface ImpersonationStartResponse {
  target_user_id: string;
  impersonator_id: string;
  session_id: string;
  started_at: string;
}

export interface SecurityQuestion {
}

export interface ResourceResponse {
  type: string;
  attributes: any[];
  createdAt: string;
  description: string;
  id: string;
  namespaceId: string;
}

export interface CreateProvider_req {
  config: any;
  isDefault: boolean;
  organizationId?: string | undefined;
  providerName: string;
  providerType: string;
}

export interface SendVerificationCodeResponse {
  expiresAt: string;
  maskedTarget: string;
  message: string;
  sent: boolean;
}

export interface AnalyticsSummary {
  totalPolicies: number;
  activePolicies: number;
  allowedCount: number;
  avgLatencyMs: number;
  deniedCount: number;
  topResourceTypes: ResourceTypeStats[];
  totalEvaluations: number;
  cacheHitRate: number;
  topPolicies: PolicyStats[];
}

export interface TOTPSecret {
}

export interface NotificationErrorResponse {
  error: string;
}

export interface CreatePolicyRequest {
  consentType: string;
  content: string;
  metadata: any;
  required: boolean;
  version: string;
  description: string;
  name: string;
  renewable: boolean;
  validityPeriod: number | undefined;
}

export interface NoOpEmailProvider {
}

export interface RequestTrustedContactVerificationResponse {
  notifiedAt: string;
  contactId: string;
  contactName: string;
  expiresAt: string;
  message: string;
}

export interface UploadDocumentResponse {
  documentId: string;
  message: string;
  processingTime: string;
  status: string;
  uploadedAt: string;
}

export interface SuccessResponse {
  data: any;
  message: string;
}

export interface mockRepository {
}

export interface stateEntry {
}

export interface VerifyCodeRequest {
  code: string;
  sessionId: string;
}

export interface SignUpRequest {
  password: string;
  username: string;
}

export interface CreateProfileRequest {
  name: string;
  passwordExpiryDays: number;
  appId: string;
  passwordRequireLower: boolean;
  retentionDays: number;
  sessionIpBinding: boolean;
  sessionMaxAge: number;
  leastPrivilege: boolean;
  passwordRequireNumber: boolean;
  passwordRequireSymbol: boolean;
  passwordRequireUpper: boolean;
  rbacRequired: boolean;
  regularAccessReview: boolean;
  sessionIdleTimeout: number;
  complianceContact: string;
  dataResidency: string;
  encryptionAtRest: boolean;
  encryptionInTransit: boolean;
  standards: string[];
  passwordMinLength: number;
  auditLogExport: boolean;
  detailedAuditTrail: boolean;
  dpoContact: string;
  metadata: any;
  mfaRequired: boolean;
}

export interface ComplianceDashboardResponse {
  metrics: any;
}

export interface StepUpVerification {
  session_id: string;
  created_at: string;
  device_id: string;
  reason: string;
  user_id: string;
  security_level: string;
  user_agent: string;
  expires_at: string;
  id: string;
  ip: string;
  metadata: any;
  method: string;
  org_id: string;
  verified_at: string;
  rule_name: string;
}

export interface Adapter {
}

export interface VerificationResponse {
  verification: any | undefined;
}

export interface ConsentAuditConfig {
  archiveInterval: any;
  archiveOldLogs: boolean;
  enabled: boolean;
  immutable: boolean;
  logIpAddress: boolean;
  signLogs: boolean;
  exportFormat: string;
  logAllChanges: boolean;
  logUserAgent: boolean;
  retentionDays: number;
}

export interface AddMember_req {
  role: string;
  user_id: string;
}

export interface NamespacesListResponse {
  namespaces: NamespaceResponse | undefined[];
  totalCount: number;
}

export interface TwoFABackupCodesResponse {
  codes: string[];
}

export interface TokenIntrospectionResponse {
  aud: string[];
  iat: number;
  iss: string;
  nbf: number;
  token_type: string;
  client_id: string;
  exp: number;
  jti: string;
  scope: string;
  sub: string;
  username: string;
  active: boolean;
}

export interface RolesResponse {
  roles: any | undefined[];
}

export interface TrustedContactsConfig {
  maximumContacts: number;
  requireVerification: boolean;
  requiredToRecover: number;
  allowEmailContacts: boolean;
  allowPhoneContacts: boolean;
  minimumContacts: number;
  verificationExpiry: any;
  cooldownPeriod: any;
  enabled: boolean;
  maxNotificationsPerDay: number;
}

export interface DocumentVerificationRequest {
}

export interface StepUpRequirement {
  route: string;
  rule_name: string;
  user_agent: string;
  user_id: string;
  created_at: string;
  fulfilled_at: string | undefined;
  org_id: string;
  resource_action: string;
  session_id: string;
  method: string;
  resource_type: string;
  amount: number;
  challenge_token: string;
  expires_at: string;
  ip: string;
  reason: string;
  required_level: string;
  risk_score: number;
  status: string;
  currency: string;
  current_level: string;
  id: string;
  metadata: any;
}

export interface StepUpAuditLogsResponse {
  audit_logs: any[];
}

export interface VideoSessionInfo {
}

export interface SignUpResponse {
  status: string;
  message: string;
}

export interface VerificationResult {
}

export interface SSOAuthResponse {
  session: any | undefined;
  token: string;
  user: any | undefined;
}

export interface UpdateProvider_req {
  config: any;
  isActive: boolean;
  isDefault: boolean;
}

export interface TokenRequest {
  client_secret: string;
  code: string;
  code_verifier: string;
  grant_type: string;
  redirect_uri: string;
  scope: string;
  audience: string;
  refresh_token: string;
  client_id: string;
}

export interface RegistrationService {
}

export interface ProviderRegisteredResponse {
  providerId: string;
  status: string;
  type: string;
}

export interface TestProvider_req {
  providerName: string;
  providerType: string;
  testRecipient: string;
}

export interface ClientAuthResult {
}

export interface IDVerificationStatusResponse {
  status: any;
}

export interface ConsentDeletionResponse {
  id: string;
  status: string;
}

export interface CreateDPARequest {
  metadata: any;
  signedByName: string;
  agreementType: string;
  effectiveDate: string;
  signedByEmail: string;
  signedByTitle: string;
  version: string;
  content: string;
  expiryDate: string | undefined;
}

export interface mockSessionService {
}

export interface ScheduleVideoSessionResponse {
  videoSessionId: string;
  instructions: string;
  joinUrl: string;
  message: string;
  scheduledAt: string;
}

export interface StepUpPolicyResponse {
  id: string;
}

export interface Plugin {
}

export interface DeclareABTestWinner_req {
  abTestGroup: string;
  winnerId: string;
}

export interface AMLMatch {
}

export interface ProviderCheckResult {
}

export interface Verify_body {
  remember_device: boolean;
  user_id: string;
  code: string;
  device_id: string;
}

export interface TokenRevocationRequest {
  token_type_hint: string;
  client_id: string;
  client_secret: string;
  token: string;
}

export interface ConsentAuditLogsResponse {
  audit_logs: any[];
}

export interface VerifySecurityAnswersRequest {
  answers: any;
  sessionId: string;
}

export interface MockService {
}

export interface FactorEnrollmentResponse {
  factorId: string;
  provisioningData: any;
  status: string;
  type: string;
}

export interface GetChallengeStatusRequest {
}

export interface UpdateConsentRequest {
  metadata: any;
  reason: string;
  granted: boolean | undefined;
}

export interface MemoryStateStore {
}

export interface AccountLockoutError {
}

export interface ComplianceViolation {
  createdAt: string;
  id: string;
  metadata: any;
  profileId: string;
  resolvedBy: string;
  severity: string;
  status: string;
  description: string;
  resolvedAt: string | undefined;
  userId: string;
  violationType: string;
  appId: string;
}

export interface ContextRule {
  security_level: string;
  condition: string;
  description: string;
  name: string;
  org_id: string;
}

export interface auditServiceAdapter {
}

export interface BeginRegisterResponse {
  challenge: string;
  options: any;
  timeout: any;
  userId: string;
}

export interface MigrationHandler {
}

export interface EncryptionConfig {
  masterKey: string;
  rotateKeyAfter: any;
  testOnStartup: boolean;
}

export interface ConsentExpiryConfig {
  requireReConsent: boolean;
  allowRenewal: boolean;
  autoExpireCheck: boolean;
  defaultValidityDays: number;
  enabled: boolean;
  expireCheckInterval: any;
  renewalReminderDays: number;
}

export interface AccountLockedResponse {
  locked_minutes: number;
  locked_until: string;
  message: string;
  code: string;
}

export interface InvitationResponse {
  invitation: any | undefined;
  message: string;
}

export interface MigrateAllResponse {
  startedAt: string;
  convertedPolicies: PolicyPreviewResponse[];
  migratedPolicies: number;
  totalPolicies: number;
  completedAt: string;
  dryRun: boolean;
  errors: MigrationErrorResponse[];
  failedPolicies: number;
  skippedPolicies: number;
}

export interface AdminPolicyRequest {
  allowedTypes: string[];
  enabled: boolean;
  gracePeriod: number;
  requiredFactors: number;
}

export interface CodesResponse {
  codes: string[];
}

export interface AddTrustedContactResponse {
  name: string;
  phone: string;
  verified: boolean;
  addedAt: string;
  contactId: string;
  email: string;
  message: string;
}

export interface CompliancePoliciesResponse {
  policies: any[];
}

export interface MemoryChallengeStore {
}

export interface VerifyResponse {
  session: any | undefined;
  success: boolean;
  token: string;
  user: any | undefined;
}

export interface SetupSecurityQuestionsResponse {
  count: number;
  message: string;
  setupAt: string;
}

export interface RequirementsResponse {
  count: number;
  requirements: any;
}

export interface SessionStatsResponse {
  totalSessions: number;
  activeSessions: number;
  deviceCount: number;
  locationCount: number;
  newestSession: string | undefined;
  oldestSession: string | undefined;
}

export interface PreviewConversionResponse {
  policyName: string;
  resourceId: string;
  resourceType: string;
  success: boolean;
  celExpression: string;
  error: string;
}

export interface RevisionHandler {
}

export interface ConsentTypeStatus {
  granted: boolean;
  grantedAt: string;
  needsRenewal: boolean;
  type: string;
  version: string;
  expiresAt: string | undefined;
}

export interface DataDeletionConfig {
  autoProcessAfterGrace: boolean;
  notifyBeforeDeletion: boolean;
  allowPartialDeletion: boolean;
  archiveBeforeDeletion: boolean;
  archivePath: string;
  enabled: boolean;
  gracePeriodDays: number;
  preserveLegalData: boolean;
  requireAdminApproval: boolean;
  retentionExemptions: string[];
}

export interface MultiStepRecoveryConfig {
  minimumSteps: number;
  requireAdminApproval: boolean;
  allowStepSkip: boolean;
  allowUserChoice: boolean;
  highRiskSteps: string[];
  lowRiskSteps: string[];
  mediumRiskSteps: string[];
  sessionExpiry: any;
  enabled: boolean;
}

export interface NotificationListResponse {
  notifications: any[];
  total: number;
}

export interface DocumentVerificationResult {
}

export interface MultiSessionListResponse {
  sessions: any[];
}

export interface InitiateChallengeRequest {
  context: string;
  factorTypes: string[];
  metadata: any;
}

export interface EmailFactorAdapter {
}

export interface Disable_body {
  user_id: string;
}

export interface ConsentSummary {
  grantedConsents: number;
  hasPendingDeletion: boolean;
  hasPendingExport: boolean;
  pendingRenewals: number;
  userId: string;
  consentsByType: any;
  expiredConsents: number;
  lastConsentUpdate: string | undefined;
  organizationId: string;
  revokedConsents: number;
  totalConsents: number;
}

export interface ImpersonationSession {
}

export interface PolicyPreviewResponse {
  actions: string[];
  description: string;
  expression: string;
  name: string;
  resourceType: string;
}

export interface ContentEntryHandler {
}

export interface OAuthErrorResponse {
  error: string;
  error_description: string;
  error_uri: string;
  state: string;
}

export interface RejectRecoveryResponse {
  rejectedAt: string;
  sessionId: string;
  message: string;
  reason: string;
  rejected: boolean;
}

export interface StartVideoSessionResponse {
  expiresAt: string;
  message: string;
  sessionUrl: string;
  startedAt: string;
  videoSessionId: string;
}

export interface RecoveryAttemptLog {
}

export interface TeamsResponse {
  teams: any | undefined[];
  total: number;
}

export interface ValidatePolicyRequest {
  expression: string;
  resourceType: string;
}

export interface ActionResponse {
  name: string;
  namespaceId: string;
  createdAt: string;
  description: string;
  id: string;
}

export interface ProviderInfo {
  domain: string;
  providerId: string;
  type: string;
  createdAt: string;
}

export interface BunRepository {
}

export interface CreateProfileFromTemplate_req {
  standard: string;
}

export interface AuditLogResponse {
  page: number;
  pageSize: number;
  totalCount: number;
  entries: AuditLogEntry | undefined[];
}

export interface WebAuthnFactorAdapter {
}

export interface TrustedDevicesConfig {
  default_expiry_days: number;
  enabled: boolean;
  max_devices_per_user: number;
  max_expiry_days: number;
}

export interface AdaptiveMFAConfig {
  location_change_risk: number;
  require_step_up_threshold: number;
  velocity_risk: number;
  enabled: boolean;
  factor_ip_reputation: boolean;
  factor_location_change: boolean;
  factor_velocity: boolean;
  new_device_risk: number;
  risk_threshold: number;
  factor_new_device: boolean;
}

export interface DashboardExtension {
}

export interface ProviderListResponse {
  providers: ProviderInfo[];
  total: number;
}

export interface ListRecoverySessionsRequest {
  organizationId: string;
  page: number;
  pageSize: number;
  requiresReview: boolean;
  status: string;
}

export interface UnbanUserRequest {
  app_id: string;
  reason: string;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface ChannelsResponse {
  channels: any;
  count: number;
}

export interface RecoveryCodesConfig {
  format: string;
  regenerateCount: number;
  allowDownload: boolean;
  allowPrint: boolean;
  autoRegenerate: boolean;
  codeCount: number;
  codeLength: number;
  enabled: boolean;
}

export interface GetSecurityQuestionsRequest {
  sessionId: string;
}

export interface GenerateReportRequest {
  format: string;
  period: string;
  reportType: string;
  standard: string;
}

export interface UserServiceAdapter {
}

export interface ComplianceViolationResponse {
  id: string;
}

export interface CreateUserRequest {
  user_organization_id: string | undefined;
  app_id: string;
  email_verified: boolean;
  password: string;
  role: string;
  username: string;
  email: string;
  metadata: any;
  name: string;
}

export interface StepUpAttempt {
  user_id: string;
  created_at: string;
  id: string;
  ip: string;
  method: string;
  success: boolean;
  failure_reason: string;
  org_id: string;
  requirement_id: string;
  user_agent: string;
}

export interface SessionTokenResponse {
  session: any;
  token: string;
}

export interface DevicesResponse {
  count: number;
  devices: any;
}

export interface IntrospectionService {
}

export interface ConsentExportFileResponse {
  content_type: string;
  data: number[];
}

export interface RateLimitConfig {
  enabled: boolean;
  window: any;
}

export interface UploadDocumentRequest {
  frontImage: string;
  selfie: string;
  sessionId: string;
  backImage: string;
  documentType: string;
}

export interface ComplianceEvidence {
  collectedBy: string;
  description: string;
  fileHash: string;
  fileUrl: string;
  id: string;
  metadata: any;
  title: string;
  appId: string;
  controlId: string;
  createdAt: string;
  evidenceType: string;
  profileId: string;
  standard: string;
}

export interface PasskeyInfo {
  aaguid: string;
  authenticatorType: string;
  createdAt: string;
  lastUsedAt: string | undefined;
  signCount: number;
  credentialId: string;
  id: string;
  isResidentKey: boolean;
  name: string;
}

export interface FactorVerificationRequest {
  code: string;
  data: any;
  factorId: string;
}

export interface RedisStateStore {
}

export interface IDVerificationErrorResponse {
  error: string;
}

export interface DataExportConfig {
  autoCleanup: boolean;
  cleanupInterval: any;
  enabled: boolean;
  expiryHours: number;
  includeSections: string[];
  maxExportSize: number;
  allowedFormats: string[];
  defaultFormat: string;
  maxRequests: number;
  requestPeriod: any;
  storagePath: string;
}

export interface StateStorageConfig {
  redisAddr: string;
  redisDb: number;
  redisPassword: string;
  stateTtl: any;
  useRedis: boolean;
}

export interface StepUpPoliciesResponse {
  policies: any[];
}

export interface NoOpNotificationProvider {
}

export interface ImpersonateUserRequest {
  duration: any;
  user_id: string;
  user_organization_id: string | undefined;
  app_id: string;
}

export interface Challenge {
  status: string;
  type: string;
  userId: string;
  verifiedAt: string | undefined;
  attempts: number;
  createdAt: string;
  expiresAt: string;
  factorId: string;
  id: string;
  metadata: any;
  userAgent: string;
  ipAddress: string;
  maxAttempts: number;
}

export interface VersioningConfig {
  autoCleanup: boolean;
  cleanupInterval: any;
  maxVersions: number;
  retentionDays: number;
}

export interface VerifyRequest {
  token: string;
}

export interface AssignRole_reqBody {
  roleID: string;
}

export interface CompleteRecoveryRequest {
  sessionId: string;
}

export interface SecurityQuestionInfo {
  id: string;
  isCustom: boolean;
  questionId: number;
  questionText: string;
}

export interface SetActive_body {
  id: string;
}

export interface BatchEvaluationResult {
  resourceType: string;
  action: string;
  allowed: boolean;
  error: string;
  evaluationTimeMs: number;
  index: number;
  policies: string[];
  resourceId: string;
}

export interface GetStatusRequest {
}

export interface OTPSentResponse {
  code: string;
  status: string;
}

export interface GetFactorRequest {
}

export interface MessageResponse {
  message: string;
}

export interface UpdatePolicy_req {
  version: string | undefined;
  content: string | undefined;
  status: string | undefined;
  title: string | undefined;
}

export interface ListPoliciesFilter {
  standard: string | undefined;
  status: string | undefined;
  appId: string | undefined;
  policyType: string | undefined;
  profileId: string | undefined;
}

export interface StepUpRequirementsResponse {
  requirements: any[];
}

export interface AuditLogEntry {
  id: string;
  ipAddress: string;
  oldValue: any;
  resourceId: string;
  action: string;
  appId: string;
  newValue: any;
  resourceType: string;
  timestamp: string;
  userAgent: string;
  userOrganizationId: string | undefined;
  actorId: string;
  environmentId: string;
}

export interface StatusResponse {
  status: string;
}

export interface ConnectionResponse {
  connection: any | undefined;
}

export interface RateLimitingConfig {
  exponentialBackoff: boolean;
  ipCooldownPeriod: any;
  lockoutAfterAttempts: number;
  lockoutDuration: any;
  maxAttemptsPerDay: number;
  maxAttemptsPerHour: number;
  maxAttemptsPerIp: number;
  enabled: boolean;
}

export interface BackupAuthSessionsResponse {
  sessions: any[];
}

export interface AuditLog {
}

export interface BeginRegisterRequest {
  userId: string;
  userVerification: string;
  authenticatorType: string;
  name: string;
  requireResidentKey: boolean;
}

export interface JWKSService {
}

export interface ConsentAuditLog {
  consentId: string;
  consentType: string;
  createdAt: string;
  id: string;
  newValue: Record<string, any>;
  organizationId: string;
  purpose: string;
  reason: string;
  action: string;
  ipAddress: string;
  previousValue: Record<string, any>;
  userAgent: string;
  userId: string;
}

export interface VerifySecurityAnswersResponse {
  valid: boolean;
  attemptsLeft: number;
  correctAnswers: number;
  message: string;
  requiredAnswers: number;
}

export interface NoOpVideoProvider {
}

export interface UpdateProfileRequest {
  mfaRequired: boolean | undefined;
  name: string | undefined;
  retentionDays: number | undefined;
  status: string | undefined;
}

export interface StepUpErrorResponse {
  error: string;
}

export interface ListFactorsRequest {
}

export interface MFASession {
  expiresAt: string;
  factorsVerified: number;
  id: string;
  sessionToken: string;
  userAgent: string;
  createdAt: string;
  factorsRequired: number;
  ipAddress: string;
  metadata: any;
  riskLevel: string;
  userId: string;
  verifiedFactors: string[];
  completedAt: string | undefined;
}

export interface VerificationsResponse {
  count: number;
  verifications: any;
}

export interface LoginResponse {
  passkeyUsed: string;
  session: any;
  token: string;
  user: any;
}

export interface CreateActionRequest {
  description: string;
  name: string;
  namespaceId: string;
}

export interface PhoneVerifyResponse {
  user: any | undefined;
  session: any | undefined;
  token: string;
}

export interface VerifyEnrolledFactorRequest {
  code: string;
  data: any;
}

export interface AuthorizeRequest {
  client_id: string;
  code_challenge_method: string;
  id_token_hint: string;
  response_type: string;
  state: string;
  ui_locales: string;
  acr_values: string;
  code_challenge: string;
  login_hint: string;
  max_age: number | undefined;
  nonce: string;
  prompt: string;
  redirect_uri: string;
  scope: string;
}

export interface PrivacySettings {
  requireExplicitConsent: boolean;
  updatedAt: string;
  allowDataPortability: boolean;
  cookieConsentStyle: string;
  dataExportExpiryHours: number;
  exportFormat: string[];
  id: string;
  autoDeleteAfterDays: number;
  ccpaMode: boolean;
  consentRequired: boolean;
  contactEmail: string;
  createdAt: string;
  deletionGracePeriodDays: number;
  gdprMode: boolean;
  metadata: Record<string, any>;
  cookieConsentEnabled: boolean;
  requireAdminApprovalForDeletion: boolean;
  anonymousConsentEnabled: boolean;
  contactPhone: string;
  dataRetentionDays: number;
  dpoEmail: string;
  organizationId: string;
}

export interface AuthURLResponse {
  url: string;
}

export interface AddTrustedContactRequest {
  email: string;
  name: string;
  phone: string;
  relationship: string;
}

export interface SetUserRole_reqBody {
  role: string;
}

export interface ComplianceTraining {
  trainingType: string;
  userId: string;
  appId: string;
  createdAt: string;
  expiresAt: string | undefined;
  id: string;
  metadata: any;
  score: number;
  standard: string;
  status: string;
  completedAt: string | undefined;
  profileId: string;
}

export interface OrganizationUIRegistry {
}

export interface JWTService {
}

export interface ConsentReportResponse {
  id: string;
}

export interface ConsentPolicyResponse {
  id: string;
}

export interface AutoCleanupConfig {
  interval: any;
  enabled: boolean;
}

export interface RejectRecoveryRequest {
  notes: string;
  reason: string;
  sessionId: string;
}

export interface ReviewDocumentRequest {
  documentId: string;
  notes: string;
  rejectionReason: string;
  approved: boolean;
}

export interface SendVerificationCodeRequest {
  target: string;
  method: string;
  sessionId: string;
}

export interface App {
}

export interface SignInResponse {
  session: any;
  token: string;
  user: any;
}

export interface VerificationRepository {
}

export interface RenderTemplate_req {
  template: string;
  variables: any;
}

export interface IDVerificationSessionResponse {
  session: any;
}

export interface UnblockUserRequest {
}

export interface KeyStats {
}

export interface ComplianceProfile {
  leastPrivilege: boolean;
  passwordMinLength: number;
  detailedAuditTrail: boolean;
  mfaRequired: boolean;
  passwordRequireSymbol: boolean;
  retentionDays: number;
  appId: string;
  complianceContact: string;
  dpoContact: string;
  name: string;
  passwordRequireLower: boolean;
  passwordRequireNumber: boolean;
  metadata: any;
  rbacRequired: boolean;
  regularAccessReview: boolean;
  status: string;
  passwordRequireUpper: boolean;
  sessionIdleTimeout: number;
  encryptionAtRest: boolean;
  encryptionInTransit: boolean;
  id: string;
  passwordExpiryDays: number;
  sessionIpBinding: boolean;
  sessionMaxAge: number;
  updatedAt: string;
  auditLogExport: boolean;
  createdAt: string;
  dataResidency: string;
  standards: string[];
}

export interface CreateTraining_req {
  userId: string;
  standard: string;
  trainingType: string;
}

export interface EmailProviderConfig {
  from: string;
  from_name: string;
  provider: string;
  reply_to: string;
  config: any;
}

export interface ConsentService {
}

export interface RequestReverification_req {
  reason: string;
}

export interface TrustedContact {
}

export interface ImpersonateUser_reqBody {
  duration?: any;
}

export interface StepUpEvaluationResponse {
  reason: string;
  required: boolean;
}

export interface StepUpVerificationResponse {
  expires_at: string;
  verified: boolean;
}

export interface BeginLoginResponse {
  challenge: string;
  options: any;
  timeout: any;
}

export interface DefaultProviderRegistry {
}

export interface DocumentVerification {
}

export interface CompliancePolicyResponse {
  id: string;
}

export interface EvaluationContext {
}

export interface SessionsResponse {
  sessions: any;
}

export interface SMSConfig {
  code_expiry_minutes: number;
  code_length: number;
  enabled: boolean;
  provider: string;
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
}

export interface WebhookResponse {
  status: string;
  received: boolean;
}

export interface DashboardConfig {
  enableExport: boolean;
  enableImport: boolean;
  enableReveal: boolean;
  enableTreeView: boolean;
  revealTimeout: any;
}

export interface ListEvidenceFilter {
  controlId: string | undefined;
  evidenceType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  appId: string | undefined;
}

export interface GetMigrationStatusRequest {
}

export interface BaseFactorAdapter {
}

export interface StripeIdentityConfig {
  useMock: boolean;
  webhookSecret: string;
  allowedTypes: string[];
  apiKey: string;
  enabled: boolean;
  requireLiveCapture: boolean;
  requireMatchingSelfie: boolean;
  returnUrl: string;
}

export interface PrivacySettingsRequest {
  gdprMode: boolean | undefined;
  requireAdminApprovalForDeletion: boolean | undefined;
  allowDataPortability: boolean | undefined;
  consentRequired: boolean | undefined;
  requireExplicitConsent: boolean | undefined;
  contactEmail: string;
  dataExportExpiryHours: number | undefined;
  anonymousConsentEnabled: boolean | undefined;
  autoDeleteAfterDays: number | undefined;
  contactPhone: string;
  dpoEmail: string;
  ccpaMode: boolean | undefined;
  cookieConsentEnabled: boolean | undefined;
  cookieConsentStyle: string;
  dataRetentionDays: number | undefined;
  deletionGracePeriodDays: number | undefined;
  exportFormat: string[];
}

export interface DataExportRequestInput {
  format: string;
  includeSections: string[];
}

export interface MockUserService {
}

export interface AdminAddProviderRequest {
  appId: string;
  clientId: string;
  clientSecret: string;
  enabled: boolean;
  provider: string;
  scopes: string[];
}

export interface StepUpVerificationsResponse {
  verifications: any[];
}

export interface mockProvider {
}

export interface ListTrainingFilter {
  appId: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
  trainingType: string | undefined;
  userId: string | undefined;
}

export interface ComplianceViolationsResponse {
  violations: any[];
}

export interface UserVerificationStatusResponse {
  status: any | undefined;
}

export interface ComplianceReport {
  standard: string;
  createdAt: string;
  expiresAt: string;
  format: string;
  generatedBy: string;
  reportType: string;
  status: string;
  summary: any;
  appId: string;
  fileSize: number;
  fileUrl: string;
  id: string;
  period: string;
  profileId: string;
}

export interface NotificationChannels {
  email: boolean;
  slack: boolean;
  webhook: boolean;
}

export interface ComplianceTemplateResponse {
  standard: string;
}

export interface RouteRule {
  description: string;
  method: string;
  org_id: string;
  pattern: string;
  security_level: string;
}

export interface KeyStore {
}

export interface ListTrustedContactsResponse {
  contacts: TrustedContactInfo[];
  count: number;
}

export interface ComplianceProfileResponse {
  id: string;
}

export interface GetPasskeyRequest {
}

export interface JWKS {
  keys: JWK[];
}

export interface VerifyRecoveryCodeResponse {
  message: string;
  remainingCodes: number;
  valid: boolean;
}

export interface StatsResponse {
  active_sessions: number;
  active_users: number;
  banned_users: number;
  timestamp: string;
  total_sessions: number;
  total_users: number;
}

export interface EmailServiceAdapter {
}

export interface PreviewConversionRequest {
  actions: string[];
  condition: string;
  resource: string;
  subject: string;
}

export interface MetadataResponse {
  metadata: string;
}

export interface MigrateAllRequest {
  dryRun: boolean;
  preserveOriginal: boolean;
}

export interface BatchEvaluateResponse {
  totalEvaluations: number;
  totalTimeMs: number;
  failureCount: number;
  results: BatchEvaluationResult | undefined[];
  successCount: number;
}

export interface User {
  email: string;
  name?: string;
  emailVerified: boolean;
  createdAt: string;
  updatedAt: string;
  organizationId?: string;
  id: string;
}

export interface JWKSResponse {
  keys: JWK[];
}

export interface bunRepository {
}

export interface ProviderConfigResponse {
  appId: string;
  message: string;
  provider: string;
}

export interface GetSecurityQuestionsResponse {
  questions: SecurityQuestionInfo[];
}

export interface VideoSessionResult {
}

export interface CookieConsentConfig {
  allowAnonymous: boolean;
  bannerVersion: string;
  categories: string[];
  defaultStyle: string;
  enabled: boolean;
  requireExplicit: boolean;
  validityPeriod: any;
}

export interface BackupAuthConfigResponse {
  config: any;
}

export interface AutomatedChecksConfig {
  sessionPolicy: boolean;
  suspiciousActivity: boolean;
  accessReview: boolean;
  dataRetention: boolean;
  enabled: boolean;
  mfaCoverage: boolean;
  checkInterval: any;
  inactiveUsers: boolean;
  passwordPolicy: boolean;
}

export interface ForgetDeviceResponse {
  message: string;
  success: boolean;
}

export interface ConsentSettingsResponse {
  settings: any;
}

export interface GenerateRecoveryCodesResponse {
  warning: string;
  codes: string[];
  count: number;
  generatedAt: string;
}

export interface ListSessionsRequest {
  app_id: string;
  limit: number;
  page: number;
  user_id: string | undefined;
  user_organization_id: string | undefined;
}

export interface MigrateRolesRequest {
  dryRun: boolean;
}

export interface TestPolicyResponse {
  passedCount: number;
  results: TestCaseResult[];
  total: number;
  error: string;
  failedCount: number;
  passed: boolean;
}

export interface EnableRequest {
}

export interface SignInRequest {
}

export interface LinkRequest {
  email: string;
  name: string;
  password: string;
}

export interface SecurityQuestionsConfig {
  predefinedQuestions: string[];
  requiredToRecover: number;
  allowCustomQuestions: boolean;
  forbidCommonAnswers: boolean;
  maxAnswerLength: number;
  minimumQuestions: number;
  requireMinLength: number;
  caseSensitive: boolean;
  enabled: boolean;
  lockoutDuration: any;
  maxAttempts: number;
}

export interface StartRecoveryResponse {
  riskScore: number;
  sessionId: string;
  status: string;
  availableMethods: string[];
  completedSteps: number;
  expiresAt: string;
  requiredSteps: number;
  requiresReview: boolean;
}

export interface ListProfilesFilter {
  appId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface TwoFAStatusResponse {
  enabled: boolean;
  method: string;
  trusted: boolean;
}

export interface TemplateEngine {
}

export interface StepUpPolicy {
  id: string;
  metadata: any;
  name: string;
  org_id: string;
  rules: any;
  updated_at: string;
  created_at: string;
  description: string;
  enabled: boolean;
  priority: number;
  user_id: string;
}

