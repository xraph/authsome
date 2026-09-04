package retention

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
)

// These drive newEnabledPolicy's closure through a real *settings.Manager,
// the same split consent_policy_test.go uses: the worker tests pin what the
// worker does with the answer, these pin what the answer is.

func TestEnabledPolicyDefaultsToOnWhenNothingStored(t *testing.T) {
	mgr := newTestSettingsManager(t, newFakeSettingsStore())
	policy := newEnabledPolicy(mgr, log.NewNoopLogger())

	assert.True(t, policy(context.Background(), id.NewAppID()),
		"the registered default for retention.enabled is true")
}

func TestEnabledPolicyPerAppOverride(t *testing.T) {
	store := newFakeSettingsStore()
	appA, appB := id.NewAppID(), id.NewAppID()
	store.set(t, SettingEnabled.Def.Key, appA.String(), false)

	mgr := newTestSettingsManager(t, store)
	policy := newEnabledPolicy(mgr, log.NewNoopLogger())

	// One process, two apps: an operator turning the switch off during an
	// incident in one app must not stop delivery for every other app.
	assert.False(t, policy(context.Background(), appA), "app A has an explicit override to false")
	assert.True(t, policy(context.Background(), appB), "app B keeps the default")
}

func TestEnabledPolicyStaysOnWhenTheSettingIsUnreadable(t *testing.T) {
	store := newFakeSettingsStore()
	store.err = assert.AnError

	mgr := newTestSettingsManager(t, store)
	policy := newEnabledPolicy(mgr, log.NewNoopLogger())

	assert.True(t, policy(context.Background(), id.NewAppID()),
		"the opposite of the consent gate, deliberately: consent gates PII leaving "+
			"and has to block when uncertain, while this gates a feature and must not "+
			"silently stop a working integration")
}
