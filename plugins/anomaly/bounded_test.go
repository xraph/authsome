package anomaly

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

// The pattern table stays within its bound no matter how many principals
// log in.
func TestPatternTableIsBounded(t *testing.T) {
	p := New(Config{MaxTrackedPrincipals: 2})
	ctx := context.Background()
	for range 5 {
		require.NoError(t, p.recordLogin(ctx, principal.UserRef(id.NewUserID()), "app", id.NewSessionID(), ""))
	}
	assert.Equal(t, 2, p.patterns.Len(), "only the newest principals are tracked")
}
