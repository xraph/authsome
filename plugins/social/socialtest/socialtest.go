// Package socialtest provides a backend-agnostic conformance suite for the
// social.Store interface. Every backend (memory, sqlite, postgres, mongo) is
// expected to pass the same contract, so behavioral drift between them is
// caught here rather than in production.
//
// An OAuth connection holds the provider tokens that stand in for a user's
// password at Google or GitHub, so the cases lean on whether those survive a
// write and whether a refresh actually replaces them.
package socialtest

import (
	"fmt"
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/social"
)

// Fixture is one backend ready to test, plus the rows its connections hang
// off.
type Fixture struct {
	Store  social.Store
	AppID  id.AppID
	UserID id.UserID
	// OtherUserID is a second user under the same app, used to prove that
	// listing a user's connections does not reach past them.
	OtherUserID id.UserID
	// OtherAppID and OtherAppUser are a second tenant.
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
		{"ConnectionCRUD", testConnectionCRUD},
		{"ConnectionNotFound", testConnectionNotFound},
		{"TokensRoundTrip", testTokensRoundTrip},
		{"ExpiryRoundTrip", testExpiryRoundTrip},
		{"MetadataRoundTrip", testMetadataRoundTrip},
		{"EmptyMetadataRoundTrip", testEmptyMetadataRoundTrip},
		{"ProviderLookupMatchesTheKeys", testProviderLookupMatchesTheKeys},
		{"ListByUserIsScopedToUser", testListByUserIsScopedToUser},
		{"UpdateReplacesTokens", testUpdateReplacesTokens},
		{"DeleteConnection", testDeleteConnection},
	}
	for _, tc := range cases {
		if skipSet[tc.name] {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newFixture(t)) })
	}
}

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

// unique returns a per-call unique fragment, so sub-tests sharing one backend
// never collide on a provider user id.
func unique(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, id.NewOAuthConnectionID().String())
}

func newConnection(userID id.UserID, appID id.AppID) *social.OAuthConnection {
	return &social.OAuthConnection{
		ID:             id.NewOAuthConnectionID(),
		AppID:          appID,
		UserID:         userID,
		Provider:       "google",
		ProviderUserID: unique("puid"),
		Email:          "social-conformance@example.test",
		AccessToken:    "ya29." + unique("at"),
		RefreshToken:   "1//" + unique("rt"),
		ExpiresAt:      now().Add(time.Hour),
		CreatedAt:      now(),
		UpdatedAt:      now(),
	}
}
