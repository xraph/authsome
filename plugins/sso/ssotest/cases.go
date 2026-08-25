package ssotest

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func testConnectionCRUD(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateConnection(ctx, c))

	got, err := f.Store.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.AppID, got.AppID)
	assert.Equal(t, c.Provider, got.Provider)
	assert.Equal(t, c.Protocol, got.Protocol)
	assert.Equal(t, c.Domain, got.Domain)
	assert.Equal(t, c.Issuer, got.Issuer)
	assert.Equal(t, c.ClientID, got.ClientID)
	assert.True(t, got.Active)

	byDomain, err := f.Store.GetConnectionByDomain(ctx, f.AppID, c.Domain)
	require.NoError(t, err)
	assert.Equal(t, c.ID, byDomain.ID, "domain lookup must resolve the same connection")
}

func testConnectionNotFound(t *testing.T, f Fixture) {
	ctx := context.Background()

	_, err := f.Store.GetConnection(ctx, id.NewSSOConnectionID())
	assert.Error(t, err, "an unknown connection id must not resolve")

	_, err = f.Store.GetConnectionByDomain(ctx, f.AppID, unique("absent")+".test")
	assert.Error(t, err, "an unconfigured domain must not resolve")
}

// testDomainLookupIsAppScoped is the one that matters most in this store.
// Domain is how an email address is routed to an identity provider, so a
// lookup missing its app predicate sends one tenant's users to another
// tenant's IdP.
func testDomainLookupIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	domain := unique("shared") + ".test"

	theirs := newConnection(f.OtherAppID, f.OtherEnvID)
	theirs.Domain = domain
	require.NoError(t, f.Store.CreateConnection(ctx, theirs))

	// The other tenant owns this domain and we have no connection for it.
	// Our lookup must come up empty rather than borrowing theirs.
	got, err := f.Store.GetConnectionByDomain(ctx, f.AppID, domain)
	if err == nil {
		assert.Equal(t, f.AppID, got.AppID,
			"domain lookup crossed a tenant boundary and returned app %s", got.AppID)
		assert.NotEqual(t, theirs.ID, got.ID, "domain lookup returned the other tenant's connection")
	}

	// With both tenants configured for the same domain, each must get its own.
	mine := newConnection(f.AppID, f.EnvID)
	mine.Domain = domain
	require.NoError(t, f.Store.CreateConnection(ctx, mine))

	got, err = f.Store.GetConnectionByDomain(ctx, f.AppID, domain)
	require.NoError(t, err)
	assert.Equal(t, mine.ID, got.ID, "domain lookup returned the wrong tenant's connection")
}

func testProviderLookupIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	provider := unique("provider")

	theirs := newConnection(f.OtherAppID, f.OtherEnvID)
	theirs.Provider = provider
	require.NoError(t, f.Store.CreateConnection(ctx, theirs))

	mine := newConnection(f.AppID, f.EnvID)
	mine.Provider = provider
	require.NoError(t, f.Store.CreateConnection(ctx, mine))

	got, err := f.Store.GetConnectionByProvider(ctx, f.AppID, provider)
	require.NoError(t, err)
	assert.Equal(t, mine.ID, got.ID, "provider lookup returned the wrong tenant's connection")
	assert.Equal(t, f.AppID, got.AppID)
}

func testListConnectionsIsAppScoped(t *testing.T, f Fixture) {
	if f.OtherAppID.IsNil() {
		t.Skip("fixture provides no second tenant")
	}
	ctx := context.Background()

	mine := newConnection(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateConnection(ctx, mine))
	theirs := newConnection(f.OtherAppID, f.OtherEnvID)
	require.NoError(t, f.Store.CreateConnection(ctx, theirs))

	got, err := f.Store.ListConnections(ctx, f.AppID)
	require.NoError(t, err)

	var found bool
	for _, c := range got {
		assert.Equal(t, f.AppID, c.AppID, "listing leaked another tenant's connection")
		if c.ID == mine.ID {
			found = true
		}
	}
	assert.True(t, found, "listing omitted a connection belonging to the queried app")
}

// testSAMLFieldsRoundTrip pushes the bulky SAML fields through storage. IdP
// metadata is multi-kilobyte XML and certificates carry embedded newlines,
// so a column that truncates or a serializer that trims whitespace breaks
// signature validation in a way that is painful to trace back here.
func testSAMLFieldsRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.AppID, f.EnvID)
	c.Protocol = "saml"
	c.IDPMetadataXML = `<?xml version="1.0"?><EntityDescriptor entityID="https://idp.example.test">` +
		strings.Repeat(`<KeyDescriptor use="signing"><X509Certificate>AAAA</X509Certificate></KeyDescriptor>`, 40) +
		`</EntityDescriptor>`
	c.IDPCertificate = "-----BEGIN CERTIFICATE-----\nMIIBoTCCAUugAwIBAgI\n+/=abcDEF\n-----END CERTIFICATE-----\n"
	c.IDPSSOURL = "https://idp.example.test/sso?foo=bar&baz=qux"
	c.EntityID = "https://sp.example.test/metadata"
	c.ACSURL = "https://sp.example.test/acs"
	c.SignRequests = true
	require.NoError(t, f.Store.CreateConnection(ctx, c))

	got, err := f.Store.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.IDPMetadataXML, got.IDPMetadataXML, "IdP metadata XML was altered in storage")
	assert.Equal(t, c.IDPCertificate, got.IDPCertificate, "the certificate lost its formatting; PEM is newline sensitive")
	assert.Equal(t, c.IDPSSOURL, got.IDPSSOURL)
	assert.Equal(t, c.EntityID, got.EntityID)
	assert.Equal(t, c.ACSURL, got.ACSURL)
	assert.True(t, got.SignRequests)
}

func testAttributeMappingsRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.AppID, f.EnvID)
	c.AttributeMappings = map[string]string{
		"email":      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
		"first_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
		"groups":     "memberOf",
	}
	require.NoError(t, f.Store.CreateConnection(ctx, c))

	got, err := f.Store.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.AttributeMappings, got.AttributeMappings,
		"attribute mappings decide which IdP claim becomes which user field; losing one silently drops that field")
}

func testEmptyAttributeMappingsRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.AppID, f.EnvID)
	c.AttributeMappings = nil
	require.NoError(t, f.Store.CreateConnection(ctx, c))

	got, err := f.Store.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	assert.Empty(t, got.AttributeMappings, "a connection with no mappings must not gain phantom entries")
}

// testSecretsRoundTrip covers the two fields that never appear in JSON. They
// are the ones least likely to be noticed if a backend mangles them, because
// nothing echoes them back to a caller.
func testSecretsRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.AppID, f.EnvID)
	c.ClientSecret = "cs_" + strings.Repeat("x9Z/+", 20)
	c.SPPrivateKey = "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0\n-----END PRIVATE KEY-----\n"
	require.NoError(t, f.Store.CreateConnection(ctx, c))

	got, err := f.Store.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.ClientSecret, got.ClientSecret, "the client secret must survive byte for byte")
	assert.Equal(t, c.SPPrivateKey, got.SPPrivateKey, "the SP private key must survive byte for byte")
}

func testUpdateConnection(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateConnection(ctx, c))

	c.Active = false
	c.Issuer = "https://idp2.example.test"
	c.UpdatedAt = now()
	require.NoError(t, f.Store.UpdateConnection(ctx, c))

	got, err := f.Store.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	assert.False(t, got.Active, "deactivating a connection must persist; it is how an IdP gets switched off")
	assert.Equal(t, "https://idp2.example.test", got.Issuer)
	assert.Equal(t, c.Domain, got.Domain, "updating one field must not disturb the others")
}

func testDeleteConnection(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConnection(f.AppID, f.EnvID)
	require.NoError(t, f.Store.CreateConnection(ctx, c))
	require.NoError(t, f.Store.DeleteConnection(ctx, c.ID))

	_, err := f.Store.GetConnection(ctx, c.ID)
	assert.Error(t, err, "a deleted connection must stop resolving")

	_, err = f.Store.GetConnectionByDomain(ctx, f.AppID, c.Domain)
	assert.Error(t, err, "a deleted connection must stop resolving by domain too")
}
