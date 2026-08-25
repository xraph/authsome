package dpop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/ceremony"
)

// ceremonyReplayCache backs a ReplayCache with a ceremony.Store.
//
// Opt-in only, and it trades exactness within a process for coverage across
// processes. The in-memory default holds one lock across its whole
// check-then-record step, so within a single instance two requests racing on
// the same jti can never both be told "not seen". This adapter cannot make
// that same promise, because it checks and records as two separate store
// calls (Get, then Set) and ceremony.Store has no compare-and-set to join
// them. Two requests carrying the same jti that land on different instances,
// or that land on the same instance close enough together to both pass Get
// before either has called Set, can both be reported as not seen.
//
// What you do get in exchange is real: without this adapter, a captured
// proof can be replayed against any instance behind the load balancer for
// the whole freshness window, because the in-memory cache on each instance
// knows nothing of what the others have seen. With it, replay only succeeds
// in that narrow concurrent race, not at any point later in the window. That
// is a meaningfully smaller opening, just not a closed one.
//
// The cost is a ceremony store round trip on every authenticated request,
// not the few writes per sign-in a ceremony store is normally sized for.
// Reach for this when you run more than one instance and that reduced,
// race-only replay window is an acceptable trade for coverage that spans
// them; stay with the in-memory default otherwise.
type ceremonyReplayCache struct {
	store ceremony.Store
}

var _ ReplayCache = (*ceremonyReplayCache)(nil)

// NewCeremonyReplayCache returns a ReplayCache backed by a ceremony store. See
// ceremonyReplayCache for what this does and does not guarantee under
// concurrent replay attempts.
func NewCeremonyReplayCache(s ceremony.Store) ReplayCache {
	return &ceremonyReplayCache{store: s}
}

const ceremonyReplayPrefix = "dpop:jti:"

func (c *ceremonyReplayCache) Seen(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	full := ceremonyReplayPrefix + key

	_, err := c.store.Get(ctx, full)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ceremony.ErrNotFound):
		// Not a replay. Fall through and record it.
	default:
		return false, fmt.Errorf("dpop: replay cache get: %w", err)
	}

	if err := c.store.Set(ctx, full, []byte{1}, ttl); err != nil {
		return false, fmt.Errorf("dpop: replay cache set: %w", err)
	}
	return false, nil
}
