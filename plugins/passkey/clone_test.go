package passkey

import (
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"
)

// TestCloneWarningError pins that a WebAuthn clone warning (a non-increasing
// sign count, which go-webauthn flags but does not itself reject) is turned
// into a login-rejecting error. Without this, a cloned/duplicated authenticator
// authenticates successfully and goes undetected.
func TestCloneWarningError(t *testing.T) {
	cloned := &webauthn.Credential{}
	cloned.Authenticator.CloneWarning = true
	if err := cloneWarningError(cloned); err == nil {
		t.Fatal("a credential flagged as cloned must produce an error")
	}

	ok := &webauthn.Credential{}
	ok.Authenticator.CloneWarning = false
	if err := cloneWarningError(ok); err != nil {
		t.Fatalf("a healthy credential must not error, got %v", err)
	}

	if err := cloneWarningError(nil); err != nil {
		t.Fatalf("nil credential must not error, got %v", err)
	}
}
