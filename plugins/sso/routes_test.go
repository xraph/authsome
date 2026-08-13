package sso

import (
	"testing"

	"github.com/xraph/forge"
)

// TestRegisterRoutes_NoConflict guards that RegisterRoutes wires cleanly — in
// particular that the OIDC browser landing (GET /:provider/callback) and the
// JSON callback (POST /:provider/callback) share a path without a router
// conflict. A regression here panics/errors the whole app at boot.
func TestRegisterRoutes_NoConflict(t *testing.T) {
	p := New()
	mux := forge.NewRouter()
	if err := p.RegisterRoutes(mux); err != nil {
		t.Fatalf("RegisterRoutes returned error: %v", err)
	}
}
