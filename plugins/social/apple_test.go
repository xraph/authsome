package social

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/oauth2"
)

func appleToken(t *testing.T, claims map[string]any) *oauth2.Token {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	idToken := "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	return (&oauth2.Token{AccessToken: "at"}).WithExtra(map[string]any{"id_token": idToken})
}

func TestApple_HonoursEmailVerified(t *testing.T) {
	p := &appleProvider{}
	verified, err := p.FetchUser(context.Background(), appleToken(t, map[string]any{"sub": "001", "email": "a@privaterelay.appleid.com", "email_verified": true}))
	require.NoError(t, err)
	assert.True(t, verified.EmailVerified)

	asString, err := p.FetchUser(context.Background(), appleToken(t, map[string]any{"sub": "002", "email": "b@example.com", "email_verified": "true"}))
	require.NoError(t, err)
	assert.True(t, asString.EmailVerified, "Apple sends the claim as a string in some tokens")

	unverified, err := p.FetchUser(context.Background(), appleToken(t, map[string]any{"sub": "003", "email": "c@example.com", "email_verified": "false"}))
	require.NoError(t, err)
	assert.False(t, unverified.EmailVerified)

	absent, err := p.FetchUser(context.Background(), appleToken(t, map[string]any{"sub": "004", "email": "d@example.com"}))
	require.NoError(t, err)
	assert.False(t, absent.EmailVerified, "no claim, no proof of ownership")
}
