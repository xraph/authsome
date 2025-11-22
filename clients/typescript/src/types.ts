// Auto-generated TypeScript types

export interface BackupAuthContactsResponse {
  contacts: any[];
}

export interface DeviceInfo {
  deviceId: string;
  metadata: any;
  name: string;
}

export interface MFASession {
  userAgent: string;
  userId: string;
  verifiedFactors: string[];
  createdAt: string;
  factorsRequired: number;
  id: string;
  riskLevel: string;
  sessionToken: string;
  completedAt: string | undefined;
  expiresAt: string;
  factorsVerified: number;
  ipAddress: string;
  metadata: any;
}

export interface StepUpVerification {
  device_id: string;
  expires_at: string;
  org_id: string;
  rule_name: string;
  reason: string;
  security_level: string;
  user_id: string;
  metadata: any;
  method: string;
  session_id: string;
  user_agent: string;
  verified_at: string;
  created_at: string;
  id: string;
  ip: string;
}

export interface CookieConsentRequest {
  analytics: boolean;
  bannerVersion: string;
  essential: boolean;
  functional: boolean;
  marketing: boolean;
  personalization: boolean;
  sessionId: string;
  thirdParty: boolean;
}

export interface ConsentsResponse {
  count: number;
  consents: any;
}

export interface ComplianceStatusDetailsResponse {
  status: string;
}

export interface DeletePasskeyRequest {
}

export interface IDVerificationListResponse {
  verifications: any[];
}

export interface FactorAdapterRegistry {
}

export interface StepUpAttempt {
  failure_reason: string;
  ip: string;
  method: string;
  org_id: string;
  requirement_id: string;
  user_agent: string;
  created_at: string;
  id: string;
  success: boolean;
  user_id: string;
}

export interface SetupSecurityQuestionRequest {
  customText: string;
  questionId: number;
  answer: string;
}

export interface OrganizationHandler {
}

export interface StepUpPolicyResponse {
  id: string;
}

export interface StepUpVerificationsResponse {
  verifications: any[];
}

export interface VerificationListResponse {
  offset: number;
  total: number;
  verifications: any | undefined[];
  limit: number;
}

export interface Status_body {
  user_id: string;
  device_id: string;
}

export interface NoOpDocumentProvider {
}

export interface CreateUser_reqBody {
  name?: string;
  password?: string;
  role?: string;
  username?: string;
  email: string;
  email_verified: boolean;
  metadata?: any;
}

export interface MockSessionService {
}

export interface DiscoverProviderRequest {
  email: string;
}

export interface SessionsResponse {
  sessions: any;
}

export interface ComplianceEvidencesResponse {
  evidence: any[];
}

export interface SecurityQuestionsConfig {
  enabled: boolean;
  lockoutDuration: any;
  maxAnswerLength: number;
  maxAttempts: number;
  minimumQuestions: number;
  requireMinLength: number;
  requiredToRecover: number;
  allowCustomQuestions: boolean;
  caseSensitive: boolean;
  forbidCommonAnswers: boolean;
  predefinedQuestions: string[];
}

export interface AddTeamMember_req {
  role: string;
  member_id: string;
}

export interface CallbackDataResponse {
  isNewUser: boolean;
  user: any | undefined;
  action: string;
}

export interface StepUpAuditLog {
  created_at: string;
  event_data: any;
  event_type: string;
  id: string;
  org_id: string;
  user_agent: string;
  ip: string;
  severity: string;
  user_id: string;
}

export interface VerifyTrustedContactResponse {
  contactId: string;
  message: string;
  verified: boolean;
  verifiedAt: string;
}

export interface ComplianceUserTrainingResponse {
  user_id: string;
}

export interface Config {
  allowImplicitSignup: boolean;
  baseURL: string;
  devExposeURL: boolean;
  expiryMinutes: number;
  rateLimitPerHour: number;
}

export interface ReviewDocumentRequest {
  notes: string;
  rejectionReason: string;
  approved: boolean;
  documentId: string;
}

export interface GetRecoveryStatsResponse {
  pendingRecoveries: number;
  successfulRecoveries: number;
  totalAttempts: number;
  adminReviewsRequired: number;
  averageRiskScore: number;
  failedRecoveries: number;
  highRiskAttempts: number;
  methodStats: any;
  successRate: number;
}

export interface OnfidoProvider {
}

export interface MockSocialAccountRepository {
}

export interface MockUserRepository {
}

export interface EvaluateRequest {
  action: string;
  amount: number;
  currency: string;
  metadata: any;
  method: string;
  resource_type: string;
  route: string;
}

export interface VerificationsResponse {
  count: number;
  verifications: any;
}

export interface FinishRegisterResponse {
  name: string;
  passkeyId: string;
  status: string;
  createdAt: string;
  credentialId: string;
}

export interface TrustedContactInfo {
  verifiedAt: string | undefined;
  active: boolean;
  email: string;
  id: string;
  name: string;
  phone: string;
  relationship: string;
  verified: boolean;
}

export interface NoOpVideoProvider {
}

export interface IPWhitelistConfig {
  enabled: boolean;
  strict_mode: boolean;
}

export interface CreateVerificationSession_req {
  cancelUrl: string;
  config: any;
  metadata: any;
  provider: string;
  requiredChecks: string[];
  successUrl: string;
}

export interface ComplianceTraining {
  appId: string;
  completedAt: string | undefined;
  createdAt: string;
  id: string;
  profileId: string;
  score: number;
  status: string;
  trainingType: string;
  expiresAt: string | undefined;
  metadata: any;
  standard: string;
  userId: string;
}

export interface DeclareABTestWinner_req {
  abTestGroup: string;
  winnerId: string;
}

export interface ConsentAuditLog {
  consentId: string;
  consentType: string;
  createdAt: string;
  ipAddress: string;
  newValue: Record<string, any>;
  userAgent: string;
  userId: string;
  action: string;
  id: string;
  organizationId: string;
  previousValue: Record<string, any>;
  purpose: string;
  reason: string;
}

export interface ProviderDetailResponse {
  oidcIssuer: string;
  samlEntryPoint: string;
  type: string;
  attributeMapping: any;
  domain: string;
  oidcRedirectURI: string;
  providerId: string;
  samlIssuer: string;
  updatedAt: string;
  createdAt: string;
  hasSamlCert: boolean;
  oidcClientID: string;
}

export interface ChallengeSession {
}

export interface BackupAuthCodesResponse {
  codes: string[];
}

export interface NoOpEmailProvider {
}

export interface FactorEnrollmentResponse {
  factorId: string;
  provisioningData: any;
  status: string;
  type: string;
}

export interface ImpersonationContext {
  impersonator_id: string | undefined;
  indicator_message: string;
  is_impersonating: boolean;
  target_user_id: string | undefined;
  impersonation_id: string | undefined;
}

export interface ConsentReport {
  dataExportsThisPeriod: number;
  dpasActive: number;
  dpasExpiringSoon: number;
  pendingDeletions: number;
  reportPeriodStart: string;
  organizationId: string;
  reportPeriodEnd: string;
  totalUsers: number;
  usersWithConsent: number;
  completedDeletions: number;
  consentRate: number;
  consentsByType: any;
}

export interface MockRepository {
}

export interface TrackNotificationEvent_req {
  event: string;
  eventData?: any;
  notificationId: string;
  organizationId?: string | undefined;
  templateId: string;
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

export interface AuditServiceAdapter {
}

export interface ContinueRecoveryRequest {
  method: string;
  sessionId: string;
}

export interface VerifyRecoveryCodeRequest {
  sessionId: string;
  code: string;
}

export interface CreateConsentRequest {
  granted: boolean;
  metadata: any;
  purpose: string;
  userId: string;
  version: string;
  consentType: string;
  expiresIn: number | undefined;
}

export interface UpdateConsentRequest {
  granted: boolean | undefined;
  metadata: any;
  reason: string;
}

export interface TokenResponse {
  access_token: string;
  expires_in: number;
  id_token: string;
  refresh_token: string;
  scope: string;
  token_type: string;
}

export interface UnbanUser_reqBody {
  reason?: string;
}

export interface LinkRequest {
  email: string;
  name: string;
  password: string;
}

export interface StepUpPoliciesResponse {
  policies: any[];
}

export interface UploadDocumentResponse {
  uploadedAt: string;
  documentId: string;
  message: string;
  processingTime: string;
  status: string;
}

export interface AdminBypassRequest {
  duration: number;
  reason: string;
  userId: string;
}

export interface ScheduleVideoSessionRequest {
  sessionId: string;
  timeZone: string;
  scheduledAt: string;
}

export interface ConsentPolicy {
  required: boolean;
  consentType: string;
  version: string;
  active: boolean;
  description: string;
  renewable: boolean;
  content: string;
  createdAt: string;
  createdBy: string;
  metadata: Record<string, any>;
  name: string;
  organizationId: string;
  publishedAt: string | undefined;
  id: string;
  updatedAt: string;
  validityPeriod: number | undefined;
}

export interface IDVerificationErrorResponse {
  error: string;
}

export interface GetChallengeStatusResponse {
  factorsRequired: number;
  factorsVerified: number;
  maxAttempts: number;
  status: string;
  attempts: number;
  availableFactors: FactorInfo[];
  challengeId: string;
}

export interface ProviderDiscoveredResponse {
  found: boolean;
  providerId: string;
  type: string;
}

export interface GenerateReportRequest {
  format: string;
  period: string;
  reportType: string;
  standard: string;
}

export interface CompliancePoliciesResponse {
  policies: any[];
}

export interface BeginLoginResponse {
  challenge: string;
  options: any;
  timeout: any;
}

export interface userServiceAdapter {
}

export interface OAuthErrorResponse {
  error: string;
  error_description: string;
  error_uri: string;
  state: string;
}

export interface AmountRule {
  min_amount: number;
  org_id: string;
  security_level: string;
  currency: string;
  description: string;
  max_amount: number;
}

export interface GenerateReport_req {
  format: string;
  period: string;
  reportType: string;
  standard: string;
}

export interface CreateProfileFromTemplate_req {
  standard: string;
}

export interface EndImpersonation_reqBody {
  impersonation_id: string;
  reason?: string;
}

export interface TemplatesResponse {
  count: number;
  templates: any;
}

export interface SecurityQuestionInfo {
  id: string;
  isCustom: boolean;
  questionId: number;
  questionText: string;
}

export interface RecoveryCodesConfig {
  allowPrint: boolean;
  autoRegenerate: boolean;
  codeCount: number;
  codeLength: number;
  enabled: boolean;
  format: string;
  regenerateCount: number;
  allowDownload: boolean;
}

export interface ConsentAuditLogsResponse {
  audit_logs: any[];
}

export interface VerificationRequest {
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
  challengeId: string;
  code: string;
}

export interface MultiSessionDeleteResponse {
  status: string;
}

export interface CodesResponse {
  codes: string[];
}

export interface UpdatePolicy_req {
  content: string | undefined;
  status: string | undefined;
  title: string | undefined;
  version: string | undefined;
}

export interface JumioConfig {
  callbackUrl: string;
  enableAMLScreening: boolean;
  enabledDocumentTypes: string[];
  verificationType: string;
  dataCenter: string;
  enableExtraction: boolean;
  enableLiveness: boolean;
  enabled: boolean;
  enabledCountries: string[];
  presetId: string;
  apiSecret: string;
  apiToken: string;
}

export interface AutomatedChecksConfig {
  accessReview: boolean;
  checkInterval: any;
  dataRetention: boolean;
  sessionPolicy: boolean;
  enabled: boolean;
  inactiveUsers: boolean;
  mfaCoverage: boolean;
  passwordPolicy: boolean;
  suspiciousActivity: boolean;
}

export interface ListTrainingFilter {
  appId: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
  trainingType: string | undefined;
  userId: string | undefined;
}

export interface AuditConfig {
  archiveInterval: any;
  archiveOldLogs: boolean;
  enabled: boolean;
  immutableLogs: boolean;
  logAllAttempts: boolean;
  logDeviceInfo: boolean;
  logFailed: boolean;
  logSuccessful: boolean;
  logIpAddress: boolean;
  logUserAgent: boolean;
  retentionDays: number;
}

export interface RequestTrustedContactVerificationResponse {
  contactId: string;
  contactName: string;
  expiresAt: string;
  message: string;
  notifiedAt: string;
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

export interface SSOAuthResponse {
  session: any | undefined;
  token: string;
  user: any | undefined;
}

export interface TemplateEngine {
}

export interface TwoFARequiredResponse {
  device_id: string;
  require_twofa: boolean;
  user: any | undefined;
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

export interface DataExportConfig {
  expiryHours: number;
  includeSections: string[];
  maxRequests: number;
  maxExportSize: number;
  requestPeriod: any;
  storagePath: string;
  allowedFormats: string[];
  autoCleanup: boolean;
  cleanupInterval: any;
  defaultFormat: string;
  enabled: boolean;
}

export interface TOTPConfig {
  enabled: boolean;
  issuer: string;
  period: number;
  window_size: number;
  algorithm: string;
  digits: number;
}

export interface EmailVerificationConfig {
  enabled: boolean;
  fromAddress: string;
  fromName: string;
  maxAttempts: number;
  requireEmailProof: boolean;
  codeExpiry: any;
  codeLength: number;
  emailTemplate: string;
}

export interface CreatePolicyRequest {
  description: string;
  name: string;
  renewable: boolean;
  version: string;
  metadata: any;
  required: boolean;
  validityPeriod: number | undefined;
  consentType: string;
  content: string;
}

export interface ImpersonateUser_reqBody {
  duration?: any;
}

export interface ResetUserMFAResponse {
  devicesRevoked: number;
  factorsReset: number;
  message: string;
  success: boolean;
}

export interface RiskEngine {
}

export interface RecoverySession {
}

export interface WebhookResponse {
  received: boolean;
  status: string;
}

export interface DiscoveryService {
}

export interface ClientRegistrationResponse {
  client_id: string;
  contacts: string[];
  post_logout_redirect_uris: string[];
  scope: string;
  tos_uri: string;
  client_secret: string;
  logo_uri: string;
  client_name: string;
  client_secret_expires_at: number;
  application_type: string;
  client_id_issued_at: number;
  grant_types: string[];
  policy_uri: string;
  redirect_uris: string[];
  response_types: string[];
  token_endpoint_auth_method: string;
}

export interface ClientAuthenticator {
}

export interface UpdatePasskeyRequest {
  name: string;
}

export interface CookieConsentConfig {
  bannerVersion: string;
  categories: string[];
  defaultStyle: string;
  enabled: boolean;
  requireExplicit: boolean;
  validityPeriod: any;
  allowAnonymous: boolean;
}

export interface StripeIdentityProvider {
}

export interface Challenge {
  attempts: number;
  factorId: string;
  id: string;
  ipAddress: string;
  status: string;
  type: string;
  userAgent: string;
  userId: string;
  createdAt: string;
  expiresAt: string;
  maxAttempts: number;
  metadata: any;
  verifiedAt: string | undefined;
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

export interface ComplianceStatus {
  overallStatus: string;
  score: number;
  violations: number;
  checksFailed: number;
  lastChecked: string;
  nextAudit: string;
  profileId: string;
  standard: string;
  appId: string;
  checksPassed: number;
  checksWarning: number;
}

export interface RateLimitRule {
  max: number;
  window: any;
}

export interface StepUpRememberedDevice {
  id: string;
  org_id: string;
  user_agent: string;
  user_id: string;
  created_at: string;
  device_name: string;
  expires_at: string;
  ip: string;
  last_used_at: string;
  remembered_at: string;
  security_level: string;
  device_id: string;
}

export interface RouteRule {
  description: string;
  method: string;
  org_id: string;
  pattern: string;
  security_level: string;
}

export interface CreateTraining_req {
  standard: string;
  trainingType: string;
  userId: string;
}

export interface TemplateDefault {
}

export interface JumioProvider {
}

export interface EmailConfig {
  code_expiry_minutes: number;
  code_length: number;
  enabled: boolean;
  provider: string;
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
}

export interface AddCustomPermission_req {
  category: string;
  description: string;
  name: string;
}

export interface Email {
}

export interface UpdatePasskeyResponse {
  updatedAt: string;
  name: string;
  passkeyId: string;
}

export interface RateLimitConfig {
  redisDb: number;
  redisPassword: string;
  signinPerIp: RateLimitRule;
  signinPerUser: RateLimitRule;
  signupPerIp: RateLimitRule;
  useRedis: boolean;
  enabled: boolean;
  redisAddr: string;
}

export interface GenerateRecoveryCodesResponse {
  warning: string;
  codes: string[];
  count: number;
  generatedAt: string;
}

export interface VerifySecurityAnswersRequest {
  answers: any;
  sessionId: string;
}

export interface JWKSService {
}

export interface BackupCodeFactorAdapter {
}

export interface OIDCState {
}

export interface StartImpersonation_reqBody {
  duration_minutes?: number;
  reason: string;
  target_user_id: string;
  ticket_number?: string;
}

export interface DefaultProviderRegistry {
}

export interface ConsentRecord {
  createdAt: string;
  metadata: Record<string, any>;
  purpose: string;
  revokedAt: string | undefined;
  consentType: string;
  expiresAt: string | undefined;
  granted: boolean;
  id: string;
  updatedAt: string;
  userAgent: string;
  userId: string;
  ipAddress: string;
  organizationId: string;
  version: string;
  grantedAt: string;
}

export interface VideoVerificationConfig {
  enabled: boolean;
  minScheduleAdvance: any;
  recordSessions: boolean;
  recordingRetention: any;
  requireLivenessCheck: boolean;
  sessionDuration: any;
  livenessThreshold: number;
  provider: string;
  requireAdminReview: boolean;
  requireScheduling: boolean;
}

export interface ClientsListResponse {
  clients: ClientSummary[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface TokenIntrospectionRequest {
  client_secret: string;
  token: string;
  token_type_hint: string;
  client_id: string;
}

export interface stateEntry {
}

export interface TimeBasedRule {
  description: string;
  max_age: any;
  operation: string;
  org_id: string;
  security_level: string;
}

export interface ComplianceTemplateResponse {
  standard: string;
}

export interface ImpersonationStartResponse {
  impersonator_id: string;
  session_id: string;
  started_at: string;
  target_user_id: string;
}

export interface StepUpPolicy {
  priority: number;
  user_id: string;
  description: string;
  enabled: boolean;
  id: string;
  rules: any;
  updated_at: string;
  created_at: string;
  metadata: any;
  name: string;
  org_id: string;
}

export interface ComplianceChecksResponse {
  checks: any[];
}

export interface WebhookConfig {
  notify_on_expiring: boolean;
  notify_on_rate_limit: boolean;
  notify_on_rotated: boolean;
  webhook_urls: string[];
  enabled: boolean;
  expiry_warning_days: number;
  notify_on_created: boolean;
  notify_on_deleted: boolean;
}

export interface NoOpSMSProvider {
}

export interface CreateSessionRequest {
}

export interface StatsResponse {
  banned_users: number;
  timestamp: string;
  total_sessions: number;
  total_users: number;
  active_sessions: number;
  active_users: number;
}

export interface MemoryStateStore {
}

export interface RedisChallengeStore {
}

export interface Session {
  id: string;
  userId: string;
  token: string;
  expiresAt: string;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
}

export interface TrustDeviceRequest {
  deviceId: string;
  metadata: any;
  name: string;
}

export interface AppServiceAdapter {
}

export interface ListProfilesFilter {
  appId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface mockImpersonationRepository {
}

export interface ImpersonationSession {
}

export interface SuccessResponse {
  success: boolean;
}

export interface BackupAuthStatusResponse {
  status: string;
}

export interface StateStore {
}

export interface DataProcessingAgreement {
  id: string;
  organizationId: string;
  signedByName: string;
  signedByTitle: string;
  createdAt: string;
  digitalSignature: string;
  effectiveDate: string;
  expiryDate: string | undefined;
  ipAddress: string;
  status: string;
  updatedAt: string;
  content: string;
  signedBy: string;
  version: string;
  agreementType: string;
  metadata: Record<string, any>;
  signedByEmail: string;
}

export interface AMLMatch {
}

export interface AuthorizeRequest {
  acr_values: string;
  code_challenge: string;
  code_challenge_method: string;
  max_age: number | undefined;
  prompt: string;
  scope: string;
  state: string;
  client_id: string;
  id_token_hint: string;
  login_hint: string;
  nonce: string;
  redirect_uri: string;
  response_type: string;
  ui_locales: string;
}

export interface CallbackResult {
}

export interface ComplianceReportResponse {
  id: string;
}

export interface UpdatePolicyRequest {
  description: string;
  metadata: any;
  name: string;
  renewable: boolean | undefined;
  required: boolean | undefined;
  validityPeriod: number | undefined;
  active: boolean | undefined;
  content: string;
}

export interface SetUserRole_reqBody {
  role: string;
}

export interface SMSConfig {
  code_expiry_minutes: number;
  code_length: number;
  enabled: boolean;
  provider: string;
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
}

export interface TwoFAStatusResponse {
  trusted: boolean;
  enabled: boolean;
  method: string;
}

export interface ComplianceEvidence {
  appId: string;
  controlId: string;
  createdAt: string;
  description: string;
  evidenceType: string;
  fileHash: string;
  profileId: string;
  title: string;
  collectedBy: string;
  fileUrl: string;
  id: string;
  metadata: any;
  standard: string;
}

export interface ComplianceStatusResponse {
  status: string;
}

export interface MockAuditService {
}

export interface VerifyRecoveryCodeResponse {
  message: string;
  remainingCodes: number;
  valid: boolean;
}

export interface CookieConsent {
  userId: string;
  expiresAt: string;
  id: string;
  organizationId: string;
  personalization: boolean;
  sessionId: string;
  thirdParty: boolean;
  consentBannerVersion: string;
  createdAt: string;
  functional: boolean;
  ipAddress: string;
  marketing: boolean;
  analytics: boolean;
  updatedAt: string;
  essential: boolean;
  userAgent: string;
}

export interface InvitationResponse {
  invitation: any | undefined;
  message: string;
}

export interface AdminAddProviderRequest {
  clientId: string;
  clientSecret: string;
  enabled: boolean;
  provider: string;
  scopes: string[];
  appId: string;
}

export interface StatusResponse {
  status: string;
}

export interface ImpersonateUserRequest {
  duration: any;
  user_id: string;
  user_organization_id: string | undefined;
  app_id: string;
}

export interface EmailFactorAdapter {
}

export interface TestProvider_req {
  config: any;
  providerName: string;
  providerType: string;
  testRecipient: string;
}

export interface BackupAuthContactResponse {
  id: string;
}

export interface UserVerificationStatusResponse {
  status: any | undefined;
}

export interface mockProvider {
}

export interface SAMLLoginRequest {
  relayState: string;
}

export interface CreateTrainingRequest {
  standard: string;
  trainingType: string;
  userId: string;
}

export interface ConsentRecordResponse {
  id: string;
}

export interface GetChallengeStatusRequest {
}

export interface ListViolationsFilter {
  userId: string | undefined;
  violationType: string | undefined;
  appId: string | undefined;
  profileId: string | undefined;
  severity: string | undefined;
  status: string | undefined;
}

export interface Adapter {
}

export interface StartRecoveryRequest {
  preferredMethod: string;
  userId: string;
  deviceId: string;
  email: string;
}

export interface ConsentExportFileResponse {
  content_type: string;
  data: number[];
}

export interface ListSessionsResponse {
  limit: number;
  page: number;
  sessions: any | undefined[];
  total: number;
  total_pages: number;
}

export interface ListFactorsResponse {
  count: number;
  factors: Factor[];
}

export interface ConsentDeletionResponse {
  id: string;
  status: string;
}

export interface MembersResponse {
  members: any | undefined[];
  total: number;
}

export interface Webhook {
  id: string;
  organizationId: string;
  url: string;
  events: string[];
  secret: string;
  enabled: boolean;
  createdAt: string;
}

export interface StartVideoSessionResponse {
  expiresAt: string;
  message: string;
  sessionUrl: string;
  startedAt: string;
  videoSessionId: string;
}

export interface KeyStore {
}

export interface LimitResult {
}

export interface DevicesResponse {
  count: number;
  devices: any;
}

export interface AdminPolicyRequest {
  allowedTypes: string[];
  enabled: boolean;
  gracePeriod: number;
  requiredFactors: number;
}

export interface SetupSecurityQuestionsResponse {
  count: number;
  message: string;
  setupAt: string;
}

export interface VerifyCodeRequest {
  code: string;
  sessionId: string;
}

export interface ConsentReportResponse {
  id: string;
}

export interface ProviderListResponse {
  providers: ProviderInfo[];
  total: number;
}

export interface AssignRole_reqBody {
  roleID: string;
}

export interface BackupAuthStatsResponse {
  stats: any;
}

export interface RecoveryCodeUsage {
}

export interface ApproveRecoveryRequest {
  sessionId: string;
  notes: string;
}

export interface TOTPSecret {
}

export interface ReportsConfig {
  includeEvidence: boolean;
  retentionDays: number;
  schedule: string;
  storagePath: string;
  enabled: boolean;
  formats: string[];
}

export interface ComplianceViolation {
  userId: string;
  metadata: any;
  profileId: string;
  resolvedBy: string;
  violationType: string;
  appId: string;
  createdAt: string;
  description: string;
  id: string;
  resolvedAt: string | undefined;
  severity: string;
  status: string;
}

export interface ListEvidenceFilter {
  controlId: string | undefined;
  evidenceType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  appId: string | undefined;
}

export interface RiskAssessmentConfig {
  mediumRiskThreshold: number;
  newIpWeight: number;
  velocityWeight: number;
  blockHighRisk: boolean;
  highRiskThreshold: number;
  lowRiskThreshold: number;
  newDeviceWeight: number;
  newLocationWeight: number;
  requireReviewAbove: number;
  enabled: boolean;
  historyWeight: number;
}

export interface MetadataResponse {
  metadata: string;
}

export interface CreateEvidenceRequest {
  title: string;
  controlId: string;
  description: string;
  evidenceType: string;
  fileUrl: string;
  standard: string;
}

export interface ComplianceTemplate {
  auditFrequencyDays: number;
  dataResidency: string;
  description: string;
  mfaRequired: boolean;
  name: string;
  passwordMinLength: number;
  requiredPolicies: string[];
  requiredTraining: string[];
  retentionDays: number;
  sessionMaxAge: number;
  standard: string;
}

export interface mockUserService {
}

export interface TokenRevocationRequest {
  client_secret: string;
  token: string;
  token_type_hint: string;
  client_id: string;
}

export interface LinkResponse {
  message: string;
  user: any;
}

export interface ListChecksFilter {
  sinceBefore: string | undefined;
  status: string | undefined;
  appId: string | undefined;
  checkType: string | undefined;
  profileId: string | undefined;
}

export interface ComplianceReportsResponse {
  reports: any[];
}

export interface NotificationResponse {
  notification: any;
}

export interface NoOpNotificationProvider {
}

export interface RecoveryConfiguration {
}

export interface DataExportRequest {
  exportSize: number;
  format: string;
  includeSections: string[];
  updatedAt: string;
  errorMessage: string;
  exportUrl: string;
  id: string;
  createdAt: string;
  expiresAt: string | undefined;
  ipAddress: string;
  userId: string;
  completedAt: string | undefined;
  exportPath: string;
  organizationId: string;
  status: string;
}

export interface StripeIdentityConfig {
  webhookSecret: string;
  allowedTypes: string[];
  apiKey: string;
  enabled: boolean;
  requireLiveCapture: boolean;
  requireMatchingSelfie: boolean;
  returnUrl: string;
  useMock: boolean;
}

export interface JWTService {
}

export interface MockStateStore {
}

export interface GetDocumentVerificationRequest {
  documentId: string;
}

export interface VerificationResponse {
  verification: any | undefined;
}

export interface ListUsersRequest {
  page: number;
  role: string;
  search: string;
  status: string;
  user_organization_id: string | undefined;
  app_id: string;
  limit: number;
}

export interface MFAConfigResponse {
  allowed_factor_types: string[];
  enabled: boolean;
  required_factor_count: number;
}

export interface SetActive_body {
  id: string;
}

export interface ResolveViolationRequest {
  notes: string;
  resolution: string;
}

export interface BunRepository {
}

export interface ListFactorsRequest {
}

export interface EvaluationContext {
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

export interface ConsentTypeStatus {
  granted: boolean;
  grantedAt: string;
  needsRenewal: boolean;
  type: string;
  version: string;
  expiresAt: string | undefined;
}

export interface AuthURLResponse {
  url: string;
}

export interface MockEmailService {
}

export interface VerifyTrustedContactRequest {
  token: string;
}

export interface UserInfoResponse {
  birthdate: string;
  email_verified: boolean;
  family_name: string;
  locale: string;
  nickname: string;
  given_name: string;
  name: string;
  preferred_username: string;
  email: string;
  phone_number_verified: boolean;
  picture: string;
  website: string;
  zoneinfo: string;
  gender: string;
  middle_name: string;
  phone_number: string;
  profile: string;
  sub: string;
  updated_at: number;
}

export interface StepUpRequirement {
  status: string;
  user_id: string;
  id: string;
  required_level: string;
  rule_name: string;
  currency: string;
  fulfilled_at: string | undefined;
  metadata: any;
  org_id: string;
  resource_type: string;
  route: string;
  user_agent: string;
  created_at: string;
  current_level: string;
  ip: string;
  method: string;
  reason: string;
  session_id: string;
  amount: number;
  challenge_token: string;
  expires_at: string;
  resource_action: string;
  risk_score: number;
}

export interface TwoFAStatusDetailResponse {
  enabled: boolean;
  method: string;
  trusted: boolean;
}

export interface AuditEvent {
}

export interface ListPasskeysResponse {
  count: number;
  passkeys: PasskeyInfo[];
}

export interface NotificationErrorResponse {
  error: string;
}

export interface IDVerificationSessionResponse {
  session: any;
}

export interface KeyPair {
}

export interface ClientUpdateRequest {
  grant_types: string[];
  logo_uri: string;
  name: string;
  require_consent: boolean | undefined;
  response_types: string[];
  tos_uri: string;
  contacts: string[];
  policy_uri: string;
  post_logout_redirect_uris: string[];
  redirect_uris: string[];
  require_pkce: boolean | undefined;
  token_endpoint_auth_method: string;
  trusted_client: boolean | undefined;
  allowed_scopes: string[];
}

export interface VerifyRequest {
  code: string;
  email: string;
  phone: string;
  remember: boolean;
}

export interface AdminUpdateProviderRequest {
  scopes: string[];
  clientId: string | undefined;
  clientSecret: string | undefined;
  enabled: boolean | undefined;
}

export interface Status {
}

export interface SaveNotificationSettings_req {
  retryAttempts: number;
  retryDelay: string;
  autoSendWelcome: boolean;
  cleanupAfter: string;
}

export interface ErrorResponse {
  code: string;
  details: any;
  error: string;
  message: string;
}

export interface MockUserService {
}

export interface CreateEvidence_req {
  controlId: string;
  description: string;
  evidenceType: string;
  fileUrl: string;
  standard: string;
  title: string;
}

export interface ComplianceViolationResponse {
  id: string;
}

export interface AccountLockoutError {
}

export interface PrivacySettings {
  allowDataPortability: boolean;
  gdprMode: boolean;
  organizationId: string;
  updatedAt: string;
  anonymousConsentEnabled: boolean;
  dpoEmail: string;
  autoDeleteAfterDays: number;
  consentRequired: boolean;
  contactEmail: string;
  contactPhone: string;
  cookieConsentEnabled: boolean;
  cookieConsentStyle: string;
  dataExportExpiryHours: number;
  dataRetentionDays: number;
  ccpaMode: boolean;
  createdAt: string;
  deletionGracePeriodDays: number;
  exportFormat: string[];
  id: string;
  metadata: Record<string, any>;
  requireAdminApprovalForDeletion: boolean;
  requireExplicitConsent: boolean;
}

export interface RequestReverification_req {
  reason: string;
}

export interface StepUpDevicesResponse {
  count: number;
  devices: any;
}

export interface StepUpAuditLogsResponse {
  audit_logs: any[];
}

export interface GenerateBackupCodes_body {
  count: number;
  user_id: string;
}

export interface BackupAuthRecoveryResponse {
  session_id: string;
}

export interface ConsentStatusResponse {
  status: string;
}

export interface VerificationSessionResponse {
  session: any | undefined;
}

export interface TrustedDevicesConfig {
  max_devices_per_user: number;
  max_expiry_days: number;
  default_expiry_days: number;
  enabled: boolean;
}

export interface SMSFactorAdapter {
}

export interface SessionTokenResponse {
  session: any;
  token: string;
}

export interface RequestTrustedContactVerificationRequest {
  contactId: string;
  sessionId: string;
}

export interface ProviderSession {
}

export interface JWKS {
  keys: JWK[];
}

export interface VerifyFactor_req {
  code: string;
}

export interface RetentionConfig {
  archiveBeforePurge: boolean;
  archivePath: string;
  enabled: boolean;
  gracePeriodDays: number;
  purgeSchedule: string;
}

export interface RunCheck_req {
  checkType: string;
}

export interface ComplianceTemplatesResponse {
  templates: any[];
}

export interface ComplianceDashboardResponse {
  metrics: any;
}

export interface AccountLockedResponse {
  message: string;
  code: string;
  locked_minutes: number;
  locked_until: string;
}

export interface ConsentStats {
  expiredCount: number;
  grantRate: number;
  grantedCount: number;
  revokedCount: number;
  totalConsents: number;
  type: string;
  averageLifetime: number;
}

export interface RateLimiter {
}

export interface GetRecoveryStatsRequest {
  endDate: string;
  organizationId: string;
  startDate: string;
}

export interface DashboardConfig {
  enabled: boolean;
  path: string;
  showRecentChecks: boolean;
  showReports: boolean;
  showScore: boolean;
  showViolations: boolean;
}

export interface ChallengeRequest {
  context: string;
  factorTypes: string[];
  metadata: any;
  userId: string;
}

export interface ProviderInfo {
  providerId: string;
  type: string;
  createdAt: string;
  domain: string;
}

export interface ResourceRule {
  security_level: string;
  sensitivity: string;
  action: string;
  description: string;
  org_id: string;
  resource_type: string;
}

export interface SignInResponse {
  token: string;
  user: any | undefined;
  session: any | undefined;
}

export interface ContinueRecoveryResponse {
  data: any;
  expiresAt: string;
  instructions: string;
  method: string;
  sessionId: string;
  totalSteps: number;
  currentStep: number;
}

export interface PrivacySettingsRequest {
  anonymousConsentEnabled: boolean | undefined;
  ccpaMode: boolean | undefined;
  contactEmail: string;
  dpoEmail: string;
  gdprMode: boolean | undefined;
  requireExplicitConsent: boolean | undefined;
  deletionGracePeriodDays: number | undefined;
  autoDeleteAfterDays: number | undefined;
  consentRequired: boolean | undefined;
  contactPhone: string;
  cookieConsentEnabled: boolean | undefined;
  dataRetentionDays: number | undefined;
  allowDataPortability: boolean | undefined;
  cookieConsentStyle: string;
  dataExportExpiryHours: number | undefined;
  exportFormat: string[];
  requireAdminApprovalForDeletion: boolean | undefined;
}

export interface ConsentRequest {
  redirect_uri: string;
  response_type: string;
  scope: string;
  state: string;
  action: string;
  client_id: string;
  code_challenge: string;
  code_challenge_method: string;
}

export interface ListUsersResponse {
  limit: number;
  page: number;
  total: number;
  total_pages: number;
  users: any | undefined[];
}

export interface BaseFactorAdapter {
}

export interface ComplianceViolationsResponse {
  violations: any[];
}

export interface NotificationTemplateListResponse {
  templates: any[];
  total: number;
}

export interface VideoSessionInfo {
}

export interface MultiStepRecoveryConfig {
  allowStepSkip: boolean;
  allowUserChoice: boolean;
  highRiskSteps: string[];
  mediumRiskSteps: string[];
  sessionExpiry: any;
  enabled: boolean;
  lowRiskSteps: string[];
  minimumSteps: number;
  requireAdminApproval: boolean;
}

export interface DiscoveryResponse {
  revocation_endpoint_auth_methods_supported: string[];
  scopes_supported: string[];
  userinfo_endpoint: string;
  authorization_endpoint: string;
  claims_parameter_supported: boolean;
  introspection_endpoint_auth_methods_supported: string[];
  request_uri_parameter_supported: boolean;
  response_modes_supported: string[];
  subject_types_supported: string[];
  token_endpoint: string;
  code_challenge_methods_supported: string[];
  id_token_signing_alg_values_supported: string[];
  jwks_uri: string;
  registration_endpoint: string;
  request_parameter_supported: boolean;
  require_request_uri_registration: boolean;
  revocation_endpoint: string;
  introspection_endpoint: string;
  issuer: string;
  token_endpoint_auth_methods_supported: string[];
  claims_supported: string[];
  grant_types_supported: string[];
  response_types_supported: string[];
}

export interface PhoneVerifyResponse {
  session: any | undefined;
  token: string;
  user: any | undefined;
}

export interface auditServiceAdapter {
}

export interface ProvidersConfig {
  sms: SMSProviderConfig | undefined;
  email: EmailProviderConfig;
}

export interface ClientDetailsResponse {
  applicationType: string;
  requireConsent: boolean;
  responseTypes: string[];
  tosURI: string;
  updatedAt: string;
  grantTypes: string[];
  isOrgLevel: boolean;
  postLogoutRedirectURIs: string[];
  redirectURIs: string[];
  requirePKCE: boolean;
  tokenEndpointAuthMethod: string;
  createdAt: string;
  name: string;
  organizationID: string;
  trustedClient: boolean;
  clientID: string;
  contacts: string[];
  logoURI: string;
  policyURI: string;
  allowedScopes: string[];
}

export interface MockService {
}

export interface App {
}

export interface AutoCleanupConfig {
  enabled: boolean;
  interval: any;
}

export interface CreateSessionHTTPRequest {
  cancelUrl: string;
  config: any;
  metadata: any;
  provider: string;
  requiredChecks: string[];
  successUrl: string;
}

export interface JWK {
  kid: string;
  kty: string;
  n: string;
  use: string;
  alg: string;
  e: string;
}

export interface ProviderConfigResponse {
  provider: string;
  appId: string;
  message: string;
}

export interface ProviderRegisteredResponse {
  providerId: string;
  status: string;
  type: string;
}

export interface TwoFAEnableResponse {
  totp_uri: string;
  status: string;
}

export interface CreatePolicy_req {
  content: string;
  policyType: string;
  standard: string;
  title: string;
  version: string;
}

export interface CreateTemplateVersion_req {
  changes: string;
}

export interface TestSendTemplate_req {
  recipient: string;
  variables: any;
}

export interface SMSVerificationConfig {
  maxAttempts: number;
  maxSmsPerDay: number;
  messageTemplate: string;
  provider: string;
  codeExpiry: any;
  codeLength: number;
  cooldownPeriod: any;
  enabled: boolean;
}

export interface BackupAuthSessionsResponse {
  sessions: any[];
}

export interface AccessTokenClaims {
  scope: string;
  token_type: string;
  client_id: string;
}

export interface ConsentService {
}

export interface NotificationWebhookResponse {
  status: string;
}

export interface BackupAuthConfigResponse {
  config: any;
}

export interface ComplianceCheck {
  nextCheckAt: string;
  profileId: string;
  status: string;
  checkType: string;
  evidence: string[];
  id: string;
  result: any;
  appId: string;
  createdAt: string;
  lastCheckedAt: string;
}

export interface Device {
  ipAddress?: string;
  userAgent?: string;
  id: string;
  userId: string;
  name?: string;
  type?: string;
  lastUsedAt: string;
}

export interface RemoveTrustedContactRequest {
  contactId: string;
}

export interface GetRecoveryConfigResponse {
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
  riskScoreThreshold: number;
}

export interface ConsentSummary {
  totalConsents: number;
  userId: string;
  consentsByType: any;
  hasPendingDeletion: boolean;
  hasPendingExport: boolean;
  lastConsentUpdate: string | undefined;
  pendingRenewals: number;
  expiredConsents: number;
  grantedConsents: number;
  organizationId: string;
  revokedConsents: number;
}

export interface DataDeletionRequest {
  approvedAt: string | undefined;
  exemptionReason: string;
  requestReason: string;
  retentionExempt: boolean;
  status: string;
  updatedAt: string;
  approvedBy: string;
  deleteSections: string[];
  ipAddress: string;
  organizationId: string;
  rejectedAt: string | undefined;
  archivePath: string;
  completedAt: string | undefined;
  createdAt: string;
  errorMessage: string;
  id: string;
  userId: string;
}

export interface RevokeTrustedDeviceRequest {
}

export interface EmailServiceAdapter {
}

export interface UpdateProfileRequest {
  name: string | undefined;
  retentionDays: number | undefined;
  status: string | undefined;
  mfaRequired: boolean | undefined;
}

export interface DataDeletionConfig {
  allowPartialDeletion: boolean;
  archivePath: string;
  autoProcessAfterGrace: boolean;
  enabled: boolean;
  requireAdminApproval: boolean;
  retentionExemptions: string[];
  archiveBeforeDeletion: boolean;
  gracePeriodDays: number;
  notifyBeforeDeletion: boolean;
  preserveLegalData: boolean;
}

export interface AppHandler {
}

export interface EvaluationResult {
  reason: string;
  requirement_id: string;
  current_level: string;
  expires_at: string;
  required: boolean;
  security_level: string;
  allowed_methods: string[];
  can_remember: boolean;
  challenge_token: string;
  grace_period_ends_at: string;
  matched_rules: string[];
  metadata: any;
}

export interface RunCheckRequest {
  checkType: string;
}

export interface CreateDPARequest {
  agreementType: string;
  signedByEmail: string;
  signedByTitle: string;
  version: string;
  content: string;
  effectiveDate: string;
  expiryDate: string | undefined;
  metadata: any;
  signedByName: string;
}

export interface WebAuthnWrapper {
}

export interface ProvidersAppResponse {
  appId: string;
  providers: string[];
}

export interface RequirementsResponse {
  count: number;
  requirements: any;
}

export interface SendOTP_body {
  user_id: string;
}

export interface ImpersonationEndResponse {
  ended_at: string;
  status: string;
}

export interface SendRequest {
  email: string;
}

export interface Middleware {
}

export interface AddTrustedContactRequest {
  phone: string;
  relationship: string;
  email: string;
  name: string;
}

export interface SAMLLoginResponse {
  providerId: string;
  redirectUrl: string;
  requestId: string;
}

export interface GenerateRecoveryCodesRequest {
  count: number;
  format: string;
}

export interface Enable_body {
  method: string;
  user_id: string;
}

export interface CompliancePolicy {
  createdAt: string;
  id: string;
  title: string;
  version: string;
  standard: string;
  status: string;
  appId: string;
  approvedAt: string | undefined;
  content: string;
  profileId: string;
  reviewDate: string;
  updatedAt: string;
  policyType: string;
  approvedBy: string;
  effectiveDate: string;
  metadata: any;
}

export interface FinishRegisterRequest {
  name: string;
  response: any;
  userId: string;
}

export interface CreateAPIKey_reqBody {
  metadata?: any;
  name: string;
  permissions?: any;
  rate_limit?: number;
  scopes: string[];
  allowed_ips?: string[];
  description?: string;
}

export interface GetSecurityQuestionsRequest {
  sessionId: string;
}

export interface VerificationResult {
}

export interface ClientSummary {
  clientID: string;
  createdAt: string;
  isOrgLevel: boolean;
  name: string;
  applicationType: string;
}

export interface FactorInfo {
  factorId: string;
  metadata: any;
  name: string;
  type: string;
}

export interface FactorVerificationRequest {
  factorId: string;
  code: string;
  data: any;
}

export interface MFAStatus {
  requiredCount: number;
  trustedDevice: boolean;
  enabled: boolean;
  enrolledFactors: FactorInfo[];
  gracePeriod: string | undefined;
  policyActive: boolean;
}

export interface LoginResponse {
  user: any;
  passkeyUsed: string;
  session: any;
  token: string;
}

export interface User {
  organizationId?: string;
  id: string;
  email: string;
  name?: string;
  emailVerified: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface BlockUserRequest {
  reason: string;
}

export interface CreateVerificationRequest {
}

export interface IDVerificationResponse {
  verification: any;
}

export interface RiskAssessment {
  metadata: any;
  recommended: string[];
  score: number;
  factors: string[];
  level: string;
}

export interface FactorEnrollmentRequest {
  metadata: any;
  name: string;
  priority: string;
  type: string;
}

export interface ProvidersResponse {
  providers: string[];
}

export interface ListPasskeysRequest {
}

export interface RecoveryAttemptLog {
}

export interface RateLimitingConfig {
  maxAttemptsPerHour: number;
  maxAttemptsPerIp: number;
  enabled: boolean;
  exponentialBackoff: boolean;
  ipCooldownPeriod: any;
  lockoutAfterAttempts: number;
  lockoutDuration: any;
  maxAttemptsPerDay: number;
}

export interface ConsentExportResponse {
  id: string;
  status: string;
}

export interface ProviderSessionRequest {
}

export interface CallbackResponse {
  session: any | undefined;
  token: string;
  user: any | undefined;
}

export interface TemplateService {
}

export interface AddMember_req {
  role: string;
  user_id: string;
}

export interface UnbanUserRequest {
  reason: string;
  user_id: string;
  user_organization_id: string | undefined;
  app_id: string;
}

export interface MFAPolicyResponse {
  gracePeriodDays: number;
  id: string;
  organizationId: string | undefined;
  requiredFactorCount: number;
  allowedFactorTypes: string[];
  appId: string;
  enabled: boolean;
}

export interface StepUpVerificationResponse {
  expires_at: string;
  verified: boolean;
}

export interface ComplianceEvidenceResponse {
  id: string;
}

export interface PreviewTemplate_req {
  variables: any;
}

export interface CompleteRecoveryRequest {
  sessionId: string;
}

export interface TokenIntrospectionResponse {
  token_type: string;
  active: boolean;
  client_id: string;
  iat: number;
  iss: string;
  nbf: number;
  username: string;
  aud: string[];
  exp: number;
  jti: string;
  scope: string;
  sub: string;
}

export interface RegistrationService {
}

export interface Plugin {
}

export interface NotificationListResponse {
  notifications: any[];
  total: number;
}

export interface BeginRegisterRequest {
  requireResidentKey: boolean;
  userId: string;
  userVerification: string;
  authenticatorType: string;
  name: string;
}

export interface EmailProviderConfig {
  config: any;
  from: string;
  from_name: string;
  provider: string;
  reply_to: string;
}

export interface ConsentNotificationsConfig {
  channels: string[];
  notifyDeletionApproved: boolean;
  notifyExportReady: boolean;
  notifyOnExpiry: boolean;
  enabled: boolean;
  notifyDeletionComplete: boolean;
  notifyDpoEmail: string;
  notifyOnGrant: boolean;
  notifyOnRevoke: boolean;
}

export interface ChallengeResponse {
  expiresAt: string;
  factorsRequired: number;
  sessionId: string;
  availableFactors: FactorInfo[];
  challengeId: string;
}

export interface IntrospectionService {
}

export interface StepUpEvaluationResponse {
  reason: string;
  required: boolean;
}

export interface Handler {
}

export interface CompleteRecoveryResponse {
  completedAt: string;
  message: string;
  sessionId: string;
  status: string;
  token: string;
}

export interface mockRepository {
}

export interface TwoFASendOTPResponse {
  status: string;
  code: string;
}

export interface ListReportsFilter {
  reportType: string | undefined;
  standard: string | undefined;
  status: string | undefined;
  appId: string | undefined;
  format: string | undefined;
  profileId: string | undefined;
}

export interface RejectRecoveryResponse {
  message: string;
  reason: string;
  rejected: boolean;
  rejectedAt: string;
  sessionId: string;
}

export interface OTPSentResponse {
  code: string;
  status: string;
}

export interface TwoFABackupCodesResponse {
  codes: string[];
}

export interface RenderTemplate_req {
  template: string;
  variables: any;
}

export interface DashboardExtension {
}

export interface DataExportRequestInput {
  format: string;
  includeSections: string[];
}

export interface CreateUserRequest {
  app_id: string;
  email: string;
  email_verified: boolean;
  role: string;
  user_organization_id: string | undefined;
  username: string;
  metadata: any;
  name: string;
  password: string;
}

export interface CreateProfileRequest {
  dpoContact: string;
  encryptionAtRest: boolean;
  name: string;
  passwordExpiryDays: number;
  sessionIdleTimeout: number;
  sessionIpBinding: boolean;
  sessionMaxAge: number;
  complianceContact: string;
  dataResidency: string;
  metadata: any;
  mfaRequired: boolean;
  passwordRequireLower: boolean;
  standards: string[];
  encryptionInTransit: boolean;
  leastPrivilege: boolean;
  passwordMinLength: number;
  passwordRequireNumber: boolean;
  passwordRequireUpper: boolean;
  rbacRequired: boolean;
  regularAccessReview: boolean;
  retentionDays: number;
  appId: string;
  auditLogExport: boolean;
  passwordRequireSymbol: boolean;
  detailedAuditTrail: boolean;
}

export interface TeamHandler {
}

export interface ListTrustedDevicesResponse {
  count: number;
  devices: TrustedDevice[];
}

export interface LinkAccountRequest {
  provider: string;
  scopes: string[];
}

export interface OIDCLoginResponse {
  authUrl: string;
  nonce: string;
  providerId: string;
  state: string;
}

export interface SendResponse {
  dev_url: string;
  status: string;
}

export interface SendWithTemplateRequest {
  variables: any;
  appId: string;
  language: string;
  metadata: any;
  recipient: string;
  templateKey: string;
  type: any;
}

export interface AdminBlockUser_req {
  reason: string;
}

export interface BanUser_reqBody {
  expires_at?: string | undefined;
  reason: string;
}

export interface EnableRequest {
}

export interface VerifySecurityAnswersResponse {
  correctAnswers: number;
  message: string;
  requiredAnswers: number;
  valid: boolean;
  attemptsLeft: number;
}

export interface ConsentSettingsResponse {
  settings: any;
}

export interface SignInRequest {
  username: string;
  password: string;
  remember: boolean;
}

export interface NotificationsResponse {
  count: number;
  notifications: any;
}

export interface StartVideoSessionRequest {
  videoSessionId: string;
}

export interface WebhookPayload {
}

export interface ClientAuthResult {
}

export interface VerifyEnrolledFactorRequest {
  data: any;
  code: string;
}

export interface WebAuthnConfig {
  authenticator_selection: any;
  enabled: boolean;
  rp_display_name: string;
  rp_id: string;
  rp_origins: string[];
  timeout: number;
  attestation_preference: string;
}

export interface ConnectionsResponse {
  connections: any | undefined[];
}

export interface Verify_body {
  remember_device: boolean;
  user_id: string;
  code: string;
  device_id: string;
}

export interface SignUpRequest {
  password: string;
  username: string;
}

export interface BanUserRequest {
  user_organization_id: string | undefined;
  app_id: string;
  expires_at: string | undefined;
  reason: string;
  user_id: string;
}

export interface ComplianceTrainingResponse {
  id: string;
}

export interface CreateProfileFromTemplateRequest {
  standard: string;
}

export interface SetupSecurityQuestionsRequest {
  questions: SetupSecurityQuestionRequest[];
}

export interface ConsentPolicyResponse {
  id: string;
}

export interface SetUserRoleRequest {
  app_id: string;
  role: string;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface ComplianceCheckResponse {
  id: string;
}

export interface FinishLoginRequest {
  remember: boolean;
  response: any;
}

export interface CompleteVideoSessionRequest {
  livenessScore: number;
  notes: string;
  verificationResult: string;
  videoSessionId: string;
  livenessPassed: boolean;
}

export interface StateStorageConfig {
  redisAddr: string;
  redisDb: number;
  redisPassword: string;
  stateTtl: any;
  useRedis: boolean;
}

export interface StepUpErrorResponse {
  error: string;
}

export interface VerifyResponse {
  device_remembered: boolean;
  error: string;
  expires_at: string;
  metadata: any;
  security_level: string;
  success: boolean;
  verification_id: string;
}

export interface ListPoliciesFilter {
  appId: string | undefined;
  policyType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface DocumentVerificationConfig {
  retentionPeriod: any;
  storagePath: string;
  enabled: boolean;
  encryptionKey: string;
  minConfidenceScore: number;
  provider: string;
  requireManualReview: boolean;
  requireSelfie: boolean;
  storageProvider: string;
  acceptedDocuments: string[];
  encryptAtRest: boolean;
  requireBothSides: boolean;
}

export interface Factor {
  status: string;
  userId: string;
  createdAt: string;
  expiresAt: string | undefined;
  metadata: any;
  name: string;
  type: string;
  updatedAt: string;
  verifiedAt: string | undefined;
  id: string;
  lastUsedAt: string | undefined;
  priority: string;
}

export interface OIDCLoginRequest {
  nonce: string;
  redirectUri: string;
  scope: string;
  state: string;
}

export interface PolicyEngine {
}

export interface RecoverySessionInfo {
  createdAt: string;
  currentStep: number;
  id: string;
  totalSteps: number;
  completedAt: string | undefined;
  expiresAt: string;
  method: string;
  requiresReview: boolean;
  riskScore: number;
  status: string;
  userEmail: string;
  userId: string;
}

export interface ConsentAuditConfig {
  archiveInterval: any;
  archiveOldLogs: boolean;
  logAllChanges: boolean;
  retentionDays: number;
  signLogs: boolean;
  enabled: boolean;
  exportFormat: string;
  immutable: boolean;
  logIpAddress: boolean;
  logUserAgent: boolean;
}

export interface RiskContext {
}

export interface ContextRule {
  description: string;
  name: string;
  org_id: string;
  security_level: string;
  condition: string;
}

export interface ApproveRecoveryResponse {
  approved: boolean;
  approvedAt: string;
  message: string;
  sessionId: string;
}

export interface MFABypassResponse {
  userId: string;
  expiresAt: string;
  id: string;
  reason: string;
}

export interface DeleteFactorRequest {
}

export interface CompleteTrainingRequest {
  score: number;
}

export interface CompliancePolicyResponse {
  id: string;
}

export interface NotificationTemplateResponse {
  template: any;
}

export interface MessageResponse {
  message: string;
}

export interface ScheduleVideoSessionResponse {
  message: string;
  scheduledAt: string;
  videoSessionId: string;
  instructions: string;
  joinUrl: string;
}

export interface OnfidoConfig {
  enabled: boolean;
  includeFacialReport: boolean;
  includeWatchlistReport: boolean;
  webhookToken: string;
  facialCheck: FacialCheckConfig;
  includeDocumentReport: boolean;
  region: string;
  workflowId: string;
  apiToken: string;
  documentCheck: DocumentCheckConfig;
}

export interface TeamsResponse {
  teams: any | undefined[];
  total: number;
}

export interface MultiSessionListResponse {
  sessions: any[];
}

export interface NotificationStatusResponse {
  status: string;
}

export interface UserServiceAdapter {
}

export interface TrustedContact {
}

export interface ProviderCheckResult {
}

export interface CheckSubResult {
}

export interface UpdateFactorRequest {
  metadata: any;
  name: string | undefined;
  priority: string | undefined;
  status: string | undefined;
}

export interface ResetUserMFARequest {
  reason: string;
}

export interface ListRecoverySessionsRequest {
  pageSize: number;
  requiresReview: boolean;
  status: string;
  organizationId: string;
  page: number;
}

export interface ConsentCookieResponse {
  preferences: any;
}

export interface ReverifyRequest {
  reason: string;
}

export interface ForgetDeviceResponse {
  success: boolean;
  message: string;
}

export interface BeginLoginRequest {
  userId: string;
  userVerification: string;
}

export interface CreateAPIKeyResponse {
  api_key: any | undefined;
  message: string;
}

export interface TrustedContactsConfig {
  requireVerification: boolean;
  requiredToRecover: number;
  allowPhoneContacts: boolean;
  minimumContacts: number;
  verificationExpiry: any;
  allowEmailContacts: boolean;
  cooldownPeriod: any;
  enabled: boolean;
  maxNotificationsPerDay: number;
  maximumContacts: number;
}

export interface BackupAuthVideoResponse {
  session_id: string;
}

export interface TrustedDevice {
  id: string;
  ipAddress: string;
  metadata: any;
  name: string;
  createdAt: string;
  lastUsedAt: string | undefined;
  userAgent: string;
  userId: string;
  deviceId: string;
  expiresAt: string;
}

export interface ComplianceReport {
  fileSize: number;
  generatedBy: string;
  id: string;
  profileId: string;
  reportType: string;
  status: string;
  createdAt: string;
  fileUrl: string;
  format: string;
  period: string;
  standard: string;
  summary: any;
  appId: string;
  expiresAt: string;
}

export interface AuditLog {
}

export interface ChannelsResponse {
  channels: any;
  count: number;
}

export interface VideoVerificationSession {
}

export interface AdaptiveMFAConfig {
  risk_threshold: number;
  velocity_risk: number;
  factor_ip_reputation: boolean;
  factor_velocity: boolean;
  new_device_risk: number;
  require_step_up_threshold: number;
  enabled: boolean;
  factor_location_change: boolean;
  factor_new_device: boolean;
  location_change_risk: number;
}

export interface GetStatusRequest {
}

export interface Disable_body {
  user_id: string;
}

export interface CompleteTraining_req {
  score: number;
}

export interface ImpersonationVerifyResponse {
  impersonator_id: string;
  is_impersonating: boolean;
  target_user_id: string;
}

export interface GetSecurityQuestionsResponse {
  questions: SecurityQuestionInfo[];
}

export interface GetDocumentVerificationResponse {
  rejectionReason: string;
  status: string;
  verifiedAt: string | undefined;
  confidenceScore: number;
  documentId: string;
  message: string;
}

export interface DataDeletionRequestInput {
  reason: string;
  deleteSections: string[];
}

export interface ListSessionsRequest {
  page: number;
  user_id: string | undefined;
  user_organization_id: string | undefined;
  app_id: string;
  limit: number;
}

export interface EnrollFactorRequest {
  priority: string;
  type: string;
  metadata: any;
  name: string;
}

export interface ComplianceProfileResponse {
  id: string;
}

export interface ImpersonationErrorResponse {
  error: string;
}

export interface CompleteVideoSessionResponse {
  completedAt: string;
  message: string;
  result: string;
  videoSessionId: string;
}

export interface ScopeInfo {
}

export interface WebAuthnFactorAdapter {
}

export interface MultiSessionSetActiveResponse {
  session: any;
  token: string;
}

export interface GetPasskeyRequest {
}

export interface SecurityQuestion {
}

export interface UnblockUserRequest {
}

export interface PasskeyInfo {
  lastUsedAt: string | undefined;
  name: string;
  signCount: number;
  createdAt: string;
  id: string;
  isResidentKey: boolean;
  aaguid: string;
  authenticatorType: string;
  credentialId: string;
}

export interface RolesResponse {
  roles: any | undefined[];
}

export interface CancelRecoveryRequest {
  sessionId: string;
  reason: string;
}

export interface IDTokenClaims {
  email: string;
  family_name: string;
  given_name: string;
  name: string;
  preferred_username: string;
  auth_time: number;
  email_verified: boolean;
  nonce: string;
  session_state: string;
}

export interface SendCodeResponse {
  dev_code: string;
  status: string;
}

export interface MFAPolicy {
  maxFailedAttempts: number;
  organizationId: string;
  updatedAt: string;
  allowedFactorTypes: string[];
  gracePeriodDays: number;
  requiredFactorCount: number;
  requiredFactorTypes: string[];
  stepUpRequired: boolean;
  trustedDeviceDays: number;
  adaptiveMfaEnabled: boolean;
  createdAt: string;
  id: string;
  lockoutDurationMinutes: number;
}

export interface StepUpRequirementResponse {
  id: string;
}

export interface ComplianceProfile {
  dpoContact: string;
  passwordExpiryDays: number;
  regularAccessReview: boolean;
  encryptionAtRest: boolean;
  encryptionInTransit: boolean;
  dataResidency: string;
  mfaRequired: boolean;
  name: string;
  createdAt: string;
  id: string;
  rbacRequired: boolean;
  sessionIpBinding: boolean;
  leastPrivilege: boolean;
  metadata: any;
  passwordRequireLower: boolean;
  retentionDays: number;
  standards: string[];
  complianceContact: string;
  passwordRequireNumber: boolean;
  passwordRequireSymbol: boolean;
  updatedAt: string;
  appId: string;
  passwordMinLength: number;
  sessionIdleTimeout: number;
  sessionMaxAge: number;
  status: string;
  auditLogExport: boolean;
  detailedAuditTrail: boolean;
  passwordRequireUpper: boolean;
}

export interface RateLimit {
  max_requests: number;
  window: any;
}

export interface NotificationPreviewResponse {
  body: string;
  subject: string;
}

export interface NotificationChannels {
  email: boolean;
  slack: boolean;
  webhook: boolean;
}

export interface SignUpResponse {
  message: string;
  status: string;
}

export interface TOTPFactorAdapter {
}

export interface ImpersonationMiddleware {
}

export interface CreateABTestVariant_req {
  body: string;
  name: string;
  subject: string;
  weight: number;
}

export interface UploadDocumentRequest {
  backImage: string;
  documentType: string;
  frontImage: string;
  selfie: string;
  sessionId: string;
}

export interface ConnectionResponse {
  connection: any | undefined;
}

export interface StepUpStatusResponse {
  status: string;
}

export interface RotateAPIKeyResponse {
  api_key: any | undefined;
  message: string;
}

export interface AddTrustedContactResponse {
  message: string;
  name: string;
  phone: string;
  verified: boolean;
  addedAt: string;
  contactId: string;
  email: string;
}

export interface mockSessionService {
}

export interface MultiSessionErrorResponse {
  error: string;
}

export interface ComplianceTrainingsResponse {
  training: any[];
}

export interface MemoryChallengeStore {
}

export interface MemberHandler {
}

export interface JWKSResponse {
  keys: JWK[];
}

export interface ClientRegistrationRequest {
  client_name: string;
  grant_types: string[];
  response_types: string[];
  scope: string;
  application_type: string;
  contacts: string[];
  logo_uri: string;
  policy_uri: string;
  require_consent: boolean;
  trusted_client: boolean;
  token_endpoint_auth_method: string;
  tos_uri: string;
  post_logout_redirect_uris: string[];
  redirect_uris: string[];
  require_pkce: boolean;
}

export interface BackupAuthDocumentResponse {
  id: string;
}

export interface BackupAuthQuestionsResponse {
  questions: string[];
}

export interface AdminHandler {
}

export interface BackupCodesConfig {
  allow_reuse: boolean;
  count: number;
  enabled: boolean;
  format: string;
  length: number;
}

export interface CreateProvider_req {
  config: any;
  isDefault: boolean;
  organizationId?: string | undefined;
  providerName: string;
  providerType: string;
}

export interface VerifyCodeResponse {
  attemptsLeft: number;
  message: string;
  valid: boolean;
}

export interface TokenRequest {
  client_secret: string;
  code: string;
  code_verifier: string;
  grant_type: string;
  redirect_uri: string;
  refresh_token: string;
  audience: string;
  client_id: string;
  scope: string;
}

export interface FactorsResponse {
  count: number;
  factors: any;
}

export interface RedisStateStore {
}

export interface UserAdapter {
}

export interface ListRecoverySessionsResponse {
  page: number;
  pageSize: number;
  sessions: RecoverySessionInfo[];
  totalCount: number;
}

export interface DocumentVerification {
}

export interface HealthCheckResponse {
  enabledMethods: string[];
  healthy: boolean;
  message: string;
  providersStatus: any;
  version: string;
}

export interface FacialCheckConfig {
  variant: string;
  enabled: boolean;
  motionCapture: boolean;
}

export interface GetFactorRequest {
}

export interface SendVerificationCodeResponse {
  expiresAt: string;
  maskedTarget: string;
  message: string;
  sent: boolean;
}

export interface UpdateRecoveryConfigRequest {
  requireMultipleSteps: boolean;
  riskScoreThreshold: number;
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
}

export interface ConsentDashboardConfig {
  path: string;
  showAuditLog: boolean;
  showConsentHistory: boolean;
  showCookiePreferences: boolean;
  showDataDeletion: boolean;
  showDataExport: boolean;
  showPolicies: boolean;
  enabled: boolean;
}

export interface RiskFactor {
}

export interface UpdateProvider_req {
  config: any;
  isActive: boolean;
  isDefault: boolean;
}

export interface ListTrustedContactsResponse {
  contacts: TrustedContactInfo[];
  count: number;
}

export interface ConsentManager {
}

export interface SendCodeRequest {
  phone: string;
}

export interface VerifyChallengeRequest {
  code: string;
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
  challengeId: string;
}

export interface IDVerificationStatusResponse {
  status: any;
}

export interface IDVerificationWebhookResponse {
  status: string;
}

export interface DocumentCheckConfig {
  enabled: boolean;
  extractData: boolean;
  validateDataConsistency: boolean;
  validateExpiry: boolean;
}

export interface TwoFAErrorResponse {
  error: string;
}

export interface SendVerificationCodeRequest {
  method: string;
  sessionId: string;
  target: string;
}

export interface RegisterProviderRequest {
  oidcRedirectURI: string;
  samlEntryPoint: string;
  samlIssuer: string;
  providerId: string;
  samlCert: string;
  type: string;
  attributeMapping: any;
  domain: string;
  oidcClientID: string;
  oidcClientSecret: string;
  oidcIssuer: string;
}

export interface RejectRecoveryRequest {
  notes: string;
  reason: string;
  sessionId: string;
}

export interface Service {
}

export interface DocumentVerificationResult {
}

export interface DocumentVerificationRequest {
}

export interface ComplianceReportFileResponse {
  content_type: string;
  data: number[];
}

export interface VideoSessionResult {
}

export interface InitiateChallengeRequest {
  context: string;
  factorTypes: string[];
  metadata: any;
}

export interface StepUpRequirementsResponse {
  requirements: any[];
}

export interface SMSProviderConfig {
  config: any;
  from: string;
  provider: string;
}

export interface BeginRegisterResponse {
  challenge: string;
  options: any;
  timeout: any;
  userId: string;
}

export interface ConsentDecision {
}

export interface RevokeTokenService {
}

export interface MockAppService {
}

