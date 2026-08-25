package oauth2provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/session"
)

// tokenExchangeGrantType is the IANA grant type for RFC 8693.
const tokenExchangeGrantType = "urn:ietf:params:oauth:grant-type:token-exchange"

// Supported token type URNs. Both access-token forms resolve to a session,
// because in this codebase an OAuth access token is a session row.
const (
	tokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"
	tokenTypeSession     = "urn:x-authsome:params:oauth:token-type:session"
)

// Denial reasons recorded on a refused exchange. Closed set: scope_escalation
// and cross_app are attack signatures rather than user error, and a fixed
// vocabulary is what makes them alertable instead of greppable.
const (
	denyNoGrant          = "no_grant"
	denyScopeEscalation  = "scope_escalation"
	denyCrossApp         = "cross_app"
	denyInvalidSubject   = "invalid_subject"
	denyUnsupportedToken = "unsupported_token_type"
	denyNoPrincipal      = "client_has_no_principal"
	denyBindingDowngrade = "binding_downgrade"
	denyUnknownPrincipal = "principal_not_found"
)

// errUnsupportedTokenType is a sentinel so the denial reason and the HTTP body
// cannot drift apart. Matching on the message string would couple them by
// accident.
var errUnsupportedTokenType = errors.New("unsupported_token_type")

// tokenExchanger is the slice of the engine this grant needs.
//
// Declared here rather than on plugin.Engine because ExchangeRequest lives in
// the root authsome package, which already imports plugin: naming the type on
// that interface would close an import cycle. A plugin may import the root
// package (the root imports no plugin packages), so the assertion happens
// here instead and costs the rest of the tree nothing.
type tokenExchanger interface {
	ExchangeToken(ctx context.Context, req *authsome.ExchangeRequest) (*session.Session, error)
}

// resolveExchangeToken resolves a subject or actor token to its session.
func (p *Plugin) resolveExchangeToken(token, tokenType string) (*session.Session, error) {
	if tokenType != tokenTypeAccessToken && tokenType != tokenTypeSession {
		return nil, errUnsupportedTokenType
	}
	if p.engine == nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: no engine"))
	}
	sess, err := p.engine.ResolveSessionByToken(token)
	if err != nil || sess == nil {
		return nil, forge.BadRequest("invalid_grant")
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, forge.BadRequest("invalid_grant")
	}
	return sess, nil
}

// narrowRequestedScopes applies the two bounds this layer owns: the subject
// token's own scopes, and the client's registered scopes.
//
// The delegation grant and the actor principal's own scopes are the other two
// bounds, and they belong to Engine.ExchangeToken, which applies them in
// intersectScopes. Checking the two cheap local bounds first means an
// obviously bad request never reaches a grant lookup.
//
// A subject holding no scopes imposes no subject-side ceiling. A password
// sign-in produces exactly that, and reading it as "nothing" would make the
// session-downgrade case impossible on the first hop. Reading it as
// "everything" is safe only because the client bound and the two engine-side
// bounds all still apply.
func narrowRequestedScopes(requested, clientScopes, subjectScopes []string) ([]string, error) {
	// Required, unlike everywhere else. RFC 8693 makes scope optional and lets
	// the server choose, but the point of an exchange is asking for less than
	// you hold, and an omitted scope asks the server to guess.
	if len(requested) == 0 {
		return nil, fmt.Errorf("scope is required for token exchange")
	}

	inClient := toScopeSet(clientScopes)
	inSubject := toScopeSet(subjectScopes)
	subjectBounds := len(subjectScopes) > 0

	out := make([]string, 0, len(requested))
	for _, s := range requested {
		if _, ok := inClient[s]; !ok {
			return nil, fmt.Errorf("scope %q is not registered for this client", s)
		}
		if subjectBounds {
			if _, ok := inSubject[s]; !ok {
				return nil, fmt.Errorf("scope %q is not held by the subject token", s)
			}
		}
		out = append(out, s)
	}
	return out, nil
}

func toScopeSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, s := range in {
		out[s] = struct{}{}
	}
	return out
}

// actorRefForClient resolves the client's linked principal to its real ref.
//
// The kind cannot be guessed. FindActiveDelegation compares the whole
// principal.Ref by value, so a grant written against kind "agent" never
// matches a probe built with kind "workload" even when the id is right. The
// store ignores the kind when looking up a non-user principal, so a probe
// resolves, and the resolved principal carries the true kind back.
func (p *Plugin) actorRefForClient(ctx context.Context, client *OAuth2Client) (principal.Ref, error) {
	probe := principal.Ref{Kind: principal.KindWorkload, ID: client.PrincipalID.String()}
	pr, err := p.engine.ResolvePrincipal(ctx, probe)
	if err != nil || pr == nil {
		return principal.Ref{}, fmt.Errorf("oauth2: resolve client principal: %w", err)
	}
	return pr.Ref, nil
}

func (p *Plugin) handleTokenExchangeGrant(ctx forge.Context, req *TokenRequest) (*TokenResponse, error) {
	// 1. Client authentication. Confidential only: a public client cannot keep
	// a secret and this is a privilege operation.
	if req.ClientID == "" || req.ClientSecret == "" {
		return nil, forge.BadRequest("client_id and client_secret required")
	}
	client, err := p.oauth2Store.GetClient(ctx.Context(), req.ClientID)
	if err != nil {
		return nil, forge.Unauthorized("invalid client")
	}
	if client.Public {
		return nil, forge.BadRequest("token exchange not allowed for public clients")
	}
	if cmpErr := bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(req.ClientSecret)); cmpErr != nil {
		return nil, forge.Unauthorized("invalid client_secret")
	}
	if !p.clientSupportsGrant(client, tokenExchangeGrantType) {
		return nil, forge.BadRequest("unauthorized_client")
	}

	// 2. The client needs a principal to act as.
	if client.PrincipalID.IsNil() {
		e := forge.BadRequest("this client has no linked principal and cannot exchange tokens")
		p.recordExchange(ctx, client, nil, nil, nil, "", denyNoPrincipal, e)
		return nil, e
	}

	// 3. Resolve the subject.
	if req.SubjectToken == "" || req.SubjectTokenType == "" {
		return nil, forge.BadRequest("subject_token and subject_token_type required")
	}
	if req.RequestedTokenType != "" && req.RequestedTokenType != tokenTypeAccessToken {
		return nil, forge.BadRequest("unsupported requested_token_type")
	}
	subject, err := p.resolveExchangeToken(req.SubjectToken, req.SubjectTokenType)
	if err != nil {
		reason := denyInvalidSubject
		if errors.Is(err, errUnsupportedTokenType) {
			reason = denyUnsupportedToken
			err = forge.BadRequest("unsupported_token_type")
		}
		p.recordExchange(ctx, client, nil, nil, nil, "", reason, err)
		return nil, err
	}

	// 4. Cross-app exchange is refused. Without this a client in one app
	// launders a session out of another.
	if subject.AppID != client.AppID {
		e := forge.BadRequest("invalid_grant")
		p.recordExchange(ctx, client, subject, nil, nil, "", denyCrossApp, e)
		return nil, e
	}

	// 5. Never trade a bound token for an unbound one. Engine.ExchangeToken
	// collects no DPoP proof, so what it mints is always unbound; exchanging a
	// bound subject would therefore launder the binding away and hand back a
	// plain bearer token carrying the same authority. That is the same
	// downgrade the refresh path already refuses.
	if subject.DPoPJKT != "" {
		e := forge.BadRequest("invalid_grant: a DPoP-bound token cannot be exchanged for an unbound one")
		p.recordExchange(ctx, client, subject, nil, nil, "", denyBindingDowngrade, e)
		return nil, e
	}

	// 6. Resolve the acting principal. An actor_token, when present, replaces
	// the client as the acting party; its identity is proven by resolving it
	// rather than asserted in a parameter, which is what lets the delegation
	// grant be the only authority that matters.
	actor, err := p.actorRefForClient(ctx.Context(), client)
	if err != nil {
		e := forge.BadRequest("invalid_grant")
		p.recordExchange(ctx, client, subject, nil, nil, "", denyUnknownPrincipal, e)
		return nil, e
	}
	if req.ActorToken != "" {
		actorSess, aErr := p.resolveExchangeToken(req.ActorToken, req.ActorTokenType)
		if aErr != nil {
			reason := denyInvalidSubject
			if errors.Is(aErr, errUnsupportedTokenType) {
				reason = denyUnsupportedToken
				aErr = forge.BadRequest("unsupported_token_type")
			}
			p.recordExchange(ctx, client, subject, nil, nil, "", reason, aErr)
			return nil, aErr
		}
		if actorSess.AppID != client.AppID {
			e := forge.BadRequest("invalid_grant")
			p.recordExchange(ctx, client, subject, nil, nil, "", denyCrossApp, e)
			return nil, e
		}
		actor = actorSess.Subject()
	}

	// 7. The two bounds this layer owns.
	requested := strings.Fields(req.Scope)
	narrowed, err := narrowRequestedScopes(requested, client.Scopes, subject.Scopes)
	if err != nil {
		e := forge.BadRequest(fmt.Sprintf("invalid_scope: %s", err.Error()))
		p.recordExchange(ctx, client, subject, requested, nil, "", denyScopeEscalation, e)
		return nil, e
	}

	// RFC 8707: the same allowlist every other grant applies.
	if _, resErr := resolveResources(client, req.Resource); resErr != nil {
		return nil, resErr
	}

	exchanger, ok := p.engine.(tokenExchanger)
	if !ok {
		return nil, forge.InternalError(fmt.Errorf("oauth2: engine does not support token exchange"))
	}

	// 8. The engine owns the grant lookup, the grant's own scope filter, the
	// actor principal's scopes, the TTL clamp and the actor chain.
	//
	// CallerActors and CallerDelegationID carry the subject session's own
	// chain so the engine's chained-exchange guard actually fires. A session
	// that was itself minted by an exchange must not be re-presented here:
	// that is the escalation where an agent trades its way into a user
	// session and then trades that for a third party's, with the agent gone
	// from the chain. Leaving these empty would silently disarm that check.
	issued, err := exchanger.ExchangeToken(ctx.Context(), &authsome.ExchangeRequest{
		AppID:              client.AppID,
		Actor:              actor,
		RequestedSubject:   subject.Subject(),
		Scopes:             narrowed,
		IPAddress:          ctx.Request().RemoteAddr,
		UserAgent:          ctx.Request().UserAgent(),
		CredentialID:       client.ClientID,
		CallerActors:       subject.Actors,
		CallerDelegationID: subject.DelegationID,
	})
	if err != nil {
		e := forge.BadRequest("invalid_grant")
		p.recordExchange(ctx, client, subject, requested, nil, "", denyNoGrant, e)
		return nil, e
	}

	p.recordExchange(ctx, client, subject, requested, issued.Scopes, issued.ID.String(), "", nil)

	return &TokenResponse{
		AccessToken:     issued.Token,
		IssuedTokenType: tokenTypeAccessToken,
		TokenType:       "Bearer",
		ExpiresIn:       int(time.Until(issued.ExpiresAt).Seconds()),
		Scope:           strings.Join(issued.Scopes, " "),
	}, nil
}

// recordExchange writes one security event per exchange attempt.
//
// Written straight to the store rather than emitted on the hook bus: the bus
// bridge builds its event from Action, Outcome, Metadata and CreatedAt only
// and never sets AppID, and securityevent.Query filters on AppID, so anything
// recorded that way is written and then unreadable.
func (p *Plugin) recordExchange(
	ctx forge.Context,
	client *OAuth2Client,
	subject *session.Session,
	requested []string,
	granted []string,
	issuedSessionID string,
	denialReason string,
	cause error,
) {
	if p.engine == nil {
		return
	}
	events := p.engine.SecurityEvents()
	if events == nil {
		return
	}

	outcome := "success"
	if denialReason != "" || cause != nil {
		outcome = "failure"
	}

	meta := map[string]string{"client_id": client.ClientID}
	if !client.PrincipalID.IsNil() {
		meta["actor_principal_id"] = client.PrincipalID.String()
	}
	if len(requested) > 0 {
		meta["requested_scopes"] = strings.Join(requested, " ")
	}
	if len(granted) > 0 {
		meta["granted_scopes"] = strings.Join(granted, " ")
	}
	if issuedSessionID != "" {
		meta["issued_session_id"] = issuedSessionID
	}
	if denialReason != "" {
		meta["denial_reason"] = denialReason
	}

	var userID id.UserID
	if subject != nil {
		userID = subject.UserID
		ref := subject.Subject()
		meta["subject_session_id"] = subject.ID.String()
		meta["subject_kind"] = string(ref.Kind)
		meta["subject_principal_id"] = ref.ID
		meta["chain_depth"] = fmt.Sprintf("%d", len(subject.Actors)+1)
	}

	//nolint:errcheck // audit is best-effort and must never fail the exchange
	_ = events.RecordSecurityEvent(ctx.Context(), &securityevent.Event{
		AppID:     client.AppID,
		UserID:    userID,
		Action:    "oauth2.token_exchange",
		Outcome:   outcome,
		Metadata:  meta,
		IPAddress: ctx.Request().RemoteAddr,
		UserAgent: ctx.Request().UserAgent(),
		CreatedAt: time.Now(),
	})
}
