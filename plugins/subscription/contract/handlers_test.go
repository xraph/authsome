package contract

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "subscription/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "subscription" {
		t.Errorf("contributor name = %q, want subscription", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 5 {
		t.Errorf("intents = %d, want 5 (plans list/detail/archive/activate + subscriptions.list)", got)
	}
	if got := len(m.Graph); got != 2 {
		t.Errorf("graph routes = %d, want 2 (/plans + /plans/:id)", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "subscription/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestPlansListHandler_UnavailableWhenServiceNil(t *testing.T) {
	h := plansListHandler(Deps{})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Code != contract.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", err)
	}
}
