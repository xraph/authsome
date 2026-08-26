package consenttest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/consent"
)

func testGrantAndGet(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConsent(f.UserID, f.AppID, "marketing")
	require.NoError(t, f.Store.GrantConsent(ctx, c))

	got, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "marketing")
	require.NoError(t, err)
	assert.Equal(t, f.UserID, got.UserID)
	assert.Equal(t, f.AppID, got.AppID)
	assert.Equal(t, "marketing", got.Purpose)
	assert.True(t, got.Granted)
	assert.Equal(t, c.Version, got.Version,
		"the policy version is what says which text was agreed to; losing it makes the record unusable as evidence")
	assert.Equal(t, c.IPAddress, got.IPAddress)
	assert.WithinDuration(t, c.GrantedAt, got.GrantedAt, time.Second,
		"the moment of consent has to survive; a shifted timestamp is a shifted audit trail")
}

func testConsentNotFound(t *testing.T, f Fixture) {
	_, err := f.Store.GetConsent(context.Background(), f.UserID, f.AppID, "never-asked")
	assert.Error(t, err, "a purpose nobody was asked about must not resolve to a consent record")
}

// testGrantIsUpsertNotDuplicate covers the documented upsert. Re-consenting
// to the same purpose has to update the record in place, because two rows for
// one purpose leaves the question of which one is current.
func testGrantIsUpsertNotDuplicate(t *testing.T, f Fixture) {
	ctx := context.Background()

	first := newConsent(f.UserID, f.AppID, "analytics")
	first.Version = "2026-01-01"
	require.NoError(t, f.Store.GrantConsent(ctx, first))

	second := newConsent(f.UserID, f.AppID, "analytics")
	second.Version = "2026-06-01"
	require.NoError(t, f.Store.GrantConsent(ctx, second))

	got, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "analytics")
	require.NoError(t, err)
	assert.Equal(t, "2026-06-01", got.Version, "re-consenting must move the record to the newer policy version")

	list, _, err := f.Store.ListConsents(ctx, &consent.Query{UserID: f.UserID, AppID: f.AppID, Limit: 100})
	require.NoError(t, err)
	var n int
	for _, c := range list {
		if c.Purpose == "analytics" {
			n++
		}
	}
	assert.Equal(t, 1, n, "re-consenting produced %d rows for one purpose; the current answer is now ambiguous", n)
}

// testPurposesAreIndependent is the core of purpose-limited consent. Agreeing
// to one thing must never imply another.
func testPurposesAreIndependent(t *testing.T, f Fixture) {
	ctx := context.Background()
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.UserID, f.AppID, "marketing")))

	_, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "analytics")
	assert.Error(t, err, "consent to marketing resolved a record for analytics")
}

// testActiveConsentHasNoRevokedAt pins the nullable timestamp. RevokedAt is a
// pointer so that an active consent carries no revocation time, and a backend
// substituting a zero time makes a live consent look withdrawn.
func testActiveConsentHasNoRevokedAt(t *testing.T, f Fixture) {
	ctx := context.Background()
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.UserID, f.AppID, "essential")))

	got, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "essential")
	require.NoError(t, err)
	require.True(t, got.Granted)
	if got.RevokedAt != nil {
		assert.True(t, got.RevokedAt.IsZero(),
			"an active consent carried a revocation time of %v", got.RevokedAt)
	}
}

// testRevokeRecordsTimeAndClearsGrant is the withdrawal path. Both halves
// matter: the flag is what gates processing, and the timestamp is what proves
// when it stopped being allowed.
func testRevokeRecordsTimeAndClearsGrant(t *testing.T, f Fixture) {
	ctx := context.Background()
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.UserID, f.AppID, "marketing")))

	require.NoError(t, f.Store.RevokeConsent(ctx, f.UserID, f.AppID, "marketing"))

	got, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "marketing")
	require.NoError(t, err, "a revoked consent must still be readable; the record is the evidence")
	assert.False(t, got.Granted, "revocation must clear the grant, or processing carries on regardless")
	if assert.NotNil(t, got.RevokedAt, "revocation must record when it happened") {
		assert.False(t, got.RevokedAt.IsZero())
	}
}

func testRevokeOnlyTheNamedPurpose(t *testing.T, f Fixture) {
	ctx := context.Background()
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.UserID, f.AppID, "marketing")))
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.UserID, f.AppID, "analytics")))

	require.NoError(t, f.Store.RevokeConsent(ctx, f.UserID, f.AppID, "marketing"))

	survived, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "analytics")
	require.NoError(t, err)
	assert.True(t, survived.Granted,
		"withdrawing one purpose withdrew another; a user unsubscribing from email must not lose the rest")
}

// testRegrantAfterRevoke covers changing your mind back. The revocation stamp
// has to clear, or the record still reads as withdrawn.
func testRegrantAfterRevoke(t *testing.T, f Fixture) {
	ctx := context.Background()
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.UserID, f.AppID, "marketing")))
	require.NoError(t, f.Store.RevokeConsent(ctx, f.UserID, f.AppID, "marketing"))

	again := newConsent(f.UserID, f.AppID, "marketing")
	again.Version = "2026-06-01"
	require.NoError(t, f.Store.GrantConsent(ctx, again))

	got, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "marketing")
	require.NoError(t, err)
	assert.True(t, got.Granted, "re-granting must restore the grant")
	if got.RevokedAt != nil {
		assert.True(t, got.RevokedAt.IsZero(),
			"a re-granted consent still carries a revocation time of %v, so it reads as withdrawn", got.RevokedAt)
	}
}

func testLookupIsScopedToUser(t *testing.T, f Fixture) {
	if f.OtherUserID.IsNil() {
		t.Skip("fixture provides no second user")
	}
	ctx := context.Background()
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.OtherUserID, f.AppID, "marketing")))

	_, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "marketing")
	assert.Error(t, err, "one user's consent answered another user's lookup")
}

func testLookupIsScopedToApp(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() || f.OtherAppUser.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.OtherAppUser, f.OtherAppID, "marketing")))

	_, err := f.Store.GetConsent(ctx, f.UserID, f.AppID, "marketing")
	assert.Error(t, err, "consent lookup crossed a tenant boundary")
}

func testListIsScopedToUserAndApp(t *testing.T, f Fixture) {
	if f.OtherUserID.IsNil() {
		t.Skip("fixture provides no second user")
	}
	ctx := context.Background()

	mine := newConsent(f.UserID, f.AppID, "marketing")
	require.NoError(t, f.Store.GrantConsent(ctx, mine))
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.OtherUserID, f.AppID, "marketing")))

	list, _, err := f.Store.ListConsents(ctx, &consent.Query{UserID: f.UserID, AppID: f.AppID, Limit: 100})
	require.NoError(t, err)

	var found bool
	for _, c := range list {
		assert.Equal(t, f.UserID, c.UserID, "listing leaked another user's consent record")
		assert.Equal(t, f.AppID, c.AppID, "listing leaked another tenant's consent record")
		if c.Purpose == "marketing" {
			found = true
		}
	}
	assert.True(t, found, "listing omitted a record belonging to the queried user")
}

func testListFiltersByPurpose(t *testing.T, f Fixture) {
	ctx := context.Background()
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.UserID, f.AppID, "marketing")))
	require.NoError(t, f.Store.GrantConsent(ctx, newConsent(f.UserID, f.AppID, "analytics")))

	list, _, err := f.Store.ListConsents(ctx, &consent.Query{
		UserID: f.UserID, AppID: f.AppID, Purpose: "analytics", Limit: 100,
	})
	require.NoError(t, err)
	require.NotEmpty(t, list, "the purpose filter dropped a record that matched it")
	for _, c := range list {
		assert.Equal(t, "analytics", c.Purpose, "the purpose filter returned a %q record", c.Purpose)
	}
}
