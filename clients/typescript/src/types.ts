// Auto-generated TypeScript types

export interface ConsentStats {
  grantRate: number;
  grantedCount: number;
  revokedCount: number;
  totalConsents: number;
  type: string;
  averageLifetime: number;
  expiredCount: number;
}

export interface BackupCodeFactorAdapter {
}

export interface VerifyChallengeRequest {
  code: string;
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
  challengeId: string;
}

export interface CancelInvitationInput {
  orgId: string;
  appId: string;
  inviteId: string;
}

export interface ConsentDeletionResponse {
  id: string;
  status: string;
}

export interface RiskFactor {
}

export interface UpdateSettingsInput {
  appId: string;
  settings: OrganizationSettingsDTO;
}

export interface DeleteAPIKeyRequest {
}

export interface CreatePolicy_req {
  content: string;
  policyType: string;
  standard: string;
  title: string;
  version: string;
}

export interface StepUpVerification {
  expires_at: string;
  method: string;
  rule_name: string;
  user_id: string;
  created_at: string;
  id: string;
  ip: string;
  security_level: string;
  user_agent: string;
  verified_at: string;
  device_id: string;
  reason: string;
  session_id: string;
  metadata: any;
  org_id: string;
}

export interface MFAPolicy {
  maxFailedAttempts: number;
  organizationId: string;
  requiredFactorCount: number;
  requiredFactorTypes: string[];
  updatedAt: string;
  allowedFactorTypes: string[];
  gracePeriodDays: number;
  id: string;
  stepUpRequired: boolean;
  trustedDeviceDays: number;
  adaptiveMfaEnabled: boolean;
  createdAt: string;
  lockoutDurationMinutes: number;
}

export interface PolicyPreviewResponse {
  actions: string[];
  description: string;
  expression: string;
  name: string;
  resourceType: string;
}

export interface FactorPriority {
  [key: string]: any;
}

export interface StepUpAuditLog {
  created_at: string;
  id: string;
  org_id: string;
  user_agent: string;
  user_id: string;
  event_data: any;
  event_type: string;
  ip: string;
  severity: string;
}

export interface UpdateTeamHandlerRequest {
}

export interface AuthAutoSendConfig {
  magic_link: boolean;
  mfa_code: boolean;
  password_reset: boolean;
  verification_email: boolean;
  welcome: boolean;
  email_otp: boolean;
}

export interface ListNotificationsHistoryResult {
  notifications: NotificationHistoryDTO[];
  pagination: PaginationDTO;
}

export interface TemplateAnalyticsDTO {
  deliveryRate: number;
  templateId: string;
  templateName: string;
  clickRate: number;
  openRate: number;
  totalClicked: number;
  totalDelivered: number;
  totalOpened: number;
  totalSent: number;
}

export interface GetProvidersResult {
  providers: ProvidersConfigDTO;
}

export interface ComplianceEvidenceResponse {
  id: string;
}

export interface DiscoveryService {
}

export interface ClientRegistrationResponse {
  application_type: string;
  client_id_issued_at: number;
  grant_types: string[];
  response_types: string[];
  tos_uri: string;
  client_secret_expires_at: number;
  scope: string;
  client_id: string;
  contacts: string[];
  policy_uri: string;
  token_endpoint_auth_method: string;
  client_name: string;
  client_secret: string;
  logo_uri: string;
  post_logout_redirect_uris: string[];
  redirect_uris: string[];
}

export interface ConsentAuditLogsResponse {
  audit_logs: any[];
}

export interface StartImpersonationRequest {
  duration_minutes: number;
  reason: string;
  target_user_id: string;
  ticket_number: string;
}

export interface SendVerificationCodeResponse {
  message: string;
  sent: boolean;
  expiresAt: string;
  maskedTarget: string;
}

export interface IDVerificationResponse {
  verification: any;
}

export interface GetOverviewStatsResult {
  stats: OverviewStatsDTO;
}

export interface InvitationResponse {
  invitation: Invitation | undefined;
  message: string;
}

export interface CreateProfileFromTemplate_req {
  standard: string;
}

export interface ListTrustedContactsResponse {
  contacts: TrustedContactInfo[];
  count: number;
}

export interface RevokeSessionResult {
  message: string;
  success: boolean;
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

export interface UpdateOrganizationInput {
  appId: string;
  logo: string;
  metadata: any;
  name: string;
  orgId: string;
}

export interface CreateProfileRequest {
  encryptionAtRest: boolean;
  encryptionInTransit: boolean;
  leastPrivilege: boolean;
  mfaRequired: boolean;
  passwordRequireLower: boolean;
  passwordRequireNumber: boolean;
  rbacRequired: boolean;
  appId: string;
  auditLogExport: boolean;
  complianceContact: string;
  dataResidency: string;
  name: string;
  passwordRequireSymbol: boolean;
  regularAccessReview: boolean;
  retentionDays: number;
  dpoContact: string;
  metadata: any;
  sessionIpBinding: boolean;
  sessionMaxAge: number;
  standards: string[];
  detailedAuditTrail: boolean;
  passwordExpiryDays: number;
  passwordMinLength: number;
  passwordRequireUpper: boolean;
  sessionIdleTimeout: number;
}

export interface SAMLLoginRequest {
  relayState: string;
}

export interface RestoreRevisionRequest {
}

export interface RecoveryCodesConfig {
  enabled: boolean;
  format: string;
  regenerateCount: number;
  allowDownload: boolean;
  allowPrint: boolean;
  autoRegenerate: boolean;
  codeCount: number;
  codeLength: number;
}

export interface StepUpRememberedDevice {
  remembered_at: string;
  user_id: string;
  created_at: string;
  id: string;
  security_level: string;
  user_agent: string;
  device_id: string;
  device_name: string;
  expires_at: string;
  ip: string;
  last_used_at: string;
  org_id: string;
}

export interface GetRoleTemplatesInput {
  appId: string;
}

export interface GetSessionStatsResult {
  stats: SessionStatsDTO;
}

export interface BatchEvaluateResponse {
  successCount: number;
  totalEvaluations: number;
  totalTimeMs: number;
  failureCount: number;
  results: BatchEvaluationResult | undefined[];
}

export interface ProviderSessionRequest {
}

export interface WebhookPayload {
}

export interface ActionResponse {
  id: string;
  name: string;
  namespaceId: string;
  createdAt: string;
  description: string;
}

export interface LinkRequest {
  email: string;
  name: string;
  password: string;
}

export interface Webhook {
  createdAt: string;
  id: string;
  organizationId: string;
  url: string;
  events: string[];
  secret: string;
  enabled: boolean;
}

export interface ListImpersonationsRequest {
}

export interface RequestTrustedContactVerificationRequest {
  contactId: string;
  sessionId: string;
}

export interface RevokeTrustedDeviceRequest {
}

export interface CreateVerificationSession_req {
  cancelUrl: string;
  config: any;
  metadata: any;
  provider: string;
  requiredChecks: string[];
  successUrl: string;
}

export interface ListEntriesRequest {
}

export interface MockEmailService {
  [key: string]: any;
}

export interface StartVideoSessionResponse {
  expiresAt: string;
  message: string;
  sessionUrl: string;
  startedAt: string;
  videoSessionId: string;
}

export interface GenerateTokenRequest {
  permissions: string[];
  scopes: string[];
  sessionId: string;
  tokenType: string;
  userId: string;
  audience: string[];
  expiresIn: Duration;
  metadata: any;
}

export interface CreateSessionHTTPRequest {
  config: any;
  metadata: any;
  provider: string;
  requiredChecks: string[];
  successUrl: string;
  cancelUrl: string;
}

export interface ResourceAttribute {
  [key: string]: any;
}

export interface GetInvitationRequest {
}

export interface GetImpersonationRequest {
}

export interface ContinueRecoveryRequest {
  method: string;
  sessionId: string;
}

export interface DeleteSecretInput {
  appId: string;
  secretId: string;
}

export interface DefaultProviderRegistry {
}

export interface DevicesResponse {
  count: number;
  devices: any;
}

export interface ResetUserMFARequest {
  reason: string;
}

export interface ConsentsResponse {
  consents: any;
  count: number;
}

export interface StepUpPolicy {
  org_id: string;
  user_id: string;
  description: string;
  priority: number;
  rules: any;
  updated_at: string;
  created_at: string;
  enabled: boolean;
  id: string;
  metadata: any;
  name: string;
}

export interface DeleteRoleTemplateResult {
  success: boolean;
}

export interface GetRolesRequest {
}

export interface ResolveViolationRequest {
  notes: string;
  resolution: string;
}

export interface FinishRegisterResponse {
  passkeyId: string;
  status: string;
  createdAt: string;
  credentialId: string;
  name: string;
}

export interface SendCodeRequest {
  phone: string;
}

export interface UpdateContentTypeRequest {
}

export interface VerifyEnrolledFactorRequest {
  code: string;
  data: any;
}

export interface GenerateReport_req {
  reportType: string;
  standard: string;
  format: string;
  period: string;
}

export interface SecretTreeNode {
  [key: string]: any;
}

export interface RecoveryCodeUsage {
}

export interface LimitResult {
}

export interface CreateEvidenceRequest {
  evidenceType: string;
  fileUrl: string;
  standard: string;
  title: string;
  controlId: string;
  description: string;
}

export interface ComplianceStatusResponse {
  status: string;
}

export interface ListAppsRequest {
}

export interface DeleteContentTypeRequest {
}

export interface ListFactorsResponse {
  count: number;
  factors: Factor[];
}

export interface UpdateRoleTemplateResult {
  template: RoleTemplateDTO;
}

export interface TemplateDTO {
  variables: string[];
  body: string;
  isDefault: boolean;
  templateKey: string;
  id: string;
  isModified: boolean;
  name: string;
  active: boolean;
  appId: string;
  createdAt: string;
  metadata: any;
  subject: string;
  updatedAt: string;
  language: string;
  type: string;
}

export interface UpdateProfileRequest {
  mfaRequired: boolean | undefined;
  name: string | undefined;
  retentionDays: number | undefined;
  status: string | undefined;
}

export interface VerifyRecoveryCodeRequest {
  code: string;
  sessionId: string;
}

export interface GetTeamsInput {
  orgId: string;
  page: number;
  search: string;
  appId: string;
  limit: number;
}

export interface SecurityQuestionInfo {
  id: string;
  isCustom: boolean;
  questionId: number;
  questionText: string;
}

export interface InviteMemberResult {
  invitation: InvitationDTO;
}

export interface PreviewTemplate_req {
  variables: any;
}

export interface MigrateAllRequest {
  dryRun: boolean;
  preserveOriginal: boolean;
}

export interface AuditEvent {
}

export interface RateLimitConfig {
  redisAddr: string;
  sendCodePerPhone: RateLimitRule;
  useRedis: boolean;
  verifyPerIp: RateLimitRule;
  verifyPerPhone: RateLimitRule;
  enabled: boolean;
  redisDb: number;
  redisPassword: string;
  sendCodePerIp: RateLimitRule;
}

export interface ApproveRecoveryRequest {
  notes: string;
  sessionId: string;
}

export interface CompleteVideoSessionRequest {
  notes: string;
  verificationResult: string;
  videoSessionId: string;
  livenessPassed: boolean;
  livenessScore: number;
}

export interface SetUserRoleRequest {
  role: string;
  user_id: string;
  user_organization_id: string | undefined;
  app_id: string;
}

export interface CreateFieldRequest {
  [key: string]: any;
}

export interface PreviewTemplateRequest {
}

export interface RolesResponse {
  roles: Role | undefined[];
}

export interface TeamHandler {
}

export interface CallbackResponse {
  session: Session | undefined;
  token: string;
  user: User | undefined;
}

export interface ConsentSettingsResponse {
  settings: any;
}

export interface StepUpVerificationResponse {
  expires_at: string;
  verified: boolean;
}

export interface AccountLockoutError {
}

export interface VerificationSessionResponse {
  session: IdentityVerificationSession | undefined;
}

export interface AsyncConfig {
  persist_failures: boolean;
  queue_size: number;
  retry_backoff: string[];
  retry_enabled: boolean;
  worker_pool_size: number;
  enabled: boolean;
  max_retries: number;
}

export interface Invitation {
  [key: string]: any;
}

export interface ResendRequest {
  email: string;
}

export interface OIDCState {
}

export interface AuditLogEntry {
  actorId: string;
  appId: string;
  id: string;
  resourceType: string;
  timestamp: string;
  userAgent: string;
  userOrganizationId: string | undefined;
  action: string;
  environmentId: string;
  ipAddress: string;
  newValue: any;
  oldValue: any;
  resourceId: string;
}

export interface ListAPIKeysRequest {
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

export interface UpdateRoleTemplateInput {
  appId: string;
  description: string;
  name: string;
  permissions: string[];
  templateId: string;
}

export interface AutoCleanupConfig {
  interval: Duration;
  enabled: boolean;
}

export interface FinishLoginRequest {
  remember: boolean;
  response: any;
}

export interface ContentTypeHandler {
}

export interface ReviewDocumentRequest {
  notes: string;
  rejectionReason: string;
  approved: boolean;
  documentId: string;
}

export interface AuthURLResponse {
  url: string;
}

export interface PrivacySettings {
  id: string;
  metadata: Record<string, any>;
  requireAdminApprovalForDeletion: boolean;
  updatedAt: string;
  allowDataPortability: boolean;
  autoDeleteAfterDays: number;
  contactPhone: string;
  dpoEmail: string;
  gdprMode: boolean;
  organizationId: string;
  ccpaMode: boolean;
  consentRequired: boolean;
  contactEmail: string;
  cookieConsentStyle: string;
  dataExportExpiryHours: number;
  deletionGracePeriodDays: number;
  exportFormat: string[];
  requireExplicitConsent: boolean;
  anonymousConsentEnabled: boolean;
  cookieConsentEnabled: boolean;
  createdAt: string;
  dataRetentionDays: number;
}

export interface IDVerificationListResponse {
  verifications: any[];
}

export interface CreateResourceRequest {
  attributes: ResourceAttributeRequest[];
  description: string;
  namespaceId: string;
  type: string;
}

export interface VersioningConfig {
  retentionDays: number;
  autoCleanup: boolean;
  cleanupInterval: Duration;
  maxVersions: number;
}

export interface GetEntryRequest {
}

export interface ListSessionsResponse {
  limit: number;
  page: number;
  sessions: Session | undefined[];
  total: number;
  total_pages: number;
}

export interface Authsome {
  [key: string]: any;
}

export interface UploadDocumentRequest {
  backImage: string;
  documentType: string;
  frontImage: string;
  selfie: string;
  sessionId: string;
}

export interface TOTPSecret {
}

export interface GetFactorRequest {
}

export interface ListTrainingFilter {
  appId: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
  trainingType: string | undefined;
  userId: string | undefined;
}

export interface CreateABTestVariant_req {
  body: string;
  name: string;
  subject: string;
  weight: number;
}

export interface RevokeAllRequest {
  includeCurrentSession: boolean;
}

export interface CompleteRecoveryResponse {
  completedAt: string;
  message: string;
  sessionId: string;
  status: string;
  token: string;
}

export interface GetSecurityQuestionsResponse {
  questions: SecurityQuestionInfo[];
}

export interface CookieConsent {
  marketing: boolean;
  updatedAt: string;
  consentBannerVersion: string;
  createdAt: string;
  functional: boolean;
  analytics: boolean;
  expiresAt: string;
  ipAddress: string;
  organizationId: string;
  personalization: boolean;
  sessionId: string;
  thirdParty: boolean;
  userId: string;
  essential: boolean;
  id: string;
  userAgent: string;
}

export interface ErrorResponse {
  code: string;
  details: any;
  error: string;
}

export interface ResendResponse {
  status: string;
}

export interface RevealValueResponse {
  [key: string]: any;
}

export interface BackupAuthContactResponse {
  id: string;
}

export interface CreateOrganizationResult {
  organization: OrganizationDetailDTO;
}

export interface GetPasskeyRequest {
}

export interface QueryEntriesRequest {
}

export interface DeviceVerifyResponse {
  clientId: string;
  clientName: string;
  logoUri: string;
  scopes: ScopeInfo[];
  userCode: string;
  userCodeFormatted: string;
  authorizeUrl: string;
}

export interface ListTrustedDevicesResponse {
  count: number;
  devices: TrustedDevice[];
}

export interface CreateAPIKeyRequest {
  metadata: any;
  name: string;
  permissions: any;
  rate_limit: number;
  scopes: string[];
  allowed_ips: string[];
  description: string;
}

export interface BulkRequest {
  ids: string[];
}

export interface UpdateFieldRequest {
}

export interface UpdateMemberHandlerRequest {
}

export interface CompleteVideoSessionResponse {
  completedAt: string;
  message: string;
  result: string;
  videoSessionId: string;
}

export interface TwoFASendOTPResponse {
  code: string;
  status: string;
}

export interface TestSendTemplateResult {
  message: string;
  success: boolean;
}

export interface TrustedContactsConfig {
  requireVerification: boolean;
  requiredToRecover: number;
  allowEmailContacts: boolean;
  cooldownPeriod: Duration;
  enabled: boolean;
  maxNotificationsPerDay: number;
  verificationExpiry: Duration;
  allowPhoneContacts: boolean;
  maximumContacts: number;
  minimumContacts: number;
}

export interface InstantiateTemplateRequest {
  description: string;
  enabled: boolean;
  name: string;
  namespaceId: string;
  parameters: any;
  priority: number;
  resourceType: string;
  actions: string[];
}

export interface StatsResponse {
  active_sessions: number;
  active_users: number;
  banned_users: number;
  timestamp: string;
  total_sessions: number;
  total_users: number;
}

export interface EvaluationContext {
}

export interface MFAPolicyResponse {
  allowedFactorTypes: string[];
  appId: string;
  enabled: boolean;
  gracePeriodDays: number;
  id: string;
  organizationId: string | undefined;
  requiredFactorCount: number;
}

export interface ComplianceStatus {
  checksFailed: number;
  nextAudit: string;
  overallStatus: string;
  profileId: string;
  score: number;
  standard: string;
  appId: string;
  checksPassed: number;
  checksWarning: number;
  lastChecked: string;
  violations: number;
}

export interface Logger {
  [key: string]: any;
}

export interface CreateAppRequest {
  [key: string]: any;
}

export interface PublishEntryRequest {
}

export interface MultiSessionErrorResponse {
  error: string;
}

export interface PreviewConversionResponse {
  celExpression: string;
  error: string;
  policyName: string;
  resourceId: string;
  resourceType: string;
  success: boolean;
}

export interface RemoveTeamMemberRequest {
}

export interface CreateEntryRequest {
}

export interface OAuthState {
  provider: string;
  redirect_url: string;
  user_organization_id: string | undefined;
  app_id: string;
  created_at: string;
  extra_scopes: string[];
  link_user_id: string | undefined;
}

export interface MigrateRBACRequest {
  keepRbacPolicies: boolean;
  namespaceId: string;
  validateEquivalence: boolean;
  dryRun: boolean;
}

export interface Role {
  [key: string]: any;
}

export interface ForgetDeviceResponse {
  message: string;
  success: boolean;
}

export interface RemoveMemberInput {
  appId: string;
  memberId: string;
  orgId: string;
}

export interface DeleteOrganizationRequest {
}

export interface OrgDetailStatsDTO {
  invitationCount: number;
  memberCount: number;
  teamCount: number;
}

export interface TwoFAStatusDetailResponse {
  enabled: boolean;
  method: string;
  trusted: boolean;
}

export interface LinkAccountRequest {
  provider: string;
  scopes: string[];
}

export interface MemoryStateStore {
  [key: string]: any;
}

export interface ConsentAuditConfig {
  signLogs: boolean;
  logAllChanges: boolean;
  retentionDays: number;
  archiveInterval: Duration;
  archiveOldLogs: boolean;
  enabled: boolean;
  exportFormat: string;
  immutable: boolean;
  logIpAddress: boolean;
  logUserAgent: boolean;
}

export interface CreatePolicyRequest {
  content: string;
  name: string;
  renewable: boolean;
  validityPeriod: number | undefined;
  consentType: string;
  description: string;
  metadata: any;
  required: boolean;
  version: string;
}

export interface CreateTeamHandlerRequest {
}

export interface JumioProvider {
}

export interface RemoveMemberRequest {
}

export interface UsernameRepository {
  [key: string]: any;
}

export interface ListAuditEventsRequest {
}

export interface GetTemplateRequest {
}

export interface AdminHandler {
}

export interface VerifyTrustedContactResponse {
  contactId: string;
  message: string;
  verified: boolean;
  verifiedAt: string;
}

export interface ConsentNotificationsConfig {
  channels: string[];
  notifyDeletionComplete: boolean;
  notifyDpoEmail: string;
  notifyOnExpiry: boolean;
  notifyOnRevoke: boolean;
  enabled: boolean;
  notifyDeletionApproved: boolean;
  notifyExportReady: boolean;
  notifyOnGrant: boolean;
}

export interface UpdateProvider_req {
  config: any;
  isActive: boolean;
  isDefault: boolean;
}

export interface CheckResult {
  status: string;
  checkType: string;
  error: Error;
  evidence: string[];
  result: any;
  score: number;
}

export interface TwoFAEnableResponse {
  totp_uri: string;
  status: string;
}

export interface AdminAddProviderRequest {
  appId: string;
  clientId: string;
  clientSecret: string;
  enabled: boolean;
  provider: string;
  scopes: string[];
}

export interface DataExportConfig {
  expiryHours: number;
  includeSections: string[];
  maxExportSize: number;
  maxRequests: number;
  requestPeriod: Duration;
  autoCleanup: boolean;
  enabled: boolean;
  storagePath: string;
  allowedFormats: string[];
  cleanupInterval: Duration;
  defaultFormat: string;
}

export interface StepUpAttempt {
  created_at: string;
  id: string;
  ip: string;
  method: string;
  org_id: string;
  requirement_id: string;
  user_agent: string;
  user_id: string;
  failure_reason: string;
  success: boolean;
}

export interface GetExtensionDataResult {
  quickLinks: QuickLinkDataDTO[];
  tabs: TabDataDTO[];
  widgets: WidgetDataDTO[];
  actions: ActionDataDTO[];
}

export interface RunCheckRequest {
  checkType: string;
}

export interface TwoFABackupCodesResponse {
  codes: string[];
}

export interface GetOrganizationsInput {
  search: string;
  appId: string;
  limit: number;
  page: number;
}

export interface UpdateProvidersInput {
  emailProvider: EmailProviderDTO | undefined;
  smsProvider: SMSProviderDTO | undefined;
}

export interface CheckDependencies {
}

export interface ListUsersRequest {
  limit: number;
  page: number;
  role: string;
  search: string;
  status: string;
  user_organization_id: string | undefined;
  app_id: string;
}

export interface ConsentSummary {
  consentsByType: any;
  expiredConsents: number;
  hasPendingDeletion: boolean;
  pendingRenewals: number;
  grantedConsents: number;
  hasPendingExport: boolean;
  lastConsentUpdate: string | undefined;
  organizationId: string;
  revokedConsents: number;
  totalConsents: number;
  userId: string;
}

export interface GetProvidersInput {
  [key: string]: any;
}

export interface SetActiveRequest {
  id: string;
}

export interface GetSessionInput {
  appId: string;
  sessionId: string;
}

export interface JWKSService {
}

export interface MFABypassResponse {
  expiresAt: string;
  id: string;
  reason: string;
  userId: string;
}

export interface UpdateProvidersResult {
  message: string;
  providers: ProvidersConfigDTO;
  success: boolean;
}

export interface CreateTemplateInput {
  metadata: any;
  name: string;
  subject: string;
  templateKey: string;
  type: string;
  variables: string[];
  body: string;
  language: string;
}

export interface AppServiceAdapter {
}

export interface SetUserRoleRequestDTO {
  role: string;
}

export interface NotificationChannels {
  email: boolean;
  slack: boolean;
  webhook: boolean;
}

export interface CompliancePolicy {
  content: string;
  metadata: any;
  reviewDate: string;
  updatedAt: string;
  effectiveDate: string;
  standard: string;
  status: string;
  appId: string;
  approvedAt: string | undefined;
  createdAt: string;
  id: string;
  profileId: string;
  policyType: string;
  title: string;
  version: string;
  approvedBy: string;
}

export interface AddTeamMemberRequest {
  member_id: string;
}

export interface JWKSResponse {
  keys: JWK[];
}

export interface StepUpRequirementResponse {
  id: string;
}

export interface MigrateAllResponse {
  completedAt: string;
  convertedPolicies: PolicyPreviewResponse[];
  errors: MigrationErrorResponse[];
  skippedPolicies: number;
  totalPolicies: number;
  dryRun: boolean;
  failedPolicies: number;
  migratedPolicies: number;
  startedAt: string;
}

export interface ComplianceCheck {
  checkType: string;
  createdAt: string;
  evidence: string[];
  lastCheckedAt: string;
  profileId: string;
  appId: string;
  id: string;
  nextCheckAt: string;
  result: any;
  status: string;
}

export interface DeviceVerificationRequest {
  user_code: string;
}

export interface DataProcessingAgreement {
  agreementType: string;
  effectiveDate: string;
  expiryDate: string | undefined;
  signedBy: string;
  signedByEmail: string;
  updatedAt: string;
  version: string;
  digitalSignature: string;
  id: string;
  organizationId: string;
  status: string;
  content: string;
  signedByTitle: string;
  createdAt: string;
  ipAddress: string;
  metadata: Record<string, any>;
  signedByName: string;
}

export interface ResetTemplateRequest {
}

export interface GetNotificationRequest {
}

export interface AddFieldRequest {
}

export interface BackupAuthDocumentResponse {
  id: string;
}

export interface VerificationsResponse {
  count: number;
  verifications: any;
}

export interface ListFactorsRequest {
}

export interface NotificationResponse {
  notification: any;
}

export interface GetRevisionRequest {
}

export interface DeviceAuthorizationRequest {
  client_id: string;
  scope: string;
}

export interface RenderTemplate_req {
  template: string;
  variables: any;
}

export interface DeletePasskeyRequest {
}

export interface UpdateProviderRequest {
}

export interface NamespaceResponse {
  userOrganizationId: string | undefined;
  actionCount: number;
  createdAt: string;
  environmentId: string;
  inheritPlatform: boolean;
  policyCount: number;
  resourceCount: number;
  updatedAt: string;
  appId: string;
  description: string;
  id: string;
  name: string;
  templateId: string | undefined;
}

export interface ListReportsFilter {
  status: string | undefined;
  appId: string | undefined;
  format: string | undefined;
  profileId: string | undefined;
  reportType: string | undefined;
  standard: string | undefined;
}

export interface PhoneVerifyResponse {
  token: string;
  user: User | undefined;
  session: Session | undefined;
}

export interface DeviceVerificationInfo {
}

export interface ChallengeResponse {
  sessionId: string;
  availableFactors: FactorInfo[];
  challengeId: string;
  expiresAt: string;
  factorsRequired: number;
}

export interface SecretItem {
  version: number;
  createdAt: string;
  description: string;
  id: string;
  path: string;
  tags: string[];
  updatedAt: string;
  valueType: string;
  key: string;
}

export interface CreateVerificationRequest {
}

export interface NotificationErrorResponse {
  error: string;
}

export interface TemplatePerformanceDTO {
  clickRate: number;
  openRate: number;
  templateId: string;
  templateName: string;
  totalSent: number;
}

export interface AuditLogResponse {
  entries: AuditLogEntry | undefined[];
  page: number;
  pageSize: number;
  totalCount: number;
}

export interface Handler {
}

export interface GetExtensionDataInput {
  appId: string;
  orgId: string;
}

export interface TestCaseResult {
  actual: boolean;
  error: string;
  evaluationTimeMs: number;
  expected: boolean;
  name: string;
  passed: boolean;
}

export interface EndImpersonationRequest {
  impersonation_id: string;
  reason: string;
}

export interface AdminBypassRequest {
  duration: number;
  reason: string;
  userId: string;
}

export interface DeleteSecretOutput {
  message: string;
  success: boolean;
}

export interface ComplianceTrainingsResponse {
  training: any[];
}

export interface CreateTrainingRequest {
  standard: string;
  trainingType: string;
  userId: string;
}

export interface BulkPublishRequest {
  ids: string[];
}

export interface ListSessionsRequest {
  user_organization_id: string | undefined;
  app_id: string;
  limit: number;
  page: number;
  user_id: string | undefined;
}

export interface PaginationDTO {
  currentPage: number;
  hasNext: boolean;
  hasPrev: boolean;
  pageSize: number;
  totalCount: number;
  totalPages: number;
}

export interface NotificationHistoryDTO {
  createdAt: string;
  id: string;
  sentAt: string | undefined;
  templateId: string | undefined;
  deliveredAt: string | undefined;
  updatedAt: string;
  appId: string;
  body: string;
  providerId: string;
  recipient: string;
  status: string;
  type: string;
  error: string;
  metadata: any;
  subject: string;
}

export interface GetTemplateVersionRequest {
}

export interface MigrationErrorResponse {
  resource: string;
  subject: string;
  error: string;
  policyIndex: number;
}

export interface CompliancePoliciesResponse {
  policies: any[];
}

export interface MultitenancyStatusResponse {
  [key: string]: any;
}

export interface ComplianceProfile {
  standards: string[];
  mfaRequired: boolean;
  createdAt: string;
  encryptionInTransit: boolean;
  metadata: any;
  appId: string;
  complianceContact: string;
  passwordMinLength: number;
  passwordRequireLower: boolean;
  rbacRequired: boolean;
  regularAccessReview: boolean;
  name: string;
  auditLogExport: boolean;
  passwordRequireUpper: boolean;
  passwordRequireSymbol: boolean;
  retentionDays: number;
  sessionIpBinding: boolean;
  status: string;
  dataResidency: string;
  encryptionAtRest: boolean;
  id: string;
  passwordExpiryDays: number;
  detailedAuditTrail: boolean;
  dpoContact: string;
  sessionMaxAge: number;
  updatedAt: string;
  leastPrivilege: boolean;
  sessionIdleTimeout: number;
  passwordRequireNumber: boolean;
}

export interface CreateProvider_req {
  providerType: string;
  config: any;
  isDefault: boolean;
  organizationId?: string | undefined;
  providerName: string;
}

export interface ApproveRecoveryResponse {
  approved: boolean;
  approvedAt: string;
  message: string;
  sessionId: string;
}

export interface CreateAPIKeyResponse {
  api_key: APIKey | undefined;
  message: string;
}

export interface ListPasskeysRequest {
}

export interface GetRecoveryConfigResponse {
  riskScoreThreshold: number;
  enabledMethods: string[];
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
}

export interface VideoSessionResult {
}

export interface ListTemplatesResult {
  pagination: PaginationDTO;
  templates: TemplateDTO[];
}

export interface APIKey {
  [key: string]: any;
}

export interface UpdateTeamRequest {
  description: string;
  name: string;
}

export interface BulkUnpublishRequest {
  ids: string[];
}

export interface StartVideoSessionRequest {
  videoSessionId: string;
}

export interface UpdateAppRequest {
}

export interface ProviderDiscoveredResponse {
  found: boolean;
  providerId: string;
  type: string;
}

export interface SocialAccount {
  [key: string]: any;
}

export interface AddMemberRequest {
  role: string;
  user_id: string;
}

export interface DeviceDecisionResponse {
  approved: boolean;
  message: string;
  success: boolean;
}

export interface SignInResponse {
  session: Session | undefined;
  token: string;
  user: User | undefined;
}

export interface UpdateTemplateInput {
  name: string | undefined;
  subject: string | undefined;
  templateId: string;
  variables: string[];
  active: boolean | undefined;
  body: string | undefined;
  metadata: any;
}

export interface MigrationResponse {
  message: string;
  migrationId: string;
  startedAt: string;
  status: string;
}

export interface AnalyticsResponse {
  generatedAt: string;
  summary: AnalyticsSummary;
  timeRange: any;
}

export interface UpdatePasskeyResponse {
  name: string;
  passkeyId: string;
  updatedAt: string;
}

export interface EnableResponse {
  status: string;
  totp_uri: string;
}

export interface SMSFactorAdapter {
}

export interface QuickLinkDataDTO {
  description: string;
  icon: string;
  id: string;
  order: number;
  requireAdmin: boolean;
  title: string;
  url: string;
}

export interface ImpersonateUserRequest {
  app_id: string;
  duration: Duration;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface GetSecretsOutput {
  page: number;
  pageSize: number;
  secrets: SecretItem[];
  total: number;
  totalPages: number;
}

export interface GenerateRecoveryCodesRequest {
  count: number;
  format: string;
}

export interface StepUpVerificationsResponse {
  verifications: any[];
}

export interface EvaluationResult {
  metadata: any;
  reason: string;
  required: boolean;
  requirement_id: string;
  security_level: string;
  can_remember: boolean;
  challenge_token: string;
  expires_at: string;
  grace_period_ends_at: string;
  allowed_methods: string[];
  current_level: string;
  matched_rules: string[];
}

export interface StepUpDevicesResponse {
  count: number;
  devices: any;
}

export interface SettingsDTO {
  allowCrossPlatform: boolean;
  enableDeviceTracking: boolean;
  maxSessionsPerUser: number;
  sessionExpiryHours: number;
}

export interface User {
  id: string;
  email: string;
  name?: string;
  emailVerified: boolean;
  createdAt: string;
  updatedAt: string;
  organizationId?: string;
}

export interface MFASession {
  id: string;
  riskLevel: string;
  sessionToken: string;
  userAgent: string;
  createdAt: string;
  expiresAt: string;
  factorsRequired: number;
  ipAddress: string;
  metadata: any;
  userId: string;
  verifiedFactors: string[];
  completedAt: string | undefined;
  factorsVerified: number;
}

export interface ListSecretsResponse {
  [key: string]: any;
}

export interface GetSessionResult {
  session: SessionDetailDTO;
}

export interface ComplianceEvidencesResponse {
  evidence: any[];
}

export interface Member {
  [key: string]: any;
}

export interface ImpersonationEndResponse {
  ended_at: string;
  status: string;
}

export interface JWTService {
}

export interface StepUpStatusResponse {
  status: string;
}

export interface TeamDTO {
  createdAt: string;
  description: string;
  id: string;
  memberCount: number;
  metadata: any;
  name: string;
}

export interface SecretDTO {
  [key: string]: any;
}

export interface MockSessionService {
  [key: string]: any;
}

export interface SessionStatsResponse {
  oldestSession: string | undefined;
  totalSessions: number;
  activeSessions: number;
  deviceCount: number;
  locationCount: number;
  newestSession: string | undefined;
}

export interface ComplianceCheckResponse {
  id: string;
}

export interface BackupAuthStatusResponse {
  status: string;
}

export interface ListSecretsRequest {
}

export interface GetRoleTemplateResult {
  template: RoleTemplateDTO;
}

export interface ListEvidenceFilter {
  evidenceType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  appId: string | undefined;
  controlId: string | undefined;
}

export interface JSONBMap {
  [key: string]: any;
}

export interface RestoreEntryRequest {
}

export interface DeleteEntryRequest {
}

export interface RiskAssessmentConfig {
  requireReviewAbove: number;
  velocityWeight: number;
  blockHighRisk: boolean;
  enabled: boolean;
  highRiskThreshold: number;
  historyWeight: number;
  lowRiskThreshold: number;
  newDeviceWeight: number;
  newLocationWeight: number;
  mediumRiskThreshold: number;
  newIpWeight: number;
}

export interface AmountRule {
  max_amount: number;
  min_amount: number;
  org_id: string;
  security_level: string;
  currency: string;
  description: string;
}

export interface DeleteRoleTemplateInput {
  appId: string;
  templateId: string;
}

export interface StripeIdentityProvider {
}

export interface CancelRecoveryRequest {
  reason: string;
  sessionId: string;
}

export interface RequestTrustedContactVerificationResponse {
  notifiedAt: string;
  contactId: string;
  contactName: string;
  expiresAt: string;
  message: string;
}

export interface CompleteRecoveryRequest {
  sessionId: string;
}

export interface ProviderConfigResponse {
  appId: string;
  message: string;
  provider: string;
}

export interface CreateDPARequest {
  content: string;
  metadata: any;
  signedByEmail: string;
  signedByTitle: string;
  agreementType: string;
  effectiveDate: string;
  expiryDate: string | undefined;
  signedByName: string;
  version: string;
}

export interface GetUserSessionsResult {
  activeCount: number;
  pagination: PaginationInfoDTO;
  sessions: SessionDTO[];
  totalCount: number;
  userId: string;
}

export interface GenerateTokenResponse {
  [key: string]: any;
}

export interface BackupAuthSessionsResponse {
  sessions: any[];
}

export interface DataDeletionRequestInput {
  deleteSections: string[];
  reason: string;
}

export interface SessionAutoSendConfig {
  device_removed: boolean;
  new_device: boolean;
  new_location: boolean;
  suspicious_login: boolean;
  all_revoked: boolean;
}

export interface ResetUserMFAResponse {
  message: string;
  success: boolean;
  devicesRevoked: number;
  factorsReset: number;
}

export interface BeginLoginResponse {
  timeout: Duration;
  challenge: string;
  options: any;
}

export interface UnpublishContentTypeRequest {
}

export interface TwoFAErrorResponse {
  error: string;
}

export interface UpdateConsentRequest {
  granted: boolean | undefined;
  metadata: any;
  reason: string;
}

export interface TrustDeviceRequest {
  deviceId: string;
  metadata: any;
  name: string;
}

export interface AddTeamMember_req {
  member_id: string;
  role: string;
}

export interface AuthorizeRequest {
  state: string;
  ui_locales: string;
  acr_values: string;
  client_id: string;
  code_challenge_method: string;
  id_token_hint: string;
  login_hint: string;
  nonce: string;
  prompt: string;
  redirect_uri: string;
  code_challenge: string;
  max_age: number | undefined;
  response_type: string;
  scope: string;
}

export interface ConsentPolicy {
  createdAt: string;
  createdBy: string;
  description: string;
  renewable: boolean;
  name: string;
  publishedAt: string | undefined;
  validityPeriod: number | undefined;
  active: boolean;
  consentType: string;
  content: string;
  id: string;
  metadata: Record<string, any>;
  organizationId: string;
  updatedAt: string;
  required: boolean;
  version: string;
}

export interface UpdateFactorRequest {
  metadata: any;
  name: string | undefined;
  priority: string | undefined;
  status: string | undefined;
}

export interface ListJWTKeysResponse {
  [key: string]: any;
}

export interface SendCodeResponse {
  dev_code: string;
  status: string;
}

export interface MockSocialAccountRepository {
  [key: string]: any;
}

export interface ConsentReportResponse {
  id: string;
}

export interface SessionDTO {
  createdAt: string;
  deviceType: string;
  expiresAt: string;
  userEmail: string;
  userId: string;
  browser: string;
  id: string;
  lastUsed: string;
  browserVersion: string;
  deviceInfo: string;
  ipAddress: string;
  os: string;
  osVersion: string;
  expiresIn: string;
  isActive: boolean;
  isExpiring: boolean;
  status: string;
  userAgent: string;
}

export interface VerifySecurityAnswersRequest {
  answers: any;
  sessionId: string;
}

export interface PaginationInfo {
  currentPage: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface IDVerificationSessionResponse {
  session: any;
}

export interface OrganizationAutoSendDTO {
  roleChanged: boolean;
  transfer: boolean;
  deleted: boolean;
  invite: boolean;
  memberAdded: boolean;
  memberLeft: boolean;
  memberRemoved: boolean;
}

export interface Client {
  [key: string]: any;
}

export interface JWTKey {
  [key: string]: any;
}

export interface UpdateSecretInput {
  appId: string;
  changeReason: string;
  description: string;
  secretId: string;
  tags: string[];
  value: any;
}

export interface RenderTemplateRequest {
}

export interface ComplianceReportResponse {
  id: string;
}

export interface TokenIntrospectionResponse {
  jti: string;
  scope: string;
  token_type: string;
  username: string;
  active: boolean;
  aud: string[];
  client_id: string;
  exp: number;
  iat: number;
  iss: string;
  nbf: number;
  sub: string;
}

export interface WebhookResponse {
  received: boolean;
  status: string;
}

export interface VideoSessionInfo {
}

export interface CreateUserRequest {
  app_id: string;
  name: string;
  email: string;
  email_verified: boolean;
  metadata: any;
  password: string;
  role: string;
  user_organization_id: string | undefined;
  username: string;
}

export interface AutomatedChecksConfig {
  accessReview: boolean;
  checkInterval: Duration;
  inactiveUsers: boolean;
  passwordPolicy: boolean;
  sessionPolicy: boolean;
  suspiciousActivity: boolean;
  dataRetention: boolean;
  enabled: boolean;
  mfaCoverage: boolean;
}

export interface ChallengeSession {
}

export interface UnpublishEntryRequest {
}

export interface AdminBlockUser_req {
  reason: string;
}

export interface DocumentVerificationConfig {
  requireBothSides: boolean;
  requireSelfie: boolean;
  storagePath: string;
  acceptedDocuments: string[];
  provider: string;
  requireManualReview: boolean;
  retentionPeriod: Duration;
  storageProvider: string;
  enabled: boolean;
  encryptAtRest: boolean;
  encryptionKey: string;
  minConfidenceScore: number;
}

export interface JumioConfig {
  dataCenter: string;
  enableAMLScreening: boolean;
  enableExtraction: boolean;
  enabledDocumentTypes: string[];
  presetId: string;
  apiSecret: string;
  enableLiveness: boolean;
  enabled: boolean;
  enabledCountries: string[];
  verificationType: string;
  apiToken: string;
  callbackUrl: string;
}

export interface ResourceResponse {
  description: string;
  id: string;
  namespaceId: string;
  type: string;
  attributes: ResourceAttribute[];
  createdAt: string;
}

export interface VerifyAPIKeyRequest {
  key: string;
}

export interface CompareRevisionsRequest {
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

export interface FactorAdapterRegistry {
}

export interface TabDataDTO {
  order: number;
  path: string;
  requireAdmin: boolean;
  icon: string;
  id: string;
  label: string;
}

export interface TemplatesListResponse {
  categories: string[];
  templates: TemplateResponse | undefined[];
  totalCount: number;
}

export interface GetMigrationStatusRequest {
  [key: string]: any;
}

export interface RateLimitingConfig {
  maxAttemptsPerIp: number;
  enabled: boolean;
  exponentialBackoff: boolean;
  ipCooldownPeriod: Duration;
  lockoutAfterAttempts: number;
  lockoutDuration: Duration;
  maxAttemptsPerDay: number;
  maxAttemptsPerHour: number;
}

export interface UnbanUserRequest {
  app_id: string;
  reason: string;
  user_id: string;
  user_organization_id: string | undefined;
}

export interface ListSessionsRequestDTO {
}

export interface StepUpRequirementsResponse {
  requirements: any[];
}

export interface InviteMemberHandlerRequest {
}

export interface VerificationListResponse {
  limit: number;
  offset: number;
  total: number;
  verifications: IdentityVerification | undefined[];
}

export interface ComplianceTraining {
  score: number;
  status: string;
  trainingType: string;
  appId: string;
  createdAt: string;
  id: string;
  metadata: any;
  standard: string;
  userId: string;
  completedAt: string | undefined;
  expiresAt: string | undefined;
  profileId: string;
}

export interface InviteMemberRequest {
  email: string;
  role: string;
}

export interface ClientSummary {
  name: string;
  applicationType: string;
  clientID: string;
  createdAt: string;
  isOrgLevel: boolean;
}

export interface StateStore {
}

export interface GetNotificationDetailResult {
  notification: NotificationHistoryDTO;
}

export interface NamespacesListResponse {
  namespaces: NamespaceResponse | undefined[];
  totalCount: number;
}

export interface MockAuditService {
  [key: string]: any;
}

export interface CheckRegistry {
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

export interface MFAConfigResponse {
  allowed_factor_types: string[];
  enabled: boolean;
  required_factor_count: number;
}

export interface GetSettingsInput {
  appId: string;
}

export interface UploadDocumentResponse {
  documentId: string;
  message: string;
  processingTime: string;
  status: string;
  uploadedAt: string;
}

export interface CodesResponse {
  codes: string[];
}

export interface CreateTemplateRequest {
  [key: string]: any;
}

export interface PublishContentTypeRequest {
}

export interface GetTeamsResult {
  canManage: boolean;
  data: TeamDTO[];
  pagination: PaginationInfo;
}

export interface UpdateOrganizationResult {
  organization: OrganizationDetailDTO;
}

export interface SessionDetailDTO {
  lastRefreshedAt: string | undefined;
  osVersion: string;
  status: string;
  appId: string;
  browserVersion: string;
  deviceInfo: string;
  expiresAtFormatted: string;
  id: string;
  lastRefreshedFormatted: string;
  organizationId: string;
  os: string;
  createdAtFormatted: string;
  deviceType: string;
  isActive: boolean;
  userAgent: string;
  userEmail: string;
  browser: string;
  createdAt: string;
  updatedAt: string;
  updatedAtFormatted: string;
  userId: string;
  environmentId: string;
  expiresAt: string;
  ipAddress: string;
  isExpiring: boolean;
}

export interface TestPolicyRequest {
  actions: string[];
  expression: string;
  resourceType: string;
  testCases: TestCase[];
}

export interface UpdatePolicy_req {
  content: string | undefined;
  status: string | undefined;
  title: string | undefined;
  version: string | undefined;
}

export interface PreviewTemplateInput {
  variables: any;
  templateId: string;
}

export interface SessionStatsDTO {
  uniqueUsers: number;
  activeCount: number;
  desktopCount: number;
  expiredCount: number;
  expiringCount: number;
  mobileCount: number;
  tabletCount: number;
  totalSessions: number;
}

export interface RemoveMemberResult {
  success: boolean;
}

export interface GetProviderRequest {
}

export interface ValidatePolicyRequest {
  expression: string;
  resourceType: string;
}

export interface StartRecoveryRequest {
  userId: string;
  deviceId: string;
  email: string;
  preferredMethod: string;
}

export interface AnalyticsSummary {
  cacheHitRate: number;
  deniedCount: number;
  topPolicies: PolicyStats[];
  topResourceTypes: ResourceTypeStats[];
  totalEvaluations: number;
  avgLatencyMs: number;
  totalPolicies: number;
  activePolicies: number;
  allowedCount: number;
}

export interface UnassignRoleRequest {
}

export interface UserServiceAdapter {
}

export interface ProviderConfig {
  [key: string]: any;
}

export interface BackupAuthQuestionsResponse {
  questions: string[];
}

export interface BunRepository {
}

export interface OAuthClientRepository {
  [key: string]: any;
}

export interface IdentityVerificationSession {
  [key: string]: any;
}

export interface AccountAutoSendDTO {
  usernameChanged: boolean;
  deleted: boolean;
  emailChangeRequest: boolean;
  emailChanged: boolean;
  passwordChanged: boolean;
  reactivated: boolean;
  suspended: boolean;
}

export interface UpdateTemplateRequest {
}

export interface ConnectionResponse {
  connection: SocialAccount | undefined;
}

export interface FactorInfo {
  metadata: any;
  name: string;
  type: string;
  factorId: string;
}

export interface StepUpErrorResponse {
  error: string;
}

export interface StepUpAuditLogsResponse {
  audit_logs: any[];
}

export interface ReverifyRequest {
  reason: string;
}

export interface StripeIdentityConfig {
  apiKey: string;
  enabled: boolean;
  requireLiveCapture: boolean;
  requireMatchingSelfie: boolean;
  returnUrl: string;
  useMock: boolean;
  webhookSecret: string;
  allowedTypes: string[];
}

export interface ScopeResolver {
}

export interface RedisChallengeStore {
  [key: string]: any;
}

export interface OTPSentResponse {
  code: string;
  status: string;
}

export interface RedisStateStore {
  client?: Client | undefined;
}

export interface ChallengeRequest {
  userId: string;
  context: string;
  factorTypes: string[];
  metadata: any;
}

export interface BackupAuthVideoResponse {
  session_id: string;
}

export interface DataDeletionConfig {
  retentionExemptions: string[];
  allowPartialDeletion: boolean;
  archivePath: string;
  autoProcessAfterGrace: boolean;
  enabled: boolean;
  preserveLegalData: boolean;
  requireAdminApproval: boolean;
  archiveBeforeDeletion: boolean;
  gracePeriodDays: number;
  notifyBeforeDeletion: boolean;
}

export interface TwoFARequiredResponse {
  device_id: string;
  require_twofa: boolean;
  user: User | undefined;
}

export interface ListOrganizationsRequest {
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

export interface GetAPIKeyRequest {
}

export interface ListUsersResponse {
  limit: number;
  page: number;
  total: number;
  total_pages: number;
  users: User | undefined[];
}

export interface ResourceTypeStats {
  allowRate: number;
  avgLatencyMs: number;
  evaluationCount: number;
  resourceType: string;
}

export interface DeleteTeamRequest {
}

export interface EncryptionService {
}

export interface AuditServiceAdapter {
}

export interface NoOpDocumentProvider {
  [key: string]: any;
}

export interface GetTreeRequest {
}

export interface OIDCLoginRequest {
  nonce: string;
  redirectUri: string;
  scope: string;
  state: string;
}

export interface VerificationResult {
}

export interface ProvidersConfigDTO {
  emailProvider: EmailProviderDTO;
  smsProvider: SMSProviderDTO;
}

export interface DeviceInfo {
  deviceId: string;
  metadata: any;
  name: string;
}

export interface GetVersionsRequest {
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

export interface GetOrganizationRequest {
}

export interface EvaluateResponse {
  cacheHit: boolean;
  error: string;
  evaluatedPolicies: number;
  evaluationTimeMs: number;
  matchedPolicies: string[];
  reason: string;
  allowed: boolean;
}

export interface Factor {
  createdAt: string;
  expiresAt: string | undefined;
  name: string;
  priority: string;
  updatedAt: string;
  userId: string;
  id: string;
  lastUsedAt: string | undefined;
  metadata: any;
  status: string;
  type: string;
  verifiedAt: string | undefined;
}

export interface RiskEngine {
}

export interface BridgeAppInput {
  appId: string;
}

export interface TrustedDevicesConfig {
  max_devices_per_user: number;
  max_expiry_days: number;
  default_expiry_days: number;
  enabled: boolean;
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

export interface MetadataResponse {
  metadata: string;
}

export interface TemplatesResponse {
  count: number;
  templates: any;
}

export interface Credential {
  [key: string]: any;
}

export interface Service {
}

export interface ConsentRecordResponse {
  id: string;
}

export interface UpdateSettingsResult {
  success: boolean;
}

export interface EmailServiceAdapter {
}

export interface EmailProvider {
  [key: string]: any;
}

export interface UserAdapter {
}

export interface DataDeletionRequest {
  userId: string;
  exemptionReason: string;
  id: string;
  completedAt: string | undefined;
  errorMessage: string;
  organizationId: string;
  rejectedAt: string | undefined;
  retentionExempt: boolean;
  updatedAt: string;
  approvedAt: string | undefined;
  approvedBy: string;
  createdAt: string;
  requestReason: string;
  status: string;
  archivePath: string;
  deleteSections: string[];
  ipAddress: string;
}

export interface NotificationTemplateListResponse {
  total: number;
  templates: any[];
}

export interface OverviewStatsDTO {
  openRate: number;
  totalBounced: number;
  totalClicked: number;
  totalDelivered: number;
  totalFailed: number;
  totalSent: number;
  bounceRate: number;
  clickRate: number;
  deliveryRate: number;
  totalOpened: number;
}

export interface TemplateDefault {
}

export interface NotificationWebhookResponse {
  status: string;
}

export interface DiscoveryResponse {
  code_challenge_methods_supported: string[];
  grant_types_supported: string[];
  introspection_endpoint_auth_methods_supported: string[];
  issuer: string;
  claims_supported: string[];
  require_request_uri_registration: boolean;
  revocation_endpoint: string;
  revocation_endpoint_auth_methods_supported: string[];
  scopes_supported: string[];
  subject_types_supported: string[];
  token_endpoint: string;
  claims_parameter_supported: boolean;
  introspection_endpoint: string;
  jwks_uri: string;
  registration_endpoint: string;
  request_uri_parameter_supported: boolean;
  response_modes_supported: string[];
  userinfo_endpoint: string;
  authorization_endpoint: string;
  device_authorization_endpoint: string;
  id_token_signing_alg_values_supported: string[];
  request_parameter_supported: boolean;
  response_types_supported: string[];
  token_endpoint_auth_methods_supported: string[];
}

export interface EmailProviderConfig {
  from: string;
  from_name: string;
  provider: string;
  reply_to: string;
  config: any;
}

export interface SecurityLevel {
  [key: string]: any;
}

export interface FinishRegisterRequest {
  name: string;
  response: any;
  userId: string;
}

export interface NoOpEmailProvider {
  [key: string]: any;
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

export interface ConsentTypeStatus {
  granted: boolean;
  grantedAt: string;
  needsRenewal: boolean;
  type: string;
  version: string;
  expiresAt: string | undefined;
}

export interface StepUpPoliciesResponse {
  policies: any[];
}

export interface ComplianceTemplatesResponse {
  templates: any[];
}

export interface TrustedContact {
}

export interface Challenge {
  attempts: number;
  createdAt: string;
  factorId: string;
  metadata: any;
  type: string;
  userAgent: string;
  verifiedAt: string | undefined;
  expiresAt: string;
  id: string;
  ipAddress: string;
  maxAttempts: number;
  status: string;
  userId: string;
}

export interface ComplianceProfileResponse {
  id: string;
}

export interface DeviceAuthorizationDecisionRequest {
  action: string;
  user_code: string;
}

export interface TokenIntrospectionRequest {
  client_id: string;
  client_secret: string;
  token: string;
  token_type_hint: string;
}

export interface MultiStepRecoveryConfig {
  allowUserChoice: boolean;
  enabled: boolean;
  requireAdminApproval: boolean;
  sessionExpiry: Duration;
  allowStepSkip: boolean;
  highRiskSteps: string[];
  lowRiskSteps: string[];
  mediumRiskSteps: string[];
  minimumSteps: number;
}

export interface UpdateMemberRequest {
  role: string;
}

export interface EmailProviderDTO {
  enabled: boolean;
  fromEmail: string;
  fromName: string;
  type: string;
  config: any;
}

export interface CreateTemplateResult {
  message: string;
  success: boolean;
  template: TemplateDTO;
}

export interface MemberHandler {
}

export interface Config {
  userverification: string;
  authenticatorattachment: string;
  rpid: string;
  rporigins: string[];
  attestationtype: string;
  challengestorage: string;
  requireresidentkey: boolean;
  rpname: string;
  timeout: Duration;
}

export interface ImpersonationErrorResponse {
  error: string;
}

export interface UpdateEntryRequest {
}

export interface ConsentService {
}

export interface TestProviderResult {
  message: string;
  success: boolean;
}

export interface ComplianceReportFileResponse {
  content_type: string;
  data: number[];
}

export interface DashboardConfig {
  enableTreeView: boolean;
  revealTimeout: Duration;
  enableExport: boolean;
  enableImport: boolean;
  enableReveal: boolean;
}

export interface PolicyStats {
  avgLatencyMs: number;
  denyCount: number;
  evaluationCount: number;
  policyId: string;
  policyName: string;
  allowCount: number;
}

export interface ContentEntryHandler {
}

export interface ClientAuthResult {
}

export interface TwoFAStatusResponse {
  enabled: boolean;
  method: string;
  trusted: boolean;
}

export interface CreateSecretOutput {
  secret: SecretItem;
}

export interface GetOrganizationInput {
  appId: string;
  orgId: string;
}

export interface SSOAuthResponse {
  session: Session | undefined;
  token: string;
  user: User | undefined;
}

export interface GetNotificationDetailInput {
  notificationId: string;
}

export interface FuncMap {
  [key: string]: any;
}

export interface ConnectionsResponse {
  connections: SocialAccount | undefined[];
}

export interface GetSecretsInput {
  pageSize: number;
  search: string;
  appId: string;
  page: number;
}

export interface CreateEvidence_req {
  controlId: string;
  description: string;
  evidenceType: string;
  fileUrl: string;
  standard: string;
  title: string;
}

export interface GetStatsRequestDTO {
}

export interface AuthAutoSendDTO {
  emailOtp: boolean;
  magicLink: boolean;
  mfaCode: boolean;
  passwordReset: boolean;
  verificationEmail: boolean;
  welcome: boolean;
}

export interface WebhookConfig {
  expiry_warning_days: number;
  notify_on_created: boolean;
  notify_on_deleted: boolean;
  notify_on_expiring: boolean;
  notify_on_rate_limit: boolean;
  notify_on_rotated: boolean;
  webhook_urls: string[];
  enabled: boolean;
}

export interface AuditServiceInterface {
  [key: string]: any;
}

export interface LoginResponse {
  passkeyUsed: string;
  session: any;
  token: string;
  user: any;
}

export interface GetChallengeStatusRequest {
}

export interface ProviderSession {
}

export interface Team {
  [key: string]: any;
}

export interface ConsentExportFileResponse {
  data: number[];
  content_type: string;
}

export interface GetValueRequest {
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

export interface RiskAssessment {
  factors: string[];
  level: string;
  metadata: any;
  recommended: string[];
  score: number;
}

export interface ComplianceReport {
  fileSize: number;
  format: string;
  generatedBy: string;
  id: string;
  reportType: string;
  standard: string;
  appId: string;
  createdAt: string;
  expiresAt: string;
  fileUrl: string;
  period: string;
  profileId: string;
  status: string;
  summary: any;
}

export interface UpdateTeamResult {
  team: TeamDTO;
}

export interface RestoreTemplateVersionRequest {
}

export interface CreateProfileFromTemplateRequest {
  standard: string;
}

export interface ListViolationsFilter {
  violationType: string | undefined;
  appId: string | undefined;
  profileId: string | undefined;
  severity: string | undefined;
  status: string | undefined;
  userId: string | undefined;
}

export interface OrganizationQuickLink {
  [key: string]: any;
}

export interface GetContentTypeRequest {
}

export interface ResourceRule {
  org_id: string;
  resource_type: string;
  security_level: string;
  sensitivity: string;
  action: string;
  description: string;
}

export interface CheckMetadata {
  autoRun: boolean;
  category: string;
  description: string;
  name: string;
  severity: string;
  standards: string[];
}

export interface StateStorageConfig {
  stateTtl: Duration;
  useRedis: boolean;
  redisAddr: string;
  redisDb: number;
  redisPassword: string;
}

export interface ConsentStatusResponse {
  status: string;
}

export interface ConsentCookieResponse {
  preferences: any;
}

export interface GetAnalyticsInput {
  days: number | undefined;
  endDate: string | undefined;
  startDate: string | undefined;
  templateId: string | undefined;
}

export interface ComplianceViolationsResponse {
  violations: any[];
}

export interface TemplateParameter {
  [key: string]: any;
}

export interface ListPasskeysResponse {
  count: number;
  passkeys: PasskeyInfo[];
}

export interface GetDocumentVerificationRequest {
  documentId: string;
}

export interface UpdateOrganizationHandlerRequest {
}

export interface MigrationHandler {
}

export interface AcceptInvitationRequest {
}

export interface NoOpNotificationProvider {
  [key: string]: any;
}

export interface VerifyResponse {
  token: string;
  user: User | undefined;
  session: Session | undefined;
  success: boolean;
}

export interface OrganizationStatsDTO {
  totalMembers: number;
  totalOrganizations: number;
  totalTeams: number;
}

export interface VerifyImpersonationRequest {
}

export interface ConsentManager {
}

export interface SetupSecurityQuestionsResponse {
  setupAt: string;
  count: number;
  message: string;
}

export interface ListMembersRequest {
}

export interface GetDocumentVerificationResponse {
  message: string;
  rejectionReason: string;
  status: string;
  verifiedAt: string | undefined;
  confidenceScore: number;
  documentId: string;
}

export interface AdminPolicyRequest {
  allowedTypes: string[];
  enabled: boolean;
  gracePeriod: number;
  requiredFactors: number;
}

export interface ProviderInfo {
  createdAt: string;
  domain: string;
  providerId: string;
  type: string;
}

export interface UnblockUserRequest {
  [key: string]: any;
}

export interface RequestReverification_req {
  reason: string;
}

export interface CreateTemplateVersion_req {
  changes: string;
}

export interface RiskLevel {
  [key: string]: any;
}

export interface EnrollFactorRequest {
  metadata: any;
  name: string;
  priority: string;
  type: string;
}

export interface FactorEnrollmentRequest {
  priority: string;
  type: string;
  metadata: any;
  name: string;
}

export interface UpdateMemberRoleInput {
  appId: string;
  memberId: string;
  orgId: string;
  role: string;
}

export interface PaginationInfoDTO {
  currentPage: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface VerifyCodeRequest {
  code: string;
  sessionId: string;
}

export interface DocumentVerification {
}

export interface SignUpResponse {
  message: string;
  status: string;
}

export interface GetSettingsResult {
  settings: OrganizationSettingsDTO;
}

export interface ProviderRegisteredResponse {
  providerId: string;
  status: string;
  type: string;
}

export interface DocumentVerificationRequest {
}

export interface ContextRule {
  description: string;
  name: string;
  org_id: string;
  security_level: string;
  condition: string;
}

export interface ListJWTKeysRequest {
}

export interface DeleteProviderRequest {
}

export interface UpdatePasskeyRequest {
  name: string;
}

export interface AddTrustedContactRequest {
  email: string;
  name: string;
  phone: string;
  relationship: string;
}

export interface SecretsConfigSource {
}

export interface TestProviderInput {
  recipient: string;
  providerType: string;
}

export interface SMSProviderConfig {
  config: any;
  from: string;
  provider: string;
}

export interface MigrateRolesRequest {
  dryRun: boolean;
}

export interface ComplianceTemplate {
  auditFrequencyDays: number;
  dataResidency: string;
  description: string;
  name: string;
  requiredPolicies: string[];
  requiredTraining: string[];
  sessionMaxAge: number;
  standard: string;
  mfaRequired: boolean;
  passwordMinLength: number;
  retentionDays: number;
}

export interface RouteRule {
  description: string;
  method: string;
  org_id: string;
  pattern: string;
  security_level: string;
}

export interface VerificationResponse {
  expiresAt: string | undefined;
  factorsRemaining: number;
  sessionComplete: boolean;
  success: boolean;
  token: string;
}

export interface UserVerificationStatusResponse {
  status: UserVerificationStatus | undefined;
}

export interface GetAnalyticsResult {
  analytics: AnalyticsDTO;
}

export interface ComplianceEvidence {
  fileHash: string;
  id: string;
  title: string;
  collectedBy: string;
  controlId: string;
  createdAt: string;
  fileUrl: string;
  metadata: any;
  profileId: string;
  standard: string;
  appId: string;
  description: string;
  evidenceType: string;
}

export interface ListChecksFilter {
  status: string | undefined;
  appId: string | undefined;
  checkType: string | undefined;
  profileId: string | undefined;
  sinceBefore: string | undefined;
}

export interface RevokeSessionRequestDTO {
}

export interface VerificationRequest {
  challengeId: string;
  code: string;
  data: any;
  deviceInfo: DeviceInfo | undefined;
  factorId: string;
  rememberDevice: boolean;
}

export interface MembersListResponse {
  [key: string]: any;
}

export interface CreateContentTypeRequest {
  [key: string]: any;
}

export interface CreateNamespaceRequest {
  description: string;
  inheritPlatform: boolean;
  name: string;
  templateId: string;
}

export interface ComplianceTrainingResponse {
  id: string;
}

export interface Repository {
  [key: string]: any;
}

export interface GetEntryStatsRequest {
}

export interface ValidateContentTypeRequest {
}

export interface CreateJWTKeyRequest {
  algorithm: string;
  curve: string;
  expiresAt: string | undefined;
  isPlatformKey: boolean;
  keyType: string;
  metadata: any;
}

export interface MigrationStatusResponse {
  appId: string;
  completedAt: string | undefined;
  environmentId: string;
  failedCount: number;
  migratedCount: number;
  progress: number;
  userOrganizationId: string | undefined;
  errors: string[];
  startedAt: string;
  status: string;
  totalPolicies: number;
  validationPassed: boolean;
}

export interface MFARepository {
  [key: string]: any;
}

export interface AsyncAdapter {
}

export interface AdminUpdateProviderRequest {
  clientId: string | undefined;
  clientSecret: string | undefined;
  enabled: boolean | undefined;
  scopes: string[];
}

export interface RetryService {
  [key: string]: any;
}

export interface CreateSecretRequest {
  metadata: any;
  path: string;
  tags: string[];
  value: any;
  valueType: string;
  description: string;
}

export interface TokenRequest {
  code: string;
  code_verifier: string;
  device_code: string;
  grant_type: string;
  redirect_uri: string;
  refresh_token: string;
  scope: string;
  audience: string;
  client_id: string;
  client_secret: string;
}

export interface ValidatePolicyResponse {
  message: string;
  valid: boolean;
  warnings: string[];
  complexity: number;
  error: string;
  errors: string[];
}

export interface GetAppRequest {
}

export interface OAuthErrorResponse {
  error: string;
  error_description: string;
  error_uri: string;
  state: string;
}

export interface DocumentVerificationResult {
}

export interface WidgetDataDTO {
  title: string;
  content: string;
  icon: string;
  id: string;
  order: number;
  requireAdmin: boolean;
  size: number;
}

export interface SMSProviderDTO {
  config: any;
  enabled: boolean;
  type: string;
}

export interface Time {
  [key: string]: any;
}

export interface OAuthTokenRepository {
  [key: string]: any;
}

export interface MockUserRepository {
  [key: string]: any;
}

export interface RollbackRequest {
  reason: string;
}

export interface InvitationDTO {
  inviterName: string;
  role: string;
  status: string;
  createdAt: string;
  email: string;
  expiresAt: string;
  id: string;
  invitedBy: string;
}

export interface FacialCheckConfig {
  enabled: boolean;
  motionCapture: boolean;
  variant: string;
}

export interface BackupAuthCodesResponse {
  codes: string[];
}

export interface OIDCLoginResponse {
  authUrl: string;
  nonce: string;
  providerId: string;
  state: string;
}

export interface ListNotificationsHistoryInput {
  recipient: string | undefined;
  status: string | undefined;
  type: string | undefined;
  limit: number;
  page: number;
}

export interface ComplianceChecksResponse {
  checks: any[];
}

export interface BackupAuthRecoveryResponse {
  session_id: string;
}

export interface VerifyTokenRequest {
  tokenType: string;
  audience: string[];
  token: string;
}

export interface GetMembersResult {
  canManage: boolean;
  data: MemberDTO[];
  pagination: PaginationInfo;
}

export interface OrganizationSettingsDTO {
  allowUserCreation: boolean;
  defaultRole: string;
  maxMembersPerOrg: number;
  requireInvitation: boolean;
  allowMultipleMemberships: boolean;
  enabled: boolean;
  invitationExpiryDays: number;
  maxOrgsPerUser: number;
  maxTeamsPerOrg: number;
}

export interface GetOrganizationsResult {
  data: OrganizationSummaryDTO[];
  pagination: PaginationInfo;
  stats: OrganizationStatsDTO;
}

export interface UpdateAPIKeyRequest {
  metadata: any;
  name: string | undefined;
  permissions: any;
  rate_limit: number | undefined;
  scopes: string[];
  allowed_ips: string[];
  description: string | undefined;
}

export interface RateLimit {
}

export interface SignInRequest {
  provider: string;
  redirectUrl: string;
  scopes: string[];
}

export interface TOTPFactorAdapter {
}

export interface BackupCodesConfig {
  allow_reuse: boolean;
  count: number;
  enabled: boolean;
  format: string;
  length: number;
}

export interface GetOrganizationBySlugRequest {
}

export interface RevokeSessionInput {
  appId: string;
  sessionId: string;
}

export interface FactorStatus {
  [key: string]: any;
}

export interface BulkDeleteRequest {
  ids: string[];
}

export interface VideoVerificationConfig {
  sessionDuration: Duration;
  enabled: boolean;
  livenessThreshold: number;
  provider: string;
  recordSessions: boolean;
  requireScheduling: boolean;
  minScheduleAdvance: Duration;
  recordingRetention: Duration;
  requireAdminReview: boolean;
  requireLivenessCheck: boolean;
}

export interface AdaptiveMFAConfig {
  enabled: boolean;
  factor_ip_reputation: boolean;
  factor_location_change: boolean;
  new_device_risk: number;
  require_step_up_threshold: number;
  velocity_risk: number;
  factor_new_device: boolean;
  factor_velocity: boolean;
  location_change_risk: number;
  risk_threshold: number;
}

export interface SessionAutoSendDTO {
  allRevoked: boolean;
  deviceRemoved: boolean;
  newDevice: boolean;
  newLocation: boolean;
  suspiciousLogin: boolean;
}

export interface ServiceImpl {
  [key: string]: any;
}

export interface ProviderRegistry {
  [key: string]: any;
}

export interface CreateTeamRequest {
  description: string;
  name: string;
}

export interface EnableRequest {
}

export interface TimeBasedRule {
  description: string;
  max_age: Duration;
  operation: string;
  org_id: string;
  security_level: string;
}

export interface DeleteFactorRequest {
}

export interface ConfigSourceConfig {
  prefix: string;
  priority: number;
  refreshInterval: Duration;
  autoRefresh: boolean;
  enabled: boolean;
}

export interface DB {
  [key: string]: any;
}

export interface GetSessionsInput {
  status: string;
  userId: string;
  appId: string;
  device: string;
  page: number;
  pageSize: number;
  search: string;
}

export interface GetMembersInput {
  appId: string;
  limit: number;
  orgId: string;
  page: number;
  search: string;
}

export interface ListAPIKeysResponse {
  [key: string]: any;
}

export interface RejectRecoveryRequest {
  reason: string;
  sessionId: string;
  notes: string;
}

export interface DataExportRequestInput {
  format: string;
  includeSections: string[];
}

export interface CreateOrganizationHandlerRequest {
  [key: string]: any;
}

export interface AccountAutoSendConfig {
  deleted: boolean;
  email_change_request: boolean;
  email_changed: boolean;
  password_changed: boolean;
  reactivated: boolean;
  suspended: boolean;
  username_changed: boolean;
}

export interface InviteMemberInput {
  appId: string;
  email: string;
  orgId: string;
  role: string;
}

export interface ImpersonationStartResponse {
  impersonator_id: string;
  session_id: string;
  started_at: string;
  target_user_id: string;
}

export interface SendResponse {
  dev_otp: string;
  status: string;
}

export interface RequirementsResponse {
  count: number;
  requirements: any;
}

export interface VerifyTokenResponse {
  [key: string]: any;
}

export interface Plugin {
}

export interface GetRoleTemplatesResult {
  templates: RoleTemplateDTO[];
}

export interface CreateSessionRequest {
}

export interface AppHandler {
}

export interface GetTeamRequest {
}

export interface FactorsResponse {
  factors: any;
  count: number;
}

export interface DeleteSecretRequest {
}

export interface PagesManager {
  [key: string]: any;
}

export interface ImpersonationVerifyResponse {
  impersonator_id: string;
  is_impersonating: boolean;
  target_user_id: string;
}

export interface TemplateResponse {
  name: string;
  parameters: TemplateParameter[];
  category: string;
  description: string;
  examples: string[];
  expression: string;
  id: string;
}

export interface App {
}

export interface ImpersonationMiddleware {
}

export interface DeviceAuthorizationResponse {
  device_code: string;
  expires_in: number;
  interval: number;
  user_code: string;
  verification_uri: string;
  verification_uri_complete: string;
}

export interface GetStatusRequest {
  user_id: string;
  device_id: string;
}

export interface StepUpPolicyResponse {
  id: string;
}

export interface SignUpRequest {
  password: string;
  username: string;
}

export interface IDVerificationStatusResponse {
  status: any;
}

export interface IPWhitelistConfig {
  enabled: boolean;
  strict_mode: boolean;
}

export interface StepUpEvaluationResponse {
  reason: string;
  required: boolean;
}

export interface EncryptionConfig {
  masterKey: string;
  rotateKeyAfter: Duration;
  testOnStartup: boolean;
}

export interface PasskeyInfo {
  authenticatorType: string;
  createdAt: string;
  credentialId: string;
  isResidentKey: boolean;
  lastUsedAt: string | undefined;
  name: string;
  aaguid: string;
  id: string;
  signCount: number;
}

export interface JWK {
  e: string;
  kid: string;
  kty: string;
  n: string;
  use: string;
  alg: string;
}

export interface CallbackResult {
}

export interface StepUpRequirement {
  currency: string;
  current_level: string;
  resource_type: string;
  amount: number;
  challenge_token: string;
  required_level: string;
  user_agent: string;
  fulfilled_at: string | undefined;
  id: string;
  ip: string;
  metadata: any;
  method: string;
  reason: string;
  resource_action: string;
  route: string;
  created_at: string;
  expires_at: string;
  org_id: string;
  risk_score: number;
  rule_name: string;
  session_id: string;
  status: string;
  user_id: string;
}

export interface GetInvitationsInput {
  appId: string;
  limit: number;
  orgId: string;
  page: number;
  status: string;
}

export interface BackupAuthContactsResponse {
  contacts: any[];
}

export interface FactorVerificationRequest {
  code: string;
  data: any;
  factorId: string;
}

export interface RevealSecretOutput {
  value: any;
  valueType: string;
}

export interface NotificationSettingsDTO {
  account: AccountAutoSendDTO;
  appName: string;
  auth: AuthAutoSendDTO;
  organization: OrganizationAutoSendDTO;
  session: SessionAutoSendDTO;
}

export interface NotificationTemplateResponse {
  template: any;
}

export interface PreviewTemplateResult {
  body: string;
  renderedAt: string;
  subject: string;
}

export interface DeviceCodeEntryResponse {
  basePath: string;
  formAction: string;
  placeholder: string;
}

export interface ReorderFieldsRequest {
}

export interface BanUserRequestDTO {
  expires_at: string | undefined;
  reason: string;
}

export interface SMSConfig {
  code_expiry_minutes: number;
  code_length: number;
  enabled: boolean;
  provider: string;
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
}

export interface UpdateTemplateResult {
  success: boolean;
  template: TemplateDTO;
  message: string;
}

export interface ListUsersRequestDTO {
}

export interface RoleTemplateDTO {
  description: string;
  id: string;
  name: string;
  permissions: string[];
  updatedAt: string;
  createdAt: string;
}

export interface DeclareABTestWinnerRequest {
}

export interface ChannelsResponse {
  channels: any;
  count: number;
}

export interface PreviewConversionRequest {
  actions: string[];
  condition: string;
  resource: string;
  subject: string;
}

export interface ComplianceViolation {
  id: string;
  resolvedBy: string;
  severity: string;
  userId: string;
  violationType: string;
  appId: string;
  createdAt: string;
  metadata: any;
  profileId: string;
  resolvedAt: string | undefined;
  status: string;
  description: string;
}

export interface ConsentExpiryConfig {
  allowRenewal: boolean;
  autoExpireCheck: boolean;
  defaultValidityDays: number;
  enabled: boolean;
  expireCheckInterval: Duration;
  renewalReminderDays: number;
  requireReConsent: boolean;
}

export interface OrganizationDetailDTO {
  metadata: any;
  name: string;
  slug: string;
  updatedAt: string;
  createdAt: string;
  id: string;
  logo: string;
}

export interface TemplateEngine {
}

export interface RunCheck_req {
  checkType: string;
}

export interface FactorType {
  [key: string]: any;
}

export interface RecoveryMethod {
  [key: string]: any;
}

export interface OrganizationHandler {
}

export interface TeamsResponse {
  total: number;
  teams: Team | undefined[];
}

export interface DeleteTeamResult {
  success: boolean;
}

export interface RetentionConfig {
  archiveBeforePurge: boolean;
  archivePath: string;
  enabled: boolean;
  gracePeriodDays: number;
  purgeSchedule: string;
}

export interface AppsListResponse {
  [key: string]: any;
}

export interface VerificationRepository {
}

export interface BaseFactorAdapter {
}

export interface DiscoverProviderRequest {
  email: string;
}

export interface LinkResponse {
  user: any;
  message: string;
}

export interface DeclineInvitationRequest {
}

export interface ListContentTypesRequest {
}

export interface VideoVerificationSession {
}

export interface ProviderListResponse {
  providers: ProviderInfo[];
  total: number;
}

export interface CreateActionRequest {
  namespaceId: string;
  description: string;
  name: string;
}

export interface Organization {
  [key: string]: any;
}

export interface TestSendTemplateInput {
  recipient: string;
  templateId: string;
  variables: any;
}

export interface UpdateMemberRoleRequest {
  role: string;
}

export interface JWKS {
  keys: JWK[];
}

export interface TrackNotificationEvent_req {
  event: string;
  eventData?: any;
  notificationId: string;
  organizationId?: string | undefined;
  templateId: string;
}

export interface ImpersonationListResponse {
  [key: string]: any;
}

export interface RecoveryAttemptLog {
}

export interface DisableRequest {
  user_id: string;
}

export interface MockRepository {
}

export interface ConsentReport {
  pendingDeletions: number;
  reportPeriodStart: string;
  totalUsers: number;
  completedDeletions: number;
  consentRate: number;
  dpasActive: number;
  dpasExpiringSoon: number;
  organizationId: string;
  reportPeriodEnd: string;
  usersWithConsent: number;
  consentsByType: any;
  dataExportsThisPeriod: number;
}

export interface MockStateStore {
}

export interface MockUserService {
  [key: string]: any;
}

export interface NoOpVideoProvider {
  [key: string]: any;
}

export interface ProviderCheckResult {
}

export interface GetTemplateResult {
  template: TemplateDTO;
}

export interface ResourceAttributeRequest {
  description: string;
  name: string;
  required: boolean;
  type: string;
  default: any;
}

export interface GenerateReportRequest {
  period: string;
  reportType: string;
  standard: string;
  format: string;
}

export interface RWMutex {
  [key: string]: any;
}

export interface UserInfoResponse {
  family_name: string;
  given_name: string;
  nickname: string;
  preferred_username: string;
  profile: string;
  website: string;
  email: string;
  locale: string;
  name: string;
  updated_at: number;
  birthdate: string;
  email_verified: boolean;
  phone_number: string;
  picture: string;
  sub: string;
  zoneinfo: string;
  gender: string;
  middle_name: string;
  phone_number_verified: boolean;
}

export interface KeyStore {
}

export interface GenerateRecoveryCodesResponse {
  warning: string;
  codes: string[];
  count: number;
  generatedAt: string;
}

export interface HealthCheckResponse {
  enabledMethods: string[];
  healthy: boolean;
  message: string;
  providersStatus: any;
  version: string;
}

export interface StatusResponse {
  emailVerified: boolean;
  emailVerifiedAt: string | undefined;
}

export interface ConsentRecord {
  consentType: string;
  id: string;
  organizationId: string;
  createdAt: string;
  granted: boolean;
  grantedAt: string;
  userId: string;
  version: string;
  userAgent: string;
  ipAddress: string;
  metadata: Record<string, any>;
  purpose: string;
  revokedAt: string | undefined;
  updatedAt: string;
  expiresAt: string | undefined;
}

export interface PrivacySettingsRequest {
  ccpaMode: boolean | undefined;
  contactEmail: string;
  dataExportExpiryHours: number | undefined;
  exportFormat: string[];
  consentRequired: boolean | undefined;
  cookieConsentStyle: string;
  deletionGracePeriodDays: number | undefined;
  requireAdminApprovalForDeletion: boolean | undefined;
  requireExplicitConsent: boolean | undefined;
  anonymousConsentEnabled: boolean | undefined;
  autoDeleteAfterDays: number | undefined;
  dpoEmail: string;
  allowDataPortability: boolean | undefined;
  contactPhone: string;
  cookieConsentEnabled: boolean | undefined;
  dataRetentionDays: number | undefined;
  gdprMode: boolean | undefined;
}

export interface CancelInvitationResult {
  success: boolean;
}

export interface IntrospectionService {
}

export interface SMSVerificationConfig {
  codeExpiry: Duration;
  codeLength: number;
  cooldownPeriod: Duration;
  enabled: boolean;
  maxAttempts: number;
  maxSmsPerDay: number;
  messageTemplate: string;
  provider: string;
}

export interface UnbanUserRequestDTO {
  reason: string;
}

export interface DeleteTemplateRequest {
}

export interface ComplianceViolationResponse {
  id: string;
}

export interface VerificationMethod {
  [key: string]: any;
}

export interface ImpersonateUserRequestDTO {
  duration: Duration;
}

export interface ListTemplatesInput {
  language: string | undefined;
  limit: number;
  page: number;
  type: string | undefined;
  active: boolean | undefined;
}

export interface RevokeAllUserSessionsResult {
  message: string;
  revokedCount: number;
  success: boolean;
}

export interface ComplianceStatusDetailsResponse {
  status: string;
}

export interface ClientAuthenticator {
}

export interface VerifyCodeResponse {
  attemptsLeft: number;
  message: string;
  valid: boolean;
}

export interface DataExportRequest {
  includeSections: string[];
  status: string;
  createdAt: string;
  expiresAt: string | undefined;
  exportUrl: string;
  updatedAt: string;
  completedAt: string | undefined;
  exportPath: string;
  format: string;
  id: string;
  ipAddress: string;
  errorMessage: string;
  exportSize: number;
  organizationId: string;
  userId: string;
}

export interface OnfidoConfig {
  workflowId: string;
  documentCheck: DocumentCheckConfig;
  enabled: boolean;
  includeFacialReport: boolean;
  webhookToken: string;
  apiToken: string;
  facialCheck: FacialCheckConfig;
  includeDocumentReport: boolean;
  includeWatchlistReport: boolean;
  region: string;
}

export interface DeleteFieldRequest {
}

export interface AuditConfig {
  logFailed: boolean;
  logIpAddress: boolean;
  logUserAgent: boolean;
  retentionDays: number;
  archiveInterval: Duration;
  archiveOldLogs: boolean;
  enabled: boolean;
  immutableLogs: boolean;
  logDeviceInfo: boolean;
  logSuccessful: boolean;
  logAllAttempts: boolean;
}

export interface RejectRecoveryResponse {
  message: string;
  reason: string;
  rejected: boolean;
  rejectedAt: string;
  sessionId: string;
}

export interface SecurityQuestion {
}

export interface VerifyRecoveryCodeResponse {
  message: string;
  remainingCodes: number;
  valid: boolean;
}

export interface VerifySecurityAnswersResponse {
  correctAnswers: number;
  message: string;
  requiredAnswers: number;
  valid: boolean;
  attemptsLeft: number;
}

export interface MockService {
}

export interface CheckSubResult {
}

export interface GetUserSessionsInput {
  appId: string;
  page: number;
  pageSize: number;
  userId: string;
}

export interface ContentTypeService {
  [key: string]: any;
}

export interface StatsDTO {
  [key: string]: any;
}

export interface CookieConsentConfig {
  defaultStyle: string;
  enabled: boolean;
  requireExplicit: boolean;
  validityPeriod: Duration;
  allowAnonymous: boolean;
  bannerVersion: string;
  categories: string[];
}

export interface PoliciesListResponse {
  policies: PolicyResponse | undefined[];
  totalCount: number;
  page: number;
  pageSize: number;
}

export interface BeginLoginRequest {
  userId: string;
  userVerification: string;
}

export interface SessionStats {
}

export interface DashboardExtension {
}

export interface OrganizationUIRegistry {
}

export interface Email {
}

export interface ScopeInfo {
}

export interface BackupAuthStatsResponse {
  stats: any;
}

export interface Device {
  lastUsedAt: string;
  ipAddress?: string;
  userAgent?: string;
  id: string;
  userId: string;
  name?: string;
  type?: string;
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

export interface AMLMatch {
}

export interface GetOrganizationResult {
  organization: OrganizationDetailDTO;
  stats: OrgDetailStatsDTO;
  userRole: string;
}

export interface IDVerificationWebhookResponse {
  status: string;
}

export interface VerifyAPIKeyResponse {
  [key: string]: any;
}

export interface NoOpSMSProvider {
  [key: string]: any;
}

export interface ProvidersConfig {
  gitlab: ProviderConfig | undefined;
  google: ProviderConfig | undefined;
  line: ProviderConfig | undefined;
  spotify: ProviderConfig | undefined;
  apple: ProviderConfig | undefined;
  dropbox: ProviderConfig | undefined;
  facebook: ProviderConfig | undefined;
  linkedin: ProviderConfig | undefined;
  microsoft: ProviderConfig | undefined;
  slack: ProviderConfig | undefined;
  twitch: ProviderConfig | undefined;
  twitter: ProviderConfig | undefined;
  bitbucket: ProviderConfig | undefined;
  discord: ProviderConfig | undefined;
  github: ProviderConfig | undefined;
  reddit: ProviderConfig | undefined;
  notion: ProviderConfig | undefined;
}

export interface ID {
  [key: string]: any;
}

export interface MockAppService {
  [key: string]: any;
}

export interface GetSecretRequest {
}

export interface BlockUserRequest {
  reason: string;
}

export interface NotificationStatusResponse {
  status: string;
}

export interface CreateTraining_req {
  userId: string;
  standard: string;
  trainingType: string;
}

export interface IdentityVerification {
  [key: string]: any;
}

export interface CreateSecretInput {
  appId: string;
  description: string;
  path: string;
  tags: string[];
  value: any;
  valueType: string;
}

export interface IDVerificationErrorResponse {
  error: string;
}

export interface RevokeAllUserSessionsInput {
  appId: string;
  userId: string;
}

export interface CompleteTrainingRequest {
  score: number;
}

export interface ClientsListResponse {
  clients: ClientSummary[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface GetRecoveryStatsResponse {
  adminReviewsRequired: number;
  failedRecoveries: number;
  highRiskAttempts: number;
  pendingRecoveries: number;
  successfulRecoveries: number;
  totalAttempts: number;
  averageRiskScore: number;
  methodStats: any;
  successRate: number;
}

export interface GetRoleTemplateInput {
  appId: string;
  templateId: string;
}

export interface DeleteTemplateResult {
  message: string;
  success: boolean;
}

export interface EnableRequest2FA {
  method: string;
  user_id: string;
}

export interface WebAuthnFactorAdapter {
}

export interface AccountLockedResponse {
  code: string;
  locked_minutes: number;
  locked_until: string;
  message: string;
}

export interface DeleteOrganizationResult {
  success: boolean;
}

export interface NotificationListResponse {
  total: number;
  notifications: any[];
}

export interface ListRecoverySessionsResponse {
  totalCount: number;
  page: number;
  pageSize: number;
  sessions: RecoverySessionInfo[];
}

export interface GetByPathRequest {
}

export interface ImpersonationSession {
  [key: string]: any;
}

export interface ConsentDecision {
}

export interface DeleteTeamInput {
  teamId: string;
  appId: string;
  orgId: string;
}

export interface ListProfilesFilter {
  appId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface CallbackDataResponse {
  user: User | undefined;
  action: string;
  isNewUser: boolean;
}

export interface CreateRoleTemplateResult {
  template: RoleTemplateDTO;
}

export interface SAMLLoginResponse {
  providerId: string;
  redirectUrl: string;
  requestId: string;
}

export interface AccessTokenClaims {
  client_id: string;
  scope: string;
  token_type: string;
}

export interface Status {
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

export interface CreateTeamResult {
  team: TeamDTO;
}

export interface Adapter {
}

export interface CompliancePolicyResponse {
  id: string;
}

export interface AccessConfig {
  allowApiAccess: boolean;
  allowDashboardAccess: boolean;
  rateLimitPerMinute: number;
  requireAuthentication: boolean;
  requireRbac: boolean;
}

export interface RegistrationService {
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

export interface DocumentCheckConfig {
  validateDataConsistency: boolean;
  validateExpiry: boolean;
  enabled: boolean;
  extractData: boolean;
}

export interface SecurityQuestionsConfig {
  forbidCommonAnswers: boolean;
  lockoutDuration: Duration;
  minimumQuestions: number;
  requiredToRecover: number;
  maxAnswerLength: number;
  maxAttempts: number;
  predefinedQuestions: string[];
  requireMinLength: number;
  allowCustomQuestions: boolean;
  caseSensitive: boolean;
  enabled: boolean;
}

export interface KeyPair {
}

export interface SetupSecurityQuestionRequest {
  answer: string;
  customText: string;
  questionId: number;
}

export interface BackupAuthConfigResponse {
  config: any;
}

export interface MemberDTO {
  userId: string;
  userName: string;
  id: string;
  joinedAt: string;
  role: string;
  status: string;
  userEmail: string;
}

export interface OrganizationSummaryDTO {
  userRole: string;
  createdAt: string;
  id: string;
  logo: string;
  memberCount: number;
  name: string;
  slug: string;
  teamCount: number;
}

export interface DailyAnalyticsDTO {
  totalOpened: number;
  totalSent: number;
  date: string;
  deliveryRate: number;
  openRate: number;
  totalClicked: number;
  totalDelivered: number;
}

export interface GetSessionStatsInput {
  appId: string;
}

export interface TeamsListResponse {
  [key: string]: any;
}

export interface RecoverySession {
}

export interface TOTPConfig {
  algorithm: string;
  digits: number;
  enabled: boolean;
  issuer: string;
  period: number;
  window_size: number;
}

export interface ComplianceUserTrainingResponse {
  user_id: string;
}

export interface ListRevisionsRequest {
}

export interface RemoveTrustedContactRequest {
  contactId: string;
}

export interface GetSecretInput {
  appId: string;
  secretId: string;
}

export interface BatchEvaluateRequest {
  requests: EvaluateRequest[];
}

export interface RevokeTokenService {
}

export interface ListRecoverySessionsRequest {
  organizationId: string;
  page: number;
  pageSize: number;
  requiresReview: boolean;
  status: string;
}

export interface MockOrganizationUIExtension {
}

export interface TemplateService {
}

export interface DeleteTemplateInput {
  templateId: string;
}

export interface TestPolicyResponse {
  error: string;
  failedCount: number;
  passed: boolean;
  passedCount: number;
  results: TestCaseResult[];
  total: number;
}

export interface RBACMigrationService {
  [key: string]: any;
}

export interface RateLimitRule {
  window: Duration;
  max: number;
}

export interface Middleware {
}

export interface AutoSendConfig {
  account: AccountAutoSendConfig;
  auth: AuthAutoSendConfig;
  organization: OrganizationAutoSendConfig;
  session: SessionAutoSendConfig;
}

export interface ListPoliciesFilter {
  appId: string | undefined;
  policyType: string | undefined;
  profileId: string | undefined;
  standard: string | undefined;
  status: string | undefined;
}

export interface AuditLog {
}

export interface ListVersionsResponse {
  [key: string]: any;
}

export interface RecoveryStatus {
  [key: string]: any;
}

export interface StartRecoveryResponse {
  expiresAt: string;
  requiredSteps: number;
  requiresReview: boolean;
  riskScore: number;
  sessionId: string;
  status: string;
  availableMethods: string[];
  completedSteps: number;
}

export interface OnfidoProvider {
}

export interface RotateAPIKeyRequest {
}

export interface BeginRegisterRequest {
  authenticatorType: string;
  name: string;
  requireResidentKey: boolean;
  userId: string;
  userVerification: string;
}

export interface ArchiveEntryRequest {
}

export interface SetupSecurityQuestionsRequest {
  questions: SetupSecurityQuestionRequest[];
}

export interface SendOTPRequest {
  user_id: string;
}

export interface ProvidersResponse {
  providers: string[];
}

export interface RevealSecretInput {
  appId: string;
  secretId: string;
}

export interface NotificationPreviewResponse {
  body: string;
  subject: string;
}

export interface TokenRevocationRequest {
  client_id: string;
  client_secret: string;
  token: string;
  token_type_hint: string;
}

export interface UpdateMemberRoleResult {
  member: MemberDTO;
}

export interface SessionTokenResponse {
  session: Session | undefined;
  token: string;
}

export interface GetMigrationStatusResponse {
  hasMigratedPolicies: boolean;
  lastMigrationAt: string;
  migratedCount: number;
  pendingRbacPolicies: number;
}

export interface ComplianceTemplateResponse {
  standard: string;
}

export interface NotificationType {
  [key: string]: any;
}

export interface BeginRegisterResponse {
  challenge: string;
  options: any;
  timeout: Duration;
  userId: string;
}

export interface ConsentPolicyResponse {
  id: string;
}

export interface ActionDataDTO {
  order: number;
  requireAdmin: boolean;
  style: string;
  action: string;
  icon: string;
  id: string;
  label: string;
}

export interface TestCase {
  action: string;
  expected: boolean;
  name: string;
  principal: any;
  request: any;
  resource: any;
}

export interface ScopeDefinition {
}

export interface TrustedContactInfo {
  name: string;
  phone: string;
  relationship: string;
  verified: boolean;
  verifiedAt: string | undefined;
  active: boolean;
  email: string;
  id: string;
}

export interface ScheduleVideoSessionRequest {
  timeZone: string;
  scheduledAt: string;
  sessionId: string;
}

export interface GetSecretOutput {
  secret: SecretItem;
}

export interface SaveBuilderTemplateResult {
  message: string;
  success: boolean;
  templateId: string;
}

export interface CompleteTraining_req {
  score: number;
}

export interface ComplianceStandard {
  [key: string]: any;
}

export interface VerifyTrustedContactRequest {
  token: string;
}

export interface CreateTeamInput {
  name: string;
  orgId: string;
  appId: string;
  description: string;
  metadata: any;
}

export interface UpdateSecretRequest {
  metadata: any;
  tags: string[];
  value: any;
  description: string;
}

export interface ComplianceReportsResponse {
  reports: any[];
}

export interface UserVerificationStatus {
  [key: string]: any;
}

export interface MemoryChallengeStore {
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

export interface Duration {
  [key: string]: any;
}

export interface ImpersonationAuditResponse {
  [key: string]: any;
}

export interface ProvidersAppResponse {
  appId: string;
  providers: string[];
}

export interface ConsentAuditLog {
  userAgent: string;
  userId: string;
  consentId: string;
  consentType: string;
  id: string;
  ipAddress: string;
  organizationId: string;
  previousValue: Record<string, any>;
  action: string;
  createdAt: string;
  newValue: Record<string, any>;
  purpose: string;
  reason: string;
}

export interface ConsentExportResponse {
  id: string;
  status: string;
}

export interface WebAuthnWrapper {
}

export interface RecoverySessionInfo {
  userEmail: string;
  completedAt: string | undefined;
  createdAt: string;
  currentStep: number;
  expiresAt: string;
  method: string;
  requiresReview: boolean;
  userId: string;
  id: string;
  riskScore: number;
  status: string;
  totalSteps: number;
}

export interface GetSecurityQuestionsRequest {
  sessionId: string;
}

export interface VerifyRequest2FA {
  code: string;
  device_id: string;
  remember_device: boolean;
  user_id: string;
}

export interface BanUserRequest {
  user_organization_id: string | undefined;
  app_id: string;
  expires_at: string | undefined;
  reason: string;
  user_id: string;
}

export interface MembersResponse {
  members: Member | undefined[];
  total: number;
}

export interface AnalyticsDTO {
  topTemplates: TemplatePerformanceDTO[];
  byDay: DailyAnalyticsDTO[];
  byTemplate: TemplateAnalyticsDTO[];
  overview: OverviewStatsDTO;
}

export interface ServiceInterface {
  [key: string]: any;
}

export interface TokenResponse {
  id_token: string;
  refresh_token: string;
  scope: string;
  token_type: string;
  access_token: string;
  expires_in: number;
}

export interface UpdateRecoveryConfigRequest {
  minimumStepsRequired: number;
  requireAdminReview: boolean;
  requireMultipleSteps: boolean;
  riskScoreThreshold: number;
  enabledMethods: string[];
}

export interface TrustedDevice {
  userId: string;
  deviceId: string;
  expiresAt: string;
  userAgent: string;
  createdAt: string;
  id: string;
  ipAddress: string;
  lastUsedAt: string | undefined;
  metadata: any;
  name: string;
}

export interface SchemaValidator {
  [key: string]: any;
}

export interface CreateRoleTemplateInput {
  description: string;
  name: string;
  permissions: string[];
  appId: string;
}

export interface DeclareABTestWinner_req {
  abTestGroup: string;
  winnerId: string;
}

export interface UpdateSecretOutput {
  secret: SecretItem;
}

export interface ActionsListResponse {
  actions: ActionResponse | undefined[];
  totalCount: number;
}

export interface RegenerateCodesRequest {
  count: number;
  user_id: string;
}

export interface GetABTestResultsRequest {
}

export interface NotificationsResponse {
  count: number;
  notifications: any;
}

export interface ChallengeStatus {
  [key: string]: any;
}

export interface ClientDetailsResponse {
  contacts: string[];
  grantTypes: string[];
  isOrgLevel: boolean;
  organizationID: string;
  redirectURIs: string[];
  requireConsent: boolean;
  tosURI: string;
  responseTypes: string[];
  tokenEndpointAuthMethod: string;
  clientID: string;
  policyURI: string;
  requirePKCE: boolean;
  updatedAt: string;
  allowedScopes: string[];
  createdAt: string;
  logoURI: string;
  name: string;
  postLogoutRedirectURIs: string[];
  trustedClient: boolean;
  applicationType: string;
}

export interface DeleteUserRequestDTO {
}

export interface CloneContentTypeRequest {
}

export interface ClientRegistrationRequest {
  client_name: string;
  require_consent: boolean;
  require_pkce: boolean;
  trusted_client: boolean;
  contacts: string[];
  grant_types: string[];
  policy_uri: string;
  post_logout_redirect_uris: string[];
  scope: string;
  application_type: string;
  logo_uri: string;
  redirect_uris: string[];
  token_endpoint_auth_method: string;
  response_types: string[];
  tos_uri: string;
}

export interface UpdateNamespaceRequest {
  description: string;
  inheritPlatform: boolean | undefined;
  name: string;
}

export interface GetEffectivePermissionsRequest {
}

export interface EmailVerificationConfig {
  enabled: boolean;
  fromAddress: string;
  fromName: string;
  maxAttempts: number;
  requireEmailProof: boolean;
  codeExpiry: Duration;
  codeLength: number;
  emailTemplate: string;
}

export interface ScheduleVideoSessionResponse {
  joinUrl: string;
  message: string;
  scheduledAt: string;
  videoSessionId: string;
  instructions: string;
}

export interface GetRecoveryStatsRequest {
  endDate: string;
  organizationId: string;
  startDate: string;
}

export interface CreateOrganizationInput {
  appId: string;
  logo: string;
  metadata: any;
  name: string;
  slug: string;
}

export interface SaveBuilderTemplateInput {
  builderJson: string;
  name: string;
  subject: string;
  templateId: string;
  templateKey: string;
}

export interface ResourcesListResponse {
  totalCount: number;
  resources: ResourceResponse | undefined[];
}

export interface ReportsConfig {
  enabled: boolean;
  formats: string[];
  includeEvidence: boolean;
  retentionDays: number;
  schedule: string;
  storagePath: string;
}

export interface GetAuditLogsRequestDTO {
}

export interface MessageResponse {
  message: string;
}

export interface GetTemplateAnalyticsRequest {
}

export interface RevokeResponse {
  revokedCount: number;
  status: string;
}

export interface ClientUpdateRequest {
  require_consent: boolean | undefined;
  require_pkce: boolean | undefined;
  token_endpoint_auth_method: string;
  trusted_client: boolean | undefined;
  allowed_scopes: string[];
  contacts: string[];
  grant_types: string[];
  logo_uri: string;
  redirect_uris: string[];
  response_types: string[];
  tos_uri: string;
  name: string;
  policy_uri: string;
  post_logout_redirect_uris: string[];
}

export interface GetTemplateInput {
  templateId: string;
}

export interface ComplianceDashboardResponse {
  metrics: any;
}

export interface ContentFieldService {
  [key: string]: any;
}

export interface SuccessResponse {
  data: any;
  message: string;
}

export interface UpdatePolicyRequest {
  active: boolean | undefined;
  content: string;
  description: string;
  metadata: any;
  name: string;
  renewable: boolean | undefined;
  required: boolean | undefined;
  validityPeriod: number | undefined;
}

export interface EmailFactorAdapter {
}

export interface GetInvitationsResult {
  pagination: PaginationInfo;
  data: InvitationDTO[];
}

export interface GetSessionsResult {
  pagination: PaginationInfoDTO;
  sessions: SessionDTO[];
  stats: SessionStatsDTO;
}

export interface ListTeamsRequest {
}

export interface CreateUserRequestDTO {
  username: string;
  email: string;
  email_verified: boolean;
  metadata: any;
  name: string;
  password: string;
  role: string;
}

export interface ProviderDetailResponse {
  oidcClientID: string;
  samlEntryPoint: string;
  type: string;
  attributeMapping: any;
  createdAt: string;
  domain: string;
  oidcIssuer: string;
  oidcRedirectURI: string;
  providerId: string;
  samlIssuer: string;
  updatedAt: string;
  hasSamlCert: boolean;
}

export interface SendRequest {
  email: string;
}

export interface EmailConfig {
  enabled: boolean;
  provider: string;
  rate_limit: RateLimitConfig | undefined;
  template_id: string;
  code_expiry_minutes: number;
  code_length: number;
}

export interface AssignRoleRequest {
  roleID: string;
}

export interface VerifyRequest {
  code: string;
  email: string;
  phone: string;
  remember: boolean;
}

export interface RecoveryConfiguration {
}

export interface RiskContext {
}

export interface DeleteOrganizationInput {
  appId: string;
  orgId: string;
}

export interface ResendNotificationRequest {
}

export interface IDTokenClaims {
  name: string;
  nonce: string;
  preferred_username: string;
  session_state: string;
  auth_time: number;
  email: string;
  given_name: string;
  email_verified: boolean;
  family_name: string;
}

export interface SendVerificationCodeRequest {
  method: string;
  sessionId: string;
  target: string;
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

export interface FactorEnrollmentResponse {
  provisioningData: any;
  status: string;
  type: string;
  factorId: string;
}

export interface TwoFARepository {
  [key: string]: any;
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

export interface UpdateTeamInput {
  metadata: any;
  name: string;
  orgId: string;
  teamId: string;
  appId: string;
  description: string;
}

export interface KeyStats {
}

export interface InitiateChallengeRequest {
  context: string;
  factorTypes: string[];
  metadata: any;
}

export interface DeleteAppRequest {
}

export interface RevisionHandler {
}

export interface PolicyResponse {
  description: string;
  id: string;
  createdAt: string;
  updatedAt: string;
  userOrganizationId: string | undefined;
  version: number;
  appId: string;
  createdBy: string;
  enabled: boolean;
  expression: string;
  namespaceId: string;
  priority: number;
  resourceType: string;
  actions: string[];
  environmentId: string;
  name: string;
}

export interface RegisterProviderRequest {
  oidcClientSecret: string;
  oidcIssuer: string;
  providerId: string;
  oidcRedirectURI: string;
  samlCert: string;
  samlEntryPoint: string;
  samlIssuer: string;
  type: string;
  attributeMapping: any;
  domain: string;
  oidcClientID: string;
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

export interface MFAStatus {
  policyActive: boolean;
  requiredCount: number;
  trustedDevice: boolean;
  enabled: boolean;
  enrolledFactors: FactorInfo[];
  gracePeriod: string | undefined;
}

export interface GetOverviewStatsInput {
  days: number | undefined;
  endDate: string | undefined;
  startDate: string | undefined;
}

export interface RotateAPIKeyResponse {
  api_key: APIKey | undefined;
  message: string;
}

export interface PolicyEngine {
}

