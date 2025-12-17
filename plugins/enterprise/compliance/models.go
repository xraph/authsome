package compliance

import (
	"time"
)

// ComplianceStandard represents different compliance frameworks
type ComplianceStandard string

const (
	StandardSOC2     ComplianceStandard = "SOC2"
	StandardHIPAA    ComplianceStandard = "HIPAA"
	StandardPCIDSS   ComplianceStandard = "PCI-DSS"
	StandardGDPR     ComplianceStandard = "GDPR"
	StandardISO27001 ComplianceStandard = "ISO27001"
	StandardCCPA     ComplianceStandard = "CCPA"
)

// ComplianceProfile defines compliance requirements for an app
type ComplianceProfile struct {
	ID        string               `json:"id" bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	AppID     string               `json:"appId" bun:"app_id,notnull"`
	Name      string               `json:"name" bun:"name,notnull"`
	Standards []ComplianceStandard `json:"standards" bun:"standards,array"`
	Status    string               `json:"status" bun:"status,notnull"` // active, suspended, audit

	// Security Requirements
	MFARequired           bool `json:"mfaRequired" bun:"mfa_required"`
	PasswordMinLength     int  `json:"passwordMinLength" bun:"password_min_length"`
	PasswordRequireUpper  bool `json:"passwordRequireUpper" bun:"password_require_upper"`
	PasswordRequireLower  bool `json:"passwordRequireLower" bun:"password_require_lower"`
	PasswordRequireNumber bool `json:"passwordRequireNumber" bun:"password_require_number"`
	PasswordRequireSymbol bool `json:"passwordRequireSymbol" bun:"password_require_symbol"`
	PasswordExpiryDays    int  `json:"passwordExpiryDays" bun:"password_expiry_days"` // 0 = never

	// Session Requirements
	SessionMaxAge      int  `json:"sessionMaxAge" bun:"session_max_age"`           // seconds
	SessionIdleTimeout int  `json:"sessionIdleTimeout" bun:"session_idle_timeout"` // seconds
	SessionIPBinding   bool `json:"sessionIpBinding" bun:"session_ip_binding"`

	// Audit Requirements
	RetentionDays      int  `json:"retentionDays" bun:"retention_days"`
	AuditLogExport     bool `json:"auditLogExport" bun:"audit_log_export"`
	DetailedAuditTrail bool `json:"detailedAuditTrail" bun:"detailed_audit_trail"`

	// Data Requirements
	DataResidency       string `json:"dataResidency" bun:"data_residency"` // US, EU, APAC
	EncryptionAtRest    bool   `json:"encryptionAtRest" bun:"encryption_at_rest"`
	EncryptionInTransit bool   `json:"encryptionInTransit" bun:"encryption_in_transit"`

	// Access Control
	RBACRequired        bool `json:"rbacRequired" bun:"rbac_required"`
	LeastPrivilege      bool `json:"leastPrivilege" bun:"least_privilege"`
	RegularAccessReview bool `json:"regularAccessReview" bun:"regular_access_review"`

	// Contact
	ComplianceContact string `json:"complianceContact" bun:"compliance_contact"`
	DPOContact        string `json:"dpoContact" bun:"dpo_contact"` // Data Protection Officer

	// Metadata
	Metadata  map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt time.Time              `json:"createdAt" bun:"created_at,notnull,default:now()"`
	UpdatedAt time.Time              `json:"updatedAt" bun:"updated_at,notnull,default:now()"`
}

// ComplianceCheck represents an automated compliance check
type ComplianceCheck struct {
	ID            string                 `json:"id" bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ProfileID     string                 `json:"profileId" bun:"profile_id,notnull"`
	AppID         string                 `json:"appId" bun:"app_id,notnull"`
	CheckType     string                 `json:"checkType" bun:"check_type,notnull"` // mfa_coverage, password_policy, etc.
	Status        string                 `json:"status" bun:"status,notnull"`        // passed, failed, warning
	Result        map[string]interface{} `json:"result" bun:"result,type:jsonb"`
	Evidence      []string               `json:"evidence" bun:"evidence,array"`
	LastCheckedAt time.Time              `json:"lastCheckedAt" bun:"last_checked_at,notnull"`
	NextCheckAt   time.Time              `json:"nextCheckAt" bun:"next_check_at,notnull"`
	CreatedAt     time.Time              `json:"createdAt" bun:"created_at,notnull,default:now()"`
}

// ComplianceViolation represents a policy violation
type ComplianceViolation struct {
	ID            string                 `json:"id" bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ProfileID     string                 `json:"profileId" bun:"profile_id,notnull"`
	AppID         string                 `json:"appId" bun:"app_id,notnull"`
	UserID        string                 `json:"userId" bun:"user_id"`
	ViolationType string                 `json:"violationType" bun:"violation_type,notnull"` // mfa_not_enabled, weak_password, etc.
	Severity      string                 `json:"severity" bun:"severity,notnull"`            // low, medium, high, critical
	Description   string                 `json:"description" bun:"description,notnull"`
	Status        string                 `json:"status" bun:"status,notnull"` // open, resolved, acknowledged
	ResolvedAt    *time.Time             `json:"resolvedAt" bun:"resolved_at"`
	ResolvedBy    string                 `json:"resolvedBy" bun:"resolved_by"`
	Metadata      map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt     time.Time              `json:"createdAt" bun:"created_at,notnull,default:now()"`
}

// ComplianceReport represents a generated compliance report
type ComplianceReport struct {
	ID          string                 `json:"id" bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ProfileID   string                 `json:"profileId" bun:"profile_id,notnull"`
	AppID       string                 `json:"appId" bun:"app_id,notnull"`
	ReportType  string                 `json:"reportType" bun:"report_type,notnull"` // soc2, hipaa, audit_export
	Standard    ComplianceStandard     `json:"standard" bun:"standard"`
	Period      string                 `json:"period" bun:"period,notnull"` // 2025-Q1, 2025-11
	Format      string                 `json:"format" bun:"format,notnull"` // pdf, json, csv
	Status      string                 `json:"status" bun:"status,notnull"` // generating, ready, failed
	FileURL     string                 `json:"fileUrl" bun:"file_url"`
	FileSize    int64                  `json:"fileSize" bun:"file_size"`
	Summary     map[string]interface{} `json:"summary" bun:"summary,type:jsonb"`
	GeneratedBy string                 `json:"generatedBy" bun:"generated_by"`
	CreatedAt   time.Time              `json:"createdAt" bun:"created_at,notnull,default:now()"`
	ExpiresAt   time.Time              `json:"expiresAt" bun:"expires_at"`
}

// ComplianceEvidence stores evidence for compliance audits
type ComplianceEvidence struct {
	ID           string                 `json:"id" bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ProfileID    string                 `json:"profileId" bun:"profile_id,notnull"`
	AppID        string                 `json:"appId" bun:"app_id,notnull"`
	EvidenceType string                 `json:"evidenceType" bun:"evidence_type,notnull"` // audit_log, policy_doc, etc.
	Standard     ComplianceStandard     `json:"standard" bun:"standard"`
	ControlID    string                 `json:"controlId" bun:"control_id"` // e.g., SOC2-CC6.1
	Title        string                 `json:"title" bun:"title,notnull"`
	Description  string                 `json:"description" bun:"description"`
	FileURL      string                 `json:"fileUrl" bun:"file_url"`
	FileHash     string                 `json:"fileHash" bun:"file_hash"` // SHA256 for integrity
	CollectedBy  string                 `json:"collectedBy" bun:"collected_by"`
	Metadata     map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt    time.Time              `json:"createdAt" bun:"created_at,notnull,default:now()"`
}

// CompliancePolicy represents a policy document
type CompliancePolicy struct {
	ID            string                 `json:"id" bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ProfileID     string                 `json:"profileId" bun:"profile_id,notnull"`
	AppID         string                 `json:"appId" bun:"app_id,notnull"`
	PolicyType    string                 `json:"policyType" bun:"policy_type,notnull"` // password, access, data_retention
	Standard      ComplianceStandard     `json:"standard" bun:"standard"`
	Title         string                 `json:"title" bun:"title,notnull"`
	Version       string                 `json:"version" bun:"version,notnull"`
	Content       string                 `json:"content" bun:"content,notnull"`
	Status        string                 `json:"status" bun:"status,notnull"` // draft, active, deprecated
	ApprovedBy    string                 `json:"approvedBy" bun:"approved_by"`
	ApprovedAt    *time.Time             `json:"approvedAt" bun:"approved_at"`
	EffectiveDate time.Time              `json:"effectiveDate" bun:"effective_date,notnull"`
	ReviewDate    time.Time              `json:"reviewDate" bun:"review_date,notnull"`
	Metadata      map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt     time.Time              `json:"createdAt" bun:"created_at,notnull,default:now()"`
	UpdatedAt     time.Time              `json:"updatedAt" bun:"updated_at,notnull,default:now()"`
}

// ComplianceTraining tracks compliance training completion
type ComplianceTraining struct {
	ID           string                 `json:"id" bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ProfileID    string                 `json:"profileId" bun:"profile_id,notnull"`
	AppID        string                 `json:"appId" bun:"app_id,notnull"`
	UserID       string                 `json:"userId" bun:"user_id,notnull"`
	TrainingType string                 `json:"trainingType" bun:"training_type,notnull"` // security_awareness, hipaa_basics
	Standard     ComplianceStandard     `json:"standard" bun:"standard"`
	Status       string                 `json:"status" bun:"status,notnull"` // required, in_progress, completed
	CompletedAt  *time.Time             `json:"completedAt" bun:"completed_at"`
	ExpiresAt    *time.Time             `json:"expiresAt" bun:"expires_at"`
	Score        int                    `json:"score" bun:"score"` // percentage
	Metadata     map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt    time.Time              `json:"createdAt" bun:"created_at,notnull,default:now()"`
}

// ComplianceStatus represents overall compliance status
type ComplianceStatus struct {
	ProfileID     string             `json:"profileId"`
	AppID         string             `json:"appId"`
	Standard      ComplianceStandard `json:"standard"`
	OverallStatus string             `json:"overallStatus"` // compliant, non_compliant, in_progress
	Score         int                `json:"score"`         // 0-100
	ChecksPassed  int                `json:"checksPassed"`
	ChecksFailed  int                `json:"checksFailed"`
	ChecksWarning int                `json:"checksWarning"`
	Violations    int                `json:"violations"`
	LastChecked   time.Time          `json:"lastChecked"`
	NextAudit     time.Time          `json:"nextAudit"`
}

// ComplianceTemplate represents a predefined compliance template
type ComplianceTemplate struct {
	Standard           ComplianceStandard `json:"standard"`
	Name               string             `json:"name"`
	Description        string             `json:"description"`
	MFARequired        bool               `json:"mfaRequired"`
	PasswordMinLength  int                `json:"passwordMinLength"`
	SessionMaxAge      int                `json:"sessionMaxAge"`
	RetentionDays      int                `json:"retentionDays"`
	DataResidency      string             `json:"dataResidency"`
	RequiredPolicies   []string           `json:"requiredPolicies"`
	RequiredTraining   []string           `json:"requiredTraining"`
	AuditFrequencyDays int                `json:"auditFrequencyDays"`
}
