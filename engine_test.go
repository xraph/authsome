package authsome_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/keysmith"
	ksmemory "github.com/xraph/keysmith/store/memory"
)

func newTestEngine(t *testing.T, opts ...authsome.Option) (*authsome.Engine, *memory.Store) {
	t.Helper()
	s := memory.New()
	allOpts := append([]authsome.Option{authsome.WithStore(s), authsome.WithDisableMigrate()}, opts...)
	eng, err := authsome.NewEngine(allOpts...)
	require.NoError(t, err)
	return eng, s
}

func TestNewEngine_RequiresStore(t *testing.T) {
	_, err := authsome.NewEngine()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store is required")
}

func TestNewEngine_DefaultConfig(t *testing.T) {
	eng, _ := newTestEngine(t)

	cfg := eng.Config()
	assert.Equal(t, "/v1/auth", cfg.BasePath)
	assert.Equal(t, 8, cfg.Password.MinLength)
	assert.True(t, cfg.Password.RequireUppercase)
	assert.True(t, cfg.Password.RequireLowercase)
	assert.True(t, cfg.Password.RequireDigit)
	assert.False(t, cfg.Password.RequireSpecial) // Default is false
}

func TestNewEngine_WithOptions(t *testing.T) {
	eng, _ := newTestEngine(t,
		authsome.WithAppID("test-app"),
		authsome.WithBasePath("/custom/auth"),
		authsome.WithDebug(true),
	)

	assert.Equal(t, "test-app", eng.Config().AppID)
	assert.Equal(t, "/custom/auth", eng.Config().BasePath)
	assert.True(t, eng.Config().Debug)
}

func TestEngine_StartStop(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()

	// Start should succeed
	err := eng.Start(ctx)
	require.NoError(t, err)

	// Start again should be idempotent
	err = eng.Start(ctx)
	require.NoError(t, err)

	// Stop should succeed
	err = eng.Stop(ctx)
	require.NoError(t, err)

	// Stop again should be idempotent
	err = eng.Stop(ctx)
	require.NoError(t, err)
}

func TestEngine_Accessors(t *testing.T) {
	eng, s := newTestEngine(t)

	assert.NotNil(t, eng.Store())
	assert.Equal(t, s, eng.Store())
	assert.NotNil(t, eng.Plugins())
	assert.NotNil(t, eng.Hooks())
	assert.NotNil(t, eng.Strategies())
	assert.NotNil(t, eng.Logger())

	// Optional bridges should be nil by default
	assert.Nil(t, eng.Chronicle())
	assert.Nil(t, eng.Authorizer())
	assert.Nil(t, eng.KeyManager())
	assert.Nil(t, eng.Relay())

	// First-class engines should be nil by default
	assert.Nil(t, eng.Keysmith())
}

func TestEngine_WithBridges(t *testing.T) {
	eng, _ := newTestEngine(t,
		authsome.WithChronicle(nil), // nil is acceptable
		authsome.WithAuthorizer(nil),
		authsome.WithKeyManager(nil),
		authsome.WithEventRelay(nil),
	)

	// They're nil because we passed nil
	assert.Nil(t, eng.Chronicle())
	assert.Nil(t, eng.Authorizer())
	assert.Nil(t, eng.KeyManager())
	assert.Nil(t, eng.Relay())
}

func TestEngine_Metrics(t *testing.T) {
	eng, _ := newTestEngine(t)

	m := eng.Metrics()
	assert.Equal(t, 0, m.PluginsLoaded)
	assert.Equal(t, 0, m.Strategies)
}

func TestEngine_Keysmith_NilByDefault(t *testing.T) {
	eng, _ := newTestEngine(t)
	assert.Nil(t, eng.Keysmith())
}

func TestEngine_WithKeysmith(t *testing.T) {
	ksStore := ksmemory.New()
	ksEng, err := keysmith.NewEngine(keysmith.WithStore(ksStore))
	require.NoError(t, err)

	eng, _ := newTestEngine(t, authsome.WithKeysmith(ksEng))

	// First-class engine should be set
	assert.NotNil(t, eng.Keysmith())
	assert.Equal(t, ksEng, eng.Keysmith())

	// Bridge should also be set for backward compat
	assert.NotNil(t, eng.KeyManager())
}

func TestEngine_APIKeyStore_WithKeysmith(t *testing.T) {
	ksStore := ksmemory.New()
	ksEng, err := keysmith.NewEngine(keysmith.WithStore(ksStore))
	require.NoError(t, err)

	eng, _ := newTestEngine(t, authsome.WithKeysmith(ksEng))

	// APIKeyStore should return a KeysmithStore
	store := eng.APIKeyStore()
	assert.NotNil(t, store)
	_, ok := store.(*apikey.KeysmithStore)
	assert.True(t, ok, "APIKeyStore should return *apikey.KeysmithStore when Keysmith is present")
}

func TestEngine_APIKeyStore_FallbackToComposite(t *testing.T) {
	eng, s := newTestEngine(t)

	// APIKeyStore should return the composite store (which implements apikey.Store)
	store := eng.APIKeyStore()
	assert.NotNil(t, store)
	assert.Equal(t, s, store, "APIKeyStore should return composite store when Keysmith is absent")
}
