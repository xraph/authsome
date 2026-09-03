package consent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestHasConsent(t *testing.T) {
	ctx := context.Background()
	p := &Plugin{}
	p.SetConsentStore(NewMemoryStore())
	userID, appID := id.NewUserID(), id.NewAppID()

	ok, err := p.HasConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	assert.False(t, ok, "no record means no consent")

	require.NoError(t, p.store.GrantConsent(ctx, &Consent{
		ID: id.NewConsentID(), UserID: userID, AppID: appID,
		Purpose: "marketing", Granted: true, Version: "1", GrantedAt: time.Now(),
	}))

	ok, err = p.HasConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = p.HasConsent(ctx, userID, appID, "analytics")
	require.NoError(t, err)
	assert.False(t, ok, "a grant for one purpose is not a grant for another")

	require.NoError(t, p.store.RevokeConsent(ctx, userID, appID, "marketing"))
	ok, err = p.HasConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	assert.False(t, ok, "a revoked grant reports false")
}

func TestHasConsentWithoutStore(t *testing.T) {
	ok, err := (&Plugin{}).HasConsent(context.Background(),
		id.NewUserID(), id.NewAppID(), "marketing")
	require.NoError(t, err)
	assert.False(t, ok, "an unconfigured store must not report consent")
}
