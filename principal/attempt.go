package principal

import (
	"time"

	"github.com/xraph/authsome/id"
)

// AuthAttempt describes a credential being turned into a session.
//
// It is the payload the risk plugins receive for machine callers. Sign-in
// carries an account.SignInRequest, which has an email and a password on it
// and means nothing for an agent. This is the machine-side equivalent: enough
// for a risk contributor to score, with nothing on it that only a human has.
type AuthAttempt struct {
	Subject Ref
	Actors  Chain

	AppID id.AppID
	EnvID id.EnvironmentID
	OrgID id.OrgID

	// CredentialKind is how the caller authenticated: "api_key",
	// "token_exchange" or "jwt".
	CredentialKind string
	// CredentialID identifies the specific credential, so a verdict can be
	// cached against it and a compromised one can be traced.
	CredentialID string

	IPAddress string
	UserAgent string

	// Ephemeral is true when the subject was minted by another principal
	// rather than registered.
	Ephemeral bool

	At time.Time
}
