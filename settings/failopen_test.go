package settings_test

import (
	"context"
	"errors"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/settings"
)

// failingStore errors on every read, simulating a settings-store outage.
type failingStore struct{}

func (failingStore) GetSetting(context.Context, string, settings.Scope, string) (*settings.Setting, error) {
	return nil, errors.New("boom")
}
func (failingStore) SetSetting(context.Context, *settings.Setting) error { return errors.New("boom") }
func (failingStore) DeleteSetting(context.Context, string, settings.Scope, string) error {
	return errors.New("boom")
}
func (failingStore) ListSettings(context.Context, settings.ListOpts) ([]*settings.Setting, error) {
	return nil, errors.New("boom")
}
func (failingStore) ResolveSettings(context.Context, string, settings.ResolveOpts) ([]*settings.Setting, error) {
	return nil, errors.New("boom")
}
func (failingStore) BatchResolve(context.Context, []string, settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	return nil, errors.New("boom")
}
func (failingStore) DeleteSettingsByNamespace(context.Context, string) error {
	return errors.New("boom")
}

// TestGet_FailsSafeToDefaultOnStoreError pins that when the settings store
// errors, Get returns the registered default (not the zero value of T). This is
// the security-relevant behavior for flags like cookie Secure/HttpOnly whose
// default is true: a store outage must not silently flip them to false.
func TestGet_FailsSafeToDefaultOnStoreError(t *testing.T) {
	mgr := settings.NewManager(failingStore{}, log.NewNoopLogger())

	secureDef := settings.Define("test.cookie_secure", true)
	require.NoError(t, settings.RegisterTyped(mgr, "test", secureDef))

	val, err := settings.Get(context.Background(), mgr, secureDef, settings.ResolveOpts{})

	require.Error(t, err, "the store error should still be surfaced")
	assert.True(t, val, "on a store error Get must fall back to the registered default (true), not the zero value")
}

// A string default must also survive a store outage.
func TestGet_FailsSafeToStringDefault(t *testing.T) {
	mgr := settings.NewManager(failingStore{}, log.NewNoopLogger())

	def := settings.Define("test.some_string", "lax")
	require.NoError(t, settings.RegisterTyped(mgr, "test", def))

	val, err := settings.Get(context.Background(), mgr, def, settings.ResolveOpts{})

	require.Error(t, err)
	assert.Equal(t, "lax", val, "string default must be returned on store error")
}
