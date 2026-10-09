package authsome_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/lockout"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// Abuse controls are on by default and live in the store's shared KV
// table, so a deployment gets them without wiring anything and keeps them
// when it scales out.

func TestDefaultConfig_AbuseControlsOn(t *testing.T) {
	cfg := authsome.DefaultConfig()
	assert.True(t, cfg.RateLimit.Enabled)
	assert.True(t, cfg.Lockout.Enabled)
	assert.False(t, cfg.RateLimit.FailOpen, "a limiter outage fails closed")
	assert.Equal(t, 20, cfg.RateLimit.APIKeyFailureLimit)
	assert.Equal(t, 5, cfg.RateLimit.WaitlistJoinLimit)
}

func engineWith(t *testing.T, cfg authsome.Config) *authsome.Engine {
	t.Helper()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(memory.New()),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(cfg),
	)
	require.NoError(t, err)
	return eng
}

func TestNewEngine_WiresSharedStateByDefault(t *testing.T) {
	eng := engineWith(t, authsome.DefaultConfig())
	assert.IsType(t, &ratelimit.KVLimiter{}, eng.RateLimiter())
	assert.IsType(t, &lockout.KVTracker{}, eng.Lockout())
	assert.IsType(t, &ceremony.KVStore{}, eng.CeremonyStore())
}

func TestNewEngine_HonoursAnExplicitOff(t *testing.T) {
	cfg := authsome.DefaultConfig()
	cfg.RateLimit.Enabled = false
	cfg.Lockout.Enabled = false
	eng := engineWith(t, cfg)
	assert.Nil(t, eng.RateLimiter())
	assert.Nil(t, eng.Lockout())
	assert.Empty(t, eng.RateLimitOptions(5), "no options are built when limiting is off")
}

func TestNewEngine_KeepsACallerSuppliedLimiter(t *testing.T) {
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	custom := ratelimit.NewMemoryLimiter()
	eng, err := authsome.NewEngine(
		authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(memory.New()),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithRateLimiter(custom),
	)
	require.NoError(t, err)
	assert.Same(t, custom, eng.RateLimiter())
}
