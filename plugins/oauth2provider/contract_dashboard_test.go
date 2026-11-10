package oauth2provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/forge/extensions/dashboard"
	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/idempotency"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// dashboardFor registers the plugin's contract the way the dashboard does and
// returns a caller that dispatches as an operator of appID.
type dashboardCaller func(kind contract.Kind, intent string, payload any) (json.RawMessage, error)

func dashboardFor(t *testing.T, f *oidcFixture) func(appID id.AppID) dashboardCaller {
	t.Helper()
	d := dispatcher.NewWithOptions(dispatcher.NoopMetricsEmitter{},
		dispatcher.WithIdempotencyStore(dashboard.AdaptIdempotencyStore(idempotency.NewInMemoryStore())))
	require.NoError(t, f.p.RegisterContract(d, contract.NewRegistry(), contract.NewWardenRegistry(), f.eng))
	return func(appID id.AppID) dashboardCaller {
		p := contract.PrincipalFor(&dashauth.UserInfo{Subject: "operator"})
		p.Claims["app_id"] = appID.String()
		return func(kind contract.Kind, intent string, payload any) (json.RawMessage, error) {
			raw, err := json.Marshal(payload)
			require.NoError(t, err)
			data, _, err := d.Dispatch(context.Background(), contract.Request{
				Envelope: "v1", Kind: kind, Contributor: "oauth2provider", Intent: intent, IntentVersion: 1,
				Payload: raw,
			}, p)
			return data, err
		}
	}
}

func requireContractError(t *testing.T, err error, code contract.ErrorCode, msg string) {
	t.Helper()
	var ce *contract.Error
	require.True(t, errors.As(err, &ce), "err = %v, want *contract.Error", err)
	assert.Equal(t, code, ce.Code)
	if msg != "" {
		assert.Equal(t, msg, ce.Message)
	}
}

func decodeInto(t *testing.T, data json.RawMessage, v any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(data, v), string(data))
}

// The dashboard drives the whole client lifecycle through the contract, and
// no answer except create and rotateSecret ever carries a secret, hashed or
// otherwise.
func TestDashboard_ClientLifecycle(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	call := dashboardFor(t, f)(f.appID)
	ctx := context.Background()

	// A confidential client needs a redirect URI.
	_, err := call(contract.KindCommand, "oauth2.clients.create", map[string]any{"name": "Web"})
	requireContractError(t, err, contract.CodeBadRequest, "redirect_uris required for confidential clients")

	data, err := call(contract.KindCommand, "oauth2.clients.create", map[string]any{
		"name": "Web", "redirectUris": []string{registeredURI}, "resources": []string{"https://api.example.com"},
	})
	require.NoError(t, err)
	var created struct{ ID, ClientID, ClientSecret string }
	decodeInto(t, data, &created)
	require.NotEmpty(t, created.ClientSecret)
	stored, err := f.p.GetClientFor(ctx, f.appID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ClientID, stored.ClientID)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.ClientSecret), []byte(created.ClientSecret)))

	// A public client gets no secret at all.
	data, err = call(contract.KindCommand, "oauth2.clients.create", map[string]any{"name": "SPA", "public": true})
	require.NoError(t, err)
	assert.NotContains(t, string(data), "clientSecret")
	var spa struct{ ID string }
	decodeInto(t, data, &spa)

	noHash := func(data json.RawMessage) {
		t.Helper()
		assert.NotContains(t, string(data), stored.ClientSecret, "the secret hash reached the wire")
		assert.NotContains(t, string(data), "$2a$", "a bcrypt hash reached the wire")
		assert.NotContains(t, strings.ToLower(string(data)), `secret"`, "a field named for a secret reached the wire")
	}

	data, err = call(contract.KindQuery, "oauth2.clients.list", struct{}{})
	require.NoError(t, err)
	noHash(data)
	var list struct {
		Clients []struct{ ID, Name string }
	}
	decodeInto(t, data, &list)
	names := map[string]string{}
	for _, c := range list.Clients {
		names[c.ID] = c.Name
	}
	assert.Equal(t, "Web", names[created.ID])
	assert.Equal(t, "SPA", names[spa.ID])

	data, err = call(contract.KindQuery, "oauth2.clients.detail", map[string]any{"id": created.ID})
	require.NoError(t, err)
	noHash(data)
	var detail struct {
		ClientID   string
		Resources  []string
		AuthMethod string
	}
	decodeInto(t, data, &detail)
	assert.Equal(t, created.ClientID, detail.ClientID)
	assert.Equal(t, []string{"https://api.example.com"}, detail.Resources)
	assert.Equal(t, "client_secret_basic", detail.AuthMethod)

	data, err = call(contract.KindCommand, "oauth2.clients.update", map[string]any{
		"id": created.ID, "name": "Web 2", "scopes": []string{"openid"},
	})
	require.NoError(t, err)
	noHash(data)
	stored, err = f.p.GetClientFor(ctx, f.appID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Web 2", stored.Name)
	assert.Equal(t, []string{"openid"}, stored.Scopes)
	assert.Equal(t, []string{registeredURI}, stored.RedirectURIs, "a field the update did not send is kept")

	_, err = call(contract.KindCommand, "oauth2.clients.update", map[string]any{"id": "not-an-id", "name": "x"})
	requireContractError(t, err, contract.CodeBadRequest, "invalid client ID")

	_, err = call(contract.KindCommand, "oauth2.clients.update", map[string]any{"id": created.ID, "grantTypes": []string{"password"}})
	requireContractError(t, err, contract.CodeBadRequest, `grant_type "password" is not supported`)

	oldHash := stored.ClientSecret
	data, err = call(contract.KindCommand, "oauth2.clients.rotateSecret", map[string]any{"id": created.ID})
	require.NoError(t, err)
	var rotated struct{ ID, ClientID, ClientSecret string }
	decodeInto(t, data, &rotated)
	assert.Equal(t, created.ID, rotated.ID)
	assert.Equal(t, created.ClientID, rotated.ClientID)
	stored, err = f.p.GetClientFor(ctx, f.appID, created.ID)
	require.NoError(t, err)
	assert.NotEqual(t, oldHash, stored.ClientSecret)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.ClientSecret), []byte(rotated.ClientSecret)))

	_, err = call(contract.KindCommand, "oauth2.clients.rotateSecret", map[string]any{"id": spa.ID})
	requireContractError(t, err, contract.CodeBadRequest, "a public client has no client_secret to rotate")

	data, err = call(contract.KindCommand, "oauth2.clients.delete", map[string]any{"id": created.ID})
	require.NoError(t, err)
	assert.JSONEq(t, `{"ok":true,"id":"`+created.ID+`"}`, string(data))
	_, err = call(contract.KindQuery, "oauth2.clients.detail", map[string]any{"id": created.ID})
	requireContractError(t, err, contract.CodeNotFound, "oauth2 client not found")
	_, err = call(contract.KindCommand, "oauth2.clients.delete", map[string]any{"id": created.ID})
	requireContractError(t, err, contract.CodeNotFound, "oauth2 client not found")
}

// Another app's operator sees none of this app's clients and cannot touch
// them: every path answers not found, and nothing changes.
func TestDashboard_OtherAppIsNotFound(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	dash := dashboardFor(t, f)
	created, err := f.p.CreateClientFor(context.Background(), f.appID, oauth2provider.CreateClientRequest{
		Name: "Web", RedirectURIs: []string{registeredURI},
	})
	require.NoError(t, err)

	other := dash(id.NewAppID())
	data, err := other(contract.KindQuery, "oauth2.clients.list", struct{}{})
	require.NoError(t, err)
	assert.JSONEq(t, `{"clients":[]}`, string(data))
	for _, c := range []struct {
		kind    contract.Kind
		intent  string
		payload map[string]any
	}{
		{contract.KindQuery, "oauth2.clients.detail", map[string]any{"id": created.ID}},
		{contract.KindCommand, "oauth2.clients.update", map[string]any{"id": created.ID, "name": "hijacked"}},
		{contract.KindCommand, "oauth2.clients.rotateSecret", map[string]any{"id": created.ID}},
		{contract.KindCommand, "oauth2.clients.delete", map[string]any{"id": created.ID}},
	} {
		_, callErr := other(c.kind, c.intent, c.payload)
		requireContractError(t, callErr, contract.CodeNotFound, "oauth2 client not found")
	}
	got, err := f.p.GetClientFor(context.Background(), f.appID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Web", got.Name)
}

func TestDashboard_IssuerAndKeys(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	call := dashboardFor(t, f)(f.appID)

	data, err := call(contract.KindQuery, "oauth2.issuer.info", struct{}{})
	require.NoError(t, err)
	assert.JSONEq(t, `{"issuer":"`+oidcIssuer+`","discoveryUrl":"`+oidcIssuer+`/.well-known/openid-configuration",`+
		`"jwksUri":"`+oidcIssuer+`/v1/oauth/jwks","appIssuer":false,"algs":["RS256"]}`, string(data))

	// The deployment issuer's key is shared, so no app may rotate it.
	_, err = call(contract.KindCommand, "oauth2.keys.rotate", struct{}{})
	requireContractError(t, err, contract.CodeBadRequest,
		"the deployment issuer's signing key is shared by every app and rotates on its own schedule")

	setAppIssuer(t, f, f.appID, true)
	data, err = call(contract.KindCommand, "oauth2.keys.rotate", struct{}{})
	require.NoError(t, err)
	var rotated struct {
		Issuer string
		Keys   []struct{ KID, Alg, Source, Status, CreatedAt string }
	}
	decodeInto(t, data, &rotated)
	assert.Equal(t, platformIssuer, rotated.Issuer)
	require.Len(t, rotated.Keys, 1)
	assert.Equal(t, "RS256", rotated.Keys[0].Alg)
	assert.Equal(t, "managed", rotated.Keys[0].Source)
	assert.Equal(t, "active", rotated.Keys[0].Status)
	_, err = time.Parse(time.RFC3339, rotated.Keys[0].CreatedAt)
	require.NoError(t, err)

	data, err = call(contract.KindQuery, "oauth2.keys.list", struct{}{})
	require.NoError(t, err)
	var listed struct {
		Issuer string
		Keys   []struct{ KID string }
	}
	decodeInto(t, data, &listed)
	assert.Equal(t, platformIssuer, listed.Issuer)
	require.Len(t, listed.Keys, 1)
	assert.Equal(t, rotated.Keys[0].KID, listed.Keys[0].KID)

	_, err = call(contract.KindCommand, "oauth2.keys.rotate", struct{}{})
	requireContractError(t, err, contract.CodeBadRequest, "the current signing key is less than an hour old; try again later")

	data, err = call(contract.KindQuery, "oauth2.issuer.info", struct{}{})
	require.NoError(t, err)
	assert.Contains(t, string(data), `"appIssuer":true`)
}

func TestDashboard_OperatorKeyCannotBeRotated(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{IDTokenSigningKey: testSigningKeyPEM(t)}, true)
	call := dashboardFor(t, f)(f.appID)
	_, err := call(contract.KindCommand, "oauth2.keys.rotate", struct{}{})
	requireContractError(t, err, contract.CodeBadRequest, "the signing key for this issuer is managed by configuration")

	data, err := call(contract.KindQuery, "oauth2.keys.list", struct{}{})
	require.NoError(t, err)
	assert.Contains(t, string(data), `"source":"operator"`)
}

// With neither an encryption key nor an operator key nothing can sign, and
// the dashboard says so rather than failing with an internal error.
func TestDashboard_NoSigningKeyIsUnavailable(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, false)
	setAppIssuer(t, f, f.appID, true)
	call := dashboardFor(t, f)(f.appID)
	_, err := call(contract.KindCommand, "oauth2.keys.rotate", struct{}{})
	requireContractError(t, err, contract.CodeUnavailable,
		"no id_token signing key is available; configure an encryption key or an operator key")
}

func TestRegisterContract_RequiresTheConcreteEngine(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{})
	err := p.RegisterContract(dispatcher.New(dispatcher.NoopMetricsEmitter{}), contract.NewRegistry(), contract.NewWardenRegistry(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires *authsome.Engine")
}
