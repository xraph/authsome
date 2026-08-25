package authsome_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/tokenformat"
)

// TestRefreshPreservesAudience proves a refresh does not widen a token.
//
// A token bound to one resource server must stay bound across rotation. If the
// audience is dropped on refresh the client keeps working and nobody notices,
// while the token silently becomes valid everywhere, which is a worse hole
// than the one resource indicators close.
//
// The opaque half already passes before the fix, because RotateSession writes
// the whole model. The JWT half fails, because service.go rebuilds the claims
// from scratch and never copies the audience across.
func TestRefreshPreservesAudience(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)

	eng, st := newTestEngine(t, authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "aud-refresh@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Aud User",
	})
	require.NoError(t, err)

	// Bind the session to one resource, the way issueTokens does.
	sess.Audience = []string{"https://api.example.com"}
	require.NoError(t, st.UpdateSession(ctx, sess))

	refreshed, err := eng.Refresh(ctx, sess.RefreshToken)
	require.NoError(t, err)

	assert.Equal(t, []string{"https://api.example.com"}, refreshed.Audience,
		"the rotated session lost its audience")

	claims, err := jwtFmt.ValidateAccessToken(refreshed.Token)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://api.example.com"}, claims.Audience,
		"the regenerated JWT lost its aud claim, so the refresh widened the token")
}

// TestNewSessionRecordsConfiguredJWTAudience proves the session row and the
// token it holds agree about the audience.
//
// A core login mints its JWT with no per-token audience, so
// GenerateAccessToken falls back to JWTConfig.Audience. The row used to be
// written with none. Two checks read those two places for the same credential,
// the JWT guard reading the claim and the session guard reading the row, so
// leaving them different is what let a refused JWT come back authenticated
// through a session lookup.
func TestNewSessionRecordsConfiguredJWTAudience(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
		Audience:      "https://api.example.com",
	})
	require.NoError(t, err)

	eng, _ := newTestEngine(t, authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))
	ctx := context.Background()

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     testAppID(t),
		Email:     "aud-newsession@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Aud User",
	})
	require.NoError(t, err)

	claims, err := jwtFmt.ValidateAccessToken(sess.Token)
	require.NoError(t, err)
	require.Equal(t, []string{"https://api.example.com"}, claims.Audience,
		"the minted JWT should carry the configured audience")

	assert.Equal(t, claims.Audience, sess.Audience,
		"the session row must record the same audience the token went out with")
}

// TestNewSessionWithoutConfiguredAudienceStaysUnaudienced is the
// backwards-compatibility half. With no JWTConfig.Audience there is nothing to
// record, and a session that has never named a resource must keep saying so:
// an audience appearing from nowhere would start failing the guard for every
// deployment that turns session.resource_identifier on.
func TestNewSessionWithoutConfiguredAudienceStaysUnaudienced(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)

	eng, _ := newTestEngine(t, authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))
	ctx := context.Background()

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     testAppID(t),
		Email:     "aud-newsession-none@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Aud User",
	})
	require.NoError(t, err)

	claims, err := jwtFmt.ValidateAccessToken(sess.Token)
	require.NoError(t, err)
	assert.Empty(t, claims.Audience, "an unconfigured app must not stamp an aud claim")
	assert.Empty(t, sess.Audience, "an unconfigured app must not stamp a session audience")
}
