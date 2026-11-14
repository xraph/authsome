package mfa

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestNew_TrustedDeviceTTLDefaultsTo30Days(t *testing.T) {
	assert.Equal(t, 30*24*time.Hour, New().config.TrustedDeviceTTL)
	assert.Equal(t, time.Hour, New(Config{TrustedDeviceTTL: time.Hour}).config.TrustedDeviceTTL)
}

func TestVerifyTrustedDevice(t *testing.T) {
	ctx := context.Background()
	userID, other := id.NewUserID(), id.NewUserID()
	seed := func(t *testing.T, p *Plugin, owner id.UserID, expires time.Time) string {
		t.Helper()
		token, d, err := p.trustDevice(ctx, owner, "198.51.100.4", "curl/8.0")
		require.NoError(t, err)
		if !expires.IsZero() {
			require.NoError(t, p.store.DeleteTrustedDevice(ctx, owner, d.ID))
			d.ExpiresAt = expires
			require.NoError(t, p.store.CreateTrustedDevice(ctx, d))
		}
		return token
	}
	newPlugin := func(cfg Config) *Plugin {
		p := New(cfg)
		p.SetStore(NewMemoryStore())
		return p
	}

	t.Run("live", func(t *testing.T) {
		p := newPlugin(Config{})
		token := seed(t, p, userID, time.Time{})
		at, ok, err := p.VerifyTrustedDevice(ctx, id.Nil, userID, token)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.False(t, at.IsZero())
	})
	t.Run("other user", func(t *testing.T) {
		p := newPlugin(Config{})
		token := seed(t, p, other, time.Time{})
		_, ok, err := p.VerifyTrustedDevice(ctx, id.Nil, userID, token)
		require.NoError(t, err)
		assert.False(t, ok)
	})
	t.Run("expired", func(t *testing.T) {
		p := newPlugin(Config{})
		token := seed(t, p, userID, time.Now().Add(-time.Second))
		_, ok, err := p.VerifyTrustedDevice(ctx, id.Nil, userID, token)
		require.NoError(t, err)
		assert.False(t, ok)
	})
	t.Run("disabled", func(t *testing.T) {
		for _, cfg := range []Config{{DisableTrustedDevices: true}, {TrustedDeviceTTL: -1}} {
			p := newPlugin(Config{})
			token := seed(t, p, userID, time.Time{})
			p.config.DisableTrustedDevices = cfg.DisableTrustedDevices
			if cfg.TrustedDeviceTTL != 0 {
				p.config.TrustedDeviceTTL = cfg.TrustedDeviceTTL
			}
			_, ok, err := p.VerifyTrustedDevice(ctx, id.Nil, userID, token)
			require.NoError(t, err)
			assert.False(t, ok, "a turned-off feature honours no token (%+v)", cfg)
		}
	})
}

// The stored row never holds the token itself.
func TestTrustDevice_StoresOnlyTheHash(t *testing.T) {
	p := New()
	p.SetStore(NewMemoryStore())
	token, d, err := p.trustDevice(context.Background(), id.NewUserID(), "", "")
	require.NoError(t, err)
	assert.NotEqual(t, token, d.TokenHash)
	assert.Equal(t, hashTrustedDeviceToken(token), d.TokenHash)
	assert.Len(t, token, 43, "32 random bytes, base64url without padding")
}
