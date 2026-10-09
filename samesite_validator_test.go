package authsome_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/settings"
)

// SameSite=None is only accepted while the CSRF check stands in front of
// cookie sessions.
func TestCookieSameSiteNoneNeedsCSRF(t *testing.T) {
	ctx := context.Background()
	set := func(eng *authsome.Engine) error {
		return eng.Settings().Set(ctx, "session.cookie_same_site", json.RawMessage(`"none"`), settings.ScopeGlobal, "", "", "", "test")
	}

	on, _ := newTestEngine(t)
	require.NoError(t, set(on), "the check is on by default, so none is allowed")

	off := false
	cfg := testEngineConfig()
	cfg.CSRF.Enabled = &off
	offEng, _ := newTestEngine(t, authsome.WithConfig(cfg))
	err := set(offEng)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "csrf.enabled")
	require.NoError(t, offEng.Settings().Set(ctx, "session.cookie_same_site", json.RawMessage(`"lax"`), settings.ScopeGlobal, "", "", "", "test"))
}
