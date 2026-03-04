package apikey_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"

	"github.com/xraph/keysmith"
	ksmemory "github.com/xraph/keysmith/store/memory"
)

func testKeymithStore(t *testing.T) *apikey.KeysmithStore {
	t.Helper()
	ksStore := ksmemory.New()
	ksEng, err := keysmith.NewEngine(keysmith.WithStore(ksStore))
	require.NoError(t, err)
	return apikey.NewKeymithStore(ksEng)
}

func ctx() context.Context { return context.Background() }

func TestKeymithStore_CRUD(t *testing.T) {
	s := testKeymithStore(t)

	appID := id.NewAppID()
	userID := id.NewUserID()
	keyID := id.NewAPIKeyID()

	now := time.Now().Truncate(time.Millisecond)
	ak := &apikey.APIKey{
		ID:        keyID,
		AppID:     appID,
		UserID:    userID,
		Name:      "test-key",
		KeyHash:   "abc123hash",
		KeyPrefix: "ask_abcd1234",
		Scopes:    []string{"read", "write"},
		Revoked:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create
	err := s.CreateAPIKey(ctx(), ak)
	require.NoError(t, err)

	// Get by ID
	got, err := s.GetAPIKey(ctx(), keyID)
	require.NoError(t, err)
	assert.Equal(t, keyID.String(), got.ID.String())
	assert.Equal(t, appID.String(), got.AppID.String())
	assert.Equal(t, userID.String(), got.UserID.String())
	assert.Equal(t, "test-key", got.Name)
	assert.Equal(t, "abc123hash", got.KeyHash)
	assert.Equal(t, "ask_abcd1234", got.KeyPrefix)
	assert.False(t, got.Revoked)

	// Get by prefix
	gotByPrefix, err := s.GetAPIKeyByPrefix(ctx(), appID, "ask_abcd1234")
	require.NoError(t, err)
	assert.Equal(t, keyID.String(), gotByPrefix.ID.String())

	// Update
	ak.Name = "updated-key"
	ak.UpdatedAt = time.Now().Truncate(time.Millisecond)
	err = s.UpdateAPIKey(ctx(), ak)
	require.NoError(t, err)

	got, err = s.GetAPIKey(ctx(), keyID)
	require.NoError(t, err)
	assert.Equal(t, "updated-key", got.Name)

	// List by app
	appKeys, err := s.ListAPIKeysByApp(ctx(), appID)
	require.NoError(t, err)
	assert.Len(t, appKeys, 1)

	// List by user
	userKeys, err := s.ListAPIKeysByUser(ctx(), appID, userID)
	require.NoError(t, err)
	assert.Len(t, userKeys, 1)

	// Delete
	err = s.DeleteAPIKey(ctx(), keyID)
	require.NoError(t, err)

	// Verify deleted
	_, err = s.GetAPIKey(ctx(), keyID)
	assert.ErrorIs(t, err, apikey.ErrNotFound)
}

func TestKeymithStore_GetNotFound(t *testing.T) {
	s := testKeymithStore(t)

	_, err := s.GetAPIKey(ctx(), id.NewAPIKeyID())
	assert.ErrorIs(t, err, apikey.ErrNotFound)
}

func TestKeymithStore_RevokeViaUpdate(t *testing.T) {
	s := testKeymithStore(t)

	appID := id.NewAppID()
	userID := id.NewUserID()
	keyID := id.NewAPIKeyID()

	now := time.Now().Truncate(time.Millisecond)
	ak := &apikey.APIKey{
		ID:        keyID,
		AppID:     appID,
		UserID:    userID,
		Name:      "revoke-test",
		KeyHash:   "hashrevoke",
		KeyPrefix: "ask_revoke12",
		CreatedAt: now,
		UpdatedAt: now,
	}

	require.NoError(t, s.CreateAPIKey(ctx(), ak))

	// Verify initially not revoked
	got, err := s.GetAPIKey(ctx(), keyID)
	require.NoError(t, err)
	assert.False(t, got.Revoked)

	// Update with Revoked=true
	ak.Revoked = true
	ak.UpdatedAt = time.Now().Truncate(time.Millisecond)
	require.NoError(t, s.UpdateAPIKey(ctx(), ak))

	// Verify revoked state
	got, err = s.GetAPIKey(ctx(), keyID)
	require.NoError(t, err)
	assert.True(t, got.Revoked)
}

func TestKeymithStore_ListFiltering(t *testing.T) {
	s := testKeymithStore(t)

	app1 := id.NewAppID()
	app2 := id.NewAppID()
	user1 := id.NewUserID()
	user2 := id.NewUserID()

	now := time.Now().Truncate(time.Millisecond)

	// Create keys across different apps and users
	keys := []*apikey.APIKey{
		{ID: id.NewAPIKeyID(), AppID: app1, UserID: user1, Name: "a1-u1-1", KeyHash: "h1", KeyPrefix: "ask_a1u1_001", CreatedAt: now, UpdatedAt: now},
		{ID: id.NewAPIKeyID(), AppID: app1, UserID: user1, Name: "a1-u1-2", KeyHash: "h2", KeyPrefix: "ask_a1u1_002", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
		{ID: id.NewAPIKeyID(), AppID: app1, UserID: user2, Name: "a1-u2-1", KeyHash: "h3", KeyPrefix: "ask_a1u2_001", CreatedAt: now.Add(2 * time.Second), UpdatedAt: now.Add(2 * time.Second)},
		{ID: id.NewAPIKeyID(), AppID: app2, UserID: user1, Name: "a2-u1-1", KeyHash: "h4", KeyPrefix: "ask_a2u1_001", CreatedAt: now.Add(3 * time.Second), UpdatedAt: now.Add(3 * time.Second)},
	}
	for _, ak := range keys {
		require.NoError(t, s.CreateAPIKey(ctx(), ak))
	}

	// List by app1 — should see 3 keys
	app1Keys, err := s.ListAPIKeysByApp(ctx(), app1)
	require.NoError(t, err)
	assert.Len(t, app1Keys, 3)

	// List by app2 — should see 1 key
	app2Keys, err := s.ListAPIKeysByApp(ctx(), app2)
	require.NoError(t, err)
	assert.Len(t, app2Keys, 1)

	// List by app1, user1 — should see 2 keys
	app1User1Keys, err := s.ListAPIKeysByUser(ctx(), app1, user1)
	require.NoError(t, err)
	assert.Len(t, app1User1Keys, 2)

	// List by app1, user2 — should see 1 key
	app1User2Keys, err := s.ListAPIKeysByUser(ctx(), app1, user2)
	require.NoError(t, err)
	assert.Len(t, app1User2Keys, 1)

	// List by app2, user1 — should see 1 key
	app2User1Keys, err := s.ListAPIKeysByUser(ctx(), app2, user1)
	require.NoError(t, err)
	assert.Len(t, app2User1Keys, 1)

	// List by app2, user2 — should see 0 keys
	app2User2Keys, err := s.ListAPIKeysByUser(ctx(), app2, user2)
	require.NoError(t, err)
	assert.Len(t, app2User2Keys, 0)
}
