package oauth2provider

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/a-h/templ"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
)

// errListClientsStore embeds Store (nil) and overrides only ListClients to
// return an error. renderClientsPage with no form action calls only
// ListClients, so the embedded nil interface is never dereferenced.
type errListClientsStore struct {
	Store
}

func (errListClientsStore) ListClients(_ context.Context, _ id.AppID) ([]*OAuth2Client, error) {
	return nil, errors.New("db unavailable")
}

// emptyClientsStore returns an empty client list with no error — the genuine
// "no clients yet" state.
type emptyClientsStore struct {
	Store
}

func (emptyClientsStore) ListClients(_ context.Context, _ id.AppID) ([]*OAuth2Client, error) {
	return nil, nil
}

func renderClientsPageHTML(ctx context.Context, t *testing.T, comp templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := comp.Render(ctx, &buf); err != nil {
		t.Fatalf("render component: %v", err)
	}
	return buf.String()
}

// TestRenderClientsPage_SurfacesListError guards against the regression where a
// failing ListClients was silently swallowed (clients = nil), rendering an
// empty table indistinguishable from a genuine "no clients" state. The error
// must be surfaced to the page instead.
func TestRenderClientsPage_SurfacesListError(t *testing.T) {
	p := &Plugin{oauth2Store: errListClientsStore{}, logger: log.NewNoopLogger()}
	ctx := dashboard.WithAppID(context.Background(), id.NewAppID())

	comp, err := p.renderClientsPage(ctx, contributor.Params{})
	if err != nil {
		t.Fatalf("renderClientsPage returned error: %v", err)
	}

	html := renderClientsPageHTML(ctx, t, comp)
	if !strings.Contains(html, "Failed to load OAuth2 clients") {
		t.Errorf("expected the list error to be surfaced on the page, got:\n%s", html)
	}
}

// TestRenderClientsPage_EmptyShowsEmptyState ensures a genuine empty (non-error)
// result still shows the empty-state message and no error banner.
func TestRenderClientsPage_EmptyShowsEmptyState(t *testing.T) {
	p := &Plugin{oauth2Store: emptyClientsStore{}, logger: log.NewNoopLogger()}
	ctx := dashboard.WithAppID(context.Background(), id.NewAppID())

	comp, err := p.renderClientsPage(ctx, contributor.Params{})
	if err != nil {
		t.Fatalf("renderClientsPage returned error: %v", err)
	}

	html := renderClientsPageHTML(ctx, t, comp)
	if strings.Contains(html, "Failed to load OAuth2 clients") {
		t.Errorf("did not expect an error banner for an empty (non-error) list, got:\n%s", html)
	}
	if !strings.Contains(html, "No OAuth2 clients have been created yet") {
		t.Errorf("expected the empty-state message, got:\n%s", html)
	}
}

// TestRenderClientsPage_CreateStoresResources covers the dashboard's create
// path for the RFC 8707 resource allowlist end to end: submitting the form
// with a resources value must actually persist it on the created client, not
// just accept the field and drop it. Checking the stored client rather than
// only the rendered success banner is what proves the value survives the
// form-parse-store round trip, and would fail if the field were parsed but
// never attached to the OAuth2Client passed to CreateClient.
func TestRenderClientsPage_CreateStoresResources(t *testing.T) {
	store := NewMemoryStore()
	p := &Plugin{oauth2Store: store, logger: log.NewNoopLogger()}
	appID := id.NewAppID()
	ctx := dashboard.WithAppID(context.Background(), appID)

	comp, err := p.renderClientsPage(ctx, contributor.Params{
		FormData: map[string]string{
			"action":        "create",
			"name":          "Resource Client",
			"redirect_uris": "https://app.example.com/cb",
			"resources":     "https://api.example.com https://files.example.com",
		},
	})
	if err != nil {
		t.Fatalf("renderClientsPage returned error: %v", err)
	}

	html := renderClientsPageHTML(ctx, t, comp)
	if strings.Contains(html, "o2-error-toast") {
		t.Fatalf("did not expect a validation error, got:\n%s", html)
	}
	if !strings.Contains(html, "OAuth2 client created successfully") {
		t.Errorf("expected the success banner, got:\n%s", html)
	}

	clients, err := store.ListClients(ctx, appID)
	if err != nil {
		t.Fatalf("ListClients: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected exactly one stored client, got %d", len(clients))
	}
	want := []string{"https://api.example.com", "https://files.example.com"}
	if !reflect.DeepEqual(clients[0].Resources, want) {
		t.Errorf("stored resources = %v, want %v", clients[0].Resources, want)
	}
}

// TestRenderClientsPage_CreateWithoutResourcesStoresEmptyAllowlist is the
// backwards-compatibility guard for every dashboard user who submitted this
// form before the resources field existed. Their form posts never carry a
// resources value at all, so an absent field must still succeed and store an
// empty allowlist rather than failing or defaulting to something broader.
func TestRenderClientsPage_CreateWithoutResourcesStoresEmptyAllowlist(t *testing.T) {
	store := NewMemoryStore()
	p := &Plugin{oauth2Store: store, logger: log.NewNoopLogger()}
	appID := id.NewAppID()
	ctx := dashboard.WithAppID(context.Background(), appID)

	comp, err := p.renderClientsPage(ctx, contributor.Params{
		FormData: map[string]string{
			"action":        "create",
			"name":          "No Resources Client",
			"redirect_uris": "https://app.example.com/cb",
			// resources intentionally absent, matching every pre-existing
			// dashboard form submission.
		},
	})
	if err != nil {
		t.Fatalf("renderClientsPage returned error: %v", err)
	}

	html := renderClientsPageHTML(ctx, t, comp)
	if strings.Contains(html, "o2-error-toast") {
		t.Fatalf("did not expect a validation error, got:\n%s", html)
	}
	if !strings.Contains(html, "OAuth2 client created successfully") {
		t.Errorf("expected the success banner, got:\n%s", html)
	}

	clients, err := store.ListClients(ctx, appID)
	if err != nil {
		t.Fatalf("ListClients: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected exactly one stored client, got %d", len(clients))
	}
	if len(clients[0].Resources) != 0 {
		t.Errorf("expected an empty allowlist, got %v", clients[0].Resources)
	}
}
