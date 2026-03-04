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
