package mfa

import (
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Enrollment model (shared across SQL stores)
// ──────────────────────────────────────────────────

type enrollmentModel struct {
	grove.BaseModel `grove:"table:authsome_mfa_enrollments,alias:me"`

	ID        string    `grove:"id,pk"`
	UserID    string    `grove:"user_id,notnull"`
	Method    string    `grove:"method,notnull"`
	Secret    string    `grove:"secret,notnull"`
	Verified  bool      `grove:"verified,notnull"`
	CreatedAt time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// Enrollment converters
// ──────────────────────────────────────────────────

func toEnrollment(m *enrollmentModel) (*Enrollment, error) {
	mfaID, err := id.ParseMFAID(m.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return &Enrollment{
		ID:        mfaID,
		UserID:    userID,
		Method:    m.Method,
		Secret:    m.Secret,
		Verified:  m.Verified,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func fromEnrollment(e *Enrollment) *enrollmentModel {
	return &enrollmentModel{
		ID:        e.ID.String(),
		UserID:    e.UserID.String(),
		Method:    e.Method,
		Secret:    e.Secret,
		Verified:  e.Verified,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Recovery code model (shared across SQL stores)
// ──────────────────────────────────────────────────

type recoveryCodeModel struct {
	grove.BaseModel `grove:"table:authsome_mfa_recovery_codes,alias:rc"`

	ID        string     `grove:"id,pk"`
	UserID    string     `grove:"user_id,notnull"`
	CodeHash  string     `grove:"code_hash,notnull"`
	Used      bool       `grove:"used,notnull"`
	UsedAt    *time.Time `grove:"used_at"`
	CreatedAt time.Time  `grove:"created_at,notnull,default:now()"`
}

func toRecoveryCode(m *recoveryCodeModel) (*RecoveryCode, error) {
	rcID, err := id.ParseRecoveryCodeID(m.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return &RecoveryCode{
		ID:        rcID,
		UserID:    userID,
		CodeHash:  m.CodeHash,
		Used:      m.Used,
		UsedAt:    m.UsedAt,
		CreatedAt: m.CreatedAt,
	}, nil
}
