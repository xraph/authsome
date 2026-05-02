// Package dashboard provides shared dashboard utilities.
package dashboard

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

// ─── Legacy single-use nonce (NOT session-bound) ────────────────────────────
//
// The functions below predate the HMAC-bound scoped nonce API. They mint a
// random hex string, store it in a global map, and consume it by lookup. They
// prevent accidental double-submits but DO NOT prove the consumer owns the
// session that minted the nonce — a nonce minted in admin-A's tab can be
// burned by a CSRF-forged POST that happens to arrive carrying admin-B's
// session cookie.
//
// New destructive call sites should use GenerateScopedNonce / ConsumeScopedNonce.
// Existing call sites continue to work; they will be migrated incrementally.

// nonceStore tracks form submission nonces to prevent duplicate submissions on refresh.
var nonceStore = struct {
	sync.Mutex
	nonces map[string]time.Time
}{nonces: make(map[string]time.Time)}

// GenerateNonce creates a random nonce and stores it for later validation.
//
// Deprecated: prefer GenerateScopedNonce, which binds the nonce to a specific
// session and action scope. This function remains for backward compatibility
// with non-destructive forms.
func GenerateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: failed to read random bytes: " + err.Error())
	}
	nonce := hex.EncodeToString(b)
	nonceStore.Lock()
	// Purge expired nonces (older than 10 minutes).
	cutoff := time.Now().Add(-10 * time.Minute)
	for k, t := range nonceStore.nonces {
		if t.Before(cutoff) {
			delete(nonceStore.nonces, k)
		}
	}
	nonceStore.nonces[nonce] = time.Now()
	nonceStore.Unlock()
	return nonce
}

// ConsumeNonce returns true if the nonce is valid and consumes it.
// A consumed nonce cannot be reused, preventing duplicate form submissions.
//
// Deprecated: prefer ConsumeScopedNonce. See GenerateNonce for the caveat.
func ConsumeNonce(nonce string) bool {
	if nonce == "" {
		return false
	}
	nonceStore.Lock()
	defer nonceStore.Unlock()
	if _, ok := nonceStore.nonces[nonce]; ok {
		delete(nonceStore.nonces, nonce)
		return true
	}
	return false
}

// ─── HMAC-bound scoped nonce ────────────────────────────────────────────────
//
// Token format:
//
//	base64url(timestampBE_uint64) + "." + base64url(hmac-sha256(secret, ts || "|" || sessionID || "|" || scope))
//
// A token is accepted by ConsumeScopedNonce only if:
//   - it decodes correctly,
//   - its HMAC verifies under (sessionID, scope, timestamp),
//   - the timestamp is not in the future beyond a small clock-skew window,
//   - the token is no older than nonceTTL,
//   - the token has not been consumed before in this process.
//
// Replay detection is per-process (in-memory). In multi-replica deployments,
// the same token replayed against two different replicas within the TTL is
// accepted by both. This is an accepted limitation — the HMAC binding still
// prevents the cross-session CSRF attack the legacy API was vulnerable to;
// a stolen but unused token replayed across replicas is no worse than the
// legacy global-map behaviour. A shared replay store is left as future work.

const (
	nonceTTL  = 10 * time.Minute
	clockSkew = 30 * time.Second
)

// ErrNonceSecretMissing is returned when the signing secret is too short or
// absent. A 16-byte minimum keeps accidental misconfiguration loud.
var ErrNonceSecretMissing = errors.New("dashboard: nonce signing secret not configured")

// nonceSigner produces and verifies HMAC-bound single-use tokens.
type nonceSigner struct {
	secret []byte
	now    func() time.Time

	mu       sync.Mutex
	consumed map[string]time.Time // mac (base64) → expiry
}

func newNonceSigner(secret []byte) (*nonceSigner, error) {
	if len(secret) < 16 {
		return nil, ErrNonceSecretMissing
	}
	cp := make([]byte, len(secret))
	copy(cp, secret)
	return &nonceSigner{
		secret:   cp,
		now:      time.Now,
		consumed: make(map[string]time.Time),
	}, nil
}

// Generate returns a token bound to (sessionID, scope) at the current time,
// or "" if either input is empty.
func (s *nonceSigner) Generate(sessionID, scope string) string {
	if sessionID == "" || scope == "" {
		return ""
	}
	ts := s.now().Unix()
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(ts))
	mac := s.sign(tsBytes, sessionID, scope)
	return base64.RawURLEncoding.EncodeToString(tsBytes) + "." + base64.RawURLEncoding.EncodeToString(mac)
}

// Consume validates the token under (sessionID, scope) and marks it used.
// It returns true at most once per token; replays return false.
func (s *nonceSigner) Consume(sessionID, scope, token string) bool {
	if sessionID == "" || scope == "" || token == "" {
		return false
	}
	dot := strings.IndexByte(token, '.')
	if dot <= 0 || dot == len(token)-1 {
		return false
	}
	tsBytes, err := base64.RawURLEncoding.DecodeString(token[:dot])
	if err != nil || len(tsBytes) != 8 {
		return false
	}
	macBytes, err := base64.RawURLEncoding.DecodeString(token[dot+1:])
	if err != nil || len(macBytes) != sha256.Size {
		return false
	}

	ts := int64(binary.BigEndian.Uint64(tsBytes))
	tokenTime := time.Unix(ts, 0)
	now := s.now()
	if tokenTime.After(now.Add(clockSkew)) {
		return false
	}
	if now.Sub(tokenTime) > nonceTTL {
		return false
	}

	expected := s.sign(tsBytes, sessionID, scope)
	if !hmac.Equal(macBytes, expected) {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Garbage-collect expired entries on every successful verification.
	for k, exp := range s.consumed {
		if exp.Before(now) {
			delete(s.consumed, k)
		}
	}

	macKey := token[dot+1:]
	if _, replay := s.consumed[macKey]; replay {
		return false
	}
	s.consumed[macKey] = tokenTime.Add(nonceTTL)
	return true
}

func (s *nonceSigner) sign(tsBytes []byte, sessionID, scope string) []byte {
	h := hmac.New(sha256.New, s.secret)
	h.Write(tsBytes)
	h.Write([]byte{'|'})
	h.Write([]byte(sessionID))
	h.Write([]byte{'|'})
	h.Write([]byte(scope))
	return h.Sum(nil)
}

// ─── Package-level facade ───────────────────────────────────────────────────

var (
	defaultSigner   *nonceSigner
	defaultSignerMu sync.Mutex
)

// InitNonceSigner installs the process-wide signer used by GenerateScopedNonce
// and ConsumeScopedNonce. Calling it again replaces the signer (and clears
// in-flight replay state, so it should normally only be called once at
// startup). A secret shorter than 16 bytes returns ErrNonceSecretMissing and
// leaves any existing signer in place.
func InitNonceSigner(secret []byte) error {
	s, err := newNonceSigner(secret)
	if err != nil {
		return err
	}
	defaultSignerMu.Lock()
	defaultSigner = s
	defaultSignerMu.Unlock()
	return nil
}

// GenerateScopedNonce mints an HMAC-bound nonce for (sessionID, scope). It
// returns "" if the signer is uninitialised or either input is empty.
func GenerateScopedNonce(sessionID, scope string) string {
	defaultSignerMu.Lock()
	s := defaultSigner
	defaultSignerMu.Unlock()
	if s == nil {
		return ""
	}
	return s.Generate(sessionID, scope)
}

// ConsumeScopedNonce verifies and burns a previously-generated scoped nonce.
// It returns false if the signer is uninitialised, the token is malformed,
// expired, replayed, or bound to a different (sessionID, scope) pair.
func ConsumeScopedNonce(sessionID, scope, token string) bool {
	defaultSignerMu.Lock()
	s := defaultSigner
	defaultSignerMu.Unlock()
	if s == nil {
		return false
	}
	return s.Consume(sessionID, scope, token)
}
