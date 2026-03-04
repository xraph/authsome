package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────
// DefaultSettingsForType tests
// ──────────────────────────────────────────────────

func TestDefaultSettingsForType_Development(t *testing.T) {
	s := DefaultSettingsForType(TypeDevelopment)
	require.NotNil(t, s)
	assert.True(t, *s.SkipEmailVerification)
	assert.True(t, *s.AllowTestCredentials)
	assert.Equal(t, 4, *s.PasswordMinLength)
	assert.False(t, *s.RateLimitEnabled)
	assert.False(t, *s.LockoutEnabled)
}

func TestDefaultSettingsForType_Staging(t *testing.T) {
	s := DefaultSettingsForType(TypeStaging)
	require.NotNil(t, s)
	assert.True(t, *s.RateLimitEnabled)
	assert.True(t, *s.LockoutEnabled)
	assert.Nil(t, s.SkipEmailVerification)
	assert.Nil(t, s.CheckBreached)
}

func TestDefaultSettingsForType_Production(t *testing.T) {
	s := DefaultSettingsForType(TypeProduction)
	require.NotNil(t, s)
	assert.True(t, *s.RateLimitEnabled)
	assert.True(t, *s.LockoutEnabled)
	assert.True(t, *s.CheckBreached)
}

func TestDefaultSettingsForType_Unknown(t *testing.T) {
	s := DefaultSettingsForType(Type("custom"))
	assert.Nil(t, s)
}

// ──────────────────────────────────────────────────
// MergeSettings tests
// ──────────────────────────────────────────────────

func TestMergeSettings_BothNil(t *testing.T) {
	result := MergeSettings(nil, nil)
	assert.Nil(t, result)
}

func TestMergeSettings_NilBase(t *testing.T) {
	override := &Settings{SkipEmailVerification: boolPtr(true)}
	result := MergeSettings(nil, override)
	require.NotNil(t, result)
	assert.True(t, *result.SkipEmailVerification)
}

func TestMergeSettings_NilOverride(t *testing.T) {
	base := &Settings{RateLimitEnabled: boolPtr(true)}
	result := MergeSettings(base, nil)
	require.NotNil(t, result)
	assert.True(t, *result.RateLimitEnabled)
}

func TestMergeSettings_OverrideWins(t *testing.T) {
	base := &Settings{RateLimitEnabled: boolPtr(true)}
	override := &Settings{RateLimitEnabled: boolPtr(false)}
	result := MergeSettings(base, override)
	require.NotNil(t, result)
	assert.False(t, *result.RateLimitEnabled)
}

func TestMergeSettings_PartialOverlay(t *testing.T) {
	base := &Settings{
		RateLimitEnabled: boolPtr(true),
		LockoutEnabled:   boolPtr(true),
	}
	override := &Settings{LockoutEnabled: boolPtr(false)}
	result := MergeSettings(base, override)
	require.NotNil(t, result)
	assert.True(t, *result.RateLimitEnabled, "base field should be retained")
	assert.False(t, *result.LockoutEnabled, "override should win")
}

func TestMergeSettings_OAuthOverridesMerge(t *testing.T) {
	base := &Settings{
		OAuthOverrides: map[string]OAuthProviderOverride{
			"google": {ClientID: "google-id"},
		},
	}
	override := &Settings{
		OAuthOverrides: map[string]OAuthProviderOverride{
			"github": {ClientID: "github-id"},
		},
	}
	result := MergeSettings(base, override)
	require.NotNil(t, result)
	require.Len(t, result.OAuthOverrides, 2)
	assert.Equal(t, "google-id", result.OAuthOverrides["google"].ClientID)
	assert.Equal(t, "github-id", result.OAuthOverrides["github"].ClientID)
}

func TestMergeSettings_WebhookURLOverride(t *testing.T) {
	base := &Settings{}
	override := &Settings{WebhookURLOverride: "https://dev.example.com/hook"}
	result := MergeSettings(base, override)
	require.NotNil(t, result)
	assert.Equal(t, "https://dev.example.com/hook", result.WebhookURLOverride)
}

// ──────────────────────────────────────────────────
// Settings helper method tests
// ──────────────────────────────────────────────────

func TestSettings_SkipEmailVerificationEnabled_Nil(t *testing.T) {
	var s *Settings
	assert.False(t, s.SkipEmailVerificationEnabled())
}

func TestSettings_SkipEmailVerificationEnabled_FieldNil(t *testing.T) {
	s := &Settings{}
	assert.False(t, s.SkipEmailVerificationEnabled())
}

func TestSettings_SkipEmailVerificationEnabled_True(t *testing.T) {
	s := &Settings{SkipEmailVerification: boolPtr(true)}
	assert.True(t, s.SkipEmailVerificationEnabled())
}

func TestSettings_SkipEmailVerificationEnabled_False(t *testing.T) {
	s := &Settings{SkipEmailVerification: boolPtr(false)}
	assert.False(t, s.SkipEmailVerificationEnabled())
}

func TestSettings_AllowTestCredentialsEnabled_Nil(t *testing.T) {
	var s *Settings
	assert.False(t, s.AllowTestCredentialsEnabled())
}

func TestSettings_AllowTestCredentialsEnabled_True(t *testing.T) {
	s := &Settings{AllowTestCredentials: boolPtr(true)}
	assert.True(t, s.AllowTestCredentialsEnabled())
}

func TestSettings_IsRateLimitEnabled_Nil(t *testing.T) {
	var s *Settings
	assert.False(t, s.IsRateLimitEnabled())
}

func TestSettings_IsRateLimitEnabled_True(t *testing.T) {
	s := &Settings{RateLimitEnabled: boolPtr(true)}
	assert.True(t, s.IsRateLimitEnabled())
}

func TestSettings_IsRateLimitEnabled_False(t *testing.T) {
	s := &Settings{RateLimitEnabled: boolPtr(false)}
	assert.False(t, s.IsRateLimitEnabled())
}

func TestSettings_IsLockoutEnabled_Nil(t *testing.T) {
	var s *Settings
	assert.False(t, s.IsLockoutEnabled())
}

func TestSettings_IsLockoutEnabled_True(t *testing.T) {
	s := &Settings{LockoutEnabled: boolPtr(true)}
	assert.True(t, s.IsLockoutEnabled())
}

func TestSettings_IsBreachCheckEnabled_Nil(t *testing.T) {
	var s *Settings
	assert.False(t, s.IsBreachCheckEnabled())
}

func TestSettings_IsBreachCheckEnabled_True(t *testing.T) {
	s := &Settings{CheckBreached: boolPtr(true)}
	assert.True(t, s.IsBreachCheckEnabled())
}
