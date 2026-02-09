package passkey

import "time"

// Config contains WebAuthn/FIDO2 configuration.
type Config struct {
	RPID                    string        `json:"rpid"                    yaml:"rpid"`
	RPName                  string        `json:"rpname"                  yaml:"rpname"`
	RPOrigins               []string      `json:"rporigins"               yaml:"rporigins"`
	Timeout                 time.Duration `json:"timeout"                 yaml:"timeout"`                 // milliseconds
	UserVerification        string        `json:"userverification"        yaml:"userverification"`        // required, preferred, discouraged
	AttestationType         string        `json:"attestationtype"         yaml:"attestationtype"`         // none, indirect, direct
	RequireResidentKey      bool          `json:"requireresidentkey"      yaml:"requireresidentkey"`      // require resident keys
	AuthenticatorAttachment string        `json:"authenticatorattachment" yaml:"authenticatorattachment"` // platform, cross-platform, or empty
	ChallengeStorage        string        `json:"challengestorage"        yaml:"challengestorage"`        // memory or redis (future)
}
