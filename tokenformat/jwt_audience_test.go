package tokenformat_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/tokenformat"
)

func newTestJWT(t *testing.T, configuredAudience string) *tokenformat.JWT {
	t.Helper()
	j, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-key-not-a-real-secret-000000"),
		Issuer:        "https://auth.example.com",
		Audience:      configuredAudience,
	})
	require.NoError(t, err)
	return j
}

func TestJWT_Audience(t *testing.T) {
	base := tokenformat.TokenClaims{
		UserID:    "user_1",
		AppID:     "app_1",
		SessionID: "sess_1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tests := []struct {
		name         string
		configured   string
		perToken     []string
		wantAudience []string
	}{
		{
			name:         "per-token audience wins over configured",
			configured:   "https://legacy.example.com",
			perToken:     []string{"https://api.example.com"},
			wantAudience: []string{"https://api.example.com"},
		},
		{
			name:         "configured audience is the fallback",
			configured:   "https://legacy.example.com",
			perToken:     nil,
			wantAudience: []string{"https://legacy.example.com"},
		},
		{
			name:         "no audience anywhere stays empty",
			configured:   "",
			perToken:     nil,
			wantAudience: nil,
		},
		{
			name:         "multiple resources round trip as an array",
			configured:   "",
			perToken:     []string{"https://api.example.com", "https://files.example.com"},
			wantAudience: []string{"https://api.example.com", "https://files.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newTestJWT(t, tt.configured)

			claims := base
			claims.Audience = tt.perToken

			token, err := j.GenerateAccessToken(claims)
			require.NoError(t, err)

			got, err := j.ValidateAccessToken(token)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAudience, got.Audience)
		})
	}
}
