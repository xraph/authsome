package passkey

import (
	"context"
	"errors"

	"github.com/xraph/authsome/id"
)

// ErrCredentialNotFound is returned when a credential is not found.
var ErrCredentialNotFound = errors.New("passkey: credential not found")

// Store persists passkey/WebAuthn credential data.
type Store interface {
	CreateCredential(ctx context.Context, c *Credential) error
	GetCredential(ctx context.Context, credentialID []byte) (*Credential, error)
	ListUserCredentials(ctx context.Context, userID id.UserID) ([]*Credential, error)
	DeleteCredential(ctx context.Context, credentialID []byte) error
	UpdateSignCount(ctx context.Context, credentialID []byte, count uint32) error
}

// FlagsStore is implemented by a Store that keeps a credential's WebAuthn
// flags. The built-in stores all do. A Store without it still works: its
// passkeys' flags are taken from each sign-in's own assertion.
type FlagsStore interface {
	UpdateCredentialFlags(ctx context.Context, credentialID []byte, flags uint8) error
}
