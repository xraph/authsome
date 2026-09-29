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
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// Consent. An authorization request from a client the operator does not own
// is not completed until the signed-in user has approved every scope it
// asks for. The approval is a Grant, kept per app, user and client, so the
// page appears once per client rather than once per login. The request
// waits in the ceremony store under a consent id while the user decides;
// the decision endpoint completes or denies it from that record, never from
// parameters the browser could rewrite.

// consentTTL bounds how long a pending request waits for a decision.
const consentTTL = 10 * time.Minute

// pendingConsent is the authorization request as it waits for a decision,
// and the shape both the page and the code issuance read from.
type pendingConsent struct {
	ClientID            string   `json:"client_id"`
	UserID              string   `json:"user_id"`
	AppID               string   `json:"app_id"`
	OrgID               string   `json:"org_id,omitempty"`
	RedirectURI         string   `json:"redirect_uri"`
	Scopes              []string `json:"scopes"`
	Resources           []string `json:"resources,omitempty"`
	State               string   `json:"state,omitempty"`
	CodeChallenge       string   `json:"code_challenge,omitempty"`
	CodeChallengeMethod string   `json:"code_challenge_method,omitempty"`
	// CSRF is minted with the record and must come back with the decision.
	CSRF string `json:"csrf"`
}

func consentKey(cid string) string { return "oauth2:consent:" + cid }

// ceremonyStore returns the pending-consent store, falling back to an
// in-process one when the plugin was wired without an engine.
func (p *Plugin) ceremonyStore() ceremony.Store {
	if p.ceremonies == nil {
		p.ceremonies = ceremony.NewMemory()
	}
	return p.ceremonies
}

// needsConsent reports whether the user must decide before a code is
// issued: never for a first-party client, always when the client asked for
// prompt=consent, otherwise whenever no grant covers the requested scopes.
func (p *Plugin) needsConsent(ctx context.Context, client *OAuth2Client, userID id.UserID, scopes []string, prompt string) bool {
	if client.FirstParty {
		return false
	}
	if strings.Contains(prompt, "consent") {
		return true
	}
	g, err := p.oauth2Store.GetGrant(ctx, client.AppID, userID, client.ClientID)
	if err != nil {
		return true
	}
	return !g.Covers(scopes)
}

// redirectToConsent parks the request and sends the browser to the page.
func (p *Plugin) redirectToConsent(ctx forge.Context, pc pendingConsent) error {
	cid, err := generateSecureToken(24)
	if err != nil {
		return forge.InternalError(fmt.Errorf("oauth2: consent id: %w", err))
	}
	if pc.CSRF, err = generateSecureToken(24); err != nil {
		return forge.InternalError(fmt.Errorf("oauth2: consent csrf: %w", err))
	}
	raw, err := json.Marshal(pc)
	if err != nil {
		return forge.InternalError(fmt.Errorf("oauth2: encode consent: %w", err))
	}
	if err := p.ceremonyStore().Set(ctx.Context(), consentKey(cid), raw, consentTTL); err != nil {
		return forge.InternalError(fmt.Errorf("oauth2: park consent: %w", err))
	}
	// The page lives beside the authorization endpoint, whatever prefix the
	// router mounted the plugin under.
	page := strings.TrimSuffix(ctx.Request().URL.Path, "/authorize") + "/consent?cid=" + url.QueryEscape(cid)
	return ctx.Redirect(http.StatusFound, page)
}

// loadConsent returns the pending request for cid if it belongs to the
// signed-in user.
func (p *Plugin) loadConsent(ctx forge.Context, cid string) (*pendingConsent, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	if cid == "" {
		return nil, forge.BadRequest("cid is required")
	}
	raw, err := p.ceremonyStore().Get(ctx.Context(), consentKey(cid))
	if err != nil {
		return nil, forge.BadRequest("this authorization request has expired; start again")
	}
	var pc pendingConsent
	if err := json.Unmarshal(raw, &pc); err != nil {
		return nil, forge.BadRequest("invalid consent request")
	}
	if pc.UserID != userID.String() {
		return nil, forge.NewHTTPError(http.StatusForbidden, "this authorization request belongs to another user")
	}
	return &pc, nil
}

// issueAuthorizationCode mints and stores the code for an approved request
// and returns the redirect the browser should follow.
func (p *Plugin) issueAuthorizationCode(ctx context.Context, pc pendingConsent) (string, error) {
	userID, err := id.ParseUserID(pc.UserID)
	if err != nil {
		return "", forge.InternalError(fmt.Errorf("oauth2: consent user: %w", err))
	}
	appID, err := id.ParseAppID(pc.AppID)
	if err != nil {
		return "", forge.InternalError(fmt.Errorf("oauth2: consent app: %w", err))
	}
	codeStr, err := generateSecureToken(32)
	if err != nil {
		return "", forge.InternalError(fmt.Errorf("oauth2: generate auth code: %w", err))
	}
	authCode := &AuthorizationCode{
		ID:                  id.NewAuthCodeID(),
		Code:                codeStr,
		ClientID:            pc.ClientID,
		UserID:              userID,
		AppID:               appID,
		RedirectURI:         pc.RedirectURI,
		Scopes:              pc.Scopes,
		Resources:           pc.Resources,
		CodeChallenge:       pc.CodeChallenge,
		CodeChallengeMethod: pc.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(p.config.AuthCodeTTL),
		CreatedAt:           time.Now(),
	}
	if createErr := p.oauth2Store.CreateAuthCode(ctx, authCode); createErr != nil {
		return "", forge.InternalError(fmt.Errorf("oauth2: store auth code: %w", createErr))
	}
	return buildRedirect(pc.RedirectURI, codeStr, pc.State)
}

// denyRedirect is where a refused request sends the browser: the client's
// redirect URI with error=access_denied (RFC 6749 4.1.2.1).
func denyRedirect(pc *pendingConsent) (string, error) {
	u, err := url.Parse(pc.RedirectURI)
	if err != nil {
		return "", forge.BadRequest("invalid redirect_uri")
	}
	q := u.Query()
	q.Set("error", "access_denied")
	q.Set("error_description", "the user denied the request")
	if pc.State != "" {
		q.Set("state", pc.State)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ConsentView is what the page and the JSON form of it show.
type ConsentView struct {
	CID        string   `json:"cid"`
	CSRF       string   `json:"csrf"`
	ClientID   string   `json:"client_id"`
	ClientName string   `json:"client_name"`
	Scopes     []string `json:"scopes"`
	// Action is the path the decision posts to.
	Action string `json:"action"`
}

var consentTemplate = template.Must(template.New("consent").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="referrer" content="no-referrer">
<title>Authorize {{.ClientName}}</title>
<style>
body{font-family:system-ui,sans-serif;background:#f6f7f9;margin:0;display:flex;min-height:100vh;align-items:center;justify-content:center}
main{background:#fff;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,.08);padding:32px;max-width:420px;width:calc(100% - 32px)}
h1{font-size:1.25rem;margin:0 0 8px}p{color:#444;margin:0 0 16px}ul{padding-left:20px;margin:0 0 24px}li{margin:4px 0}
form{display:flex;gap:12px}button{flex:1;padding:12px;border-radius:8px;border:1px solid #ccc;background:#fff;font-size:1rem;cursor:pointer}
button.approve{background:#1f6feb;border-color:#1f6feb;color:#fff}
</style>
</head>
<body>
<main>
<h1>Authorize {{.ClientName}}</h1>
<p><strong>{{.ClientName}}</strong> is asking for access to your account. It will be able to:</p>
<ul>{{range .Scopes}}<li><code>{{.}}</code></li>{{end}}</ul>
<form method="post" action="{{.Action}}">
<input type="hidden" name="cid" value="{{.CID}}">
<input type="hidden" name="csrf" value="{{.CSRF}}">
<button type="submit" name="decision" value="deny">Deny</button>
<button type="submit" name="decision" value="approve" class="approve">Allow</button>
</form>
</main>
</body>
</html>
`))

// handleConsentPage shows the pending request. A browser gets HTML; a client
// that asks for JSON gets the same fields, so a custom front end can render
// its own page and post the decision.
func (p *Plugin) handleConsentPage(ctx forge.Context) error {
	r := ctx.Request()
	pc, err := p.loadConsent(ctx, r.URL.Query().Get("cid"))
	if err != nil {
		return err
	}
	client, err := p.oauth2Store.GetClient(ctx.Context(), pc.ClientID)
	if err != nil {
		return forge.BadRequest("invalid client_id")
	}
	view := ConsentView{
		CID: r.URL.Query().Get("cid"), CSRF: pc.CSRF, ClientID: client.ClientID, ClientName: client.Name,
		Scopes: pc.Scopes, Action: r.URL.Path,
	}
	ctx.SetHeader("Cache-Control", "no-store")
	ctx.SetHeader("Pragma", "no-cache")
	if wantsJSON(r) {
		return ctx.JSON(http.StatusOK, view)
	}
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.SetHeader("X-Frame-Options", "DENY")
	ctx.SetHeader("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'")
	ctx.Response().WriteHeader(http.StatusOK)
	return consentTemplate.Execute(ctx.Response(), view)
}

// ConsentDecision is the JSON form of the decision body.
type ConsentDecision struct {
	CID      string `json:"cid"`
	CSRF     string `json:"csrf"`
	Decision string `json:"decision"`
}

// ConsentResult tells a JSON caller where to send the browser next.
type ConsentResult struct {
	RedirectURL string `json:"redirect_url"`
}

// handleConsentDecision records the decision and completes or denies the
// parked request. A form post is answered with a redirect; a JSON post with
// the redirect URL.
func (p *Plugin) handleConsentDecision(ctx forge.Context) error {
	r := ctx.Request()
	var in ConsentDecision
	isJSON := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
	if isJSON {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return forge.BadRequest("invalid consent decision")
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return forge.BadRequest("invalid consent form")
		}
		in = ConsentDecision{CID: r.PostForm.Get("cid"), CSRF: r.PostForm.Get("csrf"), Decision: r.PostForm.Get("decision")}
	}
	pc, err := p.loadConsent(ctx, in.CID)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(in.CSRF), []byte(pc.CSRF)) != 1 {
		return forge.NewHTTPError(http.StatusForbidden, "consent token mismatch")
	}
	// Single use, whatever the decision.
	_ = p.ceremonyStore().Delete(ctx.Context(), consentKey(in.CID)) //nolint:errcheck // best-effort

	var target string
	switch in.Decision {
	case "approve":
		userID, perr := id.ParseUserID(pc.UserID)
		if perr != nil {
			return forge.InternalError(fmt.Errorf("oauth2: consent user: %w", perr))
		}
		appID, perr := id.ParseAppID(pc.AppID)
		if perr != nil {
			return forge.InternalError(fmt.Errorf("oauth2: consent app: %w", perr))
		}
		if grantErr := p.recordGrant(ctx.Context(), appID, userID, pc.ClientID, pc.Scopes); grantErr != nil {
			return forge.InternalError(fmt.Errorf("oauth2: record grant: %w", grantErr))
		}
		target, err = p.issueAuthorizationCode(ctx.Context(), *pc)
	case "deny":
		target, err = denyRedirect(pc)
	default:
		return forge.BadRequest("decision must be approve or deny")
	}
	if err != nil {
		return err
	}
	ctx.SetHeader("Cache-Control", "no-store")
	if isJSON {
		return ctx.JSON(http.StatusOK, ConsentResult{RedirectURL: target})
	}
	return ctx.Redirect(http.StatusFound, target)
}

// recordGrant merges the approved scopes into the user's grant for the
// client, so an approval never narrows what an earlier one allowed.
func (p *Plugin) recordGrant(ctx context.Context, appID id.AppID, userID id.UserID, clientID string, scopes []string) error {
	merged := append([]string(nil), scopes...)
	if existing, err := p.oauth2Store.GetGrant(ctx, appID, userID, clientID); err == nil {
		seen := make(map[string]bool, len(merged))
		for _, s := range merged {
			seen[s] = true
		}
		for _, s := range existing.Scopes {
			if !seen[s] {
				merged = append(merged, s)
			}
		}
	} else if !errors.Is(err, ErrGrantNotFound) {
		return err
	}
	return p.oauth2Store.UpsertGrant(ctx, &Grant{AppID: appID, UserID: userID, ClientID: clientID, Scopes: merged})
}

func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html")
}

// GrantView is one entry of the signed-in user's grant list.
type GrantView struct {
	ClientID   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	Scopes     []string  `json:"scopes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ListGrantsResponse is the body of GET /v1/me/oauth/grants.
type ListGrantsResponse struct {
	Grants []GrantView `json:"grants"`
}

// GrantPathRequest names the client whose grant is revoked.
type GrantPathRequest struct {
	ClientID string `path:"clientId"`
}

// meAppID resolves the app the signed-in user belongs to.
func (p *Plugin) meAppID(ctx forge.Context) (appID id.AppID, userID id.UserID, err error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return id.AppID{}, id.UserID{}, forge.Unauthorized("authentication required")
	}
	if fromCtx, ok := middleware.AppIDFrom(ctx.Context()); ok {
		return fromCtx, userID, nil
	}
	u, getErr := p.store.GetUser(ctx.Context(), userID)
	if getErr != nil {
		return id.AppID{}, id.UserID{}, forge.Unauthorized("authentication required")
	}
	return u.AppID, userID, nil
}

func (p *Plugin) handleListMyGrants(ctx forge.Context, _ *ListGrantsRequest) (*ListGrantsResponse, error) {
	appID, userID, err := p.meAppID(ctx)
	if err != nil {
		return nil, err
	}
	grants, err := p.oauth2Store.ListGrantsByUser(ctx.Context(), appID, userID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: list grants: %w", err))
	}
	out := &ListGrantsResponse{Grants: make([]GrantView, 0, len(grants))}
	for _, g := range grants {
		name := g.ClientID
		if c, cerr := p.oauth2Store.GetClient(ctx.Context(), g.ClientID); cerr == nil {
			name = c.Name
		}
		out.Grants = append(out.Grants, GrantView{ClientID: g.ClientID, ClientName: name, Scopes: g.Scopes, CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt})
	}
	return out, nil
}

// ListGrantsRequest carries nothing; the user comes from the session.
type ListGrantsRequest struct{}

func (p *Plugin) handleRevokeMyGrant(ctx forge.Context, req *GrantPathRequest) (*ConsentResult, error) {
	appID, userID, err := p.meAppID(ctx)
	if err != nil {
		return nil, err
	}
	if err := p.oauth2Store.DeleteGrant(ctx.Context(), appID, userID, req.ClientID); err != nil {
		if errors.Is(err, ErrGrantNotFound) {
			return nil, forge.NotFound("no grant for this client")
		}
		return nil, forge.InternalError(fmt.Errorf("oauth2: revoke grant: %w", err))
	}
	return nil, ctx.NoContent(http.StatusNoContent)
}
