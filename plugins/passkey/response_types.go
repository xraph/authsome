package passkey

import "time"

// Response types for passkey plugin

// BeginRegisterResponse contains WebAuthn registration options
type BeginRegisterResponse struct {
	Options   interface{}   `json:"options"`   // WebAuthn PublicKeyCredentialCreationOptions
	Challenge string        `json:"challenge"` // Base64URL encoded challenge
	UserID    string        `json:"userId"`
	Timeout   time.Duration `json:"timeout"` // Timeout in milliseconds
}

// FinishRegisterResponse contains registered passkey information
type FinishRegisterResponse struct {
	PasskeyID    string    `json:"passkeyId"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	CredentialID string    `json:"credentialId"`
}

// BeginLoginResponse contains WebAuthn authentication options
type BeginLoginResponse struct {
	Options   interface{}   `json:"options"`   // WebAuthn PublicKeyCredentialRequestOptions
	Challenge string        `json:"challenge"` // Base64URL encoded challenge
	Timeout   time.Duration `json:"timeout"`   // Timeout in milliseconds
}

// LoginResponse contains authentication result with session
type LoginResponse struct {
	User        interface{} `json:"user"`
	Session     interface{} `json:"session"`
	Token       string      `json:"token"`
	PasskeyUsed string      `json:"passkeyUsed"` // ID of the passkey that was used
}

// PasskeyInfo represents detailed passkey information
type PasskeyInfo struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	CredentialID      string     `json:"credentialId"`
	AAGUID            string     `json:"aaguid,omitempty"`
	AuthenticatorType string     `json:"authenticatorType"` // "platform" or "cross-platform"
	CreatedAt         time.Time  `json:"createdAt"`
	LastUsedAt        *time.Time `json:"lastUsedAt,omitempty"`
	SignCount         uint32     `json:"signCount"`
	IsResidentKey     bool       `json:"isResidentKey"`
}

// ListPasskeysResponse contains list of user passkeys
type ListPasskeysResponse struct {
	Passkeys []PasskeyInfo `json:"passkeys"`
	Count    int           `json:"count"`
}

// UpdatePasskeyResponse contains updated passkey information
type UpdatePasskeyResponse struct {
	PasskeyID string    `json:"passkeyId"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updatedAt"`
}
