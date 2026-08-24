package oauth2provider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// TestMemoryStore_ResourcesRoundTrip pins the resource allowlist and the
// per-code grant to storage. A backend that drops either one fails open: the
// client appears to have no allowlist (so every request is rejected as
// invalid_target) or the code carries no resources (so the issued token comes
// back unrestricted).
func TestMemoryStore_ResourcesRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	appID := id.NewAppID()
	want := []string{"https://api.example.com", "https://files.example.com"}

	client := &OAuth2Client{
		ID:        id.NewOAuth2ClientID(),
		AppID:     appID,
		Name:      "test",
		ClientID:  "client-abc",
		Scopes:    []string{"openid"},
		Resources: want,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateClient(ctx, client))

	gotClient, err := s.GetClient(ctx, "client-abc")
	require.NoError(t, err)
	assert.Equal(t, want, gotClient.Resources)

	code := &AuthorizationCode{
		ID:        id.NewAuthCodeID(),
		Code:      "code-abc",
		ClientID:  "client-abc",
		UserID:    id.NewUserID(),
		AppID:     appID,
		Scopes:    []string{"openid"},
		Resources: []string{"https://api.example.com"},
		ExpiresAt: time.Now().Add(time.Minute),
		CreatedAt: time.Now(),
	}
	require.NoError(t, s.CreateAuthCode(ctx, code))

	gotCode, err := s.GetAuthCode(ctx, "code-abc")
	require.NoError(t, err)
	assert.Equal(t, []string{"https://api.example.com"}, gotCode.Resources)

	dc := &DeviceCode{
		ID:         id.NewDeviceCodeID(),
		DeviceCode: "dev-abc",
		UserCode:   "BCDF-GHJK",
		ClientID:   "client-abc",
		AppID:      appID,
		Scopes:     []string{"openid"},
		Resources:  []string{"https://files.example.com"},
		Status:     DeviceCodeStatusPending,
		ExpiresAt:  time.Now().Add(time.Minute),
		CreatedAt:  time.Now(),
	}
	require.NoError(t, s.CreateDeviceCode(ctx, dc))

	gotDC, err := s.GetDeviceCodeByDeviceCode(ctx, "dev-abc")
	require.NoError(t, err)
	assert.Equal(t, []string{"https://files.example.com"}, gotDC.Resources)
}
