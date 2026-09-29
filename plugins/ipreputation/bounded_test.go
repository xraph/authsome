package ipreputation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The reputation cache stays within its bound no matter how many
// addresses are looked up, and the newest stay cached.
func TestCacheIsBounded(t *testing.T) {
	p := New(Config{MaxCachedIPs: 2})
	for i := range 5 {
		p.setCache(fmt.Sprintf("10.0.0.%d", i), &IPReputation{IP: fmt.Sprintf("10.0.0.%d", i), Score: i})
	}
	assert.Equal(t, 2, p.cache.Len())
	assert.Nil(t, p.getCached("10.0.0.0"), "the oldest address was dropped")
	assert.NotNil(t, p.getCached("10.0.0.4"), "the newest address is cached")
}
