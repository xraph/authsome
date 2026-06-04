package oauth2provider

import (
	"bytes"
	"context"
	"errors"
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

func renderClientsPageHTML(t *testing.T, ctx context.Context, comp templ.Component) string {
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

	html := renderClientsPageHTML(t, ctx, comp)
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

	html := renderClientsPageHTML(t, ctx, comp)
	if strings.Contains(html, "Failed to load OAuth2 clients") {
		t.Errorf("did not expect an error banner for an empty (non-error) list, got:\n%s", html)
	}
	if !strings.Contains(html, "No OAuth2 clients have been created yet") {
		t.Errorf("expected the empty-state message, got:\n%s", html)
	}
}
