package sharedsignals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

type recordingRevoker struct {
	revoked []id.SessionID
	err     error
}

func (r *recordingRevoker) RevokeSession(_ context.Context, sessionID id.SessionID) error {
	if r.err != nil {
		return r.err
	}
	r.revoked = append(r.revoked, sessionID)
	return nil
}

// failingRevoker fails on exactly its failAt'th call (1-indexed) and records
// every session it actually revoked, in whatever order RevokeSession is
// called -- callers must not depend on which physical session hits the
// failure, only on how many succeeded and failed overall.
type failingRevoker struct {
	failAt  int
	calls   int
	revoked []id.SessionID
}

func (r *failingRevoker) RevokeSession(_ context.Context, sessionID id.SessionID) error {
	r.calls++
	if r.calls == r.failAt {
		return errors.New("boom")
	}
	r.revoked = append(r.revoked, sessionID)
	return nil
}

// recordingChronicle captures every audit event so a test can assert one was
// recorded without standing up a real Chronicle backend.
type recordingChronicle struct {
	events []*bridge.AuditEvent
}

func (c *recordingChronicle) Record(_ context.Context, e *bridge.AuditEvent) error {
	c.events = append(c.events, e)
	return nil
}

type actionFixture struct {
	plugin   *Plugin
	stream   *InboundStream
	revoker  *recordingRevoker
	userID   id.UserID
	sessions []id.SessionID
	// foreignSession belongs to the same user but a different app than the
	// stream. It exists so a test can prove a revocation never reaches it --
	// every OTHER fixture session lives in the stream's own app, so without
	// this one the AppID filters in applyEvent are never actually exercised:
	// deleting them changes nothing any test can observe.
	foreignSession id.SessionID
}

func newActionFixture(t *testing.T) actionFixture {
	t.Helper()
	ctx := context.Background()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	authStore := memory.New()
	var sessions []id.SessionID
	for i := 0; i < 3; i++ {
		s := &session.Session{
			ID: id.NewSessionID(), AppID: appID, EnvID: envID, UserID: userID,
			Token: "tok-" + string(rune('a'+i)), ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, authStore.CreateSession(ctx, s))
		sessions = append(sessions, s.ID)
	}

	foreign := &session.Session{
		ID: id.NewSessionID(), AppID: id.NewAppID(), EnvID: envID, UserID: userID,
		Token: "tok-foreign", ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, authStore.CreateSession(ctx, foreign))

	rev := &recordingRevoker{}
	p := New()
	p.store = NewMemoryStore()
	p.authStore = authStore
	p.revoker = rev

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID,
		Issuer: "https://org.okta.com", Status: StatusEnabled,
		EnforcementMode: EnforcementEnforce, MaxActionsPerHour: 100,
	}
	require.NoError(t, p.store.CreateInboundStream(ctx, stream))

	return actionFixture{plugin: p, stream: stream, revoker: rev,
		userID: userID, sessions: sessions, foreignSession: foreign.ID}
}

func TestActionFor_Defaults(t *testing.T) {
	p := New()
	s := &InboundStream{}
	cases := []struct {
		ev   caep.Event
		want string
	}{
		{caep.Event{Type: caep.EventSessionRevoked}, ActionRevokeAll},
		{caep.Event{Type: caep.EventTokenClaimsChange}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "revoke"}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "delete"}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "create"}, ActionSignal},
		{caep.Event{Type: caep.EventAssuranceLevelChange, ChangeDirection: "decrease"}, ActionSignal},
		{caep.Event{Type: caep.EventAssuranceLevelChange, ChangeDirection: "increase"}, ActionSignal},
		{caep.Event{Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"}, ActionSignal},
		{caep.Event{Type: caep.EventRiskLevelChange, CurrentLevel: "HIGH"}, ActionSignal},
		{caep.Event{Type: caep.EventVerification}, ActionNone},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, p.actionFor(s, tc.ev), "event %s", tc.ev.Type)
	}
}

func TestActionFor_StreamOverrideWins(t *testing.T) {
	p := New()
	s := &InboundStream{ActionOverrides: map[string]string{
		caep.EventSessionRevoked: ActionSignal,
	}}
	assert.Equal(t, ActionSignal, p.actionFor(s, caep.Event{Type: caep.EventSessionRevoked}))
}

func TestSeverityFor(t *testing.T) {
	p := New()
	assert.Equal(t, 100, p.severityFor(caep.Event{Type: caep.EventSessionRevoked}))
	assert.Equal(t, 90, p.severityFor(caep.Event{
		Type: caep.EventCredentialChange, ChangeType: "revoke"}))
	assert.Equal(t, 20, p.severityFor(caep.Event{
		Type: caep.EventCredentialChange, ChangeType: "create"}))
	assert.Equal(t, 70, p.severityFor(caep.Event{
		Type: caep.EventAssuranceLevelChange, ChangeDirection: "decrease"}))
	assert.Equal(t, 10, p.severityFor(caep.Event{
		Type: caep.EventAssuranceLevelChange, ChangeDirection: "increase"}))
	assert.Equal(t, 60, p.severityFor(caep.Event{
		Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"}))
	assert.Equal(t, 10, p.severityFor(caep.Event{
		Type: caep.EventDeviceComplianceChange, CurrentStatus: "compliant"}))
	assert.Equal(t, 80, p.severityFor(caep.Event{
		Type: caep.EventRiskLevelChange, CurrentLevel: "HIGH"}))
}

func TestApplyEvent_SessionRevokedRevokesEverySession(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionRevokeAll, action)
	assert.Len(t, f.revoker.revoked, 3)
}

// A session member in a complex subject narrows the blast radius to one
// session instead of signing the user out everywhere.
func TestApplyEvent_TargetedSessionRevoke(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, SessionID: f.sessions[1], Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionRevokeSession, action)
	require.Len(t, f.revoker.revoked, 1)
	assert.Equal(t, f.sessions[1], f.revoker.revoked[0])
}

// Mutation test: delete the `if sess.AppID != s.AppID { continue }` filter
// from the ActionRevokeAll loop and this test must fail, because
// foreignSession is the only fixture session that would newly appear in
// f.revoker.revoked.
func TestApplyEvent_RevokeAllStaysInStreamApp(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionRevokeAll, action)
	assert.Len(t, f.revoker.revoked, 3, "only the stream's own app is revoked")
	assert.NotContains(t, f.revoker.revoked, f.foreignSession,
		"a session in a different app must never be reached by this stream")
}

// A complex subject's session member can only narrow the blast radius, never
// widen it past the stream's own app. Naming a session that belongs to the
// right user but the wrong app must be refused outright, not silently
// promoted to a revoke-all of every app.
func TestApplyEvent_TargetedRevokeRejectsCrossAppSession(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, SessionID: f.foreignSession, Outcome: OutcomeApplied})
	require.Error(t, err)
	assert.Equal(t, "", action)
	assert.Empty(t, f.revoker.revoked, "a cross-app targeted session must never be revoked")
}

// A revoker that fails partway through must not abort the batch: the
// duplicate-jti replay guard means a retried SET never reaches this code
// again, so any session not revoked on this attempt stays alive permanently.
func TestApplyEvent_RevokeAllContinuesPastPartialFailure(t *testing.T) {
	ctx := context.Background()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	authStore := memory.New()
	for i := 0; i < 5; i++ {
		s := &session.Session{
			ID: id.NewSessionID(), AppID: appID, EnvID: envID, UserID: userID,
			Token: "tok-" + string(rune('a'+i)), ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, authStore.CreateSession(ctx, s))
	}

	rev := &failingRevoker{failAt: 3}
	chron := &recordingChronicle{}
	p := New()
	p.store = NewMemoryStore()
	p.authStore = authStore
	p.revoker = rev
	p.chronicle = chron

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID,
		Issuer: "https://org.okta.com", Status: StatusEnabled,
		EnforcementMode: EnforcementEnforce, MaxActionsPerHour: 100,
	}
	require.NoError(t, p.store.CreateInboundStream(ctx, stream))

	action, err := p.applyEvent(ctx, stream, caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: userID, Outcome: OutcomeApplied})
	require.Error(t, err, "a partial failure must be reported, not swallowed")
	assert.Equal(t, ActionRevokeAll, action)
	assert.Len(t, rev.revoked, 4, "the other four sessions are still revoked despite one failure")

	require.NotEmpty(t, chron.events, "a partial failure must still leave an audit trail")
	last := chron.events[len(chron.events)-1]
	assert.Equal(t, bridge.SeverityCritical, last.Severity)
	assert.Equal(t, "4", last.Metadata["revoked"])
	assert.Equal(t, "1", last.Metadata["failed"])
	assert.Equal(t, "5", last.Metadata["total"])
}

// Observe mode must record the signal and skip the revocation, so an operator
// can watch a new stream before trusting it.
func TestApplyEvent_ObserveModeDoesNotRevoke(t *testing.T) {
	f := newActionFixture(t)
	f.stream.EnforcementMode = EnforcementObserve

	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionLog, action)
	assert.Empty(t, f.revoker.revoked)

	signals, err := f.plugin.store.ListActiveSignals(context.Background(),
		f.stream.AppID, f.stream.EnvID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1, "observe mode still records the signal")
}

func TestApplyEvent_AlwaysWritesASignal(t *testing.T) {
	f := newActionFixture(t)
	_, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)

	signals, err := f.plugin.store.ListActiveSignals(context.Background(),
		f.stream.AppID, f.stream.EnvID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1)
	assert.Equal(t, 60, signals[0].Severity)
	assert.Empty(t, f.revoker.revoked)
}

// A plausible past event_timestamp is preserved as-is.
func TestApplyEvent_RecordsAPlausibleEventTimestamp(t *testing.T) {
	f := newActionFixture(t)
	past := time.Now().Add(-time.Minute).Unix()
	_, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant",
			EventTimestamp: past},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)

	signals, err := f.plugin.store.ListActiveSignals(context.Background(),
		f.stream.AppID, f.stream.EnvID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1)
	assert.WithinDuration(t, time.Unix(past, 0), signals[0].EventAt, time.Second)
}

// 99999999999 is just under the 1e11 millis/seconds boundary, so the
// heuristic reads it as seconds and lands around the year 5138. That is not
// a real event time; it must be clamped to now rather than stored as-is,
// since EventAt is a forensic field an investigator will read.
func TestApplyEvent_ClampsImplausibleEventTimestamp(t *testing.T) {
	f := newActionFixture(t)
	_, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant",
			EventTimestamp: 99999999999},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)

	signals, err := f.plugin.store.ListActiveSignals(context.Background(),
		f.stream.AppID, f.stream.EnvID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1)
	assert.WithinDuration(t, time.Now(), signals[0].EventAt, time.Minute,
		"an implausible timestamp must be clamped to now, not stored as a year-5138 date")
}

func TestApplyEvent_VerificationTakesNoUserAction(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventVerification, State: "abc"},
		Resolution{Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, "", action)
	assert.Empty(t, f.revoker.revoked)
}

func TestCircuitBreaker_TripsAndPausesStream(t *testing.T) {
	ctx := context.Background()
	f := newActionFixture(t)
	f.stream.MaxActionsPerHour = 2
	require.NoError(t, f.plugin.store.UpdateInboundStream(ctx, f.stream))

	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, f.plugin.store.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: f.stream.ID,
			JTI: "prior-" + string(rune('a'+i)), EventType: caep.EventSessionRevoked,
			Outcome: OutcomeApplied, ActionTaken: ActionRevokeAll, ReceivedAt: now,
		}))
	}

	ok, err := f.plugin.checkCircuitBreaker(ctx, f.stream)
	require.NoError(t, err)
	assert.False(t, ok, "the breaker must trip at the limit")

	after, err := f.plugin.store.GetInboundStream(ctx, f.stream.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, after.Status,
		"a tripped breaker pauses the stream so it stops acting until a human looks")
}

func TestCircuitBreaker_AllowsUnderLimit(t *testing.T) {
	f := newActionFixture(t)
	ok, err := f.plugin.checkCircuitBreaker(context.Background(), f.stream)
	require.NoError(t, err)
	assert.True(t, ok)
}

// A stream with no limit of its own (MaxActionsPerHour <= 0) falls back to
// the plugin's configured default rather than being unbounded.
func TestCircuitBreaker_FallsBackToConfigLimitWhenStreamLimitUnset(t *testing.T) {
	ctx := context.Background()
	f := newActionFixture(t)
	f.stream.MaxActionsPerHour = 0
	require.NoError(t, f.plugin.store.UpdateInboundStream(ctx, f.stream))
	f.plugin.config.MaxActionsPerHour = 1

	require.NoError(t, f.plugin.store.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: f.stream.ID,
		JTI: "prior-a", EventType: caep.EventSessionRevoked,
		Outcome: OutcomeApplied, ActionTaken: ActionRevokeAll, ReceivedAt: time.Now(),
	}))

	ok, err := f.plugin.checkCircuitBreaker(ctx, f.stream)
	require.NoError(t, err)
	assert.False(t, ok, "the fallback config limit of 1 must trip after one action")

	after, err := f.plugin.store.GetInboundStream(ctx, f.stream.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, after.Status)
}
