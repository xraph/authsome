package oauth2provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// Sign-in before authorization. A browser that reaches /authorize with no
// session is sent to a login page, carrying the authorize URL so the user
// lands back on it once signed in. That return trip is what lets a third
// party put a "Sign in with <your app>" button on its own site.

// SettingAppLoginURL gives one app its own login page.
var SettingAppLoginURL = settings.Define("oauth2.login_url", "",
	settings.WithScopes(settings.ScopeApp),
	settings.WithDisplayName("Login page"),
	settings.WithDescription("Where a signed-out user is sent when an OAuth2 client asks them to authorize. "+
		"Use a path on the issuer's origin, like /sign-in, or a full https URL. "+
		"Leave it empty to use the deployment's login page."),
	settings.WithCategory("OpenID Connect"),
	settings.WithPlaceholder("https://example.com/sign-in"),
	settings.WithValidation(func(raw json.RawMessage) error {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return fmt.Errorf("login page must be a string: %w", err)
		}
		_, err := ValidateLoginURL(s)
		return err
	}),
)

// ValidateLoginURL checks a login page value and returns it parsed. Empty
// is valid and means no login page. Otherwise it is a path starting with
// one "/" or an absolute http or https URL: anything else (a relative path,
// a scheme-relative "//host", a javascript: URL) would send the browser
// somewhere nobody configured.
func ValidateLoginURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("oauth2: login URL %q: %w", raw, err)
	}
	switch {
	case u.Scheme == "" && u.Host == "" && strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//"):
		return u, nil
	case (u.Scheme == "https" || u.Scheme == "http") && u.Host != "":
		return u, nil
	}
	return nil, fmt.Errorf("oauth2: login URL %q must be a path starting with / or an absolute http(s) URL", raw)
}

// errLoginRequired is the OpenID Connect answer to prompt=none when nobody
// is signed in (OpenID Connect Core 3.1.2.6).
const errLoginRequired = "login_required"

// signedOutAuthorize answers an /authorize request that has no session. The
// request has already been validated, so redirectURI is one the client
// registered and is safe to send an error to.
func (p *Plugin) signedOutAuthorize(ctx forge.Context, client *OAuth2Client, redirectURI string, req *AuthorizeRequest) error {
	// prompt=none forbids any UI, a login page included. The client hears
	// about it at its redirect URI and decides what to do.
	if hasPrompt(req.Prompt, "none") {
		to, err := errorRedirect(redirectURI, errLoginRequired, "the user is not signed in", req.State)
		if err != nil {
			return err
		}
		return ctx.Redirect(http.StatusFound, to)
	}

	login, err := p.loginURLFor(ctx.Context(), client.AppID)
	if err != nil {
		return err
	}
	if login == nil {
		return forge.Unauthorized("authentication required to authorize")
	}
	return ctx.Redirect(http.StatusFound, p.loginRedirect(login, returnURL(ctx.Request(), time.Now())))
}

// loginAfterParam marks an authorize URL that has already sent the user to
// sign in for prompt=login. It holds the Unix time the trip began.
const loginAfterParam = "login_after"

// returnURL is the authorize URL the login page sends the browser back to.
// The trip to the login page is what prompt=login asked for, so "login"
// leaves the prompt and login_after records when the trip began. Without
// that, the request coming back would ask for a login again, forever.
// login_after only ever adds a requirement, so a client gains nothing by
// setting it.
func returnURL(r *http.Request, now time.Time) *url.URL {
	back := *r.URL
	q := back.Query()
	if !hasPrompt(q.Get("prompt"), "login") {
		return &back
	}
	var rest []string
	for _, v := range strings.Fields(q.Get("prompt")) {
		if v != "login" {
			rest = append(rest, v)
		}
	}
	if len(rest) == 0 {
		q.Del("prompt")
	} else {
		q.Set("prompt", strings.Join(rest, " "))
	}
	q.Set(loginAfterParam, strconv.FormatInt(now.Unix(), 10))
	back.RawQuery = q.Encode()
	return &back
}

// parseMaxAge reads max_age (OpenID Connect Core 3.1.2.1): seconds since
// the user last signed in, a non-negative integer. ok is false when the
// request did not send one.
func parseMaxAge(raw string) (maxAge int64, ok bool, err error) {
	if raw == "" {
		return 0, false, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0, false, errors.New("max_age must be a non-negative integer")
	}
	return v, true, nil
}

// freshLoginRequired reports whether a signed-in user has to sign in again
// before this request is answered: prompt=login asks for it outright,
// max_age when the sign-in is older than that many seconds, and
// login_after when the session predates the trip to the login page.
func freshLoginRequired(prompt string, maxAge int64, hasMaxAge bool, loginAfter string, authTime, now time.Time) bool {
	if hasPrompt(prompt, "login") {
		return true
	}
	if hasMaxAge && now.Sub(authTime) > time.Duration(maxAge)*time.Second {
		return true
	}
	if after, err := strconv.ParseInt(loginAfter, 10, 64); err == nil && authTime.Unix() < after {
		return true
	}
	return false
}

// reauthenticate sends a signed-in user back to the login page. With
// prompt=none, or with no login page to send them to, the client hears
// login_required instead.
func (p *Plugin) reauthenticate(ctx forge.Context, client *OAuth2Client, redirectURI string, req *AuthorizeRequest) error {
	if !hasPrompt(req.Prompt, "none") {
		login, err := p.loginURLFor(ctx.Context(), client.AppID)
		if err != nil {
			return err
		}
		if login != nil {
			return ctx.Redirect(http.StatusFound, p.loginRedirect(login, returnURL(ctx.Request(), time.Now())))
		}
	}
	to, err := errorRedirect(redirectURI, errLoginRequired, "the user has to sign in again", req.State)
	if err != nil {
		return err
	}
	return ctx.Redirect(http.StatusFound, to)
}

// loginURLFor returns the login page for appID: the app's own setting when
// it has one, else the deployment's, else nil.
func (p *Plugin) loginURLFor(ctx context.Context, appID id.AppID) (*url.URL, error) {
	if raw := p.appLoginURL(ctx, appID); raw != "" {
		u, err := ValidateLoginURL(raw)
		if err == nil {
			return u, nil
		}
		// The setting validates on write, so this is a value stored before
		// validation existed. Fall back rather than strand the user.
		if p.logger != nil {
			p.logger.Warn("oauth2: ignoring an invalid per-app login URL",
				log.String("app_id", appID.String()), log.Error(err))
		}
	}
	return ValidateLoginURL(p.config.LoginURL)
}

// appLoginURL reads the app's login page setting. A read that fails is
// logged and treated as unset: the deployment's page is still a correct
// place to sign in, unlike a wrong issuer.
func (p *Plugin) appLoginURL(ctx context.Context, appID id.AppID) string {
	if p.engine == nil {
		return ""
	}
	mgr := p.engine.Settings()
	if mgr == nil {
		return ""
	}
	v, err := settings.Get(ctx, mgr, SettingAppLoginURL, settings.ResolveOpts{AppID: appID.String()})
	if err != nil {
		if !errors.Is(err, settings.ErrUnknownKey) && p.logger != nil {
			p.logger.Warn("oauth2: read "+SettingAppLoginURL.Def.Key, log.String("app_id", appID.String()), log.Error(err))
		}
		return ""
	}
	return v
}

// loginRedirect builds the login page URL with the authorize request
// attached. A login page on the issuer's origin gets a relative path, which
// is what sign-in pages that refuse off-site redirects accept; one on
// another origin gets the absolute URL, since a path would resolve against
// the wrong host.
func (p *Plugin) loginRedirect(login, authorize *url.URL) string {
	back := authorize.Path
	if authorize.RawQuery != "" {
		back += "?" + authorize.RawQuery
	}

	issuer, err := url.Parse(p.issuerURL())
	sameOrigin := login.Host == "" || (err == nil && strings.EqualFold(login.Scheme, issuer.Scheme) && strings.EqualFold(login.Host, issuer.Host))
	if !sameOrigin && err == nil {
		back = issuer.Scheme + "://" + issuer.Host + back
	}

	to := *login
	q := to.Query()
	q.Set(p.config.LoginReturnParam, back)
	to.RawQuery = q.Encode()
	return to.String()
}

// authorizeError sends an RFC 6749 4.1.2.1 error to the client's redirect
// URI, which the caller has already validated.
func (p *Plugin) authorizeError(ctx forge.Context, redirectURI, code, description, state string) error {
	to, err := errorRedirect(redirectURI, code, description, state)
	if err != nil {
		return err
	}
	return ctx.Redirect(http.StatusFound, to)
}

// hasPrompt reports whether the space-separated prompt list carries value.
func hasPrompt(prompt, value string) bool {
	for _, v := range strings.Fields(prompt) {
		if v == value {
			return true
		}
	}
	return false
}

// errorRedirect is the client's redirect URI with an RFC 6749 4.1.2.1
// error attached.
func errorRedirect(redirectURI, code, description, state string) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", forge.BadRequest("invalid redirect_uri")
	}
	q := u.Query()
	q.Set("error", code)
	if description != "" {
		q.Set("error_description", description)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
