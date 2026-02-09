package backupauth

import "time"

// Config holds the backup authentication plugin configuration.
type Config struct {
	// Enable backup authentication plugin
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Recovery codes configuration
	RecoveryCodes RecoveryCodesConfig `json:"recoveryCodes" yaml:"recoveryCodes"`

	// Security questions configuration
	SecurityQuestions SecurityQuestionsConfig `json:"securityQuestions" yaml:"securityQuestions"`

	// Trusted contacts configuration
	TrustedContacts TrustedContactsConfig `json:"trustedContacts" yaml:"trustedContacts"`

	// Email verification fallback
	EmailVerification EmailVerificationConfig `json:"emailVerification" yaml:"emailVerification"`

	// SMS verification fallback
	SMSVerification SMSVerificationConfig `json:"smsVerification" yaml:"smsVerification"`

	// Video verification
	VideoVerification VideoVerificationConfig `json:"videoVerification" yaml:"videoVerification"`

	// Document verification
	DocumentVerification DocumentVerificationConfig `json:"documentVerification" yaml:"documentVerification"`

	// Multi-step recovery flows
	MultiStepRecovery MultiStepRecoveryConfig `json:"multiStepRecovery" yaml:"multiStepRecovery"`

	// Risk assessment
	RiskAssessment RiskAssessmentConfig `json:"riskAssessment" yaml:"riskAssessment"`

	// Rate limiting
	RateLimiting RateLimitingConfig `json:"rateLimiting" yaml:"rateLimiting"`

	// Audit and logging
	Audit AuditConfig `json:"audit" yaml:"audit"`

	// Notifications
	Notifications NotificationsConfig `json:"notifications" yaml:"notifications"`
}

// RecoveryCodesConfig configures recovery codes.
type RecoveryCodesConfig struct {
	Enabled    bool `json:"enabled"    yaml:"enabled"`
	CodeCount  int  `json:"codeCount"  yaml:"codeCount"`
	CodeLength int  `json:"codeLength" yaml:"codeLength"`

	// Automatically regenerate after use
	AutoRegenerate  bool `json:"autoRegenerate"  yaml:"autoRegenerate"`
	RegenerateCount int  `json:"regenerateCount" yaml:"regenerateCount"` // New codes to generate

	// Format: alphanumeric, numeric, hex
	Format string `json:"format" yaml:"format"`

	// Allow printing/downloading
	AllowPrint    bool `json:"allowPrint"    yaml:"allowPrint"`
	AllowDownload bool `json:"allowDownload" yaml:"allowDownload"`
}

// SecurityQuestionsConfig configures security questions.
type SecurityQuestionsConfig struct {
	Enabled           bool `json:"enabled"           yaml:"enabled"`
	MinimumQuestions  int  `json:"minimumQuestions"  yaml:"minimumQuestions"`
	RequiredToRecover int  `json:"requiredToRecover" yaml:"requiredToRecover"`

	// Allow custom questions
	AllowCustomQuestions bool     `json:"allowCustomQuestions" yaml:"allowCustomQuestions"`
	PredefinedQuestions  []string `json:"predefinedQuestions"  yaml:"predefinedQuestions"`

	// Security
	CaseSensitive   bool          `json:"caseSensitive"   yaml:"caseSensitive"`
	MaxAnswerLength int           `json:"maxAnswerLength" yaml:"maxAnswerLength"`
	MaxAttempts     int           `json:"maxAttempts"     yaml:"maxAttempts"`
	LockoutDuration time.Duration `json:"lockoutDuration" yaml:"lockoutDuration"`

	// Answer complexity
	RequireMinLength    int  `json:"requireMinLength"    yaml:"requireMinLength"`
	ForbidCommonAnswers bool `json:"forbidCommonAnswers" yaml:"forbidCommonAnswers"`
}

// TrustedContactsConfig configures trusted contacts.
type TrustedContactsConfig struct {
	Enabled           bool `json:"enabled"           yaml:"enabled"`
	MinimumContacts   int  `json:"minimumContacts"   yaml:"minimumContacts"`
	MaximumContacts   int  `json:"maximumContacts"   yaml:"maximumContacts"`
	RequiredToRecover int  `json:"requiredToRecover" yaml:"requiredToRecover"`

	// Verification
	RequireVerification bool          `json:"requireVerification" yaml:"requireVerification"`
	VerificationExpiry  time.Duration `json:"verificationExpiry"  yaml:"verificationExpiry"`

	// Contact methods
	AllowEmailContacts bool `json:"allowEmailContacts" yaml:"allowEmailContacts"`
	AllowPhoneContacts bool `json:"allowPhoneContacts" yaml:"allowPhoneContacts"`

	// Notification throttling
	CooldownPeriod         time.Duration `json:"cooldownPeriod"         yaml:"cooldownPeriod"`
	MaxNotificationsPerDay int           `json:"maxNotificationsPerDay" yaml:"maxNotificationsPerDay"`
}

// EmailVerificationConfig configures email verification fallback.
type EmailVerificationConfig struct {
	Enabled     bool          `json:"enabled"     yaml:"enabled"`
	CodeExpiry  time.Duration `json:"codeExpiry"  yaml:"codeExpiry"`
	CodeLength  int           `json:"codeLength"  yaml:"codeLength"`
	MaxAttempts int           `json:"maxAttempts" yaml:"maxAttempts"`

	// Require email ownership proof
	RequireEmailProof bool `json:"requireEmailProof" yaml:"requireEmailProof"`

	// Template configuration
	EmailTemplate string `json:"emailTemplate" yaml:"emailTemplate"`
	FromAddress   string `json:"fromAddress"   yaml:"fromAddress"`
	FromName      string `json:"fromName"      yaml:"fromName"`
}

// SMSVerificationConfig configures SMS verification fallback.
type SMSVerificationConfig struct {
	Enabled     bool          `json:"enabled"     yaml:"enabled"`
	CodeExpiry  time.Duration `json:"codeExpiry"  yaml:"codeExpiry"`
	CodeLength  int           `json:"codeLength"  yaml:"codeLength"`
	MaxAttempts int           `json:"maxAttempts" yaml:"maxAttempts"`

	// Provider configuration
	Provider string `json:"provider" yaml:"provider"` // twilio, vonage, aws_sns

	// Template configuration
	MessageTemplate string `json:"messageTemplate" yaml:"messageTemplate"`

	// Rate limiting (SMS costs money)
	MaxSMSPerDay   int           `json:"maxSmsPerDay"   yaml:"maxSmsPerDay"`
	CooldownPeriod time.Duration `json:"cooldownPeriod" yaml:"cooldownPeriod"`
}

// VideoVerificationConfig configures video verification.
type VideoVerificationConfig struct {
	Enabled  bool   `json:"enabled"  yaml:"enabled"`
	Provider string `json:"provider" yaml:"provider"` // zoom, teams, custom

	// Scheduling
	RequireScheduling  bool          `json:"requireScheduling"  yaml:"requireScheduling"`
	MinScheduleAdvance time.Duration `json:"minScheduleAdvance" yaml:"minScheduleAdvance"`
	SessionDuration    time.Duration `json:"sessionDuration"    yaml:"sessionDuration"`

	// Verification requirements
	RequireLivenessCheck bool    `json:"requireLivenessCheck" yaml:"requireLivenessCheck"`
	LivenessThreshold    float64 `json:"livenessThreshold"    yaml:"livenessThreshold"`

	// Recording
	RecordSessions     bool          `json:"recordSessions"     yaml:"recordSessions"`
	RecordingRetention time.Duration `json:"recordingRetention" yaml:"recordingRetention"`

	// Admin review
	RequireAdminReview bool `json:"requireAdminReview" yaml:"requireAdminReview"`
}

// DocumentVerificationConfig configures document verification.
type DocumentVerificationConfig struct {
	Enabled  bool   `json:"enabled"  yaml:"enabled"`
	Provider string `json:"provider" yaml:"provider"` // stripe_identity, onfido, jumio

	// Accepted document types
	AcceptedDocuments []string `json:"acceptedDocuments" yaml:"acceptedDocuments"`

	// Requirements
	RequireSelfie    bool `json:"requireSelfie"    yaml:"requireSelfie"`
	RequireBothSides bool `json:"requireBothSides" yaml:"requireBothSides"`

	// Verification
	MinConfidenceScore  float64 `json:"minConfidenceScore"  yaml:"minConfidenceScore"`
	RequireManualReview bool    `json:"requireManualReview" yaml:"requireManualReview"`

	// Storage
	StorageProvider string        `json:"storageProvider" yaml:"storageProvider"` // s3, gcs, azure
	StoragePath     string        `json:"storagePath"     yaml:"storagePath"`
	RetentionPeriod time.Duration `json:"retentionPeriod" yaml:"retentionPeriod"`

	// Encryption
	EncryptAtRest bool   `json:"encryptAtRest" yaml:"encryptAtRest"`
	EncryptionKey string `json:"encryptionKey" yaml:"encryptionKey"`
}

// MultiStepRecoveryConfig configures multi-step recovery flows.
type MultiStepRecoveryConfig struct {
	Enabled      bool `json:"enabled"      yaml:"enabled"`
	MinimumSteps int  `json:"minimumSteps" yaml:"minimumSteps"`

	// Step requirements by risk level
	LowRiskSteps    []RecoveryMethod `json:"lowRiskSteps"    yaml:"lowRiskSteps"`
	MediumRiskSteps []RecoveryMethod `json:"mediumRiskSteps" yaml:"mediumRiskSteps"`
	HighRiskSteps   []RecoveryMethod `json:"highRiskSteps"   yaml:"highRiskSteps"`

	// Flow configuration
	AllowUserChoice bool          `json:"allowUserChoice" yaml:"allowUserChoice"`
	SessionExpiry   time.Duration `json:"sessionExpiry"   yaml:"sessionExpiry"`
	AllowStepSkip   bool          `json:"allowStepSkip"   yaml:"allowStepSkip"`

	// Completion
	RequireAdminApproval bool `json:"requireAdminApproval" yaml:"requireAdminApproval"`
}

// RiskAssessmentConfig configures risk scoring.
type RiskAssessmentConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Risk factors and weights
	NewDeviceWeight   float64 `json:"newDeviceWeight"   yaml:"newDeviceWeight"`
	NewLocationWeight float64 `json:"newLocationWeight" yaml:"newLocationWeight"`
	NewIPWeight       float64 `json:"newIpWeight"       yaml:"newIpWeight"`
	VelocityWeight    float64 `json:"velocityWeight"    yaml:"velocityWeight"`
	HistoryWeight     float64 `json:"historyWeight"     yaml:"historyWeight"`

	// Thresholds
	LowRiskThreshold    float64 `json:"lowRiskThreshold"    yaml:"lowRiskThreshold"`
	MediumRiskThreshold float64 `json:"mediumRiskThreshold" yaml:"mediumRiskThreshold"`
	HighRiskThreshold   float64 `json:"highRiskThreshold"   yaml:"highRiskThreshold"`

	// Actions
	BlockHighRisk      bool    `json:"blockHighRisk"      yaml:"blockHighRisk"`
	RequireReviewAbove float64 `json:"requireReviewAbove" yaml:"requireReviewAbove"`
}

// RateLimitingConfig configures rate limiting.
type RateLimitingConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Per-user limits
	MaxAttemptsPerHour int `json:"maxAttemptsPerHour" yaml:"maxAttemptsPerHour"`
	MaxAttemptsPerDay  int `json:"maxAttemptsPerDay"  yaml:"maxAttemptsPerDay"`

	// Lockout
	LockoutAfterAttempts int           `json:"lockoutAfterAttempts" yaml:"lockoutAfterAttempts"`
	LockoutDuration      time.Duration `json:"lockoutDuration"      yaml:"lockoutDuration"`
	ExponentialBackoff   bool          `json:"exponentialBackoff"   yaml:"exponentialBackoff"`

	// Per-IP limits (prevent abuse)
	MaxAttemptsPerIP int           `json:"maxAttemptsPerIp" yaml:"maxAttemptsPerIp"`
	IPCooldownPeriod time.Duration `json:"ipCooldownPeriod" yaml:"ipCooldownPeriod"`
}

// AuditConfig configures audit logging.
type AuditConfig struct {
	Enabled        bool `json:"enabled"        yaml:"enabled"`
	LogAllAttempts bool `json:"logAllAttempts" yaml:"logAllAttempts"`
	LogSuccessful  bool `json:"logSuccessful"  yaml:"logSuccessful"`
	LogFailed      bool `json:"logFailed"      yaml:"logFailed"`

	// Immutability
	ImmutableLogs bool `json:"immutableLogs" yaml:"immutableLogs"`

	// Retention
	RetentionDays   int           `json:"retentionDays"   yaml:"retentionDays"`
	ArchiveOldLogs  bool          `json:"archiveOldLogs"  yaml:"archiveOldLogs"`
	ArchiveInterval time.Duration `json:"archiveInterval" yaml:"archiveInterval"`

	// Detailed logging
	LogIPAddress  bool `json:"logIpAddress"  yaml:"logIpAddress"`
	LogUserAgent  bool `json:"logUserAgent"  yaml:"logUserAgent"`
	LogDeviceInfo bool `json:"logDeviceInfo" yaml:"logDeviceInfo"`
}

// NotificationsConfig configures notifications.
type NotificationsConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// When to notify user
	NotifyOnRecoveryStart    bool `json:"notifyOnRecoveryStart"    yaml:"notifyOnRecoveryStart"`
	NotifyOnRecoveryComplete bool `json:"notifyOnRecoveryComplete" yaml:"notifyOnRecoveryComplete"`
	NotifyOnRecoveryFailed   bool `json:"notifyOnRecoveryFailed"   yaml:"notifyOnRecoveryFailed"`

	// Admin notifications
	NotifyAdminOnHighRisk     bool `json:"notifyAdminOnHighRisk"     yaml:"notifyAdminOnHighRisk"`
	NotifyAdminOnReviewNeeded bool `json:"notifyAdminOnReviewNeeded" yaml:"notifyAdminOnReviewNeeded"`

	// Channels
	Channels []string `json:"channels" yaml:"channels"` // email, sms, webhook

	// Security officer notifications
	SecurityOfficerEmail string `json:"securityOfficerEmail" yaml:"securityOfficerEmail"`
}

// DefaultConfig returns the default backup authentication configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		RecoveryCodes: RecoveryCodesConfig{
			Enabled:         true,
			CodeCount:       10,
			CodeLength:      12,
			AutoRegenerate:  true,
			RegenerateCount: 5,
			Format:          "alphanumeric",
			AllowPrint:      true,
			AllowDownload:   true,
		},
		SecurityQuestions: SecurityQuestionsConfig{
			Enabled:              true,
			MinimumQuestions:     3,
			RequiredToRecover:    2,
			AllowCustomQuestions: true,
			PredefinedQuestions: []string{
				"What was the name of your first pet?",
				"What city were you born in?",
				"What is your mother's maiden name?",
				"What was the name of your elementary school?",
				"What was the make of your first car?",
				"What is your favorite book?",
				"What was your childhood nickname?",
				"In what city did you meet your spouse/partner?",
			},
			CaseSensitive:       false,
			MaxAnswerLength:     100,
			MaxAttempts:         3,
			LockoutDuration:     30 * time.Minute,
			RequireMinLength:    3,
			ForbidCommonAnswers: true,
		},
		TrustedContacts: TrustedContactsConfig{
			Enabled:                true,
			MinimumContacts:        1,
			MaximumContacts:        5,
			RequiredToRecover:      1,
			RequireVerification:    true,
			VerificationExpiry:     7 * 24 * time.Hour,
			AllowEmailContacts:     true,
			AllowPhoneContacts:     true,
			CooldownPeriod:         1 * time.Hour,
			MaxNotificationsPerDay: 3,
		},
		EmailVerification: EmailVerificationConfig{
			Enabled:           true,
			CodeExpiry:        15 * time.Minute,
			CodeLength:        6,
			MaxAttempts:       5,
			RequireEmailProof: true,
			EmailTemplate:     "recovery_email_verification",
			FromAddress:       "noreply@authsome.local",
			FromName:          "AuthSome Security",
		},
		SMSVerification: SMSVerificationConfig{
			Enabled:         true,
			CodeExpiry:      10 * time.Minute,
			CodeLength:      6,
			MaxAttempts:     3,
			Provider:        "twilio",
			MessageTemplate: "Your recovery code is: {{code}}. Valid for {{expiry}} minutes.",
			MaxSMSPerDay:    5,
			CooldownPeriod:  5 * time.Minute,
		},
		VideoVerification: VideoVerificationConfig{
			Enabled:              false, // Enterprise feature
			Provider:             "zoom",
			RequireScheduling:    true,
			MinScheduleAdvance:   2 * time.Hour,
			SessionDuration:      30 * time.Minute,
			RequireLivenessCheck: true,
			LivenessThreshold:    0.85,
			RecordSessions:       true,
			RecordingRetention:   90 * 24 * time.Hour,
			RequireAdminReview:   true,
		},
		DocumentVerification: DocumentVerificationConfig{
			Enabled:  false, // Enterprise feature
			Provider: "stripe_identity",
			AcceptedDocuments: []string{
				"passport",
				"drivers_license",
				"national_id",
			},
			RequireSelfie:       true,
			RequireBothSides:    true,
			MinConfidenceScore:  0.85,
			RequireManualReview: false,
			StorageProvider:     "s3",
			StoragePath:         "/var/lib/authsome/backup/documents",
			RetentionPeriod:     90 * 24 * time.Hour,
			EncryptAtRest:       true,
		},
		MultiStepRecovery: MultiStepRecoveryConfig{
			Enabled:      true,
			MinimumSteps: 2,
			LowRiskSteps: []RecoveryMethod{
				RecoveryMethodCodes,
				RecoveryMethodEmail,
			},
			MediumRiskSteps: []RecoveryMethod{
				RecoveryMethodSecurityQ,
				RecoveryMethodEmail,
				RecoveryMethodSMS,
			},
			HighRiskSteps: []RecoveryMethod{
				RecoveryMethodSecurityQ,
				RecoveryMethodTrustedContact,
				RecoveryMethodVideo,
			},
			AllowUserChoice:      true,
			SessionExpiry:        30 * time.Minute,
			AllowStepSkip:        false,
			RequireAdminApproval: false,
		},
		RiskAssessment: RiskAssessmentConfig{
			Enabled:             true,
			NewDeviceWeight:     0.25,
			NewLocationWeight:   0.20,
			NewIPWeight:         0.15,
			VelocityWeight:      0.20,
			HistoryWeight:       0.20,
			LowRiskThreshold:    30.0,
			MediumRiskThreshold: 60.0,
			HighRiskThreshold:   80.0,
			BlockHighRisk:       false,
			RequireReviewAbove:  85.0,
		},
		RateLimiting: RateLimitingConfig{
			Enabled:              true,
			MaxAttemptsPerHour:   5,
			MaxAttemptsPerDay:    10,
			LockoutAfterAttempts: 5,
			LockoutDuration:      24 * time.Hour,
			ExponentialBackoff:   true,
			MaxAttemptsPerIP:     20,
			IPCooldownPeriod:     1 * time.Hour,
		},
		Audit: AuditConfig{
			Enabled:         true,
			LogAllAttempts:  true,
			LogSuccessful:   true,
			LogFailed:       true,
			ImmutableLogs:   true,
			RetentionDays:   2555, // 7 years
			ArchiveOldLogs:  true,
			ArchiveInterval: 90 * 24 * time.Hour,
			LogIPAddress:    true,
			LogUserAgent:    true,
			LogDeviceInfo:   true,
		},
		Notifications: NotificationsConfig{
			Enabled:                   true,
			NotifyOnRecoveryStart:     true,
			NotifyOnRecoveryComplete:  true,
			NotifyOnRecoveryFailed:    true,
			NotifyAdminOnHighRisk:     true,
			NotifyAdminOnReviewNeeded: true,
			Channels:                  []string{"email"},
			SecurityOfficerEmail:      "",
		},
	}
}

// Validate ensures configuration has sensible defaults.
func (c *Config) Validate() {
	// Recovery codes validation
	if c.RecoveryCodes.CodeCount == 0 {
		c.RecoveryCodes.CodeCount = 10
	}

	if c.RecoveryCodes.CodeLength == 0 {
		c.RecoveryCodes.CodeLength = 12
	}

	if c.RecoveryCodes.Format == "" {
		c.RecoveryCodes.Format = "alphanumeric"
	}

	// Security questions validation
	if c.SecurityQuestions.MinimumQuestions == 0 {
		c.SecurityQuestions.MinimumQuestions = 3
	}

	if c.SecurityQuestions.RequiredToRecover == 0 {
		c.SecurityQuestions.RequiredToRecover = 2
	}

	if c.SecurityQuestions.MaxAttempts == 0 {
		c.SecurityQuestions.MaxAttempts = 3
	}

	// Trusted contacts validation
	if c.TrustedContacts.MaximumContacts == 0 {
		c.TrustedContacts.MaximumContacts = 5
	}

	// Email verification validation
	if c.EmailVerification.CodeExpiry == 0 {
		c.EmailVerification.CodeExpiry = 15 * time.Minute
	}

	if c.EmailVerification.CodeLength == 0 {
		c.EmailVerification.CodeLength = 6
	}

	// SMS verification validation
	if c.SMSVerification.CodeExpiry == 0 {
		c.SMSVerification.CodeExpiry = 10 * time.Minute
	}

	if c.SMSVerification.CodeLength == 0 {
		c.SMSVerification.CodeLength = 6
	}

	// Multi-step recovery validation
	if c.MultiStepRecovery.MinimumSteps == 0 {
		c.MultiStepRecovery.MinimumSteps = 2
	}

	if c.MultiStepRecovery.SessionExpiry == 0 {
		c.MultiStepRecovery.SessionExpiry = 30 * time.Minute
	}

	// Rate limiting validation
	if c.RateLimiting.MaxAttemptsPerDay == 0 {
		c.RateLimiting.MaxAttemptsPerDay = 10
	}

	if c.RateLimiting.LockoutDuration == 0 {
		c.RateLimiting.LockoutDuration = 24 * time.Hour
	}

	// Audit validation
	if c.Audit.RetentionDays == 0 {
		c.Audit.RetentionDays = 2555 // 7 years
	}
}
