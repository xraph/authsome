package authsome

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// MFATicketTTL is how long a partial-auth ticket remains valid between
// the moment the gate fires and the user submitting their second
// factor. Five minutes balances "user has time to find their
// authenticator" against "leaked ticket has a short window."
const MFATicketTTL = 5 * time.Minute

// ceremonyNamespaceMFATicket is the ceremony.Store key prefix used to
// distinguish MFA tickets from other ephemeral state.
const ceremonyNamespaceMFATicket = "mfa_ticket"

// IssueSessionRequest is the input to Engine.IssueSession. Every login
// path (password, social, magiclink, sso, phone, post-MFA-verify)
// populates one of these and hands it to the engine; the engine
// interposes the MFARequired gate before minting a session.
type IssueSessionRequest struct {
	User       *user.User
	AppID      id.AppID
	EnvID      id.EnvironmentID
	AuthMethod string
	IPAddress  string
	UserAgent  string

	// MFAJustVerified bypasses the MFARequired gate. Set this only
	// from the MFA challenge handler immediately after a code has
	// been validated against a ticket; bypassing without that
	// pairing is account hijack.
	MFAJustVerified bool

	// SessionTTL and RefreshTTL let the calling auth method shorten the
	// session it mints below the app's configured lifetime — a magic-link or
	// SSO session need not last as long as one from an interactive password
	// login. Zero means "use the app's configured value".
	//
	// These may only shorten. A plugin asking for longer than the app allows
	// is ignored, so a per-method setting can never be used to escape the
	// lifetime an operator set centrally.
	SessionTTL time.Duration
	RefreshTTL time.Duration

	// DPoPJKT binds the issued session to a client-held key (RFC 9449).
	// Empty issues an ordinary unbound session.
	DPoPJKT string
}

// IssueSessionResult is the gate's success output. On the
// MFA-needed path the gate returns (nil, *MFARequiredError) instead.
type IssueSessionResult struct {
	User    *user.User
	Session *session.Session
}

// MFARequiredError carries the ticket and available methods so the
// HTTP layer can render the 403 body without a second store lookup.
// Wraps account.ErrMFARequired so existing errors.Is checks keep
// working.
type MFARequiredError struct {
	Ticket           string
	AvailableMethods []string
}

// Error returns the underlying sentinel's message.
func (e *MFARequiredError) Error() string { return account.ErrMFARequired.Error() }

// Unwrap exposes the sentinel for errors.Is checks.
func (e *MFARequiredError) Unwrap() error { return account.ErrMFARequired }

// StatusCode lets the forge HTTP layer treat this error directly as a
// 403 without an explicit mapError call from every plugin handler.
// Plugin callbacks that bubble *MFARequiredError up unchanged still
// produce the canonical mfa_required envelope.
func (e *MFARequiredError) StatusCode() int { return 403 }

// ResponseBody returns the JSON envelope the API returns to clients.
// Mirrors codedHTTPError in api/helpers.go so plugins don't need to
// import the api package to render the same shape.
func (e *MFARequiredError) ResponseBody() any {
	methods := e.AvailableMethods
	if methods == nil {
		methods = []string{}
	}
	return map[string]any{
		"error":             account.ErrMFARequired.Error(),
		"code":              403,
		"type":              "mfa_required",
		"mfa_ticket":        e.Ticket,
		"available_methods": methods,
	}
}

// mfaTicketPayload is the JSON-encoded body persisted in ceremony.Store
// under the mfa_ticket namespace.
type mfaTicketPayload struct {
	UserID     string    `json:"user_id"`
	AppID      string    `json:"app_id"`
	EnvID      string    `json:"env_id"`
	AuthMethod string    `json:"auth_method"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	IssuedAt   time.Time `json:"issued_at"`
	// Attempts counts wrong codes submitted against this ticket.
	Attempts int `json:"attempts"`
}

// MaxMFATicketAttempts caps wrong codes against a single ticket before it is
// discarded and the user must sign in again.
//
// A ticket is partial authentication: the password leg already succeeded, so
// the only thing between the holder and a session is a 6-digit code. Route
// rate limiting is the other defence, but it returns no middleware at all when
// disabled by config — so the ticket has to bound its own attempts rather than
// rely on a limiter that may not be there.
const MaxMFATicketAttempts = 5

// MFATicketPayload is the publicly exposed shape of a loaded ticket. It
// mirrors the on-disk form but uses typed IDs so callers can use them
// directly with engine APIs.
type MFATicketPayload struct {
	UserID     id.UserID
	AppID      id.AppID
	EnvID      id.EnvironmentID
	AuthMethod string
	IPAddress  string
	UserAgent  string
	IssuedAt   time.Time
}

// IssueSession is the centralized session-mint chokepoint. Every login
// path goes through this function; the MFARequired gate has exactly
// one implementation, here.
//
// Returns (*IssueSessionResult, nil) on success.
// Returns (nil, *MFARequiredError) when the gate fires.
// Returns (nil, err) for any other failure.
func (e *Engine) IssueSession(ctx context.Context, req *IssueSessionRequest) (*IssueSessionResult, error) {
	if req == nil || req.User == nil {
		return nil, fmt.Errorf("authsome: IssueSession: nil request or user")
	}
	if req.AppID.IsNil() {
		req.AppID = req.User.AppID
	}

	// Resolve the default environment when the caller didn't supply one.
	// authsome_sessions.env_id is NOT NULL, so a session minted without an
	// env_id fails to persist. SignUp/SignIn already do this; callers that go
	// straight through IssueSession (email-verification auto-login, SSO) relied
	// on it happening here.
	if req.EnvID.IsNil() {
		if env, _ := e.GetDefaultEnvironment(ctx, req.AppID); env != nil { //nolint:errcheck // best-effort env lookup
			req.EnvID = env.ID
		}
	}

	// MFA gate. When the per-app config sets MFARequired and the
	// caller hasn't already verified via the challenge endpoint, the
	// gate fires regardless of whether the user has previously
	// enrolled MFA.
	//
	// "MFA required" is interpreted as "demand the second factor on
	// every login" — the modern MFA semantics every consumer expects.
	// The earlier inline check in service.go skipped the gate when
	// the user had a verified enrollment, which actually meant
	// "require enrollment at any point in the past," not "require
	// the second factor now." That weak semantics is what this
	// centralized gate replaces.
	//
	// First-time enrollment for a user who has none yet is a separate
	// flow (forced enrollment via partial-auth ticket); the challenge
	// handler returns "no MFA enrollment for user" in that case so the
	// UI can route to the enrollment surface.
	if !req.MFAJustVerified && e.mfaRequiredFor(ctx, req.AppID) {
		ticket, err := e.persistMFATicket(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("authsome: persist mfa ticket: %w", err)
		}
		return nil, &MFARequiredError{
			Ticket:           ticket,
			AvailableMethods: e.availableMFAMethods(ctx, req.User.ID),
		}
	}

	sessCfg := e.sessionConfigForApp(ctx, req.AppID, req.EnvID)
	applySessionTTLOverride(&sessCfg, req.SessionTTL, req.RefreshTTL)
	sess, err := e.newSession(req.AppID, req.User.ID, sessCfg, req.DPoPJKT)
	if err != nil {
		return nil, fmt.Errorf("authsome: build session: %w", err)
	}
	e.bindSessionToDevice(ctx, sess, req.AppID, req.EnvID, req.IPAddress, req.UserAgent)
	if hookErr := e.plugins.EmitBeforeSessionCreate(ctx, sess); hookErr != nil {
		return nil, fmt.Errorf("authsome: before session create: %w", hookErr)
	}
	if storeErr := e.store.CreateSession(ctx, sess); storeErr != nil {
		return nil, fmt.Errorf("authsome: persist session: %w", storeErr)
	}
	e.plugins.EmitAfterSessionCreate(ctx, sess)

	// Global hook bus parity with SignIn.
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionSignIn,
		Resource:   hook.ResourceSession,
		ResourceID: sess.ID.String(),
		ActorID:    req.User.ID.String(),
		Tenant:     req.AppID.String(),
		Metadata: map[string]string{
			"auth_method": req.AuthMethod,
			"session_id":  sess.ID.String(),
		},
	})

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "issue_session", "session",
		sess.ID.String(), req.User.ID.String(), req.AppID.String(), "auth",
		map[string]string{
			"auth_method":       req.AuthMethod,
			"mfa_just_verified": fmt.Sprintf("%v", req.MFAJustVerified),
		})

	return &IssueSessionResult{User: req.User, Session: sess}, nil
}

// mfaRequiredFor reports whether the per-app client config sets
// MFARequired = true for the given app.
func (e *Engine) mfaRequiredFor(ctx context.Context, appID id.AppID) bool {
	cfg, err := e.store.GetAppClientConfig(ctx, appID)
	if err != nil || cfg == nil || cfg.MFARequired == nil {
		return false
	}
	return *cfg.MFARequired
}

// availableMFAMethods reports which MFA methods the user could
// complete the challenge with. When the MFA plugin isn't registered,
// returns an empty slice rather than nil so downstream JSON serialises
// as `[]`.
func (e *Engine) availableMFAMethods(ctx context.Context, userID id.UserID) []string {
	out := []string{}
	type methodInspector interface {
		AvailableMethods(ctx context.Context, userID id.UserID) []string
	}
	for _, p := range e.plugins.Plugins() {
		if p.Name() != "mfa" {
			continue
		}
		if mi, ok := p.(methodInspector); ok {
			return mi.AvailableMethods(ctx, userID)
		}
		// Fallback: best-effort default since the plugin is loaded.
		return []string{"totp"}
	}
	return out
}

// applySessionTTLOverride narrows cfg to the caller's requested lifetimes.
//
// Shortening only, deliberately: the app-level config is the operator's
// ceiling, and a per-auth-method setting must not become a way around it. A
// zero or longer request leaves the configured value in place.
func applySessionTTLOverride(cfg *account.SessionConfig, sessionTTL, refreshTTL time.Duration) {
	if sessionTTL > 0 && (cfg.TokenTTL <= 0 || sessionTTL < cfg.TokenTTL) {
		cfg.TokenTTL = sessionTTL
	}
	if refreshTTL > 0 && (cfg.RefreshTokenTTL <= 0 || refreshTTL < cfg.RefreshTokenTTL) {
		cfg.RefreshTokenTTL = refreshTTL
	}
}

// persistMFATicket writes a ticket to ceremony.Store and returns the
// opaque ticket string the caller should hand back to the user.
func (e *Engine) persistMFATicket(ctx context.Context, req *IssueSessionRequest) (string, error) {
	store := e.ceremonyStoreOrFallback()
	if store == nil {
		return "", fmt.Errorf("ceremony store not configured")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	ticket := base64.RawURLEncoding.EncodeToString(raw)

	payload := mfaTicketPayload{
		UserID:     req.User.ID.String(),
		AppID:      req.AppID.String(),
		EnvID:      req.EnvID.String(),
		AuthMethod: req.AuthMethod,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		IssuedAt:   time.Now().UTC(),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if err := store.Set(ctx, ceremonyNamespaceMFATicket+":"+ticket, encoded, MFATicketTTL); err != nil {
		return "", err
	}
	return ticket, nil
}

// LoadMFATicket retrieves a ticket payload from the ceremony store
// without consuming it. Returns ceremony.ErrNotFound when the ticket
// is missing or expired.
func (e *Engine) LoadMFATicket(ctx context.Context, ticket string) (*MFATicketPayload, error) {
	store := e.ceremonyStoreOrFallback()
	if store == nil {
		return nil, fmt.Errorf("ceremony store not configured")
	}
	raw, err := store.Get(ctx, ceremonyNamespaceMFATicket+":"+ticket)
	if err != nil {
		return nil, err
	}
	var pl mfaTicketPayload
	if decodeErr := json.Unmarshal(raw, &pl); decodeErr != nil {
		return nil, fmt.Errorf("authsome: decode mfa ticket: %w", decodeErr)
	}
	uid, err := id.ParseUserID(pl.UserID)
	if err != nil {
		return nil, fmt.Errorf("authsome: invalid user id in ticket: %w", err)
	}
	out := &MFATicketPayload{
		UserID:     uid,
		AuthMethod: pl.AuthMethod,
		IPAddress:  pl.IPAddress,
		UserAgent:  pl.UserAgent,
		IssuedAt:   pl.IssuedAt,
	}
	if pl.AppID != "" {
		if a, err := id.ParseAppID(pl.AppID); err == nil {
			out.AppID = a
		}
	}
	if pl.EnvID != "" {
		if env, err := id.ParseEnvironmentID(pl.EnvID); err == nil {
			out.EnvID = env
		}
	}
	return out, nil
}

// ConsumeMFATicket deletes a ticket so it cannot be replayed.
func (e *Engine) ConsumeMFATicket(ctx context.Context, ticket string) error {
	store := e.ceremonyStoreOrFallback()
	if store == nil {
		return fmt.Errorf("ceremony store not configured")
	}
	return store.Delete(ctx, ceremonyNamespaceMFATicket+":"+ticket)
}

// FailMFATicket charges one wrong code against a ticket and reports whether
// that exhausted it. An exhausted ticket is deleted: the user must sign in
// again rather than keep guessing.
//
// The rewritten entry keeps the ticket's original deadline. Re-Setting a full
// MFATicketTTL would let a caller hold a ticket open indefinitely by
// submitting wrong codes, extending the very window the counter bounds.
//
// Errors are returned for the caller to log, not to surface: a failed write
// must not turn a wrong-code answer into a distinguishable 500, which would
// itself confirm the code was wrong.
func (e *Engine) FailMFATicket(ctx context.Context, ticket string) (exhausted bool, err error) {
	store := e.ceremonyStoreOrFallback()
	if store == nil {
		return false, fmt.Errorf("ceremony store not configured")
	}
	key := ceremonyNamespaceMFATicket + ":" + ticket

	raw, err := store.Get(ctx, key)
	if err != nil {
		return false, err
	}
	var pl mfaTicketPayload
	if decodeErr := json.Unmarshal(raw, &pl); decodeErr != nil {
		// Undecodable ticket is unusable — drop it.
		_ = store.Delete(ctx, key) //nolint:errcheck // best-effort
		return true, fmt.Errorf("authsome: decode mfa ticket: %w", decodeErr)
	}

	pl.Attempts++
	remaining := time.Until(pl.IssuedAt.Add(MFATicketTTL))
	if pl.Attempts >= MaxMFATicketAttempts || remaining <= 0 {
		return true, store.Delete(ctx, key)
	}

	encoded, err := json.Marshal(pl)
	if err != nil {
		_ = store.Delete(ctx, key) //nolint:errcheck // best-effort
		return true, err
	}
	if setErr := store.Set(ctx, key, encoded, remaining); setErr != nil {
		// Can't persist the count — drop the ticket rather than leave it
		// standing with the attempt uncounted.
		_ = store.Delete(ctx, key) //nolint:errcheck // best-effort
		return true, setErr
	}
	return false, nil
}

// IsMFATicketNotFound reports whether err indicates a missing or
// expired ticket. Hides the ceremony package from callers that don't
// otherwise depend on it.
func IsMFATicketNotFound(err error) bool {
	return errors.Is(err, ceremony.ErrNotFound)
}

// ceremonyStoreOrFallback returns the configured ceremony store. The store is
// always non-nil: NewEngine allocates an in-memory fallback at construction
// time when no store was configured, so this accessor never mutates engine
// state on a request goroutine (avoiding a data race between concurrent
// MFA-gated logins).
func (e *Engine) ceremonyStoreOrFallback() ceremony.Store {
	return e.ceremonyStore
}
