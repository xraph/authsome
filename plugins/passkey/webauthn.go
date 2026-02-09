package passkey

import (
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnWrapper wraps the go-webauthn/webauthn library for easier use.
type WebAuthnWrapper struct {
	webAuthn *webauthn.WebAuthn
	config   Config
}

// NewWebAuthnWrapper creates a new WebAuthn wrapper with configuration.
func NewWebAuthnWrapper(cfg Config) (*WebAuthnWrapper, error) {
	// Convert origins string slice if needed
	origins := cfg.RPOrigins
	if len(origins) == 0 {
		// Default to RPID if no origins specified
		origins = []string{"https://" + cfg.RPID}
	}

	wconfig := &webauthn.Config{
		RPDisplayName: cfg.RPName,
		RPID:          cfg.RPID,
		RPOrigins:     origins,
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    cfg.Timeout, // milliseconds
				TimeoutUVD: cfg.Timeout,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    cfg.Timeout,
				TimeoutUVD: cfg.Timeout,
			},
		},
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn instance: %w", err)
	}

	return &WebAuthnWrapper{
		webAuthn: w,
		config:   cfg,
	}, nil
}

// BeginRegistration initiates WebAuthn credential registration.
func (w *WebAuthnWrapper) BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	return w.webAuthn.BeginRegistration(user, opts...)
}

// FinishRegistration completes WebAuthn credential registration.
func (w *WebAuthnWrapper) FinishRegistration(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	return w.webAuthn.CreateCredential(user, session, response)
}

// BeginLogin initiates WebAuthn authentication.
func (w *WebAuthnWrapper) BeginLogin(user webauthn.User, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return w.webAuthn.BeginLogin(user, opts...)
}

// FinishLogin completes WebAuthn authentication with credential verification.
func (w *WebAuthnWrapper) FinishLogin(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
	return w.webAuthn.ValidateLogin(user, session, response)
}

// BeginDiscoverableLogin initiates authentication for discoverable credentials (usernameless).
func (w *WebAuthnWrapper) BeginDiscoverableLogin(opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return w.webAuthn.BeginDiscoverableLogin(opts...)
}

// ParseUserVerificationRequirement converts string to protocol type.
func ParseUserVerificationRequirement(s string) protocol.UserVerificationRequirement {
	switch s {
	case "required":
		return protocol.VerificationRequired
	case "preferred":
		return protocol.VerificationPreferred
	case "discouraged":
		return protocol.VerificationDiscouraged
	default:
		return protocol.VerificationPreferred
	}
}

// ParseAuthenticatorAttachment converts string to protocol type.
func ParseAuthenticatorAttachment(s string) protocol.AuthenticatorAttachment {
	switch s {
	case "platform":
		return protocol.Platform
	case "cross-platform":
		return protocol.CrossPlatform
	default:
		return ""
	}
}

// ParseResidentKeyRequirement converts bool to protocol type.
func ParseResidentKeyRequirement(required bool) protocol.ResidentKeyRequirement {
	if required {
		return protocol.ResidentKeyRequirementRequired
	}

	return protocol.ResidentKeyRequirementDiscouraged
}

// ParseConveyancePreference converts string to protocol type.
func ParseConveyancePreference(s string) protocol.ConveyancePreference {
	switch s {
	case "none":
		return protocol.PreferNoAttestation
	case "indirect":
		return protocol.PreferIndirectAttestation
	case "direct":
		return protocol.PreferDirectAttestation
	default:
		return protocol.PreferNoAttestation
	}
}
