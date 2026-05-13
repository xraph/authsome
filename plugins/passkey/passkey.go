package passkey

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"

	"github.com/xraph/grove/migrate"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin                = (*Plugin)(nil)
	_ plugin.OnInit                = (*Plugin)(nil)
	_ plugin.RouteProvider         = (*Plugin)(nil)
	_ plugin.MigrationProvider     = (*Plugin)(nil)
	_ plugin.AuthMethodContributor = (*Plugin)(nil)
	_ plugin.SettingsProvider      = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingRPDisplayName controls the relying party display name.
	SettingRPDisplayName = settings.Define("passkey.rp_display_name", "AuthSome",
		settings.WithDisplayName("RP Display Name"),
		settings.WithDescription("Relying party display name shown to users during WebAuthn ceremonies"),
		settings.WithCategory("Passkey / WebAuthn"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldText),
		settings.WithPlaceholder("AuthSome"),
		settings.WithUIValidation(formconfig.Validation{Required: true, MaxLen: 100}),
		settings.WithHelpText("The name users see when registering or authenticating with passkeys"),
		settings.WithOrder(10),
	)

	// SettingRPID controls the relying party identifier.
	SettingRPID = settings.Define("passkey.rp_id", "localhost",
		settings.WithDisplayName("RP ID"),
		settings.WithDescription("Relying party identifier (typically the domain name)"),
		settings.WithCategory("Passkey / WebAuthn"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldText),
		settings.WithPlaceholder("example.com"),
		settings.WithUIValidation(formconfig.Validation{
			Required: true,
			MaxLen:   253,
			// Hostname (RFC 1123) or "localhost" / "127.0.0.1" / "::1".
			Pattern: `^(localhost|127\.0\.0\.1|::1|([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*)$`,
		}),
		settings.WithHelpText("Must match the effective domain users browse to. Use 'localhost' for dev."),
		settings.WithOrder(20),
	)

	// SettingRPOrigins controls the allowed origins for WebAuthn ceremonies.
	// When empty, origins are derived from RPID (and the request origin for
	// localhost dev mode — see Plugin.waForRequest).
	SettingRPOrigins = settings.Define("passkey.rp_origins", []string{},
		settings.WithDisplayName("RP Origins"),
		settings.WithDescription("Allowed origins for WebAuthn ceremonies (e.g. https://app.example.com)"),
		settings.WithCategory("Passkey / WebAuthn"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldTextarea),
		settings.WithPlaceholder("https://app.example.com\nhttps://www.example.com"),
		settings.WithHelpText("One origin per line. Each must include scheme and port. Leave empty to use Config defaults; localhost dev auto-trusts the request origin regardless of port."),
		settings.WithOrder(25),
	)

	// SettingSessionTimeoutSeconds controls the WebAuthn ceremony timeout.
	SettingSessionTimeoutSeconds = settings.Define("passkey.session_timeout_seconds", 300,
		settings.WithDisplayName("Ceremony Timeout (seconds)"),
		settings.WithDescription("How long a WebAuthn ceremony session lives in seconds"),
		settings.WithCategory("Passkey / WebAuthn"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(30), Max: intPtr(600)}),
		settings.WithHelpText("Time allowed to complete a passkey ceremony. Default: 300 (5 minutes)"),
		settings.WithOrder(30),
	)
)

func intPtr(v int) *int { return &v }

// Config configures the passkey/WebAuthn plugin.
type Config struct {
	// RPDisplayName is the relying party display name shown to users.
	RPDisplayName string

	// RPID is the relying party identifier (typically the domain name).
	RPID string

	// RPOrigins are the allowed origins for WebAuthn ceremonies.
	RPOrigins []string

	// SessionTimeout is how long a WebAuthn ceremony session lives (default: 5 minutes).
	SessionTimeout time.Duration
}

// Plugin is the passkey/WebAuthn plugin.
type Plugin struct {
	config       Config
	store        Store
	wa           *webauthn.WebAuthn
	ceremonies   ceremony.Store
	chronicle    bridge.Chronicle
	relay        bridge.EventRelay
	hooks        *hook.Bus
	logger       log.Logger
	settingsMgr  *settings.Manager
	originWAOnce sync.Map // map[string]*webauthn.WebAuthn — per-origin webauthn cache for localhost dev
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "passkey", SettingRPDisplayName); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "passkey", SettingRPID); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "passkey", SettingRPOrigins); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "passkey", SettingSessionTimeoutSeconds)
}

// New creates a new passkey plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.RPDisplayName == "" {
		c.RPDisplayName = "AuthSome"
	}
	if c.RPID == "" {
		c.RPID = "localhost"
	}
	if c.SessionTimeout == 0 {
		c.SessionTimeout = 5 * time.Minute
	}
	// Auto-default RPOrigins for non-localhost RPIDs only. For localhost
	// variants, leave RPOrigins empty — waForRequest fills it in per-request
	// from the actual browser Origin header so any dev port "just works".
	if len(c.RPOrigins) == 0 && !isLocalhostHost(c.RPID) {
		c.RPOrigins = []string{"https://" + c.RPID}
	}
	p := &Plugin{
		config:     c,
		ceremonies: ceremony.NewMemory(),
	}
	// Eagerly initialize WebAuthn so the plugin works even without OnInit.
	// For localhost with no RPOrigins, supply a placeholder so webauthn.New
	// doesn't reject the empty list; per-request resolution will override.
	baseOrigins := c.RPOrigins
	if len(baseOrigins) == 0 {
		baseOrigins = []string{"http://" + c.RPID}
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: c.RPDisplayName,
		RPID:          c.RPID,
		RPOrigins:     baseOrigins,
	})
	if err == nil {
		p.wa = wa
	}
	return p
}

// isLocalhostHost reports whether host is a localhost variant (case-insensitive).
func isLocalhostHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "passkey" }

// OnInit initializes the WebAuthn library and optionally extracts bridges
// and the ceremony store from the engine.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	baseOrigins := p.config.RPOrigins
	if len(baseOrigins) == 0 {
		// Localhost path: supply a placeholder so webauthn.New accepts the
		// config; the real origin is resolved per-request in waForRequest.
		baseOrigins = []string{"http://" + p.config.RPID}
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: p.config.RPDisplayName,
		RPID:          p.config.RPID,
		RPOrigins:     baseOrigins,
	})
	if err != nil {
		return err
	}
	p.wa = wa

	if engine != nil {
		p.chronicle = engine.Chronicle()
		p.relay = engine.Relay()
		p.hooks = engine.Hooks()
		p.logger = engine.Logger()
		p.ceremonies = engine.CeremonyStore()
		p.settingsMgr = engine.Settings()
	}
	if p.ceremonies == nil {
		p.ceremonies = ceremony.NewMemory()
	}

	return nil
}

// waForRequest returns a *webauthn.WebAuthn appropriate for validating r.
//
// For non-localhost RPIDs the configured p.wa is returned unchanged unless
// the dynamic passkey.rp_origins setting adds origins beyond Config.RPOrigins.
//
// For localhost / 127.0.0.1 / ::1, the request's Origin (or Referer) header
// is parsed and — if its host matches the configured RPID — added to the
// allowed RPOrigins for the resulting per-request webauthn instance. The
// host check ensures an attacker cannot inject an arbitrary Origin to trick
// the server into accepting a foreign-origin assertion.
func (p *Plugin) waForRequest(r *http.Request) *webauthn.WebAuthn {
	if r == nil {
		return p.wa
	}

	// Start from the static config-defined origins.
	origins := append([]string(nil), p.config.RPOrigins...)
	modified := false

	// Merge dynamic settings overrides.
	if p.settingsMgr != nil {
		if dyn, err := settings.Get(r.Context(), p.settingsMgr, SettingRPOrigins, settings.ResolveOpts{}); err == nil {
			for _, o := range dyn {
				before := len(origins)
				origins = appendUnique(origins, o)
				if len(origins) != before {
					modified = true
				}
			}
		}
	}

	// Localhost dev mode: trust the request's own origin when host matches RPID.
	if isLocalhostHost(p.config.RPID) {
		if reqOrigin := requestOrigin(r); reqOrigin != "" {
			if parsed, err := url.Parse(reqOrigin); err == nil && parsed.Hostname() == p.config.RPID {
				candidate := parsed.Scheme + "://" + parsed.Host
				before := len(origins)
				origins = appendUnique(origins, candidate)
				if len(origins) != before {
					modified = true
				}
			}
		}
	}

	if !modified || len(origins) == 0 {
		return p.wa
	}

	cacheKey := joinOrigins(origins)
	if cached, ok := p.originWAOnce.Load(cacheKey); ok {
		if w, ok := cached.(*webauthn.WebAuthn); ok {
			return w
		}
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: p.config.RPDisplayName,
		RPID:          p.config.RPID,
		RPOrigins:     origins,
	})
	if err != nil {
		return p.wa
	}
	p.originWAOnce.Store(cacheKey, wa)
	return wa
}

// allowedOrigins returns the merged static origins (Config.RPOrigins +
// dynamic passkey.rp_origins setting). Used for diagnostics and tests; does
// not include any per-request origin.
func (p *Plugin) allowedOrigins(ctx context.Context) []string {
	out := make([]string, 0, len(p.config.RPOrigins)+2)
	for _, o := range p.config.RPOrigins {
		out = appendUnique(out, o)
	}
	if p.settingsMgr != nil {
		dyn, err := settings.Get(ctx, p.settingsMgr, SettingRPOrigins, settings.ResolveOpts{})
		if err == nil {
			for _, o := range dyn {
				out = appendUnique(out, o)
			}
		}
	}
	return out
}

// requestOrigin returns the request's Origin header, falling back to the
// scheme+host portion of the Referer header.
func requestOrigin(r *http.Request) string {
	if o := r.Header.Get("Origin"); o != "" {
		return o
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Host != "" {
			return u.Scheme + "://" + u.Host
		}
	}
	return ""
}

func appendUnique(dst []string, v string) []string {
	if v == "" {
		return dst
	}
	for _, x := range dst {
		if x == v {
			return dst
		}
	}
	return append(dst, v)
}

func joinOrigins(origins []string) string {
	out := ""
	for i, o := range origins {
		if i > 0 {
			out += "|"
		}
		out += o
	}
	return out
}

// MigrationGroups implements plugin.MigrationProvider.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite":
		return []*migrate.Group{SqliteMigrations}
	default:
		return nil
	}
}

// SetStore sets the credential store for testing.
func (p *Plugin) SetStore(s Store) {
	p.store = s
}

// ──────────────────────────────────────────────────
// AuthMethodContributor
// ──────────────────────────────────────────────────

// ListUserAuthMethods implements plugin.AuthMethodContributor.
// It returns one entry per registered passkey credential for the user.
func (p *Plugin) ListUserAuthMethods(ctx context.Context, userID id.UserID) ([]*plugin.AuthMethod, error) {
	if p.store == nil {
		return nil, nil
	}
	creds, err := p.store.ListUserCredentials(ctx, userID)
	if err != nil {
		return nil, nil
	}
	methods := make([]*plugin.AuthMethod, 0, len(creds))
	for _, pk := range creds {
		label := "Passkey"
		if pk.DisplayName != "" {
			label = pk.DisplayName
		}
		methods = append(methods, &plugin.AuthMethod{
			Type:     "passkey",
			Provider: "passkey",
			Label:    label,
			LinkedAt: pk.CreatedAt.Format(time.RFC3339),
		})
	}
	return methods, nil
}

// ──────────────────────────────────────────────────
// WebAuthn User adapter
// ──────────────────────────────────────────────────

// webAuthnUser adapts an authsome user.User to the webauthn.User interface.
type webAuthnUser struct {
	user        *user.User
	credentials []webauthn.Credential
}

// WebAuthnID returns the user's unique identifier as bytes.
func (u *webAuthnUser) WebAuthnID() []byte {
	return []byte(u.user.ID.String())
}

// WebAuthnName returns the user's name.
func (u *webAuthnUser) WebAuthnName() string {
	if u.user.Username != "" {
		return u.user.Username
	}
	return u.user.Email
}

// WebAuthnDisplayName returns the user's display name.
func (u *webAuthnUser) WebAuthnDisplayName() string {
	if n := u.user.Name(); n != "" {
		return n
	}
	return u.user.Email
}

// WebAuthnCredentials returns the user's registered credentials.
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// WebAuthnIcon returns an empty string (deprecated in WebAuthn spec).
func (u *webAuthnUser) WebAuthnIcon() string { return "" }

// toWebAuthnUser converts an authsome user to a WebAuthn user with credentials.
func (p *Plugin) toWebAuthnUser(ctx context.Context, u *user.User) *webAuthnUser {
	wau := &webAuthnUser{user: u}

	if p.store != nil {
		creds, _ := p.store.ListUserCredentials(ctx, u.ID) //nolint:errcheck // best-effort lookup
		for _, c := range creds {
			transports := make([]protocol.AuthenticatorTransport, 0, len(c.Transport))
			for _, t := range c.Transport {
				transports = append(transports, protocol.AuthenticatorTransport(t))
			}
			wau.credentials = append(wau.credentials, webauthn.Credential{
				ID:              c.CredentialID,
				PublicKey:       c.PublicKey,
				AttestationType: c.AttestationType,
				Transport:       transports,
				Authenticator: webauthn.Authenticator{
					AAGUID:    c.AAGUID,
					SignCount: c.SignCount,
				},
			})
		}
	}

	return wau
}

// toCredential converts a WebAuthn credential to a passkey Credential.
func toCredential(userID id.UserID, appID id.AppID, cred *webauthn.Credential, displayName string) *Credential {
	transports := make([]string, 0, len(cred.Transport))
	for _, t := range cred.Transport {
		transports = append(transports, string(t))
	}

	now := time.Now()
	return &Credential{
		ID:              id.NewPasskeyID(),
		UserID:          userID,
		AppID:           appID,
		CredentialID:    cred.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transport:       transports,
		SignCount:       cred.Authenticator.SignCount,
		AAGUID:          cred.Authenticator.AAGUID,
		DisplayName:     displayName,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// ──────────────────────────────────────────────────
// Observability helpers
// ──────────────────────────────────────────────────

// audit records an audit event to Chronicle (nil-safe).
func (p *Plugin) audit(ctx context.Context, action, resource, resourceID, actorID, tenant, outcome string) {
	if p.chronicle == nil {
		return
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
		Outcome:    outcome,
		Severity:   bridge.SeverityInfo,
		Category:   "auth",
	})
}

// relayEvent sends a webhook event to EventRelay (nil-safe).
func (p *Plugin) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if p.relay == nil {
		return
	}
	_ = p.relay.Send(ctx, &bridge.WebhookEvent{ //nolint:errcheck // best-effort webhook
		Type:     eventType,
		TenantID: tenantID,
		Data:     data,
	})
}

// emitHook fires a global hook event (nil-safe).
func (p *Plugin) emitHook(ctx context.Context, action, resource, resourceID, actorID, tenant string) {
	if p.hooks == nil {
		return
	}
	p.hooks.Emit(ctx, &hook.Event{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
	})
}
