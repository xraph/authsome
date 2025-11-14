package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// TwoFASecret stores per-user 2FA secret data
type TwoFASecret struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:twofa_secrets,alias:tfs"`

	ID      xid.ID `bun:"id,pk,type:varchar(20)"`
	UserID  xid.ID `bun:"user_id,notnull,type:varchar(20)"`
	Method  string `bun:"method,notnull"` // totp or otp
	Secret  string `bun:"secret"`
	Enabled bool   `bun:"enabled,notnull,default:false"`
}

// BackupCode stores recovery codes for 2FA
type BackupCode struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:twofa_backup_codes,alias:tbc"`

	ID       xid.ID     `bun:"id,pk,type:varchar(20)"`
	UserID   xid.ID     `bun:"user_id,notnull,type:varchar(20)"`
	CodeHash string     `bun:"code_hash,notnull"`
	UsedAt   *time.Time `bun:"used_at"`
}

// TrustedDevice allows skipping 2FA for a period
type TrustedDevice struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:trusted_devices,alias:td"`

	ID        xid.ID    `bun:"id,pk,type:varchar(20)"`
	UserID    xid.ID    `bun:"user_id,notnull,type:varchar(20)"`
	DeviceID  string    `bun:"device_id,notnull"`
	ExpiresAt time.Time `bun:"expires_at,notnull"`
}

// OTPCode stores one-time codes for OTP-based 2FA
type OTPCode struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:twofa_otpcodes,alias:toc"`

	ID        xid.ID    `bun:"id,pk,type:varchar(20)"`
	UserID    xid.ID    `bun:"user_id,notnull,type:varchar(20)"`
	CodeHash  string    `bun:"code_hash,notnull"`
	ExpiresAt time.Time `bun:"expires_at,notnull"`
	Attempts  int       `bun:"attempts,notnull,default:0"`
}
