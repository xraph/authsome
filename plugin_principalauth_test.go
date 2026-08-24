package authsome

import (
	"context"
	"errors"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/principal"
)

func attempt(credID, ip string) *principal.AuthAttempt {
	return &principal.AuthAttempt{
		Subject:        principal.Ref{Kind: principal.KindService, ID: "svc_1"},
		CredentialKind: "api_key",
		CredentialID:   credID,
		IPAddress:      ip,
		At:             time.Now(),
	}
}

// A denying plugin must deny the request. Without this the six risk plugins
// see machine traffic but cannot act on it, which is worse than not seeing it.
func TestPrincipalAuthGateDenies(t *testing.T) {
	denied := errors.New("blocked by risk")
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		return denied
	}, nil, time.Minute, log.NewNoopLogger())

	err := g.Authorize(context.Background(), attempt("akey_1", "1.2.3.4"))
	assert.ErrorIs(t, err, denied)
}

// A repeat call with the same credential and IP inside the TTL must not
// re-run the contributor chain. A chatty agent would otherwise pay for a geo
// and reputation lookup on every single call.
func TestPrincipalAuthGateCachesTheVerdict(t *testing.T) {
	var calls int
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		calls++
		return nil
	}, nil, time.Minute, log.NewNoopLogger())

	ctx := context.Background()
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	assert.Equal(t, 1, calls, "the second call must be served from the cache")
}

// A different source IP is a different verdict. The same key used from a new
// country is exactly what impossibletravel and geofence exist to catch, so it
// must not ride the cached allow.
func TestPrincipalAuthGateKeysOnIP(t *testing.T) {
	var calls int
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		calls++
		return nil
	}, nil, time.Minute, log.NewNoopLogger())

	ctx := context.Background()
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "5.6.7.8")))
	assert.Equal(t, 2, calls, "a new source IP must be scored fresh")
}

// A denial must not be cached as long as an allow, or a transient block
// outlives the condition that caused it. Denials are re-evaluated every time.
func TestPrincipalAuthGateDoesNotCacheDenials(t *testing.T) {
	var calls int
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		calls++
		return errors.New("blocked")
	}, nil, time.Minute, log.NewNoopLogger())

	ctx := context.Background()
	_ = g.Authorize(ctx, attempt("akey_1", "1.2.3.4"))
	_ = g.Authorize(ctx, attempt("akey_1", "1.2.3.4"))
	assert.Equal(t, 2, calls, "a denial must be re-evaluated, not cached")
}

// An expired entry is re-scored.
func TestPrincipalAuthGateExpiresEntries(t *testing.T) {
	var calls int
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		calls++
		return nil
	}, nil, time.Nanosecond, log.NewNoopLogger())

	ctx := context.Background()
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	time.Sleep(time.Millisecond)
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	assert.Equal(t, 2, calls)
}
