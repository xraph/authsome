package retention

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugin"
)

func TestPluginName(t *testing.T) {
	assert.Equal(t, "retention", New().Name())
}

func TestPluginImplementsExpectedInterfaces(t *testing.T) {
	p := New()
	assert.Implements(t, (*plugin.Plugin)(nil), p)
	assert.Implements(t, (*plugin.OnInit)(nil), p)
	assert.Implements(t, (*plugin.OnShutdown)(nil), p)
	assert.Implements(t, (*plugin.MigrationProvider)(nil), p)
	assert.Implements(t, (*plugin.SettingsProvider)(nil), p)
	// Hook interfaces are asserted in hooks_test.go (Task 7), where the
	// methods exist.
}

func TestPluginFallsBackToMemoryStore(t *testing.T) {
	p := New()
	require.NoError(t, p.OnInit(context.Background(), &stubEngine{}))
	assert.NotNil(t, p.store)
	require.NoError(t, p.OnShutdown(context.Background()))
}

func TestPluginShutdownIsSafeWithoutInit(t *testing.T) {
	assert.NoError(t, New().OnShutdown(context.Background()),
		"OnShutdown may run on a plugin whose OnInit never completed")
}

func TestMigrationGroupsPerDriver(t *testing.T) {
	p := New()
	assert.Len(t, p.MigrationGroups("pg"), 1)
	assert.Len(t, p.MigrationGroups("sqlite"), 1)
	assert.Len(t, p.MigrationGroups("mongo"), 1)
	assert.Empty(t, p.MigrationGroups("cassandra"))
}
