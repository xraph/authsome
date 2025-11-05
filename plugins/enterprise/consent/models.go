package consent

import "time"

// ConsentType represents different types of consent
type ConsentType string

const (
	ConsentTypeCookies        ConsentType = "cookies"
	ConsentTypeMarketing      ConsentType = "marketing"
	ConsentTypeAnalytics      ConsentType = "analytics"
	ConsentTypeTerms          ConsentType = "terms"
	ConsentTypePrivacy        ConsentType = "privacy"
	ConsentTypeDataProcessing ConsentType = "data_processing"
	ConsentTypeThirdParty     ConsentType = "third_party"
	ConsentTypeCommunications ConsentType = "communications"
)

// AgreementType represents different types of data processing agreements
type AgreementType string

const (
	AgreementTypeDPA  AgreementType = "dpa"  // Data Processing Agreement
	AgreementTypeBAA  AgreementType = "baa"  // Business Associate Agreement (HIPAA)
	AgreementTypeCCPA AgreementType = "ccpa" // California Consumer Privacy Act
	AgreementTypeGDPR AgreementType = "gdpr" // General Data Protection Regulation
)

// ConsentAction represents actions in audit log
type ConsentAction string

const (
	ActionGranted ConsentAction = "granted"
	ActionRevoked ConsentAction = "revoked"
	ActionUpdated ConsentAction = "updated"
	ActionExpired ConsentAction = "expired"
	ActionRenewed ConsentAction = "renewed"
)

// RequestStatus represents the status of data export/deletion requests
type RequestStatus string

const (
	StatusPending    RequestStatus = "pending"
	StatusApproved   RequestStatus = "approved"
	StatusProcessing RequestStatus = "processing"
	StatusCompleted  RequestStatus = "completed"
	StatusFailed     RequestStatus = "failed"
	StatusRejected   RequestStatus = "rejected"
)

// ExportFormat represents data export formats
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
	FormatXML  ExportFormat = "xml"
	FormatPDF  ExportFormat = "pdf"
)

// CreateConsentRequest represents a request to record consent
type CreateConsentRequest struct {
	UserID      string                 `json:"userId" validate:"required"`
	ConsentType string                 `json:"consentType" validate:"required"`
	Purpose     string                 `json:"purpose" validate:"required"`
	Granted     bool                   `json:"granted"`
	Version     string                 `json:"version" validate:"required"`
	ExpiresIn   *int                   `json:"expiresIn,omitempty"` // Days until expiry
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateConsentRequest represents a request to update consent
type UpdateConsentRequest struct {
	Granted  *bool                  `json:"granted,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Reason   string                 `json:"reason,omitempty"`
}

// CreatePolicyRequest represents a request to create a consent policy
type CreatePolicyRequest struct {
	ConsentType    string                 `json:"consentType" validate:"required"`
	Name           string                 `json:"name" validate:"required"`
	Description    string                 `json:"description"`
	Version        string                 `json:"version" validate:"required"`
	Content        string                 `json:"content" validate:"required"`
	Required       bool                   `json:"required"`
	Renewable      bool                   `json:"renewable"`
	ValidityPeriod *int                   `json:"validityPeriod,omitempty"` // Days
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UpdatePolicyRequest represents a request to update a policy
type UpdatePolicyRequest struct {
	Name           string                 `json:"name,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Content        string                 `json:"content,omitempty"`
	Required       *bool                  `json:"required,omitempty"`
	Renewable      *bool                  `json:"renewable,omitempty"`
	ValidityPeriod *int                   `json:"validityPeriod,omitempty"`
	Active         *bool                  `json:"active,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// CreateDPARequest represents a request to create a data processing agreement
type CreateDPARequest struct {
	AgreementType string                 `json:"agreementType" validate:"required"`
	Version       string                 `json:"version" validate:"required"`
	Content       string                 `json:"content" validate:"required"`
	SignedByName  string                 `json:"signedByName" validate:"required"`
	SignedByTitle string                 `json:"signedByTitle" validate:"required"`
	SignedByEmail string                 `json:"signedByEmail" validate:"required,email"`
	EffectiveDate time.Time              `json:"effectiveDate" validate:"required"`
	ExpiryDate    *time.Time             `json:"expiryDate,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CookieConsentRequest represents a cookie consent preference
type CookieConsentRequest struct {
	Essential       bool   `json:"essential"`
	Functional      bool   `json:"functional"`
	Analytics       bool   `json:"analytics"`
	Marketing       bool   `json:"marketing"`
	Personalization bool   `json:"personalization"`
	ThirdParty      bool   `json:"thirdParty"`
	SessionID       string `json:"sessionId,omitempty"` // For anonymous users
	BannerVersion   string `json:"bannerVersion,omitempty"`
}

// DataExportRequestInput represents a data export request
type DataExportRequestInput struct {
	Format          string   `json:"format" validate:"required,oneof=json csv xml pdf"`
	IncludeSections []string `json:"includeSections,omitempty"` // profile, sessions, consents, audit, all
}

// DataDeletionRequestInput represents a data deletion request
type DataDeletionRequestInput struct {
	Reason         string   `json:"reason" validate:"required"`
	DeleteSections []string `json:"deleteSections,omitempty"` // all, profile, sessions, consents
}

// ConsentSummary provides a summary of user's consent status
type ConsentSummary struct {
	UserID             string                       `json:"userId"`
	OrganizationID     string                       `json:"organizationId"`
	TotalConsents      int                          `json:"totalConsents"`
	GrantedConsents    int                          `json:"grantedConsents"`
	RevokedConsents    int                          `json:"revokedConsents"`
	ExpiredConsents    int                          `json:"expiredConsents"`
	PendingRenewals    int                          `json:"pendingRenewals"`
	ConsentsByType     map[string]ConsentTypeStatus `json:"consentsByType"`
	LastConsentUpdate  *time.Time                   `json:"lastConsentUpdate,omitempty"`
	HasPendingDeletion bool                         `json:"hasPendingDeletion"`
	HasPendingExport   bool                         `json:"hasPendingExport"`
}

// ConsentTypeStatus represents consent status for a specific type
type ConsentTypeStatus struct {
	Type         string     `json:"type"`
	Granted      bool       `json:"granted"`
	Version      string     `json:"version"`
	GrantedAt    time.Time  `json:"grantedAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	NeedsRenewal bool       `json:"needsRenewal"`
}

// PrivacySettingsRequest represents a request to update privacy settings
type PrivacySettingsRequest struct {
	ConsentRequired                 *bool    `json:"consentRequired,omitempty"`
	CookieConsentEnabled            *bool    `json:"cookieConsentEnabled,omitempty"`
	CookieConsentStyle              string   `json:"cookieConsentStyle,omitempty"`
	DataRetentionDays               *int     `json:"dataRetentionDays,omitempty"`
	AnonymousConsentEnabled         *bool    `json:"anonymousConsentEnabled,omitempty"`
	GDPRMode                        *bool    `json:"gdprMode,omitempty"`
	CCPAMode                        *bool    `json:"ccpaMode,omitempty"`
	AutoDeleteAfterDays             *int     `json:"autoDeleteAfterDays,omitempty"`
	RequireExplicitConsent          *bool    `json:"requireExplicitConsent,omitempty"`
	AllowDataPortability            *bool    `json:"allowDataPortability,omitempty"`
	ExportFormat                    []string `json:"exportFormat,omitempty"`
	DataExportExpiryHours           *int     `json:"dataExportExpiryHours,omitempty"`
	RequireAdminApprovalForDeletion *bool    `json:"requireAdminApprovalForDeletion,omitempty"`
	DeletionGracePeriodDays         *int     `json:"deletionGracePeriodDays,omitempty"`
	ContactEmail                    string   `json:"contactEmail,omitempty"`
	ContactPhone                    string   `json:"contactPhone,omitempty"`
	DPOEmail                        string   `json:"dpoEmail,omitempty"`
}

// ConsentReport provides analytics and reporting data
type ConsentReport struct {
	OrganizationID        string                  `json:"organizationId"`
	ReportPeriodStart     time.Time               `json:"reportPeriodStart"`
	ReportPeriodEnd       time.Time               `json:"reportPeriodEnd"`
	TotalUsers            int                     `json:"totalUsers"`
	UsersWithConsent      int                     `json:"usersWithConsent"`
	ConsentRate           float64                 `json:"consentRate"`
	ConsentsByType        map[string]ConsentStats `json:"consentsByType"`
	PendingDeletions      int                     `json:"pendingDeletions"`
	CompletedDeletions    int                     `json:"completedDeletions"`
	DataExportsThisPeriod int                     `json:"dataExportsThisPeriod"`
	DPAsActive            int                     `json:"dpasActive"`
	DPAsExpiringSoon      int                     `json:"dpasExpiringSoon"`
}

// ConsentStats provides statistics for a consent type
type ConsentStats struct {
	Type            string  `json:"type"`
	TotalConsents   int     `json:"totalConsents"`
	GrantedCount    int     `json:"grantedCount"`
	RevokedCount    int     `json:"revokedCount"`
	ExpiredCount    int     `json:"expiredCount"`
	GrantRate       float64 `json:"grantRate"`
	AverageLifetime int     `json:"averageLifetime"` // Days
}
