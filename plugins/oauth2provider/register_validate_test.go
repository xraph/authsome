package oauth2provider

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		ok   bool
	}{
		{"https with host", "https://app.example.com/cb", true},
		{"https with port", "https://app.example.com:8443/cb", true},
		{"loopback v4", "http://127.0.0.1:9000/cb", true},
		{"loopback v4 no port", "http://127.0.0.1/cb", true},
		{"loopback v6", "http://[::1]:9000/cb", true},
		{"private-use scheme", "com.example.app:/callback", true},

		{"empty", "", false},
		{"http non-loopback", "http://app.example.com/cb", false},
		{"localhost by name", "http://localhost:9000/cb", false},
		{"https no host", "https:///cb", false},
		{"fragment", "https://app.example.com/cb#frag", false},
		{"userinfo", "https://user:pw@app.example.com/cb", false},
		{"wildcard host", "https://*.example.com/cb", false},
		{"scheme without dot", "myapp:/callback", false},
		{"javascript scheme", "javascript:alert(1)", false},
		{"not a url", "://////", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedirectURI(tc.uri)
			if tc.ok {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}

func TestClampGrantTypes(t *testing.T) {
	t.Run("defaults to authorization_code when empty", func(t *testing.T) {
		got, err := clampGrantTypes(nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"authorization_code"}, got)
	})

	t.Run("keeps the allowed pair", func(t *testing.T) {
		got, err := clampGrantTypes([]string{"authorization_code", "refresh_token"})
		require.NoError(t, err)
		assert.Equal(t, []string{"authorization_code", "refresh_token"}, got)
	})

	// A dynamic client holding client_credentials gets a session token with
	// no user and no consent step, so this is a rejection and not a drop.
	t.Run("rejects client_credentials", func(t *testing.T) {
		_, err := clampGrantTypes([]string{"authorization_code", "client_credentials"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client_credentials")
	})

	t.Run("rejects the device grant", func(t *testing.T) {
		_, err := clampGrantTypes([]string{deviceCodeGrantType})
		assert.Error(t, err)
	})

	t.Run("dedups, first occurrence wins", func(t *testing.T) {
		got, err := clampGrantTypes([]string{"refresh_token", "authorization_code", "refresh_token"})
		require.NoError(t, err)
		assert.Equal(t, []string{"refresh_token", "authorization_code"}, got)
	})
}

func TestClampScopes(t *testing.T) {
	allow := []string{"openid", "profile", "email", "offline_access"}

	t.Run("empty request yields the allowlist", func(t *testing.T) {
		assert.Equal(t, allow, clampScopes(nil, allow))
	})

	t.Run("drops what is outside the allowlist", func(t *testing.T) {
		got := clampScopes([]string{"openid", "admin:all", "email"}, allow)
		assert.Equal(t, []string{"openid", "email"}, got)
	})

	t.Run("everything dropped yields empty, not the allowlist", func(t *testing.T) {
		assert.Empty(t, clampScopes([]string{"admin:all"}, allow))
	})

	t.Run("dedups, first occurrence wins", func(t *testing.T) {
		got := clampScopes([]string{"openid", "email", "openid"}, allow)
		assert.Equal(t, []string{"openid", "email"}, got)
	})
}

// RFC 7591 section 3.2.2 fixes the error body. MCP clients parse these two
// fields and will not find them in the forge envelope.
func TestRegErrorWireShape(t *testing.T) {
	e := regError(http.StatusBadRequest, errInvalidRedirectURI, "loopback only")
	assert.Equal(t, http.StatusBadRequest, e.StatusCode())

	body, ok := e.ResponseBody().(*oauthRegError)
	require.True(t, ok)
	assert.Equal(t, errInvalidRedirectURI, body.Code)
	assert.Equal(t, "loopback only", body.Desc)
}
