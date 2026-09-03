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
}
