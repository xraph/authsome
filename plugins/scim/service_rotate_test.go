package scim

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func newRotationService(t *testing.T) (*Service, *MemoryStore, *SCIMConfig) {
	t.Helper()
	store := NewMemoryStore()
	svc := &Service{store: store}

	cfg := &SCIMConfig{
		ID:        id.NewSCIMConfigID(),
		AppID:     id.NewAppID(),
		Name:      "test-config",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateConfig(context.Background(), cfg))
	return svc, store, cfg
}

func TestRotateToken_OldKeptValidDuringGrace(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newRotationService(t)
	ctx := context.Background()

	oldPlain, oldTok, err := svc.GenerateToken(ctx, cfg.ID, "old", nil)
	require.NoError(t, err)
	require.NotEmpty(t, oldPlain)
	require.Nil(t, oldTok.ExpiresAt, "fresh non-expiring token should have nil ExpiresAt")

	newPlain, newTok, err := svc.RotateToken(ctx, oldTok.ID, "rotated", time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, newPlain)
	require.NotEqual(t, oldPlain, newPlain, "rotation must mint a fresh secret")
	require.Equal(t, cfg.ID, newTok.ConfigID)
	require.Nil(t, newTok.ExpiresAt, "the freshly minted token has no expiry")

	// Reload old token: ExpiresAt should now be set to roughly now+grace.
	reloaded, err := svc.store.GetToken(ctx, oldTok.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.ExpiresAt, "rotation must schedule old token for expiry")
	require.WithinDuration(t, time.Now().Add(time.Hour), *reloaded.ExpiresAt, 5*time.Second)
	require.False(t, reloaded.IsExpired(), "old token must remain valid through the grace window")
}

func TestRotateToken_OldExpiresAfterGrace(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newRotationService(t)
	ctx := context.Background()

	_, oldTok, err := svc.GenerateToken(ctx, cfg.ID, "old", nil)
	require.NoError(t, err)

	// Grace below the floor; service clamps to 1 minute. We then
	// simulate post-grace by rewriting ExpiresAt into the past.
	_, _, err = svc.RotateToken(ctx, oldTok.ID, "rotated", time.Nanosecond)
	require.NoError(t, err)

	reloaded, err := svc.store.GetToken(ctx, oldTok.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.ExpiresAt)
	require.True(t, reloaded.ExpiresAt.After(time.Now()),
		"grace must be clamped to >= 1 minute, so old token can't already be expired")

	past := time.Now().Add(-time.Minute)
	reloaded.ExpiresAt = &past
	require.NoError(t, svc.store.UpdateToken(ctx, reloaded))

	post, err := svc.store.GetToken(ctx, oldTok.ID)
	require.NoError(t, err)
	require.True(t, post.IsExpired(), "after grace window, old token must report expired")
}

func TestRotateToken_DoesNotExtendExistingTighterExpiry(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newRotationService(t)
	ctx := context.Background()

	tight := time.Now().Add(5 * time.Minute)
	_, oldTok, err := svc.GenerateToken(ctx, cfg.ID, "old", &tight)
	require.NoError(t, err)

	// Request a generous grace; the existing tighter expiry must win.
	_, _, err = svc.RotateToken(ctx, oldTok.ID, "rotated", 24*time.Hour)
	require.NoError(t, err)

	reloaded, err := svc.store.GetToken(ctx, oldTok.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.ExpiresAt)
	require.WithinDuration(t, tight, *reloaded.ExpiresAt, time.Second,
		"rotation must never extend a token that was already on a tighter expiry")
}

func TestRotateToken_ClampsExtremeGrace(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newRotationService(t)
	ctx := context.Background()

	_, oldTok, err := svc.GenerateToken(ctx, cfg.ID, "old", nil)
	require.NoError(t, err)

	// 100 days requested; clamped to 30.
	_, _, err = svc.RotateToken(ctx, oldTok.ID, "rotated", 100*24*time.Hour)
	require.NoError(t, err)

	reloaded, err := svc.store.GetToken(ctx, oldTok.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.ExpiresAt)
	upperBound := time.Now().Add(30 * 24 * time.Hour).Add(time.Minute)
	require.True(t, reloaded.ExpiresAt.Before(upperBound),
		"grace must clamp at 30 days; got %v", reloaded.ExpiresAt)
}

func TestRotateToken_DefaultsName(t *testing.T) {
	t.Parallel()
	svc, _, cfg := newRotationService(t)
	ctx := context.Background()

	_, oldTok, err := svc.GenerateToken(ctx, cfg.ID, "ci-runner", nil)
	require.NoError(t, err)

	_, newTok, err := svc.RotateToken(ctx, oldTok.ID, "", time.Hour)
	require.NoError(t, err)
	require.Equal(t, "ci-runner (rotated)", newTok.Name,
		"empty name must default to '<old> (rotated)' so operators can audit lineage")
}

func TestRotateToken_UnknownTokenErrors(t *testing.T) {
	t.Parallel()
	svc, _, _ := newRotationService(t)
	ctx := context.Background()

	_, _, err := svc.RotateToken(ctx, id.NewSCIMTokenID(), "x", time.Hour)
	require.Error(t, err, "rotating an unknown token must surface a clear error")
}
