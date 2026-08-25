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

// resourcesCases is the shared table for converter round-trip coverage: one
// case with a populated resource list, one with none. The empty case matters
// as much as the populated one: it is what catches an encoding that
// flattens []string into a delimited string (round-tripping "" back out as a
// spurious one-element slice instead of staying empty).
var resourcesCases = []struct {
	name      string
	resources []string
}{
	{name: "with resources", resources: []string{"https://api.example.com", "https://files.example.com"}},
	{name: "no resources", resources: nil},
}

func assertResourcesRoundTrip(t *testing.T, want, got []string) {
	t.Helper()
	if len(want) == 0 {
		assert.Empty(t, got)
		return
	}
	assert.Equal(t, want, got)
}

// TestOAuth2ClientSQLConverters_ResourcesRoundTrip exercises fromOAuth2Client
// and toOAuth2Client directly, the pair the memory-store test above never
// reaches. Deleting the encode block in fromOAuth2Client leaves the model's
// Resources field at its zero value, so the populated case comes back empty
// instead of matching; deleting the decode block in toOAuth2Client leaves the
// returned struct's Resources nil for the same reason.
func TestOAuth2ClientSQLConverters_ResourcesRoundTrip(t *testing.T) {
	for _, tt := range resourcesCases {
		t.Run(tt.name, func(t *testing.T) {
			client := &OAuth2Client{
				ID:        id.NewOAuth2ClientID(),
				AppID:     id.NewAppID(),
				Name:      "test",
				ClientID:  "client-abc",
				Scopes:    []string{"openid"},
				Resources: tt.resources,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			got, err := toOAuth2Client(fromOAuth2Client(client))
			require.NoError(t, err)
			assertResourcesRoundTrip(t, tt.resources, got.Resources)
		})
	}
}

// TestAuthCodeSQLConverters_ResourcesRoundTrip exercises fromAuthCode and
// toAuthCode directly. See TestOAuth2ClientSQLConverters_ResourcesRoundTrip
// for what a dropped encode or decode would do to these assertions.
func TestAuthCodeSQLConverters_ResourcesRoundTrip(t *testing.T) {
	for _, tt := range resourcesCases {
		t.Run(tt.name, func(t *testing.T) {
			code := &AuthorizationCode{
				ID:        id.NewAuthCodeID(),
				Code:      "code-abc",
				ClientID:  "client-abc",
				UserID:    id.NewUserID(),
				AppID:     id.NewAppID(),
				Scopes:    []string{"openid"},
				Resources: tt.resources,
				ExpiresAt: time.Now().Add(time.Minute),
				CreatedAt: time.Now(),
			}

			got, err := toAuthCode(fromAuthCode(code))
			require.NoError(t, err)
			assertResourcesRoundTrip(t, tt.resources, got.Resources)
		})
	}
}

// TestDeviceCodeSQLConverters_ResourcesRoundTrip exercises fromDeviceCode and
// toDeviceCode directly. See TestOAuth2ClientSQLConverters_ResourcesRoundTrip
// for what a dropped encode or decode would do to these assertions.
func TestDeviceCodeSQLConverters_ResourcesRoundTrip(t *testing.T) {
	for _, tt := range resourcesCases {
		t.Run(tt.name, func(t *testing.T) {
			dc := &DeviceCode{
				ID:         id.NewDeviceCodeID(),
				DeviceCode: "dev-abc",
				UserCode:   "BCDF-GHJK",
				ClientID:   "client-abc",
				AppID:      id.NewAppID(),
				Scopes:     []string{"openid"},
				Resources:  tt.resources,
				Status:     DeviceCodeStatusPending,
				ExpiresAt:  time.Now().Add(time.Minute),
				CreatedAt:  time.Now(),
			}

			got, err := toDeviceCode(fromDeviceCode(dc))
			require.NoError(t, err)
			assertResourcesRoundTrip(t, tt.resources, got.Resources)
		})
	}
}

// TestOAuth2ClientMongoConverters_ResourcesRoundTrip exercises
// oauth2ClientToDoc and oauth2ClientDocToModel directly. These are
// hand-written bson structs, not grove-mapped, so the only guard against a
// dropped Resources assignment in either converter is this test.
func TestOAuth2ClientMongoConverters_ResourcesRoundTrip(t *testing.T) {
	for _, tt := range resourcesCases {
		t.Run(tt.name, func(t *testing.T) {
			client := &OAuth2Client{
				ID:        id.NewOAuth2ClientID(),
				AppID:     id.NewAppID(),
				Name:      "test",
				ClientID:  "client-abc",
				Scopes:    []string{"openid"},
				Resources: tt.resources,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			got, err := oauth2ClientDocToModel(oauth2ClientToDoc(client))
			require.NoError(t, err)
			assertResourcesRoundTrip(t, tt.resources, got.Resources)
		})
	}
}

// TestAuthCodeMongoConverters_ResourcesRoundTrip exercises authCodeToDoc and
// authCodeDocToModel directly.
func TestAuthCodeMongoConverters_ResourcesRoundTrip(t *testing.T) {
	for _, tt := range resourcesCases {
		t.Run(tt.name, func(t *testing.T) {
			code := &AuthorizationCode{
				ID:        id.NewAuthCodeID(),
				Code:      "code-abc",
				ClientID:  "client-abc",
				UserID:    id.NewUserID(),
				AppID:     id.NewAppID(),
				Scopes:    []string{"openid"},
				Resources: tt.resources,
				ExpiresAt: time.Now().Add(time.Minute),
				CreatedAt: time.Now(),
			}

			got, err := authCodeDocToModel(authCodeToDoc(code))
			require.NoError(t, err)
			assertResourcesRoundTrip(t, tt.resources, got.Resources)
		})
	}
}

// TestDeviceCodeMongoConverters_ResourcesRoundTrip exercises
// deviceCodeToDoc and deviceCodeDocToModel directly.
func TestDeviceCodeMongoConverters_ResourcesRoundTrip(t *testing.T) {
	for _, tt := range resourcesCases {
		t.Run(tt.name, func(t *testing.T) {
			dc := &DeviceCode{
				ID:         id.NewDeviceCodeID(),
				DeviceCode: "dev-abc",
				UserCode:   "BCDF-GHJK",
				ClientID:   "client-abc",
				AppID:      id.NewAppID(),
				Scopes:     []string{"openid"},
				Resources:  tt.resources,
				Status:     DeviceCodeStatusPending,
				ExpiresAt:  time.Now().Add(time.Minute),
				CreatedAt:  time.Now(),
			}

			got, err := deviceCodeDocToModel(deviceCodeToDoc(dc))
			require.NoError(t, err)
			assertResourcesRoundTrip(t, tt.resources, got.Resources)
		})
	}
}
