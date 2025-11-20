package passkey

import (
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// UserAdapter adapts AuthSome User to WebAuthn User interface
type UserAdapter struct {
	userID      xid.ID
	userName    string
	displayName string
	credentials []webauthn.Credential
}

// NewUserAdapter creates a new user adapter for WebAuthn
func NewUserAdapter(userID xid.ID, userName, displayName string, passkeys []schema.Passkey) *UserAdapter {
	adapter := &UserAdapter{
		userID:      userID,
		userName:    userName,
		displayName: displayName,
		credentials: make([]webauthn.Credential, 0, len(passkeys)),
	}

	// Convert passkeys to WebAuthn credentials
	for _, pk := range passkeys {
		adapter.credentials = append(adapter.credentials, webauthn.Credential{
			ID:              []byte(pk.CredentialID),
			PublicKey:       pk.PublicKey,
			AttestationType: "none", // Could be stored if needed
			Authenticator: webauthn.Authenticator{
				AAGUID:    pk.AAGUID,
				SignCount: pk.SignCount,
			},
		})
	}

	return adapter
}

// WebAuthnID returns the user's ID as bytes
func (u *UserAdapter) WebAuthnID() []byte {
	return []byte(u.userID.String())
}

// WebAuthnName returns the user's username
func (u *UserAdapter) WebAuthnName() string {
	return u.userName
}

// WebAuthnDisplayName returns the user's display name
func (u *UserAdapter) WebAuthnDisplayName() string {
	if u.displayName != "" {
		return u.displayName
	}
	return u.userName
}

// WebAuthnCredentials returns the user's credentials
func (u *UserAdapter) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// WebAuthnIcon returns the user's icon URL (optional)
func (u *UserAdapter) WebAuthnIcon() string {
	return ""
}

// AddCredential adds a new credential to the user adapter
func (u *UserAdapter) AddCredential(cred webauthn.Credential) {
	u.credentials = append(u.credentials, cred)
}

// UpdateCredential updates an existing credential's sign count
func (u *UserAdapter) UpdateCredential(credentialID []byte, signCount uint32) {
	for i, cred := range u.credentials {
		if string(cred.ID) == string(credentialID) {
			u.credentials[i].Authenticator.SignCount = signCount
			break
		}
	}
}

