package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
)

func TestPlugin_Name(t *testing.T) {
	assert.Equal(t, "sharedsignals", New().Name())
}

func TestPlugin_ConfigDefaults(t *testing.T) {
	p := New()
	assert.Equal(t, 24*time.Hour, p.config.SignalTTL)
	assert.Equal(t, 100, p.config.MaxActionsPerHour)
	assert.Equal(t, 5*time.Minute, p.config.ClockSkew)
	assert.Equal(t, 24*time.Hour, p.config.MaxSETAge)
	assert.Equal(t, int64(64*1024), p.config.MaxBodyBytes)
}

func TestPlugin_ConfigOverridesSurvive(t *testing.T) {
	p := New(Config{Audience: "https://a/ssf", SignalTTL: time.Hour, MaxBodyBytes: 1234})
	assert.Equal(t, "https://a/ssf", p.config.Audience)
	assert.Equal(t, time.Hour, p.config.SignalTTL)
	assert.Equal(t, int64(1234), p.config.MaxBodyBytes)
	// Unset fields still get defaults.
	assert.Equal(t, 100, p.config.MaxActionsPerHour)
}

func TestPlugin_MigrationGroups(t *testing.T) {
	p := New()
	assert.Len(t, p.MigrationGroups("pg"), 1)
	assert.Len(t, p.MigrationGroups("sqlite"), 1)
	assert.Len(t, p.MigrationGroups("mongo"), 1)
	assert.Empty(t, p.MigrationGroups("memory"))
}

func TestPlugin_DeclareSettings(t *testing.T) {
	m := settings.NewManager(nil, log.NewNoopLogger())
	require.NoError(t, New().DeclareSettings(m))
}

// Without a database the plugin must still come up on the memory store rather
// than leaving a nil store for the receiver to panic on.
func TestPlugin_OnInitFallsBackToMemory(t *testing.T) {
	p := New()
	require.NoError(t, p.OnInit(context.Background(), stubEngine{}))
	require.NotNil(t, p.store)
	_, err := p.store.GetInboundStreamByPushPathHash(context.Background(), "x")
	require.ErrorIs(t, err, ErrNotFound)
	require.NotNil(t, p.jwks)
}

// Compile-time proof that the plugin implements what the engine looks for.
func TestPlugin_ImplementsHooks(t *testing.T) {
	var p any = New()
	_, ok := p.(plugin.Plugin)
	assert.True(t, ok)
	_, ok = p.(plugin.OnInit)
	assert.True(t, ok)
	_, ok = p.(plugin.MigrationProvider)
	assert.True(t, ok)
	_, ok = p.(plugin.SettingsProvider)
	assert.True(t, ok)
	_, ok = p.(plugin.RouteProvider)
	assert.True(t, ok)
}
