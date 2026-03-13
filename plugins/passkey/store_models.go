package passkey

import (
	"strings"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Credential model (shared across SQL stores)
// ──────────────────────────────────────────────────

type credentialModel struct {
	grove.BaseModel `grove:"table:authsome_passkey_credentials,alias:pc"`

	ID              string    `grove:"id,pk"`
	UserID          string    `grove:"user_id,notnull"`
	AppID           string    `grove:"app_id,notnull"`
	CredentialID    []byte    `grove:"credential_id,notnull"`
	PublicKey       []byte    `grove:"public_key,notnull"`
	AttestationType string    `grove:"attestation_type,notnull"`
	Transport       string    `grove:"transport,notnull"` // comma-separated
	SignCount       int       `grove:"sign_count,notnull"`
	AAGUID          []byte    `grove:"aaguid"`
	DisplayName     string    `grove:"display_name,notnull"`
	CreatedAt       time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt       time.Time `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// Credential converters
// ──────────────────────────────────────────────────

func credentialFromModel(m *credentialModel) (*Credential, error) {
	pkID, err := id.ParsePasskeyID(m.ID)
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

	var transport []string
	if m.Transport != "" {
		transport = strings.Split(m.Transport, ",")
	}

	return &Credential{
		ID:              pkID,
		UserID:          userID,
		AppID:           appID,
		CredentialID:    m.CredentialID,
		PublicKey:       m.PublicKey,
		AttestationType: m.AttestationType,
		Transport:       transport,
		SignCount:       uint32(m.SignCount),
		AAGUID:          m.AAGUID,
		DisplayName:     m.DisplayName,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}, nil
}

func credentialToModel(c *Credential) *credentialModel {
	return &credentialModel{
		ID:              c.ID.String(),
		UserID:          c.UserID.String(),
		AppID:           c.AppID.String(),
		CredentialID:    c.CredentialID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Transport:       strings.Join(c.Transport, ","),
		SignCount:       int(c.SignCount),
		AAGUID:          c.AAGUID,
		DisplayName:     c.DisplayName,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}
