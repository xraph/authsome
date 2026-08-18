package sso

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/id"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

// The SQLite store shares fromConnection/toConnection + ssoConnectionModel with
// the Postgres store, so running the conformance suite against embedded SQLite
// exercises the same SQL column mapping (including the SAML fields and the JSON
// attribute_mappings round-trip) that Postgres uses — without needing Docker.

func newMemoryConformanceStore(_ *testing.T) Store { return NewMemoryStore() }

func newSQLiteConformanceStore(t *testing.T) Store {
	t.Helper()
	ctx := context.Background()
	// No foreign_keys pragma: the connections table FK-references authsome_apps,
	// and this suite exercises the SSO store in isolation without seeding the
	// full authsome schema.
	dsn := "file:" + filepath.Join(t.TempDir(), "sso-conformance.db") + "?cache=shared"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// core migrations satisfy the SSO group's DependsOn("authsome"); the SSO
	// group creates authsome_sso_connections with the full column set.
	require.NoError(t, sqlitestore.New(db).Migrate(ctx, SqliteMigrations))
	return NewSqliteStore(db)
}

func TestStoreConformance_Memory(t *testing.T) { runStoreConformance(t, newMemoryConformanceStore) }
func TestStoreConformance_SQLite(t *testing.T) { runStoreConformance(t, newSQLiteConformanceStore) }

func samlConnection(appID id.AppID, orgID id.OrgID, domain string) *Connection {
	return &Connection{
		ID:                id.NewSSOConnectionID(),
		AppID:             appID,
		EnvID:             id.NewEnvironmentID().String(),
		OrgID:             orgID,
		Provider:          domain,
		Protocol:          "saml",
		Domain:            domain,
		Active:            true,
		MetadataURL:       "https://idp." + domain + "/metadata",
		IDPMetadataXML:    "<EntityDescriptor/>",
		IDPSSOURL:         "https://idp." + domain + "/sso",
		IDPCertificate:    "-----BEGIN CERTIFICATE-----idp",
		EntityID:          "https://sp." + domain + "/entity",
		ACSURL:            "https://sp." + domain + "/acs",
		SPCertificate:     "-----BEGIN CERTIFICATE-----sp",
		SPPrivateKey:      "-----BEGIN PRIVATE KEY-----sp",
		SignRequests:      true,
		AttributeMappings: map[string]string{"email": "mail", "groups": "memberOf"},
	}
}

func oidcConnection(appID id.AppID, orgID id.OrgID, domain string) *Connection {
	return &Connection{
		ID:           id.NewSSOConnectionID(),
		AppID:        appID,
		EnvID:        id.NewEnvironmentID().String(),
		OrgID:        orgID,
		Provider:     domain,
		Protocol:     "oidc",
		Domain:       domain,
		Active:       true,
		Issuer:       "https://" + domain,
		ClientID:     "client-" + domain,
		ClientSecret: "secret-" + domain,
	}
}

// runStoreConformance exercises the full sso.Store contract against a backend.
func runStoreConformance(t *testing.T, newStore func(t *testing.T) Store) {
	ctx := context.Background()
	appA := id.NewAppID()
	appB := id.NewAppID()
	orgID := id.NewOrgID()

	t.Run("SAML round-trip preserves every field", func(t *testing.T) {
		s := newStore(t)
		in := samlConnection(appA, orgID, "acme.com")
		require.NoError(t, s.CreateConnection(ctx, in))

		got, err := s.GetConnection(ctx, in.ID)
		require.NoError(t, err)
		assert.Equal(t, in.ID.String(), got.ID.String())
		assert.Equal(t, in.AppID.String(), got.AppID.String())
		assert.Equal(t, in.OrgID.String(), got.OrgID.String())
		assert.Equal(t, in.EnvID, got.EnvID)
		assert.Equal(t, "saml", got.Protocol)
		assert.Equal(t, "acme.com", got.Domain)
		assert.Equal(t, in.Provider, got.Provider)
		assert.True(t, got.Active)
		// SAML-specific fields — the ones the SQL model must map.
		assert.Equal(t, in.MetadataURL, got.MetadataURL)
		assert.Equal(t, in.IDPMetadataXML, got.IDPMetadataXML)
		assert.Equal(t, in.IDPSSOURL, got.IDPSSOURL)
		assert.Equal(t, in.IDPCertificate, got.IDPCertificate)
		assert.Equal(t, in.EntityID, got.EntityID)
		assert.Equal(t, in.ACSURL, got.ACSURL)
		assert.Equal(t, in.SPCertificate, got.SPCertificate)
		assert.Equal(t, in.SPPrivateKey, got.SPPrivateKey)
		assert.True(t, got.SignRequests)
		// attribute_mappings round-trips through JSON in the SQL stores.
		assert.Equal(t, in.AttributeMappings, got.AttributeMappings)
	})

	t.Run("OIDC round-trip preserves secrets", func(t *testing.T) {
		s := newStore(t)
		in := oidcConnection(appA, orgID, "beta.com")
		require.NoError(t, s.CreateConnection(ctx, in))

		got, err := s.GetConnection(ctx, in.ID)
		require.NoError(t, err)
		assert.Equal(t, "oidc", got.Protocol)
		assert.Equal(t, in.Issuer, got.Issuer)
		assert.Equal(t, in.ClientID, got.ClientID)
		assert.Equal(t, in.ClientSecret, got.ClientSecret)
	})

	t.Run("lookup by domain and provider returns active only", func(t *testing.T) {
		s := newStore(t)
		in := samlConnection(appA, orgID, "gamma.com")
		require.NoError(t, s.CreateConnection(ctx, in))

		byDomain, err := s.GetConnectionByDomain(ctx, appA, "gamma.com")
		require.NoError(t, err)
		assert.Equal(t, in.ID.String(), byDomain.ID.String())

		byProvider, err := s.GetConnectionByProvider(ctx, appA, in.Provider)
		require.NoError(t, err)
		assert.Equal(t, in.ID.String(), byProvider.ID.String())

		// Deactivate → lookups by domain/provider no longer resolve it.
		in.Active = false
		require.NoError(t, s.UpdateConnection(ctx, in))
		_, err = s.GetConnectionByDomain(ctx, appA, "gamma.com")
		assert.ErrorIs(t, err, ErrConnectionNotFound)
		_, err = s.GetConnectionByProvider(ctx, appA, in.Provider)
		assert.ErrorIs(t, err, ErrConnectionNotFound)
		// …but it is still fetchable by id and via List.
		_, err = s.GetConnection(ctx, in.ID)
		require.NoError(t, err)
	})

	t.Run("same domain resolves per-org and is isolated across orgs", func(t *testing.T) {
		s := newStore(t)
		orgA := id.NewOrgID()
		orgB := id.NewOrgID()
		// The same email domain configured for SSO in two different orgs — the
		// multi-tenant case the org-scoped unique index must allow.
		inA := samlConnection(appA, orgA, "shared.com")
		inB := oidcConnection(appA, orgB, "shared.com")
		require.NoError(t, s.CreateConnection(ctx, inA))
		require.NoError(t, s.CreateConnection(ctx, inB))

		gotA, err := s.GetConnectionByDomainAndOrg(ctx, appA, orgA, "shared.com")
		require.NoError(t, err)
		assert.Equal(t, inA.ID.String(), gotA.ID.String())

		gotB, err := s.GetConnectionByDomainAndOrg(ctx, appA, orgB, "shared.com")
		require.NoError(t, err)
		assert.Equal(t, inB.ID.String(), gotB.ID.String())

		// An org with no connection for that domain resolves to not-found.
		_, err = s.GetConnectionByDomainAndOrg(ctx, appA, id.NewOrgID(), "shared.com")
		assert.ErrorIs(t, err, ErrConnectionNotFound)

		// Deactivating one org's connection removes only that org's row.
		inA.Active = false
		require.NoError(t, s.UpdateConnection(ctx, inA))
		_, err = s.GetConnectionByDomainAndOrg(ctx, appA, orgA, "shared.com")
		assert.ErrorIs(t, err, ErrConnectionNotFound)
		stillB, err := s.GetConnectionByDomainAndOrg(ctx, appA, orgB, "shared.com")
		require.NoError(t, err)
		assert.Equal(t, inB.ID.String(), stillB.ID.String())
	})

	t.Run("list is scoped to the app", func(t *testing.T) {
		s := newStore(t)
		require.NoError(t, s.CreateConnection(ctx, samlConnection(appA, orgID, "one.com")))
		require.NoError(t, s.CreateConnection(ctx, oidcConnection(appA, orgID, "two.com")))
		require.NoError(t, s.CreateConnection(ctx, samlConnection(appB, orgID, "three.com")))

		listA, err := s.ListConnections(ctx, appA)
		require.NoError(t, err)
		assert.Len(t, listA, 2)
		for _, c := range listA {
			assert.Equal(t, appA.String(), c.AppID.String())
		}

		listB, err := s.ListConnections(ctx, appB)
		require.NoError(t, err)
		assert.Len(t, listB, 1)
	})

	t.Run("update mutates stored fields", func(t *testing.T) {
		s := newStore(t)
		in := oidcConnection(appA, orgID, "delta.com")
		require.NoError(t, s.CreateConnection(ctx, in))

		in.ClientSecret = "rotated-secret"
		in.Active = false
		require.NoError(t, s.UpdateConnection(ctx, in))

		got, err := s.GetConnection(ctx, in.ID)
		require.NoError(t, err)
		assert.Equal(t, "rotated-secret", got.ClientSecret)
		assert.False(t, got.Active)
	})

	t.Run("delete removes the connection", func(t *testing.T) {
		s := newStore(t)
		in := samlConnection(appA, orgID, "epsilon.com")
		require.NoError(t, s.CreateConnection(ctx, in))
		require.NoError(t, s.DeleteConnection(ctx, in.ID))

		_, err := s.GetConnection(ctx, in.ID)
		assert.ErrorIs(t, err, ErrConnectionNotFound)
	})

	t.Run("missing connection returns ErrConnectionNotFound", func(t *testing.T) {
		s := newStore(t)
		_, err := s.GetConnection(ctx, id.NewSSOConnectionID())
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrConnectionNotFound))
	})
}
