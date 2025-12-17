package authsome

import (
	"time"

	"github.com/rs/xid"
)

// Auto-generated types

// Placeholder type aliases for undefined enum/custom types
type (
	RecoveryMethod       = string
	RecoveryStatus       = string
	ComplianceStandard   = string
	VerificationMethod   = string
	FactorPriority       = string
	FactorType           = string
	FactorStatus         = string
	RiskLevel            = string
	SecurityLevel        = string
	ChallengeStatus      = string
	JSONBMap             = map[string]interface{}
)

type (
	schema               schemaPlaceholder
	session              sessionPlaceholder
	user                 userPlaceholder
	providers            providersPlaceholder
	apikey               apikeyPlaceholder
	organization         organizationPlaceholder
)

// Placeholder structs for package-qualified types
type schemaPlaceholder struct {
	IdentityVerificationSession interface{}
	SocialAccount               interface{}
	IdentityVerification        interface{}
	UserVerificationStatus      interface{}
	User                        interface{}
}

type providersPlaceholder struct {
	EmailProvider interface{}
	OAuthProvider interface{}
	SAMLProvider  interface{}
	SMSProvider   interface{}
}

type sessionPlaceholder struct {
	Session interface{}
}

type userPlaceholder struct {
	User interface{}
}

type apikeyPlaceholder struct {
	APIKey interface{}
	Role   interface{}
}

type organizationPlaceholder struct {
	Team       interface{}
	Invitation interface{}
	Member     interface{}
}

type redisPlaceholder struct {
	Client interface{}
}

var redis = redisPlaceholder{}

// Placeholder types for undefined/missing types
type (
	Time                        = time.Time
	ID                          = xid.ID
	IdentityVerification        struct {}
	SocialAccount               struct {}
	IdentityVerificationSession struct {}
	NotificationType            = string
	Team                        struct {}
	APIKey                      struct {}
	Invitation                  struct {}
	UserVerificationStatus      = string
	Role                        struct {}
	ProviderConfig              struct {}
	Member                      struct {}
)

type WebhookPayload struct {
}

type UpdateRecoveryConfigResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type CompleteRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type VerifySecurityAnswersRequest struct {
	Answers interface{} `json:"answers"`
	SessionId xid.ID `json:"sessionId"`
}

type ComplianceViolationsResponse struct {
	Violations []*interface{} `json:"violations"`
}

type CreateConsentRequest struct {
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	ExpiresIn *int `json:"expiresIn"`
	Granted bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Purpose string `json:"purpose"`
	UserId string `json:"userId"`
}

type RejectRecoveryRequest struct {
	Notes string `json:"notes"`
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type SecurityQuestionInfo struct {
	IsCustom bool `json:"isCustom"`
	QuestionId int `json:"questionId"`
	QuestionText string `json:"questionText"`
	Id xid.ID `json:"id"`
}

type MigrateAllRequest struct {
	DryRun bool `json:"dryRun"`
	PreserveOriginal bool `json:"preserveOriginal"`
}

type CreateResourceRequest struct {
	NamespaceId string `json:"namespaceId"`
	Type string `json:"type"`
	Attributes []ResourceAttributeRequest `json:"attributes"`
	Description string `json:"description"`
}

type AuditLogEntry struct {
	Action string `json:"action"`
	AppId string `json:"appId"`
	NewValue interface{} `json:"newValue"`
	ResourceType string `json:"resourceType"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string `json:"userAgent"`
	UserOrganizationId *string `json:"userOrganizationId"`
	ActorId string `json:"actorId"`
	EnvironmentId string `json:"environmentId"`
	Id string `json:"id"`
	IpAddress string `json:"ipAddress"`
	OldValue interface{} `json:"oldValue"`
	ResourceId string `json:"resourceId"`
}

type PolicyResponse struct {
	Actions []string `json:"actions"`
	Expression string `json:"expression"`
	NamespaceId string `json:"namespaceId"`
	Description string `json:"description"`
	EnvironmentId string `json:"environmentId"`
	Version int `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string `json:"createdBy"`
	Enabled bool `json:"enabled"`
	Id string `json:"id"`
	Name string `json:"name"`
	ResourceType string `json:"resourceType"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserOrganizationId *string `json:"userOrganizationId"`
	AppId string `json:"appId"`
	Priority int `json:"priority"`
}

type Verify_body struct {
	Code string `json:"code"`
	Device_id string `json:"device_id"`
	Remember_device bool `json:"remember_device"`
	User_id string `json:"user_id"`
}

type VerificationsResponse struct {
	Count int `json:"count"`
	Verifications interface{} `json:"verifications"`
}

type RevokeAllResponse struct {
	RevokedCount int `json:"revokedCount"`
	Status string `json:"status"`
}

type ListFactorsRequest struct {
}

type IntrospectionService struct {
}

type ReverifyRequest struct {
	Reason string `json:"reason"`
}

type SecretsConfigSource struct {
}

type SchemaValidator struct {
}

type AddTeamMemberRequest struct {
	Member_id xid.ID `json:"member_id"`
	Role string `json:"role"`
}

type StartRecoveryResponse struct {
	SessionId xid.ID `json:"sessionId"`
	Status RecoveryStatus `json:"status"`
	AvailableMethods []RecoveryMethod `json:"availableMethods"`
	CompletedSteps int `json:"completedSteps"`
	ExpiresAt time.Time `json:"expiresAt"`
	RequiredSteps int `json:"requiredSteps"`
	RequiresReview bool `json:"requiresReview"`
	RiskScore float64 `json:"riskScore"`
}

type RequirementsResponse struct {
	Count int `json:"count"`
	Requirements interface{} `json:"requirements"`
}

type RouteRule struct {
	Description string `json:"description"`
	Method string `json:"method"`
	Org_id string `json:"org_id"`
	Pattern string `json:"pattern"`
	Security_level SecurityLevel `json:"security_level"`
}

type FactorsResponse struct {
	Count int `json:"count"`
	Factors interface{} `json:"factors"`
}

type ChallengeStatusResponse struct {
	FactorsRemaining int `json:"factorsRemaining"`
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
	SessionId xid.ID `json:"sessionId"`
	Status string `json:"status"`
	CompletedAt Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type GetSessionResponse struct {
	User User `json:"user"`
	Session Session `json:"session"`
}

type VideoSessionInfo struct {
}

type BackupAuthDocumentResponse struct {
	Id string `json:"id"`
}

type CreateProfileFromTemplate_req struct {
	Standard ComplianceStandard `json:"standard"`
}

type StartImpersonationRequest struct {
	Duration_minutes *int `json:"duration_minutes,omitempty"`
	Reason string `json:"reason"`
	Target_user_id string `json:"target_user_id"`
	Ticket_number *string `json:"ticket_number,omitempty"`
}

type TeamHandler struct {
}

type BatchEvaluateRequest struct {
	Requests []EvaluateRequest `json:"requests"`
}

type FacialCheckConfig struct {
	Variant string `json:"variant"`
	Enabled bool `json:"enabled"`
	MotionCapture bool `json:"motionCapture"`
}

// SuccessResponse represents Success boolean response
type SuccessResponse struct {
	Success bool `json:"success"`
}

type ConsentExpiryConfig struct {
	RenewalReminderDays int `json:"renewalReminderDays"`
	RequireReConsent bool `json:"requireReConsent"`
	AllowRenewal bool `json:"allowRenewal"`
	AutoExpireCheck bool `json:"autoExpireCheck"`
	DefaultValidityDays int `json:"defaultValidityDays"`
	Enabled bool `json:"enabled"`
	ExpireCheckInterval time.Duration `json:"expireCheckInterval"`
}

type ProvidersAppResponse struct {
	AppId string `json:"appId"`
	Providers []string `json:"providers"`
}

type RecoveryCodeUsage struct {
}

type FinishRegisterRequest struct {
	Name string `json:"name"`
	Response interface{} `json:"response"`
	UserId string `json:"userId"`
}

type MigrateRBACRequest struct {
	KeepRbacPolicies bool `json:"keepRbacPolicies"`
	NamespaceId string `json:"namespaceId"`
	ValidateEquivalence bool `json:"validateEquivalence"`
	DryRun bool `json:"dryRun"`
}

type PoliciesListResponse struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Policies []*PolicyResponse `json:"policies"`
	TotalCount int `json:"totalCount"`
}

type ConsentExportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type ConsentNotificationsConfig struct {
	NotifyDeletionApproved bool `json:"notifyDeletionApproved"`
	NotifyDeletionComplete bool `json:"notifyDeletionComplete"`
	NotifyDpoEmail string `json:"notifyDpoEmail"`
	NotifyExportReady bool `json:"notifyExportReady"`
	NotifyOnExpiry bool `json:"notifyOnExpiry"`
	NotifyOnRevoke bool `json:"notifyOnRevoke"`
	NotifyOnGrant bool `json:"notifyOnGrant"`
	Channels []string `json:"channels"`
	Enabled bool `json:"enabled"`
}

type ImpersonationSession struct {
}

type ImpersonateUserRequest struct {
	App_id xid.ID `json:"app_id"`
	Duration time.Duration `json:"duration"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
}

type EmailServiceAdapter struct {
}

type StepUpPolicyResponse struct {
	Id string `json:"id"`
}

type JumioProvider struct {
}

type BunRepository struct {
}

type AuditServiceAdapter struct {
}

type UpdateProvider_req struct {
	Config interface{} `json:"config"`
	IsActive bool `json:"isActive"`
	IsDefault bool `json:"isDefault"`
}

type TokenIntrospectionRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Error string `json:"error"`
}

type BackupAuthCodesResponse struct {
	Codes []string `json:"codes"`
}

type CompliancePolicy struct {
	Content string `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	ReviewDate time.Time `json:"reviewDate"`
	Standard ComplianceStandard `json:"standard"`
	Status string `json:"status"`
	Title string `json:"title"`
	ApprovedAt Time `json:"approvedAt"`
	EffectiveDate time.Time `json:"effectiveDate"`
	ProfileId string `json:"profileId"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version string `json:"version"`
	AppId string `json:"appId"`
	ApprovedBy string `json:"approvedBy"`
	PolicyType string `json:"policyType"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
}

type WebAuthnFactorAdapter struct {
}

type TwoFAStatusResponse struct {
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
	Enabled bool `json:"enabled"`
}

type ChannelsResponse struct {
	Channels interface{} `json:"channels"`
	Count int `json:"count"`
}

type TokenRequest struct {
	Scope string `json:"scope"`
	Audience string `json:"audience"`
	Refresh_token string `json:"refresh_token"`
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Code string `json:"code"`
	Code_verifier string `json:"code_verifier"`
	Grant_type string `json:"grant_type"`
	Redirect_uri string `json:"redirect_uri"`
}

type AMLMatch struct {
}

type PrivacySettings struct {
	ContactPhone string `json:"contactPhone"`
	DataRetentionDays int `json:"dataRetentionDays"`
	DpoEmail string `json:"dpoEmail"`
	OrganizationId string `json:"organizationId"`
	RequireExplicitConsent bool `json:"requireExplicitConsent"`
	UpdatedAt time.Time `json:"updatedAt"`
	AllowDataPortability bool `json:"allowDataPortability"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DataExportExpiryHours int `json:"dataExportExpiryHours"`
	ExportFormat []string `json:"exportFormat"`
	Id xid.ID `json:"id"`
	AutoDeleteAfterDays int `json:"autoDeleteAfterDays"`
	CcpaMode bool `json:"ccpaMode"`
	ConsentRequired bool `json:"consentRequired"`
	ContactEmail string `json:"contactEmail"`
	CreatedAt time.Time `json:"createdAt"`
	DeletionGracePeriodDays int `json:"deletionGracePeriodDays"`
	GdprMode bool `json:"gdprMode"`
	Metadata JSONBMap `json:"metadata"`
	CookieConsentEnabled bool `json:"cookieConsentEnabled"`
	RequireAdminApprovalForDeletion bool `json:"requireAdminApprovalForDeletion"`
	AnonymousConsentEnabled bool `json:"anonymousConsentEnabled"`
}

type RecoveryAttemptLog struct {
}

type PolicyPreviewResponse struct {
	Actions []string `json:"actions"`
	Description string `json:"description"`
	Expression string `json:"expression"`
	Name string `json:"name"`
	ResourceType string `json:"resourceType"`
}

type TwoFAStatusDetailResponse struct {
	Enabled bool `json:"enabled"`
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
}

type IDTokenClaims struct {
	Preferred_username string `json:"preferred_username"`
	Session_state string `json:"session_state"`
	Auth_time int64 `json:"auth_time"`
	Email string `json:"email"`
	Family_name string `json:"family_name"`
	Nonce string `json:"nonce"`
	Email_verified bool `json:"email_verified"`
	Given_name string `json:"given_name"`
	Name string `json:"name"`
}

type EncryptionConfig struct {
	MasterKey string `json:"masterKey"`
	RotateKeyAfter time.Duration `json:"rotateKeyAfter"`
	TestOnStartup bool `json:"testOnStartup"`
}

type RevokeDeviceRequest struct {
	DeviceId string `json:"deviceId"`
}

type CreateRequest struct {
	Events []string `json:"events"`
	Secret *string `json:"secret,omitempty"`
	Url string `json:"url"`
}

type DataExportConfig struct {
	AutoCleanup bool `json:"autoCleanup"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	Enabled bool `json:"enabled"`
	ExpiryHours int `json:"expiryHours"`
	IncludeSections []string `json:"includeSections"`
	MaxExportSize int64 `json:"maxExportSize"`
	AllowedFormats []string `json:"allowedFormats"`
	DefaultFormat string `json:"defaultFormat"`
	MaxRequests int `json:"maxRequests"`
	RequestPeriod time.Duration `json:"requestPeriod"`
	StoragePath string `json:"storagePath"`
}

type RemoveTrustedContactRequest struct {
	ContactId xid.ID `json:"contactId"`
}

type RejectRecoveryResponse struct {
	Rejected bool `json:"rejected"`
	RejectedAt time.Time `json:"rejectedAt"`
	SessionId xid.ID `json:"sessionId"`
	Message string `json:"message"`
	Reason string `json:"reason"`
}

type DocumentVerification struct {
}

type ComplianceStatusDetailsResponse struct {
	Status string `json:"status"`
}

type ListPasskeysRequest struct {
}

type RequestReverification_req struct {
	Reason string `json:"reason"`
}

type RotateAPIKeyResponse struct {
	Api_key APIKey `json:"api_key"`
	Message string `json:"message"`
}

type RetentionConfig struct {
	ArchivePath string `json:"archivePath"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	PurgeSchedule string `json:"purgeSchedule"`
	ArchiveBeforePurge bool `json:"archiveBeforePurge"`
}

type VerificationRequest struct {
	RememberDevice bool `json:"rememberDevice"`
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data interface{} `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
}

type CookieConsentConfig struct {
	ValidityPeriod time.Duration `json:"validityPeriod"`
	AllowAnonymous bool `json:"allowAnonymous"`
	BannerVersion string `json:"bannerVersion"`
	Categories []string `json:"categories"`
	DefaultStyle string `json:"defaultStyle"`
	Enabled bool `json:"enabled"`
	RequireExplicit bool `json:"requireExplicit"`
}

type StepUpVerificationResponse struct {
	Expires_at string `json:"expires_at"`
	Verified bool `json:"verified"`
}

type SMSProviderConfig struct {
	Provider string `json:"provider"`
	Config interface{} `json:"config"`
	From string `json:"from"`
}

type TestProvider_req struct {
	ProviderType string `json:"providerType"`
	TestRecipient string `json:"testRecipient"`
	ProviderName string `json:"providerName"`
}

type SetActiveRequest struct {
	Id string `json:"id"`
}

type PrivacySettingsRequest struct {
	ContactEmail string `json:"contactEmail"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	ContactPhone string `json:"contactPhone"`
	DpoEmail string `json:"dpoEmail"`
	CcpaMode *bool `json:"ccpaMode"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DataRetentionDays *int `json:"dataRetentionDays"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	ExportFormat []string `json:"exportFormat"`
	GdprMode *bool `json:"gdprMode"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	AllowDataPortability *bool `json:"allowDataPortability"`
	ConsentRequired *bool `json:"consentRequired"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
}

type WebhookConfig struct {
	Notify_on_rotated bool `json:"notify_on_rotated"`
	Webhook_urls []string `json:"webhook_urls"`
	Enabled bool `json:"enabled"`
	Expiry_warning_days int `json:"expiry_warning_days"`
	Notify_on_created bool `json:"notify_on_created"`
	Notify_on_deleted bool `json:"notify_on_deleted"`
	Notify_on_expiring bool `json:"notify_on_expiring"`
	Notify_on_rate_limit bool `json:"notify_on_rate_limit"`
}

type ListRecoverySessionsRequest struct {
	OrganizationId string `json:"organizationId"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	RequiresReview bool `json:"requiresReview"`
	Status RecoveryStatus `json:"status"`
}

type ListChecksFilter struct {
	SinceBefore Time `json:"sinceBefore"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
	CheckType *string `json:"checkType"`
	ProfileId *string `json:"profileId"`
}

type BeginRegisterResponse struct {
	Challenge string `json:"challenge"`
	Options interface{} `json:"options"`
	Timeout time.Duration `json:"timeout"`
	UserId string `json:"userId"`
}

type ListPasskeysResponse struct {
	Count int `json:"count"`
	Passkeys []PasskeyInfo `json:"passkeys"`
}

type RefreshResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type RequestReverificationRequest struct {
	Reason string `json:"reason"`
}

type UnbanUser_reqBody struct {
	Reason *string `json:"reason,omitempty"`
}

// StatusResponse represents Status response
type StatusResponse struct {
	Status string `json:"status"`
}

type SetupSecurityQuestionsResponse struct {
	Count int `json:"count"`
	Message string `json:"message"`
	SetupAt time.Time `json:"setupAt"`
}

type SendRequest struct {
	Email string `json:"email"`
}

type ComplianceProfile struct {
	LeastPrivilege bool `json:"leastPrivilege"`
	PasswordMinLength int `json:"passwordMinLength"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	MfaRequired bool `json:"mfaRequired"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	RetentionDays int `json:"retentionDays"`
	AppId string `json:"appId"`
	ComplianceContact string `json:"complianceContact"`
	DpoContact string `json:"dpoContact"`
	Name string `json:"name"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	Metadata interface{} `json:"metadata"`
	RbacRequired bool `json:"rbacRequired"`
	RegularAccessReview bool `json:"regularAccessReview"`
	Status string `json:"status"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	Id string `json:"id"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	SessionMaxAge int `json:"sessionMaxAge"`
	UpdatedAt time.Time `json:"updatedAt"`
	AuditLogExport bool `json:"auditLogExport"`
	CreatedAt time.Time `json:"createdAt"`
	DataResidency string `json:"dataResidency"`
	Standards []ComplianceStandard `json:"standards"`
}

type TestPolicyResponse struct {
	Error string `json:"error"`
	FailedCount int `json:"failedCount"`
	Passed bool `json:"passed"`
	PassedCount int `json:"passedCount"`
	Results []TestCaseResult `json:"results"`
	Total int `json:"total"`
}

type MFABypassResponse struct {
	UserId xid.ID `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	Reason string `json:"reason"`
}

type TemplateDefault struct {
}

type VerifyTrustedContactResponse struct {
	VerifiedAt time.Time `json:"verifiedAt"`
	ContactId xid.ID `json:"contactId"`
	Message string `json:"message"`
	Verified bool `json:"verified"`
}

type MockService struct {
}

type ResourcesListResponse struct {
	Resources []*ResourceResponse `json:"resources"`
	TotalCount int `json:"totalCount"`
}

type SaveNotificationSettings_req struct {
	AutoSendWelcome bool `json:"autoSendWelcome"`
	CleanupAfter string `json:"cleanupAfter"`
	RetryAttempts int `json:"retryAttempts"`
	RetryDelay string `json:"retryDelay"`
}

type NotificationTemplateResponse struct {
	Template interface{} `json:"template"`
}

type JWTService struct {
}

type GenerateBackupCodesRequest struct {
	Count int `json:"count"`
	User_id string `json:"user_id"`
}

type NotificationChannels struct {
	Email bool `json:"email"`
	Slack bool `json:"slack"`
	Webhook bool `json:"webhook"`
}

type NotificationErrorResponse struct {
	Error string `json:"error"`
}

type ScopeInfo struct {
}

type DeleteResponse struct {
	Success bool `json:"success"`
}

type Config struct {
	Audit ConsentAuditConfig `json:"audit"`
	Dashboard ConsentDashboardConfig `json:"dashboard"`
	Enabled bool `json:"enabled"`
	Notifications ConsentNotificationsConfig `json:"notifications"`
	CcpaEnabled bool `json:"ccpaEnabled"`
	CookieConsent CookieConsentConfig `json:"cookieConsent"`
	DataDeletion DataDeletionConfig `json:"dataDeletion"`
	DataExport DataExportConfig `json:"dataExport"`
	Expiry ConsentExpiryConfig `json:"expiry"`
	GdprEnabled bool `json:"gdprEnabled"`
}

type ConsentDashboardConfig struct {
	ShowCookiePreferences bool `json:"showCookiePreferences"`
	ShowDataDeletion bool `json:"showDataDeletion"`
	ShowDataExport bool `json:"showDataExport"`
	ShowPolicies bool `json:"showPolicies"`
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
	ShowAuditLog bool `json:"showAuditLog"`
	ShowConsentHistory bool `json:"showConsentHistory"`
}

type EvaluateResponse struct {
	Allowed bool `json:"allowed"`
	CacheHit bool `json:"cacheHit"`
	Error string `json:"error"`
	EvaluatedPolicies int `json:"evaluatedPolicies"`
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
	MatchedPolicies []string `json:"matchedPolicies"`
	Reason string `json:"reason"`
}

type Challenge struct {
	Attempts int `json:"attempts"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorId xid.ID `json:"factorId"`
	Id xid.ID `json:"id"`
	Metadata interface{} `json:"metadata"`
	UserAgent string `json:"userAgent"`
	IpAddress string `json:"ipAddress"`
	MaxAttempts int `json:"maxAttempts"`
	Status ChallengeStatus `json:"status"`
	Type FactorType `json:"type"`
	UserId xid.ID `json:"userId"`
	VerifiedAt Time `json:"verifiedAt"`
}

type ReviewDocumentResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type AdminUpdatePolicyRequest struct {
	AllowedTypes []string `json:"allowedTypes"`
	Enabled bool `json:"enabled"`
	GracePeriod int `json:"gracePeriod"`
	RequiredFactors int `json:"requiredFactors"`
}

type NotificationsConfig struct {
	NotifyOnRecoveryComplete bool `json:"notifyOnRecoveryComplete"`
	NotifyOnRecoveryFailed bool `json:"notifyOnRecoveryFailed"`
	NotifyOnRecoveryStart bool `json:"notifyOnRecoveryStart"`
	SecurityOfficerEmail string `json:"securityOfficerEmail"`
	Channels []string `json:"channels"`
	Enabled bool `json:"enabled"`
	NotifyAdminOnHighRisk bool `json:"notifyAdminOnHighRisk"`
	NotifyAdminOnReviewNeeded bool `json:"notifyAdminOnReviewNeeded"`
}

type RiskAssessmentConfig struct {
	BlockHighRisk bool `json:"blockHighRisk"`
	Enabled bool `json:"enabled"`
	HistoryWeight float64 `json:"historyWeight"`
	NewDeviceWeight float64 `json:"newDeviceWeight"`
	NewLocationWeight float64 `json:"newLocationWeight"`
	RequireReviewAbove float64 `json:"requireReviewAbove"`
	HighRiskThreshold float64 `json:"highRiskThreshold"`
	LowRiskThreshold float64 `json:"lowRiskThreshold"`
	MediumRiskThreshold float64 `json:"mediumRiskThreshold"`
	NewIpWeight float64 `json:"newIpWeight"`
	VelocityWeight float64 `json:"velocityWeight"`
}

type ValidatePolicyRequest struct {
	Expression string `json:"expression"`
	ResourceType string `json:"resourceType"`
}

type MFASession struct {
	UserId xid.ID `json:"userId"`
	VerifiedFactors ID `json:"verifiedFactors"`
	CompletedAt Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsVerified int `json:"factorsVerified"`
	Id xid.ID `json:"id"`
	SessionToken string `json:"sessionToken"`
	UserAgent string `json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
	FactorsRequired int `json:"factorsRequired"`
	IpAddress string `json:"ipAddress"`
	Metadata interface{} `json:"metadata"`
	RiskLevel RiskLevel `json:"riskLevel"`
}

type TwoFAEnableResponse struct {
	Status string `json:"status"`
	Totp_uri string `json:"totp_uri"`
}

type ContentTypeHandler struct {
}

type VerificationRepository struct {
}

type NotificationWebhookResponse struct {
	Status string `json:"status"`
}

type BackupAuthContactResponse struct {
	Id string `json:"id"`
}

type BackupAuthVideoResponse struct {
	Session_id string `json:"session_id"`
}

type FinishRegisterResponse struct {
	CredentialId string `json:"credentialId"`
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type RiskEngine struct {
}

type BaseFactorAdapter struct {
}

type ResetUserMFARequest struct {
	Reason string `json:"reason"`
}

type GenerateBackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type RegisterClientRequest struct {
	Application_type string `json:"application_type"`
	Client_name string `json:"client_name"`
	Grant_types []string `json:"grant_types"`
	Policy_uri string `json:"policy_uri"`
	Require_consent bool `json:"require_consent"`
	Trusted_client bool `json:"trusted_client"`
	Logo_uri string `json:"logo_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Tos_uri string `json:"tos_uri"`
	Contacts []string `json:"contacts"`
	Scope string `json:"scope"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Require_pkce bool `json:"require_pkce"`
	Response_types []string `json:"response_types"`
}

type CreateUser_reqBody struct {
	Metadata *interface{} `json:"metadata,omitempty"`
	Name *string `json:"name,omitempty"`
	Password *string `json:"password,omitempty"`
	Role *string `json:"role,omitempty"`
	Username *string `json:"username,omitempty"`
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
}

type NamespacesListResponse struct {
	Namespaces []*NamespaceResponse `json:"namespaces"`
	TotalCount int `json:"totalCount"`
}

type RiskAssessment struct {
	Score float64 `json:"score"`
	Factors []string `json:"factors"`
	Level RiskLevel `json:"level"`
	Metadata interface{} `json:"metadata"`
	Recommended []FactorType `json:"recommended"`
}

type Factor struct {
	LastUsedAt Time `json:"lastUsedAt"`
	Metadata interface{} `json:"metadata"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
	UserId xid.ID `json:"userId"`
	VerifiedAt Time `json:"verifiedAt"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
}

type NotificationPreviewResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

type UpdatePrivacySettingsRequest struct {
	CcpaMode *bool `json:"ccpaMode"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	GdprMode *bool `json:"gdprMode"`
	ConsentRequired *bool `json:"consentRequired"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
	AllowDataPortability *bool `json:"allowDataPortability"`
	ExportFormat []string `json:"exportFormat"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	DataRetentionDays *int `json:"dataRetentionDays"`
	DpoEmail string `json:"dpoEmail"`
}

type SAMLCallbackResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type DefaultProviderRegistry struct {
}

type NoOpVideoProvider struct {
}

type TokenResponse struct {
	Access_token string `json:"access_token"`
	Expires_in int `json:"expires_in"`
	Id_token string `json:"id_token"`
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
}

type ReviewDocumentRequest struct {
	Approved bool `json:"approved"`
	DocumentId xid.ID `json:"documentId"`
	Notes string `json:"notes"`
	RejectionReason string `json:"rejectionReason"`
}

type ScheduleVideoSessionRequest struct {
	ScheduledAt time.Time `json:"scheduledAt"`
	SessionId xid.ID `json:"sessionId"`
	TimeZone string `json:"timeZone"`
}

type OnfidoConfig struct {
	ApiToken string `json:"apiToken"`
	DocumentCheck DocumentCheckConfig `json:"documentCheck"`
	Enabled bool `json:"enabled"`
	FacialCheck FacialCheckConfig `json:"facialCheck"`
	IncludeDocumentReport bool `json:"includeDocumentReport"`
	Region string `json:"region"`
	IncludeFacialReport bool `json:"includeFacialReport"`
	IncludeWatchlistReport bool `json:"includeWatchlistReport"`
	WebhookToken string `json:"webhookToken"`
	WorkflowId string `json:"workflowId"`
}

type EndImpersonationRequest struct {
	Reason *string `json:"reason,omitempty"`
	Impersonation_id string `json:"impersonation_id"`
}

type AssignRole_reqBody struct {
	RoleID string `json:"roleID"`
}

type StartRecoveryRequest struct {
	Email string `json:"email"`
	PreferredMethod RecoveryMethod `json:"preferredMethod"`
	UserId string `json:"userId"`
	DeviceId string `json:"deviceId"`
}

type SessionTokenResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type GetPasskeyRequest struct {
}

type PhoneVerifyResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type StepUpAttempt struct {
	Ip string `json:"ip"`
	Method VerificationMethod `json:"method"`
	Success bool `json:"success"`
	Failure_reason string `json:"failure_reason"`
	Org_id string `json:"org_id"`
	Requirement_id string `json:"requirement_id"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
}

type EmailConfig struct {
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
	Code_expiry_minutes int `json:"code_expiry_minutes"`
}

type TrackNotificationEvent_req struct {
	EventData *interface{} `json:"eventData,omitempty"`
	NotificationId string `json:"notificationId"`
	OrganizationId *string `json:"organizationId,omitempty"`
	TemplateId string `json:"templateId"`
	Event string `json:"event"`
}

type ProviderDiscoveredResponse struct {
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
	Found bool `json:"found"`
}

type RateLimitingConfig struct {
	MaxAttemptsPerDay int `json:"maxAttemptsPerDay"`
	MaxAttemptsPerHour int `json:"maxAttemptsPerHour"`
	MaxAttemptsPerIp int `json:"maxAttemptsPerIp"`
	Enabled bool `json:"enabled"`
	ExponentialBackoff bool `json:"exponentialBackoff"`
	IpCooldownPeriod time.Duration `json:"ipCooldownPeriod"`
	LockoutAfterAttempts int `json:"lockoutAfterAttempts"`
	LockoutDuration time.Duration `json:"lockoutDuration"`
}

type SMSVerificationConfig struct {
	CodeExpiry time.Duration `json:"codeExpiry"`
	CodeLength int `json:"codeLength"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	Enabled bool `json:"enabled"`
	MaxAttempts int `json:"maxAttempts"`
	MaxSmsPerDay int `json:"maxSmsPerDay"`
	MessageTemplate string `json:"messageTemplate"`
	Provider string `json:"provider"`
}

type CreateAPIKeyRequest struct {
	Permissions *interface{} `json:"permissions,omitempty"`
	Rate_limit *int `json:"rate_limit,omitempty"`
	Scopes []string `json:"scopes"`
	Allowed_ips *[]string `json:"allowed_ips,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata *interface{} `json:"metadata,omitempty"`
	Name string `json:"name"`
}

type UpdateConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
}

type AddMember_req struct {
	Role string `json:"role"`
	User_id string `json:"user_id"`
}

type ContinueRecoveryResponse struct {
	Instructions string `json:"instructions"`
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
	TotalSteps int `json:"totalSteps"`
	CurrentStep int `json:"currentStep"`
	Data interface{} `json:"data"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type ComplianceCheck struct {
	Id string `json:"id"`
	NextCheckAt time.Time `json:"nextCheckAt"`
	ProfileId string `json:"profileId"`
	CreatedAt time.Time `json:"createdAt"`
	Evidence []string `json:"evidence"`
	LastCheckedAt time.Time `json:"lastCheckedAt"`
	Result interface{} `json:"result"`
	Status string `json:"status"`
	AppId string `json:"appId"`
	CheckType string `json:"checkType"`
}

type CreatePolicyRequest struct {
	ConsentType string `json:"consentType"`
	Content string `json:"content"`
	Metadata interface{} `json:"metadata"`
	Required bool `json:"required"`
	Version string `json:"version"`
	Description string `json:"description"`
	Name string `json:"name"`
	Renewable bool `json:"renewable"`
	ValidityPeriod *int `json:"validityPeriod"`
}

type BanUser_reqBody struct {
	Expires_at Time `json:"expires_at,omitempty"`
	Reason string `json:"reason"`
}

type StepUpEvaluationResponse struct {
	Reason string `json:"reason"`
	Required bool `json:"required"`
}

type OrganizationUIRegistry struct {
}

type MigrationStatusResponse struct {
	ValidationPassed bool `json:"validationPassed"`
	FailedCount int `json:"failedCount"`
	MigratedCount int `json:"migratedCount"`
	Progress float64 `json:"progress"`
	Status string `json:"status"`
	TotalPolicies int `json:"totalPolicies"`
	AppId string `json:"appId"`
	CompletedAt Time `json:"completedAt"`
	EnvironmentId string `json:"environmentId"`
	Errors []string `json:"errors"`
	StartedAt time.Time `json:"startedAt"`
	UserOrganizationId *string `json:"userOrganizationId"`
}

type EnableRequest struct {
}

type ConsentManager struct {
}

type mockUserService struct {
}

type KeyStats struct {
}

type UpdateRecoveryConfigRequest struct {
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
}

type MockAuditService struct {
}

type MockUserService struct {
}

type MockAppService struct {
}

type ComplianceTemplateResponse struct {
	Standard string `json:"standard"`
}

type ImpersonationContext struct {
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id ID `json:"target_user_id"`
	Impersonation_id ID `json:"impersonation_id"`
	Impersonator_id ID `json:"impersonator_id"`
	Indicator_message string `json:"indicator_message"`
}

type DocumentVerificationResult struct {
}

type ListPoliciesFilter struct {
	AppId *string `json:"appId"`
	PolicyType *string `json:"policyType"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
}

type ResolveViolationRequest struct {
	Notes string `json:"notes"`
	Resolution string `json:"resolution"`
}

type CreatePolicyResponse struct {
	Description string `json:"description"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Org_id string `json:"org_id"`
	Rules interface{} `json:"rules"`
	Enabled bool `json:"enabled"`
	Name string `json:"name"`
	Priority int `json:"priority"`
	Updated_at time.Time `json:"updated_at"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
}

type UnbanUserRequest struct {
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
}

type Email struct {
}

type MFAPolicy struct {
	RequiredFactorCount int `json:"requiredFactorCount"`
	StepUpRequired bool `json:"stepUpRequired"`
	AdaptiveMfaEnabled bool `json:"adaptiveMfaEnabled"`
	MaxFailedAttempts int `json:"maxFailedAttempts"`
	OrganizationId xid.ID `json:"organizationId"`
	RequiredFactorTypes []FactorType `json:"requiredFactorTypes"`
	TrustedDeviceDays int `json:"trustedDeviceDays"`
	UpdatedAt time.Time `json:"updatedAt"`
	AllowedFactorTypes []FactorType `json:"allowedFactorTypes"`
	CreatedAt time.Time `json:"createdAt"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
	LockoutDurationMinutes int `json:"lockoutDurationMinutes"`
}

// MessageResponse represents Simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

type AutomatedChecksConfig struct {
	DataRetention bool `json:"dataRetention"`
	Enabled bool `json:"enabled"`
	MfaCoverage bool `json:"mfaCoverage"`
	CheckInterval time.Duration `json:"checkInterval"`
	InactiveUsers bool `json:"inactiveUsers"`
	PasswordPolicy bool `json:"passwordPolicy"`
	SessionPolicy bool `json:"sessionPolicy"`
	SuspiciousActivity bool `json:"suspiciousActivity"`
	AccessReview bool `json:"accessReview"`
}

type WebAuthnWrapper struct {
}

type UserVerificationStatusResponse struct {
	Status UserVerificationStatus `json:"status"`
}

type Handler struct {
}

type ListSessionsRequest struct {
	App_id xid.ID `json:"app_id"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	User_id ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
}

type RevokeConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
}

type UpdateResponse struct {
	Webhook Webhook `json:"webhook"`
}

type AddTrustedContactResponse struct {
	AddedAt time.Time `json:"addedAt"`
	ContactId xid.ID `json:"contactId"`
	Email string `json:"email"`
	Message string `json:"message"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Verified bool `json:"verified"`
}

type CreateVerificationSession_req struct {
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
}

type ComplianceEvidence struct {
	AppId string `json:"appId"`
	ControlId string `json:"controlId"`
	CreatedAt time.Time `json:"createdAt"`
	EvidenceType string `json:"evidenceType"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	CollectedBy string `json:"collectedBy"`
	Description string `json:"description"`
	FileHash string `json:"fileHash"`
	FileUrl string `json:"fileUrl"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Title string `json:"title"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type OAuthErrorResponse struct {
	Error_uri string `json:"error_uri"`
	State string `json:"state"`
	Error string `json:"error"`
	Error_description string `json:"error_description"`
}

type ProvidersConfig struct {
	Gitlab ProviderConfig `json:"gitlab"`
	Reddit ProviderConfig `json:"reddit"`
	Slack ProviderConfig `json:"slack"`
	Twitch ProviderConfig `json:"twitch"`
	Twitter ProviderConfig `json:"twitter"`
	Facebook ProviderConfig `json:"facebook"`
	Google ProviderConfig `json:"google"`
	Line ProviderConfig `json:"line"`
	Apple ProviderConfig `json:"apple"`
	Discord ProviderConfig `json:"discord"`
	Dropbox ProviderConfig `json:"dropbox"`
	Github ProviderConfig `json:"github"`
	Spotify ProviderConfig `json:"spotify"`
	Linkedin ProviderConfig `json:"linkedin"`
	Microsoft ProviderConfig `json:"microsoft"`
	Notion ProviderConfig `json:"notion"`
	Bitbucket ProviderConfig `json:"bitbucket"`
}

type VideoVerificationConfig struct {
	Enabled bool `json:"enabled"`
	LivenessThreshold float64 `json:"livenessThreshold"`
	MinScheduleAdvance time.Duration `json:"minScheduleAdvance"`
	RequireLivenessCheck bool `json:"requireLivenessCheck"`
	RequireScheduling bool `json:"requireScheduling"`
	Provider string `json:"provider"`
	RecordSessions bool `json:"recordSessions"`
	RecordingRetention time.Duration `json:"recordingRetention"`
	RequireAdminReview bool `json:"requireAdminReview"`
	SessionDuration time.Duration `json:"sessionDuration"`
}

type EvaluateRequest struct {
	Amount float64 `json:"amount"`
	Currency string `json:"currency"`
	Metadata interface{} `json:"metadata"`
	Method string `json:"method"`
	Resource_type string `json:"resource_type"`
	Route string `json:"route"`
	Action string `json:"action"`
}

type CallbackDataResponse struct {
	Action string `json:"action"`
	IsNewUser bool `json:"isNewUser"`
	User User `json:"user"`
}

type UpdatePolicy_req struct {
	Title *string `json:"title"`
	Version *string `json:"version"`
	Content *string `json:"content"`
	Status *string `json:"status"`
}

type CompleteTraining_req struct {
	Score int `json:"score"`
}

type StepUpRequirement struct {
	Amount float64 `json:"amount"`
	Challenge_token string `json:"challenge_token"`
	Expires_at time.Time `json:"expires_at"`
	Ip string `json:"ip"`
	Reason string `json:"reason"`
	Required_level SecurityLevel `json:"required_level"`
	Risk_score float64 `json:"risk_score"`
	Status string `json:"status"`
	Currency string `json:"currency"`
	Current_level SecurityLevel `json:"current_level"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Route string `json:"route"`
	Rule_name string `json:"rule_name"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Fulfilled_at Time `json:"fulfilled_at"`
	Org_id string `json:"org_id"`
	Resource_action string `json:"resource_action"`
	Session_id string `json:"session_id"`
	Method string `json:"method"`
	Resource_type string `json:"resource_type"`
}

type MultiSessionSetActiveResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type BackupCodesConfig struct {
	Allow_reuse bool `json:"allow_reuse"`
	Count int `json:"count"`
	Enabled bool `json:"enabled"`
	Format string `json:"format"`
	Length int `json:"length"`
}

type NotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type ConfigSourceConfig struct {
	AutoRefresh bool `json:"autoRefresh"`
	Enabled bool `json:"enabled"`
	Prefix string `json:"prefix"`
	Priority int `json:"priority"`
	RefreshInterval time.Duration `json:"refreshInterval"`
}

type ImpersonationMiddleware struct {
}

type NoOpSMSProvider struct {
}

type ResendRequest struct {
	Email string `json:"email"`
}

type UpdatePolicyResponse struct {
	Updated_at time.Time `json:"updated_at"`
	User_id string `json:"user_id"`
	Description string `json:"description"`
	Id string `json:"id"`
	Name string `json:"name"`
	Rules interface{} `json:"rules"`
	Created_at time.Time `json:"created_at"`
	Enabled bool `json:"enabled"`
	Metadata interface{} `json:"metadata"`
	Org_id string `json:"org_id"`
	Priority int `json:"priority"`
}

type UpdateRequest struct {
	Url *string `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Enabled *bool `json:"enabled,omitempty"`
	Id string `json:"id"`
}

type CreateVerificationSessionResponse struct {
	Session IdentityVerificationSession `json:"session"`
}

type GetUserVerificationStatusResponse struct {
	Status UserVerificationStatus `json:"status"`
}

type InstantiateTemplateRequest struct {
	NamespaceId string `json:"namespaceId"`
	Parameters interface{} `json:"parameters"`
	Priority int `json:"priority"`
	ResourceType string `json:"resourceType"`
	Actions []string `json:"actions"`
	Description string `json:"description"`
	Enabled bool `json:"enabled"`
	Name string `json:"name"`
}

type UpdateNamespaceRequest struct {
	Description string `json:"description"`
	InheritPlatform *bool `json:"inheritPlatform"`
	Name string `json:"name"`
}

type EmailFactorAdapter struct {
}

type CheckSubResult struct {
}

type ConsentAuditConfig struct {
	ExportFormat string `json:"exportFormat"`
	LogAllChanges bool `json:"logAllChanges"`
	LogUserAgent bool `json:"logUserAgent"`
	RetentionDays int `json:"retentionDays"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	Enabled bool `json:"enabled"`
	Immutable bool `json:"immutable"`
	LogIpAddress bool `json:"logIpAddress"`
	SignLogs bool `json:"signLogs"`
}

type SignUpResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

type ListFactorsResponse struct {
	Count int `json:"count"`
	Factors []Factor `json:"factors"`
}

type TwoFABackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type ImpersonationVerifyResponse struct {
	Target_user_id string `json:"target_user_id"`
	Impersonator_id string `json:"impersonator_id"`
	Is_impersonating bool `json:"is_impersonating"`
}

type Middleware struct {
}

type RateLimitRule struct {
	Max int `json:"max"`
	Window time.Duration `json:"window"`
}

type ListTrainingFilter struct {
	UserId *string `json:"userId"`
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	TrainingType *string `json:"trainingType"`
}

type ResourceRule struct {
	Security_level SecurityLevel `json:"security_level"`
	Sensitivity string `json:"sensitivity"`
	Action string `json:"action"`
	Description string `json:"description"`
	Org_id string `json:"org_id"`
	Resource_type string `json:"resource_type"`
}

type TrustedDevice struct {
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	LastUsedAt Time `json:"lastUsedAt"`
	CreatedAt time.Time `json:"createdAt"`
	DeviceId string `json:"deviceId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
}

type TwoFASendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type CreateConsentPolicyRequest struct {
	ConsentType string `json:"consentType"`
	Description string `json:"description"`
	Name string `json:"name"`
	Renewable bool `json:"renewable"`
	Required bool `json:"required"`
	ValidityPeriod *int `json:"validityPeriod"`
	Content string `json:"content"`
	Metadata interface{} `json:"metadata"`
	Version string `json:"version"`
}

type DocumentVerificationConfig struct {
	AcceptedDocuments []string `json:"acceptedDocuments"`
	EncryptionKey string `json:"encryptionKey"`
	MinConfidenceScore float64 `json:"minConfidenceScore"`
	Provider string `json:"provider"`
	RequireBothSides bool `json:"requireBothSides"`
	RequireManualReview bool `json:"requireManualReview"`
	RequireSelfie bool `json:"requireSelfie"`
	StoragePath string `json:"storagePath"`
	Enabled bool `json:"enabled"`
	EncryptAtRest bool `json:"encryptAtRest"`
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	StorageProvider string `json:"storageProvider"`
}

type GetDocumentVerificationRequest struct {
	DocumentId xid.ID `json:"documentId"`
}

type CreateUserRequest struct {
	Name string `json:"name"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Email_verified bool `json:"email_verified"`
	Password string `json:"password"`
	Role string `json:"role"`
	Username string `json:"username"`
	Email string `json:"email"`
	Metadata interface{} `json:"metadata"`
}

type BeginLoginResponse struct {
	Challenge string `json:"challenge"`
	Options interface{} `json:"options"`
	Timeout time.Duration `json:"timeout"`
}

type BatchEvaluationResult struct {
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
	Index int `json:"index"`
	Policies []string `json:"policies"`
	ResourceId string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Action string `json:"action"`
	Allowed bool `json:"allowed"`
	Error string `json:"error"`
}

type RevokeTrustedDeviceRequest struct {
}

type ComplianceStatusResponse struct {
	Status string `json:"status"`
}

type LoginResponse struct {
	PasskeyUsed string `json:"passkeyUsed"`
	Session interface{} `json:"session"`
	Token string `json:"token"`
	User interface{} `json:"user"`
}

type MigrationResponse struct {
	Message string `json:"message"`
	MigrationId string `json:"migrationId"`
	StartedAt time.Time `json:"startedAt"`
	Status string `json:"status"`
}

type AnalyticsResponse struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Summary AnalyticsSummary `json:"summary"`
	TimeRange interface{} `json:"timeRange"`
}

type SendCodeRequest struct {
	Phone string `json:"phone"`
}

type ConsentSummary struct {
	HasPendingExport bool `json:"hasPendingExport"`
	PendingRenewals int `json:"pendingRenewals"`
	UserId string `json:"userId"`
	ConsentsByType interface{} `json:"consentsByType"`
	ExpiredConsents int `json:"expiredConsents"`
	LastConsentUpdate Time `json:"lastConsentUpdate"`
	OrganizationId string `json:"organizationId"`
	RevokedConsents int `json:"revokedConsents"`
	TotalConsents int `json:"totalConsents"`
	GrantedConsents int `json:"grantedConsents"`
	HasPendingDeletion bool `json:"hasPendingDeletion"`
}

type GenerateRecoveryCodesResponse struct {
	Codes []string `json:"codes"`
	Count int `json:"count"`
	GeneratedAt time.Time `json:"generatedAt"`
	Warning string `json:"warning"`
}

type TimeBasedRule struct {
	Description string `json:"description"`
	Max_age time.Duration `json:"max_age"`
	Operation string `json:"operation"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type MultiSessionErrorResponse struct {
	Error string `json:"error"`
}

type SendCodeResponse struct {
	Dev_code string `json:"dev_code"`
	Status string `json:"status"`
}

type ResetUserMFAResponse struct {
	DevicesRevoked int `json:"devicesRevoked"`
	FactorsReset int `json:"factorsReset"`
	Message string `json:"message"`
	Success bool `json:"success"`
}

type SAMLSPMetadataResponse struct {
	Metadata string `json:"metadata"`
}

type ChallengeSession struct {
}

type TrustedDevicesConfig struct {
	Default_expiry_days int `json:"default_expiry_days"`
	Enabled bool `json:"enabled"`
	Max_devices_per_user int `json:"max_devices_per_user"`
	Max_expiry_days int `json:"max_expiry_days"`
}

type UnblockUserRequest struct {
}

type ConsentPolicyResponse struct {
	Id string `json:"id"`
}

type ListRecoverySessionsResponse struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Sessions []RecoverySessionInfo `json:"sessions"`
	TotalCount int `json:"totalCount"`
}

type PasskeyInfo struct {
	CredentialId string `json:"credentialId"`
	Id string `json:"id"`
	IsResidentKey bool `json:"isResidentKey"`
	Name string `json:"name"`
	Aaguid string `json:"aaguid"`
	AuthenticatorType string `json:"authenticatorType"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsedAt Time `json:"lastUsedAt"`
	SignCount uint `json:"signCount"`
}

type ActionsListResponse struct {
	Actions []*ActionResponse `json:"actions"`
	TotalCount int `json:"totalCount"`
}

type RenderTemplate_req struct {
	Template string `json:"template"`
	Variables interface{} `json:"variables"`
}

type NotificationListResponse struct {
	Notifications []*interface{} `json:"notifications"`
	Total int `json:"total"`
}

type MemberHandler struct {
}

type AppServiceAdapter struct {
}

type AnalyticsSummary struct {
	TotalEvaluations int64 `json:"totalEvaluations"`
	CacheHitRate float64 `json:"cacheHitRate"`
	TopPolicies []PolicyStats `json:"topPolicies"`
	TotalPolicies int `json:"totalPolicies"`
	ActivePolicies int `json:"activePolicies"`
	AllowedCount int64 `json:"allowedCount"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	DeniedCount int64 `json:"deniedCount"`
	TopResourceTypes []ResourceTypeStats `json:"topResourceTypes"`
}

type Disable_body struct {
	User_id string `json:"user_id"`
}

type NoOpDocumentProvider struct {
}

type ContinueRecoveryRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
}

type GenerateReportRequest struct {
	Format string `json:"format"`
	Period string `json:"period"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
}

type AuditEvent struct {
}

type StepUpAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type TemplatesListResponse struct {
	Categories []string `json:"categories"`
	Templates []*TemplateResponse `json:"templates"`
	TotalCount int `json:"totalCount"`
}

type ResendResponse struct {
	Status string `json:"status"`
}

type ClientRegistrationResponse struct {
	Client_id string `json:"client_id"`
	Client_id_issued_at int64 `json:"client_id_issued_at"`
	Client_secret_expires_at int64 `json:"client_secret_expires_at"`
	Scope string `json:"scope"`
	Tos_uri string `json:"tos_uri"`
	Logo_uri string `json:"logo_uri"`
	Response_types []string `json:"response_types"`
	Application_type string `json:"application_type"`
	Policy_uri string `json:"policy_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Client_name string `json:"client_name"`
	Client_secret string `json:"client_secret"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
}

type DataDeletionConfig struct {
	RequireAdminApproval bool `json:"requireAdminApproval"`
	RetentionExemptions []string `json:"retentionExemptions"`
	AutoProcessAfterGrace bool `json:"autoProcessAfterGrace"`
	NotifyBeforeDeletion bool `json:"notifyBeforeDeletion"`
	AllowPartialDeletion bool `json:"allowPartialDeletion"`
	ArchiveBeforeDeletion bool `json:"archiveBeforeDeletion"`
	ArchivePath string `json:"archivePath"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	PreserveLegalData bool `json:"preserveLegalData"`
}

type ConsentStats struct {
	Type string `json:"type"`
	AverageLifetime int `json:"averageLifetime"`
	ExpiredCount int `json:"expiredCount"`
	GrantRate float64 `json:"grantRate"`
	GrantedCount int `json:"grantedCount"`
	RevokedCount int `json:"revokedCount"`
	TotalConsents int `json:"totalConsents"`
}

type UploadDocumentResponse struct {
	Message string `json:"message"`
	ProcessingTime string `json:"processingTime"`
	Status string `json:"status"`
	UploadedAt time.Time `json:"uploadedAt"`
	DocumentId xid.ID `json:"documentId"`
}

type TwoFAErrorResponse struct {
	Error string `json:"error"`
}

type SendWithTemplateRequest struct {
	Variables interface{} `json:"variables"`
	AppId xid.ID `json:"appId"`
	Language string `json:"language"`
	Metadata interface{} `json:"metadata"`
	Recipient string `json:"recipient"`
	TemplateKey string `json:"templateKey"`
	Type NotificationType `json:"type"`
}

type JWK struct {
	E string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N string `json:"n"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

type ProviderDetailResponse struct {
	HasSamlCert bool `json:"hasSamlCert"`
	OidcClientID string `json:"oidcClientID"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
	SamlIssuer string `json:"samlIssuer"`
	Domain string `json:"domain"`
	OidcIssuer string `json:"oidcIssuer"`
	ProviderId string `json:"providerId"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	Type string `json:"type"`
	UpdatedAt string `json:"updatedAt"`
	AttributeMapping interface{} `json:"attributeMapping"`
	CreatedAt string `json:"createdAt"`
}

type ProviderSession struct {
}

type AddTeamMember_req struct {
	Role string `json:"role"`
	Member_id xid.ID `json:"member_id"`
}

type AutoCleanupConfig struct {
	Enabled bool `json:"enabled"`
	Interval time.Duration `json:"interval"`
}

type EvaluationContext struct {
}

type RiskFactor struct {
}

type RegistrationService struct {
}

type SignOutResponse struct {
	Success bool `json:"success"`
}

type RenderTemplateRequest struct {
	Template string `json:"template"`
	Variables interface{} `json:"variables"`
}

type RecoverySessionInfo struct {
	UserEmail string `json:"userEmail"`
	CompletedAt Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	Method RecoveryMethod `json:"method"`
	RequiresReview bool `json:"requiresReview"`
	Status RecoveryStatus `json:"status"`
	TotalSteps int `json:"totalSteps"`
	UserId xid.ID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	CurrentStep int `json:"currentStep"`
	RiskScore float64 `json:"riskScore"`
}

type TOTPFactorAdapter struct {
}

type AdminPolicyRequest struct {
	RequiredFactors int `json:"requiredFactors"`
	AllowedTypes []string `json:"allowedTypes"`
	Enabled bool `json:"enabled"`
	GracePeriod int `json:"gracePeriod"`
}

type TemplateService struct {
}

type ListProvidersResponse struct {
	Providers []string `json:"providers"`
}

type ListRememberedDevicesResponse struct {
	Count int `json:"count"`
	Devices interface{} `json:"devices"`
}

type UpdateUserRequest struct {
	Name *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

type NoOpEmailProvider struct {
}

type MultiSessionListResponse struct {
	Sessions []*interface{} `json:"sessions"`
}

type TestPolicyRequest struct {
	Actions []string `json:"actions"`
	Expression string `json:"expression"`
	ResourceType string `json:"resourceType"`
	TestCases []TestCase `json:"testCases"`
}

type WebAuthnConfig struct {
	Timeout int `json:"timeout"`
	Attestation_preference string `json:"attestation_preference"`
	Authenticator_selection interface{} `json:"authenticator_selection"`
	Enabled bool `json:"enabled"`
	Rp_display_name string `json:"rp_display_name"`
	Rp_id string `json:"rp_id"`
	Rp_origins []string `json:"rp_origins"`
}

type CreateTemplateVersion_req struct {
	Changes string `json:"changes"`
}

type ProviderCheckResult struct {
}

type SendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type DeleteFactorRequest struct {
}

type ConsentReport struct {
	ConsentRate float64 `json:"consentRate"`
	DataExportsThisPeriod int `json:"dataExportsThisPeriod"`
	DpasActive int `json:"dpasActive"`
	DpasExpiringSoon int `json:"dpasExpiringSoon"`
	TotalUsers int `json:"totalUsers"`
	UsersWithConsent int `json:"usersWithConsent"`
	CompletedDeletions int `json:"completedDeletions"`
	ConsentsByType interface{} `json:"consentsByType"`
	OrganizationId string `json:"organizationId"`
	PendingDeletions int `json:"pendingDeletions"`
	ReportPeriodEnd time.Time `json:"reportPeriodEnd"`
	ReportPeriodStart time.Time `json:"reportPeriodStart"`
}

type ImpersonationErrorResponse struct {
	Error string `json:"error"`
}

type DeletePasskeyRequest struct {
}

type RevokeDeviceResponse struct {
	Success bool `json:"success"`
}

type SecurityQuestion struct {
}

type StepUpRememberedDevice struct {
	Ip string `json:"ip"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Device_id string `json:"device_id"`
	Expires_at time.Time `json:"expires_at"`
	Id string `json:"id"`
	Last_used_at time.Time `json:"last_used_at"`
	Org_id string `json:"org_id"`
	Remembered_at time.Time `json:"remembered_at"`
	Security_level SecurityLevel `json:"security_level"`
	Device_name string `json:"device_name"`
}

type FactorVerificationRequest struct {
	Data interface{} `json:"data"`
	FactorId xid.ID `json:"factorId"`
	Code string `json:"code"`
}

type Status_body struct {
	Device_id string `json:"device_id"`
	User_id string `json:"user_id"`
}

type AdminHandler struct {
}

type RequestDataDeletionRequest struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type GetCurrentResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type ConsentPolicy struct {
	OrganizationId string `json:"organizationId"`
	Renewable bool `json:"renewable"`
	UpdatedAt time.Time `json:"updatedAt"`
	Content string `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	Name string `json:"name"`
	PublishedAt Time `json:"publishedAt"`
	Active bool `json:"active"`
	Metadata JSONBMap `json:"metadata"`
	Required bool `json:"required"`
	ValidityPeriod *int `json:"validityPeriod"`
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	CreatedBy string `json:"createdBy"`
	Description string `json:"description"`
}

type SendOTP_body struct {
	User_id string `json:"user_id"`
}

type SetActiveResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type ConsentReportResponse struct {
	Id string `json:"id"`
}

type UpdatePolicyRequest struct {
	Required *bool `json:"required"`
	ValidityPeriod *int `json:"validityPeriod"`
	Active *bool `json:"active"`
	Content string `json:"content"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Renewable *bool `json:"renewable"`
}

type BackupAuthStatusResponse struct {
	Status string `json:"status"`
}

type AddCustomPermission_req struct {
	Category string `json:"category"`
	Description string `json:"description"`
	Name string `json:"name"`
}

type UpdatePasskeyResponse struct {
	UpdatedAt time.Time `json:"updatedAt"`
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
}

type DiscoveryResponse struct {
	Claims_parameter_supported bool `json:"claims_parameter_supported"`
	Id_token_signing_alg_values_supported []string `json:"id_token_signing_alg_values_supported"`
	Introspection_endpoint_auth_methods_supported []string `json:"introspection_endpoint_auth_methods_supported"`
	Jwks_uri string `json:"jwks_uri"`
	Registration_endpoint string `json:"registration_endpoint"`
	Request_uri_parameter_supported bool `json:"request_uri_parameter_supported"`
	Authorization_endpoint string `json:"authorization_endpoint"`
	Claims_supported []string `json:"claims_supported"`
	Introspection_endpoint string `json:"introspection_endpoint"`
	Subject_types_supported []string `json:"subject_types_supported"`
	Token_endpoint string `json:"token_endpoint"`
	Token_endpoint_auth_methods_supported []string `json:"token_endpoint_auth_methods_supported"`
	Userinfo_endpoint string `json:"userinfo_endpoint"`
	Code_challenge_methods_supported []string `json:"code_challenge_methods_supported"`
	Grant_types_supported []string `json:"grant_types_supported"`
	Issuer string `json:"issuer"`
	Require_request_uri_registration bool `json:"require_request_uri_registration"`
	Response_modes_supported []string `json:"response_modes_supported"`
	Revocation_endpoint string `json:"revocation_endpoint"`
	Scopes_supported []string `json:"scopes_supported"`
	Request_parameter_supported bool `json:"request_parameter_supported"`
	Response_types_supported []string `json:"response_types_supported"`
	Revocation_endpoint_auth_methods_supported []string `json:"revocation_endpoint_auth_methods_supported"`
}

type KeyPair struct {
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type CreateProfileRequest struct {
	LeastPrivilege bool `json:"leastPrivilege"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	RbacRequired bool `json:"rbacRequired"`
	RegularAccessReview bool `json:"regularAccessReview"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	ComplianceContact string `json:"complianceContact"`
	DataResidency string `json:"dataResidency"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	Standards []ComplianceStandard `json:"standards"`
	PasswordMinLength int `json:"passwordMinLength"`
	AuditLogExport bool `json:"auditLogExport"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	DpoContact string `json:"dpoContact"`
	Metadata interface{} `json:"metadata"`
	MfaRequired bool `json:"mfaRequired"`
	Name string `json:"name"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	AppId string `json:"appId"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	RetentionDays int `json:"retentionDays"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	SessionMaxAge int `json:"sessionMaxAge"`
}

type VerificationSessionResponse struct {
	Session IdentityVerificationSession `json:"session"`
}

type BulkPublishRequest struct {
	Ids []string `json:"ids"`
}

type UpdateClientRequest struct {
	Allowed_scopes []string `json:"allowed_scopes"`
	Grant_types []string `json:"grant_types"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
	Require_consent *bool `json:"require_consent"`
	Response_types []string `json:"response_types"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Contacts []string `json:"contacts"`
	Logo_uri string `json:"logo_uri"`
	Name string `json:"name"`
	Require_pkce *bool `json:"require_pkce"`
	Tos_uri string `json:"tos_uri"`
	Trusted_client *bool `json:"trusted_client"`
}

type DataProcessingAgreement struct {
	SignedByTitle string `json:"signedByTitle"`
	AgreementType string `json:"agreementType"`
	Content string `json:"content"`
	ExpiryDate Time `json:"expiryDate"`
	Id xid.ID `json:"id"`
	Metadata JSONBMap `json:"metadata"`
	OrganizationId string `json:"organizationId"`
	UpdatedAt time.Time `json:"updatedAt"`
	EffectiveDate time.Time `json:"effectiveDate"`
	IpAddress string `json:"ipAddress"`
	Version string `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	SignedBy string `json:"signedBy"`
	SignedByName string `json:"signedByName"`
	Status string `json:"status"`
	DigitalSignature string `json:"digitalSignature"`
	SignedByEmail string `json:"signedByEmail"`
}

type DashboardExtension struct {
}

type TrustedContact struct {
}

type TwoFARequiredResponse struct {
	Device_id string `json:"device_id"`
	Require_twofa bool `json:"require_twofa"`
	User User `json:"user"`
}

type AmountRule struct {
	Currency string `json:"currency"`
	Description string `json:"description"`
	Max_amount float64 `json:"max_amount"`
	Min_amount float64 `json:"min_amount"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type SMSConfig struct {
	Template_id string `json:"template_id"`
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Rate_limit *RateLimitConfig `json:"rate_limit"`
}

type UpdateUserResponse struct {
	User User `json:"user"`
}

type ContextRule struct {
	Description string `json:"description"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Condition string `json:"condition"`
}

type AuditLogResponse struct {
	Entries []*AuditLogEntry `json:"entries"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
}

type ListPendingRequirementsResponse struct {
	Requirements interface{} `json:"requirements"`
	Count int `json:"count"`
}

type VerifyRecoveryCodeResponse struct {
	Message string `json:"message"`
	RemainingCodes int `json:"remainingCodes"`
	Valid bool `json:"valid"`
}

type PreviewTemplate_req struct {
	Variables interface{} `json:"variables"`
}

type BackupAuthContactsResponse struct {
	Contacts []*interface{} `json:"contacts"`
}

type DataDeletionRequest struct {
	ExemptionReason string `json:"exemptionReason"`
	RequestReason string `json:"requestReason"`
	UserId string `json:"userId"`
	ApprovedAt Time `json:"approvedAt"`
	ApprovedBy string `json:"approvedBy"`
	CompletedAt Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	DeleteSections []string `json:"deleteSections"`
	Status string `json:"status"`
	ArchivePath string `json:"archivePath"`
	Id xid.ID `json:"id"`
	RejectedAt Time `json:"rejectedAt"`
	RetentionExempt bool `json:"retentionExempt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ErrorMessage string `json:"errorMessage"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
}

type MultiSessionDeleteResponse struct {
	Status string `json:"status"`
}

type BeginLoginRequest struct {
	UserVerification string `json:"userVerification"`
	UserId string `json:"userId"`
}

type ContentEntryHandler struct {
}

type CancelRecoveryResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type ConsentTypeStatus struct {
	Version string `json:"version"`
	ExpiresAt Time `json:"expiresAt"`
	Granted bool `json:"granted"`
	GrantedAt time.Time `json:"grantedAt"`
	NeedsRenewal bool `json:"needsRenewal"`
	Type string `json:"type"`
}

type ConnectionsResponse struct {
	Connections SocialAccount `json:"connections"`
}

type MockSocialAccountRepository struct {
}

type NotificationsResponse struct {
	Count int `json:"count"`
	Notifications interface{} `json:"notifications"`
}

type ConsentRequest struct {
	Redirect_uri string `json:"redirect_uri"`
	Response_type string `json:"response_type"`
	Scope string `json:"scope"`
	State string `json:"state"`
	Action string `json:"action"`
	Client_id string `json:"client_id"`
	Code_challenge string `json:"code_challenge"`
	Code_challenge_method string `json:"code_challenge_method"`
}

type DiscoverProviderRequest struct {
	Email string `json:"email"`
}

type LinkAccountRequest struct {
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
}

type SessionStats struct {
}

type GetStatusRequest struct {
}

// Webhook represents Webhook configuration
type Webhook struct {
	Id string `json:"id"`
	OrganizationId string `json:"organizationId"`
	Url string `json:"url"`
	Events []string `json:"events"`
	Secret string `json:"secret"`
	Enabled bool `json:"enabled"`
	CreatedAt string `json:"createdAt"`
}

type TokenIntrospectionResponse struct {
	Client_id string `json:"client_id"`
	Exp int64 `json:"exp"`
	Jti string `json:"jti"`
	Scope string `json:"scope"`
	Sub string `json:"sub"`
	Username string `json:"username"`
	Active bool `json:"active"`
	Aud []string `json:"aud"`
	Iat int64 `json:"iat"`
	Iss string `json:"iss"`
	Nbf int64 `json:"nbf"`
	Token_type string `json:"token_type"`
}

type ClientRegistrationRequest struct {
	Require_consent bool `json:"require_consent"`
	Require_pkce bool `json:"require_pkce"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Logo_uri string `json:"logo_uri"`
	Response_types []string `json:"response_types"`
	Scope string `json:"scope"`
	Contacts []string `json:"contacts"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Tos_uri string `json:"tos_uri"`
	Application_type string `json:"application_type"`
	Grant_types []string `json:"grant_types"`
	Redirect_uris []string `json:"redirect_uris"`
	Trusted_client bool `json:"trusted_client"`
	Client_name string `json:"client_name"`
}

type CreateAPIKeyResponse struct {
	Api_key APIKey `json:"api_key"`
	Message string `json:"message"`
}

type ComplianceChecksResponse struct {
	Checks []*interface{} `json:"checks"`
}

type TrustDeviceRequest struct {
	DeviceId string `json:"deviceId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
}

type EmailProviderConfig struct {
	From_name string `json:"from_name"`
	Provider string `json:"provider"`
	Reply_to string `json:"reply_to"`
	Config interface{} `json:"config"`
	From string `json:"from"`
}

type UserInfoResponse struct {
	Gender string `json:"gender"`
	Middle_name string `json:"middle_name"`
	Family_name string `json:"family_name"`
	Locale string `json:"locale"`
	Nickname string `json:"nickname"`
	Phone_number string `json:"phone_number"`
	Sub string `json:"sub"`
	Birthdate string `json:"birthdate"`
	Given_name string `json:"given_name"`
	Profile string `json:"profile"`
	Zoneinfo string `json:"zoneinfo"`
	Name string `json:"name"`
	Phone_number_verified bool `json:"phone_number_verified"`
	Picture string `json:"picture"`
	Preferred_username string `json:"preferred_username"`
	Updated_at int64 `json:"updated_at"`
	Website string `json:"website"`
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
}

type mockSessionService struct {
}

type StartVideoSessionRequest struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type CompliancePolicyResponse struct {
	Id string `json:"id"`
}

type ComplianceUserTrainingResponse struct {
	User_id string `json:"user_id"`
}

type MigrationErrorResponse struct {
	PolicyIndex int `json:"policyIndex"`
	Resource string `json:"resource"`
	Subject string `json:"subject"`
	Error string `json:"error"`
}

type ChallengeResponse struct {
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	SessionId xid.ID `json:"sessionId"`
}

type FactorAdapterRegistry struct {
}

type FactorEnrollmentResponse struct {
	FactorId xid.ID `json:"factorId"`
	ProvisioningData interface{} `json:"provisioningData"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
}

type ListReportsFilter struct {
	AppId *string `json:"appId"`
	Format *string `json:"format"`
	ProfileId *string `json:"profileId"`
	ReportType *string `json:"reportType"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
}

type InitiateChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata interface{} `json:"metadata"`
}

type TOTPConfig struct {
	Period int `json:"period"`
	Window_size int `json:"window_size"`
	Algorithm string `json:"algorithm"`
	Digits int `json:"digits"`
	Enabled bool `json:"enabled"`
	Issuer string `json:"issuer"`
}

type CodesResponse struct {
	Codes []string `json:"codes"`
}

type KeyStore struct {
}

type UpdateProfileRequest struct {
	MfaRequired *bool `json:"mfaRequired"`
	Name *string `json:"name"`
	RetentionDays *int `json:"retentionDays"`
	Status *string `json:"status"`
}

type ComplianceStatus struct {
	ChecksFailed int `json:"checksFailed"`
	ChecksWarning int `json:"checksWarning"`
	LastChecked time.Time `json:"lastChecked"`
	NextAudit time.Time `json:"nextAudit"`
	Score int `json:"score"`
	Standard ComplianceStandard `json:"standard"`
	Violations int `json:"violations"`
	AppId string `json:"appId"`
	ChecksPassed int `json:"checksPassed"`
	OverallStatus string `json:"overallStatus"`
	ProfileId string `json:"profileId"`
}

type RegisterProviderRequest struct {
	OidcRedirectURI string `json:"oidcRedirectURI"`
	ProviderId string `json:"providerId"`
	SamlIssuer string `json:"samlIssuer"`
	Type string `json:"type"`
	OidcClientID string `json:"oidcClientID"`
	OidcClientSecret string `json:"oidcClientSecret"`
	SamlCert string `json:"samlCert"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	AttributeMapping interface{} `json:"attributeMapping"`
	Domain string `json:"domain"`
	OidcIssuer string `json:"oidcIssuer"`
}

type WebhookResponse struct {
	Status string `json:"status"`
	Received bool `json:"received"`
}

type SessionsResponse struct {
	Sessions interface{} `json:"sessions"`
}

type MemoryChallengeStore struct {
}

type GetMigrationStatusRequest struct {
}

type TemplatesResponse struct {
	Count int `json:"count"`
	Templates interface{} `json:"templates"`
}

type SignInRequest struct {
	Provider string `json:"provider"`
	RedirectUrl string `json:"redirectUrl"`
	Scopes []string `json:"scopes"`
}

type PolicyEngine struct {
}

type AccessTokenClaims struct {
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
}

type VerificationListResponse struct {
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Total int `json:"total"`
	Verifications IdentityVerification `json:"verifications"`
}

type IDVerificationErrorResponse struct {
	Error string `json:"error"`
}

type IDVerificationStatusResponse struct {
	Status interface{} `json:"status"`
}

type RedisStateStore struct {
	Client Client `json:"client,omitempty"`
}

type SecurityQuestionsConfig struct {
	MinimumQuestions int `json:"minimumQuestions"`
	RequireMinLength int `json:"requireMinLength"`
	CaseSensitive bool `json:"caseSensitive"`
	Enabled bool `json:"enabled"`
	LockoutDuration time.Duration `json:"lockoutDuration"`
	MaxAttempts int `json:"maxAttempts"`
	PredefinedQuestions []string `json:"predefinedQuestions"`
	RequiredToRecover int `json:"requiredToRecover"`
	AllowCustomQuestions bool `json:"allowCustomQuestions"`
	ForbidCommonAnswers bool `json:"forbidCommonAnswers"`
	MaxAnswerLength int `json:"maxAnswerLength"`
}

type VerifyResponse struct {
	Device_remembered bool `json:"device_remembered"`
	Error string `json:"error"`
	Expires_at time.Time `json:"expires_at"`
	Metadata interface{} `json:"metadata"`
	Security_level SecurityLevel `json:"security_level"`
	Success bool `json:"success"`
	Verification_id string `json:"verification_id"`
}

type TeamsResponse struct {
	Total int `json:"total"`
	Teams Team `json:"teams"`
}

type Service struct {
}

type mockImpersonationRepository struct {
}

type CreateEvidence_req struct {
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	ControlId string `json:"controlId"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
}

type StepUpPolicy struct {
	User_id string `json:"user_id"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
	Rules interface{} `json:"rules"`
	Updated_at time.Time `json:"updated_at"`
	Created_at time.Time `json:"created_at"`
	Description string `json:"description"`
	Enabled bool `json:"enabled"`
	Priority int `json:"priority"`
}

type SendResponse struct {
	Dev_url string `json:"dev_url"`
	Status string `json:"status"`
}

type OrganizationHandler struct {
}

type ResourceResponse struct {
	Attributes ResourceAttribute `json:"attributes"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
	NamespaceId string `json:"namespaceId"`
	Type string `json:"type"`
}

type ConsentExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type CreateDPARequest struct {
	SignedByTitle string `json:"signedByTitle"`
	Version string `json:"version"`
	Content string `json:"content"`
	ExpiryDate Time `json:"expiryDate"`
	Metadata interface{} `json:"metadata"`
	SignedByName string `json:"signedByName"`
	AgreementType string `json:"agreementType"`
	EffectiveDate time.Time `json:"effectiveDate"`
	SignedByEmail string `json:"signedByEmail"`
}

type ListSessionsResponse struct {
	Page int `json:"page"`
	Sessions Session `json:"sessions"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
	Limit int `json:"limit"`
}

type CreateTraining_req struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type MigrationHandler struct {
}

type GetChallengeStatusResponse struct {
	FactorsVerified int `json:"factorsVerified"`
	MaxAttempts int `json:"maxAttempts"`
	Status ChallengeStatus `json:"status"`
	Attempts int `json:"attempts"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
	FactorsRequired int `json:"factorsRequired"`
}

type Status struct {
}

type LinkRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Password string `json:"password"`
}

// Session represents User session
type Session struct {
	Token string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	IpAddress *string `json:"ipAddress,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
	UserId string `json:"userId"`
}

type ImpersonationEndResponse struct {
	Ended_at string `json:"ended_at"`
	Status string `json:"status"`
}

type stateEntry struct {
}

type GetRecoveryStatsRequest struct {
	EndDate time.Time `json:"endDate"`
	OrganizationId string `json:"organizationId"`
	StartDate time.Time `json:"startDate"`
}

type MockOrganizationUIExtension struct {
}

type TestSendTemplate_req struct {
	Recipient string `json:"recipient"`
	Variables interface{} `json:"variables"`
}

type CreateProvider_req struct {
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
	Config interface{} `json:"config"`
	IsDefault bool `json:"isDefault"`
	OrganizationId *string `json:"organizationId,omitempty"`
}

type StripeIdentityProvider struct {
}

type MockRepository struct {
}

type GetDocumentVerificationResponse struct {
	VerifiedAt Time `json:"verifiedAt"`
	ConfidenceScore float64 `json:"confidenceScore"`
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
	RejectionReason string `json:"rejectionReason"`
	Status string `json:"status"`
}

type ComplianceTrainingResponse struct {
	Id string `json:"id"`
}

type AuditLog struct {
}

type NotificationTemplateListResponse struct {
	Templates []*interface{} `json:"templates"`
	Total int `json:"total"`
}

type ClientSummary struct {
	ApplicationType string `json:"applicationType"`
	ClientID string `json:"clientID"`
	CreatedAt string `json:"createdAt"`
	IsOrgLevel bool `json:"isOrgLevel"`
	Name string `json:"name"`
}

type ConsentDecision struct {
}

type StateStore struct {
}

type ImpersonateUser_reqBody struct {
	Duration *time.Duration `json:"duration,omitempty"`
}

type DataExportRequestInput struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type sessionStats struct {
}

type ResourceAttributeRequest struct {
	Name string `json:"name"`
	Required bool `json:"required"`
	Type string `json:"type"`
	Default interface{} `json:"default"`
	Description string `json:"description"`
}

type MetadataResponse struct {
	Metadata string `json:"metadata"`
}

type navItem struct {
}

type CookieConsentRequest struct {
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	Analytics bool `json:"analytics"`
	BannerVersion string `json:"bannerVersion"`
	Essential bool `json:"essential"`
	Functional bool `json:"functional"`
}

type ImpersonationStartResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Session_id string `json:"session_id"`
	Started_at string `json:"started_at"`
	Target_user_id string `json:"target_user_id"`
}

type AddTrustedContactRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
}

type ListUsersRequest struct {
	App_id xid.ID `json:"app_id"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	Role string `json:"role"`
	Search string `json:"search"`
	Status string `json:"status"`
	User_organization_id ID `json:"user_organization_id"`
}

type TestCase struct {
	Action string `json:"action"`
	Expected bool `json:"expected"`
	Name string `json:"name"`
	Principal interface{} `json:"principal"`
	Request interface{} `json:"request"`
	Resource interface{} `json:"resource"`
}

type AdminBlockUser_req struct {
	Reason string `json:"reason"`
}

type ListResponse struct {
	Sessions interface{} `json:"sessions"`
}

type RollbackResponse struct {
	Code string `json:"code"`
	Error string `json:"error"`
	Message string `json:"message"`
}

type ConsentSettingsResponse struct {
	Settings interface{} `json:"settings"`
}

type VideoVerificationSession struct {
}

type SignUpRequest struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type BeginRegisterRequest struct {
	AuthenticatorType string `json:"authenticatorType"`
	Name string `json:"name"`
	RequireResidentKey bool `json:"requireResidentKey"`
	UserId string `json:"userId"`
	UserVerification string `json:"userVerification"`
}

type ActionResponse struct {
	Description string `json:"description"`
	Id string `json:"id"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
	CreatedAt time.Time `json:"createdAt"`
}

type ConsentService struct {
}

type StartImpersonation_reqBody struct {
	Ticket_number *string `json:"ticket_number,omitempty"`
	Duration_minutes *int `json:"duration_minutes,omitempty"`
	Reason string `json:"reason"`
	Target_user_id string `json:"target_user_id"`
}

type MultiStepRecoveryConfig struct {
	AllowStepSkip bool `json:"allowStepSkip"`
	AllowUserChoice bool `json:"allowUserChoice"`
	HighRiskSteps []RecoveryMethod `json:"highRiskSteps"`
	LowRiskSteps []RecoveryMethod `json:"lowRiskSteps"`
	MediumRiskSteps []RecoveryMethod `json:"mediumRiskSteps"`
	SessionExpiry time.Duration `json:"sessionExpiry"`
	Enabled bool `json:"enabled"`
	MinimumSteps int `json:"minimumSteps"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
}

type UploadDocumentRequest struct {
	BackImage string `json:"backImage"`
	DocumentType string `json:"documentType"`
	FrontImage string `json:"frontImage"`
	Selfie string `json:"selfie"`
	SessionId xid.ID `json:"sessionId"`
}

type SessionStatsResponse struct {
	OldestSession *string `json:"oldestSession"`
	TotalSessions int `json:"totalSessions"`
	ActiveSessions int `json:"activeSessions"`
	DeviceCount int `json:"deviceCount"`
	LocationCount int `json:"locationCount"`
	NewestSession *string `json:"newestSession"`
}

type GetMigrationStatusResponse struct {
	PendingRbacPolicies int `json:"pendingRbacPolicies"`
	HasMigratedPolicies bool `json:"hasMigratedPolicies"`
	LastMigrationAt string `json:"lastMigrationAt"`
	MigratedCount int `json:"migratedCount"`
}

type SMSFactorAdapter struct {
}

type GetPolicyResponse struct {
	Allowed_factor_types []string `json:"allowed_factor_types"`
	Enabled bool `json:"enabled"`
	Required_factor_count int `json:"required_factor_count"`
}

type GetByPathResponse struct {
	Message string `json:"message"`
	Code string `json:"code"`
	Error string `json:"error"`
}

type SendVerificationCodeRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
	Target string `json:"target"`
}

type CreateProfileFromTemplateRequest struct {
	Standard ComplianceStandard `json:"standard"`
}

type ComplianceReportResponse struct {
	Id string `json:"id"`
}

type mockProvider struct {
}

type BlockUserRequest struct {
	Reason string `json:"reason"`
}

type ConnectionResponse struct {
	Connection SocialAccount `json:"connection"`
}

type CompleteVideoSessionResponse struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	Result string `json:"result"`
}

type StepUpDevicesResponse struct {
	Count int `json:"count"`
	Devices interface{} `json:"devices"`
}

type PreviewConversionRequest struct {
	Resource string `json:"resource"`
	Subject string `json:"subject"`
	Actions []string `json:"actions"`
	Condition string `json:"condition"`
}

type MFAConfigResponse struct {
	Allowed_factor_types []string `json:"allowed_factor_types"`
	Enabled bool `json:"enabled"`
	Required_factor_count int `json:"required_factor_count"`
}

type VerifyChallengeRequest struct {
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data interface{} `json:"data"`
}

type VerifyTrustedContactRequest struct {
	Token string `json:"token"`
}

type ClientAuthResult struct {
}

type AdminUpdateProviderRequest struct {
	ClientSecret *string `json:"clientSecret"`
	Enabled *bool `json:"enabled"`
	Scopes []string `json:"scopes"`
	ClientId *string `json:"clientId"`
}

type BackupAuthStatsResponse struct {
	Stats interface{} `json:"stats"`
}

type HealthCheckResponse struct {
	Message string `json:"message"`
	ProvidersStatus interface{} `json:"providersStatus"`
	Version string `json:"version"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	Healthy bool `json:"healthy"`
}

type ComplianceTrainingsResponse struct {
	Training []*interface{} `json:"training"`
}

type mockRepository struct {
}

type NamespaceResponse struct {
	ActionCount int `json:"actionCount"`
	Description string `json:"description"`
	EnvironmentId string `json:"environmentId"`
	Name string `json:"name"`
	PolicyCount int `json:"policyCount"`
	ResourceCount int `json:"resourceCount"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	InheritPlatform bool `json:"inheritPlatform"`
	TemplateId *string `json:"templateId"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserOrganizationId *string `json:"userOrganizationId"`
}

type CreateNamespaceRequest struct {
	Description string `json:"description"`
	InheritPlatform bool `json:"inheritPlatform"`
	Name string `json:"name"`
	TemplateId string `json:"templateId"`
}

type EndImpersonation_reqBody struct {
	Impersonation_id string `json:"impersonation_id"`
	Reason *string `json:"reason,omitempty"`
}

type ScopeDefinition struct {
}

type ComplianceEvidencesResponse struct {
	Evidence []*interface{} `json:"evidence"`
}

type CreateActionRequest struct {
	Description string `json:"description"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
}

type GetFactorRequest struct {
}

type ClientDetailsResponse struct {
	AllowedScopes []string `json:"allowedScopes"`
	Contacts []string `json:"contacts"`
	CreatedAt string `json:"createdAt"`
	GrantTypes []string `json:"grantTypes"`
	RequireConsent bool `json:"requireConsent"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	TosURI string `json:"tosURI"`
	TrustedClient bool `json:"trustedClient"`
	ApplicationType string `json:"applicationType"`
	RedirectURIs []string `json:"redirectURIs"`
	RequirePKCE bool `json:"requirePKCE"`
	ClientID string `json:"clientID"`
	IsOrgLevel bool `json:"isOrgLevel"`
	LogoURI string `json:"logoURI"`
	OrganizationID string `json:"organizationID"`
	PolicyURI string `json:"policyURI"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	UpdatedAt string `json:"updatedAt"`
	Name string `json:"name"`
	ResponseTypes []string `json:"responseTypes"`
}

type RemoveTrustedContactResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type RevokeAll_body struct {
	IncludeCurrentSession bool `json:"includeCurrentSession"`
}

type ResourceTypeStats struct {
	ResourceType string `json:"resourceType"`
	AllowRate float64 `json:"allowRate"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	EvaluationCount int64 `json:"evaluationCount"`
}

type VerifyFactor_req struct {
	Code string `json:"code"`
}

type EnrollFactorRequest struct {
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	Type FactorType `json:"type"`
}

type userServiceAdapter struct {
}

type CreateAPIKey_reqBody struct {
	Allowed_ips *[]string `json:"allowed_ips,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata *interface{} `json:"metadata,omitempty"`
	Name string `json:"name"`
	Permissions *interface{} `json:"permissions,omitempty"`
	Rate_limit *int `json:"rate_limit,omitempty"`
	Scopes []string `json:"scopes"`
}

type MFAPolicyResponse struct {
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
	OrganizationId ID `json:"organizationId"`
	RequiredFactorCount int `json:"requiredFactorCount"`
	AllowedFactorTypes []string `json:"allowedFactorTypes"`
	AppId xid.ID `json:"appId"`
}

type BackupCodeFactorAdapter struct {
}

type VerifyEnrolledFactorRequest struct {
	Code string `json:"code"`
	Data interface{} `json:"data"`
}

type BulkDeleteRequest struct {
	Ids []string `json:"ids"`
}

type ComplianceViolation struct {
	ResolvedAt Time `json:"resolvedAt"`
	UserId string `json:"userId"`
	ViolationType string `json:"violationType"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	ProfileId string `json:"profileId"`
	ResolvedBy string `json:"resolvedBy"`
	Severity string `json:"severity"`
	Status string `json:"status"`
	Description string `json:"description"`
}

type DashboardConfig struct {
	Path string `json:"path"`
	ShowRecentChecks bool `json:"showRecentChecks"`
	ShowReports bool `json:"showReports"`
	ShowScore bool `json:"showScore"`
	ShowViolations bool `json:"showViolations"`
	Enabled bool `json:"enabled"`
}

type StepUpVerification struct {
	Expires_at time.Time `json:"expires_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Metadata interface{} `json:"metadata"`
	Method VerificationMethod `json:"method"`
	Org_id string `json:"org_id"`
	Verified_at time.Time `json:"verified_at"`
	Rule_name string `json:"rule_name"`
	Session_id string `json:"session_id"`
	Created_at time.Time `json:"created_at"`
	Device_id string `json:"device_id"`
	Reason string `json:"reason"`
	User_id string `json:"user_id"`
	Security_level SecurityLevel `json:"security_level"`
	User_agent string `json:"user_agent"`
}

type DeviceInfo struct {
}

type ComplianceTraining struct {
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt Time `json:"expiresAt"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Score int `json:"score"`
	Standard ComplianceStandard `json:"standard"`
	Status string `json:"status"`
	CompletedAt Time `json:"completedAt"`
	ProfileId string `json:"profileId"`
}

type ForgetDeviceResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type TestCaseResult struct {
	Passed bool `json:"passed"`
	Actual bool `json:"actual"`
	Error string `json:"error"`
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
	Expected bool `json:"expected"`
	Name string `json:"name"`
}

type TOTPSecret struct {
}

type NotificationStatusResponse struct {
	Status string `json:"status"`
}

type CreateVerificationRequest struct {
}

type bunRepository struct {
}

type GetUserVerificationsResponse struct {
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Total int `json:"total"`
	Verifications IdentityVerification `json:"verifications"`
}

type StepUpAuditLog struct {
	Ip string `json:"ip"`
	Org_id string `json:"org_id"`
	Severity string `json:"severity"`
	User_agent string `json:"user_agent"`
	Event_type string `json:"event_type"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Event_data interface{} `json:"event_data"`
	Id string `json:"id"`
}

type SaveBuilderTemplate_req struct {
	TemplateKey string `json:"templateKey"`
	Document Document `json:"document"`
	Name string `json:"name"`
	Subject string `json:"subject"`
	TemplateId *string `json:"templateId,omitempty"`
}

type CreateABTestVariant_req struct {
	Body string `json:"body"`
	Name string `json:"name"`
	Subject string `json:"subject"`
	Weight int `json:"weight"`
}

type VideoSessionResult struct {
}

type OnfidoProvider struct {
}

type GetByIDResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type TrustedContactInfo struct {
	VerifiedAt Time `json:"verifiedAt"`
	Active bool `json:"active"`
	Email string `json:"email"`
	Id xid.ID `json:"id"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
	Verified bool `json:"verified"`
}

type CompleteTrainingRequest struct {
	Score int `json:"score"`
}

type DevicesResponse struct {
	Count int `json:"count"`
	Devices interface{} `json:"devices"`
}

type AdaptiveMFAConfig struct {
	New_device_risk float64 `json:"new_device_risk"`
	Risk_threshold float64 `json:"risk_threshold"`
	Factor_new_device bool `json:"factor_new_device"`
	Location_change_risk float64 `json:"location_change_risk"`
	Require_step_up_threshold float64 `json:"require_step_up_threshold"`
	Velocity_risk float64 `json:"velocity_risk"`
	Enabled bool `json:"enabled"`
	Factor_ip_reputation bool `json:"factor_ip_reputation"`
	Factor_location_change bool `json:"factor_location_change"`
	Factor_velocity bool `json:"factor_velocity"`
}

type GetVerificationSessionResponse struct {
	Session IdentityVerificationSession `json:"session"`
}

type RollbackRequest struct {
	Reason string `json:"reason"`
}

type ComplianceProfileResponse struct {
	Id string `json:"id"`
}

type BatchEvaluateResponse struct {
	SuccessCount int `json:"successCount"`
	TotalEvaluations int `json:"totalEvaluations"`
	TotalTimeMs float64 `json:"totalTimeMs"`
	FailureCount int `json:"failureCount"`
	Results []*BatchEvaluationResult `json:"results"`
}

type MFAStatus struct {
	TrustedDevice bool `json:"trustedDevice"`
	Enabled bool `json:"enabled"`
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
	GracePeriod Time `json:"gracePeriod"`
	PolicyActive bool `json:"policyActive"`
	RequiredCount int `json:"requiredCount"`
}

type ProviderSessionRequest struct {
}

type SignInResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type ComplianceViolationResponse struct {
	Id string `json:"id"`
}

type EvaluationResult struct {
	Reason string `json:"reason"`
	Security_level SecurityLevel `json:"security_level"`
	Can_remember bool `json:"can_remember"`
	Matched_rules []string `json:"matched_rules"`
	Required bool `json:"required"`
	Requirement_id string `json:"requirement_id"`
	Allowed_methods []VerificationMethod `json:"allowed_methods"`
	Challenge_token string `json:"challenge_token"`
	Current_level SecurityLevel `json:"current_level"`
	Expires_at time.Time `json:"expires_at"`
	Grace_period_ends_at time.Time `json:"grace_period_ends_at"`
	Metadata interface{} `json:"metadata"`
}

type BulkRequest struct {
	Ids []string `json:"ids"`
}

type GetVerificationResponse struct {
	Verification IdentityVerification `json:"verification"`
}

type IPWhitelistConfig struct {
	Enabled bool `json:"enabled"`
	Strict_mode bool `json:"strict_mode"`
}

type GetSecurityQuestionsResponse struct {
	Questions []SecurityQuestionInfo `json:"questions"`
}

type CreateEvidenceRequest struct {
	ControlId string `json:"controlId"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
}

type RedisChallengeStore struct {
}

type RevisionHandler struct {
}

type TemplateEngine struct {
}

type ClientsListResponse struct {
	Clients []ClientSummary `json:"clients"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Total int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type ProviderListResponse struct {
	Providers []ProviderInfo `json:"providers"`
	Total int `json:"total"`
}

type ConsentDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type UserServiceAdapter struct {
}

type ListProfilesFilter struct {
	AppId *string `json:"appId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
}

type CreatePolicy_req struct {
	PolicyType string `json:"policyType"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	Version string `json:"version"`
	Content string `json:"content"`
}

type ChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata interface{} `json:"metadata"`
	UserId xid.ID `json:"userId"`
}

type OTPSentResponse struct {
	Status string `json:"status"`
	Code string `json:"code"`
}

type Adapter struct {
}

type EncryptionService struct {
}

type GetRecoveryStatsResponse struct {
	AdminReviewsRequired int `json:"adminReviewsRequired"`
	FailedRecoveries int `json:"failedRecoveries"`
	MethodStats interface{} `json:"methodStats"`
	PendingRecoveries int `json:"pendingRecoveries"`
	TotalAttempts int `json:"totalAttempts"`
	AverageRiskScore float64 `json:"averageRiskScore"`
	HighRiskAttempts int `json:"highRiskAttempts"`
	SuccessRate float64 `json:"successRate"`
	SuccessfulRecoveries int `json:"successfulRecoveries"`
}

type CompliancePoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type ComplianceTemplate struct {
	PasswordMinLength int `json:"passwordMinLength"`
	RequiredPolicies []string `json:"requiredPolicies"`
	RequiredTraining []string `json:"requiredTraining"`
	RetentionDays int `json:"retentionDays"`
	SessionMaxAge int `json:"sessionMaxAge"`
	AuditFrequencyDays int `json:"auditFrequencyDays"`
	DataResidency string `json:"dataResidency"`
	Description string `json:"description"`
	MfaRequired bool `json:"mfaRequired"`
	Name string `json:"name"`
	Standard ComplianceStandard `json:"standard"`
}

type UpdateFactorRequest struct {
	Metadata interface{} `json:"metadata"`
	Name *string `json:"name"`
	Priority *FactorPriority `json:"priority"`
	Status *FactorStatus `json:"status"`
}

type DocumentCheckConfig struct {
	ValidateExpiry bool `json:"validateExpiry"`
	Enabled bool `json:"enabled"`
	ExtractData bool `json:"extractData"`
	ValidateDataConsistency bool `json:"validateDataConsistency"`
}

type VerifyFactorRequest struct {
	Code string `json:"code"`
}

type AdminGetUserVerificationStatusResponse struct {
	Status UserVerificationStatus `json:"status"`
}

type AppHandler struct {
}

type BackupAuthSessionsResponse struct {
	Sessions []*interface{} `json:"sessions"`
}

type ListEvidenceFilter struct {
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	AppId *string `json:"appId"`
	ControlId *string `json:"controlId"`
	EvidenceType *string `json:"evidenceType"`
}

type ConsentStatusResponse struct {
	Status string `json:"status"`
}

type MockUserRepository struct {
}

type RequestTrustedContactVerificationRequest struct {
	ContactId xid.ID `json:"contactId"`
	SessionId xid.ID `json:"sessionId"`
}

type GenerateReport_req struct {
	Format string `json:"format"`
	Period string `json:"period"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
}

type SSOAuthResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type SAMLLoginRequest struct {
	RelayState string `json:"relayState"`
}

type DisableRequest struct {
	User_id string `json:"user_id"`
}

type DataExportRequest struct {
	ExportUrl string `json:"exportUrl"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	Status string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	ExpiresAt Time `json:"expiresAt"`
	ExportSize int64 `json:"exportSize"`
	Format string `json:"format"`
	IpAddress string `json:"ipAddress"`
	IncludeSections []string `json:"includeSections"`
	UpdatedAt time.Time `json:"updatedAt"`
	CompletedAt Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	ExportPath string `json:"exportPath"`
	UserId string `json:"userId"`
}

type AdminAddProviderRequest struct {
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
	AppId xid.ID `json:"appId"`
	ClientId string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type ScheduleVideoSessionResponse struct {
	Instructions string `json:"instructions"`
	JoinUrl string `json:"joinUrl"`
	Message string `json:"message"`
	ScheduledAt time.Time `json:"scheduledAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type SessionDetailResponse struct {
	Device interface{} `json:"device"`
	Session interface{} `json:"session"`
}

type LimitResult struct {
}

type StepUpVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type UpdatePasskeyRequest struct {
	Name string `json:"name"`
}

type MigrateAllResponse struct {
	CompletedAt string `json:"completedAt"`
	DryRun bool `json:"dryRun"`
	Errors []MigrationErrorResponse `json:"errors"`
	FailedPolicies int `json:"failedPolicies"`
	SkippedPolicies int `json:"skippedPolicies"`
	StartedAt string `json:"startedAt"`
	ConvertedPolicies []PolicyPreviewResponse `json:"convertedPolicies"`
	MigratedPolicies int `json:"migratedPolicies"`
	TotalPolicies int `json:"totalPolicies"`
}

type FactorInfo struct {
	FactorId xid.ID `json:"factorId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Type FactorType `json:"type"`
}

type DiscoveryService struct {
}

type AuthURLResponse struct {
	Url string `json:"url"`
}

type VerifySecurityAnswersResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	CorrectAnswers int `json:"correctAnswers"`
	Message string `json:"message"`
	RequiredAnswers int `json:"requiredAnswers"`
	Valid bool `json:"valid"`
}

type SetUserRole_reqBody struct {
	Role string `json:"role"`
}

type CreateTrainingRequest struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type MockSessionService struct {
}

type AdminBlockUserRequest struct {
	Reason string `json:"reason"`
}

type CookieConsent struct {
	ThirdParty bool `json:"thirdParty"`
	CreatedAt time.Time `json:"createdAt"`
	IpAddress string `json:"ipAddress"`
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	UserId string `json:"userId"`
	Analytics bool `json:"analytics"`
	UpdatedAt time.Time `json:"updatedAt"`
	Essential bool `json:"essential"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	UserAgent string `json:"userAgent"`
	ConsentBannerVersion string `json:"consentBannerVersion"`
	Functional bool `json:"functional"`
	SessionId string `json:"sessionId"`
}

type NoOpNotificationProvider struct {
}

type ListUsersResponse struct {
	Page int `json:"page"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
	Users User `json:"users"`
	Limit int `json:"limit"`
}

type MockEmailService struct {
}

type VerifyRequest struct {
	User_agent string `json:"user_agent"`
	Credential string `json:"credential"`
	Ip string `json:"ip"`
	Method VerificationMethod `json:"method"`
	Remember_device bool `json:"remember_device"`
	Requirement_id string `json:"requirement_id"`
	Challenge_token string `json:"challenge_token"`
	Device_id string `json:"device_id"`
	Device_name string `json:"device_name"`
}

type BulkUnpublishRequest struct {
	Ids []string `json:"ids"`
}

type DataDeletionRequestInput struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type ConsentAuditLog struct {
	ConsentId string `json:"consentId"`
	ConsentType string `json:"consentType"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	NewValue JSONBMap `json:"newValue"`
	OrganizationId string `json:"organizationId"`
	Purpose string `json:"purpose"`
	Reason string `json:"reason"`
	Action string `json:"action"`
	IpAddress string `json:"ipAddress"`
	PreviousValue JSONBMap `json:"previousValue"`
	UserAgent string `json:"userAgent"`
	UserId string `json:"userId"`
}

type StateStorageConfig struct {
	RedisPassword string `json:"redisPassword"`
	StateTtl time.Duration `json:"stateTtl"`
	UseRedis bool `json:"useRedis"`
	RedisAddr string `json:"redisAddr"`
	RedisDb int `json:"redisDb"`
}

type VerifyCodeResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	Message string `json:"message"`
	Valid bool `json:"valid"`
}

type ComplianceReportsResponse struct {
	Reports []*interface{} `json:"reports"`
}

type Enable_body struct {
	Method string `json:"method"`
	User_id string `json:"user_id"`
}

type TokenRevocationRequest struct {
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
	Client_id string `json:"client_id"`
}

type ConsentRecord struct {
	RevokedAt Time `json:"revokedAt"`
	ExpiresAt Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	Purpose string `json:"purpose"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserAgent string `json:"userAgent"`
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	GrantedAt time.Time `json:"grantedAt"`
	IpAddress string `json:"ipAddress"`
	UserId string `json:"userId"`
	Granted bool `json:"granted"`
	Metadata JSONBMap `json:"metadata"`
}

type ConsentAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type RunCheck_req struct {
	CheckType string `json:"checkType"`
}

type StepUpPoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type StatusRequest struct {
	Device_id string `json:"device_id"`
	User_id string `json:"user_id"`
}

// Device represents User device
type Device struct {
	Id string `json:"id"`
	UserId string `json:"userId"`
	Name *string `json:"name,omitempty"`
	Type *string `json:"type,omitempty"`
	LastUsedAt string `json:"lastUsedAt"`
	IpAddress *string `json:"ipAddress,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
}

type ProvidersResponse struct {
	Providers []string `json:"providers"`
}

type SetUserRoleRequest struct {
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Role string `json:"role"`
}

type StepUpRequirementResponse struct {
	Id string `json:"id"`
}

type InvitationResponse struct {
	Invitation Invitation `json:"invitation"`
	Message string `json:"message"`
}

type ClientAuthenticator struct {
}

type ConsentsResponse struct {
	Consents interface{} `json:"consents"`
	Count int `json:"count"`
}

type VerifyRecoveryCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type JumioConfig struct {
	EnableAMLScreening bool `json:"enableAMLScreening"`
	VerificationType string `json:"verificationType"`
	ApiSecret string `json:"apiSecret"`
	EnableExtraction bool `json:"enableExtraction"`
	EnableLiveness bool `json:"enableLiveness"`
	Enabled bool `json:"enabled"`
	EnabledCountries []string `json:"enabledCountries"`
	EnabledDocumentTypes []string `json:"enabledDocumentTypes"`
	PresetId string `json:"presetId"`
	ApiToken string `json:"apiToken"`
	CallbackUrl string `json:"callbackUrl"`
	DataCenter string `json:"dataCenter"`
}

type Rollback_req struct {
	Reason string `json:"reason"`
}

type ConsentRecordResponse struct {
	Id string `json:"id"`
}

type RateLimitConfig struct {
	Enabled bool `json:"enabled"`
	Window time.Duration `json:"window"`
}

type StepUpErrorResponse struct {
	Error string `json:"error"`
}

type SAMLLoginResponse struct {
	ProviderId string `json:"providerId"`
	RedirectUrl string `json:"redirectUrl"`
	RequestId string `json:"requestId"`
}

type GetResponse struct {
	Code string `json:"code"`
	Error string `json:"error"`
	Message string `json:"message"`
}

type BackupAuthRecoveryResponse struct {
	Session_id string `json:"session_id"`
}

type CancelRecoveryRequest struct {
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type ApproveRecoveryRequest struct {
	Notes string `json:"notes"`
	SessionId xid.ID `json:"sessionId"`
}

type ProviderInfo struct {
	CreatedAt string `json:"createdAt"`
	Domain string `json:"domain"`
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
}

type StripeIdentityConfig struct {
	ReturnUrl string `json:"returnUrl"`
	UseMock bool `json:"useMock"`
	WebhookSecret string `json:"webhookSecret"`
	AllowedTypes []string `json:"allowedTypes"`
	ApiKey string `json:"apiKey"`
	Enabled bool `json:"enabled"`
	RequireLiveCapture bool `json:"requireLiveCapture"`
	RequireMatchingSelfie bool `json:"requireMatchingSelfie"`
}

type PreviewTemplateRequest struct {
	Variables interface{} `json:"variables"`
}

type ProviderConfigResponse struct {
	AppId string `json:"appId"`
	Message string `json:"message"`
	Provider string `json:"provider"`
}

type ApproveRecoveryResponse struct {
	Approved bool `json:"approved"`
	ApprovedAt time.Time `json:"approvedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
}

type DocumentVerificationRequest struct {
}

type ComplianceReportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type SetActive_body struct {
	Id string `json:"id"`
}

type IDVerificationSessionResponse struct {
	Session interface{} `json:"session"`
}

type ListDevicesResponse struct {
	Devices []*Device `json:"devices"`
}

type ReportsConfig struct {
	Enabled bool `json:"enabled"`
	Formats []string `json:"formats"`
	IncludeEvidence bool `json:"includeEvidence"`
	RetentionDays int `json:"retentionDays"`
	Schedule string `json:"schedule"`
	StoragePath string `json:"storagePath"`
}

type App struct {
}

type ProviderRegisteredResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Type string `json:"type"`
}

type AdminGetUserVerificationsResponse struct {
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Total int `json:"total"`
	Verifications IdentityVerification `json:"verifications"`
}

type StatsResponse struct {
	Total_users int `json:"total_users"`
	Active_sessions int `json:"active_sessions"`
	Active_users int `json:"active_users"`
	Banned_users int `json:"banned_users"`
	Timestamp string `json:"timestamp"`
	Total_sessions int `json:"total_sessions"`
}

type BanUserRequest struct {
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Expires_at Time `json:"expires_at"`
}

type ComplianceDashboardResponse struct {
	Metrics interface{} `json:"metrics"`
}

type UserAdapter struct {
}

type FinishLoginRequest struct {
	Remember bool `json:"remember"`
	Response interface{} `json:"response"`
}

type RevokeAllRequest struct {
	IncludeCurrentSession bool `json:"includeCurrentSession"`
}

type RateLimit struct {
}

type GetSecurityQuestionsRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type RevokeTokenService struct {
}

type OIDCLoginResponse struct {
	AuthUrl string `json:"authUrl"`
	Nonce string `json:"nonce"`
	ProviderId string `json:"providerId"`
	State string `json:"state"`
}

type VerifyCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type AccountLockoutError struct {
}

type PolicyStats struct {
	DenyCount int64 `json:"denyCount"`
	EvaluationCount int64 `json:"evaluationCount"`
	PolicyId string `json:"policyId"`
	PolicyName string `json:"policyName"`
	AllowCount int64 `json:"allowCount"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
}

type CompleteRecoveryResponse struct {
	Status RecoveryStatus `json:"status"`
	Token string `json:"token"`
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
}

type MockStateStore struct {
}

type AuditConfig struct {
	LogAllAttempts bool `json:"logAllAttempts"`
	LogDeviceInfo bool `json:"logDeviceInfo"`
	LogFailed bool `json:"logFailed"`
	LogIpAddress bool `json:"logIpAddress"`
	LogUserAgent bool `json:"logUserAgent"`
	Enabled bool `json:"enabled"`
	ImmutableLogs bool `json:"immutableLogs"`
	LogSuccessful bool `json:"logSuccessful"`
	RetentionDays int `json:"retentionDays"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
	ArchiveOldLogs bool `json:"archiveOldLogs"`
}

type SendVerificationCodeResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	MaskedTarget string `json:"maskedTarget"`
	Message string `json:"message"`
	Sent bool `json:"sent"`
}

type ComplianceEvidenceResponse struct {
	Id string `json:"id"`
}

type ListViolationsFilter struct {
	Severity *string `json:"severity"`
	Status *string `json:"status"`
	UserId *string `json:"userId"`
	ViolationType *string `json:"violationType"`
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
}

type GetChallengeStatusRequest struct {
}

type RiskContext struct {
}

type CallbackResult struct {
}

type MemoryStateStore struct {
}

type GetRecoveryConfigResponse struct {
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
}

type RecoveryConfiguration struct {
}

type SetupSecurityQuestionRequest struct {
	Answer string `json:"answer"`
	CustomText string `json:"customText"`
	QuestionId int `json:"questionId"`
}

type GenerateBackupCodes_body struct {
	Count int `json:"count"`
	User_id string `json:"user_id"`
}

type AccessConfig struct {
	AllowApiAccess bool `json:"allowApiAccess"`
	AllowDashboardAccess bool `json:"allowDashboardAccess"`
	RateLimitPerMinute int `json:"rateLimitPerMinute"`
	RequireAuthentication bool `json:"requireAuthentication"`
	RequireRbac bool `json:"requireRbac"`
}

type OAuthState struct {
	Extra_scopes []string `json:"extra_scopes"`
	Link_user_id ID `json:"link_user_id"`
	Provider string `json:"provider"`
	Redirect_url string `json:"redirect_url"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Created_at time.Time `json:"created_at"`
}

type StartVideoSessionResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	SessionUrl string `json:"sessionUrl"`
	StartedAt time.Time `json:"startedAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type RequestTrustedContactVerificationResponse struct {
	ContactId xid.ID `json:"contactId"`
	ContactName string `json:"contactName"`
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	NotifiedAt time.Time `json:"notifiedAt"`
}

type StepUpStatusResponse struct {
	Status string `json:"status"`
}

type ValidatePolicyResponse struct {
	Complexity int `json:"complexity"`
	Error string `json:"error"`
	Errors []string `json:"errors"`
	Message string `json:"message"`
	Valid bool `json:"valid"`
	Warnings []string `json:"warnings"`
}

type PreviewConversionResponse struct {
	ResourceType string `json:"resourceType"`
	Success bool `json:"success"`
	CelExpression string `json:"celExpression"`
	Error string `json:"error"`
	PolicyName string `json:"policyName"`
	ResourceId string `json:"resourceId"`
}

type AuthorizeRequest struct {
	State string `json:"state"`
	Ui_locales string `json:"ui_locales"`
	Acr_values string `json:"acr_values"`
	Code_challenge string `json:"code_challenge"`
	Login_hint string `json:"login_hint"`
	Max_age *int `json:"max_age"`
	Nonce string `json:"nonce"`
	Prompt string `json:"prompt"`
	Redirect_uri string `json:"redirect_uri"`
	Scope string `json:"scope"`
	Client_id string `json:"client_id"`
	Code_challenge_method string `json:"code_challenge_method"`
	Id_token_hint string `json:"id_token_hint"`
	Response_type string `json:"response_type"`
}

type ClientUpdateRequest struct {
	Logo_uri string `json:"logo_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Require_pkce *bool `json:"require_pkce"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Trusted_client *bool `json:"trusted_client"`
	Allowed_scopes []string `json:"allowed_scopes"`
	Contacts []string `json:"contacts"`
	Name string `json:"name"`
	Policy_uri string `json:"policy_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Require_consent *bool `json:"require_consent"`
	Response_types []string `json:"response_types"`
	Tos_uri string `json:"tos_uri"`
	Grant_types []string `json:"grant_types"`
}

type CallbackResponse struct {
	Token string `json:"token"`
	User User `json:"user"`
	Session Session `json:"session"`
}

type RolesResponse struct {
	Roles Role `json:"roles"`
}

type GenerateRecoveryCodesRequest struct {
	Count int `json:"count"`
	Format string `json:"format"`
}

type ComplianceReport struct {
	Id string `json:"id"`
	Period string `json:"period"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Format string `json:"format"`
	GeneratedBy string `json:"generatedBy"`
	ReportType string `json:"reportType"`
	Status string `json:"status"`
	Summary interface{} `json:"summary"`
	AppId string `json:"appId"`
	FileSize int64 `json:"fileSize"`
	FileUrl string `json:"fileUrl"`
}

type ComplianceCheckResponse struct {
	Id string `json:"id"`
}

type StepUpRequirementsResponse struct {
	Requirements []*interface{} `json:"requirements"`
}

type IDVerificationListResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type VersioningConfig struct {
	AutoCleanup bool `json:"autoCleanup"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	MaxVersions int `json:"maxVersions"`
	RetentionDays int `json:"retentionDays"`
}

// User represents User account
type User struct {
	Id string `json:"id"`
	Email string `json:"email"`
	Name *string `json:"name,omitempty"`
	EmailVerified bool `json:"emailVerified"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	OrganizationId *string `json:"organizationId,omitempty"`
}

type RecoveryCodesConfig struct {
	AllowDownload bool `json:"allowDownload"`
	AllowPrint bool `json:"allowPrint"`
	AutoRegenerate bool `json:"autoRegenerate"`
	CodeCount int `json:"codeCount"`
	CodeLength int `json:"codeLength"`
	Enabled bool `json:"enabled"`
	Format string `json:"format"`
	RegenerateCount int `json:"regenerateCount"`
}

type EmailVerificationConfig struct {
	CodeExpiry time.Duration `json:"codeExpiry"`
	CodeLength int `json:"codeLength"`
	EmailTemplate string `json:"emailTemplate"`
	Enabled bool `json:"enabled"`
	FromAddress string `json:"fromAddress"`
	FromName string `json:"fromName"`
	MaxAttempts int `json:"maxAttempts"`
	RequireEmailProof bool `json:"requireEmailProof"`
}

type SetupSecurityQuestionsRequest struct {
	Questions []SetupSecurityQuestionRequest `json:"questions"`
}

type RequestDataExportRequest struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type LinkAccountResponse struct {
	Url string `json:"url"`
}

type CreateVerificationSessionRequest struct {
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
}

type ConsentCookieResponse struct {
	Preferences interface{} `json:"preferences"`
}

type CompleteVideoSessionRequest struct {
	LivenessPassed bool `json:"livenessPassed"`
	LivenessScore float64 `json:"livenessScore"`
	Notes string `json:"notes"`
	VerificationResult string `json:"verificationResult"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type BackupAuthQuestionsResponse struct {
	Questions []string `json:"questions"`
}

type BackupAuthConfigResponse struct {
	Config interface{} `json:"config"`
}

type ComplianceTemplatesResponse struct {
	Templates []*interface{} `json:"templates"`
}

type auditServiceAdapter struct {
}

type MembersResponse struct {
	Members Member `json:"members"`
	Total int `json:"total"`
}

type TemplateResponse struct {
	Name string `json:"name"`
	Parameters TemplateParameter `json:"parameters"`
	Category string `json:"category"`
	Description string `json:"description"`
	Examples []string `json:"examples"`
	Expression string `json:"expression"`
	Id string `json:"id"`
}

type TrustedContactsConfig struct {
	MaxNotificationsPerDay int `json:"maxNotificationsPerDay"`
	MaximumContacts int `json:"maximumContacts"`
	RequireVerification bool `json:"requireVerification"`
	RequiredToRecover int `json:"requiredToRecover"`
	AllowEmailContacts bool `json:"allowEmailContacts"`
	AllowPhoneContacts bool `json:"allowPhoneContacts"`
	MinimumContacts int `json:"minimumContacts"`
	VerificationExpiry time.Duration `json:"verificationExpiry"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	Enabled bool `json:"enabled"`
}

type RecoverySession struct {
}

type AccountLockedResponse struct {
	Code string `json:"code"`
	Locked_minutes int `json:"locked_minutes"`
	Locked_until time.Time `json:"locked_until"`
	Message string `json:"message"`
}

type RunCheckRequest struct {
	CheckType string `json:"checkType"`
}

type MigrateRolesRequest struct {
	DryRun bool `json:"dryRun"`
}

type AdminBypassRequest struct {
	Duration int `json:"duration"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
}

type LinkResponse struct {
	Message string `json:"message"`
	User interface{} `json:"user"`
}

type JWKSService struct {
}

type FactorEnrollmentRequest struct {
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	Type FactorType `json:"type"`
}

type IDVerificationWebhookResponse struct {
	Status string `json:"status"`
}

type CreateSessionHTTPRequest struct {
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
}

type SendOTPRequest struct {
	User_id string `json:"user_id"`
}

type CreateResponse struct {
	Webhook Webhook `json:"webhook"`
}

type RateLimiter struct {
}

type ListTrustedContactsResponse struct {
	Contacts []TrustedContactInfo `json:"contacts"`
	Count int `json:"count"`
}

type VerificationResponse struct {
	FactorsRemaining int `json:"factorsRemaining"`
	SessionComplete bool `json:"sessionComplete"`
	Success bool `json:"success"`
	Token string `json:"token"`
	ExpiresAt Time `json:"expiresAt"`
}

type IDVerificationResponse struct {
	Verification interface{} `json:"verification"`
}

type CreateSessionRequest struct {
}

type RecordCookieConsentRequest struct {
	Functional bool `json:"functional"`
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	Analytics bool `json:"analytics"`
	BannerVersion string `json:"bannerVersion"`
	Essential bool `json:"essential"`
}

type RegisterProviderResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Type string `json:"type"`
}

type OIDCCallbackResponse struct {
	Token string `json:"token"`
	User User `json:"user"`
	Session Session `json:"session"`
}

type DeclareABTestWinner_req struct {
	AbTestGroup string `json:"abTestGroup"`
	WinnerId string `json:"winnerId"`
}

type VerificationResult struct {
}

type DeleteRequest struct {
	Id string `json:"id"`
}

type ListTrustedDevicesResponse struct {
	Count int `json:"count"`
	Devices []TrustedDevice `json:"devices"`
}

type OIDCState struct {
}

type OIDCLoginRequest struct {
	Nonce string `json:"nonce"`
	RedirectUri string `json:"redirectUri"`
	Scope string `json:"scope"`
	State string `json:"state"`
}

