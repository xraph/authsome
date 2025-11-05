package schema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// MFAFactor stores enrolled authentication factors for users
type MFAFactor struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:mfa_factors,alias:mff"`

	ID             xid.ID     `bun:"id,pk,type:varchar(20)"`
	UserID         xid.ID     `bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID *xid.ID    `bun:"organization_id,type:varchar(20)"`
	Type           string     `bun:"type,notnull"`        // totp, sms, email, webauthn, backup, etc.
	Status         string     `bun:"status,notnull"`      // pending, active, disabled, revoked
	Priority       string     `bun:"priority,notnull"`    // primary, backup, optional
	Name           string     `bun:"name"`                // User-friendly name
	Secret         string     `bun:"secret"`              // Encrypted factor secret
	Metadata       JSONMap    `bun:"metadata,type:jsonb"` // Factor-specific metadata
	LastUsedAt     *time.Time `bun:"last_used_at"`
	VerifiedAt     *time.Time `bun:"verified_at"`
	ExpiresAt      *time.Time `bun:"expires_at"`
}

// MFAChallenge stores active MFA verification challenges
type MFAChallenge struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:mfa_challenges,alias:mfc"`

	ID          xid.ID     `bun:"id,pk,type:varchar(20)"`
	SessionID   xid.ID     `bun:"session_id,notnull,type:varchar(20)"` // Links to MFA session
	UserID      xid.ID     `bun:"user_id,notnull,type:varchar(20)"`
	FactorID    xid.ID     `bun:"factor_id,notnull,type:varchar(20)"`
	Type        string     `bun:"type,notnull"`   // Factor type
	Status      string     `bun:"status,notnull"` // pending, verified, failed, expired
	CodeHash    string     `bun:"code_hash"`      // Hashed verification code
	Metadata    JSONMap    `bun:"metadata,type:jsonb"`
	Attempts    int        `bun:"attempts,notnull,default:0"`
	MaxAttempts int        `bun:"max_attempts,notnull,default:3"`
	IPAddress   string     `bun:"ip_address"`
	UserAgent   string     `bun:"user_agent"`
	ExpiresAt   time.Time  `bun:"expires_at,notnull"`
	VerifiedAt  *time.Time `bun:"verified_at"`
}

// MFASession represents an MFA verification session
type MFASession struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:mfa_sessions,alias:mfs"`

	ID              xid.ID      `bun:"id,pk,type:varchar(20)"`
	UserID          xid.ID      `bun:"user_id,notnull,type:varchar(20)"`
	SessionToken    string      `bun:"session_token,unique,notnull"`
	FactorsRequired int         `bun:"factors_required,notnull"`
	FactorsVerified int         `bun:"factors_verified,notnull,default:0"`
	VerifiedFactors StringArray `bun:"verified_factors,type:text[]"` // Array of factor IDs
	RiskLevel       string      `bun:"risk_level"`                   // low, medium, high, critical
	RiskScore       float64     `bun:"risk_score"`                   // 0-100
	Context         string      `bun:"context"`                      // login, transaction, step-up
	IPAddress       string      `bun:"ip_address"`
	UserAgent       string      `bun:"user_agent"`
	Metadata        JSONMap     `bun:"metadata,type:jsonb"`
	ExpiresAt       time.Time   `bun:"expires_at,notnull"`
	CompletedAt     *time.Time  `bun:"completed_at"`
}

// MFATrustedDevice stores trusted devices that can skip MFA
type MFATrustedDevice struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:mfa_trusted_devices,alias:mtd"`

	ID         xid.ID     `bun:"id,pk,type:varchar(20)"`
	UserID     xid.ID     `bun:"user_id,notnull,type:varchar(20)"`
	DeviceID   string     `bun:"device_id,notnull"`   // Fingerprint/identifier
	Name       string     `bun:"name"`                // User-friendly name
	Metadata   JSONMap    `bun:"metadata,type:jsonb"` // Device info
	IPAddress  string     `bun:"ip_address"`
	UserAgent  string     `bun:"user_agent"`
	LastUsedAt *time.Time `bun:"last_used_at"`
	ExpiresAt  time.Time  `bun:"expires_at,notnull"`
}

// MFAPolicy defines organization-level MFA requirements
type MFAPolicy struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:mfa_policies,alias:mfp"`

	ID                     xid.ID      `bun:"id,pk,type:varchar(20)"`
	OrganizationID         xid.ID      `bun:"organization_id,unique,notnull,type:varchar(20)"`
	Enabled                bool        `bun:"enabled,notnull,default:true"`
	RequiredFactorCount    int         `bun:"required_factor_count,notnull,default:1"`
	AllowedFactorTypes     StringArray `bun:"allowed_factor_types,type:text[]"`
	RequiredFactorTypes    StringArray `bun:"required_factor_types,type:text[]"`
	GracePeriodDays        int         `bun:"grace_period_days,notnull,default:7"`
	TrustedDeviceDays      int         `bun:"trusted_device_days,notnull,default:30"`
	StepUpRequired         bool        `bun:"step_up_required,notnull,default:false"`
	AdaptiveMFAEnabled     bool        `bun:"adaptive_mfa_enabled,notnull,default:false"`
	MaxFailedAttempts      int         `bun:"max_failed_attempts,notnull,default:5"`
	LockoutDurationMinutes int         `bun:"lockout_duration_minutes,notnull,default:30"`
	Metadata               JSONMap     `bun:"metadata,type:jsonb"`
}

// MFAAttempt tracks verification attempts for rate limiting and security
type MFAAttempt struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:mfa_attempts,alias:mfa"`

	ID            xid.ID  `bun:"id,pk,type:varchar(20)"`
	UserID        xid.ID  `bun:"user_id,notnull,type:varchar(20)"`
	FactorID      *xid.ID `bun:"factor_id,type:varchar(20)"`
	ChallengeID   *xid.ID `bun:"challenge_id,type:varchar(20)"`
	Type          string  `bun:"type,notnull"` // Factor type
	Success       bool    `bun:"success,notnull"`
	FailureReason string  `bun:"failure_reason"`
	IPAddress     string  `bun:"ip_address"`
	UserAgent     string  `bun:"user_agent"`
	Metadata      JSONMap `bun:"metadata,type:jsonb"`
}

// MFARiskAssessment stores risk assessment results
type MFARiskAssessment struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:mfa_risk_assessments,alias:mra"`

	ID          xid.ID      `bun:"id,pk,type:varchar(20)"`
	UserID      xid.ID      `bun:"user_id,notnull,type:varchar(20)"`
	SessionID   *xid.ID     `bun:"session_id,type:varchar(20)"`
	RiskLevel   string      `bun:"risk_level,notnull"`      // low, medium, high, critical
	RiskScore   float64     `bun:"risk_score,notnull"`      // 0-100
	Factors     StringArray `bun:"factors,type:text[]"`     // Risk factors identified
	Recommended StringArray `bun:"recommended,type:text[]"` // Recommended factor types
	IPAddress   string      `bun:"ip_address"`
	UserAgent   string      `bun:"user_agent"`
	Location    string      `bun:"location"` // Geographic location
	Metadata    JSONMap     `bun:"metadata,type:jsonb"`
}

// JSONMap is a helper type for storing JSON data
type JSONMap map[string]interface{}

// Scan implements sql.Scanner for JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONMap value: %v", value)
	}

	result := make(JSONMap)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// Value implements driver.Valuer for JSONMap
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// StringArray is a helper type for storing arrays of strings
type StringArray []string

// Scan implements sql.Scanner for StringArray
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal StringArray value: %v", value)
	}

	var result []string
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*s = result
	return nil
}

// Value implements driver.Valuer for StringArray
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}
