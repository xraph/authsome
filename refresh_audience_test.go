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
