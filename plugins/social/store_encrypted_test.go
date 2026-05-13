package social

import (
	"context"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
)

func newKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, 32)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return k
}

func newTestConn(t *testing.T) *OAuthConnection {
	t.Helper()
	return &OAuthConnection{
		ID:             id.New(id.PrefixOAuthConnection),
		AppID:          id.NewAppID(),
		UserID:         id.NewUserID(),
		Provider:       "google",
		ProviderUserID: "google-uid-123",
		Email:          "user@example.com",
		AccessToken:    "plain-access-token-secret",
		RefreshToken:   "plain-refresh-token-secret",
	}
}

// TestEncryptedStore_RoundTrip verifies that tokens are ciphertext at the
// underlying-storage layer but plaintext at the API layer.
func TestEncryptedStore_RoundTrip(t *testing.T) {
	inner := NewMemoryStore()
	enc, err := bridge.NewAESGCMEncryptor(newKey(t))
	require.NoError(t, err)

	wrapped := NewEncryptedStore(inner, enc)

	conn := newTestConn(t)
	plainAccess := conn.AccessToken
	plainRefresh := conn.RefreshToken

	require.NoError(t, wrapped.CreateOAuthConnection(context.Background(), conn))

	// Caller's struct must not be mutated.
	require.Equal(t, plainAccess, conn.AccessToken)
	require.Equal(t, plainRefresh, conn.RefreshToken)

	// At the underlying storage layer, tokens MUST be ciphertext.
	rawConns, err := inner.GetOAuthConnectionsByUserID(context.Background(), conn.UserID)
	require.NoError(t, err)
	require.Len(t, rawConns, 1)
	require.True(t, strings.HasPrefix(rawConns[0].AccessToken, "v1:"), "stored access_token should be v1-encrypted, got %q", rawConns[0].AccessToken)
	require.True(t, strings.HasPrefix(rawConns[0].RefreshToken, "v1:"))
	require.NotEqual(t, plainAccess, rawConns[0].AccessToken)

	// Through the wrapper, reads must yield plaintext.
	got, err := wrapped.GetOAuthConnection(context.Background(), "google", "google-uid-123")
	require.NoError(t, err)
	require.Equal(t, plainAccess, got.AccessToken)
	require.Equal(t, plainRefresh, got.RefreshToken)
}

// TestEncryptedStore_LegacyPlaintextRead verifies that a row written
// before encryption was deployed (plaintext in the underlying store)
// is still readable through the wrapper.
func TestEncryptedStore_LegacyPlaintextRead(t *testing.T) {
	inner := NewMemoryStore()
	enc, err := bridge.NewAESGCMEncryptor(newKey(t))
	require.NoError(t, err)
	wrapped := NewEncryptedStore(inner, enc)

	// Simulate a legacy row by writing directly to the inner store.
	legacy := newTestConn(t)
	require.NoError(t, inner.CreateOAuthConnection(context.Background(), legacy))

	got, err := wrapped.GetOAuthConnection(context.Background(), "google", "google-uid-123")
	require.NoError(t, err)
	require.Equal(t, "plain-access-token-secret", got.AccessToken)
	require.Equal(t, "plain-refresh-token-secret", got.RefreshToken)
}

// TestEncryptedStore_NilEncryptorIsNoop verifies that constructing with
// a nil Encryptor falls back to NoopEncryptor (no crash, no encryption).
func TestEncryptedStore_NilEncryptorIsNoop(t *testing.T) {
	inner := NewMemoryStore()
	wrapped := NewEncryptedStore(inner, nil)

	conn := newTestConn(t)
	require.NoError(t, wrapped.CreateOAuthConnection(context.Background(), conn))

	rawConns, err := inner.GetOAuthConnectionsByUserID(context.Background(), conn.UserID)
	require.NoError(t, err)
	require.Len(t, rawConns, 1)
	require.Equal(t, "plain-access-token-secret", rawConns[0].AccessToken)
}
