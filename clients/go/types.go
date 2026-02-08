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

type PolicyResponse struct {
	ResourceType string `json:"resourceType"`
	Actions []string `json:"actions"`
	EnvironmentId string `json:"environmentId"`
	Name string `json:"name"`
	Description string `json:"description"`
	Id string `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserOrganizationId *string `json:"userOrganizationId"`
	Version int `json:"version"`
	AppId string `json:"appId"`
	CreatedBy string `json:"createdBy"`
	Enabled bool `json:"enabled"`
	Expression string `json:"expression"`
	NamespaceId string `json:"namespaceId"`
	Priority int `json:"priority"`
}

type UnbanUserRequestDTO struct {
	Reason string `json:"reason"`
}

type MFAPolicyResponse struct {
	AllowedFactorTypes []string `json:"allowedFactorTypes"`
	AppId xid.ID `json:"appId"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
	OrganizationId ID `json:"organizationId"`
	RequiredFactorCount int `json:"requiredFactorCount"`
}

type VerificationRequest struct {
	Data interface{} `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
}

type PreviewConversionResponse struct {
	ResourceType string `json:"resourceType"`
	Success bool `json:"success"`
	CelExpression string `json:"celExpression"`
	Error string `json:"error"`
	PolicyName string `json:"policyName"`
	ResourceId string `json:"resourceId"`
}

type VideoSessionInfo struct {
}

type GetDocumentVerificationResponse struct {
	Message string `json:"message"`
	RejectionReason string `json:"rejectionReason"`
	Status string `json:"status"`
	VerifiedAt Time `json:"verifiedAt"`
	ConfidenceScore float64 `json:"confidenceScore"`
	DocumentId xid.ID `json:"documentId"`
}

type ProvidersResponse struct {
	Providers []string `json:"providers"`
}

type DocumentCheckConfig struct {
	Enabled bool `json:"enabled"`
	ExtractData bool `json:"extractData"`
	ValidateDataConsistency bool `json:"validateDataConsistency"`
	ValidateExpiry bool `json:"validateExpiry"`
}

type BeginLoginResponse struct {
	Challenge string `json:"challenge"`
	Options interface{} `json:"options"`
	Timeout time.Duration `json:"timeout"`
}

type StepUpVerification struct {
	Verified_at time.Time `json:"verified_at"`
	Device_id string `json:"device_id"`
	Reason string `json:"reason"`
	Session_id string `json:"session_id"`
	Metadata interface{} `json:"metadata"`
	Org_id string `json:"org_id"`
	Expires_at time.Time `json:"expires_at"`
	Method VerificationMethod `json:"method"`
	Rule_name string `json:"rule_name"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Security_level SecurityLevel `json:"security_level"`
	User_agent string `json:"user_agent"`
}

type StartImpersonationResponse struct {
	Session_id string `json:"session_id"`
	Started_at string `json:"started_at"`
	Target_user_id string `json:"target_user_id"`
	Impersonator_id string `json:"impersonator_id"`
}

type TemplatePerformanceDTO struct {
	OpenRate float64 `json:"openRate"`
	TemplateId string `json:"templateId"`
	TemplateName string `json:"templateName"`
	TotalSent int64 `json:"totalSent"`
	ClickRate float64 `json:"clickRate"`
}

type TestCase struct {
	Expected bool `json:"expected"`
	Name string `json:"name"`
	Principal interface{} `json:"principal"`
	Request interface{} `json:"request"`
	Resource interface{} `json:"resource"`
	Action string `json:"action"`
}

type GetAPIKeyRequest struct {
}

type RunCheck_req struct {
	CheckType string `json:"checkType"`
}

type ConsentExportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type InviteMemberInput struct {
	Email string `json:"email"`
	OrgId string `json:"orgId"`
	Role string `json:"role"`
	AppId string `json:"appId"`
}

type LoginResponse struct {
	PasskeyUsed string `json:"passkeyUsed"`
	Session interface{} `json:"session"`
	Token string `json:"token"`
	User interface{} `json:"user"`
}

type VideoVerificationSession struct {
}

type MFAPolicy struct {
	AdaptiveMfaEnabled bool `json:"adaptiveMfaEnabled"`
	CreatedAt time.Time `json:"createdAt"`
	LockoutDurationMinutes int `json:"lockoutDurationMinutes"`
	MaxFailedAttempts int `json:"maxFailedAttempts"`
	OrganizationId xid.ID `json:"organizationId"`
	RequiredFactorCount int `json:"requiredFactorCount"`
	RequiredFactorTypes []FactorType `json:"requiredFactorTypes"`
	UpdatedAt time.Time `json:"updatedAt"`
	AllowedFactorTypes []FactorType `json:"allowedFactorTypes"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
	StepUpRequired bool `json:"stepUpRequired"`
	TrustedDeviceDays int `json:"trustedDeviceDays"`
}

type UpdateTemplateResponse struct {
	Template interface{} `json:"template"`
}

type TestSendTemplateInput struct {
	Recipient string `json:"recipient"`
	TemplateId string `json:"templateId"`
	Variables interface{} `json:"variables"`
}

type EvaluateResponse struct {
	Error string `json:"error"`
	EvaluatedPolicies int `json:"evaluatedPolicies"`
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
	MatchedPolicies []string `json:"matchedPolicies"`
	Reason string `json:"reason"`
	Allowed bool `json:"allowed"`
	CacheHit bool `json:"cacheHit"`
}

type ConsentAuditConfig struct {
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	Enabled bool `json:"enabled"`
	ExportFormat string `json:"exportFormat"`
	Immutable bool `json:"immutable"`
	LogIpAddress bool `json:"logIpAddress"`
	LogUserAgent bool `json:"logUserAgent"`
	SignLogs bool `json:"signLogs"`
	LogAllChanges bool `json:"logAllChanges"`
	RetentionDays int `json:"retentionDays"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
}

type ScopeInfo struct {
}

type StartImpersonationRequest struct {
	Reason string `json:"reason"`
	Target_user_id string `json:"target_user_id"`
	Ticket_number string `json:"ticket_number"`
	Duration_minutes int `json:"duration_minutes"`
}

type GetAnalyticsResult struct {
	Analytics AnalyticsDTO `json:"analytics"`
}

type GetEffectivePermissionsRequest struct {
}

type ComplianceUserTrainingResponse struct {
	User_id string `json:"user_id"`
}

type CreatePolicy_req struct {
	Content string `json:"content"`
	PolicyType string `json:"policyType"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	Version string `json:"version"`
}

type OIDCState struct {
}

type ResourceTypeStats struct {
	AllowRate float64 `json:"allowRate"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	EvaluationCount int64 `json:"evaluationCount"`
	ResourceType string `json:"resourceType"`
}

type ApproveRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
	Notes string `json:"notes"`
}

type ListUsersRequestDTO struct {
}

type Config struct {
	App_name string `json:"app_name"`
	Async AsyncConfig `json:"async"`
	Auto_populate_templates bool `json:"auto_populate_templates"`
	Auto_send AutoSendConfig `json:"auto_send"`
	Default_language string `json:"default_language"`
	Providers ProvidersConfig `json:"providers"`
	Rate_limits interface{} `json:"rate_limits"`
	Allow_app_overrides bool `json:"allow_app_overrides"`
	Allow_template_reset bool `json:"allow_template_reset"`
	Auto_send_welcome bool `json:"auto_send_welcome"`
	Cleanup_after time.Duration `json:"cleanup_after"`
	Retry_attempts int `json:"retry_attempts"`
	Retry_delay time.Duration `json:"retry_delay"`
	Add_default_templates bool `json:"add_default_templates"`
}

type ValidatePolicyRequest struct {
	ResourceType string `json:"resourceType"`
	Expression string `json:"expression"`
}

type ConsentReportResponse struct {
	Id string `json:"id"`
}

type IDVerificationSessionResponse struct {
	Session interface{} `json:"session"`
}

type CreateResourceRequest struct {
	Attributes []ResourceAttributeRequest `json:"attributes"`
	Description string `json:"description"`
	NamespaceId string `json:"namespaceId"`
	Type string `json:"type"`
}

type RemoveTeamMemberRequest struct {
}

type DeviceAuthorizationRequest struct {
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
}

type StartVideoSessionRequest struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type UpdateNamespaceRequest struct {
	Name string `json:"name"`
	Description string `json:"description"`
	InheritPlatform *bool `json:"inheritPlatform"`
}

type ReviewDocumentRequest struct {
	Approved bool `json:"approved"`
	DocumentId xid.ID `json:"documentId"`
	Notes string `json:"notes"`
	RejectionReason string `json:"rejectionReason"`
}

type CreateUserRequestDTO struct {
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Password string `json:"password"`
	Role string `json:"role"`
	Username string `json:"username"`
}

type AuditConfig struct {
	Immutable bool `json:"immutable"`
	MaxRetentionDays int `json:"maxRetentionDays"`
	MinRetentionDays int `json:"minRetentionDays"`
	SignLogs bool `json:"signLogs"`
	DetailedTrail bool `json:"detailedTrail"`
	ExportFormat string `json:"exportFormat"`
}

type Email struct {
}

type DeleteTeamResult struct {
	Success bool `json:"success"`
}

type GetTemplateDefaultsResponse struct {
	Templates []*interface{} `json:"templates"`
	Total int `json:"total"`
}

type RequestDataDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type ListNotificationsHistoryInput struct {
	Page int `json:"page"`
	Recipient *string `json:"recipient"`
	Status *string `json:"status"`
	Type *string `json:"type"`
	Limit int `json:"limit"`
}

type AppServiceAdapter struct {
}

type RateLimitRule struct {
	Max int `json:"max"`
	Window time.Duration `json:"window"`
}

type ConsentRequest struct {
	State string `json:"state"`
	Action string `json:"action"`
	Client_id string `json:"client_id"`
	Code_challenge string `json:"code_challenge"`
	Code_challenge_method string `json:"code_challenge_method"`
	Redirect_uri string `json:"redirect_uri"`
	Response_type string `json:"response_type"`
	Scope string `json:"scope"`
}

type DeviceVerificationInfo struct {
}

type CompleteRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type GenerateRecoveryCodesRequest struct {
	Count int `json:"count"`
	Format string `json:"format"`
}

type CodesResponse struct {
	Codes []string `json:"codes"`
}

type AutoCleanupConfig struct {
	Interval time.Duration `json:"interval"`
	Enabled bool `json:"enabled"`
}

type TokenRevocationRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type GetRoleTemplateInput struct {
	AppId string `json:"appId"`
	TemplateId string `json:"templateId"`
}

type CreateConsentResponse struct {
	Id string `json:"id"`
}

type BunRepository struct {
}

type RejectRecoveryRequest struct {
	Notes string `json:"notes"`
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type BanUserRequest struct {
	Expires_at Time `json:"expires_at"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
}

type StepUpRequirementResponse struct {
	Id string `json:"id"`
}

type UpdateTeamInput struct {
	OrgId string `json:"orgId"`
	TeamId string `json:"teamId"`
	AppId string `json:"appId"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
}

type ProviderDiscoveredResponse struct {
	Found bool `json:"found"`
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
}

type UpdatePrivacySettingsResponse struct {
	Settings interface{} `json:"settings"`
}

type SessionStatsResponse struct {
	LocationCount int `json:"locationCount"`
	NewestSession *string `json:"newestSession"`
	OldestSession *string `json:"oldestSession"`
	TotalSessions int `json:"totalSessions"`
	ActiveSessions int `json:"activeSessions"`
	DeviceCount int `json:"deviceCount"`
}

type CompliancePoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type AddTeamMemberRequest struct {
	Member_id string `json:"member_id"`
}

type AddFieldRequest struct {
}

type GetPrivacySettingsResponse struct {
	Settings interface{} `json:"settings"`
}

type ResolveViolationRequest struct {
	Notes string `json:"notes"`
	Resolution string `json:"resolution"`
}

type MockStateStore struct {
}

type GetSecretRequest struct {
}

type AccessConfig struct {
	AllowApiAccess bool `json:"allowApiAccess"`
	AllowDashboardAccess bool `json:"allowDashboardAccess"`
	RateLimitPerMinute int `json:"rateLimitPerMinute"`
	RequireAuthentication bool `json:"requireAuthentication"`
	RequireRbac bool `json:"requireRbac"`
}

type bunRepository struct {
}

type MockOrganizationUIExtension struct {
}

type ProviderInfo struct {
	CreatedAt string `json:"createdAt"`
	Domain string `json:"domain"`
	ProviderId string `json:"providerId"`
	Type string `json:"type"`
}

type LinkAccountResponse struct {
	Url string `json:"url"`
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

type GetOverviewStatsInput struct {
	Days *int `json:"days"`
	EndDate *string `json:"endDate"`
	StartDate *string `json:"startDate"`
}

type RestoreTemplateVersionRequest struct {
}

type mockSessionService struct {
}

type GetStatusRequest struct {
	Device_id string `json:"device_id"`
	User_id string `json:"user_id"`
}

type ConfigSourceConfig struct {
	Priority int `json:"priority"`
	RefreshInterval time.Duration `json:"refreshInterval"`
	AutoRefresh bool `json:"autoRefresh"`
	Enabled bool `json:"enabled"`
	Prefix string `json:"prefix"`
}

type DeleteOrganizationRequest struct {
}

type DeleteTemplateResult struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type VerificationListResponse struct {
	Offset int `json:"offset"`
	Total int `json:"total"`
	Verifications IdentityVerification `json:"verifications"`
	Limit int `json:"limit"`
}

type RunCheckRequest struct {
	CheckType string `json:"checkType"`
}

type GetEntryStatsRequest struct {
}

type DiscoveryService struct {
}

type CallbackResult struct {
}

type GenerateTokenRequest struct {
	SessionId string `json:"sessionId"`
	TokenType string `json:"tokenType"`
	UserId string `json:"userId"`
	Audience []string `json:"audience"`
	ExpiresIn time.Duration `json:"expiresIn"`
	Metadata interface{} `json:"metadata"`
	Permissions []string `json:"permissions"`
	Scopes []string `json:"scopes"`
}

// User represents User account
type User struct {
	Email string `json:"email"`
	Name *string `json:"name,omitempty"`
	EmailVerified bool `json:"emailVerified"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	OrganizationId *string `json:"organizationId,omitempty"`
	Id string `json:"id"`
}

type SMSFactorAdapter struct {
}

type DeleteSecretRequest struct {
}

type ListAPIKeysRequest struct {
}

type DataDeletionRequestInput struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type ListFactorsRequest struct {
}

type userServiceAdapter struct {
}

type EncryptionConfig struct {
	MasterKey string `json:"masterKey"`
	RotateKeyAfter time.Duration `json:"rotateKeyAfter"`
	TestOnStartup bool `json:"testOnStartup"`
}

type UpdateOrganizationResult struct {
	Organization OrganizationDetailDTO `json:"organization"`
}

type ListTemplatesInput struct {
	Active *bool `json:"active"`
	Language *string `json:"language"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	Type *string `json:"type"`
}

type AuthAutoSendDTO struct {
	EmailOtp bool `json:"emailOtp"`
	MagicLink bool `json:"magicLink"`
	MfaCode bool `json:"mfaCode"`
	PasswordReset bool `json:"passwordReset"`
	VerificationEmail bool `json:"verificationEmail"`
	Welcome bool `json:"welcome"`
}

type RemoveMemberRequest struct {
}

type GetFactorRequest struct {
}

type DeleteOrganizationResult struct {
	Success bool `json:"success"`
}

type BridgeAppInput struct {
	AppId string `json:"appId"`
}

type ChallengeStatusResponse struct {
	SessionId xid.ID `json:"sessionId"`
	Status string `json:"status"`
	CompletedAt Time `json:"completedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRemaining int `json:"factorsRemaining"`
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
}

type EncryptionService struct {
}

type AdminUpdateProviderRequest struct {
	ClientId *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
	Enabled *bool `json:"enabled"`
	Scopes []string `json:"scopes"`
}

type TrustedDevice struct {
	IpAddress string `json:"ipAddress"`
	LastUsedAt Time `json:"lastUsedAt"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	UserId xid.ID `json:"userId"`
	DeviceId string `json:"deviceId"`
	ExpiresAt time.Time `json:"expiresAt"`
	UserAgent string `json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
}

type DashboardExtension struct {
}

type WebhookConfig struct {
	Enabled bool `json:"enabled"`
	Expiry_warning_days int `json:"expiry_warning_days"`
	Notify_on_created bool `json:"notify_on_created"`
	Notify_on_deleted bool `json:"notify_on_deleted"`
	Notify_on_expiring bool `json:"notify_on_expiring"`
	Notify_on_rate_limit bool `json:"notify_on_rate_limit"`
	Notify_on_rotated bool `json:"notify_on_rotated"`
	Webhook_urls []string `json:"webhook_urls"`
}

type CompliancePolicyResponse struct {
	Id string `json:"id"`
}

type PolicyEngine struct {
}

type SecurityQuestionsConfig struct {
	AllowCustomQuestions bool `json:"allowCustomQuestions"`
	CaseSensitive bool `json:"caseSensitive"`
	Enabled bool `json:"enabled"`
	ForbidCommonAnswers bool `json:"forbidCommonAnswers"`
	LockoutDuration time.Duration `json:"lockoutDuration"`
	MinimumQuestions int `json:"minimumQuestions"`
	RequiredToRecover int `json:"requiredToRecover"`
	MaxAnswerLength int `json:"maxAnswerLength"`
	MaxAttempts int `json:"maxAttempts"`
	PredefinedQuestions []string `json:"predefinedQuestions"`
	RequireMinLength int `json:"requireMinLength"`
}

type ConsentDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type SAMLCallbackResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type ReorderFieldsRequest struct {
}

type KeyStore struct {
}

type StepUpRequirementsResponse struct {
	Requirements []*interface{} `json:"requirements"`
}

type GetTeamsResult struct {
	CanManage bool `json:"canManage"`
	Data []TeamDTO `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
}

type UpdateMemberHandlerRequest struct {
}

type RequestPasswordResetResponse struct {
	Message string `json:"message"`
}

type PreviewTemplateResult struct {
	Body string `json:"body"`
	RenderedAt string `json:"renderedAt"`
	Subject string `json:"subject"`
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type DeviceCodeEntryResponse struct {
	BasePath string `json:"basePath"`
	FormAction string `json:"formAction"`
	Placeholder string `json:"placeholder"`
}

type RecoveryConfiguration struct {
}

type CompleteVideoSessionResponse struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	Result string `json:"result"`
}

type EvaluationResult struct {
	Matched_rules []string `json:"matched_rules"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
	Required bool `json:"required"`
	Requirement_id string `json:"requirement_id"`
	Security_level SecurityLevel `json:"security_level"`
	Can_remember bool `json:"can_remember"`
	Challenge_token string `json:"challenge_token"`
	Expires_at time.Time `json:"expires_at"`
	Grace_period_ends_at time.Time `json:"grace_period_ends_at"`
	Allowed_methods []VerificationMethod `json:"allowed_methods"`
	Current_level SecurityLevel `json:"current_level"`
}

type GetUserTrainingResponse struct {
	User_id string `json:"user_id"`
}

type EnrollFactorResponse struct {
	Type FactorType `json:"type"`
	FactorId xid.ID `json:"factorId"`
	ProvisioningData interface{} `json:"provisioningData"`
	Status FactorStatus `json:"status"`
}

type GetUserVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type ComplianceTemplatesResponse struct {
	Templates []*interface{} `json:"templates"`
}

type ImpersonationStartResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Session_id string `json:"session_id"`
	Started_at string `json:"started_at"`
	Target_user_id string `json:"target_user_id"`
}

type GetNotificationDetailInput struct {
	NotificationId string `json:"notificationId"`
}

type DataExportRequest struct {
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt Time `json:"expiresAt"`
	ExportUrl string `json:"exportUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
	CompletedAt Time `json:"completedAt"`
	ExportPath string `json:"exportPath"`
	Format string `json:"format"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	ErrorMessage string `json:"errorMessage"`
	ExportSize int64 `json:"exportSize"`
	OrganizationId string `json:"organizationId"`
	UserId string `json:"userId"`
	IncludeSections []string `json:"includeSections"`
	Status string `json:"status"`
}

type AmountRule struct {
	Currency string `json:"currency"`
	Description string `json:"description"`
	Max_amount float64 `json:"max_amount"`
	Min_amount float64 `json:"min_amount"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type SAMLLoginResponse struct {
	ProviderId string `json:"providerId"`
	RedirectUrl string `json:"redirectUrl"`
	RequestId string `json:"requestId"`
}

type DeletePolicyResponse struct {
	Status string `json:"status"`
}

type SMSProviderDTO struct {
	Config interface{} `json:"config"`
	Enabled bool `json:"enabled"`
	Type string `json:"type"`
}

type FinishLoginRequest struct {
	Remember bool `json:"remember"`
	Response interface{} `json:"response"`
}

type RiskAssessmentConfig struct {
	BlockHighRisk bool `json:"blockHighRisk"`
	Enabled bool `json:"enabled"`
	HighRiskThreshold float64 `json:"highRiskThreshold"`
	HistoryWeight float64 `json:"historyWeight"`
	LowRiskThreshold float64 `json:"lowRiskThreshold"`
	NewDeviceWeight float64 `json:"newDeviceWeight"`
	NewLocationWeight float64 `json:"newLocationWeight"`
	MediumRiskThreshold float64 `json:"mediumRiskThreshold"`
	NewIpWeight float64 `json:"newIpWeight"`
	RequireReviewAbove float64 `json:"requireReviewAbove"`
	VelocityWeight float64 `json:"velocityWeight"`
}

type EnableRequest2FA struct {
	Method string `json:"method"`
	User_id string `json:"user_id"`
}

type AdminPolicyRequest struct {
	RequiredFactors int `json:"requiredFactors"`
	AllowedTypes []string `json:"allowedTypes"`
	Enabled bool `json:"enabled"`
	GracePeriod int `json:"gracePeriod"`
}

type CreateSecretRequest struct {
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Path string `json:"path"`
	Tags []string `json:"tags"`
	Value interface{} `json:"value"`
	ValueType string `json:"valueType"`
}

type VerifyImpersonationResponse struct {
	Target_user_id string `json:"target_user_id"`
	Impersonator_id string `json:"impersonator_id"`
	Is_impersonating bool `json:"is_impersonating"`
}

type DeleteContentTypeRequest struct {
}

type RotateAPIKeyResponse struct {
	Message string `json:"message"`
	Api_key APIKey `json:"api_key"`
}

type MockService struct {
}

type CreateVerificationRequest struct {
}

type ComplianceChecksResponse struct {
	Checks []*interface{} `json:"checks"`
}

type ContentTypeHandler struct {
}

type BackupAuthRecoveryResponse struct {
	Session_id string `json:"session_id"`
}

type ConsentRecordResponse struct {
	Id string `json:"id"`
}

type CookieConsentConfig struct {
	BannerVersion string `json:"bannerVersion"`
	Categories []string `json:"categories"`
	DefaultStyle string `json:"defaultStyle"`
	Enabled bool `json:"enabled"`
	RequireExplicit bool `json:"requireExplicit"`
	ValidityPeriod time.Duration `json:"validityPeriod"`
	AllowAnonymous bool `json:"allowAnonymous"`
}

type GenerateBackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type DeleteProviderRequest struct {
}

type UpdateTemplateResult struct {
	Message string `json:"message"`
	Success bool `json:"success"`
	Template TemplateDTO `json:"template"`
}

type NoOpEmailProvider struct {
}

type OnfidoConfig struct {
	IncludeDocumentReport bool `json:"includeDocumentReport"`
	IncludeWatchlistReport bool `json:"includeWatchlistReport"`
	Region string `json:"region"`
	WorkflowId string `json:"workflowId"`
	DocumentCheck DocumentCheckConfig `json:"documentCheck"`
	Enabled bool `json:"enabled"`
	IncludeFacialReport bool `json:"includeFacialReport"`
	WebhookToken string `json:"webhookToken"`
	ApiToken string `json:"apiToken"`
	FacialCheck FacialCheckConfig `json:"facialCheck"`
}

type AnalyticsSummary struct {
	ActivePolicies int `json:"activePolicies"`
	AllowedCount int64 `json:"allowedCount"`
	CacheHitRate float64 `json:"cacheHitRate"`
	DeniedCount int64 `json:"deniedCount"`
	TopPolicies []PolicyStats `json:"topPolicies"`
	TopResourceTypes []ResourceTypeStats `json:"topResourceTypes"`
	TotalEvaluations int64 `json:"totalEvaluations"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	TotalPolicies int `json:"totalPolicies"`
}

type DeviceAuthorizationResponse struct {
	Verification_uri_complete string `json:"verification_uri_complete"`
	Device_code string `json:"device_code"`
	Expires_in int `json:"expires_in"`
	Interval int `json:"interval"`
	User_code string `json:"user_code"`
	Verification_uri string `json:"verification_uri"`
}

type GenerateReportResponse struct {
	Id string `json:"id"`
}

type ListRememberedDevicesResponse struct {
	Count int `json:"count"`
	Devices interface{} `json:"devices"`
}

type ComplianceTemplateResponse struct {
	Standard string `json:"standard"`
}

type ConsentsResponse struct {
	Consents interface{} `json:"consents"`
	Count int `json:"count"`
}

type ConsentDashboardConfig struct {
	ShowDataDeletion bool `json:"showDataDeletion"`
	ShowDataExport bool `json:"showDataExport"`
	ShowPolicies bool `json:"showPolicies"`
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
	ShowAuditLog bool `json:"showAuditLog"`
	ShowConsentHistory bool `json:"showConsentHistory"`
	ShowCookiePreferences bool `json:"showCookiePreferences"`
}

type VersioningConfig struct {
	MaxVersions int `json:"maxVersions"`
	RetentionDays int `json:"retentionDays"`
	AutoCleanup bool `json:"autoCleanup"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
}

type GetVersionsRequest struct {
}

type RequestReverificationResponse struct {
	Session interface{} `json:"session"`
}

type GetTemplateInput struct {
	TemplateId string `json:"templateId"`
}

type GetTemplateRequest struct {
}

type GetUserSessionsResult struct {
	UserId string `json:"userId"`
	ActiveCount int `json:"activeCount"`
	Pagination PaginationInfoDTO `json:"pagination"`
	Sessions []SessionDTO `json:"sessions"`
	TotalCount int `json:"totalCount"`
}

type CreateSessionRequest struct {
}

type TemplatesResponse struct {
	Count int `json:"count"`
	Templates interface{} `json:"templates"`
}

type CompleteTraining_req struct {
	Score int `json:"score"`
}

type AccountLockoutError struct {
}

type SendWithTemplateRequest struct {
	Language string `json:"language"`
	Metadata interface{} `json:"metadata"`
	Recipient string `json:"recipient"`
	TemplateKey string `json:"templateKey"`
	Type NotificationType `json:"type"`
	Variables interface{} `json:"variables"`
	AppId xid.ID `json:"appId"`
}

type SendVerificationCodeResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	MaskedTarget string `json:"maskedTarget"`
	Message string `json:"message"`
	Sent bool `json:"sent"`
}

type FactorEnrollmentRequest struct {
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	Type FactorType `json:"type"`
}

type PasskeyInfo struct {
	AuthenticatorType string `json:"authenticatorType"`
	CreatedAt time.Time `json:"createdAt"`
	CredentialId string `json:"credentialId"`
	IsResidentKey bool `json:"isResidentKey"`
	LastUsedAt Time `json:"lastUsedAt"`
	Name string `json:"name"`
	Aaguid string `json:"aaguid"`
	Id string `json:"id"`
	SignCount uint `json:"signCount"`
}

type RevokeTokenService struct {
}

type ResetUserMFARequest struct {
	Reason string `json:"reason"`
}

type mockNotificationProvider struct {
}

type DownloadReportResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type ProvidersConfig struct {
	Sms *SMSProviderConfig `json:"sms"`
	Email EmailProviderConfig `json:"email"`
}

type ComplianceEvidence struct {
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	AppId string `json:"appId"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileHash string `json:"fileHash"`
	Id string `json:"id"`
	Title string `json:"title"`
	CollectedBy string `json:"collectedBy"`
	ControlId string `json:"controlId"`
	CreatedAt time.Time `json:"createdAt"`
	FileUrl string `json:"fileUrl"`
	Metadata interface{} `json:"metadata"`
}

type ComplianceEvidenceResponse struct {
	Id string `json:"id"`
}

type TwoFAEnableResponse struct {
	Status string `json:"status"`
	Totp_uri string `json:"totp_uri"`
}

type TemplateEngine struct {
}

type CompareRevisionsRequest struct {
}

type SecretsConfigSource struct {
}

type PreviewTemplateRequest struct {
}

type RejectRecoveryResponse struct {
	SessionId xid.ID `json:"sessionId"`
	Message string `json:"message"`
	Reason string `json:"reason"`
	Rejected bool `json:"rejected"`
	RejectedAt time.Time `json:"rejectedAt"`
}

type ListUsersResponse struct {
	Limit int `json:"limit"`
	Page int `json:"page"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
	Users User `json:"users"`
}

type RiskContext struct {
}

type GetSessionResponse struct {
	Session Session `json:"session"`
	User User `json:"user"`
}

type ChallengeSession struct {
}

type GetEntryRequest struct {
}

type BackupAuthSessionsResponse struct {
	Sessions []*interface{} `json:"sessions"`
}

type ListJWTKeysRequest struct {
}

type DeviceAuthorizeRequest struct {
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
}

type RecordCookieConsentRequest struct {
	BannerVersion string `json:"bannerVersion"`
	Essential bool `json:"essential"`
	Functional bool `json:"functional"`
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	Analytics bool `json:"analytics"`
}

type UpdateUserResponse struct {
	User User `json:"user"`
}

type RegisterProviderResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Type string `json:"type"`
}

type Adapter struct {
}

type ComplianceCheckResponse struct {
	Id string `json:"id"`
}

type BulkUnpublishRequest struct {
	Ids []string `json:"ids"`
}

type GetSecurityQuestionsResponse struct {
	Questions []SecurityQuestionInfo `json:"questions"`
}

type GetAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type GetSettingsResult struct {
	Settings NotificationSettingsDTO `json:"settings"`
}

type MigrationErrorResponse struct {
	Error string `json:"error"`
	PolicyIndex int `json:"policyIndex"`
	Resource string `json:"resource"`
	Subject string `json:"subject"`
}

type BatchEvaluateRequest struct {
	Requests []EvaluateRequest `json:"requests"`
}

type ListMembersRequest struct {
}

type EndImpersonationResponse struct {
	Ended_at string `json:"ended_at"`
	Status string `json:"status"`
}

type NotificationPreviewResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

type ListReportsFilter struct {
	ProfileId *string `json:"profileId"`
	ReportType *string `json:"reportType"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
	Format *string `json:"format"`
}

type DeclineInvitationRequest struct {
}

type StartRecoveryRequest struct {
	UserId string `json:"userId"`
	DeviceId string `json:"deviceId"`
	Email string `json:"email"`
	PreferredMethod RecoveryMethod `json:"preferredMethod"`
}

type ResendNotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type HandleWebhookResponse struct {
	Status string `json:"status"`
}

type GetCheckResponse struct {
	Id string `json:"id"`
}

// SuccessResponse represents Success boolean response
type SuccessResponse struct {
	Success bool `json:"success"`
}

type PolicyPreviewResponse struct {
	Actions []string `json:"actions"`
	Description string `json:"description"`
	Expression string `json:"expression"`
	Name string `json:"name"`
	ResourceType string `json:"resourceType"`
}

type ComplianceStatusDetailsResponse struct {
	Status string `json:"status"`
}

type ListChecksFilter struct {
	SinceBefore Time `json:"sinceBefore"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
	CheckType *string `json:"checkType"`
	ProfileId *string `json:"profileId"`
}

type navItem struct {
}

type GetSecretInput struct {
	AppId string `json:"appId"`
	SecretId string `json:"secretId"`
}

type WebhookPayload struct {
}

type IntrospectTokenRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type RevokeAllRequest struct {
	IncludeCurrentSession bool `json:"includeCurrentSession"`
}

type AnalyticsResponse struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Summary AnalyticsSummary `json:"summary"`
	TimeRange interface{} `json:"timeRange"`
}

type NotificationChannels struct {
	Email bool `json:"email"`
	Slack bool `json:"slack"`
	Webhook bool `json:"webhook"`
}

type CloneContentTypeRequest struct {
}

type BaseFactorAdapter struct {
}

type GetRequest struct {
}

type AdminGetUserVerificationStatusResponse struct {
	Status interface{} `json:"status"`
}

type ComplianceStatus struct {
	LastChecked time.Time `json:"lastChecked"`
	Violations int `json:"violations"`
	ChecksFailed int `json:"checksFailed"`
	NextAudit time.Time `json:"nextAudit"`
	OverallStatus string `json:"overallStatus"`
	ProfileId string `json:"profileId"`
	Score int `json:"score"`
	Standard ComplianceStandard `json:"standard"`
	AppId string `json:"appId"`
	ChecksPassed int `json:"checksPassed"`
	ChecksWarning int `json:"checksWarning"`
}

type SecurityQuestion struct {
}

type TwoFAStatusDetailResponse struct {
	Enabled bool `json:"enabled"`
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
}

type CallbackDataResponse struct {
	Action string `json:"action"`
	IsNewUser bool `json:"isNewUser"`
	User User `json:"user"`
}

type PrivacySettingsRequest struct {
	DataRetentionDays *int `json:"dataRetentionDays"`
	GdprMode *bool `json:"gdprMode"`
	CcpaMode *bool `json:"ccpaMode"`
	ContactEmail string `json:"contactEmail"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	ExportFormat []string `json:"exportFormat"`
	ConsentRequired *bool `json:"consentRequired"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	DpoEmail string `json:"dpoEmail"`
	AllowDataPortability *bool `json:"allowDataPortability"`
	ContactPhone string `json:"contactPhone"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
}

type TabDataDTO struct {
	Order int `json:"order"`
	Path string `json:"path"`
	RequireAdmin bool `json:"requireAdmin"`
	Icon string `json:"icon"`
	Id string `json:"id"`
	Label string `json:"label"`
}

type SendCodeResponse struct {
	Dev_code string `json:"dev_code"`
	Status string `json:"status"`
}

type GetAppProfileResponse struct {
	Id string `json:"id"`
}

type SaveBuilderTemplateInput struct {
	BuilderJson string `json:"builderJson"`
	Name string `json:"name"`
	Subject string `json:"subject"`
	TemplateId string `json:"templateId"`
	TemplateKey string `json:"templateKey"`
}

type TimeBasedRule struct {
	Operation string `json:"operation"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Description string `json:"description"`
	Max_age time.Duration `json:"max_age"`
}

type TOTPFactorAdapter struct {
}

type NotificationsResponse struct {
	Count int `json:"count"`
	Notifications interface{} `json:"notifications"`
}

type BatchEvaluateResponse struct {
	FailureCount int `json:"failureCount"`
	Results []*BatchEvaluationResult `json:"results"`
	SuccessCount int `json:"successCount"`
	TotalEvaluations int `json:"totalEvaluations"`
	TotalTimeMs float64 `json:"totalTimeMs"`
}

type ListEvidenceFilter struct {
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	AppId *string `json:"appId"`
	ControlId *string `json:"controlId"`
	EvidenceType *string `json:"evidenceType"`
}

type ListViolationsFilter struct {
	Status *string `json:"status"`
	UserId *string `json:"userId"`
	ViolationType *string `json:"violationType"`
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Severity *string `json:"severity"`
}

type WidgetDataDTO struct {
	Content string `json:"content"`
	Icon string `json:"icon"`
	Id string `json:"id"`
	Order int `json:"order"`
	RequireAdmin bool `json:"requireAdmin"`
	Size int `json:"size"`
	Title string `json:"title"`
}

type StripeIdentityConfig struct {
	WebhookSecret string `json:"webhookSecret"`
	AllowedTypes []string `json:"allowedTypes"`
	ApiKey string `json:"apiKey"`
	Enabled bool `json:"enabled"`
	RequireLiveCapture bool `json:"requireLiveCapture"`
	RequireMatchingSelfie bool `json:"requireMatchingSelfie"`
	ReturnUrl string `json:"returnUrl"`
	UseMock bool `json:"useMock"`
}

type ImpersonationErrorResponse struct {
	Error string `json:"error"`
}

type ResetUserMFAResponse struct {
	FactorsReset int `json:"factorsReset"`
	Message string `json:"message"`
	Success bool `json:"success"`
	DevicesRevoked int `json:"devicesRevoked"`
}

type FactorVerificationRequest struct {
	Code string `json:"code"`
	Data interface{} `json:"data"`
	FactorId xid.ID `json:"factorId"`
}

type GetOrganizationResult struct {
	Organization OrganizationDetailDTO `json:"organization"`
	Stats OrgDetailStatsDTO `json:"stats"`
	UserRole string `json:"userRole"`
}

type InvitationResponse struct {
	Invitation Invitation `json:"invitation"`
	Message string `json:"message"`
}

type CreateResponse struct {
	Webhook Webhook `json:"webhook"`
}

type RevokeConsentRequest struct {
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
	Granted *bool `json:"granted"`
}

type ComplianceViolationResponse struct {
	Id string `json:"id"`
}

type SetupSecurityQuestionsResponse struct {
	Count int `json:"count"`
	Message string `json:"message"`
	SetupAt time.Time `json:"setupAt"`
}

type VerifySecurityAnswersResponse struct {
	Valid bool `json:"valid"`
	AttemptsLeft int `json:"attemptsLeft"`
	CorrectAnswers int `json:"correctAnswers"`
	Message string `json:"message"`
	RequiredAnswers int `json:"requiredAnswers"`
}

type TeamsResponse struct {
	Total int `json:"total"`
	Teams Team `json:"teams"`
}

type MetadataResponse struct {
	Metadata string `json:"metadata"`
}

type RevokeTokenRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type ReviewDocumentResponse struct {
	Status string `json:"status"`
}

type GetByPathResponse struct {
	Message string `json:"message"`
	Code string `json:"code"`
	Error string `json:"error"`
}

type PaginationInfoDTO struct {
	PageSize int `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
}

type DeleteTeamRequest struct {
}

type auditServiceAdapter struct {
}

type mockProvider struct {
}

type GetUserVerificationStatusResponse struct {
	Status interface{} `json:"status"`
}

type GetTemplateAnalyticsRequest struct {
}

type GetSessionsResult struct {
	Pagination PaginationInfoDTO `json:"pagination"`
	Sessions []SessionDTO `json:"sessions"`
	Stats SessionStatsDTO `json:"stats"`
}

type DeleteAPIKeyRequest struct {
}

type CreateProfileRequest struct {
	DataResidency string `json:"dataResidency"`
	Name string `json:"name"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	RegularAccessReview bool `json:"regularAccessReview"`
	RetentionDays int `json:"retentionDays"`
	DpoContact string `json:"dpoContact"`
	Metadata interface{} `json:"metadata"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	SessionMaxAge int `json:"sessionMaxAge"`
	Standards []ComplianceStandard `json:"standards"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	PasswordMinLength int `json:"passwordMinLength"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	LeastPrivilege bool `json:"leastPrivilege"`
	MfaRequired bool `json:"mfaRequired"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	RbacRequired bool `json:"rbacRequired"`
	AppId string `json:"appId"`
	AuditLogExport bool `json:"auditLogExport"`
	ComplianceContact string `json:"complianceContact"`
}

type AddMemberRequest struct {
	Role string `json:"role"`
	User_id string `json:"user_id"`
}

type OAuthErrorResponse struct {
	Error string `json:"error"`
	Error_description string `json:"error_description"`
	Error_uri string `json:"error_uri"`
	State string `json:"state"`
}

type PublishEntryRequest struct {
}

type NoOpNotificationProvider struct {
}

type AddTrustedContactRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
}

type GetOrganizationsInput struct {
	AppId string `json:"appId"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	Search string `json:"search"`
}

type CreateTrainingResponse struct {
	Id string `json:"id"`
}

type SAMLSPMetadataResponse struct {
	Metadata string `json:"metadata"`
}

type AdminHandler struct {
}

type DeviceDecisionResponse struct {
	Approved bool `json:"approved"`
	Message string `json:"message"`
	Success bool `json:"success"`
}

type SetUserRoleRequest struct {
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Role string `json:"role"`
	User_id xid.ID `json:"user_id"`
}

type SignInResponse struct {
	Token string `json:"token"`
	User interface{} `json:"user"`
	Session interface{} `json:"session"`
}

type BackupAuthContactsResponse struct {
	Contacts []*interface{} `json:"contacts"`
}

type StepUpPolicyResponse struct {
	Id string `json:"id"`
}

type GetRequirementResponse struct {
	Id string `json:"id"`
}

type AnalyticsDTO struct {
	TopTemplates []TemplatePerformanceDTO `json:"topTemplates"`
	ByDay []DailyAnalyticsDTO `json:"byDay"`
	ByTemplate []TemplateAnalyticsDTO `json:"byTemplate"`
	Overview OverviewStatsDTO `json:"overview"`
}

type PhoneVerifyResponse struct {
	Token string `json:"token"`
	User User `json:"user"`
	Session Session `json:"session"`
}

type ImpersonationEndResponse struct {
	Ended_at string `json:"ended_at"`
	Status string `json:"status"`
}

type StateStorageConfig struct {
	RedisAddr string `json:"redisAddr"`
	RedisDb int `json:"redisDb"`
	RedisPassword string `json:"redisPassword"`
	StateTtl time.Duration `json:"stateTtl"`
	UseRedis bool `json:"useRedis"`
}

type FactorsResponse struct {
	Count int `json:"count"`
	Factors interface{} `json:"factors"`
}

type ReverifyRequest struct {
	Reason string `json:"reason"`
}

type CreateTemplateResponse struct {
	Template interface{} `json:"template"`
}

type ResetAllTemplatesResponse struct {
	Status string `json:"status"`
}

type CreateAPIKeyRequest struct {
	Allowed_ips []string `json:"allowed_ips"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Permissions interface{} `json:"permissions"`
	Rate_limit int `json:"rate_limit"`
	Scopes []string `json:"scopes"`
}

type CreateActionRequest struct {
	Description string `json:"description"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
}

type TestPolicyResponse struct {
	Results []TestCaseResult `json:"results"`
	Total int `json:"total"`
	Error string `json:"error"`
	FailedCount int `json:"failedCount"`
	Passed bool `json:"passed"`
	PassedCount int `json:"passedCount"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

type GetAppRequest struct {
}

type MFAStatus struct {
	RequiredCount int `json:"requiredCount"`
	TrustedDevice bool `json:"trustedDevice"`
	Enabled bool `json:"enabled"`
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
	GracePeriod Time `json:"gracePeriod"`
	PolicyActive bool `json:"policyActive"`
}

type GetTemplateVersionRequest struct {
}

type SessionAutoSendDTO struct {
	AllRevoked bool `json:"allRevoked"`
	DeviceRemoved bool `json:"deviceRemoved"`
	NewDevice bool `json:"newDevice"`
	NewLocation bool `json:"newLocation"`
	SuspiciousLogin bool `json:"suspiciousLogin"`
}

type MultiStepRecoveryConfig struct {
	AllowStepSkip bool `json:"allowStepSkip"`
	HighRiskSteps []RecoveryMethod `json:"highRiskSteps"`
	LowRiskSteps []RecoveryMethod `json:"lowRiskSteps"`
	MediumRiskSteps []RecoveryMethod `json:"mediumRiskSteps"`
	MinimumSteps int `json:"minimumSteps"`
	AllowUserChoice bool `json:"allowUserChoice"`
	Enabled bool `json:"enabled"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
	SessionExpiry time.Duration `json:"sessionExpiry"`
}

type GetStatsRequestDTO struct {
}

type DeleteUserRequestDTO struct {
}

type ConsentPolicy struct {
	Description string `json:"description"`
	Renewable bool `json:"renewable"`
	Name string `json:"name"`
	PublishedAt Time `json:"publishedAt"`
	ValidityPeriod *int `json:"validityPeriod"`
	Active bool `json:"active"`
	ConsentType string `json:"consentType"`
	Content string `json:"content"`
	Id xid.ID `json:"id"`
	Metadata JSONBMap `json:"metadata"`
	OrganizationId string `json:"organizationId"`
	UpdatedAt time.Time `json:"updatedAt"`
	Required bool `json:"required"`
	Version string `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string `json:"createdBy"`
}

type UpdateMemberRequest struct {
	Role string `json:"role"`
}

type CreateTrainingRequest struct {
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
	Standard ComplianceStandard `json:"standard"`
}

type ConsentSettingsResponse struct {
	Settings interface{} `json:"settings"`
}

type CreatePolicyResponse struct {
	Id string `json:"id"`
}

type RotateAPIKeyRequest struct {
}

type RiskEngine struct {
}

type GetSessionsInput struct {
	UserId string `json:"userId"`
	AppId string `json:"appId"`
	Device string `json:"device"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Search string `json:"search"`
	Status string `json:"status"`
}

type StepUpAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type IDVerificationErrorResponse struct {
	Error string `json:"error"`
}

type BeginRegisterResponse struct {
	UserId string `json:"userId"`
	Challenge string `json:"challenge"`
	Options interface{} `json:"options"`
	Timeout time.Duration `json:"timeout"`
}

type Challenge struct {
	UserAgent string `json:"userAgent"`
	VerifiedAt Time `json:"verifiedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	MaxAttempts int `json:"maxAttempts"`
	Status ChallengeStatus `json:"status"`
	UserId xid.ID `json:"userId"`
	Attempts int `json:"attempts"`
	CreatedAt time.Time `json:"createdAt"`
	FactorId xid.ID `json:"factorId"`
	Metadata interface{} `json:"metadata"`
	Type FactorType `json:"type"`
}

type RequestReverification_req struct {
	Reason string `json:"reason"`
}

type EvaluateRequest struct {
	Resource interface{} `json:"resource"`
	ResourceId string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Action string `json:"action"`
	Context interface{} `json:"context"`
	Principal interface{} `json:"principal"`
	Request interface{} `json:"request"`
}

type VerifyCodeResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	Message string `json:"message"`
	Valid bool `json:"valid"`
}

type ResolveViolationResponse struct {
	Status string `json:"status"`
}

type DeleteEvidenceResponse struct {
	Status string `json:"status"`
}

type AsyncAdapter struct {
}

type AuditLogEntry struct {
	EnvironmentId string `json:"environmentId"`
	IpAddress string `json:"ipAddress"`
	NewValue interface{} `json:"newValue"`
	OldValue interface{} `json:"oldValue"`
	ResourceId string `json:"resourceId"`
	ActorId string `json:"actorId"`
	AppId string `json:"appId"`
	Id string `json:"id"`
	ResourceType string `json:"resourceType"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string `json:"userAgent"`
	UserOrganizationId *string `json:"userOrganizationId"`
	Action string `json:"action"`
}

type GetAuditLogsRequestDTO struct {
}

type StepUpPoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type CreateOrganizationInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	AppId string `json:"appId"`
	Logo string `json:"logo"`
	Metadata interface{} `json:"metadata"`
}

type SSOAuthResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type UserVerificationStatusResponse struct {
	Status UserVerificationStatus `json:"status"`
}

type GetStatusResponse struct {
	PolicyActive bool `json:"policyActive"`
	RequiredCount int `json:"requiredCount"`
	TrustedDevice bool `json:"trustedDevice"`
	Enabled bool `json:"enabled"`
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
	GracePeriod Time `json:"gracePeriod"`
}

type CompleteTrainingRequest struct {
	Score int `json:"score"`
}

type DefaultProviderRegistry struct {
}

type GetProfileResponse struct {
	Id string `json:"id"`
}

type GetDataExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type OIDCCallbackResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type CreateTraining_req struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type BackupCodesConfig struct {
	Allow_reuse bool `json:"allow_reuse"`
	Count int `json:"count"`
	Enabled bool `json:"enabled"`
	Format string `json:"format"`
	Length int `json:"length"`
}

type IDVerificationWebhookResponse struct {
	Status string `json:"status"`
}

type FacialCheckConfig struct {
	Enabled bool `json:"enabled"`
	MotionCapture bool `json:"motionCapture"`
	Variant string `json:"variant"`
}

type RegistrationService struct {
}

type GetRecoveryConfigResponse struct {
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
}

type CreateProfileFromTemplate_req struct {
	Standard ComplianceStandard `json:"standard"`
}

type UnpublishContentTypeRequest struct {
}

type RedisStateStore struct {
	Client Client `json:"client,omitempty"`
}

type GetTeamsInput struct {
	AppId string `json:"appId"`
	Limit int `json:"limit"`
	OrgId string `json:"orgId"`
	Page int `json:"page"`
	Search string `json:"search"`
}

type Handler struct {
}

type EndImpersonationRequest struct {
	Impersonation_id string `json:"impersonation_id"`
	Reason string `json:"reason"`
}

type MembersResponse struct {
	Members Member `json:"members"`
	Total int `json:"total"`
}

type NotificationSettingsDTO struct {
	Account AccountAutoSendDTO `json:"account"`
	AppName string `json:"appName"`
	Auth AuthAutoSendDTO `json:"auth"`
	Organization OrganizationAutoSendDTO `json:"organization"`
	Session SessionAutoSendDTO `json:"session"`
}

type GetProvidersInput struct {
}

type EmailFactorAdapter struct {
}

type TemplatesListResponse struct {
	TotalCount int `json:"totalCount"`
	Categories []string `json:"categories"`
	Templates []*TemplateResponse `json:"templates"`
}

type ComplianceReportsResponse struct {
	Reports []*interface{} `json:"reports"`
}

type CreateAppRequest struct {
}

type ClientAuthResult struct {
}

type ListAuditEventsRequest struct {
}

type ResendResponse struct {
	Status string `json:"status"`
}

type ConsentStatusResponse struct {
	Status string `json:"status"`
}

type QueryEntriesRequest struct {
}

type RecoveryCodeUsage struct {
}

type ImpersonateUserRequest struct {
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
	Duration time.Duration `json:"duration"`
	User_id xid.ID `json:"user_id"`
}

type ListTrustedDevicesResponse struct {
	Count int `json:"count"`
	Devices []TrustedDevice `json:"devices"`
}

type AuditServiceAdapter struct {
}

type ListRecoverySessionsRequest struct {
	OrganizationId string `json:"organizationId"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	RequiresReview bool `json:"requiresReview"`
	Status RecoveryStatus `json:"status"`
}

type ConsentRecord struct {
	ConsentType string `json:"consentType"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	CreatedAt time.Time `json:"createdAt"`
	Granted bool `json:"granted"`
	GrantedAt time.Time `json:"grantedAt"`
	UserId string `json:"userId"`
	Version string `json:"version"`
	UserAgent string `json:"userAgent"`
	IpAddress string `json:"ipAddress"`
	Metadata JSONBMap `json:"metadata"`
	Purpose string `json:"purpose"`
	RevokedAt Time `json:"revokedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ExpiresAt Time `json:"expiresAt"`
}

type UpdateSecretInput struct {
	Description string `json:"description"`
	SecretId string `json:"secretId"`
	Tags []string `json:"tags"`
	Value interface{} `json:"value"`
	AppId string `json:"appId"`
	ChangeReason string `json:"changeReason"`
}

type ProviderCheckResult struct {
}

type RestoreRevisionRequest struct {
}

type DocumentVerification struct {
}

type ConsentExpiryConfig struct {
	DefaultValidityDays int `json:"defaultValidityDays"`
	Enabled bool `json:"enabled"`
	ExpireCheckInterval time.Duration `json:"expireCheckInterval"`
	RenewalReminderDays int `json:"renewalReminderDays"`
	RequireReConsent bool `json:"requireReConsent"`
	AllowRenewal bool `json:"allowRenewal"`
	AutoExpireCheck bool `json:"autoExpireCheck"`
}

type UpdateFactorRequest struct {
	Priority *FactorPriority `json:"priority"`
	Status *FactorStatus `json:"status"`
	Metadata interface{} `json:"metadata"`
	Name *string `json:"name"`
}

type MigrationStatusResponse struct {
	StartedAt time.Time `json:"startedAt"`
	Status string `json:"status"`
	TotalPolicies int `json:"totalPolicies"`
	ValidationPassed bool `json:"validationPassed"`
	AppId string `json:"appId"`
	CompletedAt Time `json:"completedAt"`
	EnvironmentId string `json:"environmentId"`
	FailedCount int `json:"failedCount"`
	MigratedCount int `json:"migratedCount"`
	Progress float64 `json:"progress"`
	UserOrganizationId *string `json:"userOrganizationId"`
	Errors []string `json:"errors"`
}

type InviteMemberRequest struct {
	Email string `json:"email"`
	Role string `json:"role"`
}

type DeleteFieldRequest struct {
}

type ErrorResponse struct {
	Code string `json:"code"`
	Details interface{} `json:"details"`
	Error string `json:"error"`
}

type UpdateEntryRequest struct {
}

type AuthURLResponse struct {
	Url string `json:"url"`
}

type IDVerificationListResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type CreateConsentPolicyRequest struct {
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	Content string `json:"content"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Renewable bool `json:"renewable"`
	Required bool `json:"required"`
	ValidityPeriod *int `json:"validityPeriod"`
}

type RetentionConfig struct {
	ArchiveBeforePurge bool `json:"archiveBeforePurge"`
	ArchivePath string `json:"archivePath"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	PurgeSchedule string `json:"purgeSchedule"`
}

type BackupAuthVideoResponse struct {
	Session_id string `json:"session_id"`
}

type VerifyRecoveryCodeResponse struct {
	Message string `json:"message"`
	RemainingCodes int `json:"remainingCodes"`
	Valid bool `json:"valid"`
}

type LinkAccountRequest struct {
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
}

type VerifyChallengeResponse struct {
	Success bool `json:"success"`
	Token string `json:"token"`
	ExpiresAt Time `json:"expiresAt"`
	FactorsRemaining int `json:"factorsRemaining"`
	SessionComplete bool `json:"sessionComplete"`
}

// Session represents User session
type Session struct {
	UserAgent *string `json:"userAgent,omitempty"`
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
	UserId string `json:"userId"`
	Token string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	IpAddress *string `json:"ipAddress,omitempty"`
}

type PaginationDTO struct {
	PageSize int `json:"pageSize"`
	TotalCount int64 `json:"totalCount"`
	TotalPages int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
	HasNext bool `json:"hasNext"`
	HasPrev bool `json:"hasPrev"`
}

type UpdateContentTypeRequest struct {
}

type VerifyTrustedContactResponse struct {
	ContactId xid.ID `json:"contactId"`
	Message string `json:"message"`
	Verified bool `json:"verified"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

type ContinueRecoveryResponse struct {
	Data interface{} `json:"data"`
	ExpiresAt time.Time `json:"expiresAt"`
	Instructions string `json:"instructions"`
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
	TotalSteps int `json:"totalSteps"`
	CurrentStep int `json:"currentStep"`
}

type GetComplianceStatusResponse struct {
	Status string `json:"status"`
}

type NotificationWebhookResponse struct {
	Status string `json:"status"`
}

type DataDeletionConfig struct {
	PreserveLegalData bool `json:"preserveLegalData"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
	ArchiveBeforeDeletion bool `json:"archiveBeforeDeletion"`
	GracePeriodDays int `json:"gracePeriodDays"`
	NotifyBeforeDeletion bool `json:"notifyBeforeDeletion"`
	RetentionExemptions []string `json:"retentionExemptions"`
	AllowPartialDeletion bool `json:"allowPartialDeletion"`
	ArchivePath string `json:"archivePath"`
	AutoProcessAfterGrace bool `json:"autoProcessAfterGrace"`
	Enabled bool `json:"enabled"`
}

type ConsentExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type ConsentPolicyResponse struct {
	Id string `json:"id"`
}

type TrustDeviceRequest struct {
	DeviceId string `json:"deviceId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
}

type UpdatePolicyResponse struct {
	Id string `json:"id"`
}

type UpdateTeamRequest struct {
	Description string `json:"description"`
	Name string `json:"name"`
}

type serviceAdapter struct {
}

type ConsentTypeStatus struct {
	NeedsRenewal bool `json:"needsRenewal"`
	Type string `json:"type"`
	Version string `json:"version"`
	ExpiresAt Time `json:"expiresAt"`
	Granted bool `json:"granted"`
	GrantedAt time.Time `json:"grantedAt"`
}

type RouteRule struct {
	Description string `json:"description"`
	Method string `json:"method"`
	Org_id string `json:"org_id"`
	Pattern string `json:"pattern"`
	Security_level SecurityLevel `json:"security_level"`
}

type OrganizationUIRegistry struct {
}

type EmailVerificationConfig struct {
	CodeLength int `json:"codeLength"`
	EmailTemplate string `json:"emailTemplate"`
	Enabled bool `json:"enabled"`
	FromAddress string `json:"fromAddress"`
	FromName string `json:"fromName"`
	MaxAttempts int `json:"maxAttempts"`
	RequireEmailProof bool `json:"requireEmailProof"`
	CodeExpiry time.Duration `json:"codeExpiry"`
}

type DeleteSecretOutput struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type GetReportResponse struct {
	Id string `json:"id"`
}

type HandleConsentRequest struct {
	State string `json:"state"`
	Action string `json:"action"`
	Client_id string `json:"client_id"`
	Code_challenge string `json:"code_challenge"`
	Code_challenge_method string `json:"code_challenge_method"`
	Redirect_uri string `json:"redirect_uri"`
	Response_type string `json:"response_type"`
	Scope string `json:"scope"`
}

type UpdateSettingsResult struct {
	Message string `json:"message"`
	Settings NotificationSettingsDTO `json:"settings"`
	Success bool `json:"success"`
}

// Device represents User device
type Device struct {
	Type *string `json:"type,omitempty"`
	LastUsedAt string `json:"lastUsedAt"`
	IpAddress *string `json:"ipAddress,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
	Id string `json:"id"`
	UserId string `json:"userId"`
	Name *string `json:"name,omitempty"`
}

type ComplianceProfileResponse struct {
	Id string `json:"id"`
}

type CookieConsent struct {
	Essential bool `json:"essential"`
	Id xid.ID `json:"id"`
	UserAgent string `json:"userAgent"`
	Marketing bool `json:"marketing"`
	UpdatedAt time.Time `json:"updatedAt"`
	ConsentBannerVersion string `json:"consentBannerVersion"`
	CreatedAt time.Time `json:"createdAt"`
	Functional bool `json:"functional"`
	Analytics bool `json:"analytics"`
	ExpiresAt time.Time `json:"expiresAt"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	UserId string `json:"userId"`
}

type ConsentAuditLog struct {
	UserAgent string `json:"userAgent"`
	UserId string `json:"userId"`
	ConsentId string `json:"consentId"`
	ConsentType string `json:"consentType"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	PreviousValue JSONBMap `json:"previousValue"`
	Action string `json:"action"`
	CreatedAt time.Time `json:"createdAt"`
	NewValue JSONBMap `json:"newValue"`
	Purpose string `json:"purpose"`
	Reason string `json:"reason"`
}

type GetInvitationsInput struct {
	Page int `json:"page"`
	Status string `json:"status"`
	AppId string `json:"appId"`
	Limit int `json:"limit"`
	OrgId string `json:"orgId"`
}

type RevokeDeviceResponse struct {
	Status string `json:"status"`
}

type DeclareABTestWinnerRequest struct {
}

type CompleteVideoSessionRequest struct {
	LivenessPassed bool `json:"livenessPassed"`
	LivenessScore float64 `json:"livenessScore"`
	Notes string `json:"notes"`
	VerificationResult string `json:"verificationResult"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type GetInvitationsResult struct {
	Data []InvitationDTO `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
}

type ListNotificationsResponse struct {
	Notifications []*interface{} `json:"notifications"`
	Total int `json:"total"`
}

type TestSendTemplateResult struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type TestProviderInput struct {
	ProviderType string `json:"providerType"`
	Recipient string `json:"recipient"`
}

type CreateTemplateResult struct {
	Message string `json:"message"`
	Success bool `json:"success"`
	Template TemplateDTO `json:"template"`
}

type NamespaceResponse struct {
	CreatedAt time.Time `json:"createdAt"`
	EnvironmentId string `json:"environmentId"`
	InheritPlatform bool `json:"inheritPlatform"`
	PolicyCount int `json:"policyCount"`
	ResourceCount int `json:"resourceCount"`
	UpdatedAt time.Time `json:"updatedAt"`
	AppId string `json:"appId"`
	Description string `json:"description"`
	Id string `json:"id"`
	Name string `json:"name"`
	TemplateId *string `json:"templateId"`
	UserOrganizationId *string `json:"userOrganizationId"`
	ActionCount int `json:"actionCount"`
}

type RecoveryAttemptLog struct {
}

type BackupAuthConfigResponse struct {
	Config interface{} `json:"config"`
}

type GetChallengeStatusRequest struct {
}

// StatusResponse represents Status response
type StatusResponse struct {
	Status string `json:"status"`
}

type PreviewTemplate_req struct {
	Variables interface{} `json:"variables"`
}

type ResourceAttributeRequest struct {
	Type string `json:"type"`
	Default interface{} `json:"default"`
	Description string `json:"description"`
	Name string `json:"name"`
	Required bool `json:"required"`
}

type Middleware struct {
}

type ListProfilesFilter struct {
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
}

type VerifyRequest struct {
	Code string `json:"code"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	Remember bool `json:"remember"`
}

type GetRevisionRequest struct {
}

type IDVerificationStatusResponse struct {
	Status interface{} `json:"status"`
}

type UpdateProvider_req struct {
	Config interface{} `json:"config"`
	IsActive bool `json:"isActive"`
	IsDefault bool `json:"isDefault"`
}

type ImpersonationMiddleware struct {
}

type ListImpersonationsRequest struct {
}

type ListUsersRequest struct {
	Limit int `json:"limit"`
	Page int `json:"page"`
	Role string `json:"role"`
	Search string `json:"search"`
	Status string `json:"status"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
}

type UpdateOrganizationHandlerRequest struct {
}

type ProviderSessionRequest struct {
}

type JumioConfig struct {
	ApiToken string `json:"apiToken"`
	CallbackUrl string `json:"callbackUrl"`
	DataCenter string `json:"dataCenter"`
	EnableAMLScreening bool `json:"enableAMLScreening"`
	EnableExtraction bool `json:"enableExtraction"`
	EnabledDocumentTypes []string `json:"enabledDocumentTypes"`
	PresetId string `json:"presetId"`
	ApiSecret string `json:"apiSecret"`
	EnableLiveness bool `json:"enableLiveness"`
	Enabled bool `json:"enabled"`
	EnabledCountries []string `json:"enabledCountries"`
	VerificationType string `json:"verificationType"`
}

type RecordCookieConsentResponse struct {
	Preferences interface{} `json:"preferences"`
}

type TrustedContact struct {
}

type ListSessionsRequestDTO struct {
}

type CreateSessionHTTPRequest struct {
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
}

type UpdatePrivacySettingsRequest struct {
	DataRetentionDays *int `json:"dataRetentionDays"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
	ConsentRequired *bool `json:"consentRequired"`
	DpoEmail string `json:"dpoEmail"`
	AllowDataPortability *bool `json:"allowDataPortability"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	ExportFormat []string `json:"exportFormat"`
	GdprMode *bool `json:"gdprMode"`
	CcpaMode *bool `json:"ccpaMode"`
}

type UploadDocumentRequest struct {
	BackImage string `json:"backImage"`
	DocumentType string `json:"documentType"`
	FrontImage string `json:"frontImage"`
	Selfie string `json:"selfie"`
	SessionId xid.ID `json:"sessionId"`
}

type BackupAuthContactResponse struct {
	Id string `json:"id"`
}

type TemplateService struct {
}

type StepUpErrorResponse struct {
	Error string `json:"error"`
}

type ProviderRegisteredResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Type string `json:"type"`
}

type AccountAutoSendConfig struct {
	Reactivated bool `json:"reactivated"`
	Suspended bool `json:"suspended"`
	Username_changed bool `json:"username_changed"`
	Deleted bool `json:"deleted"`
	Email_change_request bool `json:"email_change_request"`
	Email_changed bool `json:"email_changed"`
	Password_changed bool `json:"password_changed"`
}

type PoliciesListResponse struct {
	Policies []*PolicyResponse `json:"policies"`
	TotalCount int `json:"totalCount"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
}

type CheckMetadata struct {
	Category string `json:"category"`
	Description string `json:"description"`
	Name string `json:"name"`
	Severity string `json:"severity"`
	Standards []string `json:"standards"`
	AutoRun bool `json:"autoRun"`
}

type ListEvidenceResponse struct {
	Evidence []*interface{} `json:"evidence"`
}

type PreviewConversionRequest struct {
	Actions []string `json:"actions"`
	Condition string `json:"condition"`
	Resource string `json:"resource"`
	Subject string `json:"subject"`
}

type UpdateAPIKeyRequest struct {
	Permissions interface{} `json:"permissions"`
	Rate_limit *int `json:"rate_limit"`
	Scopes []string `json:"scopes"`
	Allowed_ips []string `json:"allowed_ips"`
	Description *string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name *string `json:"name"`
}

type JWK struct {
	Use string `json:"use"`
	Alg string `json:"alg"`
	E string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N string `json:"n"`
}

type UpdateSecretOutput struct {
	Secret SecretItem `json:"secret"`
}

type DownloadDataExportResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type GetSessionStatsResult struct {
	Stats SessionStatsDTO `json:"stats"`
}

type GetConsentAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type GetTemplateResult struct {
	Template TemplateDTO `json:"template"`
}

type AuthorizeRequest struct {
	Max_age *int `json:"max_age"`
	Response_type string `json:"response_type"`
	Scope string `json:"scope"`
	State string `json:"state"`
	Ui_locales string `json:"ui_locales"`
	Acr_values string `json:"acr_values"`
	Client_id string `json:"client_id"`
	Code_challenge_method string `json:"code_challenge_method"`
	Id_token_hint string `json:"id_token_hint"`
	Login_hint string `json:"login_hint"`
	Nonce string `json:"nonce"`
	Prompt string `json:"prompt"`
	Redirect_uri string `json:"redirect_uri"`
	Code_challenge string `json:"code_challenge"`
}

type ApproveRecoveryResponse struct {
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
	Approved bool `json:"approved"`
	ApprovedAt time.Time `json:"approvedAt"`
}

type UpdateTeamResult struct {
	Team TeamDTO `json:"team"`
}

type ListPendingRequirementsResponse struct {
	Requirements []*interface{} `json:"requirements"`
}

type UpdateProviderRequest struct {
}

type MockAuditService struct {
}

type DeletePasskeyRequest struct {
}

type DocumentVerificationConfig struct {
	StoragePath string `json:"storagePath"`
	AcceptedDocuments []string `json:"acceptedDocuments"`
	Provider string `json:"provider"`
	RequireManualReview bool `json:"requireManualReview"`
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	StorageProvider string `json:"storageProvider"`
	Enabled bool `json:"enabled"`
	EncryptAtRest bool `json:"encryptAtRest"`
	EncryptionKey string `json:"encryptionKey"`
	MinConfidenceScore float64 `json:"minConfidenceScore"`
	RequireBothSides bool `json:"requireBothSides"`
	RequireSelfie bool `json:"requireSelfie"`
}

type CancelInvitationResult struct {
	Success bool `json:"success"`
}

type MigrationHandler struct {
}

type VerifySecurityAnswersRequest struct {
	Answers interface{} `json:"answers"`
	SessionId xid.ID `json:"sessionId"`
}

type UploadDocumentResponse struct {
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
	ProcessingTime string `json:"processingTime"`
	Status string `json:"status"`
	UploadedAt time.Time `json:"uploadedAt"`
}

type AMLMatch struct {
}

type CompleteTrainingResponse struct {
	Status string `json:"status"`
}

type StatsResponse struct {
	Banned_users int `json:"banned_users"`
	Timestamp string `json:"timestamp"`
	Total_sessions int `json:"total_sessions"`
	Total_users int `json:"total_users"`
	Active_sessions int `json:"active_sessions"`
	Active_users int `json:"active_users"`
}

type ChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata interface{} `json:"metadata"`
	UserId xid.ID `json:"userId"`
}

type UpdateRecoveryConfigResponse struct {
	Config interface{} `json:"config"`
}

type ListNotificationsHistoryResult struct {
	Notifications []NotificationHistoryDTO `json:"notifications"`
	Pagination PaginationDTO `json:"pagination"`
}

type ListAppsRequest struct {
}

type UpdatePasskeyRequest struct {
	Name string `json:"name"`
}

type RecoverySession struct {
}

type DeleteTemplateRequest struct {
}

type CreateDPARequest struct {
	SignedByEmail string `json:"signedByEmail"`
	SignedByTitle string `json:"signedByTitle"`
	AgreementType string `json:"agreementType"`
	EffectiveDate time.Time `json:"effectiveDate"`
	ExpiryDate Time `json:"expiryDate"`
	SignedByName string `json:"signedByName"`
	Version string `json:"version"`
	Content string `json:"content"`
	Metadata interface{} `json:"metadata"`
}

type GetExtensionDataInput struct {
	AppId string `json:"appId"`
	OrgId string `json:"orgId"`
}

type DailyAnalyticsDTO struct {
	TotalOpened int64 `json:"totalOpened"`
	TotalSent int64 `json:"totalSent"`
	Date string `json:"date"`
	DeliveryRate float64 `json:"deliveryRate"`
	OpenRate float64 `json:"openRate"`
	TotalClicked int64 `json:"totalClicked"`
	TotalDelivered int64 `json:"totalDelivered"`
}

type ComplianceReportResponse struct {
	Id string `json:"id"`
}

type DeleteAppRequest struct {
}

type GetSecurityQuestionsRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type ComplianceProfile struct {
	Standards []ComplianceStandard `json:"standards"`
	MfaRequired bool `json:"mfaRequired"`
	CreatedAt time.Time `json:"createdAt"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	Metadata interface{} `json:"metadata"`
	AppId string `json:"appId"`
	ComplianceContact string `json:"complianceContact"`
	PasswordMinLength int `json:"passwordMinLength"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	RbacRequired bool `json:"rbacRequired"`
	RegularAccessReview bool `json:"regularAccessReview"`
	Name string `json:"name"`
	AuditLogExport bool `json:"auditLogExport"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
	RetentionDays int `json:"retentionDays"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	Status string `json:"status"`
	DataResidency string `json:"dataResidency"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	Id string `json:"id"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	DpoContact string `json:"dpoContact"`
	SessionMaxAge int `json:"sessionMaxAge"`
	UpdatedAt time.Time `json:"updatedAt"`
	LeastPrivilege bool `json:"leastPrivilege"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
}

type SendResponse struct {
	Dev_otp string `json:"dev_otp"`
	Status string `json:"status"`
}

type ConsentReport struct {
	TotalUsers int `json:"totalUsers"`
	CompletedDeletions int `json:"completedDeletions"`
	ConsentRate float64 `json:"consentRate"`
	DpasActive int `json:"dpasActive"`
	DpasExpiringSoon int `json:"dpasExpiringSoon"`
	OrganizationId string `json:"organizationId"`
	ReportPeriodEnd time.Time `json:"reportPeriodEnd"`
	UsersWithConsent int `json:"usersWithConsent"`
	ConsentsByType interface{} `json:"consentsByType"`
	DataExportsThisPeriod int `json:"dataExportsThisPeriod"`
	PendingDeletions int `json:"pendingDeletions"`
	ReportPeriodStart time.Time `json:"reportPeriodStart"`
}

type EnableResponse struct {
	Totp_uri string `json:"totp_uri"`
	Status string `json:"status"`
}

type Status struct {
}

type AdaptiveMFAConfig struct {
	Enabled bool `json:"enabled"`
	Factor_ip_reputation bool `json:"factor_ip_reputation"`
	Factor_location_change bool `json:"factor_location_change"`
	New_device_risk float64 `json:"new_device_risk"`
	Require_step_up_threshold float64 `json:"require_step_up_threshold"`
	Velocity_risk float64 `json:"velocity_risk"`
	Factor_new_device bool `json:"factor_new_device"`
	Factor_velocity bool `json:"factor_velocity"`
	Location_change_risk float64 `json:"location_change_risk"`
	Risk_threshold float64 `json:"risk_threshold"`
}

type PaginationInfo struct {
	CurrentPage int `json:"currentPage"`
	PageSize int `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

type RegisterClientResponse struct {
	Application_type string `json:"application_type"`
	Contacts []string `json:"contacts"`
	Response_types []string `json:"response_types"`
	Client_id_issued_at int64 `json:"client_id_issued_at"`
	Policy_uri string `json:"policy_uri"`
	Client_id string `json:"client_id"`
	Client_name string `json:"client_name"`
	Client_secret string `json:"client_secret"`
	Client_secret_expires_at int64 `json:"client_secret_expires_at"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Scope string `json:"scope"`
	Tos_uri string `json:"tos_uri"`
}

type RequestEmailChangeRequest struct {
	NewEmail string `json:"newEmail"`
}

type TeamHandler struct {
}

type OTPSentResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type ProvidersAppResponse struct {
	AppId string `json:"appId"`
	Providers []string `json:"providers"`
}

type PreviewTemplateResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

type NotificationTemplateResponse struct {
	Template interface{} `json:"template"`
}

type SignInRequest struct {
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

type TemplateDefault struct {
}

type MemoryChallengeStore struct {
}

type ClientSummary struct {
	Name string `json:"name"`
	ApplicationType string `json:"applicationType"`
	ClientID string `json:"clientID"`
	CreatedAt string `json:"createdAt"`
	IsOrgLevel bool `json:"isOrgLevel"`
}

type ListFactorsResponse struct {
	Count int `json:"count"`
	Factors []Factor `json:"factors"`
}

type GetPolicyResponse struct {
	Id string `json:"id"`
}

type ConfirmEmailChangeRequest struct {
	Token string `json:"token"`
}

type ComplianceCheck struct {
	CheckType string `json:"checkType"`
	CreatedAt time.Time `json:"createdAt"`
	Evidence []string `json:"evidence"`
	LastCheckedAt time.Time `json:"lastCheckedAt"`
	ProfileId string `json:"profileId"`
	AppId string `json:"appId"`
	Id string `json:"id"`
	NextCheckAt time.Time `json:"nextCheckAt"`
	Result interface{} `json:"result"`
	Status string `json:"status"`
}

type ListEntriesRequest struct {
}

type IDTokenClaims struct {
	Email_verified bool `json:"email_verified"`
	Family_name string `json:"family_name"`
	Name string `json:"name"`
	Nonce string `json:"nonce"`
	Preferred_username string `json:"preferred_username"`
	Session_state string `json:"session_state"`
	Auth_time int64 `json:"auth_time"`
	Email string `json:"email"`
	Given_name string `json:"given_name"`
}

type ListSessionsResponse struct {
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	Sessions Session `json:"sessions"`
}

type EmailConfig struct {
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
}

type MFASession struct {
	VerifiedFactors ID `json:"verifiedFactors"`
	CompletedAt Time `json:"completedAt"`
	FactorsVerified int `json:"factorsVerified"`
	Id xid.ID `json:"id"`
	RiskLevel RiskLevel `json:"riskLevel"`
	SessionToken string `json:"sessionToken"`
	UserAgent string `json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	IpAddress string `json:"ipAddress"`
	Metadata interface{} `json:"metadata"`
	UserId xid.ID `json:"userId"`
}

type InviteMemberHandlerRequest struct {
}

type SessionTokenResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type ComplianceDashboardResponse struct {
	Metrics interface{} `json:"metrics"`
}

type UnassignRoleRequest struct {
}

type ClientRegistrationResponse struct {
	Logo_uri string `json:"logo_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
	Application_type string `json:"application_type"`
	Client_id_issued_at int64 `json:"client_id_issued_at"`
	Grant_types []string `json:"grant_types"`
	Response_types []string `json:"response_types"`
	Tos_uri string `json:"tos_uri"`
	Client_secret_expires_at int64 `json:"client_secret_expires_at"`
	Scope string `json:"scope"`
	Client_id string `json:"client_id"`
	Contacts []string `json:"contacts"`
	Policy_uri string `json:"policy_uri"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Client_name string `json:"client_name"`
	Client_secret string `json:"client_secret"`
}

type VerifyCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type StepUpRequirement struct {
	Metadata interface{} `json:"metadata"`
	Method string `json:"method"`
	Reason string `json:"reason"`
	Resource_action string `json:"resource_action"`
	Route string `json:"route"`
	Created_at time.Time `json:"created_at"`
	Expires_at time.Time `json:"expires_at"`
	Org_id string `json:"org_id"`
	Risk_score float64 `json:"risk_score"`
	Rule_name string `json:"rule_name"`
	Session_id string `json:"session_id"`
	Status string `json:"status"`
	User_id string `json:"user_id"`
	Currency string `json:"currency"`
	Current_level SecurityLevel `json:"current_level"`
	Resource_type string `json:"resource_type"`
	Amount float64 `json:"amount"`
	Challenge_token string `json:"challenge_token"`
	Required_level SecurityLevel `json:"required_level"`
	User_agent string `json:"user_agent"`
	Fulfilled_at Time `json:"fulfilled_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
}

type FactorAdapterRegistry struct {
}

type InviteMemberResult struct {
	Invitation InvitationDTO `json:"invitation"`
}

type ListOrganizationsRequest struct {
}

type UpdatePolicyRequest struct {
	Name string `json:"name"`
	Priority int `json:"priority"`
	ResourceType string `json:"resourceType"`
	Actions []string `json:"actions"`
	Description string `json:"description"`
	Enabled *bool `json:"enabled"`
	Expression string `json:"expression"`
}

type ValidatePolicyResponse struct {
	Valid bool `json:"valid"`
	Warnings []string `json:"warnings"`
	Complexity int `json:"complexity"`
	Error string `json:"error"`
	Errors []string `json:"errors"`
	Message string `json:"message"`
}

type BulkPublishRequest struct {
	Ids []string `json:"ids"`
}

type UpdateTeamHandlerRequest struct {
}

type DeviceAuthorizeResponse struct {
	Device_code string `json:"device_code"`
	Expires_in int `json:"expires_in"`
	Interval int `json:"interval"`
	User_code string `json:"user_code"`
	Verification_uri string `json:"verification_uri"`
	Verification_uri_complete string `json:"verification_uri_complete"`
}

type RolesResponse struct {
	Roles Role `json:"roles"`
}

type ScopeDefinition struct {
}

type SendNotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type GetClientResponse struct {
	IsOrgLevel bool `json:"isOrgLevel"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	RedirectURIs []string `json:"redirectURIs"`
	RequireConsent bool `json:"requireConsent"`
	Contacts []string `json:"contacts"`
	Name string `json:"name"`
	PolicyURI string `json:"policyURI"`
	RequirePKCE bool `json:"requirePKCE"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	CreatedAt string `json:"createdAt"`
	OrganizationID string `json:"organizationID"`
	GrantTypes []string `json:"grantTypes"`
	LogoURI string `json:"logoURI"`
	ResponseTypes []string `json:"responseTypes"`
	TosURI string `json:"tosURI"`
	TrustedClient bool `json:"trustedClient"`
	UpdatedAt string `json:"updatedAt"`
	AllowedScopes []string `json:"allowedScopes"`
	ApplicationType string `json:"applicationType"`
	ClientID string `json:"clientID"`
}

type UpdateClientResponse struct {
	LogoURI string `json:"logoURI"`
	Name string `json:"name"`
	PolicyURI string `json:"policyURI"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	ResponseTypes []string `json:"responseTypes"`
	RedirectURIs []string `json:"redirectURIs"`
	UpdatedAt string `json:"updatedAt"`
	ClientID string `json:"clientID"`
	Contacts []string `json:"contacts"`
	IsOrgLevel bool `json:"isOrgLevel"`
	TosURI string `json:"tosURI"`
	TrustedClient bool `json:"trustedClient"`
	AllowedScopes []string `json:"allowedScopes"`
	OrganizationID string `json:"organizationID"`
	RequireConsent bool `json:"requireConsent"`
	RequirePKCE bool `json:"requirePKCE"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	ApplicationType string `json:"applicationType"`
	CreatedAt string `json:"createdAt"`
	GrantTypes []string `json:"grantTypes"`
}

type GetFactorResponse struct {
	Priority FactorPriority `json:"priority"`
	Type FactorType `json:"type"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserId xid.ID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsedAt Time `json:"lastUsedAt"`
	Name string `json:"name"`
	Status FactorStatus `json:"status"`
	VerifiedAt Time `json:"verifiedAt"`
	ExpiresAt Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	Metadata interface{} `json:"metadata"`
}

type ClientsListResponse struct {
	Clients []ClientSummary `json:"clients"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Total int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type MemberDTO struct {
	Status string `json:"status"`
	UserEmail string `json:"userEmail"`
	UserId string `json:"userId"`
	UserName string `json:"userName"`
	Id string `json:"id"`
	JoinedAt time.Time `json:"joinedAt"`
	Role string `json:"role"`
}

type JumioProvider struct {
}

type DeviceAuthorizeDecisionRequest struct {
	User_code string `json:"user_code"`
	Action string `json:"action"`
}

type GetAnalyticsInput struct {
	Days *int `json:"days"`
	EndDate *string `json:"endDate"`
	StartDate *string `json:"startDate"`
	TemplateId *string `json:"templateId"`
}

type StepUpVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type TOTPConfig struct {
	Algorithm string `json:"algorithm"`
	Digits int `json:"digits"`
	Enabled bool `json:"enabled"`
	Issuer string `json:"issuer"`
	Period int `json:"period"`
	Window_size int `json:"window_size"`
}

type SignUpRequest struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type ResetPasswordRequest struct {
	Token string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type ListTemplatesResult struct {
	Pagination PaginationDTO `json:"pagination"`
	Templates []TemplateDTO `json:"templates"`
}

type OrganizationHandler struct {
}

type ResetTemplateResponse struct {
	Status string `json:"status"`
}

type GetNotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type DashboardConfig struct {
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
	ShowRecentChecks bool `json:"showRecentChecks"`
	ShowReports bool `json:"showReports"`
	ShowScore bool `json:"showScore"`
	ShowViolations bool `json:"showViolations"`
}

type ScopeResolver struct {
}

type AddTeamMember_req struct {
	Member_id xid.ID `json:"member_id"`
	Role string `json:"role"`
}

type GetPasskeyRequest struct {
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type ListViolationsResponse struct {
	Violations []*interface{} `json:"violations"`
}

type MigrateAllResponse struct {
	DryRun bool `json:"dryRun"`
	FailedPolicies int `json:"failedPolicies"`
	MigratedPolicies int `json:"migratedPolicies"`
	StartedAt string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
	ConvertedPolicies []PolicyPreviewResponse `json:"convertedPolicies"`
	Errors []MigrationErrorResponse `json:"errors"`
	SkippedPolicies int `json:"skippedPolicies"`
	TotalPolicies int `json:"totalPolicies"`
}

type GetTeamRequest struct {
}

type SendCodeRequest struct {
	Phone string `json:"phone"`
}

type mockUserService struct {
}

type GetCurrentResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type WebAuthnWrapper struct {
}

type DataProcessingAgreement struct {
	CreatedAt time.Time `json:"createdAt"`
	IpAddress string `json:"ipAddress"`
	Metadata JSONBMap `json:"metadata"`
	SignedByName string `json:"signedByName"`
	AgreementType string `json:"agreementType"`
	EffectiveDate time.Time `json:"effectiveDate"`
	ExpiryDate Time `json:"expiryDate"`
	SignedBy string `json:"signedBy"`
	SignedByEmail string `json:"signedByEmail"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version string `json:"version"`
	DigitalSignature string `json:"digitalSignature"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	Status string `json:"status"`
	Content string `json:"content"`
	SignedByTitle string `json:"signedByTitle"`
}

type WebhookResponse struct {
	Status string `json:"status"`
	Received bool `json:"received"`
}

type ScheduleVideoSessionResponse struct {
	Instructions string `json:"instructions"`
	JoinUrl string `json:"joinUrl"`
	Message string `json:"message"`
	ScheduledAt time.Time `json:"scheduledAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type EnableRequest struct {
}

type MFAConfigResponse struct {
	Required_factor_count int `json:"required_factor_count"`
	Allowed_factor_types []string `json:"allowed_factor_types"`
	Enabled bool `json:"enabled"`
}

type CreateProvider_req struct {
	Config interface{} `json:"config"`
	IsDefault bool `json:"isDefault"`
	OrganizationId *string `json:"organizationId,omitempty"`
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
}

type ResourceResponse struct {
	Attributes ResourceAttribute `json:"attributes"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
	NamespaceId string `json:"namespaceId"`
	Type string `json:"type"`
}

type IPWhitelistConfig struct {
	Strict_mode bool `json:"strict_mode"`
	Enabled bool `json:"enabled"`
}

type TrustedContactsConfig struct {
	AllowPhoneContacts bool `json:"allowPhoneContacts"`
	MaximumContacts int `json:"maximumContacts"`
	MinimumContacts int `json:"minimumContacts"`
	RequireVerification bool `json:"requireVerification"`
	RequiredToRecover int `json:"requiredToRecover"`
	AllowEmailContacts bool `json:"allowEmailContacts"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	Enabled bool `json:"enabled"`
	MaxNotificationsPerDay int `json:"maxNotificationsPerDay"`
	VerificationExpiry time.Duration `json:"verificationExpiry"`
}

type DeclareABTestWinner_req struct {
	AbTestGroup string `json:"abTestGroup"`
	WinnerId string `json:"winnerId"`
}

type RevokeSessionInput struct {
	AppId string `json:"appId"`
	SessionId string `json:"sessionId"`
}

type RevokeSessionResult struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type OrganizationSummaryDTO struct {
	MemberCount int64 `json:"memberCount"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	TeamCount int64 `json:"teamCount"`
	UserRole string `json:"userRole"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	Logo string `json:"logo"`
}

type AccountAutoSendDTO struct {
	EmailChanged bool `json:"emailChanged"`
	PasswordChanged bool `json:"passwordChanged"`
	Reactivated bool `json:"reactivated"`
	Suspended bool `json:"suspended"`
	UsernameChanged bool `json:"usernameChanged"`
	Deleted bool `json:"deleted"`
	EmailChangeRequest bool `json:"emailChangeRequest"`
}

type GetSettingsInput struct {
}

type VerifyEnrolledFactorRequest struct {
	Code string `json:"code"`
	Data interface{} `json:"data"`
}

type CreateVerificationSession_req struct {
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
}

type ImpersonationVerifyResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id string `json:"target_user_id"`
}

type UnbanUserRequest struct {
	App_id xid.ID `json:"app_id"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
	User_organization_id ID `json:"user_organization_id"`
}

type UpdateConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
	Reason string `json:"reason"`
}

type GetByIDResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type ListTrainingResponse struct {
	Training []*interface{} `json:"training"`
}

type RegisterClientRequest struct {
	Tos_uri string `json:"tos_uri"`
	Trusted_client bool `json:"trusted_client"`
	Grant_types []string `json:"grant_types"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
	Response_types []string `json:"response_types"`
	Client_name string `json:"client_name"`
	Scope string `json:"scope"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Contacts []string `json:"contacts"`
	Logo_uri string `json:"logo_uri"`
	Require_consent bool `json:"require_consent"`
	Application_type string `json:"application_type"`
	Require_pkce bool `json:"require_pkce"`
}

type ListTeamsRequest struct {
}

type NoOpSMSProvider struct {
}

type EnrollFactorRequest struct {
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	Type FactorType `json:"type"`
}

type CreateABTestVariant_req struct {
	Name string `json:"name"`
	Subject string `json:"subject"`
	Weight int `json:"weight"`
	Body string `json:"body"`
}

type TemplateDTO struct {
	Active bool `json:"active"`
	AppId string `json:"appId"`
	CreatedAt string `json:"createdAt"`
	Metadata interface{} `json:"metadata"`
	Subject string `json:"subject"`
	UpdatedAt string `json:"updatedAt"`
	Language string `json:"language"`
	Type string `json:"type"`
	Variables []string `json:"variables"`
	Body string `json:"body"`
	IsDefault bool `json:"isDefault"`
	TemplateKey string `json:"templateKey"`
	Id string `json:"id"`
	IsModified bool `json:"isModified"`
	Name string `json:"name"`
}

type SMSProviderConfig struct {
	Config interface{} `json:"config"`
	From string `json:"from"`
	Provider string `json:"provider"`
}

type TrustedContactInfo struct {
	Id xid.ID `json:"id"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
	Verified bool `json:"verified"`
	VerifiedAt Time `json:"verifiedAt"`
	Active bool `json:"active"`
	Email string `json:"email"`
}

type StepUpVerificationResponse struct {
	Expires_at string `json:"expires_at"`
	Verified bool `json:"verified"`
}

type RemoveMemberInput struct {
	AppId string `json:"appId"`
	MemberId string `json:"memberId"`
	OrgId string `json:"orgId"`
}

type SetActiveRequest struct {
	Id string `json:"id"`
}

type FinishRegisterResponse struct {
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	CredentialId string `json:"credentialId"`
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
}

type GetSecretOutput struct {
	Secret SecretItem `json:"secret"`
}

type ListPoliciesResponse struct {
	Policies []*interface{} `json:"policies"`
}

type GetVerificationSessionResponse struct {
	Session interface{} `json:"session"`
}

type GetNotificationDetailResult struct {
	Notification NotificationHistoryDTO `json:"notification"`
}

type TwoFAStatusResponse struct {
	Enabled bool `json:"enabled"`
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
}

type ValidateResetTokenResponse struct {
	Valid bool `json:"valid"`
}

// MessageResponse represents Simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

type UpdateMemberRoleResult struct {
	Member MemberDTO `json:"member"`
}

type DataExportConfig struct {
	AllowedFormats []string `json:"allowedFormats"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	DefaultFormat string `json:"defaultFormat"`
	ExpiryHours int `json:"expiryHours"`
	IncludeSections []string `json:"includeSections"`
	MaxExportSize int64 `json:"maxExportSize"`
	MaxRequests int `json:"maxRequests"`
	RequestPeriod time.Duration `json:"requestPeriod"`
	AutoCleanup bool `json:"autoCleanup"`
	Enabled bool `json:"enabled"`
	StoragePath string `json:"storagePath"`
}

type ComplianceReport struct {
	Period string `json:"period"`
	ProfileId string `json:"profileId"`
	Status string `json:"status"`
	Summary interface{} `json:"summary"`
	FileSize int64 `json:"fileSize"`
	Format string `json:"format"`
	GeneratedBy string `json:"generatedBy"`
	Id string `json:"id"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FileUrl string `json:"fileUrl"`
}

type DeleteRoleTemplateResult struct {
	Success bool `json:"success"`
}

type ListReportsResponse struct {
	Reports []*interface{} `json:"reports"`
}

type OrganizationAutoSendConfig struct {
	Deleted bool `json:"deleted"`
	Invite bool `json:"invite"`
	Member_added bool `json:"member_added"`
	Member_left bool `json:"member_left"`
	Member_removed bool `json:"member_removed"`
	Role_changed bool `json:"role_changed"`
	Transfer bool `json:"transfer"`
}

type GetSessionStatsInput struct {
	AppId string `json:"appId"`
}

type LinkRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Password string `json:"password"`
}

type DeviceVerifyResponse struct {
	UserCode string `json:"userCode"`
	UserCodeFormatted string `json:"userCodeFormatted"`
	AuthorizeUrl string `json:"authorizeUrl"`
	ClientId string `json:"clientId"`
	ClientName string `json:"clientName"`
	LogoUri string `json:"logoUri"`
	Scopes []ScopeInfo `json:"scopes"`
}

type EmailServiceAdapter struct {
}

type RevisionHandler struct {
}

type UpdateRequest struct {
	Id string `json:"id"`
	Url *string `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Enabled *bool `json:"enabled,omitempty"`
}

type OverviewStatsDTO struct {
	OpenRate float64 `json:"openRate"`
	TotalBounced int64 `json:"totalBounced"`
	TotalClicked int64 `json:"totalClicked"`
	TotalDelivered int64 `json:"totalDelivered"`
	TotalFailed int64 `json:"totalFailed"`
	TotalSent int64 `json:"totalSent"`
	BounceRate float64 `json:"bounceRate"`
	ClickRate float64 `json:"clickRate"`
	DeliveryRate float64 `json:"deliveryRate"`
	TotalOpened int64 `json:"totalOpened"`
}

type EmailProviderDTO struct {
	Config interface{} `json:"config"`
	Enabled bool `json:"enabled"`
	FromEmail string `json:"fromEmail"`
	FromName string `json:"fromName"`
	Type string `json:"type"`
}

type MigrationResponse struct {
	Message string `json:"message"`
	MigrationId string `json:"migrationId"`
	StartedAt time.Time `json:"startedAt"`
	Status string `json:"status"`
}

type OAuthState struct {
	Created_at time.Time `json:"created_at"`
	Extra_scopes []string `json:"extra_scopes"`
	Link_user_id ID `json:"link_user_id"`
	Provider string `json:"provider"`
	Redirect_url string `json:"redirect_url"`
	User_organization_id ID `json:"user_organization_id"`
	App_id xid.ID `json:"app_id"`
}

type VerificationsResponse struct {
	Count int `json:"count"`
	Verifications interface{} `json:"verifications"`
}

type GetDashboardResponse struct {
	Metrics interface{} `json:"metrics"`
}

type SessionStatsDTO struct {
	TabletCount int `json:"tabletCount"`
	TotalSessions int64 `json:"totalSessions"`
	UniqueUsers int `json:"uniqueUsers"`
	ActiveCount int `json:"activeCount"`
	DesktopCount int `json:"desktopCount"`
	ExpiredCount int `json:"expiredCount"`
	ExpiringCount int `json:"expiringCount"`
	MobileCount int `json:"mobileCount"`
}

type CreateTeamHandlerRequest struct {
}

type ListClientsResponse struct {
	TotalPages int `json:"totalPages"`
	Clients []ClientSummary `json:"clients"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Total int `json:"total"`
}

type ConsentDecision struct {
}

type CookieConsentRequest struct {
	Analytics bool `json:"analytics"`
	BannerVersion string `json:"bannerVersion"`
	Essential bool `json:"essential"`
	Functional bool `json:"functional"`
	Marketing bool `json:"marketing"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
}

type ListDevicesResponse struct {
	Devices []*Device `json:"devices"`
}

type settingField struct {
}

type CheckResult struct {
	Status string `json:"status"`
	CheckType string `json:"checkType"`
	Error error `json:"error"`
	Evidence []string `json:"evidence"`
	Result interface{} `json:"result"`
	Score float64 `json:"score"`
}

type ClientAuthenticator struct {
}

type MemoryStateStore struct {
}

type GetSecretsOutput struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Secrets []SecretItem `json:"secrets"`
	Total int64 `json:"total"`
	TotalPages int `json:"totalPages"`
}

type TeamDTO struct {
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
	MemberCount int64 `json:"memberCount"`
}

type BlockUserRequest struct {
	Reason string `json:"reason"`
}

type ChangePasswordRequest struct {
	NewPassword string `json:"newPassword"`
	OldPassword string `json:"oldPassword"`
}

type StepUpStatusResponse struct {
	Status string `json:"status"`
}

type mockSentNotification struct {
}

type ComplianceTrainingResponse struct {
	Id string `json:"id"`
}

type ChallengeResponse struct {
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	SessionId xid.ID `json:"sessionId"`
}

type GetSecretsInput struct {
	AppId string `json:"appId"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Search string `json:"search"`
}

type CreatePolicyRequest struct {
	Priority int `json:"priority"`
	ResourceType string `json:"resourceType"`
	Actions []string `json:"actions"`
	Description string `json:"description"`
	Enabled bool `json:"enabled"`
	Expression string `json:"expression"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
}

type DevicesResponse struct {
	Count int `json:"count"`
	Devices interface{} `json:"devices"`
}

type CreateOrganizationResult struct {
	Organization OrganizationDetailDTO `json:"organization"`
}

type DeleteProfileResponse struct {
	Status string `json:"status"`
}

type ActionDataDTO struct {
	Label string `json:"label"`
	Order int `json:"order"`
	RequireAdmin bool `json:"requireAdmin"`
	Style string `json:"style"`
	Action string `json:"action"`
	Icon string `json:"icon"`
	Id string `json:"id"`
}

type SessionStats struct {
}

type ValidateContentTypeRequest struct {
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

type BackupCodeFactorAdapter struct {
}

type ConfirmEmailChangeResponse struct {
	Message string `json:"message"`
}

type CreateEvidence_req struct {
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	ControlId string `json:"controlId"`
}

type SetupSecurityQuestionRequest struct {
	Answer string `json:"answer"`
	CustomText string `json:"customText"`
	QuestionId int `json:"questionId"`
}

type DeleteTemplateResponse struct {
	Status string `json:"status"`
}

type RunCheckResponse struct {
	Id string `json:"id"`
}

type AdminBlockUserResponse struct {
	Status interface{} `json:"status"`
}

type GetABTestResultsRequest struct {
}

type DeleteTemplateInput struct {
	TemplateId string `json:"templateId"`
}

type GetMigrationStatusResponse struct {
	HasMigratedPolicies bool `json:"hasMigratedPolicies"`
	LastMigrationAt string `json:"lastMigrationAt"`
	MigratedCount int `json:"migratedCount"`
	PendingRbacPolicies int `json:"pendingRbacPolicies"`
}

type ConnectionResponse struct {
	Connection SocialAccount `json:"connection"`
}

type VerificationResponse struct {
	ExpiresAt Time `json:"expiresAt"`
	FactorsRemaining int `json:"factorsRemaining"`
	SessionComplete bool `json:"sessionComplete"`
	Success bool `json:"success"`
	Token string `json:"token"`
}

type UpdateProfileResponse struct {
	Id string `json:"id"`
}

type SendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type RevokeConsentResponse struct {
	Status string `json:"status"`
}

type ResourcesListResponse struct {
	Resources []*ResourceResponse `json:"resources"`
	TotalCount int `json:"totalCount"`
}

type RequestTrustedContactVerificationResponse struct {
	Message string `json:"message"`
	NotifiedAt time.Time `json:"notifiedAt"`
	ContactId xid.ID `json:"contactId"`
	ContactName string `json:"contactName"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type RecoverySessionInfo struct {
	TotalSteps int `json:"totalSteps"`
	UserEmail string `json:"userEmail"`
	CompletedAt Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	CurrentStep int `json:"currentStep"`
	ExpiresAt time.Time `json:"expiresAt"`
	Method RecoveryMethod `json:"method"`
	RequiresReview bool `json:"requiresReview"`
	UserId xid.ID `json:"userId"`
	Id xid.ID `json:"id"`
	RiskScore float64 `json:"riskScore"`
	Status RecoveryStatus `json:"status"`
}

type DisableRequest struct {
	User_id string `json:"user_id"`
}

type WebAuthnConfig struct {
	Rp_display_name string `json:"rp_display_name"`
	Rp_id string `json:"rp_id"`
	Rp_origins []string `json:"rp_origins"`
	Timeout int `json:"timeout"`
	Attestation_preference string `json:"attestation_preference"`
	Authenticator_selection interface{} `json:"authenticator_selection"`
	Enabled bool `json:"enabled"`
}

type SetActiveResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type GetConsentResponse struct {
	Id string `json:"id"`
}

type CreateConsentPolicyResponse struct {
	Id string `json:"id"`
}

type GetMigrationStatusRequest struct {
}

type UserServiceAdapter struct {
}

type BeginRegisterRequest struct {
	Name string `json:"name"`
	RequireResidentKey bool `json:"requireResidentKey"`
	UserId string `json:"userId"`
	UserVerification string `json:"userVerification"`
	AuthenticatorType string `json:"authenticatorType"`
}

// Webhook represents Webhook configuration
type Webhook struct {
	OrganizationId string `json:"organizationId"`
	Url string `json:"url"`
	Events []string `json:"events"`
	Secret string `json:"secret"`
	Enabled bool `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
}

type RequirementsResponse struct {
	Count int `json:"count"`
	Requirements interface{} `json:"requirements"`
}

type SignUpResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

type AdminGetUserVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type OrganizationStatsDTO struct {
	TotalMembers int64 `json:"totalMembers"`
	TotalOrganizations int64 `json:"totalOrganizations"`
	TotalTeams int64 `json:"totalTeams"`
}

type ListPasskeysResponse struct {
	Passkeys []PasskeyInfo `json:"passkeys"`
	Count int `json:"count"`
}

type CallbackResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
	User User `json:"user"`
}

type DiscoverProviderRequest struct {
	Email string `json:"email"`
}

type UpdateConsentResponse struct {
	Id string `json:"id"`
}

type RevokeAllUserSessionsResult struct {
	Message string `json:"message"`
	RevokedCount int `json:"revokedCount"`
	Success bool `json:"success"`
}

type FinishRegisterRequest struct {
	Name string `json:"name"`
	Response interface{} `json:"response"`
	UserId string `json:"userId"`
}

type IDVerificationResponse struct {
	Verification interface{} `json:"verification"`
}

type DeviceAuthorizationDecisionRequest struct {
	Action string `json:"action"`
	User_code string `json:"user_code"`
}

type MockSocialAccountRepository struct {
}

type BatchEvaluationResult struct {
	Policies []string `json:"policies"`
	ResourceId string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Action string `json:"action"`
	Allowed bool `json:"allowed"`
	Error string `json:"error"`
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
	Index int `json:"index"`
}

type ListPasskeysRequest struct {
}

type DeleteTeamInput struct {
	AppId string `json:"appId"`
	OrgId string `json:"orgId"`
	TeamId string `json:"teamId"`
}

type CreateEvidenceResponse struct {
	Id string `json:"id"`
}

type UpdateClientRequest struct {
	Require_pkce *bool `json:"require_pkce"`
	Response_types []string `json:"response_types"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Tos_uri string `json:"tos_uri"`
	Trusted_client *bool `json:"trusted_client"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Name string `json:"name"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Redirect_uris []string `json:"redirect_uris"`
	Require_consent *bool `json:"require_consent"`
	Allowed_scopes []string `json:"allowed_scopes"`
}

type ComplianceViolation struct {
	Description string `json:"description"`
	Id string `json:"id"`
	ResolvedBy string `json:"resolvedBy"`
	Severity string `json:"severity"`
	UserId string `json:"userId"`
	ViolationType string `json:"violationType"`
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Metadata interface{} `json:"metadata"`
	ProfileId string `json:"profileId"`
	ResolvedAt Time `json:"resolvedAt"`
	Status string `json:"status"`
}

type EvaluationContext struct {
}

type MigrateRBACRequest struct {
	KeepRbacPolicies bool `json:"keepRbacPolicies"`
	NamespaceId string `json:"namespaceId"`
	ValidateEquivalence bool `json:"validateEquivalence"`
	DryRun bool `json:"dryRun"`
}

type DataDeletionRequest struct {
	CompletedAt Time `json:"completedAt"`
	ErrorMessage string `json:"errorMessage"`
	OrganizationId string `json:"organizationId"`
	RejectedAt Time `json:"rejectedAt"`
	RetentionExempt bool `json:"retentionExempt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ApprovedAt Time `json:"approvedAt"`
	ApprovedBy string `json:"approvedBy"`
	CreatedAt time.Time `json:"createdAt"`
	RequestReason string `json:"requestReason"`
	Status string `json:"status"`
	ArchivePath string `json:"archivePath"`
	DeleteSections []string `json:"deleteSections"`
	IpAddress string `json:"ipAddress"`
	UserId string `json:"userId"`
	ExemptionReason string `json:"exemptionReason"`
	Id xid.ID `json:"id"`
}

type ProviderSession struct {
}

type CheckSubResult struct {
}

type ApproveDeletionRequestResponse struct {
	Status string `json:"status"`
}

type RevokeAllUserSessionsInput struct {
	AppId string `json:"appId"`
	UserId string `json:"userId"`
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

type GetRolesRequest struct {
}

type GetExtensionDataResult struct {
	Tabs []TabDataDTO `json:"tabs"`
	Widgets []WidgetDataDTO `json:"widgets"`
	Actions []ActionDataDTO `json:"actions"`
	QuickLinks []QuickLinkDataDTO `json:"quickLinks"`
}

type RemoveTrustedContactResponse struct {
	Status string `json:"status"`
}

type GetProvidersResult struct {
	Providers ProvidersConfigDTO `json:"providers"`
}

type SetUserRoleRequestDTO struct {
	Role string `json:"role"`
}

type RollbackRequest struct {
	Reason string `json:"reason"`
}

type VerificationResult struct {
}

type AuditLogResponse struct {
	Entries []*AuditLogEntry `json:"entries"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
}

type ConsentService struct {
}

type StepUpPolicy struct {
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
	User_id string `json:"user_id"`
	Description string `json:"description"`
	Priority int `json:"priority"`
	Rules interface{} `json:"rules"`
	Updated_at time.Time `json:"updated_at"`
	Created_at time.Time `json:"created_at"`
	Enabled bool `json:"enabled"`
}

type MFABypassResponse struct {
	Id xid.ID `json:"id"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type CreateSecretOutput struct {
	Secret SecretItem `json:"secret"`
}

type ListChecksResponse struct {
	Checks []*interface{} `json:"checks"`
}

type VideoVerificationConfig struct {
	LivenessThreshold float64 `json:"livenessThreshold"`
	Provider string `json:"provider"`
	RecordSessions bool `json:"recordSessions"`
	RequireScheduling bool `json:"requireScheduling"`
	MinScheduleAdvance time.Duration `json:"minScheduleAdvance"`
	RecordingRetention time.Duration `json:"recordingRetention"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireLivenessCheck bool `json:"requireLivenessCheck"`
	SessionDuration time.Duration `json:"sessionDuration"`
	Enabled bool `json:"enabled"`
}

type BackupAuthStatusResponse struct {
	Status string `json:"status"`
}

type OrganizationDetailDTO struct {
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	Logo string `json:"logo"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type GetRoleTemplateResult struct {
	Template RoleTemplateDTO `json:"template"`
}

type RefreshSessionRequest struct {
	RefreshToken string `json:"refreshToken"`
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

type mockRepository struct {
}

type ListTemplatesResponse struct {
	Templates []*interface{} `json:"templates"`
	Total int `json:"total"`
}

type TemplateAnalyticsDTO struct {
	TotalDelivered int64 `json:"totalDelivered"`
	TotalOpened int64 `json:"totalOpened"`
	TotalSent int64 `json:"totalSent"`
	DeliveryRate float64 `json:"deliveryRate"`
	TemplateId string `json:"templateId"`
	TemplateName string `json:"templateName"`
	ClickRate float64 `json:"clickRate"`
	OpenRate float64 `json:"openRate"`
	TotalClicked int64 `json:"totalClicked"`
}

type NotificationsConfig struct {
	Enabled bool `json:"enabled"`
	FailedChecks bool `json:"failedChecks"`
	NotifyComplianceContact bool `json:"notifyComplianceContact"`
	NotifyOwners bool `json:"notifyOwners"`
	Violations bool `json:"violations"`
	AuditReminders bool `json:"auditReminders"`
	Channels NotificationChannels `json:"channels"`
}

type TrustedDevicesConfig struct {
	Default_expiry_days int `json:"default_expiry_days"`
	Enabled bool `json:"enabled"`
	Max_devices_per_user int `json:"max_devices_per_user"`
	Max_expiry_days int `json:"max_expiry_days"`
}

type ListResponse struct {
	Webhooks []*Webhook `json:"webhooks"`
}

type sessionStats struct {
}

type CreateTeamRequest struct {
	Description string `json:"description"`
	Name string `json:"name"`
}

type ResendRequest struct {
	Email string `json:"email"`
}

type ContextRule struct {
	Security_level SecurityLevel `json:"security_level"`
	Condition string `json:"condition"`
	Description string `json:"description"`
	Name string `json:"name"`
	Org_id string `json:"org_id"`
}

type StateStore struct {
}

type RenderTemplateRequest struct {
}

type SessionDTO struct {
	BrowserVersion string `json:"browserVersion"`
	DeviceInfo string `json:"deviceInfo"`
	IpAddress string `json:"ipAddress"`
	Os string `json:"os"`
	OsVersion string `json:"osVersion"`
	ExpiresIn string `json:"expiresIn"`
	IsActive bool `json:"isActive"`
	IsExpiring bool `json:"isExpiring"`
	Status string `json:"status"`
	UserAgent string `json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
	DeviceType string `json:"deviceType"`
	ExpiresAt time.Time `json:"expiresAt"`
	UserEmail string `json:"userEmail"`
	UserId string `json:"userId"`
	Browser string `json:"browser"`
	Id string `json:"id"`
	LastUsed string `json:"lastUsed"`
}

type ContentEntryHandler struct {
}

type RiskAssessment struct {
	Score float64 `json:"score"`
	Factors []string `json:"factors"`
	Level RiskLevel `json:"level"`
	Metadata interface{} `json:"metadata"`
	Recommended []FactorType `json:"recommended"`
}

type ClientUpdateRequest struct {
	Name string `json:"name"`
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Require_consent *bool `json:"require_consent"`
	Require_pkce *bool `json:"require_pkce"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Trusted_client *bool `json:"trusted_client"`
	Allowed_scopes []string `json:"allowed_scopes"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
	Logo_uri string `json:"logo_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Response_types []string `json:"response_types"`
	Tos_uri string `json:"tos_uri"`
}

type VerifyChallengeRequest struct {
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data interface{} `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
}

type QuickLinkDataDTO struct {
	Url string `json:"url"`
	Description string `json:"description"`
	Icon string `json:"icon"`
	Id string `json:"id"`
	Order int `json:"order"`
	RequireAdmin bool `json:"requireAdmin"`
	Title string `json:"title"`
}

type RefreshResponse struct {
	Session Session `json:"session"`
	Token string `json:"token"`
}

type CreateProfileFromTemplateResponse struct {
	Id string `json:"id"`
}

type DeviceVerifyRequest struct {
	User_code string `json:"user_code"`
}

type CreateNamespaceRequest struct {
	InheritPlatform bool `json:"inheritPlatform"`
	Name string `json:"name"`
	TemplateId string `json:"templateId"`
	Description string `json:"description"`
}

type MockEmailService struct {
}

type GetByPathRequest struct {
}

type UpdateSecretRequest struct {
	Tags []string `json:"tags"`
	Value interface{} `json:"value"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
}

type UnblockUserRequest struct {
}

type UpdateUserRequest struct {
	Name *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

type GetImpersonationRequest struct {
}

type ConsentNotificationsConfig struct {
	Enabled bool `json:"enabled"`
	NotifyDeletionApproved bool `json:"notifyDeletionApproved"`
	NotifyExportReady bool `json:"notifyExportReady"`
	NotifyOnGrant bool `json:"notifyOnGrant"`
	Channels []string `json:"channels"`
	NotifyDeletionComplete bool `json:"notifyDeletionComplete"`
	NotifyDpoEmail string `json:"notifyDpoEmail"`
	NotifyOnExpiry bool `json:"notifyOnExpiry"`
	NotifyOnRevoke bool `json:"notifyOnRevoke"`
}

type RevokeTrustedDeviceRequest struct {
}

type SAMLLoginRequest struct {
	RelayState string `json:"relayState"`
}

type InitiateChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata interface{} `json:"metadata"`
}

type GetMembersInput struct {
	Search string `json:"search"`
	AppId string `json:"appId"`
	Limit int `json:"limit"`
	OrgId string `json:"orgId"`
	Page int `json:"page"`
}

type UpdateSettingsInput struct {
	Auth *AuthAutoSendDTO `json:"auth"`
	Organization *OrganizationAutoSendDTO `json:"organization"`
	Session *SessionAutoSendDTO `json:"session"`
	Account *AccountAutoSendDTO `json:"account"`
	AppName string `json:"appName"`
}

type GenerateReport_req struct {
	Format string `json:"format"`
	Period string `json:"period"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
}

type CompliancePolicy struct {
	Standard ComplianceStandard `json:"standard"`
	Status string `json:"status"`
	AppId string `json:"appId"`
	ApprovedAt Time `json:"approvedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	ProfileId string `json:"profileId"`
	PolicyType string `json:"policyType"`
	Title string `json:"title"`
	Version string `json:"version"`
	ApprovedBy string `json:"approvedBy"`
	Content string `json:"content"`
	Metadata interface{} `json:"metadata"`
	ReviewDate time.Time `json:"reviewDate"`
	UpdatedAt time.Time `json:"updatedAt"`
	EffectiveDate time.Time `json:"effectiveDate"`
}

type JWTService struct {
}

type ListRecoverySessionsResponse struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Sessions []RecoverySessionInfo `json:"sessions"`
	TotalCount int `json:"totalCount"`
}

type DocumentVerificationResult struct {
}

type TwoFABackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type BanUserRequestDTO struct {
	Expires_at Time `json:"expires_at"`
	Reason string `json:"reason"`
}

type RevokeResponse struct {
	Status string `json:"status"`
	RevokedCount int `json:"revokedCount"`
}

type StartVideoSessionResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	SessionUrl string `json:"sessionUrl"`
	StartedAt time.Time `json:"startedAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type ProviderDetailResponse struct {
	AttributeMapping interface{} `json:"attributeMapping"`
	CreatedAt string `json:"createdAt"`
	Domain string `json:"domain"`
	OidcIssuer string `json:"oidcIssuer"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
	ProviderId string `json:"providerId"`
	SamlIssuer string `json:"samlIssuer"`
	UpdatedAt string `json:"updatedAt"`
	HasSamlCert bool `json:"hasSamlCert"`
	OidcClientID string `json:"oidcClientID"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	Type string `json:"type"`
}

type RevokeAllResponse struct {
	RevokedCount int `json:"revokedCount"`
	Status string `json:"status"`
}

type DeleteRequest struct {
	Id string `json:"id"`
}

type IntrospectTokenResponse struct {
	Username string `json:"username"`
	Active bool `json:"active"`
	Aud []string `json:"aud"`
	Client_id string `json:"client_id"`
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
	Iss string `json:"iss"`
	Jti string `json:"jti"`
	Scope string `json:"scope"`
	Nbf int64 `json:"nbf"`
	Sub string `json:"sub"`
	Token_type string `json:"token_type"`
}

type CompleteRecoveryResponse struct {
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
	Status RecoveryStatus `json:"status"`
	Token string `json:"token"`
}

type ProviderListResponse struct {
	Providers []ProviderInfo `json:"providers"`
	Total int `json:"total"`
}

type CreateRequest struct {
	Url string `json:"url"`
	Events []string `json:"events"`
	Secret *string `json:"secret,omitempty"`
}

type RequestDataDeletionRequest struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type NotificationTemplateListResponse struct {
	Templates []*interface{} `json:"templates"`
	Total int `json:"total"`
}

type UpdateProvidersResult struct {
	Message string `json:"message"`
	Providers ProvidersConfigDTO `json:"providers"`
	Success bool `json:"success"`
}

type ComplianceStatusResponse struct {
	Status string `json:"status"`
}

type SendOTPRequest struct {
	User_id string `json:"user_id"`
}

type StepUpEvaluationResponse struct {
	Reason string `json:"reason"`
	Required bool `json:"required"`
}

type WebAuthnFactorAdapter struct {
}

type GetRoleTemplatesResult struct {
	Templates []RoleTemplateDTO `json:"templates"`
}

type ListSessionsRequest struct {
	CreatedFrom *string `json:"createdFrom"`
	CreatedTo *string `json:"createdTo"`
	Offset int `json:"offset"`
	SortOrder *string `json:"sortOrder"`
	Active *bool `json:"active"`
	IpAddress *string `json:"ipAddress"`
	Limit int `json:"limit"`
	SortBy *string `json:"sortBy"`
	UserAgent *string `json:"userAgent"`
}

type SettingsDTO struct {
	EnableDeviceTracking bool `json:"enableDeviceTracking"`
	MaxSessionsPerUser int `json:"maxSessionsPerUser"`
	SessionExpiryHours int `json:"sessionExpiryHours"`
	AllowCrossPlatform bool `json:"allowCrossPlatform"`
}

type DeviceInfo struct {
}

type ListPoliciesFilter struct {
	PolicyType *string `json:"policyType"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
}

type DocumentVerificationRequest struct {
}

type UpdateTemplateInput struct {
	Metadata interface{} `json:"metadata"`
	Name *string `json:"name"`
	Subject *string `json:"subject"`
	TemplateId string `json:"templateId"`
	Variables []string `json:"variables"`
	Active *bool `json:"active"`
	Body *string `json:"body"`
}

type RateLimiter struct {
}

type GenerateConsentReportResponse struct {
	Id string `json:"id"`
}

type RequestReverificationRequest struct {
	Reason string `json:"reason"`
}

type ComplianceTemplate struct {
	AuditFrequencyDays int `json:"auditFrequencyDays"`
	DataResidency string `json:"dataResidency"`
	Description string `json:"description"`
	Name string `json:"name"`
	RequiredPolicies []string `json:"requiredPolicies"`
	RequiredTraining []string `json:"requiredTraining"`
	SessionMaxAge int `json:"sessionMaxAge"`
	Standard ComplianceStandard `json:"standard"`
	MfaRequired bool `json:"mfaRequired"`
	PasswordMinLength int `json:"passwordMinLength"`
	RetentionDays int `json:"retentionDays"`
}

type ComplianceReportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type AuditLog struct {
}

type ComplianceEvidencesResponse struct {
	Evidence []*interface{} `json:"evidence"`
}

type AppHandler struct {
}

type Factor struct {
	UserId xid.ID `json:"userId"`
	Id xid.ID `json:"id"`
	LastUsedAt Time `json:"lastUsedAt"`
	Metadata interface{} `json:"metadata"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
	VerifiedAt Time `json:"verifiedAt"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt Time `json:"expiresAt"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ListVerificationsResponse struct {
	Verifications []*interface{} `json:"verifications"`
}

type deviceFlowServiceAdapter struct {
}

type ActionResponse struct {
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
	Name string `json:"name"`
	NamespaceId string `json:"namespaceId"`
}

type UpdateAppRequest struct {
}

type ClientRegistrationRequest struct {
	Policy_uri string `json:"policy_uri"`
	Post_logout_redirect_uris []string `json:"post_logout_redirect_uris"`
	Scope string `json:"scope"`
	Application_type string `json:"application_type"`
	Logo_uri string `json:"logo_uri"`
	Redirect_uris []string `json:"redirect_uris"`
	Token_endpoint_auth_method string `json:"token_endpoint_auth_method"`
	Response_types []string `json:"response_types"`
	Tos_uri string `json:"tos_uri"`
	Client_name string `json:"client_name"`
	Require_consent bool `json:"require_consent"`
	Require_pkce bool `json:"require_pkce"`
	Trusted_client bool `json:"trusted_client"`
	Contacts []string `json:"contacts"`
	Grant_types []string `json:"grant_types"`
}

type BackupAuthDocumentResponse struct {
	Id string `json:"id"`
}

type RemoveTrustedContactRequest struct {
	ContactId xid.ID `json:"contactId"`
}

type RateLimit struct {
	Max_requests int `json:"max_requests"`
	Window time.Duration `json:"window"`
}

type UserInfoResponse struct {
	Picture string `json:"picture"`
	Sub string `json:"sub"`
	Zoneinfo string `json:"zoneinfo"`
	Gender string `json:"gender"`
	Middle_name string `json:"middle_name"`
	Phone_number_verified bool `json:"phone_number_verified"`
	Family_name string `json:"family_name"`
	Given_name string `json:"given_name"`
	Nickname string `json:"nickname"`
	Preferred_username string `json:"preferred_username"`
	Profile string `json:"profile"`
	Website string `json:"website"`
	Email string `json:"email"`
	Locale string `json:"locale"`
	Name string `json:"name"`
	Updated_at int64 `json:"updated_at"`
	Birthdate string `json:"birthdate"`
	Email_verified bool `json:"email_verified"`
	Phone_number string `json:"phone_number"`
}

type JWKSService struct {
}

type CancelInvitationInput struct {
	OrgId string `json:"orgId"`
	AppId string `json:"appId"`
	InviteId string `json:"inviteId"`
}

type ActionsListResponse struct {
	Actions []*ActionResponse `json:"actions"`
	TotalCount int `json:"totalCount"`
}

type ComplianceTraining struct {
	AppId string `json:"appId"`
	CreatedAt time.Time `json:"createdAt"`
	Id string `json:"id"`
	Metadata interface{} `json:"metadata"`
	Standard ComplianceStandard `json:"standard"`
	UserId string `json:"userId"`
	CompletedAt Time `json:"completedAt"`
	ExpiresAt Time `json:"expiresAt"`
	ProfileId string `json:"profileId"`
	Score int `json:"score"`
	Status string `json:"status"`
	TrainingType string `json:"trainingType"`
}

type CreateUserRequest struct {
	Email_verified bool `json:"email_verified"`
	Metadata interface{} `json:"metadata"`
	Password string `json:"password"`
	Role string `json:"role"`
	User_organization_id ID `json:"user_organization_id"`
	Username string `json:"username"`
	App_id xid.ID `json:"app_id"`
	Name string `json:"name"`
	Email string `json:"email"`
}

type SecretItem struct {
	UpdatedAt string `json:"updatedAt"`
	ValueType string `json:"valueType"`
	Key string `json:"key"`
	Version int `json:"version"`
	CreatedAt string `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
	Path string `json:"path"`
	Tags []string `json:"tags"`
}

type GetSessionResult struct {
	Session SessionDetailDTO `json:"session"`
}

type Service struct {
}

type ArchiveEntryRequest struct {
}

type GetTreeRequest struct {
}

type GetCookieConsentResponse struct {
	Preferences interface{} `json:"preferences"`
}

type AsyncConfig struct {
	Retry_backoff []string `json:"retry_backoff"`
	Retry_enabled bool `json:"retry_enabled"`
	Worker_pool_size int `json:"worker_pool_size"`
	Enabled bool `json:"enabled"`
	Max_retries int `json:"max_retries"`
	Persist_failures bool `json:"persist_failures"`
	Queue_size int `json:"queue_size"`
}

type CreateTemplateInput struct {
	Subject string `json:"subject"`
	TemplateKey string `json:"templateKey"`
	Type string `json:"type"`
	Variables []string `json:"variables"`
	Body string `json:"body"`
	Language string `json:"language"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
}

type NotificationErrorResponse struct {
	Error string `json:"error"`
}

type ConsentManager struct {
}

type ConsentStats struct {
	AverageLifetime int `json:"averageLifetime"`
	ExpiredCount int `json:"expiredCount"`
	GrantRate float64 `json:"grantRate"`
	GrantedCount int `json:"grantedCount"`
	RevokedCount int `json:"revokedCount"`
	TotalConsents int `json:"totalConsents"`
	Type string `json:"type"`
}

type GetOrganizationBySlugRequest struct {
}

type UpdateResponse struct {
	Webhook Webhook `json:"webhook"`
}

type SetupSecurityQuestionsRequest struct {
	Questions []SetupSecurityQuestionRequest `json:"questions"`
}

type ImpersonationContext struct {
	Impersonation_id ID `json:"impersonation_id"`
	Impersonator_id ID `json:"impersonator_id"`
	Indicator_message string `json:"indicator_message"`
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id ID `json:"target_user_id"`
}

type CancelRecoveryRequest struct {
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type BackupAuthCodesResponse struct {
	Codes []string `json:"codes"`
}

type ResourceRule struct {
	Action string `json:"action"`
	Description string `json:"description"`
	Org_id string `json:"org_id"`
	Resource_type string `json:"resource_type"`
	Security_level SecurityLevel `json:"security_level"`
	Sensitivity string `json:"sensitivity"`
}

type DeleteOrganizationInput struct {
	AppId string `json:"appId"`
	OrgId string `json:"orgId"`
}

type StripeIdentityProvider struct {
}

type CreateProfileResponse struct {
	Id string `json:"id"`
}

type BeginLoginRequest struct {
	UserVerification string `json:"userVerification"`
	UserId string `json:"userId"`
}

type BulkRequest struct {
	Ids []string `json:"ids"`
}

type NoOpDocumentProvider struct {
}

type ProviderConfigResponse struct {
	AppId string `json:"appId"`
	Message string `json:"message"`
	Provider string `json:"provider"`
}

type InitiateChallengeResponse struct {
	SessionId xid.ID `json:"sessionId"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
}

type NotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type MockRepository struct {
}

type ContinueRecoveryRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
}

type GetUserSessionsInput struct {
	AppId string `json:"appId"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	UserId string `json:"userId"`
}

type FactorInfo struct {
	FactorId xid.ID `json:"factorId"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	Type FactorType `json:"type"`
}

type OrgDetailStatsDTO struct {
	InvitationCount int64 `json:"invitationCount"`
	MemberCount int64 `json:"memberCount"`
	TeamCount int64 `json:"teamCount"`
}

type TestPolicyRequest struct {
	Actions []string `json:"actions"`
	Expression string `json:"expression"`
	ResourceType string `json:"resourceType"`
	TestCases []TestCase `json:"testCases"`
}

type AcceptInvitationRequest struct {
}

type ListContentTypesRequest struct {
}

type DeleteEntryRequest struct {
}

type OnfidoProvider struct {
}

type RefreshSessionResponse struct {
	Session interface{} `json:"session"`
	AccessToken string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt string `json:"expiresAt"`
	RefreshExpiresAt string `json:"refreshExpiresAt"`
}

type PublishContentTypeRequest struct {
}

type RequestDataExportResponse struct {
	Status string `json:"status"`
	Id string `json:"id"`
}

type SendRequest struct {
	Email string `json:"email"`
}

type RenderTemplate_req struct {
	Variables interface{} `json:"variables"`
	Template string `json:"template"`
}

type RateLimitConfig struct {
	Enabled bool `json:"enabled"`
	Window time.Duration `json:"window"`
}

type VideoSessionResult struct {
}

type ListSecretsRequest struct {
}

type RequestEmailChangeResponse struct {
	Message string `json:"message"`
}

type AdminUnblockUserResponse struct {
	Status interface{} `json:"status"`
}

type ListTrainingFilter struct {
	UserId *string `json:"userId"`
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	TrainingType *string `json:"trainingType"`
}

type NotificationHistoryDTO struct {
	AppId string `json:"appId"`
	Body string `json:"body"`
	ProviderId string `json:"providerId"`
	Recipient string `json:"recipient"`
	Status string `json:"status"`
	Type string `json:"type"`
	Error string `json:"error"`
	Metadata interface{} `json:"metadata"`
	Subject string `json:"subject"`
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
	SentAt *string `json:"sentAt"`
	TemplateId *string `json:"templateId"`
	DeliveredAt *string `json:"deliveredAt"`
	UpdatedAt string `json:"updatedAt"`
}

type SessionDetailDTO struct {
	ExpiresAtFormatted string `json:"expiresAtFormatted"`
	Id string `json:"id"`
	LastRefreshedFormatted string `json:"lastRefreshedFormatted"`
	OrganizationId string `json:"organizationId"`
	Os string `json:"os"`
	CreatedAtFormatted string `json:"createdAtFormatted"`
	DeviceType string `json:"deviceType"`
	IsActive bool `json:"isActive"`
	UserAgent string `json:"userAgent"`
	UserEmail string `json:"userEmail"`
	Browser string `json:"browser"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedAtFormatted string `json:"updatedAtFormatted"`
	UserId string `json:"userId"`
	EnvironmentId string `json:"environmentId"`
	ExpiresAt time.Time `json:"expiresAt"`
	IpAddress string `json:"ipAddress"`
	IsExpiring bool `json:"isExpiring"`
	LastRefreshedAt Time `json:"lastRefreshedAt"`
	OsVersion string `json:"osVersion"`
	Status string `json:"status"`
	AppId string `json:"appId"`
	BrowserVersion string `json:"browserVersion"`
	DeviceInfo string `json:"deviceInfo"`
}

type ListTrustedContactsResponse struct {
	Contacts []TrustedContactInfo `json:"contacts"`
	Count int `json:"count"`
}

type UpdateRoleTemplateResult struct {
	Template RoleTemplateDTO `json:"template"`
}

type TokenResponse struct {
	Access_token string `json:"access_token"`
	Expires_in int `json:"expires_in"`
	Id_token string `json:"id_token"`
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
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

type ResendNotificationRequest struct {
}

type AuthAutoSendConfig struct {
	Email_otp bool `json:"email_otp"`
	Magic_link bool `json:"magic_link"`
	Mfa_code bool `json:"mfa_code"`
	Password_reset bool `json:"password_reset"`
	Verification_email bool `json:"verification_email"`
	Welcome bool `json:"welcome"`
}

type SaveBuilderTemplateResult struct {
	Success bool `json:"success"`
	TemplateId string `json:"templateId"`
	Message string `json:"message"`
}

type SessionAutoSendConfig struct {
	All_revoked bool `json:"all_revoked"`
	Device_removed bool `json:"device_removed"`
	New_device bool `json:"new_device"`
	New_location bool `json:"new_location"`
	Suspicious_login bool `json:"suspicious_login"`
}

type CheckRegistry struct {
}

type GetRoleTemplatesInput struct {
	AppId string `json:"appId"`
}

type MigrateAllRequest struct {
	DryRun bool `json:"dryRun"`
	PreserveOriginal bool `json:"preserveOriginal"`
}

type NotificationListResponse struct {
	Notifications []*interface{} `json:"notifications"`
	Total int `json:"total"`
}

type GetSessionInput struct {
	SessionId string `json:"sessionId"`
	AppId string `json:"appId"`
}

type UpdatePasskeyResponse struct {
	UpdatedAt time.Time `json:"updatedAt"`
	Name string `json:"name"`
	PasskeyId string `json:"passkeyId"`
}

type CreateSecretInput struct {
	Description string `json:"description"`
	Path string `json:"path"`
	Tags []string `json:"tags"`
	Value interface{} `json:"value"`
	ValueType string `json:"valueType"`
	AppId string `json:"appId"`
}

type AdminAddProviderRequest struct {
	ClientSecret string `json:"clientSecret"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
	AppId xid.ID `json:"appId"`
	ClientId string `json:"clientId"`
}

type TestCaseResult struct {
	Passed bool `json:"passed"`
	Actual bool `json:"actual"`
	Error string `json:"error"`
	EvaluationTimeMs float64 `json:"evaluationTimeMs"`
	Expected bool `json:"expected"`
	Name string `json:"name"`
}

type GenerateReportRequest struct {
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
	Format string `json:"format"`
	Period string `json:"period"`
}

type VerificationRepository struct {
}

type AdminBypassRequest struct {
	Duration int `json:"duration"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
}

type VerifyTrustedContactRequest struct {
	Token string `json:"token"`
}

type HealthCheckResponse struct {
	Message string `json:"message"`
	ProvidersStatus interface{} `json:"providersStatus"`
	Version string `json:"version"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	Healthy bool `json:"healthy"`
}

type ScheduleVideoSessionRequest struct {
	TimeZone string `json:"timeZone"`
	ScheduledAt time.Time `json:"scheduledAt"`
	SessionId xid.ID `json:"sessionId"`
}

type TwoFASendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type RiskFactor struct {
}

type ListRequest struct {
	CreatedTo *string `json:"createdTo"`
	Limit int `json:"limit"`
	SortBy *string `json:"sortBy"`
	UserAgent *string `json:"userAgent"`
	CreatedFrom *string `json:"createdFrom"`
	IpAddress *string `json:"ipAddress"`
	Offset int `json:"offset"`
	SortOrder *string `json:"sortOrder"`
	Active *bool `json:"active"`
}

type UpdateFieldRequest struct {
}

type UpdateRecoveryConfigRequest struct {
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
}

type UpdateOrganizationInput struct {
	AppId string `json:"appId"`
	Logo string `json:"logo"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	OrgId string `json:"orgId"`
}

type SignOutResponse struct {
	Success bool `json:"success"`
}

type NotificationStatusResponse struct {
	Status string `json:"status"`
}

type KeyPair struct {
}

type UpdateRoleTemplateInput struct {
	AppId string `json:"appId"`
	Description string `json:"description"`
	Name string `json:"name"`
	Permissions []string `json:"permissions"`
	TemplateId string `json:"templateId"`
}

type GetVerificationResponse struct {
	Verification interface{} `json:"verification"`
}

type ComplianceTrainingsResponse struct {
	Training []*interface{} `json:"training"`
}

type VerifyImpersonationRequest struct {
}

type AutoSendConfig struct {
	Account AccountAutoSendConfig `json:"account"`
	Auth AuthAutoSendConfig `json:"auth"`
	Organization OrganizationAutoSendConfig `json:"organization"`
	Session SessionAutoSendConfig `json:"session"`
}

type DataExportRequestInput struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type OIDCLoginResponse struct {
	Nonce string `json:"nonce"`
	ProviderId string `json:"providerId"`
	State string `json:"state"`
	AuthUrl string `json:"authUrl"`
}

type ImpersonationSession struct {
}

type ConnectionsResponse struct {
	Connections SocialAccount `json:"connections"`
}

type StepUpRememberedDevice struct {
	Device_id string `json:"device_id"`
	Device_name string `json:"device_name"`
	Expires_at time.Time `json:"expires_at"`
	Ip string `json:"ip"`
	Last_used_at time.Time `json:"last_used_at"`
	Org_id string `json:"org_id"`
	Remembered_at time.Time `json:"remembered_at"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
	Security_level SecurityLevel `json:"security_level"`
	User_agent string `json:"user_agent"`
}

type VerifyTokenRequest struct {
	Audience []string `json:"audience"`
	Token string `json:"token"`
	TokenType string `json:"tokenType"`
}

type DeleteRoleTemplateInput struct {
	AppId string `json:"appId"`
	TemplateId string `json:"templateId"`
}

type RoleTemplateDTO struct {
	Id string `json:"id"`
	Name string `json:"name"`
	Permissions []string `json:"permissions"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
}

type ListProvidersResponse struct {
	Providers []string `json:"providers"`
}

type GetNotificationRequest struct {
}

type CreateAPIKeyResponse struct {
	Api_key APIKey `json:"api_key"`
	Message string `json:"message"`
}

type CreateEvidenceRequest struct {
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	ControlId string `json:"controlId"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
}

type TokenIntrospectionRequest struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Token string `json:"token"`
	Token_type_hint string `json:"token_type_hint"`
}

type AdminBlockUser_req struct {
	Reason string `json:"reason"`
}

type GetStatsResponse struct {
	ActiveSessions int `json:"activeSessions"`
	DeviceCount int `json:"deviceCount"`
	LocationCount int `json:"locationCount"`
	NewestSession *string `json:"newestSession"`
	OldestSession *string `json:"oldestSession"`
	TotalSessions int `json:"totalSessions"`
}

type GetViolationResponse struct {
	Id string `json:"id"`
}

type AuditEvent struct {
}

type stateEntry struct {
}

type SchemaValidator struct {
}

type CreateRoleTemplateResult struct {
	Template RoleTemplateDTO `json:"template"`
}

type GetEvidenceResponse struct {
	Id string `json:"id"`
}

type CreateJWTKeyRequest struct {
	Curve string `json:"curve"`
	ExpiresAt Time `json:"expiresAt"`
	IsPlatformKey bool `json:"isPlatformKey"`
	KeyType string `json:"keyType"`
	Metadata interface{} `json:"metadata"`
	Algorithm string `json:"algorithm"`
}

type CreateOrganizationHandlerRequest struct {
}

type RemoveMemberResult struct {
	Success bool `json:"success"`
}

type PolicyStats struct {
	PolicyId string `json:"policyId"`
	PolicyName string `json:"policyName"`
	AllowCount int64 `json:"allowCount"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	DenyCount int64 `json:"denyCount"`
	EvaluationCount int64 `json:"evaluationCount"`
}

type RegenerateCodesRequest struct {
	Count int `json:"count"`
	User_id string `json:"user_id"`
}

type OrganizationAutoSendDTO struct {
	Deleted bool `json:"deleted"`
	Invite bool `json:"invite"`
	MemberAdded bool `json:"memberAdded"`
	MemberLeft bool `json:"memberLeft"`
	MemberRemoved bool `json:"memberRemoved"`
	RoleChanged bool `json:"roleChanged"`
	Transfer bool `json:"transfer"`
}

type RequestTrustedContactVerificationRequest struct {
	ContactId xid.ID `json:"contactId"`
	SessionId xid.ID `json:"sessionId"`
}

type TwoFAErrorResponse struct {
	Error string `json:"error"`
}

type GetRecoveryStatsRequest struct {
	EndDate time.Time `json:"endDate"`
	OrganizationId string `json:"organizationId"`
	StartDate time.Time `json:"startDate"`
}

type AccountLockedResponse struct {
	Code string `json:"code"`
	Locked_minutes int `json:"locked_minutes"`
	Locked_until time.Time `json:"locked_until"`
	Message string `json:"message"`
}

type GetInvitationRequest struct {
}

type CancelRecoveryResponse struct {
	Status string `json:"status"`
}

type CreateProfileFromTemplateRequest struct {
	Standard ComplianceStandard `json:"standard"`
}

type RestoreEntryRequest struct {
}

type MemberHandler struct {
}

type BackupAuthStatsResponse struct {
	Stats interface{} `json:"stats"`
}

type ForgetDeviceResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type DeleteFactorRequest struct {
}

type CreateTeamResult struct {
	Team TeamDTO `json:"team"`
}

type ReportsConfig struct {
	RetentionDays int `json:"retentionDays"`
	Schedule string `json:"schedule"`
	StoragePath string `json:"storagePath"`
	Enabled bool `json:"enabled"`
	Formats []string `json:"formats"`
	IncludeEvidence bool `json:"includeEvidence"`
}

type RevealSecretInput struct {
	AppId string `json:"appId"`
	SecretId string `json:"secretId"`
}

type CreateTeamInput struct {
	AppId string `json:"appId"`
	Description string `json:"description"`
	Metadata interface{} `json:"metadata"`
	Name string `json:"name"`
	OrgId string `json:"orgId"`
}

type RegisterProviderRequest struct {
	SamlCert string `json:"samlCert"`
	SamlEntryPoint string `json:"samlEntryPoint"`
	SamlIssuer string `json:"samlIssuer"`
	Type string `json:"type"`
	AttributeMapping interface{} `json:"attributeMapping"`
	Domain string `json:"domain"`
	OidcClientID string `json:"oidcClientID"`
	OidcClientSecret string `json:"oidcClientSecret"`
	OidcIssuer string `json:"oidcIssuer"`
	ProviderId string `json:"providerId"`
	OidcRedirectURI string `json:"oidcRedirectURI"`
}

type VerificationSessionResponse struct {
	Session IdentityVerificationSession `json:"session"`
}

type GetTemplateResponse struct {
	Template interface{} `json:"template"`
}

type RevokeDeviceRequest struct {
	Fingerprint string `json:"fingerprint"`
}

type UpdateTemplateRequest struct {
}

type AssignRoleRequest struct {
	RoleID string `json:"roleID"`
}

type VerifyRecoveryCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type DeleteSecretInput struct {
	AppId string `json:"appId"`
	SecretId string `json:"secretId"`
}

type InvitationDTO struct {
	InvitedBy string `json:"invitedBy"`
	InviterName string `json:"inviterName"`
	Role string `json:"role"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	Email string `json:"email"`
	ExpiresAt time.Time `json:"expiresAt"`
	Id string `json:"id"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email"`
}

type AdminBlockUserRequest struct {
	Reason string `json:"reason"`
}

type KeyStats struct {
}

type GetContentTypeRequest struct {
}

type DeviceVerificationRequest struct {
	User_code string `json:"user_code"`
}

type SendVerificationCodeRequest struct {
	SessionId xid.ID `json:"sessionId"`
	Target string `json:"target"`
	Method RecoveryMethod `json:"method"`
}

type VerifyResponse struct {
	Success bool `json:"success"`
	Token string `json:"token"`
	User User `json:"user"`
	Session Session `json:"session"`
}

type TrackNotificationEvent_req struct {
	Event string `json:"event"`
	EventData *interface{} `json:"eventData,omitempty"`
	NotificationId string `json:"notificationId"`
	OrganizationId *string `json:"organizationId,omitempty"`
	TemplateId string `json:"templateId"`
}

type VerifyAPIKeyRequest struct {
	Key string `json:"key"`
}

type mockImpersonationRepository struct {
}

type ImpersonateUserRequestDTO struct {
	Duration time.Duration `json:"duration"`
}

type StepUpDevicesResponse struct {
	Devices interface{} `json:"devices"`
	Count int `json:"count"`
}

type OIDCLoginRequest struct {
	Nonce string `json:"nonce"`
	RedirectUri string `json:"redirectUri"`
	Scope string `json:"scope"`
	State string `json:"state"`
}

type ProvidersConfigDTO struct {
	SmsProvider SMSProviderDTO `json:"smsProvider"`
	EmailProvider EmailProviderDTO `json:"emailProvider"`
}

type ComplianceViolationsResponse struct {
	Violations []*interface{} `json:"violations"`
}

type BulkDeleteRequest struct {
	Ids []string `json:"ids"`
}

type DiscoveryResponse struct {
	Revocation_endpoint string `json:"revocation_endpoint"`
	Revocation_endpoint_auth_methods_supported []string `json:"revocation_endpoint_auth_methods_supported"`
	Scopes_supported []string `json:"scopes_supported"`
	Subject_types_supported []string `json:"subject_types_supported"`
	Token_endpoint string `json:"token_endpoint"`
	Claims_parameter_supported bool `json:"claims_parameter_supported"`
	Introspection_endpoint string `json:"introspection_endpoint"`
	Jwks_uri string `json:"jwks_uri"`
	Registration_endpoint string `json:"registration_endpoint"`
	Request_uri_parameter_supported bool `json:"request_uri_parameter_supported"`
	Response_modes_supported []string `json:"response_modes_supported"`
	Userinfo_endpoint string `json:"userinfo_endpoint"`
	Authorization_endpoint string `json:"authorization_endpoint"`
	Device_authorization_endpoint string `json:"device_authorization_endpoint"`
	Id_token_signing_alg_values_supported []string `json:"id_token_signing_alg_values_supported"`
	Request_parameter_supported bool `json:"request_parameter_supported"`
	Response_types_supported []string `json:"response_types_supported"`
	Token_endpoint_auth_methods_supported []string `json:"token_endpoint_auth_methods_supported"`
	Code_challenge_methods_supported []string `json:"code_challenge_methods_supported"`
	Grant_types_supported []string `json:"grant_types_supported"`
	Introspection_endpoint_auth_methods_supported []string `json:"introspection_endpoint_auth_methods_supported"`
	Issuer string `json:"issuer"`
	Claims_supported []string `json:"claims_supported"`
	Require_request_uri_registration bool `json:"require_request_uri_registration"`
}

type VerifyRequest2FA struct {
	User_id string `json:"user_id"`
	Code string `json:"code"`
	Device_id string `json:"device_id"`
	Remember_device bool `json:"remember_device"`
}

type RevokeOthersResponse struct {
	Status string `json:"status"`
	RevokedCount int `json:"revokedCount"`
}

type GetDataDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type ResetPasswordResponse struct {
	Message string `json:"message"`
}

type GetRecoveryStatsResponse struct {
	AdminReviewsRequired int `json:"adminReviewsRequired"`
	FailedRecoveries int `json:"failedRecoveries"`
	HighRiskAttempts int `json:"highRiskAttempts"`
	PendingRecoveries int `json:"pendingRecoveries"`
	SuccessfulRecoveries int `json:"successfulRecoveries"`
	TotalAttempts int `json:"totalAttempts"`
	AverageRiskScore float64 `json:"averageRiskScore"`
	MethodStats interface{} `json:"methodStats"`
	SuccessRate float64 `json:"successRate"`
}

type GetValueRequest struct {
}

type GetConsentPolicyResponse struct {
	Id string `json:"id"`
}

type RequestDataExportRequest struct {
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
}

type CreateTemplateVersion_req struct {
	Changes string `json:"changes"`
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

type UpdateMemberRoleInput struct {
	AppId string `json:"appId"`
	MemberId string `json:"memberId"`
	OrgId string `json:"orgId"`
	Role string `json:"role"`
}

type ResetTemplateRequest struct {
}

type LimitResult struct {
}

type RevealSecretOutput struct {
	Value interface{} `json:"value"`
	ValueType string `json:"valueType"`
}

type GetOrganizationRequest struct {
}

type MockSessionService struct {
}

type CreateConsentRequest struct {
	Purpose string `json:"purpose"`
	UserId string `json:"userId"`
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	ExpiresIn *int `json:"expiresIn"`
	Granted bool `json:"granted"`
	Metadata interface{} `json:"metadata"`
}

type ConsentSummary struct {
	ConsentsByType interface{} `json:"consentsByType"`
	ExpiredConsents int `json:"expiredConsents"`
	HasPendingDeletion bool `json:"hasPendingDeletion"`
	PendingRenewals int `json:"pendingRenewals"`
	GrantedConsents int `json:"grantedConsents"`
	HasPendingExport bool `json:"hasPendingExport"`
	LastConsentUpdate Time `json:"lastConsentUpdate"`
	OrganizationId string `json:"organizationId"`
	RevokedConsents int `json:"revokedConsents"`
	TotalConsents int `json:"totalConsents"`
	UserId string `json:"userId"`
}

type StepUpAttempt struct {
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Method VerificationMethod `json:"method"`
	Org_id string `json:"org_id"`
	Requirement_id string `json:"requirement_id"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Failure_reason string `json:"failure_reason"`
	Success bool `json:"success"`
}

type UpdatePolicy_req struct {
	Content *string `json:"content"`
	Status *string `json:"status"`
	Title *string `json:"title"`
	Version *string `json:"version"`
}

type IntrospectionService struct {
}

type SMSConfig struct {
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
}

type GetMembersResult struct {
	CanManage bool `json:"canManage"`
	Data []MemberDTO `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
}

type GetOrganizationInput struct {
	AppId string `json:"appId"`
	OrgId string `json:"orgId"`
}

type CreateEntryRequest struct {
}

type StepUpAuditLog struct {
	User_id string `json:"user_id"`
	Event_data interface{} `json:"event_data"`
	Event_type string `json:"event_type"`
	Ip string `json:"ip"`
	Severity string `json:"severity"`
	Created_at time.Time `json:"created_at"`
	Id string `json:"id"`
	Org_id string `json:"org_id"`
	User_agent string `json:"user_agent"`
}

type UpdateProfileRequest struct {
	Name *string `json:"name"`
	RetentionDays *int `json:"retentionDays"`
	Status *string `json:"status"`
	MfaRequired *bool `json:"mfaRequired"`
}

type UpdateProvidersInput struct {
	EmailProvider *EmailProviderDTO `json:"emailProvider"`
	SmsProvider *SMSProviderDTO `json:"smsProvider"`
}

type EmailProviderConfig struct {
	Config interface{} `json:"config"`
	From string `json:"from"`
	From_name string `json:"from_name"`
	Provider string `json:"provider"`
	Reply_to string `json:"reply_to"`
}

type CheckDependencies struct {
}

type ListRevisionsRequest struct {
}

type MockUserRepository struct {
}

type GetOrganizationsResult struct {
	Data []OrganizationSummaryDTO `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
	Stats OrganizationStatsDTO `json:"stats"`
}

type LinkResponse struct {
	Message string `json:"message"`
	User interface{} `json:"user"`
}

type AutomatedChecksConfig struct {
	PasswordPolicy bool `json:"passwordPolicy"`
	SessionPolicy bool `json:"sessionPolicy"`
	SuspiciousActivity bool `json:"suspiciousActivity"`
	DataRetention bool `json:"dataRetention"`
	Enabled bool `json:"enabled"`
	MfaCoverage bool `json:"mfaCoverage"`
	AccessReview bool `json:"accessReview"`
	CheckInterval time.Duration `json:"checkInterval"`
	InactiveUsers bool `json:"inactiveUsers"`
}

type CreateRoleTemplateInput struct {
	AppId string `json:"appId"`
	Description string `json:"description"`
	Name string `json:"name"`
	Permissions []string `json:"permissions"`
}

type DeleteResponse struct {
	Success bool `json:"success"`
}

type ChangePasswordResponse struct {
	Message string `json:"message"`
}

type MockAppService struct {
}

type RevokeSessionRequestDTO struct {
}

type ConsentAuditLogsResponse struct {
	Audit_logs []*interface{} `json:"audit_logs"`
}

type DeviceAuthorizeDecisionResponse struct {
	Success bool `json:"success"`
	Approved bool `json:"approved"`
	Message string `json:"message"`
}

type MultiSessionErrorResponse struct {
	Error string `json:"error"`
}

type GetDocumentVerificationRequest struct {
	DocumentId xid.ID `json:"documentId"`
}

type OrganizationSettingsDTO struct {
	MaxTeamsPerOrg int `json:"maxTeamsPerOrg"`
	AllowUserCreation bool `json:"allowUserCreation"`
	DefaultRole string `json:"defaultRole"`
	MaxMembersPerOrg int `json:"maxMembersPerOrg"`
	RequireInvitation bool `json:"requireInvitation"`
	AllowMultipleMemberships bool `json:"allowMultipleMemberships"`
	Enabled bool `json:"enabled"`
	InvitationExpiryDays int `json:"invitationExpiryDays"`
	MaxOrgsPerUser int `json:"maxOrgsPerUser"`
}

type FactorEnrollmentResponse struct {
	FactorId xid.ID `json:"factorId"`
	ProvisioningData interface{} `json:"provisioningData"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
}

type SecurityQuestionInfo struct {
	Id xid.ID `json:"id"`
	IsCustom bool `json:"isCustom"`
	QuestionId int `json:"questionId"`
	QuestionText string `json:"questionText"`
}

type NoOpVideoProvider struct {
}

type RenderTemplateResponse struct {
	Subject string `json:"subject"`
	Body string `json:"body"`
}

type CreateVerificationSessionRequest struct {
	CancelUrl string `json:"cancelUrl"`
	Config interface{} `json:"config"`
	Metadata interface{} `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
}

type CreateVerificationSessionResponse struct {
	Session interface{} `json:"session"`
}

type PreviewTemplateInput struct {
	TemplateId string `json:"templateId"`
	Variables interface{} `json:"variables"`
}

type RedisChallengeStore struct {
}

type TokenRequest struct {
	Audience string `json:"audience"`
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Code string `json:"code"`
	Code_verifier string `json:"code_verifier"`
	Device_code string `json:"device_code"`
	Grant_type string `json:"grant_type"`
	Redirect_uri string `json:"redirect_uri"`
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
}

type ConsentCookieResponse struct {
	Preferences interface{} `json:"preferences"`
}

type UserAdapter struct {
}

type UnpublishEntryRequest struct {
}

type TokenIntrospectionResponse struct {
	Active bool `json:"active"`
	Aud []string `json:"aud"`
	Client_id string `json:"client_id"`
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
	Iss string `json:"iss"`
	Nbf int64 `json:"nbf"`
	Sub string `json:"sub"`
	Jti string `json:"jti"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
	Username string `json:"username"`
}

type AccessTokenClaims struct {
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
}

type PrivacySettings struct {
	CcpaMode bool `json:"ccpaMode"`
	ConsentRequired bool `json:"consentRequired"`
	ContactEmail string `json:"contactEmail"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DataExportExpiryHours int `json:"dataExportExpiryHours"`
	DeletionGracePeriodDays int `json:"deletionGracePeriodDays"`
	ExportFormat []string `json:"exportFormat"`
	RequireExplicitConsent bool `json:"requireExplicitConsent"`
	AnonymousConsentEnabled bool `json:"anonymousConsentEnabled"`
	CookieConsentEnabled bool `json:"cookieConsentEnabled"`
	CreatedAt time.Time `json:"createdAt"`
	DataRetentionDays int `json:"dataRetentionDays"`
	Id xid.ID `json:"id"`
	Metadata JSONBMap `json:"metadata"`
	RequireAdminApprovalForDeletion bool `json:"requireAdminApprovalForDeletion"`
	UpdatedAt time.Time `json:"updatedAt"`
	AllowDataPortability bool `json:"allowDataPortability"`
	AutoDeleteAfterDays int `json:"autoDeleteAfterDays"`
	ContactPhone string `json:"contactPhone"`
	DpoEmail string `json:"dpoEmail"`
	GdprMode bool `json:"gdprMode"`
	OrganizationId string `json:"organizationId"`
}

type ClientDetailsResponse struct {
	RequireConsent bool `json:"requireConsent"`
	TosURI string `json:"tosURI"`
	ResponseTypes []string `json:"responseTypes"`
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
	ClientID string `json:"clientID"`
	PolicyURI string `json:"policyURI"`
	RequirePKCE bool `json:"requirePKCE"`
	UpdatedAt string `json:"updatedAt"`
	AllowedScopes []string `json:"allowedScopes"`
	CreatedAt string `json:"createdAt"`
	LogoURI string `json:"logoURI"`
	Name string `json:"name"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	TrustedClient bool `json:"trustedClient"`
	ApplicationType string `json:"applicationType"`
	Contacts []string `json:"contacts"`
	GrantTypes []string `json:"grantTypes"`
	IsOrgLevel bool `json:"isOrgLevel"`
	OrganizationID string `json:"organizationID"`
	RedirectURIs []string `json:"redirectURIs"`
}

type TwoFARequiredResponse struct {
	Device_id string `json:"device_id"`
	Require_twofa bool `json:"require_twofa"`
	User User `json:"user"`
}

type ChannelsResponse struct {
	Channels interface{} `json:"channels"`
	Count int `json:"count"`
}

type MockUserService struct {
}

type GenerateRecoveryCodesResponse struct {
	Warning string `json:"warning"`
	Codes []string `json:"codes"`
	Count int `json:"count"`
	GeneratedAt time.Time `json:"generatedAt"`
}

type BackupAuthQuestionsResponse struct {
	Questions []string `json:"questions"`
}

type GetProviderRequest struct {
}

type GetOverviewStatsResult struct {
	Stats OverviewStatsDTO `json:"stats"`
}

type TestProviderResult struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type NamespacesListResponse struct {
	Namespaces []*NamespaceResponse `json:"namespaces"`
	TotalCount int `json:"totalCount"`
}

type App struct {
}

type AddTrustedContactResponse struct {
	ContactId xid.ID `json:"contactId"`
	Email string `json:"email"`
	Message string `json:"message"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Verified bool `json:"verified"`
	AddedAt time.Time `json:"addedAt"`
}

type TOTPSecret struct {
}

type MigrateRolesRequest struct {
	DryRun bool `json:"dryRun"`
}

