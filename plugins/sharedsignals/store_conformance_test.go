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
//
// Mongo shares neither, so it gets its own runner rather than a stand-in:
// store_mongo_conformance_test.go feeds a live mongo store into this same
// suite, behind the integration build tag and AUTHSOME_MONGO_URI.

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

		// The dedupe key is (stream_id, jti, event_type), not (stream_id,
		// jti) alone: RFC 8417 keys a SET's events object by type, so one
		// delivery legitimately carries several types under the same jti.
		// A second event type on the SAME stream and jti must insert fine.
		otherType := &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
			EventType: "other-event-type", Outcome: OutcomePending, ReceivedAt: now,
		}
		require.NoError(t, s.InsertReceivedEvent(ctx, otherType),
			"a different event_type under the same (stream_id, jti) must not collide")

		// DeleteReceivedEvent undoes an insert -- the compensating action
		// servePush takes when processing an event fails for an
		// infrastructure reason, so the transmitter's retry is reprocessed
		// rather than read back later as a replay.
		require.NoError(t, s.DeleteReceivedEvent(ctx, otherType.ID))
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
			EventType: "other-event-type", Outcome: OutcomePending, ReceivedAt: now,
		}), "after DeleteReceivedEvent the same (stream_id, jti, event_type) must insert again")
		require.ErrorIs(t, s.DeleteReceivedEvent(ctx, id.NewSSFEventID()), ErrNotFound,
			"deleting an event ID that was never inserted must not silently succeed")

		ev.Outcome = OutcomeApplied
		ev.ActionTaken = "revoked_all"
		require.NoError(t, s.UpdateReceivedEvent(ctx, ev))
	})

	t.Run("received event reads back through a real database", func(t *testing.T) {
		// Until this existed the audit trail was write-only, and the two
		// fields that make it an audit trail rather than a dedupe key --
		// subject_json and resolved_user_id -- were proven end to end
		// only against the memory backend. Everything else had nothing
		// but a struct-to-model-to-struct conversion with no database in
		// it, so a column that silently never persisted would have looked
		// exactly like a column that worked.
		ctx := context.Background()
		s := newStore(t)
		appID, envID := id.NewAppID(), id.NewEnvironmentID()
		userID := id.NewUserID()
		now := time.Now().UTC().Truncate(time.Second)

		stream := &InboundStream{
			ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID,
			Name: "okta", Issuer: "https://org.okta.com",
			Audience:     "https://authsome.example/ssf",
			JWKSURI:      "https://org.okta.com/keys",
			PushPathHash: "hash-read-back", PushTokenHash: "tok-read-back",
			EnforcementMode: EnforcementEnforce, Status: StatusEnabled,
			MaxActionsPerHour: 100, CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, s.CreateInboundStream(ctx, stream))

		// The subject exactly as a transmitter would send it, offending
		// value and all: this string is where the receiver puts the
		// detail it deliberately keeps out of its HTTP error bodies.
		subjectJSON := `{"format":"iss_sub","iss":"https://org.okta.com","sub":"okta-user-42"}`
		ev := &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: stream.ID, JTI: "jti-read-back",
			EventType:   "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
			SubjectJSON: subjectJSON, ResolvedUserID: userID,
			Outcome: OutcomeApplied, ActionTaken: "revoke_all", ReceivedAt: now,
		}
		require.NoError(t, s.InsertReceivedEvent(ctx, ev))

		got, err := s.GetReceivedEvent(ctx, appID, ev.ID)
		require.NoError(t, err)
		assert.Equal(t, ev.ID, got.ID)
		assert.Equal(t, stream.ID, got.StreamID)
		assert.Equal(t, "jti-read-back", got.JTI)
		assert.Equal(t, ev.EventType, got.EventType)
		assert.Equal(t, OutcomeApplied, got.Outcome)
		assert.Equal(t, "revoke_all", got.ActionTaken)
		assert.Equal(t, subjectJSON, got.SubjectJSON,
			"the subject the transmitter named must survive the database round trip")
		assert.Equal(t, userID, got.ResolvedUserID,
			"the user whose sessions were ended must survive the database round trip")

		// An event ID nobody ever wrote is a miss, not an empty struct.
		_, err = s.GetReceivedEvent(ctx, appID, id.NewSSFEventID())
		require.ErrorIs(t, err, ErrNotFound)

		// Another tenant asking for this row by ID gets NotFound, not the
		// row and not a distinguishable Forbidden. See streamOwnedBy.
		_, err = s.GetReceivedEvent(ctx, id.NewAppID(), ev.ID)
		require.ErrorIs(t, err, ErrNotFound,
			"a row from another tenant's stream must be indistinguishable from a miss")

		list, err := s.ListReceivedEvents(ctx, appID, ReceivedEventFilter{StreamID: stream.ID})
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, subjectJSON, list[0].SubjectJSON)
		assert.Equal(t, userID, list[0].ResolvedUserID)

		_, err = s.ListReceivedEvents(ctx, id.NewAppID(), ReceivedEventFilter{StreamID: stream.ID})
		require.ErrorIs(t, err, ErrNotFound,
			"listing another tenant's stream must not answer an empty list, which would confirm the stream exists")

		_, err = s.ListReceivedEvents(ctx, appID, ReceivedEventFilter{StreamID: id.NewSSFStreamID()})
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("received event list windows, orders and limits", func(t *testing.T) {
		// An operator investigating a sign-out asks "what did this stream
		// send me, around when". That is the whole query: one stream, a
		// time window, newest first, bounded.
		ctx := context.Background()
		s := newStore(t)
		appID := id.NewAppID()
		now := time.Now().UTC().Truncate(time.Second)

		stream := &InboundStream{
			ID: id.NewSSFStreamID(), AppID: appID, EnvID: id.NewEnvironmentID(),
			Name: "okta", Issuer: "https://org.okta.com",
			Audience: "https://authsome.example/ssf", JWKSURI: "https://org.okta.com/keys",
			PushPathHash: "hash-window", PushTokenHash: "tok-window",
			EnforcementMode: EnforcementEnforce, Status: StatusEnabled,
			MaxActionsPerHour: 100, CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, s.CreateInboundStream(ctx, stream))

		// A stream belonging to the same app, so a backend that dropped
		// the stream_id filter would over-return rather than under-return.
		other := &InboundStream{
			ID: id.NewSSFStreamID(), AppID: appID, EnvID: id.NewEnvironmentID(),
			Name: "entra", Issuer: "https://login.microsoftonline.com",
			Audience: "https://authsome.example/ssf", JWKSURI: "https://login.microsoftonline.com/keys",
			PushPathHash: "hash-window-other", PushTokenHash: "tok-window-other",
			EnforcementMode: EnforcementEnforce, Status: StatusEnabled,
			MaxActionsPerHour: 100, CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, s.CreateInboundStream(ctx, other))
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: other.ID, JTI: "elsewhere",
			EventType: "e", Outcome: OutcomeApplied, ReceivedAt: now,
		}))

		// Four hours apart so no backend's timestamp precision can
		// reorder them.
		for i, jti := range []string{"oldest", "older", "newer", "newest"} {
			require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
				ID: id.NewSSFEventID(), StreamID: stream.ID, JTI: jti,
				EventType: "e", Outcome: OutcomeApplied,
				ReceivedAt: now.Add(time.Duration(i-3) * 4 * time.Hour),
			}))
		}

		all, err := s.ListReceivedEvents(ctx, appID, ReceivedEventFilter{StreamID: stream.ID})
		require.NoError(t, err)
		require.Len(t, all, 4, "only this stream's rows, and all of them")
		assert.Equal(t, []string{"newest", "newer", "older", "oldest"},
			[]string{all[0].JTI, all[1].JTI, all[2].JTI, all[3].JTI},
			"newest first, so a dashboard's first page is the recent traffic")

		// The window is half-open [Since, Until): the two rows at -4h and
		// 0h are in, the -8h and -12h rows are not.
		windowed, err := s.ListReceivedEvents(ctx, appID, ReceivedEventFilter{
			StreamID: stream.ID,
			Since:    now.Add(-5 * time.Hour),
			Until:    now.Add(time.Second),
		})
		require.NoError(t, err)
		require.Len(t, windowed, 2)
		assert.Equal(t, []string{"newest", "newer"},
			[]string{windowed[0].JTI, windowed[1].JTI})

		// Until is exclusive, so a bound landing exactly on a row's
		// received_at excludes that row.
		exclusive, err := s.ListReceivedEvents(ctx, appID, ReceivedEventFilter{
			StreamID: stream.ID, Until: now,
		})
		require.NoError(t, err)
		require.Len(t, exclusive, 3)
		assert.Equal(t, "newer", exclusive[0].JTI)

		limited, err := s.ListReceivedEvents(ctx, appID, ReceivedEventFilter{
			StreamID: stream.ID, Limit: 2,
		})
		require.NoError(t, err)
		require.Len(t, limited, 2)
		assert.Equal(t, "newest", limited[0].JTI,
			"a limit takes the newest rows, not an arbitrary two")

		// A limit above the cap is clamped, not honoured, so no caller
		// turns a dashboard request into a full table scan.
		clamped, err := s.ListReceivedEvents(ctx, appID, ReceivedEventFilter{
			StreamID: stream.ID, Limit: MaxReceivedEventLimit * 10,
		})
		require.NoError(t, err)
		assert.Len(t, clamped, 4)

		// A window with nothing in it is a non-nil, zero-length slice:
		// callers marshal this straight to JSON, where nil and empty
		// serialize differently.
		empty, err := s.ListReceivedEvents(ctx, appID, ReceivedEventFilter{
			StreamID: stream.ID, Since: now.Add(time.Hour),
		})
		require.NoError(t, err)
		assert.NotNil(t, empty)
		assert.Len(t, empty, 0)
	})

	t.Run("circuit breaker counts every recorded event", func(t *testing.T) {
		// The breaker's counter used to skip any row whose action_taken was
		// empty, which left the whole signal-only half of the action matrix
		// unbounded: an authentic but hostile transmitter could push
		// risk-level-change at HIGH forever with the counter at zero. Every
		// backend must now count every recorded event in the window.
		ctx := context.Background()
		s := newStore(t)
		streamID := id.NewSSFStreamID()
		now := time.Now().UTC().Truncate(time.Second)

		// Took an action.
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "acted",
			EventType: "e", Outcome: OutcomeApplied, ActionTaken: "revoke_all",
			ReceivedAt: now,
		}))
		// Signal only: no action, still counts.
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "signal-only",
			EventType: "e", Outcome: OutcomeApplied, ReceivedAt: now,
		}))
		// Never resolved to anyone: no action, still counts.
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "unresolved",
			EventType: "e", Outcome: OutcomeUnresolved, ReceivedAt: now,
		}))
		// Outside the window.
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "old",
			EventType: "e", Outcome: OutcomeApplied, ActionTaken: "revoke_all",
			ReceivedAt: now.Add(-2 * time.Hour),
		}))
		// Another stream entirely, so a backend that dropped the stream_id
		// filter would over-count.
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "other-stream",
			EventType: "e", Outcome: OutcomeApplied, ActionTaken: "revoke_all",
			ReceivedAt: now,
		}))

		count, err := s.CountEventsSince(ctx, streamID, now.Add(-time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 3, count,
			"every event recorded for this stream in the window counts, action or not")
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
