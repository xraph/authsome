package authsome

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/dpop"
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
	// DPoPJKT is the thumbprint the first factor proved. Public by
	// construction (it is a hash of a public key), so it needs no more
	// protection here than the ticket itself already has.
	DPoPJKT string `json:"dpop_jkt,omitempty"`
	// Attempts is kept for tickets written before the counter moved to its
	// own ceremony key; it is no longer read.
	Attempts int `json:"attempts,omitempty"`
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

	// DPoPJKT is the thumbprint proved at sign-in, carried across the
	// ceremony so the session minted after the second factor keeps the
	// binding the first factor established.
	DPoPJKT string
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
	// Every sign-in path mints here, so this is the one place a ban has to
	// hold for passkeys, magic links, SSO and social alike.
	if req.User.IsBanned(time.Now()) {
		return nil, account.ErrUserBanned
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

	// DPoP gate. An app on mode=required is stating that every session
	// under it is bound; a caller that resolved no thumbprint cannot
	// satisfy that, so it is refused rather than quietly granted a
	// session the mandate will never apply to.
	//
	// Ahead of the MFA gate deliberately. Both refuse the same request,
	// but the MFA gate refuses by persisting a ticket first, and there is
	// no reason to spend a ceremony-store write on a login that cannot
	// complete however the second factor goes.
	if req.DPoPJKT == "" && e.DPoPModeForApp(ctx, req.AppID) == dpop.ModeRequired {
		return nil, &DPoPRequiredError{}
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
	e.enforceSessionCap(ctx, req.User.ID, sessCfg.MaxActiveSessions)
	if storeErr := e.store.CreateSession(ctx, sess); storeErr != nil {
		return nil, fmt.Errorf("authsome: persist session: %w", storeErr)
	}
	e.plugins.EmitAfterSessionCreate(ctx, sess)

	// Every sign-in path mints here, so this is the one sign-in record on
	// the audit trail.
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionSignIn,
		Resource:   hook.ResourceSession,
		ResourceID: sess.ID.String(),
		ActorID:    req.User.ID.String(),
		Tenant:     req.AppID.String(),
		Category:   "auth",
		SessionID:  sess.ID.String(),
		Metadata: map[string]string{
			"auth_method":       req.AuthMethod,
			"session_id":        sess.ID.String(),
			"mfa_just_verified": fmt.Sprintf("%v", req.MFAJustVerified),
		},
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
		DPoPJKT:    req.DPoPJKT,
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
		DPoPJKT:    pl.DPoPJKT,
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

	remaining := time.Until(pl.IssuedAt.Add(MFATicketTTL))
	if remaining <= 0 {
		return true, store.Delete(ctx, key)
	}

	// One atomic increment per wrong code, bounded by the ticket's own
	// deadline. Rewriting the ticket with a bumped count let two replicas
	// each count one guess as one and let a guess slip in uncounted between
	// the read and the write.
	attempts, incErr := store.Increment(ctx, key+":attempts", remaining)
	if incErr != nil {
		// Can't count the attempt — drop the ticket rather than leave it
		// standing with the guess uncounted.
		_ = store.Delete(ctx, key) //nolint:errcheck // best-effort
		return true, incErr
	}
	if attempts >= MaxMFATicketAttempts {
		_ = store.Delete(ctx, key+":attempts") //nolint:errcheck // best-effort
		return true, store.Delete(ctx, key)
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

// enforceSessionCap makes room for one more session under maxActive by
// revoking the user's oldest sessions, so a cap of N is N, not N plus
// however many sign-ins nobody counted. Each eviction is audited like any
// other revocation, with the cap as the reason. A zero or negative cap is
// no cap. Failures are logged and do not block the sign-in: the cap is a
// hygiene control, and a listing error must not lock a user out.
func (e *Engine) enforceSessionCap(ctx context.Context, userID id.UserID, maxActive int) {
	if maxActive <= 0 {
		return
	}
	sessions, err := e.store.ListUserSessions(ctx, userID)
	if err != nil {
		e.logger.Warn("authsome: session cap: list sessions", log.String("error", err.Error()))
		return
	}
	now := time.Now()
	live := sessions[:0]
	for _, s := range sessions {
		if now.Before(s.ExpiresAt) || now.Before(s.RefreshTokenExpiresAt) {
			live = append(live, s)
		}
	}
	if len(live) < maxActive {
		return
	}
	sort.Slice(live, func(i, j int) bool { return live[i].CreatedAt.Before(live[j].CreatedAt) })
	for _, s := range live[:len(live)-maxActive+1] {
		if delErr := e.store.DeleteSession(ctx, s.ID); delErr != nil {
			e.logger.Warn("authsome: session cap: revoke oldest", log.String("session_id", s.ID.String()), log.String("error", delErr.Error()))
			continue
		}
		e.plugins.EmitAfterSessionRevoke(ctx, s.ID)
		e.hooks.Emit(ctx, &hook.Event{
			Action:     hook.ActionSessionRevoke,
			Resource:   hook.ResourceSession,
			ResourceID: s.ID.String(),
			ActorID:    userID.String(),
			Tenant:     s.AppID.String(),
			Metadata:   map[string]string{"reason": "max_active_sessions"},
		})
		e.relayEvent(ctx, "session.revoked", s.AppID.String(), map[string]string{
			"user_id": userID.String(), "session_id": s.ID.String(), "reason": "max_active_sessions",
		})
	}
}
