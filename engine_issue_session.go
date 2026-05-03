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
}

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

	sess, err := e.newSession(req.AppID, req.User.ID, e.sessionConfigForApp(ctx, req.AppID, req.EnvID))
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
	if err := json.Unmarshal(raw, &pl); err != nil {
		return nil, fmt.Errorf("authsome: decode mfa ticket: %w", err)
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

// IsMFATicketNotFound reports whether err indicates a missing or
// expired ticket. Hides the ceremony package from callers that don't
// otherwise depend on it.
func IsMFATicketNotFound(err error) bool {
	return errors.Is(err, ceremony.ErrNotFound)
}

// ceremonyStoreOrFallback returns the configured ceremony store,
// lazily allocating a process-wide in-memory store if none was
// configured. The lazy allocation is single-use (sync.Once-like via
// nil check + assignment under the engine's existing serial
// initialisation contract) so two IssueSession calls share a backing
// map and tickets actually persist between calls.
func (e *Engine) ceremonyStoreOrFallback() ceremony.Store {
	if e.ceremonyStore == nil {
		e.ceremonyStore = ceremony.NewMemory()
	}
	return e.ceremonyStore
}
