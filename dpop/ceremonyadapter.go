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
// Opt-in only. A ceremony store handles a few writes per sign-in, while this
// takes a write on every authenticated request, so pointing it at a
// database-backed store buys a round trip per API call. Worth it when you run
// several instances and want replay detection to hold across all of them.
type ceremonyReplayCache struct {
	store ceremony.Store
}

var _ ReplayCache = (*ceremonyReplayCache)(nil)

// NewCeremonyReplayCache returns a ReplayCache backed by a ceremony store.
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
