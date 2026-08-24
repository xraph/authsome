package principal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/principal"
)

func TestEmptyChain(t *testing.T) {
	var c principal.Chain
	_, ok := c.Actor()
	assert.False(t, ok)
	_, ok = c.Root()
	assert.False(t, ok)
	assert.Equal(t, 0, c.Depth())
}

// Chains are ordered nearest-caller-first, so Actor is the immediate caller
// and Root is the outermost hop. A multi-hop chain is an ephemeral child
// acting through its registered parent.
func TestChainOrdering(t *testing.T) {
	child := principal.Ref{Kind: principal.KindAgent, ID: "svc_child"}
	parent := principal.Ref{Kind: principal.KindAgent, ID: "svc_parent"}
	c := principal.Chain{child, parent}

	got, ok := c.Actor()
	assert.True(t, ok)
	assert.Equal(t, child, got, "Actor must be the immediate caller")

	got, ok = c.Root()
	assert.True(t, ok)
	assert.Equal(t, parent, got, "Root must be the outermost hop")

	assert.Equal(t, 2, c.Depth())
	assert.True(t, c.Contains(parent))
	assert.False(t, c.Contains(principal.Ref{Kind: principal.KindUser, ID: "ausr_1"}))
}
