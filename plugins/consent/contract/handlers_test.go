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
	m, err := loader.Load(bytes.NewReader(manifestYAML), "consent/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "consent" {
		t.Errorf("contributor name = %q, want consent", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 4 {
		t.Errorf("intents = %d, want 4 (list/userConsents/grant/revoke)", got)
	}
	if got := len(m.Graph); got != 1 {
		t.Errorf("graph routes = %d, want 1 (/compliance/consent)", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "consent/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestListHandler_UnavailableWhenStoreNil(t *testing.T) {
	h := listHandler(Deps{})
	_, err := h(context.Background(), ListConsentsInput{}, contract.Principal{})
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Code != contract.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", err)
	}
}
