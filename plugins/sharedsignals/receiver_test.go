package sharedsignals

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
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
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
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

// signMultiEventSET builds a SET carrying two events under one jti: a
// session-revoked bundled with a credential-change, both for fixtureSub.
// RFC 8417 keys the events object by event type URI, so this is exactly the
// shape a real transmitter sends when it wants to report more than one thing
// about the same subject in one delivery.
func (f *receiverFixture) signMultiEventSET(t *testing.T, jti string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": fixtureIssuer,
		"aud": fixtureAud,
		"jti": jti,
		"iat": time.Now().Unix(),
		"events": map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub,
				},
				"reason_admin": map[string]any{"en": "Account compromised"},
			},
			caep.EventCredentialChange: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub,
				},
				"credential_type": "password",
				"change_type":     "update",
			},
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["typ"] = "secevent+jwt"
	tok.Header["kid"] = fixtureKID
	signed, err := tok.SignedString(f.key)
	require.NoError(t, err)
	return signed
}

// countReceivedEvents returns how many ReceivedEvent rows exist for a given
// (stream, jti), reaching directly into MemoryStore's map since this test
// file lives in the same package. It is how the multi-event tests prove
// every event in a SET was actually recorded, not just the SET's HTTP
// response code.
func countReceivedEvents(t *testing.T, s Store, streamID id.SSFStreamID, jti string) int {
	t.Helper()
	mem, ok := s.(*MemoryStore)
	require.True(t, ok, "countReceivedEvents needs the MemoryStore backing this test fixture")
	count := 0
	for _, e := range mem.events {
		if e.StreamID == streamID && e.JTI == jti {
			count++
		}
	}
	return count
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

// The dedupe key used to be (stream_id, jti) alone, so the second event in a
// multi-event SET collided with the row the first one had just inserted --
// on the very first delivery, before any replay ever happened. Which event
// went first was decided by Go's randomized map iteration order, so the
// damage was intermittent: a bundled session-revoked was a coin flip on
// whether it ever actually revoked anything, and the transmitter was told
// 202 regardless. This test runs enough iterations, with a fresh jti each
// time, that iteration order cannot hide a regression back to that key.
func TestPush_MultiEventSETRecordsBothEventsAndRevokesEveryRun(t *testing.T) {
	f := newReceiverFixture(t)

	const runs = 30
	for i := 0; i < runs; i++ {
		jti := fmt.Sprintf("multi-jti-%d", i)
		f.revoker.revoked = nil

		rec := f.post(t, f.pushPath, f.pushToken, f.signMultiEventSET(t, jti))
		require.Equal(t, http.StatusAccepted, rec.Code, "run %d", i)

		assert.Equal(t, 2, countReceivedEvents(t, f.plugin.store, f.stream.ID, jti),
			"run %d: both events in the SET must be recorded, not just whichever the map visited first", i)
		assert.Len(t, f.revoker.revoked, 1,
			"run %d: the bundled session-revoked must revoke exactly once regardless of map iteration order", i)
	}
}

// Replaying a two-event SET must be a no-op the second time -- 202, nothing
// revoked again -- and since every event it carries was already recorded,
// this is also the "a SET whose events are all already seen" case: the
// second post has nothing new to insert at all, not even a partial mix of
// new and already-seen events.
func TestPush_ReplayedMultiEventSETIsAcceptedWithNoSecondRevocation(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signMultiEventSET(t, "multi-replay-jti")

	first := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, first.Code)
	assert.Equal(t, 2, countReceivedEvents(t, f.plugin.store, f.stream.ID, "multi-replay-jti"))
	assert.Len(t, f.revoker.revoked, 1)

	second := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, second.Code, "every event was already seen, so this is a replay, and a replay is a success")
	assert.Equal(t, 2, countReceivedEvents(t, f.plugin.store, f.stream.ID, "multi-replay-jti"),
		"no new rows from the replay")
	assert.Len(t, f.revoker.revoked, 1, "a replay must not revoke a second time")
}

// flakyLinkStore fails GetSubjectLink a fixed number of times before
// behaving normally, simulating a transient store outage (a dropped
// connection, a timeout) that resolves on its own -- as opposed to
// ErrNotFound, which is a real answer, not a failure.
type flakyLinkStore struct {
	*MemoryStore
	failsLeft int
}

func (s *flakyLinkStore) GetSubjectLink(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	if s.failsLeft > 0 {
		s.failsLeft--
		return nil, errors.New("simulated transient store outage")
	}
	return s.MemoryStore.GetSubjectLink(ctx, appID, envID, issuer, subject)
}

// A store error during subject resolution is an infrastructure failure, not
// a policy verdict about the event. Answering 202 (as the code used to)
// would tell the transmitter its delivery succeeded when nothing was ever
// attempted, permanently losing what might be a genuine compromise event
// since M1 ships no dispatcher to retry a merely-pending row. The fix must
// answer with an error AND leave no dedupe row behind, so the transmitter's
// own retry of the identical SET is actually reprocessed rather than read
// back as a replay of a delivery that never happened.
func TestPush_InfrastructureFailureDuringResolutionRetriesInsteadOfReplaying(t *testing.T) {
	f := newReceiverFixture(t)
	mem, ok := f.plugin.store.(*MemoryStore)
	require.True(t, ok)
	f.plugin.store = &flakyLinkStore{MemoryStore: mem, failsLeft: 1}

	body := f.signSET(t, nil)

	jti := extractJTI(t, body)

	first := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusInternalServerError, first.Code)
	assert.Empty(t, first.Body.String(), "an infrastructure failure carries no RFC 8935 body -- none of its codes describe our own store breaking")
	assert.Empty(t, f.revoker.revoked)
	assert.Equal(t, 0, countReceivedEvents(t, mem, f.stream.ID, jti),
		"the dedupe row from the failed attempt must have been undone")

	second := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, second.Code)
	assert.Len(t, f.revoker.revoked, 1, "the retry must actually be processed, not swallowed as a replay")
}

// extractJTI pulls the jti claim back out of a signed SET body, without
// verifying anything -- it only exists so a test can look up the
// ReceivedEvent row it should not find.
func extractJTI(t *testing.T, signedSET string) string {
	t.Helper()
	parser := jwt.NewParser()
	tok, _, err := parser.ParseUnverified(signedSET, jwt.MapClaims{})
	require.NoError(t, err)
	claims, ok := tok.Claims.(jwt.MapClaims)
	require.True(t, ok)
	jti, _ := claims["jti"].(string)
	return jti
}

// A subject format the stream does not allow is a REJECTED subject, not an
// unresolved one -- a distinct code path from TestPush_UnknownSubjectIs202AndDoesNothing,
// which covers a well-formed, allowed subject that simply names nobody we
// know. Both must still answer 202: an error here would let the transmitter
// learn something about our policy configuration from the outside.
func TestPush_RejectedSubjectFormatIs202AndDoesNothing(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) {
		c["events"] = map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "email", "email": "someone@example.com",
				},
			},
		}
	})

	rec := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Empty(t, f.revoker.revoked)
}

func stringReader(s string) *strings.Reader { return strings.NewReader(s) }
