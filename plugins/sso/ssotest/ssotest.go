// Package ssotest provides a backend-agnostic conformance suite for the
// sso.Store interface. Every backend (memory, sqlite, postgres, mongo) is
// expected to pass the same contract, so behavioral drift between them is
// caught here rather than in production.
//
// SSO connections route by domain, so most of the risk in this store is
// tenancy: a domain lookup that forgets its app predicate hands one tenant's
// users to another tenant's identity provider.
package ssotest

import (
	"fmt"
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sso"
)

// Fixture is one backend ready to test, plus the tenants its rows hang off.
type Fixture struct {
	Store sso.Store
	AppID id.AppID
	// EnvID is the app's default environment. Postgres declares a foreign key
	// on it; memory, sqlite and mongo do not, so a fixture that omits it
	// passes on three backends and fails on the fourth.
	EnvID string
	// OtherAppID is a second tenant, used to prove that domain, provider and
	// listing lookups are all scoped to one app.
	OtherAppID id.AppID
	// OtherEnvID is that second tenant's default environment.
	OtherEnvID string
	// OrgID and OtherOrgID are two organizations inside AppID. Connections
	// route per organization as well as per app, so proving that lookup needs
	// two orgs under one tenant rather than two tenants. org_id carries no
	// foreign key on any backend, so these need no row of their own.
	OrgID      id.OrgID
	OtherOrgID id.OrgID
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
		{"DomainLookupIsAppScoped", testDomainLookupIsAppScoped},
		{"DomainLookupIsOrgScoped", testDomainLookupIsOrgScoped},
		{"ProviderLookupIsAppScoped", testProviderLookupIsAppScoped},
		{"ListConnectionsIsAppScoped", testListConnectionsIsAppScoped},
		{"SAMLFieldsRoundTrip", testSAMLFieldsRoundTrip},
		{"AttributeMappingsRoundTrip", testAttributeMappingsRoundTrip},
		{"EmptyAttributeMappingsRoundTrip", testEmptyAttributeMappingsRoundTrip},
		{"SecretsRoundTrip", testSecretsRoundTrip},
		{"UpdateConnection", testUpdateConnection},
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
// never collide on domain or provider.
func unique(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, id.NewSSOConnectionID().String())
}

func newConnection(appID id.AppID, envID string) *sso.Connection {
	return &sso.Connection{
		ID:       id.NewSSOConnectionID(),
		AppID:    appID,
		EnvID:    envID,
		Provider: unique("okta"),
		Protocol: "oidc",
		Domain:   unique("example") + ".test",
		Issuer:   "https://idp.example.test",
		ClientID: "client-abc",
		Active:   true,

		CreatedAt: now(),
		UpdatedAt: now(),
	}
}
