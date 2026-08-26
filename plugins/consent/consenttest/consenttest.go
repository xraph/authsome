// Package consenttest provides a backend-agnostic conformance suite for the
// consent.Store interface. Every backend (memory, sqlite, postgres, mongo) is
// expected to pass the same contract, so behavioral drift between them is
// caught here rather than in production.
//
// Consent records are the evidence that someone agreed to something, so the
// cases lean on the parts a regulator would ask about: that a grant for one
// purpose does not imply another, that revocation is recorded with a time,
// and that one tenant cannot read another's record of who agreed to what.
package consenttest

import (
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/consent"
)

// Fixture is one backend ready to test, plus the rows its records hang off.
type Fixture struct {
	Store  consent.Store
	AppID  id.AppID
	UserID id.UserID
	// OtherUserID is a second user under the same app, and OtherAppID a
	// second tenant. Consent is keyed on user, app and purpose together, so
	// both boundaries need proving.
	OtherUserID  id.UserID
	OtherAppID   id.AppID
	OtherAppUser id.UserID
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
		{"GrantAndGet", testGrantAndGet},
		{"ConsentNotFound", testConsentNotFound},
		{"GrantIsUpsertNotDuplicate", testGrantIsUpsertNotDuplicate},
		{"PurposesAreIndependent", testPurposesAreIndependent},
		{"ActiveConsentHasNoRevokedAt", testActiveConsentHasNoRevokedAt},
		{"RevokeRecordsTimeAndClearsGrant", testRevokeRecordsTimeAndClearsGrant},
		{"RevokeOnlyTheNamedPurpose", testRevokeOnlyTheNamedPurpose},
		{"RegrantAfterRevoke", testRegrantAfterRevoke},
		{"LookupIsScopedToUser", testLookupIsScopedToUser},
		{"LookupIsScopedToApp", testLookupIsScopedToApp},
		{"ListIsScopedToUserAndApp", testListIsScopedToUserAndApp},
		{"ListFiltersByPurpose", testListFiltersByPurpose},
	}
	for _, tc := range cases {
		if skipSet[tc.name] {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newFixture(t)) })
	}
}

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

func newConsent(userID id.UserID, appID id.AppID, purpose string) *consent.Consent {
	return &consent.Consent{
		ID:        id.NewConsentID(),
		UserID:    userID,
		AppID:     appID,
		Purpose:   purpose,
		Granted:   true,
		Version:   "2026-01-01",
		IPAddress: "203.0.113.9",
		GrantedAt: now(),
		CreatedAt: now(),
		UpdatedAt: now(),
	}
}
