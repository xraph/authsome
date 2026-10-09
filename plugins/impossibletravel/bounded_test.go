package impossibletravel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

// The last-login table and the alert list stay within their bounds no
// matter how many principals log in or how many alerts fire.
func TestInMemoryStateIsBounded(t *testing.T) {
	p := newTestPlugin(Config{MaxTrackedPrincipals: 2, MaxEvents: 3}, defaultMapping)
	ctx := context.Background()
	for range 5 {
		require.NoError(t, p.recordLocation(ctx, principal.UserRef(id.NewUserID()), "app", id.NewSessionID(), "1.1.1.1"))
	}
	assert.Equal(t, 2, p.lastLogins.Len(), "only the newest principals are tracked")

	ref := principal.UserRef(id.NewUserID())
	for i := range 10 {
		ip := "1.1.1.1"
		if i%2 == 1 {
			ip = "3.3.3.3" // New York to Sydney and back, instantly
		}
		require.NoError(t, p.recordLocation(ctx, ref, "app", id.NewSessionID(), ip))
	}
	assert.Len(t, p.RecordedEvents(), 3, "only the newest alerts are kept")
}
