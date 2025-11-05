package consent

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// ConsentRecord tracks user consent for various purposes
type ConsentRecord struct {
	bun.BaseModel `bun:"table:consent_records,alias:cr"`

	ID             xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID         string     `json:"userId" bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID string     `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	ConsentType    string     `json:"consentType" bun:"consent_type,notnull"` // cookies, marketing, analytics, terms, privacy, data_processing
	Purpose        string     `json:"purpose" bun:"purpose,notnull"`          // specific purpose description
	Granted        bool       `json:"granted" bun:"granted,notnull"`
	Version        string     `json:"version" bun:"version,notnull"` // version of policy/terms
	IPAddress      string     `json:"ipAddress" bun:"ip_address"`
	UserAgent      string     `json:"userAgent" bun:"user_agent"`
	Metadata       JSONBMap   `json:"metadata" bun:"metadata,type:jsonb"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty" bun:"expires_at"` // consent expiry
	GrantedAt      time.Time  `json:"grantedAt" bun:"granted_at,notnull"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty" bun:"revoked_at"`
	CreatedAt      time.Time  `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time  `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

// ConsentPolicy defines consent policies per organization
type ConsentPolicy struct {
	bun.BaseModel `bun:"table:consent_policies,alias:cp"`

	ID             xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID string     `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	ConsentType    string     `json:"consentType" bun:"consent_type,notnull"`
	Name           string     `json:"name" bun:"name,notnull"`
	Description    string     `json:"description" bun:"description"`
	Version        string     `json:"version" bun:"version,notnull"`
	Content        string     `json:"content" bun:"content,type:text"`                // Full policy text
	Required       bool       `json:"required" bun:"required"`                        // Block access if not granted
	Renewable      bool       `json:"renewable" bun:"renewable"`                      // Allow re-consent
	ValidityPeriod *int       `json:"validityPeriod,omitempty" bun:"validity_period"` // Days until re-consent required
	Active         bool       `json:"active" bun:"active,notnull,default:true"`
	PublishedAt    *time.Time `json:"publishedAt,omitempty" bun:"published_at"`
	Metadata       JSONBMap   `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedBy      string     `json:"createdBy" bun:"created_by,type:varchar(20)"`
	CreatedAt      time.Time  `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time  `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

// DataProcessingAgreement tracks DPA acceptance
type DataProcessingAgreement struct {
	bun.BaseModel `bun:"table:data_processing_agreements,alias:dpa"`

	ID               xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID   string     `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	AgreementType    string     `json:"agreementType" bun:"agreement_type,notnull"` // dpa, baa, ccpa, gdpr
	Version          string     `json:"version" bun:"version,notnull"`
	Content          string     `json:"content" bun:"content,type:text"`
	SignedBy         string     `json:"signedBy" bun:"signed_by,type:varchar(20)"` // User ID who signed
	SignedByName     string     `json:"signedByName" bun:"signed_by_name"`
	SignedByTitle    string     `json:"signedByTitle" bun:"signed_by_title"`
	SignedByEmail    string     `json:"signedByEmail" bun:"signed_by_email"`
	IPAddress        string     `json:"ipAddress" bun:"ip_address"`
	DigitalSignature string     `json:"digitalSignature" bun:"digital_signature,type:text"` // Cryptographic signature
	EffectiveDate    time.Time  `json:"effectiveDate" bun:"effective_date,notnull"`
	ExpiryDate       *time.Time `json:"expiryDate,omitempty" bun:"expiry_date"`
	Status           string     `json:"status" bun:"status,notnull"` // active, expired, revoked
	Metadata         JSONBMap   `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt        time.Time  `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt        time.Time  `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

// ConsentAuditLog provides immutable audit trail for consent changes
type ConsentAuditLog struct {
	bun.BaseModel `bun:"table:consent_audit_logs,alias:cal"`

	ID             xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID         string    `json:"userId" bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID string    `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	ConsentID      string    `json:"consentId" bun:"consent_id,type:varchar(20)"` // Reference to consent record
	Action         string    `json:"action" bun:"action,notnull"`                 // granted, revoked, updated, expired
	ConsentType    string    `json:"consentType" bun:"consent_type,notnull"`
	Purpose        string    `json:"purpose" bun:"purpose"`
	PreviousValue  JSONBMap  `json:"previousValue" bun:"previous_value,type:jsonb"`
	NewValue       JSONBMap  `json:"newValue" bun:"new_value,type:jsonb"`
	IPAddress      string    `json:"ipAddress" bun:"ip_address"`
	UserAgent      string    `json:"userAgent" bun:"user_agent"`
	Reason         string    `json:"reason" bun:"reason"` // Reason for change
	CreatedAt      time.Time `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
}

// CookieConsent tracks cookie consent preferences
type CookieConsent struct {
	bun.BaseModel `bun:"table:cookie_consents,alias:cc"`

	ID                   xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID               string    `json:"userId" bun:"user_id,type:varchar(20)"` // Nullable for anonymous users
	OrganizationID       string    `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	SessionID            string    `json:"sessionId" bun:"session_id"`                     // Track anonymous sessions
	Essential            bool      `json:"essential" bun:"essential,notnull,default:true"` // Always true
	Functional           bool      `json:"functional" bun:"functional"`
	Analytics            bool      `json:"analytics" bun:"analytics"`
	Marketing            bool      `json:"marketing" bun:"marketing"`
	Personalization      bool      `json:"personalization" bun:"personalization"`
	ThirdParty           bool      `json:"thirdParty" bun:"third_party"`
	IPAddress            string    `json:"ipAddress" bun:"ip_address"`
	UserAgent            string    `json:"userAgent" bun:"user_agent"`
	ConsentBannerVersion string    `json:"consentBannerVersion" bun:"consent_banner_version"`
	ExpiresAt            time.Time `json:"expiresAt" bun:"expires_at,notnull"`
	CreatedAt            time.Time `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt            time.Time `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

// DataExportRequest tracks GDPR data export requests
type DataExportRequest struct {
	bun.BaseModel `bun:"table:data_export_requests,alias:der"`

	ID              xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID          string     `json:"userId" bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID  string     `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	Status          string     `json:"status" bun:"status,notnull"`                        // pending, processing, completed, failed
	Format          string     `json:"format" bun:"format,notnull"`                        // json, csv, xml
	IncludeSections []string   `json:"includeSections" bun:"include_sections,type:text[]"` // profile, sessions, consents, audit
	ExportURL       string     `json:"exportUrl" bun:"export_url"`
	ExportPath      string     `json:"exportPath" bun:"export_path"`
	ExportSize      int64      `json:"exportSize" bun:"export_size"`         // bytes
	ExpiresAt       *time.Time `json:"expiresAt,omitempty" bun:"expires_at"` // URL expiry
	IPAddress       string     `json:"ipAddress" bun:"ip_address"`
	CompletedAt     *time.Time `json:"completedAt,omitempty" bun:"completed_at"`
	ErrorMessage    string     `json:"errorMessage,omitempty" bun:"error_message"`
	CreatedAt       time.Time  `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt       time.Time  `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

// DataDeletionRequest tracks GDPR right to be forgotten requests
type DataDeletionRequest struct {
	bun.BaseModel `bun:"table:data_deletion_requests,alias:ddr"`

	ID              xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID          string     `json:"userId" bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID  string     `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	Status          string     `json:"status" bun:"status,notnull"` // pending, approved, processing, completed, rejected
	RequestReason   string     `json:"requestReason" bun:"request_reason,type:text"`
	RetentionExempt bool       `json:"retentionExempt" bun:"retention_exempt"` // Legal hold or other exemption
	ExemptionReason string     `json:"exemptionReason" bun:"exemption_reason"`
	DeleteSections  []string   `json:"deleteSections" bun:"delete_sections,type:text[]"` // all, profile, sessions, consents
	IPAddress       string     `json:"ipAddress" bun:"ip_address"`
	ApprovedBy      string     `json:"approvedBy" bun:"approved_by,type:varchar(20)"` // Admin who approved
	ApprovedAt      *time.Time `json:"approvedAt,omitempty" bun:"approved_at"`
	CompletedAt     *time.Time `json:"completedAt,omitempty" bun:"completed_at"`
	RejectedAt      *time.Time `json:"rejectedAt,omitempty" bun:"rejected_at"`
	ErrorMessage    string     `json:"errorMessage,omitempty" bun:"error_message"`
	ArchivePath     string     `json:"archivePath" bun:"archive_path"` // Backup before deletion
	CreatedAt       time.Time  `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt       time.Time  `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

// PrivacySettings stores per-organization privacy configurations
type PrivacySettings struct {
	bun.BaseModel `bun:"table:privacy_settings,alias:ps"`

	ID                              xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID                  string    `json:"organizationId" bun:"organization_id,notnull,unique,type:varchar(20)"`
	ConsentRequired                 bool      `json:"consentRequired" bun:"consent_required,notnull,default:true"`
	CookieConsentEnabled            bool      `json:"cookieConsentEnabled" bun:"cookie_consent_enabled,notnull,default:true"`
	CookieConsentStyle              string    `json:"cookieConsentStyle" bun:"cookie_consent_style"` // banner, modal, popup
	DataRetentionDays               int       `json:"dataRetentionDays" bun:"data_retention_days"`
	AnonymousConsentEnabled         bool      `json:"anonymousConsentEnabled" bun:"anonymous_consent_enabled"`
	GDPRMode                        bool      `json:"gdprMode" bun:"gdpr_mode,notnull,default:false"`
	CCPAMode                        bool      `json:"ccpaMode" bun:"ccpa_mode,notnull,default:false"`
	AutoDeleteAfterDays             int       `json:"autoDeleteAfterDays" bun:"auto_delete_after_days"`
	RequireExplicitConsent          bool      `json:"requireExplicitConsent" bun:"require_explicit_consent"` // No implied consent
	AllowDataPortability            bool      `json:"allowDataPortability" bun:"allow_data_portability,notnull,default:true"`
	ExportFormat                    []string  `json:"exportFormat" bun:"export_format,type:text[]"` // json, csv, xml
	DataExportExpiryHours           int       `json:"dataExportExpiryHours" bun:"data_export_expiry_hours"`
	RequireAdminApprovalForDeletion bool      `json:"requireAdminApprovalForDeletion" bun:"require_admin_approval_for_deletion"`
	DeletionGracePeriodDays         int       `json:"deletionGracePeriodDays" bun:"deletion_grace_period_days"`
	ContactEmail                    string    `json:"contactEmail" bun:"contact_email"`
	ContactPhone                    string    `json:"contactPhone" bun:"contact_phone"`
	DPOEmail                        string    `json:"dpoEmail" bun:"dpo_email"` // Data Protection Officer
	Metadata                        JSONBMap  `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt                       time.Time `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt                       time.Time `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

// JSONBMap is a helper type for JSONB fields
type JSONBMap map[string]interface{}
