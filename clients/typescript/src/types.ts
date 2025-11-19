// Auto-generated TypeScript types

export interface AdminAddProviderRequest {
  enabled: boolean;
  provider: string;
  scopes: string[];
  appId: string;
  clientId: string;
  clientSecret: string;
}

export interface TrustedContactsConfig {
  maxNotificationsPerDay: number;
  minimumContacts: number;
  requireVerification: boolean;
  requiredToRecover: number;
  verificationExpiry: any;
  allowEmailContacts: boolean;
  cooldownPeriod: any;
  maximumContacts: number;
  allowPhoneContacts: boolean;
  enabled: boolean;
}

export interface ConsentCookieResponse {
  preferences: any;
}

export interface ResetUserMFAResponse {
  success: boolean;
  devicesRevoked: number;
  factorsReset: number;
  message: string;
}

export interface UsernameStatusResponse {
  status: string;
}

export interface IDVerificationSessionResponse {
  session: any;
}

export interface SocialCallbackResponse {
  user: any;
  token: string;
}

export interface GetDocumentVerificationResponse {
  status: string;
  verifiedAt: string | undefined;
  confidenceScore: number;
  documentId: string;
  message: string;
  rejectionReason: string;
}

export interface MultiSessionSetActiveResponse {
  session: any;
  token: string;
}

export interface TrustedDevice {
  createdAt: string;
  deviceId: string;
  id: string;
  ipAddress: string;
  lastUsedAt: string | undefined;
  metadata: any;
  name: string;
  expiresAt: string;
  userAgent: string;
  userId: string;
}

export interface StartImpersonation_reqBody {
  duration_minutes?: number;
  reason: string;
  target_user_id: string;
  ticket_number?: string;
}

export interface Status {
}

export interface SMSFactorAdapter {
}

export interface SMSConfig {
  template_id: string;
  code_expiry_minutes: number;
  code_length: number;
  enabled: boolean;
  provider: string;
  rate_limit: RateLimitConfig | undefined;
}

export interface ComplianceTemplate {
  requiredPolicies: string[];
  retentionDays: number;
  auditFrequencyDays: number;
  mfaRequired: boolean;
  requiredTraining: string[];
  sessionMaxAge: number;
  standard: string;
  dataResidency: string;
  description: string;
  name: string;
  passwordMinLength: number;
}

export interface RequestReverification_req {
  reason: string;
}

export interface UpdateProvider_req {
  config: any;
  isActive: boolean;
  isDefault: boolean;
}

export interface RiskAssessment {
  factors: string[];
  level: string;
  metadata: any;
  recommended: string[];
  score: number;
}

export interface AdminListProvidersRequest {
  appId: string;
}

export interface DeclareABTestWinner_req {
  abTestGroup: string;
  winnerId: string;
}

export interface UnbanUser_reqBody {
  reason?: string;
}

export interface GetStatusRequest {
}

export interface BackupAuthCodesResponse {
  codes: string[];
}

export interface ContextRule {
  condition: string;
  description: string;
  name: string;
  org_id: string;
  security_level: string;
}

export interface SaveNotificationSettings_req {
  autoSendWelcome: boolean;
  cleanupAfter: string;
  retryAttempts: number;
  retryDelay: string;
}

export interface MembersResponse {
  members: any | undefined[];
  total: number;
}

export interface FactorInfo {
  metadata: any;
  name: string;
  type: string;
  factorId: string;
}

export interface DefaultProviderRegistry {
}

export interface AutoCleanupConfig {
  enabled: boolean;
  interval: any;
}

export interface mockImpersonationRepository {
}

export interface testRouter {
}

export interface OnfidoConfig {
  apiToken: string;
  documentCheck: DocumentCheckConfig;
  enabled: boolean;
  facialCheck: FacialCheckConfig;
  includeFacialReport: boolean;
  webhookToken: string;
  workflowId: string;
  includeDocumentReport: boolean;
  includeWatchlistReport: boolean;
  region: string;
}

export interface ContinueRecoveryRequest {
  method: string;
  sessionId: string;
}

export interface StepUpAuditLog {
  event_data: any;
  id: string;
  ip: string;
  org_id: string;
  severity: string;
  user_agent: string;
  user_id: string;
  event_type: string;
  created_at: string;
}

export interface StatsResponse {
  total_sessions: number;
  total_users: number;
  active_sessions: number;
  active_users: number;
  banned_users: number;
  timestamp: string;
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

export interface DashboardConfig {
  enabled: boolean;
  path: string;
  showRecentChecks: boolean;
  showReports: boolean;
  showScore: boolean;
  showViolations: boolean;
}

export interface ComplianceChecksResponse {
  checks: any[];
}

export interface AnonymousErrorResponse {
  error: string;
}

export interface RemoveTrustedContactRequest {
  contactId: string;
}

export interface SendWithTemplateRequest {
  appId: string;
  language: string;
  metadata: any;
  recipient: string;
  templateKey: string;
  type: any;
  variables: any;
}

export interface ImpersonationContext {
  impersonation_id: string | undefined;
  impersonator_id: string | undefined;
  indicator_message: string;
  is_impersonating: boolean;
  target_user_id: string | undefined;
}

export interface RateLimiter {
}

export interface NotificationResponse {
  notification: any;
}

export interface ConsentStatusResponse {
  status: string;
}

export interface SSOErrorResponse {
  error: string;
}

export interface AnonymousAuthResponse {
  session: any;
  token: string;
  user: any;
}

export interface BackupAuthRecoveryResponse {
  session_id: string;
}

export interface mockForgeContext {
}

export interface TokenRequest {
  grant_type: string;
  redirect_uri: string;
  client_id: string;
  client_secret: string;
  code: string;
  code_verifier: string;
}

export interface ComplianceViolation {
  metadata: any;
  profileId: string;
  resolvedAt: string | undefined;
  severity: string;
  violationType: string;
  id: string;
  resolvedBy: string;
  status: string;
  userId: string;
  appId: string;
  createdAt: string;
  description: string;
}

export interface SignInRequest {
  redirectUrl: string;
  scopes: string[];
  provider: string;
}

export interface OIDCErrorResponse {
  error: string;
  error_description: string;
}

export interface SecurityQuestionInfo {
  id: string;
  isCustom: boolean;
  questionId: number;
  questionText: string;
}

export interface RegisterClient_req {
  name: string;
  redirect_uri: string;
}

export interface ReportsConfig {
  formats: string[];
  includeEvidence: boolean;
  retentionDays: number;
  schedule: string;
  storagePath: string;
  enabled: boolean;
}

export interface GenerateRecoveryCodesRequest {
  count: number;
  format: string;
}

export interface ApproveRecoveryResponse {
  approved: boolean;
  approvedAt: string;
  message: string;
  sessionId: string;
}

export interface StepUpErrorResponse {
  error: string;
}

export interface RotateAPIKeyResponse {
  api_key: any | undefined;
  message: string;
}

export interface MultiSessionListResponse {
  sessions: any[];
}

export interface GenerateReport_req {
  reportType: string;
  standard: string;
  format: string;
  period: string;
}

export interface MockUserService {
}

export interface StripeIdentityProvider {
}

export interface MemberHandler {
}

export interface BackupAuthSessionsResponse {
  sessions: any[];
}

export interface CompliancePolicyResponse {
  id: string;
}

export interface BackupAuthQuestionsResponse {
  questions: string[];
}

export interface StepUpPoliciesResponse {
  policies: any[];
}

export interface ListUsersRequest {
  user_organization_id: string | undefined;
  app_id: string;
  limit: number;
  page: number;
  role: string;
  search: string;
  status: string;
}

export interface ChallengeRequest {
  factorTypes: string[];
  metadata: any;
  userId: string;
  context: string;
}

export interface ComplianceViolationResponse {
  id: string;
}

export interface VerifySecurityAnswersRequest {
  answers: any;
  sessionId: string;
}

export interface StepUpPolicy {
  name: string;
  org_id: string;
  updated_at: string;
  created_at: string;
  description: string;
  enabled: boolean;
  priority: number;
  rules: any;
  user_id: string;
  id: string;
  metadata: any;
}

export interface StepUpDevicesResponse {
  devices: any[];
}

export interface ChannelsResponse {
  channels: any;
  count: number;
}

export interface ConsentAuditLog {
  userAgent: string;
  userId: string;
  consentId: string;
  consentType: string;
  createdAt: string;
  id: string;
  ipAddress: string;
  organizationId: string;
  previousValue: Record<string, any>;
  purpose: string;
  action: string;
  newValue: Record<string, any>;
  reason: string;
}

export interface ConsentPolicyResponse {
  id: string;
}

export interface OIDCJWKSResponse {
  keys: any[];
}

export interface UnbanUserRequest {
  app_id: string;
  reason: string;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface RiskFactor {
}

export interface JWTService {
}

export interface PasskeyListResponse {
  passkeys: any[];
}

export interface ComplianceEvidencesResponse {
  evidence: any[];
}

export interface ProvidersConfig {
  bitbucket: any | undefined;
  dropbox: any | undefined;
  slack: any | undefined;
  twitch: any | undefined;
  apple: any | undefined;
  discord: any | undefined;
  line: any | undefined;
  facebook: any | undefined;
  gitlab: any | undefined;
  google: any | undefined;
  linkedin: any | undefined;
  microsoft: any | undefined;
  twitter: any | undefined;
  github: any | undefined;
  notion: any | undefined;
  reddit: any | undefined;
  spotify: any | undefined;
}

export interface StartRecoveryRequest {
  deviceId: string;
  email: string;
  preferredMethod: string;
  userId: string;
}

export interface NotificationWebhookResponse {
  status: string;
}

export interface ImpersonationSession {
}

export interface BackupCodesConfig {
  length: number;
  allow_reuse: boolean;
  count: number;
  enabled: boolean;
  format: string;
}

export interface BackupAuthDocumentResponse {
  id: string;
}

export interface EvaluationResult {
  required: boolean;
  requirement_id: string;
  security_level: string;
  allowed_methods: string[];
  challenge_token: string;
  expires_at: string;
  matched_rules: string[];
  reason: string;
  can_remember: boolean;
  current_level: string;
  grace_period_ends_at: string;
  metadata: any;
}

export interface PrivacySettings {
  createdAt: string;
  dataExportExpiryHours: number;
  metadata: Record<string, any>;
  requireAdminApprovalForDeletion: boolean;
  updatedAt: string;
  consentRequired: boolean;
  contactPhone: string;
  deletionGracePeriodDays: number;
  exportFormat: string[];
  allowDataPortability: boolean;
  contactEmail: string;
  cookieConsentStyle: string;
  dpoEmail: string;
  organizationId: string;
  requireExplicitConsent: boolean;
  anonymousConsentEnabled: boolean;
  autoDeleteAfterDays: number;
  ccpaMode: boolean;
  dataRetentionDays: number;
  gdprMode: boolean;
  id: string;
  cookieConsentEnabled: boolean;
}

export interface ConsentExportResponse {
  id: string;
  status: string;
}

export interface ImpersonationErrorResponse {
  error: string;
}

export interface CompleteTraining_req {
  score: number;
}

export interface IDVerificationWebhookResponse {
  status: string;
}

export interface Link_body {
  email: string;
  name: string;
  password: string;
}

export interface ListRecoverySessionsResponse {
  totalCount: number;
  page: number;
  pageSize: number;
  sessions: RecoverySessionInfo[];
}

export interface UploadDocumentRequest {
  documentType: string;
  frontImage: string;
  selfie: string;
  sessionId: string;
  backImage: string;
}

export interface TeamsResponse {
  teams: any | undefined[];
  total: number;
}

export interface CookieConsent {
  expiresAt: string;
  functional: boolean;
  marketing: boolean;
  analytics: boolean;
  id: string;
  organizationId: string;
  personalization: boolean;
  essential: boolean;
  thirdParty: boolean;
  userAgent: string;
  userId: string;
  consentBannerVersion: string;
  ipAddress: string;
  sessionId: string;
  updatedAt: string;
  createdAt: string;
}

export interface EnableRequest {
}

export interface ComplianceReportFileResponse {
  content_type: string;
  data: number[];
}

export interface FinishLogin_body {
  remember: boolean;
  user_id: string;
}

export interface DocumentVerificationResult {
}

export interface Challenge {
  createdAt: string;
  expiresAt: string;
  id: string;
  ipAddress: string;
  maxAttempts: number;
  metadata: any;
  status: string;
  type: string;
  attempts: number;
  factorId: string;
  userAgent: string;
  userId: string;
  verifiedAt: string | undefined;
}

export interface ComplianceCheck {
  profileId: string;
  result: any;
  status: string;
  appId: string;
  checkType: string;
  createdAt: string;
  lastCheckedAt: string;
  nextCheckAt: string;
  evidence: string[];
  id: string;
}

export interface VerifyRecoveryCodeRequest {
  code: string;
  sessionId: string;
}

export interface StepUpVerificationsResponse {
  verifications: any[];
}

export interface SMSProviderConfig {
  config: any;
  from: string;
  provider: string;
}

export interface Verify_body {
  code: string;
  device_id: string;
  remember_device: boolean;
  user_id: string;
}

export interface SocialStatusResponse {
  status: string;
}

export interface VideoVerificationConfig {
  enabled: boolean;
  livenessThreshold: number;
  minScheduleAdvance: any;
  provider: string;
  recordSessions: boolean;
  requireAdminReview: boolean;
  requireLivenessCheck: boolean;
  requireScheduling: boolean;
  recordingRetention: any;
  sessionDuration: any;
}

export interface StepUpRememberedDevice {
  security_level: string;
  user_id: string;
  created_at: string;
  device_id: string;
  id: string;
  ip: string;
  user_agent: string;
  device_name: string;
  expires_at: string;
  last_used_at: string;
  org_id: string;
  remembered_at: string;
}

export interface ErrorResponse {
  error: string;
  message: string;
}

export interface NotificationsResponse {
  count: number;
  notifications: any;
}

export interface TOTPFactorAdapter {
}

export interface ConsentRecord {
  createdAt: string;
  grantedAt: string;
  purpose: string;
  consentType: string;
  id: string;
  ipAddress: string;
  organizationId: string;
  updatedAt: string;
  userId: string;
  version: string;
  expiresAt: string | undefined;
  granted: boolean;
  metadata: Record<string, any>;
  revokedAt: string | undefined;
  userAgent: string;
}

export interface ConsentSummary {
  consentsByType: any;
  expiredConsents: number;
  hasPendingExport: boolean;
  lastConsentUpdate: string | undefined;
  revokedConsents: number;
  totalConsents: number;
  grantedConsents: number;
  hasPendingDeletion: boolean;
  organizationId: string;
  pendingRenewals: number;
  userId: string;
}

export interface SendOTP_body {
  user_id: string;
}

export interface RiskContext {
}

export interface ComplianceProfile {
  id: string;
  passwordRequireLower: boolean;
  rbacRequired: boolean;
  sessionIpBinding: boolean;
  sessionMaxAge: number;
  appId: string;
  auditLogExport: boolean;
  detailedAuditTrail: boolean;
  passwordRequireNumber: boolean;
  passwordRequireUpper: boolean;
  complianceContact: string;
  leastPrivilege: boolean;
  status: string;
  createdAt: string;
  encryptionAtRest: boolean;
  encryptionInTransit: boolean;
  passwordMinLength: number;
  regularAccessReview: boolean;
  metadata: any;
  name: string;
  retentionDays: number;
  dataResidency: string;
  passwordRequireSymbol: boolean;
  dpoContact: string;
  mfaRequired: boolean;
  standards: string[];
  passwordExpiryDays: number;
  sessionIdleTimeout: number;
  updatedAt: string;
}

export interface ProviderCheckResult {
}

export interface NoOpSMSProvider {
}

export interface CreateUserRequest {
  username: string;
  email: string;
  email_verified: boolean;
  name: string;
  password: string;
  app_id: string;
  metadata: any;
  role: string;
  user_organization_id: string | undefined;
}

export interface VerifyChallengeRequest {
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
  challengeId: string;
  code: string;
  data: any;
}

export interface TwoFARequiredResponse {
  require_twofa: boolean;
  user: any;
  device_id: string;
}

export interface ComplianceTrainingResponse {
  id: string;
}

export interface CallbackDataResponse {
  action: string;
  isNewUser: boolean;
  user: any;
}

export interface ProvidersAppResponse {
  appId: string;
  providers: any;
}

export interface SetupSecurityQuestionRequest {
  answer: string;
  customText: string;
  questionId: number;
}

export interface VerifyTrustedContactRequest {
  token: string;
}

export interface DataDeletionRequestInput {
  deleteSections: string[];
  reason: string;
}

export interface mockRepository {
}

export interface AddTeamMember_req {
  member_id: string;
  role: string;
}

export interface StepUpRequirement {
  current_level: string;
  id: string;
  ip: string;
  org_id: string;
  rule_name: string;
  user_agent: string;
  expires_at: string;
  method: string;
  risk_score: number;
  status: string;
  amount: number;
  created_at: string;
  fulfilled_at: string | undefined;
  metadata: any;
  reason: string;
  required_level: string;
  route: string;
  session_id: string;
  challenge_token: string;
  resource_action: string;
  resource_type: string;
  user_id: string;
  currency: string;
}

export interface MultiSessionDeleteResponse {
  status: string;
}

export interface TwoFABackupCodesResponse {
  codes: string[];
}

export interface PhoneErrorResponse {
  error: string;
}

export interface ComplianceDashboardResponse {
  metrics: any;
}

export interface SSOSAMLMetadataResponse {
  metadata: string;
}

export interface ConsentTypeStatus {
  type: string;
  version: string;
  expiresAt: string | undefined;
  granted: boolean;
  grantedAt: string;
  needsRenewal: boolean;
}

export interface EmailServiceAdapter {
}

export interface NotificationListResponse {
  notifications: any[];
  total: number;
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

export interface ImpersonateUser_reqBody {
  duration?: any;
}

export interface StepUpRequirementsResponse {
  requirements: any[];
}

export interface SuccessResponse {
  success: boolean;
}

export interface Status_body {
  device_id: string;
  user_id: string;
}

export interface MFASession {
  id: string;
  riskLevel: string;
  sessionToken: string;
  userId: string;
  verifiedFactors: string[];
  completedAt: string | undefined;
  createdAt: string;
  factorsRequired: number;
  ipAddress: string;
  metadata: any;
  userAgent: string;
  expiresAt: string;
  factorsVerified: number;
}

export interface ListReportsFilter {
  format: string | undefined;
  profileId: string | undefined;
  reportType: string | undefined;
  standard: string | undefined;
  status: string | undefined;
  appId: string | undefined;
}

export interface CreateVerificationSession_req {
  requiredChecks: string[];
  successUrl: string;
  cancelUrl: string;
  config: any;
  metadata: any;
  provider: string;
}

export interface EmailProviderConfig {
  config: any;
  from: string;
  from_name: string;
  provider: string;
  reply_to: string;
}

export interface AccessTokenClaims {
  client_id: string;
  scope: string;
  token_type: string;
}

export interface EmailOTPVerifyResponse {
  session: any;
  token: string;
  user: any;
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

export interface MagicLinkSendResponse {
  dev_url: string;
  status: string;
}

export interface TwoFAStatusDetailResponse {
  enabled: boolean;
  method: string;
  trusted: boolean;
}

export interface NotificationTemplateListResponse {
  total: number;
  templates: any[];
}

export interface MessageResponse {
  message: string;
}

export interface VerifyTrustedContactResponse {
  contactId: string;
  message: string;
  verified: boolean;
  verifiedAt: string;
}

export interface ConsentReport {
  reportPeriodStart: string;
  usersWithConsent: number;
  dpasActive: number;
  dpasExpiringSoon: number;
  pendingDeletions: number;
  reportPeriodEnd: string;
  totalUsers: number;
  completedDeletions: number;
  consentRate: number;
  consentsByType: any;
  dataExportsThisPeriod: number;
  organizationId: string;
}

export interface AutomatedChecksConfig {
  accessReview: boolean;
  checkInterval: any;
  enabled: boolean;
  mfaCoverage: boolean;
  passwordPolicy: boolean;
  dataRetention: boolean;
  inactiveUsers: boolean;
  sessionPolicy: boolean;
  suspiciousActivity: boolean;
}

export interface CreateProfileRequest {
}

export interface ListChecksFilter {
  sinceBefore: string | undefined;
  status: string | undefined;
  appId: string | undefined;
  checkType: string | undefined;
  profileId: string | undefined;
}

export interface DataDeletionRequest {
  ipAddress: string;
  requestReason: string;
  updatedAt: string;
  approvedAt: string | undefined;
  completedAt: string | undefined;
  deleteSections: string[];
  errorMessage: string;
  rejectedAt: string | undefined;
  approvedBy: string;
  archivePath: string;
  createdAt: string;
  id: string;
  organizationId: string;
  status: string;
  userId: string;
  exemptionReason: string;
  retentionExempt: boolean;
}

export interface ConsentsResponse {
  consents: any;
  count: number;
}

export interface SendCode_body {
  phone: string;
}

export interface VerificationsResponse {
  count: number;
  verifications: any;
}

export interface IPWhitelistConfig {
  enabled: boolean;
  strict_mode: boolean;
}

export interface Middleware {
}

export interface MockEmailService {
}

export interface ComplianceEvidence {
  profileId: string;
  standard: string;
  collectedBy: string;
  description: string;
  evidenceType: string;
  id: string;
  title: string;
  appId: string;
  controlId: string;
  createdAt: string;
  fileHash: string;
  fileUrl: string;
  metadata: any;
}

export interface ComplianceStatusDetailsResponse {
  status: string;
}

export interface ScheduleVideoSessionRequest {
  scheduledAt: string;
  sessionId: string;
  timeZone: string;
}

export interface auditServiceAdapter {
}

export interface RenderTemplate_req {
  template: string;
  variables: any;
}

export interface AppHandler {
}

export interface ConnectionsResponse {
  connections: any;
}

export interface TemplateService {
}

export interface InitiateChallengeRequest {
  context: string;
  factorTypes: string[];
  metadata: any;
}

export interface UsernameSignInResponse {
  session: any;
  token: string;
  user: any;
}

export interface GetRecoveryStatsResponse {
  adminReviewsRequired: number;
  averageRiskScore: number;
  methodStats: any;
  pendingRecoveries: number;
  successRate: number;
  successfulRecoveries: number;
  totalAttempts: number;
  failedRecoveries: number;
  highRiskAttempts: number;
}

export interface ConsentRecordResponse {
  id: string;
}

export interface SignUp_body {
  password: string;
  username: string;
}

export interface ImpersonationVerifyResponse {
  is_impersonating: boolean;
  target_user_id: string;
  impersonator_id: string;
}

export interface EmailOTPSendResponse {
  dev_otp: string;
  status: string;
}

export interface LimitResult {
}

export interface TeamHandler {
}

export interface HealthCheckResponse {
  providersStatus: any;
  version: string;
  enabledMethods: string[];
  healthy: boolean;
  message: string;
}

export interface EvaluateRequest {
  currency: string;
  metadata: any;
  method: string;
  resource_type: string;
  route: string;
  action: string;
  amount: number;
}

export interface mockSessionService {
}

export interface AuditLog {
}

export interface TrustedContact {
}

export interface RolesResponse {
  roles: any | undefined[];
}

export interface BeginLogin_body {
  user_id: string;
}

export interface JumioProvider {
}

export interface AddMember_req {
  role: string;
  user_id: string;
}

export interface SSOProviderResponse {
  providerId: string;
  status: string;
}

export interface mockUserService {
}

export interface PasskeyLoginResponse {
  user: any;
  session: any;
  token: string;
}

export interface ComplianceReport {
  status: string;
  summary: any;
  appId: string;
  expiresAt: string;
  fileSize: number;
  format: string;
  id: string;
  period: string;
  standard: string;
  createdAt: string;
  fileUrl: string;
  generatedBy: string;
  profileId: string;
  reportType: string;
}

export interface Email {
}

export interface UpdateProfileRequest {
}

export interface LinkAccountRequest {
  provider: string;
  scopes: string[];
}

export interface StepUpVerificationResponse {
  expires_at: string;
  verified: boolean;
}

export interface CallbackResult {
}

export interface CompleteRecoveryRequest {
  sessionId: string;
}

export interface UploadDocumentResponse {
  documentId: string;
  message: string;
  processingTime: string;
  status: string;
  uploadedAt: string;
}

export interface MagicLinkErrorResponse {
  error: string;
}

export interface AuthorizeRequest {
}

export interface AdminBypassRequest {
  duration: number;
  reason: string;
  userId: string;
}

export interface MultiStepRecoveryConfig {
  allowStepSkip: boolean;
  allowUserChoice: boolean;
  mediumRiskSteps: string[];
  minimumSteps: number;
  enabled: boolean;
  highRiskSteps: string[];
  lowRiskSteps: string[];
  requireAdminApproval: boolean;
  sessionExpiry: any;
}

export interface StepUpEvaluationResponse {
  reason: string;
  required: boolean;
}

export interface DevicesResponse {
  devices: any;
  count: number;
}

export interface FinishRegister_body {
  credential_id: string;
  user_id: string;
}

export interface UsernameErrorResponse {
  error: string;
}

export interface MockOrganizationService {
}

export interface CompleteVideoSessionResponse {
  completedAt: string;
  message: string;
  result: string;
  videoSessionId: string;
}

export interface RecoveryCodeUsage {
}

export interface RateLimitConfig {
  enabled: boolean;
  window: any;
}

export interface ImpersonationStartResponse {
  impersonator_id: string;
  session_id: string;
  started_at: string;
  target_user_id: string;
}

export interface EmailOTPErrorResponse {
  error: string;
}

export interface SMSVerificationConfig {
  codeLength: number;
  cooldownPeriod: any;
  enabled: boolean;
  maxAttempts: number;
  maxSmsPerDay: number;
  messageTemplate: string;
  provider: string;
  codeExpiry: any;
}

export interface ConsentPolicy {
  active: boolean;
  createdBy: string;
  id: string;
  name: string;
  publishedAt: string | undefined;
  renewable: boolean;
  organizationId: string;
  version: string;
  updatedAt: string;
  content: string;
  metadata: Record<string, any>;
  validityPeriod: number | undefined;
  consentType: string;
  createdAt: string;
  description: string;
  required: boolean;
}

export interface SetActive_body {
  id: string;
}

export interface testContext {
}

export interface CompliancePoliciesResponse {
  policies: any[];
}

export interface CompliancePolicy {
  createdAt: string;
  profileId: string;
  reviewDate: string;
  status: string;
  title: string;
  appId: string;
  policyType: string;
  standard: string;
  version: string;
  content: string;
  id: string;
  metadata: any;
  updatedAt: string;
  approvedAt: string | undefined;
  approvedBy: string;
  effectiveDate: string;
}

export interface NoOpEmailProvider {
}

export interface AdminUpdateProviderRequest {
  clientId: string | undefined;
  clientSecret: string | undefined;
  enabled: boolean | undefined;
  scopes: string[];
}

export interface VerifyEnrolledFactorRequest {
  code: string;
  data: any;
}

export interface StartVideoSessionRequest {
  videoSessionId: string;
}

export interface VideoSessionInfo {
}

export interface NoOpDocumentProvider {
}

export interface DataExportRequest {
  createdAt: string;
  errorMessage: string;
  exportSize: number;
  exportUrl: string;
  userId: string;
  completedAt: string | undefined;
  expiresAt: string | undefined;
  format: string;
  updatedAt: string;
  exportPath: string;
  ipAddress: string;
  organizationId: string;
  status: string;
  id: string;
  includeSections: string[];
}

export interface RetentionConfig {
  archiveBeforePurge: boolean;
  archivePath: string;
  enabled: boolean;
  gracePeriodDays: number;
  purgeSchedule: string;
}

export interface IDVerificationListResponse {
  verifications: any[];
}

export interface PreviewTemplate_req {
  variables: any;
}

export interface GetFactorRequest {
}

export interface DeleteFactorRequest {
}

export interface RejectRecoveryResponse {
  message: string;
  reason: string;
  rejected: boolean;
  rejectedAt: string;
  sessionId: string;
}

export interface TemplateEngine {
}

export interface CreateABTestVariant_req {
  body: string;
  name: string;
  subject: string;
  weight: number;
}

export interface CreatePolicy_req {
  content: string;
  policyType: string;
  standard: string;
  title: string;
  version: string;
}

export interface StepUpRequirementResponse {
  id: string;
}

export interface BackupCodeFactorAdapter {
}

export interface BeginRegister_body {
  user_id: string;
}

export interface PhoneSendCodeResponse {
  dev_code: string;
  status: string;
}

export interface ContinueRecoveryResponse {
  sessionId: string;
  totalSteps: number;
  currentStep: number;
  data: any;
  expiresAt: string;
  instructions: string;
  method: string;
}

export interface GenerateBackupCodes_body {
  count: number;
  user_id: string;
}

export interface AdminPolicyRequest {
  allowedTypes: string[];
  enabled: boolean;
  gracePeriod: number;
  requiredFactors: number;
}

export interface CreateProfileFromTemplate_req {
  standard: string;
  appId: string;
}

export interface StepUpAttempt {
  created_at: string;
  failure_reason: string;
  id: string;
  method: string;
  org_id: string;
  requirement_id: string;
  success: boolean;
  ip: string;
  user_agent: string;
  user_id: string;
}

export interface Plugin {
}

export interface Config {
  allow_query_param: boolean;
  default_expiry: any;
  key_length: number;
  max_keys_per_org: number;
  auto_cleanup: AutoCleanupConfig;
  default_rate_limit: number;
  ip_whitelisting: IPWhitelistConfig;
  max_keys_per_user: number;
  max_rate_limit: number;
  rate_limiting: RateLimitConfig;
  webhooks: WebhookConfig;
}

export interface AddTrustedContactResponse {
  addedAt: string;
  contactId: string;
  email: string;
  message: string;
  name: string;
  phone: string;
  verified: boolean;
}

export interface CreateProvider_req {
  config: any;
  isDefault: boolean;
  organizationId?: string | undefined;
  providerName: string;
  providerType: string;
}

export interface WebAuthnConfig {
  timeout: number;
  attestation_preference: string;
  authenticator_selection: any;
  enabled: boolean;
  rp_display_name: string;
  rp_id: string;
  rp_origins: string[];
}

export interface SocialSignInResponse {
  redirect_url: string;
}

export interface MetadataResponse {
  metadata: string;
}

export interface Send_body {
  email: string;
}

export interface InvitationResponse {
  invitation: any | undefined;
  message: string;
}

export interface ListFactorsRequest {
}

export interface VerificationResponse {
  expiresAt: string | undefined;
  factorsRemaining: number;
  sessionComplete: boolean;
  success: boolean;
  token: string;
}

export interface KeyPair {
}

export interface PasskeyErrorResponse {
  error: string;
}

export interface PhoneVerifyResponse {
  session: any;
  token: string;
  user: any;
}

export interface DataProcessingAgreement {
  signedByName: string;
  signedByTitle: string;
  digitalSignature: string;
  effectiveDate: string;
  organizationId: string;
  signedBy: string;
  updatedAt: string;
  agreementType: string;
  signedByEmail: string;
  content: string;
  createdAt: string;
  ipAddress: string;
  metadata: Record<string, any>;
  version: string;
  expiryDate: string | undefined;
  id: string;
  status: string;
}

export interface TwoFASendOTPResponse {
  code: string;
  status: string;
}

export interface AdaptiveMFAConfig {
  velocity_risk: number;
  enabled: boolean;
  factor_ip_reputation: boolean;
  factor_location_change: boolean;
  new_device_risk: number;
  require_step_up_threshold: number;
  risk_threshold: number;
  factor_new_device: boolean;
  factor_velocity: boolean;
  location_change_risk: number;
}

export interface CallbackResponse {
  session: any;
  token: string;
  user: any;
}

export interface ProvidersResponse {
  providers: any;
}

export interface ReviewDocumentRequest {
  documentId: string;
  notes: string;
  rejectionReason: string;
  approved: boolean;
}

export interface VerifyRecoveryCodeResponse {
  message: string;
  remainingCodes: number;
  valid: boolean;
}

export interface ConsentAuditLogsResponse {
  audit_logs: any[];
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

export interface SendVerificationCodeRequest {
  method: string;
  sessionId: string;
  target: string;
}

export interface DocumentVerificationConfig {
  enabled: boolean;
  encryptAtRest: boolean;
  requireSelfie: boolean;
  retentionPeriod: any;
  storageProvider: string;
  encryptionKey: string;
  minConfidenceScore: number;
  provider: string;
  requireBothSides: boolean;
  requireManualReview: boolean;
  storagePath: string;
  acceptedDocuments: string[];
}

export interface CompleteVideoSessionRequest {
  livenessPassed: boolean;
  livenessScore: number;
  notes: string;
  verificationResult: string;
  videoSessionId: string;
}

export interface BackupAuthStatusResponse {
  status: string;
}

export interface ConsentAuditConfig {
  archiveOldLogs: boolean;
  enabled: boolean;
  exportFormat: string;
  immutable: boolean;
  logAllChanges: boolean;
  retentionDays: number;
  signLogs: boolean;
  archiveInterval: any;
  logIpAddress: boolean;
  logUserAgent: boolean;
}

export interface ListEvidenceFilter {
  appId: string | undefined;
  controlId: string | undefined;
  evidenceType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
}

export interface IDVerificationErrorResponse {
  error: string;
}

export interface GetRecoveryConfigResponse {
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
  riskScoreThreshold: number;
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

export interface BaseFactorAdapter {
}

export interface AuditEvent {
}

export interface StartRecoveryResponse {
  requiresReview: boolean;
  riskScore: number;
  sessionId: string;
  status: string;
  availableMethods: string[];
  completedSteps: number;
  expiresAt: string;
  requiredSteps: number;
}

export interface AssignRole_reqBody {
  roleID: string;
}

export interface CreateAPIKeyResponse {
  api_key: any | undefined;
  message: string;
}

export interface CreateDPARequest {
  content: string;
  expiryDate: string | undefined;
  signedByTitle: string;
  version: string;
  agreementType: string;
  effectiveDate: string;
  metadata: any;
  signedByEmail: string;
  signedByName: string;
}

export interface ComplianceTemplatesResponse {
  templates: any[];
}

export interface ConsentExportFileResponse {
  content_type: string;
  data: number[];
}

export interface VerificationRequest {
  challengeId: string;
  code: string;
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
}

export interface FactorEnrollmentRequest {
  metadata: any;
  name: string;
  priority: string;
  type: string;
}

export interface NotificationChannels {
  email: boolean;
  slack: boolean;
  webhook: boolean;
}

export interface PolicyEngine {
}

export interface ComplianceTemplateResponse {
  standard: string;
}

export interface RiskAssessmentConfig {
  enabled: boolean;
  lowRiskThreshold: number;
  mediumRiskThreshold: number;
  newIpWeight: number;
  requireReviewAbove: number;
  velocityWeight: number;
  blockHighRisk: boolean;
  highRiskThreshold: number;
  historyWeight: number;
  newDeviceWeight: number;
  newLocationWeight: number;
}

export interface DocumentVerificationRequest {
}

export interface DataDeletionConfig {
  allowPartialDeletion: boolean;
  archivePath: string;
  autoProcessAfterGrace: boolean;
  enabled: boolean;
  gracePeriodDays: number;
  archiveBeforeDeletion: boolean;
  notifyBeforeDeletion: boolean;
  preserveLegalData: boolean;
  requireAdminApproval: boolean;
  retentionExemptions: string[];
}

export interface StepUpVerification {
  id: string;
  ip: string;
  org_id: string;
  session_id: string;
  user_agent: string;
  device_id: string;
  expires_at: string;
  metadata: any;
  reason: string;
  verified_at: string;
  user_id: string;
  created_at: string;
  method: string;
  rule_name: string;
  security_level: string;
}

export interface WebhookConfig {
  notify_on_rotated: boolean;
  webhook_urls: string[];
  enabled: boolean;
  expiry_warning_days: number;
  notify_on_created: boolean;
  notify_on_deleted: boolean;
  notify_on_expiring: boolean;
  notify_on_rate_limit: boolean;
}

export interface Webhook {
  secret: string;
  enabled: boolean;
  createdAt: string;
  id: string;
  organizationId: string;
  url: string;
  events: string[];
}

export interface DocumentCheckConfig {
  enabled: boolean;
  extractData: boolean;
  validateDataConsistency: boolean;
  validateExpiry: boolean;
}

export interface ResourceRule {
  description: string;
  org_id: string;
  resource_type: string;
  security_level: string;
  sensitivity: string;
  action: string;
}

export interface ImpersonationMiddleware {
}

export interface ComplianceUserTrainingResponse {
  user_id: string;
}

export interface ListPoliciesFilter {
  appId: string | undefined;
  policyType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface ScopeInfo {
}

export interface OTPSentResponse {
  code: string;
  status: string;
}

export interface RiskEngine {
}

export interface ScheduleVideoSessionResponse {
  joinUrl: string;
  message: string;
  scheduledAt: string;
  videoSessionId: string;
  instructions: string;
}

export interface Factor {
  priority: string;
  status: string;
  type: string;
  updatedAt: string;
  userId: string;
  verifiedAt: string | undefined;
  createdAt: string;
  expiresAt: string | undefined;
  id: string;
  lastUsedAt: string | undefined;
  metadata: any;
  name: string;
}

export interface GenerateRecoveryCodesResponse {
  codes: string[];
  count: number;
  generatedAt: string;
  warning: string;
}

export interface SignIn_body {
  code: string;
  email: string;
  phone: string;
  remember: boolean;
}

export interface AppServiceAdapter {
}

export interface ComplianceReportResponse {
  id: string;
}

export interface ComplianceProfileResponse {
  id: string;
}

export interface TrustedContactInfo {
  verified: boolean;
  verifiedAt: string | undefined;
  active: boolean;
  email: string;
  id: string;
  name: string;
  phone: string;
  relationship: string;
}

export interface GetRecoveryStatsRequest {
  endDate: string;
  organizationId: string;
  startDate: string;
}

export interface ConsentSettingsResponse {
  settings: any;
}

export interface ListSessionsRequest {
  user_organization_id: string | undefined;
  app_id: string;
  limit: number;
  page: number;
  user_id: string | undefined;
}

export interface EmailFactorAdapter {
}

export interface StripeIdentityConfig {
  allowedTypes: string[];
  apiKey: string;
  enabled: boolean;
  requireLiveCapture: boolean;
  requireMatchingSelfie: boolean;
  returnUrl: string;
  useMock: boolean;
  webhookSecret: string;
}

export interface RecoveryAttemptLog {
}

export interface TokenResponse {
  scope: string;
  token_type: string;
  access_token: string;
  expires_in: number;
  id_token: string;
  refresh_token: string;
}

export interface ResetUserMFARequest {
  reason: string;
}

export interface FactorAdapterRegistry {
}

export interface ListProfilesFilter {
  appId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface BackupAuthStatsResponse {
  stats: any;
}

export interface ConsentDashboardConfig {
  showCookiePreferences: boolean;
  showDataDeletion: boolean;
  showDataExport: boolean;
  showPolicies: boolean;
  enabled: boolean;
  path: string;
  showAuditLog: boolean;
  showConsentHistory: boolean;
}

export interface MagicLinkVerifyResponse {
  token: string;
  user: any;
  session: any;
}

export interface TwoFAErrorResponse {
  error: string;
}

export interface AuthResponse {
  session: any;
  token: string;
  user: any;
}

export interface SignInResponse {
  token: string;
  user: any;
  session: any;
}

export interface IDVerificationStatusResponse {
  status: any;
}

export interface UpdateRecoveryConfigRequest {
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
  riskScoreThreshold: number;
}

export interface EvaluationContext {
}

export interface RunCheck_req {
  checkType: string;
}

export interface ComplianceCheckResponse {
  id: string;
}

export interface ComplianceViolationsResponse {
  violations: any[];
}

export interface ProviderConfigResponse {
  message: string;
  provider: string;
  appId: string;
}

export interface TwoFAStatusResponse {
  status: string;
}

export interface ComplianceTraining {
  appId: string;
  expiresAt: string | undefined;
  id: string;
  profileId: string;
  score: number;
  standard: string;
  status: string;
  trainingType: string;
  completedAt: string | undefined;
  createdAt: string;
  metadata: any;
  userId: string;
}

export interface ComplianceEvidenceResponse {
  id: string;
}

export interface BackupAuthConfigResponse {
  config: any;
}

export interface CompleteRecoveryResponse {
  completedAt: string;
  message: string;
  sessionId: string;
  status: string;
  token: string;
}

export interface JWKS {
  keys: JWK[];
}

export interface OIDCTokenResponse {
  access_token: string;
  expires_in: number;
  id_token: string;
  refresh_token: string;
  scope: string;
  token_type: string;
}

export interface ConnectionResponse {
  connection: any;
}

export interface SessionTokenResponse {
  session: any;
  token: string;
}

export interface DashboardExtension {
}

export interface OrganizationHandler {
}

export interface GetChallengeStatusRequest {
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
}

export interface OIDCClientResponse {
  client_id: string;
  client_secret: string;
  redirect_uris: string[];
}

export interface MFAConfigResponse {
  allowed_factor_types: string[];
  enabled: boolean;
  required_factor_count: number;
}

export interface AuditServiceAdapter {
}

export interface ProviderSessionRequest {
}

export interface WebhookPayload {
}

export interface SSOInitResponse {
  redirect_url: string;
  request_id: string;
}

export interface NoOpNotificationProvider {
}

export interface TrustDeviceRequest {
  deviceId: string;
  metadata: any;
  name: string;
}

export interface TOTPConfig {
  enabled: boolean;
  issuer: string;
  period: number;
  window_size: number;
  algorithm: string;
  digits: number;
}

export interface PasskeyStatusResponse {
  status: string;
}

export interface AmountRule {
  security_level: string;
  currency: string;
  description: string;
  max_amount: number;
  min_amount: number;
  org_id: string;
}

export interface ConsentReportResponse {
  id: string;
}

export interface VerifyRequest {
}

export interface TestProvider_req {
  config: any;
  providerName: string;
  providerType: string;
  testRecipient: string;
}

export interface ImpersonationEndResponse {
  status: string;
  ended_at: string;
}

export interface JWK {
  kid: string;
  kty: string;
  n: string;
  use: string;
  alg: string;
  e: string;
}

export interface FactorsResponse {
  count: number;
  factors: any;
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

export interface BackupAuthVideoResponse {
  session_id: string;
}

export interface NotificationErrorResponse {
  error: string;
}

export interface TwoFAEnableResponse {
  status: string;
  totp_uri: string;
}

export interface NotificationPreviewResponse {
  body: string;
  subject: string;
}

export interface OIDCUserInfoResponse {
  sub: string;
  email: string;
  email_verified: boolean;
  family_name: string;
  given_name: string;
  name: string;
  picture: string;
}

export interface AddTrustedContactRequest {
  email: string;
  name: string;
  phone: string;
  relationship: string;
}

export interface VideoSessionResult {
}

export interface CreatePolicyRequest {
  content: string;
  metadata: any;
  name: string;
  validityPeriod: number | undefined;
  consentType: string;
  description: string;
  renewable: boolean;
  required: boolean;
  version: string;
}

export interface ConsentDecision {
}

export interface mockProvider {
}

export interface OAuthState {
}

export interface DocumentVerification {
}

export interface ListRecoverySessionsRequest {
  organizationId: string;
  page: number;
  pageSize: number;
  requiresReview: boolean;
  status: string;
}

export interface CodesResponse {
  codes: string[];
}

export interface ImpersonateUserRequest {
  app_id: string;
  duration: any;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface ChallengeResponse {
  factorsRequired: number;
  sessionId: string;
  availableFactors: FactorInfo[];
  challengeId: string;
  expiresAt: string;
}

export interface CreateTemplateVersion_req {
  changes: string;
}

export interface EndImpersonation_reqBody {
  impersonation_id: string;
  reason?: string;
}

export interface FactorVerificationRequest {
  code: string;
  data: any;
  factorId: string;
}

export interface UpdatePolicy_req {
  content: string | undefined;
  status: string | undefined;
  title: string | undefined;
  version: string | undefined;
}

export interface AMLMatch {
}

export interface SocialLinkResponse {
  linked: boolean;
}

export interface UpdateConsentRequest {
  granted: boolean | undefined;
  metadata: any;
  reason: string;
}

export interface ConsentDeletionResponse {
  id: string;
  status: string;
}

export interface SessionsResponse {
  sessions: any;
}

export interface ListTrainingFilter {
  trainingType: string | undefined;
  userId: string | undefined;
  appId: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface SendVerificationCodeResponse {
  expiresAt: string;
  maskedTarget: string;
  message: string;
  sent: boolean;
}

export interface ListTrustedContactsResponse {
  contacts: TrustedContactInfo[];
  count: number;
}

export interface Adapter {
}

export interface UpdatePolicyRequest {
  renewable: boolean | undefined;
  required: boolean | undefined;
  validityPeriod: number | undefined;
  active: boolean | undefined;
  content: string;
  description: string;
  metadata: any;
  name: string;
}

export interface IDTokenClaims {
  nonce: string;
  preferred_username: string;
  session_state: string;
  email_verified: boolean;
  name: string;
  auth_time: number;
  email: string;
  family_name: string;
  given_name: string;
}

export interface Disable_body {
  user_id: string;
}

export interface VerifyFactor_req {
  code: string;
}

export interface ComplianceTrainingsResponse {
  training: any[];
}

export interface SocialErrorResponse {
  error: string;
}

export interface AddCustomPermission_req {
  category: string;
  description: string;
  name: string;
}

export interface MFAPolicy {
  updatedAt: string;
  allowedFactorTypes: string[];
  createdAt: string;
  id: string;
  maxFailedAttempts: number;
  organizationId: string;
  requiredFactorCount: number;
  stepUpRequired: boolean;
  adaptiveMfaEnabled: boolean;
  gracePeriodDays: number;
  lockoutDurationMinutes: number;
  requiredFactorTypes: string[];
  trustedDeviceDays: number;
}

export interface OnfidoProvider {
}

export interface EmailVerificationConfig {
  fromName: string;
  maxAttempts: number;
  requireEmailProof: boolean;
  codeExpiry: any;
  codeLength: number;
  emailTemplate: string;
  enabled: boolean;
  fromAddress: string;
}

export interface ApproveRecoveryRequest {
  notes: string;
  sessionId: string;
}

export interface TemplateDefault {
}

export interface ListSessionsResponse {
  limit: number;
  page: number;
  sessions: any | undefined[];
  total: number;
  total_pages: number;
}

export interface CheckSubResult {
}

export interface VerifyCodeRequest {
  code: string;
  sessionId: string;
}

export interface RequestTrustedContactVerificationRequest {
  contactId: string;
  sessionId: string;
}

export interface Handler {
}

export interface MFAStatus {
  requiredCount: number;
  trustedDevice: boolean;
  enabled: boolean;
  enrolledFactors: FactorInfo[];
  gracePeriod: string | undefined;
  policyActive: boolean;
}

export interface CancelRecoveryRequest {
  reason: string;
  sessionId: string;
}

export interface TrackNotificationEvent_req {
  templateId: string;
  event: string;
  eventData?: any;
  notificationId: string;
  organizationId?: string | undefined;
}

export interface DataExportRequestInput {
  format: string;
  includeSections: string[];
}

export interface StatusResponse {
  status: string;
}

export interface EnrollFactorRequest {
  name: string;
  priority: string;
  type: string;
  metadata: any;
}

export interface TimeBasedRule {
  security_level: string;
  description: string;
  max_age: any;
  operation: string;
  org_id: string;
}

export interface NotificationStatusResponse {
  status: string;
}

export interface ConsentExpiryConfig {
  renewalReminderDays: number;
  requireReConsent: boolean;
  allowRenewal: boolean;
  autoExpireCheck: boolean;
  defaultValidityDays: number;
  enabled: boolean;
  expireCheckInterval: any;
}

export interface ConsentNotificationsConfig {
  notifyOnExpiry: boolean;
  notifyOnRevoke: boolean;
  channels: string[];
  enabled: boolean;
  notifyDeletionApproved: boolean;
  notifyDeletionComplete: boolean;
  notifyDpoEmail: string;
  notifyExportReady: boolean;
  notifyOnGrant: boolean;
}

export interface CreateEvidence_req {
  controlId: string;
  description: string;
  evidenceType: string;
  fileUrl: string;
  standard: string;
  title: string;
}

export interface AuthURLResponse {
  url: string;
}

export interface RecoveryConfiguration {
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

export interface BunRepository {
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

export interface DocumentTypesResponse {
  document_types: string[];
}

export interface RequestTrustedContactVerificationResponse {
  notifiedAt: string;
  contactId: string;
  contactName: string;
  expiresAt: string;
  message: string;
}

export interface DataExportConfig {
  allowedFormats: string[];
  defaultFormat: string;
  enabled: boolean;
  maxRequests: number;
  requestPeriod: any;
  autoCleanup: boolean;
  cleanupInterval: any;
  expiryHours: number;
  includeSections: string[];
  maxExportSize: number;
  storagePath: string;
}

export interface SetUserRole_reqBody {
  role: string;
}

export interface VerificationResult {
}

export interface RegisterProvider_req {
  OIDCClientSecret: string;
  OIDCIssuer: string;
  domain: string;
  providerId: string;
  OIDCClientID: string;
  OIDCRedirectURI: string;
  SAMLCert: string;
  SAMLEntryPoint: string;
  SAMLIssuer: string;
  type: string;
}

export interface VerifySecurityAnswersResponse {
  attemptsLeft: number;
  correctAnswers: number;
  message: string;
  requiredAnswers: number;
  valid: boolean;
}

export interface GetDocumentVerificationRequest {
  documentId: string;
}

export interface VerifyCodeResponse {
  attemptsLeft: number;
  message: string;
  valid: boolean;
}

export interface App {
}

export interface UserServiceAdapter {
}

export interface JumioConfig {
  enabled: boolean;
  enabledDocumentTypes: string[];
  presetId: string;
  enableAMLScreening: boolean;
  enableExtraction: boolean;
  enabledCountries: string[];
  verificationType: string;
  apiSecret: string;
  apiToken: string;
  callbackUrl: string;
  dataCenter: string;
  enableLiveness: boolean;
}

export interface BackupAuthContactResponse {
  id: string;
}

export interface GetSecurityQuestionsRequest {
  sessionId: string;
}

export interface VerifyResponse {
  session: any;
  token: string;
  user: any;
}

export interface TOTPSecret {
}

export interface RecoveryCodesConfig {
  regenerateCount: number;
  allowDownload: boolean;
  allowPrint: boolean;
  autoRegenerate: boolean;
  codeCount: number;
  codeLength: number;
  enabled: boolean;
  format: string;
}

export interface MockRepository {
}

export interface EmailConfig {
  provider: string;
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
  code_expiry_minutes: number;
  code_length: number;
  enabled: boolean;
}

export interface ListTrustedDevicesResponse {
  count: number;
  devices: TrustedDevice[];
}

export interface AuditConfig {
  minRetentionDays: number;
  signLogs: boolean;
  detailedTrail: boolean;
  exportFormat: string;
  immutable: boolean;
  maxRetentionDays: number;
}

export interface AdminBlockUser_req {
  reason: string;
}

export interface RecoverySession {
}

export interface BanUserRequest {
  app_id: string;
  expires_at: string | undefined;
  reason: string;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface DeviceInfo {
  deviceId: string;
  metadata: any;
  name: string;
}

export interface StepUpPolicyResponse {
  id: string;
}

export interface ComplianceStatus {
  lastChecked: string;
  overallStatus: string;
  score: number;
  standard: string;
  checksFailed: number;
  checksPassed: number;
  nextAudit: string;
  profileId: string;
  violations: number;
  appId: string;
  checksWarning: number;
}

export interface SSOSAMLCallbackResponse {
  issuer: string;
  providerId: string;
  status: string;
  subject: string;
  attributes: any;
}

export interface PrivacySettingsRequest {
  ccpaMode: boolean | undefined;
  dpoEmail: string;
  deletionGracePeriodDays: number | undefined;
  allowDataPortability: boolean | undefined;
  contactPhone: string;
  cookieConsentEnabled: boolean | undefined;
  cookieConsentStyle: string;
  dataExportExpiryHours: number | undefined;
  dataRetentionDays: number | undefined;
  exportFormat: string[];
  gdprMode: boolean | undefined;
  requireAdminApprovalForDeletion: boolean | undefined;
  requireExplicitConsent: boolean | undefined;
  anonymousConsentEnabled: boolean | undefined;
  autoDeleteAfterDays: number | undefined;
  consentRequired: boolean | undefined;
  contactEmail: string;
}

export interface KeyStore {
}

export interface RevokeTrustedDeviceRequest {
}

export interface ComplianceStatusResponse {
  status: string;
}

export interface SocialProvidersResponse {
  providers: string[];
}

export interface JWKSService {
}

export interface FacialCheckConfig {
  enabled: boolean;
  motionCapture: boolean;
  variant: string;
}

export interface MockService {
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

export interface CreateTraining_req {
  standard: string;
  trainingType: string;
  userId: string;
}

export interface StepUpAuditLogsResponse {
  audit_logs: any[];
}

export interface SetUserRoleRequest {
  app_id: string;
  role: string;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface ListFactorsResponse {
  count: number;
  factors: Factor[];
}

export interface TrustedDevicesConfig {
  default_expiry_days: number;
  enabled: boolean;
  max_devices_per_user: number;
  max_expiry_days: number;
}

export interface Username2FARequiredResponse {
  device_id: string;
  require_twofa: boolean;
  user: any;
}

export interface AnonymousSignInResponse {
  session: any;
  token: string;
}

export interface Enable_body {
  method: string;
  user_id: string;
}

export interface RouteRule {
  method: string;
  org_id: string;
  pattern: string;
  security_level: string;
  description: string;
}

export interface TemplatesResponse {
  count: number;
  templates: any;
}

export interface ProviderSession {
}

export interface UpdateFactorRequest {
  metadata: any;
  name: string | undefined;
  priority: string | undefined;
  status: string | undefined;
}

export interface VideoVerificationSession {
}

export interface SecurityQuestionsConfig {
  lockoutDuration: any;
  maxAnswerLength: number;
  requireMinLength: number;
  requiredToRecover: number;
  enabled: boolean;
  forbidCommonAnswers: boolean;
  maxAttempts: number;
  minimumQuestions: number;
  predefinedQuestions: string[];
  allowCustomQuestions: boolean;
  caseSensitive: boolean;
}

export interface StepUpStatusResponse {
  status: string;
}

export interface FactorEnrollmentResponse {
  status: string;
  type: string;
  factorId: string;
  provisioningData: any;
}

export interface GetSecurityQuestionsResponse {
  questions: SecurityQuestionInfo[];
}

export interface Service {
}

export interface PasskeyLoginOptionsResponse {
  options: any;
}

export interface CreateConsentRequest {
  metadata: any;
  purpose: string;
  userId: string;
  version: string;
  consentType: string;
  expiresIn: number | undefined;
  granted: boolean;
}

export interface ListUsersResponse {
  limit: number;
  page: number;
  total: number;
  total_pages: number;
  users: any | undefined[];
}

export interface ComplianceReportsResponse {
  reports: any[];
}

export interface NoOpVideoProvider {
}

export interface RecoverySessionInfo {
  totalSteps: number;
  userEmail: string;
  userId: string;
  createdAt: string;
  method: string;
  requiresReview: boolean;
  status: string;
  completedAt: string | undefined;
  currentStep: number;
  expiresAt: string;
  id: string;
  riskScore: number;
}

export interface RateLimit {
  max_requests: number;
  window: any;
}

export interface WebAuthnFactorAdapter {
}

export interface SetupSecurityQuestionsResponse {
  count: number;
  message: string;
  setupAt: string;
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

export interface RejectRecoveryRequest {
  notes: string;
  reason: string;
  sessionId: string;
}

export interface OIDCConfigResponse {
  config: any;
  issuer: string;
}

export interface MultiSessionErrorResponse {
  error: string;
}

export interface BanUser_reqBody {
  expires_at?: string | undefined;
  reason: string;
}

export interface NotificationsConfig {
  notifyOwners: boolean;
  violations: boolean;
  auditReminders: boolean;
  channels: NotificationChannels;
  enabled: boolean;
  failedChecks: boolean;
  notifyComplianceContact: boolean;
}

export interface CreateSessionRequest {
}

export interface CreateVerificationRequest {
}

export interface SecurityQuestion {
}

export interface SetupSecurityQuestionsRequest {
  questions: SetupSecurityQuestionRequest[];
}

export interface BackupAuthContactsResponse {
  contacts: any[];
}

export interface PasskeyRegistrationOptionsResponse {
  options: any;
}

export interface NotificationTemplateResponse {
  template: any;
}

export interface TestSendTemplate_req {
  recipient: string;
  variables: any;
}

