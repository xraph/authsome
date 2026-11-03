package oauth2provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWKS_KidMatchesActive(t *testing.T) {
	ctx := context.Background()
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	m := newTestKeyManager(t, p.config, p.oauth2Store, testEncryptor(t))
	p.keys = m

	active, err := m.Active(ctx, "")
	require.NoError(t, err)

	got, degraded := p.jwksFor(ctx, "")
	require.False(t, degraded)
	require.Len(t, got.Keys, 1)
	assert.Equal(t, active.kid, got.Keys[0].KID)
}
