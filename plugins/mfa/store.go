package mfa

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
)

// Enrollment represents a user's MFA enrollment for a specific method.
type Enrollment struct {
	ID        id.MFAID  `json:"id"`
	UserID    id.UserID `json:"user_id"`
	Method    string    `json:"method"`   // "totp"
	Secret    string    `json:"-"`        // Base32-encoded TOTP secret (hidden from JSON)
	Verified  bool      `json:"verified"` // Set true after first successful verification
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TrustedDevice is a browser or app that passed a second factor and was
// trusted to skip it on later sign-ins until ExpiresAt.
//
// The device is identified by an opaque random token it holds (a cookie or a
// header), never by its user agent: only the token's SHA-256 is stored, so a
// database read cannot be replayed as a trusted device.
type TrustedDevice struct {
	ID         id.TrustedDeviceID `json:"id"`
	UserID     id.UserID          `json:"user_id"`
	TokenHash  string             `json:"-"`
	Label      string             `json:"label"`
	UserAgent  string             `json:"user_agent"`
	IPAddress  string             `json:"ip_address"`
	CreatedAt  time.Time          `json:"created_at"`
	LastUsedAt time.Time          `json:"last_used_at"`
	ExpiresAt  time.Time          `json:"expires_at"`
}

// Store persists MFA enrollment data and recovery codes.
type Store interface {
	// Enrollment CRUD
	CreateEnrollment(ctx context.Context, e *Enrollment) error
	GetEnrollment(ctx context.Context, userID id.UserID, method string) (*Enrollment, error)
	GetEnrollmentByID(ctx context.Context, mfaID id.MFAID) (*Enrollment, error)
	UpdateEnrollment(ctx context.Context, e *Enrollment) error
	DeleteEnrollment(ctx context.Context, mfaID id.MFAID) error
	ListEnrollments(ctx context.Context, userID id.UserID) ([]*Enrollment, error)

	// Recovery codes
	CreateRecoveryCodes(ctx context.Context, codes []*RecoveryCode) error
	GetRecoveryCodes(ctx context.Context, userID id.UserID) ([]*RecoveryCode, error)
	ConsumeRecoveryCode(ctx context.Context, codeID id.RecoveryCodeID) error
	DeleteRecoveryCodes(ctx context.Context, userID id.UserID) error

	// Trusted devices. GetTrustedDeviceByTokenHash returns
	// ErrTrustedDeviceNotFound when no row has that hash; it does not check
	// expiry, which is the caller's to judge. DeleteTrustedDevice is scoped
	// to the user, so one user cannot remove another's device by id.
	CreateTrustedDevice(ctx context.Context, d *TrustedDevice) error
	GetTrustedDeviceByTokenHash(ctx context.Context, tokenHash string) (*TrustedDevice, error)
	ListTrustedDevices(ctx context.Context, userID id.UserID) ([]*TrustedDevice, error)
	TouchTrustedDevice(ctx context.Context, deviceID id.TrustedDeviceID, at time.Time) error
	DeleteTrustedDevice(ctx context.Context, userID id.UserID, deviceID id.TrustedDeviceID) error
	DeleteTrustedDevices(ctx context.Context, userID id.UserID) error
}
