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
	if got := len(m.Intents); got != 2 {
		t.Errorf("intents = %d, want 2 (login + logout)", got)
	}
	if got := len(m.Graph); got != 1 {
		t.Errorf("graph routes = %d, want 1 (/login)", got)
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
	// Sanity-check the /login graph route survived registration with the
	// expected slot shape (form.edit > fields > 2x form.field).
	root, ok := reg.MergedGraph("auth", "/login")
	if !ok {
		t.Fatal("expected /login route to be registered")
	}
	if root.Intent != "form.edit" || root.Op != "auth.login" {
		t.Errorf("unexpected /login root: intent=%s op=%s", root.Intent, root.Op)
	}
	fields, ok := root.Slots["fields"]
	if !ok {
		t.Fatal("expected fields slot")
	}
	if len(fields) != 2 {
		t.Errorf("expected 2 form.field children, got %d", len(fields))
	}
	for _, f := range fields {
		if f.Intent != "form.field" {
			t.Errorf("unexpected child intent: %s", f.Intent)
		}
	}
}
