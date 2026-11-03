package oauth2provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

func TestClientService_CrossAppIsNotFound(t *testing.T) {
	p, st, _ := newFixture(t)
	ctx := context.Background()
	appA, appB := id.NewAppID(), id.NewAppID()
	b := seedClient(t, st, appB, "client-of-b")
	bID := b.ID.String()

	_, err := p.GetClientFor(ctx, appA, bID)
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)

	name := "hijacked"
	_, err = p.UpdateClientFor(ctx, appA, bID, oauth2provider.UpdateClientRequest{Name: &name})
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)

	_, err = p.RotateClientSecretFor(ctx, appA, bID)
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)

	assert.ErrorIs(t, p.DeleteClientFor(ctx, appA, bID), oauth2provider.ErrClientNotFound)

	list, err := p.ListClientsFor(ctx, appA)
	require.NoError(t, err)
	assert.Empty(t, list, "app A sees none of app B's clients")

	// Nothing changed, and the owner still sees it.
	got, err := p.GetClientFor(ctx, appB, bID)
	require.NoError(t, err)
	assert.Equal(t, "Seeded client-of-b", got.Name)
	list, err = p.ListClientsFor(ctx, appB)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, p.DeleteClientFor(ctx, appB, bID))
	_, err = p.GetClientFor(ctx, appB, bID)
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)
}

func TestClientService_CreateAndValidate(t *testing.T) {
	p, _, _ := newFixture(t)
	ctx := context.Background()
	app := id.NewAppID()

	_, err := p.CreateClientFor(ctx, app, oauth2provider.CreateClientRequest{Name: "x"})
	assert.Error(t, err, "a confidential client needs redirect URIs")

	resp, err := p.CreateClientFor(ctx, app, oauth2provider.CreateClientRequest{
		Name: "x", RedirectURIs: []string{registeredURI}, DPoPMode: "required",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ClientSecret)
	assert.Equal(t, []string{"authorization_code"}, resp.GrantTypes)

	got, err := p.GetClientFor(ctx, app, resp.ID)
	require.NoError(t, err)
	assert.Equal(t, "required", got.DPoPMode)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(got.ClientSecret), []byte(resp.ClientSecret)))

	bad := []string{"javascript:alert(1)"}
	_, err = p.UpdateClientFor(ctx, app, resp.ID, oauth2provider.UpdateClientRequest{RedirectURIs: &bad})
	assert.Error(t, err)
}

func TestClientService_RotateReturnsNewSecretOnce(t *testing.T) {
	p, _, _ := newFixture(t)
	ctx := context.Background()
	app := id.NewAppID()

	created, err := p.CreateClientFor(ctx, app, oauth2provider.CreateClientRequest{
		Name: "x", RedirectURIs: []string{registeredURI},
	})
	require.NoError(t, err)

	secret, err := p.RotateClientSecretFor(ctx, app, created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.NotEqual(t, created.ClientSecret, secret)

	got, err := p.GetClientFor(ctx, app, created.ID)
	require.NoError(t, err)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(got.ClientSecret), []byte(secret)), "the new secret is stored hashed")
	assert.Error(t, bcrypt.CompareHashAndPassword([]byte(got.ClientSecret), []byte(created.ClientSecret)), "the old one stops working")
	assert.NotContains(t, got.ClientSecret, secret, "the clear secret is not stored")

	// A public client has no secret to rotate.
	pub, err := p.CreateClientFor(ctx, app, oauth2provider.CreateClientRequest{Name: "p", Public: true})
	require.NoError(t, err)
	_, err = p.RotateClientSecretFor(ctx, app, pub.ID)
	assert.Error(t, err)
}

func TestRotateSigningKey_OperatorKeyRefused(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{IDTokenSigningKey: testSigningKeyPEM(t)}, true)
	_, err := f.p.RotateSigningKey(context.Background(), f.appID)
	assert.ErrorIs(t, err, oauth2provider.ErrOperatorKey, "the app is on the deployment issuer, served by the operator key")
}

// The deployment issuer's managed key signs for every app that has not
// opted into its own issuer, so no single app's dashboard may rotate it.
func TestRotateSigningKey_SharedDeploymentKeyRefused(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	ctx := context.Background()
	decodeTokenBody(t, f.codeGrant(t, publicID, "openid", "n-shared", time.Now()))
	_, before, err := f.p.SigningKeys(ctx, f.appID)
	require.NoError(t, err)
	require.Len(t, before, 1)

	_, err = f.p.RotateSigningKey(ctx, f.appID)
	require.ErrorIs(t, err, oauth2provider.ErrSharedKey)

	_, after, err := f.p.SigningKeys(ctx, f.appID)
	require.NoError(t, err)
	assert.Equal(t, before, after, "nothing was rotated")
}

// An app's own key can be rotated from the dashboard at most once an hour.
func TestRotateSigningKey_AppKeyTooNewRefused(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	setAppIssuer(t, f, f.appID, true)
	ctx := context.Background()

	_, err := f.p.RotateSigningKey(ctx, f.appID)
	require.NoError(t, err, "the app has no key yet, so the first one is not too new")
	_, err = f.p.RotateSigningKey(ctx, f.appID)
	require.ErrorIs(t, err, oauth2provider.ErrKeyTooNew)

	_, keys, err := f.p.SigningKeys(ctx, f.appID)
	require.NoError(t, err)
	assert.Len(t, keys, 1, "the refused rotation added no key")
}

func TestRotateSigningKey_ManagedKeyUsesThumbprint(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	setAppIssuer(t, f, f.appID, true)
	ctx := context.Background()

	_, before, err := f.p.SigningKeys(ctx, f.appID)
	require.NoError(t, err)

	info, err := f.p.RotateSigningKey(ctx, f.appID)
	require.NoError(t, err)
	assert.Equal(t, "RS256", info.Alg)
	assert.Equal(t, "active", info.Status)

	_, after, err := f.p.SigningKeys(ctx, f.appID)
	require.NoError(t, err)
	assert.Len(t, after, len(before)+1)
	assert.Equal(t, info.KID, after[0].KID, "the returned KID is the one the dashboard lists and the JWKS publishes")

	rec := getPath(t, f.mux, "/apps/"+platformSlug+"/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var doc jwksDoc
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	assert.Contains(t, kids(doc), info.KID)
}

func TestIssuerInfo_AppOptIn(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	ctx := context.Background()

	info, err := f.p.IssuerInfo(ctx, f.appID)
	require.NoError(t, err)
	assert.Equal(t, oidcIssuer, info.Issuer)
	assert.False(t, info.AppIssuer)
	assert.Equal(t, oidcIssuer+"/.well-known/openid-configuration", info.DiscoveryURL)
	assert.Equal(t, oidcIssuer+"/v1/oauth/jwks", info.JWKSURI)
	assert.NotEmpty(t, info.Algs)

	setAppIssuer(t, f, f.appID, true)
	info, err = f.p.IssuerInfo(ctx, f.appID)
	require.NoError(t, err)
	assert.Equal(t, platformIssuer, info.Issuer)
	assert.True(t, info.AppIssuer)
	assert.Equal(t, platformIssuer+"/.well-known/openid-configuration", info.DiscoveryURL)
	assert.Equal(t, platformIssuer+"/v1/oauth/jwks", info.JWKSURI)
	assert.Equal(t, []string{"RS256"}, info.Algs)

	issuer, _, err := f.p.SigningKeys(ctx, f.appID)
	require.NoError(t, err)
	assert.Equal(t, platformIssuer, issuer)
}
