package sso

import (
	"context"
	"strings"
	"testing"

	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/id"
)

// stubProvider is a minimal Provider for exercising startLogin without a real
// IdP: LoginURL just echoes the state so we can recover it.
type stubProvider struct{}

func (stubProvider) Name() string     { return "stub" }
func (stubProvider) Protocol() string { return "oidc" }
func (stubProvider) LoginURL(state string) (string, error) {
	return "https://idp.example.com/authorize?state=" + state, nil
}
func (stubProvider) HandleCallback(context.Context, map[string]string) (*User, error) {
	return nil, nil
}

// The OIDC redirect_uri must be query-free: Google (and Entra) reject or strip a
// `?connection=` param, causing redirect_uri_mismatch. The connection is
// recovered from the login state instead (see TestStartLogin_CarriesConnID).
func TestOIDCRedirectURLFor_IsQueryFree(t *testing.T) {
	p := &Plugin{config: Config{PublicBaseURL: "https://api.example.com/api/identity"}}
	conn := &Connection{ID: id.NewSSOConnectionID(), Provider: "example.com"}

	got := p.oidcRedirectURLFor(conn)
	want := "https://api.example.com/api/identity/v1/sso/example.com/callback"
	if got != want {
		t.Fatalf("oidcRedirectURLFor = %q, want %q", got, want)
	}
	if strings.ContainsAny(got, "?") || strings.Contains(got, "connection=") {
		t.Fatalf("redirect_uri must be query-free for IdP compatibility, got %q", got)
	}
}

// startLogin must persist the exact connection id in the state ceremony, so the
// query-free callback can recover it (multi-tenant safe — each login's state
// carries its own connection).
func TestStartLogin_CarriesConnID(t *testing.T) {
	p := &Plugin{
		config:     Config{PublicBaseURL: "https://api.example.com/api/identity"},
		ceremonies: ceremony.NewMemory(),
	}
	conn := &Connection{ID: id.NewSSOConnectionID(), Provider: "example.com"}

	resp, err := p.startLogin(context.Background(), id.NewAppID(), stubProvider{}, conn.Provider, conn.ID.String(), "")
	if err != nil {
		t.Fatalf("startLogin: %v", err)
	}

	// The callback recovers the connection purely from the state token.
	st, err := p.loadState(context.Background(), resp.State, conn.Provider)
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if st.ConnID != conn.ID.String() {
		t.Fatalf("state ConnID = %q, want %q", st.ConnID, conn.ID.String())
	}
}
