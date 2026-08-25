package scim

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// newValidationService builds a service backed by a memory store holding a
// config with a real, non-zero AppID — the shape every config created through
// the dashboard has.
func newValidationService(t *testing.T) (*Service, *MemoryStore, *SCIMConfig) {
	t.Helper()
	store := NewMemoryStore()
	svc := &Service{store: store}

	cfg := &SCIMConfig{
		ID:        id.NewSCIMConfigID(),
		AppID:     id.NewAppID(),
		Name:      "okta-prod",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateConfig(context.Background(), cfg))
	return svc, store, cfg
}

// This is the regression test for the bug where ValidateToken resolved configs
// via ListConfigs(ctx, "") — an empty appID filter that only ever matched a
// config whose AppID was the zero value. Every real config carries a genuine
// AppID, so every genuine SCIM bearer token was rejected.
func TestValidateToken_AcceptsTokenOnConfigWithRealAppID(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newValidationService(t)
	ctx := context.Background()

	require.False(t, cfg.AppID.IsNil(), "guard: the config under test must carry a real AppID")

	plaintext, issued, err := svc.GenerateToken(ctx, cfg.ID, "provisioning", nil)
	require.NoError(t, err)

	tok, gotCfg, err := svc.ValidateToken(ctx, plaintext)
	require.NoError(t, err, "a token issued against a config with a real AppID must validate")
	require.Equal(t, issued.ID, tok.ID)
	require.Equal(t, cfg.ID, gotCfg.ID)
	require.Equal(t, cfg.AppID, gotCfg.AppID, "the resolved config must carry the app scope handlers rely on")
}

func TestValidateToken_ResolvesCorrectConfigAcrossApps(t *testing.T) {
	t.Parallel()
	svc, store, cfgA := newValidationService(t)
	ctx := context.Background()

	cfgB := &SCIMConfig{
		ID:        id.NewSCIMConfigID(),
		AppID:     id.NewAppID(),
		Name:      "entra-prod",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateConfig(ctx, cfgB))

	_, _, err := svc.GenerateToken(ctx, cfgA.ID, "a", nil)
	require.NoError(t, err)
	plainB, _, err := svc.GenerateToken(ctx, cfgB.ID, "b", nil)
	require.NoError(t, err)

	_, gotCfg, err := svc.ValidateToken(ctx, plainB)
	require.NoError(t, err)
	require.Equal(t, cfgB.ID, gotCfg.ID, "a token must resolve to its own config, not a sibling app's")
}

func TestValidateToken_RejectsUnknownToken(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newValidationService(t)
	ctx := context.Background()

	_, _, err := svc.GenerateToken(ctx, cfg.ID, "provisioning", nil)
	require.NoError(t, err)

	_, _, err = svc.ValidateToken(ctx, "scim_deadbeef")
	require.EqualError(t, err, "scim: invalid token")
}

func TestValidateToken_RejectsExpiredToken(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newValidationService(t)
	ctx := context.Background()

	past := time.Now().Add(-time.Minute)
	plaintext, _, err := svc.GenerateToken(ctx, cfg.ID, "stale", &past)
	require.NoError(t, err)

	_, _, err = svc.ValidateToken(ctx, plaintext)
	require.EqualError(t, err, "scim: token expired",
		"an expired token must be distinguishable from an unknown one")
}

func TestValidateToken_RevokedTokenNoLongerValidates(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newValidationService(t)
	ctx := context.Background()

	plaintext, issued, err := svc.GenerateToken(ctx, cfg.ID, "provisioning", nil)
	require.NoError(t, err)
	require.NoError(t, svc.RevokeToken(ctx, issued.ID))

	_, _, err = svc.ValidateToken(ctx, plaintext)
	require.EqualError(t, err, "scim: invalid token")
}

func TestValidateToken_PersistsLastUsedAt(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newValidationService(t)
	ctx := context.Background()

	plaintext, issued, err := svc.GenerateToken(ctx, cfg.ID, "provisioning", nil)
	require.NoError(t, err)
	require.Nil(t, issued.LastUsedAt, "a freshly issued token has never been used")

	_, _, err = svc.ValidateToken(ctx, plaintext)
	require.NoError(t, err)

	reloaded, err := svc.store.GetToken(ctx, issued.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.LastUsedAt, "validation must record usage so the dashboard can show it")
	require.WithinDuration(t, time.Now(), *reloaded.LastUsedAt, 5*time.Second)
}
