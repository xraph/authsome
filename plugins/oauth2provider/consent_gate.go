package oauth2provider

import (
	"context"

	"github.com/xraph/authsome/id"
)

// ConsentGate is an optional hook another plugin implements to veto an
// authorization before a code is issued. agentauth registers itself as the
// gate so an org's policy on delegated agents is enforced at the moment a
// user consents, which is the only point where "may this agent touch our
// data" is a well-formed question.
//
// A nil gate allows everything, so the provider's behavior is unchanged when
// no plugin registers one.
type ConsentGate interface {
	// Evaluate returns a non-nil error to refuse the authorization. The error
	// is surfaced to the caller, so it should be a forge HTTP error.
	//
	// orgID is the caller's active organization, taken from the session that
	// is granting consent. It is the zero value (id.OrgID{}) whenever that
	// session is not org-scoped — that is not an assertion that no policy
	// applies. An implementation must not treat a zero orgID as permission to
	// skip its policy check; do the same evaluation you would for any org.
	//
	// appID is the app the OAuth2 client (clientID) belongs to, resolved by
	// the caller from its own client/device-code record rather than trusted
	// from the request. A gate that resolves clientID to some other record
	// globally (not scoped to an app) must bind its decision to appID, or a
	// client_id collision across two apps lets one app's registration govern
	// the other's client.
	Evaluate(ctx context.Context, clientID string, userID id.UserID, orgID id.OrgID, appID id.AppID, scopes []string) error
}

// SetConsentGate registers a gate consulted before every authorization code is
// issued. Use it when the gate is only available after the provider has been
// constructed; otherwise set Config.ConsentGate.
func (p *Plugin) SetConsentGate(g ConsentGate) { p.consentGate = g }

// EvaluateConsent runs the registered gate, if any.
func (p *Plugin) EvaluateConsent(ctx context.Context, clientID string, userID id.UserID, orgID id.OrgID, appID id.AppID, scopes []string) error {
	if p.consentGate == nil {
		return nil
	}
	return p.consentGate.Evaluate(ctx, clientID, userID, orgID, appID, scopes)
}
