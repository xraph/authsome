package waitlisttest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/waitlist"
)

func testEntryCRUD(t *testing.T, f Fixture) {
	ctx := context.Background()
	e := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, e))

	got, err := f.Store.GetEntry(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, e.AppID, got.AppID)
	assert.Equal(t, e.Email, got.Email)
	assert.Equal(t, e.Name, got.Name)
	assert.Equal(t, e.Status, got.Status)
	assert.Equal(t, e.IPAddress, got.IPAddress)

	byEmail, err := f.Store.GetEntryByEmail(ctx, f.AppID, e.Email)
	require.NoError(t, err)
	assert.Equal(t, e.ID, byEmail.ID, "email lookup must resolve the same entry")
}

func testEntryNotFound(t *testing.T, f Fixture) {
	ctx := context.Background()

	_, err := f.Store.GetEntry(ctx, id.NewWaitlistID())
	require.Error(t, err)
	assert.True(t, errors.Is(err, waitlist.ErrNotFound),
		"an unknown entry id must return ErrNotFound, got %v", err)

	_, err = f.Store.GetEntryByEmail(ctx, f.AppID, uniqueEmail())
	require.Error(t, err)
	assert.True(t, errors.Is(err, waitlist.ErrNotFound),
		"an unknown email must return ErrNotFound, got %v", err)
}

// testEmailLookupIsAppScoped keeps one app from reading another's signups.
// The waitlist holds an email address and often a name before the person has
// any account, so a leak here is a leak of people who never signed up.
func testEmailLookupIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	email := uniqueEmail()
	theirs := newEntry(f.OtherAppID, email)
	require.NoError(t, f.Store.CreateEntry(ctx, theirs))

	_, err := f.Store.GetEntryByEmail(ctx, f.AppID, email)
	require.Error(t, err, "email lookup crossed a tenant boundary")
	assert.True(t, errors.Is(err, waitlist.ErrNotFound), "got %v", err)
}

// testSameEmailAllowedInDifferentApps is the other half of that boundary. The
// uniqueness constraint is per app, so one person joining two products'
// waitlists must not be rejected the second time.
func testSameEmailAllowedInDifferentApps(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	email := uniqueEmail()
	mine := newEntry(f.AppID, email)
	require.NoError(t, f.Store.CreateEntry(ctx, mine))

	theirs := newEntry(f.OtherAppID, email)
	assert.NoError(t, f.Store.CreateEntry(ctx, theirs),
		"the same address must be allowed on two different apps' waitlists")

	got, err := f.Store.GetEntryByEmail(ctx, f.AppID, email)
	require.NoError(t, err)
	assert.Equal(t, mine.ID, got.ID, "email lookup returned the wrong tenant's entry")
}

func testDuplicateEmailIsRejected(t *testing.T, f Fixture) {
	if !f.EnforcesUniqueEmail {
		t.Skip("backend declares no unique index on (app_id, email); duplicates are accepted")
	}
	ctx := context.Background()

	email := uniqueEmail()
	require.NoError(t, f.Store.CreateEntry(ctx, newEntry(f.AppID, email)))

	err := f.Store.CreateEntry(ctx, newEntry(f.AppID, email))
	require.Error(t, err, "the same address must not be able to join one waitlist twice")
	assert.True(t, errors.Is(err, waitlist.ErrDuplicateEmail),
		"a repeat signup must report ErrDuplicateEmail so the caller can answer politely, got %v", err)
}

func testUpdateEntryStatus(t *testing.T, f Fixture) {
	ctx := context.Background()
	e := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, e))

	require.NoError(t, f.Store.UpdateEntryStatus(ctx, e.ID, waitlist.StatusApproved, "looks good"))

	got, err := f.Store.GetEntry(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, waitlist.StatusApproved, got.Status, "approval must persist; it is what unlocks sign-up")
	assert.Equal(t, "looks good", got.Note)
	assert.Equal(t, e.Email, got.Email, "changing status must not disturb the entry")
}

// testUnsetUserIDStaysUnset pins the nullable pointer. UserID is set only
// once the person actually signs up, so an entry that gains a zero-valued id
// looks converted when it is not.
func testUnsetUserIDStaysUnset(t *testing.T, f Fixture) {
	ctx := context.Background()
	e := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, e))

	got, err := f.Store.GetEntry(ctx, e.ID)
	require.NoError(t, err)
	if got.UserID != nil {
		assert.True(t, got.UserID.IsNil(),
			"an entry that has not converted carried a real user id of %v", got.UserID)
	}
}

func testListIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	mine := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, mine))
	theirs := newEntry(f.OtherAppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, theirs))

	list, err := f.Store.ListEntries(ctx, &waitlist.WaitlistQuery{AppID: f.AppID, Limit: 100})
	require.NoError(t, err)

	var found bool
	for _, e := range list.Entries {
		assert.Equal(t, f.AppID, e.AppID, "listing leaked another tenant's waitlist entry")
		if e.ID == mine.ID {
			found = true
		}
	}
	assert.True(t, found, "listing omitted an entry belonging to the queried app")
}

func testListFiltersByStatus(t *testing.T, f Fixture) {
	ctx := context.Background()

	pending := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, pending))
	approved := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, approved))
	require.NoError(t, f.Store.UpdateEntryStatus(ctx, approved.ID, waitlist.StatusApproved, ""))

	list, err := f.Store.ListEntries(ctx, &waitlist.WaitlistQuery{
		AppID:  f.AppID,
		Status: waitlist.StatusApproved,
		Limit:  100,
	})
	require.NoError(t, err)

	var sawApproved bool
	for _, e := range list.Entries {
		assert.Equal(t, waitlist.StatusApproved, e.Status, "status filter returned a %s entry", e.Status)
		if e.ID == approved.ID {
			sawApproved = true
		}
	}
	assert.True(t, sawApproved, "status filter dropped an entry that matched it")
}

func testCountByStatusIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	basePending, baseApproved, baseRejected, err := f.Store.CountByStatus(ctx, f.AppID)
	require.NoError(t, err)

	// Two for us, one approved; one for the other tenant, which must not move
	// our numbers.
	a := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, a))
	b := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, b))
	require.NoError(t, f.Store.UpdateEntryStatus(ctx, b.ID, waitlist.StatusApproved, ""))
	require.NoError(t, f.Store.CreateEntry(ctx, newEntry(f.OtherAppID, uniqueEmail())))

	pending, approved, rejected, err := f.Store.CountByStatus(ctx, f.AppID)
	require.NoError(t, err)
	assert.Equal(t, basePending+1, pending, "pending count is wrong or is counting another tenant")
	assert.Equal(t, baseApproved+1, approved, "approved count is wrong or is counting another tenant")
	assert.Equal(t, baseRejected, rejected, "rejected count moved without a rejection")
}

func testDeleteEntry(t *testing.T, f Fixture) {
	ctx := context.Background()
	e := newEntry(f.AppID, uniqueEmail())
	require.NoError(t, f.Store.CreateEntry(ctx, e))
	require.NoError(t, f.Store.DeleteEntry(ctx, e.ID))

	_, err := f.Store.GetEntry(ctx, e.ID)
	assert.Error(t, err, "a deleted entry must stop resolving")

	// And the address must be free to join again, or a removal leaves the
	// person permanently unable to sign up.
	assert.NoError(t, f.Store.CreateEntry(ctx, newEntry(f.AppID, e.Email)),
		"deleting an entry must release its email address")
}
