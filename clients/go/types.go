package authsome

// Auto-generated types

type VerifyRecoveryCodeResponse struct {
	Message string `json:"message"`
	RemainingCodes int `json:"remainingCodes"`
	Valid bool `json:"valid"`
}

type RiskAssessmentConfig struct {
	NewLocationWeight float64 `json:"newLocationWeight"`
	RequireReviewAbove float64 `json:"requireReviewAbove"`
	Enabled bool `json:"enabled"`
	HighRiskThreshold float64 `json:"highRiskThreshold"`
	LowRiskThreshold float64 `json:"lowRiskThreshold"`
	MediumRiskThreshold float64 `json:"mediumRiskThreshold"`
	NewDeviceWeight float64 `json:"newDeviceWeight"`
	VelocityWeight float64 `json:"velocityWeight"`
	BlockHighRisk bool `json:"blockHighRisk"`
	HistoryWeight float64 `json:"historyWeight"`
	NewIpWeight float64 `json:"newIpWeight"`
}

type IDTokenClaims struct {
	Auth_time int64 `json:"auth_time"`
	Email string `json:"email"`
	Family_name string `json:"family_name"`
	Given_name string `json:"given_name"`
	Preferred_username string `json:"preferred_username"`
	Email_verified bool `json:"email_verified"`
	Name string `json:"name"`
	Nonce string `json:"nonce"`
	Session_state string `json:"session_state"`
}

type AuthorizeRequest struct {
	 *string `json:",omitempty"`
}

type SignInRequest struct {
	Provider string `json:"provider"`
	RedirectUrl string `json:"redirectUrl"`
	Scopes []string `json:"scopes"`
}

type StartRecoveryRequest struct {
	PreferredMethod RecoveryMethod `json:"preferredMethod"`
	UserId string `json:"userId"`
	DeviceId string `json:"deviceId"`
	Email string `json:"email"`
}

type MultiSessionSetActiveResponse struct {
	Token string `json:"token"`
	Session  `json:"session"`
}

type FactorEnrollmentResponse struct {
	FactorId xid.ID `json:"factorId"`
	ProvisioningData  `json:"provisioningData"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
}

type VerifyCodeResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	Message string `json:"message"`
	Valid bool `json:"valid"`
}

type AmountRule struct {
	Min_amount float64 `json:"min_amount"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Currency string `json:"currency"`
	Description string `json:"description"`
	Max_amount float64 `json:"max_amount"`
}

type SuccessResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

type ImpersonationMiddleware struct {
	 *Config `json:",omitempty"`
}

type AutomatedChecksConfig struct {
	CheckInterval time.Duration `json:"checkInterval"`
	SuspiciousActivity bool `json:"suspiciousActivity"`
	DataRetention bool `json:"dataRetention"`
	Enabled bool `json:"enabled"`
	InactiveUsers bool `json:"inactiveUsers"`
	MfaCoverage bool `json:"mfaCoverage"`
	PasswordPolicy bool `json:"passwordPolicy"`
	SessionPolicy bool `json:"sessionPolicy"`
	AccessReview bool `json:"accessReview"`
}

type BanUser_reqBody struct {
	Expires_at **time.Time `json:"expires_at,omitempty"`
	Reason string `json:"reason"`
}

type VerifyRecoveryCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type ConsentRecordResponse struct {
	Id string `json:"id"`
}

type CookieConsent struct {
	Analytics bool `json:"analytics"`
	Essential bool `json:"essential"`
	IpAddress string `json:"ipAddress"`
	UserId string `json:"userId"`
	ConsentBannerVersion string `json:"consentBannerVersion"`
	ExpiresAt time.Time `json:"expiresAt"`
	OrganizationId string `json:"organizationId"`
	Personalization bool `json:"personalization"`
	SessionId string `json:"sessionId"`
	ThirdParty bool `json:"thirdParty"`
	Functional bool `json:"functional"`
	Id xid.ID `json:"id"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Marketing bool `json:"marketing"`
	UserAgent string `json:"userAgent"`
}

type Challenge struct {
	Attempts int `json:"attempts"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	IpAddress string `json:"ipAddress"`
	Status ChallengeStatus `json:"status"`
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
	FactorId xid.ID `json:"factorId"`
	Id xid.ID `json:"id"`
	MaxAttempts int `json:"maxAttempts"`
	Metadata  `json:"metadata"`
	Type FactorType `json:"type"`
	VerifiedAt *time.Time `json:"verifiedAt"`
	- string `json:"-"`
}

type AdminBypassRequest struct {
	Duration int `json:"duration"`
	Reason string `json:"reason"`
	UserId xid.ID `json:"userId"`
}

type BanUserRequest struct {
	Expires_at *time.Time `json:"expires_at"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
	User_organization_id *xid.ID `json:"user_organization_id"`
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
}

type CreateProfileRequest struct {
	 *bool `json:",omitempty"`
}

type StepUpPolicy struct {
	Description string `json:"description"`
	Enabled bool `json:"enabled"`
	Id string `json:"id"`
	Name string `json:"name"`
	Priority int `json:"priority"`
	Metadata  `json:"metadata"`
	Org_id string `json:"org_id"`
	Rules  `json:"rules"`
	Updated_at time.Time `json:"updated_at"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
}

type mockImpersonationRepository struct {
	 * `json:",omitempty"`
}

type ComplianceStatus struct {
	ChecksFailed int `json:"checksFailed"`
	ChecksPassed int `json:"checksPassed"`
	ChecksWarning int `json:"checksWarning"`
	NextAudit time.Time `json:"nextAudit"`
	OverallStatus string `json:"overallStatus"`
	Score int `json:"score"`
	AppId string `json:"appId"`
	LastChecked time.Time `json:"lastChecked"`
	ProfileId string `json:"profileId"`
	Standard ComplianceStandard `json:"standard"`
	Violations int `json:"violations"`
}

type BackupAuthStatusResponse struct {
	Status string `json:"status"`
}

type ConnectionResponse struct {
	Connection  `json:"connection"`
}

type CreateUserRequest struct {
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
	Metadata  `json:"metadata"`
	Password string `json:"password"`
	User_organization_id *xid.ID `json:"user_organization_id"`
	Username string `json:"username"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type ScheduleVideoSessionResponse struct {
	Instructions string `json:"instructions"`
	JoinUrl string `json:"joinUrl"`
	Message string `json:"message"`
	ScheduledAt time.Time `json:"scheduledAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type CompliancePoliciesResponse struct {
	Policies []* `json:"policies"`
}

type AuditConfig struct {
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	LogAllAttempts bool `json:"logAllAttempts"`
	LogDeviceInfo bool `json:"logDeviceInfo"`
	LogFailed bool `json:"logFailed"`
	LogSuccessful bool `json:"logSuccessful"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
	Enabled bool `json:"enabled"`
	ImmutableLogs bool `json:"immutableLogs"`
	LogIpAddress bool `json:"logIpAddress"`
	LogUserAgent bool `json:"logUserAgent"`
	RetentionDays int `json:"retentionDays"`
}

type TrustedDevice struct {
	CreatedAt time.Time `json:"createdAt"`
	DeviceId string `json:"deviceId"`
	ExpiresAt time.Time `json:"expiresAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	Name string `json:"name"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	Metadata  `json:"metadata"`
	UserAgent string `json:"userAgent"`
	UserId xid.ID `json:"userId"`
}

type RecoveryAttemptLog struct {
	 *string `json:",omitempty"`
}

type CompleteRecoveryRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type DataProcessingAgreement struct {
	SignedByName string `json:"signedByName"`
	Version string `json:"version"`
	Content string `json:"content"`
	DigitalSignature string `json:"digitalSignature"`
	Id xid.ID `json:"id"`
	OrganizationId string `json:"organizationId"`
	AgreementType string `json:"agreementType"`
	CreatedAt time.Time `json:"createdAt"`
	EffectiveDate time.Time `json:"effectiveDate"`
	SignedBy string `json:"signedBy"`
	Status string `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
	IpAddress string `json:"ipAddress"`
	Metadata JSONBMap `json:"metadata"`
	SignedByEmail string `json:"signedByEmail"`
	SignedByTitle string `json:"signedByTitle"`
	ExpiryDate *time.Time `json:"expiryDate"`
}

type StepUpVerification struct {
	Verified_at time.Time `json:"verified_at"`
	Created_at time.Time `json:"created_at"`
	Reason string `json:"reason"`
	Security_level SecurityLevel `json:"security_level"`
	Method VerificationMethod `json:"method"`
	Rule_name string `json:"rule_name"`
	User_agent string `json:"user_agent"`
	Expires_at time.Time `json:"expires_at"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Org_id string `json:"org_id"`
	User_id string `json:"user_id"`
	Device_id string `json:"device_id"`
	Metadata  `json:"metadata"`
	Session_id string `json:"session_id"`
}

type TimeBasedRule struct {
	Description string `json:"description"`
	Max_age time.Duration `json:"max_age"`
	Operation string `json:"operation"`
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
}

type ScopeInfo struct {
	 *string `json:",omitempty"`
}

type ComplianceDashboardResponse struct {
	Metrics  `json:"metrics"`
}

type RateLimiter struct {
	 **repository.MFARepository `json:",omitempty"`
}

type ConsentNotificationsConfig struct {
	NotifyDeletionApproved bool `json:"notifyDeletionApproved"`
	NotifyDeletionComplete bool `json:"notifyDeletionComplete"`
	NotifyDpoEmail string `json:"notifyDpoEmail"`
	NotifyExportReady bool `json:"notifyExportReady"`
	NotifyOnExpiry bool `json:"notifyOnExpiry"`
	Channels []string `json:"channels"`
	Enabled bool `json:"enabled"`
	NotifyOnGrant bool `json:"notifyOnGrant"`
	NotifyOnRevoke bool `json:"notifyOnRevoke"`
}

type ConsentExportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type ComplianceViolationsResponse struct {
	Violations []* `json:"violations"`
}

type ListTrustedDevicesResponse struct {
	Count int `json:"count"`
	Devices []TrustedDevice `json:"devices"`
}

type DeleteFactorRequest struct {
	 *string `json:",omitempty"`
}

type VerifyFactor_req struct {
	Code string `json:"code"`
}

type AuthURLResponse struct {
	Url string `json:"url"`
}

type CompleteVideoSessionResponse struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	Result string `json:"result"`
}

type RateLimitConfig struct {
	Window time.Duration `json:"window"`
	Enabled bool `json:"enabled"`
}

type EnableRequest struct {
	 *string `json:",omitempty"`
}

type StatsResponse struct {
	Total_sessions int `json:"total_sessions"`
	Total_users int `json:"total_users"`
	Active_sessions int `json:"active_sessions"`
	Active_users int `json:"active_users"`
	Banned_users int `json:"banned_users"`
	Timestamp string `json:"timestamp"`
}

type ScheduleVideoSessionRequest struct {
	ScheduledAt time.Time `json:"scheduledAt"`
	SessionId xid.ID `json:"sessionId"`
	TimeZone string `json:"timeZone"`
}

type ConsentDeletionResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
}

type PasskeyStatusResponse struct {
	Status string `json:"status"`
}

type UsernameStatusResponse struct {
	Status string `json:"status"`
}

type EmailOTPVerifyResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
	User  `json:"user"`
}

type OIDCClientResponse struct {
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Redirect_uris []string `json:"redirect_uris"`
}

type VerifyRequest struct {
	Device_id string `json:"device_id"`
	Ip string `json:"ip"`
	Method VerificationMethod `json:"method"`
	Remember_device bool `json:"remember_device"`
	Requirement_id string `json:"requirement_id"`
	User_agent string `json:"user_agent"`
	Device_name string `json:"device_name"`
	Challenge_token string `json:"challenge_token"`
	Credential string `json:"credential"`
}

type StepUpAuditLog struct {
	User_id string `json:"user_id"`
	Event_type string `json:"event_type"`
	Org_id string `json:"org_id"`
	Created_at time.Time `json:"created_at"`
	Event_data  `json:"event_data"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Severity string `json:"severity"`
	User_agent string `json:"user_agent"`
}

type VideoVerificationSession struct {
	 *time.Time `json:",omitempty"`
}

type SSOInitResponse struct {
	Redirect_url string `json:"redirect_url"`
	Request_id string `json:"request_id"`
}

type MultiSessionDeleteResponse struct {
	Status string `json:"status"`
}

type TwoFAStatusResponse struct {
	Status string `json:"status"`
}

type EvaluationContext struct {
	 *string `json:",omitempty"`
}

type CallbackDataResponse struct {
	Action string `json:"action"`
	IsNewUser bool `json:"isNewUser"`
	User  `json:"user"`
}

type CreateTemplateVersion_req struct {
	Changes string `json:"changes"`
}

type Plugin struct {
	 **Service `json:",omitempty"`
}

type VideoSessionResult struct {
	 *bool `json:",omitempty"`
}

type RequestReverification_req struct {
	Reason string `json:"reason"`
}

type ChannelsResponse struct {
	Channels  `json:"channels"`
	Count int `json:"count"`
}

type SetUserRole_reqBody struct {
	Role string `json:"role"`
}

type ConsentReportResponse struct {
	Id string `json:"id"`
}

type App struct {
	 *string `json:",omitempty"`
}

type Factor struct {
	- string `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	Priority FactorPriority `json:"priority"`
	Status FactorStatus `json:"status"`
	Type FactorType `json:"type"`
	UserId xid.ID `json:"userId"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Id xid.ID `json:"id"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	UpdatedAt time.Time `json:"updatedAt"`
	VerifiedAt *time.Time `json:"verifiedAt"`
}

type CallbackResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
	User  `json:"user"`
}

type Service struct {
	 **session.Service `json:",omitempty"`
}

type BunRepository struct {
	 **bun.DB `json:",omitempty"`
}

type NotificationsConfig struct {
	NotifyAdminOnHighRisk bool `json:"notifyAdminOnHighRisk"`
	NotifyAdminOnReviewNeeded bool `json:"notifyAdminOnReviewNeeded"`
	NotifyOnRecoveryComplete bool `json:"notifyOnRecoveryComplete"`
	NotifyOnRecoveryFailed bool `json:"notifyOnRecoveryFailed"`
	NotifyOnRecoveryStart bool `json:"notifyOnRecoveryStart"`
	SecurityOfficerEmail string `json:"securityOfficerEmail"`
	Channels []string `json:"channels"`
	Enabled bool `json:"enabled"`
}

type ListRecoverySessionsRequest struct {
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	RequiresReview bool `json:"requiresReview"`
	Status RecoveryStatus `json:"status"`
	OrganizationId string `json:"organizationId"`
}

type StepUpRequirementsResponse struct {
	Requirements []* `json:"requirements"`
}

type StartImpersonation_reqBody struct {
	Duration_minutes *int `json:"duration_minutes,omitempty"`
	Reason string `json:"reason"`
	Target_user_id string `json:"target_user_id"`
	Ticket_number *string `json:"ticket_number,omitempty"`
}

type RotateAPIKeyResponse struct {
	Api_key *apikey.APIKey `json:"api_key"`
	Message string `json:"message"`
}

type TokenRequest struct {
	Grant_type string `json:"grant_type"`
	Redirect_uri string `json:"redirect_uri"`
	Client_id string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Code string `json:"code"`
	Code_verifier string `json:"code_verifier"`
}

type MockService struct {
	 *error `json:",omitempty"`
}

type ProvidersAppResponse struct {
	AppId string `json:"appId"`
	Providers  `json:"providers"`
}

type NotificationsResponse struct {
	Count int `json:"count"`
	Notifications  `json:"notifications"`
}

type RiskEngine struct {
	 **repository.MFARepository `json:",omitempty"`
}

type FactorEnrollmentRequest struct {
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
	Type FactorType `json:"type"`
}

type VerifySecurityAnswersResponse struct {
	AttemptsLeft int `json:"attemptsLeft"`
	CorrectAnswers int `json:"correctAnswers"`
	Message string `json:"message"`
	RequiredAnswers int `json:"requiredAnswers"`
	Valid bool `json:"valid"`
}

type TeamsResponse struct {
	Teams []*organization.Team `json:"teams"`
	Total int `json:"total"`
}

type MagicLinkVerifyResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
	User  `json:"user"`
}

type DeclareABTestWinner_req struct {
	AbTestGroup string `json:"abTestGroup"`
	WinnerId string `json:"winnerId"`
}

type Handler struct {
	 **Service `json:",omitempty"`
}

type SecurityQuestionInfo struct {
	Id xid.ID `json:"id"`
	IsCustom bool `json:"isCustom"`
	QuestionId int `json:"questionId"`
	QuestionText string `json:"questionText"`
}

type PrivacySettingsRequest struct {
	AllowDataPortability *bool `json:"allowDataPortability"`
	AutoDeleteAfterDays *int `json:"autoDeleteAfterDays"`
	ConsentRequired *bool `json:"consentRequired"`
	ContactEmail string `json:"contactEmail"`
	CookieConsentEnabled *bool `json:"cookieConsentEnabled"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	AnonymousConsentEnabled *bool `json:"anonymousConsentEnabled"`
	CcpaMode *bool `json:"ccpaMode"`
	ContactPhone string `json:"contactPhone"`
	DataExportExpiryHours *int `json:"dataExportExpiryHours"`
	DataRetentionDays *int `json:"dataRetentionDays"`
	ExportFormat []string `json:"exportFormat"`
	DeletionGracePeriodDays *int `json:"deletionGracePeriodDays"`
	DpoEmail string `json:"dpoEmail"`
	GdprMode *bool `json:"gdprMode"`
	RequireAdminApprovalForDeletion *bool `json:"requireAdminApprovalForDeletion"`
	RequireExplicitConsent *bool `json:"requireExplicitConsent"`
}

type WebhookPayload struct {
	 * `json:",omitempty"`
}

type SessionTokenResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
}

type MultiSessionListResponse struct {
	Sessions []* `json:"sessions"`
}

type PhoneVerifyResponse struct {
	Token string `json:"token"`
	User  `json:"user"`
	Session  `json:"session"`
}

type AdminListProvidersRequest struct {
	AppId xid.ID `json:"appId"`
}

type SetupSecurityQuestionsResponse struct {
	SetupAt time.Time `json:"setupAt"`
	Count int `json:"count"`
	Message string `json:"message"`
}

type ComplianceTemplate struct {
	DataResidency string `json:"dataResidency"`
	Name string `json:"name"`
	PasswordMinLength int `json:"passwordMinLength"`
	RequiredTraining []string `json:"requiredTraining"`
	RetentionDays int `json:"retentionDays"`
	Standard ComplianceStandard `json:"standard"`
	AuditFrequencyDays int `json:"auditFrequencyDays"`
	Description string `json:"description"`
	MfaRequired bool `json:"mfaRequired"`
	RequiredPolicies []string `json:"requiredPolicies"`
	SessionMaxAge int `json:"sessionMaxAge"`
}

type ListTrustedContactsResponse struct {
	Contacts []TrustedContactInfo `json:"contacts"`
	Count int `json:"count"`
}

type UpdateProvider_req struct {
	Config  `json:"config"`
	IsActive bool `json:"isActive"`
	IsDefault bool `json:"isDefault"`
}

type UpdatePolicy_req struct {
	Content *string `json:"content"`
	Status *string `json:"status"`
	Title *string `json:"title"`
	Version *string `json:"version"`
}

type DocumentVerificationConfig struct {
	Enabled bool `json:"enabled"`
	EncryptAtRest bool `json:"encryptAtRest"`
	MinConfidenceScore float64 `json:"minConfidenceScore"`
	Provider string `json:"provider"`
	RequireSelfie bool `json:"requireSelfie"`
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	StorageProvider string `json:"storageProvider"`
	EncryptionKey string `json:"encryptionKey"`
	RequireBothSides bool `json:"requireBothSides"`
	RequireManualReview bool `json:"requireManualReview"`
	StoragePath string `json:"storagePath"`
	AcceptedDocuments []string `json:"acceptedDocuments"`
}

type EmailOTPErrorResponse struct {
	Error string `json:"error"`
}

type CreateVerificationSession_req struct {
	CancelUrl string `json:"cancelUrl"`
	Config  `json:"config"`
	Metadata  `json:"metadata"`
	Provider string `json:"provider"`
	RequiredChecks []string `json:"requiredChecks"`
	SuccessUrl string `json:"successUrl"`
}

type ListTrainingFilter struct {
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	TrainingType *string `json:"trainingType"`
	UserId *string `json:"userId"`
	AppId *string `json:"appId"`
}

type ComplianceViolation struct {
	Status string `json:"status"`
	UserId string `json:"userId"`
	ViolationType string `json:"violationType"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Id string `json:"id"`
	Metadata  `json:"metadata"`
	ProfileId string `json:"profileId"`
	ResolvedAt *time.Time `json:"resolvedAt"`
	ResolvedBy string `json:"resolvedBy"`
	Severity string `json:"severity"`
	AppId string `json:"appId"`
}

type CompliancePolicyResponse struct {
	Id string `json:"id"`
}

type EmailFactorAdapter struct {
	 **emailotp.Service `json:",omitempty"`
}

type SocialStatusResponse struct {
	Status string `json:"status"`
}

type InvitationResponse struct {
	Message string `json:"message"`
	Invitation *organization.Invitation `json:"invitation"`
}

type IDVerificationStatusResponse struct {
	Status  `json:"status"`
}

type ComplianceTraining struct {
	Id string `json:"id"`
	Status string `json:"status"`
	UserId string `json:"userId"`
	AppId string `json:"appId"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Metadata  `json:"metadata"`
	ProfileId string `json:"profileId"`
	Score int `json:"score"`
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type PolicyEngine struct {
	 **Service `json:",omitempty"`
}

type ReviewDocumentRequest struct {
	Notes string `json:"notes"`
	RejectionReason string `json:"rejectionReason"`
	Approved bool `json:"approved"`
	DocumentId xid.ID `json:"documentId"`
}

type mockRepository struct {
}

type EvaluateRequest struct {
	Resource_type string `json:"resource_type"`
	Route string `json:"route"`
	Action string `json:"action"`
	Amount float64 `json:"amount"`
	Currency string `json:"currency"`
	Metadata  `json:"metadata"`
	Method string `json:"method"`
}

type DashboardConfig struct {
	Enabled bool `json:"enabled"`
	Path string `json:"path"`
	ShowRecentChecks bool `json:"showRecentChecks"`
	ShowReports bool `json:"showReports"`
	ShowScore bool `json:"showScore"`
	ShowViolations bool `json:"showViolations"`
}

type GetSecurityQuestionsResponse struct {
	Questions []SecurityQuestionInfo `json:"questions"`
}

type VerificationResult struct {
	 *int `json:",omitempty"`
}

type FacialCheckConfig struct {
	Enabled bool `json:"enabled"`
	MotionCapture bool `json:"motionCapture"`
	Variant string `json:"variant"`
}

type AdminBlockUser_req struct {
	Reason string `json:"reason"`
}

type RetentionConfig struct {
	ArchiveBeforePurge bool `json:"archiveBeforePurge"`
	ArchivePath string `json:"archivePath"`
	Enabled bool `json:"enabled"`
	GracePeriodDays int `json:"gracePeriodDays"`
	PurgeSchedule string `json:"purgeSchedule"`
}

type ListProfilesFilter struct {
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
}

type VideoSessionInfo struct {
	 *bool `json:",omitempty"`
}

type AddMember_req struct {
	Role string `json:"role"`
	User_id string `json:"user_id"`
}

type LinkAccountRequest struct {
	Provider string `json:"provider"`
	Scopes []string `json:"scopes"`
}

type RecoverySession struct {
	 *int `json:",omitempty"`
}

type DocumentVerificationResult struct {
	 *string `json:",omitempty"`
}

type ProviderCheckResult struct {
	 *string `json:",omitempty"`
}

type ListSessionsRequest struct {
	App_id xid.ID `json:"app_id"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	User_id *xid.ID `json:"user_id"`
	User_organization_id *xid.ID `json:"user_organization_id"`
	- xid.ID `json:"-"`
}

type TemplatesResponse struct {
	Count int `json:"count"`
	Templates  `json:"templates"`
}

type RiskContext struct {
	 *time.Time `json:",omitempty"`
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

type CreatePolicyRequest struct {
	Content string `json:"content"`
	Description string `json:"description"`
	Name string `json:"name"`
	Required bool `json:"required"`
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	Metadata  `json:"metadata"`
	Renewable bool `json:"renewable"`
	ValidityPeriod *int `json:"validityPeriod"`
}

type StepUpVerificationResponse struct {
	Expires_at string `json:"expires_at"`
	Verified bool `json:"verified"`
}

type EnrollFactorRequest struct {
	Type FactorType `json:"type"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Priority FactorPriority `json:"priority"`
}

type MultiSessionErrorResponse struct {
	Error string `json:"error"`
}

type Adapter struct {
	 **TemplateService `json:",omitempty"`
}

type MFASession struct {
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
	IpAddress string `json:"ipAddress"`
	Metadata  `json:"metadata"`
	UserAgent string `json:"userAgent"`
	VerifiedFactors []xid.ID `json:"verifiedFactors"`
	CompletedAt *time.Time `json:"completedAt"`
	Id xid.ID `json:"id"`
	RiskLevel RiskLevel `json:"riskLevel"`
	SessionToken string `json:"sessionToken"`
	UserId xid.ID `json:"userId"`
}

type ConsentAuditLogsResponse struct {
	Audit_logs []* `json:"audit_logs"`
}

type DashboardExtension struct {
	 *string `json:",omitempty"`
}

type DocumentTypesResponse struct {
	Document_types []string `json:"document_types"`
}

type AppHandler struct {
	 **coreapp.ServiceImpl `json:",omitempty"`
}

type ListEvidenceFilter struct {
	AppId *string `json:"appId"`
	ControlId *string `json:"controlId"`
	EvidenceType *string `json:"evidenceType"`
	ProfileId *string `json:"profileId"`
	Standard *ComplianceStandard `json:"standard"`
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

type CreateSessionRequest struct {
	 *string `json:",omitempty"`
}

type TemplateDefault struct {
	 *string `json:",omitempty"`
}

type VerificationRequest struct {
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data  `json:"data"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
}

type CodesResponse struct {
	Codes []string `json:"codes"`
}

type VerifyTrustedContactRequest struct {
	Token string `json:"token"`
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

type StepUpRememberedDevice struct {
	Last_used_at time.Time `json:"last_used_at"`
	Org_id string `json:"org_id"`
	Remembered_at time.Time `json:"remembered_at"`
	User_id string `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Device_id string `json:"device_id"`
	Id string `json:"id"`
	Security_level SecurityLevel `json:"security_level"`
	User_agent string `json:"user_agent"`
	Device_name string `json:"device_name"`
	Expires_at time.Time `json:"expires_at"`
	Ip string `json:"ip"`
}

type ListPoliciesFilter struct {
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
	PolicyType *string `json:"policyType"`
	ProfileId *string `json:"profileId"`
}

type BaseFactorAdapter struct {
	 *bool `json:",omitempty"`
}

type SecurityQuestion struct {
	 *bool `json:",omitempty"`
}

type ConsentDecision struct {
	 *bool `json:",omitempty"`
}

type ListSessionsResponse struct {
	Total_pages int `json:"total_pages"`
	Limit int `json:"limit"`
	Page int `json:"page"`
	Sessions []*session.Session `json:"sessions"`
	Total int `json:"total"`
}

type StepUpPolicyResponse struct {
	Id string `json:"id"`
}

type RateLimit struct {
	Max_requests int `json:"max_requests"`
	Window time.Duration `json:"window"`
}

type ComplianceReportResponse struct {
	Id string `json:"id"`
}

type SMSFactorAdapter struct {
	 **phone.Service `json:",omitempty"`
}

type RecoverySessionInfo struct {
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	CurrentStep int `json:"currentStep"`
	Id xid.ID `json:"id"`
	Status RecoveryStatus `json:"status"`
	TotalSteps int `json:"totalSteps"`
	UserId xid.ID `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Method RecoveryMethod `json:"method"`
	RequiresReview bool `json:"requiresReview"`
	RiskScore float64 `json:"riskScore"`
	UserEmail string `json:"userEmail"`
}

type BackupAuthContactResponse struct {
	Id string `json:"id"`
}

type JWTService struct {
	 *string `json:",omitempty"`
}

type OnfidoConfig struct {
	Enabled bool `json:"enabled"`
	IncludeDocumentReport bool `json:"includeDocumentReport"`
	IncludeFacialReport bool `json:"includeFacialReport"`
	FacialCheck FacialCheckConfig `json:"facialCheck"`
	IncludeWatchlistReport bool `json:"includeWatchlistReport"`
	Region string `json:"region"`
	WebhookToken string `json:"webhookToken"`
	WorkflowId string `json:"workflowId"`
	ApiToken string `json:"apiToken"`
	DocumentCheck DocumentCheckConfig `json:"documentCheck"`
}

type ComplianceChecksResponse struct {
	Checks []* `json:"checks"`
}

type OIDCConfigResponse struct {
	Config  `json:"config"`
	Issuer string `json:"issuer"`
}

type VerificationsResponse struct {
	Count int `json:"count"`
	Verifications  `json:"verifications"`
}

type ListRecoverySessionsResponse struct {
	PageSize int `json:"pageSize"`
	Sessions []RecoverySessionInfo `json:"sessions"`
	TotalCount int `json:"totalCount"`
	Page int `json:"page"`
}

type UpdatePolicyRequest struct {
	Metadata  `json:"metadata"`
	Name string `json:"name"`
	Renewable *bool `json:"renewable"`
	Required *bool `json:"required"`
	ValidityPeriod *int `json:"validityPeriod"`
	Active *bool `json:"active"`
	Content string `json:"content"`
	Description string `json:"description"`
}

type IDVerificationWebhookResponse struct {
	Status string `json:"status"`
}

type MockUserService struct {
	 *[]*User `json:",omitempty"`
}

type AdminUpdateProviderRequest struct {
	ClientSecret *string `json:"clientSecret"`
	Enabled *bool `json:"enabled"`
	Scopes []string `json:"scopes"`
	ClientId *string `json:"clientId"`
}

type UnbanUser_reqBody struct {
	Reason *string `json:"reason,omitempty"`
}

type RecoveryCodesConfig struct {
	Format string `json:"format"`
	RegenerateCount int `json:"regenerateCount"`
	AllowDownload bool `json:"allowDownload"`
	AllowPrint bool `json:"allowPrint"`
	AutoRegenerate bool `json:"autoRegenerate"`
	CodeCount int `json:"codeCount"`
	CodeLength int `json:"codeLength"`
	Enabled bool `json:"enabled"`
}

type DataExportConfig struct {
	AllowedFormats []string `json:"allowedFormats"`
	Enabled bool `json:"enabled"`
	ExpiryHours int `json:"expiryHours"`
	IncludeSections []string `json:"includeSections"`
	MaxExportSize int64 `json:"maxExportSize"`
	MaxRequests int `json:"maxRequests"`
	StoragePath string `json:"storagePath"`
	AutoCleanup bool `json:"autoCleanup"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	DefaultFormat string `json:"defaultFormat"`
	RequestPeriod time.Duration `json:"requestPeriod"`
}

type StepUpStatusResponse struct {
	Status string `json:"status"`
}

type TrackNotificationEvent_req struct {
	TemplateId string `json:"templateId"`
	Event string `json:"event"`
	EventData * `json:"eventData,omitempty"`
	NotificationId string `json:"notificationId"`
	OrganizationId **string `json:"organizationId,omitempty"`
}

type StartVideoSessionResponse struct {
	StartedAt time.Time `json:"startedAt"`
	VideoSessionId xid.ID `json:"videoSessionId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	SessionUrl string `json:"sessionUrl"`
}

type VerifyResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
	User  `json:"user"`
}

type mockUserService struct {
	 * `json:",omitempty"`
}

type RiskAssessment struct {
	Level RiskLevel `json:"level"`
	Metadata  `json:"metadata"`
	Recommended []FactorType `json:"recommended"`
	Score float64 `json:"score"`
	Factors []string `json:"factors"`
}

type PasskeyListResponse struct {
	Passkeys []* `json:"passkeys"`
}

type TwoFARequiredResponse struct {
	Require_twofa bool `json:"require_twofa"`
	User  `json:"user"`
	Device_id string `json:"device_id"`
}

type TwoFAEnableResponse struct {
	Status string `json:"status"`
	Totp_uri string `json:"totp_uri"`
}

type StepUpPoliciesResponse struct {
	Policies []* `json:"policies"`
}

type TokenResponse struct {
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
	Access_token string `json:"access_token"`
	Expires_in int `json:"expires_in"`
	Id_token string `json:"id_token"`
}

type ListChecksFilter struct {
	AppId *string `json:"appId"`
	CheckType *string `json:"checkType"`
	ProfileId *string `json:"profileId"`
	SinceBefore *time.Time `json:"sinceBefore"`
	Status *string `json:"status"`
}

type Config struct {
	AutoConvert bool `json:"autoConvert"`
	CleanupIntervalHours int `json:"cleanupIntervalHours"`
	EnableAnonymous bool `json:"enableAnonymous"`
	SessionExpiryHours int `json:"sessionExpiryHours"`
}

type CreateDPARequest struct {
	Content string `json:"content"`
	EffectiveDate time.Time `json:"effectiveDate"`
	SignedByName string `json:"signedByName"`
	Version string `json:"version"`
	AgreementType string `json:"agreementType"`
	ExpiryDate *time.Time `json:"expiryDate"`
	Metadata  `json:"metadata"`
	SignedByEmail string `json:"signedByEmail"`
	SignedByTitle string `json:"signedByTitle"`
}

type ImpersonationEndResponse struct {
	Ended_at string `json:"ended_at"`
	Status string `json:"status"`
}

type CompliancePolicy struct {
	ApprovedBy string `json:"approvedBy"`
	Content string `json:"content"`
	Metadata  `json:"metadata"`
	ProfileId string `json:"profileId"`
	UpdatedAt time.Time `json:"updatedAt"`
	EffectiveDate time.Time `json:"effectiveDate"`
	ReviewDate time.Time `json:"reviewDate"`
	Version string `json:"version"`
	Id string `json:"id"`
	Standard ComplianceStandard `json:"standard"`
	Status string `json:"status"`
	Title string `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	PolicyType string `json:"policyType"`
	AppId string `json:"appId"`
	ApprovedAt *time.Time `json:"approvedAt"`
}

type TOTPConfig struct {
	Algorithm string `json:"algorithm"`
	Digits int `json:"digits"`
	Enabled bool `json:"enabled"`
	Issuer string `json:"issuer"`
	Period int `json:"period"`
	Window_size int `json:"window_size"`
}

type WebAuthnFactorAdapter struct {
	 **passkey.Service `json:",omitempty"`
}

type CreateConsentRequest struct {
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
	ExpiresIn *int `json:"expiresIn"`
	Granted bool `json:"granted"`
	Metadata  `json:"metadata"`
	Purpose string `json:"purpose"`
	UserId string `json:"userId"`
}

type MemberHandler struct {
	 **coreapp.ServiceImpl `json:",omitempty"`
}

type MFAStatus struct {
	Enabled bool `json:"enabled"`
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
	GracePeriod *time.Time `json:"gracePeriod"`
	PolicyActive bool `json:"policyActive"`
	RequiredCount int `json:"requiredCount"`
	TrustedDevice bool `json:"trustedDevice"`
}

type SendCode_body struct {
	Phone string `json:"phone"`
}

type BackupAuthConfigResponse struct {
	Config  `json:"config"`
}

type GenerateReport_req struct {
	Format string `json:"format"`
	Period string `json:"period"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
}

type IDVerificationErrorResponse struct {
	Error string `json:"error"`
}

type TemplateService struct {
	 *Config `json:",omitempty"`
}

type UserServiceAdapter struct {
	 *user.ServiceInterface `json:",omitempty"`
}

type SocialProvidersResponse struct {
	Providers []string `json:"providers"`
}

type SetUserRoleRequest struct {
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
	Role string `json:"role"`
	User_id xid.ID `json:"user_id"`
	User_organization_id *xid.ID `json:"user_organization_id"`
}

type ConsentAuditLog struct {
	UserId string `json:"userId"`
	Action string `json:"action"`
	ConsentType string `json:"consentType"`
	CreatedAt time.Time `json:"createdAt"`
	Id xid.ID `json:"id"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	Reason string `json:"reason"`
	UserAgent string `json:"userAgent"`
	ConsentId string `json:"consentId"`
	NewValue JSONBMap `json:"newValue"`
	PreviousValue JSONBMap `json:"previousValue"`
	Purpose string `json:"purpose"`
}

type ConsentSummary struct {
	ExpiredConsents int `json:"expiredConsents"`
	HasPendingDeletion bool `json:"hasPendingDeletion"`
	LastConsentUpdate *time.Time `json:"lastConsentUpdate"`
	PendingRenewals int `json:"pendingRenewals"`
	RevokedConsents int `json:"revokedConsents"`
	TotalConsents int `json:"totalConsents"`
	GrantedConsents int `json:"grantedConsents"`
	HasPendingExport bool `json:"hasPendingExport"`
	OrganizationId string `json:"organizationId"`
	UserId string `json:"userId"`
	ConsentsByType  `json:"consentsByType"`
}

type RegisterProvider_req struct {
	SAMLIssuer string `json:"SAMLIssuer"`
	ProviderId string `json:"providerId"`
	OIDCClientID string `json:"OIDCClientID"`
	Domain string `json:"domain"`
	Type string `json:"type"`
	OIDCClientSecret string `json:"OIDCClientSecret"`
	OIDCIssuer string `json:"OIDCIssuer"`
	OIDCRedirectURI string `json:"OIDCRedirectURI"`
	SAMLCert string `json:"SAMLCert"`
	SAMLEntryPoint string `json:"SAMLEntryPoint"`
}

type JWKSService struct {
	 **KeyStore `json:",omitempty"`
}

type ConsentPolicyResponse struct {
	Id string `json:"id"`
}

type ConsentExpiryConfig struct {
	Enabled bool `json:"enabled"`
	ExpireCheckInterval time.Duration `json:"expireCheckInterval"`
	RenewalReminderDays int `json:"renewalReminderDays"`
	RequireReConsent bool `json:"requireReConsent"`
	AllowRenewal bool `json:"allowRenewal"`
	AutoExpireCheck bool `json:"autoExpireCheck"`
	DefaultValidityDays int `json:"defaultValidityDays"`
}

type ConsentReport struct {
	ConsentsByType  `json:"consentsByType"`
	DpasActive int `json:"dpasActive"`
	PendingDeletions int `json:"pendingDeletions"`
	ReportPeriodEnd time.Time `json:"reportPeriodEnd"`
	TotalUsers int `json:"totalUsers"`
	CompletedDeletions int `json:"completedDeletions"`
	ConsentRate float64 `json:"consentRate"`
	DataExportsThisPeriod int `json:"dataExportsThisPeriod"`
	DpasExpiringSoon int `json:"dpasExpiringSoon"`
	OrganizationId string `json:"organizationId"`
	ReportPeriodStart time.Time `json:"reportPeriodStart"`
	UsersWithConsent int `json:"usersWithConsent"`
}

type auditServiceAdapter struct {
	 **audit.Service `json:",omitempty"`
}

type LimitResult struct {
	 **time.Duration `json:",omitempty"`
}

type GetFactorRequest struct {
	 *string `json:",omitempty"`
}

type SecurityQuestionsConfig struct {
	MaxAnswerLength int `json:"maxAnswerLength"`
	MinimumQuestions int `json:"minimumQuestions"`
	AllowCustomQuestions bool `json:"allowCustomQuestions"`
	ForbidCommonAnswers bool `json:"forbidCommonAnswers"`
	LockoutDuration time.Duration `json:"lockoutDuration"`
	MaxAttempts int `json:"maxAttempts"`
	PredefinedQuestions []string `json:"predefinedQuestions"`
	RequireMinLength int `json:"requireMinLength"`
	RequiredToRecover int `json:"requiredToRecover"`
	CaseSensitive bool `json:"caseSensitive"`
	Enabled bool `json:"enabled"`
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

type OIDCUserInfoResponse struct {
	Email string `json:"email"`
	Email_verified bool `json:"email_verified"`
	Family_name string `json:"family_name"`
	Given_name string `json:"given_name"`
	Name string `json:"name"`
	Picture string `json:"picture"`
	Sub string `json:"sub"`
}

type AddTrustedContactRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
}

type StepUpDevicesResponse struct {
	Devices []* `json:"devices"`
}

type ImpersonationErrorResponse struct {
	Error string `json:"error"`
}

type RequestTrustedContactVerificationRequest struct {
	ContactId xid.ID `json:"contactId"`
	SessionId xid.ID `json:"sessionId"`
}

type BackupAuthVideoResponse struct {
	Session_id string `json:"session_id"`
}

type BeginLogin_body struct {
	User_id string `json:"user_id"`
}

type TestProvider_req struct {
	Config  `json:"config"`
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
	TestRecipient string `json:"testRecipient"`
}

type ComplianceTemplateResponse struct {
	Standard string `json:"standard"`
}

type ComplianceReportFileResponse struct {
	Content_type string `json:"content_type"`
	Data []byte `json:"data"`
}

type RiskFactor struct {
	 *string `json:",omitempty"`
}

type GetDocumentVerificationResponse struct {
	Status string `json:"status"`
	VerifiedAt *time.Time `json:"verifiedAt"`
	ConfidenceScore float64 `json:"confidenceScore"`
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
	RejectionReason string `json:"rejectionReason"`
}

type EvaluationResult struct {
	Matched_rules []string `json:"matched_rules"`
	Reason string `json:"reason"`
	Required bool `json:"required"`
	Requirement_id string `json:"requirement_id"`
	Security_level SecurityLevel `json:"security_level"`
	Current_level SecurityLevel `json:"current_level"`
	Expires_at time.Time `json:"expires_at"`
	Grace_period_ends_at time.Time `json:"grace_period_ends_at"`
	Metadata  `json:"metadata"`
	Allowed_methods []VerificationMethod `json:"allowed_methods"`
	Can_remember bool `json:"can_remember"`
	Challenge_token string `json:"challenge_token"`
}

type IDVerificationResponse struct {
	Verification  `json:"verification"`
}

type NotificationPreviewResponse struct {
	Body string `json:"body"`
	Subject string `json:"subject"`
}

type FactorVerificationRequest struct {
	Code string `json:"code"`
	Data  `json:"data"`
	FactorId xid.ID `json:"factorId"`
}

type ProviderConfigResponse struct {
	AppId string `json:"appId"`
	Message string `json:"message"`
	Provider string `json:"provider"`
}

type DocumentVerificationRequest struct {
	 *[]byte `json:",omitempty"`
}

type NoOpDocumentProvider struct {
}

type ConsentsResponse struct {
	Count int `json:"count"`
	Consents  `json:"consents"`
}

type UpdateConsentRequest struct {
	Granted *bool `json:"granted"`
	Metadata  `json:"metadata"`
	Reason string `json:"reason"`
}

type ConsentTypeStatus struct {
	Granted bool `json:"granted"`
	GrantedAt time.Time `json:"grantedAt"`
	NeedsRenewal bool `json:"needsRenewal"`
	Type string `json:"type"`
	Version string `json:"version"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type SSOProviderResponse struct {
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
}

type ResourceRule struct {
	Org_id string `json:"org_id"`
	Resource_type string `json:"resource_type"`
	Security_level SecurityLevel `json:"security_level"`
	Sensitivity string `json:"sensitivity"`
	Action string `json:"action"`
	Description string `json:"description"`
}

type AccessTokenClaims struct {
	Client_id string `json:"client_id"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
}

type RejectRecoveryRequest struct {
	Notes string `json:"notes"`
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type RecoveryCodeUsage struct {
	 *string `json:",omitempty"`
}

type DataExportRequest struct {
	Id xid.ID `json:"id"`
	UserId string `json:"userId"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Format string `json:"format"`
	IncludeSections []string `json:"includeSections"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	UpdatedAt time.Time `json:"updatedAt"`
	ExpiresAt *time.Time `json:"expiresAt"`
	ExportPath string `json:"exportPath"`
	ExportSize int64 `json:"exportSize"`
	Status string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	ExportUrl string `json:"exportUrl"`
}

type AssignRole_reqBody struct {
	RoleID string `json:"roleID"`
}

type ProviderSession struct {
	 *time.Time `json:",omitempty"`
}

type ComplianceStatusResponse struct {
	Status string `json:"status"`
}

type FactorInfo struct {
	Name string `json:"name"`
	Type FactorType `json:"type"`
	FactorId xid.ID `json:"factorId"`
	Metadata  `json:"metadata"`
}

type NotificationStatusResponse struct {
	Status string `json:"status"`
}

type ComplianceCheck struct {
	NextCheckAt time.Time `json:"nextCheckAt"`
	ProfileId string `json:"profileId"`
	Result  `json:"result"`
	AppId string `json:"appId"`
	LastCheckedAt time.Time `json:"lastCheckedAt"`
	Status string `json:"status"`
	CheckType string `json:"checkType"`
	CreatedAt time.Time `json:"createdAt"`
	Evidence []string `json:"evidence"`
	Id string `json:"id"`
}

type ComplianceTrainingsResponse struct {
	Training []* `json:"training"`
}

type GetRecoveryStatsResponse struct {
	AdminReviewsRequired int `json:"adminReviewsRequired"`
	FailedRecoveries int `json:"failedRecoveries"`
	SuccessRate float64 `json:"successRate"`
	TotalAttempts int `json:"totalAttempts"`
	AverageRiskScore float64 `json:"averageRiskScore"`
	HighRiskAttempts int `json:"highRiskAttempts"`
	MethodStats  `json:"methodStats"`
	PendingRecoveries int `json:"pendingRecoveries"`
	SuccessfulRecoveries int `json:"successfulRecoveries"`
}

type PrivacySettings struct {
	OrganizationId string `json:"organizationId"`
	RequireAdminApprovalForDeletion bool `json:"requireAdminApprovalForDeletion"`
	ConsentRequired bool `json:"consentRequired"`
	ContactPhone string `json:"contactPhone"`
	CookieConsentStyle string `json:"cookieConsentStyle"`
	DeletionGracePeriodDays int `json:"deletionGracePeriodDays"`
	ExportFormat []string `json:"exportFormat"`
	GdprMode bool `json:"gdprMode"`
	Id xid.ID `json:"id"`
	Metadata JSONBMap `json:"metadata"`
	AllowDataPortability bool `json:"allowDataPortability"`
	AutoDeleteAfterDays int `json:"autoDeleteAfterDays"`
	CcpaMode bool `json:"ccpaMode"`
	DpoEmail string `json:"dpoEmail"`
	RequireExplicitConsent bool `json:"requireExplicitConsent"`
	UpdatedAt time.Time `json:"updatedAt"`
	AnonymousConsentEnabled bool `json:"anonymousConsentEnabled"`
	CreatedAt time.Time `json:"createdAt"`
	DataExportExpiryHours int `json:"dataExportExpiryHours"`
	DataRetentionDays int `json:"dataRetentionDays"`
	ContactEmail string `json:"contactEmail"`
	CookieConsentEnabled bool `json:"cookieConsentEnabled"`
}

type SSOSAMLCallbackResponse struct {
	Attributes  `json:"attributes"`
	Issuer string `json:"issuer"`
	ProviderId string `json:"providerId"`
	Status string `json:"status"`
	Subject string `json:"subject"`
}

type mockForgeContext struct {
	 **http.Request `json:",omitempty"`
}

type AdminPolicyRequest struct {
	AllowedTypes []string `json:"allowedTypes"`
	Enabled bool `json:"enabled"`
	GracePeriod int `json:"gracePeriod"`
	RequiredFactors int `json:"requiredFactors"`
}

type ListUsersResponse struct {
	Page int `json:"page"`
	Total int `json:"total"`
	Total_pages int `json:"total_pages"`
	Users []*user.User `json:"users"`
	Limit int `json:"limit"`
}

type DocumentVerification struct {
	 *string `json:",omitempty"`
}

type MultiStepRecoveryConfig struct {
	MediumRiskSteps []RecoveryMethod `json:"mediumRiskSteps"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
	AllowStepSkip bool `json:"allowStepSkip"`
	AllowUserChoice bool `json:"allowUserChoice"`
	Enabled bool `json:"enabled"`
	MinimumSteps int `json:"minimumSteps"`
	SessionExpiry time.Duration `json:"sessionExpiry"`
	HighRiskSteps []RecoveryMethod `json:"highRiskSteps"`
	LowRiskSteps []RecoveryMethod `json:"lowRiskSteps"`
}

type VerifyTrustedContactResponse struct {
	Message string `json:"message"`
	Verified bool `json:"verified"`
	VerifiedAt time.Time `json:"verifiedAt"`
	ContactId xid.ID `json:"contactId"`
}

type ApproveRecoveryRequest struct {
	Notes string `json:"notes"`
	SessionId xid.ID `json:"sessionId"`
}

type DeviceInfo struct {
	DeviceId string `json:"deviceId"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
}

type TrustedContactInfo struct {
	Relationship string `json:"relationship"`
	Verified bool `json:"verified"`
	VerifiedAt *time.Time `json:"verifiedAt"`
	Active bool `json:"active"`
	Email string `json:"email"`
	Id xid.ID `json:"id"`
	Name string `json:"name"`
	Phone string `json:"phone"`
}

type ApproveRecoveryResponse struct {
	Approved bool `json:"approved"`
	ApprovedAt time.Time `json:"approvedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type SSOSAMLMetadataResponse struct {
	Metadata string `json:"metadata"`
}

type Verify_body struct {
	Remember bool `json:"remember"`
	Email string `json:"email"`
	Otp string `json:"otp"`
}

type StepUpAuditLogsResponse struct {
	Audit_logs []* `json:"audit_logs"`
}

type VerificationResponse struct {
	Verification  `json:"verification"`
}

type ComplianceReportsResponse struct {
	Reports []* `json:"reports"`
}

type StartVideoSessionRequest struct {
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type ConsentStatusResponse struct {
	Status string `json:"status"`
}

type mockSessionService struct {
	 * `json:",omitempty"`
}

type TeamHandler struct {
	 **app.ServiceImpl `json:",omitempty"`
}

type ListViolationsFilter struct {
	ViolationType *string `json:"violationType"`
	AppId *string `json:"appId"`
	ProfileId *string `json:"profileId"`
	Severity *string `json:"severity"`
	Status *string `json:"status"`
	UserId *string `json:"userId"`
}

type PhoneErrorResponse struct {
	Error string `json:"error"`
}

type UploadDocumentRequest struct {
	BackImage string `json:"backImage"`
	DocumentType string `json:"documentType"`
	FrontImage string `json:"frontImage"`
	Selfie string `json:"selfie"`
	SessionId xid.ID `json:"sessionId"`
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

type EmailConfig struct {
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
}

type CompleteVideoSessionRequest struct {
	LivenessPassed bool `json:"livenessPassed"`
	LivenessScore float64 `json:"livenessScore"`
	Notes string `json:"notes"`
	VerificationResult string `json:"verificationResult"`
	VideoSessionId xid.ID `json:"videoSessionId"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type NotificationTemplateListResponse struct {
	Total int `json:"total"`
	Templates []* `json:"templates"`
}

type MFAPolicy struct {
	AdaptiveMfaEnabled bool `json:"adaptiveMfaEnabled"`
	AllowedFactorTypes []FactorType `json:"allowedFactorTypes"`
	CreatedAt time.Time `json:"createdAt"`
	MaxFailedAttempts int `json:"maxFailedAttempts"`
	RequiredFactorCount int `json:"requiredFactorCount"`
	RequiredFactorTypes []FactorType `json:"requiredFactorTypes"`
	TrustedDeviceDays int `json:"trustedDeviceDays"`
	UpdatedAt time.Time `json:"updatedAt"`
	GracePeriodDays int `json:"gracePeriodDays"`
	Id xid.ID `json:"id"`
	LockoutDurationMinutes int `json:"lockoutDurationMinutes"`
	OrganizationId xid.ID `json:"organizationId"`
	StepUpRequired bool `json:"stepUpRequired"`
}

type PhoneSendCodeResponse struct {
	Dev_code string `json:"dev_code"`
	Status string `json:"status"`
}

type StartRecoveryResponse struct {
	RequiredSteps int `json:"requiredSteps"`
	RequiresReview bool `json:"requiresReview"`
	RiskScore float64 `json:"riskScore"`
	SessionId xid.ID `json:"sessionId"`
	Status RecoveryStatus `json:"status"`
	AvailableMethods []RecoveryMethod `json:"availableMethods"`
	CompletedSteps int `json:"completedSteps"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type NoOpSMSProvider struct {
}

type StepUpAttempt struct {
	Failure_reason string `json:"failure_reason"`
	Method VerificationMethod `json:"method"`
	Success bool `json:"success"`
	User_id string `json:"user_id"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Org_id string `json:"org_id"`
	Requirement_id string `json:"requirement_id"`
	User_agent string `json:"user_agent"`
	Created_at time.Time `json:"created_at"`
}

type IDVerificationSessionResponse struct {
	Session  `json:"session"`
}

type ComplianceViolationResponse struct {
	Id string `json:"id"`
}

type GetChallengeStatusResponse struct {
	MaxAttempts int `json:"maxAttempts"`
	Status ChallengeStatus `json:"status"`
	Attempts int `json:"attempts"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ChallengeId xid.ID `json:"challengeId"`
	FactorsRequired int `json:"factorsRequired"`
	FactorsVerified int `json:"factorsVerified"`
}

type OAuthState struct {
	 *string `json:",omitempty"`
}

type Disable_body struct {
	User_id string `json:"user_id"`
}

type SignInResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
	User  `json:"user"`
}

type RouteRule struct {
	Method string `json:"method"`
	Org_id string `json:"org_id"`
	Pattern string `json:"pattern"`
	Security_level SecurityLevel `json:"security_level"`
	Description string `json:"description"`
}

type ComplianceTemplatesResponse struct {
	Templates []* `json:"templates"`
}

type ListFactorsRequest struct {
	 *bool `json:",omitempty"`
}

type MagicLinkErrorResponse struct {
	Error string `json:"error"`
}

type SessionsResponse struct {
	Sessions  `json:"sessions"`
}

type ResetUserMFARequest struct {
	 *string `json:",omitempty"`
	Reason string `json:"reason"`
}

type DataDeletionConfig struct {
	Enabled bool `json:"enabled"`
	NotifyBeforeDeletion bool `json:"notifyBeforeDeletion"`
	RequireAdminApproval bool `json:"requireAdminApproval"`
	ArchiveBeforeDeletion bool `json:"archiveBeforeDeletion"`
	GracePeriodDays int `json:"gracePeriodDays"`
	PreserveLegalData bool `json:"preserveLegalData"`
	RetentionExemptions []string `json:"retentionExemptions"`
	AllowPartialDeletion bool `json:"allowPartialDeletion"`
	ArchivePath string `json:"archivePath"`
	AutoProcessAfterGrace bool `json:"autoProcessAfterGrace"`
}

type GetSecurityQuestionsRequest struct {
	SessionId xid.ID `json:"sessionId"`
}

type BackupAuthContactsResponse struct {
	Contacts []* `json:"contacts"`
}

type ConsentAuditConfig struct {
	RetentionDays int `json:"retentionDays"`
	ArchiveInterval time.Duration `json:"archiveInterval"`
	ArchiveOldLogs bool `json:"archiveOldLogs"`
	ExportFormat string `json:"exportFormat"`
	LogIpAddress bool `json:"logIpAddress"`
	LogUserAgent bool `json:"logUserAgent"`
	SignLogs bool `json:"signLogs"`
	Enabled bool `json:"enabled"`
	Immutable bool `json:"immutable"`
	LogAllChanges bool `json:"logAllChanges"`
}

type InitiateChallengeRequest struct {
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata  `json:"metadata"`
}

type ProvidersResponse struct {
	Providers  `json:"providers"`
}

type ConsentExportResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
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

type TwoFABackupCodesResponse struct {
	Codes []string `json:"codes"`
}

type GetRecoveryStatsRequest struct {
	OrganizationId string `json:"organizationId"`
	StartDate time.Time `json:"startDate"`
	EndDate time.Time `json:"endDate"`
}

type NotificationWebhookResponse struct {
	Status string `json:"status"`
}

type RequestTrustedContactVerificationResponse struct {
	ContactName string `json:"contactName"`
	ExpiresAt time.Time `json:"expiresAt"`
	Message string `json:"message"`
	NotifiedAt time.Time `json:"notifiedAt"`
	ContactId xid.ID `json:"contactId"`
}

type BackupAuthRecoveryResponse struct {
	Session_id string `json:"session_id"`
}

type ConsentSettingsResponse struct {
	Settings  `json:"settings"`
}

type OIDCErrorResponse struct {
	Error string `json:"error"`
	Error_description string `json:"error_description"`
}

type OIDCJWKSResponse struct {
	Keys []* `json:"keys"`
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

type ComplianceEvidenceResponse struct {
	Id string `json:"id"`
}

type SendVerificationCodeRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
	Target string `json:"target"`
}

type SignUp_body struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type Username2FARequiredResponse struct {
	Device_id string `json:"device_id"`
	Require_twofa bool `json:"require_twofa"`
	User  `json:"user"`
}

type CreateABTestVariant_req struct {
	Body string `json:"body"`
	Name string `json:"name"`
	Subject string `json:"subject"`
	Weight int `json:"weight"`
}

type UpdateFactorRequest struct {
	 *string `json:",omitempty"`
	Metadata  `json:"metadata"`
	Name *string `json:"name"`
	Priority *FactorPriority `json:"priority"`
	Status *FactorStatus `json:"status"`
}

type FactorAdapterRegistry struct {
	 * `json:",omitempty"`
}

type Middleware struct {
	 **Config `json:",omitempty"`
}

type EndImpersonation_reqBody struct {
	Impersonation_id string `json:"impersonation_id"`
	Reason *string `json:"reason,omitempty"`
}

type NotificationListResponse struct {
	Notifications []* `json:"notifications"`
	Total int `json:"total"`
}

type MockAuditService struct {
	 *[]*AuditEvent `json:",omitempty"`
}

type ConsentPolicy struct {
	Active bool `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	Renewable bool `json:"renewable"`
	Content string `json:"content"`
	CreatedBy string `json:"createdBy"`
	Id xid.ID `json:"id"`
	Name string `json:"name"`
	PublishedAt *time.Time `json:"publishedAt"`
	Required bool `json:"required"`
	UpdatedAt time.Time `json:"updatedAt"`
	ValidityPeriod *int `json:"validityPeriod"`
	Metadata JSONBMap `json:"metadata"`
	OrganizationId string `json:"organizationId"`
	Version string `json:"version"`
	ConsentType string `json:"consentType"`
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

type DataExportRequestInput struct {
	IncludeSections []string `json:"includeSections"`
	Format string `json:"format"`
}

type SSOErrorResponse struct {
	Error string `json:"error"`
}

type AdaptiveMFAConfig struct {
	Location_change_risk float64 `json:"location_change_risk"`
	Require_step_up_threshold float64 `json:"require_step_up_threshold"`
	Velocity_risk float64 `json:"velocity_risk"`
	New_device_risk float64 `json:"new_device_risk"`
	Risk_threshold float64 `json:"risk_threshold"`
	Enabled bool `json:"enabled"`
	Factor_ip_reputation bool `json:"factor_ip_reputation"`
	Factor_location_change bool `json:"factor_location_change"`
	Factor_new_device bool `json:"factor_new_device"`
	Factor_velocity bool `json:"factor_velocity"`
}

type SetupSecurityQuestionRequest struct {
	Answer string `json:"answer"`
	CustomText string `json:"customText"`
	QuestionId int `json:"questionId"`
}

type HealthCheckResponse struct {
	Version string `json:"version"`
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	Healthy bool `json:"healthy"`
	Message string `json:"message"`
	ProvidersStatus  `json:"providersStatus"`
}

type JumioProvider struct {
	 *JumioConfig `json:",omitempty"`
}

type ReportsConfig struct {
	Enabled bool `json:"enabled"`
	Formats []string `json:"formats"`
	IncludeEvidence bool `json:"includeEvidence"`
	RetentionDays int `json:"retentionDays"`
	Schedule string `json:"schedule"`
	StoragePath string `json:"storagePath"`
}

type TOTPFactorAdapter struct {
	 **twofa.Service `json:",omitempty"`
}

type DataDeletionRequestInput struct {
	DeleteSections []string `json:"deleteSections"`
	Reason string `json:"reason"`
}

type CompleteTraining_req struct {
	Score int `json:"score"`
}

type TrustedContactsConfig struct {
	RequireVerification bool `json:"requireVerification"`
	RequiredToRecover int `json:"requiredToRecover"`
	AllowEmailContacts bool `json:"allowEmailContacts"`
	AllowPhoneContacts bool `json:"allowPhoneContacts"`
	Enabled bool `json:"enabled"`
	VerificationExpiry time.Duration `json:"verificationExpiry"`
	CooldownPeriod time.Duration `json:"cooldownPeriod"`
	MaxNotificationsPerDay int `json:"maxNotificationsPerDay"`
	MaximumContacts int `json:"maximumContacts"`
	MinimumContacts int `json:"minimumContacts"`
}

type OIDCTokenResponse struct {
	Id_token string `json:"id_token"`
	Refresh_token string `json:"refresh_token"`
	Scope string `json:"scope"`
	Token_type string `json:"token_type"`
	Access_token string `json:"access_token"`
	Expires_in int `json:"expires_in"`
}

type UpdateProfileRequest struct {
	 **string `json:",omitempty"`
}

type GetChallengeStatusRequest struct {
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

type AnonymousSignInResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
}

type MembersResponse struct {
	Total int `json:"total"`
	Members []*organization.Member `json:"members"`
}

type ComplianceCheckResponse struct {
	Id string `json:"id"`
}

type Link_body struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Password string `json:"password"`
}

type UploadDocumentResponse struct {
	DocumentId xid.ID `json:"documentId"`
	Message string `json:"message"`
	ProcessingTime string `json:"processingTime"`
	Status string `json:"status"`
	UploadedAt time.Time `json:"uploadedAt"`
}

type BackupAuthCodesResponse struct {
	Codes []string `json:"codes"`
}

type ImpersonationContext struct {
	Impersonation_id *xid.ID `json:"impersonation_id"`
	Impersonator_id *xid.ID `json:"impersonator_id"`
	Indicator_message string `json:"indicator_message"`
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id *xid.ID `json:"target_user_id"`
}

type JWK struct {
	Use string `json:"use"`
	Alg string `json:"alg"`
	E string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N string `json:"n"`
}

type MockOrganizationService struct {
	 * `json:",omitempty"`
}

type ListReportsFilter struct {
	ProfileId *string `json:"profileId"`
	ReportType *string `json:"reportType"`
	Standard *ComplianceStandard `json:"standard"`
	Status *string `json:"status"`
	AppId *string `json:"appId"`
	Format *string `json:"format"`
}

type ComplianceProfileResponse struct {
	Id string `json:"id"`
}

type VerifyCodeRequest struct {
	Code string `json:"code"`
	SessionId xid.ID `json:"sessionId"`
}

type MockRepository struct {
	 * `json:",omitempty"`
}

type StepUpVerificationsResponse struct {
	Verifications []* `json:"verifications"`
}

type ComplianceUserTrainingResponse struct {
	User_id string `json:"user_id"`
}

type SocialCallbackResponse struct {
	Token string `json:"token"`
	User  `json:"user"`
}

type GetStatusRequest struct {
	 *string `json:",omitempty"`
}

type RecoveryConfiguration struct {
	 *[]RecoveryMethod `json:",omitempty"`
}

type MockEmailService struct {
	 *[]*Email `json:",omitempty"`
}

type ComplianceTrainingResponse struct {
	Id string `json:"id"`
}

type PasskeyLoginOptionsResponse struct {
	Options  `json:"options"`
}

type NotificationTemplateResponse struct {
	Template  `json:"template"`
}

type AuditLog struct {
	 *time.Time `json:",omitempty"`
}

type TwoFAStatusDetailResponse struct {
	Method string `json:"method"`
	Trusted bool `json:"trusted"`
	Enabled bool `json:"enabled"`
}

type FactorsResponse struct {
	Count int `json:"count"`
	Factors  `json:"factors"`
}

type SocialSignInResponse struct {
	Redirect_url string `json:"redirect_url"`
}

type SocialErrorResponse struct {
	Error string `json:"error"`
}

type TOTPSecret struct {
	 *string `json:",omitempty"`
}

type ChallengeRequest struct {
	UserId xid.ID `json:"userId"`
	Context string `json:"context"`
	FactorTypes []FactorType `json:"factorTypes"`
	Metadata  `json:"metadata"`
}

type ContinueRecoveryRequest struct {
	Method RecoveryMethod `json:"method"`
	SessionId xid.ID `json:"sessionId"`
}

type StepUpRequirementResponse struct {
	Id string `json:"id"`
}

type KeyStore struct {
	 *string `json:",omitempty"`
}

type IPWhitelistConfig struct {
	Enabled bool `json:"enabled"`
	Strict_mode bool `json:"strict_mode"`
}

type CreateProvider_req struct {
	IsDefault bool `json:"isDefault"`
	OrganizationId **string `json:"organizationId,omitempty"`
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`
	Config  `json:"config"`
}

type BackupCodesConfig struct {
	Allow_reuse bool `json:"allow_reuse"`
	Count int `json:"count"`
	Enabled bool `json:"enabled"`
	Format string `json:"format"`
	Length int `json:"length"`
}

type ContextRule struct {
	Org_id string `json:"org_id"`
	Security_level SecurityLevel `json:"security_level"`
	Condition string `json:"condition"`
	Description string `json:"description"`
	Name string `json:"name"`
}

type DevicesResponse struct {
	Count int `json:"count"`
	Devices  `json:"devices"`
}

type RejectRecoveryResponse struct {
	Message string `json:"message"`
	Reason string `json:"reason"`
	Rejected bool `json:"rejected"`
	RejectedAt time.Time `json:"rejectedAt"`
	SessionId xid.ID `json:"sessionId"`
}

type NotificationChannels struct {
	Webhook bool `json:"webhook"`
	Email bool `json:"email"`
	Slack bool `json:"slack"`
}

type CreateTraining_req struct {
	Standard ComplianceStandard `json:"standard"`
	TrainingType string `json:"trainingType"`
	UserId string `json:"userId"`
}

type ComplianceReport struct {
	ProfileId string `json:"profileId"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	FileSize int64 `json:"fileSize"`
	GeneratedBy string `json:"generatedBy"`
	ReportType string `json:"reportType"`
	Standard ComplianceStandard `json:"standard"`
	Summary  `json:"summary"`
	AppId string `json:"appId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FileUrl string `json:"fileUrl"`
	Format string `json:"format"`
	Id string `json:"id"`
	Period string `json:"period"`
}

type RevokeTrustedDeviceRequest struct {
	 *string `json:",omitempty"`
}

type MetadataResponse struct {
	Metadata string `json:"metadata"`
}

type AnonymousAuthResponse struct {
	Token string `json:"token"`
	User  `json:"user"`
	Session  `json:"session"`
}

type BackupAuthQuestionsResponse struct {
	Questions []string `json:"questions"`
}

type SaveNotificationSettings_req struct {
	AutoSendWelcome bool `json:"autoSendWelcome"`
	CleanupAfter string `json:"cleanupAfter"`
	RetryAttempts int `json:"retryAttempts"`
	RetryDelay string `json:"retryDelay"`
}

type ImpersonateUserRequest struct {
	- string `json:"-"`
	App_id xid.ID `json:"app_id"`
	Duration time.Duration `json:"duration"`
	User_id xid.ID `json:"user_id"`
	User_organization_id *xid.ID `json:"user_organization_id"`
}

type UpdateRecoveryConfigRequest struct {
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
}

type RegisterClient_req struct {
	Name string `json:"name"`
	Redirect_uri string `json:"redirect_uri"`
}

type IDVerificationListResponse struct {
	Verifications []* `json:"verifications"`
}

type FinishLogin_body struct {
	Remember bool `json:"remember"`
	User_id string `json:"user_id"`
}

type PasskeyErrorResponse struct {
	Error string `json:"error"`
}

type AddTeamMember_req struct {
	Member_id xid.ID `json:"member_id"`
	Role string `json:"role"`
}

type ChallengeResponse struct {
	ChallengeId xid.ID `json:"challengeId"`
	ExpiresAt time.Time `json:"expiresAt"`
	FactorsRequired int `json:"factorsRequired"`
	SessionId xid.ID `json:"sessionId"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
}

type ListFactorsResponse struct {
	Count int `json:"count"`
	Factors []Factor `json:"factors"`
}

type StripeIdentityConfig struct {
	UseMock bool `json:"useMock"`
	WebhookSecret string `json:"webhookSecret"`
	AllowedTypes []string `json:"allowedTypes"`
	ApiKey string `json:"apiKey"`
	Enabled bool `json:"enabled"`
	RequireLiveCapture bool `json:"requireLiveCapture"`
	RequireMatchingSelfie bool `json:"requireMatchingSelfie"`
	ReturnUrl string `json:"returnUrl"`
}

type CreateProfileFromTemplate_req struct {
	AppId string `json:"appId"`
	Standard ComplianceStandard `json:"standard"`
}

type SocialLinkResponse struct {
	Linked bool `json:"linked"`
}

type GenerateBackupCodes_body struct {
	Count int `json:"count"`
	User_id string `json:"user_id"`
}

type ConsentCookieResponse struct {
	Preferences  `json:"preferences"`
}

type PasskeyLoginResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
	User  `json:"user"`
}

type AMLMatch struct {
	 *string `json:",omitempty"`
}

type DocumentCheckConfig struct {
	ValidateDataConsistency bool `json:"validateDataConsistency"`
	ValidateExpiry bool `json:"validateExpiry"`
	Enabled bool `json:"enabled"`
	ExtractData bool `json:"extractData"`
}

type TestSendTemplate_req struct {
	Recipient string `json:"recipient"`
	Variables  `json:"variables"`
}

type AppServiceAdapter struct {
	 * `json:",omitempty"`
}

type CreateEvidence_req struct {
	ControlId string `json:"controlId"`
	Description string `json:"description"`
	EvidenceType string `json:"evidenceType"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
}

type DefaultProviderRegistry struct {
	 *NotificationProvider `json:",omitempty"`
}

type ConsentRecord struct {
	ConsentType string `json:"consentType"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Granted bool `json:"granted"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	CreatedAt time.Time `json:"createdAt"`
	GrantedAt time.Time `json:"grantedAt"`
	Id xid.ID `json:"id"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserAgent string `json:"userAgent"`
	UserId string `json:"userId"`
	Metadata JSONBMap `json:"metadata"`
	RevokedAt *time.Time `json:"revokedAt"`
	Version string `json:"version"`
	Purpose string `json:"purpose"`
}

type mockProvider struct {
	 *error `json:",omitempty"`
}

type OnfidoProvider struct {
	 *OnfidoConfig `json:",omitempty"`
}

type SendWithTemplateRequest struct {
	Type notification.NotificationType `json:"type"`
	Variables  `json:"variables"`
	AppId xid.ID `json:"appId"`
	Language string `json:"language"`
	Metadata  `json:"metadata"`
	Recipient string `json:"recipient"`
	TemplateKey string `json:"templateKey"`
}

type ProvidersConfig struct {
	Email EmailProviderConfig `json:"email"`
	Sms *SMSProviderConfig `json:"sms"`
}

type GenerateRecoveryCodesResponse struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Warning string `json:"warning"`
	Codes []string `json:"codes"`
	Count int `json:"count"`
}

// Webhook represents Webhook configuration
type Webhook struct {
	Enabled bool `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
	OrganizationId string `json:"organizationId"`
	Url string `json:"url"`
	Events []string `json:"events"`
	Secret string `json:"secret"`
}

type AuditEvent struct {
	 * `json:",omitempty"`
}

type GetDocumentVerificationRequest struct {
	DocumentId xid.ID `json:"documentId"`
}

type GenerateRecoveryCodesRequest struct {
	Format string `json:"format"`
	Count int `json:"count"`
}

type DataDeletionRequest struct {
	ApprovedBy string `json:"approvedBy"`
	ErrorMessage string `json:"errorMessage"`
	ExemptionReason string `json:"exemptionReason"`
	Id xid.ID `json:"id"`
	UserId string `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	DeleteSections []string `json:"deleteSections"`
	RetentionExempt bool `json:"retentionExempt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ApprovedAt *time.Time `json:"approvedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	IpAddress string `json:"ipAddress"`
	OrganizationId string `json:"organizationId"`
	RejectedAt *time.Time `json:"rejectedAt"`
	Status string `json:"status"`
	ArchivePath string `json:"archivePath"`
	RequestReason string `json:"requestReason"`
}

type StepUpRequirement struct {
	Resource_action string `json:"resource_action"`
	Challenge_token string `json:"challenge_token"`
	Currency string `json:"currency"`
	Reason string `json:"reason"`
	Risk_score float64 `json:"risk_score"`
	Rule_name string `json:"rule_name"`
	Status string `json:"status"`
	Current_level SecurityLevel `json:"current_level"`
	Id string `json:"id"`
	Ip string `json:"ip"`
	Method string `json:"method"`
	Org_id string `json:"org_id"`
	Route string `json:"route"`
	Created_at time.Time `json:"created_at"`
	Expires_at time.Time `json:"expires_at"`
	Metadata  `json:"metadata"`
	Resource_type string `json:"resource_type"`
	Session_id string `json:"session_id"`
	User_agent string `json:"user_agent"`
	User_id string `json:"user_id"`
	Amount float64 `json:"amount"`
	Fulfilled_at *time.Time `json:"fulfilled_at"`
	Required_level SecurityLevel `json:"required_level"`
}

type CreateVerificationRequest struct {
	 * `json:",omitempty"`
}

type UsernameErrorResponse struct {
	Error string `json:"error"`
}

type StepUpErrorResponse struct {
	Error string `json:"error"`
}

type OrganizationHandler struct {
	 **organization.Service `json:",omitempty"`
}

type JumioConfig struct {
	VerificationType string `json:"verificationType"`
	ApiToken string `json:"apiToken"`
	CallbackUrl string `json:"callbackUrl"`
	DataCenter string `json:"dataCenter"`
	EnableExtraction bool `json:"enableExtraction"`
	EnableLiveness bool `json:"enableLiveness"`
	Enabled bool `json:"enabled"`
	ApiSecret string `json:"apiSecret"`
	EnableAMLScreening bool `json:"enableAMLScreening"`
	EnabledCountries []string `json:"enabledCountries"`
	EnabledDocumentTypes []string `json:"enabledDocumentTypes"`
	PresetId string `json:"presetId"`
}

type testContext struct {
	 **http.Request `json:",omitempty"`
}

type SMSProviderConfig struct {
	Config  `json:"config"`
	From string `json:"from"`
	Provider string `json:"provider"`
}

type RunCheck_req struct {
	CheckType string `json:"checkType"`
}

type TwoFASendOTPResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
}

type TrustedContact struct {
	 *bool `json:",omitempty"`
}

type BackupAuthStatsResponse struct {
	Stats  `json:"stats"`
}

type NotificationErrorResponse struct {
	Error string `json:"error"`
}

type ConnectionsResponse struct {
	Connections  `json:"connections"`
}

type Enable_body struct {
	Method string `json:"method"`
	User_id string `json:"user_id"`
}

type CancelRecoveryRequest struct {
	Reason string `json:"reason"`
	SessionId xid.ID `json:"sessionId"`
}

type TrustedDevicesConfig struct {
	Default_expiry_days int `json:"default_expiry_days"`
	Enabled bool `json:"enabled"`
	Max_devices_per_user int `json:"max_devices_per_user"`
	Max_expiry_days int `json:"max_expiry_days"`
}

type NoOpEmailProvider struct {
}

type testRouter struct {
	 *string `json:",omitempty"`
}

type BackupAuthDocumentResponse struct {
	Id string `json:"id"`
}

type CompleteRecoveryResponse struct {
	CompletedAt time.Time `json:"completedAt"`
	Message string `json:"message"`
	SessionId xid.ID `json:"sessionId"`
	Status RecoveryStatus `json:"status"`
	Token string `json:"token"`
}

type GetRecoveryConfigResponse struct {
	EnabledMethods []RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
}

type ComplianceEvidence struct {
	CollectedBy string `json:"collectedBy"`
	ControlId string `json:"controlId"`
	CreatedAt time.Time `json:"createdAt"`
	Description string `json:"description"`
	FileHash string `json:"fileHash"`
	FileUrl string `json:"fileUrl"`
	Standard ComplianceStandard `json:"standard"`
	AppId string `json:"appId"`
	EvidenceType string `json:"evidenceType"`
	Id string `json:"id"`
	Metadata  `json:"metadata"`
	ProfileId string `json:"profileId"`
	Title string `json:"title"`
}

type MFAConfigResponse struct {
	Allowed_factor_types []string `json:"allowed_factor_types"`
	Enabled bool `json:"enabled"`
	Required_factor_count int `json:"required_factor_count"`
}

type SendOTP_body struct {
	User_id string `json:"user_id"`
}

type ComplianceProfile struct {
	AppId string `json:"appId"`
	Id string `json:"id"`
	PasswordExpiryDays int `json:"passwordExpiryDays"`
	RegularAccessReview bool `json:"regularAccessReview"`
	SessionIpBinding bool `json:"sessionIpBinding"`
	CreatedAt time.Time `json:"createdAt"`
	EncryptionInTransit bool `json:"encryptionInTransit"`
	PasswordRequireLower bool `json:"passwordRequireLower"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	SessionIdleTimeout int `json:"sessionIdleTimeout"`
	Status string `json:"status"`
	ComplianceContact string `json:"complianceContact"`
	EncryptionAtRest bool `json:"encryptionAtRest"`
	SessionMaxAge int `json:"sessionMaxAge"`
	AuditLogExport bool `json:"auditLogExport"`
	DpoContact string `json:"dpoContact"`
	Metadata  `json:"metadata"`
	PasswordRequireUpper bool `json:"passwordRequireUpper"`
	LeastPrivilege bool `json:"leastPrivilege"`
	MfaRequired bool `json:"mfaRequired"`
	RetentionDays int `json:"retentionDays"`
	DataResidency string `json:"dataResidency"`
	DetailedAuditTrail bool `json:"detailedAuditTrail"`
	Name string `json:"name"`
	PasswordMinLength int `json:"passwordMinLength"`
	RbacRequired bool `json:"rbacRequired"`
	Standards []ComplianceStandard `json:"standards"`
	UpdatedAt time.Time `json:"updatedAt"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol"`
}

type TemplateEngine struct {
	 *template.FuncMap `json:",omitempty"`
}

type ImpersonateUser_reqBody struct {
	Duration *time.Duration `json:"duration,omitempty"`
}

type SendVerificationCodeResponse struct {
	ExpiresAt time.Time `json:"expiresAt"`
	MaskedTarget string `json:"maskedTarget"`
	Message string `json:"message"`
	Sent bool `json:"sent"`
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

type ImpersonationStartResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Session_id string `json:"session_id"`
	Started_at string `json:"started_at"`
	Target_user_id string `json:"target_user_id"`
}

type AutoCleanupConfig struct {
	Enabled bool `json:"enabled"`
	Interval time.Duration `json:"interval"`
}

type MagicLinkSendResponse struct {
	Dev_url string `json:"dev_url"`
	Status string `json:"status"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type NoOpNotificationProvider struct {
}

type RolesResponse struct {
	Roles []*apikey.Role `json:"roles"`
}

type CreateAPIKey_reqBody struct {
	Rate_limit *int `json:"rate_limit,omitempty"`
	Scopes []string `json:"scopes"`
	Allowed_ips *[]string `json:"allowed_ips,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata * `json:"metadata,omitempty"`
	Name string `json:"name"`
	Permissions * `json:"permissions,omitempty"`
}

type RenderTemplate_req struct {
	Template string `json:"template"`
	Variables  `json:"variables"`
}

type Status struct {
	 *bool `json:",omitempty"`
}

type NoOpVideoProvider struct {
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

type StepUpEvaluationResponse struct {
	Reason string `json:"reason"`
	Required bool `json:"required"`
}

type KeyPair struct {
	 **rsa.PublicKey `json:",omitempty"`
}

type EmailProviderConfig struct {
	From string `json:"from"`
	From_name string `json:"from_name"`
	Provider string `json:"provider"`
	Reply_to string `json:"reply_to"`
	Config  `json:"config"`
}

type AuditServiceAdapter struct {
	 **audit.Service `json:",omitempty"`
}

type AuthResponse struct {
	Session  `json:"session"`
	Token string `json:"token"`
	User  `json:"user"`
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

type CheckSubResult struct {
	 *string `json:",omitempty"`
}

type PreviewTemplate_req struct {
	Variables  `json:"variables"`
}

type CreatePolicy_req struct {
	Content string `json:"content"`
	PolicyType string `json:"policyType"`
	Standard ComplianceStandard `json:"standard"`
	Title string `json:"title"`
	Version string `json:"version"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	Message string `json:"message"`
	Code string `json:"code"`
	Details  `json:"details"`
}

type TwoFAErrorResponse struct {
	Error string `json:"error"`
}

type SetupSecurityQuestionsRequest struct {
	Questions []SetupSecurityQuestionRequest `json:"questions"`
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

type VerifySecurityAnswersRequest struct {
	SessionId xid.ID `json:"sessionId"`
	Answers  `json:"answers"`
}

type BeginRegister_body struct {
	User_id string `json:"user_id"`
}

type ComplianceEvidencesResponse struct {
	Evidence []* `json:"evidence"`
}

type WebAuthnConfig struct {
	Timeout int `json:"timeout"`
	Attestation_preference string `json:"attestation_preference"`
	Authenticator_selection  `json:"authenticator_selection"`
	Enabled bool `json:"enabled"`
	Rp_display_name string `json:"rp_display_name"`
	Rp_id string `json:"rp_id"`
	Rp_origins []string `json:"rp_origins"`
}

type VerifyChallengeRequest struct {
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
	FactorId xid.ID `json:"factorId"`
	RememberDevice bool `json:"rememberDevice"`
	ChallengeId xid.ID `json:"challengeId"`
	Code string `json:"code"`
	Data  `json:"data"`
}

type Status_body struct {
	Device_id string `json:"device_id"`
	User_id string `json:"user_id"`
}

type AddCustomPermission_req struct {
	Name string `json:"name"`
	Category string `json:"category"`
	Description string `json:"description"`
}

type ProviderSessionRequest struct {
	 *string `json:",omitempty"`
}

type VerifyEnrolledFactorRequest struct {
	 *string `json:",omitempty"`
	Code string `json:"code"`
	Data  `json:"data"`
}

type UnbanUserRequest struct {
	- xid.ID `json:"-"`
	App_id xid.ID `json:"app_id"`
	Reason string `json:"reason"`
	User_id xid.ID `json:"user_id"`
	User_organization_id *xid.ID `json:"user_organization_id"`
}

type RemoveTrustedContactRequest struct {
	ContactId xid.ID `json:"contactId"`
}

type ImpersonationVerifyResponse struct {
	Impersonator_id string `json:"impersonator_id"`
	Is_impersonating bool `json:"is_impersonating"`
	Target_user_id string `json:"target_user_id"`
}

type StripeIdentityProvider struct {
	 *bool `json:",omitempty"`
}

type EmailServiceAdapter struct {
	 **notification.Service `json:",omitempty"`
}

type AnonymousErrorResponse struct {
	Error string `json:"error"`
}

type UsernameSignInResponse struct {
	Token string `json:"token"`
	User  `json:"user"`
	Session  `json:"session"`
}

type Send_body struct {
	Email string `json:"email"`
}

type TrustDeviceRequest struct {
	DeviceId string `json:"deviceId"`
	Metadata  `json:"metadata"`
	Name string `json:"name"`
}

type OTPSentResponse struct {
	Code string `json:"code"`
	Status string `json:"status"`
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

type VideoVerificationConfig struct {
	Enabled bool `json:"enabled"`
	LivenessThreshold float64 `json:"livenessThreshold"`
	RecordSessions bool `json:"recordSessions"`
	RecordingRetention time.Duration `json:"recordingRetention"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireLivenessCheck bool `json:"requireLivenessCheck"`
	RequireScheduling bool `json:"requireScheduling"`
	MinScheduleAdvance time.Duration `json:"minScheduleAdvance"`
	Provider string `json:"provider"`
	SessionDuration time.Duration `json:"sessionDuration"`
}

type NotificationResponse struct {
	Notification  `json:"notification"`
}

type FinishRegister_body struct {
	User_id string `json:"user_id"`
	Credential_id string `json:"credential_id"`
}

type CreateAPIKeyResponse struct {
	Api_key *apikey.APIKey `json:"api_key"`
	Message string `json:"message"`
}

type SetActive_body struct {
	Id string `json:"id"`
}

type ResetUserMFAResponse struct {
	FactorsReset int `json:"factorsReset"`
	Message string `json:"message"`
	Success bool `json:"success"`
	DevicesRevoked int `json:"devicesRevoked"`
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

type BackupAuthSessionsResponse struct {
	Sessions []* `json:"sessions"`
}

type EmailOTPSendResponse struct {
	Dev_otp string `json:"dev_otp"`
	Status string `json:"status"`
}

type Email struct {
	 *string `json:",omitempty"`
}

type ComplianceStatusDetailsResponse struct {
	Status string `json:"status"`
}

type CallbackResult struct {
	 *string `json:",omitempty"`
}

type ImpersonationSession struct {
}

// Session represents User session
type Session struct {
	CreatedAt string `json:"createdAt"`
	Id string `json:"id"`
	UserId string `json:"userId"`
	Token string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	IpAddress *string `json:"ipAddress,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
}

type SignIn_body struct {
}

type PasskeyRegistrationOptionsResponse struct {
	Options  `json:"options"`
}

type SMSConfig struct {
	Code_expiry_minutes int `json:"code_expiry_minutes"`
	Code_length int `json:"code_length"`
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	Rate_limit *RateLimitConfig `json:"rate_limit"`
	Template_id string `json:"template_id"`
}

type BackupCodeFactorAdapter struct {
	 **twofa.Service `json:",omitempty"`
}

