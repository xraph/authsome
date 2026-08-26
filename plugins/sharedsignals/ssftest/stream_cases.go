package ssftest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	ssf "github.com/xraph/authsome/plugins/sharedsignals"
)

func testStreamCRUD(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := newStream(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateInboundStream(ctx, s))

	got, err := f.Store.GetInboundStream(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.AppID, got.AppID)
	assert.Equal(t, s.EnvID, got.EnvID)
	assert.Equal(t, s.Issuer, got.Issuer)
	assert.Equal(t, s.Audience, got.Audience)
	assert.Equal(t, s.JWKSURI, got.JWKSURI)
	assert.Equal(t, s.PushTokenHash, got.PushTokenHash,
		"the push token hash is what authenticates a transmitter; it has to survive intact")
	assert.Equal(t, s.EnforcementMode, got.EnforcementMode)
	assert.Equal(t, s.MaxActionsPerHour, got.MaxActionsPerHour)
}

func testStreamNotFound(t *testing.T, f Fixture) {
	ctx := context.Background()

	_, err := f.Store.GetInboundStream(ctx, id.NewSSFStreamID())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ssf.ErrNotFound), "got %v", err)

	_, err = f.Store.GetInboundStreamByPushPathHash(ctx, unique("absent"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ssf.ErrNotFound), "got %v", err)
}

// testStreamLookupByPushPathHash covers the path an inbound delivery takes.
// The hash is derived from a secret URL segment, so this lookup is the first
// thing an unauthenticated request touches.
func testStreamLookupByPushPathHash(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := newStream(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateInboundStream(ctx, s))

	got, err := f.Store.GetInboundStreamByPushPathHash(ctx, s.PushPathHash)
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.ID)
	assert.Equal(t, s.AppID, got.AppID, "the resolved stream must carry the tenant it belongs to")
}

func testListStreamsIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	mine := newStream(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateInboundStream(ctx, mine))
	theirs := newStream(f.OtherAppID, f.OtherEnvID)
	require.NoError(t, f.Store.CreateInboundStream(ctx, theirs))

	got, err := f.Store.ListInboundStreams(ctx, f.AppID)
	require.NoError(t, err)

	var found bool
	for _, s := range got {
		assert.Equal(t, f.AppID, s.AppID, "listing leaked another tenant's stream")
		if s.ID == mine.ID {
			found = true
		}
	}
	assert.True(t, found, "listing omitted a stream belonging to the queried app")
}

// testStreamSliceFieldsRoundTrip matters because these lists are allow-lists.
// A dropped entry silently narrows what the stream accepts; a phantom entry
// silently widens it.
func testStreamSliceFieldsRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := newStream(f.AppID, f.EnvID)
	s.AllowedEventTypes = []string{
		"https://schemas.openid.net/secevent/caep/event-type/session-revoked",
		"https://schemas.openid.net/secevent/caep/event-type/credential-change",
	}
	s.AllowedSubjectFormats = []string{"iss_sub", "email", "opaque"}
	s.VerifiedDomains = []string{"example.test", "corp.example.test"}
	s.ActionOverrides = map[string]string{
		"https://schemas.openid.net/secevent/caep/event-type/session-revoked": "revoke_sessions",
	}
	require.NoError(t, f.Store.CreateInboundStream(ctx, s))

	got, err := f.Store.GetInboundStream(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.AllowedEventTypes, got.AllowedEventTypes, "the event-type allow-list changed in storage")
	assert.Equal(t, s.AllowedSubjectFormats, got.AllowedSubjectFormats)
	assert.Equal(t, s.VerifiedDomains, got.VerifiedDomains)
	assert.Equal(t, s.ActionOverrides, got.ActionOverrides)
}

func testUpdateStream(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := newStream(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateInboundStream(ctx, s))

	s.Status = "paused"
	s.EnforcementMode = "monitor"
	s.MaxActionsPerHour = 5
	s.UpdatedAt = now()
	require.NoError(t, f.Store.UpdateInboundStream(ctx, s))

	got, err := f.Store.GetInboundStream(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "paused", got.Status, "pausing a stream must persist; it is how a bad transmitter gets stopped")
	assert.Equal(t, "monitor", got.EnforcementMode)
	assert.Equal(t, 5, got.MaxActionsPerHour, "the breaker limit must persist")
	assert.Equal(t, s.PushTokenHash, got.PushTokenHash, "updating one field must not disturb the credentials")
}

func testDeleteStream(t *testing.T, f Fixture) {
	ctx := context.Background()
	s := newStream(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateInboundStream(ctx, s))
	require.NoError(t, f.Store.DeleteInboundStream(ctx, s.ID))

	_, err := f.Store.GetInboundStream(ctx, s.ID)
	assert.Error(t, err, "a deleted stream must stop resolving")

	_, err = f.Store.GetInboundStreamByPushPathHash(ctx, s.PushPathHash)
	assert.Error(t, err, "a deleted stream must stop accepting pushes on its old path")
}

// testSubjectLinkUpsertIsIdempotent covers the upsert. The same (issuer,
// subject) pair arrives on every event for that user, so a second call must
// update the existing link rather than accumulate duplicates.
func testSubjectLinkUpsertIsIdempotent(t *testing.T, f Fixture) {
	ctx := context.Background()
	issuer := "https://transmitter.example.test"
	subject := unique("subject")

	l := &ssf.SubjectLink{
		ID: id.NewSSFLinkID(), AppID: f.AppID, EnvID: f.EnvID,
		Issuer: issuer, Subject: subject, UserID: f.UserID,
		Source: "verified", CreatedAt: now(), LastSeenAt: now(),
	}
	require.NoError(t, f.Store.UpsertSubjectLink(ctx, l))

	later := now().Add(time.Minute)
	l2 := &ssf.SubjectLink{
		ID: id.NewSSFLinkID(), AppID: f.AppID, EnvID: f.EnvID,
		Issuer: issuer, Subject: subject, UserID: f.UserID,
		Source: "verified", CreatedAt: now(), LastSeenAt: later,
	}
	require.NoError(t, f.Store.UpsertSubjectLink(ctx, l2), "a repeat link must upsert, not collide")

	got, err := f.Store.GetSubjectLink(ctx, f.AppID, f.EnvID, issuer, subject)
	require.NoError(t, err)
	assert.Equal(t, f.UserID, got.UserID, "the link must still resolve to the same user")
}

// testSubjectLinkLookupIsTenantScoped keeps one tenant's identity mapping out
// of another's. Resolving a subject to the wrong user is how an event about
// somebody at one company revokes sessions for somebody at another.
func testSubjectLinkLookupIsTenantScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()
	issuer := "https://transmitter.example.test"
	subject := unique("shared-subject")

	require.NoError(t, f.Store.UpsertSubjectLink(ctx, &ssf.SubjectLink{
		ID: id.NewSSFLinkID(), AppID: f.OtherAppID, EnvID: f.OtherEnvID,
		Issuer: issuer, Subject: subject, UserID: f.UserID,
		Source: "verified", CreatedAt: now(), LastSeenAt: now(),
	}))

	_, err := f.Store.GetSubjectLink(ctx, f.AppID, f.EnvID, issuer, subject)
	require.Error(t, err, "subject link lookup crossed a tenant boundary")
	assert.True(t, errors.Is(err, ssf.ErrNotFound), "got %v", err)
}

// testSubjectLinkUpsertIsConcurrencySafe fires the same tuple at the store
// from many goroutines at once. This is not ceremony and the goroutine count
// is not arbitrary: a subject link is written on every SSO sign-in, so two
// writes for one subject arriving together is what happens when somebody
// opens two tabs. testSubjectLinkUpsertIsIdempotent proves the second call
// updates rather than collides when it arrives afterwards. It says nothing
// about what happens when both arrive at once.
//
// A read-then-write implementation loses that race. Both readers see "not
// found", both insert, and the loser hits the unique index and surfaces a raw
// constraint error instead of succeeding. So every call here must return nil,
// and exactly one row must survive.
func testSubjectLinkUpsertIsConcurrencySafe(t *testing.T, f Fixture) {
	ctx := context.Background()
	issuer := "https://" + unique("race") + ".test"
	subject := unique("subject")

	const n = 30
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			errs[i] = f.Store.UpsertSubjectLink(ctx, &ssf.SubjectLink{
				ID: id.NewSSFLinkID(), AppID: f.AppID, EnvID: f.EnvID,
				Issuer: issuer, Subject: subject, UserID: id.NewUserID(),
				Source: "verified", CreatedAt: now(), LastSeenAt: now(),
			})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "concurrent upsert %d must not surface a constraint error", i)
	}

	// The storm proves no writer surfaces a raw constraint error, but real
	// concurrency leaves no way to know which of the n writes landed last.
	// One more deterministic write after it settles pins that down: if this
	// one, and only this one, comes back, then exactly one row survived and
	// the operation is genuinely last-write-wins.
	last := id.NewUserID()
	require.NoError(t, f.Store.UpsertSubjectLink(ctx, &ssf.SubjectLink{
		ID: id.NewSSFLinkID(), AppID: f.AppID, EnvID: f.EnvID,
		Issuer: issuer, Subject: subject, UserID: last,
		Source: "verified", CreatedAt: now(), LastSeenAt: now(),
	}))

	got, err := f.Store.GetSubjectLink(ctx, f.AppID, f.EnvID, issuer, subject)
	require.NoError(t, err)
	assert.Equal(t, last, got.UserID,
		"exactly one row must survive the race, holding the last-written UserID")
}
