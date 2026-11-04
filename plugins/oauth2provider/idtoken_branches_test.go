package oauth2provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

const idTokenFailures = "forge_authsome_oauth2_id_token_failures_"

// mintPlugin boots a plugin on an engine whose metrics the test can scrape.
func mintPlugin(t *testing.T, st Store) (*Plugin, *authsome.Engine, func() string) {
	t.Helper()
	reg, scrape := telemetrytest.Registry(t)
	eng := secutil.NewTestEngine(t, authsome.WithMetricsSink(reg), authsome.WithTokenEncryptor(testEncryptor(t)))
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(st)
	require.NoError(t, p.OnInit(context.Background(), eng))
	return p, eng, scrape
}

func failureSeries(scrape func() string, reason string) []string {
	var out []string
	for _, s := range telemetrytest.Series(scrape(), idTokenFailures) {
		if strings.Contains(s, `reason="`+reason+`"`) {
			out = append(out, s)
		}
	}
	return out
}

func testClient(appID id.AppID) *OAuth2Client {
	return &OAuth2Client{ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: "rp-client"}
}

// A grant for a user that no longer exists gets no id_token, and the failure
// is counted as a user lookup.
func TestIDTokenFor_MissingUser(t *testing.T) {
	p, _, scrape := mintPlugin(t, NewMemoryStore())
	tok, err := p.idTokenFor(context.Background(), testClient(id.NewAppID()), id.NewUserID(), []string{"openid"}, "at",
		&oidcRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load id_token subject")
	assert.Empty(t, tok)
	assert.Len(t, failureSeries(scrape, "user_lookup"), 1)
}

func TestIDTokenKeyFor_NoKeyManager(t *testing.T) {
	p, _, scrape := mintPlugin(t, NewMemoryStore())
	p.keys = nil
	_, err := p.idTokenKeyFor(context.Background(), testClient(id.NewAppID()))
	require.ErrorIs(t, err, errNoSigningKey)
	assert.Len(t, failureSeries(scrape, "no_key"), 1)
}

// A key store outage is a signing error, not "no key configured": the
// operator needs to look at the database, not the configuration.
func TestIDTokenKeyFor_StoreFailureIsSignError(t *testing.T) {
	st := &faultyStore{Store: NewMemoryStore(), listKeys: errStoreDown}
	p, _, scrape := mintPlugin(t, st)
	_, err := p.idTokenKeyFor(context.Background(), testClient(id.NewAppID()))
	require.ErrorIs(t, err, errStoreDown)
	assert.Len(t, failureSeries(scrape, "sign_error"), 1)
	assert.Empty(t, failureSeries(scrape, "no_key"))
}

func TestSignIDTokenWith_Failures(t *testing.T) {
	p, _, scrape := mintPlugin(t, NewMemoryStore())
	rk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	in := idTokenInput{Client: testClient(id.NewAppID()), User: &user.User{ID: id.NewUserID()}, AccessToken: "at"}

	// An alg at_hash can't be computed for.
	_, err = p.signIDTokenWith(&idTokenKey{issuer: "https://iss", s: &signer{kid: "k", alg: "PS256", key: rk}}, in)
	require.Error(t, err)
	// A key that doesn't match its alg.
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tok, err := p.signIDTokenWith(&idTokenKey{issuer: "https://iss", s: &signer{kid: "k", alg: "RS256", key: ec}}, in)
	require.Error(t, err)
	assert.Empty(t, tok)

	series := failureSeries(scrape, "sign_error")
	require.Len(t, series, 1)
	assert.Contains(t, scrape(), series[0]+" 2", "both failures are counted")
	assert.Empty(t, telemetrytest.Series(scrape(), "forge_authsome_oauth2_id_tokens_issued_"))
}

func TestSignIDToken_UnsupportedAlg(t *testing.T) {
	rk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	_, err = signIDToken(&signer{kid: "k", alg: "PS256", key: rk}, jwt.MapClaims{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported id_token alg "PS256"`)
}

func TestPlanRefreshIDToken_NoCoreStore(t *testing.T) {
	p := New(Config{})
	plan, err := p.planRefreshIDToken(context.Background(), testClient(id.NewAppID()), "rt")
	require.NoError(t, err)
	assert.Nil(t, plan)
}

// A refresh token whose user is gone refuses before anything rotates, and
// says why.
func TestPlanRefreshIDToken_MissingUser(t *testing.T) {
	ctx := context.Background()
	p, eng, scrape := mintPlugin(t, NewMemoryStore())
	client := testClient(id.NewAppID())
	require.NoError(t, eng.Store().CreateSession(ctx, &session.Session{
		ID: id.NewSessionID(), AppID: client.AppID, UserID: id.NewUserID(),
		RefreshToken: "rt-orphan", ClientID: client.ClientID, Scopes: []string{"openid"},
	}))

	plan, err := p.planRefreshIDToken(ctx, client, "rt-orphan")
	require.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "load id_token subject")
	assert.Len(t, failureSeries(scrape, "user_lookup"), 1)
}
