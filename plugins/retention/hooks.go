package retention

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks for the hooks this file implements.
var (
	_ plugin.AfterSignUp     = (*Plugin)(nil)
	_ plugin.AfterSignIn     = (*Plugin)(nil)
	_ plugin.AfterUserUpdate = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Exported hooks: thin unwrappers over the internal helpers below.
//
// Every one of these returns nil unconditionally. A retention failure must
// never fail a login, a signup or a profile update; the store error (if any)
// is logged inside enqueueFor and swallowed here.
// ──────────────────────────────────────────────────

// OnAfterSignUp implements plugin.AfterSignUp.
func (p *Plugin) OnAfterSignUp(ctx context.Context, u *user.User, s *session.Session) error {
	_ = p.afterSignUpFor(ctx, u.AppID, u.EnvID, u.ID, signupSigninEventID(s)) //nolint:errcheck // afterSignUpFor never returns a non-nil error; see its doc comment
	return nil
}

// OnAfterSignIn implements plugin.AfterSignIn.
func (p *Plugin) OnAfterSignIn(ctx context.Context, u *user.User, s *session.Session) error {
	_ = p.afterSignInFor(ctx, u.AppID, u.EnvID, u.ID, signupSigninEventID(s)) //nolint:errcheck // afterSignInFor never returns a non-nil error; see its doc comment
	return nil
}

// Sign-out is deliberately not tracked. OnAfterSignOut carries only a session
// id, and the engine deletes the session row before it emits the hook
// (service.go:423 then :428), so there is no way to reach the user, app or
// environment the activity would belong to. Reading the app id off the request
// context would cover HTTP sign-outs and silently miss every other path.
// Unblocking this needs the hook to carry the user, or the emit to move ahead
// of the delete; both are engine changes.

// OnAfterUserUpdate implements plugin.AfterUserUpdate.
func (p *Plugin) OnAfterUserUpdate(ctx context.Context, u *user.User) error {
	// UpdatedAt is stable across two dispatches of the same edit and changes
	// on the next one, which is exactly the shape idempotencyKey needs.
	eventID := u.UpdatedAt.UTC().Format(time.RFC3339Nano)
	_ = p.afterUserUpdateFor(ctx, u.AppID, u.EnvID, u.ID, eventID) //nolint:errcheck // afterUserUpdateFor never returns a non-nil error; see its doc comment
	return nil
}

// signupSigninEventID derives the idempotency event id for the signup/signin
// hooks from the session the engine just issued. A session id is stable
// across two dispatches of the same signup or signin and distinct across
// different ones, which is what idempotencyKey needs; a value read from the
// clock at call time is neither.
//
// If s is nil there is nothing stable to key on, so this falls back to a
// fresh timestamp and accepts losing dedup for that one call. Enqueuing the
// same event twice is recoverable, the worker's ensureRef and the CRM's own
// upsert-by-email both tolerate it; silently refusing to enqueue because
// there is no session to key on is not.
func signupSigninEventID(s *session.Session) string {
	if s != nil {
		return s.ID.String()
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// ──────────────────────────────────────────────────
// Internal helpers: take plain ids so tests do not have to build a
// forge.Context or a full user.User/session.Session.
// ──────────────────────────────────────────────────

// afterSignUpFor enqueues a contact upsert and a signup activity. Both are
// queued from the same event because a brand new user has no contact ref
// yet; the worker's ensureRef creates it as part of delivering the activity
// job, but sending the upsert too means the contact record is populated
// (name, email) rather than created bare and back-filled later.
func (p *Plugin) afterSignUpFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID, userID id.UserID, eventID string) error { //nolint:unparam // error return matches hooks_test.go's require.NoError(t, p.xxxFor(...)) contract; every hook must swallow its own failures
	p.enqueueFor(ctx, appID, envID, userID, KindContactUpsert, "signed_up", eventID)
	p.enqueueFor(ctx, appID, envID, userID, KindActivityLog, "signed_up", eventID)
	return nil
}

// afterSignInFor enqueues a login activity only. It deliberately does not
// look up whether a contact ref exists first: that would put a read on the
// login path. The worker's ensureRef upserts a missing contact as part of
// delivering the activity job, which also makes this self-healing - a
// contact deleted upstream, or a provider enabled after the user already
// existed, both recover on the next login with no backfill step.
func (p *Plugin) afterSignInFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID, userID id.UserID, eventID string) error { //nolint:unparam // error return matches hooks_test.go's require.NoError(t, p.xxxFor(...)) contract; every hook must swallow its own failures
	p.enqueueFor(ctx, appID, envID, userID, KindActivityLog, "logged_in", eventID)
	return nil
}

// afterUserUpdateFor enqueues a contact upsert so a changed email or name
// reaches the CRM. loadContact (plugin.go) re-reads the user at delivery
// time, so the worker always sends the current profile regardless of how
// stale this enqueue-time snapshot is by the time it is delivered.
func (p *Plugin) afterUserUpdateFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID, userID id.UserID, eventID string) error { //nolint:unparam // error return matches hooks_test.go's require.NoError(t, p.xxxFor(...)) contract; every hook must swallow its own failures
	p.enqueueFor(ctx, appID, envID, userID, KindContactUpsert, "profile_updated", eventID)
	return nil
}

// enqueueFor writes one job per configured provider. It is the only thing a
// hook does. No reads: a lookup here would put a query on the login path, and
// the whole point of the outbox is that a login writes one row and gets out.
//
// eventID must be stable for one logical event and different across events.
// The callers pass the session id for sign-up and sign-in (one session is one
// login) and the user's UpdatedAt for a profile change. Deriving it from
// time.Now() here instead would defeat the whole mechanism: two dispatches of
// the same hook would stamp two different nanosecond values, hash to two keys,
// and enqueue the work twice.
func (p *Plugin) enqueueFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, kind, activityType, eventID string) {
	providers := p.providers.Load()
	if p.store == nil || len(providers) == 0 {
		return
	}
	now := time.Now()
	for name := range providers {
		j := &Job{
			ID: id.NewRetentionJobID(), AppID: appID, EnvID: envID, UserID: userID,
			Provider: name, Kind: kind,
			Payload:        map[string]string{"activity_type": activityType},
			IdempotencyKey: idempotencyKey(name, userID.String(), kind+":"+activityType, eventID),
			State:          StatePending,
			NextAttemptAt:  now,
			CreatedAt:      now,
		}
		if err := p.store.Enqueue(ctx, j); err != nil {
			// Swallowed on purpose. This runs in the login path and a CRM
			// bookkeeping miss must never turn into a failed sign-in.
			p.logger.Warn("retention: enqueue failed",
				log.String("provider", name),
				log.String("kind", kind),
				log.String("error", err.Error()))
		}
	}
}

// idempotencyKey is a stable hash of the parts that make one delivery unique,
// so a hook that fires twice for the same event enqueues once. Every part must
// be stable across those two firings; anything derived from the clock at call
// time is not.
func idempotencyKey(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
