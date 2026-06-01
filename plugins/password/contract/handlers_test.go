package contract

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

// TestManifest_Loads is the tripwire for accidental wire-shape drift.
// Bump the want values when intentionally adding or removing the
// password contributor's surface.
func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "password/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "password" {
		t.Errorf("contributor name = %q, want password", m.Contributor.Name)
	}
	// One intent: password.policy. settings.* lives on the auth
	// contributor and is auto-discovered by settings.tabs.
	if got := len(m.Intents); got != 1 {
		t.Errorf("intents = %d, want 1 (policy)", got)
	}
	// One graph route: /auth/password (deep-link page rendering
	// the settings.panel for the password namespace).
	if got := len(m.Graph); got != 1 {
		t.Errorf("graph routes = %d, want 1 (/auth/password)", got)
	}
	// No extends — the global /settings page auto-discovers via
	// settings.tabs + settings.namespaces.
	if got := len(m.Extends); got != 0 {
		t.Errorf("extends = %d, want 0 (no manual extension; auto-discovered)", got)
	}
}

// TestManifest_Validates ensures every intent referenced by the graph
// resolves cleanly against the wider catalog.
func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "password/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

// TestPolicyHandler_UnavailableWhenEngineNil mirrors the auth
// contributor's UnavailableWhenEngineNil pattern. A nil engine is the
// "engine not configured" sentinel — handlers must return
// CodeUnavailable rather than panicking on a nil deref.
func TestPolicyHandler_UnavailableWhenEngineNil(t *testing.T) {
	h := policyHandler(Deps{Engine: nil})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeUnavailable)
}

func expectCode(t *testing.T, err error, want contract.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %q, got nil", want)
	}
	var ce *contract.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected *contract.Error, got %T: %v", err, err)
	}
	if ce.Code != want {
		t.Fatalf("error code = %q, want %q (message=%q)", ce.Code, want, ce.Message)
	}
}
