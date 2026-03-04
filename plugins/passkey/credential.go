package passkey

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Credential represents a WebAuthn/passkey credential stored for a user.
type Credential struct {
	ID              id.PasskeyID `json:"id"`
	UserID          id.UserID    `json:"user_id"`
	AppID           id.AppID     `json:"app_id"`
	CredentialID    []byte       `json:"credential_id"`    // Raw credential ID from WebAuthn
	PublicKey       []byte       `json:"public_key"`       // COSE-encoded public key
	AttestationType string       `json:"attestation_type"` // "none", "packed", etc.
	Transport       []string     `json:"transport"`        // "usb", "ble", "nfc", "internal"
	SignCount       uint32       `json:"sign_count"`
	AAGUID          []byte       `json:"aaguid"`
	DisplayName     string       `json:"display_name"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}
