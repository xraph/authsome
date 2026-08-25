package dpop

import (
	"context"
	"sync"
)

// One HTTP request can reach more than one enforcement point. The global auth
// middleware runs, then a route-level auth provider runs, then a handler that
// binds or rotates a token runs, and all three read the same DPoP header off
// the same request.
//
// Every check in Validate is a pure comparison except one. The replay cache
// records a jti the first time it sees it, so the second enforcement point on
// a single request would see the request's own proof as a replay of itself and
// refuse a perfectly good presentation. The fix is not to weaken the replay
// cache but to remember, for the life of one request, which proofs already
// passed it.
//
// The record lives in the request context and nowhere else, so it cannot
// outlive the request or be reached from another one. A proof genuinely
// replayed on a second request arrives with a fresh context, finds nothing
// recorded, and is caught by the cache exactly as before.

// requestScope records the proofs that have already cleared the replay cache
// on one request.
type requestScope struct {
	mu       sync.Mutex
	accepted map[string]struct{}
}

type requestScopeKey struct{}

// WithRequestScope returns a context that remembers which proofs have already
// been validated under it, so a second enforcement point on the same request
// does not report the request's own proof as a replay.
//
// Installing it twice is a no-op: the first scope wins, which keeps the record
// shared across every check on the request rather than split between nested
// contexts.
func WithRequestScope(ctx context.Context) context.Context {
	if requestScopeFrom(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, requestScopeKey{}, &requestScope{
		accepted: make(map[string]struct{}, 1),
	})
}

// requestScopeFrom returns the scope installed on ctx, or nil. Nil is the
// ordinary case for a caller that never installed one, and it means every
// validation consults the replay cache directly, which is the behaviour that
// predates this file.
func requestScopeFrom(ctx context.Context) *requestScope {
	if ctx == nil {
		return nil
	}
	s, _ := ctx.Value(requestScopeKey{}).(*requestScope) //nolint:errcheck // type-safe via key
	return s
}

// accept records that key passed the replay cache on this request.
func (s *requestScope) accept(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accepted[key] = struct{}{}
}

// accepted reports whether key already passed the replay cache on this
// request.
func (s *requestScope) alreadyAccepted(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.accepted[key]
	return ok
}
