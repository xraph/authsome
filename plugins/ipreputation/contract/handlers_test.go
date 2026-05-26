package contract

import (
	"bytes"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "ipreputation/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "ipreputation" {
		t.Errorf("contributor name = %q, want ipreputation", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 0 {
		t.Errorf("intents = %d, want 0", got)
	}
	if got := len(m.Graph); got != 1 {
		t.Errorf("graph routes = %d, want 1", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "ipreputation/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}
