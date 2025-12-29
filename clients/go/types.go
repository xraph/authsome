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

type VerifyCodeResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	Message string `json:"message"`
	Valid bool `json:"valid"`
}

type NoOpVideoProvider struct {
}

type ChallengeResponse struct {
	ChallengeId xid.ID `json:"challengeId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	SessionId xid.ID `json:"sessionId"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
}

type RevokeAllRequest struct {
	IncludeCurrentSession bool `json:"includeCurrentSession"`
}

type IDVerificationWebhookResponse struct {
	Status string `json:"status"`
}

type ComplianceStatusDetailsResponse struct {
	Status string `json:"status"`
}

type stateEntry struct {
}

type RenderTemplateRequest struct {
}

type StartRecoveryRequest struct {
	UserId string `json:"userId"`
	DeviceId string `json:"deviceId"`
	Email string `json:"email"`
	PreferredMethod RecoveryMethod `json:"preferredMethod"`
}

type CreateTraining_req struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type BeginLoginResponse struct {
	Challenge string `json:"challenge"`
	Options interface{} `json:"options"`
	Timeout time.Duration `json:"timeout"`
}

type RegistrationService struct {
}

type RateLimit struct {
}

type GetAuditLogsRequestDTO struct {
}

type TwoFASendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type PolicyEngine struct {
}

type SignInResponse struct {
	Session interface{} `json:"session"`
	Token string `json:"token"`
	User interface{} `json:"user"`
}

type ImpersonationEndResponse struct {
	Ended_at string `json:"ended_at"`
	Status string `json:"status"`
}

type ListNotificationsResponse struct {
	Notifications []*interface{} `json:"notifications"`
	Total int `json:"total"`
}

type ProviderListResponse struct {
	Providers []ProviderInfo `json:"providers"`
	Total int `json:"total"`
}

type MockSocialAccountRepository struct {
}

type GetTemplateRequest struct {
}

type ComplianceReportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type ComplianceEvidencesResponse struct {
	Evidence []*interface{} `json:"evidence"`
}

type RefreshSessionResponse struct {
	AccessToken string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt string `json:"expiresAt"`
	RefreshExpiresAt string `json:"refreshExpiresAt"`
	Session interface{} `json:"session"`
}

type StepUpPoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type MFAConfigResponse struct {
	Allowed_factor_types []string `json:"allowed_factor_types"`
	Enabled bool `json:"enabled"`
	Required_factor_count int `json:"required_factor_count"`
}

type AccountLockoutError struct {
}

type GetUserVerificationStatusResponse struct {
	Status interface{} `json:"status"`
}

type SchemaValidator struct {
}

type ProvidersConfig struct {
	Reddit ProviderConfig `json:"reddit"`
	Apple ProviderConfig `json:"apple"`
	Github ProviderConfig `json:"github"`
	Spotify ProviderConfig `json:"spotify"`
	Line ProviderConfig `json:"line"`
	Microsoft ProviderConfig `json:"microsoft"`
	Twitch ProviderConfig `json:"twitch"`
	Twitter ProviderConfig `json:"twitter"`
	Dropbox ProviderConfig `json:"dropbox"`
	Facebook ProviderConfig `json:"facebook"`
	Gitlab ProviderConfig `json:"gitlab"`
	Google ProviderConfig `json:"google"`
	Notion ProviderConfig `json:"notion"`
	Slack ProviderConfig `json:"slack"`
	Bitbucket ProviderConfig `json:"bitbucket"`
	Discord ProviderConfig `json:"discord"`
	Linkedin ProviderConfig `json:"linkedin"`
}

type SignInRequest struct {
	RedirectUrl string `json:"redirectUrl"`
	Scopes []string `json:"scopes"`
	Provider string `json:"provider"`
}

type ListChecksFilter struct {
	AppId *string `json:"appId"`
	CheckType *string `json:"checkType"`
	ProfileId *string `json:"profileId"`
	SinceBefore Time `json:"sinceBefore"`
	Status *string `json:"status"`
}

type ConsentManager struct {
}

type ListUsersRequest struct {
	Page int `json:"page"`
	Role string `json:"role"`
	Search string `json:"search"`
	Status string `json:"status"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Limit int `json:"limit"`
}

type ListTrustedDevicesResponse struct {
	Count int `json:"count"`
	Devices []TrustedDevice `json:"devices"`
}

type ResolveViolationResponse struct {
	Status string `json:"status"`
}

type PolicyStats struct {
	AllowCount int64 `json:"allowCount"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	DenyCount int64 `json:"denyCount"`
	EvaluationCount int64 `json:"evaluationCount"`
	PolicyId string `json:"policyId"`
	PolicyName string `json:"policyName"`
}

type RolesResponse struct {
	Roles Role `json:"roles"`
}

type ComplianceReport struct {
	Status string `json:"status"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	Period string `json:"period"`
	ProfileId string `json:"profileId"`
	Summary interface{} `json:"summary"`
	ExpiresAt time.Time `json:"expiresAt"`
	FileSize int64 `json:"fileSize"`
	FileUrl string `json:"fileUrl"`
	Format string `json:"format"`
	GeneratedBy string `json:"generatedBy"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
}

type ComplianceStatus struct {
	NextAudit time.Time `json:"nextAudit"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	LastChecked time.Time `json:"lastChecked"`
	OverallStatus string `json:"overallStatus"`
	Score int `json:"score"`
	Violations int `json:"violations"`
	AppId string `json:"appId"`
	ChecksFailed int `json:"checksFailed"`
	ChecksPassed int `json:"checksPassed"`
	ChecksWarning int `json:"checksWarning"`
}

type DownloadReportResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type UploadDocumentResponse struct {
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
	ProcessingTime string `json:"processingTime"`
	Status string `json:"status"`
	UploadedAt time.Time `json:"uploadedAt"`
}

type ComplianceTraining struct {
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt Time `json:"expiresAt"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Score int `json:"score"`
	Status string `json:"status"`
	TrainingType string `json:"trainingType"`
	CompletedAt Time `json:"completedAt"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	UserId string `json:"userId"`
}

type RouteRule struct {
	Security_level SecurityLevel `json:"security_level"`
	Description string `json:"description"`
	Method string `json:"method"`
	Org_id string `json:"org_id"`
	Pattern string `json:"pattern"`
}

type StepUpVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type PrivacySettings struct {
	DataRetentionDays int `json:"dataRetentionDays"`
	Metadata JSONBMap `json:"metadata"`
	OrganizationId string `json:"organizationId"`
	RequireExplicitConsent bool `json:"requireExplicitConsent"`
	AnonymousConsentEnabled bool `json:"anonymousConsentEnabled"`
	AutoDeleteAfterDays int `json:"autoDeleteAfterDays"`
	ConsentRequired bool `json:"consentRequired"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DpoEmail string `json:"dpoEmail"`
	ExportFormat []string `json:"exportFormat"`
	Id xid.ID `json:"id"`
	ContactPhone string `json:"contactPhone"`
	CookieConsentEnabled bool `json:"cookieConsentEnabled"`
	RequireAdminApprovalForDeletion bool `json:"requireAdminApprovalForDeletion"`
	CcpaMode bool `json:"ccpaMode"`
	CreatedAt time.Time `json:"createdAt"`
	DeletionGracePeriodDays int `json:"deletionGracePeriodDays"`
	GdprMode bool `json:"gdprMode"`
	UpdatedAt time.Time `json:"updatedAt"`
	AllowDataPortability bool `json:"allowDataPortability"`
	ContactEmail string `json:"contactEmail"`
	DataExportExpiryHours int `json:"dataExportExpiryHours"`
}

type KeyStore struct {
}

type VerificationSessionResponse struct {
	Session IdentityVerificationSession `json:"session"`
}

type SendCodeRequest struct {
	Phone string `json:"phone"`
}

type ListReportsResponse struct {
	Reports []*interface{} `json:"reports"`
}

type Status struct {
}

type KeyStats struct {
}

type ListUsersResponse struct {
	Users User `json:"users"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
}

type UnpublishEntryRequest struct {
}

type VerificationListResponse struct {
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Total int `json:"total"`
	Verifications IdentityVerification `json:"verifications"`
}

type AccountLockedResponse struct {
	Code string `json:"code"`
	Locked_minutes int `json:"locked_minutes"`
	Locked_until time.Time `json:"locked_until"`
	Message string `json:"message"`
}

type DeleteSecretRequest struct {
}

type CodesResponse struct {
	Codes []string `json:"codes"`
}

type CancelRecoveryRequest struct {
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type StartImpersonationResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Session_id string `json:"session_id"`
	Started_at string `json:"started_at"`
	Target_user_id string `json:"target_user_id"`
}

type RegisterClientRequest struct {
	Client_name string `json:"client_name"`
	Contacts []string `json:"contacts"`
	Logo_uri string `json:"logo_uri"`
	Tos_uri string `json:"tos_uri"`
	Trusted_client bool `json:"trusted_client"`
	Grant_types []string `json:"grant_types"`
	Require_pkce bool `json:"require_pkce"`
	Policy_uri string `json:"policy_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Require_consent bool `json:"require_consent"`
	Scope string `json:"scope"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Application_type string `json:"application_type"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Response_types []string `json:"response_types"`
}

type DataExportRequestInput struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type MigrationStatusResponse struct {
	StartedAt time.Time `json:"startedAt"`
	UserOrganizationId *string `json:"userOrganizationId"`
	ValidationPassed bool `json:"validationPassed"`
	CompletedAt Time `json:"completedAt"`
	Errors []string `json:"errors"`
	FailedCount int `json:"failedCount"`
	Progress float64 `json:"progress"`
	Status string `json:"status"`
	TotalPolicies int `json:"totalPolicies"`
	AppId string `json:"appId"`
	EnvironmentId string `json:"environmentId"`
	MigratedCount int `json:"migratedCount"`
}

type SendOTPRequest struct {
	User_id string `json:"user_id"`
}

type LinkRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Password string `json:"password"`
}

type RefreshSessionRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type ConfirmEmailChangeResponse struct {
	Message string `json:"message"`
}

type GetSecurityQuestionsResponse struct {
	Questions []SecurityQuestionInfo `json:"questions"`
}

type ComplianceDashboardResponse struct {
	Metrics interface{} `json:"metrics"`
}

type ListResponse struct {
	Webhooks []*Webhook `json:"webhooks"`
}

type ListVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type ImpersonationMiddleware struct {
}

type ListSessionsRequest struct {
	Page int `json:"page"`
	User_id ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Limit int `json:"limit"`
}

type RequestDataExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type JWKSService struct {
}

type ProviderSessionRequest struct {
}

type FactorInfo struct {
	FactorId xid.ID `json:"factorId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Type FactorType `json:"type"`
}

type ConsentPolicy struct {
	CreatedBy string `json:"createdBy"`
	Id xid.ID `json:"id"`
	Active bool `json:"active"`
	ConsentType string `json:"consentType"`
	Content string `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Metadata JSONBMap `json:"metadata"`
	PublishedAt Time `json:"publishedAt"`
	ValidityPeriod *int `json:"validityPeriod"`
	Name string `json:"name"`
	OrganizationId string `json:"organizationId"`
	Renewable bool `json:"renewable"`
	Required bool `json:"required"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version string `json:"version"`
}

type ConsentDeletionResponse struct {
	Status string `json:"status"`
	Id string `json:"id"`
}

type GetOrganizationRequest struct {
}

type DeleteUserRequestDTO struct {
}

type BulkDeleteRequest struct {
	Ids []string `json:"ids"`
}

type CompleteRecoveryResponse struct {
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
	Status RecoveryStatus `json:"status"`
	Token string `json:"token"`
}

type AuditLog struct {
}

type ImpersonationErrorResponse struct {
	Error string `json:"error"`
}

type mockImpersonationRepository struct {
}

type GenerateConsentReportResponse struct {
	Id string `json:"id"`
}

type SendResponse struct {
	DevToken string `json:"devToken"`
	Status string `json:"status"`
}

type RiskAssessmentConfig struct {
	BlockHighRisk bool `json:"blockHighRisk"`
	Enabled bool `json:"enabled"`
	MediumRiskThreshold float64 `json:"mediumRiskThreshold"`
	NewDeviceWeight float64 `json:"newDeviceWeight"`
	RequireReviewAbove float64 `json:"requireReviewAbove"`
	VelocityWeight float64 `json:"velocityWeight"`
	HighRiskThreshold float64 `json:"highRiskThreshold"`
	HistoryWeight float64 `json:"historyWeight"`
	LowRiskThreshold float64 `json:"lowRiskThreshold"`
	NewIpWeight float64 `json:"newIpWeight"`
	NewLocationWeight float64 `json:"newLocationWeight"`
}

type ComplianceCheck struct {
	CheckType string `json:"checkType"`
	CreatedAt time.Time `json:"createdAt"`
	Evidence []string `json:"evidence"`
	LastCheckedAt time.Time `json:"lastCheckedAt"`
	NextCheckAt time.Time `json:"nextCheckAt"`
	ProfileId string `json:"profileId"`
	Result interface{} `json:"result"`
	Status string `json:"status"`
	AppId string `json:"appId"`
	Id string `json:"id"`
}

type ResetUserMFARequest struct {
	Reason string `json:"reason"`
}

type RedisChallengeStore struct {
}

type FinishLoginRequest struct {
	Remember bool `json:"remember"`
	Response interface{} `json:"response"`
}

type RecordCookieConsentRequest struct {
	Analytics bool `json:"analytics"`
	BannerVersion string `json:"bannerVersion"`
	Essential bool `json:"essential"`
	Functional bool `json:"functional"`
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
}

type EnableRequest struct {
}

type InitiateChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata interface{} `json:"metadata"`
}

type IPWhitelistConfig struct {
	Enabled bool `json:"enabled"`
	Strict_mode bool `json:"strict_mode"`
}

type AssignRoleRequest struct {
	RoleID string `json:"roleID"`
}

type VerifyRecoveryCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type GetVerificationSessionResponse struct {
	Session interface{} `json:"session"`
}

type CreateActionRequest struct {
	Description string `json:"description"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
}

type SaveBuilderTemplate_req struct {
	TemplateKey string `json:"templateKey"`
	Document Document `json:"document"`
	Name string `json:"name"`
	Subject string `json:"subject"`
	TemplateId *string `json:"templateId,omitempty"`
}

type RecoveryConfiguration struct {
}

type IntrospectTokenRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type DeleteRequest struct {
}

type MockUserRepository struct {
}

type ComplianceViolation struct {
	ProfileId string `json:"profileId"`
	ResolvedAt Time `json:"resolvedAt"`
	Status string `json:"status"`
	ViolationType string `json:"violationType"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Metadata interface{} `json:"metadata"`
	ResolvedBy string `json:"resolvedBy"`
	Severity string `json:"severity"`
	UserId string `json:"userId"`
	Description string `json:"description"`
	Id string `json:"id"`
}

type MemberHandler struct {
}

type PasskeyInfo struct {
	Aaguid string `json:"aaguid"`
	CredentialId string `json:"credentialId"`
	Id string `json:"id"`
	IsResidentKey bool `json:"isResidentKey"`
	LastUsedAt Time `json:"lastUsedAt"`
	Name string `json:"name"`
	SignCount uint `json:"signCount"`
	AuthenticatorType string `json:"authenticatorType"`
	CreatedAt time.Time `json:"createdAt"`
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

type BulkUnpublishRequest struct {
	Ids []string `json:"ids"`
}

type RenderTemplate_req struct {
	Variables interface{} `json:"variables"`
	Template string `json:"template"`
}

type ImpersonationStartResponse struct {
	Started_at string `json:"started_at"`
	Target_user_id string `json:"target_user_id"`
	Impersonator_id string `json:"impersonator_id"`
	Session_id string `json:"session_id"`
}

type GetEntryRequest struct {
}

type GetProviderRequest struct {
}

type UpdateRequest struct {
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Tags []string `json:"tags"`
	Value interface{} `json:"value"`
}

type AccessTokenClaims struct {
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
}

type IDVerificationErrorResponse struct {
	Error string `json:"error"`
}

type WebhookPayload struct {
}

type DeclareABTestWinner_req struct {
	AbTestGroup string `json:"abTestGroup"`
	WinnerId string `json:"winnerId"`
}

type TOTPConfig struct {
	Window_size int `json:"window_size"`
	Algorithm string `json:"algorithm"`
	Digits int `json:"digits"`
	Enabled bool `json:"enabled"`
	Issuer string `json:"issuer"`
	Period int `json:"period"`
}

type ActionResponse struct {
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
}

type TemplatesResponse struct {
	Count int `json:"count"`
	Templates interface{} `json:"templates"`
}

type GetContentTypeRequest struct {
}

type DeleteProviderRequest struct {
}

type SMSVerificationConfig struct {
	Provider string `json:"provider"`
	CodeExpiry time.Duration `json:"codeExpiry"`
	CodeLength int `json:"codeLength"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	Enabled bool `json:"enabled"`
	MaxAttempts int `json:"maxAttempts"`
	MaxSmsPerDay int `json:"maxSmsPerDay"`
	MessageTemplate string `json:"messageTemplate"`
}

type UpdatePasskeyRequest struct {
	Name string `json:"name"`
}

type ResetTemplateRequest struct {
}

type GenerateRecoveryCodesRequest struct {
	Count int `json:"count"`
	Format string `json:"format"`
}

type GetSecurityQuestionsRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type NoOpNotificationProvider struct {
}

type CompliancePoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type ConsentRecordResponse struct {
	Id string `json:"id"`
}

type VerifyResponse struct {
	Session Session `json:"session"`
	Success bool `json:"success"`
	Token string `json:"token"`
	User User `json:"user"`
}

type EmailConfig struct {
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
}

type SendVerificationCodeRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
	Target string `json:"target"`
}

type ListTrainingResponse struct {
	Training []*interface{} `json:"training"`
}

type OrganizationUIRegistry struct {
}

type ListAPIKeysRequest struct {
}

type AddTrustedContactResponse struct {
	Name string `json:"name"`
	Phone string `json:"phone"`
	Verified bool `json:"verified"`
	AddedAt time.Time `json:"addedAt"`
	ContactId xid.ID `json:"contactId"`
	Email string `json:"email"`
	Message string `json:"message"`
}

type TokenRevocationRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type RedisStateStore struct {
}

type CreateUserRequestDTO struct {
	Role string `json:"role"`
	Username string `json:"username"`
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Password string `json:"password"`
}

type ReverifyRequest struct {
	Reason string `json:"reason"`
}

type AsyncAdapter struct {
}

type RevokeResponse struct {
	Status string `json:"status"`
	RevokedCount int `json:"revokedCount"`
}

type PreviewConversionRequest struct {
	Condition string `json:"condition"`
	Resource string `json:"resource"`
	Subject string `json:"subject"`
	Actions []string `json:"actions"`
}

type DeclareABTestWinnerRequest struct {
}

type DocumentVerification struct {
}

type CompleteVideoSessionRequest struct {
	LivenessScore float64 `json:"livenessScore"`
	Notes string `json:"notes"`
	VerificationResult string `json:"verificationResult"`
	VideoSessionId xid.ID `json:"videoSessionId"`
	LivenessPassed bool `json:"livenessPassed"`
}

type BackupCodeFactorAdapter struct {
}

type CreateTrainingResponse struct {
	Id string `json:"id"`
}

type InviteMemberRequest struct {
	Email string `json:"email"`
	Role string `json:"role"`
}

type WebAuthnConfig struct {
	Rp_origins []string `json:"rp_origins"`
	Timeout int `json:"timeout"`
	Attestation_preference string `json:"attestation_preference"`
	Authenticator_selection interface{} `json:"authenticator_selection"`
	Enabled bool `json:"enabled"`
	Rp_display_name string `json:"rp_display_name"`
	Rp_id string `json:"rp_id"`
}

type MemoryChallengeStore struct {
}

type ListViolationsResponse struct {
	Violations []*interface{} `json:"violations"`
}

type AdminBlockUserResponse struct {
	Status interface{} `json:"status"`
}

type DeleteEntryRequest struct {
}

type JumioConfig struct {
	EnableLiveness bool `json:"enableLiveness"`
	EnabledCountries []string `json:"enabledCountries"`
	PresetId string `json:"presetId"`
	VerificationType string `json:"verificationType"`
	ApiToken string `json:"apiToken"`
	EnableAMLScreening bool `json:"enableAMLScreening"`
	Enabled bool `json:"enabled"`
	EnabledDocumentTypes []string `json:"enabledDocumentTypes"`
	ApiSecret string `json:"apiSecret"`
	CallbackUrl string `json:"callbackUrl"`
	DataCenter string `json:"dataCenter"`
	EnableExtraction bool `json:"enableExtraction"`
}

type settingField struct {
}

type CheckMetadata struct {
	Name string `json:"name"`
	Severity string `json:"severity"`
	Standards []string `json:"standards"`
	AutoRun bool `json:"autoRun"`
	Category string `json:"category"`
	Description string `json:"description"`
}

type GetDataDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type MigrationResponse struct {
	Message string `json:"message"`
	MigrationId string `json:"migrationId"`
	StartedAt time.Time `json:"startedAt"`
	Status string `json:"status"`
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

type RunCheck_req struct {
	CheckType string `json:"checkType"`
}

type StepUpRequirementResponse struct {
	Id string `json:"id"`
}

type EvaluationResult struct {
	Can_remember bool `json:"can_remember"`
	Grace_period_ends_at time.Time `json:"grace_period_ends_at"`
	Matched_rules []string `json:"matched_rules"`
	Requirement_id string `json:"requirement_id"`
	Security_level SecurityLevel `json:"security_level"`
	Allowed_methods []VerificationMethod `json:"allowed_methods"`
	Challenge_token string `json:"challenge_token"`
	Current_level SecurityLevel `json:"current_level"`
	Expires_at time.Time `json:"expires_at"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
	Required bool `json:"required"`
}

type LinkResponse struct {
	Message string `json:"message"`
	User interface{} `json:"user"`
}

type UpdateProfileResponse struct {
	Id string `json:"id"`
}

type ConsentAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type ListAuditEventsRequest struct {
}

type CreateResourceRequest struct {
	NamespaceId string `json:"namespaceId"`
	Type string `json:"type"`
	Attributes []ResourceAttributeRequest `json:"attributes"`
	Description string `json:"description"`
}

type GetFactorRequest struct {
}

type ListClientsResponse struct {
	Clients []ClientSummary `json:"clients"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Total int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type UpdateUserRequest struct {
	Name *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

type ListEntriesRequest struct {
}

type TemplateService struct {
}

type ChallengeStatusResponse struct {
	Status string `json:"status"`
	CompletedAt Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRemaining int `json:"factorsRemaining"`
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
	SessionId xid.ID `json:"sessionId"`
}

type RequestDataDeletionRequest struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type ListProvidersResponse struct {
	Providers []string `json:"providers"`
}

type AdminGetUserVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type ContentEntryHandler struct {
}

type CreateSessionRequest struct {
}

type StepUpAttempt struct {
	Failure_reason string `json:"failure_reason"`
	Org_id string `json:"org_id"`
	Requirement_id string `json:"requirement_id"`
	Success bool `json:"success"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Method VerificationMethod `json:"method"`
}

type RequirementsResponse struct {
	Count int `json:"count"`
	Requirements interface{} `json:"requirements"`
}

type ChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata interface{} `json:"metadata"`
	UserId xid.ID `json:"userId"`
}

type RequestReverification_req struct {
	Reason string `json:"reason"`
}

type BackupAuthQuestionsResponse struct {
	Questions []string `json:"questions"`
}

type ComplianceProfile struct {
	MfaRequired bool `json:"mfaRequired"`
	RbacRequired bool `json:"rbacRequired"`
	AppId string `json:"appId"`
	PasswordMinLength int `json:"passwordMinLength"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	RegularAccessReview bool `json:"regularAccessReview"`
	AuditLogExport bool `json:"auditLogExport"`
	RetentionDays int `json:"retentionDays"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	Name string `json:"name"`
	ComplianceContact string `json:"complianceContact"`
	Id string `json:"id"`
	Status string `json:"status"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	LeastPrivilege bool `json:"leastPrivilege"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	CreatedAt time.Time `json:"createdAt"`
	Metadata interface{} `json:"metadata"`
	UpdatedAt time.Time `json:"updatedAt"`
	DataResidency string `json:"dataResidency"`
	DpoContact string `json:"dpoContact"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	SessionMaxAge int `json:"sessionMaxAge"`
	Standards []ComplianceStandard `json:"standards"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
}

type CreateConsentResponse struct {
	Id string `json:"id"`
}

type GetEvidenceResponse struct {
	Id string `json:"id"`
}

type MigrationHandler struct {
}

type ListRevisionsRequest struct {
}

type VideoVerificationConfig struct {
	LivenessThreshold float64 `json:"livenessThreshold"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireScheduling bool `json:"requireScheduling"`
	SessionDuration time.Duration `json:"sessionDuration"`
	Enabled bool `json:"enabled"`
	MinScheduleAdvance time.Duration `json:"minScheduleAdvance"`
	Provider string `json:"provider"`
	RecordSessions bool `json:"recordSessions"`
	RecordingRetention time.Duration `json:"recordingRetention"`
	RequireLivenessCheck bool `json:"requireLivenessCheck"`
}

type MFASession struct {
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
	IpAddress string `json:"ipAddress"`
	Metadata interface{} `json:"metadata"`
	VerifiedFactors ID `json:"verifiedFactors"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	RiskLevel RiskLevel `json:"riskLevel"`
	SessionToken string `json:"sessionToken"`
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
	CompletedAt Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type UpdateUserResponse struct {
	User User `json:"user"`
}

type IDTokenClaims struct {
	Email_verified bool `json:"email_verified"`
	Family_name string `json:"family_name"`
	Name string `json:"name"`
	Nonce string `json:"nonce"`
	Auth_time int64 `json:"auth_time"`
	Email string `json:"email"`
	Given_name string `json:"given_name"`
	Preferred_username string `json:"preferred_username"`
	Session_state string `json:"session_state"`
}

type ListReportsFilter struct {
	Format *string `json:"format"`
	ProfileId *string `json:"profileId"`
	ReportType *string `json:"reportType"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
}

type MFABypassResponse struct {
	Id xid.ID `json:"id"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type SAMLLoginResponse struct {
	RedirectUrl string `json:"redirectUrl"`
	RequestId string `json:"requestId"`
	ProviderId string `json:"providerId"`
}

type GetPasskeyRequest struct {
}

type ScopeInfo struct {
}

type HealthCheckResponse struct {
	Version string `json:"version"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	Healthy bool `json:"healthy"`
	Message string `json:"message"`
	ProvidersStatus interface{} `json:"providersStatus"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type TeamsResponse struct {
	Teams Team `json:"teams"`
	Total int `json:"total"`
}

type CreateProfileFromTemplate_req struct {
	Standard ComplianceStandard `json:"standard"`
}

type TrustedDevicesConfig struct {
	Default_expiry_days int `json:"default_expiry_days"`
	Enabled bool `json:"enabled"`
	Max_devices_per_user int `json:"max_devices_per_user"`
	Max_expiry_days int `json:"max_expiry_days"`
}

type RevokeTokenService struct {
}

type GetMigrationStatusRequest struct {
}

type ReorderFieldsRequest struct {
}

type mockNotificationProvider struct {
}

type GenerateReport_req struct {
	Standard ComplianceStandard `json:"standard"`
	Format string `json:"format"`
	Period string `json:"period"`
	ReportType string `json:"reportType"`
}

type MFAPolicyResponse struct {
	OrganizationId ID `json:"organizationId"`
	RequiredFactorCount int `json:"requiredFactorCount"`
	AllowedFactorTypes []string `json:"allowedFactorTypes"`
	AppId xid.ID `json:"appId"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
}

type UpdatePrivacySettingsRequest struct {
	ContactPhone string `json:"contactPhone"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	ContactEmail string `json:"contactEmail"`
	CcpaMode *bool `json:"ccpaMode"`
	ConsentRequired *bool `json:"consentRequired"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	DataRetentionDays *int `json:"dataRetentionDays"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	DpoEmail string `json:"dpoEmail"`
	GdprMode *bool `json:"gdprMode"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	ExportFormat []string `json:"exportFormat"`
	AllowDataPortability *bool `json:"allowDataPortability"`
}

type GetFactorResponse struct {
	VerifiedAt Time `json:"verifiedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	LastUsedAt Time `json:"lastUsedAt"`
	Metadata interface{} `json:"metadata"`
	Priority FactorPriority `json:"priority"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserId xid.ID `json:"userId"`
	ExpiresAt Time `json:"expiresAt"`
	Name string `json:"name"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
}

type RemoveTrustedContactRequest struct {
	ContactId xid.ID `json:"contactId"`
}

type UpdateContentTypeRequest struct {
}

type OnfidoProvider struct {
}

type OTPSentResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type GetByPathResponse struct {
	Code string `json:"code"`
	Error string `json:"error"`
	Message string `json:"message"`
}

type CreateVerificationSessionResponse struct {
	Session interface{} `json:"session"`
}

type CreateDPARequest struct {
	AgreementType string `json:"agreementType"`
	EffectiveDate time.Time `json:"effectiveDate"`
	ExpiryDate Time `json:"expiryDate"`
	SignedByEmail string `json:"signedByEmail"`
	SignedByName string `json:"signedByName"`
	SignedByTitle string `json:"signedByTitle"`
	Version string `json:"version"`
	Content string `json:"content"`
	Metadata interface{} `json:"metadata"`
}

type DataDeletionConfig struct {
	ArchivePath string `json:"archivePath"`
	RetentionExemptions []string `json:"retentionExemptions"`
	AllowPartialDeletion bool `json:"allowPartialDeletion"`
	ArchiveBeforeDeletion bool `json:"archiveBeforeDeletion"`
	AutoProcessAfterGrace bool `json:"autoProcessAfterGrace"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	NotifyBeforeDeletion bool `json:"notifyBeforeDeletion"`
	PreserveLegalData bool `json:"preserveLegalData"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
}

type GetVersionsRequest struct {
}

type GenerateTokenRequest struct {
	Audience []string `json:"audience"`
	ExpiresIn time.Duration `json:"expiresIn"`
	Metadata interface{} `json:"metadata"`
	Permissions []string `json:"permissions"`
	Scopes []string `json:"scopes"`
	SessionId string `json:"sessionId"`
	TokenType string `json:"tokenType"`
	UserId string `json:"userId"`
}

type GetDashboardResponse struct {
	Metrics interface{} `json:"metrics"`
}

type GetPolicyResponse struct {
	Id string `json:"id"`
}

type ArchiveEntryRequest struct {
}

type OnfidoConfig struct {
	DocumentCheck DocumentCheckConfig `json:"documentCheck"`
	IncludeDocumentReport bool `json:"includeDocumentReport"`
	IncludeWatchlistReport bool `json:"includeWatchlistReport"`
	WorkflowId string `json:"workflowId"`
	Enabled bool `json:"enabled"`
	FacialCheck FacialCheckConfig `json:"facialCheck"`
	IncludeFacialReport bool `json:"includeFacialReport"`
	Region string `json:"region"`
	WebhookToken string `json:"webhookToken"`
	ApiToken string `json:"apiToken"`
}

type AmountRule struct {
	Currency string `json:"currency"`
	Description string `json:"description"`
	Max_amount float64 `json:"max_amount"`
	Min_amount float64 `json:"min_amount"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type RateLimitRule struct {
	Max int `json:"max"`
	Window time.Duration `json:"window"`
}

type ResendNotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type ConsentPolicyResponse struct {
	Id string `json:"id"`
}

type SecurityQuestionInfo struct {
	QuestionId int `json:"questionId"`
	QuestionText string `json:"questionText"`
	Id xid.ID `json:"id"`
	IsCustom bool `json:"isCustom"`
}

type CreateEvidence_req struct {
	ControlId string `json:"controlId"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
}

type AppHandler struct {
}

type RevokeConsentResponse struct {
	Status string `json:"status"`
}

type DeleteProfileResponse struct {
	Status string `json:"status"`
}

type ProviderSession struct {
}

type GetChallengeStatusRequest struct {
}

type ListSecretsRequest struct {
}

type BanUserRequestDTO struct {
	Expires_at Time `json:"expires_at"`
	Reason string `json:"reason"`
}

type UpdateFactorRequest struct {
	Metadata interface{} `json:"metadata"`
	Name *string `json:"name"`
	Priority *FactorPriority `json:"priority"`
	Status *FactorStatus `json:"status"`
}

type ConsentAuditConfig struct {
	Enabled bool `json:"enabled"`
	ExportFormat string `json:"exportFormat"`
	Immutable bool `json:"immutable"`
	LogUserAgent bool `json:"logUserAgent"`
	RetentionDays int `json:"retentionDays"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	LogAllChanges bool `json:"logAllChanges"`
	LogIpAddress bool `json:"logIpAddress"`
	SignLogs bool `json:"signLogs"`
}

type GetValueRequest struct {
}

type AutoCleanupConfig struct {
	Enabled bool `json:"enabled"`
	Interval time.Duration `json:"interval"`
}

type ListViolationsFilter struct {
	UserId *string `json:"userId"`
	ViolationType *string `json:"violationType"`
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Severity *string `json:"severity"`
	Status *string `json:"status"`
}

type FactorVerificationRequest struct {
	Data interface{} `json:"data"`
	FactorId xid.ID `json:"factorId"`
	Code string `json:"code"`
}

type AddTeamMemberRequest struct {
	Member_id string `json:"member_id"`
}

type ListPasskeysResponse struct {
	Count int `json:"count"`
	Passkeys []PasskeyInfo `json:"passkeys"`
}

type DefaultProviderRegistry struct {
}

type DocumentVerificationRequest struct {
}

type RestoreEntryRequest struct {
}

type NoOpSMSProvider struct {
}

type TimeBasedRule struct {
	Operation string `json:"operation"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Description string `json:"description"`
	Max_age time.Duration `json:"max_age"`
}

type FinishRegisterRequest struct {
	Name string `json:"name"`
	Response interface{} `json:"response"`
	UserId string `json:"userId"`
}

type DownloadDataExportResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type RenderTemplateResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

type UpdatePolicy_req struct {
	Content *string `json:"content"`
	Status *string `json:"status"`
	Title *string `json:"title"`
	Version *string `json:"version"`
}

type LinkAccountResponse struct {
	Url string `json:"url"`
}

type CreateResponse struct {
	Webhook Webhook `json:"webhook"`
}

type ChangePasswordResponse struct {
	Message string `json:"message"`
}

type GetTemplateDefaultsResponse struct {
	Templates []*interface{} `json:"templates"`
	Total int `json:"total"`
}

type AdminBlockUser_req struct {
	Reason string `json:"reason"`
}

type EvaluationContext struct {
}

type StepUpAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type FactorEnrollmentRequest struct {
	Type FactorType `json:"type"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
}

type ChallengeSession struct {
}

type UpdateProfileRequest struct {
	MfaRequired *bool `json:"mfaRequired"`
	Name *string `json:"name"`
	RetentionDays *int `json:"retentionDays"`
	Status *string `json:"status"`
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

type GetAppProfileResponse struct {
	Id string `json:"id"`
}

type OAuthState struct {
	Redirect_url string `json:"redirect_url"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Created_at time.Time `json:"created_at"`
	Extra_scopes []string `json:"extra_scopes"`
	Link_user_id ID `json:"link_user_id"`
	Provider string `json:"provider"`
}

type UserAdapter struct {
}

type WebAuthnWrapper struct {
}

type CreateOrganizationHandlerRequest struct {
}

type NotificationErrorResponse struct {
	Error string `json:"error"`
}

type GetTemplateVersionRequest struct {
}

type BackupAuthConfigResponse struct {
	Config interface{} `json:"config"`
}

type UpdateAppRequest struct {
}

type UpdateTemplateResponse struct {
	Template interface{} `json:"template"`
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

type ResourcesListResponse struct {
	Resources []*ResourceResponse `json:"resources"`
	TotalCount int `json:"totalCount"`
}

type AuthAutoSendConfig struct {
	Email_otp bool `json:"email_otp"`
	Magic_link bool `json:"magic_link"`
	Mfa_code bool `json:"mfa_code"`
	Password_reset bool `json:"password_reset"`
	Verification_email bool `json:"verification_email"`
	Welcome bool `json:"welcome"`
}

type BackupAuthCodesResponse struct {
	Codes []string `json:"codes"`
}

type ComplianceTemplateResponse struct {
	Standard string `json:"standard"`
}

type SessionStatsResponse struct {
	ActiveSessions int `json:"activeSessions"`
	DeviceCount int `json:"deviceCount"`
	LocationCount int `json:"locationCount"`
	NewestSession *string `json:"newestSession"`
	OldestSession *string `json:"oldestSession"`
	TotalSessions int `json:"totalSessions"`
}

type ConsentDecision struct {
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type InvitationResponse struct {
	Invitation Invitation `json:"invitation"`
	Message string `json:"message"`
}

type IDVerificationStatusResponse struct {
	Status interface{} `json:"status"`
}

type StepUpEvaluationResponse struct {
	Required bool `json:"required"`
	Reason string `json:"reason"`
}

type KeyPair struct {
}

type DocumentVerificationConfig struct {
	RequireSelfie bool `json:"requireSelfie"`
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	StorageProvider string `json:"storageProvider"`
	MinConfidenceScore float64 `json:"minConfidenceScore"`
	RequireBothSides bool `json:"requireBothSides"`
	RequireManualReview bool `json:"requireManualReview"`
	StoragePath string `json:"storagePath"`
	AcceptedDocuments []string `json:"acceptedDocuments"`
	Enabled bool `json:"enabled"`
	EncryptAtRest bool `json:"encryptAtRest"`
	EncryptionKey string `json:"encryptionKey"`
	Provider string `json:"provider"`
}

type DocumentVerificationResult struct {
}

type CreateTemplateResponse struct {
	Template interface{} `json:"template"`
}

type ActionsListResponse struct {
	Actions []*ActionResponse `json:"actions"`
	TotalCount int `json:"totalCount"`
}

type RevokeDeviceRequest struct {
	Fingerprint string `json:"fingerprint"`
}

type CompleteRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type AdaptiveMFAConfig struct {
	Factor_ip_reputation bool `json:"factor_ip_reputation"`
	Factor_new_device bool `json:"factor_new_device"`
	Factor_velocity bool `json:"factor_velocity"`
	Location_change_risk float64 `json:"location_change_risk"`
	New_device_risk float64 `json:"new_device_risk"`
	Risk_threshold float64 `json:"risk_threshold"`
	Velocity_risk float64 `json:"velocity_risk"`
	Enabled bool `json:"enabled"`
	Factor_location_change bool `json:"factor_location_change"`
	Require_step_up_threshold float64 `json:"require_step_up_threshold"`
}

type SetActiveResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type ListOrganizationsRequest struct {
}

type ProviderInfo struct {
	CreatedAt string `json:"createdAt"`
	Domain string `json:"domain"`
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
}

type ProviderDetailResponse struct {
	AttributeMapping interface{} `json:"attributeMapping"`
	OidcClientID string `json:"oidcClientID"`
	OidcIssuer string `json:"oidcIssuer"`
	ProviderId string `json:"providerId"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	SamlIssuer string `json:"samlIssuer"`
	Type string `json:"type"`
	UpdatedAt string `json:"updatedAt"`
	CreatedAt string `json:"createdAt"`
	Domain string `json:"domain"`
	HasSamlCert bool `json:"hasSamlCert"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
}

type GetStatsRequestDTO struct {
}

type ConsentReportResponse struct {
	Id string `json:"id"`
}

type CheckSubResult struct {
}

type VerifyTrustedContactRequest struct {
	Token string `json:"token"`
}

type DataDeletionRequest struct {
	ArchivePath string `json:"archivePath"`
	CompletedAt Time `json:"completedAt"`
	ExemptionReason string `json:"exemptionReason"`
	RequestReason string `json:"requestReason"`
	UpdatedAt time.Time `json:"updatedAt"`
	ApprovedAt Time `json:"approvedAt"`
	ApprovedBy string `json:"approvedBy"`
	ErrorMessage string `json:"errorMessage"`
	DeleteSections []string `json:"deleteSections"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	RejectedAt Time `json:"rejectedAt"`
	RetentionExempt bool `json:"retentionExempt"`
	UserId string `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	Status string `json:"status"`
}

type UpdateTeamHandlerRequest struct {
}

type GenerateReportRequest struct {
	Format string `json:"format"`
	Period string `json:"period"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
}

type CreateEvidenceRequest struct {
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	ControlId string `json:"controlId"`
}

type ListDevicesResponse struct {
	Devices []*Device `json:"devices"`
}

type MetadataResponse struct {
	Metadata string `json:"metadata"`
}

type EncryptionConfig struct {
	RotateKeyAfter time.Duration `json:"rotateKeyAfter"`
	TestOnStartup bool `json:"testOnStartup"`
	MasterKey string `json:"masterKey"`
}

type ConfigSourceConfig struct {
	AutoRefresh bool `json:"autoRefresh"`
	Enabled bool `json:"enabled"`
	Prefix string `json:"prefix"`
	Priority int `json:"priority"`
	RefreshInterval time.Duration `json:"refreshInterval"`
}

type MemoryStateStore struct {
}

type UploadDocumentRequest struct {
	BackImage string `json:"backImage"`
	DocumentType string `json:"documentType"`
	FrontImage string `json:"frontImage"`
	Selfie string `json:"selfie"`
	SessionId xid.ID `json:"sessionId"`
}

type EnrollFactorRequest struct {
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	Type FactorType `json:"type"`
}

type MFAPolicy struct {
	MaxFailedAttempts int `json:"maxFailedAttempts"`
	OrganizationId xid.ID `json:"organizationId"`
	RequiredFactorCount int `json:"requiredFactorCount"`
	RequiredFactorTypes []FactorType `json:"requiredFactorTypes"`
	StepUpRequired bool `json:"stepUpRequired"`
	TrustedDeviceDays int `json:"trustedDeviceDays"`
	AdaptiveMfaEnabled bool `json:"adaptiveMfaEnabled"`
	AllowedFactorTypes []FactorType `json:"allowedFactorTypes"`
	CreatedAt time.Time `json:"createdAt"`
	GracePeriodDays int `json:"gracePeriodDays"`
	UpdatedAt time.Time `json:"updatedAt"`
	Id xid.ID `json:"id"`
	LockoutDurationMinutes int `json:"lockoutDurationMinutes"`
}

type StepUpRequirement struct {
	User_agent string `json:"user_agent"`
	Amount float64 `json:"amount"`
	Fulfilled_at Time `json:"fulfilled_at"`
	Challenge_token string `json:"challenge_token"`
	Current_level SecurityLevel `json:"current_level"`
	Expires_at time.Time `json:"expires_at"`
	Ip string `json:"ip"`
	Metadata interface{} `json:"metadata"`
	Resource_type string `json:"resource_type"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Currency string `json:"currency"`
	Id string `json:"id"`
	Org_id string `json:"org_id"`
	Required_level SecurityLevel `json:"required_level"`
	Risk_score float64 `json:"risk_score"`
	Route string `json:"route"`
	Rule_name string `json:"rule_name"`
	Method string `json:"method"`
	Reason string `json:"reason"`
	Resource_action string `json:"resource_action"`
	Session_id string `json:"session_id"`
	Status string `json:"status"`
}

type GetUserVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type BunRepository struct {
}

type DashboardConfig struct {
	EnableImport bool `json:"enableImport"`
	EnableReveal bool `json:"enableReveal"`
	EnableTreeView bool `json:"enableTreeView"`
	RevealTimeout time.Duration `json:"revealTimeout"`
	EnableExport bool `json:"enableExport"`
}

type CompareRevisionsRequest struct {
}

type VerifyRequest2FA struct {
	Code string `json:"code"`
	Device_id string `json:"device_id"`
	Remember_device bool `json:"remember_device"`
	User_id string `json:"user_id"`
}

type RegisterProviderResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Type string `json:"type"`
}

type InstantiateTemplateRequest struct {
	Description string `json:"description"`
	Enabled bool `json:"enabled"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
	Parameters interface{} `json:"parameters"`
	Priority int `json:"priority"`
	ResourceType string `json:"resourceType"`
	Actions []string `json:"actions"`
}

type CreateEntryRequest struct {
}

type GetTeamRequest struct {
}

type GetCurrentResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type UpdateEntryRequest struct {
}

type TestSendTemplate_req struct {
	Recipient string `json:"recipient"`
	Variables interface{} `json:"variables"`
}

type CallbackDataResponse struct {
	Action string `json:"action"`
	IsNewUser bool `json:"isNewUser"`
	User User `json:"user"`
}

type CreateTrainingRequest struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type ChannelsResponse struct {
	Channels interface{} `json:"channels"`
	Count int `json:"count"`
}

type ListRememberedDevicesResponse struct {
	Devices interface{} `json:"devices"`
	Count int `json:"count"`
}

type BulkPublishRequest struct {
	Ids []string `json:"ids"`
}

type CreatePolicy_req struct {
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	Version string `json:"version"`
	Content string `json:"content"`
	PolicyType string `json:"policyType"`
}

type EndImpersonationResponse struct {
	Ended_at string `json:"ended_at"`
	Status string `json:"status"`
}

type DeleteTemplateResponse struct {
	Status string `json:"status"`
}

type RefreshResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type ClientAuthResult struct {
}

type CreateNamespaceRequest struct {
	Name string `json:"name"`
	TemplateId string `json:"templateId"`
	Description string `json:"description"`
	InheritPlatform bool `json:"inheritPlatform"`
}

type SendCodeResponse struct {
	Status string `json:"status"`
	Dev_code string `json:"dev_code"`
}

type DeleteAPIKeyRequest struct {
}

type OrganizationHandler struct {
}

type CreateTeamRequest struct {
	Description string `json:"description"`
	Name string `json:"name"`
}

type BeginRegisterRequest struct {
	RequireResidentKey bool `json:"requireResidentKey"`
	UserId string `json:"userId"`
	UserVerification string `json:"userVerification"`
	AuthenticatorType string `json:"authenticatorType"`
	Name string `json:"name"`
}

type ListTemplatesResponse struct {
	Templates []*interface{} `json:"templates"`
	Total int `json:"total"`
}

type UpdateProvider_req struct {
	Config interface{} `json:"config"`
	IsActive bool `json:"isActive"`
	IsDefault bool `json:"isDefault"`
}

type AccountAutoSendConfig struct {
	Deleted bool `json:"deleted"`
	Email_change_request bool `json:"email_change_request"`
	Email_changed bool `json:"email_changed"`
	Password_changed bool `json:"password_changed"`
	Reactivated bool `json:"reactivated"`
	Suspended bool `json:"suspended"`
	Username_changed bool `json:"username_changed"`
}

type StepUpStatusResponse struct {
	Status string `json:"status"`
}

type RevokeTrustedDeviceRequest struct {
}

type UpdatePrivacySettingsResponse struct {
	Settings interface{} `json:"settings"`
}

type ReportsConfig struct {
	Enabled bool `json:"enabled"`
	Formats []string `json:"formats"`
	IncludeEvidence bool `json:"includeEvidence"`
	RetentionDays int `json:"retentionDays"`
	Schedule string `json:"schedule"`
	StoragePath string `json:"storagePath"`
}

type RegisterClientResponse struct {
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Tos_uri string `json:"tos_uri"`
	Application_type string `json:"application_type"`
	Client_secret string `json:"client_secret"`
	Response_types []string `json:"response_types"`
	Scope string `json:"scope"`
	Client_id string `json:"client_id"`
	Client_name string `json:"client_name"`
	Client_secret_expires_at int64 `json:"client_secret_expires_at"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
	Client_id_issued_at int64 `json:"client_id_issued_at"`
	Logo_uri string `json:"logo_uri"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
}

type PolicyPreviewResponse struct {
	ResourceType string `json:"resourceType"`
	Actions []string `json:"actions"`
	Description string `json:"description"`
	Expression string `json:"expression"`
	Name string `json:"name"`
}

type IDVerificationListResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type NotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type ClientRegistrationResponse struct {
	Logo_uri string `json:"logo_uri"`
	Tos_uri string `json:"tos_uri"`
	Client_name string `json:"client_name"`
	Response_types []string `json:"response_types"`
	Client_id string `json:"client_id"`
	Client_secret_expires_at int64 `json:"client_secret_expires_at"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Redirect_uris []string `json:"redirect_uris"`
	Scope string `json:"scope"`
	Application_type string `json:"application_type"`
	Client_id_issued_at int64 `json:"client_id_issued_at"`
	Client_secret string `json:"client_secret"`
}

type StripeIdentityProvider struct {
}

type ComplianceTemplate struct {
	Name string `json:"name"`
	PasswordMinLength int `json:"passwordMinLength"`
	RequiredPolicies []string `json:"requiredPolicies"`
	Standard ComplianceStandard `json:"standard"`
	AuditFrequencyDays int `json:"auditFrequencyDays"`
	DataResidency string `json:"dataResidency"`
	Description string `json:"description"`
	RequiredTraining []string `json:"requiredTraining"`
	RetentionDays int `json:"retentionDays"`
	SessionMaxAge int `json:"sessionMaxAge"`
	MfaRequired bool `json:"mfaRequired"`
}

type CreateProfileFromTemplateRequest struct {
	Standard ComplianceStandard `json:"standard"`
}

type DevicesResponse struct {
	Devices interface{} `json:"devices"`
	Count int `json:"count"`
}

type SignUpRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateRequest struct {
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Path string `json:"path"`
	Tags []string `json:"tags"`
	Value interface{} `json:"value"`
	ValueType string `json:"valueType"`
}

type AdminGetUserVerificationStatusResponse struct {
	Status interface{} `json:"status"`
}

type TokenIntrospectionResponse struct {
	Client_id string `json:"client_id"`
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
	Iss string `json:"iss"`
	Jti string `json:"jti"`
	Nbf int64 `json:"nbf"`
	Sub string `json:"sub"`
	Username string `json:"username"`
	Active bool `json:"active"`
	Aud []string `json:"aud"`
}

type VerificationResponse struct {
	Verification IdentityVerification `json:"verification"`
}

type HandleUpdateSettings_updates struct {
}

type auditServiceAdapter struct {
}

type ResetUserMFAResponse struct {
	DevicesRevoked int `json:"devicesRevoked"`
	FactorsReset int `json:"factorsReset"`
	Message string `json:"message"`
	Success bool `json:"success"`
}

type GetComplianceStatusResponse struct {
	Status string `json:"status"`
}

type ResetTemplateResponse struct {
	Status string `json:"status"`
}

type EnrollFactorResponse struct {
	ProvisioningData interface{} `json:"provisioningData"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
	FactorId xid.ID `json:"factorId"`
}

type ImpersonationSession struct {
}

type LinkAccountRequest struct {
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
}

type RecordCookieConsentResponse struct {
	Preferences interface{} `json:"preferences"`
}

type VerifyChallengeResponse struct {
	FactorsRemaining int `json:"factorsRemaining"`
	SessionComplete bool `json:"sessionComplete"`
	Success bool `json:"success"`
	Token string `json:"token"`
	ExpiresAt Time `json:"expiresAt"`
}

type AuditLogEntry struct {
	AppId string `json:"appId"`
	NewValue interface{} `json:"newValue"`
	ResourceType string `json:"resourceType"`
	UserOrganizationId *string `json:"userOrganizationId"`
	ActorId string `json:"actorId"`
	EnvironmentId string `json:"environmentId"`
	Id string `json:"id"`
	IpAddress string `json:"ipAddress"`
	OldValue interface{} `json:"oldValue"`
	ResourceId string `json:"resourceId"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string `json:"userAgent"`
	Action string `json:"action"`
}

type CreateProvider_req struct {
	OrganizationId *string `json:"organizationId,omitempty"`
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
	Config interface{} `json:"config"`
	IsDefault bool `json:"isDefault"`
}

type BackupAuthStatusResponse struct {
	Status string `json:"status"`
}

type CompliancePolicy struct {
	PolicyType string `json:"policyType"`
	ReviewDate time.Time `json:"reviewDate"`
	ApprovedBy string `json:"approvedBy"`
	CreatedAt time.Time `json:"createdAt"`
	EffectiveDate time.Time `json:"effectiveDate"`
	Metadata interface{} `json:"metadata"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version string `json:"version"`
	AppId string `json:"appId"`
	ApprovedAt Time `json:"approvedAt"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	Content string `json:"content"`
	Status string `json:"status"`
	Title string `json:"title"`
	Id string `json:"id"`
}

type FactorAdapterRegistry struct {
}

type ClientsListResponse struct {
	Clients []ClientSummary `json:"clients"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Total int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type AdminHandler struct {
}

type VideoVerificationSession struct {
}

type StepUpAuditLog struct {
	User_agent string `json:"user_agent"`
	Event_data interface{} `json:"event_data"`
	Event_type string `json:"event_type"`
	Org_id string `json:"org_id"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Severity string `json:"severity"`
}

type RequestReverificationRequest struct {
	Reason string `json:"reason"`
}

type BanUserRequest struct {
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Expires_at Time `json:"expires_at"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
}

type RejectRecoveryResponse struct {
	SessionId xid.ID `json:"sessionId"`
	Message string `json:"message"`
	Reason string `json:"reason"`
	Rejected bool `json:"rejected"`
	RejectedAt time.Time `json:"rejectedAt"`
}

type ValidatePolicyRequest struct {
	ResourceType string `json:"resourceType"`
	Expression string `json:"expression"`
}

type ProviderCheckResult struct {
}

type GenerateReportResponse struct {
	Id string `json:"id"`
}

type ResourceAttributeRequest struct {
	Required bool `json:"required"`
	Type string `json:"type"`
	Default interface{} `json:"default"`
	Description string `json:"description"`
	Name string `json:"name"`
}

type GetAPIKeyRequest struct {
}

type SMSFactorAdapter struct {
}

type ListFactorsResponse struct {
	Count int `json:"count"`
	Factors []Factor `json:"factors"`
}

type RequestPasswordResetResponse struct {
	Message string `json:"message"`
}

type ResetPasswordRequest struct {
	Token string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type ListRequest struct {
}

type DocumentCheckConfig struct {
	ValidateExpiry bool `json:"validateExpiry"`
	Enabled bool `json:"enabled"`
	ExtractData bool `json:"extractData"`
	ValidateDataConsistency bool `json:"validateDataConsistency"`
}

type AddTrustedContactRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
}

type Config struct {
	CcpaEnabled bool `json:"ccpaEnabled"`
	DataDeletion DataDeletionConfig `json:"dataDeletion"`
	DataExport DataExportConfig `json:"dataExport"`
	GdprEnabled bool `json:"gdprEnabled"`
	Audit ConsentAuditConfig `json:"audit"`
	CookieConsent CookieConsentConfig `json:"cookieConsent"`
	Dashboard ConsentDashboardConfig `json:"dashboard"`
	Enabled bool `json:"enabled"`
	Expiry ConsentExpiryConfig `json:"expiry"`
	Notifications ConsentNotificationsConfig `json:"notifications"`
}

type VerifyAPIKeyRequest struct {
	Key string `json:"key"`
}

type UpdatePasskeyResponse struct {
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ConsentExpiryConfig struct {
	AutoExpireCheck bool `json:"autoExpireCheck"`
	DefaultValidityDays int `json:"defaultValidityDays"`
	Enabled bool `json:"enabled"`
	ExpireCheckInterval time.Duration `json:"expireCheckInterval"`
	RenewalReminderDays int `json:"renewalReminderDays"`
	RequireReConsent bool `json:"requireReConsent"`
	AllowRenewal bool `json:"allowRenewal"`
}

type UpdateSecretRequest struct {
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Tags []string `json:"tags"`
	Value interface{} `json:"value"`
}

type BackupAuthContactsResponse struct {
	Contacts []*interface{} `json:"contacts"`
}

type DeviceInfo struct {
	DeviceId string `json:"deviceId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
}

type RiskFactor struct {
}

type RequestDataDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type CallbackResult struct {
}

type CreateConsentPolicyResponse struct {
	Id string `json:"id"`
}

type StateStore struct {
}

type TemplatesListResponse struct {
	TotalCount int `json:"totalCount"`
	Categories []string `json:"categories"`
	Templates []*TemplateResponse `json:"templates"`
}

type TokenIntrospectionRequest struct {
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
	Client_id string `json:"client_id"`
}

type DeleteContentTypeRequest struct {
}

type SetupSecurityQuestionRequest struct {
	CustomText string `json:"customText"`
	QuestionId int `json:"questionId"`
	Answer string `json:"answer"`
}

type NotificationsConfig struct {
	NotifyOnRecoveryFailed bool `json:"notifyOnRecoveryFailed"`
	NotifyOnRecoveryStart bool `json:"notifyOnRecoveryStart"`
	SecurityOfficerEmail string `json:"securityOfficerEmail"`
	Channels []string `json:"channels"`
	Enabled bool `json:"enabled"`
	NotifyAdminOnHighRisk bool `json:"notifyAdminOnHighRisk"`
	NotifyAdminOnReviewNeeded bool `json:"notifyAdminOnReviewNeeded"`
	NotifyOnRecoveryComplete bool `json:"notifyOnRecoveryComplete"`
}

type ConsentExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type ImpersonateUserRequest struct {
	Duration time.Duration `json:"duration"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
}

type RecoverySession struct {
}

type HandleWebhookResponse struct {
	Status string `json:"status"`
}

type DeleteEvidenceResponse struct {
	Status string `json:"status"`
}

type GetByIDResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type CreateTemplateVersion_req struct {
	Changes string `json:"changes"`
}

type GetRolesRequest struct {
}

type ComplianceProfileResponse struct {
	Id string `json:"id"`
}

type bunRepository struct {
}

type RevokeSessionRequestDTO struct {
}

type NotificationChannels struct {
	Webhook bool `json:"webhook"`
	Email bool `json:"email"`
	Slack bool `json:"slack"`
}

type MFAStatus struct {
	TrustedDevice bool `json:"trustedDevice"`
	Enabled bool `json:"enabled"`
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
	GracePeriod Time `json:"gracePeriod"`
	PolicyActive bool `json:"policyActive"`
	RequiredCount int `json:"requiredCount"`
}

type HandleConsentRequest struct {
	Code_challenge string `json:"code_challenge"`
	Code_challenge_method string `json:"code_challenge_method"`
	Redirect_uri string `json:"redirect_uri"`
	Response_type string `json:"response_type"`
	Scope string `json:"scope"`
	State string `json:"state"`
	Action string `json:"action"`
	Client_id string `json:"client_id"`
}

// StatusResponse represents Status response
type StatusResponse struct {
	Status string `json:"status"`
}

type AddFieldRequest struct {
}

type VerificationResult struct {
}

type GetDocumentVerificationResponse struct {
	ConfidenceScore float64 `json:"confidenceScore"`
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
	RejectionReason string `json:"rejectionReason"`
	Status string `json:"status"`
	VerifiedAt Time `json:"verifiedAt"`
}

type RiskContext struct {
}

type CompleteTraining_req struct {
	Score int `json:"score"`
}

type VerifyChallengeRequest struct {
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data interface{} `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
}

type GetConsentResponse struct {
	Id string `json:"id"`
}

type ListChecksResponse struct {
	Checks []*interface{} `json:"checks"`
}

type MockRepository struct {
}

type ImpersonationVerifyResponse struct {
	Target_user_id string `json:"target_user_id"`
	Impersonator_id string `json:"impersonator_id"`
	Is_impersonating bool `json:"is_impersonating"`
}

type ClientSummary struct {
	ApplicationType string `json:"applicationType"`
	ClientID string `json:"clientID"`
	CreatedAt string `json:"createdAt"`
	IsOrgLevel bool `json:"isOrgLevel"`
	Name string `json:"name"`
}

type NotificationTemplateListResponse struct {
	Total int `json:"total"`
	Templates []*interface{} `json:"templates"`
}

type ListSessionsResponse struct {
	Limit int `json:"limit"`
	Page int `json:"page"`
	Sessions Session `json:"sessions"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
}

type ListSessionsRequestDTO struct {
}

type TemplateResponse struct {
	Category string `json:"category"`
	Description string `json:"description"`
	Examples []string `json:"examples"`
	Expression string `json:"expression"`
	Id string `json:"id"`
	Name string `json:"name"`
	Parameters TemplateParameter `json:"parameters"`
}

type AnalyticsSummary struct {
	TopPolicies []PolicyStats `json:"topPolicies"`
	TopResourceTypes []ResourceTypeStats `json:"topResourceTypes"`
	AllowedCount int64 `json:"allowedCount"`
	CacheHitRate float64 `json:"cacheHitRate"`
	TotalEvaluations int64 `json:"totalEvaluations"`
	TotalPolicies int `json:"totalPolicies"`
	ActivePolicies int `json:"activePolicies"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	DeniedCount int64 `json:"deniedCount"`
}

type GetVerificationResponse struct {
	Verification interface{} `json:"verification"`
}

type GetCheckResponse struct {
	Id string `json:"id"`
}

type JumioProvider struct {
}

type NoOpDocumentProvider struct {
}

type ComplianceStatusResponse struct {
	Status string `json:"status"`
}

type UpdateMemberRequest struct {
	Role string `json:"role"`
}

type BeginLoginRequest struct {
	UserId string `json:"userId"`
	UserVerification string `json:"userVerification"`
}

type UnassignRoleRequest struct {
}

type GetRecoveryStatsRequest struct {
	EndDate time.Time `json:"endDate"`
	OrganizationId string `json:"organizationId"`
	StartDate time.Time `json:"startDate"`
}

type RetentionConfig struct {
	ArchiveBeforePurge bool `json:"archiveBeforePurge"`
	ArchivePath string `json:"archivePath"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	PurgeSchedule string `json:"purgeSchedule"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email"`
}

type StateStorageConfig struct {
	RedisAddr string `json:"redisAddr"`
	RedisDb int `json:"redisDb"`
	RedisPassword string `json:"redisPassword"`
	StateTtl time.Duration `json:"stateTtl"`
	UseRedis bool `json:"useRedis"`
}

type ListUsersRequestDTO struct {
}

type Email struct {
}

type AnalyticsResponse struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Summary AnalyticsSummary `json:"summary"`
	TimeRange interface{} `json:"timeRange"`
}

type ReviewDocumentRequest struct {
	Notes string `json:"notes"`
	RejectionReason string `json:"rejectionReason"`
	Approved bool `json:"approved"`
	DocumentId xid.ID `json:"documentId"`
}

type VideoSessionResult struct {
}

type TokenResponse struct {
	Token_type string `json:"token_type"`
	Access_token string `json:"access_token"`
	Expires_in int `json:"expires_in"`
	Id_token string `json:"id_token"`
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
}

type ResolveViolationRequest struct {
	Notes string `json:"notes"`
	Resolution string `json:"resolution"`
}

type ForgetDeviceResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type CheckResult struct {
	CheckType string `json:"checkType"`
	Error error `json:"error"`
	Evidence []string `json:"evidence"`
	Result interface{} `json:"result"`
	Score float64 `json:"score"`
	Status string `json:"status"`
}

type RejectRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
	Notes string `json:"notes"`
	Reason string `json:"reason"`
}

type ConsentAuditLog struct {
	UserId string `json:"userId"`
	ConsentId string `json:"consentId"`
	IpAddress string `json:"ipAddress"`
	NewValue JSONBMap `json:"newValue"`
	Purpose string `json:"purpose"`
	Reason string `json:"reason"`
	Action string `json:"action"`
	ConsentType string `json:"consentType"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	PreviousValue JSONBMap `json:"previousValue"`
	UserAgent string `json:"userAgent"`
}

type AuditConfig struct {
	AutoCleanup bool `json:"autoCleanup"`
	EnableAccessLog bool `json:"enableAccessLog"`
	LogReads bool `json:"logReads"`
	LogWrites bool `json:"logWrites"`
	RetentionDays int `json:"retentionDays"`
}

type TestProvider_req struct {
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
	TestRecipient string `json:"testRecipient"`
}

type TwoFAStatusResponse struct {
	Enabled bool `json:"enabled"`
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
}

type RequestTrustedContactVerificationRequest struct {
	ContactId xid.ID `json:"contactId"`
	SessionId xid.ID `json:"sessionId"`
}

type MultiStepRecoveryConfig struct {
	HighRiskSteps []RecoveryMethod `json:"highRiskSteps"`
	LowRiskSteps []RecoveryMethod `json:"lowRiskSteps"`
	MediumRiskSteps []RecoveryMethod `json:"mediumRiskSteps"`
	MinimumSteps int `json:"minimumSteps"`
	SessionExpiry time.Duration `json:"sessionExpiry"`
	AllowStepSkip bool `json:"allowStepSkip"`
	AllowUserChoice bool `json:"allowUserChoice"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
	Enabled bool `json:"enabled"`
}

type VerifySecurityAnswersRequest struct {
	Answers interface{} `json:"answers"`
	SessionId xid.ID `json:"sessionId"`
}

type CreateProfileResponse struct {
	Id string `json:"id"`
}

// User represents User account
type User struct {
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	OrganizationId *string `json:"organizationId,omitempty"`
	Id string `json:"id"`
	Email string `json:"email"`
	Name *string `json:"name,omitempty"`
	EmailVerified bool `json:"emailVerified"`
}

type SSOAuthResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type PoliciesListResponse struct {
	PageSize int `json:"pageSize"`
	Policies []*PolicyResponse `json:"policies"`
	TotalCount int `json:"totalCount"`
	Page int `json:"page"`
}

type LoginResponse struct {
	PasskeyUsed string `json:"passkeyUsed"`
	Session interface{} `json:"session"`
	Token string `json:"token"`
	User interface{} `json:"user"`
}

type VerifyTokenRequest struct {
	Token string `json:"token"`
	TokenType string `json:"tokenType"`
	Audience []string `json:"audience"`
}

type NamespacesListResponse struct {
	Namespaces []*NamespaceResponse `json:"namespaces"`
	TotalCount int `json:"totalCount"`
}

type CreateVerificationSession_req struct {
	Config interface{} `json:"config"`
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
}

type ConsentSummary struct {
	ExpiredConsents int `json:"expiredConsents"`
	GrantedConsents int `json:"grantedConsents"`
	OrganizationId string `json:"organizationId"`
	PendingRenewals int `json:"pendingRenewals"`
	RevokedConsents int `json:"revokedConsents"`
	UserId string `json:"userId"`
	ConsentsByType interface{} `json:"consentsByType"`
	HasPendingDeletion bool `json:"hasPendingDeletion"`
	HasPendingExport bool `json:"hasPendingExport"`
	LastConsentUpdate Time `json:"lastConsentUpdate"`
	TotalConsents int `json:"totalConsents"`
}

type ClientUpdateRequest struct {
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Require_pkce *bool `json:"require_pkce"`
	Tos_uri string `json:"tos_uri"`
	Trusted_client *bool `json:"trusted_client"`
	Allowed_scopes []string `json:"allowed_scopes"`
	Name string `json:"name"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
	Require_consent *bool `json:"require_consent"`
	Response_types []string `json:"response_types"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
	Name string `json:"name"`
	Role string `json:"role"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Metadata interface{} `json:"metadata"`
	Password string `json:"password"`
}

type BlockUserRequest struct {
	Reason string `json:"reason"`
}

type ApproveRecoveryResponse struct {
	SessionId xid.ID `json:"sessionId"`
	Approved bool `json:"approved"`
	ApprovedAt time.Time `json:"approvedAt"`
	Message string `json:"message"`
}

type TOTPFactorAdapter struct {
}

type AdminBlockUserRequest struct {
	Reason string `json:"reason"`
}

type SetupSecurityQuestionsResponse struct {
	Count int `json:"count"`
	Message string `json:"message"`
	SetupAt time.Time `json:"setupAt"`
}

type ComplianceReportsResponse struct {
	Reports []*interface{} `json:"reports"`
}

type PhoneVerifyResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type EncryptionService struct {
}

type StepUpPolicy struct {
	Id string `json:"id"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
	Priority int `json:"priority"`
	Rules interface{} `json:"rules"`
	Updated_at time.Time `json:"updated_at"`
	Enabled bool `json:"enabled"`
}

type UnblockUserRequest struct {
}

type GetTemplateResponse struct {
	Template interface{} `json:"template"`
}

type RevokeOthersResponse struct {
	RevokedCount int `json:"revokedCount"`
	Status string `json:"status"`
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

type VerificationRepository struct {
}

type TeamHandler struct {
}

type SignUpResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

type RevokeConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
}

type StartImpersonationRequest struct {
	Reason string `json:"reason"`
	Target_user_id string `json:"target_user_id"`
	Ticket_number string `json:"ticket_number"`
	Duration_minutes int `json:"duration_minutes"`
}

type TestPolicyResponse struct {
	FailedCount int `json:"failedCount"`
	Passed bool `json:"passed"`
	PassedCount int `json:"passedCount"`
	Results []TestCaseResult `json:"results"`
	Total int `json:"total"`
	Error string `json:"error"`
}

type NotificationStatusResponse struct {
	Status string `json:"status"`
}

type DeleteAppRequest struct {
}

type GetAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type InitiateChallengeResponse struct {
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	SessionId xid.ID `json:"sessionId"`
}

// MessageResponse represents Simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

type TrustedDevice struct {
	Metadata interface{} `json:"metadata"`
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	DeviceId string `json:"deviceId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	Name string `json:"name"`
	LastUsedAt Time `json:"lastUsedAt"`
}

type RemoveTrustedContactResponse struct {
	Status string `json:"status"`
}

type ListEvidenceResponse struct {
	Evidence []*interface{} `json:"evidence"`
}

type OIDCLoginRequest struct {
	Nonce string `json:"nonce"`
	RedirectUri string `json:"redirectUri"`
	Scope string `json:"scope"`
	State string `json:"state"`
}

type NotificationTemplateResponse struct {
	Template interface{} `json:"template"`
}

type CheckRegistry struct {
}

type ScopeResolver struct {
}

type ConfirmEmailChangeRequest struct {
	Token string `json:"token"`
}

type AuthURLResponse struct {
	Url string `json:"url"`
}

type ImpersonateUserRequestDTO struct {
	Duration time.Duration `json:"duration"`
}

type ConsentSettingsResponse struct {
	Settings interface{} `json:"settings"`
}

type EndImpersonationRequest struct {
	Impersonation_id string `json:"impersonation_id"`
	Reason string `json:"reason"`
}

type RevisionHandler struct {
}

type NotificationsResponse struct {
	Count int `json:"count"`
	Notifications interface{} `json:"notifications"`
}

type ApproveRecoveryRequest struct {
	Notes string `json:"notes"`
	SessionId xid.ID `json:"sessionId"`
}

type GetRecoveryStatsResponse struct {
	PendingRecoveries int `json:"pendingRecoveries"`
	TotalAttempts int `json:"totalAttempts"`
	AdminReviewsRequired int `json:"adminReviewsRequired"`
	FailedRecoveries int `json:"failedRecoveries"`
	HighRiskAttempts int `json:"highRiskAttempts"`
	SuccessRate float64 `json:"successRate"`
	SuccessfulRecoveries int `json:"successfulRecoveries"`
	AverageRiskScore float64 `json:"averageRiskScore"`
	MethodStats interface{} `json:"methodStats"`
}

type ComplianceCheckResponse struct {
	Id string `json:"id"`
}

type GetInvitationRequest struct {
}

type UpdateResponse struct {
	Webhook Webhook `json:"webhook"`
}

type ListPendingRequirementsResponse struct {
	Requirements []*interface{} `json:"requirements"`
}

type RequestReverificationResponse struct {
	Session interface{} `json:"session"`
}

type ComplianceTrainingsResponse struct {
	Training []*interface{} `json:"training"`
}

type ListPasskeysRequest struct {
}

// SuccessResponse represents Success boolean response
type SuccessResponse struct {
	Success bool `json:"success"`
}

type ConsentRecord struct {
	Granted bool `json:"granted"`
	IpAddress string `json:"ipAddress"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt Time `json:"expiresAt"`
	GrantedAt time.Time `json:"grantedAt"`
	Metadata JSONBMap `json:"metadata"`
	Purpose string `json:"purpose"`
	RevokedAt Time `json:"revokedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserId string `json:"userId"`
	ConsentType string `json:"consentType"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	Version string `json:"version"`
	UserAgent string `json:"userAgent"`
}

type CreateSecretRequest struct {
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Path string `json:"path"`
	Tags []string `json:"tags"`
	Value interface{} `json:"value"`
	ValueType string `json:"valueType"`
}

type GetStatusRequest struct {
	Device_id string `json:"device_id"`
	User_id string `json:"user_id"`
}

type RecoveryAttemptLog struct {
}

type SetupSecurityQuestionsRequest struct {
	Questions []SetupSecurityQuestionRequest `json:"questions"`
}

type GetSecretRequest struct {
}

type PolicyResponse struct {
	Actions []string `json:"actions"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserOrganizationId *string `json:"userOrganizationId"`
	Version int `json:"version"`
	CreatedBy string `json:"createdBy"`
	Description string `json:"description"`
	Id string `json:"id"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled bool `json:"enabled"`
	EnvironmentId string `json:"environmentId"`
	Priority int `json:"priority"`
	Expression string `json:"expression"`
	ResourceType string `json:"resourceType"`
}

type GetRequest struct {
}

type ConsentCookieResponse struct {
	Preferences interface{} `json:"preferences"`
}

type AuditLogResponse struct {
	Entries []*AuditLogEntry `json:"entries"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
}

type AcceptInvitationRequest struct {
	Token string `json:"token"`
}

type ValidatePolicyResponse struct {
	Message string `json:"message"`
	Valid bool `json:"valid"`
	Warnings []string `json:"warnings"`
	Complexity int `json:"complexity"`
	Error string `json:"error"`
	Errors []string `json:"errors"`
}

type ClientRegistrationRequest struct {
	Redirect_uris []string `json:"redirect_uris"`
	Trusted_client bool `json:"trusted_client"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Scope string `json:"scope"`
	Require_consent bool `json:"require_consent"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Tos_uri string `json:"tos_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Require_pkce bool `json:"require_pkce"`
	Response_types []string `json:"response_types"`
	Application_type string `json:"application_type"`
	Client_name string `json:"client_name"`
	Policy_uri string `json:"policy_uri"`
}

type ProviderDiscoveredResponse struct {
	Type string `json:"type"`
	Found bool `json:"found"`
	ProviderId string `json:"providerId"`
}

type CreateVerificationRequest struct {
}

type DeleteTeamRequest struct {
}

type WebAuthnFactorAdapter struct {
}

type UpdateClientRequest struct {
	Require_consent *bool `json:"require_consent"`
	Response_types []string `json:"response_types"`
	Trusted_client *bool `json:"trusted_client"`
	Allowed_scopes []string `json:"allowed_scopes"`
	Contacts []string `json:"contacts"`
	Name string `json:"name"`
	Require_pkce *bool `json:"require_pkce"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Tos_uri string `json:"tos_uri"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
}

type ConsentTypeStatus struct {
	Version string `json:"version"`
	ExpiresAt Time `json:"expiresAt"`
	Granted bool `json:"granted"`
	GrantedAt time.Time `json:"grantedAt"`
	NeedsRenewal bool `json:"needsRenewal"`
	Type string `json:"type"`
}

type AddCustomPermission_req struct {
	Category string `json:"category"`
	Description string `json:"description"`
	Name string `json:"name"`
}

type OIDCState struct {
}

type MultiSessionErrorResponse struct {
	Error string `json:"error"`
}

type SessionTokenResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type RunCheckResponse struct {
	Id string `json:"id"`
}

type RemoveMemberRequest struct {
}

type ContextRule struct {
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Condition string `json:"condition"`
	Description string `json:"description"`
	Name string `json:"name"`
}

type AuditEvent struct {
}

type GetReportResponse struct {
	Id string `json:"id"`
}

type SAMLLoginRequest struct {
	RelayState string `json:"relayState"`
}

type AccessConfig struct {
	AllowApiAccess bool `json:"allowApiAccess"`
	AllowDashboardAccess bool `json:"allowDashboardAccess"`
	RateLimitPerMinute int `json:"rateLimitPerMinute"`
	RequireAuthentication bool `json:"requireAuthentication"`
	RequireRbac bool `json:"requireRbac"`
}

type SessionAutoSendConfig struct {
	New_location bool `json:"new_location"`
	Suspicious_login bool `json:"suspicious_login"`
	All_revoked bool `json:"all_revoked"`
	Device_removed bool `json:"device_removed"`
	New_device bool `json:"new_device"`
}

type ScheduleVideoSessionResponse struct {
	Instructions string `json:"instructions"`
	JoinUrl string `json:"joinUrl"`
	Message string `json:"message"`
	ScheduledAt time.Time `json:"scheduledAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type UserServiceAdapter struct {
}

type VerifyRecoveryCodeResponse struct {
	Message string `json:"message"`
	RemainingCodes int `json:"remainingCodes"`
	Valid bool `json:"valid"`
}

type DataExportRequest struct {
	Status string `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
	ExportPath string `json:"exportPath"`
	Id xid.ID `json:"id"`
	IncludeSections []string `json:"includeSections"`
	UserId string `json:"userId"`
	ExpiresAt Time `json:"expiresAt"`
	ExportSize int64 `json:"exportSize"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	CompletedAt Time `json:"completedAt"`
	ErrorMessage string `json:"errorMessage"`
	Format string `json:"format"`
	CreatedAt time.Time `json:"createdAt"`
	ExportUrl string `json:"exportUrl"`
}

type ConnectionsResponse struct {
	Connections SocialAccount `json:"connections"`
}

type GetMigrationStatusResponse struct {
	HasMigratedPolicies bool `json:"hasMigratedPolicies"`
	LastMigrationAt string `json:"lastMigrationAt"`
	MigratedCount int `json:"migratedCount"`
	PendingRbacPolicies int `json:"pendingRbacPolicies"`
}

type CheckDependencies struct {
}

type ListFactorsRequest struct {
}

type IntrospectTokenResponse struct {
	Active bool `json:"active"`
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
	Sub string `json:"sub"`
	Username string `json:"username"`
	Aud []string `json:"aud"`
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
	Iss string `json:"iss"`
	Jti string `json:"jti"`
	Nbf int64 `json:"nbf"`
	Token_type string `json:"token_type"`
}

type SAMLCallbackResponse struct {
	Token string `json:"token"`
	User User `json:"user"`
	Session Session `json:"session"`
}

type ResourceTypeStats struct {
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	EvaluationCount int64 `json:"evaluationCount"`
	ResourceType string `json:"resourceType"`
	AllowRate float64 `json:"allowRate"`
}

type CreateAPIKeyResponse struct {
	Api_key APIKey `json:"api_key"`
	Message string `json:"message"`
}

type FactorsResponse struct {
	Count int `json:"count"`
	Factors interface{} `json:"factors"`
}

type CreateAppRequest struct {
}

type GetStatusResponse struct {
	Enabled bool `json:"enabled"`
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
	GracePeriod Time `json:"gracePeriod"`
	PolicyActive bool `json:"policyActive"`
	RequiredCount int `json:"requiredCount"`
	TrustedDevice bool `json:"trustedDevice"`
}

type MockSessionService struct {
}

type MigrateRBACRequest struct {
	DryRun bool `json:"dryRun"`
	KeepRbacPolicies bool `json:"keepRbacPolicies"`
	NamespaceId string `json:"namespaceId"`
	ValidateEquivalence bool `json:"validateEquivalence"`
}

type CreateABTestVariant_req struct {
	Weight int `json:"weight"`
	Body string `json:"body"`
	Name string `json:"name"`
	Subject string `json:"subject"`
}

type NotificationWebhookResponse struct {
	Status string `json:"status"`
}

type RecoverySessionInfo struct {
	CompletedAt Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	CurrentStep int `json:"currentStep"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	RequiresReview bool `json:"requiresReview"`
	TotalSteps int `json:"totalSteps"`
	UserEmail string `json:"userEmail"`
	Method RecoveryMethod `json:"method"`
	RiskScore float64 `json:"riskScore"`
	Status RecoveryStatus `json:"status"`
	UserId xid.ID `json:"userId"`
}

type CompliancePolicyResponse struct {
	Id string `json:"id"`
}

type AppServiceAdapter struct {
}

type FactorEnrollmentResponse struct {
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
	FactorId xid.ID `json:"factorId"`
	ProvisioningData interface{} `json:"provisioningData"`
}

type JWTService struct {
}

type ComplianceViolationResponse struct {
	Id string `json:"id"`
}

type ApproveDeletionRequestResponse struct {
	Status string `json:"status"`
}

type RegisterProviderRequest struct {
	ProviderId string `json:"providerId"`
	SamlCert string `json:"samlCert"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	Domain string `json:"domain"`
	OidcClientID string `json:"oidcClientID"`
	OidcClientSecret string `json:"oidcClientSecret"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
	SamlIssuer string `json:"samlIssuer"`
	Type string `json:"type"`
	AttributeMapping interface{} `json:"attributeMapping"`
	OidcIssuer string `json:"oidcIssuer"`
}

type CreateSessionHTTPRequest struct {
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
	Metadata interface{} `json:"metadata"`
}

type UpdateProviderRequest struct {
}

type RecoveryCodeUsage struct {
}

type GetStatsResponse struct {
	LocationCount int `json:"locationCount"`
	NewestSession *string `json:"newestSession"`
	OldestSession *string `json:"oldestSession"`
	TotalSessions int `json:"totalSessions"`
	ActiveSessions int `json:"activeSessions"`
	DeviceCount int `json:"deviceCount"`
}

type ListTeamsRequest struct {
}

type SecurityQuestionsConfig struct {
	RequireMinLength int `json:"requireMinLength"`
	AllowCustomQuestions bool `json:"allowCustomQuestions"`
	CaseSensitive bool `json:"caseSensitive"`
	MaxAttempts int `json:"maxAttempts"`
	PredefinedQuestions []string `json:"predefinedQuestions"`
	RequiredToRecover int `json:"requiredToRecover"`
	Enabled bool `json:"enabled"`
	ForbidCommonAnswers bool `json:"forbidCommonAnswers"`
	LockoutDuration time.Duration `json:"lockoutDuration"`
	MaxAnswerLength int `json:"maxAnswerLength"`
	MinimumQuestions int `json:"minimumQuestions"`
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

type RequestDataExportRequest struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type DataProcessingAgreement struct {
	CreatedAt time.Time `json:"createdAt"`
	ExpiryDate Time `json:"expiryDate"`
	OrganizationId string `json:"organizationId"`
	SignedByTitle string `json:"signedByTitle"`
	Version string `json:"version"`
	IpAddress string `json:"ipAddress"`
	Metadata JSONBMap `json:"metadata"`
	SignedByName string `json:"signedByName"`
	Content string `json:"content"`
	DigitalSignature string `json:"digitalSignature"`
	EffectiveDate time.Time `json:"effectiveDate"`
	AgreementType string `json:"agreementType"`
	Id xid.ID `json:"id"`
	SignedBy string `json:"signedBy"`
	SignedByEmail string `json:"signedByEmail"`
	Status string `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ValidateContentTypeRequest struct {
}

type UpdateAPIKeyRequest struct {
	Metadata interface{} `json:"metadata"`
	Name *string `json:"name"`
	Permissions interface{} `json:"permissions"`
	Rate_limit *int `json:"rate_limit"`
	Scopes []string `json:"scopes"`
	Allowed_ips []string `json:"allowed_ips"`
	Description *string `json:"description"`
}

type ListRecoverySessionsRequest struct {
	Status RecoveryStatus `json:"status"`
	OrganizationId string `json:"organizationId"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	RequiresReview bool `json:"requiresReview"`
}

type TOTPSecret struct {
}

type ComplianceTrainingResponse struct {
	Id string `json:"id"`
}

type DeletePasskeyRequest struct {
}

type MockStateStore struct {
}

type UnbanUserRequestDTO struct {
	Reason string `json:"reason"`
}

type DeleteResponse struct {
	Data interface{} `json:"data"`
	Message string `json:"message"`
	Success bool `json:"success"`
}

type SendVerificationCodeResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	MaskedTarget string `json:"maskedTarget"`
	Message string `json:"message"`
	Sent bool `json:"sent"`
}

type GetSessionResponse struct {
	User User `json:"user"`
	Session Session `json:"session"`
}

type RevokeDeviceResponse struct {
	Status string `json:"status"`
}

type CreatePolicyRequest struct {
	Name string `json:"name"`
	Renewable bool `json:"renewable"`
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	Description string `json:"description"`
	Required bool `json:"required"`
	ValidityPeriod *int `json:"validityPeriod"`
	Content string `json:"content"`
	Metadata interface{} `json:"metadata"`
}

type EnableResponse struct {
	Status string `json:"status"`
	Totp_uri string `json:"totp_uri"`
}

type BackupAuthSessionsResponse struct {
	Sessions []*interface{} `json:"sessions"`
}

type CreateEvidenceResponse struct {
	Id string `json:"id"`
}

type UpdatePolicyRequest struct {
	ValidityPeriod *int `json:"validityPeriod"`
	Active *bool `json:"active"`
	Content string `json:"content"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Renewable *bool `json:"renewable"`
	Required *bool `json:"required"`
}

type ConsentStats struct {
	GrantRate float64 `json:"grantRate"`
	GrantedCount int `json:"grantedCount"`
	RevokedCount int `json:"revokedCount"`
	TotalConsents int `json:"totalConsents"`
	Type string `json:"type"`
	AverageLifetime int `json:"averageLifetime"`
	ExpiredCount int `json:"expiredCount"`
}

type Handler struct {
}

type StatsResponse struct {
	Active_sessions int `json:"active_sessions"`
	Active_users int `json:"active_users"`
	Banned_users int `json:"banned_users"`
	Timestamp string `json:"timestamp"`
	Total_sessions int `json:"total_sessions"`
	Total_users int `json:"total_users"`
}

type ResourceResponse struct {
	NamespaceId string `json:"namespaceId"`
	Type string `json:"type"`
	Attributes ResourceAttribute `json:"attributes"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
}

type ListProfilesFilter struct {
	AppId *string `json:"appId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
}

type CreateProfileFromTemplateResponse struct {
	Id string `json:"id"`
}

type ClientDetailsResponse struct {
	RedirectURIs []string `json:"redirectURIs"`
	RequirePKCE bool `json:"requirePKCE"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	TrustedClient bool `json:"trustedClient"`
	AllowedScopes []string `json:"allowedScopes"`
	CreatedAt string `json:"createdAt"`
	GrantTypes []string `json:"grantTypes"`
	LogoURI string `json:"logoURI"`
	TosURI string `json:"tosURI"`
	Name string `json:"name"`
	OrganizationID string `json:"organizationID"`
	PolicyURI string `json:"policyURI"`
	UpdatedAt string `json:"updatedAt"`
	ApplicationType string `json:"applicationType"`
	ClientID string `json:"clientID"`
	RequireConsent bool `json:"requireConsent"`
	ResponseTypes []string `json:"responseTypes"`
	Contacts []string `json:"contacts"`
	IsOrgLevel bool `json:"isOrgLevel"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
}

type ListContentTypesRequest struct {
}

type ValidateResetTokenResponse struct {
	Valid bool `json:"valid"`
}

type GenerateBackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type VerifyCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type CreateProfileRequest struct {
	AppId string `json:"appId"`
	AuditLogExport bool `json:"auditLogExport"`
	DataResidency string `json:"dataResidency"`
	MfaRequired bool `json:"mfaRequired"`
	Name string `json:"name"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	RbacRequired bool `json:"rbacRequired"`
	ComplianceContact string `json:"complianceContact"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	PasswordMinLength int `json:"passwordMinLength"`
	RegularAccessReview bool `json:"regularAccessReview"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	Metadata interface{} `json:"metadata"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	Standards []ComplianceStandard `json:"standards"`
	DpoContact string `json:"dpoContact"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	LeastPrivilege bool `json:"leastPrivilege"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	RetentionDays int `json:"retentionDays"`
	SessionMaxAge int `json:"sessionMaxAge"`
}

type Challenge struct {
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
	VerifiedAt Time `json:"verifiedAt"`
	Attempts int `json:"attempts"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorId xid.ID `json:"factorId"`
	IpAddress string `json:"ipAddress"`
	MaxAttempts int `json:"maxAttempts"`
	Metadata interface{} `json:"metadata"`
	Id xid.ID `json:"id"`
	Status ChallengeStatus `json:"status"`
	Type FactorType `json:"type"`
}

type UpdateOrganizationHandlerRequest struct {
}

type ProvidersResponse struct {
	Providers []string `json:"providers"`
}

type ResendResponse struct {
	Status string `json:"status"`
}

type SMSProviderConfig struct {
	Config interface{} `json:"config"`
	From string `json:"from"`
	Provider string `json:"provider"`
}

type StepUpVerificationResponse struct {
	Expires_at string `json:"expires_at"`
	Verified bool `json:"verified"`
}

type AdminBypassRequest struct {
	Duration int `json:"duration"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
}

type GetUserTrainingResponse struct {
	User_id string `json:"user_id"`
}

type DashboardExtension struct {
}

type SMSConfig struct {
	Provider string `json:"provider"`
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
}

type RequestEmailChangeRequest struct {
	NewEmail string `json:"newEmail"`
}

type mockProvider struct {
}

type sessionStats struct {
}

type MockUserService struct {
}

type ProviderConfigResponse struct {
	Provider string `json:"provider"`
	AppId string `json:"appId"`
	Message string `json:"message"`
}

type TestPolicyRequest struct {
	Expression string `json:"expression"`
	ResourceType string `json:"resourceType"`
	TestCases []TestCase `json:"testCases"`
	Actions []string `json:"actions"`
}

type WebhookResponse struct {
	Received bool `json:"received"`
	Status string `json:"status"`
}

type AsyncConfig struct {
	Max_retries int `json:"max_retries"`
	Persist_failures bool `json:"persist_failures"`
	Queue_size int `json:"queue_size"`
	Retry_backoff []string `json:"retry_backoff"`
	Retry_enabled bool `json:"retry_enabled"`
	Worker_pool_size int `json:"worker_pool_size"`
	Enabled bool `json:"enabled"`
}

type BackupCodesConfig struct {
	Length int `json:"length"`
	Allow_reuse bool `json:"allow_reuse"`
	Count int `json:"count"`
	Enabled bool `json:"enabled"`
	Format string `json:"format"`
}

type Service struct {
}

type UpdateNamespaceRequest struct {
	InheritPlatform *bool `json:"inheritPlatform"`
	Name string `json:"name"`
	Description string `json:"description"`
}

type RequestEmailChangeResponse struct {
	Message string `json:"message"`
}

type NotificationPreviewResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

type FinishRegisterResponse struct {
	CreatedAt time.Time `json:"createdAt"`
	CredentialId string `json:"credentialId"`
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
	Status string `json:"status"`
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

type CreateTeamHandlerRequest struct {
}

type BatchEvaluateRequest struct {
	Requests []EvaluateRequest `json:"requests"`
}

type UserVerificationStatusResponse struct {
	Status UserVerificationStatus `json:"status"`
}

type Adapter struct {
}

type VerifyEnrolledFactorRequest struct {
	Code string `json:"code"`
	Data interface{} `json:"data"`
}

type GetConsentPolicyResponse struct {
	Id string `json:"id"`
}

type PreviewTemplateResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

// Session represents User session
type Session struct {
	ExpiresAt string `json:"expiresAt"`
	IpAddress *string `json:"ipAddress,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
	UserId string `json:"userId"`
	Token string `json:"token"`
}

type DataExportConfig struct {
	MaxRequests int `json:"maxRequests"`
	StoragePath string `json:"storagePath"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	ExpiryHours int `json:"expiryHours"`
	MaxExportSize int64 `json:"maxExportSize"`
	RequestPeriod time.Duration `json:"requestPeriod"`
	AllowedFormats []string `json:"allowedFormats"`
	AutoCleanup bool `json:"autoCleanup"`
	DefaultFormat string `json:"defaultFormat"`
	Enabled bool `json:"enabled"`
	IncludeSections []string `json:"includeSections"`
}

type PrivacySettingsRequest struct {
	ConsentRequired *bool `json:"consentRequired"`
	DataRetentionDays *int `json:"dataRetentionDays"`
	DpoEmail string `json:"dpoEmail"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
	AllowDataPortability *bool `json:"allowDataPortability"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	ExportFormat []string `json:"exportFormat"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	CcpaMode *bool `json:"ccpaMode"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	GdprMode *bool `json:"gdprMode"`
}

type SetUserRoleRequest struct {
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Role string `json:"role"`
}

type TrustedContact struct {
}

type TrustDeviceRequest struct {
	DeviceId string `json:"deviceId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
}

type MigrateAllResponse struct {
	SkippedPolicies int `json:"skippedPolicies"`
	CompletedAt string `json:"completedAt"`
	DryRun bool `json:"dryRun"`
	StartedAt string `json:"startedAt"`
	TotalPolicies int `json:"totalPolicies"`
	ConvertedPolicies []PolicyPreviewResponse `json:"convertedPolicies"`
	Errors []MigrationErrorResponse `json:"errors"`
	FailedPolicies int `json:"failedPolicies"`
	MigratedPolicies int `json:"migratedPolicies"`
}

type PublishEntryRequest struct {
}

type AutomatedChecksConfig struct {
	AccessReview bool `json:"accessReview"`
	CheckInterval time.Duration `json:"checkInterval"`
	DataRetention bool `json:"dataRetention"`
	Enabled bool `json:"enabled"`
	InactiveUsers bool `json:"inactiveUsers"`
	MfaCoverage bool `json:"mfaCoverage"`
	SuspiciousActivity bool `json:"suspiciousActivity"`
	PasswordPolicy bool `json:"passwordPolicy"`
	SessionPolicy bool `json:"sessionPolicy"`
}

type GetClientResponse struct {
	ClientID string `json:"clientID"`
	Name string `json:"name"`
	OrganizationID string `json:"organizationID"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	RequirePKCE bool `json:"requirePKCE"`
	Contacts []string `json:"contacts"`
	IsOrgLevel bool `json:"isOrgLevel"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	TrustedClient bool `json:"trustedClient"`
	RequireConsent bool `json:"requireConsent"`
	CreatedAt string `json:"createdAt"`
	GrantTypes []string `json:"grantTypes"`
	PolicyURI string `json:"policyURI"`
	RedirectURIs []string `json:"redirectURIs"`
	UpdatedAt string `json:"updatedAt"`
	AllowedScopes []string `json:"allowedScopes"`
	ApplicationType string `json:"applicationType"`
	LogoURI string `json:"logoURI"`
	ResponseTypes []string `json:"responseTypes"`
	TosURI string `json:"tosURI"`
}

type FacialCheckConfig struct {
	Enabled bool `json:"enabled"`
	MotionCapture bool `json:"motionCapture"`
	Variant string `json:"variant"`
}

type mockSentNotification struct {
}

type BeginRegisterResponse struct {
	Challenge string `json:"challenge"`
	Options interface{} `json:"options"`
	Timeout time.Duration `json:"timeout"`
	UserId string `json:"userId"`
}

type VerifyImpersonationRequest struct {
}

type PreviewConversionResponse struct {
	Success bool `json:"success"`
	CelExpression string `json:"celExpression"`
	Error string `json:"error"`
	PolicyName string `json:"policyName"`
	ResourceId string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
}

type ComplianceReportResponse struct {
	Id string `json:"id"`
}

type ComplianceEvidenceResponse struct {
	Id string `json:"id"`
}

type EmailFactorAdapter struct {
}

type VerificationRequest struct {
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data interface{} `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
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

type ComplianceEvidence struct {
	EvidenceType string `json:"evidenceType"`
	FileHash string `json:"fileHash"`
	FileUrl string `json:"fileUrl"`
	Id string `json:"id"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	AppId string `json:"appId"`
	CollectedBy string `json:"collectedBy"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	ControlId string `json:"controlId"`
}

type AuditServiceAdapter struct {
}

type Factor struct {
	Id xid.ID `json:"id"`
	LastUsedAt Time `json:"lastUsedAt"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	VerifiedAt Time `json:"verifiedAt"`
	Metadata interface{} `json:"metadata"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserId xid.ID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt Time `json:"expiresAt"`
}

type AMLMatch struct {
}

type StepUpErrorResponse struct {
	Error string `json:"error"`
}

type RemoveTeamMemberRequest struct {
}

type CreateConsentPolicyRequest struct {
	Name string `json:"name"`
	Required bool `json:"required"`
	Version string `json:"version"`
	Content string `json:"content"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Renewable bool `json:"renewable"`
	ValidityPeriod *int `json:"validityPeriod"`
	ConsentType string `json:"consentType"`
}

type BatchEvaluateResponse struct {
	TotalEvaluations int `json:"totalEvaluations"`
	TotalTimeMs float64 `json:"totalTimeMs"`
	FailureCount int `json:"failureCount"`
	Results []*BatchEvaluationResult `json:"results"`
	SuccessCount int `json:"successCount"`
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

type NotificationListResponse struct {
	Total int `json:"total"`
	Notifications []*interface{} `json:"notifications"`
}

type MigrationErrorResponse struct {
	Error string `json:"error"`
	PolicyIndex int `json:"policyIndex"`
	Resource string `json:"resource"`
	Subject string `json:"subject"`
}

type SendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type VideoSessionInfo struct {
}

type StepUpPolicyResponse struct {
	Id string `json:"id"`
}

type StepUpVerification struct {
	Verified_at time.Time `json:"verified_at"`
	Ip string `json:"ip"`
	Reason string `json:"reason"`
	Security_level SecurityLevel `json:"security_level"`
	User_id string `json:"user_id"`
	Device_id string `json:"device_id"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Method VerificationMethod `json:"method"`
	Org_id string `json:"org_id"`
	Session_id string `json:"session_id"`
	Expires_at time.Time `json:"expires_at"`
	User_agent string `json:"user_agent"`
	Created_at time.Time `json:"created_at"`
	Rule_name string `json:"rule_name"`
}

type SecurityQuestion struct {
}

type AddTeamMember_req struct {
	Member_id xid.ID `json:"member_id"`
	Role string `json:"role"`
}

type ReviewDocumentResponse struct {
	Status string `json:"status"`
}

type UpdatePolicyResponse struct {
	Id string `json:"id"`
}

type ListImpersonationsRequest struct {
}

type MockService struct {
}

type ComplianceViolationsResponse struct {
	Violations []*interface{} `json:"violations"`
}

type UserInfoResponse struct {
	Phone_number_verified bool `json:"phone_number_verified"`
	Picture string `json:"picture"`
	Email string `json:"email"`
	Family_name string `json:"family_name"`
	Given_name string `json:"given_name"`
	Phone_number string `json:"phone_number"`
	Updated_at int64 `json:"updated_at"`
	Email_verified bool `json:"email_verified"`
	Middle_name string `json:"middle_name"`
	Sub string `json:"sub"`
	Birthdate string `json:"birthdate"`
	Gender string `json:"gender"`
	Locale string `json:"locale"`
	Nickname string `json:"nickname"`
	Profile string `json:"profile"`
	Zoneinfo string `json:"zoneinfo"`
	Name string `json:"name"`
	Preferred_username string `json:"preferred_username"`
	Website string `json:"website"`
}

type ProvidersAppResponse struct {
	AppId string `json:"appId"`
	Providers []string `json:"providers"`
}

type DeleteFieldRequest struct {
}

type IDVerificationSessionResponse struct {
	Session interface{} `json:"session"`
}

type GetProfileResponse struct {
	Id string `json:"id"`
}

type IntrospectionService struct {
}

type NamespaceResponse struct {
	Id string `json:"id"`
	Name string `json:"name"`
	ResourceCount int `json:"resourceCount"`
	UpdatedAt time.Time `json:"updatedAt"`
	AppId string `json:"appId"`
	Description string `json:"description"`
	InheritPlatform bool `json:"inheritPlatform"`
	PolicyCount int `json:"policyCount"`
	TemplateId *string `json:"templateId"`
	UserOrganizationId *string `json:"userOrganizationId"`
	ActionCount int `json:"actionCount"`
	CreatedAt time.Time `json:"createdAt"`
	EnvironmentId string `json:"environmentId"`
}

type RequestTrustedContactVerificationResponse struct {
	ContactId xid.ID `json:"contactId"`
	ContactName string `json:"contactName"`
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	NotifiedAt time.Time `json:"notifiedAt"`
}

type VersioningConfig struct {
	AutoCleanup bool `json:"autoCleanup"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	MaxVersions int `json:"maxVersions"`
	RetentionDays int `json:"retentionDays"`
}

type BatchEvaluationResult struct {
	Error string `json:"error"`
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
	Index int `json:"index"`
	Policies []string `json:"policies"`
	ResourceId string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Action string `json:"action"`
	Allowed bool `json:"allowed"`
}

type TemplateDefault struct {
}

type RegenerateCodesRequest struct {
	Count int `json:"count"`
	User_id string `json:"user_id"`
}

type ScopeDefinition struct {
}

type VerifyTrustedContactResponse struct {
	ContactId xid.ID `json:"contactId"`
	Message string `json:"message"`
	Verified bool `json:"verified"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

type ContinueRecoveryRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
}

type UpdateClientResponse struct {
	Contacts []string `json:"contacts"`
	CreatedAt string `json:"createdAt"`
	LogoURI string `json:"logoURI"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	RequirePKCE bool `json:"requirePKCE"`
	UpdatedAt string `json:"updatedAt"`
	ApplicationType string `json:"applicationType"`
	Name string `json:"name"`
	PolicyURI string `json:"policyURI"`
	RequireConsent bool `json:"requireConsent"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	TosURI string `json:"tosURI"`
	TrustedClient bool `json:"trustedClient"`
	AllowedScopes []string `json:"allowedScopes"`
	ClientID string `json:"clientID"`
	GrantTypes []string `json:"grantTypes"`
	OrganizationID string `json:"organizationID"`
	RedirectURIs []string `json:"redirectURIs"`
	ResponseTypes []string `json:"responseTypes"`
	IsOrgLevel bool `json:"isOrgLevel"`
}

type CallbackResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type UnbanUserRequest struct {
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
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

type EmailServiceAdapter struct {
}

type ListPoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type AdminUnblockUserResponse struct {
	Status interface{} `json:"status"`
}

type ConsentsResponse struct {
	Consents interface{} `json:"consents"`
	Count int `json:"count"`
}

type RotateAPIKeyRequest struct {
}

type BackupAuthStatsResponse struct {
	Stats interface{} `json:"stats"`
}

type ResourceRule struct {
	Description string `json:"description"`
	Org_id string `json:"org_id"`
	Resource_type string `json:"resource_type"`
	Security_level SecurityLevel `json:"security_level"`
	Sensitivity string `json:"sensitivity"`
	Action string `json:"action"`
}

type GetAppRequest struct {
}

type RateLimiter struct {
}

type UpdateTeamRequest struct {
	Description string `json:"description"`
	Name string `json:"name"`
}

type ConnectionResponse struct {
	Connection SocialAccount `json:"connection"`
}

type EnableRequest2FA struct {
	Method string `json:"method"`
	User_id string `json:"user_id"`
}

type TokenRequest struct {
	Audience string `json:"audience"`
	Client_id string `json:"client_id"`
	Code_verifier string `json:"code_verifier"`
	Grant_type string `json:"grant_type"`
	Scope string `json:"scope"`
	Client_secret string `json:"client_secret"`
	Code string `json:"code"`
	Redirect_uri string `json:"redirect_uri"`
	Refresh_token string `json:"refresh_token"`
}

type ListJWTKeysRequest struct {
}

// Webhook represents Webhook configuration
type Webhook struct {
	Secret string `json:"secret"`
	Enabled bool `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
	OrganizationId string `json:"organizationId"`
	Url string `json:"url"`
	Events []string `json:"events"`
}

type CookieConsentRequest struct {
	BannerVersion string `json:"bannerVersion"`
	Essential bool `json:"essential"`
	Functional bool `json:"functional"`
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	Analytics bool `json:"analytics"`
}

type CreateJWTKeyRequest struct {
	Algorithm string `json:"algorithm"`
	Curve string `json:"curve"`
	ExpiresAt Time `json:"expiresAt"`
	IsPlatformKey bool `json:"isPlatformKey"`
	KeyType string `json:"keyType"`
	Metadata interface{} `json:"metadata"`
}

type CompleteTrainingRequest struct {
	Score int `json:"score"`
}

type GetImpersonationRequest struct {
}

type TemplateEngine struct {
}

type TwoFAStatusDetailResponse struct {
	Enabled bool `json:"enabled"`
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
}

type ConsentExportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type ConsentService struct {
}

type TrustedContactInfo struct {
	Active bool `json:"active"`
	Email string `json:"email"`
	Id xid.ID `json:"id"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
	Verified bool `json:"verified"`
	VerifiedAt Time `json:"verifiedAt"`
}

type RollbackRequest struct {
	Reason string `json:"reason"`
}

type DisableRequest struct {
	User_id string `json:"user_id"`
}

type AuthorizeRequest struct {
	Login_hint string `json:"login_hint"`
	Max_age *int `json:"max_age"`
	Nonce string `json:"nonce"`
	Prompt string `json:"prompt"`
	Scope string `json:"scope"`
	Acr_values string `json:"acr_values"`
	Id_token_hint string `json:"id_token_hint"`
	Redirect_uri string `json:"redirect_uri"`
	Response_type string `json:"response_type"`
	State string `json:"state"`
	Ui_locales string `json:"ui_locales"`
	Client_id string `json:"client_id"`
	Code_challenge string `json:"code_challenge"`
	Code_challenge_method string `json:"code_challenge_method"`
}

type GetDataExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type ListTrustedContactsResponse struct {
	Count int `json:"count"`
	Contacts []TrustedContactInfo `json:"contacts"`
}

type MockOrganizationUIExtension struct {
}

type ProviderRegisteredResponse struct {
	Status string `json:"status"`
	Type string `json:"type"`
	ProviderId string `json:"providerId"`
}

type TestCase struct {
	Action string `json:"action"`
	Expected bool `json:"expected"`
	Name string `json:"name"`
	Principal interface{} `json:"principal"`
	Request interface{} `json:"request"`
	Resource interface{} `json:"resource"`
}

type UpdateFieldRequest struct {
}

type TwoFAEnableResponse struct {
	Status string `json:"status"`
	Totp_uri string `json:"totp_uri"`
}

type RotateAPIKeyResponse struct {
	Api_key APIKey `json:"api_key"`
	Message string `json:"message"`
}

type GetByPathRequest struct {
}

type Middleware struct {
}

type ResendNotificationRequest struct {
}

type GetTemplateAnalyticsRequest struct {
}

type MockAuditService struct {
}

type CompleteTrainingResponse struct {
	Status string `json:"status"`
}

type AutoSendConfig struct {
	Organization OrganizationAutoSendConfig `json:"organization"`
	Session SessionAutoSendConfig `json:"session"`
	Account AccountAutoSendConfig `json:"account"`
	Auth AuthAutoSendConfig `json:"auth"`
}

type StepUpRequirementsResponse struct {
	Requirements []*interface{} `json:"requirements"`
}

type GetOrganizationBySlugRequest struct {
}

type TwoFABackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type TrustedContactsConfig struct {
	AllowEmailContacts bool `json:"allowEmailContacts"`
	AllowPhoneContacts bool `json:"allowPhoneContacts"`
	Enabled bool `json:"enabled"`
	MaxNotificationsPerDay int `json:"maxNotificationsPerDay"`
	MaximumContacts int `json:"maximumContacts"`
	MinimumContacts int `json:"minimumContacts"`
	VerificationExpiry time.Duration `json:"verificationExpiry"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	RequireVerification bool `json:"requireVerification"`
	RequiredToRecover int `json:"requiredToRecover"`
}

type MockEmailService struct {
}

type GetConsentAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type UpdateRecoveryConfigResponse struct {
	Config interface{} `json:"config"`
}

type ComplianceChecksResponse struct {
	Checks []*interface{} `json:"checks"`
}

type mockUserService struct {
}

type SetUserRoleRequestDTO struct {
	Role string `json:"role"`
}

type RestoreTemplateVersionRequest struct {
}

type AddMemberRequest struct {
	Role string `json:"role"`
	User_id string `json:"user_id"`
}

type navItem struct {
}

type SendRequest struct {
	Email string `json:"email"`
}

type GetEffectivePermissionsRequest struct {
}

type ListRecoverySessionsResponse struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Sessions []RecoverySessionInfo `json:"sessions"`
	TotalCount int `json:"totalCount"`
}

type TwoFARequiredResponse struct {
	Device_id string `json:"device_id"`
	Require_twofa bool `json:"require_twofa"`
	User User `json:"user"`
}

type ConsentDashboardConfig struct {
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
	ShowAuditLog bool `json:"showAuditLog"`
	ShowConsentHistory bool `json:"showConsentHistory"`
	ShowCookiePreferences bool `json:"showCookiePreferences"`
	ShowDataDeletion bool `json:"showDataDeletion"`
	ShowDataExport bool `json:"showDataExport"`
	ShowPolicies bool `json:"showPolicies"`
}

type MembersResponse struct {
	Members Member `json:"members"`
	Total int `json:"total"`
}

type TrackNotificationEvent_req struct {
	Event string `json:"event"`
	EventData *interface{} `json:"eventData,omitempty"`
	NotificationId string `json:"notificationId"`
	OrganizationId *string `json:"organizationId,omitempty"`
	TemplateId string `json:"templateId"`
}

type MockAppService struct {
}

type TestCaseResult struct {
	Expected bool `json:"expected"`
	Name string `json:"name"`
	Passed bool `json:"passed"`
	Actual bool `json:"actual"`
	Error string `json:"error"`
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
}

type VerificationsResponse struct {
	Count int `json:"count"`
	Verifications interface{} `json:"verifications"`
}

type LimitResult struct {
}

type ListAppsRequest struct {
}

type SignOutResponse struct {
	Success bool `json:"success"`
}

type ResetAllTemplatesResponse struct {
	Status string `json:"status"`
}

type OAuthErrorResponse struct {
	Error string `json:"error"`
	Error_description string `json:"error_description"`
	Error_uri string `json:"error_uri"`
	State string `json:"state"`
}

type BackupAuthVideoResponse struct {
	Session_id string `json:"session_id"`
}

type ListTrainingFilter struct {
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	TrainingType *string `json:"trainingType"`
	UserId *string `json:"userId"`
	AppId *string `json:"appId"`
}

type JWK struct {
	Alg string `json:"alg"`
	E string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N string `json:"n"`
	Use string `json:"use"`
}

type DeleteOrganizationRequest struct {
}

type GetEntryStatsRequest struct {
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

type GenerateRecoveryCodesResponse struct {
	Codes []string `json:"codes"`
	Count int `json:"count"`
	GeneratedAt time.Time `json:"generatedAt"`
	Warning string `json:"warning"`
}

type ScheduleVideoSessionRequest struct {
	TimeZone string `json:"timeZone"`
	ScheduledAt time.Time `json:"scheduledAt"`
	SessionId xid.ID `json:"sessionId"`
}

type InviteMemberHandlerRequest struct {
}

type ResendRequest struct {
	Email string `json:"email"`
}

type RateLimitConfig struct {
	Enabled bool `json:"enabled"`
	Window time.Duration `json:"window"`
}

type ListPoliciesFilter struct {
	PolicyType *string `json:"policyType"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
}

type DeclineInvitationRequest struct {
	Token string `json:"token"`
}

type GetNotificationRequest struct {
}

type ComplianceTemplatesResponse struct {
	Templates []*interface{} `json:"templates"`
}

type ContentTypeHandler struct {
}

type OIDCLoginResponse struct {
	AuthUrl string `json:"authUrl"`
	Nonce string `json:"nonce"`
	ProviderId string `json:"providerId"`
	State string `json:"state"`
}

type MigrateAllRequest struct {
	DryRun bool `json:"dryRun"`
	PreserveOriginal bool `json:"preserveOriginal"`
}

type RiskEngine struct {
}

type BulkRequest struct {
	Ids []string `json:"ids"`
}

type RestoreRevisionRequest struct {
}

type QueryEntriesRequest struct {
}

type CreateAPIKeyRequest struct {
	Scopes []string `json:"scopes"`
	Allowed_ips []string `json:"allowed_ips"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Permissions interface{} `json:"permissions"`
	Rate_limit int `json:"rate_limit"`
}

type RunCheckRequest struct {
	CheckType string `json:"checkType"`
}

type DeleteFactorRequest struct {
}

type ErrorResponse struct {
	Error string `json:"error"`
	Message string `json:"message"`
}

type UpdateConsentResponse struct {
	Id string `json:"id"`
}

type SessionStats struct {
}

type VerifyRequest struct {
}

type EmailProviderConfig struct {
	From_name string `json:"from_name"`
	Provider string `json:"provider"`
	Reply_to string `json:"reply_to"`
	Config interface{} `json:"config"`
	From string `json:"from"`
}

type GetRequirementResponse struct {
	Id string `json:"id"`
}

type DiscoveryService struct {
}

type CreateVerificationSessionRequest struct {
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
	Metadata interface{} `json:"metadata"`
}

type ConsentStatusResponse struct {
	Status string `json:"status"`
}

type DataDeletionRequestInput struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type ImpersonationContext struct {
	Indicator_message string `json:"indicator_message"`
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id ID `json:"target_user_id"`
	Impersonation_id ID `json:"impersonation_id"`
	Impersonator_id ID `json:"impersonator_id"`
}

type SaveNotificationSettings_req struct {
	RetryDelay string `json:"retryDelay"`
	AutoSendWelcome bool `json:"autoSendWelcome"`
	CleanupAfter string `json:"cleanupAfter"`
	RetryAttempts int `json:"retryAttempts"`
}

type AdminPolicyRequest struct {
	AllowedTypes []string `json:"allowedTypes"`
	Enabled bool `json:"enabled"`
	GracePeriod int `json:"gracePeriod"`
	RequiredFactors int `json:"requiredFactors"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

type RevokeTokenRequest struct {
	Token_type_hint string `json:"token_type_hint"`
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
}

type ResetPasswordResponse struct {
	Message string `json:"message"`
}

type userServiceAdapter struct {
}

type IDVerificationResponse struct {
	Verification interface{} `json:"verification"`
}

type GetABTestResultsRequest struct {
}

type ComplianceUserTrainingResponse struct {
	User_id string `json:"user_id"`
}

type EvaluateRequest struct {
	Action string `json:"action"`
	Context interface{} `json:"context"`
	Principal interface{} `json:"principal"`
	Request interface{} `json:"request"`
	Resource interface{} `json:"resource"`
	ResourceId string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
}

type StepUpDevicesResponse struct {
	Count int `json:"count"`
	Devices interface{} `json:"devices"`
}

type SendNotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type UpdateMemberHandlerRequest struct {
}

type PreviewTemplate_req struct {
	Variables interface{} `json:"variables"`
}

type GetPrivacySettingsResponse struct {
	Settings interface{} `json:"settings"`
}

type mockSessionService struct {
}

type AdminUpdateProviderRequest struct {
	ClientId *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
	Enabled *bool `json:"enabled"`
	Scopes []string `json:"scopes"`
}

type OrganizationAutoSendConfig struct {
	Role_changed bool `json:"role_changed"`
	Transfer bool `json:"transfer"`
	Deleted bool `json:"deleted"`
	Invite bool `json:"invite"`
	Member_added bool `json:"member_added"`
	Member_left bool `json:"member_left"`
	Member_removed bool `json:"member_removed"`
}

type App struct {
}

type RiskAssessment struct {
	Metadata interface{} `json:"metadata"`
	Recommended []FactorType `json:"recommended"`
	Score float64 `json:"score"`
	Factors []string `json:"factors"`
	Level RiskLevel `json:"level"`
}

type GetCookieConsentResponse struct {
	Preferences interface{} `json:"preferences"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type UnpublishContentTypeRequest struct {
}

type GetNotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type DiscoveryResponse struct {
	Revocation_endpoint_auth_methods_supported []string `json:"revocation_endpoint_auth_methods_supported"`
	Subject_types_supported []string `json:"subject_types_supported"`
	Token_endpoint string `json:"token_endpoint"`
	Claims_parameter_supported bool `json:"claims_parameter_supported"`
	Grant_types_supported []string `json:"grant_types_supported"`
	Issuer string `json:"issuer"`
	Request_uri_parameter_supported bool `json:"request_uri_parameter_supported"`
	Require_request_uri_registration bool `json:"require_request_uri_registration"`
	Response_modes_supported []string `json:"response_modes_supported"`
	Response_types_supported []string `json:"response_types_supported"`
	Userinfo_endpoint string `json:"userinfo_endpoint"`
	Authorization_endpoint string `json:"authorization_endpoint"`
	Introspection_endpoint string `json:"introspection_endpoint"`
	Introspection_endpoint_auth_methods_supported []string `json:"introspection_endpoint_auth_methods_supported"`
	Request_parameter_supported bool `json:"request_parameter_supported"`
	Scopes_supported []string `json:"scopes_supported"`
	Code_challenge_methods_supported []string `json:"code_challenge_methods_supported"`
	Id_token_signing_alg_values_supported []string `json:"id_token_signing_alg_values_supported"`
	Registration_endpoint string `json:"registration_endpoint"`
	Revocation_endpoint string `json:"revocation_endpoint"`
	Token_endpoint_auth_methods_supported []string `json:"token_endpoint_auth_methods_supported"`
	Claims_supported []string `json:"claims_supported"`
	Jwks_uri string `json:"jwks_uri"`
}

type TwoFAErrorResponse struct {
	Error string `json:"error"`
}

type BackupAuthDocumentResponse struct {
	Id string `json:"id"`
}

type ConsentNotificationsConfig struct {
	NotifyOnExpiry bool `json:"notifyOnExpiry"`
	NotifyOnGrant bool `json:"notifyOnGrant"`
	Channels []string `json:"channels"`
	NotifyDeletionComplete bool `json:"notifyDeletionComplete"`
	NotifyOnRevoke bool `json:"notifyOnRevoke"`
	Enabled bool `json:"enabled"`
	NotifyDeletionApproved bool `json:"notifyDeletionApproved"`
	NotifyDpoEmail string `json:"notifyDpoEmail"`
	NotifyExportReady bool `json:"notifyExportReady"`
}

type MigrateRolesRequest struct {
	DryRun bool `json:"dryRun"`
}

type BackupAuthContactResponse struct {
	Id string `json:"id"`
}

type ListEvidenceFilter struct {
	AppId *string `json:"appId"`
	ControlId *string `json:"controlId"`
	EvidenceType *string `json:"evidenceType"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
}

type mockRepository struct {
}

type UpdateTemplateRequest struct {
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

type CreatePolicyResponse struct {
	Id string `json:"id"`
}

type StartVideoSessionResponse struct {
	Message string `json:"message"`
	SessionUrl string `json:"sessionUrl"`
	StartedAt time.Time `json:"startedAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type StepUpRememberedDevice struct {
	Security_level SecurityLevel `json:"security_level"`
	User_agent string `json:"user_agent"`
	Device_id string `json:"device_id"`
	Device_name string `json:"device_name"`
	Expires_at time.Time `json:"expires_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Org_id string `json:"org_id"`
	Remembered_at time.Time `json:"remembered_at"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Last_used_at time.Time `json:"last_used_at"`
}

type DeletePolicyResponse struct {
	Status string `json:"status"`
}

type GetRecoveryConfigResponse struct {
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
}

type DiscoverProviderRequest struct {
	Email string `json:"email"`
}

type BackupAuthRecoveryResponse struct {
	Session_id string `json:"session_id"`
}

type StartVideoSessionRequest struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type GetDocumentVerificationRequest struct {
	DocumentId xid.ID `json:"documentId"`
}

type SAMLSPMetadataResponse struct {
	Metadata string `json:"metadata"`
}

type RevokeAllResponse struct {
	RevokedCount int `json:"revokedCount"`
	Status string `json:"status"`
}

type SetActiveRequest struct {
	Id string `json:"id"`
}

type ConsentRequest struct {
	Code_challenge string `json:"code_challenge"`
	Code_challenge_method string `json:"code_challenge_method"`
	Redirect_uri string `json:"redirect_uri"`
	Response_type string `json:"response_type"`
	Scope string `json:"scope"`
	State string `json:"state"`
	Action string `json:"action"`
	Client_id string `json:"client_id"`
}

type ClientAuthenticator struct {
}

type SecretsConfigSource struct {
}

type AdminAddProviderRequest struct {
	AppId xid.ID `json:"appId"`
	ClientId string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
}

type PreviewTemplateRequest struct {
}

type NoOpEmailProvider struct {
}

type VerifySecurityAnswersResponse struct {
	Valid bool `json:"valid"`
	AttemptsLeft int `json:"attemptsLeft"`
	CorrectAnswers int `json:"correctAnswers"`
	Message string `json:"message"`
	RequiredAnswers int `json:"requiredAnswers"`
}

type OIDCCallbackResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
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

type VerifyImpersonationResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id string `json:"target_user_id"`
}

type GetViolationResponse struct {
	Id string `json:"id"`
}

type DeleteTemplateRequest struct {
}

type CompleteVideoSessionResponse struct {
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	Result string `json:"result"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type CancelRecoveryResponse struct {
	Status string `json:"status"`
}

type ListMembersRequest struct {
}

type CloneContentTypeRequest struct {
}

type BaseFactorAdapter struct {
}

type GetTreeRequest struct {
}

type GetRevisionRequest struct {
}

type ConsentReport struct {
	ConsentRate float64 `json:"consentRate"`
	DataExportsThisPeriod int `json:"dataExportsThisPeriod"`
	DpasActive int `json:"dpasActive"`
	ReportPeriodEnd time.Time `json:"reportPeriodEnd"`
	UsersWithConsent int `json:"usersWithConsent"`
	CompletedDeletions int `json:"completedDeletions"`
	ConsentsByType interface{} `json:"consentsByType"`
	DpasExpiringSoon int `json:"dpasExpiringSoon"`
	OrganizationId string `json:"organizationId"`
	PendingDeletions int `json:"pendingDeletions"`
	ReportPeriodStart time.Time `json:"reportPeriodStart"`
	TotalUsers int `json:"totalUsers"`
}

type UpdateConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
}

type CookieConsent struct {
	Marketing bool `json:"marketing"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserAgent string `json:"userAgent"`
	UserId string `json:"userId"`
	ConsentBannerVersion string `json:"consentBannerVersion"`
	ExpiresAt time.Time `json:"expiresAt"`
	OrganizationId string `json:"organizationId"`
	CreatedAt time.Time `json:"createdAt"`
	Essential bool `json:"essential"`
	Functional bool `json:"functional"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	Analytics bool `json:"analytics"`
}

type PublishContentTypeRequest struct {
}

type UpdateRecoveryConfigRequest struct {
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
}

