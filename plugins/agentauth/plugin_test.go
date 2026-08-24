package agentauth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestPlugin_DefaultGrantTTL(t *testing.T) {
	assert.Equal(t, 90*24*time.Hour, agentauth.New().DefaultGrantTTL())

	p := agentauth.New(agentauth.WithDefaultGrantTTL(7 * 24 * time.Hour))
	assert.Equal(t, 7*24*time.Hour, p.DefaultGrantTTL())
}
