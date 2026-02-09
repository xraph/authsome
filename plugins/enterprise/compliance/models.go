package compliance

import (
	"time"

	"github.com/uptrace/bun"
)

// ComplianceStandard represents different compliance frameworks.
type ComplianceStandard string

const (
	StandardSOC2     ComplianceStandard = "SOC2"
	StandardHIPAA    ComplianceStandard = "HIPAA"
	StandardPCIDSS   ComplianceStandard = "PCI-DSS"
	StandardGDPR     ComplianceStandard = "GDPR"
	StandardISO27001 ComplianceStandard = "ISO27001"
	StandardCCPA     ComplianceStandard = "CCPA"
)

// ComplianceProfile defines compliance requirements for an app.
type ComplianceProfile struct {
	bun.BaseModel `bun:"table:compliance_profiles,alias:cp"`

	ID        string               `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	AppID     string               `bun:"app_id,notnull"                            json:"appId"`
	Name      string               `bun:"name,notnull"                              json:"name"`
	Standards []ComplianceStandard `bun:"standards,array"                           json:"standards"`
	Status    string               `bun:"status,notnull"                            json:"status"` // active, suspended, audit

	// Security Requirements
	MFARequired           bool `bun:"mfa_required"            json:"mfaRequired"`
	PasswordMinLength     int  `bun:"password_min_length"     json:"passwordMinLength"`
	PasswordRequireUpper  bool `bun:"password_require_upper"  json:"passwordRequireUpper"`
	PasswordRequireLower  bool `bun:"password_require_lower"  json:"passwordRequireLower"`
	PasswordRequireNumber bool `bun:"password_require_number" json:"passwordRequireNumber"`
	PasswordRequireSymbol bool `bun:"password_require_symbol" json:"passwordRequireSymbol"`
	PasswordExpiryDays    int  `bun:"password_expiry_days"    json:"passwordExpiryDays"` // 0 = never

	// Session Requirements
	SessionMaxAge      int  `bun:"session_max_age"      json:"sessionMaxAge"`      // seconds
	SessionIdleTimeout int  `bun:"session_idle_timeout" json:"sessionIdleTimeout"` // seconds
	SessionIPBinding   bool `bun:"session_ip_binding"   json:"sessionIpBinding"`

	// Audit Requirements
	RetentionDays      int  `bun:"retention_days"       json:"retentionDays"`
	AuditLogExport     bool `bun:"audit_log_export"     json:"auditLogExport"`
	DetailedAuditTrail bool `bun:"detailed_audit_trail" json:"detailedAuditTrail"`

	// Data Requirements
	DataResidency       string `bun:"data_residency"        json:"dataResidency"` // US, EU, APAC
	EncryptionAtRest    bool   `bun:"encryption_at_rest"    json:"encryptionAtRest"`
	EncryptionInTransit bool   `bun:"encryption_in_transit" json:"encryptionInTransit"`

	// Access Control
	RBACRequired        bool `bun:"rbac_required"         json:"rbacRequired"`
	LeastPrivilege      bool `bun:"least_privilege"       json:"leastPrivilege"`
	RegularAccessReview bool `bun:"regular_access_review" json:"regularAccessReview"`

	// Contact
	ComplianceContact string `bun:"compliance_contact" json:"complianceContact"`
	DPOContact        string `bun:"dpo_contact"        json:"dpoContact"` // Data Protection Officer

	// Metadata
	Metadata  map[string]any `bun:"metadata,type:jsonb"              json:"metadata"`
	CreatedAt time.Time      `bun:"created_at,notnull,default:now()" json:"createdAt"`
	UpdatedAt time.Time      `bun:"updated_at,notnull,default:now()" json:"updatedAt"`
}

// ComplianceCheck represents an automated compliance check.
type ComplianceCheck struct {
	bun.BaseModel `bun:"table:compliance_checks,alias:cc"`

	ID            string         `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ProfileID     string         `bun:"profile_id,notnull"                        json:"profileId"`
	AppID         string         `bun:"app_id,notnull"                            json:"appId"`
	CheckType     string         `bun:"check_type,notnull"                        json:"checkType"` // mfa_coverage, password_policy, etc.
	Status        string         `bun:"status,notnull"                            json:"status"`    // passed, failed, warning
	Result        map[string]any `bun:"result,type:jsonb"                         json:"result"`
	Evidence      []string       `bun:"evidence,array"                            json:"evidence"`
	LastCheckedAt time.Time      `bun:"last_checked_at,notnull"                   json:"lastCheckedAt"`
	NextCheckAt   time.Time      `bun:"next_check_at,notnull"                     json:"nextCheckAt"`
	CreatedAt     time.Time      `bun:"created_at,notnull,default:now()"          json:"createdAt"`
}

// ComplianceViolation represents a policy violation.
type ComplianceViolation struct {
	bun.BaseModel `bun:"table:compliance_violations,alias:cv"`

	ID            string         `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ProfileID     string         `bun:"profile_id,notnull"                        json:"profileId"`
	AppID         string         `bun:"app_id,notnull"                            json:"appId"`
	UserID        string         `bun:"user_id"                                   json:"userId"`
	ViolationType string         `bun:"violation_type,notnull"                    json:"violationType"` // mfa_not_enabled, weak_password, etc.
	Severity      string         `bun:"severity,notnull"                          json:"severity"`      // low, medium, high, critical
	Description   string         `bun:"description,notnull"                       json:"description"`
	Status        string         `bun:"status,notnull"                            json:"status"` // open, resolved, acknowledged
	ResolvedAt    *time.Time     `bun:"resolved_at"                               json:"resolvedAt"`
	ResolvedBy    string         `bun:"resolved_by"                               json:"resolvedBy"`
	Metadata      map[string]any `bun:"metadata,type:jsonb"                       json:"metadata"`
	CreatedAt     time.Time      `bun:"created_at,notnull,default:now()"          json:"createdAt"`
}

// ComplianceReport represents a generated compliance report.
type ComplianceReport struct {
	bun.BaseModel `bun:"table:compliance_reports,alias:cr"`

	ID          string             `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ProfileID   string             `bun:"profile_id,notnull"                        json:"profileId"`
	AppID       string             `bun:"app_id,notnull"                            json:"appId"`
	ReportType  string             `bun:"report_type,notnull"                       json:"reportType"` // soc2, hipaa, audit_export
	Standard    ComplianceStandard `bun:"standard"                                  json:"standard"`
	Period      string             `bun:"period,notnull"                            json:"period"` // 2025-Q1, 2025-11
	Format      string             `bun:"format,notnull"                            json:"format"` // pdf, json, csv
	Status      string             `bun:"status,notnull"                            json:"status"` // generating, ready, failed
	FileURL     string             `bun:"file_url"                                  json:"fileUrl"`
	FileSize    int64              `bun:"file_size"                                 json:"fileSize"`
	Summary     map[string]any     `bun:"summary,type:jsonb"                        json:"summary"`
	GeneratedBy string             `bun:"generated_by"                              json:"generatedBy"`
	CreatedAt   time.Time          `bun:"created_at,notnull,default:now()"          json:"createdAt"`
	ExpiresAt   time.Time          `bun:"expires_at"                                json:"expiresAt"`
}

// ComplianceEvidence stores evidence for compliance audits.
type ComplianceEvidence struct {
	bun.BaseModel `bun:"table:compliance_evidence,alias:ce"`

	ID           string             `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ProfileID    string             `bun:"profile_id,notnull"                        json:"profileId"`
	AppID        string             `bun:"app_id,notnull"                            json:"appId"`
	EvidenceType string             `bun:"evidence_type,notnull"                     json:"evidenceType"` // audit_log, policy_doc, etc.
	Standard     ComplianceStandard `bun:"standard"                                  json:"standard"`
	ControlID    string             `bun:"control_id"                                json:"controlId"` // e.g., SOC2-CC6.1
	Title        string             `bun:"title,notnull"                             json:"title"`
	Description  string             `bun:"description"                               json:"description"`
	FileURL      string             `bun:"file_url"                                  json:"fileUrl"`
	FileHash     string             `bun:"file_hash"                                 json:"fileHash"` // SHA256 for integrity
	CollectedBy  string             `bun:"collected_by"                              json:"collectedBy"`
	Metadata     map[string]any     `bun:"metadata,type:jsonb"                       json:"metadata"`
	CreatedAt    time.Time          `bun:"created_at,notnull,default:now()"          json:"createdAt"`
}

// CompliancePolicy represents a policy document.
type CompliancePolicy struct {
	bun.BaseModel `bun:"table:compliance_policies,alias:cpol"`

	ID            string             `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ProfileID     string             `bun:"profile_id,notnull"                        json:"profileId"`
	AppID         string             `bun:"app_id,notnull"                            json:"appId"`
	PolicyType    string             `bun:"policy_type,notnull"                       json:"policyType"` // password, access, data_retention
	Standard      ComplianceStandard `bun:"standard"                                  json:"standard"`
	Title         string             `bun:"title,notnull"                             json:"title"`
	Version       string             `bun:"version,notnull"                           json:"version"`
	Content       string             `bun:"content,notnull"                           json:"content"`
	Status        string             `bun:"status,notnull"                            json:"status"` // draft, active, deprecated
	ApprovedBy    string             `bun:"approved_by"                               json:"approvedBy"`
	ApprovedAt    *time.Time         `bun:"approved_at"                               json:"approvedAt"`
	EffectiveDate time.Time          `bun:"effective_date,notnull"                    json:"effectiveDate"`
	ReviewDate    time.Time          `bun:"review_date,notnull"                       json:"reviewDate"`
	Metadata      map[string]any     `bun:"metadata,type:jsonb"                       json:"metadata"`
	CreatedAt     time.Time          `bun:"created_at,notnull,default:now()"          json:"createdAt"`
	UpdatedAt     time.Time          `bun:"updated_at,notnull,default:now()"          json:"updatedAt"`
}

// ComplianceTraining tracks compliance training completion.
type ComplianceTraining struct {
	bun.BaseModel `bun:"table:compliance_training,alias:ct"`

	ID           string             `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ProfileID    string             `bun:"profile_id,notnull"                        json:"profileId"`
	AppID        string             `bun:"app_id,notnull"                            json:"appId"`
	UserID       string             `bun:"user_id,notnull"                           json:"userId"`
	TrainingType string             `bun:"training_type,notnull"                     json:"trainingType"` // security_awareness, hipaa_basics
	Standard     ComplianceStandard `bun:"standard"                                  json:"standard"`
	Status       string             `bun:"status,notnull"                            json:"status"` // required, in_progress, completed
	CompletedAt  *time.Time         `bun:"completed_at"                              json:"completedAt"`
	ExpiresAt    *time.Time         `bun:"expires_at"                                json:"expiresAt"`
	Score        int                `bun:"score"                                     json:"score"` // percentage
	Metadata     map[string]any     `bun:"metadata,type:jsonb"                       json:"metadata"`
	CreatedAt    time.Time          `bun:"created_at,notnull,default:now()"          json:"createdAt"`
}

// ComplianceStatus represents overall compliance status.
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

// ComplianceTemplate represents a predefined compliance template.
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
