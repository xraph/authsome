package retention

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	_ plugin.AfterSignOut    = (*Plugin)(nil)
	_ plugin.AfterUserUpdate = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Exported hooks: thin unwrappers over the internal helpers below.
//
// Every one of these returns nil unconditionally. A retention failure must
// never fail a login, a signup, a signout or a profile update; the store
// error (if any) is logged inside enqueueFor and swallowed here.
// ──────────────────────────────────────────────────

// OnAfterSignUp implements plugin.AfterSignUp.
func (p *Plugin) OnAfterSignUp(ctx context.Context, u *user.User, _ *session.Session) error {
	_ = p.afterSignUpFor(ctx, u.AppID, u.EnvID, u.ID) //nolint:errcheck // afterSignUpFor never returns a non-nil error; see its doc comment
	return nil
}

// OnAfterSignIn implements plugin.AfterSignIn.
func (p *Plugin) OnAfterSignIn(ctx context.Context, u *user.User, _ *session.Session) error {
	_ = p.afterSignInFor(ctx, u.AppID, u.EnvID, u.ID) //nolint:errcheck // afterSignInFor never returns a non-nil error; see its doc comment
	return nil
}

// OnAfterSignOut implements plugin.AfterSignOut.
//
// The interface only carries the session id:
//
//	OnAfterSignOut(ctx context.Context, sessionID id.SessionID) error
//
// That is not enough to build a Job, which needs AppID, EnvID and UserID.
// Nor does the gap have a store-read workaround: engine.SignOut (service.go)
// calls e.store.DeleteSession before it emits this hook, so the session row
// is already gone by the time OnAfterSignOut runs, and a read would find
// nothing even if this design allowed one.
//
// The one avenue that does carry those ids is request-scoped context set by
// the auth middleware earlier in the same request
// (middleware.AppIDFrom/UserIDFrom/EnvIDFrom - api.handleSignOut itself
// reads middleware.SessionIDFrom(ctx) for this exact sessionID). Reaching
// for that from here would couple this plugin to the HTTP middleware layer,
// which no plugin does today and which would silently stop covering any
// signout path that does not go through that specific middleware (a
// programmatic engine.SignOut call, a different transport). That is a real
// design decision, not a one-line fix, so it is left to Task 8 or a
// deliberate signature change rather than wired in here; see the task
// report for the full writeup. For now this hook is a true no-op.
//
// afterSignOutFor below is implemented and unit-tested against plain ids so
// wiring it in later, once the ids are available, is a one-line change.
func (p *Plugin) OnAfterSignOut(_ context.Context, _ id.SessionID) error {
	return nil
}

// OnAfterUserUpdate implements plugin.AfterUserUpdate.
func (p *Plugin) OnAfterUserUpdate(ctx context.Context, u *user.User) error {
	_ = p.afterUserUpdateFor(ctx, u.AppID, u.EnvID, u.ID) //nolint:errcheck // afterUserUpdateFor never returns a non-nil error; see its doc comment
	return nil
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
func (p *Plugin) afterSignUpFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID, userID id.UserID) error { //nolint:unparam // error return matches hooks_test.go's require.NoError(t, p.xxxFor(...)) contract; every hook must swallow its own failures
	p.enqueueFor(ctx, appID, envID, userID, KindContactUpsert, "signed_up")
	p.enqueueFor(ctx, appID, envID, userID, KindActivityLog, "signed_up")
	return nil
}

// afterSignInFor enqueues a login activity only. It deliberately does not
// look up whether a contact ref exists first: that would put a read on the
// login path. The worker's ensureRef upserts a missing contact as part of
// delivering the activity job, which also makes this self-healing - a
// contact deleted upstream, or a provider enabled after the user already
// existed, both recover on the next login with no backfill step.
func (p *Plugin) afterSignInFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID, userID id.UserID) error { //nolint:unparam // error return matches hooks_test.go's require.NoError(t, p.xxxFor(...)) contract; every hook must swallow its own failures
	p.enqueueFor(ctx, appID, envID, userID, KindActivityLog, "logged_in")
	return nil
}

// afterSignOutFor enqueues a logout activity, gated on SettingTrackSignOut.
//
// It reads the setting's own static code-default rather than going through
// p.settingsMgr: per-app settings resolution is a Task 8 concern, and a
// settings.Manager call from a hook would put a read on the same path this
// whole design exists to keep read-free. Until Task 8 wires per-app
// resolution in, this always sees the code default (false), so with no
// config override sign-out tracking is effectively off - that is the
// tension to flag, not a bug in this function.
//
// Not yet reachable from OnAfterSignOut; see that method's doc comment.
func (p *Plugin) afterSignOutFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID, userID id.UserID) error { //nolint:unparam // error return matches hooks_test.go's require.NoError(t, p.xxxFor(...)) contract; every hook must swallow its own failures
	if !trackSignOutDefault() {
		return nil
	}
	p.enqueueFor(ctx, appID, envID, userID, KindActivityLog, "logged_out")
	return nil
}

// afterUserUpdateFor enqueues a contact upsert so a changed email or name
// reaches the CRM. loadContact (plugin.go) re-reads the user at delivery
// time, so the worker always sends the current profile regardless of how
// stale this enqueue-time snapshot is by the time it is delivered.
func (p *Plugin) afterUserUpdateFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID, userID id.UserID) error { //nolint:unparam // error return matches hooks_test.go's require.NoError(t, p.xxxFor(...)) contract; every hook must swallow its own failures
	p.enqueueFor(ctx, appID, envID, userID, KindContactUpsert, "profile_updated")
	return nil
}

// trackSignOutDefault decodes SettingTrackSignOut's static code default.
// This is a decode of an in-process constant, not a settings read: no
// manager, no per-app resolution, no I/O.
func trackSignOutDefault() bool {
	var v bool
	_ = json.Unmarshal(SettingTrackSignOut.Def.Default, &v) //nolint:errcheck // Default is produced by settings.Define's own json.Marshal of a bool; it cannot fail to decode
	return v
}

// enqueueFor writes one job per configured provider. It is the only thing a
// hook does. No reads: a lookup here would put a query on the login path, and
// the whole point of the outbox is that a login writes one row and gets out.
func (p *Plugin) enqueueFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, kind, activityType string) {
	if p.store == nil || len(p.providers) == 0 {
		return
	}
	now := time.Now()
	stamp := now.UTC().Format(time.RFC3339Nano)
	for name := range p.providers {
		j := &Job{
			ID: id.NewRetentionJobID(), AppID: appID, EnvID: envID, UserID: userID,
			Provider: name, Kind: kind,
			Payload:        map[string]string{"activity_type": activityType},
			IdempotencyKey: idempotencyKey(name, userID.String(), kind+":"+activityType, stamp),
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
// so a hook that fires twice for the same event enqueues once.
func idempotencyKey(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
