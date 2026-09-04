package retention

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// storeFactory builds an empty store for one subtest.
type storeFactory func(t *testing.T) Store

func newJob(appID id.AppID, userID id.UserID, kind, key string, due time.Time) *Job {
	return &Job{
		ID:             id.NewRetentionJobID(),
		AppID:          appID,
		UserID:         userID,
		Provider:       "fake",
		Kind:           kind,
		Payload:        map[string]string{"email": "a@example.com"},
		IdempotencyKey: key,
		State:          StatePending,
		NextAttemptAt:  due,
		CreatedAt:      due,
	}
}

// runStoreConformance is the single suite every backend must satisfy.
func runStoreConformance(t *testing.T, newStore storeFactory) {
	ctx := context.Background()
	base := time.Now().UTC().Truncate(time.Second)

	t.Run("enqueue then claim", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindContactUpsert, "k1", base)
		require.NoError(t, s.Enqueue(ctx, j))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, j.ID.String(), got[0].ID.String())
		assert.Equal(t, StateInFlight, got[0].State)
		assert.Equal(t, "a@example.com", got[0].Payload["email"])
	})

	t.Run("enqueue is idempotent on key", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "dupe", base)))
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "dupe", base)))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		assert.Len(t, got, 1, "the same idempotency key must not enqueue twice")
	})

	t.Run("claim skips jobs not yet due", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "later", base.Add(time.Hour))))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("claim does not re-take a live lease", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "lease", base)))

		first, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.Len(t, first, 1)

		second, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(30*time.Second))
		require.NoError(t, err)
		assert.Empty(t, second, "a job under a live lease must not be claimed twice")
	})

	t.Run("claim reclaims an expired lease", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "expire", base)))

		_, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)

		again, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(2*time.Minute))
		require.NoError(t, err)
		require.Len(t, again, 1, "a crashed worker's job must come back after the lease expires")
	})

	t.Run("mark done removes the job from the queue", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindActivityLog, "done", base)
		require.NoError(t, s.Enqueue(ctx, j))
		_, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.NoError(t, s.MarkDone(ctx, j.ID, base))

		stored, err := s.GetJob(ctx, j.ID)
		require.NoError(t, err)
		assert.Equal(t, StateDone, stored.State)

		got, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(time.Hour))
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("mark retry increments attempts and defers", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindActivityLog, "retry", base)
		require.NoError(t, s.Enqueue(ctx, j))
		_, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.NoError(t, s.MarkRetry(ctx, j.ID, base.Add(10*time.Second), "429"))

		stored, err := s.GetJob(ctx, j.ID)
		require.NoError(t, err)
		assert.Equal(t, StatePending, stored.State)
		assert.Equal(t, 1, stored.Attempts)
		assert.Equal(t, "429", stored.LastError)

		got, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(11*time.Second))
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, 1, got[0].Attempts, "attempts must survive the reclaim")
	})

	t.Run("dead and suppressed are terminal and distinguishable", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		dead := newJob(appID, userID, KindActivityLog, "dead", base)
		supp := newJob(appID, userID, KindActivityLog, "supp", base)
		require.NoError(t, s.Enqueue(ctx, dead))
		require.NoError(t, s.Enqueue(ctx, supp))
		require.NoError(t, s.MarkDead(ctx, dead.ID, "400 invalid email"))
		require.NoError(t, s.MarkSuppressed(ctx, supp.ID, "no marketing consent"))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(time.Hour))
		require.NoError(t, err)
		assert.Empty(t, got, "terminal jobs must never be claimed again")

		listed, err := s.ListDead(ctx, appID, 10)
		require.NoError(t, err)
		require.Len(t, listed, 1, "suppressed must not show up as dead-lettered")
		assert.Equal(t, dead.ID.String(), listed[0].ID.String())

		storedSupp, err := s.GetJob(ctx, supp.ID)
		require.NoError(t, err)
		assert.Equal(t, StateSuppressed, storedSupp.State)
		assert.Equal(t, "no marketing consent", storedSupp.LastError)
	})

	t.Run("claim honours limit and takes oldest first", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "a", base.Add(-2*time.Hour))))
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "b", base.Add(-time.Hour))))
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "c", base)))

		got, err := s.ClaimDue(ctx, 2, time.Minute, base)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "a", got[0].IdempotencyKey)
		assert.Equal(t, "b", got[1].IdempotencyKey)
	})

	t.Run("claim with a non-positive limit takes everything due", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		for _, key := range []string{"n1", "n2", "n3"} {
			require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, key, base)))
		}

		got, err := s.ClaimDue(ctx, 0, time.Minute, base)
		require.NoError(t, err)
		assert.Len(t, got, 3,
			"limit <= 0 means no limit; binding it into a LIMIT clause would claim nothing")
	})

	t.Run("marked deferred returns to pending without spending an attempt", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindActivityLog, "deferred", base)
		require.NoError(t, s.Enqueue(ctx, j))
		_, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.NoError(t, s.MarkRetry(ctx, j.ID, base, "503"))
		_, err = s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)

		require.NoError(t, s.MarkDeferred(ctx, j.ID, base.Add(10*time.Second), "delivery disabled"))

		stored, err := s.GetJob(ctx, j.ID)
		require.NoError(t, err)
		assert.Equal(t, StatePending, stored.State)
		assert.Equal(t, 1, stored.Attempts,
			"the attempt from the earlier real failure stands; deferring adds nothing")
		assert.Equal(t, "delivery disabled", stored.LastError)

		got, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(11*time.Second))
		require.NoError(t, err)
		require.Len(t, got, 1, "a deferred job comes back once its new due time passes")
	})

	// hooks.go passes user.EnvID straight through, and that is never empty in
	// a multi-environment deployment, so the non-empty parse branch in every
	// mapper needs a backend that actually runs it.
	t.Run("job round trips a non-empty env id", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		envID := id.NewEnvironmentID()
		j := newJob(appID, userID, KindActivityLog, "envjob", base)
		j.EnvID = envID
		require.NoError(t, s.Enqueue(ctx, j))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, envID.String(), got[0].EnvID.String(),
			"a claimed job must carry its environment, not a parse failure")

		stored, err := s.GetJob(ctx, j.ID)
		require.NoError(t, err)
		assert.Equal(t, envID.String(), stored.EnvID.String())
	})

	t.Run("ref round trips a non-empty env id and is keyed on it", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		envA, envB := id.NewEnvironmentID(), id.NewEnvironmentID()

		require.NoError(t, s.PutRef(ctx, &ContactRef{
			ID: id.NewRetentionRefID(), AppID: appID, EnvID: envA, UserID: userID,
			Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "env-a", SyncedAt: base,
		}))

		got, err := s.GetRef(ctx, appID, envA, userID, "hubspot")
		require.NoError(t, err)
		assert.Equal(t, envA.String(), got.EnvID.String())
		assert.Equal(t, "env-a", got.RemoteID)

		_, err = s.GetRef(ctx, appID, envB, userID, "hubspot")
		assert.ErrorIs(t, err, ErrNotFound,
			"the same user in another environment is a different contact")

		refs, err := s.ListRefsForUser(ctx, userID)
		require.NoError(t, err)
		require.Len(t, refs, 1)
		assert.Equal(t, envA.String(), refs[0].EnvID.String(),
			"the export must not lose the environment either")
	})

	t.Run("get job missing returns ErrNotFound", func(t *testing.T) {
		s := newStore(t)
		_, err := s.GetJob(ctx, id.NewRetentionJobID())
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("ref put get delete round trip", func(t *testing.T) {
		s := newStore(t)
		appID, envID, userID := id.NewAppID(), id.EnvironmentID{}, id.NewUserID()

		_, err := s.GetRef(ctx, appID, envID, userID, "hubspot")
		assert.ErrorIs(t, err, ErrNotFound)

		r := &ContactRef{
			ID: id.NewRetentionRefID(), AppID: appID, EnvID: envID, UserID: userID,
			Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "501", SyncedAt: base,
		}
		require.NoError(t, s.PutRef(ctx, r))

		got, err := s.GetRef(ctx, appID, envID, userID, "hubspot")
		require.NoError(t, err)
		assert.Equal(t, "501", got.RemoteID)
		assert.Equal(t, RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "501"}, got.Ref())

		r.RemoteID = "777"
		require.NoError(t, s.PutRef(ctx, r), "PutRef must upsert, not conflict")
		got, err = s.GetRef(ctx, appID, envID, userID, "hubspot")
		require.NoError(t, err)
		assert.Equal(t, "777", got.RemoteID)

		require.NoError(t, s.DeleteRef(ctx, appID, envID, userID, "hubspot"))
		_, err = s.GetRef(ctx, appID, envID, userID, "hubspot")
		assert.ErrorIs(t, err, ErrNotFound)
		require.NoError(t, s.DeleteRef(ctx, appID, envID, userID, "hubspot"),
			"deleting an absent ref is not an error")
	})

	t.Run("refs are isolated per provider", func(t *testing.T) {
		s := newStore(t)
		appID, envID, userID := id.NewAppID(), id.EnvironmentID{}, id.NewUserID()
		require.NoError(t, s.PutRef(ctx, &ContactRef{
			ID: id.NewRetentionRefID(), AppID: appID, EnvID: envID, UserID: userID,
			Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "1", SyncedAt: base,
		}))
		_, err := s.GetRef(ctx, appID, envID, userID, "generic")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("list refs for user spans apps and providers", func(t *testing.T) {
		s := newStore(t)
		userID := id.NewUserID()
		appA, appB := id.NewAppID(), id.NewAppID()
		envID := id.EnvironmentID{}
		other := id.NewUserID()

		empty, err := s.ListRefsForUser(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, empty, "a user with no refs gets an empty slice, not an error")

		require.NoError(t, s.PutRef(ctx, &ContactRef{
			ID: id.NewRetentionRefID(), AppID: appA, EnvID: envID, UserID: userID,
			Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "1", SyncedAt: base,
		}))
		require.NoError(t, s.PutRef(ctx, &ContactRef{
			ID: id.NewRetentionRefID(), AppID: appB, EnvID: envID, UserID: userID,
			Provider: "generic", RemoteObjectType: "contact", RemoteID: "2", SyncedAt: base,
		}))
		// A different user's ref must never leak into this user's export.
		require.NoError(t, s.PutRef(ctx, &ContactRef{
			ID: id.NewRetentionRefID(), AppID: appA, EnvID: envID, UserID: other,
			Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "3", SyncedAt: base,
		}))

		got, err := s.ListRefsForUser(ctx, userID)
		require.NoError(t, err)
		require.Len(t, got, 2, "a data-subject export covers the person, not one app's view of them")
		providers := []string{got[0].Provider, got[1].Provider}
		assert.ElementsMatch(t, []string{"hubspot", "generic"}, providers)
	})

	// ── Reclaimed ──────────────────────────────────────────────────
	//
	// The only signal anywhere that a job may already have been delivered
	// once. A provider call that succeeded and a MarkDone that then failed
	// leaves the row in_flight with Attempts untouched, because nothing got
	// to record a failure. Every backend has to compute this from the state
	// the row was in before the claim flipped it.

	t.Run("a job claimed from an expired lease is flagged as reclaimed", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "reclaim-flag", base)))

		first, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.Len(t, first, 1)
		assert.False(t, first[0].Reclaimed, "the first time out is not a redelivery")
		assert.False(t, first[0].Redelivered())

		again, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(2*time.Minute))
		require.NoError(t, err)
		require.Len(t, again, 1)
		assert.True(t, again[0].Reclaimed,
			"this row was already out once; the provider has to be told")
		assert.Equal(t, 0, again[0].Attempts,
			"nothing recorded a failure, so the attempt count cannot carry this")
		assert.True(t, again[0].Redelivered())

		// It describes the claim, not the row. A later read must not
		// inherit it.
		stored, err := s.GetJob(ctx, again[0].ID)
		require.NoError(t, err)
		assert.False(t, stored.Reclaimed, "Reclaimed is not a column")
	})

	t.Run("a retried job comes back pending rather than reclaimed", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindActivityLog, "retry-flag", base)
		require.NoError(t, s.Enqueue(ctx, j))
		_, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.NoError(t, s.MarkRetry(ctx, j.ID, base, "503"))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(time.Second))
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.False(t, got[0].Reclaimed,
			"MarkRetry cleared the lease, so this is a plain pending claim")
		assert.Equal(t, 1, got[0].Attempts)
		assert.True(t, got[0].Redelivered(),
			"the attempt count carries it instead; a failed attempt is no proof the CRM did nothing")
	})

	// ── PurgeTerminal ──────────────────────────────────────────────
	//
	// Nothing pruned the outbox before this, so every login ever served sat
	// in the table forever. The cases below pin the three properties that
	// make pruning safe: age decides, state decides harder, and a
	// non-terminal row is out of reach at any age.

	t.Run("purge removes done rows past the window and keeps the rest", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()

		old := newJob(appID, userID, KindActivityLog, "old-done", base.Add(-60*24*time.Hour))
		fresh := newJob(appID, userID, KindActivityLog, "fresh-done", base.Add(-24*time.Hour))
		require.NoError(t, s.Enqueue(ctx, old))
		require.NoError(t, s.Enqueue(ctx, fresh))
		require.NoError(t, s.MarkDone(ctx, old.ID, base))
		require.NoError(t, s.MarkDone(ctx, fresh.ID, base))

		removed, err := s.PurgeTerminal(ctx, base.Add(-30*24*time.Hour), base.Add(-180*24*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 1, removed)

		_, err = s.GetJob(ctx, old.ID)
		assert.ErrorIs(t, err, ErrNotFound, "a done row past the window must be gone")
		_, err = s.GetJob(ctx, fresh.ID)
		assert.NoError(t, err, "a done row inside the window must survive")
	})

	t.Run("purge never touches a non-terminal row however old", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()

		// Older than any window anyone would configure. A pending row this
		// old is a stuck job, not litter, and deleting it destroys work
		// nobody knows is missing.
		stuck := newJob(appID, userID, KindActivityLog, "stuck", base.Add(-400*24*time.Hour))
		require.NoError(t, s.Enqueue(ctx, stuck))

		inflight := newJob(appID, userID, KindContactUpsert, "inflight", base.Add(-400*24*time.Hour))
		require.NoError(t, s.Enqueue(ctx, inflight))
		claimed, err := s.ClaimDue(ctx, 1, time.Minute, base.Add(-400*24*time.Hour))
		require.NoError(t, err)
		require.Len(t, claimed, 1)

		// Cutoffs in the future: everything eligible is eligible.
		removed, err := s.PurgeTerminal(ctx, base.Add(time.Hour), base.Add(time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 0, removed, "pending and in_flight are not terminal at any age")

		_, err = s.GetJob(ctx, stuck.ID)
		assert.NoError(t, err, "a pending row must survive its own age")
		_, err = s.GetJob(ctx, inflight.ID)
		assert.NoError(t, err, "an in_flight row must survive its own age")
	})

	t.Run("dead and suppressed keep the longer audit window", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()

		dead := newJob(appID, userID, KindActivityLog, "aged-dead", base.Add(-60*24*time.Hour))
		supp := newJob(appID, userID, KindActivityLog, "aged-supp", base.Add(-60*24*time.Hour))
		require.NoError(t, s.Enqueue(ctx, dead))
		require.NoError(t, s.Enqueue(ctx, supp))
		require.NoError(t, s.MarkDead(ctx, dead.ID, "500"))
		require.NoError(t, s.MarkSuppressed(ctx, supp.ID, "no consent"))

		// Sixty days old, past the done window and well inside the audit one.
		removed, err := s.PurgeTerminal(ctx, base.Add(-30*24*time.Hour), base.Add(-180*24*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 0, removed, "the audit trail outlives the done window")
		_, err = s.GetJob(ctx, dead.ID)
		assert.NoError(t, err)
		_, err = s.GetJob(ctx, supp.ID)
		assert.NoError(t, err)

		// Same rows, once the audit window has passed them too.
		removed, err = s.PurgeTerminal(ctx, base.Add(-30*24*time.Hour), base.Add(-30*24*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 2, removed)
		_, err = s.GetJob(ctx, dead.ID)
		assert.ErrorIs(t, err, ErrNotFound)
		_, err = s.GetJob(ctx, supp.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("a zero cutoff purges nothing", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindActivityLog, "zero-cutoff", base.Add(-400*24*time.Hour))
		require.NoError(t, s.Enqueue(ctx, j))
		require.NoError(t, s.MarkDone(ctx, j.ID, base))

		removed, err := s.PurgeTerminal(ctx, time.Time{}, time.Time{})
		require.NoError(t, err)
		assert.Equal(t, 0, removed,
			"a zero cutoff is how a negative retention setting says keep everything")
		_, err = s.GetJob(ctx, j.ID)
		assert.NoError(t, err)
	})

	t.Run("purging a done row releases its idempotency key", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()

		first := newJob(appID, userID, KindActivityLog, "recycled", base.Add(-60*24*time.Hour))
		require.NoError(t, s.Enqueue(ctx, first))
		require.NoError(t, s.MarkDone(ctx, first.ID, base))

		// While the row is there, the key is the replay guard.
		blocked := newJob(appID, userID, KindActivityLog, "recycled", base)
		require.NoError(t, s.Enqueue(ctx, blocked))
		_, err := s.GetJob(ctx, blocked.ID)
		assert.ErrorIs(t, err, ErrNotFound, "the key still holds while the done row lives")

		removed, err := s.PurgeTerminal(ctx, base.Add(-30*24*time.Hour), base.Add(-180*24*time.Hour))
		require.NoError(t, err)
		require.Equal(t, 1, removed)

		// Once it is gone the key is free again. That is the cost of
		// pruning, stated out loud rather than discovered later: the
		// retention window IS the replay window.
		reused := newJob(appID, userID, KindActivityLog, "recycled", base)
		require.NoError(t, s.Enqueue(ctx, reused))
		stored, err := s.GetJob(ctx, reused.ID)
		require.NoError(t, err)
		assert.Equal(t, StatePending, stored.State)
	})
}
