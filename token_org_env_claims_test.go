package authsome_test

// A JWT access token is validated without loading its session row, so the
// org and environment a resource server scopes the request to have to be in
// the token itself. TokenClaims always declared org_id and env_id, but no
// signing site set them, so every JWT went out without either.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/tokenformat"
)

// rawJWTClaims decodes a JWT payload without verifying it. The test reads the
// raw map rather than tokenformat's struct so it can tell an omitted claim
// from an empty one.
func rawJWTClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	require.True(t, tokenformat.IsJWT(token), "expected a JWT, got %q", token)
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	return claims
}

// newJWTOrgEngine builds a JWT-format engine with the fake organization
// plugin registered, so SwitchActiveOrg can check membership.
func newJWTOrgEngine(t *testing.T) (*authsome.Engine, *memory.Store, *fakeOrgPlugin) {
	t.Helper()
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)
	orgs := &fakeOrgPlugin{orgs: map[string][]id.OrgID{}}
	eng, s := newTestEngine(t,
		authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt),
		authsome.WithPlugin(orgs),
	)
	seedDefaultEnv(t, eng)
	return eng, s, orgs
}

// seedDefaultEnv gives the test app a default environment. The test engine
// starts without one, and a session with no environment would make the
// env_id assertions vacuous.
func seedDefaultEnv(t *testing.T, eng *authsome.Engine) *environment.Environment {
	t.Helper()
	env := &environment.Environment{
		AppID:     testAppID(t),
		Name:      "Production",
		Slug:      "production",
		Type:      environment.TypeProduction,
		IsDefault: true,
	}
	require.NoError(t, eng.CreateEnvironment(context.Background(), env))
	return env
}

// signUpJWT signs a user up and back in, returning the sign-in session.
// Sign-up and sign-in mint through different callers of newSession, and
// both are checked.
func signUpJWT(t *testing.T, eng *authsome.Engine, email string) (signUp, signIn *session.Session) {
	t.Helper()
	ctx := context.Background()
	appID := testAppID(t)
	u, up, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID: appID, Email: email, Password: "SecureP@ss123", FirstName: "Claims",
	})
	require.NoError(t, err)
	// The seeded production environment asks for a verified address.
	secutil.VerifyEmail(t, eng, u.ID)
	_, in, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID: appID, Email: email, Password: "SecureP@ss123",
	})
	require.NoError(t, err)
	return up, in
}

func TestJWT_SignInStampsEnvAndOmitsOrgWithoutOne(t *testing.T) {
	eng, _, _ := newJWTOrgEngine(t)
	up, in := signUpJWT(t, eng, "claims-signin@example.com")

	for name, sess := range map[string]*session.Session{"sign-up": up, "sign-in": in} {
		t.Run(name, func(t *testing.T) {
			require.False(t, sess.EnvID.IsNil(), "the session should be bound to the default environment")
			claims := rawJWTClaims(t, sess.Token)
			assert.Equal(t, sess.EnvID.String(), claims["env_id"], "env_id has to match the session row")
			_, hasOrg := claims["org_id"]
			assert.False(t, hasOrg, "a session with no active org must omit org_id, not send it empty")
		})
	}
}

func TestJWT_RefreshWithoutOrgKeepsEnv(t *testing.T) {
	eng, _, _ := newJWTOrgEngine(t)
	_, in := signUpJWT(t, eng, "claims-refresh-noorg@example.com")

	refreshed, err := eng.Refresh(context.Background(), in.RefreshToken)
	require.NoError(t, err)

	claims := rawJWTClaims(t, refreshed.Token)
	assert.Equal(t, in.EnvID.String(), claims["env_id"])
	_, hasOrg := claims["org_id"]
	assert.False(t, hasOrg)
}

func TestJWT_SwitchOrgMintsTokenForNewOrgAndRefreshKeepsIt(t *testing.T) {
	eng, s, orgs := newJWTOrgEngine(t)
	ctx := context.Background()
	_, in := signUpJWT(t, eng, "claims-switch@example.com")

	orgID := id.NewOrgID()
	orgs.orgs[in.UserID.String()] = []id.OrgID{orgID}

	switched, err := eng.SwitchActiveOrg(ctx, in.ID, orgID)
	require.NoError(t, err)
	require.NotEmpty(t, switched.Token, "a JWT app has to get a token that carries the new org")
	assert.NotEqual(t, in.Token, switched.Token)

	claims := rawJWTClaims(t, switched.Token)
	assert.Equal(t, orgID.String(), claims["org_id"])
	assert.Equal(t, in.EnvID.String(), claims["env_id"])
	assert.Equal(t, in.ID.String(), claims["sid"], "switching org keeps the session")
	assert.Equal(t, float64(in.ExpiresAt.Unix()), claims["exp"], "switching org must not extend the access token")

	// The row has to know the new token, or a cookie refresh keyed on the
	// access token would stop finding the session.
	stored, err := s.GetSession(ctx, in.ID)
	require.NoError(t, err)
	assert.Equal(t, store.HashToken(switched.Token), stored.TokenHash)
	assert.Equal(t, orgID.String(), stored.OrgID.String())

	refreshed, err := eng.Refresh(ctx, in.RefreshToken)
	require.NoError(t, err)
	claims = rawJWTClaims(t, refreshed.Token)
	assert.Equal(t, orgID.String(), claims["org_id"], "refresh has to keep the active org")
	assert.Equal(t, in.EnvID.String(), claims["env_id"], "refresh has to keep the environment")
}

func TestJWT_SwitchOrgToNoneDropsOrgClaim(t *testing.T) {
	eng, _, orgs := newJWTOrgEngine(t)
	ctx := context.Background()
	_, in := signUpJWT(t, eng, "claims-clear@example.com")

	orgID := id.NewOrgID()
	orgs.orgs[in.UserID.String()] = []id.OrgID{orgID}
	_, err := eng.SwitchActiveOrg(ctx, in.ID, orgID)
	require.NoError(t, err)

	cleared, err := eng.SwitchActiveOrg(ctx, in.ID, id.OrgID{})
	require.NoError(t, err)
	claims := rawJWTClaims(t, cleared.Token)
	_, hasOrg := claims["org_id"]
	assert.False(t, hasOrg, "clearing the active org has to clear it from the token too")
	assert.Equal(t, in.EnvID.String(), claims["env_id"])
}

// An opaque token has no claims to go stale: the middleware reads the org off
// the row on every request. So switching org leaves the caller's token alone.
func TestOpaque_SwitchOrgKeepsToken(t *testing.T) {
	orgs := &fakeOrgPlugin{orgs: map[string][]id.OrgID{}}
	eng, _ := newTestEngine(t, authsome.WithPlugin(orgs))
	seedDefaultEnv(t, eng)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID: appID, Email: "claims-opaque@example.com", Password: "SecureP@ss123", FirstName: "Opaque",
	})
	require.NoError(t, err)
	require.False(t, tokenformat.IsJWT(sess.Token))

	orgID := id.NewOrgID()
	orgs.orgs[sess.UserID.String()] = []id.OrgID{orgID}
	switched, err := eng.SwitchActiveOrg(ctx, sess.ID, orgID)
	require.NoError(t, err)
	assert.Empty(t, switched.Token, "an opaque session keeps its token, so there is nothing new to hand back")

	resolved, err := eng.ResolveSessionByToken(ctx, sess.Token)
	require.NoError(t, err)
	assert.Equal(t, orgID.String(), resolved.OrgID.String())
}

// The OAuth2 grants have no org of their own: an authorization code and a
// device code record the user, client and scopes, not an organization, and
// the provider session starts with none. So the token carries the environment
// it was bound to and no org_id.
func TestOAuth2IssuedJWTStampsEnv(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)
	eng, _ := newTestEngine(t, authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))
	appID := testAppID(t)
	env := seedDefaultEnv(t, eng)

	issued, refresh := issueOAuth2JWT(t, eng, appID)
	tokens := map[string]string{
		"issued by the code grant":    issued.AccessToken,
		"issued by the refresh grant": refresh(issued.RefreshToken).AccessToken,
	}
	for name, token := range tokens {
		t.Run(name, func(t *testing.T) {
			claims := rawJWTClaims(t, token)
			assert.Equal(t, env.ID.String(), claims["env_id"])
			_, hasOrg := claims["org_id"]
			assert.False(t, hasOrg)
		})
	}
}
