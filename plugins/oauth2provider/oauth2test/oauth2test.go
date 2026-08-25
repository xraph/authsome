// Package oauth2test provides a backend-agnostic conformance suite for the
// oauth2provider.Store interface. Every backend (memory, sqlite, postgres,
// mongo) is expected to pass the same contract, so behavioral drift between
// implementations is caught here rather than in production.
//
// It mirrors store/storetest, which does the same job for the core store.
package oauth2test

import (
	"fmt"
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// Fixture is one backend ready to test: a migrated plugin store plus the
// tenant rows its foreign keys point at. The SQL backends reference
// authsome_apps and authsome_users, so a suite cannot invent those ids.
type Fixture struct {
	Store oauth2provider.Store
	AppID id.AppID
	// UserID must exist in authsome_users for backends that enforce the
	// auth-code foreign key.
	UserID id.UserID
	// OtherAppID is a second tenant, used to prove app scoping. Backends that
	// cannot seed one may leave it zero and the scoping case is skipped.
	OtherAppID id.AppID
}

// Factory builds a fresh, empty, migrated fixture for a single test.
type Factory func(t *testing.T) Fixture

// RunConformance runs every contract test against fixtures from newFixture.
// Names in skip are not run, for backends that legitimately cannot support a
// case; skipping anything else is a bug in the backend, not in the suite.
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
		{"ClientCRUD", testClientCRUD},
		{"ClientNotFound", testClientNotFound},
		{"ClientListIsAppScoped", testClientListIsAppScoped},
		{"ClientEmptySlicesRoundTrip", testClientEmptySlicesRoundTrip},
		{"ClientSlicesRoundTrip", testClientSlicesRoundTrip},
		{"UpdateClientPersistsEveryField", testUpdateClientPersistsEveryField},
		{"UpdateClientOnMissingClient", testUpdateClientOnMissingClient},
		{"AuthCodeRoundTrip", testAuthCodeRoundTrip},
		{"ConsumeAuthCodeOnce", testConsumeAuthCodeOnce},
		{"ConsumeAuthCodeReplay", testConsumeAuthCodeReplay},
		{"ConsumeAuthCodeIsAtomic", testConsumeAuthCodeIsAtomic},
		{"ConsumeAuthCodeUnknown", testConsumeAuthCodeUnknown},
		{"DeviceCodeRoundTrip", testDeviceCodeRoundTrip},
		{"DeviceCodeUpdate", testDeviceCodeUpdate},
		{"DeleteExpiredDeviceCodes", testDeleteExpiredDeviceCodes},
	}
	for _, tc := range cases {
		if skipSet[tc.name] {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newFixture(t)) })
	}
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

// unique returns a per-call unique string so sub-tests sharing one backend
// (mongo and postgres reuse a single container) never collide.
func unique(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, id.NewOAuth2ClientID().String())
}

func newClient(appID id.AppID) *oauth2provider.OAuth2Client {
	return &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		Name:         "Test Client",
		ClientID:     unique("cid"),
		ClientSecret: "hashed-secret",
		RedirectURIs: []string{"https://example.test/cb"},
		Scopes:       []string{"openid", "profile"},
		GrantTypes:   []string{"authorization_code"},
		Resources:    []string{"https://api.example.test"},
		CreatedAt:    now(),
		UpdatedAt:    now(),
	}
}
