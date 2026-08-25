package oauth2provider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
)

// newDashboardClientFixture seeds one confidential client and returns the
// plugin, the store, the app context and the client's primary key.
func newDashboardClientFixture(t *testing.T) (*Plugin, Store, context.Context, id.OAuth2ClientID) {
	t.Helper()
	st := NewMemoryStore()
	appID := id.NewAppID()
	clientPK := id.NewOAuth2ClientID()
	now := time.Now()
	require.NoError(t, st.CreateClient(context.Background(), &OAuth2Client{
		ID:           clientPK,
		AppID:        appID,
		Name:         "Before",
		ClientID:     "dash-client",
		RedirectURIs: []string{"https://app.example.com/cb"},
		Scopes:       []string{"openid"},
		GrantTypes:   []string{"authorization_code"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}))
	p := &Plugin{oauth2Store: st, logger: log.NewNoopLogger()}
	return p, st, dashboard.WithAppID(context.Background(), appID), clientPK
}

// The dashboard edit path has to reach the same outcome as the admin API.
// Without it an operator working in the UI still has only create and delete,
// which is the gap this whole change exists to close.
func TestRenderClientsPage_UpdateAppliesChanges(t *testing.T) {
	p, st, ctx, clientPK := newDashboardClientFixture(t)

	_, err := p.renderClientsPage(ctx, contributor.Params{FormData: map[string]string{
		"action":                   "update",
		"client_id":                clientPK.String(),
		"name":                     "After",
		"scopes":                   "openid email",
		"resources":                "https://api.example.com",
		"redirect_uris":            "https://app.example.com/cb",
		"grant_authorization_code": "on",
	}})
	require.NoError(t, err)

	got, err := st.GetClientByID(context.Background(), clientPK)
	require.NoError(t, err)
	assert.Equal(t, "After", got.Name)
	assert.Equal(t, []string{"openid", "email"}, got.Scopes)
	assert.Equal(t, []string{"https://api.example.com"}, got.Resources)
	// Editing must not rotate the identifier, the same guarantee the API gives.
	assert.Equal(t, "dash-client", got.ClientID)
}

// A bad resource URI is refused in the dashboard for the same reason it is on
// the API: the allowlist must never hold a value the request-time resolver
// would then reject.
func TestRenderClientsPage_UpdateRejectsBadResource(t *testing.T) {
	p, st, ctx, clientPK := newDashboardClientFixture(t)

	comp, err := p.renderClientsPage(ctx, contributor.Params{FormData: map[string]string{
		"action":                   "update",
		"client_id":                clientPK.String(),
		"name":                     "After",
		"resources":                "not-a-uri",
		"redirect_uris":            "https://app.example.com/cb",
		"grant_authorization_code": "on",
	}})
	require.NoError(t, err)

	html := renderClientsPageHTML(ctx, t, comp)
	assert.Contains(t, html, "is not an absolute URI")

	got, err := st.GetClientByID(context.Background(), clientPK)
	require.NoError(t, err)
	assert.Equal(t, "Before", got.Name, "a refused update must not partially apply")
	assert.Empty(t, got.Resources)
}

// The dashboard update path needs the same tenancy rule as the dashboard
// delete path, which compares client.AppID against the caller's app and treats
// a mismatch as not-found. Without it an operator on one app could edit
// another app's client by posting its primary key, and the dashboard would be
// a softer target than the API for exactly the same attack.
func TestRenderClientsPage_UpdateRefusesAnotherAppsClient(t *testing.T) {
	p, st, _, clientPK := newDashboardClientFixture(t)
	otherCtx := dashboard.WithAppID(context.Background(), id.NewAppID())

	comp, err := p.renderClientsPage(otherCtx, contributor.Params{FormData: map[string]string{
		"action":                   "update",
		"client_id":                clientPK.String(),
		"name":                     "Stolen",
		"redirect_uris":            "https://app.example.com/cb",
		"grant_authorization_code": "on",
	}})
	require.NoError(t, err)
	_ = comp

	got, err := st.GetClientByID(context.Background(), clientPK)
	require.NoError(t, err)
	assert.Equal(t, "Before", got.Name, "another app's client must not be editable")
}

// An operator cannot manage what they cannot see. Scopes and the resource
// allowlist both drive authorization decisions, and neither was rendered.
func TestRenderClientsPage_ShowsScopesAndResources(t *testing.T) {
	p, st, ctx, clientPK := newDashboardClientFixture(t)

	loaded, err := st.GetClientByID(context.Background(), clientPK)
	require.NoError(t, err)
	updated := *loaded
	updated.Scopes = []string{"openid", "profile"}
	updated.Resources = []string{"https://api.example.com"}
	require.NoError(t, st.UpdateClient(context.Background(), &updated))

	comp, err := p.renderClientsPage(ctx, contributor.Params{})
	require.NoError(t, err)

	html := renderClientsPageHTML(ctx, t, comp)
	assert.Contains(t, html, "openid, profile")
	assert.Contains(t, html, "https://api.example.com")
	_ = clientPK
}

// The edit action pre-fills the form rather than mutating anything, so an
// operator sees the current values before changing them.
func TestRenderClientsPage_EditPrefillsWithoutMutating(t *testing.T) {
	p, st, ctx, clientPK := newDashboardClientFixture(t)

	comp, err := p.renderClientsPage(ctx, contributor.Params{FormData: map[string]string{
		"action":    "edit",
		"client_id": clientPK.String(),
	}})
	require.NoError(t, err)

	html := renderClientsPageHTML(ctx, t, comp)
	assert.Contains(t, html, `value="update"`, "the pre-filled form submits as an update")
	assert.Contains(t, html, "Before", "current name is pre-filled")

	got, err := st.GetClientByID(context.Background(), clientPK)
	require.NoError(t, err)
	assert.Equal(t, "Before", got.Name, "rendering the edit form must not write anything")
}
