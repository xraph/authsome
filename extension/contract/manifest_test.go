package contract

import (
	"strings"
	"testing"

	dashcontract "github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(strings.NewReader(string(manifestYAML)), "authsome/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "auth" {
		t.Errorf("contributor name = %q, want auth", m.Contributor.Name)
	}
	// 68 intents: 66 prior + 2 new feature-toggle intents
	// (auth.featureToggles, auth.toggleFeature). apikeys.* are owned
	// by the apikey plugin manifest, not declared here.
	if got := len(m.Intents); got != 68 {
		t.Errorf("intents = %d, want 68 (with feature toggles)", got)
	}
	// 28 top-level graph routes: 32 prior - 4 routes that moved to
	// their owning plugins (/organizations, /organizations/:id, /plans,
	// /plans/:id).
	if got := len(m.Graph); got != 28 {
		t.Errorf("graph routes = %d, want 28 (org + plan pages moved to plugins)", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(strings.NewReader(string(manifestYAML)), "authsome/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, dashcontract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestManifest_RegistersWithRegistry(t *testing.T) {
	reg := dashcontract.NewRegistry()
	m, err := loader.Load(strings.NewReader(string(manifestYAML)), "authsome/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := reg.Register(m); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Sanity-check the /login graph route survived registration. Slice
	// (l.5) shifted the route from a hardcoded form.edit to the dynamic
	// auth.login.form intent backed by the auth.config query, so the
	// expectation flips to verifying the data binding.
	root, ok := reg.MergedGraph("auth", "/login")
	if !ok {
		t.Fatal("expected /login route to be registered")
	}
	if root.Intent != "auth.login.form" {
		t.Errorf("unexpected /login root: intent=%s", root.Intent)
	}
	if root.Data == nil || root.Data.QueryRef != "queries.config" {
		t.Errorf("expected data: queries.config, got %+v", root.Data)
	}
}
