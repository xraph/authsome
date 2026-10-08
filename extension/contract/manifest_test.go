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
	if m.Contributor.Name != "authsome" {
		t.Errorf("contributor name = %q, want authsome", m.Contributor.Name)
	}
	// 68 intents: 66 prior + 2 new feature-toggle intents
	// (auth.featureToggles, auth.toggleFeature). apikeys.* are owned
	// by the apikey plugin manifest, not declared here.
	if got := len(m.Intents); got != 68 {
		t.Errorf("intents = %d, want 68 (with feature toggles)", got)
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
	// The auth.config query backs the React login form, so it has to
	// survive registration alongside the intent it names.
	got, ok := reg.Contributor("authsome")
	if !ok {
		t.Fatal("expected authsome contributor to be registered")
	}
	q, ok := got.Queries["config"]
	if !ok || q.Intent != "auth.config" {
		t.Errorf("expected queries.config -> auth.config, got %+v (present=%v)", q, ok)
	}
	if _, ok := reg.Intent("authsome", "auth.config", 1); !ok {
		t.Error("expected auth.config v1 to be registered")
	}
}
