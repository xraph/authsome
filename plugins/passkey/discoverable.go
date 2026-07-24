package passkey

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// ceremonyCookieName correlates a discoverable-login begin with its finish. It
// is HttpOnly and short-lived; the value is an opaque random ceremony id that
// keys the stored WebAuthn session, so concurrent passwordless logins never
// share a single global ceremony slot.
const ceremonyCookieName = "authsome_passkey_ceremony"

// newCeremonyID returns a random, URL-safe ceremony identifier.
func newCeremonyID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("passkey: generate ceremony id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// discoverableKey is the ceremony-store key for a discoverable login ceremony.
func discoverableKey(ceremonyID string) string {
	return "passkey:discoverable:" + ceremonyID
}

// setCeremonyCookie writes the ceremony-correlation cookie. secure is set from
// the request scheme (HTTPS) by the caller.
func setCeremonyCookie(w http.ResponseWriter, ceremonyID string, ttl time.Duration, secure bool) {
	// #nosec G124 -- HttpOnly, SameSite=Lax, and Secure (over HTTPS) are all set below.
	http.SetCookie(w, &http.Cookie{
		Name:     ceremonyCookieName,
		Value:    ceremonyID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

// readCeremonyCookie returns the ceremony id from the request, or "" when the
// request carries no discoverable ceremony (i.e. an identified/step-up login).
func readCeremonyCookie(r *http.Request) string {
	if r == nil {
		return ""
	}
	if c, err := r.Cookie(ceremonyCookieName); err == nil {
		return c.Value
	}
	return ""
}

// clearCeremonyCookie expires the ceremony-correlation cookie.
func clearCeremonyCookie(w http.ResponseWriter, secure bool) {
	// #nosec G124 -- HttpOnly, SameSite=Lax, and Secure (over HTTPS) are all set below.
	http.SetCookie(w, &http.Cookie{
		Name:     ceremonyCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// resolveDiscoverableUser maps a WebAuthn user handle (the authsome user id) to
// the user and its webauthn.User view, WITHOUT requiring a pre-authenticated
// session — this is what makes passwordless login work. Errors on an
// unparseable handle, a missing engine, or an unknown user.
func (p *Plugin) resolveDiscoverableUser(ctx context.Context, userHandle []byte) (u *user.User, waUser webauthn.User, err error) {
	uid, perr := id.ParseUserID(string(userHandle))
	if perr != nil {
		return nil, nil, fmt.Errorf("passkey: invalid user handle: %w", perr)
	}
	if p.engine == nil {
		return nil, nil, fmt.Errorf("passkey: engine unavailable for passwordless login")
	}
	loaded, gerr := p.engine.GetUser(ctx, uid)
	if gerr != nil {
		return nil, nil, gerr
	}
	return loaded, p.toWebAuthnUser(ctx, loaded), nil
}

// finishDiscoverableLogin completes a passwordless WebAuthn ceremony: it loads
// the ceremony session correlated by cookie, verifies the assertion against the
// resolved user (looked up by user handle — no prior authentication), rejects
// cloned authenticators, and issues a real session.
func (p *Plugin) finishDiscoverableLogin(ctx forge.Context, ceremonyID string) (*LoginFinishResponse, error) {
	key := discoverableKey(ceremonyID)
	sessionJSON, err := p.ceremonies.Get(ctx.Context(), key)
	if err != nil {
		return nil, forge.BadRequest("no pending login ceremony")
	}
	_ = p.ceremonies.Delete(ctx.Context(), key) //nolint:errcheck // best-effort cleanup
	clearCeremonyCookie(ctx.Response(), ctx.Request().TLS != nil)

	var waSession webauthn.SessionData
	if unmarshalErr := json.Unmarshal(sessionJSON, &waSession); unmarshalErr != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to parse session: %w", unmarshalErr))
	}

	var loggedIn *user.User
	handler := func(_, userHandle []byte) (webauthn.User, error) {
		u, wau, herr := p.resolveDiscoverableUser(ctx.Context(), userHandle)
		if herr != nil {
			return nil, herr
		}
		loggedIn = u
		return wau, nil
	}

	cred, err := p.waForRequest(ctx.Request()).FinishDiscoverableLogin(handler, waSession, ctx.Request())
	if err != nil {
		return nil, forge.Unauthorized(fmt.Sprintf("passkey: finish discoverable login: %v", err))
	}
	if loggedIn == nil {
		return nil, forge.Unauthorized("passkey: could not resolve user for passwordless login")
	}

	// Clone detection: reject a possibly-duplicated authenticator.
	if cwErr := cloneWarningError(cred); cwErr != nil {
		p.audit(ctx.Context(), hook.ActionPasskeyLogin, hook.ResourcePasskey, "", loggedIn.ID.String(), "", bridge.OutcomeFailure)
		return nil, forge.Unauthorized(cwErr.Error())
	}

	if p.store != nil {
		_ = p.store.UpdateSignCount(ctx.Context(), cred.ID, cred.Authenticator.SignCount) //nolint:errcheck // best-effort update
	}

	sess, err := p.issueSession(ctx, loggedIn)
	if err != nil {
		return nil, err
	}

	userIDStr := loggedIn.ID.String()
	p.audit(ctx.Context(), hook.ActionPasskeyLogin, hook.ResourcePasskey, "", userIDStr, "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.passkey.authenticated", "", map[string]string{"user_id": userIDStr})
	p.emitHook(ctx.Context(), hook.ActionPasskeyLogin, hook.ResourcePasskey, "", userIDStr, "")

	return &LoginFinishResponse{
		UserID:       userIDStr,
		Status:       "authenticated",
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
	}, nil
}

// issueSession mints a real session for a passwordless login via the engine's
// centralized IssueSession (so the MFA gate applies), returning it.
func (p *Plugin) issueSession(ctx forge.Context, u *user.User) (*session.Session, error) {
	eng, ok := p.engine.(*authsome.Engine)
	if !ok || eng == nil {
		return nil, forge.InternalError(fmt.Errorf("passkey: session issuance unavailable"))
	}
	result, err := eng.IssueSession(ctx.Context(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      u.AppID,
		AuthMethod: "passkey",
		IPAddress:  ctx.Request().RemoteAddr,
		UserAgent:  ctx.Request().UserAgent(),
	})
	if err != nil {
		return nil, err
	}
	return result.Session, nil
}
