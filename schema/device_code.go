package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// DeviceCode represents an OAuth 2.0 device authorization code (RFC 8628)
type DeviceCode struct {
	AuditableModel
	bun.BaseModel `bun:"table:device_codes"`

	// Context fields
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appID"`
	EnvironmentID  xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentID"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"` // Optional org context

	// OAuth client
	ClientID string `bun:"client_id,notnull" json:"clientID"` // OAuth client ID

	// Device Flow Fields (RFC 8628)
	DeviceCode      string    `bun:"device_code,unique,notnull" json:"deviceCode"`    // Long random string (URL-safe)
	UserCode        string    `bun:"user_code,unique,notnull" json:"userCode"`        // Short human-typable code (e.g., "WDJB-MJHT")
	VerificationURI string    `bun:"verification_uri,notnull" json:"verificationURI"` // Where user goes to authorize
	ExpiresAt       time.Time `bun:"expires_at,notnull" json:"expiresAt"`             // Code expiration
	Interval        int       `bun:"interval,notnull,default:5" json:"interval"`      // Polling interval in seconds
	Scope           string    `bun:"scope" json:"scope,omitempty"`                    // Requested scopes

	// Authorization State
	Status    string  `bun:"status,notnull,default:'pending'" json:"status"`         // pending, authorized, denied, expired, consumed
	UserID    *xid.ID `bun:"user_id,type:varchar(20)" json:"userID,omitempty"`       // Set when user authorizes
	SessionID *xid.ID `bun:"session_id,type:varchar(20)" json:"sessionID,omitempty"` // Set when user authorizes

	// PKCE Support (optional but recommended)
	CodeChallenge       string `bun:"code_challenge" json:"codeChallenge,omitempty"`              // PKCE code challenge
	CodeChallengeMethod string `bun:"code_challenge_method" json:"codeChallengeMethod,omitempty"` // PKCE challenge method (S256, plain)

	// Rate Limiting
	PollCount    int        `bun:"poll_count,notnull,default:0" json:"pollCount"`
	LastPolledAt *time.Time `bun:"last_polled_at" json:"lastPolledAt,omitempty"`

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	User         *User         `bun:"rel:belongs-to,join:user_id=id"`
	Session      *Session      `bun:"rel:belongs-to,join:session_id=id"`
}

// Device code status constants
const (
	DeviceCodeStatusPending    = "pending"
	DeviceCodeStatusAuthorized = "authorized"
	DeviceCodeStatusDenied     = "denied"
	DeviceCodeStatusExpired    = "expired"
	DeviceCodeStatusConsumed   = "consumed"
)

// IsExpired checks if the device code has expired
func (dc *DeviceCode) IsExpired() bool {
	return time.Now().After(dc.ExpiresAt)
}

// IsPending checks if the device code is awaiting user authorization
func (dc *DeviceCode) IsPending() bool {
	return dc.Status == DeviceCodeStatusPending && !dc.IsExpired()
}

// IsAuthorized checks if the device code has been authorized by the user
func (dc *DeviceCode) IsAuthorized() bool {
	return dc.Status == DeviceCodeStatusAuthorized && !dc.IsExpired()
}

// CanPoll checks if the device can poll for this code
func (dc *DeviceCode) CanPoll() bool {
	if dc.IsExpired() {
		return false
	}
	if dc.Status == DeviceCodeStatusConsumed {
		return false
	}
	return true
}

// ShouldSlowDown checks if the device is polling too frequently
func (dc *DeviceCode) ShouldSlowDown() bool {
	if dc.LastPolledAt == nil {
		return false
	}
	elapsed := time.Since(*dc.LastPolledAt).Seconds()
	return elapsed < float64(dc.Interval)
}

// FormattedUserCode returns the user code in a display-friendly format (e.g., "BCDF-GHJK")
func (dc *DeviceCode) FormattedUserCode() string {
	// Default format is "XXXX-XXXX" (8 characters with hyphen in the middle)
	if len(dc.UserCode) == 8 {
		return dc.UserCode[:4] + "-" + dc.UserCode[4:]
	}
	// For other lengths, just return as-is
	return dc.UserCode
}
