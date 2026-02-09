package mtls

import (
	"errors"
	"time"
)

// Config holds the mTLS plugin configuration.
type Config struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Certificate Validation
	Validation ValidationConfig `json:"validation" yaml:"validation"`

	// Revocation Checking
	Revocation RevocationConfig `json:"revocation" yaml:"revocation"`

	// PIV/CAC Smart Card Support
	SmartCard SmartCardConfig `json:"smartCard" yaml:"smartCard"`

	// HSM Integration
	HSM HSMConfig `json:"hsm" yaml:"hsm"`

	// Certificate Pinning
	Pinning PinningConfig `json:"pinning" yaml:"pinning"`

	// Trust Anchors
	TrustAnchors TrustAnchorsConfig `json:"trustAnchors" yaml:"trustAnchors"`

	// Session Management
	Session SessionConfig `json:"session" yaml:"session"`

	// API Endpoints
	API APIConfig `json:"api" yaml:"api"`

	// Security
	Security SecurityConfig `json:"security" yaml:"security"`
}

// ValidationConfig configures certificate validation.
type ValidationConfig struct {
	// Basic Validation
	CheckExpiration       bool `json:"checkExpiration"       yaml:"checkExpiration"`
	CheckNotBefore        bool `json:"checkNotBefore"        yaml:"checkNotBefore"`
	CheckSignature        bool `json:"checkSignature"        yaml:"checkSignature"`
	CheckKeyUsage         bool `json:"checkKeyUsage"         yaml:"checkKeyUsage"`
	CheckExtendedKeyUsage bool `json:"checkExtendedKeyUsage" yaml:"checkExtendedKeyUsage"`

	// Chain Validation
	ValidateChain   bool `json:"validateChain"   yaml:"validateChain"`
	MaxChainLength  int  `json:"maxChainLength"  yaml:"maxChainLength"`
	AllowSelfSigned bool `json:"allowSelfSigned" yaml:"allowSelfSigned"`

	// Key Requirements
	MinKeySize           int      `json:"minKeySize"           yaml:"minKeySize"` // bits
	AllowedKeyAlgorithms []string `json:"allowedKeyAlgorithms" yaml:"allowedKeyAlgorithms"`
	AllowedSignatureAlgs []string `json:"allowedSignatureAlgs" yaml:"allowedSignatureAlgs"`

	// Validity Requirements
	MaxCertificateAge    int `json:"maxCertificateAge"    yaml:"maxCertificateAge"`    // days
	MinRemainingValidity int `json:"minRemainingValidity" yaml:"minRemainingValidity"` // days

	// Required Extensions
	RequiredKeyUsage []string `json:"requiredKeyUsage" yaml:"requiredKeyUsage"`
	RequiredEKU      []string `json:"requiredEku"      yaml:"requiredEku"`
}

// RevocationConfig configures certificate revocation checking.
type RevocationConfig struct {
	// CRL Configuration
	EnableCRL        bool          `json:"enableCrl"        yaml:"enableCrl"`
	CRLCacheDuration time.Duration `json:"crlCacheDuration" yaml:"crlCacheDuration"`
	CRLFetchTimeout  time.Duration `json:"crlFetchTimeout"  yaml:"crlFetchTimeout"`
	CRLMaxSize       int64         `json:"crlMaxSize"       yaml:"crlMaxSize"` // bytes
	AutoFetchCRL     bool          `json:"autoFetchCrl"     yaml:"autoFetchCrl"`

	// OCSP Configuration
	EnableOCSP        bool          `json:"enableOcsp"        yaml:"enableOcsp"`
	OCSPCacheDuration time.Duration `json:"ocspCacheDuration" yaml:"ocspCacheDuration"`
	OCSPTimeout       time.Duration `json:"ocspTimeout"       yaml:"ocspTimeout"`
	OCSPStapling      bool          `json:"ocspStapling"      yaml:"ocspStapling"`

	// Fallback Behavior
	FailOpen   bool `json:"failOpen"   yaml:"failOpen"` // Allow auth if revocation unavailable
	PreferOCSP bool `json:"preferOcsp" yaml:"preferOcsp"`
}

// SmartCardConfig configures PIV/CAC smart card support.
type SmartCardConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// PIV (Personal Identity Verification)
	EnablePIV       bool     `json:"enablePiv"       yaml:"enablePiv"`
	PIVAuthCertOnly bool     `json:"pivAuthCertOnly" yaml:"pivAuthCertOnly"` // Only accept PIV auth certificate
	PIVRequiredOIDs []string `json:"pivRequiredOids" yaml:"pivRequiredOids"`

	// CAC (Common Access Card)
	EnableCAC       bool     `json:"enableCac"       yaml:"enableCac"`
	CACRequiredOIDs []string `json:"cacRequiredOids" yaml:"cacRequiredOids"`

	// Card Reader Configuration
	Readers       []string      `json:"readers"       yaml:"readers"` // Specific readers to use (empty = all)
	ReaderTimeout time.Duration `json:"readerTimeout" yaml:"readerTimeout"`

	// PIN Configuration
	RequirePIN     bool `json:"requirePin"     yaml:"requirePin"`
	MaxPINAttempts int  `json:"maxPinAttempts" yaml:"maxPinAttempts"`
	PINMinLength   int  `json:"pinMinLength"   yaml:"pinMinLength"`
	PINMaxLength   int  `json:"pinMaxLength"   yaml:"pinMaxLength"`

	// Security
	LockCardOnFailure bool          `json:"lockCardOnFailure" yaml:"lockCardOnFailure"`
	CardTimeout       time.Duration `json:"cardTimeout"       yaml:"cardTimeout"`
}

// HSMConfig configures Hardware Security Module integration.
type HSMConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Provider Configuration
	Provider       string            `json:"provider"       yaml:"provider"` // pkcs11, cloudhsm, yubihsm, etc.
	ProviderConfig map[string]string `json:"providerConfig" yaml:"providerConfig"`

	// PKCS#11 Configuration
	PKCS11Library string `json:"pkcs11Library" yaml:"pkcs11Library"`
	PKCS11SlotID  int    `json:"pkcs11SlotId"  yaml:"pkcs11SlotId"`
	PKCS11PIN     string `json:"pkcs11Pin"     yaml:"pkcs11Pin"`

	// AWS CloudHSM
	CloudHSMClusterID string `json:"cloudHsmClusterId" yaml:"cloudHsmClusterId"`
	CloudHSMRegion    string `json:"cloudHsmRegion"    yaml:"cloudHsmRegion"`

	// Azure Key Vault
	AzureVaultURL string `json:"azureVaultUrl" yaml:"azureVaultUrl"`
	AzureTenantID string `json:"azureTenantId" yaml:"azureTenantId"`

	// GCP Cloud HSM
	GCPProjectID string `json:"gcpProjectId" yaml:"gcpProjectId"`
	GCPLocation  string `json:"gcpLocation"  yaml:"gcpLocation"`
	GCPKeyRing   string `json:"gcpKeyRing"   yaml:"gcpKeyRing"`

	// Connection
	ConnectionTimeout time.Duration `json:"connectionTimeout" yaml:"connectionTimeout"`
	MaxConnections    int           `json:"maxConnections"    yaml:"maxConnections"`

	// Security
	RequireHSM       bool     `json:"requireHsm"       yaml:"requireHsm"` // Reject certs not backed by HSM
	AllowedProviders []string `json:"allowedProviders" yaml:"allowedProviders"`
}

// PinningConfig configures certificate pinning.
type PinningConfig struct {
	Enabled            bool          `json:"enabled"            yaml:"enabled"`
	Required           bool          `json:"required"           yaml:"required"` // Reject unpinned certs
	PinExpiration      time.Duration `json:"pinExpiration"      yaml:"pinExpiration"`
	AutoPin            bool          `json:"autoPin"            yaml:"autoPin"` // Auto-pin on first use
	PinRotationWarning time.Duration `json:"pinRotationWarning" yaml:"pinRotationWarning"`
}

// TrustAnchorsConfig configures trust anchor management.
type TrustAnchorsConfig struct {
	// System Trust Store
	UseSystemStore  bool   `json:"useSystemStore"  yaml:"useSystemStore"`
	SystemStorePath string `json:"systemStorePath" yaml:"systemStorePath"`

	// Custom Trust Anchors
	CustomAnchors []string `json:"customAnchors" yaml:"customAnchors"` // Paths to CA certs

	// Auto-Update
	AutoUpdate     bool          `json:"autoUpdate"     yaml:"autoUpdate"`
	UpdateInterval time.Duration `json:"updateInterval" yaml:"updateInterval"`

	// Validation
	ValidateAnchors bool `json:"validateAnchors" yaml:"validateAnchors"`
	RejectExpired   bool `json:"rejectExpired"   yaml:"rejectExpired"`
}

// SessionConfig configures mTLS session management.
type SessionConfig struct {
	// Session Creation
	CreateSession   bool          `json:"createSession"   yaml:"createSession"`
	SessionDuration time.Duration `json:"sessionDuration" yaml:"sessionDuration"`

	// Certificate Binding
	BindToFingerprint bool `json:"bindToFingerprint" yaml:"bindToFingerprint"` // Bind session to cert
	RequireSameCert   bool `json:"requireSameCert"   yaml:"requireSameCert"`   // Require same cert for session

	// Re-validation
	RevalidateOnUse    bool          `json:"revalidateOnUse"    yaml:"revalidateOnUse"`
	RevalidateInterval time.Duration `json:"revalidateInterval" yaml:"revalidateInterval"`
}

// APIConfig configures mTLS API endpoints.
type APIConfig struct {
	BasePath         string `json:"basePath"         yaml:"basePath"`
	EnableManagement bool   `json:"enableManagement" yaml:"enableManagement"` // Certificate management APIs
	EnableValidation bool   `json:"enableValidation" yaml:"enableValidation"` // Validation endpoint
	EnableMetrics    bool   `json:"enableMetrics"    yaml:"enableMetrics"`
}

// SecurityConfig configures security settings.
type SecurityConfig struct {
	// Rate Limiting
	RateLimitEnabled     bool `json:"rateLimitEnabled"     yaml:"rateLimitEnabled"`
	MaxAttemptsPerMinute int  `json:"maxAttemptsPerMinute" yaml:"maxAttemptsPerMinute"`
	MaxAttemptsPerHour   int  `json:"maxAttemptsPerHour"   yaml:"maxAttemptsPerHour"`

	// Audit Logging
	AuditAllAttempts bool `json:"auditAllAttempts" yaml:"auditAllAttempts"`
	AuditFailures    bool `json:"auditFailures"    yaml:"auditFailures"`
	AuditValidation  bool `json:"auditValidation"  yaml:"auditValidation"`

	// Certificate Storage
	StoreCertificates bool `json:"storeCertificates" yaml:"storeCertificates"`
	StorePrivateKeys  bool `json:"storePrivateKeys"  yaml:"storePrivateKeys"` // Usually false
	EncryptStorage    bool `json:"encryptStorage"    yaml:"encryptStorage"`

	// Notifications
	NotifyOnRevocation bool `json:"notifyOnRevocation" yaml:"notifyOnRevocation"`
	NotifyOnExpiration bool `json:"notifyOnExpiration" yaml:"notifyOnExpiration"`
	ExpirationWarning  int  `json:"expirationWarning"  yaml:"expirationWarning"` // days
}

// DefaultConfig returns the default mTLS configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		Validation: ValidationConfig{
			CheckExpiration:       true,
			CheckNotBefore:        true,
			CheckSignature:        true,
			CheckKeyUsage:         true,
			CheckExtendedKeyUsage: true,
			ValidateChain:         true,
			MaxChainLength:        5,
			AllowSelfSigned:       false,
			MinKeySize:            2048,
			AllowedKeyAlgorithms:  []string{"RSA", "ECDSA", "Ed25519"},
			AllowedSignatureAlgs:  []string{"SHA256-RSA", "SHA384-RSA", "SHA512-RSA", "ECDSA-SHA256", "ECDSA-SHA384", "ECDSA-SHA512"},
			MaxCertificateAge:     365,
			MinRemainingValidity:  30,
			RequiredKeyUsage:      []string{"digitalSignature", "keyEncipherment"},
			RequiredEKU:           []string{"clientAuth"},
		},
		Revocation: RevocationConfig{
			EnableCRL:         true,
			CRLCacheDuration:  24 * time.Hour,
			CRLFetchTimeout:   10 * time.Second,
			CRLMaxSize:        10 * 1024 * 1024, // 10MB
			AutoFetchCRL:      true,
			EnableOCSP:        true,
			OCSPCacheDuration: 1 * time.Hour,
			OCSPTimeout:       5 * time.Second,
			OCSPStapling:      true,
			FailOpen:          false, // Fail closed by default for security
			PreferOCSP:        true,
		},
		SmartCard: SmartCardConfig{
			Enabled:           true,
			EnablePIV:         true,
			PIVAuthCertOnly:   true,
			EnableCAC:         true,
			ReaderTimeout:     30 * time.Second,
			RequirePIN:        false, // PIN handled by OS/reader
			MaxPINAttempts:    3,
			PINMinLength:      4,
			PINMaxLength:      8,
			LockCardOnFailure: true,
			CardTimeout:       5 * time.Minute,
		},
		HSM: HSMConfig{
			Enabled:           false,
			ConnectionTimeout: 10 * time.Second,
			MaxConnections:    10,
			RequireHSM:        false,
		},
		Pinning: PinningConfig{
			Enabled:            true,
			Required:           false,
			PinExpiration:      365 * 24 * time.Hour,
			AutoPin:            false,
			PinRotationWarning: 30 * 24 * time.Hour,
		},
		TrustAnchors: TrustAnchorsConfig{
			UseSystemStore:  true,
			AutoUpdate:      true,
			UpdateInterval:  24 * time.Hour,
			ValidateAnchors: true,
			RejectExpired:   true,
		},
		Session: SessionConfig{
			CreateSession:      true,
			SessionDuration:    24 * time.Hour,
			BindToFingerprint:  true,
			RequireSameCert:    false,
			RevalidateOnUse:    true,
			RevalidateInterval: 1 * time.Hour,
		},
		API: APIConfig{
			BasePath:         "/auth/mtls",
			EnableManagement: true,
			EnableValidation: true,
			EnableMetrics:    true,
		},
		Security: SecurityConfig{
			RateLimitEnabled:     true,
			MaxAttemptsPerMinute: 10,
			MaxAttemptsPerHour:   100,
			AuditAllAttempts:     true,
			AuditFailures:        true,
			AuditValidation:      false,
			StoreCertificates:    true,
			StorePrivateKeys:     false,
			EncryptStorage:       true,
			NotifyOnRevocation:   true,
			NotifyOnExpiration:   true,
			ExpirationWarning:    30,
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Validation.MinKeySize < 1024 {
		return errors.New("minimum key size must be at least 1024 bits")
	}

	if c.Validation.MaxChainLength < 1 || c.Validation.MaxChainLength > 10 {
		return errors.New("max chain length must be between 1 and 10")
	}

	if c.Revocation.CRLMaxSize < 1024 {
		return errors.New("CRL max size must be at least 1024 bytes")
	}

	if c.Session.SessionDuration < 1*time.Minute {
		return errors.New("session duration must be at least 1 minute")
	}

	return nil
}
