package oauth2provider_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store/memory"
)

// Update, rotate and delete act on the caller's own app, so a request
// carrying no caller app is refused before anything is read or written.
func TestAdminClients_WritesNeedACallerApp(t *testing.T) {
	_, st, mux := newFixture(t)
	owner := id.NewAppID()
	c := seedClient(t, st, owner, "needs-caller")

	rec := serve(mux, adminReq(t, http.MethodPatch, "/v1/admin/oauth/clients/"+c.ID.String(), nil,
		map[string]any{"name": "renamed"}))
	require.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())

	rec = serve(mux, adminReq(t, http.MethodPost, "/v1/admin/oauth/clients/"+c.ID.String()+"/secret", nil, nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "client_secret")

	rec = serve(mux, adminReq(t, http.MethodDelete, "/v1/admin/oauth/clients/"+c.ID.String(), nil, nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())

	got, err := st.GetClientByID(context.Background(), c.ID)
	require.NoError(t, err, "the client was not deleted")
	assert.Equal(t, c.Name, got.Name)
	assert.Equal(t, c.ClientSecret, got.ClientSecret)
}

// listFailStore fails every client listing.
type listFailStore struct{ oauth2provider.Store }

func (listFailStore) ListClients(context.Context, id.AppID) ([]*oauth2provider.OAuth2Client, error) {
	return nil, errors.New("pq: connection refused to 10.0.0.7")
}

// A store failure behind the admin API is a 500 that names nothing about
// the store.
func TestAdminClients_StoreFailureIs500(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(listFailStore{oauth2provider.NewMemoryStore()})
	p.SetStore(memory.New())
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	caller := id.NewAppID()
	rec := serve(mux, adminReq(t, http.MethodGet, "/v1/admin/oauth/clients?app_id="+caller.String(), &caller, nil))
	require.Equal(t, http.StatusInternalServerError, rec.Code, "body: %s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "10.0.0.7")
}

func TestMigrationGroups_PerDriver(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{})
	for driver, want := range map[string]*migrate.Group{
		"pg": oauth2provider.PostgresMigrations, "postgres": oauth2provider.PostgresMigrations,
		"sqlite": oauth2provider.SqliteMigrations, "sqlite3": oauth2provider.SqliteMigrations,
		"mongo": oauth2provider.MongoMigrations, "mongodb": oauth2provider.MongoMigrations,
	} {
		got := p.MigrationGroups(driver)
		require.Len(t, got, 1, driver)
		assert.Same(t, want, got[0], driver)
	}
	assert.Nil(t, p.MigrationGroups("mysql"))
}
