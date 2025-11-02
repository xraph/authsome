package idverification

import (
	"time"
)

// Config holds the identity verification plugin configuration
type Config struct {
	// General settings
	Enabled               bool          `json:"enabled" yaml:"enabled"`
	DefaultProvider       string        `json:"defaultProvider" yaml:"defaultProvider"` // onfido, jumio, stripe_identity
	SessionExpiryDuration time.Duration `json:"sessionExpiryDuration" yaml:"sessionExpiryDuration"`
	VerificationExpiry    time.Duration `json:"verificationExpiry" yaml:"verificationExpiry"` // How long verification is valid
	
	// Required checks
	RequireDocumentVerification bool `json:"requireDocumentVerification" yaml:"requireDocumentVerification"`
	RequireLivenessDetection    bool `json:"requireLivenessDetection" yaml:"requireLivenessDetection"`
	RequireAgeVerification      bool `json:"requireAgeVerification" yaml:"requireAgeVerification"`
	RequireAMLScreening         bool `json:"requireAMLScreening" yaml:"requireAMLScreening"`
	MinimumAge                  int  `json:"minimumAge" yaml:"minimumAge"` // For age verification
	
	// Accepted document types
	AcceptedDocuments []string `json:"acceptedDocuments" yaml:"acceptedDocuments"` // passport, drivers_license, national_id
	AcceptedCountries []string `json:"acceptedCountries" yaml:"acceptedCountries"` // ISO 3166-1 alpha-2 codes, empty = all
	
	// Risk scoring
	MaxAllowedRiskScore int    `json:"maxAllowedRiskScore" yaml:"maxAllowedRiskScore"` // 0-100
	AutoRejectHighRisk  bool   `json:"autoRejectHighRisk" yaml:"autoRejectHighRisk"`
	MinConfidenceScore  int    `json:"minConfidenceScore" yaml:"minConfidenceScore"` // Minimum confidence to pass
	
	// Document retention
	RetainDocuments         bool          `json:"retainDocuments" yaml:"retainDocuments"`
	DocumentRetentionPeriod time.Duration `json:"documentRetentionPeriod" yaml:"documentRetentionPeriod"`
	AutoDeleteAfterExpiry   bool          `json:"autoDeleteAfterExpiry" yaml:"autoDeleteAfterExpiry"`
	
	// Webhook configuration
	WebhooksEnabled    bool     `json:"webhooksEnabled" yaml:"webhooksEnabled"`
	WebhookURL         string   `json:"webhookUrl" yaml:"webhookUrl"`
	WebhookEvents      []string `json:"webhookEvents" yaml:"webhookEvents"` // verification.completed, verification.failed, etc.
	WebhookSecret      string   `json:"webhookSecret" yaml:"webhookSecret"`
	WebhookRetryCount  int      `json:"webhookRetryCount" yaml:"webhookRetryCount"`
	
	// Callback URLs (defaults)
	DefaultSuccessURL string `json:"defaultSuccessUrl" yaml:"defaultSuccessUrl"`
	DefaultCancelURL  string `json:"defaultCancelUrl" yaml:"defaultCancelUrl"`
	
	// Provider configurations
	Onfido         OnfidoConfig         `json:"onfido" yaml:"onfido"`
	Jumio          JumioConfig          `json:"jumio" yaml:"jumio"`
	StripeIdentity StripeIdentityConfig `json:"stripeIdentity" yaml:"stripeIdentity"`
	
	// Features
	EnableManualReview      bool `json:"enableManualReview" yaml:"enableManualReview"` // Allow manual review of failed verifications
	EnableReverification    bool `json:"enableReverification" yaml:"enableReverification"` // Allow re-verification
	MaxVerificationAttempts int  `json:"maxVerificationAttempts" yaml:"maxVerificationAttempts"`
	
	// Compliance
	EnableAuditLog       bool   `json:"enableAuditLog" yaml:"enableAuditLog"`
	ComplianceMode       string `json:"complianceMode" yaml:"complianceMode"` // standard, strict, custom
	GDPRCompliant        bool   `json:"gdprCompliant" yaml:"gdprCompliant"`
	DataResidency        string `json:"dataResidency" yaml:"dataResidency"` // us, eu, uk, global
	
	// Rate limiting
	RateLimitEnabled          bool `json:"rateLimitEnabled" yaml:"rateLimitEnabled"`
	MaxVerificationsPerHour   int  `json:"maxVerificationsPerHour" yaml:"maxVerificationsPerHour"`
	MaxVerificationsPerDay    int  `json:"maxVerificationsPerDay" yaml:"maxVerificationsPerDay"`
	
	// Metadata
	CustomFields map[string]interface{} `json:"customFields" yaml:"customFields"`
}

// OnfidoConfig holds Onfido-specific configuration
type OnfidoConfig struct {
	Enabled       bool   `json:"enabled" yaml:"enabled"`
	APIToken      string `json:"apiToken" yaml:"apiToken"`
	Region        string `json:"region" yaml:"region"` // us, eu, ca
	WebhookToken  string `json:"webhookToken" yaml:"webhookToken"`
	
	// Check configuration
	DocumentCheck DocumentCheckConfig `json:"documentCheck" yaml:"documentCheck"`
	FacialCheck   FacialCheckConfig   `json:"facialCheck" yaml:"facialCheck"`
	
	// Workflow
	WorkflowID string `json:"workflowId" yaml:"workflowId"` // Predefined Onfido workflow
	
	// Reports
	IncludeDocumentReport bool `json:"includeDocumentReport" yaml:"includeDocumentReport"`
	IncludeFacialReport   bool `json:"includeFacialReport" yaml:"includeFacialReport"`
	IncludeWatchlistReport bool `json:"includeWatchlistReport" yaml:"includeWatchlistReport"`
}

// DocumentCheckConfig configures document verification
type DocumentCheckConfig struct {
	Enabled            bool `json:"enabled" yaml:"enabled"`
	ValidateExpiry     bool `json:"validateExpiry" yaml:"validateExpiry"`
	ValidateDataConsistency bool `json:"validateDataConsistency" yaml:"validateDataConsistency"`
	ExtractData        bool `json:"extractData" yaml:"extractData"`
}

// FacialCheckConfig configures facial/liveness verification
type FacialCheckConfig struct {
	Enabled         bool   `json:"enabled" yaml:"enabled"`
	Variant         string `json:"variant" yaml:"variant"` // standard, video
	MotionCapture   bool   `json:"motionCapture" yaml:"motionCapture"`
}

// JumioConfig holds Jumio-specific configuration
type JumioConfig struct {
	Enabled       bool   `json:"enabled" yaml:"enabled"`
	APIToken      string `json:"apiToken" yaml:"apiToken"`
	APISecret     string `json:"apiSecret" yaml:"apiSecret"`
	DataCenter    string `json:"dataCenter" yaml:"dataCenter"` // us, eu, sg
	
	// Verification settings
	VerificationType   string   `json:"verificationType" yaml:"verificationType"` // identity, document, similarity
	PresetID           string   `json:"presetId" yaml:"presetId"` // Jumio preset configuration
	
	// Document settings
	EnabledDocumentTypes []string `json:"enabledDocumentTypes" yaml:"enabledDocumentTypes"`
	EnabledCountries     []string `json:"enabledCountries" yaml:"enabledCountries"`
	
	// Features
	EnableLiveness     bool `json:"enableLiveness" yaml:"enableLiveness"`
	EnableAMLScreening bool `json:"enableAMLScreening" yaml:"enableAMLScreening"`
	EnableExtraction   bool `json:"enableExtraction" yaml:"enableExtraction"`
	
	// Callback
	CallbackURL string `json:"callbackUrl" yaml:"callbackUrl"`
}

// StripeIdentityConfig holds Stripe Identity-specific configuration
type StripeIdentityConfig struct {
	Enabled       bool   `json:"enabled" yaml:"enabled"`
	APIKey        string `json:"apiKey" yaml:"apiKey"`
	WebhookSecret string `json:"webhookSecret" yaml:"webhookSecret"`
	
	// Verification options
	RequireLiveCapture bool     `json:"requireLiveCapture" yaml:"requireLiveCapture"`
	AllowedTypes       []string `json:"allowedTypes" yaml:"allowedTypes"` // document, id_number
	
	// Document options
	RequireMatchingSelfie bool `json:"requireMatchingSelfie" yaml:"requireMatchingSelfie"`
	
	// Return URL
	ReturnURL string `json:"returnUrl" yaml:"returnUrl"`
	
	// Testing
	UseMock bool `json:"useMock" yaml:"useMock"` // Use mock implementation for testing/development
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		Enabled:               true,
		DefaultProvider:       "onfido",
		SessionExpiryDuration: 24 * time.Hour,
		VerificationExpiry:    365 * 24 * time.Hour, // 1 year
		
		RequireDocumentVerification: true,
		RequireLivenessDetection:    true,
		RequireAgeVerification:      false,
		RequireAMLScreening:         false,
		MinimumAge:                  18,
		
		AcceptedDocuments: []string{"passport", "drivers_license", "national_id"},
		AcceptedCountries: []string{}, // Empty = all countries
		
		MaxAllowedRiskScore: 70,
		AutoRejectHighRisk:  true,
		MinConfidenceScore:  80,
		
		RetainDocuments:         true,
		DocumentRetentionPeriod: 90 * 24 * time.Hour,
		AutoDeleteAfterExpiry:   true,
		
		WebhooksEnabled:   true,
		WebhookEvents:     []string{"verification.completed", "verification.failed", "verification.expired"},
		WebhookRetryCount: 3,
		
		EnableManualReview:      true,
		EnableReverification:    true,
		MaxVerificationAttempts: 3,
		
		EnableAuditLog: true,
		ComplianceMode: "standard",
		GDPRCompliant:  true,
		DataResidency:  "global",
		
		RateLimitEnabled:        true,
		MaxVerificationsPerHour: 10,
		MaxVerificationsPerDay:  50,
		
		Onfido: OnfidoConfig{
			Enabled: false,
			Region:  "eu",
			DocumentCheck: DocumentCheckConfig{
				Enabled:                 true,
				ValidateExpiry:          true,
				ValidateDataConsistency: true,
				ExtractData:             true,
			},
			FacialCheck: FacialCheckConfig{
				Enabled:       true,
				Variant:       "video",
				MotionCapture: true,
			},
			IncludeDocumentReport:  true,
			IncludeFacialReport:    true,
			IncludeWatchlistReport: true,
		},
		
		Jumio: JumioConfig{
			Enabled:              false,
			DataCenter:           "us",
			VerificationType:     "identity",
			EnabledDocumentTypes: []string{"PASSPORT", "DRIVING_LICENSE", "ID_CARD"},
			EnableLiveness:       true,
			EnableAMLScreening:   false,
			EnableExtraction:     true,
		},
		
		StripeIdentity: StripeIdentityConfig{
			Enabled:               false,
			RequireLiveCapture:    true,
			AllowedTypes:          []string{"document"},
			RequireMatchingSelfie: true,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	
	// Check that at least one provider is enabled
	if !c.Onfido.Enabled && !c.Jumio.Enabled && !c.StripeIdentity.Enabled {
		return ErrNoProviderEnabled
	}
	
	// Validate default provider
	if c.DefaultProvider == "" {
		return ErrInvalidDefaultProvider
	}
	
	switch c.DefaultProvider {
	case "onfido":
		if !c.Onfido.Enabled {
			return ErrProviderNotEnabled
		}
		if c.Onfido.APIToken == "" {
			return ErrMissingAPIToken
		}
	case "jumio":
		if !c.Jumio.Enabled {
			return ErrProviderNotEnabled
		}
		if c.Jumio.APIToken == "" || c.Jumio.APISecret == "" {
			return ErrMissingAPICredentials
		}
	case "stripe_identity":
		if !c.StripeIdentity.Enabled {
			return ErrProviderNotEnabled
		}
		if c.StripeIdentity.APIKey == "" {
			return ErrMissingAPIKey
		}
	default:
		return ErrUnsupportedProvider
	}
	
	// Validate risk scores
	if c.MaxAllowedRiskScore < 0 || c.MaxAllowedRiskScore > 100 {
		return ErrInvalidRiskScore
	}
	
	if c.MinConfidenceScore < 0 || c.MinConfidenceScore > 100 {
		return ErrInvalidConfidenceScore
	}
	
	// Validate minimum age
	if c.RequireAgeVerification && c.MinimumAge < 0 {
		return ErrInvalidMinimumAge
	}
	
	// Validate rate limits
	if c.RateLimitEnabled {
		if c.MaxVerificationsPerHour < 0 || c.MaxVerificationsPerDay < 0 {
			return ErrInvalidRateLimit
		}
	}
	
	// Validate verification attempts
	if c.MaxVerificationAttempts < 1 {
		return ErrInvalidMaxAttempts
	}
	
	return nil
}

