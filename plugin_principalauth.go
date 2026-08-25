package authsome

import (
	"context"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// principalAuthGate runs the BeforePrincipalAuth chain for machine callers and
// caches the verdict.
//
// The cache is what makes scoring machine traffic affordable. Sign-in happens
// once a day per human; an agent authenticates on every call, and the risk
// contributors do geo lookups and reputation lookups. Keyed on credential and
// source IP together, so the same key appearing from a new address is scored
// fresh rather than riding an earlier allow.
type principalAuthGate struct {
	authorize func(context.Context, *principal.AuthAttempt) error
	observe   func(context.Context, *principal.AuthAttempt, *session.Session)
	ttl       time.Duration
	logger    log.Logger

	mu      sync.Mutex
	entries map[string]time.Time // cache key -> when the allow expires
}

func newPrincipalAuthGate(
	authorize func(context.Context, *principal.AuthAttempt) error,
	observe func(context.Context, *principal.AuthAttempt, *session.Session),
	ttl time.Duration,
	logger log.Logger,
) *principalAuthGate {
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	return &principalAuthGate{
		authorize: authorize,
		observe:   observe,
		ttl:       ttl,
		logger:    logger,
		entries:   make(map[string]time.Time),
	}
}

// cacheKey returns the cache key for a and whether the verdict may be cached
// at all.
//
// a.CredentialID is required. id.ID.String() returns "" for a nil ID, so an
// attempt with no credential id (and, worse, no IP either) would collapse to
// the same key, "|" or "|<ip>", as every other caller missing a credential
// id, letting one such caller's allow be served to all of them for the TTL.
// This gate is a general engine API reachable from any plugin via
// PrincipalAuthGate() any, so that is not a theoretical caller, it is one
// this function must not trust to always fill the field in. Score those
// attempts fresh every time instead of sharing a key for them.
func cacheKey(a *principal.AuthAttempt) (key string, cacheable bool) {
	if a.CredentialID == "" {
		return "", false
	}
	return a.CredentialID + "|" + a.IPAddress, true
}

// Authorize scores a, denying if any plugin does.
//
// Only allows are cached. A denial is re-evaluated every time, because the
// condition behind it (a reputation listing, a travel impossibility) can clear
// within the TTL, and a cached denial would keep a caller locked out after the
// reason had gone.
func (g *principalAuthGate) Authorize(ctx context.Context, a *principal.AuthAttempt) error {
	if g.authorize == nil {
		return nil
	}

	// Cache bookkeeping, the expiry stored and compared below, always runs
	// on server time. a.At is caller-supplied (the apikey strategy sets it
	// to its own time.Now(), but this is a general API) and is passed to
	// g.authorize below unchanged for the risk contributors' own use; letting
	// it drive the cache too would let a caller push an entry's expiry
	// arbitrarily into the future.
	now := time.Now()

	key, cacheable := cacheKey(a)
	if cacheable {
		g.mu.Lock()
		expires, cached := g.entries[key]
		g.mu.Unlock()
		if cached && now.Before(expires) {
			return nil
		}
	}

	if err := g.authorize(ctx, a); err != nil {
		return err
	}

	if cacheable {
		g.mu.Lock()
		g.entries[key] = now.Add(g.ttl)
		g.evictLocked(now)
		g.mu.Unlock()
	}
	return nil
}

// Observe runs the after hooks and emits the bus event.
func (g *principalAuthGate) Observe(ctx context.Context, a *principal.AuthAttempt, s *session.Session) {
	if g.observe != nil {
		g.observe(ctx, a, s)
	}
}

// evictLocked removes entries that can no longer serve a hit. The caller holds
// g.mu.
//
// This runs on the authentication path for every machine caller, so what it
// costs matters as much as what it reclaims.
func (g *principalAuthGate) evictLocked(now time.Time) { //nolint:revive // now feeds the eviction policy below, which is deliberately unimplemented
	// TODO: eviction policy is the repository owner's decision. Deliberately
	// left unimplemented so the map's growth bound is chosen explicitly
	// rather than defaulted into. See task 12 of the non-human principals plan.
}
