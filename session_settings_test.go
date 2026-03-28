package authsome

import (
	"encoding/json"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/settings"
)

func TestRegisterCoreSessionSettings_AllRegistered(t *testing.T) {
	mgr := settings.NewManager(nil, log.NewNoopLogger())
	err := registerCoreSessionSettings(mgr)
	require.NoError(t, err)

	// Verify all settings are registered by checking key settings from each category.
	defs := mgr.Definitions()
	keys := make(map[string]bool, len(defs))
	for _, d := range defs {
		keys[d.Key] = true
	}

	// Existing settings
	assert.True(t, keys["session.token_ttl_seconds"], "token TTL should be registered")
	assert.True(t, keys["session.refresh_token_ttl_seconds"], "refresh token TTL should be registered")
	assert.True(t, keys["session.rotate_refresh_token"], "rotate refresh token should be registered")
	assert.True(t, keys["session.bind_to_ip"], "bind to IP should be registered")
	assert.True(t, keys["session.bind_to_device"], "bind to device should be registered")
	assert.True(t, keys["session.cookie_name"], "cookie name should be registered")
	assert.True(t, keys["session.cookie_same_site"], "cookie same site should be registered")
	assert.True(t, keys["session.auto_refresh_enabled"], "auto-refresh enabled should be registered")

	// New settings from security fixes
	assert.True(t, keys["session.auto_refresh_expose_refresh_token"], "auto-refresh expose refresh token should be registered")
	assert.True(t, keys["session.jwt_require_active_session"], "JWT require active session should be registered")
}

func TestSettingJWTRequireActiveSession_DefaultFalse(t *testing.T) {
	// The default value should be false (opt-in feature).
	var val bool
	err := json.Unmarshal(SettingJWTRequireActiveSession.Def.Default, &val)
	require.NoError(t, err)
	assert.False(t, val, "JWT require active session should default to false")
}

func TestSettingAutoRefreshExposeRefreshToken_DefaultFalse(t *testing.T) {
	// The default value should be false (secure by default).
	var val bool
	err := json.Unmarshal(SettingAutoRefreshExposeRefreshToken.Def.Default, &val)
	require.NoError(t, err)
	assert.False(t, val, "auto-refresh expose refresh token should default to false")
}

func TestDefaultConfig_RefreshLimit(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 10, cfg.RateLimit.RefreshLimit, "refresh rate limit should default to 10")
}

func TestDefaultConfig_IntrospectLimit(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 20, cfg.RateLimit.IntrospectLimit, "introspect rate limit should default to 20")
}
