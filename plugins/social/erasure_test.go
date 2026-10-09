package social_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/social"
)

// A deleted user's provider connections, and the provider tokens on them,
// go with the account.
func TestOnBeforeUserDelete_RemovesConnections(t *testing.T) {
	p, _, oauthStore := newTestPlugin(t, newMockProvider("google"))
	ctx := context.Background()
	appID := id.NewAppID()
	userID, other := id.NewUserID(), id.NewUserID()
	for _, uid := range []id.UserID{userID, other} {
		require.NoError(t, oauthStore.CreateOAuthConnection(ctx, &social.OAuthConnection{
			ID: id.NewOAuthConnectionID(), AppID: appID, UserID: uid, Provider: "google", ProviderUserID: "g-" + uid.String(),
			AccessToken: "at", RefreshToken: "rt", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	require.NoError(t, p.OnBeforeUserDelete(ctx, userID))

	mine, err := oauthStore.GetOAuthConnectionsByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, mine)
	theirs, err := oauthStore.GetOAuthConnectionsByUserID(ctx, other)
	require.NoError(t, err)
	assert.Len(t, theirs, 1, "another user's connections stay")
}
