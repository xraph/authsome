package consent

import (
	"database/sql"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Consent model (shared across SQL stores)
// ──────────────────────────────────────────────────

type consentModel struct {
	grove.BaseModel `grove:"table:authsome_consents,alias:cn"`

	ID        string       `grove:"id,pk"`
	UserID    string       `grove:"user_id,notnull"`
	AppID     string       `grove:"app_id,notnull"`
	Purpose   string       `grove:"purpose,notnull"`
	Granted   bool         `grove:"granted,notnull"`
	Version   string       `grove:"version,notnull"`
	IPAddress string       `grove:"ip_address,notnull"`
	GrantedAt time.Time    `grove:"granted_at,notnull,default:now()"`
	RevokedAt sql.NullTime `grove:"revoked_at"`
	CreatedAt time.Time    `grove:"created_at,notnull,default:now()"`
	UpdatedAt time.Time    `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// Consent converters
// ──────────────────────────────────────────────────

func toConsent(m *consentModel) (*Consent, error) {
	consentID, err := id.ParseConsentID(m.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}

	c := &Consent{
		ID:        consentID,
		UserID:    userID,
		AppID:     appID,
		Purpose:   m.Purpose,
		Granted:   m.Granted,
		Version:   m.Version,
		IPAddress: m.IPAddress,
		GrantedAt: m.GrantedAt,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.RevokedAt.Valid {
		c.RevokedAt = &m.RevokedAt.Time
	}
	return c, nil
}

func fromConsent(c *Consent) *consentModel {
	m := &consentModel{
		ID:        c.ID.String(),
		UserID:    c.UserID.String(),
		AppID:     c.AppID.String(),
		Purpose:   c.Purpose,
		Granted:   c.Granted,
		Version:   c.Version,
		IPAddress: c.IPAddress,
		GrantedAt: c.GrantedAt,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
	if c.RevokedAt != nil {
		m.RevokedAt = sql.NullTime{Time: *c.RevokedAt, Valid: true}
	}
	return m
}
