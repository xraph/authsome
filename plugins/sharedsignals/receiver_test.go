package sharedsignals

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

type receiverFixture struct {
	plugin    *Plugin
	stream    *InboundStream
	revoker   *recordingRevoker
	key       *rsa.PrivateKey
	jwksSrv   *httptest.Server
	pushPath  string
	pushToken string
	userID    id.UserID
}

const (
	fixtureIssuer = "https://org.okta.com"
	fixtureAud    = "https://authsome.example/ssf"
	fixtureKID    = "kid-1"
	fixtureSub    = "okta-user-1"
)

func newReceiverFixture(t *testing.T) *receiverFixture {
	t.Helper()
	ctx := context.Background()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// A fake transmitter serving its own JWKS, which is the only way to test
	// the verification path the way it actually runs.
	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":%q,"n":%q,"e":%q}]}`,
			fixtureKID, n, e)
	}))
	t.Cleanup(jwksSrv.Close)

	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	authStore := memory.New()
	require.NoError(t, authStore.CreateSession(ctx, &session.Session{
		ID: id.NewSessionID(), AppID: appID, EnvID: envID, UserID: userID,
		Token: "tok-1", ExpiresAt: time.Now().Add(time.Hour),
	}))

	rev := &recordingRevoker{}
	p := New(Config{Audience: fixtureAud})
	p.store = NewMemoryStore()
	p.authStore = authStore
	p.revoker = rev
	require.NoError(t, p.OnInit(ctx, stubEngine{}))
	// OnInit installs a memory store and a jwks client; keep our store and
	// revoker, but replace the jwks client. The production default requires
	// https on a public address (see jwksclient.ValidateURI), which is
	// exactly right for a real transmitter and exactly wrong for a fake one
	// running on an httptest loopback server, so tests need the same
	// permissive override jwksclient's own test suite uses.
	p.store = NewMemoryStore()
	p.authStore = authStore
	p.revoker = rev
	p.jwks = jwksclient.New(jwksclient.Options{
		HTTPClient:  jwksSrv.Client(),
		ValidateURI: func(string) error { return nil },
	})

	pushPath, pushPathHash, err := NewPushSecret()
	require.NoError(t, err)
	pushToken, pushTokenHash, err := NewPushSecret()
	require.NoError(t, err)

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID, Name: "okta",
		Issuer: fixtureIssuer, Audience: fixtureAud, JWKSURI: jwksSrv.URL,
		PushPathHash: pushPathHash, PushTokenHash: pushTokenHash,
		AllowedEventTypes: []string{
			caep.EventSessionRevoked, caep.EventCredentialChange, caep.EventVerification,
		},
		AllowedSubjectFormats: []string{caep.FormatIssSub},
		EnforcementMode:       EnforcementEnforce,
		Status:                StatusEnabled,
		MaxActionsPerHour:     100,
	}
	require.NoError(t, p.store.CreateInboundStream(ctx, stream))
	require.NoError(t, p.store.UpsertSubjectLink(ctx, &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
		Issuer: fixtureIssuer, Subject: fixtureSub, UserID: userID, Source: SourceSSO,
	}))

	return &receiverFixture{
		plugin: p, stream: stream, revoker: rev, key: key, jwksSrv: jwksSrv,
		pushPath: pushPath, pushToken: pushToken, userID: userID,
	}
}

// signSET builds a SET the fixture's fake transmitter would send.
func (f *receiverFixture) signSET(t *testing.T, mutate func(jwt.MapClaims)) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": fixtureIssuer,
		"aud": fixtureAud,
		"jti": "jti-" + id.NewSSFEventID().String(),
		"iat": time.Now().Unix(),
		"events": map[string]any{
			caep.EventSessionRevoked: map[string]any{
				// Okta ships "subject", not "sub_id".
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub,
				},
				"reason_admin": map[string]any{"en": "Account compromised"},
			},
		},
	}
	if mutate != nil {
		mutate(claims)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["typ"] = "secevent+jwt"
	tok.Header["kid"] = fixtureKID
	signed, err := tok.SignedString(f.key)
	require.NoError(t, err)
	return signed
}

// post drives the handler through the plugin's own router so the route
// pattern and the parameter binding are exercised, not bypassed.
func (f *receiverFixture) post(t *testing.T, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/v1/ssf/streams/"+path+"/events", stringReader(body))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.plugin.servePushForTest(rec, req, path)
	return rec
}

func errBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var out map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func TestPush_ValidSETRevokesSessions(t *testing.T) {
	f := newReceiverFixture(t)
	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, rec.Body.String(), "a 202 carries no body")
	assert.Len(t, f.revoker.revoked, 1)
}

func TestPush_UnknownPathIs404(t *testing.T) {
	f := newReceiverFixture(t)
	rec := f.post(t, "not-a-real-path", f.pushToken, f.signSET(t, nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, f.revoker.revoked)
}

func TestPush_WrongTokenIs401(t *testing.T) {
	f := newReceiverFixture(t)
	rec := f.post(t, f.pushPath, "wrong-token", f.signSET(t, nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "authentication_failed", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

func TestPush_MissingTokenIs401(t *testing.T) {
	f := newReceiverFixture(t)
	rec := f.post(t, f.pushPath, "", f.signSET(t, nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPush_PausedStreamIs403(t *testing.T) {
	f := newReceiverFixture(t)
	f.stream.Status = StatusPaused
	require.NoError(t, f.plugin.store.UpdateInboundStream(context.Background(), f.stream))

	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, "access_denied", errBody(t, rec)["err"])
}

func TestPush_WrongIssuerIs400(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) { c["iss"] = "https://evil.example" })
	rec := f.post(t, f.pushPath, f.pushToken, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_issuer", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

func TestPush_WrongAudienceIs400(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) { c["aud"] = "https://elsewhere" })
	rec := f.post(t, f.pushPath, f.pushToken, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_audience", errBody(t, rec)["err"])
}

func TestPush_UnsignedSETIs400(t *testing.T) {
	f := newReceiverFixture(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"iss": fixtureIssuer, "aud": fixtureAud, "jti": "j", "iat": time.Now().Unix(),
		"events": map[string]any{caep.EventSessionRevoked: map[string]any{
			"subject": map[string]any{"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub},
		}},
	})
	tok.Header["typ"] = "secevent+jwt"
	body, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	rec := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_key", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

// A replayed SET must be accepted and ignored. Answering with an error would
// make the transmitter retry forever.
func TestPush_ReplayedJTIIsAcceptedOnceOnly(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, nil)

	first := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, first.Code)
	assert.Len(t, f.revoker.revoked, 1)

	second := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, second.Code)
	assert.Len(t, f.revoker.revoked, 1, "a replay must not revoke a second time")
}

// A subject we cannot place returns 202. An error would tell the transmitter
// which of its users have accounts here.
func TestPush_UnknownSubjectIs202AndDoesNothing(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) {
		c["events"] = map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": "nobody-here",
				},
			},
		}
	})

	rec := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, f.revoker.revoked)
}

// An event type the stream did not subscribe to is recorded and dropped.
func TestPush_EventTypeNotAllowedIsIgnored(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) {
		c["events"] = map[string]any{
			caep.EventDeviceComplianceChange: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub,
				},
				"current_status": "not-compliant", "previous_status": "compliant",
			},
		}
	})

	rec := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, f.revoker.revoked)
}

func TestPush_OversizedBodyIs400(t *testing.T) {
	f := newReceiverFixture(t)
	huge := make([]byte, f.plugin.config.MaxBodyBytes+1024)
	for i := range huge {
		huge[i] = 'A'
	}
	rec := f.post(t, f.pushPath, f.pushToken, string(huge))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// The whole point of pinning keys per stream: a genuine SET aimed at one
// tenant must not work against another tenant's URL.
func TestPush_CrossStreamReplayFails(t *testing.T) {
	ctx := context.Background()
	f := newReceiverFixture(t)

	otherPath, otherPathHash, err := NewPushSecret()
	require.NoError(t, err)
	otherToken, otherTokenHash, err := NewPushSecret()
	require.NoError(t, err)
	require.NoError(t, f.plugin.store.CreateInboundStream(ctx, &InboundStream{
		ID: id.NewSSFStreamID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Issuer: "https://other-idp.example", Audience: fixtureAud,
		JWKSURI:      f.jwksSrv.URL,
		PushPathHash: otherPathHash, PushTokenHash: otherTokenHash,
		AllowedEventTypes:     []string{caep.EventSessionRevoked},
		AllowedSubjectFormats: []string{caep.FormatIssSub},
		EnforcementMode:       EnforcementEnforce, Status: StatusEnabled,
		MaxActionsPerHour: 100,
	}))

	rec := f.post(t, otherPath, otherToken, f.signSET(t, nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_issuer", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

func TestHashSecret_IsStableAndNotThePlaintext(t *testing.T) {
	raw, hash, err := NewPushSecret()
	require.NoError(t, err)
	assert.NotEqual(t, raw, hash)
	assert.Equal(t, hash, HashSecret(raw))
	assert.NotEqual(t, hash, HashSecret(raw+"x"))
}

func stringReader(s string) *strings.Reader { return strings.NewReader(s) }
