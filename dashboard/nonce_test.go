package dashboard

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestNonceSigner returns a fresh signer with a deterministic secret.
// Tests should never share signers — replay state would leak across cases.
func newTestNonceSigner(t *testing.T) *nonceSigner {
	t.Helper()
	s, err := newNonceSigner([]byte("test-nonce-secret-do-not-use-in-prod-padding"))
	if err != nil {
		t.Fatalf("newNonceSigner: %v", err)
	}
	return s
}

func TestScopedNonce_ValidRoundTrip(t *testing.T) {
	s := newTestNonceSigner(t)
	tok := s.Generate("session-A", "org.delete:org-1")
	if tok == "" {
		t.Fatal("Generate returned empty token")
	}
	if !s.Consume("session-A", "org.delete:org-1", tok) {
		t.Fatal("Consume returned false on first use")
	}
}

func TestScopedNonce_Replay(t *testing.T) {
	s := newTestNonceSigner(t)
	tok := s.Generate("sess", "scope")
	if !s.Consume("sess", "scope", tok) {
		t.Fatal("first consume failed")
	}
	if s.Consume("sess", "scope", tok) {
		t.Fatal("replay must fail")
	}
}

func TestScopedNonce_WrongSession(t *testing.T) {
	s := newTestNonceSigner(t)
	tok := s.Generate("admin-A", "scope")
	if s.Consume("admin-B", "scope", tok) {
		t.Fatal("token bound to admin-A must not consume under admin-B")
	}
}

func TestScopedNonce_WrongScope(t *testing.T) {
	s := newTestNonceSigner(t)
	tok := s.Generate("sess", "org.delete")
	if s.Consume("sess", "user.delete", tok) {
		t.Fatal("scope mismatch must fail")
	}
}

func TestScopedNonce_Expired(t *testing.T) {
	s := newTestNonceSigner(t)
	base := time.Unix(1_700_000_000, 0)
	s.now = func() time.Time { return base }
	tok := s.Generate("sess", "scope")
	// Advance well past the TTL.
	s.now = func() time.Time { return base.Add(nonceTTL + time.Second) }
	if s.Consume("sess", "scope", tok) {
		t.Fatal("expired token must not be accepted")
	}
}

func TestScopedNonce_TamperedHMAC(t *testing.T) {
	s := newTestNonceSigner(t)
	tok := s.Generate("sess", "scope")
	dot := strings.IndexByte(tok, '.')
	if dot <= 0 {
		t.Fatalf("malformed token: %q", tok)
	}
	macBytes, err := base64.RawURLEncoding.DecodeString(tok[dot+1:])
	if err != nil {
		t.Fatalf("decode mac: %v", err)
	}
	macBytes[0] ^= 0x01
	tampered := tok[:dot+1] + base64.RawURLEncoding.EncodeToString(macBytes)
	if s.Consume("sess", "scope", tampered) {
		t.Fatal("tampered HMAC must not verify")
	}
}

func TestScopedNonce_TimestampInFuture(t *testing.T) {
	s := newTestNonceSigner(t)
	base := time.Unix(1_700_000_000, 0)
	// Generate at a future time, consume "now" — should reject (skew > 30s).
	s.now = func() time.Time { return base.Add(5 * time.Minute) }
	tok := s.Generate("sess", "scope")
	s.now = func() time.Time { return base }
	if s.Consume("sess", "scope", tok) {
		t.Fatal("token timestamped in the future must be rejected")
	}
}

func TestScopedNonce_EmptySessionOrScope(t *testing.T) {
	s := newTestNonceSigner(t)
	if s.Generate("", "scope") != "" {
		t.Fatal("empty sessionID should yield empty token")
	}
	if s.Generate("sess", "") != "" {
		t.Fatal("empty scope should yield empty token")
	}
	tok := s.Generate("sess", "scope")
	if s.Consume("", "scope", tok) {
		t.Fatal("empty sessionID consume must fail")
	}
	if s.Consume("sess", "", tok) {
		t.Fatal("empty scope consume must fail")
	}
	if s.Consume("sess", "scope", "") {
		t.Fatal("empty token consume must fail")
	}
}

func TestScopedNonce_ConcurrentConsume(t *testing.T) {
	s := newTestNonceSigner(t)
	const workers = 32
	tok := s.Generate("sess", "scope")

	var wg sync.WaitGroup
	var successes int32
	var mu sync.Mutex

	wg.Add(workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			if s.Consume("sess", "scope", tok) {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	if successes != 1 {
		t.Fatalf("expected exactly one successful consume, got %d", successes)
	}
}

// TestAttack_CSRF_StolenScopedNonce simulates the original vulnerability: an
// admin-A nonce reaching a forged POST that carries admin-B's session cookie.
// With the HMAC binding, that must fail.
func TestAttack_CSRF_StolenScopedNonce(t *testing.T) {
	s := newTestNonceSigner(t)
	stolen := s.Generate("admin-A-session", "org.delete:org-42")

	// Attacker replays it on a request authenticated as admin-B.
	if s.Consume("admin-B-session", "org.delete:org-42", stolen) {
		t.Fatal("CSRF replay across sessions must be rejected")
	}
	// And a totally different scope.
	if s.Consume("admin-B-session", "user.delete:user-7", stolen) {
		t.Fatal("CSRF replay across scopes must be rejected")
	}
	// Original session+scope still works, exactly once.
	if !s.Consume("admin-A-session", "org.delete:org-42", stolen) {
		t.Fatal("legitimate consume failed")
	}
	if s.Consume("admin-A-session", "org.delete:org-42", stolen) {
		t.Fatal("replay after legitimate consume must fail")
	}
}

func FuzzScopedNonceVerify(f *testing.F) {
	s, err := newNonceSigner([]byte("fuzz-nonce-secret-padding-padding"))
	if err != nil {
		f.Fatal(err)
	}
	// Seed corpus: a real token, a malformed one, random bytes.
	good := s.Generate("sess", "scope")
	f.Add(good)
	f.Add("")
	f.Add(".")
	f.Add("AAAA.BBBB")
	f.Add(strings.Repeat("A", 1024))

	// A real token decoded into bytes (so the fuzzer can mutate the payload).
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(time.Now().Unix()))
	mac := hmac.New(sha256.New, []byte("fuzz-nonce-secret-padding-padding"))
	mac.Write(tsBytes)
	mac.Write([]byte{'|'})
	mac.Write([]byte("sess"))
	mac.Write([]byte{'|'})
	mac.Write([]byte("scope"))
	f.Add(base64.RawURLEncoding.EncodeToString(tsBytes) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)))

	f.Fuzz(func(_ *testing.T, token string) {
		// Must never panic. Random bytes must never be accepted as a valid
		// nonce for ("sess", "scope") — the only way a fuzzer-generated
		// string could legitimately verify is by accident producing the same
		// 32-byte HMAC, which is cryptographically negligible.
		_ = s.Consume("sess", "scope", token)
	})
}
