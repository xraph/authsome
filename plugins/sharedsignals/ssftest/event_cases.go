package ssftest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	ssf "github.com/xraph/authsome/plugins/sharedsignals"
)

const sessionRevoked = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"

// seedStream creates one stream under the fixture's own tenant.
func seedStream(t *testing.T, f Fixture) *ssf.InboundStream {
	t.Helper()
	s := newStream(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateInboundStream(context.Background(), s))
	return s
}

// testDuplicateJTIIsRejected is the replay guard. A SET replayed against the
// same stream must be refused on its jti, or a captured session-revoked event
// can be fired again at any later moment.
func testDuplicateJTIIsRejected(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := seedStream(t, f)
	jti := unique("jti")

	require.NoError(t, f.Store.InsertReceivedEvent(ctx, newEvent(s.ID, jti)))

	err := f.Store.InsertReceivedEvent(ctx, newEvent(s.ID, jti))
	require.Error(t, err, "a replayed jti must not be accepted a second time")
	assert.True(t, errors.Is(err, ssf.ErrDuplicateJTI),
		"a replay must report ErrDuplicateJTI so the caller can tell it from a real failure, got %v", err)
}

// testSameJTIOnAnotherStreamIsAllowed is the other half of that key. The
// uniqueness is per (stream, jti, event type), because two transmitters have
// no shared jti namespace and one must not be able to burn the other's ids.
func testSameJTIOnAnotherStreamIsAllowed(t *testing.T, f Fixture) {
	ctx := context.Background()
	a := seedStream(t, f)
	b := seedStream(t, f)
	jti := unique("jti")

	require.NoError(t, f.Store.InsertReceivedEvent(ctx, newEvent(a.ID, jti)))
	assert.NoError(t, f.Store.InsertReceivedEvent(ctx, newEvent(b.ID, jti)),
		"one transmitter's jti must not block another transmitter's")
}

// testGetReceivedEventIsAppScoped pins the documented contract: a row that
// belongs to another tenant answers ErrNotFound, never a distinguishable
// "forbidden", so the endpoint cannot be used to probe for existence.
func testGetReceivedEventIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	theirStream := newStream(f.OtherAppID, f.OtherEnvID)
	require.NoError(t, f.Store.CreateInboundStream(ctx, theirStream))
	theirEvent := newEvent(theirStream.ID, unique("jti"))
	require.NoError(t, f.Store.InsertReceivedEvent(ctx, theirEvent))

	// The row exists, but not for us.
	_, err := f.Store.GetReceivedEvent(ctx, f.AppID, theirEvent.ID)
	require.Error(t, err, "audit read crossed a tenant boundary")
	assert.True(t, errors.Is(err, ssf.ErrNotFound),
		"a cross-tenant audit read must answer ErrNotFound, not a distinguishable error, got %v", err)

	// And an id that exists nowhere must answer identically, so the two are
	// indistinguishable to a caller.
	_, missing := f.Store.GetReceivedEvent(ctx, f.AppID, id.NewSSFEventID())
	require.Error(t, missing)
	assert.Equal(t, errors.Is(err, ssf.ErrNotFound), errors.Is(missing, ssf.ErrNotFound),
		"a foreign row and an absent row must be indistinguishable")

	// Our own row still reads back.
	ourStream := seedStream(t, f)
	ourEvent := newEvent(ourStream.ID, unique("jti"))
	require.NoError(t, f.Store.InsertReceivedEvent(ctx, ourEvent))
	got, err := f.Store.GetReceivedEvent(ctx, f.AppID, ourEvent.ID)
	require.NoError(t, err)
	assert.Equal(t, ourEvent.JTI, got.JTI)
}

// testListReceivedEventsRefusesForeignStream pins the second documented
// contract: a stream that is not yours answers ErrNotFound rather than an
// empty list, so a probe cannot tell "not yours" from "yours but quiet".
func testListReceivedEventsRefusesForeignStream(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	theirStream := newStream(f.OtherAppID, f.OtherEnvID)
	require.NoError(t, f.Store.CreateInboundStream(ctx, theirStream))

	_, err := f.Store.ListReceivedEvents(ctx, f.AppID, ssf.ReceivedEventFilter{StreamID: theirStream.ID})
	require.Error(t, err, "listing another tenant's stream returned rather than refusing")
	assert.True(t, errors.Is(err, ssf.ErrNotFound),
		"a foreign stream must answer ErrNotFound, not an empty list, got %v", err)
}

// testListReceivedEventsRespectsWindow checks the half-open [Since, Until)
// bound the interface documents.
func testListReceivedEventsRespectsWindow(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := seedStream(t, f)

	base := now().Add(-time.Hour)
	for i, offset := range []time.Duration{0, time.Minute, 2 * time.Minute} {
		e := newEvent(s.ID, unique("jti"))
		e.ReceivedAt = base.Add(offset)
		e.ActionTaken = "n" + string(rune('0'+i))
		require.NoError(t, f.Store.InsertReceivedEvent(ctx, e))
	}

	// [base+1m, base+2m) must hold exactly the middle event.
	got, err := f.Store.ListReceivedEvents(ctx, f.AppID, ssf.ReceivedEventFilter{
		StreamID: s.ID,
		Since:    base.Add(time.Minute),
		Until:    base.Add(2 * time.Minute),
	})
	require.NoError(t, err)
	assert.Len(t, got, 1, "the window is half-open: Since is inclusive and Until is exclusive")

	all, err := f.Store.ListReceivedEvents(ctx, f.AppID, ssf.ReceivedEventFilter{StreamID: s.ID})
	require.NoError(t, err)
	assert.Len(t, all, 3, "an unbounded window must return every row on the stream")
}

// testListReceivedEventsClampsLimit covers the two documented bounds: no
// limit means a page rather than the table, and an enormous limit is clamped
// rather than honoured.
func testListReceivedEventsClampsLimit(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := seedStream(t, f)

	const written = ssf.DefaultReceivedEventLimit + 5
	for range written {
		require.NoError(t, f.Store.InsertReceivedEvent(ctx, newEvent(s.ID, unique("jti"))))
	}

	unlimited, err := f.Store.ListReceivedEvents(ctx, f.AppID, ssf.ReceivedEventFilter{StreamID: s.ID})
	require.NoError(t, err)
	assert.Len(t, unlimited, ssf.DefaultReceivedEventLimit,
		"a caller that names no limit must get one page, not the whole table")

	huge, err := f.Store.ListReceivedEvents(ctx, f.AppID, ssf.ReceivedEventFilter{
		StreamID: s.ID,
		Limit:    ssf.MaxReceivedEventLimit * 10,
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(huge), ssf.MaxReceivedEventLimit,
		"an enormous limit must be clamped, or a dashboard request becomes a full scan")
}

// testDeleteReceivedEventFreesTheJTI covers the undo the interface describes.
// When processing fails for an infrastructure reason the dedupe row is
// removed, and a transmitter retry then has to be genuinely reprocessed
// rather than read back as a replay of a delivery that never happened.
func testDeleteReceivedEventFreesTheJTI(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := seedStream(t, f)
	jti := unique("jti")

	e := newEvent(s.ID, jti)
	require.NoError(t, f.Store.InsertReceivedEvent(ctx, e))
	require.NoError(t, f.Store.DeleteReceivedEvent(ctx, e.ID))

	assert.NoError(t, f.Store.InsertReceivedEvent(ctx, newEvent(s.ID, jti)),
		"after the dedupe row is withdrawn the retry must be accepted, not rejected as a replay")
}

func testCountEventsSince(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := seedStream(t, f)

	base := now().Add(-time.Hour)
	old := newEvent(s.ID, unique("jti"))
	old.ReceivedAt = base
	require.NoError(t, f.Store.InsertReceivedEvent(ctx, old))

	for range 3 {
		e := newEvent(s.ID, unique("jti"))
		e.ReceivedAt = now()
		require.NoError(t, f.Store.InsertReceivedEvent(ctx, e))
	}

	n, err := f.Store.CountEventsSince(ctx, s.ID, now().Add(-time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 3, n, "the breaker count must see recent events and not the older one")

	// The breaker counts every recorded event whatever outcome it reached, so
	// a signal-only event still has to be counted.
	signal := newEvent(s.ID, unique("jti"))
	signal.ActionTaken = ""
	signal.Outcome = "ignored"
	signal.ReceivedAt = now()
	require.NoError(t, f.Store.InsertReceivedEvent(ctx, signal))

	n, err = f.Store.CountEventsSince(ctx, s.ID, now().Add(-time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 4, n, "the breaker must count events that took no action, not just the ones that did")
}

// testExpiredSignalIsNotActive checks the expiry boundary on the risk-signal
// lookup, which is what decides whether a past event still constrains a
// sign-in now. It takes the caller's clock as an argument, so the case passes
// a local-zone time on purpose: that is what the risk path actually does, and
// on a backend storing expires_at as text the comparison is only correct if
// the store normalises what it is handed.
func testExpiredSignalIsNotActive(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := seedStream(t, f)

	expired := &ssf.Signal{
		ID: id.NewSSFSignalID(), AppID: f.AppID, EnvID: f.EnvID, UserID: f.UserID,
		StreamID: s.ID, EventType: sessionRevoked, Severity: 5, Reason: "expired",
		EventAt: now().Add(-2 * time.Hour), ExpiresAt: now().Add(-time.Hour),
		CreatedAt: now().Add(-2 * time.Hour),
	}
	require.NoError(t, f.Store.CreateSignal(ctx, expired))

	live := &ssf.Signal{
		ID: id.NewSSFSignalID(), AppID: f.AppID, EnvID: f.EnvID, UserID: f.UserID,
		StreamID: s.ID, EventType: sessionRevoked, Severity: 7, Reason: "live",
		EventAt: now(), ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, f.Store.CreateSignal(ctx, live))

	// time.Now() unqualified, matching the risk path's own call.
	got, err := f.Store.ListActiveSignals(ctx, f.AppID, f.EnvID, f.UserID, time.Now())
	require.NoError(t, err)

	var sawExpired, sawLive bool
	for _, sig := range got {
		if sig.ID == expired.ID {
			sawExpired = true
		}
		if sig.ID == live.ID {
			sawLive = true
		}
	}
	assert.False(t, sawExpired,
		"a risk signal that expired an hour ago is still constraining sign-in")
	assert.True(t, sawLive, "a signal with an hour left must still apply")
}
