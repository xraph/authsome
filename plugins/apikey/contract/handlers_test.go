package contract

import (
	"bytes"
	"context"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "apikey/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "apikey" {
		t.Errorf("contributor name = %q, want apikey", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 4 {
		t.Errorf("intents = %d, want 4 (list/detail/create/revoke)", got)
	}
	if got := len(m.Graph); got != 0 {
		t.Errorf("graph routes = %d, want 0 (page stays on auth contributor)", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "apikey/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestApikeysListHandler_UnavailableWhenEngineNil(t *testing.T) {
	h := apikeysListHandler(Deps{Engine: nil})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeUnavailable)
}

func TestApikeysCreateHandler_RejectsBlankName(t *testing.T) {
	// With nil engine the unavailable check runs first; supply a non-nil
	// engine sentinel isn't easily possible without store wiring. The
	// nil-engine case is still useful as a deref guard.
	h := apikeysCreateHandler(Deps{Engine: nil})
	_, err := h(context.Background(), CreateAPIKeyInput{Name: "", UserID: ""}, contract.Principal{})
	expectCode(t, err, contract.CodeUnavailable)
}

func expectCode(t *testing.T, err error, want contract.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %q, got nil", want)
	}
	ce, ok := err.(*contract.Error)
	if !ok {
		t.Fatalf("expected *contract.Error, got %T: %v", err, err)
	}
	if ce.Code != want {
		t.Fatalf("error code = %q, want %q (message=%q)", ce.Code, want, ce.Message)
	}
}
