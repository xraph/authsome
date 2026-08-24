package sharedsignals

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/id"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

// The SQLite store shares the models and mappers with Postgres, so running
// the suite against embedded SQLite exercises the same column mapping and
// the same JSON round trip without needing Docker.

func newMemoryConformanceStore(_ *testing.T) Store { return NewMemoryStore() }

func newSQLiteConformanceStore(t *testing.T) Store {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "ssf-conformance.db") + "?cache=shared"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// The core migrations satisfy the group's DependsOn("authsome").
	require.NoError(t, sqlitestore.New(db).Migrate(ctx, SqliteMigrations))
	return NewSqliteStore(db)
}

func TestStoreConformance_Memory(t *testing.T) { runStoreConformance(t, newMemoryConformanceStore) }
func TestStoreConformance_SQLite(t *testing.T) { runStoreConformance(t, newSQLiteConformanceStore) }

func runStoreConformance(t *testing.T, newStore func(*testing.T) Store) {
	t.Run("inbound stream round trip", func(t *testing.T) {
		ctx := context.Background()
		s := newStore(t)
		appID := id.NewAppID()
		now := time.Now().UTC().Truncate(time.Second)

		in := &InboundStream{
			ID: id.NewSSFStreamID(), AppID: appID, EnvID: id.NewEnvironmentID(),
			Name: "okta", Issuer: "https://org.okta.com",
			Audience:     "https://authsome.example/ssf",
			JWKSURI:      "https://org.okta.com/keys",
			PushPathHash: "hash-a", PushTokenHash: "tok-a",
			AllowedEventTypes:     []string{"a", "b"},
			AllowedSubjectFormats: []string{"iss_sub"},
			VerifiedDomains:       []string{"corp.com"},
			ActionOverrides:       map[string]string{"a": "signal"},
			EnforcementMode:       EnforcementEnforce,
			Status:                StatusEnabled,
			MaxActionsPerHour:     100,
			CreatedAt:             now, UpdatedAt: now,
		}
		require.NoError(t, s.CreateInboundStream(ctx, in))

		got, err := s.GetInboundStream(ctx, in.ID)
		require.NoError(t, err)
		assert.Equal(t, "okta", got.Name)
		assert.Equal(t, []string{"a", "b"}, got.AllowedEventTypes)
		assert.Equal(t, map[string]string{"a": "signal"}, got.ActionOverrides)

		byHash, err := s.GetInboundStreamByPushPathHash(ctx, "hash-a")
		require.NoError(t, err)
		assert.Equal(t, in.ID, byHash.ID)

		_, err = s.GetInboundStreamByPushPathHash(ctx, "does-not-exist")
		require.ErrorIs(t, err, ErrNotFound)

		got.Status = StatusPaused
		got.MaxActionsPerHour = 5
		require.NoError(t, s.UpdateInboundStream(ctx, got))
		after, err := s.GetInboundStream(ctx, in.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusPaused, after.Status)
		assert.Equal(t, 5, after.MaxActionsPerHour)

		list, err := s.ListInboundStreams(ctx, appID)
		require.NoError(t, err)
		require.Len(t, list, 1)

		require.NoError(t, s.DeleteInboundStream(ctx, in.ID))
		_, err = s.GetInboundStream(ctx, in.ID)
		require.ErrorIs(t, err, ErrNotFound)

		// A zero-result list must be a non-nil, zero-length slice, not
		// nil: callers marshal this straight to JSON, where nil and an
		// empty slice serialize differently (null vs []).
		empty, err := s.ListInboundStreams(ctx, appID)
		require.NoError(t, err)
		assert.NotNil(t, empty)
		assert.Len(t, empty, 0)
	})

	t.Run("subject link upsert is scoped", func(t *testing.T) {
		ctx := context.Background()
		s := newStore(t)
		appID, envID := id.NewAppID(), id.NewEnvironmentID()
		userID := id.NewUserID()
		now := time.Now().UTC().Truncate(time.Second)

		require.NoError(t, s.UpsertSubjectLink(ctx, &SubjectLink{
			ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
			Issuer: "https://i", Subject: "u1", UserID: userID,
			Source: SourceSSO, CreatedAt: now, LastSeenAt: now,
		}))

		got, err := s.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
		require.NoError(t, err)
		assert.Equal(t, userID, got.UserID)

		// Same tuple, different user: replace, do not duplicate.
		other := id.NewUserID()
		require.NoError(t, s.UpsertSubjectLink(ctx, &SubjectLink{
			ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
			Issuer: "https://i", Subject: "u1", UserID: other,
			Source: SourceSSO, CreatedAt: now, LastSeenAt: now,
		}))
		got, err = s.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
		require.NoError(t, err)
		assert.Equal(t, other, got.UserID)

		// A link belongs to one app only.
		_, err = s.GetSubjectLink(ctx, id.NewAppID(), envID, "https://i", "u1")
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("subject link upsert is concurrency safe", func(t *testing.T) {
		// A later task writes a subject link on every SSO sign-in, so
		// concurrent upserts of the same (app_id, env_id, issuer, subject)
		// tuple are the normal case, not the edge case. A read-then-write
		// implementation loses this race: both readers see "not found" and
		// both insert, so the loser hits the unique index and must surface
		// that as a raw constraint error instead of succeeding. Every call
		// here must return nil, and exactly one row must remain afterward.
		ctx := context.Background()
		s := newStore(t)
		appID, envID := id.NewAppID(), id.NewEnvironmentID()
		now := time.Now().UTC().Truncate(time.Second)

		const n = 30
		userIDs := make([]id.UserID, n)
		for i := range userIDs {
			userIDs[i] = id.NewUserID()
		}

		errs := make([]error, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(i int) {
				defer wg.Done()
				errs[i] = s.UpsertSubjectLink(ctx, &SubjectLink{
					ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
					Issuer: "https://race", Subject: "shared", UserID: userIDs[i],
					Source: SourceSSO, CreatedAt: now, LastSeenAt: now,
				})
			}(i)
		}
		wg.Wait()

		for i, err := range errs {
			assert.NoError(t, err, "concurrent upsert %d must not surface a constraint error", i)
		}

		// The storm above proves no writer ever surfaces a raw constraint
		// error, but true concurrency leaves no way to know which of the
		// n writes actually lands last. Issue one more, deterministic
		// write after the storm settles and confirm it — and only it — is
		// what GetSubjectLink returns, proving both that exactly one row
		// survives and that the operation is genuinely last-write-wins.
		last := id.NewUserID()
		require.NoError(t, s.UpsertSubjectLink(ctx, &SubjectLink{
			ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
			Issuer: "https://race", Subject: "shared", UserID: last,
			Source: SourceSSO, CreatedAt: now, LastSeenAt: now,
		}))

		got, err := s.GetSubjectLink(ctx, appID, envID, "https://race", "shared")
		require.NoError(t, err)
		assert.Equal(t, last, got.UserID,
			"exactly one row must survive the race, holding the last-written UserID")
	})

	t.Run("received event dedupe", func(t *testing.T) {
		ctx := context.Background()
		s := newStore(t)
		streamID := id.NewSSFStreamID()
		now := time.Now().UTC().Truncate(time.Second)

		ev := &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
			EventType: "e", Outcome: OutcomePending, ReceivedAt: now,
		}
		require.NoError(t, s.InsertReceivedEvent(ctx, ev))

		err := s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
			EventType: "e", Outcome: OutcomePending, ReceivedAt: now,
		})
		require.ErrorIs(t, err, ErrDuplicateJTI,
			"a replayed jti must be reported as a duplicate, not a generic write error")

		// This event lives on a DIFFERENT stream and already carries an
		// ActionTaken, so a backend that dropped the stream_id filter in
		// CountActionsSince would count it too and the assertion below
		// would see 2 instead of 1.
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "jti-1",
			EventType: "e", Outcome: OutcomeApplied, ActionTaken: "revoked_all",
			ReceivedAt: now,
		}))

		ev.Outcome = OutcomeApplied
		ev.ActionTaken = "revoked_all"
		require.NoError(t, s.UpdateReceivedEvent(ctx, ev))

		count, err := s.CountActionsSince(ctx, streamID, now.Add(-time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 1, count, "CountActionsSince must be scoped to stream_id")
	})

	t.Run("signals expire", func(t *testing.T) {
		ctx := context.Background()
		s := newStore(t)
		appID, envID, userID := id.NewAppID(), id.NewEnvironmentID(), id.NewUserID()
		now := time.Now().UTC().Truncate(time.Second)

		require.NoError(t, s.CreateSignal(ctx, &Signal{
			ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID,
			UserID: userID, StreamID: id.NewSSFStreamID(),
			EventType: "live", Severity: 90, Reason: "compromised",
			EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
		}))
		require.NoError(t, s.CreateSignal(ctx, &Signal{
			ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID,
			UserID: userID, StreamID: id.NewSSFStreamID(),
			EventType: "stale", Severity: 90, Reason: "old",
			EventAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
			CreatedAt: now.Add(-2 * time.Hour),
		}))

		got, err := s.ListActiveSignals(ctx, appID, envID, userID, now)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "live", got[0].EventType)
		assert.Equal(t, 90, got[0].Severity)

		got, err = s.ListActiveSignals(ctx, appID, envID, id.NewUserID(), now)
		require.NoError(t, err)
		// Non-nil, zero-length: assert.Empty cannot tell a nil slice from
		// an empty one, and callers marshal this straight to JSON.
		assert.NotNil(t, got)
		assert.Len(t, got, 0)
	})

	t.Run("signals are scoped to environment", func(t *testing.T) {
		// A production compromise signal must not raise the risk score on
		// a development sign-in, so a signal recorded in a different
		// environment for the same app and user must be excluded.
		ctx := context.Background()
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		prodEnv, devEnv := id.NewEnvironmentID(), id.NewEnvironmentID()
		now := time.Now().UTC().Truncate(time.Second)

		require.NoError(t, s.CreateSignal(ctx, &Signal{
			ID: id.NewSSFSignalID(), AppID: appID, EnvID: prodEnv,
			UserID: userID, StreamID: id.NewSSFStreamID(),
			EventType: "compromised", Severity: 100, Reason: "credential-leak",
			EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
		}))

		got, err := s.ListActiveSignals(ctx, appID, devEnv, userID, now)
		require.NoError(t, err)
		assert.NotNil(t, got, "a signal from a different environment must not leak in")
		assert.Len(t, got, 0, "a signal from a different environment must not leak in")

		got, err = s.ListActiveSignals(ctx, appID, prodEnv, userID, now)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "compromised", got[0].EventType)
	})
}
