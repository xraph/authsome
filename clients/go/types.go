package authsome

// Auto-generated types

type ConsentRecordResponse struct {
	Id string `json:"id"`
}

type SendCodeRequest struct {
	Phone string `json:"phone"`
}

type VerifyEnrolledFactorRequest struct {
	 *string `json:",omitempty"`
	Code string `json:"code"`
	Data  `json:"data"`
}

type RevokeTrustedDeviceRequest struct {
	 *string `json:",omitempty"`
}

type AdminAddProviderRequest struct {
	AppId xid.ID `json:"appId"`
	ClientId string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
}

type SignInResponse struct {
	Token string `json:"token"`
	User  `json:"user"`
	Session  `json:"session"`
}

type NotificationsResponse struct {
	Count int `json:"count"`
	Notifications  `json:"notifications"`
}

type KeyPair struct {
	 *string `json:",omitempty"`
}

type RiskFactor struct {
	 *float64 `json:",omitempty"`
}

type ScheduleVideoSessionRequest struct {
	ScheduledAt time.Time `json:"scheduledAt"`
	SessionId xid.ID `json:"sessionId"`
	TimeZone string `json:"timeZone"`
}

type BackupAuthRecoveryResponse struct {
	Session_id string `json:"session_id"`
}

type EvaluateRequest struct {
	Route string `json:"route"`
	Action string `json:"action"`
	Amount float64 `json:"amount"`
	Currency string `json:"currency"`
	Metadata  `json:"metadata"`
	Method string `json:"method"`
	Resource_type string `json:"resource_type"`
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

type StartVideoSessionResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	SessionUrl string `json:"sessionUrl"`
	StartedAt time.Time `json:"startedAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type EmailConfig struct {
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
}

type RiskContext struct {
	 *xid.ID `json:",omitempty"`
}

type StepUpRememberedDevice struct {
	Last_used_at time.Time `json:"last_used_at"`
	Remembered_at time.Time `json:"remembered_at"`
	Security_level SecurityLevel `json:"security_level"`
	Device_id string `json:"device_id"`
	Id string `json:"id"`
	Org_id string `json:"org_id"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Device_name string `json:"device_name"`
	Expires_at time.Time `json:"expires_at"`
	Ip string `json:"ip"`
}

type TestProvider_req struct {
	ProviderType string `json:"providerType"`
	TestRecipient string `json:"testRecipient"`
	Config  `json:"config"`
	ProviderName string `json:"providerName"`
}

type ConsentNotificationsConfig struct {
	Enabled bool `json:"enabled"`
	NotifyDeletionComplete bool `json:"notifyDeletionComplete"`
	NotifyDpoEmail string `json:"notifyDpoEmail"`
	NotifyOnGrant bool `json:"notifyOnGrant"`
	NotifyOnRevoke bool `json:"notifyOnRevoke"`
	Channels []string `json:"channels"`
	NotifyDeletionApproved bool `json:"notifyDeletionApproved"`
	NotifyExportReady bool `json:"notifyExportReady"`
	NotifyOnExpiry bool `json:"notifyOnExpiry"`
}

type ResetUserMFARequest struct {
	 *string `json:",omitempty"`
	Reason string `json:"reason"`
}

type ListFactorsRequest struct {
	 *bool `json:",omitempty"`
}

type MembersResponse struct {
	Members []*organization.Member `json:"members"`
	Total int `json:"total"`
}

type RedisStateStore struct {
	 **redis.Client `json:",omitempty"`
}

type LinkRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Password string `json:"password"`
}

type DocumentVerificationResult struct {
	 *string `json:",omitempty"`
}

type TrackNotificationEvent_req struct {
	Event string `json:"event"`
	EventData * `json:"eventData,omitempty"`
	NotificationId string `json:"notificationId"`
	OrganizationId **string `json:"organizationId,omitempty"`
	TemplateId string `json:"templateId"`
}

type VerifyRecoveryCodeResponse struct {
	Message string `json:"message"`
	RemainingCodes int `json:"remainingCodes"`
	Valid bool `json:"valid"`
}

type SendWithTemplateRequest struct {
	AppId xid.ID `json:"appId"`
	Language string `json:"language"`
	Metadata  `json:"metadata"`
	Recipient string `json:"recipient"`
	TemplateKey string `json:"templateKey"`
	Type notification.NotificationType `json:"type"`
	Variables  `json:"variables"`
}

type ListReportsFilter struct {
	AppId *string `json:"appId"`
	Format *string `json:"format"`
	ProfileId *string `json:"profileId"`
	ReportType *string `json:"reportType"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
}

type BackupAuthContactsResponse struct {
	Contacts []* `json:"contacts"`
}

type AddCustomPermission_req struct {
	Category string `json:"category"`
	Description string `json:"description"`
	Name string `json:"name"`
}

type StepUpPolicy struct {
	Created_at time.Time `json:"created_at"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
	Priority int `json:"priority"`
	User_id string `json:"user_id"`
	Description string `json:"description"`
	Enabled bool `json:"enabled"`
	Id string `json:"id"`
	Rules  `json:"rules"`
	Updated_at time.Time `json:"updated_at"`
}

type StepUpAuditLogsResponse struct {
	Audit_logs []* `json:"audit_logs"`
}

type SetupSecurityQuestionRequest struct {
	CustomText string `json:"customText"`
	QuestionId int `json:"questionId"`
	Answer string `json:"answer"`
}

type ListRecoverySessionsResponse struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Sessions []RecoverySessionInfo `json:"sessions"`
	TotalCount int `json:"totalCount"`
}

type CancelRecoveryRequest struct {
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type PrivacySettingsRequest struct {
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	ConsentRequired *bool `json:"consentRequired"`
	ContactPhone string `json:"contactPhone"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	DataRetentionDays *int `json:"dataRetentionDays"`
	AllowDataPortability *bool `json:"allowDataPortability"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	ExportFormat []string `json:"exportFormat"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	CcpaMode *bool `json:"ccpaMode"`
	ContactEmail string `json:"contactEmail"`
	DpoEmail string `json:"dpoEmail"`
	GdprMode *bool `json:"gdprMode"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
}

type DataExportRequest struct {
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt"`
	IpAddress string `json:"ipAddress"`
	UserId string `json:"userId"`
	CompletedAt *time.Time `json:"completedAt"`
	ExportPath string `json:"exportPath"`
	OrganizationId string `json:"organizationId"`
	Status string `json:"status"`
	ExportSize int64 `json:"exportSize"`
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
	UpdatedAt time.Time `json:"updatedAt"`
	ErrorMessage string `json:"errorMessage"`
	ExportUrl string `json:"exportUrl"`
	Id xid.ID `json:"id"`
}

type ListTrustedContactsResponse struct {
	Contacts []TrustedContactInfo `json:"contacts"`
	Count int `json:"count"`
}

type GenerateRecoveryCodesRequest struct {
	Count int `json:"count"`
	Format string `json:"format"`
}

type ChallengeResponse struct {
	ChallengeId xid.ID `json:"challengeId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	SessionId xid.ID `json:"sessionId"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
}

type ComplianceTrainingsResponse struct {
	Training []* `json:"training"`
}

// Device represents User device
type Device struct {
	LastUsedAt string `json:"lastUsedAt"`
	IpAddress *string `json:"ipAddress,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
	Id string `json:"id"`
	UserId string `json:"userId"`
	Name *string `json:"name,omitempty"`
	Type *string `json:"type,omitempty"`
}

type VerifyRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Remember bool `json:"remember"`
	Code string `json:"code"`
}

type WebhookPayload struct {
	 * `json:",omitempty"`
}

type JWK struct {
	Alg string `json:"alg"`
	E string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N string `json:"n"`
	Use string `json:"use"`
}

type ProviderInfo struct {
	CreatedAt string `json:"createdAt"`
	Domain string `json:"domain"`
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
}

type ComplianceProfile struct {
	PasswordMinLength int `json:"passwordMinLength"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	SessionMaxAge int `json:"sessionMaxAge"`
	Status string `json:"status"`
	AuditLogExport bool `json:"auditLogExport"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	DpoContact string `json:"dpoContact"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	RegularAccessReview bool `json:"regularAccessReview"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	DataResidency string `json:"dataResidency"`
	MfaRequired bool `json:"mfaRequired"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	RbacRequired bool `json:"rbacRequired"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	LeastPrivilege bool `json:"leastPrivilege"`
	Metadata  `json:"metadata"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	RetentionDays int `json:"retentionDays"`
	Standards []ComplianceStandard `json:"standards"`
	ComplianceContact string `json:"complianceContact"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	UpdatedAt time.Time `json:"updatedAt"`
	AppId string `json:"appId"`
}

type CreateEvidenceRequest struct {
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	ControlId string `json:"controlId"`
}

type IntrospectionService struct {
	 *UserService `json:",omitempty"`
}

type StepUpVerificationResponse struct {
	Expires_at string `json:"expires_at"`
	Verified bool `json:"verified"`
}

type ComplianceEvidence struct {
	Metadata  `json:"metadata"`
	Standard ComplianceStandard `json:"standard"`
	AppId string `json:"appId"`
	ControlId string `json:"controlId"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileHash string `json:"fileHash"`
	ProfileId string `json:"profileId"`
	Title string `json:"title"`
	CollectedBy string `json:"collectedBy"`
	FileUrl string `json:"fileUrl"`
	Id string `json:"id"`
}

type StartImpersonation_reqBody struct {
	Reason string `json:"reason"`
	Target_user_id string `json:"target_user_id"`
	Ticket_number *string `json:"ticket_number,omitempty"`
	Duration_minutes *int `json:"duration_minutes,omitempty"`
}

// MessageResponse represents Simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

type CompleteRecoveryResponse struct {
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
	Status RecoveryStatus `json:"status"`
	Token string `json:"token"`
}

type IDVerificationErrorResponse struct {
	Error string `json:"error"`
}

type RecoveryAttemptLog struct {
	 * `json:",omitempty"`
}

type VerifySecurityAnswersResponse struct {
	CorrectAnswers int `json:"correctAnswers"`
	Message string `json:"message"`
	RequiredAnswers int `json:"requiredAnswers"`
	Valid bool `json:"valid"`
	AttemptsLeft int `json:"attemptsLeft"`
}

type RevokeTokenService struct {
	 **repo.OAuthTokenRepository `json:",omitempty"`
}

type SMSProviderConfig struct {
	Config  `json:"config"`
	From string `json:"from"`
	Provider string `json:"provider"`
}

type RolesResponse struct {
	Roles []*apikey.Role `json:"roles"`
}

type ContinueRecoveryRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
}

type ProvidersAppResponse struct {
	AppId string `json:"appId"`
	Providers []string `json:"providers"`
}

type SendVerificationCodeResponse struct {
	MaskedTarget string `json:"maskedTarget"`
	Message string `json:"message"`
	Sent bool `json:"sent"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type IDVerificationSessionResponse struct {
	Session  `json:"session"`
}

type AMLMatch struct {
	 *string `json:",omitempty"`
}

type RateLimitRule struct {
	Max int `json:"max"`
	Window time.Duration `json:"window"`
}

type DevicesResponse struct {
	Count int `json:"count"`
	Devices  `json:"devices"`
}

type ProviderDetailResponse struct {
	AttributeMapping  `json:"attributeMapping"`
	Domain string `json:"domain"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
	ProviderId string `json:"providerId"`
	SamlIssuer string `json:"samlIssuer"`
	UpdatedAt string `json:"updatedAt"`
	CreatedAt string `json:"createdAt"`
	HasSamlCert bool `json:"hasSamlCert"`
	OidcClientID string `json:"oidcClientID"`
	OidcIssuer string `json:"oidcIssuer"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	Type string `json:"type"`
}

type ImpersonationVerifyResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id string `json:"target_user_id"`
}

type AssignRole_reqBody struct {
	RoleID string `json:"roleID"`
}

type NotificationStatusResponse struct {
	Status string `json:"status"`
}

type ImpersonateUser_reqBody struct {
	Duration *time.Duration `json:"duration,omitempty"`
}

type MockStateStore struct {
	 * `json:",omitempty"`
}

type AmountRule struct {
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Currency string `json:"currency"`
	Description string `json:"description"`
	Max_amount float64 `json:"max_amount"`
	Min_amount float64 `json:"min_amount"`
}

type AuditServiceAdapter struct {
	 **audit.Service `json:",omitempty"`
}

type NotificationChannels struct {
	Email bool `json:"email"`
	Slack bool `json:"slack"`
	Webhook bool `json:"webhook"`
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

type ComplianceCheckResponse struct {
	Id string `json:"id"`
}

type PasskeyInfo struct {
	Aaguid string `json:"aaguid"`
	AuthenticatorType string `json:"authenticatorType"`
	CredentialId string `json:"credentialId"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	Name string `json:"name"`
	SignCount uint `json:"signCount"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	IsResidentKey bool `json:"isResidentKey"`
}

type GetRecoveryStatsResponse struct {
	TotalAttempts int `json:"totalAttempts"`
	AdminReviewsRequired int `json:"adminReviewsRequired"`
	AverageRiskScore float64 `json:"averageRiskScore"`
	FailedRecoveries int `json:"failedRecoveries"`
	HighRiskAttempts int `json:"highRiskAttempts"`
	MethodStats  `json:"methodStats"`
	SuccessRate float64 `json:"successRate"`
	PendingRecoveries int `json:"pendingRecoveries"`
	SuccessfulRecoveries int `json:"successfulRecoveries"`
}

type TOTPConfig struct {
	Algorithm string `json:"algorithm"`
	Digits int `json:"digits"`
	Enabled bool `json:"enabled"`
	Issuer string `json:"issuer"`
	Period int `json:"period"`
	Window_size int `json:"window_size"`
}

type ImpersonationErrorResponse struct {
	Error string `json:"error"`
}

type SaveNotificationSettings_req struct {
	RetryDelay string `json:"retryDelay"`
	AutoSendWelcome bool `json:"autoSendWelcome"`
	CleanupAfter string `json:"cleanupAfter"`
	RetryAttempts int `json:"retryAttempts"`
}

type EnrollFactorRequest struct {
	Type FactorType `json:"type"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
}

type SMSConfig struct {
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
}

type CreateProfileRequest struct {
	RbacRequired bool `json:"rbacRequired"`
	RegularAccessReview bool `json:"regularAccessReview"`
	RetentionDays int `json:"retentionDays"`
	AppId string `json:"appId"`
	AuditLogExport bool `json:"auditLogExport"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	DpoContact string `json:"dpoContact"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	Name string `json:"name"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	SessionMaxAge int `json:"sessionMaxAge"`
	ComplianceContact string `json:"complianceContact"`
	DataResidency string `json:"dataResidency"`
	Metadata  `json:"metadata"`
	MfaRequired bool `json:"mfaRequired"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	Standards []ComplianceStandard `json:"standards"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	LeastPrivilege bool `json:"leastPrivilege"`
	PasswordMinLength int `json:"passwordMinLength"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
}

type mockRepository struct {
	 *error `json:",omitempty"`
}

type TokenRequest struct {
	Audience string `json:"audience"`
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
	Client_secret string `json:"client_secret"`
	Code string `json:"code"`
	Code_verifier string `json:"code_verifier"`
	Grant_type string `json:"grant_type"`
	Redirect_uri string `json:"redirect_uri"`
	Refresh_token string `json:"refresh_token"`
}

type ListSessionsResponse struct {
	Limit int `json:"limit"`
	Page int `json:"page"`
	Sessions []*session.Session `json:"sessions"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
}

type SAMLLoginRequest struct {
	RelayState string `json:"relayState"`
}

type VerifyCodeResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	Message string `json:"message"`
	Valid bool `json:"valid"`
}

type RejectRecoveryRequest struct {
	Notes string `json:"notes"`
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type ApproveRecoveryResponse struct {
	Approved bool `json:"approved"`
	ApprovedAt time.Time `json:"approvedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
}

type DefaultProviderRegistry struct {
	 *SMSProvider `json:",omitempty"`
}

type SendRequest struct {
	Email string `json:"email"`
}

type AdminUpdateProviderRequest struct {
	ClientId *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
	Enabled *bool `json:"enabled"`
	Scopes []string `json:"scopes"`
}

type RenderTemplate_req struct {
	Template string `json:"template"`
	Variables  `json:"variables"`
}

type UserInfoResponse struct {
	Email string `json:"email"`
	Phone_number_verified bool `json:"phone_number_verified"`
	Picture string `json:"picture"`
	Website string `json:"website"`
	Zoneinfo string `json:"zoneinfo"`
	Gender string `json:"gender"`
	Middle_name string `json:"middle_name"`
	Phone_number string `json:"phone_number"`
	Profile string `json:"profile"`
	Sub string `json:"sub"`
	Updated_at int64 `json:"updated_at"`
	Birthdate string `json:"birthdate"`
	Email_verified bool `json:"email_verified"`
	Family_name string `json:"family_name"`
	Locale string `json:"locale"`
	Nickname string `json:"nickname"`
	Given_name string `json:"given_name"`
	Name string `json:"name"`
	Preferred_username string `json:"preferred_username"`
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

type AuditConfig struct {
	LogUserAgent bool `json:"logUserAgent"`
	RetentionDays int `json:"retentionDays"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	Enabled bool `json:"enabled"`
	ImmutableLogs bool `json:"immutableLogs"`
	LogAllAttempts bool `json:"logAllAttempts"`
	LogDeviceInfo bool `json:"logDeviceInfo"`
	LogFailed bool `json:"logFailed"`
	LogSuccessful bool `json:"logSuccessful"`
	LogIpAddress bool `json:"logIpAddress"`
}

type DataExportConfig struct {
	RequestPeriod time.Duration `json:"requestPeriod"`
	StoragePath string `json:"storagePath"`
	AllowedFormats []string `json:"allowedFormats"`
	AutoCleanup bool `json:"autoCleanup"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	DefaultFormat string `json:"defaultFormat"`
	Enabled bool `json:"enabled"`
	ExpiryHours int `json:"expiryHours"`
	IncludeSections []string `json:"includeSections"`
	MaxRequests int `json:"maxRequests"`
	MaxExportSize int64 `json:"maxExportSize"`
}

type ClientsListResponse struct {
	Clients []ClientSummary `json:"clients"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Total int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type ClientAuthResult struct {
	 *string `json:",omitempty"`
}

type ReportsConfig struct {
	Enabled bool `json:"enabled"`
	Formats []string `json:"formats"`
	IncludeEvidence bool `json:"includeEvidence"`
	RetentionDays int `json:"retentionDays"`
	Schedule string `json:"schedule"`
	StoragePath string `json:"storagePath"`
}

type VideoSessionInfo struct {
	 *bool `json:",omitempty"`
}

type VideoVerificationConfig struct {
	SessionDuration time.Duration `json:"sessionDuration"`
	LivenessThreshold float64 `json:"livenessThreshold"`
	Provider string `json:"provider"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireScheduling bool `json:"requireScheduling"`
	Enabled bool `json:"enabled"`
	MinScheduleAdvance time.Duration `json:"minScheduleAdvance"`
	RecordSessions bool `json:"recordSessions"`
	RecordingRetention time.Duration `json:"recordingRetention"`
	RequireLivenessCheck bool `json:"requireLivenessCheck"`
}

type Config struct {
	EmailVerification EmailVerificationConfig `json:"emailVerification"`
	Enabled bool `json:"enabled"`
	Notifications NotificationsConfig `json:"notifications"`
	RateLimiting RateLimitingConfig `json:"rateLimiting"`
	RiskAssessment RiskAssessmentConfig `json:"riskAssessment"`
	TrustedContacts TrustedContactsConfig `json:"trustedContacts"`
	VideoVerification VideoVerificationConfig `json:"videoVerification"`
	Audit AuditConfig `json:"audit"`
	DocumentVerification DocumentVerificationConfig `json:"documentVerification"`
	MultiStepRecovery MultiStepRecoveryConfig `json:"multiStepRecovery"`
	RecoveryCodes RecoveryCodesConfig `json:"recoveryCodes"`
	SecurityQuestions SecurityQuestionsConfig `json:"securityQuestions"`
	SmsVerification SMSVerificationConfig `json:"smsVerification"`
}

type TwoFAStatusDetailResponse struct {
	Enabled bool `json:"enabled"`
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
}

type SMSVerificationConfig struct {
	MaxAttempts int `json:"maxAttempts"`
	MaxSmsPerDay int `json:"maxSmsPerDay"`
	MessageTemplate string `json:"messageTemplate"`
	Provider string `json:"provider"`
	CodeExpiry time.Duration `json:"codeExpiry"`
	CodeLength int `json:"codeLength"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	Enabled bool `json:"enabled"`
}

type VerifySecurityAnswersRequest struct {
	Answers  `json:"answers"`
	SessionId xid.ID `json:"sessionId"`
}

type StateStorageConfig struct {
	StateTtl time.Duration `json:"stateTtl"`
	UseRedis bool `json:"useRedis"`
	RedisAddr string `json:"redisAddr"`
	RedisDb int `json:"redisDb"`
	RedisPassword string `json:"redisPassword"`
}

type BeginLoginResponse struct {
	Timeout time.Duration `json:"timeout"`
	Challenge string `json:"challenge"`
	Options  `json:"options"`
}

type BackupAuthContactResponse struct {
	Id string `json:"id"`
}

type RecoveryConfiguration struct {
	 * `json:",omitempty"`
}

type UpdateFactorRequest struct {
	Metadata  `json:"metadata"`
	Name *string `json:"name"`
	Priority *FactorPriority `json:"priority"`
	Status *FactorStatus `json:"status"`
	 *string `json:",omitempty"`
}

type RiskEngine struct {
	 **repository.MFARepository `json:",omitempty"`
}

type MultiSessionDeleteResponse struct {
	Status string `json:"status"`
}

type ComplianceStatus struct {
	ChecksFailed int `json:"checksFailed"`
	LastChecked time.Time `json:"lastChecked"`
	NextAudit time.Time `json:"nextAudit"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	AppId string `json:"appId"`
	ChecksPassed int `json:"checksPassed"`
	ChecksWarning int `json:"checksWarning"`
	OverallStatus string `json:"overallStatus"`
	Score int `json:"score"`
	Violations int `json:"violations"`
}

type BackupAuthStatsResponse struct {
	Stats  `json:"stats"`
}

type SetUserRole_reqBody struct {
	Role string `json:"role"`
}

type BackupCodesConfig struct {
	Enabled bool `json:"enabled"`
	Format string `json:"format"`
	Length int `json:"length"`
	Allow_reuse bool `json:"allow_reuse"`
	Count int `json:"count"`
}

type ComplianceViolation struct {
	ProfileId string `json:"profileId"`
	ResolvedBy string `json:"resolvedBy"`
	ViolationType string `json:"violationType"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
	ResolvedAt *time.Time `json:"resolvedAt"`
	Severity string `json:"severity"`
	Status string `json:"status"`
	UserId string `json:"userId"`
	Metadata  `json:"metadata"`
}

type userServiceAdapter struct {
	 **user.Service `json:",omitempty"`
}

type ConsentManager struct {
	 *EnterpriseConsentService `json:",omitempty"`
}

type MockSocialAccountRepository struct {
}

type StateStore struct {
	 * `json:",omitempty"`
}

type ResolveViolationRequest struct {
	Notes string `json:"notes"`
	Resolution string `json:"resolution"`
}

type TestSendTemplate_req struct {
	Recipient string `json:"recipient"`
	Variables  `json:"variables"`
}

type IDVerificationResponse struct {
	Verification  `json:"verification"`
}

type AddTeamMember_req struct {
	Member_id xid.ID `json:"member_id"`
	Role string `json:"role"`
}

type MFAStatus struct {
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
	GracePeriod *time.Time `json:"gracePeriod"`
	PolicyActive bool `json:"policyActive"`
	RequiredCount int `json:"requiredCount"`
	TrustedDevice bool `json:"trustedDevice"`
	Enabled bool `json:"enabled"`
}

type MockService struct {
	 * `json:",omitempty"`
}

type UpdatePasskeyRequest struct {
	 *string `json:",omitempty"`
	Name string `json:"name"`
}

type CompleteTrainingRequest struct {
	Score int `json:"score"`
}

type GetSecurityQuestionsRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type SetupSecurityQuestionsResponse struct {
	SetupAt time.Time `json:"setupAt"`
	Count int `json:"count"`
	Message string `json:"message"`
}

type DataProcessingAgreement struct {
	IpAddress string `json:"ipAddress"`
	Status string `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
	Content string `json:"content"`
	SignedBy string `json:"signedBy"`
	Version string `json:"version"`
	AgreementType string `json:"agreementType"`
	Metadata JSONBMap `json:"metadata"`
	SignedByEmail string `json:"signedByEmail"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	SignedByName string `json:"signedByName"`
	SignedByTitle string `json:"signedByTitle"`
	CreatedAt time.Time `json:"createdAt"`
	DigitalSignature string `json:"digitalSignature"`
	EffectiveDate time.Time `json:"effectiveDate"`
	ExpiryDate *time.Time `json:"expiryDate"`
}

type CreateDPARequest struct {
	Version string `json:"version"`
	Content string `json:"content"`
	EffectiveDate time.Time `json:"effectiveDate"`
	ExpiryDate *time.Time `json:"expiryDate"`
	Metadata  `json:"metadata"`
	SignedByName string `json:"signedByName"`
	AgreementType string `json:"agreementType"`
	SignedByEmail string `json:"signedByEmail"`
	SignedByTitle string `json:"signedByTitle"`
}

type TwoFAEnableResponse struct {
	Status string `json:"status"`
	Totp_uri string `json:"totp_uri"`
}

type DocumentVerificationConfig struct {
	StorageProvider string `json:"storageProvider"`
	AcceptedDocuments []string `json:"acceptedDocuments"`
	EncryptAtRest bool `json:"encryptAtRest"`
	RequireBothSides bool `json:"requireBothSides"`
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	StoragePath string `json:"storagePath"`
	Enabled bool `json:"enabled"`
	EncryptionKey string `json:"encryptionKey"`
	MinConfidenceScore float64 `json:"minConfidenceScore"`
	Provider string `json:"provider"`
	RequireManualReview bool `json:"requireManualReview"`
	RequireSelfie bool `json:"requireSelfie"`
}

type DiscoveryService struct {
	 *Config `json:",omitempty"`
}

type ListUsersRequest struct {
	Limit int `json:"limit"`
	Page int `json:"page"`
	Role string `json:"role"`
	Search string `json:"search"`
	Status string `json:"status"`
	User_organization_id *xid.ID `json:"user_organization_id"`
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
}

type UnbanUserRequest struct {
	User_organization_id *xid.ID `json:"user_organization_id"`
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
}

type ImpersonateUserRequest struct {
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
	Duration time.Duration `json:"duration"`
	User_id xid.ID `json:"user_id"`
	User_organization_id *xid.ID `json:"user_organization_id"`
}

type VerificationsResponse struct {
	Count int `json:"count"`
	Verifications  `json:"verifications"`
}

type BackupAuthDocumentResponse struct {
	Id string `json:"id"`
}

type AuthorizeRequest struct {
	Nonce string `json:"nonce"`
	Redirect_uri string `json:"redirect_uri"`
	Response_type string `json:"response_type"`
	Ui_locales string `json:"ui_locales"`
	Acr_values string `json:"acr_values"`
	Code_challenge string `json:"code_challenge"`
	Code_challenge_method string `json:"code_challenge_method"`
	Max_age *int `json:"max_age"`
	Prompt string `json:"prompt"`
	Scope string `json:"scope"`
	State string `json:"state"`
	Client_id string `json:"client_id"`
	Id_token_hint string `json:"id_token_hint"`
	Login_hint string `json:"login_hint"`
}

type CreateUser_reqBody struct {
	Metadata * `json:"metadata,omitempty"`
	Name *string `json:"name,omitempty"`
	Password *string `json:"password,omitempty"`
	Role *string `json:"role,omitempty"`
	Username *string `json:"username,omitempty"`
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
}

type ProvidersConfig struct {
	Dropbox *providers.ProviderConfig `json:"dropbox"`
	Github *providers.ProviderConfig `json:"github"`
	Spotify *providers.ProviderConfig `json:"spotify"`
	Apple *providers.ProviderConfig `json:"apple"`
	Google *providers.ProviderConfig `json:"google"`
	Linkedin *providers.ProviderConfig `json:"linkedin"`
	Bitbucket *providers.ProviderConfig `json:"bitbucket"`
	Line *providers.ProviderConfig `json:"line"`
	Microsoft *providers.ProviderConfig `json:"microsoft"`
	Notion *providers.ProviderConfig `json:"notion"`
	Twitch *providers.ProviderConfig `json:"twitch"`
	Twitter *providers.ProviderConfig `json:"twitter"`
	Discord *providers.ProviderConfig `json:"discord"`
	Facebook *providers.ProviderConfig `json:"facebook"`
	Gitlab *providers.ProviderConfig `json:"gitlab"`
	Reddit *providers.ProviderConfig `json:"reddit"`
	Slack *providers.ProviderConfig `json:"slack"`
}

type VideoVerificationSession struct {
	 *string `json:",omitempty"`
}

type ConsentStatusResponse struct {
	Status string `json:"status"`
}

type GenerateReport_req struct {
	Format string `json:"format"`
	Period string `json:"period"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
}

type CreateVerificationSession_req struct {
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
	Config  `json:"config"`
	Metadata  `json:"metadata"`
}

type BanUserRequest struct {
	User_organization_id *xid.ID `json:"user_organization_id"`
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
	Expires_at *time.Time `json:"expires_at"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
}

type MFABypassResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
}

type EvaluationContext struct {
	 *string `json:",omitempty"`
}

type CreateABTestVariant_req struct {
	Body string `json:"body"`
	Name string `json:"name"`
	Subject string `json:"subject"`
	Weight int `json:"weight"`
}

type UpdatePasskeyResponse struct {
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateAPIKeyResponse struct {
	Message string `json:"message"`
	Api_key *apikey.APIKey `json:"api_key"`
}

type VerifyTrustedContactRequest struct {
	Token string `json:"token"`
}

type TokenIntrospectionRequest struct {
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
}

type LimitResult struct {
	 *bool `json:",omitempty"`
}

type ApproveRecoveryRequest struct {
	Notes string `json:"notes"`
	SessionId xid.ID `json:"sessionId"`
}

type GetChallengeStatusRequest struct {
	 *string `json:",omitempty"`
}

type StepUpRequirementResponse struct {
	Id string `json:"id"`
}

type CookieConsent struct {
	OrganizationId string `json:"organizationId"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	ConsentBannerVersion string `json:"consentBannerVersion"`
	CreatedAt time.Time `json:"createdAt"`
	Functional bool `json:"functional"`
	IpAddress string `json:"ipAddress"`
	Marketing bool `json:"marketing"`
	Analytics bool `json:"analytics"`
	UpdatedAt time.Time `json:"updatedAt"`
	Essential bool `json:"essential"`
	UserAgent string `json:"userAgent"`
	UserId string `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
}

type DocumentCheckConfig struct {
	ExtractData bool `json:"extractData"`
	ValidateDataConsistency bool `json:"validateDataConsistency"`
	ValidateExpiry bool `json:"validateExpiry"`
	Enabled bool `json:"enabled"`
}

type ConsentDecision struct {
	 *[]string `json:",omitempty"`
}

type ComplianceStatusDetailsResponse struct {
	Status string `json:"status"`
}

type ListViolationsFilter struct {
	ViolationType *string `json:"violationType"`
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Severity *string `json:"severity"`
	Status *string `json:"status"`
	UserId *string `json:"userId"`
}

type AuditLog struct {
	 *time.Time `json:",omitempty"`
}

type PrivacySettings struct {
	OrganizationId string `json:"organizationId"`
	UpdatedAt time.Time `json:"updatedAt"`
	AnonymousConsentEnabled bool `json:"anonymousConsentEnabled"`
	DpoEmail string `json:"dpoEmail"`
	AutoDeleteAfterDays int `json:"autoDeleteAfterDays"`
	ConsentRequired bool `json:"consentRequired"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
	CookieConsentEnabled bool `json:"cookieConsentEnabled"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DataExportExpiryHours int `json:"dataExportExpiryHours"`
	DataRetentionDays int `json:"dataRetentionDays"`
	CcpaMode bool `json:"ccpaMode"`
	CreatedAt time.Time `json:"createdAt"`
	DeletionGracePeriodDays int `json:"deletionGracePeriodDays"`
	ExportFormat []string `json:"exportFormat"`
	Id xid.ID `json:"id"`
	Metadata JSONBMap `json:"metadata"`
	RequireAdminApprovalForDeletion bool `json:"requireAdminApprovalForDeletion"`
	RequireExplicitConsent bool `json:"requireExplicitConsent"`
	AllowDataPortability bool `json:"allowDataPortability"`
	GdprMode bool `json:"gdprMode"`
}

type ConsentPolicy struct {
	Renewable bool `json:"renewable"`
	Content string `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string `json:"createdBy"`
	Metadata JSONBMap `json:"metadata"`
	Name string `json:"name"`
	OrganizationId string `json:"organizationId"`
	PublishedAt *time.Time `json:"publishedAt"`
	Id xid.ID `json:"id"`
	UpdatedAt time.Time `json:"updatedAt"`
	ValidityPeriod *int `json:"validityPeriod"`
	Required bool `json:"required"`
	ConsentType string `json:"consentType"`
	Version string `json:"version"`
	Active bool `json:"active"`
	Description string `json:"description"`
}

type OIDCLoginResponse struct {
	Nonce string `json:"nonce"`
	ProviderId string `json:"providerId"`
	State string `json:"state"`
	AuthUrl string `json:"authUrl"`
}

type SetActive_body struct {
	Id string `json:"id"`
}

type EnableRequest struct {
	 *string `json:",omitempty"`
}

type CreateEvidence_req struct {
	ControlId string `json:"controlId"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
}

type OnfidoConfig struct {
	ApiToken string `json:"apiToken"`
	DocumentCheck DocumentCheckConfig `json:"documentCheck"`
	Enabled bool `json:"enabled"`
	IncludeFacialReport bool `json:"includeFacialReport"`
	IncludeWatchlistReport bool `json:"includeWatchlistReport"`
	WebhookToken string `json:"webhookToken"`
	FacialCheck FacialCheckConfig `json:"facialCheck"`
	IncludeDocumentReport bool `json:"includeDocumentReport"`
	Region string `json:"region"`
	WorkflowId string `json:"workflowId"`
}

type ResetUserMFAResponse struct {
	DevicesRevoked int `json:"devicesRevoked"`
	FactorsReset int `json:"factorsReset"`
	Message string `json:"message"`
	Success bool `json:"success"`
}

type WebAuthnFactorAdapter struct {
	 **passkey.Service `json:",omitempty"`
}

type DeviceInfo struct {
	DeviceId string `json:"deviceId"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
}

type ChallengeRequest struct {
	Metadata  `json:"metadata"`
	UserId xid.ID `json:"userId"`
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
}

type AppServiceAdapter struct {
	 * `json:",omitempty"`
}

type ComplianceTemplateResponse struct {
	Standard string `json:"standard"`
}

type FinishRegisterResponse struct {
	CredentialId string `json:"credentialId"`
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type RejectRecoveryResponse struct {
	Message string `json:"message"`
	Reason string `json:"reason"`
	Rejected bool `json:"rejected"`
	RejectedAt time.Time `json:"rejectedAt"`
	SessionId xid.ID `json:"sessionId"`
}

type BackupAuthVideoResponse struct {
	Session_id string `json:"session_id"`
}

type JumioProvider struct {
	 *JumioConfig `json:",omitempty"`
}

type KeyStore struct {
	 *time.Duration `json:",omitempty"`
}

type ListFactorsResponse struct {
	Count int `json:"count"`
	Factors []Factor `json:"factors"`
}

type StepUpAttempt struct {
	User_id string `json:"user_id"`
	Failure_reason string `json:"failure_reason"`
	Ip string `json:"ip"`
	Method VerificationMethod `json:"method"`
	Org_id string `json:"org_id"`
	Requirement_id string `json:"requirement_id"`
	User_agent string `json:"user_agent"`
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
	Success bool `json:"success"`
}

type BeginRegisterRequest struct {
	AuthenticatorType string `json:"authenticatorType"`
	Name string `json:"name"`
	RequireResidentKey bool `json:"requireResidentKey"`
	UserId string `json:"userId"`
	UserVerification string `json:"userVerification"`
}

type TwoFARequiredResponse struct {
	Require_twofa bool `json:"require_twofa"`
	User *user.User `json:"user"`
	Device_id string `json:"device_id"`
}

type ReviewDocumentRequest struct {
	Approved bool `json:"approved"`
	DocumentId xid.ID `json:"documentId"`
	Notes string `json:"notes"`
	RejectionReason string `json:"rejectionReason"`
}

type AdminPolicyRequest struct {
	AllowedTypes []string `json:"allowedTypes"`
	Enabled bool `json:"enabled"`
	GracePeriod int `json:"gracePeriod"`
	RequiredFactors int `json:"requiredFactors"`
}

type SignInRequest struct {
	Scopes []string `json:"scopes"`
	Provider string `json:"provider"`
	RedirectUrl string `json:"redirectUrl"`
}

type ComplianceReport struct {
	ProfileId string `json:"profileId"`
	ReportType string `json:"reportType"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	FileUrl string `json:"fileUrl"`
	Format string `json:"format"`
	Period string `json:"period"`
	Standard ComplianceStandard `json:"standard"`
	Summary  `json:"summary"`
	AppId string `json:"appId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FileSize int64 `json:"fileSize"`
	GeneratedBy string `json:"generatedBy"`
	Id string `json:"id"`
}

type ComplianceTraining struct {
	AppId string `json:"appId"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	ProfileId string `json:"profileId"`
	Score int `json:"score"`
	Status string `json:"status"`
	TrainingType string `json:"trainingType"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Metadata  `json:"metadata"`
	Standard ComplianceStandard `json:"standard"`
	UserId string `json:"userId"`
}

type RemoveTrustedContactRequest struct {
	ContactId xid.ID `json:"contactId"`
}

type VerifyCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type ConsentCookieResponse struct {
	Preferences  `json:"preferences"`
}

type VerifyResponse struct {
	Success bool `json:"success"`
	Verification_id string `json:"verification_id"`
	Device_remembered bool `json:"device_remembered"`
	Error string `json:"error"`
	Expires_at time.Time `json:"expires_at"`
	Metadata  `json:"metadata"`
	Security_level SecurityLevel `json:"security_level"`
}

type CreatePolicy_req struct {
	Title string `json:"title"`
	Version string `json:"version"`
	Content string `json:"content"`
	PolicyType string `json:"policyType"`
	Standard ComplianceStandard `json:"standard"`
}

type Email struct {
	 *string `json:",omitempty"`
}

type CreateAPIKey_reqBody struct {
	Allowed_ips *[]string `json:"allowed_ips,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata * `json:"metadata,omitempty"`
	Name string `json:"name"`
	Permissions * `json:"permissions,omitempty"`
	Rate_limit *int `json:"rate_limit,omitempty"`
	Scopes []string `json:"scopes"`
}

type AutoCleanupConfig struct {
	Enabled bool `json:"enabled"`
	Interval time.Duration `json:"interval"`
}

type ClientSummary struct {
	ApplicationType string `json:"applicationType"`
	ClientID string `json:"clientID"`
	CreatedAt string `json:"createdAt"`
	IsOrgLevel bool `json:"isOrgLevel"`
	Name string `json:"name"`
}

type Factor struct {
	Status FactorStatus `json:"status"`
	UserId xid.ID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Type FactorType `json:"type"`
	UpdatedAt time.Time `json:"updatedAt"`
	VerifiedAt *time.Time `json:"verifiedAt"`
	- string `json:"-"`
	Id xid.ID `json:"id"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	Priority FactorPriority `json:"priority"`
}

type BackupCodeFactorAdapter struct {
	 **twofa.Service `json:",omitempty"`
}

type NoOpDocumentProvider struct {
}

type NoOpNotificationProvider struct {
}

type ConsentReportResponse struct {
	Id string `json:"id"`
}

type FactorAdapterRegistry struct {
	 * `json:",omitempty"`
}

type StepUpPoliciesResponse struct {
	Policies []* `json:"policies"`
}

type Enable_body struct {
	Method string `json:"method"`
	User_id string `json:"user_id"`
}

type UserAdapter struct {
	 *[]webauthn.Credential `json:",omitempty"`
}

type IDVerificationStatusResponse struct {
	Status  `json:"status"`
}

type EmailVerificationConfig struct {
	MaxAttempts int `json:"maxAttempts"`
	RequireEmailProof bool `json:"requireEmailProof"`
	CodeExpiry time.Duration `json:"codeExpiry"`
	CodeLength int `json:"codeLength"`
	EmailTemplate string `json:"emailTemplate"`
	Enabled bool `json:"enabled"`
	FromAddress string `json:"fromAddress"`
	FromName string `json:"fromName"`
}

type GetRecoveryConfigResponse struct {
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
}

type VerificationResult struct {
	 *int `json:",omitempty"`
}

type VerificationListResponse struct {
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Total int `json:"total"`
	Verifications []*schema.IdentityVerification `json:"verifications"`
}

type TrustedContactInfo struct {
	Active bool `json:"active"`
	Email string `json:"email"`
	Id xid.ID `json:"id"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
	Verified bool `json:"verified"`
	VerifiedAt *time.Time `json:"verifiedAt"`
}

type SAMLLoginResponse struct {
	ProviderId string `json:"providerId"`
	RedirectUrl string `json:"redirectUrl"`
	RequestId string `json:"requestId"`
}

type ComplianceTemplate struct {
	SessionMaxAge int `json:"sessionMaxAge"`
	Standard ComplianceStandard `json:"standard"`
	AuditFrequencyDays int `json:"auditFrequencyDays"`
	DataResidency string `json:"dataResidency"`
	Description string `json:"description"`
	MfaRequired bool `json:"mfaRequired"`
	Name string `json:"name"`
	PasswordMinLength int `json:"passwordMinLength"`
	RequiredPolicies []string `json:"requiredPolicies"`
	RequiredTraining []string `json:"requiredTraining"`
	RetentionDays int `json:"retentionDays"`
}

type GetSecurityQuestionsResponse struct {
	Questions []SecurityQuestionInfo `json:"questions"`
}

type MFAConfigResponse struct {
	Allowed_factor_types []string `json:"allowed_factor_types"`
	Enabled bool `json:"enabled"`
	Required_factor_count int `json:"required_factor_count"`
}

type ConsentsResponse struct {
	Consents  `json:"consents"`
	Count int `json:"count"`
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type TokenResponse struct {
	Token_type string `json:"token_type"`
	Access_token string `json:"access_token"`
	Expires_in int `json:"expires_in"`
	Id_token string `json:"id_token"`
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
}

type NotificationResponse struct {
	Notification  `json:"notification"`
}

type EmailProviderConfig struct {
	From string `json:"from"`
	From_name string `json:"from_name"`
	Provider string `json:"provider"`
	Reply_to string `json:"reply_to"`
	Config  `json:"config"`
}

type SignUpResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

type SecurityQuestionInfo struct {
	QuestionText string `json:"questionText"`
	Id xid.ID `json:"id"`
	IsCustom bool `json:"isCustom"`
	QuestionId int `json:"questionId"`
}

type ConsentSettingsResponse struct {
	Settings  `json:"settings"`
}

type StripeIdentityProvider struct {
	 *bool `json:",omitempty"`
}

type DiscoveryResponse struct {
	Subject_types_supported []string `json:"subject_types_supported"`
	Token_endpoint string `json:"token_endpoint"`
	Code_challenge_methods_supported []string `json:"code_challenge_methods_supported"`
	Id_token_signing_alg_values_supported []string `json:"id_token_signing_alg_values_supported"`
	Jwks_uri string `json:"jwks_uri"`
	Registration_endpoint string `json:"registration_endpoint"`
	Request_parameter_supported bool `json:"request_parameter_supported"`
	Require_request_uri_registration bool `json:"require_request_uri_registration"`
	Revocation_endpoint string `json:"revocation_endpoint"`
	Introspection_endpoint string `json:"introspection_endpoint"`
	Issuer string `json:"issuer"`
	Token_endpoint_auth_methods_supported []string `json:"token_endpoint_auth_methods_supported"`
	Claims_supported []string `json:"claims_supported"`
	Grant_types_supported []string `json:"grant_types_supported"`
	Response_types_supported []string `json:"response_types_supported"`
	Revocation_endpoint_auth_methods_supported []string `json:"revocation_endpoint_auth_methods_supported"`
	Scopes_supported []string `json:"scopes_supported"`
	Userinfo_endpoint string `json:"userinfo_endpoint"`
	Authorization_endpoint string `json:"authorization_endpoint"`
	Claims_parameter_supported bool `json:"claims_parameter_supported"`
	Introspection_endpoint_auth_methods_supported []string `json:"introspection_endpoint_auth_methods_supported"`
	Request_uri_parameter_supported bool `json:"request_uri_parameter_supported"`
	Response_modes_supported []string `json:"response_modes_supported"`
}

type AdminBypassRequest struct {
	Duration int `json:"duration"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
}

type ChallengeStatusResponse struct {
	Status string `json:"status"`
	CompletedAt *time.Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRemaining int `json:"factorsRemaining"`
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
	SessionId xid.ID `json:"sessionId"`
}

type EmailServiceAdapter struct {
	 **notification.Service `json:",omitempty"`
}

type VerificationResponse struct {
	Verification *schema.IdentityVerification `json:"verification"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	- xid.ID `json:"-"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Password string `json:"password"`
	App_id xid.ID `json:"app_id"`
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
	Role string `json:"role"`
	User_organization_id *xid.ID `json:"user_organization_id"`
}

type ConsentDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type CreateSessionHTTPRequest struct {
	Config  `json:"config"`
	Metadata  `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
}

type CreateVerificationRequest struct {
	 *xid.ID `json:",omitempty"`
}

type SendCodeResponse struct {
	Dev_code string `json:"dev_code"`
	Status string `json:"status"`
}

type ConnectionResponse struct {
	Connection *schema.SocialAccount `json:"connection"`
}

type TOTPSecret struct {
	 *string `json:",omitempty"`
}

type UpdateProvider_req struct {
	Config  `json:"config"`
	IsActive bool `json:"isActive"`
	IsDefault bool `json:"isDefault"`
}

type NotificationTemplateResponse struct {
	Template  `json:"template"`
}

type OnfidoProvider struct {
	 *OnfidoConfig `json:",omitempty"`
}

type NoOpVideoProvider struct {
}

type Service struct {
	 *Repository `json:",omitempty"`
}

type MemoryStateStore struct {
	 * `json:",omitempty"`
}

type OTPSentResponse struct {
	Status string `json:"status"`
	Code string `json:"code"`
}

type CallbackDataResponse struct {
	Action string `json:"action"`
	IsNewUser bool `json:"isNewUser"`
	User *schema.User `json:"user"`
}

type StepUpVerificationsResponse struct {
	Verifications []* `json:"verifications"`
}

type ComplianceCheck struct {
	Result  `json:"result"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	LastCheckedAt time.Time `json:"lastCheckedAt"`
	NextCheckAt time.Time `json:"nextCheckAt"`
	ProfileId string `json:"profileId"`
	Status string `json:"status"`
	CheckType string `json:"checkType"`
	Evidence []string `json:"evidence"`
	Id string `json:"id"`
}

type TrustedContact struct {
	 **time.Time `json:",omitempty"`
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

type RiskAssessment struct {
	Factors []string `json:"factors"`
	Level RiskLevel `json:"level"`
	Metadata  `json:"metadata"`
	Recommended []FactorType `json:"recommended"`
	Score float64 `json:"score"`
}

type WebhookConfig struct {
	Notify_on_created bool `json:"notify_on_created"`
	Notify_on_deleted bool `json:"notify_on_deleted"`
	Notify_on_expiring bool `json:"notify_on_expiring"`
	Notify_on_rate_limit bool `json:"notify_on_rate_limit"`
	Notify_on_rotated bool `json:"notify_on_rotated"`
	Webhook_urls []string `json:"webhook_urls"`
	Enabled bool `json:"enabled"`
	Expiry_warning_days int `json:"expiry_warning_days"`
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

type FactorVerificationRequest struct {
	Code string `json:"code"`
	Data  `json:"data"`
	FactorId xid.ID `json:"factorId"`
}

type GetStatusRequest struct {
	 *string `json:",omitempty"`
}

type mockSessionService struct {
	 **bun.DB `json:",omitempty"`
}

type IPWhitelistConfig struct {
	Enabled bool `json:"enabled"`
	Strict_mode bool `json:"strict_mode"`
}

type ScheduleVideoSessionResponse struct {
	Instructions string `json:"instructions"`
	JoinUrl string `json:"joinUrl"`
	Message string `json:"message"`
	ScheduledAt time.Time `json:"scheduledAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type ConsentSummary struct {
	UserId string `json:"userId"`
	ConsentsByType  `json:"consentsByType"`
	HasPendingDeletion bool `json:"hasPendingDeletion"`
	HasPendingExport bool `json:"hasPendingExport"`
	LastConsentUpdate *time.Time `json:"lastConsentUpdate"`
	PendingRenewals int `json:"pendingRenewals"`
	ExpiredConsents int `json:"expiredConsents"`
	GrantedConsents int `json:"grantedConsents"`
	OrganizationId string `json:"organizationId"`
	RevokedConsents int `json:"revokedConsents"`
	TotalConsents int `json:"totalConsents"`
}

type FactorInfo struct {
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Type FactorType `json:"type"`
	FactorId xid.ID `json:"factorId"`
}

type HealthCheckResponse struct {
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	Healthy bool `json:"healthy"`
	Message string `json:"message"`
	ProvidersStatus  `json:"providersStatus"`
	Version string `json:"version"`
}

type CreatePolicyRequest struct {
	Name string `json:"name"`
	Renewable bool `json:"renewable"`
	Version string `json:"version"`
	Metadata  `json:"metadata"`
	Required bool `json:"required"`
	ValidityPeriod *int `json:"validityPeriod"`
	ConsentType string `json:"consentType"`
	Content string `json:"content"`
	Description string `json:"description"`
}

type ClientAuthenticator struct {
	 **repo.OAuthClientRepository `json:",omitempty"`
}

type MultiSessionSetActiveResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
}

type ComplianceReportResponse struct {
	Id string `json:"id"`
}

type TokenIntrospectionResponse struct {
	Username string `json:"username"`
	Aud []string `json:"aud"`
	Exp int64 `json:"exp"`
	Jti string `json:"jti"`
	Scope string `json:"scope"`
	Sub string `json:"sub"`
	Token_type string `json:"token_type"`
	Active bool `json:"active"`
	Client_id string `json:"client_id"`
	Iat int64 `json:"iat"`
	Iss string `json:"iss"`
	Nbf int64 `json:"nbf"`
}

type VerificationRequest struct {
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data  `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
}

type RateLimit struct {
	 *time.Duration `json:",omitempty"`
}

type SessionsResponse struct {
	Sessions  `json:"sessions"`
}

type ComplianceChecksResponse struct {
	Checks []* `json:"checks"`
}

type StartVideoSessionRequest struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type GetDocumentVerificationResponse struct {
	ConfidenceScore float64 `json:"confidenceScore"`
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
	RejectionReason string `json:"rejectionReason"`
	Status string `json:"status"`
	VerifiedAt *time.Time `json:"verifiedAt"`
}

type DiscoverProviderRequest struct {
	Email string `json:"email"`
}

type StepUpRequirement struct {
	User_id string `json:"user_id"`
	Id string `json:"id"`
	Required_level SecurityLevel `json:"required_level"`
	Rule_name string `json:"rule_name"`
	Currency string `json:"currency"`
	Fulfilled_at *time.Time `json:"fulfilled_at"`
	Metadata  `json:"metadata"`
	Org_id string `json:"org_id"`
	Resource_type string `json:"resource_type"`
	Route string `json:"route"`
	User_agent string `json:"user_agent"`
	Created_at time.Time `json:"created_at"`
	Current_level SecurityLevel `json:"current_level"`
	Ip string `json:"ip"`
	Method string `json:"method"`
	Reason string `json:"reason"`
	Session_id string `json:"session_id"`
	Amount float64 `json:"amount"`
	Challenge_token string `json:"challenge_token"`
	Expires_at time.Time `json:"expires_at"`
	Resource_action string `json:"resource_action"`
	Risk_score float64 `json:"risk_score"`
	Status string `json:"status"`
}

type RequirementsResponse struct {
	Count int `json:"count"`
	Requirements  `json:"requirements"`
}

type CodesResponse struct {
	Codes []string `json:"codes"`
}

type ListPoliciesFilter struct {
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
	PolicyType *string `json:"policyType"`
}

type Session struct {
	 *time.Time `json:",omitempty"`
}

type ConsentReport struct {
	ReportPeriodEnd time.Time `json:"reportPeriodEnd"`
	TotalUsers int `json:"totalUsers"`
	UsersWithConsent int `json:"usersWithConsent"`
	CompletedDeletions int `json:"completedDeletions"`
	ConsentRate float64 `json:"consentRate"`
	ConsentsByType  `json:"consentsByType"`
	DataExportsThisPeriod int `json:"dataExportsThisPeriod"`
	DpasActive int `json:"dpasActive"`
	DpasExpiringSoon int `json:"dpasExpiringSoon"`
	PendingDeletions int `json:"pendingDeletions"`
	ReportPeriodStart time.Time `json:"reportPeriodStart"`
	OrganizationId string `json:"organizationId"`
}

type ConsentStats struct {
	TotalConsents int `json:"totalConsents"`
	Type string `json:"type"`
	AverageLifetime int `json:"averageLifetime"`
	ExpiredCount int `json:"expiredCount"`
	GrantRate float64 `json:"grantRate"`
	GrantedCount int `json:"grantedCount"`
	RevokedCount int `json:"revokedCount"`
}

type ComplianceTrainingResponse struct {
	Id string `json:"id"`
}

type CompleteVideoSessionResponse struct {
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	Result string `json:"result"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type FacialCheckConfig struct {
	Variant string `json:"variant"`
	Enabled bool `json:"enabled"`
	MotionCapture bool `json:"motionCapture"`
}

type UserVerificationStatusResponse struct {
	Status *schema.UserVerificationStatus `json:"status"`
}

type JumioConfig struct {
	Enabled bool `json:"enabled"`
	EnabledCountries []string `json:"enabledCountries"`
	PresetId string `json:"presetId"`
	ApiSecret string `json:"apiSecret"`
	ApiToken string `json:"apiToken"`
	CallbackUrl string `json:"callbackUrl"`
	EnableAMLScreening bool `json:"enableAMLScreening"`
	EnabledDocumentTypes []string `json:"enabledDocumentTypes"`
	VerificationType string `json:"verificationType"`
	DataCenter string `json:"dataCenter"`
	EnableExtraction bool `json:"enableExtraction"`
	EnableLiveness bool `json:"enableLiveness"`
}

type ComplianceProfileResponse struct {
	Id string `json:"id"`
}

type RecoveryCodeUsage struct {
	 *string `json:",omitempty"`
}

type AppHandler struct {
	 **coreapp.ServiceImpl `json:",omitempty"`
}

type MFAPolicyResponse struct {
	AllowedFactorTypes []string `json:"allowedFactorTypes"`
	AppId xid.ID `json:"appId"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
	OrganizationId *xid.ID `json:"organizationId"`
	RequiredFactorCount int `json:"requiredFactorCount"`
}

type stateEntry struct {
	 *time.Time `json:",omitempty"`
}

type ProvidersResponse struct {
	Providers []string `json:"providers"`
}

type StepUpPolicyResponse struct {
	Id string `json:"id"`
}

type ListPasskeysRequest struct {
	 *string `json:",omitempty"`
}

type CreateSessionRequest struct {
	 * `json:",omitempty"`
}

type GetFactorRequest struct {
	 *string `json:",omitempty"`
}

type RegisterProviderRequest struct {
	SamlIssuer string `json:"samlIssuer"`
	ProviderId string `json:"providerId"`
	SamlCert string `json:"samlCert"`
	Type string `json:"type"`
	AttributeMapping  `json:"attributeMapping"`
	Domain string `json:"domain"`
	OidcClientID string `json:"oidcClientID"`
	OidcClientSecret string `json:"oidcClientSecret"`
	OidcIssuer string `json:"oidcIssuer"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
	SamlEntryPoint string `json:"samlEntryPoint"`
}

type RequestTrustedContactVerificationResponse struct {
	ContactId xid.ID `json:"contactId"`
	ContactName string `json:"contactName"`
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	NotifiedAt time.Time `json:"notifiedAt"`
}

type RequestReverification_req struct {
	Reason string `json:"reason"`
}

type TOTPFactorAdapter struct {
	 **twofa.Service `json:",omitempty"`
}

type TwoFASendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type BackupAuthStatusResponse struct {
	Status string `json:"status"`
}

type StepUpStatusResponse struct {
	Status string `json:"status"`
}

type StepUpVerification struct {
	Metadata  `json:"metadata"`
	Method VerificationMethod `json:"method"`
	Session_id string `json:"session_id"`
	User_agent string `json:"user_agent"`
	Verified_at time.Time `json:"verified_at"`
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Device_id string `json:"device_id"`
	Expires_at time.Time `json:"expires_at"`
	Org_id string `json:"org_id"`
	Rule_name string `json:"rule_name"`
	Reason string `json:"reason"`
	Security_level SecurityLevel `json:"security_level"`
	User_id string `json:"user_id"`
}

type SMSFactorAdapter struct {
	 **notificationPlugin.Adapter `json:",omitempty"`
}

type InvitationResponse struct {
	Invitation *organization.Invitation `json:"invitation"`
	Message string `json:"message"`
}

type AutomatedChecksConfig struct {
	MfaCoverage bool `json:"mfaCoverage"`
	PasswordPolicy bool `json:"passwordPolicy"`
	SuspiciousActivity bool `json:"suspiciousActivity"`
	AccessReview bool `json:"accessReview"`
	CheckInterval time.Duration `json:"checkInterval"`
	DataRetention bool `json:"dataRetention"`
	SessionPolicy bool `json:"sessionPolicy"`
	Enabled bool `json:"enabled"`
	InactiveUsers bool `json:"inactiveUsers"`
}

type ComplianceReportsResponse struct {
	Reports []* `json:"reports"`
}

type ListRecoverySessionsRequest struct {
	PageSize int `json:"pageSize"`
	RequiresReview bool `json:"requiresReview"`
	Status RecoveryStatus `json:"status"`
	OrganizationId string `json:"organizationId"`
	Page int `json:"page"`
}

type SendVerificationCodeRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
	Target string `json:"target"`
}

type EmailFactorAdapter struct {
	 **notificationPlugin.Adapter `json:",omitempty"`
}

type ListTrainingFilter struct {
	UserId *string `json:"userId"`
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	TrainingType *string `json:"trainingType"`
}

type CompleteVideoSessionRequest struct {
	LivenessPassed bool `json:"livenessPassed"`
	LivenessScore float64 `json:"livenessScore"`
	Notes string `json:"notes"`
	VerificationResult string `json:"verificationResult"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type ConsentTypeStatus struct {
	Granted bool `json:"granted"`
	GrantedAt time.Time `json:"grantedAt"`
	NeedsRenewal bool `json:"needsRenewal"`
	Type string `json:"type"`
	Version string `json:"version"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type ClientRegistrationRequest struct {
	Require_pkce bool `json:"require_pkce"`
	Client_name string `json:"client_name"`
	Grant_types []string `json:"grant_types"`
	Response_types []string `json:"response_types"`
	Scope string `json:"scope"`
	Application_type string `json:"application_type"`
	Contacts []string `json:"contacts"`
	Logo_uri string `json:"logo_uri"`
	Policy_uri string `json:"policy_uri"`
	Require_consent bool `json:"require_consent"`
	Trusted_client bool `json:"trusted_client"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Tos_uri string `json:"tos_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
}

type StepUpEvaluationResponse struct {
	Reason string `json:"reason"`
	Required bool `json:"required"`
}

type UpdatePolicy_req struct {
	Content *string `json:"content"`
	Status *string `json:"status"`
	Title *string `json:"title"`
	Version *string `json:"version"`
}

type UpdateProfileRequest struct {
	MfaRequired *bool `json:"mfaRequired"`
	Name *string `json:"name"`
	RetentionDays *int `json:"retentionDays"`
	Status *string `json:"status"`
}

type RecoverySession struct {
	 *int `json:",omitempty"`
}

type mockProvider struct {
	 *error `json:",omitempty"`
}

type SendResponse struct {
	Dev_otp string `json:"dev_otp"`
	Status string `json:"status"`
}

type ChannelsResponse struct {
	Channels  `json:"channels"`
	Count int `json:"count"`
}

type CreateTemplateVersion_req struct {
	Changes string `json:"changes"`
}

type DocumentVerification struct {
	 *float64 `json:",omitempty"`
}

type AddMember_req struct {
	Role string `json:"role"`
	User_id string `json:"user_id"`
}

type RegistrationService struct {
	 *Config `json:",omitempty"`
}

type RouteRule struct {
	Security_level SecurityLevel `json:"security_level"`
	Description string `json:"description"`
	Method string `json:"method"`
	Org_id string `json:"org_id"`
	Pattern string `json:"pattern"`
}

type SendOTP_body struct {
	User_id string `json:"user_id"`
}

type ComplianceEvidenceResponse struct {
	Id string `json:"id"`
}

type CompliancePolicy struct {
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	Title string `json:"title"`
	Version string `json:"version"`
	Standard ComplianceStandard `json:"standard"`
	Status string `json:"status"`
	AppId string `json:"appId"`
	ApprovedAt *time.Time `json:"approvedAt"`
	Content string `json:"content"`
	ProfileId string `json:"profileId"`
	ReviewDate time.Time `json:"reviewDate"`
	UpdatedAt time.Time `json:"updatedAt"`
	PolicyType string `json:"policyType"`
	ApprovedBy string `json:"approvedBy"`
	EffectiveDate time.Time `json:"effectiveDate"`
	Metadata  `json:"metadata"`
}

type GetDocumentVerificationRequest struct {
	DocumentId xid.ID `json:"documentId"`
}

type ConsentExportFileResponse struct {
	Data []byte `json:"data"`
	Content_type string `json:"content_type"`
}

type CheckSubResult struct {
	 *string `json:",omitempty"`
}

type AdminHandler struct {
	 **RegistrationService `json:",omitempty"`
}

type LinkAccountRequest struct {
	Scopes []string `json:"scopes"`
	Provider string `json:"provider"`
}

type SSOAuthResponse struct {
	User *user.User `json:"user"`
	Session *session.Session `json:"session"`
	Token string `json:"token"`
}

type Middleware struct {
	 **Service `json:",omitempty"`
}

type RunCheckRequest struct {
	CheckType string `json:"checkType"`
}

type BackupAuthConfigResponse struct {
	Config  `json:"config"`
}

type SetupSecurityQuestionsRequest struct {
	Questions []SetupSecurityQuestionRequest `json:"questions"`
}

type GenerateReportRequest struct {
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
	Format string `json:"format"`
	Period string `json:"period"`
}

type RetentionConfig struct {
	ArchiveBeforePurge bool `json:"archiveBeforePurge"`
	ArchivePath string `json:"archivePath"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	PurgeSchedule string `json:"purgeSchedule"`
}

type MockAppService struct {
}

type DeclareABTestWinner_req struct {
	AbTestGroup string `json:"abTestGroup"`
	WinnerId string `json:"winnerId"`
}

type PhoneVerifyResponse struct {
	Token string `json:"token"`
	User *user.User `json:"user"`
	Session *session.Session `json:"session"`
}

type TemplateEngine struct {
	 *template.FuncMap `json:",omitempty"`
}

type AdminBlockUser_req struct {
	Reason string `json:"reason"`
}

type DashboardExtension struct {
	 *string `json:",omitempty"`
}

type AdaptiveMFAConfig struct {
	Factor_location_change bool `json:"factor_location_change"`
	Factor_new_device bool `json:"factor_new_device"`
	Location_change_risk float64 `json:"location_change_risk"`
	Risk_threshold float64 `json:"risk_threshold"`
	Velocity_risk float64 `json:"velocity_risk"`
	Factor_ip_reputation bool `json:"factor_ip_reputation"`
	Factor_velocity bool `json:"factor_velocity"`
	New_device_risk float64 `json:"new_device_risk"`
	Require_step_up_threshold float64 `json:"require_step_up_threshold"`
	Enabled bool `json:"enabled"`
}

type ImpersonationContext struct {
	Target_user_id *xid.ID `json:"target_user_id"`
	Impersonation_id *xid.ID `json:"impersonation_id"`
	Impersonator_id *xid.ID `json:"impersonator_id"`
	Indicator_message string `json:"indicator_message"`
	Is_impersonating bool `json:"is_impersonating"`
}

type ListSessionsRequest struct {
	User_id *xid.ID `json:"user_id"`
	User_organization_id *xid.ID `json:"user_organization_id"`
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
	Limit int `json:"limit"`
	Page int `json:"page"`
}

type TrustedDevicesConfig struct {
	Default_expiry_days int `json:"default_expiry_days"`
	Enabled bool `json:"enabled"`
	Max_devices_per_user int `json:"max_devices_per_user"`
	Max_expiry_days int `json:"max_expiry_days"`
}

type BaseFactorAdapter struct {
	 *bool `json:",omitempty"`
}

type UserServiceAdapter struct {
	 *user.ServiceInterface `json:",omitempty"`
}

type BeginRegisterResponse struct {
	Challenge string `json:"challenge"`
	Options  `json:"options"`
	Timeout time.Duration `json:"timeout"`
	UserId string `json:"userId"`
}

type DataDeletionRequestInput struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type ProviderSessionRequest struct {
	 * `json:",omitempty"`
}

type JWKSService struct {
	 **KeyStore `json:",omitempty"`
}

type MockUserService struct {
}

type ListEvidenceFilter struct {
	AppId *string `json:"appId"`
	ControlId *string `json:"controlId"`
	EvidenceType *string `json:"evidenceType"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
}

type CreateProfileFromTemplate_req struct {
	Standard ComplianceStandard `json:"standard"`
}

type StartRecoveryRequest struct {
	DeviceId string `json:"deviceId"`
	Email string `json:"email"`
	PreferredMethod RecoveryMethod `json:"preferredMethod"`
	UserId string `json:"userId"`
}

type CreateConsentRequest struct {
	ConsentType string `json:"consentType"`
	ExpiresIn *int `json:"expiresIn"`
	Granted bool `json:"granted"`
	Metadata  `json:"metadata"`
	Purpose string `json:"purpose"`
	UserId string `json:"userId"`
	Version string `json:"version"`
}

type FactorEnrollmentResponse struct {
	FactorId xid.ID `json:"factorId"`
	ProvisioningData  `json:"provisioningData"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
}

type MultiSessionErrorResponse struct {
	Error string `json:"error"`
}

type ComplianceTemplatesResponse struct {
	Templates []* `json:"templates"`
}

type GetPasskeyRequest struct {
	 *string `json:",omitempty"`
}

type SuccessResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type WebAuthnConfig struct {
	Attestation_preference string `json:"attestation_preference"`
	Authenticator_selection  `json:"authenticator_selection"`
	Enabled bool `json:"enabled"`
	Rp_display_name string `json:"rp_display_name"`
	Rp_id string `json:"rp_id"`
	Rp_origins []string `json:"rp_origins"`
	Timeout int `json:"timeout"`
}

type ListProfilesFilter struct {
	AppId *string `json:"appId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
}

type UpdateRecoveryConfigRequest struct {
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
}

type ConsentDashboardConfig struct {
	ShowDataExport bool `json:"showDataExport"`
	ShowPolicies bool `json:"showPolicies"`
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
	ShowAuditLog bool `json:"showAuditLog"`
	ShowConsentHistory bool `json:"showConsentHistory"`
	ShowCookiePreferences bool `json:"showCookiePreferences"`
	ShowDataDeletion bool `json:"showDataDeletion"`
}

type OIDCLoginRequest struct {
	RedirectUri string `json:"redirectUri"`
	Scope string `json:"scope"`
	State string `json:"state"`
	Nonce string `json:"nonce"`
}

type ImpersonationMiddleware struct {
	 *Config `json:",omitempty"`
}

type VerifyRecoveryCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type TrustedContactsConfig struct {
	RequireVerification bool `json:"requireVerification"`
	RequiredToRecover int `json:"requiredToRecover"`
	AllowPhoneContacts bool `json:"allowPhoneContacts"`
	MinimumContacts int `json:"minimumContacts"`
	VerificationExpiry time.Duration `json:"verificationExpiry"`
	AllowEmailContacts bool `json:"allowEmailContacts"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	Enabled bool `json:"enabled"`
	MaxNotificationsPerDay int `json:"maxNotificationsPerDay"`
	MaximumContacts int `json:"maximumContacts"`
}

type SetUserRoleRequest struct {
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
	Role string `json:"role"`
	User_id xid.ID `json:"user_id"`
	User_organization_id *xid.ID `json:"user_organization_id"`
}

type ProviderListResponse struct {
	Providers []ProviderInfo `json:"providers"`
	Total int `json:"total"`
}

type CreateProfileFromTemplateRequest struct {
	Standard ComplianceStandard `json:"standard"`
}

type ContinueRecoveryResponse struct {
	CurrentStep int `json:"currentStep"`
	Data  `json:"data"`
	ExpiresAt time.Time `json:"expiresAt"`
	Instructions string `json:"instructions"`
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
	TotalSteps int `json:"totalSteps"`
}

type ComplianceViolationResponse struct {
	Id string `json:"id"`
}

type BunRepository struct {
	 **bun.DB `json:",omitempty"`
}

type RiskAssessmentConfig struct {
	RequireReviewAbove float64 `json:"requireReviewAbove"`
	Enabled bool `json:"enabled"`
	HistoryWeight float64 `json:"historyWeight"`
	MediumRiskThreshold float64 `json:"mediumRiskThreshold"`
	NewIpWeight float64 `json:"newIpWeight"`
	VelocityWeight float64 `json:"velocityWeight"`
	BlockHighRisk bool `json:"blockHighRisk"`
	HighRiskThreshold float64 `json:"highRiskThreshold"`
	LowRiskThreshold float64 `json:"lowRiskThreshold"`
	NewDeviceWeight float64 `json:"newDeviceWeight"`
	NewLocationWeight float64 `json:"newLocationWeight"`
}

type TeamsResponse struct {
	Teams []*organization.Team `json:"teams"`
	Total int `json:"total"`
}

type CallbackResponse struct {
	Session *session.Session `json:"session"`
	Token string `json:"token"`
	User *user.User `json:"user"`
}

type mockImpersonationRepository struct {
	 * `json:",omitempty"`
}

// StatusResponse represents Status response
type StatusResponse struct {
	Status string `json:"status"`
}

type CompleteRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type GetRecoveryStatsRequest struct {
	EndDate time.Time `json:"endDate"`
	OrganizationId string `json:"organizationId"`
	StartDate time.Time `json:"startDate"`
}

type ListUsersResponse struct {
	Limit int `json:"limit"`
	Page int `json:"page"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
	Users []*user.User `json:"users"`
}

type TwoFABackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type UploadDocumentRequest struct {
	DocumentType string `json:"documentType"`
	FrontImage string `json:"frontImage"`
	Selfie string `json:"selfie"`
	SessionId xid.ID `json:"sessionId"`
	BackImage string `json:"backImage"`
}

type RequestTrustedContactVerificationRequest struct {
	ContactId xid.ID `json:"contactId"`
	SessionId xid.ID `json:"sessionId"`
}

type ConsentAuditLog struct {
	Purpose string `json:"purpose"`
	Reason string `json:"reason"`
	ConsentId string `json:"consentId"`
	ConsentType string `json:"consentType"`
	CreatedAt time.Time `json:"createdAt"`
	IpAddress string `json:"ipAddress"`
	NewValue JSONBMap `json:"newValue"`
	UserAgent string `json:"userAgent"`
	UserId string `json:"userId"`
	Action string `json:"action"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	PreviousValue JSONBMap `json:"previousValue"`
}

type DataDeletionRequest struct {
	RetentionExempt bool `json:"retentionExempt"`
	Status string `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
	ApprovedBy string `json:"approvedBy"`
	DeleteSections []string `json:"deleteSections"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	RejectedAt *time.Time `json:"rejectedAt"`
	ArchivePath string `json:"archivePath"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	ErrorMessage string `json:"errorMessage"`
	Id xid.ID `json:"id"`
	UserId string `json:"userId"`
	ApprovedAt *time.Time `json:"approvedAt"`
	ExemptionReason string `json:"exemptionReason"`
	RequestReason string `json:"requestReason"`
}

type OrganizationHandler struct {
	 **organization.Service `json:",omitempty"`
}

type NotificationListResponse struct {
	Notifications []* `json:"notifications"`
	Total int `json:"total"`
}

type SignUpRequest struct {
	Password string `json:"password"`
	Username string `json:"username"`
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

type ProviderCheckResult struct {
	 *bool `json:",omitempty"`
}

type ProviderSession struct {
	 *string `json:",omitempty"`
}

type AccountLockedResponse struct {
	Code string `json:"code"`
	Locked_minutes int `json:"locked_minutes"`
	Locked_until time.Time `json:"locked_until"`
	Message string `json:"message"`
}

type ClientUpdateRequest struct {
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
	Require_pkce *bool `json:"require_pkce"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Trusted_client *bool `json:"trusted_client"`
	Allowed_scopes []string `json:"allowed_scopes"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Name string `json:"name"`
	Require_consent *bool `json:"require_consent"`
	Response_types []string `json:"response_types"`
	Tos_uri string `json:"tos_uri"`
	Contacts []string `json:"contacts"`
	Policy_uri string `json:"policy_uri"`
}

type ResourceRule struct {
	Org_id string `json:"org_id"`
	Resource_type string `json:"resource_type"`
	Security_level SecurityLevel `json:"security_level"`
	Sensitivity string `json:"sensitivity"`
	Action string `json:"action"`
	Description string `json:"description"`
}

type CreateTrainingRequest struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type ConsentAuditConfig struct {
	LogIpAddress bool `json:"logIpAddress"`
	LogUserAgent bool `json:"logUserAgent"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	LogAllChanges bool `json:"logAllChanges"`
	RetentionDays int `json:"retentionDays"`
	SignLogs bool `json:"signLogs"`
	Enabled bool `json:"enabled"`
	ExportFormat string `json:"exportFormat"`
	Immutable bool `json:"immutable"`
}

type DataExportRequestInput struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type RateLimitConfig struct {
	RedisDb int `json:"redisDb"`
	SendCodePerPhone RateLimitRule `json:"sendCodePerPhone"`
	UseRedis bool `json:"useRedis"`
	VerifyPerIp RateLimitRule `json:"verifyPerIp"`
	RedisPassword string `json:"redisPassword"`
	SendCodePerIp RateLimitRule `json:"sendCodePerIp"`
	VerifyPerPhone RateLimitRule `json:"verifyPerPhone"`
	Enabled bool `json:"enabled"`
	RedisAddr string `json:"redisAddr"`
}

type ForgetDeviceResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

type SessionTokenResponse struct {
	Token string `json:"token"`
	Session  `json:"session"`
}

type ComplianceReportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type ComplianceUserTrainingResponse struct {
	User_id string `json:"user_id"`
}

type VideoSessionResult struct {
	 *string `json:",omitempty"`
}

type BlockUserRequest struct {
	Reason string `json:"reason"`
}

type GetChallengeStatusResponse struct {
	Attempts int `json:"attempts"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
	MaxAttempts int `json:"maxAttempts"`
	Status ChallengeStatus `json:"status"`
}

type Status_body struct {
	Device_id string `json:"device_id"`
	User_id string `json:"user_id"`
}

type DashboardConfig struct {
	ShowRecentChecks bool `json:"showRecentChecks"`
	ShowReports bool `json:"showReports"`
	ShowScore bool `json:"showScore"`
	ShowViolations bool `json:"showViolations"`
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
}

type DeletePasskeyRequest struct {
	 *string `json:",omitempty"`
}

type FinishRegisterRequest struct {
	Name string `json:"name"`
	Response  `json:"response"`
	UserId string `json:"userId"`
}

type FinishLoginRequest struct {
	Remember bool `json:"remember"`
	Response  `json:"response"`
}

type Plugin struct {
	 **bun.DB `json:",omitempty"`
}

type LinkResponse struct {
	Message string `json:"message"`
	User  `json:"user"`
}

type StepUpDevicesResponse struct {
	Count int `json:"count"`
	Devices  `json:"devices"`
}

type PolicyEngine struct {
	 **Service `json:",omitempty"`
}

type NotificationErrorResponse struct {
	Error string `json:"error"`
}

type TemplateService struct {
	 **notification.Service `json:",omitempty"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type Disable_body struct {
	User_id string `json:"user_id"`
}

type ImpersonationSession struct {
}

type NotificationWebhookResponse struct {
	Status string `json:"status"`
}

type LoginResponse struct {
	PasskeyUsed string `json:"passkeyUsed"`
	Session  `json:"session"`
	Token string `json:"token"`
	User  `json:"user"`
}

type ConsentPolicyResponse struct {
	Id string `json:"id"`
}

type UnblockUserRequest struct {
}

type MockSessionService struct {
}

type NoOpSMSProvider struct {
}

type UploadDocumentResponse struct {
	ProcessingTime string `json:"processingTime"`
	Status string `json:"status"`
	UploadedAt time.Time `json:"uploadedAt"`
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
}

type MultiStepRecoveryConfig struct {
	Enabled bool `json:"enabled"`
	LowRiskSteps []RecoveryMethod `json:"lowRiskSteps"`
	MinimumSteps int `json:"minimumSteps"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
	AllowStepSkip bool `json:"allowStepSkip"`
	AllowUserChoice bool `json:"allowUserChoice"`
	HighRiskSteps []RecoveryMethod `json:"highRiskSteps"`
	MediumRiskSteps []RecoveryMethod `json:"mediumRiskSteps"`
	SessionExpiry time.Duration `json:"sessionExpiry"`
}

type ConsentExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type MFASession struct {
	UserId xid.ID `json:"userId"`
	VerifiedFactors []xid.ID `json:"verifiedFactors"`
	CreatedAt time.Time `json:"createdAt"`
	FactorsRequired int `json:"factorsRequired"`
	Id xid.ID `json:"id"`
	RiskLevel RiskLevel `json:"riskLevel"`
	SessionToken string `json:"sessionToken"`
	CompletedAt *time.Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsVerified int `json:"factorsVerified"`
	IpAddress string `json:"ipAddress"`
	Metadata  `json:"metadata"`
	UserAgent string `json:"userAgent"`
}

type VerifyFactor_req struct {
	Code string `json:"code"`
}

type ComplianceEvidencesResponse struct {
	Evidence []* `json:"evidence"`
}

type ImpersonationStartResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Session_id string `json:"session_id"`
	Started_at string `json:"started_at"`
	Target_user_id string `json:"target_user_id"`
}

type FactorEnrollmentRequest struct {
	Type FactorType `json:"type"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
}

type Verify_body struct {
	Code string `json:"code"`
	Device_id string `json:"device_id"`
	Remember_device bool `json:"remember_device"`
	User_id string `json:"user_id"`
}

type TemplatesResponse struct {
	Count int `json:"count"`
	Templates  `json:"templates"`
}

type BackupAuthQuestionsResponse struct {
	Questions []string `json:"questions"`
}

type TokenRevocationRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type JWTService struct {
	 *string `json:",omitempty"`
}

type CompleteTraining_req struct {
	Score int `json:"score"`
}

type RunCheck_req struct {
	CheckType string `json:"checkType"`
}

type ChallengeSession struct {
	 *time.Time `json:",omitempty"`
}

type AddTrustedContactResponse struct {
	Phone string `json:"phone"`
	Verified bool `json:"verified"`
	AddedAt time.Time `json:"addedAt"`
	ContactId xid.ID `json:"contactId"`
	Email string `json:"email"`
	Message string `json:"message"`
	Name string `json:"name"`
}

type UpdateConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata  `json:"metadata"`
	Reason string `json:"reason"`
}

type CompliancePoliciesResponse struct {
	Policies []* `json:"policies"`
}

type SecurityQuestionsConfig struct {
	MinimumQuestions int `json:"minimumQuestions"`
	RequireMinLength int `json:"requireMinLength"`
	RequiredToRecover int `json:"requiredToRecover"`
	AllowCustomQuestions bool `json:"allowCustomQuestions"`
	CaseSensitive bool `json:"caseSensitive"`
	ForbidCommonAnswers bool `json:"forbidCommonAnswers"`
	PredefinedQuestions []string `json:"predefinedQuestions"`
	Enabled bool `json:"enabled"`
	LockoutDuration time.Duration `json:"lockoutDuration"`
	MaxAnswerLength int `json:"maxAnswerLength"`
	MaxAttempts int `json:"maxAttempts"`
}

type MetadataResponse struct {
	Metadata string `json:"metadata"`
}

type MockAuditService struct {
}

type MemoryChallengeStore struct {
	 * `json:",omitempty"`
}

type VerificationSessionResponse struct {
	Session *schema.IdentityVerificationSession `json:"session"`
}

type PreviewTemplate_req struct {
	Variables  `json:"variables"`
}

type NotificationsConfig struct {
	NotifyAdminOnReviewNeeded bool `json:"notifyAdminOnReviewNeeded"`
	NotifyOnRecoveryComplete bool `json:"notifyOnRecoveryComplete"`
	NotifyOnRecoveryFailed bool `json:"notifyOnRecoveryFailed"`
	NotifyOnRecoveryStart bool `json:"notifyOnRecoveryStart"`
	SecurityOfficerEmail string `json:"securityOfficerEmail"`
	Channels []string `json:"channels"`
	Enabled bool `json:"enabled"`
	NotifyAdminOnHighRisk bool `json:"notifyAdminOnHighRisk"`
}

type TrustDeviceRequest struct {
	DeviceId string `json:"deviceId"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
}

type FactorsResponse struct {
	Count int `json:"count"`
	Factors  `json:"factors"`
}

type MFAPolicy struct {
	UpdatedAt time.Time `json:"updatedAt"`
	AllowedFactorTypes []FactorType `json:"allowedFactorTypes"`
	GracePeriodDays int `json:"gracePeriodDays"`
	RequiredFactorCount int `json:"requiredFactorCount"`
	RequiredFactorTypes []FactorType `json:"requiredFactorTypes"`
	StepUpRequired bool `json:"stepUpRequired"`
	TrustedDeviceDays int `json:"trustedDeviceDays"`
	AdaptiveMfaEnabled bool `json:"adaptiveMfaEnabled"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	LockoutDurationMinutes int `json:"lockoutDurationMinutes"`
	MaxFailedAttempts int `json:"maxFailedAttempts"`
	OrganizationId xid.ID `json:"organizationId"`
}

type ListPasskeysResponse struct {
	Count int `json:"count"`
	Passkeys []PasskeyInfo `json:"passkeys"`
}

type IDVerificationWebhookResponse struct {
	Status string `json:"status"`
}

type IDTokenClaims struct {
	Auth_time int64 `json:"auth_time"`
	Email_verified bool `json:"email_verified"`
	Nonce string `json:"nonce"`
	Session_state string `json:"session_state"`
	Email string `json:"email"`
	Family_name string `json:"family_name"`
	Given_name string `json:"given_name"`
	Name string `json:"name"`
	Preferred_username string `json:"preferred_username"`
}

type AuthURLResponse struct {
	Url string `json:"url"`
}

type NotificationPreviewResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

type ClientRegistrationResponse struct {
	Response_types []string `json:"response_types"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Client_id string `json:"client_id"`
	Contacts []string `json:"contacts"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Scope string `json:"scope"`
	Tos_uri string `json:"tos_uri"`
	Client_secret string `json:"client_secret"`
	Logo_uri string `json:"logo_uri"`
	Client_name string `json:"client_name"`
	Client_secret_expires_at int64 `json:"client_secret_expires_at"`
	Application_type string `json:"application_type"`
	Client_id_issued_at int64 `json:"client_id_issued_at"`
	Grant_types []string `json:"grant_types"`
	Policy_uri string `json:"policy_uri"`
	Redirect_uris []string `json:"redirect_uris"`
}

type ProviderRegisteredResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Type string `json:"type"`
}

type ImpersonationEndResponse struct {
	Ended_at string `json:"ended_at"`
	Status string `json:"status"`
}

type NotificationTemplateListResponse struct {
	Templates []* `json:"templates"`
	Total int `json:"total"`
}

type SecurityQuestion struct {
	 *string `json:",omitempty"`
}

type DocumentVerificationRequest struct {
	 *[]byte `json:",omitempty"`
}

type Handler struct {
	 **Service `json:",omitempty"`
}

type ConsentAuditLogsResponse struct {
	Audit_logs []* `json:"audit_logs"`
}

type ScopeInfo struct {
	 *string `json:",omitempty"`
}

type BanUser_reqBody struct {
	Expires_at **time.Time `json:"expires_at,omitempty"`
	Reason string `json:"reason"`
}

type StatsResponse struct {
	Banned_users int `json:"banned_users"`
	Timestamp string `json:"timestamp"`
	Total_sessions int `json:"total_sessions"`
	Total_users int `json:"total_users"`
	Active_sessions int `json:"active_sessions"`
	Active_users int `json:"active_users"`
}

type ListChecksFilter struct {
	AppId *string `json:"appId"`
	CheckType *string `json:"checkType"`
	ProfileId *string `json:"profileId"`
	SinceBefore *time.Time `json:"sinceBefore"`
	Status *string `json:"status"`
}

type CookieConsentRequest struct {
	Functional bool `json:"functional"`
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	Analytics bool `json:"analytics"`
	BannerVersion string `json:"bannerVersion"`
	Essential bool `json:"essential"`
}

type TeamHandler struct {
	 **app.ServiceImpl `json:",omitempty"`
}

type AuditEvent struct {
	 *string `json:",omitempty"`
}

type OIDCState struct {
	 *string `json:",omitempty"`
}

type StepUpRequirementsResponse struct {
	Requirements []* `json:"requirements"`
}

type MockEmailService struct {
}

type ComplianceDashboardResponse struct {
	Metrics  `json:"metrics"`
}

type BeginLoginRequest struct {
	UserId string `json:"userId"`
	UserVerification string `json:"userVerification"`
}

type RateLimiter struct {
	 **repository.MFARepository `json:",omitempty"`
}

type StepUpErrorResponse struct {
	Error string `json:"error"`
}

type ContextRule struct {
	Condition string `json:"condition"`
	Description string `json:"description"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type ComplianceStatusResponse struct {
	Status string `json:"status"`
}

type CreateProvider_req struct {
	Config  `json:"config"`
	IsDefault bool `json:"isDefault"`
	OrganizationId **string `json:"organizationId,omitempty"`
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
}

type VerifyTrustedContactResponse struct {
	Verified bool `json:"verified"`
	VerifiedAt time.Time `json:"verifiedAt"`
	ContactId xid.ID `json:"contactId"`
	Message string `json:"message"`
}

type ReverifyRequest struct {
	Reason string `json:"reason"`
}

type ProviderConfigResponse struct {
	AppId string `json:"appId"`
	Message string `json:"message"`
	Provider string `json:"provider"`
}

type TimeBasedRule struct {
	Operation string `json:"operation"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Description string `json:"description"`
	Max_age time.Duration `json:"max_age"`
}

type App struct {
	 *string `json:",omitempty"`
}

type CookieConsentConfig struct {
	DefaultStyle string `json:"defaultStyle"`
	Enabled bool `json:"enabled"`
	RequireExplicit bool `json:"requireExplicit"`
	ValidityPeriod time.Duration `json:"validityPeriod"`
	AllowAnonymous bool `json:"allowAnonymous"`
	BannerVersion string `json:"bannerVersion"`
	Categories []string `json:"categories"`
}

type ConsentService struct {
	 **repo.OAuthClientRepository `json:",omitempty"`
}

type Status struct {
	 *bool `json:",omitempty"`
}

type User struct {
	 *time.Time `json:",omitempty"`
}

type WebhookResponse struct {
	Status string `json:"status"`
	Received bool `json:"received"`
}

type ClientDetailsResponse struct {
	ClientID string `json:"clientID"`
	Contacts []string `json:"contacts"`
	LogoURI string `json:"logoURI"`
	PolicyURI string `json:"policyURI"`
	AllowedScopes []string `json:"allowedScopes"`
	ApplicationType string `json:"applicationType"`
	RequireConsent bool `json:"requireConsent"`
	ResponseTypes []string `json:"responseTypes"`
	TosURI string `json:"tosURI"`
	UpdatedAt string `json:"updatedAt"`
	GrantTypes []string `json:"grantTypes"`
	IsOrgLevel bool `json:"isOrgLevel"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	RedirectURIs []string `json:"redirectURIs"`
	RequirePKCE bool `json:"requirePKCE"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	CreatedAt string `json:"createdAt"`
	Name string `json:"name"`
	OrganizationID string `json:"organizationID"`
	TrustedClient bool `json:"trustedClient"`
}

type ConnectionsResponse struct {
	Connections []*schema.SocialAccount `json:"connections"`
}

type MultiSessionListResponse struct {
	Sessions []* `json:"sessions"`
}

type TwoFAStatusResponse struct {
	Enabled bool `json:"enabled"`
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
}

type RateLimitingConfig struct {
	ExponentialBackoff bool `json:"exponentialBackoff"`
	IpCooldownPeriod time.Duration `json:"ipCooldownPeriod"`
	LockoutAfterAttempts int `json:"lockoutAfterAttempts"`
	LockoutDuration time.Duration `json:"lockoutDuration"`
	MaxAttemptsPerDay int `json:"maxAttemptsPerDay"`
	MaxAttemptsPerHour int `json:"maxAttemptsPerHour"`
	MaxAttemptsPerIp int `json:"maxAttemptsPerIp"`
	Enabled bool `json:"enabled"`
}

type GenerateBackupCodes_body struct {
	User_id string `json:"user_id"`
	Count int `json:"count"`
}

type RecoverySessionInfo struct {
	CompletedAt *time.Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Method RecoveryMethod `json:"method"`
	RequiresReview bool `json:"requiresReview"`
	RiskScore float64 `json:"riskScore"`
	Status RecoveryStatus `json:"status"`
	UserEmail string `json:"userEmail"`
	UserId xid.ID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	CurrentStep int `json:"currentStep"`
	Id xid.ID `json:"id"`
	TotalSteps int `json:"totalSteps"`
}

type UpdatePolicyRequest struct {
	Renewable *bool `json:"renewable"`
	Required *bool `json:"required"`
	ValidityPeriod *int `json:"validityPeriod"`
	Active *bool `json:"active"`
	Content string `json:"content"`
	Description string `json:"description"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
}

type ListTrustedDevicesResponse struct {
	Count int `json:"count"`
	Devices []TrustedDevice `json:"devices"`
}

type CallbackResult struct {
	 *bool `json:",omitempty"`
}

type StepUpAuditLog struct {
	Created_at time.Time `json:"created_at"`
	Event_data  `json:"event_data"`
	Event_type string `json:"event_type"`
	Id string `json:"id"`
	Org_id string `json:"org_id"`
	User_agent string `json:"user_agent"`
	Ip string `json:"ip"`
	Severity string `json:"severity"`
	User_id string `json:"user_id"`
}

type RotateAPIKeyResponse struct {
	Api_key *apikey.APIKey `json:"api_key"`
	Message string `json:"message"`
}

type UnbanUser_reqBody struct {
	Reason *string `json:"reason,omitempty"`
}

type TrustedDevice struct {
	Name string `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
	DeviceId string `json:"deviceId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	Metadata  `json:"metadata"`
}

type MemberHandler struct {
	 **coreapp.ServiceImpl `json:",omitempty"`
}

type BackupAuthSessionsResponse struct {
	Sessions []* `json:"sessions"`
}

type ErrorResponse struct {
	Code string `json:"code"`
	Details  `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

type CompliancePolicyResponse struct {
	Id string `json:"id"`
}

type BackupAuthCodesResponse struct {
	Codes []string `json:"codes"`
}

type NoOpEmailProvider struct {
}

type ConsentRecord struct {
	GrantedAt time.Time `json:"grantedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Metadata JSONBMap `json:"metadata"`
	Purpose string `json:"purpose"`
	RevokedAt *time.Time `json:"revokedAt"`
	ConsentType string `json:"consentType"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Granted bool `json:"granted"`
	Id xid.ID `json:"id"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserAgent string `json:"userAgent"`
	UserId string `json:"userId"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	Version string `json:"version"`
}

type MockRepository struct {
	 * `json:",omitempty"`
}

type IDVerificationListResponse struct {
	Verifications []* `json:"verifications"`
}

type AccessTokenClaims struct {
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
}

type VerifyChallengeRequest struct {
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data  `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
}

type Challenge struct {
	- string `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	MaxAttempts int `json:"maxAttempts"`
	Metadata  `json:"metadata"`
	VerifiedAt *time.Time `json:"verifiedAt"`
	Attempts int `json:"attempts"`
	FactorId xid.ID `json:"factorId"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	Status ChallengeStatus `json:"status"`
	Type FactorType `json:"type"`
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
}

type auditServiceAdapter struct {
	 **audit.Service `json:",omitempty"`
}

type Adapter struct {
	 **TemplateService `json:",omitempty"`
}

type AccountLockoutError struct {
	 *int `json:",omitempty"`
}

type ProviderDiscoveredResponse struct {
	Found bool `json:"found"`
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
}

type EvaluationResult struct {
	Metadata  `json:"metadata"`
	Reason string `json:"reason"`
	Requirement_id string `json:"requirement_id"`
	Current_level SecurityLevel `json:"current_level"`
	Expires_at time.Time `json:"expires_at"`
	Required bool `json:"required"`
	Security_level SecurityLevel `json:"security_level"`
	Allowed_methods []VerificationMethod `json:"allowed_methods"`
	Can_remember bool `json:"can_remember"`
	Challenge_token string `json:"challenge_token"`
	Grace_period_ends_at time.Time `json:"grace_period_ends_at"`
	Matched_rules []string `json:"matched_rules"`
}

type mockUserService struct {
	 * `json:",omitempty"`
}

type EndImpersonation_reqBody struct {
	Impersonation_id string `json:"impersonation_id"`
	Reason *string `json:"reason,omitempty"`
}

type TemplateDefault struct {
	 *string `json:",omitempty"`
}

type DataDeletionConfig struct {
	NotifyBeforeDeletion bool `json:"notifyBeforeDeletion"`
	PreserveLegalData bool `json:"preserveLegalData"`
	AllowPartialDeletion bool `json:"allowPartialDeletion"`
	ArchivePath string `json:"archivePath"`
	AutoProcessAfterGrace bool `json:"autoProcessAfterGrace"`
	Enabled bool `json:"enabled"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
	RetentionExemptions []string `json:"retentionExemptions"`
	ArchiveBeforeDeletion bool `json:"archiveBeforeDeletion"`
	GracePeriodDays int `json:"gracePeriodDays"`
}

type OAuthErrorResponse struct {
	Error string `json:"error"`
	Error_description string `json:"error_description"`
	Error_uri string `json:"error_uri"`
	State string `json:"state"`
}

type OAuthState struct {
	Link_user_id *xid.ID `json:"link_user_id"`
	Provider string `json:"provider"`
	Redirect_url string `json:"redirect_url"`
	User_organization_id *xid.ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Created_at time.Time `json:"created_at"`
	Extra_scopes []string `json:"extra_scopes"`
}

type MockUserRepository struct {
}

type ComplianceViolationsResponse struct {
	Violations []* `json:"violations"`
}

type WebAuthnWrapper struct {
	 *Config `json:",omitempty"`
}

type RedisChallengeStore struct {
}

type InitiateChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata  `json:"metadata"`
}

type CreateTraining_req struct {
	UserId string `json:"userId"`
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
}

type AddTrustedContactRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
}

type GenerateRecoveryCodesResponse struct {
	Codes []string `json:"codes"`
	Count int `json:"count"`
	GeneratedAt time.Time `json:"generatedAt"`
	Warning string `json:"warning"`
}

type DeleteFactorRequest struct {
	 *string `json:",omitempty"`
}

type TwoFAErrorResponse struct {
	Error string `json:"error"`
}

