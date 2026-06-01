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
	m, err := loader.Load(bytes.NewReader(manifestYAML), "waitlist/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "waitlist" {
		t.Errorf("contributor name = %q, want waitlist", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 6 {
		t.Errorf("intents = %d, want 6 (list/detail/approve/reject/delete/counts)", got)
	}
	if got := len(m.Graph); got != 1 {
		t.Errorf("graph routes = %d, want 1 (/waitlist)", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "waitlist/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestListHandler_UnavailableWhenStoreNil(t *testing.T) {
	h := listHandler(Deps{})
	_, err := h(context.Background(), ListEntriesInput{}, contract.Principal{})
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Code != contract.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", err)
	}
}
