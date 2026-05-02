package social

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// PKCE
// ──────────────────────────────────────────────────

func TestPKCE_ChallengeS256MatchesSpec(t *testing.T) {
	t.Parallel()
	// RFC 7636 §4.6 worked example.
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	got := pkceChallengeS256(verifier)
	if got != want {
		t.Errorf("pkceChallengeS256(%q) = %q, want %q (RFC 7636 example)", verifier, got, want)
	}
}

func TestPKCE_ChallengeMatchesManualSHA(t *testing.T) {
	t.Parallel()
	verifier := "abc123-defXYZ"
	h := sha256.Sum256([]byte(verifier))
	expect := base64.RawURLEncoding.EncodeToString(h[:])
	if got := pkceChallengeS256(verifier); got != expect {
		t.Errorf("pkceChallengeS256(%q) = %q, want %q", verifier, got, expect)
	}
}

// ──────────────────────────────────────────────────
// OAuth state key namespacing
// ──────────────────────────────────────────────────

func TestSocialStateKey_NamespacedByApp(t *testing.T) {
	t.Parallel()
	appA := mustParseAppID(t, "aapp_01jf0000000000000000000001")
	appB := mustParseAppID(t, "aapp_01jf0000000000000000000002")
	if socialStateKey(appA, "abc") == socialStateKey(appB, "abc") {
		t.Fatal("state key for app A must NOT equal state key for app B with the same state token")
	}
	if !strings.Contains(socialStateKey(appA, "abc"), appA.String()) {
		t.Errorf("state key %q must contain app id %q", socialStateKey(appA, "abc"), appA.String())
	}
	if !strings.HasSuffix(socialStateKey(appA, "abc"), ":abc") {
		t.Errorf("state key %q must end with the state token", socialStateKey(appA, "abc"))
	}
}

func mustParseAppID(t *testing.T, s string) id.AppID {
	t.Helper()
	parsed, err := id.ParseAppID(s)
	if err != nil {
		t.Fatalf("parse app id %q: %v", s, err)
	}
	return parsed
}

// ──────────────────────────────────────────────────
// OIDC nonce
// ──────────────────────────────────────────────────

func TestVerifyOIDCNonce_NoExpectedNoncePasses(t *testing.T) {
	t.Parallel()
	// expectedNonce empty → non-OIDC flow. Always passes regardless of token.
	tok := &oauth2.Token{}
	if !verifyOIDCNonce(tok, "") {
		t.Error("empty expectedNonce must pass")
	}
}

func TestVerifyOIDCNonce_NoIDTokenPasses(t *testing.T) {
	t.Parallel()
	// Provider didn't return an ID token (non-OIDC). Pass through so
	// non-OIDC providers (Twitter, GitHub legacy) keep working.
	tok := &oauth2.Token{}
	if !verifyOIDCNonce(tok, "expected-nonce") {
		t.Error("missing id_token should pass (non-OIDC provider)")
	}
}

func TestVerifyOIDCNonce_MatchingNoncePasses(t *testing.T) {
	t.Parallel()
	tok := tokenWithIDToken(t, map[string]any{"nonce": "the-correct-nonce"})
	if !verifyOIDCNonce(tok, "the-correct-nonce") {
		t.Error("matching nonce must verify")
	}
}

func TestVerifyOIDCNonce_MismatchedNonceFails(t *testing.T) {
	t.Parallel()
	tok := tokenWithIDToken(t, map[string]any{"nonce": "wrong-nonce"})
	if verifyOIDCNonce(tok, "expected-nonce") {
		t.Error("mismatched nonce must fail")
	}
}

func TestVerifyOIDCNonce_MissingNonceClaimFails(t *testing.T) {
	t.Parallel()
	tok := tokenWithIDToken(t, map[string]any{"sub": "user-123"})
	if verifyOIDCNonce(tok, "expected-nonce") {
		t.Error("absent nonce claim with non-empty expectation must fail")
	}
}

func TestVerifyOIDCNonce_MalformedIDTokenFails(t *testing.T) {
	t.Parallel()
	tok := &oauth2.Token{}
	tok = tok.WithExtra(map[string]any{"id_token": "not.a.jwt.with.too.many.parts"})
	if verifyOIDCNonce(tok, "expected-nonce") {
		t.Error("malformed ID token must fail")
	}
}

func TestVerifyOIDCNonce_BadBase64Fails(t *testing.T) {
	t.Parallel()
	tok := &oauth2.Token{}
	tok = tok.WithExtra(map[string]any{"id_token": "header.\x00\x00not-base64.sig"})
	if verifyOIDCNonce(tok, "expected-nonce") {
		t.Error("undecodable payload must fail")
	}
}

// tokenWithIDToken builds an oauth2.Token with a synthetic ID token whose
// payload encodes the given claims. Signature is "sig" — verifyOIDCNonce
// does NOT validate signatures (separate hardening item), so a synthetic
// JWT is sufficient.
func tokenWithIDToken(t *testing.T, claims map[string]any) *oauth2.Token {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	encPayload := base64.RawURLEncoding.EncodeToString(payload)
	idToken := header + "." + encPayload + ".sig"

	tok := &oauth2.Token{}
	return tok.WithExtra(map[string]any{"id_token": idToken})
}
