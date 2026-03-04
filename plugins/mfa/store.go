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
}
