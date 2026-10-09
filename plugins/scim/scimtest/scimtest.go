// Package scimtest is the store contract every SCIM store implementation
// must satisfy. Memory, sqlite, postgres and mongo run the same cases.
package scimtest

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/scim"
	"github.com/xraph/authsome/store"
)

// Fixture is one store under test with the tenant ids the cases use.
type Fixture struct {
	Store scim.Store
	AppID id.AppID
	OrgID id.OrgID
}

// Factory builds a fresh fixture for a case.
type Factory func(t *testing.T) Fixture

// RunConformance runs every case against fixtures from newFixture.
func RunConformance(t *testing.T, newFixture Factory) {
	t.Helper()
	cases := []struct {
		name string
		fn   func(t *testing.T, f Fixture)
	}{
		{"ConfigCRUD", testConfigCRUD},
		{"TokenLifecycle", testTokenLifecycle},
		{"LegacyTokenIsFoundAndUpgraded", testLegacyTokenIsFoundAndUpgraded},
		{"LogsListAndCount", testLogsListAndCount},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.fn(t, newFixture(t)) })
	}
}

var seq atomic.Int64

func unique(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), seq.Add(1))
}

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

func newConfig(f Fixture) *scim.SCIMConfig {
	return &scim.SCIMConfig{
		ID: id.NewSCIMConfigID(), AppID: f.AppID, OrgID: f.OrgID, Name: unique("okta"), Enabled: true,
		AutoCreate: true, AutoSuspend: true, DefaultRole: "member", Metadata: map[string]string{"region": "eu"},
		CreatedAt: now(), UpdatedAt: now(),
	}
}

func hashed(t *testing.T, plaintext string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func testConfigCRUD(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConfig(f)
	require.NoError(t, f.Store.CreateConfig(ctx, c))

	got, err := f.Store.GetConfig(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.Name, got.Name)
	assert.Equal(t, f.AppID.String(), got.AppID.String())
	assert.Equal(t, f.OrgID.String(), got.OrgID.String())
	assert.True(t, got.Enabled)
	assert.True(t, got.AutoCreate)
	assert.Equal(t, "member", got.DefaultRole)
	assert.Equal(t, map[string]string{"region": "eu"}, got.Metadata)

	got.Name = "okta-renamed"
	got.Enabled = false
	got.Metadata = map[string]string{"region": "us", "tier": "gold"}
	require.NoError(t, f.Store.UpdateConfig(ctx, got))
	again, err := f.Store.GetConfig(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "okta-renamed", again.Name)
	assert.False(t, again.Enabled)
	assert.Equal(t, map[string]string{"region": "us", "tier": "gold"}, again.Metadata)

	other := newConfig(f)
	other.AppID = id.NewAppID()
	other.OrgID = id.NewOrgID()
	require.NoError(t, f.Store.CreateConfig(ctx, other))
	byApp, err := f.Store.ListConfigs(ctx, f.AppID.String())
	require.NoError(t, err)
	var seen bool
	for _, x := range byApp {
		assert.Equal(t, f.AppID.String(), x.AppID.String(), "listing by app is scoped to the app")
		if x.ID == c.ID {
			seen = true
		}
	}
	assert.True(t, seen)
	byOrg, err := f.Store.ListConfigsByOrg(ctx, f.OrgID)
	require.NoError(t, err)
	for _, x := range byOrg {
		assert.Equal(t, f.OrgID.String(), x.OrgID.String())
	}

	require.NoError(t, f.Store.DeleteConfig(ctx, c.ID))
	_, err = f.Store.GetConfig(ctx, c.ID)
	assert.ErrorIs(t, err, scim.ErrConfigNotFound)
	assert.ErrorIs(t, f.Store.UpdateConfig(ctx, c), scim.ErrConfigNotFound)
	_, err = f.Store.GetConfig(ctx, id.NewSCIMConfigID())
	assert.ErrorIs(t, err, scim.ErrConfigNotFound)
}

func testTokenLifecycle(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConfig(f)
	require.NoError(t, f.Store.CreateConfig(ctx, c))

	plaintext := unique("scim_tok")
	exp := now().Add(time.Hour)
	tok := &scim.Token{
		ID: id.NewSCIMTokenID(), ConfigID: c.ID, Name: "provisioner", TokenHash: hashed(t, plaintext),
		TokenLookup: store.HashToken(plaintext), ExpiresAt: &exp, CreatedAt: now(),
	}
	require.NoError(t, f.Store.CreateToken(ctx, tok))

	got, err := f.Store.GetToken(ctx, tok.ID)
	require.NoError(t, err)
	assert.Equal(t, "provisioner", got.Name)
	assert.Equal(t, tok.TokenLookup, got.TokenLookup)
	require.NotNil(t, got.ExpiresAt)
	assert.WithinDuration(t, exp, *got.ExpiresAt, time.Second)
	assert.Nil(t, got.LastUsedAt)

	found, cfg, err := f.Store.FindTokenByPlaintext(ctx, plaintext)
	require.NoError(t, err)
	assert.Equal(t, tok.ID.String(), found.ID.String())
	assert.Equal(t, c.ID.String(), cfg.ID.String(), "the owning configuration rides along")
	_, _, err = f.Store.FindTokenByPlaintext(ctx, plaintext+"x")
	assert.ErrorIs(t, err, scim.ErrTokenNotFound)
	_, _, err = f.Store.FindTokenByPlaintext(ctx, store.HashToken(plaintext))
	assert.ErrorIs(t, err, scim.ErrTokenNotFound, "the lookup digest is not a credential")

	used := now()
	found.LastUsedAt = &used
	require.NoError(t, f.Store.UpdateToken(ctx, found))
	again, err := f.Store.GetToken(ctx, tok.ID)
	require.NoError(t, err)
	require.NotNil(t, again.LastUsedAt)
	assert.WithinDuration(t, used, *again.LastUsedAt, time.Second)

	listed, err := f.Store.ListTokens(ctx, c.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	require.NoError(t, f.Store.DeleteToken(ctx, tok.ID))
	_, err = f.Store.GetToken(ctx, tok.ID)
	assert.ErrorIs(t, err, scim.ErrTokenNotFound)
	_, _, err = f.Store.FindTokenByPlaintext(ctx, plaintext)
	assert.ErrorIs(t, err, scim.ErrTokenNotFound)
}

func testLegacyTokenIsFoundAndUpgraded(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConfig(f)
	require.NoError(t, f.Store.CreateConfig(ctx, c))
	plaintext := unique("scim_legacy")
	tok := &scim.Token{ID: id.NewSCIMTokenID(), ConfigID: c.ID, Name: "old", TokenHash: hashed(t, plaintext), CreatedAt: now()}
	require.NoError(t, f.Store.CreateToken(ctx, tok), "a token with no lookup digest is what earlier releases wrote")

	found, _, err := f.Store.FindTokenByPlaintext(ctx, plaintext)
	require.NoError(t, err, "a legacy token is still found by comparing")
	assert.Equal(t, tok.ID.String(), found.ID.String())
	assert.Equal(t, store.HashToken(plaintext), found.TokenLookup, "the match hands back the digest it now carries")

	stored, err := f.Store.GetToken(ctx, tok.ID)
	require.NoError(t, err)
	assert.Equal(t, store.HashToken(plaintext), stored.TokenLookup, "the row was upgraded on use")
}

func testLogsListAndCount(t *testing.T, f Fixture) {
	ctx := context.Background()
	c := newConfig(f)
	require.NoError(t, f.Store.CreateConfig(ctx, c))
	other := newConfig(f)
	other.AppID = id.NewAppID()
	require.NoError(t, f.Store.CreateConfig(ctx, other))

	base := now().Add(-time.Minute)
	for i, status := range []string{scim.LogStatusSuccess, scim.LogStatusSuccess, scim.LogStatusError, scim.LogStatusSkipped} {
		require.NoError(t, f.Store.CreateLog(ctx, &scim.ProvisionLog{
			ID: id.NewSCIMLogID(), ConfigID: c.ID, Action: scim.ActionCreateUser, ResourceType: "User",
			ExternalID: fmt.Sprintf("ext-%d", i), InternalID: id.NewUserID().String(), Status: status, Detail: "d",
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		}))
	}
	require.NoError(t, f.Store.CreateLog(ctx, &scim.ProvisionLog{
		ID: id.NewSCIMLogID(), ConfigID: other.ID, Action: scim.ActionCreateUser, ResourceType: "User",
		Status: scim.LogStatusError, CreatedAt: now(),
	}))

	logs, err := f.Store.ListLogs(ctx, c.ID, 0)
	require.NoError(t, err)
	require.Len(t, logs, 4)
	assert.Equal(t, "ext-3", logs[0].ExternalID, "newest first")
	limited, err := f.Store.ListLogs(ctx, c.ID, 2)
	require.NoError(t, err)
	assert.Len(t, limited, 2)

	all, err := f.Store.ListAllLogs(ctx, f.AppID.String(), 0)
	require.NoError(t, err)
	for _, l := range all {
		assert.NotEqual(t, other.ID.String(), l.ConfigID.String(), "another app's logs stay out")
	}

	success, errs, skipped, err := f.Store.CountLogsByStatus(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, []int{2, 1, 1}, []int{success, errs, skipped})
	success, errs, skipped, err = f.Store.CountAllLogsByStatus(ctx, other.AppID.String())
	require.NoError(t, err)
	assert.Equal(t, []int{0, 1, 0}, []int{success, errs, skipped})
}
