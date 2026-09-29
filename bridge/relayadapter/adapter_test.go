package relayadapter_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/relay"
	"github.com/xraph/relay/id"
	"github.com/xraph/relay/signature"
	"github.com/xraph/relay/store/memory"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/bridge/relayadapter"
	"github.com/xraph/authsome/webhook"
)

// An endpoint managed through the adapter round-trips to Relay's own
// endpoint service: created with the secret authsome chose, updated,
// disabled, rotated and deleted, and a delivery Relay signs verifies with
// the secret the caller holds.
func TestEndpointLifecycle(t *testing.T) {
	r, err := relay.New(relay.WithStore(memory.New()))
	require.NoError(t, err)
	a := relayadapter.New(r)
	ctx := context.Background()

	epID, err := a.CreateEndpoint(ctx, bridge.EndpointInput{
		TenantID: "app-1", URL: "https://receiver.example/hook", Secret: "whsec_first",
		EventTypes: []string{"user.created"}, Metadata: map[string]string{"authsome_webhook_id": "awhk_1"},
	})
	require.NoError(t, err)
	parsed, err := id.Parse(epID)
	require.NoError(t, err)
	ep, err := r.Endpoints().Get(ctx, parsed)
	require.NoError(t, err)
	assert.Equal(t, "whsec_first", ep.Secret, "relay signs with the secret authsome chose")
	assert.True(t, ep.Enabled)

	body := []byte(`{"id":"u_1"}`)
	ts := time.Now().Unix()
	sig := signature.Sign(body, ep.Secret, ts)
	assert.NoError(t, webhook.VerifyRelay(body, "whsec_first", strconv.FormatInt(ts, 10), sig, 0),
		"a receiver verifies relay's signature with the once-shown secret")
	assert.ErrorIs(t, webhook.VerifyRelay(body, "whsec_other", strconv.FormatInt(ts, 10), sig, 0), webhook.ErrSignatureMismatch)

	require.NoError(t, a.UpdateEndpoint(ctx, epID, bridge.EndpointInput{
		TenantID: "app-1", URL: "https://receiver.example/v2", Secret: "whsec_first", EventTypes: []string{"user.*"},
	}))
	require.NoError(t, a.SetEndpointEnabled(ctx, epID, false))
	ep, err = r.Endpoints().Get(ctx, parsed)
	require.NoError(t, err)
	assert.Equal(t, "https://receiver.example/v2", ep.URL)
	assert.Equal(t, []string{"user.*"}, ep.EventTypes)
	assert.False(t, ep.Enabled)

	rotated, err := a.RotateEndpointSecret(ctx, epID)
	require.NoError(t, err)
	assert.NotEqual(t, "whsec_first", rotated)
	ep, err = r.Endpoints().Get(ctx, parsed)
	require.NoError(t, err)
	assert.Equal(t, rotated, ep.Secret)

	require.NoError(t, a.DeleteEndpoint(ctx, epID))
	_, err = r.Endpoints().Get(ctx, parsed)
	assert.Error(t, err)
	assert.NoError(t, a.DeleteEndpoint(ctx, epID), "deleting an endpoint that is already gone is not an error")
}
