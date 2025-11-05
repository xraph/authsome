package stepup

import "time"

// StepUpVerification represents a step-up authentication verification record
type StepUpVerification struct {
	ID            string                 `bun:"id,pk" json:"id"`
	UserID        string                 `bun:"user_id,notnull" json:"user_id"`
	OrgID         string                 `bun:"org_id,notnull" json:"org_id"`
	SessionID     string                 `bun:"session_id" json:"session_id"`
	SecurityLevel SecurityLevel          `bun:"security_level,notnull" json:"security_level"`
	Method        VerificationMethod     `bun:"method,notnull" json:"method"`
	IP            string                 `bun:"ip" json:"ip"`
	UserAgent     string                 `bun:"user_agent" json:"user_agent"`
	DeviceID      string                 `bun:"device_id" json:"device_id"`
	Reason        string                 `bun:"reason" json:"reason"`       // Why step-up was required
	RuleName      string                 `bun:"rule_name" json:"rule_name"` // Which rule triggered
	VerifiedAt    time.Time              `bun:"verified_at,notnull" json:"verified_at"`
	ExpiresAt     time.Time              `bun:"expires_at,notnull" json:"expires_at"`
	Metadata      map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt     time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

// StepUpRequirement represents a step-up requirement record
type StepUpRequirement struct {
	ID             string                 `bun:"id,pk" json:"id"`
	UserID         string                 `bun:"user_id,notnull" json:"user_id"`
	OrgID          string                 `bun:"org_id,notnull" json:"org_id"`
	SessionID      string                 `bun:"session_id" json:"session_id"`
	RequiredLevel  SecurityLevel          `bun:"required_level,notnull" json:"required_level"`
	CurrentLevel   SecurityLevel          `bun:"current_level,notnull" json:"current_level"`
	Route          string                 `bun:"route" json:"route"`
	Method         string                 `bun:"method" json:"method"`
	Amount         float64                `bun:"amount" json:"amount,omitempty"`
	Currency       string                 `bun:"currency" json:"currency,omitempty"`
	ResourceType   string                 `bun:"resource_type" json:"resource_type,omitempty"`
	ResourceAction string                 `bun:"resource_action" json:"resource_action,omitempty"`
	RuleName       string                 `bun:"rule_name" json:"rule_name"`
	Reason         string                 `bun:"reason" json:"reason"`
	Status         string                 `bun:"status,notnull" json:"status"` // pending, fulfilled, expired, bypassed
	ChallengeToken string                 `bun:"challenge_token" json:"challenge_token,omitempty"`
	IP             string                 `bun:"ip" json:"ip"`
	UserAgent      string                 `bun:"user_agent" json:"user_agent"`
	RiskScore      float64                `bun:"risk_score" json:"risk_score,omitempty"`
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt      time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	ExpiresAt      time.Time              `bun:"expires_at,notnull" json:"expires_at"`
	FulfilledAt    *time.Time             `bun:"fulfilled_at" json:"fulfilled_at,omitempty"`
}

// StepUpRememberedDevice represents a device that's remembered for step-up bypass
type StepUpRememberedDevice struct {
	ID            string        `bun:"id,pk" json:"id"`
	UserID        string        `bun:"user_id,notnull" json:"user_id"`
	OrgID         string        `bun:"org_id,notnull" json:"org_id"`
	DeviceID      string        `bun:"device_id,notnull" json:"device_id"`
	DeviceName    string        `bun:"device_name" json:"device_name"`
	SecurityLevel SecurityLevel `bun:"security_level,notnull" json:"security_level"`
	IP            string        `bun:"ip" json:"ip"`
	UserAgent     string        `bun:"user_agent" json:"user_agent"`
	RememberedAt  time.Time     `bun:"remembered_at,notnull" json:"remembered_at"`
	ExpiresAt     time.Time     `bun:"expires_at,notnull" json:"expires_at"`
	LastUsedAt    time.Time     `bun:"last_used_at" json:"last_used_at"`
	CreatedAt     time.Time     `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

// StepUpAttempt represents an attempt to complete step-up authentication
type StepUpAttempt struct {
	ID            string             `bun:"id,pk" json:"id"`
	RequirementID string             `bun:"requirement_id,notnull" json:"requirement_id"`
	UserID        string             `bun:"user_id,notnull" json:"user_id"`
	OrgID         string             `bun:"org_id,notnull" json:"org_id"`
	Method        VerificationMethod `bun:"method,notnull" json:"method"`
	Success       bool               `bun:"success,notnull" json:"success"`
	FailureReason string             `bun:"failure_reason" json:"failure_reason,omitempty"`
	IP            string             `bun:"ip" json:"ip"`
	UserAgent     string             `bun:"user_agent" json:"user_agent"`
	CreatedAt     time.Time          `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

// StepUpPolicy represents an organization or user-specific step-up policy
type StepUpPolicy struct {
	ID          string                 `bun:"id,pk" json:"id"`
	OrgID       string                 `bun:"org_id,notnull" json:"org_id"`
	UserID      string                 `bun:"user_id" json:"user_id,omitempty"` // Optional: user-specific override
	Name        string                 `bun:"name,notnull" json:"name"`
	Description string                 `bun:"description" json:"description"`
	Enabled     bool                   `bun:"enabled,notnull,default:true" json:"enabled"`
	Priority    int                    `bun:"priority,notnull,default:0" json:"priority"` // Higher priority = evaluated first
	Rules       map[string]interface{} `bun:"rules,type:jsonb,notnull" json:"rules"`
	Metadata    map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt   time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time              `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// StepUpAuditLog represents audit logs for step-up events
type StepUpAuditLog struct {
	ID        string                 `bun:"id,pk" json:"id"`
	UserID    string                 `bun:"user_id,notnull" json:"user_id"`
	OrgID     string                 `bun:"org_id,notnull" json:"org_id"`
	EventType string                 `bun:"event_type,notnull" json:"event_type"`
	EventData map[string]interface{} `bun:"event_data,type:jsonb" json:"event_data"`
	IP        string                 `bun:"ip" json:"ip"`
	UserAgent string                 `bun:"user_agent" json:"user_agent"`
	Severity  string                 `bun:"severity,notnull" json:"severity"` // info, warning, critical
	CreatedAt time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

// Table names
func (*StepUpVerification) TableName() string     { return "stepup_verifications" }
func (*StepUpRequirement) TableName() string      { return "stepup_requirements" }
func (*StepUpRememberedDevice) TableName() string { return "stepup_remembered_devices" }
func (*StepUpAttempt) TableName() string          { return "stepup_attempts" }
func (*StepUpPolicy) TableName() string           { return "stepup_policies" }
func (*StepUpAuditLog) TableName() string         { return "stepup_audit_logs" }
