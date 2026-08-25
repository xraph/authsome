package dpop_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/dpop"
)

var nonceSecret = []byte("a-test-secret-at-least-16-bytes-long")

func TestNonceSigner_IssueThenVerify(t *testing.T) {
	s, err := dpop.NewNonceSigner(nonceSecret, dpop.DefaultNonceTTL)
	require.NoError(t, err)

	n := s.Issue("jkt-abc")
	assert.NotEmpty(t, n)
	assert.True(t, s.Verify("jkt-abc", n))
}

// TestNonceSigner_IsReusable is the difference from dashboard/nonce.go. A DPoP
// nonce is used for every request in its lifetime, so consuming it on first
// use would break the client on its second request.
func TestNonceSigner_IsReusable(t *testing.T) {
	s, err := dpop.NewNonceSigner(nonceSecret, dpop.DefaultNonceTTL)
	require.NoError(t, err)

	n := s.Issue("jkt-abc")
	for i := range 5 {
		assert.True(t, s.Verify("jkt-abc", n), "verification %d must still succeed", i)
	}
}

// TestNonceSigner_BoundToKey stops a nonce issued to one client being used by
// another.
func TestNonceSigner_BoundToKey(t *testing.T) {
	s, err := dpop.NewNonceSigner(nonceSecret, dpop.DefaultNonceTTL)
	require.NoError(t, err)

	n := s.Issue("jkt-abc")
	assert.False(t, s.Verify("jkt-different", n))
}

func TestNonceSigner_RejectsForeignSecret(t *testing.T) {
	mine, err := dpop.NewNonceSigner(nonceSecret, dpop.DefaultNonceTTL)
	require.NoError(t, err)
	theirs, err := dpop.NewNonceSigner([]byte("a-completely-different-secret-16"), dpop.DefaultNonceTTL)
	require.NoError(t, err)

	assert.False(t, mine.Verify("jkt-abc", theirs.Issue("jkt-abc")))
}

func TestNonceSigner_RejectsExpired(t *testing.T) {
	s, err := dpop.NewNonceSigner(nonceSecret, time.Millisecond)
	require.NoError(t, err)

	n := s.Issue("jkt-abc")
	time.Sleep(10 * time.Millisecond)
	assert.False(t, s.Verify("jkt-abc", n))
}

func TestNonceSigner_RejectsGarbage(t *testing.T) {
	s, err := dpop.NewNonceSigner(nonceSecret, dpop.DefaultNonceTTL)
	require.NoError(t, err)

	for _, bad := range []string{"", ".", "abc", "abc.def", "....", "!!!.???"} {
		assert.False(t, s.Verify("jkt-abc", bad), "must reject %q", bad)
	}
}

// TestNewNonceSigner_RequiresSecret fails closed. Engine.NonceSecret can return
// nil, and minting nonces from a per-process random secret would produce
// nonces no sibling instance can verify.
func TestNewNonceSigner_RequiresSecret(t *testing.T) {
	_, err := dpop.NewNonceSigner(nil, dpop.DefaultNonceTTL)
	assert.ErrorIs(t, err, dpop.ErrNonceSecretMissing)

	_, err = dpop.NewNonceSigner([]byte("too-short"), dpop.DefaultNonceTTL)
	assert.ErrorIs(t, err, dpop.ErrNonceSecretMissing)
}

func TestNonceSigner_NeedsRefresh(t *testing.T) {
	s, err := dpop.NewNonceSigner(nonceSecret, dpop.DefaultNonceTTL)
	require.NoError(t, err)

	assert.False(t, s.NeedsRefresh(s.Issue("jkt-abc")), "a fresh nonce does not need rotating")
	assert.True(t, s.NeedsRefresh("garbage"), "an unparseable nonce always needs rotating")
}

// TestNonceSigner_NilReceiverFailsClosed pins down what happens when a
// caller assigns a still-nil *NonceSigner into an interface field (e.g.
// dpop.Config.Nonce) before a real signer has been derived. The interface
// value produced that way is not == nil, so a `cfg.Nonce == nil` guard at
// the call site does not save you: the method actually gets invoked on a
// nil receiver. These three methods must not panic when that happens, and
// must return the answer that reads as "no nonce support", not a crash.
//
// Verify and NeedsRefresh are fed a well-formed nonce minted by a real
// signer, not a garbage string. A garbage string like "anything" has no
// '.' in it, so splitNonce's own ok=false check would return before either
// method ever touches the receiver. That would pass with or without the
// nil guard and prove nothing. Feeding a real nonce forces execution past
// splitNonce and into s.now()/s.sign(), which is where the guard is the
// only thing standing between "returns false/true" and a nil-pointer panic.
func TestNonceSigner_NilReceiverFailsClosed(t *testing.T) {
	valid, err := dpop.NewNonceSigner(nonceSecret, dpop.DefaultNonceTTL)
	require.NoError(t, err)
	wellFormedNonce := valid.Issue("jkt-abc")
	require.NotEmpty(t, wellFormedNonce)

	var s *dpop.NonceSigner

	assert.Equal(t, "", s.Issue("jkt-abc"), "a nil signer cannot mint a nonce")
	assert.False(t, s.Verify("jkt-abc", wellFormedNonce), "a nil signer cannot verify a nonce, even a well-formed one")
	assert.True(t, s.NeedsRefresh(wellFormedNonce), "a nil signer can't vouch that a well-formed nonce is still fresh")
}

// TestNonceSigner_NeedsRefresh_Boundary pins the half-TTL crossover with a
// real clock instead of just the two endpoints above.
//
// Issue encodes the timestamp as whole Unix seconds, so up to ~1s of the
// current second is truncated away before NeedsRefresh ever sees it: right
// after issuing a nonce, now().Sub(issued) can already read close to 1s even
// though no real time has passed. A short TTL (say, low hundreds of ms)
// would make that truncation noise bigger than the boundary itself, so this
// uses a TTL long enough that the noise is a small fraction of the halfway
// point, plus a sleep well clear of it in both directions, to stay
// deterministic under ordinary scheduling jitter without an injectable
// clock.
func TestNonceSigner_NeedsRefresh_Boundary(t *testing.T) {
	const ttl = 4 * time.Second
	s, err := dpop.NewNonceSigner(nonceSecret, ttl)
	require.NoError(t, err)

	n := s.Issue("jkt-abc")
	assert.False(t, s.NeedsRefresh(n), "immediately after issue, well inside the first half of the TTL")

	time.Sleep(2500 * time.Millisecond)
	assert.True(t, s.NeedsRefresh(n), "past the halfway point, comfortably clear of the 2s boundary")
}
