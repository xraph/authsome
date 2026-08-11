package authsome

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/account"
)

// A per-auth-method TTL exists so a magic-link or SSO session can be shorter
// than one from an interactive login. It must only ever shorten: the app-level
// config is the operator's ceiling, and a plugin setting must not become a way
// around it.
func TestApplySessionTTLOverride(t *testing.T) {
	const (
		appToken   = time.Hour
		appRefresh = 30 * 24 * time.Hour
	)

	tests := []struct {
		name        string
		sessionTTL  time.Duration
		refreshTTL  time.Duration
		wantToken   time.Duration
		wantRefresh time.Duration
	}{
		{"zero leaves app config untouched", 0, 0, appToken, appRefresh},
		{"shorter session TTL applies", 15 * time.Minute, 0, 15 * time.Minute, appRefresh},
		{"shorter refresh TTL applies", 0, 24 * time.Hour, appToken, 24 * time.Hour},
		{"both shorter apply", time.Minute, time.Hour, time.Minute, time.Hour},
		{"longer session TTL is ignored", 10 * time.Hour, 0, appToken, appRefresh},
		{"longer refresh TTL is ignored", 0, 365 * 24 * time.Hour, appToken, appRefresh},
		{"equal is a no-op", appToken, appRefresh, appToken, appRefresh},
		{"negative is ignored", -time.Hour, -time.Hour, appToken, appRefresh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := account.SessionConfig{TokenTTL: appToken, RefreshTokenTTL: appRefresh}
			applySessionTTLOverride(&cfg, tt.sessionTTL, tt.refreshTTL)
			assert.Equal(t, tt.wantToken, cfg.TokenTTL)
			assert.Equal(t, tt.wantRefresh, cfg.RefreshTokenTTL)
		})
	}
}

// With no app-level ceiling configured, a requested TTL is adopted rather than
// discarded — otherwise the plugin setting would be inert exactly where there
// is nothing else governing the lifetime.
func TestApplySessionTTLOverride_NoAppCeiling(t *testing.T) {
	cfg := account.SessionConfig{}
	applySessionTTLOverride(&cfg, 20*time.Minute, 12*time.Hour)
	assert.Equal(t, 20*time.Minute, cfg.TokenTTL)
	assert.Equal(t, 12*time.Hour, cfg.RefreshTokenTTL)
}
