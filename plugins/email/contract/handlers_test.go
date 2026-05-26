package contract

import (
	"bytes"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

// TestManifest_Loads is the tripwire for accidental wire-shape drift.
func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "email/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "email" {
		t.Errorf("contributor name = %q, want email", m.Contributor.Name)
	}
	// No intents — settings flow through the auth contributor.
	if got := len(m.Intents); got != 0 {
		t.Errorf("intents = %d, want 0", got)
	}
	// One graph route: /auth/email deep-link page.
	if got := len(m.Graph); got != 1 {
		t.Errorf("graph routes = %d, want 1 (/auth/email)", got)
	}
	if got := len(m.Extends); got != 0 {
		t.Errorf("extends = %d, want 0 (auto-discovered via settings.tabs)", got)
	}
}

// TestManifest_Validates ensures every intent referenced by the graph
// resolves cleanly against the wider catalog.
func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "email/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}
