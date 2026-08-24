package oauth2provider_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

func TestMemoryStore_ClientRoundTripsRegistrationFields(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	ctx := context.Background()
	expires := time.Now().Add(90 * 24 * time.Hour).UTC().Truncate(time.Second)

	want := &oauth2provider.OAuth2Client{
		ID:                      id.NewOAuth2ClientID(),
		AppID:                   id.NewAppID(),
		Name:                    "Dynamic Client",
		ClientID:                "dyn-1",
		RedirectURIs:            []string{"http://127.0.0.1:9000/cb"},
		Scopes:                  []string{"openid"},
		GrantTypes:              []string{"authorization_code"},
		Public:                  true,
		TokenEndpointAuthMethod: "none",
		RegistrationTokenHash:   "$2a$04$abcdefghijklmnopqrstuv",
		DynamicallyRegistered:   true,
		ClientSecretExpiresAt:   &expires,
		Metadata: map[string]any{
			"client_uri":  "https://example.com",
			"software_id": "mcp-cli",
		},
	}
	require.NoError(t, st.CreateClient(ctx, want))

	got, err := st.GetClient(ctx, "dyn-1")
	require.NoError(t, err)
	assert.Equal(t, "none", got.TokenEndpointAuthMethod)
	assert.Equal(t, want.RegistrationTokenHash, got.RegistrationTokenHash)
	assert.True(t, got.DynamicallyRegistered)
	require.NotNil(t, got.ClientSecretExpiresAt)
	assert.Equal(t, expires, got.ClientSecretExpiresAt.UTC().Truncate(time.Second))
	assert.Equal(t, "mcp-cli", got.Metadata["software_id"])
}

// A client with no expiry must serialise without the key at all, not as the
// zero time. A bare time.Time would defeat omitempty here, since
// encoding/json never treats a struct as empty.
func TestOAuth2Client_OmitsUnsetSecretExpiry(t *testing.T) {
	b, err := json.Marshal(&oauth2provider.OAuth2Client{ClientID: "x"})
	require.NoError(t, err)
	assert.NotContains(t, string(b), "client_secret_expires_at")
}

// The registration token hash is a credential. It must never reach a JSON
// response body, the way ClientSecret already does not.
func TestOAuth2Client_RegistrationTokenHashIsNotSerialised(t *testing.T) {
	c := &oauth2provider.OAuth2Client{
		ClientID:              "dyn-1",
		RegistrationTokenHash: "$2a$04$secret",
	}
	b, err := json.Marshal(c)
	require.NoError(t, err)
	// Checking for the bare substring "secret" would also match the
	// unrelated client_secret_expires_at key, so assert on the actual
	// hash value and the field's own json key instead.
	assert.NotContains(t, string(b), c.RegistrationTokenHash)
	assert.NotContains(t, string(b), "registration_token_hash")
}
