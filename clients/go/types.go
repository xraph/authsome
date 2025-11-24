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

type PreviewTemplateRequest struct {
	Variables interface{} `json:"variables"`
}

type WebAuthnFactorAdapter struct {
}

type UpdateProfileRequest struct {
	Status *string `json:"status"`
	MfaRequired *bool `json:"mfaRequired"`
	Name *string `json:"name"`
	RetentionDays *int `json:"retentionDays"`
}

type MultiSessionSetActiveResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type Disable_body struct {
	User_id string `json:"user_id"`
}

type RiskEngine struct {
}

type FactorInfo struct {
	FactorId xid.ID `json:"factorId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Type FactorType `json:"type"`
}

type RequestTrustedContactVerificationRequest struct {
	ContactId xid.ID `json:"contactId"`
	SessionId xid.ID `json:"sessionId"`
}

type UserVerificationStatusResponse struct {
	Status UserVerificationStatus `json:"status"`
}

type ConsentDecision struct {
}

type CreateUserRequest struct {
	App_id xid.ID `json:"app_id"`
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
	Metadata interface{} `json:"metadata"`
	Password string `json:"password"`
	Role string `json:"role"`
	User_organization_id ID `json:"user_organization_id"`
	Username string `json:"username"`
	Name string `json:"name"`
}

type AdaptiveMFAConfig struct {
	Factor_new_device bool `json:"factor_new_device"`
	Factor_velocity bool `json:"factor_velocity"`
	Location_change_risk float64 `json:"location_change_risk"`
	Velocity_risk float64 `json:"velocity_risk"`
	Enabled bool `json:"enabled"`
	Factor_location_change bool `json:"factor_location_change"`
	New_device_risk float64 `json:"new_device_risk"`
	Require_step_up_threshold float64 `json:"require_step_up_threshold"`
	Risk_threshold float64 `json:"risk_threshold"`
	Factor_ip_reputation bool `json:"factor_ip_reputation"`
}

type FactorAdapterRegistry struct {
}

type DocumentVerificationResult struct {
}

type ConsentPolicyResponse struct {
	Id string `json:"id"`
}

type TrackNotificationEvent_req struct {
	Event string `json:"event"`
	EventData *interface{} `json:"eventData,omitempty"`
	NotificationId string `json:"notificationId"`
	OrganizationId *string `json:"organizationId,omitempty"`
	TemplateId string `json:"templateId"`
}

type ImpersonationMiddleware struct {
}

type CallbackResponse struct {
	User User `json:"user"`
	Session Session `json:"session"`
	Token string `json:"token"`
}

type MFAPolicy struct {
	AdaptiveMfaEnabled bool `json:"adaptiveMfaEnabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
	LockoutDurationMinutes int `json:"lockoutDurationMinutes"`
	OrganizationId xid.ID `json:"organizationId"`
	RequiredFactorCount int `json:"requiredFactorCount"`
	RequiredFactorTypes []FactorType `json:"requiredFactorTypes"`
	StepUpRequired bool `json:"stepUpRequired"`
	AllowedFactorTypes []FactorType `json:"allowedFactorTypes"`
	CreatedAt time.Time `json:"createdAt"`
	MaxFailedAttempts int `json:"maxFailedAttempts"`
	TrustedDeviceDays int `json:"trustedDeviceDays"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UpdateConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
}

type OnfidoConfig struct {
	Enabled bool `json:"enabled"`
	FacialCheck FacialCheckConfig `json:"facialCheck"`
	IncludeWatchlistReport bool `json:"includeWatchlistReport"`
	WebhookToken string `json:"webhookToken"`
	ApiToken string `json:"apiToken"`
	DocumentCheck DocumentCheckConfig `json:"documentCheck"`
	IncludeDocumentReport bool `json:"includeDocumentReport"`
	IncludeFacialReport bool `json:"includeFacialReport"`
	Region string `json:"region"`
	WorkflowId string `json:"workflowId"`
}

type DeleteRequest struct {
	Id string `json:"id"`
}

type AuthorizeRequest struct {
	Prompt string `json:"prompt"`
	Scope string `json:"scope"`
	State string `json:"state"`
	Acr_values string `json:"acr_values"`
	Client_id string `json:"client_id"`
	Id_token_hint string `json:"id_token_hint"`
	Login_hint string `json:"login_hint"`
	Nonce string `json:"nonce"`
	Redirect_uri string `json:"redirect_uri"`
	Response_type string `json:"response_type"`
	Ui_locales string `json:"ui_locales"`
	Code_challenge string `json:"code_challenge"`
	Code_challenge_method string `json:"code_challenge_method"`
	Max_age *int `json:"max_age"`
}

type RiskFactor struct {
}

type RecoverySession struct {
}

type NoOpEmailProvider struct {
}

type DefaultProviderRegistry struct {
}

type MockSessionService struct {
}

type VerifyRecoveryCodeResponse struct {
	Message string `json:"message"`
	RemainingCodes int `json:"remainingCodes"`
	Valid bool `json:"valid"`
}

type ContinueRecoveryResponse struct {
	SessionId xid.ID `json:"sessionId"`
	TotalSteps int `json:"totalSteps"`
	CurrentStep int `json:"currentStep"`
	Data interface{} `json:"data"`
	ExpiresAt time.Time `json:"expiresAt"`
	Instructions string `json:"instructions"`
	Method RecoveryMethod `json:"method"`
}

type LinkRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Password string `json:"password"`
}

type TwoFAStatusResponse struct {
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
	Enabled bool `json:"enabled"`
}

type BackupCodeFactorAdapter struct {
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

type AppServiceAdapter struct {
}

type ConsentAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type StepUpAttempt struct {
	Created_at time.Time `json:"created_at"`
	Ip string `json:"ip"`
	Method VerificationMethod `json:"method"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Failure_reason string `json:"failure_reason"`
	Id string `json:"id"`
	Org_id string `json:"org_id"`
	Requirement_id string `json:"requirement_id"`
	Success bool `json:"success"`
}

// MessageResponse represents Simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

type ComplianceTemplatesResponse struct {
	Templates []*interface{} `json:"templates"`
}

type DataDeletionConfig struct {
	ArchivePath string `json:"archivePath"`
	GracePeriodDays int `json:"gracePeriodDays"`
	NotifyBeforeDeletion bool `json:"notifyBeforeDeletion"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
	RetentionExemptions []string `json:"retentionExemptions"`
	ArchiveBeforeDeletion bool `json:"archiveBeforeDeletion"`
	AutoProcessAfterGrace bool `json:"autoProcessAfterGrace"`
	Enabled bool `json:"enabled"`
	PreserveLegalData bool `json:"preserveLegalData"`
	AllowPartialDeletion bool `json:"allowPartialDeletion"`
}

type DataDeletionRequest struct {
	ArchivePath string `json:"archivePath"`
	CompletedAt Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	ErrorMessage string `json:"errorMessage"`
	ExemptionReason string `json:"exemptionReason"`
	OrganizationId string `json:"organizationId"`
	RejectedAt Time `json:"rejectedAt"`
	Status string `json:"status"`
	ApprovedBy string `json:"approvedBy"`
	IpAddress string `json:"ipAddress"`
	RetentionExempt bool `json:"retentionExempt"`
	UserId string `json:"userId"`
	DeleteSections []string `json:"deleteSections"`
	Id xid.ID `json:"id"`
	UpdatedAt time.Time `json:"updatedAt"`
	ApprovedAt Time `json:"approvedAt"`
	RequestReason string `json:"requestReason"`
}

type StepUpEvaluationResponse struct {
	Reason string `json:"reason"`
	Required bool `json:"required"`
}

type TwoFASendOTPResponse struct {
	Status string `json:"status"`
	Code string `json:"code"`
}

type VerifyChallengeRequest struct {
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data interface{} `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
}

type NoOpNotificationProvider struct {
}

type VerifySecurityAnswersRequest struct {
	Answers interface{} `json:"answers"`
	SessionId xid.ID `json:"sessionId"`
}

type ComplianceProfileResponse struct {
	Id string `json:"id"`
}

type ComplianceTemplate struct {
	RequiredTraining []string `json:"requiredTraining"`
	RetentionDays int `json:"retentionDays"`
	SessionMaxAge int `json:"sessionMaxAge"`
	Description string `json:"description"`
	Name string `json:"name"`
	Standard ComplianceStandard `json:"standard"`
	AuditFrequencyDays int `json:"auditFrequencyDays"`
	DataResidency string `json:"dataResidency"`
	MfaRequired bool `json:"mfaRequired"`
	PasswordMinLength int `json:"passwordMinLength"`
	RequiredPolicies []string `json:"requiredPolicies"`
}

type UnbanUserRequest struct {
	App_id xid.ID `json:"app_id"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
}

type AdminBlockUserRequest struct {
	Reason string `json:"reason"`
}

type RegisterProviderResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Type string `json:"type"`
}

type ComplianceEvidenceResponse struct {
	Id string `json:"id"`
}

type RunCheck_req struct {
	CheckType string `json:"checkType"`
}

type JumioConfig struct {
	PresetId string `json:"presetId"`
	VerificationType string `json:"verificationType"`
	ApiToken string `json:"apiToken"`
	CallbackUrl string `json:"callbackUrl"`
	Enabled bool `json:"enabled"`
	EnabledCountries []string `json:"enabledCountries"`
	ApiSecret string `json:"apiSecret"`
	DataCenter string `json:"dataCenter"`
	EnableAMLScreening bool `json:"enableAMLScreening"`
	EnableExtraction bool `json:"enableExtraction"`
	EnableLiveness bool `json:"enableLiveness"`
	EnabledDocumentTypes []string `json:"enabledDocumentTypes"`
}

type IntrospectionService struct {
}

type GetDocumentVerificationRequest struct {
	DocumentId xid.ID `json:"documentId"`
}

type ProvidersConfig struct {
	Email EmailProviderConfig `json:"email"`
	Sms *SMSProviderConfig `json:"sms"`
}

type TestProvider_req struct {
	Config interface{} `json:"config"`
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
	TestRecipient string `json:"testRecipient"`
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

type WebAuthnConfig struct {
	Attestation_preference string `json:"attestation_preference"`
	Authenticator_selection interface{} `json:"authenticator_selection"`
	Enabled bool `json:"enabled"`
	Rp_display_name string `json:"rp_display_name"`
	Rp_id string `json:"rp_id"`
	Rp_origins []string `json:"rp_origins"`
	Timeout int `json:"timeout"`
}

type SMSVerificationConfig struct {
	CodeLength int `json:"codeLength"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	Enabled bool `json:"enabled"`
	MaxAttempts int `json:"maxAttempts"`
	MaxSmsPerDay int `json:"maxSmsPerDay"`
	MessageTemplate string `json:"messageTemplate"`
	Provider string `json:"provider"`
	CodeExpiry time.Duration `json:"codeExpiry"`
}

type WebhookResponse struct {
	Received bool `json:"received"`
	Status string `json:"status"`
}

type RateLimit struct {
	Max_requests int `json:"max_requests"`
	Window time.Duration `json:"window"`
}

type ProviderDetailResponse struct {
	AttributeMapping interface{} `json:"attributeMapping"`
	Domain string `json:"domain"`
	HasSamlCert bool `json:"hasSamlCert"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
	ProviderId string `json:"providerId"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	SamlIssuer string `json:"samlIssuer"`
	UpdatedAt string `json:"updatedAt"`
	CreatedAt string `json:"createdAt"`
	OidcClientID string `json:"oidcClientID"`
	OidcIssuer string `json:"oidcIssuer"`
	Type string `json:"type"`
}

type CodesResponse struct {
	Codes []string `json:"codes"`
}

type TwoFAErrorResponse struct {
	Error string `json:"error"`
}

type GetSecurityQuestionsResponse struct {
	Questions []SecurityQuestionInfo `json:"questions"`
}

type StepUpStatusResponse struct {
	Status string `json:"status"`
}

type LinkResponse struct {
	User interface{} `json:"user"`
	Message string `json:"message"`
}

type ChallengeResponse struct {
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	SessionId xid.ID `json:"sessionId"`
}

type UpdatePolicy_req struct {
	Status *string `json:"status"`
	Title *string `json:"title"`
	Version *string `json:"version"`
	Content *string `json:"content"`
}

type StepUpPolicyResponse struct {
	Id string `json:"id"`
}

type SetActive_body struct {
	Id string `json:"id"`
}

type StartImpersonation_reqBody struct {
	Duration_minutes *int `json:"duration_minutes,omitempty"`
	Reason string `json:"reason"`
	Target_user_id string `json:"target_user_id"`
	Ticket_number *string `json:"ticket_number,omitempty"`
}

type StatsResponse struct {
	Active_users int `json:"active_users"`
	Banned_users int `json:"banned_users"`
	Timestamp string `json:"timestamp"`
	Total_sessions int `json:"total_sessions"`
	Total_users int `json:"total_users"`
	Active_sessions int `json:"active_sessions"`
}

type Factor struct {
	CreatedAt time.Time `json:"createdAt"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	UserId xid.ID `json:"userId"`
	ExpiresAt Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	LastUsedAt Time `json:"lastUsedAt"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
	UpdatedAt time.Time `json:"updatedAt"`
	VerifiedAt Time `json:"verifiedAt"`
}

type CreatePolicyRequest struct {
	PolicyType string `json:"policyType"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	Version string `json:"version"`
	Content string `json:"content"`
}

type RegisterClientRequest struct {
	Policy_uri string `json:"policy_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Tos_uri string `json:"tos_uri"`
	Require_consent bool `json:"require_consent"`
	Require_pkce bool `json:"require_pkce"`
	Application_type string `json:"application_type"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Response_types []string `json:"response_types"`
	Scope string `json:"scope"`
	Client_name string `json:"client_name"`
	Contacts []string `json:"contacts"`
	Trusted_client bool `json:"trusted_client"`
}

type EndImpersonationRequest struct {
	Impersonation_id string `json:"impersonation_id"`
	Reason *string `json:"reason,omitempty"`
}

type IPWhitelistConfig struct {
	Enabled bool `json:"enabled"`
	Strict_mode bool `json:"strict_mode"`
}

type BeginLoginRequest struct {
	UserId string `json:"userId"`
	UserVerification string `json:"userVerification"`
}

type DisableRequest struct {
	User_id string `json:"user_id"`
}

type LimitResult struct {
}

type DevicesResponse struct {
	Count int `json:"count"`
	Devices interface{} `json:"devices"`
}

type UserAdapter struct {
}

type UnbanUser_reqBody struct {
	Reason *string `json:"reason,omitempty"`
}

type MockEmailService struct {
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

type SendOTP_body struct {
	User_id string `json:"user_id"`
}

type ListResponse struct {
	Sessions interface{} `json:"sessions"`
}

type ProviderConfigResponse struct {
	AppId string `json:"appId"`
	Message string `json:"message"`
	Provider string `json:"provider"`
}

type MFAConfigResponse struct {
	Allowed_factor_types []string `json:"allowed_factor_types"`
	Enabled bool `json:"enabled"`
	Required_factor_count int `json:"required_factor_count"`
}

type VerifyFactor_req struct {
	Code string `json:"code"`
}

type AutomatedChecksConfig struct {
	SuspiciousActivity bool `json:"suspiciousActivity"`
	AccessReview bool `json:"accessReview"`
	Enabled bool `json:"enabled"`
	InactiveUsers bool `json:"inactiveUsers"`
	PasswordPolicy bool `json:"passwordPolicy"`
	CheckInterval time.Duration `json:"checkInterval"`
	DataRetention bool `json:"dataRetention"`
	MfaCoverage bool `json:"mfaCoverage"`
	SessionPolicy bool `json:"sessionPolicy"`
}

type MockAuditService struct {
}

type ConsentTypeStatus struct {
	NeedsRenewal bool `json:"needsRenewal"`
	Type string `json:"type"`
	Version string `json:"version"`
	ExpiresAt Time `json:"expiresAt"`
	Granted bool `json:"granted"`
	GrantedAt time.Time `json:"grantedAt"`
}

type OIDCLoginRequest struct {
	Nonce string `json:"nonce"`
	RedirectUri string `json:"redirectUri"`
	Scope string `json:"scope"`
	State string `json:"state"`
}

type SAMLSPMetadataResponse struct {
	Metadata string `json:"metadata"`
}

type ChannelsResponse struct {
	Channels interface{} `json:"channels"`
	Count int `json:"count"`
}

type ImpersonationEndResponse struct {
	Status string `json:"status"`
	Ended_at string `json:"ended_at"`
}

type stateEntry struct {
}

type BackupAuthDocumentResponse struct {
	Id string `json:"id"`
}

type UserServiceAdapter struct {
}

type AuditEvent struct {
}

type ConsentStatusResponse struct {
	Status string `json:"status"`
}

type OnfidoProvider struct {
}

type SessionsResponse struct {
	Sessions interface{} `json:"sessions"`
}

type TokenIntrospectionResponse struct {
	Client_id string `json:"client_id"`
	Exp int64 `json:"exp"`
	Nbf int64 `json:"nbf"`
	Scope string `json:"scope"`
	Sub string `json:"sub"`
	Token_type string `json:"token_type"`
	Username string `json:"username"`
	Active bool `json:"active"`
	Aud []string `json:"aud"`
	Iat int64 `json:"iat"`
	Iss string `json:"iss"`
	Jti string `json:"jti"`
}

type SMSFactorAdapter struct {
}

type MockUserService struct {
}

type mockRepository struct {
}

type ProviderCheckResult struct {
}

type GenerateBackupCodes_body struct {
	Count int `json:"count"`
	User_id string `json:"user_id"`
}

type GetUserVerificationsResponse struct {
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Total int `json:"total"`
	Verifications IdentityVerification `json:"verifications"`
}

type Challenge struct {
	Attempts int `json:"attempts"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	MaxAttempts int `json:"maxAttempts"`
	Metadata interface{} `json:"metadata"`
	Status ChallengeStatus `json:"status"`
	Type FactorType `json:"type"`
	FactorId xid.ID `json:"factorId"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
	VerifiedAt Time `json:"verifiedAt"`
}

type EmailFactorAdapter struct {
}

type SetupSecurityQuestionsRequest struct {
	Questions []SetupSecurityQuestionRequest `json:"questions"`
}

type AddTeamMember_req struct {
	Member_id xid.ID `json:"member_id"`
	Role string `json:"role"`
}

type RevokeDeviceResponse struct {
	Success bool `json:"success"`
}

type ClientsListResponse struct {
	TotalPages int `json:"totalPages"`
	Clients []ClientSummary `json:"clients"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Total int `json:"total"`
}

type EmailServiceAdapter struct {
}

type AdminBypassRequest struct {
	Duration int `json:"duration"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
}

type AutoCleanupConfig struct {
	Enabled bool `json:"enabled"`
	Interval time.Duration `json:"interval"`
}

type SendWithTemplateRequest struct {
	TemplateKey string `json:"templateKey"`
	Type NotificationType `json:"type"`
	Variables interface{} `json:"variables"`
	AppId xid.ID `json:"appId"`
	Language string `json:"language"`
	Metadata interface{} `json:"metadata"`
	Recipient string `json:"recipient"`
}

type SendCodeRequest struct {
	Phone string `json:"phone"`
}

type UpdatePolicyResponse struct {
	Updated_at time.Time `json:"updated_at"`
	Enabled bool `json:"enabled"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Priority int `json:"priority"`
	Rules interface{} `json:"rules"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Description string `json:"description"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
}

type RenderTemplateRequest struct {
	Template string `json:"template"`
	Variables interface{} `json:"variables"`
}

type BackupAuthStatsResponse struct {
	Stats interface{} `json:"stats"`
}

type ConsentCookieResponse struct {
	Preferences interface{} `json:"preferences"`
}

type CreateAPIKeyResponse struct {
	Api_key APIKey `json:"api_key"`
	Message string `json:"message"`
}

type StepUpErrorResponse struct {
	Error string `json:"error"`
}

type ClientAuthenticator struct {
}

type OIDCState struct {
}

type TwoFARequiredResponse struct {
	Device_id string `json:"device_id"`
	Require_twofa bool `json:"require_twofa"`
	User User `json:"user"`
}

type GenerateBackupCodesRequest struct {
	User_id string `json:"user_id"`
	Count int `json:"count"`
}

type VerifyEnrolledFactorRequest struct {
	Code string `json:"code"`
	Data interface{} `json:"data"`
}

type AdminPolicyRequest struct {
	AllowedTypes []string `json:"allowedTypes"`
	Enabled bool `json:"enabled"`
	GracePeriod int `json:"gracePeriod"`
	RequiredFactors int `json:"requiredFactors"`
}

type GetSecurityQuestionsRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type NotificationChannels struct {
	Email bool `json:"email"`
	Slack bool `json:"slack"`
	Webhook bool `json:"webhook"`
}

type ComplianceStatus struct {
	NextAudit time.Time `json:"nextAudit"`
	ProfileId string `json:"profileId"`
	Score int `json:"score"`
	AppId string `json:"appId"`
	OverallStatus string `json:"overallStatus"`
	Standard ComplianceStandard `json:"standard"`
	Violations int `json:"violations"`
	ChecksFailed int `json:"checksFailed"`
	ChecksPassed int `json:"checksPassed"`
	ChecksWarning int `json:"checksWarning"`
	LastChecked time.Time `json:"lastChecked"`
}

type GetVerificationResponse struct {
	Verification IdentityVerification `json:"verification"`
}

type VerifySecurityAnswersResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	CorrectAnswers int `json:"correctAnswers"`
	Message string `json:"message"`
	RequiredAnswers int `json:"requiredAnswers"`
	Valid bool `json:"valid"`
}

type SetupSecurityQuestionRequest struct {
	QuestionId int `json:"questionId"`
	Answer string `json:"answer"`
	CustomText string `json:"customText"`
}

type AddTrustedContactResponse struct {
	Email string `json:"email"`
	Message string `json:"message"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Verified bool `json:"verified"`
	AddedAt time.Time `json:"addedAt"`
	ContactId xid.ID `json:"contactId"`
}

type CreateUser_reqBody struct {
	Email_verified bool `json:"email_verified"`
	Metadata *interface{} `json:"metadata,omitempty"`
	Name *string `json:"name,omitempty"`
	Password *string `json:"password,omitempty"`
	Role *string `json:"role,omitempty"`
	Username *string `json:"username,omitempty"`
	Email string `json:"email"`
}

type EnableRequest struct {
}

type ConsentExpiryConfig struct {
	AllowRenewal bool `json:"allowRenewal"`
	AutoExpireCheck bool `json:"autoExpireCheck"`
	DefaultValidityDays int `json:"defaultValidityDays"`
	Enabled bool `json:"enabled"`
	ExpireCheckInterval time.Duration `json:"expireCheckInterval"`
	RenewalReminderDays int `json:"renewalReminderDays"`
	RequireReConsent bool `json:"requireReConsent"`
}

type BeginRegisterRequest struct {
	RequireResidentKey bool `json:"requireResidentKey"`
	UserId string `json:"userId"`
	UserVerification string `json:"userVerification"`
	AuthenticatorType string `json:"authenticatorType"`
	Name string `json:"name"`
}

type ImpersonationStartResponse struct {
	Target_user_id string `json:"target_user_id"`
	Impersonator_id string `json:"impersonator_id"`
	Session_id string `json:"session_id"`
	Started_at string `json:"started_at"`
}

type SSOAuthResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type UpdateResponse struct {
	Webhook Webhook `json:"webhook"`
}

type EnrollFactorRequest struct {
	Type FactorType `json:"type"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
}

type MockStateStore struct {
}

type CreateABTestVariant_req struct {
	Body string `json:"body"`
	Name string `json:"name"`
	Subject string `json:"subject"`
	Weight int `json:"weight"`
}

type TOTPConfig struct {
	Digits int `json:"digits"`
	Enabled bool `json:"enabled"`
	Issuer string `json:"issuer"`
	Period int `json:"period"`
	Window_size int `json:"window_size"`
	Algorithm string `json:"algorithm"`
}

type GetRecoveryStatsRequest struct {
	EndDate time.Time `json:"endDate"`
	OrganizationId string `json:"organizationId"`
	StartDate time.Time `json:"startDate"`
}

type BackupAuthContactsResponse struct {
	Contacts []*interface{} `json:"contacts"`
}

type ConsentDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type CreateSessionRequest struct {
}

type SAMLCallbackResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type SendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type NotificationResponse struct {
	Notification interface{} `json:"notification"`
}

// SuccessResponse represents Success boolean response
type SuccessResponse struct {
	Success bool `json:"success"`
}

type DiscoverProviderRequest struct {
	Email string `json:"email"`
}

type ListDevicesResponse struct {
	Devices []*Device `json:"devices"`
}

type ComplianceReportsResponse struct {
	Reports []*interface{} `json:"reports"`
}

type mockUserService struct {
}

type TrustedDevicesConfig struct {
	Default_expiry_days int `json:"default_expiry_days"`
	Enabled bool `json:"enabled"`
	Max_devices_per_user int `json:"max_devices_per_user"`
	Max_expiry_days int `json:"max_expiry_days"`
}

type RiskContext struct {
}

type ListTrainingFilter struct {
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	TrainingType *string `json:"trainingType"`
	UserId *string `json:"userId"`
}

type auditServiceAdapter struct {
}

type VerifyResponse struct {
	Expires_at time.Time `json:"expires_at"`
	Metadata interface{} `json:"metadata"`
	Security_level SecurityLevel `json:"security_level"`
	Success bool `json:"success"`
	Verification_id string `json:"verification_id"`
	Device_remembered bool `json:"device_remembered"`
	Error string `json:"error"`
}

type CallbackResult struct {
}

type StripeIdentityProvider struct {
}

type SignUpRequest struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type AdminUpdateProviderRequest struct {
	ClientId *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
	Enabled *bool `json:"enabled"`
	Scopes []string `json:"scopes"`
}

type SecurityQuestion struct {
}

type GetDocumentVerificationResponse struct {
	Message string `json:"message"`
	RejectionReason string `json:"rejectionReason"`
	Status string `json:"status"`
	VerifiedAt Time `json:"verifiedAt"`
	ConfidenceScore float64 `json:"confidenceScore"`
	DocumentId xid.ID `json:"documentId"`
}

type ComplianceTraining struct {
	AppId string `json:"appId"`
	ExpiresAt Time `json:"expiresAt"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
	CompletedAt Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	ProfileId string `json:"profileId"`
	Score int `json:"score"`
	Status string `json:"status"`
}

type StepUpRequirement struct {
	Required_level SecurityLevel `json:"required_level"`
	Challenge_token string `json:"challenge_token"`
	Created_at time.Time `json:"created_at"`
	Currency string `json:"currency"`
	Current_level SecurityLevel `json:"current_level"`
	Id string `json:"id"`
	Org_id string `json:"org_id"`
	Amount float64 `json:"amount"`
	Fulfilled_at Time `json:"fulfilled_at"`
	Reason string `json:"reason"`
	Resource_type string `json:"resource_type"`
	Route string `json:"route"`
	Session_id string `json:"session_id"`
	Status string `json:"status"`
	User_id string `json:"user_id"`
	Expires_at time.Time `json:"expires_at"`
	Method string `json:"method"`
	Risk_score float64 `json:"risk_score"`
	Rule_name string `json:"rule_name"`
	User_agent string `json:"user_agent"`
	Ip string `json:"ip"`
	Metadata interface{} `json:"metadata"`
	Resource_action string `json:"resource_action"`
}

// Webhook represents Webhook configuration
type Webhook struct {
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
	OrganizationId string `json:"organizationId"`
	Url string `json:"url"`
	Events []string `json:"events"`
	Secret string `json:"secret"`
	Enabled bool `json:"enabled"`
}

type VerifyRequest struct {
	Otp string `json:"otp"`
	Remember bool `json:"remember"`
	Email string `json:"email"`
}

type ComplianceViolationsResponse struct {
	Violations []*interface{} `json:"violations"`
}

type ComplianceTemplateResponse struct {
	Standard string `json:"standard"`
}

type ContextRule struct {
	Condition string `json:"condition"`
	Description string `json:"description"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type MockSocialAccountRepository struct {
}

type SignOutResponse struct {
	Success bool `json:"success"`
}

type CompliancePoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type CookieConsentConfig struct {
	AllowAnonymous bool `json:"allowAnonymous"`
	BannerVersion string `json:"bannerVersion"`
	Categories []string `json:"categories"`
	DefaultStyle string `json:"defaultStyle"`
	Enabled bool `json:"enabled"`
	RequireExplicit bool `json:"requireExplicit"`
	ValidityPeriod time.Duration `json:"validityPeriod"`
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type BackupAuthVideoResponse struct {
	Session_id string `json:"session_id"`
}

type ConsentStats struct {
	ExpiredCount int `json:"expiredCount"`
	GrantRate float64 `json:"grantRate"`
	GrantedCount int `json:"grantedCount"`
	RevokedCount int `json:"revokedCount"`
	TotalConsents int `json:"totalConsents"`
	Type string `json:"type"`
	AverageLifetime int `json:"averageLifetime"`
}

type ClientRegistrationRequest struct {
	Scope string `json:"scope"`
	Trusted_client bool `json:"trusted_client"`
	Redirect_uris []string `json:"redirect_uris"`
	Application_type string `json:"application_type"`
	Contacts []string `json:"contacts"`
	Policy_uri string `json:"policy_uri"`
	Require_pkce bool `json:"require_pkce"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Logo_uri string `json:"logo_uri"`
	Require_consent bool `json:"require_consent"`
	Response_types []string `json:"response_types"`
	Tos_uri string `json:"tos_uri"`
	Client_name string `json:"client_name"`
	Grant_types []string `json:"grant_types"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
}

type VerifyFactorRequest struct {
	Code string `json:"code"`
}

type RiskAssessmentConfig struct {
	NewIpWeight float64 `json:"newIpWeight"`
	RequireReviewAbove float64 `json:"requireReviewAbove"`
	VelocityWeight float64 `json:"velocityWeight"`
	BlockHighRisk bool `json:"blockHighRisk"`
	Enabled bool `json:"enabled"`
	HighRiskThreshold float64 `json:"highRiskThreshold"`
	LowRiskThreshold float64 `json:"lowRiskThreshold"`
	NewLocationWeight float64 `json:"newLocationWeight"`
	HistoryWeight float64 `json:"historyWeight"`
	MediumRiskThreshold float64 `json:"mediumRiskThreshold"`
	NewDeviceWeight float64 `json:"newDeviceWeight"`
}

type SetupSecurityQuestionsResponse struct {
	Count int `json:"count"`
	Message string `json:"message"`
	SetupAt time.Time `json:"setupAt"`
}

type ComplianceEvidencesResponse struct {
	Evidence []*interface{} `json:"evidence"`
}

type ComplianceViolation struct {
	Description string `json:"description"`
	ProfileId string `json:"profileId"`
	ResolvedAt Time `json:"resolvedAt"`
	Severity string `json:"severity"`
	UserId string `json:"userId"`
	ViolationType string `json:"violationType"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	ResolvedBy string `json:"resolvedBy"`
	Status string `json:"status"`
}

type PolicyEngine struct {
}

type StepUpVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type RenderTemplate_req struct {
	Variables interface{} `json:"variables"`
	Template string `json:"template"`
}

type UserInfoResponse struct {
	Email_verified bool `json:"email_verified"`
	Family_name string `json:"family_name"`
	Gender string `json:"gender"`
	Preferred_username string `json:"preferred_username"`
	Phone_number string `json:"phone_number"`
	Phone_number_verified bool `json:"phone_number_verified"`
	Given_name string `json:"given_name"`
	Locale string `json:"locale"`
	Picture string `json:"picture"`
	Profile string `json:"profile"`
	Updated_at int64 `json:"updated_at"`
	Website string `json:"website"`
	Birthdate string `json:"birthdate"`
	Email string `json:"email"`
	Middle_name string `json:"middle_name"`
	Name string `json:"name"`
	Nickname string `json:"nickname"`
	Sub string `json:"sub"`
	Zoneinfo string `json:"zoneinfo"`
}

type ConsentRecordResponse struct {
	Id string `json:"id"`
}

type FacialCheckConfig struct {
	MotionCapture bool `json:"motionCapture"`
	Variant string `json:"variant"`
	Enabled bool `json:"enabled"`
}

type ListUsersRequest struct {
	Status string `json:"status"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	Role string `json:"role"`
	Search string `json:"search"`
}

type CallbackDataResponse struct {
	Action string `json:"action"`
	IsNewUser bool `json:"isNewUser"`
	User User `json:"user"`
}

type RevokeTokenService struct {
}

type OAuthState struct {
	Link_user_id ID `json:"link_user_id"`
	Provider string `json:"provider"`
	Redirect_url string `json:"redirect_url"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Created_at time.Time `json:"created_at"`
	Extra_scopes []string `json:"extra_scopes"`
}

type StatusRequest struct {
	Device_id string `json:"device_id"`
	User_id string `json:"user_id"`
}

type ContinueRecoveryRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
}

type VerifyCodeResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	Message string `json:"message"`
	Valid bool `json:"valid"`
}

type VerifyCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type ListRecoverySessionsResponse struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Sessions []RecoverySessionInfo `json:"sessions"`
	TotalCount int `json:"totalCount"`
}

type StartVideoSessionResponse struct {
	StartedAt time.Time `json:"startedAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	SessionUrl string `json:"sessionUrl"`
}

type ListPoliciesFilter struct {
	AppId *string `json:"appId"`
	PolicyType *string `json:"policyType"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
}

type ListPasskeysRequest struct {
}

type ProviderRegisteredResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Type string `json:"type"`
}

type SetUserRole_reqBody struct {
	Role string `json:"role"`
}

type Service struct {
}

type WebhookPayload struct {
}

type StepUpVerificationResponse struct {
	Expires_at string `json:"expires_at"`
	Verified bool `json:"verified"`
}

type FinishRegisterResponse struct {
	CredentialId string `json:"credentialId"`
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type BeginLoginResponse struct {
	Challenge string `json:"challenge"`
	Options interface{} `json:"options"`
	Timeout time.Duration `json:"timeout"`
}

type LinkAccountRequest struct {
	Scopes []string `json:"scopes"`
	Provider string `json:"provider"`
}

type CreateVerificationSessionResponse struct {
	Session IdentityVerificationSession `json:"session"`
}

type CreateAPIKeyRequest struct {
	Rate_limit *int `json:"rate_limit,omitempty"`
	Scopes []string `json:"scopes"`
	Allowed_ips *[]string `json:"allowed_ips,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata *interface{} `json:"metadata,omitempty"`
	Name string `json:"name"`
	Permissions *interface{} `json:"permissions,omitempty"`
}

type CreateProfileFromTemplateRequest struct {
	Standard ComplianceStandard `json:"standard"`
}

type DocumentCheckConfig struct {
	ValidateExpiry bool `json:"validateExpiry"`
	Enabled bool `json:"enabled"`
	ExtractData bool `json:"extractData"`
	ValidateDataConsistency bool `json:"validateDataConsistency"`
}

type SMSProviderConfig struct {
	Config interface{} `json:"config"`
	From string `json:"from"`
	Provider string `json:"provider"`
}

type KeyStore struct {
}

type WebAuthnWrapper struct {
}

type ImpersonationContext struct {
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id ID `json:"target_user_id"`
	Impersonation_id ID `json:"impersonation_id"`
	Impersonator_id ID `json:"impersonator_id"`
	Indicator_message string `json:"indicator_message"`
}

type StartImpersonationRequest struct {
	Ticket_number *string `json:"ticket_number,omitempty"`
	Duration_minutes *int `json:"duration_minutes,omitempty"`
	Reason string `json:"reason"`
	Target_user_id string `json:"target_user_id"`
}

type GetRecoveryStatsResponse struct {
	SuccessRate float64 `json:"successRate"`
	SuccessfulRecoveries int `json:"successfulRecoveries"`
	FailedRecoveries int `json:"failedRecoveries"`
	PendingRecoveries int `json:"pendingRecoveries"`
	TotalAttempts int `json:"totalAttempts"`
	AdminReviewsRequired int `json:"adminReviewsRequired"`
	AverageRiskScore float64 `json:"averageRiskScore"`
	HighRiskAttempts int `json:"highRiskAttempts"`
	MethodStats interface{} `json:"methodStats"`
}

type GenerateRecoveryCodesResponse struct {
	Codes []string `json:"codes"`
	Count int `json:"count"`
	GeneratedAt time.Time `json:"generatedAt"`
	Warning string `json:"warning"`
}

type BackupAuthRecoveryResponse struct {
	Session_id string `json:"session_id"`
}

type IDVerificationSessionResponse struct {
	Session interface{} `json:"session"`
}

type IDVerificationResponse struct {
	Verification interface{} `json:"verification"`
}

type MockService struct {
}

type ListRecoverySessionsRequest struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	RequiresReview bool `json:"requiresReview"`
	Status RecoveryStatus `json:"status"`
	OrganizationId string `json:"organizationId"`
}

type CompleteRecoveryResponse struct {
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
	Status RecoveryStatus `json:"status"`
	Token string `json:"token"`
}

type EndImpersonation_reqBody struct {
	Impersonation_id string `json:"impersonation_id"`
	Reason *string `json:"reason,omitempty"`
}

type ReportsConfig struct {
	IncludeEvidence bool `json:"includeEvidence"`
	RetentionDays int `json:"retentionDays"`
	Schedule string `json:"schedule"`
	StoragePath string `json:"storagePath"`
	Enabled bool `json:"enabled"`
	Formats []string `json:"formats"`
}

type ClientSummary struct {
	ApplicationType string `json:"applicationType"`
	ClientID string `json:"clientID"`
	CreatedAt string `json:"createdAt"`
	IsOrgLevel bool `json:"isOrgLevel"`
	Name string `json:"name"`
}

type InvitationResponse struct {
	Invitation Invitation `json:"invitation"`
	Message string `json:"message"`
}

type ChallengeSession struct {
}

type ImpersonateUser_reqBody struct {
	Duration *time.Duration `json:"duration,omitempty"`
}

type ConsentReport struct {
	ConsentRate float64 `json:"consentRate"`
	DataExportsThisPeriod int `json:"dataExportsThisPeriod"`
	DpasExpiringSoon int `json:"dpasExpiringSoon"`
	OrganizationId string `json:"organizationId"`
	ReportPeriodEnd time.Time `json:"reportPeriodEnd"`
	UsersWithConsent int `json:"usersWithConsent"`
	CompletedDeletions int `json:"completedDeletions"`
	ConsentsByType interface{} `json:"consentsByType"`
	DpasActive int `json:"dpasActive"`
	PendingDeletions int `json:"pendingDeletions"`
	ReportPeriodStart time.Time `json:"reportPeriodStart"`
	TotalUsers int `json:"totalUsers"`
}

type AMLMatch struct {
}

type AdminAddProviderRequest struct {
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
	AppId xid.ID `json:"appId"`
	ClientId string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type UpdateUserResponse struct {
	User User `json:"user"`
}

type BackupAuthQuestionsResponse struct {
	Questions []string `json:"questions"`
}

type RegistrationService struct {
}

type RedisStateStore struct {
	Client Client `json:"client,omitempty"`
}

type UpdateClientRequest struct {
	Tos_uri string `json:"tos_uri"`
	Name string `json:"name"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Require_pkce *bool `json:"require_pkce"`
	Response_types []string `json:"response_types"`
	Trusted_client *bool `json:"trusted_client"`
	Allowed_scopes []string `json:"allowed_scopes"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Require_consent *bool `json:"require_consent"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
}

type RateLimitingConfig struct {
	LockoutDuration time.Duration `json:"lockoutDuration"`
	MaxAttemptsPerDay int `json:"maxAttemptsPerDay"`
	MaxAttemptsPerHour int `json:"maxAttemptsPerHour"`
	MaxAttemptsPerIp int `json:"maxAttemptsPerIp"`
	Enabled bool `json:"enabled"`
	ExponentialBackoff bool `json:"exponentialBackoff"`
	IpCooldownPeriod time.Duration `json:"ipCooldownPeriod"`
	LockoutAfterAttempts int `json:"lockoutAfterAttempts"`
}

type ProviderDiscoveredResponse struct {
	Found bool `json:"found"`
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
}

type DeviceInfo struct {
	Name string `json:"name"`
	DeviceId string `json:"deviceId"`
	Metadata interface{} `json:"metadata"`
}

type Handler struct {
}

type BackupAuthCodesResponse struct {
	Codes []string `json:"codes"`
}

type ReviewDocumentRequest struct {
	DocumentId xid.ID `json:"documentId"`
	Notes string `json:"notes"`
	RejectionReason string `json:"rejectionReason"`
	Approved bool `json:"approved"`
}

type GenerateReportRequest struct {
	Period string `json:"period"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
	Format string `json:"format"`
}

type ConsentsResponse struct {
	Consents interface{} `json:"consents"`
	Count int `json:"count"`
}

type ConsentExportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type BlockUserRequest struct {
	Reason string `json:"reason"`
}

type MetadataResponse struct {
	Metadata string `json:"metadata"`
}

type BackupAuthSessionsResponse struct {
	Sessions []*interface{} `json:"sessions"`
}

type ResolveViolationRequest struct {
	Notes string `json:"notes"`
	Resolution string `json:"resolution"`
}

type DeclareABTestWinner_req struct {
	AbTestGroup string `json:"abTestGroup"`
	WinnerId string `json:"winnerId"`
}

type RequestDataDeletionRequest struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type GetVerificationSessionResponse struct {
	Session IdentityVerificationSession `json:"session"`
}

type CompleteVideoSessionResponse struct {
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	Result string `json:"result"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type CreatePolicy_req struct {
	PolicyType string `json:"policyType"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	Version string `json:"version"`
	Content string `json:"content"`
}

type AdminBlockUser_req struct {
	Reason string `json:"reason"`
}

type GenerateReport_req struct {
	Standard ComplianceStandard `json:"standard"`
	Format string `json:"format"`
	Period string `json:"period"`
	ReportType string `json:"reportType"`
}

type AssignRole_reqBody struct {
	RoleID string `json:"roleID"`
}

type Verify_body struct {
	Code string `json:"code"`
	Device_id string `json:"device_id"`
	Remember_device bool `json:"remember_device"`
	User_id string `json:"user_id"`
}

type StartRecoveryRequest struct {
	DeviceId string `json:"deviceId"`
	Email string `json:"email"`
	PreferredMethod RecoveryMethod `json:"preferredMethod"`
	UserId string `json:"userId"`
}

type ComplianceChecksResponse struct {
	Checks []*interface{} `json:"checks"`
}

type JWKSService struct {
}

type GetPolicyResponse struct {
	Allowed_factor_types []string `json:"allowed_factor_types"`
	Enabled bool `json:"enabled"`
	Required_factor_count int `json:"required_factor_count"`
}

type MFAPolicyResponse struct {
	RequiredFactorCount int `json:"requiredFactorCount"`
	AllowedFactorTypes []string `json:"allowedFactorTypes"`
	AppId xid.ID `json:"appId"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
	OrganizationId ID `json:"organizationId"`
}

type CreateEvidence_req struct {
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	ControlId string `json:"controlId"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
}

type FinishLoginRequest struct {
	Remember bool `json:"remember"`
	Response interface{} `json:"response"`
}

type TwoFAEnableResponse struct {
	Totp_uri string `json:"totp_uri"`
	Status string `json:"status"`
}

type UpdateRequest struct {
	Id string `json:"id"`
	Url *string `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Enabled *bool `json:"enabled,omitempty"`
}

type ComplianceUserTrainingResponse struct {
	User_id string `json:"user_id"`
}

type DataExportRequestInput struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type ProviderSession struct {
}

type EvaluationContext struct {
}

type SetUserRoleRequest struct {
	App_id xid.ID `json:"app_id"`
	Role string `json:"role"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
}

type SMSConfig struct {
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
}

type RecoveryAttemptLog struct {
}

type UploadDocumentRequest struct {
	BackImage string `json:"backImage"`
	DocumentType string `json:"documentType"`
	FrontImage string `json:"frontImage"`
	Selfie string `json:"selfie"`
	SessionId xid.ID `json:"sessionId"`
}

type CreateEvidenceRequest struct {
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	ControlId string `json:"controlId"`
	Description string `json:"description"`
}

type ForgetDeviceResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

type PreviewTemplate_req struct {
	Variables interface{} `json:"variables"`
}

type userServiceAdapter struct {
}

type AdminGetUserVerificationsResponse struct {
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Total int `json:"total"`
	Verifications IdentityVerification `json:"verifications"`
}

type OrganizationHandler struct {
}

type ResetUserMFAResponse struct {
	DevicesRevoked int `json:"devicesRevoked"`
	FactorsReset int `json:"factorsReset"`
	Message string `json:"message"`
	Success bool `json:"success"`
}

type SendVerificationCodeRequest struct {
	SessionId xid.ID `json:"sessionId"`
	Target string `json:"target"`
	Method RecoveryMethod `json:"method"`
}

type VerificationSessionResponse struct {
	Session IdentityVerificationSession `json:"session"`
}

type ProviderSessionRequest struct {
}

type ImpersonationVerifyResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id string `json:"target_user_id"`
}

type ProviderInfo struct {
	CreatedAt string `json:"createdAt"`
	Domain string `json:"domain"`
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
}

type TOTPFactorAdapter struct {
}

type DocumentVerification struct {
}

type TrustedContactInfo struct {
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
	Verified bool `json:"verified"`
	VerifiedAt Time `json:"verifiedAt"`
	Active bool `json:"active"`
	Email string `json:"email"`
	Id xid.ID `json:"id"`
	Name string `json:"name"`
}

type UpdateRecoveryConfigRequest struct {
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
}

type ComplianceStatusResponse struct {
	Status string `json:"status"`
}

type ReviewDocumentResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

// Device represents User device
type Device struct {
	UserAgent *string `json:"userAgent,omitempty"`
	Id string `json:"id"`
	UserId string `json:"userId"`
	Name *string `json:"name,omitempty"`
	Type *string `json:"type,omitempty"`
	LastUsedAt string `json:"lastUsedAt"`
	IpAddress *string `json:"ipAddress,omitempty"`
}

type RejectRecoveryRequest struct {
	Notes string `json:"notes"`
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type RetentionConfig struct {
	ArchiveBeforePurge bool `json:"archiveBeforePurge"`
	ArchivePath string `json:"archivePath"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	PurgeSchedule string `json:"purgeSchedule"`
}

type ReverifyRequest struct {
	Reason string `json:"reason"`
}

type TimeBasedRule struct {
	Description string `json:"description"`
	Max_age time.Duration `json:"max_age"`
	Operation string `json:"operation"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type ClientDetailsResponse struct {
	AllowedScopes []string `json:"allowedScopes"`
	IsOrgLevel bool `json:"isOrgLevel"`
	Name string `json:"name"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	TosURI string `json:"tosURI"`
	ApplicationType string `json:"applicationType"`
	ClientID string `json:"clientID"`
	Contacts []string `json:"contacts"`
	CreatedAt string `json:"createdAt"`
	RedirectURIs []string `json:"redirectURIs"`
	ResponseTypes []string `json:"responseTypes"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	GrantTypes []string `json:"grantTypes"`
	LogoURI string `json:"logoURI"`
	PolicyURI string `json:"policyURI"`
	RequireConsent bool `json:"requireConsent"`
	TrustedClient bool `json:"trustedClient"`
	UpdatedAt string `json:"updatedAt"`
	OrganizationID string `json:"organizationID"`
	RequirePKCE bool `json:"requirePKCE"`
}

type OTPSentResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type ComplianceTrainingsResponse struct {
	Training []*interface{} `json:"training"`
}

type AdminHandler struct {
}

type ListFactorsResponse struct {
	Count int `json:"count"`
	Factors []Factor `json:"factors"`
}

type ComplianceViolationResponse struct {
	Id string `json:"id"`
}

type EmailConfig struct {
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
}

type DeleteFactorRequest struct {
}

type UploadDocumentResponse struct {
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
	ProcessingTime string `json:"processingTime"`
	Status string `json:"status"`
	UploadedAt time.Time `json:"uploadedAt"`
}

type ApproveRecoveryResponse struct {
	Approved bool `json:"approved"`
	ApprovedAt time.Time `json:"approvedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
}

type ComplianceTrainingResponse struct {
	Id string `json:"id"`
}

type ConsentRecord struct {
	ExpiresAt Time `json:"expiresAt"`
	Metadata JSONBMap `json:"metadata"`
	OrganizationId string `json:"organizationId"`
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	CreatedAt time.Time `json:"createdAt"`
	GrantedAt time.Time `json:"grantedAt"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	Purpose string `json:"purpose"`
	RevokedAt Time `json:"revokedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserId string `json:"userId"`
	Granted bool `json:"granted"`
	UserAgent string `json:"userAgent"`
}

type CreateVerificationSession_req struct {
	Config interface{} `json:"config"`
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
}

type CompleteTrainingRequest struct {
	Score int `json:"score"`
}

type ComplianceReportResponse struct {
	Id string `json:"id"`
}

type CreateAPIKey_reqBody struct {
	Rate_limit *int `json:"rate_limit,omitempty"`
	Scopes []string `json:"scopes"`
	Allowed_ips *[]string `json:"allowed_ips,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata *interface{} `json:"metadata,omitempty"`
	Name string `json:"name"`
	Permissions *interface{} `json:"permissions,omitempty"`
}

type MultiSessionListResponse struct {
	Sessions []*interface{} `json:"sessions"`
}

type ClientUpdateRequest struct {
	Allowed_scopes []string `json:"allowed_scopes"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
	Require_consent *bool `json:"require_consent"`
	Require_pkce *bool `json:"require_pkce"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Logo_uri string `json:"logo_uri"`
	Name string `json:"name"`
	Policy_uri string `json:"policy_uri"`
	Response_types []string `json:"response_types"`
	Tos_uri string `json:"tos_uri"`
	Trusted_client *bool `json:"trusted_client"`
}

type TOTPSecret struct {
}

type RemoveTrustedContactResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type LinkAccountResponse struct {
	Url string `json:"url"`
}

// StatusResponse represents Status response
type StatusResponse struct {
	Status string `json:"status"`
}

type AddTrustedContactRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
}

type RecoveryCodesConfig struct {
	AllowPrint bool `json:"allowPrint"`
	AutoRegenerate bool `json:"autoRegenerate"`
	CodeCount int `json:"codeCount"`
	CodeLength int `json:"codeLength"`
	Enabled bool `json:"enabled"`
	Format string `json:"format"`
	RegenerateCount int `json:"regenerateCount"`
	AllowDownload bool `json:"allowDownload"`
}

type ComplianceEvidence struct {
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileHash string `json:"fileHash"`
	FileUrl string `json:"fileUrl"`
	Id string `json:"id"`
	ProfileId string `json:"profileId"`
	AppId string `json:"appId"`
	CollectedBy string `json:"collectedBy"`
	ControlId string `json:"controlId"`
	CreatedAt time.Time `json:"createdAt"`
	Metadata interface{} `json:"metadata"`
}

type ComplianceDashboardResponse struct {
	Metrics interface{} `json:"metrics"`
}

type StepUpPolicy struct {
	Created_at time.Time `json:"created_at"`
	Description string `json:"description"`
	Enabled bool `json:"enabled"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
	Rules interface{} `json:"rules"`
	Priority int `json:"priority"`
	Updated_at time.Time `json:"updated_at"`
	User_id string `json:"user_id"`
}

type AuditConfig struct {
	LogSuccessful bool `json:"logSuccessful"`
	LogUserAgent bool `json:"logUserAgent"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	LogAllAttempts bool `json:"logAllAttempts"`
	LogFailed bool `json:"logFailed"`
	RetentionDays int `json:"retentionDays"`
	Enabled bool `json:"enabled"`
	ImmutableLogs bool `json:"immutableLogs"`
	LogDeviceInfo bool `json:"logDeviceInfo"`
	LogIpAddress bool `json:"logIpAddress"`
}

type CheckSubResult struct {
}

type EvaluationResult struct {
	Challenge_token string `json:"challenge_token"`
	Current_level SecurityLevel `json:"current_level"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
	Security_level SecurityLevel `json:"security_level"`
	Can_remember bool `json:"can_remember"`
	Expires_at time.Time `json:"expires_at"`
	Grace_period_ends_at time.Time `json:"grace_period_ends_at"`
	Matched_rules []string `json:"matched_rules"`
	Required bool `json:"required"`
	Requirement_id string `json:"requirement_id"`
	Allowed_methods []VerificationMethod `json:"allowed_methods"`
}

type StepUpRememberedDevice struct {
	User_id string `json:"user_id"`
	Device_name string `json:"device_name"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Last_used_at time.Time `json:"last_used_at"`
	Remembered_at time.Time `json:"remembered_at"`
	User_agent string `json:"user_agent"`
	Created_at time.Time `json:"created_at"`
	Device_id string `json:"device_id"`
	Expires_at time.Time `json:"expires_at"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type StepUpRequirementsResponse struct {
	Requirements []*interface{} `json:"requirements"`
}

type NotificationTemplateListResponse struct {
	Total int `json:"total"`
	Templates []*interface{} `json:"templates"`
}

type TemplateService struct {
}

type DiscoveryResponse struct {
	Request_parameter_supported bool `json:"request_parameter_supported"`
	Request_uri_parameter_supported bool `json:"request_uri_parameter_supported"`
	Token_endpoint_auth_methods_supported []string `json:"token_endpoint_auth_methods_supported"`
	Claims_parameter_supported bool `json:"claims_parameter_supported"`
	Grant_types_supported []string `json:"grant_types_supported"`
	Issuer string `json:"issuer"`
	Jwks_uri string `json:"jwks_uri"`
	Response_types_supported []string `json:"response_types_supported"`
	Subject_types_supported []string `json:"subject_types_supported"`
	Userinfo_endpoint string `json:"userinfo_endpoint"`
	Require_request_uri_registration bool `json:"require_request_uri_registration"`
	Authorization_endpoint string `json:"authorization_endpoint"`
	Scopes_supported []string `json:"scopes_supported"`
	Token_endpoint string `json:"token_endpoint"`
	Introspection_endpoint string `json:"introspection_endpoint"`
	Response_modes_supported []string `json:"response_modes_supported"`
	Revocation_endpoint string `json:"revocation_endpoint"`
	Revocation_endpoint_auth_methods_supported []string `json:"revocation_endpoint_auth_methods_supported"`
	Claims_supported []string `json:"claims_supported"`
	Code_challenge_methods_supported []string `json:"code_challenge_methods_supported"`
	Id_token_signing_alg_values_supported []string `json:"id_token_signing_alg_values_supported"`
	Introspection_endpoint_auth_methods_supported []string `json:"introspection_endpoint_auth_methods_supported"`
	Registration_endpoint string `json:"registration_endpoint"`
}

type VerificationResult struct {
}

type Adapter struct {
}

type UpdateProvider_req struct {
	IsDefault bool `json:"isDefault"`
	Config interface{} `json:"config"`
	IsActive bool `json:"isActive"`
}

type TokenIntrospectionRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type RateLimitRule struct {
	Max int `json:"max"`
	Window time.Duration `json:"window"`
}

type RateLimitConfig struct {
	Lockout_minutes int `json:"lockout_minutes"`
	Max_attempts int `json:"max_attempts"`
	Window_minutes int `json:"window_minutes"`
	Enabled bool `json:"enabled"`
}

type InitiateChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata interface{} `json:"metadata"`
}

type BaseFactorAdapter struct {
}

type VideoSessionInfo struct {
}

type CompleteTraining_req struct {
	Score int `json:"score"`
}

type ConsentNotificationsConfig struct {
	NotifyDeletionComplete bool `json:"notifyDeletionComplete"`
	NotifyExportReady bool `json:"notifyExportReady"`
	NotifyOnExpiry bool `json:"notifyOnExpiry"`
	NotifyOnGrant bool `json:"notifyOnGrant"`
	NotifyOnRevoke bool `json:"notifyOnRevoke"`
	Channels []string `json:"channels"`
	Enabled bool `json:"enabled"`
	NotifyDeletionApproved bool `json:"notifyDeletionApproved"`
	NotifyDpoEmail string `json:"notifyDpoEmail"`
}

type AddMember_req struct {
	Role string `json:"role"`
	User_id string `json:"user_id"`
}

type NotificationStatusResponse struct {
	Status string `json:"status"`
}

type MFASession struct {
	IpAddress string `json:"ipAddress"`
	Metadata interface{} `json:"metadata"`
	RiskLevel RiskLevel `json:"riskLevel"`
	FactorsRequired int `json:"factorsRequired"`
	SessionToken string `json:"sessionToken"`
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
	VerifiedFactors ID `json:"verifiedFactors"`
	CompletedAt Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsVerified int `json:"factorsVerified"`
	Id xid.ID `json:"id"`
}

type VerifyTrustedContactResponse struct {
	ContactId xid.ID `json:"contactId"`
	Message string `json:"message"`
	Verified bool `json:"verified"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

type VideoSessionResult struct {
}

type NotificationPreviewResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

type ImpersonationErrorResponse struct {
	Error string `json:"error"`
}

type RevokeConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
}

type CreateConsentPolicyRequest struct {
	ValidityPeriod *int `json:"validityPeriod"`
	ConsentType string `json:"consentType"`
	Version string `json:"version"`
	Content string `json:"content"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Renewable bool `json:"renewable"`
	Required bool `json:"required"`
}

type RequestReverification_req struct {
	Reason string `json:"reason"`
}

type ListRememberedDevicesResponse struct {
	Devices interface{} `json:"devices"`
	Count int `json:"count"`
}

type SetActiveRequest struct {
	Id string `json:"id"`
}

type PrivacySettings struct {
	CookieConsentStyle string `json:"cookieConsentStyle"`
	RequireAdminApprovalForDeletion bool `json:"requireAdminApprovalForDeletion"`
	AnonymousConsentEnabled bool `json:"anonymousConsentEnabled"`
	CookieConsentEnabled bool `json:"cookieConsentEnabled"`
	DeletionGracePeriodDays int `json:"deletionGracePeriodDays"`
	OrganizationId string `json:"organizationId"`
	AutoDeleteAfterDays int `json:"autoDeleteAfterDays"`
	ContactPhone string `json:"contactPhone"`
	DataExportExpiryHours int `json:"dataExportExpiryHours"`
	DpoEmail string `json:"dpoEmail"`
	GdprMode bool `json:"gdprMode"`
	Id xid.ID `json:"id"`
	Metadata JSONBMap `json:"metadata"`
	AllowDataPortability bool `json:"allowDataPortability"`
	CreatedAt time.Time `json:"createdAt"`
	DataRetentionDays int `json:"dataRetentionDays"`
	ExportFormat []string `json:"exportFormat"`
	RequireExplicitConsent bool `json:"requireExplicitConsent"`
	UpdatedAt time.Time `json:"updatedAt"`
	CcpaMode bool `json:"ccpaMode"`
	ConsentRequired bool `json:"consentRequired"`
	ContactEmail string `json:"contactEmail"`
}

type IDVerificationWebhookResponse struct {
	Status string `json:"status"`
}

type StepUpAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type AccessTokenClaims struct {
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
}

type PhoneVerifyResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type AdminUpdatePolicyRequest struct {
	AllowedTypes []string `json:"allowedTypes"`
	Enabled bool `json:"enabled"`
	GracePeriod int `json:"gracePeriod"`
	RequiredFactors int `json:"requiredFactors"`
}

type ListPendingRequirementsResponse struct {
	Count int `json:"count"`
	Requirements interface{} `json:"requirements"`
}

type SendOTPRequest struct {
	User_id string `json:"user_id"`
}

type VerificationRequest struct {
	RememberDevice bool `json:"rememberDevice"`
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data interface{} `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
}

type ListTrustedDevicesResponse struct {
	Count int `json:"count"`
	Devices []TrustedDevice `json:"devices"`
}

type SecurityQuestionInfo struct {
	Id xid.ID `json:"id"`
	IsCustom bool `json:"isCustom"`
	QuestionId int `json:"questionId"`
	QuestionText string `json:"questionText"`
}

type VideoVerificationConfig struct {
	Enabled bool `json:"enabled"`
	LivenessThreshold float64 `json:"livenessThreshold"`
	MinScheduleAdvance time.Duration `json:"minScheduleAdvance"`
	RecordSessions bool `json:"recordSessions"`
	SessionDuration time.Duration `json:"sessionDuration"`
	Provider string `json:"provider"`
	RecordingRetention time.Duration `json:"recordingRetention"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireLivenessCheck bool `json:"requireLivenessCheck"`
	RequireScheduling bool `json:"requireScheduling"`
}

type MemoryStateStore struct {
}

type VideoVerificationSession struct {
}

type AppHandler struct {
}

type UpdatePolicyRequest struct {
	StepUpRequired *bool `json:"stepUpRequired"`
	AdaptiveMfaEnabled *bool `json:"adaptiveMfaEnabled"`
	AllowedFactorTypes []FactorType `json:"allowedFactorTypes"`
	GracePeriodDays *int `json:"gracePeriodDays"`
	LockoutDurationMinutes *int `json:"lockoutDurationMinutes"`
	MaxFailedAttempts *int `json:"maxFailedAttempts"`
	TrustedDeviceDays *int `json:"trustedDeviceDays"`
	RequiredFactorCount *int `json:"requiredFactorCount"`
	RequiredFactorTypes []FactorType `json:"requiredFactorTypes"`
}

type Config struct {
	Challenge_expiry_minutes int `json:"challenge_expiry_minutes"`
	Email EmailConfig `json:"email"`
	Enabled bool `json:"enabled"`
	Rate_limit RateLimitConfig `json:"rate_limit"`
	Required_factor_count int `json:"required_factor_count"`
	Allowed_factor_types []FactorType `json:"allowed_factor_types"`
	Require_for_all_users bool `json:"require_for_all_users"`
	Trusted_devices TrustedDevicesConfig `json:"trusted_devices"`
	Max_attempts int `json:"max_attempts"`
	Session_expiry_minutes int `json:"session_expiry_minutes"`
	Webauthn WebAuthnConfig `json:"webauthn"`
	Adaptive_mfa AdaptiveMFAConfig `json:"adaptive_mfa"`
	Backup_codes BackupCodesConfig `json:"backup_codes"`
	Grace_period_days int `json:"grace_period_days"`
	Sms SMSConfig `json:"sms"`
	Totp TOTPConfig `json:"totp"`
}

type NoOpSMSProvider struct {
}

type VerifyRecoveryCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type RemoveTrustedContactRequest struct {
	ContactId xid.ID `json:"contactId"`
}

type ComplianceProfile struct {
	ComplianceContact string `json:"complianceContact"`
	DataResidency string `json:"dataResidency"`
	AppId string `json:"appId"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	DpoContact string `json:"dpoContact"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	Id string `json:"id"`
	Name string `json:"name"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	RetentionDays int `json:"retentionDays"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	PasswordMinLength int `json:"passwordMinLength"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	Status string `json:"status"`
	LeastPrivilege bool `json:"leastPrivilege"`
	MfaRequired bool `json:"mfaRequired"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	AuditLogExport bool `json:"auditLogExport"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	RegularAccessReview bool `json:"regularAccessReview"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	Standards []ComplianceStandard `json:"standards"`
	CreatedAt time.Time `json:"createdAt"`
	Metadata interface{} `json:"metadata"`
	RbacRequired bool `json:"rbacRequired"`
	SessionMaxAge int `json:"sessionMaxAge"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type IDVerificationErrorResponse struct {
	Error string `json:"error"`
}

type StepUpAuditLog struct {
	Event_type string `json:"event_type"`
	Ip string `json:"ip"`
	Org_id string `json:"org_id"`
	Severity string `json:"severity"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Event_data interface{} `json:"event_data"`
	Id string `json:"id"`
	Created_at time.Time `json:"created_at"`
}

type FactorsResponse struct {
	Count int `json:"count"`
	Factors interface{} `json:"factors"`
}

type RiskAssessment struct {
	Metadata interface{} `json:"metadata"`
	Recommended []FactorType `json:"recommended"`
	Score float64 `json:"score"`
	Factors []string `json:"factors"`
	Level RiskLevel `json:"level"`
}

type CompleteRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type NoOpVideoProvider struct {
}

type ConsentPolicy struct {
	Description string `json:"description"`
	Renewable bool `json:"renewable"`
	ValidityPeriod *int `json:"validityPeriod"`
	ConsentType string `json:"consentType"`
	Content string `json:"content"`
	Name string `json:"name"`
	OrganizationId string `json:"organizationId"`
	CreatedAt time.Time `json:"createdAt"`
	PublishedAt Time `json:"publishedAt"`
	Required bool `json:"required"`
	Version string `json:"version"`
	CreatedBy string `json:"createdBy"`
	Id xid.ID `json:"id"`
	Metadata JSONBMap `json:"metadata"`
	UpdatedAt time.Time `json:"updatedAt"`
	Active bool `json:"active"`
}

type ConsentService struct {
}

type GetUserVerificationStatusResponse struct {
	Status UserVerificationStatus `json:"status"`
}

type DeleteResponse struct {
	Success bool `json:"success"`
}

type ConsentAuditConfig struct {
	ArchiveInterval time.Duration `json:"archiveInterval"`
	Enabled bool `json:"enabled"`
	Immutable bool `json:"immutable"`
	LogIpAddress bool `json:"logIpAddress"`
	LogUserAgent bool `json:"logUserAgent"`
	RetentionDays int `json:"retentionDays"`
	SignLogs bool `json:"signLogs"`
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	ExportFormat string `json:"exportFormat"`
	LogAllChanges bool `json:"logAllChanges"`
}

type TwoFABackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type GetChallengeStatusRequest struct {
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type StateStorageConfig struct {
	RedisAddr string `json:"redisAddr"`
	RedisDb int `json:"redisDb"`
	RedisPassword string `json:"redisPassword"`
	StateTtl time.Duration `json:"stateTtl"`
	UseRedis bool `json:"useRedis"`
}

type UpdateRecoveryConfigResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type UpdatePrivacySettingsRequest struct {
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	DataRetentionDays *int `json:"dataRetentionDays"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	DpoEmail string `json:"dpoEmail"`
	CcpaMode *bool `json:"ccpaMode"`
	ConsentRequired *bool `json:"consentRequired"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	GdprMode *bool `json:"gdprMode"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	ExportFormat []string `json:"exportFormat"`
	AllowDataPortability *bool `json:"allowDataPortability"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
}

type BackupAuthContactResponse struct {
	Id string `json:"id"`
}

type StripeIdentityConfig struct {
	Enabled bool `json:"enabled"`
	RequireLiveCapture bool `json:"requireLiveCapture"`
	RequireMatchingSelfie bool `json:"requireMatchingSelfie"`
	ReturnUrl string `json:"returnUrl"`
	UseMock bool `json:"useMock"`
	WebhookSecret string `json:"webhookSecret"`
	AllowedTypes []string `json:"allowedTypes"`
	ApiKey string `json:"apiKey"`
}

type EmailProviderConfig struct {
	Config interface{} `json:"config"`
	From string `json:"from"`
	From_name string `json:"from_name"`
	Provider string `json:"provider"`
	Reply_to string `json:"reply_to"`
}

type OIDCCallbackResponse struct {
	User User `json:"user"`
	Session Session `json:"session"`
	Token string `json:"token"`
}

type ListSessionsRequest struct {
	App_id xid.ID `json:"app_id"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	User_id ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
}

type AuthURLResponse struct {
	Url string `json:"url"`
}

type GenerateBackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type SendRequest struct {
	Email string `json:"email"`
}

type BackupAuthConfigResponse struct {
	Config interface{} `json:"config"`
}

type DashboardExtension struct {
}

type SAMLLoginResponse struct {
	ProviderId string `json:"providerId"`
	RedirectUrl string `json:"redirectUrl"`
	RequestId string `json:"requestId"`
}

type UpdateFactorRequest struct {
	Metadata interface{} `json:"metadata"`
	Name *string `json:"name"`
	Priority *FactorPriority `json:"priority"`
	Status *FactorStatus `json:"status"`
}

type GetStatusRequest struct {
}

type DataProcessingAgreement struct {
	ExpiryDate Time `json:"expiryDate"`
	Metadata JSONBMap `json:"metadata"`
	SignedByTitle string `json:"signedByTitle"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedAt time.Time `json:"createdAt"`
	DigitalSignature string `json:"digitalSignature"`
	EffectiveDate time.Time `json:"effectiveDate"`
	IpAddress string `json:"ipAddress"`
	SignedByName string `json:"signedByName"`
	Status string `json:"status"`
	Version string `json:"version"`
	SignedBy string `json:"signedBy"`
	SignedByEmail string `json:"signedByEmail"`
	AgreementType string `json:"agreementType"`
	Content string `json:"content"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
}

type RolesResponse struct {
	Roles Role `json:"roles"`
}

type UpdatePasskeyResponse struct {
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type OIDCLoginResponse struct {
	AuthUrl string `json:"authUrl"`
	Nonce string `json:"nonce"`
	ProviderId string `json:"providerId"`
	State string `json:"state"`
}

type RequestReverificationRequest struct {
	Reason string `json:"reason"`
}

type AddTeamMemberRequest struct {
	Member_id xid.ID `json:"member_id"`
	Role string `json:"role"`
}

type ChallengeRequest struct {
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata interface{} `json:"metadata"`
	UserId xid.ID `json:"userId"`
	Context string `json:"context"`
}

type ListTrustedContactsResponse struct {
	Contacts []TrustedContactInfo `json:"contacts"`
	Count int `json:"count"`
}

type IDVerificationListResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type RotateAPIKeyResponse struct {
	Api_key APIKey `json:"api_key"`
	Message string `json:"message"`
}

type StartRecoveryResponse struct {
	RiskScore float64 `json:"riskScore"`
	SessionId xid.ID `json:"sessionId"`
	Status RecoveryStatus `json:"status"`
	AvailableMethods []RecoveryMethod `json:"availableMethods"`
	CompletedSteps int `json:"completedSteps"`
	ExpiresAt time.Time `json:"expiresAt"`
	RequiredSteps int `json:"requiredSteps"`
	RequiresReview bool `json:"requiresReview"`
}

type TrustedContactsConfig struct {
	AllowEmailContacts bool `json:"allowEmailContacts"`
	MaximumContacts int `json:"maximumContacts"`
	RequireVerification bool `json:"requireVerification"`
	AllowPhoneContacts bool `json:"allowPhoneContacts"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	Enabled bool `json:"enabled"`
	MaxNotificationsPerDay int `json:"maxNotificationsPerDay"`
	MinimumContacts int `json:"minimumContacts"`
	RequiredToRecover int `json:"requiredToRecover"`
	VerificationExpiry time.Duration `json:"verificationExpiry"`
}

type CreateDPARequest struct {
	ExpiryDate Time `json:"expiryDate"`
	SignedByEmail string `json:"signedByEmail"`
	SignedByName string `json:"signedByName"`
	SignedByTitle string `json:"signedByTitle"`
	Version string `json:"version"`
	AgreementType string `json:"agreementType"`
	Content string `json:"content"`
	Metadata interface{} `json:"metadata"`
	EffectiveDate time.Time `json:"effectiveDate"`
}

type ResourceRule struct {
	Sensitivity string `json:"sensitivity"`
	Action string `json:"action"`
	Description string `json:"description"`
	Org_id string `json:"org_id"`
	Resource_type string `json:"resource_type"`
	Security_level SecurityLevel `json:"security_level"`
}

type NotificationListResponse struct {
	Notifications []*interface{} `json:"notifications"`
	Total int `json:"total"`
}

type ListFactorsRequest struct {
}

type Email struct {
}

type VerificationsResponse struct {
	Count int `json:"count"`
	Verifications interface{} `json:"verifications"`
}

type OAuthErrorResponse struct {
	Error string `json:"error"`
	Error_description string `json:"error_description"`
	Error_uri string `json:"error_uri"`
	State string `json:"state"`
}

type ResetUserMFARequest struct {
	Reason string `json:"reason"`
}

type FactorEnrollmentRequest struct {
	Type FactorType `json:"type"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
}

type ListEvidenceFilter struct {
	Standard *ComplianceStandard `json:"standard"`
	AppId *string `json:"appId"`
	ControlId *string `json:"controlId"`
	EvidenceType *string `json:"evidenceType"`
	ProfileId *string `json:"profileId"`
}

type ConsentSettingsResponse struct {
	Settings interface{} `json:"settings"`
}

type RouteRule struct {
	Description string `json:"description"`
	Method string `json:"method"`
	Org_id string `json:"org_id"`
	Pattern string `json:"pattern"`
	Security_level SecurityLevel `json:"security_level"`
}

type BeginRegisterResponse struct {
	Timeout time.Duration `json:"timeout"`
	UserId string `json:"userId"`
	Challenge string `json:"challenge"`
	Options interface{} `json:"options"`
}

type ImpersonationSession struct {
}

type RequestTrustedContactVerificationResponse struct {
	Message string `json:"message"`
	NotifiedAt time.Time `json:"notifiedAt"`
	ContactId xid.ID `json:"contactId"`
	ContactName string `json:"contactName"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type MockRepository struct {
}

type MemberHandler struct {
}

type RequirementsResponse struct {
	Count int `json:"count"`
	Requirements interface{} `json:"requirements"`
}

type CookieConsent struct {
	ThirdParty bool `json:"thirdParty"`
	UserId string `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	SessionId string `json:"sessionId"`
	UpdatedAt time.Time `json:"updatedAt"`
	Analytics bool `json:"analytics"`
	Functional bool `json:"functional"`
	IpAddress string `json:"ipAddress"`
	Personalization bool `json:"personalization"`
	Essential bool `json:"essential"`
	Id xid.ID `json:"id"`
	Marketing bool `json:"marketing"`
	UserAgent string `json:"userAgent"`
	ConsentBannerVersion string `json:"consentBannerVersion"`
	CreatedAt time.Time `json:"createdAt"`
	OrganizationId string `json:"organizationId"`
}

type StepUpPoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type FinishRegisterRequest struct {
	Name string `json:"name"`
	Response interface{} `json:"response"`
	UserId string `json:"userId"`
}

type IDTokenClaims struct {
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
	Name string `json:"name"`
	Preferred_username string `json:"preferred_username"`
	Session_state string `json:"session_state"`
	Auth_time int64 `json:"auth_time"`
	Family_name string `json:"family_name"`
	Given_name string `json:"given_name"`
	Nonce string `json:"nonce"`
}

type CompleteVideoSessionRequest struct {
	LivenessPassed bool `json:"livenessPassed"`
	LivenessScore float64 `json:"livenessScore"`
	Notes string `json:"notes"`
	VerificationResult string `json:"verificationResult"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type CreateSessionHTTPRequest struct {
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
}

type mockProvider struct {
}

type StepUpVerification struct {
	Metadata interface{} `json:"metadata"`
	Session_id string `json:"session_id"`
	Device_id string `json:"device_id"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Reason string `json:"reason"`
	Security_level SecurityLevel `json:"security_level"`
	User_agent string `json:"user_agent"`
	Method VerificationMethod `json:"method"`
	Rule_name string `json:"rule_name"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Org_id string `json:"org_id"`
	Verified_at time.Time `json:"verified_at"`
	Expires_at time.Time `json:"expires_at"`
}

type CreateRequest struct {
	Url string `json:"url"`
	Events []string `json:"events"`
	Secret *string `json:"secret,omitempty"`
}

type ClientRegistrationResponse struct {
	Client_name string `json:"client_name"`
	Contacts []string `json:"contacts"`
	Logo_uri string `json:"logo_uri"`
	Policy_uri string `json:"policy_uri"`
	Scope string `json:"scope"`
	Tos_uri string `json:"tos_uri"`
	Application_type string `json:"application_type"`
	Client_id_issued_at int64 `json:"client_id_issued_at"`
	Client_secret string `json:"client_secret"`
	Grant_types []string `json:"grant_types"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Client_secret_expires_at int64 `json:"client_secret_expires_at"`
	Response_types []string `json:"response_types"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Client_id string `json:"client_id"`
	Redirect_uris []string `json:"redirect_uris"`
}

// User represents User account
type User struct {
	Name *string `json:"name,omitempty"`
	EmailVerified bool `json:"emailVerified"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	OrganizationId *string `json:"organizationId,omitempty"`
	Id string `json:"id"`
	Email string `json:"email"`
}

type ApproveRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
	Notes string `json:"notes"`
}

type MockAppService struct {
}

type CreateConsentRequest struct {
	ConsentType string `json:"consentType"`
	ExpiresIn *int `json:"expiresIn"`
	Granted bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Purpose string `json:"purpose"`
	UserId string `json:"userId"`
	Version string `json:"version"`
}

type ConsentExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type DataExportRequest struct {
	ExportPath string `json:"exportPath"`
	ExportSize int64 `json:"exportSize"`
	Format string `json:"format"`
	OrganizationId string `json:"organizationId"`
	CompletedAt Time `json:"completedAt"`
	UserId string `json:"userId"`
	UpdatedAt time.Time `json:"updatedAt"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	ErrorMessage string `json:"errorMessage"`
	ExpiresAt Time `json:"expiresAt"`
	ExportUrl string `json:"exportUrl"`
	Id xid.ID `json:"id"`
	IncludeSections []string `json:"includeSections"`
	IpAddress string `json:"ipAddress"`
}

type ConnectionsResponse struct {
	Connections SocialAccount `json:"connections"`
}

type SecurityQuestionsConfig struct {
	AllowCustomQuestions bool `json:"allowCustomQuestions"`
	CaseSensitive bool `json:"caseSensitive"`
	ForbidCommonAnswers bool `json:"forbidCommonAnswers"`
	LockoutDuration time.Duration `json:"lockoutDuration"`
	MaxAttempts int `json:"maxAttempts"`
	MinimumQuestions int `json:"minimumQuestions"`
	PredefinedQuestions []string `json:"predefinedQuestions"`
	Enabled bool `json:"enabled"`
	MaxAnswerLength int `json:"maxAnswerLength"`
	RequireMinLength int `json:"requireMinLength"`
	RequiredToRecover int `json:"requiredToRecover"`
}

type RecoveryCodeUsage struct {
}

type TeamHandler struct {
}

type TemplateEngine struct {
}

type DiscoveryService struct {
}

type LoginResponse struct {
	User interface{} `json:"user"`
	PasskeyUsed string `json:"passkeyUsed"`
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type PasskeyInfo struct {
	IsResidentKey bool `json:"isResidentKey"`
	LastUsedAt Time `json:"lastUsedAt"`
	Name string `json:"name"`
	SignCount uint `json:"signCount"`
	AuthenticatorType string `json:"authenticatorType"`
	CreatedAt time.Time `json:"createdAt"`
	CredentialId string `json:"credentialId"`
	Aaguid string `json:"aaguid"`
	Id string `json:"id"`
}

type VerificationListResponse struct {
	Verifications IdentityVerification `json:"verifications"`
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Total int `json:"total"`
}

type StepUpRequirementResponse struct {
	Id string `json:"id"`
}

type RecordCookieConsentRequest struct {
	Essential bool `json:"essential"`
	Functional bool `json:"functional"`
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	Analytics bool `json:"analytics"`
	BannerVersion string `json:"bannerVersion"`
}

type RateLimiter struct {
}

type TrustDeviceRequest struct {
	DeviceId string `json:"deviceId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
}

type GetFactorRequest struct {
}

type HealthCheckResponse struct {
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	Healthy bool `json:"healthy"`
	Message string `json:"message"`
	ProvidersStatus interface{} `json:"providersStatus"`
	Version string `json:"version"`
}

type CreateTraining_req struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type MockUserRepository struct {
}

type DocumentVerificationConfig struct {
	Enabled bool `json:"enabled"`
	MinConfidenceScore float64 `json:"minConfidenceScore"`
	Provider string `json:"provider"`
	RequireBothSides bool `json:"requireBothSides"`
	RequireManualReview bool `json:"requireManualReview"`
	RequireSelfie bool `json:"requireSelfie"`
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	StoragePath string `json:"storagePath"`
	AcceptedDocuments []string `json:"acceptedDocuments"`
	EncryptAtRest bool `json:"encryptAtRest"`
	EncryptionKey string `json:"encryptionKey"`
	StorageProvider string `json:"storageProvider"`
}

type StepUpDevicesResponse struct {
	Count int `json:"count"`
	Devices interface{} `json:"devices"`
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

type DataDeletionRequestInput struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type TokenRequest struct {
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
	Audience string `json:"audience"`
	Client_secret string `json:"client_secret"`
	Code string `json:"code"`
	Redirect_uri string `json:"redirect_uri"`
	Client_id string `json:"client_id"`
	Code_verifier string `json:"code_verifier"`
	Grant_type string `json:"grant_type"`
}

type SignInResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type ProviderListResponse struct {
	Providers []ProviderInfo `json:"providers"`
	Total int `json:"total"`
}

type CompliancePolicy struct {
	ApprovedAt Time `json:"approvedAt"`
	Status string `json:"status"`
	AppId string `json:"appId"`
	ApprovedBy string `json:"approvedBy"`
	EffectiveDate time.Time `json:"effectiveDate"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	Content string `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	PolicyType string `json:"policyType"`
	ReviewDate time.Time `json:"reviewDate"`
	Title string `json:"title"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version string `json:"version"`
}

type mockImpersonationRepository struct {
}

type ImpersonateUserRequest struct {
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Duration time.Duration `json:"duration"`
}

type RejectRecoveryResponse struct {
	Message string `json:"message"`
	Reason string `json:"reason"`
	Rejected bool `json:"rejected"`
	RejectedAt time.Time `json:"rejectedAt"`
	SessionId xid.ID `json:"sessionId"`
}

type Middleware struct {
}

type AmountRule struct {
	Description string `json:"description"`
	Max_amount float64 `json:"max_amount"`
	Min_amount float64 `json:"min_amount"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Currency string `json:"currency"`
}

type AddCustomPermission_req struct {
	Description string `json:"description"`
	Name string `json:"name"`
	Category string `json:"category"`
}

type Enable_body struct {
	Method string `json:"method"`
	User_id string `json:"user_id"`
}

type RevokeDeviceRequest struct {
	DeviceId string `json:"deviceId"`
}

type CancelRecoveryResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type App struct {
}

type DashboardConfig struct {
	ShowViolations bool `json:"showViolations"`
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
	ShowRecentChecks bool `json:"showRecentChecks"`
	ShowReports bool `json:"showReports"`
	ShowScore bool `json:"showScore"`
}

type NotificationTemplateResponse struct {
	Template interface{} `json:"template"`
}

type BanUser_reqBody struct {
	Expires_at Time `json:"expires_at,omitempty"`
	Reason string `json:"reason"`
}

type ProvidersResponse struct {
	Providers []string `json:"providers"`
}

type Status struct {
}

type FactorEnrollmentResponse struct {
	FactorId xid.ID `json:"factorId"`
	ProvisioningData interface{} `json:"provisioningData"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
}

type ListViolationsFilter struct {
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Severity *string `json:"severity"`
	Status *string `json:"status"`
	UserId *string `json:"userId"`
	ViolationType *string `json:"violationType"`
}

type UpdatePasskeyRequest struct {
	Name string `json:"name"`
}

type SignInRequest struct {
}

type StateStore struct {
}

type MFABypassResponse struct {
	Id xid.ID `json:"id"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type FactorVerificationRequest struct {
	Code string `json:"code"`
	Data interface{} `json:"data"`
	FactorId xid.ID `json:"factorId"`
}

type ListProfilesFilter struct {
	AppId *string `json:"appId"`
	Standard *ComplianceStandard `json:"standard"`
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

type RegisterProviderRequest struct {
	AttributeMapping interface{} `json:"attributeMapping"`
	Domain string `json:"domain"`
	OidcClientID string `json:"oidcClientID"`
	OidcClientSecret string `json:"oidcClientSecret"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
	ProviderId string `json:"providerId"`
	SamlCert string `json:"samlCert"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	OidcIssuer string `json:"oidcIssuer"`
	SamlIssuer string `json:"samlIssuer"`
	Type string `json:"type"`
}

type UnblockUserRequest struct {
}

type MultiSessionDeleteResponse struct {
	Status string `json:"status"`
}

type NotificationErrorResponse struct {
	Error string `json:"error"`
}

type SendCodeResponse struct {
	Dev_code string `json:"dev_code"`
	Status string `json:"status"`
}

type Status_body struct {
	Device_id string `json:"device_id"`
	User_id string `json:"user_id"`
}

type ErrorResponse struct {
	Code string `json:"code"`
	Details interface{} `json:"details"`
	Error string `json:"error"`
}

type MFAStatus struct {
	GracePeriod Time `json:"gracePeriod"`
	PolicyActive bool `json:"policyActive"`
	RequiredCount int `json:"requiredCount"`
	TrustedDevice bool `json:"trustedDevice"`
	Enabled bool `json:"enabled"`
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
}

type ScheduleVideoSessionResponse struct {
	Instructions string `json:"instructions"`
	JoinUrl string `json:"joinUrl"`
	Message string `json:"message"`
	ScheduledAt time.Time `json:"scheduledAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type TokenResponse struct {
	Expires_in int `json:"expires_in"`
	Id_token string `json:"id_token"`
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
	Access_token string `json:"access_token"`
}

type BanUserRequest struct {
	App_id xid.ID `json:"app_id"`
	Expires_at Time `json:"expires_at"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
}

type SendResponse struct {
	Dev_otp string `json:"dev_otp"`
	Status string `json:"status"`
}

type NotificationsResponse struct {
	Count int `json:"count"`
	Notifications interface{} `json:"notifications"`
}

type TemplateDefault struct {
}

type ConnectionResponse struct {
	Connection SocialAccount `json:"connection"`
}

type TwoFAStatusDetailResponse struct {
	Enabled bool `json:"enabled"`
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
}

type RedisChallengeStore struct {
}

type SAMLLoginRequest struct {
	RelayState string `json:"relayState"`
}

type IDVerificationStatusResponse struct {
	Status interface{} `json:"status"`
}

type SessionTokenResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type CreateProvider_req struct {
	IsDefault bool `json:"isDefault"`
	OrganizationId *string `json:"organizationId,omitempty"`
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
	Config interface{} `json:"config"`
}

type TemplatesResponse struct {
	Count int `json:"count"`
	Templates interface{} `json:"templates"`
}

type KeyPair struct {
}

type MembersResponse struct {
	Members Member `json:"members"`
	Total int `json:"total"`
}

type AdminGetUserVerificationStatusResponse struct {
	Status UserVerificationStatus `json:"status"`
}

type AuditServiceAdapter struct {
}

type JWTService struct {
}

type GetPasskeyRequest struct {
}

type ScheduleVideoSessionRequest struct {
	ScheduledAt time.Time `json:"scheduledAt"`
	SessionId xid.ID `json:"sessionId"`
	TimeZone string `json:"timeZone"`
}

type ComplianceReport struct {
	Period string `json:"period"`
	ReportType string `json:"reportType"`
	Summary interface{} `json:"summary"`
	ExpiresAt time.Time `json:"expiresAt"`
	FileSize int64 `json:"fileSize"`
	Format string `json:"format"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	Status string `json:"status"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	FileUrl string `json:"fileUrl"`
	GeneratedBy string `json:"generatedBy"`
	Id string `json:"id"`
}

type ConsentDashboardConfig struct {
	ShowAuditLog bool `json:"showAuditLog"`
	ShowConsentHistory bool `json:"showConsentHistory"`
	ShowCookiePreferences bool `json:"showCookiePreferences"`
	ShowDataDeletion bool `json:"showDataDeletion"`
	ShowDataExport bool `json:"showDataExport"`
	ShowPolicies bool `json:"showPolicies"`
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
}

type MultiSessionErrorResponse struct {
	Error string `json:"error"`
}

type RequestDataExportRequest struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type TrustedDevice struct {
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	LastUsedAt Time `json:"lastUsedAt"`
	UserId xid.ID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	DeviceId string `json:"deviceId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	UserAgent string `json:"userAgent"`
}

type ComplianceCheckResponse struct {
	Id string `json:"id"`
}

type ListChecksFilter struct {
	ProfileId *string `json:"profileId"`
	SinceBefore Time `json:"sinceBefore"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
	CheckType *string `json:"checkType"`
}

type DataExportConfig struct {
	AllowedFormats []string `json:"allowedFormats"`
	AutoCleanup bool `json:"autoCleanup"`
	DefaultFormat string `json:"defaultFormat"`
	ExpiryHours int `json:"expiryHours"`
	IncludeSections []string `json:"includeSections"`
	RequestPeriod time.Duration `json:"requestPeriod"`
	StoragePath string `json:"storagePath"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	Enabled bool `json:"enabled"`
	MaxExportSize int64 `json:"maxExportSize"`
	MaxRequests int `json:"maxRequests"`
}

type mockSessionService struct {
}

type CreateResponse struct {
	Webhook Webhook `json:"webhook"`
}

type ListProvidersResponse struct {
	Providers []string `json:"providers"`
}

type TrustedContact struct {
}

type GenerateRecoveryCodesRequest struct {
	Count int `json:"count"`
	Format string `json:"format"`
}

type DocumentVerificationRequest struct {
}

type BackupAuthStatusResponse struct {
	Status string `json:"status"`
}

type RunCheckRequest struct {
	CheckType string `json:"checkType"`
}

type GetChallengeStatusResponse struct {
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
	MaxAttempts int `json:"maxAttempts"`
	Status ChallengeStatus `json:"status"`
	Attempts int `json:"attempts"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
}

type RecoverySessionInfo struct {
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	Method RecoveryMethod `json:"method"`
	RequiresReview bool `json:"requiresReview"`
	RiskScore float64 `json:"riskScore"`
	Status RecoveryStatus `json:"status"`
	TotalSteps int `json:"totalSteps"`
	UserId xid.ID `json:"userId"`
	CompletedAt Time `json:"completedAt"`
	CurrentStep int `json:"currentStep"`
	ExpiresAt time.Time `json:"expiresAt"`
	UserEmail string `json:"userEmail"`
}

type CreateTemplateVersion_req struct {
	Changes string `json:"changes"`
}

type NotificationWebhookResponse struct {
	Status string `json:"status"`
}

type TokenRevocationRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type TeamsResponse struct {
	Teams Team `json:"teams"`
	Total int `json:"total"`
}

type UpdateUserRequest struct {
	Name *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

type RevokeTrustedDeviceRequest struct {
}

type CancelRecoveryRequest struct {
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type RecoveryConfiguration struct {
}

type JumioProvider struct {
}

type ConsentManager struct {
}

type ListSessionsResponse struct {
	Limit int `json:"limit"`
	Page int `json:"page"`
	Sessions Session `json:"sessions"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
}

type BackupCodesConfig struct {
	Format string `json:"format"`
	Length int `json:"length"`
	Allow_reuse bool `json:"allow_reuse"`
	Count int `json:"count"`
	Enabled bool `json:"enabled"`
}

type VerifyTrustedContactRequest struct {
	Token string `json:"token"`
}

type CreateProfileFromTemplate_req struct {
	Standard ComplianceStandard `json:"standard"`
}

type ComplianceCheck struct {
	CheckType string `json:"checkType"`
	Evidence []string `json:"evidence"`
	LastCheckedAt time.Time `json:"lastCheckedAt"`
	NextCheckAt time.Time `json:"nextCheckAt"`
	ProfileId string `json:"profileId"`
	Result interface{} `json:"result"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	Status string `json:"status"`
}

type ConsentSummary struct {
	ConsentsByType interface{} `json:"consentsByType"`
	HasPendingExport bool `json:"hasPendingExport"`
	OrganizationId string `json:"organizationId"`
	PendingRenewals int `json:"pendingRenewals"`
	RevokedConsents int `json:"revokedConsents"`
	ExpiredConsents int `json:"expiredConsents"`
	GrantedConsents int `json:"grantedConsents"`
	HasPendingDeletion bool `json:"hasPendingDeletion"`
	LastConsentUpdate Time `json:"lastConsentUpdate"`
	TotalConsents int `json:"totalConsents"`
	UserId string `json:"userId"`
}

type ListUsersResponse struct {
	Limit int `json:"limit"`
	Page int `json:"page"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
	Users User `json:"users"`
}

type ProvidersAppResponse struct {
	AppId string `json:"appId"`
	Providers []string `json:"providers"`
}

type CreateVerificationSessionRequest struct {
	Config interface{} `json:"config"`
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
}

type ComplianceStatusDetailsResponse struct {
	Status string `json:"status"`
}

type TestSendTemplate_req struct {
	Recipient string `json:"recipient"`
	Variables interface{} `json:"variables"`
}

type SetActiveResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
}

type VerificationResponse struct {
	ExpiresAt Time `json:"expiresAt"`
	FactorsRemaining int `json:"factorsRemaining"`
	SessionComplete bool `json:"sessionComplete"`
	Success bool `json:"success"`
	Token string `json:"token"`
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

type MultiStepRecoveryConfig struct {
	AllowUserChoice bool `json:"allowUserChoice"`
	Enabled bool `json:"enabled"`
	HighRiskSteps []RecoveryMethod `json:"highRiskSteps"`
	LowRiskSteps []RecoveryMethod `json:"lowRiskSteps"`
	MediumRiskSteps []RecoveryMethod `json:"mediumRiskSteps"`
	SessionExpiry time.Duration `json:"sessionExpiry"`
	AllowStepSkip bool `json:"allowStepSkip"`
	MinimumSteps int `json:"minimumSteps"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
}

type CreateProfileRequest struct {
	DpoContact string `json:"dpoContact"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	LeastPrivilege bool `json:"leastPrivilege"`
	Name string `json:"name"`
	PasswordMinLength int `json:"passwordMinLength"`
	RegularAccessReview bool `json:"regularAccessReview"`
	AuditLogExport bool `json:"auditLogExport"`
	MfaRequired bool `json:"mfaRequired"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	RetentionDays int `json:"retentionDays"`
	ComplianceContact string `json:"complianceContact"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	SessionMaxAge int `json:"sessionMaxAge"`
	AppId string `json:"appId"`
	DataResidency string `json:"dataResidency"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	Metadata interface{} `json:"metadata"`
	RbacRequired bool `json:"rbacRequired"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	Standards []ComplianceStandard `json:"standards"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
}

type EvaluateRequest struct {
	Action string `json:"action"`
	Amount float64 `json:"amount"`
	Currency string `json:"currency"`
	Metadata interface{} `json:"metadata"`
	Method string `json:"method"`
	Resource_type string `json:"resource_type"`
	Route string `json:"route"`
}

type SaveNotificationSettings_req struct {
	RetryAttempts int `json:"retryAttempts"`
	RetryDelay string `json:"retryDelay"`
	AutoSendWelcome bool `json:"autoSendWelcome"`
	CleanupAfter string `json:"cleanupAfter"`
}

type ScopeInfo struct {
}

type JWK struct {
	Alg string `json:"alg"`
	E string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N string `json:"n"`
	Use string `json:"use"`
}

type ChallengeStatusResponse struct {
	CompletedAt Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRemaining int `json:"factorsRemaining"`
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
	SessionId xid.ID `json:"sessionId"`
	Status string `json:"status"`
}

type SendVerificationCodeResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	MaskedTarget string `json:"maskedTarget"`
	Message string `json:"message"`
	Sent bool `json:"sent"`
}

type GetRecoveryConfigResponse struct {
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
}

type AuditLog struct {
}

type ConsentReportResponse struct {
	Id string `json:"id"`
}

type CreateVerificationRequest struct {
}

type SignUpResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

type GetSessionResponse struct {
	User User `json:"user"`
	Session Session `json:"session"`
}

type BunRepository struct {
}

type StartVideoSessionRequest struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type ConsentAuditLog struct {
	OrganizationId string `json:"organizationId"`
	PreviousValue JSONBMap `json:"previousValue"`
	Purpose string `json:"purpose"`
	UserAgent string `json:"userAgent"`
	ConsentType string `json:"consentType"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	NewValue JSONBMap `json:"newValue"`
	Reason string `json:"reason"`
	UserId string `json:"userId"`
	Action string `json:"action"`
	ConsentId string `json:"consentId"`
	IpAddress string `json:"ipAddress"`
}

type PrivacySettingsRequest struct {
	CcpaMode *bool `json:"ccpaMode"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	DataRetentionDays *int `json:"dataRetentionDays"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	ConsentRequired *bool `json:"consentRequired"`
	ContactEmail string `json:"contactEmail"`
	ExportFormat []string `json:"exportFormat"`
	GdprMode *bool `json:"gdprMode"`
	ContactPhone string `json:"contactPhone"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	AllowDataPortability *bool `json:"allowDataPortability"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	DpoEmail string `json:"dpoEmail"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
}

type ListPasskeysResponse struct {
	Count int `json:"count"`
	Passkeys []PasskeyInfo `json:"passkeys"`
}

type MemoryChallengeStore struct {
}

type AccountLockoutError struct {
}

type ComplianceReportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type DeletePasskeyRequest struct {
}

type NoOpDocumentProvider struct {
}

type CreateTrainingRequest struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type AccountLockedResponse struct {
	Code string `json:"code"`
	Locked_minutes int `json:"locked_minutes"`
	Locked_until time.Time `json:"locked_until"`
	Message string `json:"message"`
}

type ListReportsFilter struct {
	ReportType *string `json:"reportType"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
	Format *string `json:"format"`
	ProfileId *string `json:"profileId"`
}

type CompliancePolicyResponse struct {
	Id string `json:"id"`
}

type ClientAuthResult struct {
}

type CreatePolicyResponse struct {
	User_id string `json:"user_id"`
	Description string `json:"description"`
	Org_id string `json:"org_id"`
	Updated_at time.Time `json:"updated_at"`
	Created_at time.Time `json:"created_at"`
	Enabled bool `json:"enabled"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority int `json:"priority"`
	Rules interface{} `json:"rules"`
}

