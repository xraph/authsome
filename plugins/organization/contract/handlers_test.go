package contract

import (
	"bytes"
	"context"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "organization/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "organization" {
		t.Errorf("contributor name = %q, want organization", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 7 {
		t.Errorf("intents = %d, want 7 (list/detail/create/update/delete/members/removeMember)", got)
	}
	if got := len(m.Graph); got != 3 {
		t.Errorf("graph routes = %d, want 3 (/organizations + /:id + /create)", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "organization/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestOrgsListHandler_UnavailableWhenEngineNil(t *testing.T) {
	h := orgsListHandler(Deps{})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	if ce, ok := err.(*contract.Error); !ok || ce.Code != contract.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", err)
	}
}
