package oauth2provider

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	oauth2contract "github.com/xraph/authsome/plugins/oauth2provider/contract"
	"github.com/xraph/authsome/settings"
)

var errStoreDown = errors.New("store is down")

// faultyStore fails the operations whose error field is set, the way a
// database outage would, and passes everything else through. listKeysQueue
// answers successive ListSigningKeys calls before listKeys applies; a nil
// entry passes that call through.
type faultyStore struct {
	Store
	listClients, getClientByID, createClient, updateClient, deleteClient error
	listKeys, createKey, retireKey                                       error
	listKeysQueue                                                        []error
	getClientByIDCalls                                                   int
}

func (s *faultyStore) ListClients(ctx context.Context, appID id.AppID) ([]*OAuth2Client, error) {
	if s.listClients != nil {
		return nil, s.listClients
	}
	return s.Store.ListClients(ctx, appID)
}

func (s *faultyStore) GetClientByID(ctx context.Context, cid id.OAuth2ClientID) (*OAuth2Client, error) {
	s.getClientByIDCalls++
	if s.getClientByID != nil {
		return nil, s.getClientByID
	}
	return s.Store.GetClientByID(ctx, cid)
}

func (s *faultyStore) CreateClient(ctx context.Context, c *OAuth2Client) error {
	if s.createClient != nil {
		return s.createClient
	}
	return s.Store.CreateClient(ctx, c)
}

func (s *faultyStore) UpdateClient(ctx context.Context, c *OAuth2Client) error {
	if s.updateClient != nil {
		return s.updateClient
	}
	return s.Store.UpdateClient(ctx, c)
}

func (s *faultyStore) DeleteClient(ctx context.Context, cid id.OAuth2ClientID) error {
	if s.deleteClient != nil {
		return s.deleteClient
	}
	return s.Store.DeleteClient(ctx, cid)
}

func (s *faultyStore) ListSigningKeys(ctx context.Context, scope string) ([]*SigningKey, error) {
	if len(s.listKeysQueue) > 0 {
		err := s.listKeysQueue[0]
		s.listKeysQueue = s.listKeysQueue[1:]
		if err != nil {
			return nil, err
		}
		return s.Store.ListSigningKeys(ctx, scope)
	}
	if s.listKeys != nil {
		return nil, s.listKeys
	}
	return s.Store.ListSigningKeys(ctx, scope)
}

func (s *faultyStore) CreateSigningKey(ctx context.Context, k *SigningKey) error {
	if s.createKey != nil {
		return s.createKey
	}
	return s.Store.CreateSigningKey(ctx, k)
}

func (s *faultyStore) RetireSigningKey(ctx context.Context, keyID id.OAuth2SigningKeyID, at time.Time) error {
	if s.retireKey != nil {
		return s.retireKey
	}
	return s.Store.RetireSigningKey(ctx, keyID, at)
}

// servicePlugin is a plugin over a faultyStore with a managed key manager and
// no engine, so every app is on the deployment issuer.
func servicePlugin(t *testing.T) (*Plugin, *faultyStore) {
	t.Helper()
	st := &faultyStore{Store: NewMemoryStore()}
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(st)
	require.NoError(t, p.initSigningKeys(context.Background(), testEncryptor(t)))
	return p, st
}

func seedServiceClient(t *testing.T, st Store, appID id.AppID, public bool) *OAuth2Client {
	t.Helper()
	c := &OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: "cid-" + id.NewOAuth2ClientID().String(),
		ClientSecret: "$2a$10$storedhashstoredhashstoredhashstoredhashstoredhash", Name: "Seeded",
		RedirectURIs: []string{"https://app.example.com/cb"}, Scopes: []string{"openid"},
		GrantTypes: []string{"authorization_code"}, Public: public,
	}
	require.NoError(t, st.CreateClient(context.Background(), c))
	return c
}

func requireValidation(t *testing.T, err error, msg string) {
	t.Helper()
	var v *clientValidationError
	require.True(t, errors.As(err, &v), "err = %v, want a validation error", err)
	assert.Equal(t, msg, v.Error())
}

// A blank or malformed id is refused before the store is asked, by every
// method that takes one, and the adapter hands the dashboard the same text.
func TestClientService_RefusesMissingAndMalformedIDs(t *testing.T) {
	p, st := servicePlugin(t)
	ctx := context.Background()
	appID := id.NewAppID()
	name := "x"
	for _, c := range []struct{ raw, msg string }{{"", "client ID required"}, {"not-an-id", "invalid client ID"}} {
		_, err := p.GetClientFor(ctx, appID, c.raw)
		requireValidation(t, err, c.msg)
		_, err = p.UpdateClientFor(ctx, appID, c.raw, UpdateClientRequest{Name: &name})
		requireValidation(t, err, c.msg)
		requireValidation(t, p.DeleteClientFor(ctx, appID, c.raw), c.msg)
		_, err = p.RotateClientSecretFor(ctx, appID, c.raw)
		requireValidation(t, err, c.msg)

		_, err = contractService{p}.GetClientFor(ctx, appID, c.raw)
		var v *oauth2contract.ValidationError
		require.True(t, errors.As(err, &v), "adapter err = %v", err)
		assert.Equal(t, c.msg, v.Msg)
	}
	assert.Zero(t, st.getClientByIDCalls, "a malformed id never reaches the store")
}

// A store failure is wrapped with what was being done, is never mistaken for
// "not found", and the adapter passes it on untranslated so the handler
// answers a generic internal error.
func TestClientService_StoreFailuresAreWrapped(t *testing.T) {
	ctx := context.Background()
	appID := id.NewAppID()
	name := "renamed"

	cases := []struct {
		name       string
		breakStore func(*faultyStore)
		call       func(p *Plugin, clientID string) error
		prefix     string
	}{
		{"get", func(s *faultyStore) { s.getClientByID = errStoreDown }, func(p *Plugin, cid string) error {
			_, err := p.GetClientFor(ctx, appID, cid)
			return err
		}, "oauth2: load client"},
		{"list", func(s *faultyStore) { s.listClients = errStoreDown }, func(p *Plugin, _ string) error {
			_, err := p.ListClientsFor(ctx, appID)
			return err
		}, "oauth2: list clients"},
		{"create", func(s *faultyStore) { s.createClient = errStoreDown }, func(p *Plugin, _ string) error {
			_, err := p.CreateClientFor(ctx, appID, CreateClientRequest{Name: "n", Public: true})
			return err
		}, "oauth2: create client"},
		{"update", func(s *faultyStore) { s.updateClient = errStoreDown }, func(p *Plugin, cid string) error {
			_, err := p.UpdateClientFor(ctx, appID, cid, UpdateClientRequest{Name: &name})
			return err
		}, "oauth2: update client"},
		{"delete", func(s *faultyStore) { s.deleteClient = errStoreDown }, func(p *Plugin, cid string) error {
			return p.DeleteClientFor(ctx, appID, cid)
		}, "oauth2: delete client"},
		{"rotate", func(s *faultyStore) { s.updateClient = errStoreDown }, func(p *Plugin, cid string) error {
			_, err := p.RotateClientSecretFor(ctx, appID, cid)
			return err
		}, "oauth2: rotate client secret"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, st := servicePlugin(t)
			seeded := seedServiceClient(t, st.Store, appID, false)
			c.breakStore(st)
			err := c.call(p, seeded.ID.String())
			require.ErrorIs(t, err, errStoreDown)
			assert.True(t, strings.HasPrefix(err.Error(), c.prefix), "err = %v", err)
			assert.NotErrorIs(t, err, ErrClientNotFound)
			assert.Same(t, err, toContractErr(err), "the adapter passes a store failure through as is")

			// Nothing was changed by the failed write.
			got, getErr := st.Store.GetClientByID(ctx, seeded.ID)
			if c.name == "delete" || c.name == "update" || c.name == "rotate" {
				require.NoError(t, getErr)
				assert.Equal(t, seeded.Name, got.Name)
				assert.Equal(t, seeded.ClientSecret, got.ClientSecret)
			}
		})
	}
}

// The adapter reports each store failure from the method that hit it.
func TestContractService_PassesStoreFailuresThrough(t *testing.T) {
	ctx := context.Background()
	p, st := servicePlugin(t)
	appID := id.NewAppID()
	seeded := seedServiceClient(t, st.Store, appID, false)
	s := contractService{p}

	st.listClients = errStoreDown
	_, err := s.ListClientsFor(ctx, appID)
	require.ErrorIs(t, err, errStoreDown)

	st.updateClient = errStoreDown
	_, err = s.UpdateClientFor(ctx, appID, seeded.ID.String(), oauth2contract.UpdateParams{})
	require.ErrorIs(t, err, errStoreDown)
	_, err = s.RotateClientSecretFor(ctx, appID, seeded.ID.String())
	require.ErrorIs(t, err, errStoreDown)

	st.deleteClient = errStoreDown
	require.ErrorIs(t, s.DeleteClientFor(ctx, appID, seeded.ID.String()), errStoreDown)

	st.listKeys = errStoreDown
	_, _, err = s.SigningKeys(ctx, appID)
	require.ErrorIs(t, err, errStoreDown)
}

func TestClientService_UpdateAppliesFirstParty(t *testing.T) {
	p, st := servicePlugin(t)
	ctx := context.Background()
	appID := id.NewAppID()
	seeded := seedServiceClient(t, st.Store, appID, false)
	require.False(t, seeded.FirstParty)

	yes := true
	got, err := p.UpdateClientFor(ctx, appID, seeded.ID.String(), UpdateClientRequest{FirstParty: &yes})
	require.NoError(t, err)
	assert.True(t, got.FirstParty)
	stored, err := st.Store.GetClientByID(ctx, seeded.ID)
	require.NoError(t, err)
	assert.True(t, stored.FirstParty)
	assert.Equal(t, seeded.Name, stored.Name, "fields the update did not send are kept")
}

func TestClientService_ListIsNeverNil(t *testing.T) {
	p, _ := servicePlugin(t)
	list, err := p.ListClientsFor(context.Background(), id.NewAppID())
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Empty(t, list)
}

func TestClientService_CreateRefusesBadResource(t *testing.T) {
	p, st := servicePlugin(t)
	_, err := p.CreateClientFor(context.Background(), id.NewAppID(), CreateClientRequest{
		Name: "n", Public: true, Resources: []string{"not a uri"},
	})
	var v *clientValidationError
	require.ErrorAs(t, err, &v)
	clients, listErr := st.Store.ListClients(context.Background(), id.NewAppID())
	require.NoError(t, listErr)
	assert.Empty(t, clients)
}

func TestClientValidationError_IsItsMessage(t *testing.T) {
	assert.Equal(t, "name required", invalidClient("name required").Error())
}

// Without a key manager nothing can sign, and both key methods say so with
// errNoSigningKey, which the adapter turns into the contract's sentinel.
func TestSigningKeys_NoKeyManager(t *testing.T) {
	ctx := context.Background()
	p := New(Config{Issuer: "https://auth.example.com"})
	_, _, err := p.SigningKeys(ctx, id.NewAppID())
	require.ErrorIs(t, err, errNoSigningKey)
	_, err = p.RotateSigningKey(ctx, id.NewAppID())
	require.ErrorIs(t, err, errNoSigningKey)

	s := contractService{p}
	_, _, err = s.SigningKeys(ctx, id.NewAppID())
	require.ErrorIs(t, err, oauth2contract.ErrNoSigningKey)
	_, err = s.RotateSigningKey(ctx, id.NewAppID())
	require.ErrorIs(t, err, oauth2contract.ErrNoSigningKey)
}

// An app that opted into its own issuer but whose row cannot be loaded must
// not be answered with the deployment issuer or its keys.
func TestIssuerMethods_FailWhenTheIssuerCannotBeResolved(t *testing.T) {
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))
	require.NoError(t, p.DeclareSettings(eng.Settings()))
	require.NoError(t, p.initSigningKeys(ctx, testEncryptor(t)))

	ghost := id.NewAppID()
	raw, err := json.Marshal(true)
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, "oauth2.app_issuer", raw,
		settings.ScopeApp, ghost.String(), ghost.String(), "", "test"))

	info, err := p.IssuerInfo(ctx, ghost)
	require.Error(t, err)
	assert.Empty(t, info.Issuer)
	issuer, keys, err := p.SigningKeys(ctx, ghost)
	require.Error(t, err)
	assert.Empty(t, issuer)
	assert.Empty(t, keys)
	_, err = p.RotateSigningKey(ctx, ghost)
	require.Error(t, err)
	all, err := p.oauth2Store.ListSigningKeys(ctx, ghost.String())
	require.NoError(t, err)
	assert.Empty(t, all, "no key was minted for an issuer that could not be resolved")

	_, err = contractService{p}.IssuerInfo(ctx, ghost)
	require.Error(t, err)
}

// A failed rotation of an app's own key is reported, not swallowed.
func TestRotateSigningKey_StoreFailure(t *testing.T) {
	ctx := context.Background()
	eng := secutil.NewTestEngine(t)
	st := &faultyStore{Store: NewMemoryStore()}
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(st)
	require.NoError(t, p.OnInit(ctx, eng))
	require.NoError(t, p.DeclareSettings(eng.Settings()))
	require.NoError(t, p.initSigningKeys(ctx, testEncryptor(t)))
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	raw, err := json.Marshal(true)
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, "oauth2.app_issuer", raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	st.listKeys = errStoreDown
	_, err = p.RotateSigningKey(ctx, appID)
	require.ErrorIs(t, err, errStoreDown)
}

// The adapter's projections carry every field the dashboard shows and have
// nowhere to put a secret hash.
func TestContractProjections(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := created.Add(time.Hour)
	c := &OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: id.NewAppID(), ClientID: "pub", ClientSecret: "$2a$10$hash", Name: "Web",
		Public: false, FirstParty: true, RedirectURIs: []string{"https://a/cb"}, Scopes: []string{"openid"},
		GrantTypes: []string{"authorization_code"}, Resources: []string{"https://api"},
		TokenEndpointAuthMethod: "client_secret_basic", DPoPMode: "required", CreatedAt: created, UpdatedAt: updated,
	}
	got := toContractClient(c)
	assert.Equal(t, oauth2contract.Client{
		ID: c.ID.String(), ClientID: "pub", Name: "Web", FirstParty: true,
		RedirectURIs: []string{"https://a/cb"}, Scopes: []string{"openid"}, GrantTypes: []string{"authorization_code"},
		Resources: []string{"https://api"}, AuthMethod: "client_secret_basic", DPoPMode: "required",
		CreatedAt: created, UpdatedAt: updated,
	}, got)
	ty := reflect.TypeOf(oauth2contract.Client{})
	for i := range ty.NumField() {
		assert.NotContains(t, strings.ToLower(ty.Field(i).Name), "secret", "contract.Client must not carry a secret")
	}

	retires := updated
	k := toContractKey(KeyInfo{KID: "k", Alg: "ES256", Source: keySourceOperator, Status: keyStatusRetiring, CreatedAt: created, RetiresAt: &retires})
	assert.Equal(t, oauth2contract.KeyInfo{KID: "k", Alg: "ES256", Source: "operator", Status: "retiring", CreatedAt: created, RetiresAt: &retires}, k)
}

func TestToContractErr_UnwrapsAndPassesUnknownThrough(t *testing.T) {
	assert.ErrorIs(t, toContractErr(errors.Join(errors.New("ctx"), ErrClientNotFound)), oauth2contract.ErrClientNotFound)
	var v *oauth2contract.ValidationError
	require.ErrorAs(t, toContractErr(errors.Join(errors.New("ctx"), invalidClient("bad"))), &v)
	assert.Equal(t, "bad", v.Msg)
	other := errors.New("anything else")
	assert.Same(t, other, toContractErr(other))
}

// The adapter answers with the plugin's data: the secret it returns on
// create and rotate is the one the store's hash verifies.
func TestContractService_RoundTrip(t *testing.T) {
	ctx := context.Background()
	p, st := servicePlugin(t)
	s := contractService{p}
	appID := id.NewAppID()

	created, err := s.CreateClientFor(ctx, appID, oauth2contract.CreateParams{
		Name: "Web", FirstParty: true, RedirectURIs: []string{"https://app.example.com/cb"},
		Scopes: []string{"openid"}, GrantTypes: []string{"authorization_code"}, Resources: []string{"https://api.example.com"},
	})
	require.NoError(t, err)
	cid, err := id.ParseOAuth2ClientID(created.ID)
	require.NoError(t, err)
	stored, err := st.Store.GetClientByID(ctx, cid)
	require.NoError(t, err)
	assert.Equal(t, created.ClientID, stored.ClientID)
	assert.True(t, stored.FirstParty)
	assert.Equal(t, []string{"https://api.example.com"}, stored.Resources)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.ClientSecret), []byte(created.ClientSecret)))

	list, err := s.ListClientsFor(ctx, appID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, created.ID, list[0].ID)

	name := "Web 2"
	uris := []string{"https://app.example.com/other"}
	upd, err := s.UpdateClientFor(ctx, appID, created.ID, oauth2contract.UpdateParams{Name: &name, RedirectURIs: &uris})
	require.NoError(t, err)
	assert.Equal(t, "Web 2", upd.Name)
	assert.Equal(t, uris, upd.RedirectURIs)

	rot, err := s.RotateClientSecretFor(ctx, appID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, rot.ID)
	assert.Equal(t, created.ClientID, rot.ClientID)
	stored, err = st.Store.GetClientByID(ctx, cid)
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.ClientSecret), []byte(rot.ClientSecret)))

	issuer, keys, err := s.SigningKeys(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, "https://auth.example.com", issuer)
	assert.Empty(t, keys, "listing keys never mints one")

	_, err = s.RotateSigningKey(ctx, appID)
	require.ErrorIs(t, err, oauth2contract.ErrSharedKey)

	info, err := s.IssuerInfo(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, oauth2contract.IssuerDetails{
		Issuer: "https://auth.example.com", DiscoveryURL: "https://auth.example.com/.well-known/openid-configuration",
		JWKSURI: "https://auth.example.com/v1/oauth/jwks", Algs: []string{"RS256"},
	}, info)

	require.NoError(t, s.DeleteClientFor(ctx, appID, created.ID))
	_, err = s.GetClientFor(ctx, appID, created.ID)
	require.ErrorIs(t, err, oauth2contract.ErrClientNotFound)
}
