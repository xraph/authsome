package oauth2provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

// OpenID Connect Core A.3 gives this access token and its at_hash.
func TestAtHash_SpecVector(t *testing.T) {
	got, err := atHash("jHkWEdUXMU1BwAsC4vtUsZwnNvTIxEl0z9K3vx5KF0Y", "RS256")
	require.NoError(t, err)
	assert.Equal(t, "77QmUPtjPfzWtF2AnpK9RQ", got)
}

func TestAtHash_UnknownAlgIsAnError(t *testing.T) {
	for _, alg := range []string{"", "HS256", "none", "PS512"} {
		_, err := atHash("token", alg)
		assert.Error(t, err, alg)
	}
}

// The signer's alg and key type must agree. A mismatch is refused instead of
// signed under the wrong method.
func TestSignIDToken_KeyAlgMismatchIsRefused(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	_, err = signIDToken(&signer{kid: "k", alg: "RS256", key: ec}, jwt.MapClaims{})
	require.Error(t, err)

	rk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	_, err = signIDToken(&signer{kid: "k", alg: "ES256", key: rk}, jwt.MapClaims{})
	require.Error(t, err)
}

// standardClaims hands out pointers to copies, so a claim set never aliases
// the user it was built from.
func TestStandardClaims_VerifiedFlagsDoNotAliasUser(t *testing.T) {
	u := &user.User{ID: id.NewUserID(), EmailVerified: true, PhoneVerified: true}
	c := standardClaims(u, func(string) bool { return true })
	u.EmailVerified, u.PhoneVerified = false, false
	require.NotNil(t, c.EmailVerified)
	require.NotNil(t, c.PhoneNumberVerified)
	assert.True(t, *c.EmailVerified)
	assert.True(t, *c.PhoneNumberVerified)
}
