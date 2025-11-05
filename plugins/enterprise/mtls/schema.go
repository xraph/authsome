package mtls

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Certificate represents a client certificate in the system
type Certificate struct {
	bun.BaseModel `bun:"table:mtls_certificates,alias:cert"`

	ID             string `bun:"id,pk,type:varchar(36)" json:"id"`
	OrganizationID string `bun:"organization_id,notnull,type:varchar(36)" json:"organizationId"`
	UserID         string `bun:"user_id,nullzero,type:varchar(36)" json:"userId,omitempty"` // null for device/machine certs
	DeviceID       string `bun:"device_id,nullzero,type:varchar(36)" json:"deviceId,omitempty"`

	// Certificate Info
	Subject         string `bun:"subject,notnull" json:"subject"`
	Issuer          string `bun:"issuer,notnull" json:"issuer"`
	SerialNumber    string `bun:"serial_number,notnull,unique" json:"serialNumber"`
	Fingerprint     string `bun:"fingerprint,notnull,unique" json:"fingerprint"` // SHA-256
	FingerprintSHA1 string `bun:"fingerprint_sha1,notnull" json:"fingerprintSha1"`

	// Certificate Data
	CertificatePEM string `bun:"certificate_pem,notnull,type:text" json:"-"` // Don't expose in JSON
	PublicKeyPEM   string `bun:"public_key_pem,notnull,type:text" json:"-"`

	// Validity
	NotBefore time.Time `bun:"not_before,notnull" json:"notBefore"`
	NotAfter  time.Time `bun:"not_after,notnull" json:"notAfter"`

	// Certificate Type
	CertificateType  string `bun:"certificate_type,notnull" json:"certificateType"`   // user, device, service
	CertificateClass string `bun:"certificate_class,notnull" json:"certificateClass"` // standard, piv, cac, smartcard

	// Status
	Status        string     `bun:"status,notnull,default:'active'" json:"status"` // active, revoked, expired, suspended
	RevokedAt     *time.Time `bun:"revoked_at,nullzero" json:"revokedAt,omitempty"`
	RevokedReason string     `bun:"revoked_reason,nullzero" json:"revokedReason,omitempty"`

	// PIV/CAC Specific
	PIVCardID string `bun:"piv_card_id,nullzero" json:"pivCardId,omitempty"`
	CACNumber string `bun:"cac_number,nullzero" json:"cacNumber,omitempty"`

	// HSM Integration
	HSMKeyID    string `bun:"hsm_key_id,nullzero" json:"hsmKeyId,omitempty"`
	HSMProvider string `bun:"hsm_provider,nullzero" json:"hsmProvider,omitempty"`

	// Pinning
	IsPinned     bool       `bun:"is_pinned,notnull,default:false" json:"isPinned"`
	PinExpiresAt *time.Time `bun:"pin_expires_at,nullzero" json:"pinExpiresAt,omitempty"`

	// Extensions
	KeyUsage         []string    `bun:"key_usage,array,type:text[]" json:"keyUsage"`
	ExtendedKeyUsage []string    `bun:"extended_key_usage,array,type:text[]" json:"extendedKeyUsage"`
	SubjectAltNames  StringArray `bun:"subject_alt_names,type:jsonb" json:"subjectAltNames,omitempty"`

	// Metadata
	Metadata map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	// Audit
	CreatedAt  time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt  time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	LastUsedAt *time.Time `bun:"last_used_at,nullzero" json:"lastUsedAt,omitempty"`
	UseCount   int        `bun:"use_count,notnull,default:0" json:"useCount"`
}

// TrustAnchor represents a trusted CA certificate
type TrustAnchor struct {
	bun.BaseModel `bun:"table:mtls_trust_anchors,alias:ta"`

	ID             string `bun:"id,pk,type:varchar(36)" json:"id"`
	OrganizationID string `bun:"organization_id,notnull,type:varchar(36)" json:"organizationId"`

	// CA Info
	Name         string `bun:"name,notnull" json:"name"`
	Subject      string `bun:"subject,notnull" json:"subject"`
	Issuer       string `bun:"issuer,notnull" json:"issuer"`
	SerialNumber string `bun:"serial_number,notnull" json:"serialNumber"`
	Fingerprint  string `bun:"fingerprint,notnull,unique" json:"fingerprint"`

	// Certificate
	CertificatePEM string `bun:"certificate_pem,notnull,type:text" json:"-"`

	// Validity
	NotBefore time.Time `bun:"not_before,notnull" json:"notBefore"`
	NotAfter  time.Time `bun:"not_after,notnull" json:"notAfter"`

	// Trust Level
	TrustLevel string `bun:"trust_level,notnull" json:"trustLevel"` // root, intermediate, self_signed
	IsRootCA   bool   `bun:"is_root_ca,notnull,default:false" json:"isRootCA"`

	// Revocation Checking
	CRLEndpoints  StringArray `bun:"crl_endpoints,type:jsonb" json:"crlEndpoints,omitempty"`
	OCSPEndpoints StringArray `bun:"ocsp_endpoints,type:jsonb" json:"ocspEndpoints,omitempty"`

	// Status
	Status string `bun:"status,notnull,default:'active'" json:"status"` // active, revoked, expired

	// Metadata
	Metadata map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	// Audit
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// CertificateRevocationList stores CRL data
type CertificateRevocationList struct {
	bun.BaseModel `bun:"table:mtls_crls,alias:crl"`

	ID             string `bun:"id,pk,type:varchar(36)" json:"id"`
	OrganizationID string `bun:"organization_id,notnull,type:varchar(36)" json:"organizationId"`
	TrustAnchorID  string `bun:"trust_anchor_id,notnull,type:varchar(36)" json:"trustAnchorId"`

	// CRL Info
	Issuer     string    `bun:"issuer,notnull" json:"issuer"`
	ThisUpdate time.Time `bun:"this_update,notnull" json:"thisUpdate"`
	NextUpdate time.Time `bun:"next_update,notnull" json:"nextUpdate"`

	// CRL Data
	CRLPEM    string `bun:"crl_pem,notnull,type:text" json:"-"`
	CRLNumber string `bun:"crl_number,nullzero" json:"crlNumber,omitempty"`

	// Distribution
	DistributionPoint string `bun:"distribution_point,nullzero" json:"distributionPoint,omitempty"`

	// Status
	Status string `bun:"status,notnull,default:'valid'" json:"status"` // valid, expired, superseded

	// Stats
	RevokedCertCount int `bun:"revoked_cert_count,notnull,default:0" json:"revokedCertCount"`

	// Audit
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	LastFetchedAt time.Time `bun:"last_fetched_at,notnull,default:current_timestamp" json:"lastFetchedAt"`
}

// OCSPResponse stores OCSP response cache
type OCSPResponse struct {
	bun.BaseModel `bun:"table:mtls_ocsp_responses,alias:ocsp"`

	ID            string `bun:"id,pk,type:varchar(36)" json:"id"`
	CertificateID string `bun:"certificate_id,notnull,type:varchar(36)" json:"certificateId"`

	// OCSP Response
	Status     string     `bun:"status,notnull" json:"status"` // good, revoked, unknown
	ProducedAt time.Time  `bun:"produced_at,notnull" json:"producedAt"`
	ThisUpdate time.Time  `bun:"this_update,notnull" json:"thisUpdate"`
	NextUpdate *time.Time `bun:"next_update,nullzero" json:"nextUpdate,omitempty"`

	// Response Data
	ResponseData string `bun:"response_data,type:text" json:"-"`
	ResponderID  string `bun:"responder_id,nullzero" json:"responderId,omitempty"`

	// Revocation Info (if revoked)
	RevokedAt        *time.Time `bun:"revoked_at,nullzero" json:"revokedAt,omitempty"`
	RevocationReason string     `bun:"revocation_reason,nullzero" json:"revocationReason,omitempty"`

	// Cache
	ExpiresAt time.Time `bun:"expires_at,notnull" json:"expiresAt"`

	// Audit
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// CertificateAuthEvent tracks authentication events using certificates
type CertificateAuthEvent struct {
	bun.BaseModel `bun:"table:mtls_auth_events,alias:cae"`

	ID             string `bun:"id,pk,type:varchar(36)" json:"id"`
	CertificateID  string `bun:"certificate_id,notnull,type:varchar(36)" json:"certificateId"`
	OrganizationID string `bun:"organization_id,notnull,type:varchar(36)" json:"organizationId"`
	UserID         string `bun:"user_id,nullzero,type:varchar(36)" json:"userId,omitempty"`

	// Event Details
	EventType string `bun:"event_type,notnull" json:"eventType"` // auth_success, auth_failure, validation_error
	Status    string `bun:"status,notnull" json:"status"`        // success, failed, error

	// Validation Details
	ValidationSteps map[string]interface{} `bun:"validation_steps,type:jsonb" json:"validationSteps,omitempty"`
	FailureReason   string                 `bun:"failure_reason,nullzero" json:"failureReason,omitempty"`
	ErrorCode       string                 `bun:"error_code,nullzero" json:"errorCode,omitempty"`

	// Request Context
	IPAddress string `bun:"ip_address,nullzero" json:"ipAddress,omitempty"`
	UserAgent string `bun:"user_agent,nullzero" json:"userAgent,omitempty"`
	RequestID string `bun:"request_id,nullzero" json:"requestId,omitempty"`

	// Smart Card Info (if applicable)
	SmartCardID    string `bun:"smart_card_id,nullzero" json:"smartCardId,omitempty"`
	CardReaderName string `bun:"card_reader_name,nullzero" json:"cardReaderName,omitempty"`

	// Metadata
	Metadata map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	// Timestamp
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
}

// CertificatePolicy defines certificate validation policies
type CertificatePolicy struct {
	bun.BaseModel `bun:"table:mtls_policies,alias:cp"`

	ID             string `bun:"id,pk,type:varchar(36)" json:"id"`
	OrganizationID string `bun:"organization_id,notnull,type:varchar(36)" json:"organizationId"`

	// Policy Info
	Name        string `bun:"name,notnull" json:"name"`
	Description string `bun:"description,nullzero" json:"description,omitempty"`

	// Validation Rules
	RequirePinning   bool `bun:"require_pinning,notnull,default:false" json:"requirePinning"`
	AllowSelfSigned  bool `bun:"allow_self_signed,notnull,default:false" json:"allowSelfSigned"`
	RequireCRLCheck  bool `bun:"require_crl_check,notnull,default:true" json:"requireCrlCheck"`
	RequireOCSPCheck bool `bun:"require_ocsp_check,notnull,default:true" json:"requireOcspCheck"`
	OCSPStapling     bool `bun:"ocsp_stapling,notnull,default:false" json:"ocspStapling"`

	// Certificate Requirements
	MinKeySize           int         `bun:"min_key_size,notnull,default:2048" json:"minKeySize"`
	AllowedKeyAlgorithms StringArray `bun:"allowed_key_algorithms,type:jsonb" json:"allowedKeyAlgorithms"`
	AllowedSignatureAlgs StringArray `bun:"allowed_signature_algs,type:jsonb" json:"allowedSignatureAlgs"`
	RequiredKeyUsage     StringArray `bun:"required_key_usage,type:jsonb" json:"requiredKeyUsage,omitempty"`
	RequiredEKU          StringArray `bun:"required_eku,type:jsonb" json:"requiredEku,omitempty"`

	// Validity Requirements
	MaxCertificateAge    int `bun:"max_certificate_age,notnull,default:365" json:"maxCertificateAge"`      // days
	MinRemainingValidity int `bun:"min_remaining_validity,notnull,default:30" json:"minRemainingValidity"` // days

	// Trust Requirements
	AllowedCAs         StringArray `bun:"allowed_cas,type:jsonb" json:"allowedCas,omitempty"` // fingerprints
	RequiredTrustLevel string      `bun:"required_trust_level,notnull,default:'root'" json:"requiredTrustLevel"`

	// Smart Card/PIV/CAC
	RequirePIV      bool `bun:"require_piv,notnull,default:false" json:"requirePiv"`
	RequireCAC      bool `bun:"require_cac,notnull,default:false" json:"requireCac"`
	PIVAuthCertOnly bool `bun:"piv_auth_cert_only,notnull,default:true" json:"pivAuthCertOnly"`

	// HSM Requirements
	RequireHSM          bool        `bun:"require_hsm,notnull,default:false" json:"requireHsm"`
	AllowedHSMProviders StringArray `bun:"allowed_hsm_providers,type:jsonb" json:"allowedHsmProviders,omitempty"`

	// Status
	Status    string `bun:"status,notnull,default:'active'" json:"status"` // active, inactive
	IsDefault bool   `bun:"is_default,notnull,default:false" json:"isDefault"`

	// Metadata
	Metadata map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	// Audit
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// StringArray is a custom type for string arrays stored as JSONB
type StringArray []string

// Value implements the driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	return json.Unmarshal(b, s)
}

// BeforeInsert hook for Certificate
func (c *Certificate) BeforeInsert() error {
	if c.ID == "" {
		c.ID = generateID()
	}
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

// BeforeUpdate hook for Certificate
func (c *Certificate) BeforeUpdate() error {
	c.UpdatedAt = time.Now()
	return nil
}

// Helper function to generate IDs (would use uuid in real implementation)
func generateID() string {
	// Use actual UUID generation in production
	return fmt.Sprintf("cert_%d", time.Now().UnixNano())
}
