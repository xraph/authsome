package apikey_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
)

// A key's last use is written once a minute, not once a request, and the
// key's hash is left alone when it needs no rehash.
func TestAuthenticate_TouchesLastUsedOncePerMinute(t *testing.T) {
	p, store := newTestPlugin()
	require.NoError(t, p.OnInit(context.Background(), &mockEngine{logger: log.NewNoopLogger(), store: store}))
	s := p.Strategy()

	appID, userID := id.NewAppID(), id.NewUserID()
	raw, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)
	now := time.Now()
	key := &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: userID, Name: "Touch Key", KeyHash: hash, KeyPrefix: prefix,
		CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, store.CreateAPIKey(context.Background(), key))

	authenticate := func() {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		req.Header.Set("X-App-ID", appID.String())
		_, authErr := s.Authenticate(context.Background(), req)
		require.NoError(t, authErr)
	}

	authenticate()
	got, err := store.GetAPIKey(context.Background(), key.ID)
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt, "the first use is recorded")
	first := *got.LastUsedAt
	assert.Equal(t, hash, got.KeyHash, "a key that needs no rehash keeps its digest")

	authenticate()
	got, err = store.GetAPIKey(context.Background(), key.ID)
	require.NoError(t, err)
	assert.Equal(t, first, *got.LastUsedAt, "a use inside the minute is not written")

	old := first.Add(-2 * time.Minute)
	require.NoError(t, store.TouchAPIKey(context.Background(), key.ID, old))
	authenticate()
	got, err = store.GetAPIKey(context.Background(), key.ID)
	require.NoError(t, err)
	assert.True(t, got.LastUsedAt.After(old), "a use after the minute is written")
}
