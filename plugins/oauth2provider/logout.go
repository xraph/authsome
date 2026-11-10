package oauth2provider

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/middleware"
)

// RP-initiated logout (OpenID Connect RP-Initiated Logout 1.0). A client
// sends the browser to /v1/oauth/logout to end the user's session here,
// then gets it back at a post_logout_redirect_uri it registered.
//
// The session ends straight away only when the request carries an
// id_token_hint we issued for the signed-in user. Anything less (no hint,
// a hint for someone else, a session in another app) gets a confirmation
// page, so a link on some other site can't sign people out.

const (
	// maxIDTokenHintBytes bounds the hint before it is parsed.
	maxIDTokenHintBytes = 16 << 10
	// logoutTTL is how long a confirmation page stays answerable.
	logoutTTL = 10 * time.Minute
)

// pendingLogout is a parked confirmation.
type pendingLogout struct {
	SessionID  string `json:"session_id"`
	UserID     string `json:"user_id"`
	AppID      string `json:"app_id"`
	Target     string `json:"target,omitempty"`
	ClientName string `json:"client_name,omitempty"`
	CSRF       string `json:"csrf"`
}

func logoutKey(lid string) string { return "oauth2:logout:" + lid }

// handleLogout is the end_session_endpoint. It takes its parameters from
// the query on GET and from the form body on POST, as the spec allows.
func (p *Plugin) handleLogout(ctx forge.Context) error {
	r := ctx.Request()
	if err := r.ParseForm(); err != nil {
		return forge.BadRequest("invalid logout request")
	}
	hintRaw := r.Form.Get("id_token_hint")
	clientIDParam := r.Form.Get("client_id")
	postLogout := r.Form.Get("post_logout_redirect_uri")
	state := r.Form.Get("state")

	var (
		hintSub string
		client  *OAuth2Client
	)
	if hintRaw != "" {
		sub, c, err := p.verifyIDTokenHint(ctx.Context(), hintRaw)
		if err != nil {
			return forge.BadRequest("id_token_hint is not a valid id_token from this server")
		}
		hintSub, client = sub, c
	}
	if clientIDParam != "" {
		if client != nil && client.ClientID != clientIDParam {
			return forge.BadRequest("client_id does not match the id_token_hint")
		}
		if client == nil {
			c, err := p.oauth2Store.GetClient(ctx.Context(), clientIDParam)
			if err != nil {
				return forge.BadRequest("invalid client_id")
			}
			client = c
		}
	}

	target := ""
	if postLogout != "" {
		if client == nil {
			return forge.BadRequest("post_logout_redirect_uri needs client_id or id_token_hint")
		}
		if !slices.Contains(client.PostLogoutRedirectURIs, postLogout) {
			return forge.BadRequest("post_logout_redirect_uri is not registered for this client")
		}
		var err error
		if target, err = withState(postLogout, state); err != nil {
			return forge.BadRequest("invalid post_logout_redirect_uri")
		}
	}

	sessID, signedIn := middleware.SessionIDFrom(ctx.Context())
	userID, hasUser := middleware.UserIDFrom(ctx.Context())
	if !signedIn || !hasUser {
		return p.loggedOut(ctx, target)
	}
	if hintSub != "" && hintSub == userID.String() && p.callerAppMatches(ctx, client.AppID) {
		if err := p.endSession(ctx, sessID); err != nil {
			return err
		}
		return p.loggedOut(ctx, target)
	}

	appID, _ := middleware.AppIDFrom(ctx.Context())
	pl := pendingLogout{SessionID: sessID.String(), UserID: userID.String(), AppID: appID.String(), Target: target}
	if client != nil {
		pl.ClientName = client.Name
	}
	return p.confirmLogout(ctx, pl)
}

// verifyIDTokenHint checks that raw is an id_token this server signed and
// returns its subject and client. Expiry is ignored (RP-Initiated Logout
// 1.0 section 2: the hint may have expired); signature, issuer and audience
// are not.
func (p *Plugin) verifyIDTokenHint(ctx context.Context, raw string) (string, *OAuth2Client, error) {
	if len(raw) > maxIDTokenHintBytes {
		return "", nil, errors.New("id_token_hint too long")
	}
	if p.keys == nil {
		return "", nil, errNoSigningKey
	}
	var unverified idTokenClaims
	if _, _, err := jwt.NewParser().ParseUnverified(raw, &unverified); err != nil {
		return "", nil, err
	}
	clientID := unverified.Azp
	if clientID == "" && len(unverified.Audience) == 1 {
		clientID = unverified.Audience[0]
	}
	if clientID == "" {
		return "", nil, errors.New("id_token_hint names no client")
	}
	client, err := p.oauth2Store.GetClient(ctx, clientID)
	if err != nil {
		return "", nil, err
	}
	issuer, scope, err := p.issuerFor(ctx, client.AppID)
	if err != nil {
		return "", nil, err
	}
	keys, err := p.keys.Published(ctx, scope)
	if err != nil {
		return "", nil, err
	}

	var claims idTokenClaims
	_, err = jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string) //nolint:errcheck // a missing kid fails the lookup below
		for i := range keys {
			if keys[i].KID != kid || kid == "" {
				continue
			}
			if keys[i].ALG != "" && keys[i].ALG != t.Method.Alg() {
				return nil, fmt.Errorf("alg %s does not match key %s", t.Method.Alg(), kid)
			}
			doc, merr := json.Marshal(keys[i])
			if merr != nil {
				return nil, merr
			}
			pub, _, perr := jwkutil.ParseJSON(doc)
			return pub, perr
		}
		return nil, fmt.Errorf("unknown signing key %q", kid)
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return "", nil, err
	}
	if claims.Issuer != issuer {
		return "", nil, errors.New("id_token_hint issuer mismatch")
	}
	if !slices.Contains(claims.Audience, client.ClientID) {
		return "", nil, errors.New("id_token_hint audience mismatch")
	}
	if claims.Subject == "" {
		return "", nil, errors.New("id_token_hint has no subject")
	}
	return claims.Subject, client, nil
}

// sessionSigner ends a session with the sign-out hooks firing.
// *authsome.Engine satisfies it.
type sessionSigner interface {
	SignOut(ctx context.Context, sessionID id.SessionID) error
}

// endSession signs the session out and clears its cookie.
func (p *Plugin) endSession(ctx forge.Context, sessID id.SessionID) error {
	so, ok := p.engine.(sessionSigner)
	if !ok {
		return middleware.InternalError(ctx, errors.New("oauth2: logout needs an engine that can sign out"))
	}
	if err := so.SignOut(ctx.Context(), sessID); err != nil {
		return middleware.InternalError(ctx, fmt.Errorf("oauth2: sign out: %w", err))
	}
	if mgr := p.engine.Settings(); mgr != nil {
		appID := ""
		if a, ok := middleware.AppIDFrom(ctx.Context()); ok {
			appID = a.String()
		}
		c := authsome.SessionCookieTemplate(ctx.Context(), mgr, appID, middleware.IsHTTPS(ctx.Request())) // #nosec G124 -- template sets HttpOnly+SameSite+Secure
		c.Value = ""
		c.MaxAge = -1
		http.SetCookie(ctx.Response(), c)
	}
	return nil
}

// withState is the post-logout redirect with state attached.
func withState(uri, state string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if state != "" {
		q := u.Query()
		q.Set("state", state)
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

// loggedOut finishes: back to the client when it gave a registered URI,
// otherwise a page saying the user is signed out.
func (p *Plugin) loggedOut(ctx forge.Context, target string) error {
	ctx.SetHeader("Cache-Control", "no-store")
	if target != "" {
		return ctx.Redirect(http.StatusFound, target)
	}
	return renderLogoutPage(ctx, logoutView{Title: "You're signed out", Message: "You can close this window."})
}

// confirmLogout parks the request and shows the confirmation page.
func (p *Plugin) confirmLogout(ctx forge.Context, pl pendingLogout) error {
	lid, err := generateSecureToken(24)
	if err != nil {
		return middleware.InternalError(ctx, fmt.Errorf("oauth2: logout id: %w", err))
	}
	if pl.CSRF, err = generateSecureToken(24); err != nil {
		return middleware.InternalError(ctx, fmt.Errorf("oauth2: logout csrf: %w", err))
	}
	raw, err := json.Marshal(pl)
	if err != nil {
		return middleware.InternalError(ctx, fmt.Errorf("oauth2: encode logout: %w", err))
	}
	if err := p.ceremonyStore().Set(ctx.Context(), logoutKey(lid), raw, logoutTTL); err != nil {
		return middleware.InternalError(ctx, fmt.Errorf("oauth2: park logout: %w", err))
	}
	msg := "Do you want to sign out?"
	if pl.ClientName != "" {
		msg = pl.ClientName + " is asking to sign you out. Do you want to sign out?"
	}
	ctx.SetHeader("Cache-Control", "no-store")
	return renderLogoutPage(ctx, logoutView{
		Title: "Sign out", Message: msg, Confirm: true,
		Action: strings.TrimSuffix(ctx.Request().URL.Path, "/") + "/confirm", LID: lid, CSRF: pl.CSRF,
	})
}

// handleLogoutConfirm takes the answer from the confirmation page.
func (p *Plugin) handleLogoutConfirm(ctx forge.Context) error {
	r := ctx.Request()
	if err := r.ParseForm(); err != nil {
		return forge.BadRequest("invalid logout form")
	}
	lid := r.PostForm.Get("lid")
	if lid == "" {
		return forge.BadRequest("lid is required")
	}
	raw, err := p.ceremonyStore().Get(ctx.Context(), logoutKey(lid))
	if err != nil {
		return forge.BadRequest("this sign-out request has expired; start again")
	}
	var pl pendingLogout
	if err := json.Unmarshal(raw, &pl); err != nil {
		return forge.BadRequest("invalid sign-out request")
	}
	if subtle.ConstantTimeCompare([]byte(r.PostForm.Get("csrf")), []byte(pl.CSRF)) != 1 {
		return forge.NewHTTPError(http.StatusForbidden, "sign-out token mismatch")
	}
	_ = p.ceremonyStore().Delete(ctx.Context(), logoutKey(lid)) //nolint:errcheck // best-effort, single use

	if r.PostForm.Get("decision") != "logout" {
		return renderLogoutPage(ctx, logoutView{Title: "You're still signed in", Message: "Nothing changed. You can close this window."})
	}
	// Only the session that was asked about. If it has changed since, the
	// browser already moved on and there is nothing to end.
	if sessID, ok := middleware.SessionIDFrom(ctx.Context()); ok && sessID.String() == pl.SessionID {
		if err := p.endSession(ctx, sessID); err != nil {
			return err
		}
	}
	return p.loggedOut(ctx, pl.Target)
}

type logoutView struct {
	Title, Message string
	Confirm        bool
	Action         string
	LID, CSRF      string
}

var logoutTemplate = template.Must(template.New("logout").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="referrer" content="no-referrer">
<title>{{.Title}}</title>
<style>
body{font-family:system-ui,sans-serif;background:#f6f7f9;margin:0;display:flex;min-height:100vh;align-items:center;justify-content:center}
main{background:#fff;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,.08);padding:32px;max-width:420px;width:calc(100% - 32px)}
h1{font-size:1.25rem;margin:0 0 8px}p{color:#444;margin:0 0 24px}
form{display:flex;gap:12px}button{flex:1;padding:12px;border-radius:8px;border:1px solid #ccc;background:#fff;font-size:1rem;cursor:pointer}
button.primary{background:#1f6feb;border-color:#1f6feb;color:#fff}
</style>
</head>
<body>
<main>
<h1>{{.Title}}</h1>
<p>{{.Message}}</p>
{{if .Confirm}}<form method="post" action="{{.Action}}">
<input type="hidden" name="lid" value="{{.LID}}">
<input type="hidden" name="csrf" value="{{.CSRF}}">
<button type="submit" name="decision" value="stay">Stay signed in</button>
<button type="submit" name="decision" value="logout" class="primary">Sign out</button>
</form>{{end}}
</main>
</body>
</html>
`))

func renderLogoutPage(ctx forge.Context, v logoutView) error {
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.SetHeader("X-Frame-Options", "DENY")
	ctx.SetHeader("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'")
	ctx.Response().WriteHeader(http.StatusOK)
	return logoutTemplate.Execute(ctx.Response(), v)
}
