//go:build integration

package mongo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/grove/drivers/mongodriver"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/store"
)

// EnsureTTLIndexes creates a TTL index per kind, changes its window in
// place on a later call, and leaves out kinds that are not asked for.
func TestEnsureTTLIndexes(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	raw := mongodriver.New()
	require.NoError(t, raw.Open(ctx, os.Getenv("AUTHSOME_MONGO_URI")))
	t.Cleanup(func() { _ = raw.Close() })

	ttlOf := func(col, field string) *int32 {
		cur, err := raw.Collection(col).Indexes().List(ctx)
		require.NoError(t, err)
		var specs []struct {
			Key                bson.D `bson:"key"`
			ExpireAfterSeconds *int32 `bson:"expireAfterSeconds"`
		}
		require.NoError(t, cur.All(ctx, &specs))
		for _, ix := range specs {
			if len(ix.Key) == 1 && ix.Key[0].Key == field {
				return ix.ExpireAfterSeconds
			}
		}
		return nil
	}

	require.NoError(t, s.EnsureTTLIndexes(ctx, map[string]time.Duration{
		store.RetentionSessions:      time.Hour,
		store.RetentionVerifications: 2 * time.Hour,
	}))
	require.NotNil(t, ttlOf("authsome_sessions", "refresh_token_expires_at"))
	assert.EqualValues(t, 3600, *ttlOf("authsome_sessions", "refresh_token_expires_at"))
	assert.EqualValues(t, 7200, *ttlOf("authsome_verifications", "expires_at"))
	assert.Nil(t, ttlOf("authsome_password_resets", "expires_at"), "a kind not asked for gets no index")

	require.NoError(t, s.EnsureTTLIndexes(ctx, map[string]time.Duration{store.RetentionSessions: 3 * time.Hour}))
	assert.EqualValues(t, 10800, *ttlOf("authsome_sessions", "refresh_token_expires_at"), "the window is changed in place")
	require.NoError(t, s.EnsureTTLIndexes(ctx, map[string]time.Duration{store.RetentionSessions: 3 * time.Hour}), "idempotent")
}
