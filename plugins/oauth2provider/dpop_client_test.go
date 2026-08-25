package oauth2provider_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

func TestClientDPoPModeRoundTrip(t *testing.T) {
	s := oauth2provider.NewMemoryStore()
	ctx := context.Background()

	client := &oauth2provider.OAuth2Client{
		ClientID:   "client-dpop",
		Name:       "DPoP client",
		DPoPMode:   "required",
		GrantTypes: []string{"authorization_code"},
	}
	require.NoError(t, s.CreateClient(ctx, client))

	got, err := s.GetClient(ctx, "client-dpop")
	require.NoError(t, err)
	assert.Equal(t, "required", got.DPoPMode)
}

func TestClientDPoPModeDefaultsToInherit(t *testing.T) {
	s := oauth2provider.NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ClientID: "client-plain", Name: "Plain", GrantTypes: []string{"authorization_code"},
	}))

	got, err := s.GetClient(ctx, "client-plain")
	require.NoError(t, err)
	assert.Empty(t, got.DPoPMode, "empty means inherit from the app, matching TokenFormat")
}
