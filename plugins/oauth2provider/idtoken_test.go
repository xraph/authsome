package oauth2provider_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

const (
	oidcIssuer   = "https://auth.example.com"
	oidcVerifier = "verifier-verifier-verifier-verifier-1234"
	noOpenIDID   = "no-openid-client"
	oidcDeviceID = "oidc-device-client"
)

// oidcFixture boots the plugin against a real engine and signs up one user.
// withEncryptor decides whether a managed key can be stored, so a fixture
// without one (and without an operator key) cannot sign anything.
type oidcFixture struct {
	p     *oauth2provider.Plugin
	eng   *authsome.Engine
	st    oauth2provider.Store
	mux   forge.Router
	appID id.AppID
	user  *user.User
}

func newOIDCFixture(t *testing.T, cfg oauth2provider.Config, withEncryptor bool, extra ...authsome.Option) *oidcFixture {
	t.Helper()
	var opts []authsome.Option
	if withEncryptor {
		enc, err := bridge.NewAESGCMEncryptor(bytes.Repeat([]byte{7}, 32))
		require.NoError(t, err)
		opts = append(opts, authsome.WithTokenEncryptor(enc))
	}
	eng := secutil.NewTestEngine(t, append(opts, extra...)...)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	if cfg.Issuer == "" {
		cfg.Issuer = oidcIssuer
	}
	p := oauth2provider.New(cfg)
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	require.NoError(t, p.OnInit(context.Background(), eng))
	// The engine only declares settings for plugins registered with it.
	require.NoError(t, p.DeclareSettings(eng.Settings()))

	ctx := context.Background()
	require.NoError(t, st.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: publicID, Name: "Public", FirstParty: true,
		RedirectURIs: []string{registeredURI}, Scopes: []string{"openid", "profile", "email"},
		GrantTypes: []string{"authorization_code"}, Public: true, TokenEndpointAuthMethod: "none",
	}))
	require.NoError(t, st.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: noOpenIDID, Name: "No OpenID", FirstParty: true,
		RedirectURIs: []string{registeredURI}, Scopes: []string{"profile", "email"},
		GrantTypes: []string{"authorization_code"}, Public: true, TokenEndpointAuthMethod: "none",
	}))
	require.NoError(t, st.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: oidcDeviceID, Name: "Device", FirstParty: true,
		Scopes: []string{"openid", "email"}, GrantTypes: []string{"urn:ietf:params:oauth:grant-type:device_code"},
		Public: true, TokenEndpointAuthMethod: "none",
	}))

	// Written straight to the store: SignUp would hash a password at the
	// production bcrypt cost, which under -race costs about two seconds per
	// fixture and nothing here signs in with a password.
	now := time.Now()
	u := &user.User{
		ID: id.NewUserID(), AppID: appID, Email: "oidc@example.com", FirstName: "Ada", LastName: "Lovelace",
		CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, eng.Store().CreateUser(ctx, u))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return &oidcFixture{p: p, eng: eng, st: st, mux: mux, appID: appID, user: u}
}

// codeGrant runs authorize then the code exchange for clientID and returns
// the raw token response. A non-empty nonce is sent with the authorize call.
func (f *oidcFixture) codeGrant(t *testing.T, clientID, scope, nonce string, loggedIn time.Time) *httptest.ResponseRecorder {
	t.Helper()
	q := baseAuthorizeQuery(clientID)
	if scope != "" {
		q.Set("scope", scope)
	}
	if nonce != "" {
		q.Set("nonce", nonce)
	}
	q.Set("code_challenge", s256(oidcVerifier))
	q.Set("code_challenge_method", "S256")
	code := codeFrom(t, authorizeWith(sessionCtx(f.user.ID, f.appID, loggedIn), f.mux, q))
	return postToken(t, f.mux, map[string]string{
		"grant_type": "authorization_code", "code": code, "redirect_uri": registeredURI,
		"client_id": clientID, "code_verifier": oidcVerifier,
	})
}

func decodeTokenBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

// verifyIDToken checks the id_token's signature against the published JWKS
// and returns its header and claims.
func verifyIDToken(t *testing.T, mux forge.Router, raw string) (map[string]any, jwt.MapClaims) {
	t.Helper()
	rec := getPath(t, mux, "/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var doc struct {
		Keys []json.RawMessage `json:"keys"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))

	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(tok *jwt.Token) (any, error) {
		for _, k := range doc.Keys {
			pub, j, err := jwkutil.ParseJSON(k)
			if err != nil {
				return nil, err
			}
			if j.KID == tok.Header["kid"] {
				return pub, nil
			}
		}
		return nil, assert.AnError
	}, jwt.WithValidMethods([]string{"RS256", "ES256"}), jwt.WithIssuer(oidcIssuer))
	require.NoError(t, err)
	require.True(t, tok.Valid)
	return tok.Header, claims
}

func sha256AtHash(accessToken string) string {
	sum := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

func TestCodeGrant_IssuesIDTokenForOpenID(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	loggedIn := time.Now().Add(-20 * time.Minute).Truncate(time.Second)

	body := decodeTokenBody(t, f.codeGrant(t, publicID, "openid profile email", "n-0S6_WzA2Mj", loggedIn))
	raw, ok := body["id_token"].(string)
	require.True(t, ok, "the response carries an id_token: %v", body)

	header, claims := verifyIDToken(t, f.mux, raw)
	assert.Equal(t, "JWT", header["typ"])
	assert.Equal(t, "RS256", header["alg"])
	assert.NotEmpty(t, header["kid"])

	assert.Equal(t, oidcIssuer, claims["iss"])
	assert.Equal(t, f.user.ID.String(), claims["sub"])
	assert.Equal(t, []any{publicID}, claims["aud"])
	assert.Equal(t, publicID, claims["azp"])
	assert.Equal(t, "n-0S6_WzA2Mj", claims["nonce"])
	assert.InDelta(t, float64(loggedIn.Unix()), claims["auth_time"], 0)
	assert.Equal(t, sha256AtHash(body["access_token"].(string)), claims["at_hash"])
	iat, exp := claims["iat"].(float64), claims["exp"].(float64)
	assert.InDelta(t, 3600, exp-iat, 0)
	assert.WithinDuration(t, time.Now(), time.Unix(int64(iat), 0), 10*time.Second)

	// The granted scopes decide the standard claims, booleans included.
	assert.Equal(t, "oidc@example.com", claims["email"])
	assert.Equal(t, false, claims["email_verified"])
	assert.Equal(t, "Ada", claims["given_name"])
	assert.NotContains(t, claims, "phone_number_verified", "phone was not granted")
}

func TestCodeGrant_CountsIDToken(t *testing.T) {
	reg, scrape := telemetrytest.Registry(t)
	f := newOIDCFixture(t, oauth2provider.Config{}, true, authsome.WithMetricsSink(reg))

	decodeTokenBody(t, f.codeGrant(t, publicID, "openid profile", "", time.Time{}))
	decodeTokenBody(t, f.codeGrant(t, noOpenIDID, "profile", "", time.Time{}))

	assert.Equal(t, []string{
		`forge_authsome_oauth2_id_tokens_issued_total{grant_type="authorization_code",issuer="deployment"}`,
	}, telemetrytest.Series(scrape(), "forge_authsome_oauth2_id_tokens_issued_"))
	// The first key a scope gets is not a rotation.
	assert.Empty(t, telemetrytest.Series(scrape(), "forge_authsome_oauth2_signing_key_"))
	assert.Empty(t, telemetrytest.Series(scrape(), "forge_authsome_oauth2_id_token_failures_"))
}

func TestCodeGrant_NoOpenIDNoIDToken(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	body := decodeTokenBody(t, f.codeGrant(t, publicID, "profile email", "n-1", time.Now()))
	assert.NotContains(t, body, "id_token")
	assert.NotEmpty(t, body["access_token"])
}

// Review Focus 2: a client whose registration leaves out openid never gets an
// id_token. Asking for openid is refused at /authorize, and the default scope
// set is the registered one, so nothing downstream can grant it.
func TestCodeGrant_ClientWithoutOpenIDScope(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)

	q := baseAuthorizeQuery(noOpenIDID)
	q.Set("scope", "openid profile")
	q.Set("code_challenge", s256(oidcVerifier))
	q.Set("code_challenge_method", "S256")
	refused, _ := redirectedError(t, authorizeWith(sessionCtx(f.user.ID, f.appID, time.Now()), f.mux, q))
	assert.Equal(t, "invalid_scope", refused, "openid is not registered for this client")

	body := decodeTokenBody(t, f.codeGrant(t, noOpenIDID, "", "n-2", time.Now()))
	assert.Equal(t, "profile email", body["scope"])
	assert.NotContains(t, body, "id_token")
}

func TestRefreshGrant_IDTokenOmitsNonceAndAuthTime(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	first := decodeTokenBody(t, f.codeGrant(t, publicID, "openid email", "n-refresh", time.Now().Add(-time.Hour)))
	require.NotEmpty(t, first["refresh_token"])

	rotated := decodeTokenBody(t, postToken(t, f.mux, map[string]string{
		"grant_type": "refresh_token", "refresh_token": first["refresh_token"].(string), "client_id": publicID,
	}))
	raw, ok := rotated["id_token"].(string)
	require.True(t, ok, "a refresh of an openid grant carries an id_token: %v", rotated)

	_, claims := verifyIDToken(t, f.mux, raw)
	assert.Equal(t, f.user.ID.String(), claims["sub"])
	assert.Equal(t, []any{publicID}, claims["aud"])
	assert.Equal(t, sha256AtHash(rotated["access_token"].(string)), claims["at_hash"])
	assert.Equal(t, "oidc@example.com", claims["email"])
	assert.NotContains(t, claims, "nonce")
	assert.NotContains(t, claims, "auth_time")
}

func TestRefreshGrant_NoOpenIDNoIDToken(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	first := decodeTokenBody(t, f.codeGrant(t, publicID, "profile", "", time.Now()))
	rotated := decodeTokenBody(t, postToken(t, f.mux, map[string]string{
		"grant_type": "refresh_token", "refresh_token": first["refresh_token"].(string), "client_id": publicID,
	}))
	assert.NotContains(t, rotated, "id_token")
}

func TestDeviceGrant_IssuesIDTokenWithAuthTime(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	auth := deviceAuthorize(t, f.mux, oidcDeviceID)
	loggedIn := time.Now().Add(-2 * time.Hour).Truncate(time.Second)

	// Stand in for the user approving at the verification URI after signing
	// in at loggedIn.
	dc, err := f.st.GetDeviceCodeByDeviceCode(context.Background(), auth.DeviceCode)
	require.NoError(t, err)
	dc.Status = oauth2provider.DeviceCodeStatusAuthorized
	dc.UserID = f.user.ID
	dc.AuthTime = loggedIn
	require.NoError(t, f.st.UpdateDeviceCode(context.Background(), dc))

	body := decodeTokenBody(t, pollDeviceToken(t, f.mux, map[string]string{
		"device_code": auth.DeviceCode, "client_id": oidcDeviceID,
	}))
	raw, ok := body["id_token"].(string)
	require.True(t, ok, "the device grant carries an id_token: %v", body)

	_, claims := verifyIDToken(t, f.mux, raw)
	assert.Equal(t, f.user.ID.String(), claims["sub"])
	assert.Equal(t, []any{oidcDeviceID}, claims["aud"])
	assert.InDelta(t, float64(loggedIn.Unix()), claims["auth_time"], 0)
	assert.Equal(t, sha256AtHash(body["access_token"].(string)), claims["at_hash"])
	assert.NotContains(t, claims, "nonce")
}

// With no operator key and no encryptor no key can sign. The grant fails as
// server_error before the session is stored, so no access token is left
// behind for a client that asked for an id_token and got none.
func TestCodeGrant_NoKeyIsServerErrorAndNoAccessToken(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, false)

	rec := f.codeGrant(t, publicID, "openid profile", "n-3", time.Now())
	// The handler answers server_error. Forge masks every 5xx body to its
	// status text, so the status is what a client can rely on.
	require.Equal(t, http.StatusInternalServerError, rec.Code, "body: %s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "access_token")

	sessions, err := f.eng.Store().ListUserSessions(context.Background(), f.user.ID)
	require.NoError(t, err)
	for _, s := range sessions {
		assert.Empty(t, s.ClientID, "no session may be stored for the client: %+v", s)
	}

	// A client that didn't ask for openid is unaffected by the missing key.
	decodeTokenBody(t, f.codeGrant(t, publicID, "profile", "", time.Now()))
}

// Review Focus 5: an EC P-256 operator key signs ES256, and at_hash still
// uses SHA-256 because the hash follows the algorithm's own.
func TestCodeGrant_ECKeyHeaderAndAtHash(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(k)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})

	f := newOIDCFixture(t, oauth2provider.Config{IDTokenSigningKey: keyPEM, IDTokenSigningKeyID: "ec-1"}, false)
	body := decodeTokenBody(t, f.codeGrant(t, publicID, "openid", "n-ec", time.Now()))
	raw, ok := body["id_token"].(string)
	require.True(t, ok, "the response carries an id_token: %v", body)

	header, claims := verifyIDToken(t, f.mux, raw)
	assert.Equal(t, "ES256", header["alg"])
	assert.Equal(t, "ec-1", header["kid"])
	assert.Equal(t, "JWT", header["typ"])
	assert.Equal(t, sha256AtHash(body["access_token"].(string)), claims["at_hash"])

	// A JWS ES256 signature is the raw 64-byte r||s, not ASN.1 DER.
	parts := bytes.Split([]byte(raw), []byte("."))
	require.Len(t, parts, 3)
	sig, err := base64.RawURLEncoding.DecodeString(string(parts[2]))
	require.NoError(t, err)
	assert.Len(t, sig, 64)
}

// seedProviderSession stores a session the provider issued to clientID with
// the given scopes, as the code grant would have.
func seedProviderSession(t *testing.T, f *oidcFixture, appID id.AppID, clientID string, scopes ...string) *session.Session {
	t.Helper()
	sess, err := account.NewSession(appID, f.user.ID, account.SessionConfig{
		TokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour,
	})
	require.NoError(t, err)
	sess.ClientID = clientID
	sess.Scopes = scopes
	require.NoError(t, f.eng.Store().CreateSession(context.Background(), sess))
	return sess
}

// The refresh grant spends the refresh token when it rotates, so it checks
// that an id_token can be signed first. With no key nothing can sign: the
// grant fails as server_error and the session is left as it was.
func TestRefreshGrant_NoKeyRefusesWithoutRotating(t *testing.T) {
	// No encryptor and no operator key. The provider session is seeded
	// directly, since the code grant itself would refuse.
	f := newOIDCFixture(t, oauth2provider.Config{}, false)
	ctx := context.Background()
	sess := seedProviderSession(t, f, f.appID, publicID, "openid")

	rec := postToken(t, f.mux, map[string]string{
		"grant_type": "refresh_token", "refresh_token": sess.RefreshToken, "client_id": publicID,
	})
	require.Equal(t, http.StatusInternalServerError, rec.Code, "body: %s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "access_token")

	got, err := f.eng.Store().GetSessionByRefreshToken(ctx, sess.RefreshToken)
	require.NoError(t, err, "the refresh token was not spent")
	assert.Equal(t, sess.ID, got.ID)
}

// A settings or store failure while resolving the issuer must not cost the
// user their session: the refresh answers 500 without rotating, and the
// same refresh token works once the failure clears.
func TestRefreshGrant_IssuerFailureKeepsRefreshToken(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	ctx := context.Background()

	// An app that opted into its own issuer but whose row can't be loaded:
	// issuerFor refuses to guess and fails.
	ghost := id.NewAppID()
	const ghostClient = "ghost-app-client"
	require.NoError(t, f.st.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: ghost, ClientID: ghostClient, Name: "Ghost", FirstParty: true,
		RedirectURIs: []string{registeredURI}, Scopes: []string{"openid"},
		GrantTypes: []string{"authorization_code"}, Public: true, TokenEndpointAuthMethod: "none",
	}))
	setAppIssuer(t, f, ghost, true)
	sess := seedProviderSession(t, f, f.appID, ghostClient, "openid")
	refresh := map[string]string{
		"grant_type": "refresh_token", "refresh_token": sess.RefreshToken, "client_id": ghostClient,
	}

	rec := postToken(t, f.mux, refresh)
	require.Equal(t, http.StatusInternalServerError, rec.Code, "body: %s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "access_token")
	got, err := f.eng.Store().GetSessionByRefreshToken(ctx, sess.RefreshToken)
	require.NoError(t, err, "the refresh token was not spent")
	assert.Equal(t, sess.ID, got.ID)

	// The failure clears and the client retries with the token it holds.
	setAppIssuer(t, f, ghost, false)
	body := decodeTokenBody(t, postToken(t, f.mux, refresh))
	raw, ok := body["id_token"].(string)
	require.True(t, ok, "the retried refresh carries an id_token: %v", body)
	_, claims := verifyIDToken(t, f.mux, raw)
	assert.Equal(t, f.user.ID.String(), claims["sub"])
	assert.Equal(t, sha256AtHash(body["access_token"].(string)), claims["at_hash"])
}
