// Package waitlisttest provides a backend-agnostic conformance suite for the
// waitlist.Store interface. Every backend (memory, sqlite, postgres, mongo)
// is expected to pass the same contract, so behavioral drift between them is
// caught here rather than in production.
package waitlisttest

import (
	"fmt"
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/waitlist"
)

// Fixture is one backend ready to test, plus the tenants its rows hang off.
type Fixture struct {
	Store waitlist.Store
	AppID id.AppID
	// OtherAppID is a second tenant. Waitlist entries are keyed by email
	// within an app, so the interesting cases all involve two apps.
	OtherAppID id.AppID
	// EnforcesUniqueEmail says whether this backend rejects a second entry
	// for the same address in the same app. Every backend does now, so every
	// backend sets it. The flag stays because it is what made the gap visible
	// as a skip while mongo was missing the index, and a future backend
	// arriving without one should report the same way rather than passing
	// a case it never ran.
	EnforcesUniqueEmail bool
}

// Factory builds a fresh, empty, migrated fixture for a single test.
type Factory func(t *testing.T) Fixture

// RunConformance runs every contract test against fixtures from newFixture.
func RunConformance(t *testing.T, newFixture Factory, skip ...string) {
	t.Helper()
	skipSet := make(map[string]bool, len(skip))
	for _, name := range skip {
		skipSet[name] = true
	}
	cases := []struct {
		name string
		fn   func(t *testing.T, f Fixture)
	}{
		{"EntryCRUD", testEntryCRUD},
		{"EntryNotFound", testEntryNotFound},
		{"EmailLookupIsAppScoped", testEmailLookupIsAppScoped},
		{"SameEmailAllowedInDifferentApps", testSameEmailAllowedInDifferentApps},
		{"DuplicateEmailIsRejected", testDuplicateEmailIsRejected},
		{"UpdateEntryStatus", testUpdateEntryStatus},
		{"UnsetUserIDStaysUnset", testUnsetUserIDStaysUnset},
		{"ListIsAppScoped", testListIsAppScoped},
		{"ListFiltersByStatus", testListFiltersByStatus},
		{"CountByStatusIsAppScoped", testCountByStatusIsAppScoped},
		{"DeleteEntry", testDeleteEntry},
	}
	for _, tc := range cases {
		if skipSet[tc.name] {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newFixture(t)) })
	}
}

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

// uniqueEmail returns a per-call unique address, so sub-tests sharing one
// backend never collide on the email uniqueness constraint.
func uniqueEmail() string {
	return fmt.Sprintf("wl-%s@example.test", id.NewWaitlistID().String())
}

func newEntry(appID id.AppID, email string) *waitlist.WaitlistEntry {
	return &waitlist.WaitlistEntry{
		ID:        id.NewWaitlistID(),
		AppID:     appID,
		Email:     email,
		Name:      "Test Person",
		Status:    waitlist.StatusPending,
		IPAddress: "203.0.113.7",
		CreatedAt: now(),
		UpdatedAt: now(),
	}
}
