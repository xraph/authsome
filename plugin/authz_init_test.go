package plugin_test

import (
	"context"
	"errors"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugin"
)

// failingPlugin fails during OnInit, standing in for a plugin that cannot
// resolve its store, keys, or bridges.
type failingPlugin struct{ name string }

func (p *failingPlugin) Name() string { return p.name }
func (p *failingPlugin) OnInit(context.Context, plugin.Engine) error {
	return errors.New("could not resolve store")
}

type okPlugin struct {
	name   string
	inited bool
}

func (p *okPlugin) Name() string { return p.name }
func (p *okPlugin) OnInit(context.Context, plugin.Engine) error {
	p.inited = true
	return nil
}

// A plugin that fails OnInit is absent, not degraded. Swallowing the error left
// the engine serving traffic with that plugin's control silently missing — an
// MFA plugin that never wired its store enforces nothing.
func TestEmitOnInit_PropagatesFailure(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	r.Register(&failingPlugin{name: "broken"})

	err := r.EmitOnInit(context.Background(), nil)

	require.Error(t, err, "an OnInit failure must not be swallowed")
	assert.Contains(t, err.Error(), "broken", "the error must name the failing plugin")
	assert.Contains(t, err.Error(), "could not resolve store")
}

// Initialization stops at the first failure rather than continuing with a
// half-initialized set.
func TestEmitOnInit_StopsAtFirstFailure(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	later := &okPlugin{name: "later"}
	r.Register(&failingPlugin{name: "broken"})
	r.Register(later)

	require.Error(t, r.EmitOnInit(context.Background(), nil))
	assert.False(t, later.inited, "plugins after the failure must not be initialized")
}

func TestEmitOnInit_SucceedsWhenAllPluginsInit(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	a, b := &okPlugin{name: "a"}, &okPlugin{name: "b"}
	r.Register(a)
	r.Register(b)

	require.NoError(t, r.EmitOnInit(context.Background(), nil))
	assert.True(t, a.inited)
	assert.True(t, b.inited)
}
