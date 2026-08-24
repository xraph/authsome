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
