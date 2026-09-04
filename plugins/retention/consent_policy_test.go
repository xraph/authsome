package retention

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// newTestSettingsManager registers the same definitions DeclareSettings
// registers in production, against a manager backed by store.
func newTestSettingsManager(t *testing.T, store settings.Store) *settings.Manager {
	t.Helper()
	m := settings.NewManager(store, log.NewNoopLogger())
	require.NoError(t, settings.RegisterTyped(m, "retention", SettingEnabled))
	require.NoError(t, settings.RegisterTyped(m, "retention", SettingRequireConsent))
	require.NoError(t, settings.RegisterTyped(m, "retention", SettingConsentPurpose))
	return m
}

// These tests drive newConsentPolicy's closure through a real
// *settings.Manager, unlike the allowSend tests in consent_test.go, which
// stub consentPolicy directly. That is the right split: allowSend's tests
// pin what allowSend does with whatever the policy returns, and these pin
// what the policy itself resolves against a real settings cascade —
// something stubEngine cannot exercise, because stubEngine.Settings()
// returns nil and OnInit never builds a real closure under it.

func TestConsentPolicyDefaultsToGateOffWhenNothingStored(t *testing.T) {
	mgr := newTestSettingsManager(t, newFakeSettingsStore())
	policy := newConsentPolicy(mgr, log.NewNoopLogger())

	gateOn, purpose := policy(context.Background(), id.NewAppID())
	assert.False(t, gateOn, "the registered default for require_consent is false")
	assert.Equal(t, "marketing", purpose, "the registered default for consent_purpose")
}

func TestConsentPolicyPerAppOverride(t *testing.T) {
	store := newFakeSettingsStore()
	appA, appB := id.NewAppID(), id.NewAppID()
	store.set(t, SettingRequireConsent.Def.Key, appA.String(), true)

	mgr := newTestSettingsManager(t, store)
	policy := newConsentPolicy(mgr, log.NewNoopLogger())

	gateOnA, _ := policy(context.Background(), appA)
	gateOnB, _ := policy(context.Background(), appB)

	// This is the property startup caching would have broken: one plugin
	// instance, two apps, two different answers from the same policy value.
	assert.True(t, gateOnA, "app A has an explicit override to true")
	assert.False(t, gateOnB, "app B has no override and falls back to the false default")
}

func TestConsentPolicyGatesOnWhenSettingUnreadable(t *testing.T) {
	store := newFakeSettingsStore()
	store.err = assert.AnError

	mgr := newTestSettingsManager(t, store)
	policy := newConsentPolicy(mgr, log.NewNoopLogger())

	gateOn, purpose := policy(context.Background(), id.NewAppID())
	assert.True(t, gateOn, "an unreadable setting must gate on, not fail open")
	assert.Equal(t, "marketing", purpose)
}

func TestConsentPolicyFallsBackToDefaultPurposeWhenStoredPurposeIsEmpty(t *testing.T) {
	store := newFakeSettingsStore()
	appID := id.NewAppID()
	store.set(t, SettingConsentPurpose.Def.Key, appID.String(), "")

	mgr := newTestSettingsManager(t, store)
	policy := newConsentPolicy(mgr, log.NewNoopLogger())

	_, purpose := policy(context.Background(), appID)
	assert.Equal(t, "marketing", purpose, "an empty stored purpose falls back to the default, not through as empty")
}
