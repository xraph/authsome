package consent

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// ConsentRecord tracks user consent for various purposes.
type ConsentRecord struct {
	bun.BaseModel `bun:"table:consent_records,alias:cr"`

	ID             xid.ID     `bun:"id,pk,type:varchar(20)"                       json:"id"`
	UserID         string     `bun:"user_id,notnull,type:varchar(20)"             json:"userId"`
	OrganizationID string     `bun:"organization_id,notnull,type:varchar(20)"     json:"organizationId"`
	ConsentType    string     `bun:"consent_type,notnull"                         json:"consentType"` // cookies, marketing, analytics, terms, privacy, data_processing
	Purpose        string     `bun:"purpose,notnull"                              json:"purpose"`     // specific purpose description
	Granted        bool       `bun:"granted,notnull"                              json:"granted"`
	Version        string     `bun:"version,notnull"                              json:"version"` // version of policy/terms
	IPAddress      string     `bun:"ip_address"                                   json:"ipAddress"`
	UserAgent      string     `bun:"user_agent"                                   json:"userAgent"`
	Metadata       JSONBMap   `bun:"metadata,type:jsonb"                          json:"metadata"`
	ExpiresAt      *time.Time `bun:"expires_at"                                   json:"expiresAt,omitempty"` // consent expiry
	GrantedAt      time.Time  `bun:"granted_at,notnull"                           json:"grantedAt"`
	RevokedAt      *time.Time `bun:"revoked_at"                                   json:"revokedAt,omitempty"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// ConsentPolicy defines consent policies per organization.
type ConsentPolicy struct {
	bun.BaseModel `bun:"table:consent_policies,alias:cp"`

	ID             xid.ID     `bun:"id,pk,type:varchar(20)"                       json:"id"`
	OrganizationID string     `bun:"organization_id,notnull,type:varchar(20)"     json:"organizationId"`
	ConsentType    string     `bun:"consent_type,notnull"                         json:"consentType"`
	Name           string     `bun:"name,notnull"                                 json:"name"`
	Description    string     `bun:"description"                                  json:"description"`
	Version        string     `bun:"version,notnull"                              json:"version"`
	Content        string     `bun:"content,type:text"                            json:"content"`                  // Full policy text
	Required       bool       `bun:"required"                                     json:"required"`                 // Block access if not granted
	Renewable      bool       `bun:"renewable"                                    json:"renewable"`                // Allow re-consent
	ValidityPeriod *int       `bun:"validity_period"                              json:"validityPeriod,omitempty"` // Days until re-consent required
	Active         bool       `bun:"active,notnull,default:true"                  json:"active"`
	PublishedAt    *time.Time `bun:"published_at"                                 json:"publishedAt,omitempty"`
	Metadata       JSONBMap   `bun:"metadata,type:jsonb"                          json:"metadata"`
	CreatedBy      string     `bun:"created_by,type:varchar(20)"                  json:"createdBy"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// DataProcessingAgreement tracks DPA acceptance.
type DataProcessingAgreement struct {
	bun.BaseModel `bun:"table:data_processing_agreements,alias:dpa"`

	ID               xid.ID     `bun:"id,pk,type:varchar(20)"                       json:"id"`
	OrganizationID   string     `bun:"organization_id,notnull,type:varchar(20)"     json:"organizationId"`
	AgreementType    string     `bun:"agreement_type,notnull"                       json:"agreementType"` // dpa, baa, ccpa, gdpr
	Version          string     `bun:"version,notnull"                              json:"version"`
	Content          string     `bun:"content,type:text"                            json:"content"`
	SignedBy         string     `bun:"signed_by,type:varchar(20)"                   json:"signedBy"` // User ID who signed
	SignedByName     string     `bun:"signed_by_name"                               json:"signedByName"`
	SignedByTitle    string     `bun:"signed_by_title"                              json:"signedByTitle"`
	SignedByEmail    string     `bun:"signed_by_email"                              json:"signedByEmail"`
	IPAddress        string     `bun:"ip_address"                                   json:"ipAddress"`
	DigitalSignature string     `bun:"digital_signature,type:text"                  json:"digitalSignature"` // Cryptographic signature
	EffectiveDate    time.Time  `bun:"effective_date,notnull"                       json:"effectiveDate"`
	ExpiryDate       *time.Time `bun:"expiry_date"                                  json:"expiryDate,omitempty"`
	Status           string     `bun:"status,notnull"                               json:"status"` // active, expired, revoked
	Metadata         JSONBMap   `bun:"metadata,type:jsonb"                          json:"metadata"`
	CreatedAt        time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// ConsentAuditLog provides immutable audit trail for consent changes.
type ConsentAuditLog struct {
	bun.BaseModel `bun:"table:consent_audit_logs,alias:cal"`

	ID             xid.ID    `bun:"id,pk,type:varchar(20)"                       json:"id"`
	UserID         string    `bun:"user_id,notnull,type:varchar(20)"             json:"userId"`
	OrganizationID string    `bun:"organization_id,notnull,type:varchar(20)"     json:"organizationId"`
	ConsentID      string    `bun:"consent_id,type:varchar(20)"                  json:"consentId"` // Reference to consent record
	Action         string    `bun:"action,notnull"                               json:"action"`    // granted, revoked, updated, expired
	ConsentType    string    `bun:"consent_type,notnull"                         json:"consentType"`
	Purpose        string    `bun:"purpose"                                      json:"purpose"`
	PreviousValue  JSONBMap  `bun:"previous_value,type:jsonb"                    json:"previousValue"`
	NewValue       JSONBMap  `bun:"new_value,type:jsonb"                         json:"newValue"`
	IPAddress      string    `bun:"ip_address"                                   json:"ipAddress"`
	UserAgent      string    `bun:"user_agent"                                   json:"userAgent"`
	Reason         string    `bun:"reason"                                       json:"reason"` // Reason for change
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
}

// CookieConsent tracks cookie consent preferences.
type CookieConsent struct {
	bun.BaseModel `bun:"table:cookie_consents,alias:cc"`

	ID                   xid.ID    `bun:"id,pk,type:varchar(20)"                       json:"id"`
	UserID               string    `bun:"user_id,type:varchar(20)"                     json:"userId"` // Nullable for anonymous users
	OrganizationID       string    `bun:"organization_id,notnull,type:varchar(20)"     json:"organizationId"`
	SessionID            string    `bun:"session_id"                                   json:"sessionId"` // Track anonymous sessions
	Essential            bool      `bun:"essential,notnull,default:true"               json:"essential"` // Always true
	Functional           bool      `bun:"functional"                                   json:"functional"`
	Analytics            bool      `bun:"analytics"                                    json:"analytics"`
	Marketing            bool      `bun:"marketing"                                    json:"marketing"`
	Personalization      bool      `bun:"personalization"                              json:"personalization"`
	ThirdParty           bool      `bun:"third_party"                                  json:"thirdParty"`
	IPAddress            string    `bun:"ip_address"                                   json:"ipAddress"`
	UserAgent            string    `bun:"user_agent"                                   json:"userAgent"`
	ConsentBannerVersion string    `bun:"consent_banner_version"                       json:"consentBannerVersion"`
	ExpiresAt            time.Time `bun:"expires_at,notnull"                           json:"expiresAt"`
	CreatedAt            time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt            time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// DataExportRequest tracks GDPR data export requests.
type DataExportRequest struct {
	bun.BaseModel `bun:"table:data_export_requests,alias:der"`

	ID              xid.ID     `bun:"id,pk,type:varchar(20)"                       json:"id"`
	UserID          string     `bun:"user_id,notnull,type:varchar(20)"             json:"userId"`
	OrganizationID  string     `bun:"organization_id,notnull,type:varchar(20)"     json:"organizationId"`
	Status          string     `bun:"status,notnull"                               json:"status"`          // pending, processing, completed, failed
	Format          string     `bun:"format,notnull"                               json:"format"`          // json, csv, xml
	IncludeSections []string   `bun:"include_sections,type:text[]"                 json:"includeSections"` // profile, sessions, consents, audit
	ExportURL       string     `bun:"export_url"                                   json:"exportUrl"`
	ExportPath      string     `bun:"export_path"                                  json:"exportPath"`
	ExportSize      int64      `bun:"export_size"                                  json:"exportSize"`          // bytes
	ExpiresAt       *time.Time `bun:"expires_at"                                   json:"expiresAt,omitempty"` // URL expiry
	IPAddress       string     `bun:"ip_address"                                   json:"ipAddress"`
	CompletedAt     *time.Time `bun:"completed_at"                                 json:"completedAt,omitempty"`
	ErrorMessage    string     `bun:"error_message"                                json:"errorMessage,omitempty"`
	CreatedAt       time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt       time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// DataDeletionRequest tracks GDPR right to be forgotten requests.
type DataDeletionRequest struct {
	bun.BaseModel `bun:"table:data_deletion_requests,alias:ddr"`

	ID              xid.ID     `bun:"id,pk,type:varchar(20)"                       json:"id"`
	UserID          string     `bun:"user_id,notnull,type:varchar(20)"             json:"userId"`
	OrganizationID  string     `bun:"organization_id,notnull,type:varchar(20)"     json:"organizationId"`
	Status          string     `bun:"status,notnull"                               json:"status"` // pending, approved, processing, completed, rejected
	RequestReason   string     `bun:"request_reason,type:text"                     json:"requestReason"`
	RetentionExempt bool       `bun:"retention_exempt"                             json:"retentionExempt"` // Legal hold or other exemption
	ExemptionReason string     `bun:"exemption_reason"                             json:"exemptionReason"`
	DeleteSections  []string   `bun:"delete_sections,type:text[]"                  json:"deleteSections"` // all, profile, sessions, consents
	IPAddress       string     `bun:"ip_address"                                   json:"ipAddress"`
	ApprovedBy      string     `bun:"approved_by,type:varchar(20)"                 json:"approvedBy"` // Admin who approved
	ApprovedAt      *time.Time `bun:"approved_at"                                  json:"approvedAt,omitempty"`
	CompletedAt     *time.Time `bun:"completed_at"                                 json:"completedAt,omitempty"`
	RejectedAt      *time.Time `bun:"rejected_at"                                  json:"rejectedAt,omitempty"`
	ErrorMessage    string     `bun:"error_message"                                json:"errorMessage,omitempty"`
	ArchivePath     string     `bun:"archive_path"                                 json:"archivePath"` // Backup before deletion
	CreatedAt       time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt       time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// PrivacySettings stores per-organization privacy configurations.
type PrivacySettings struct {
	bun.BaseModel `bun:"table:privacy_settings,alias:ps"`

	ID                              xid.ID    `bun:"id,pk,type:varchar(20)"                          json:"id"`
	OrganizationID                  string    `bun:"organization_id,notnull,unique,type:varchar(20)" json:"organizationId"`
	ConsentRequired                 bool      `bun:"consent_required,notnull,default:true"           json:"consentRequired"`
	CookieConsentEnabled            bool      `bun:"cookie_consent_enabled,notnull,default:true"     json:"cookieConsentEnabled"`
	CookieConsentStyle              string    `bun:"cookie_consent_style"                            json:"cookieConsentStyle"` // banner, modal, popup
	DataRetentionDays               int       `bun:"data_retention_days"                             json:"dataRetentionDays"`
	AnonymousConsentEnabled         bool      `bun:"anonymous_consent_enabled"                       json:"anonymousConsentEnabled"`
	GDPRMode                        bool      `bun:"gdpr_mode,notnull,default:false"                 json:"gdprMode"`
	CCPAMode                        bool      `bun:"ccpa_mode,notnull,default:false"                 json:"ccpaMode"`
	AutoDeleteAfterDays             int       `bun:"auto_delete_after_days"                          json:"autoDeleteAfterDays"`
	RequireExplicitConsent          bool      `bun:"require_explicit_consent"                        json:"requireExplicitConsent"` // No implied consent
	AllowDataPortability            bool      `bun:"allow_data_portability,notnull,default:true"     json:"allowDataPortability"`
	ExportFormat                    []string  `bun:"export_format,type:text[]"                       json:"exportFormat"` // json, csv, xml
	DataExportExpiryHours           int       `bun:"data_export_expiry_hours"                        json:"dataExportExpiryHours"`
	RequireAdminApprovalForDeletion bool      `bun:"require_admin_approval_for_deletion"             json:"requireAdminApprovalForDeletion"`
	DeletionGracePeriodDays         int       `bun:"deletion_grace_period_days"                      json:"deletionGracePeriodDays"`
	ContactEmail                    string    `bun:"contact_email"                                   json:"contactEmail"`
	ContactPhone                    string    `bun:"contact_phone"                                   json:"contactPhone"`
	DPOEmail                        string    `bun:"dpo_email"                                       json:"dpoEmail"` // Data Protection Officer
	Metadata                        JSONBMap  `bun:"metadata,type:jsonb"                             json:"metadata"`
	CreatedAt                       time.Time `bun:"created_at,notnull,default:current_timestamp"    json:"createdAt"`
	UpdatedAt                       time.Time `bun:"updated_at,notnull,default:current_timestamp"    json:"updatedAt"`
}

// JSONBMap is a helper type for JSONB fields.
type JSONBMap map[string]any
