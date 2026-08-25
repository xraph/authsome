package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/agentauth"
)

func TestPlugin_SatisfiesPluginInterface(t *testing.T) {
	var p plugin.Plugin = agentauth.New()
	assert.Equal(t, "agentauth", p.Name())
}

func TestPlugin_WithScope_RegistersMapping(t *testing.T) {
	p := agentauth.New(
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)

	perm, ok := p.Scopes().Lookup("invoices:read")

	require.True(t, ok)
	assert.Equal(t, "read", perm.Action)
	assert.Equal(t, "invoice", perm.Resource)
}

func TestPlugin_DefaultsToMemoryStore(t *testing.T) {
	p := agentauth.New()
	assert.NotNil(t, p.Store(), "a plugin with no store option must still be usable")
}

// Final review item 7: WithStore(nil) used to install a true nil Store
// interface value, since Store is an interface and the option had no guard.
// Every store call in the plugin — Authorize's grant read among them, the
// very first thing an agent request does — would then panic on a nil
// interface method call instead of erroring. A nil argument must leave the
// default memory store New() already installed untouched.
func TestPlugin_WithStoreNilIsANoOp(t *testing.T) {
	p := agentauth.New(agentauth.WithStore(nil))

	require.NotNil(t, p.Store(), "a nil store option must not clobber the default memory store")
	_, err := p.Store().GetAgent(context.Background(), id.NewAgentID())
	assert.ErrorIs(t, err, agentauth.ErrNotFound, "the default store must still be a working store, not a nil one")
}

func TestPlugin_DefaultGrantTTL(t *testing.T) {
	assert.Equal(t, 90*24*time.Hour, agentauth.New().DefaultGrantTTL())

	p := agentauth.New(agentauth.WithDefaultGrantTTL(7 * 24 * time.Hour))
	assert.Equal(t, 7*24*time.Hour, p.DefaultGrantTTL())
}
