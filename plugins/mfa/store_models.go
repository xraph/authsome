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

// ──────────────────────────────────────────────────
// Trusted device model (shared across SQL stores)
// ──────────────────────────────────────────────────

type trustedDeviceModel struct {
	grove.BaseModel `grove:"table:authsome_mfa_trusted_devices,alias:td"`

	ID         string    `grove:"id,pk"`
	UserID     string    `grove:"user_id,notnull"`
	TokenHash  string    `grove:"token_hash,notnull"`
	Label      string    `grove:"label,notnull"`
	UserAgent  string    `grove:"user_agent,notnull"`
	IPAddress  string    `grove:"ip_address,notnull"`
	CreatedAt  time.Time `grove:"created_at,notnull"`
	LastUsedAt time.Time `grove:"last_used_at,notnull"`
	ExpiresAt  time.Time `grove:"expires_at,notnull"`
}

func toTrustedDevice(m *trustedDeviceModel) (*TrustedDevice, error) {
	deviceID, err := id.ParseTrustedDeviceID(m.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return &TrustedDevice{
		ID:         deviceID,
		UserID:     userID,
		TokenHash:  m.TokenHash,
		Label:      m.Label,
		UserAgent:  m.UserAgent,
		IPAddress:  m.IPAddress,
		CreatedAt:  m.CreatedAt,
		LastUsedAt: m.LastUsedAt,
		ExpiresAt:  m.ExpiresAt,
	}, nil
}

func fromTrustedDevice(d *TrustedDevice) *trustedDeviceModel {
	return &trustedDeviceModel{
		ID:         d.ID.String(),
		UserID:     d.UserID.String(),
		TokenHash:  d.TokenHash,
		Label:      d.Label,
		UserAgent:  d.UserAgent,
		IPAddress:  d.IPAddress,
		CreatedAt:  d.CreatedAt,
		LastUsedAt: d.LastUsedAt,
		ExpiresAt:  d.ExpiresAt,
	}
}

func toTrustedDevices(models []trustedDeviceModel) ([]*TrustedDevice, error) {
	result := make([]*TrustedDevice, 0, len(models))
	for i := range models {
		d, err := toTrustedDevice(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}
