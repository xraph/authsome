package sharedsignals

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store/memory"
)

// This file covers the wiring that the whole feature depends on and that
// nothing else exercised: the SSF verification handshake, what happens when
// our own JWKS fetch fails, what the audit row actually records, and the
// operator's kill switch.

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// signVerificationSET builds the SSF verification event, which carries a
// `state` and no subject at all -- the shape that used to be rejected twice
// over, first by the parser and then by subject resolution.
func (f *receiverFixture) signVerificationSET(t *testing.T, state string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": fixtureIssuer,
		"aud": fixtureAud,
		"jti": "jti-" + id.NewSSFEventID().String(),
		"iat": time.Now().Unix(),
		"events": map[string]any{
			caep.EventVerification: map[string]any{"state": state},
		},
	})
	tok.Header["typ"] = "secevent+jwt"
	tok.Header["kid"] = fixtureKID
	signed, err := tok.SignedString(f.key)
	require.NoError(t, err)
	return signed
}

// storedEvents returns every received-event row the fixture's memory store
// holds. The row is the audit trail, and nothing else in the package reads it
// back.
func (f *receiverFixture) storedEvents(t *testing.T) []*ReceivedEvent {
	t.Helper()
	mem, ok := f.plugin.store.(*MemoryStore)
	require.True(t, ok, "the fixture uses the memory store")
	mem.mu.RLock()
	defer mem.mu.RUnlock()
	out := make([]*ReceivedEvent, 0, len(mem.events))
	for _, e := range mem.events {
		out = append(out, e)
	}
	return out
}

// withSettings gives the fixture's plugin a real settings manager backed by
// the same in-memory store the auth data lives in.
func (f *receiverFixture) withSettings(t *testing.T) *settings.Manager {
	t.Helper()
	mgr := settings.NewManager(memory.New(), log.NewNoopLogger())
	require.NoError(t, f.plugin.DeclareSettings(mgr))
	f.plugin.settingsMgr = mgr
	return mgr
}

// ──────────────────────────────────────────────────
// F2: the SSF verification handshake
// ──────────────────────────────────────────────────

// Two separate blockers used to make this event impossible to process:
// caep.ParseEvent refused any event carrying neither sub_id nor subject, and
// the receiver stopped every event whose subject did not resolve before
// applyEvent was ever reached. A verification event has no subject to
// resolve, so the entire handshake was unreachable.
func TestPush_VerificationEventStampsLastVerifiedAt(t *testing.T) {
	ctx := context.Background()
	f := newReceiverFixture(t)

	f.stream.PendingVerifyState = "state-abc"
	require.NoError(t, f.plugin.store.UpdateInboundStream(ctx, f.stream))

	rec := f.post(t, f.pushPath, f.pushToken, f.signVerificationSET(t, "state-abc"))
	require.Equal(t, http.StatusAccepted, rec.Code, "body: %s", rec.Body.String())

	after, err := f.plugin.store.GetInboundStream(ctx, f.stream.ID)
	require.NoError(t, err)
	require.NotNil(t, after.LastVerifiedAt,
		"a verification event echoing the challenge must mark the stream verified")
	assert.WithinDuration(t, time.Now(), *after.LastVerifiedAt, time.Minute)
	assert.Empty(t, after.PendingVerifyState, "the challenge is spent once it is matched")
	assert.Empty(t, f.revoker.revoked, "a stream-level event touches nobody's sessions")
}

// The challenge is what stops a transmitter from declaring itself verified
// whenever it likes, so a state that does not match ours must change nothing.
func TestPush_VerificationEventWithWrongStateDoesNotStamp(t *testing.T) {
	ctx := context.Background()
	f := newReceiverFixture(t)

	f.stream.PendingVerifyState = "state-abc"
	require.NoError(t, f.plugin.store.UpdateInboundStream(ctx, f.stream))

	rec := f.post(t, f.pushPath, f.pushToken, f.signVerificationSET(t, "state-wrong"))
	// Still a 202: the SET was valid, we simply did not accept the claim it
	// made. Answering an error would leak whether a challenge is outstanding.
	require.Equal(t, http.StatusAccepted, rec.Code)

	after, err := f.plugin.store.GetInboundStream(ctx, f.stream.ID)
	require.NoError(t, err)
	assert.Nil(t, after.LastVerifiedAt,
		"a state that does not match the outstanding challenge must not verify the stream")
	assert.Equal(t, "state-abc", after.PendingVerifyState,
		"a failed match must not consume the outstanding challenge")
}

// A stream with no challenge outstanding cannot be verified by anything.
func TestPush_VerificationEventWithNoPendingChallengeDoesNotStamp(t *testing.T) {
	ctx := context.Background()
	f := newReceiverFixture(t)

	rec := f.post(t, f.pushPath, f.pushToken, f.signVerificationSET(t, "anything"))
	require.Equal(t, http.StatusAccepted, rec.Code)

	after, err := f.plugin.store.GetInboundStream(ctx, f.stream.ID)
	require.NoError(t, err)
	assert.Nil(t, after.LastVerifiedAt)
}

// ──────────────────────────────────────────────────
// F3: a JWKS fetch that fails is our problem, not the token's
// ──────────────────────────────────────────────────

// A 400 tells a well-behaved transmitter to stop retrying. Answering one
// because our own key fetch had a bad five minutes permanently drops whatever
// that SET was carrying, and the thing it is most likely to be carrying is a
// compromise event.
func TestPush_JWKSFetchFailureAnswers5xxSoTheTransmitterRetries(t *testing.T) {
	ctx := context.Background()
	f := newReceiverFixture(t)

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer dead.Close()

	f.stream.JWKSURI = dead.URL
	require.NoError(t, f.plugin.store.UpdateInboundStream(ctx, f.stream))

	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code,
		"an unreachable key set must invite a retry, not tell the transmitter to give up")
	assert.Empty(t, rec.Body.String(),
		"none of the RFC 8935 codes describe our own fetch failing, so send no error object")
	assert.Empty(t, f.revoker.revoked)
}

// The other side of the same coin: a token naming a key the issuer really
// does not publish is permanently wrong, and must keep saying so.
func TestPush_UnknownKidStillAnswers400InvalidKey(t *testing.T) {
	f := newReceiverFixture(t)

	body := f.signSET(t, nil)
	// Re-sign with a kid the fixture's JWKS does not carry.
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": fixtureIssuer, "aud": fixtureAud,
		"jti": "jti-" + id.NewSSFEventID().String(), "iat": time.Now().Unix(),
		"events": map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub,
				},
			},
		},
	})
	tok.Header["typ"] = "secevent+jwt"
	tok.Header["kid"] = "kid-nobody-publishes"
	unknownKid, err := tok.SignedString(f.key)
	require.NoError(t, err)
	require.NotEqual(t, body, unknownKid)

	rec := f.post(t, f.pushPath, f.pushToken, unknownKid)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_key", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

// A signature that does not verify against a key set we loaded perfectly well
// is also permanently wrong.
func TestPush_BadSignatureStillAnswers400InvalidKey(t *testing.T) {
	f := newReceiverFixture(t)

	attacker, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": fixtureIssuer, "aud": fixtureAud,
		"jti": "jti-" + id.NewSSFEventID().String(), "iat": time.Now().Unix(),
		"events": map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub,
				},
			},
		},
	})
	tok.Header["typ"] = "secevent+jwt"
	tok.Header["kid"] = fixtureKID
	forged, err := tok.SignedString(attacker)
	require.NoError(t, err)

	rec := f.post(t, f.pushPath, f.pushToken, forged)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_key", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

// ──────────────────────────────────────────────────
// F4: the breaker bounds signal-only traffic too
// ──────────────────────────────────────────────────

// A signal-only event takes no action, so under the old counter it was free.
// An authentic but hostile transmitter could push risk-level-change at HIGH
// all day, raising every user's score, with the breaker's counter at zero.
func TestCircuitBreaker_CountsSignalOnlyEvents(t *testing.T) {
	ctx := context.Background()
	f := newActionFixture(t)
	f.stream.MaxActionsPerHour = 2
	require.NoError(t, f.plugin.store.UpdateInboundStream(ctx, f.stream))

	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, f.plugin.store.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: f.stream.ID,
			JTI: "risk-" + string(rune('a'+i)), EventType: caep.EventRiskLevelChange,
			// Applied, and no action taken: exactly the rows the old
			// predicate skipped.
			Outcome: OutcomeApplied, ReceivedAt: now,
		}))
	}

	ok, err := f.plugin.checkCircuitBreaker(ctx, f.stream)
	require.NoError(t, err)
	assert.False(t, ok, "signal-only events must count against the breaker")

	after, err := f.plugin.store.GetInboundStream(ctx, f.stream.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, after.Status)
}

// ──────────────────────────────────────────────────
// F5: the audit row carries the subject and the resolved user
// ──────────────────────────────────────────────────

// The receiver deliberately keeps offending values out of its error responses
// on the reasoning that they belong in the audit record instead. That only
// works if the audit record has them, and until now every row stored "".
func TestPush_ReceivedEventRowRecordsTheSubjectAndTheResolvedUser(t *testing.T) {
	f := newReceiverFixture(t)

	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))
	require.Equal(t, http.StatusAccepted, rec.Code)

	rows := f.storedEvents(t)
	require.Len(t, rows, 1)
	row := rows[0]

	assert.Equal(t, OutcomeApplied, row.Outcome)
	assert.Equal(t, f.userID, row.ResolvedUserID,
		"the row must name the user whose sessions were ended")
	require.NotEmpty(t, row.SubjectJSON, "the row must carry the subject the transmitter named")

	var subject map[string]any
	require.NoError(t, json.Unmarshal([]byte(row.SubjectJSON), &subject))
	assert.Equal(t, "iss_sub", subject["format"])
	assert.Equal(t, fixtureSub, subject["sub"])
}

// An event nobody could be resolved from still has to leave a record of who
// the transmitter said it was about -- that is the only place an operator
// debugging a broken integration can look, because the response body tells
// them nothing on purpose.
func TestPush_UnresolvedEventStillRecordsTheSubject(t *testing.T) {
	f := newReceiverFixture(t)
	chron := &recordingChronicle{}
	f.plugin.chronicle = chron

	body := f.signSET(t, func(claims jwt.MapClaims) {
		claims["events"] = map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": "nobody-we-know",
				},
			},
		}
	})
	rec := f.post(t, f.pushPath, f.pushToken, body)
	require.Equal(t, http.StatusAccepted, rec.Code)

	rows := f.storedEvents(t)
	require.Len(t, rows, 1)
	assert.Equal(t, OutcomeUnresolved, rows[0].Outcome)
	assert.Contains(t, rows[0].SubjectJSON, "nobody-we-know")
	assert.True(t, rows[0].ResolvedUserID.IsNil())

	// And the spec's "every accepted and rejected event goes to Chronicle"
	// now actually holds for the outcomes that take no action.
	require.NotEmpty(t, chron.events, "an unresolved event must still be audited")
	assert.Equal(t, "ssf_event_recorded", chron.events[0].Action)
	assert.Equal(t, OutcomeUnresolved, chron.events[0].Metadata["outcome"])
	assert.Contains(t, chron.events[0].Metadata["subject"], "nobody-we-know")
}

// ──────────────────────────────────────────────────
// F6: the kill switch
// ──────────────────────────────────────────────────

// sharedsignals.enabled is declared WithEnforceable(), so an operator who
// turns it off believes they have stopped the receiver. Until now they still
// had a live remote session-kill endpoint.
func TestPush_DisabledSettingStopsTheReceiverActing(t *testing.T) {
	ctx := context.Background()
	f := newReceiverFixture(t)
	mgr := f.withSettings(t)

	require.NoError(t, mgr.Set(ctx, SettingEnabled.Def.Key, json.RawMessage("false"),
		settings.ScopeApp, f.stream.AppID.String(), f.stream.AppID.String(), "", "operator"))

	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code,
		"a temporary operator decision must not make the transmitter give up permanently")
	assert.Equal(t, "300", rec.Header().Get("Retry-After"))
	assert.Empty(t, f.revoker.revoked,
		"the kill switch has to actually stop the endpoint revoking sessions")
	assert.Empty(t, f.storedEvents(t), "a refused delivery must not burn its dedupe row")
}

// With the switch left alone, the same delivery works: the gate is the
// setting, not the settings manager merely existing.
func TestPush_EnabledSettingLeavesTheReceiverWorking(t *testing.T) {
	f := newReceiverFixture(t)
	f.withSettings(t)

	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))
	require.Equal(t, http.StatusAccepted, rec.Code)
	assert.Len(t, f.revoker.revoked, 1)
}

// The other three settings are read where they are used. The TTL is the one
// with an observable effect on a single push.
func TestPush_SignalTTLComesFromTheSettingsManager(t *testing.T) {
	ctx := context.Background()
	f := newReceiverFixture(t)
	mgr := f.withSettings(t)

	require.NoError(t, mgr.Set(ctx, SettingSignalTTLHours.Def.Key, json.RawMessage("1"),
		settings.ScopeApp, f.stream.AppID.String(), f.stream.AppID.String(), "", "operator"))

	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))
	require.Equal(t, http.StatusAccepted, rec.Code)

	signals, err := f.plugin.store.ListActiveSignals(ctx,
		f.stream.AppID, f.stream.EnvID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1)
	assert.WithinDuration(t, time.Now().Add(time.Hour), signals[0].ExpiresAt, time.Minute,
		"the operator's TTL must win over the compiled-in default of 24h")
}

// ──────────────────────────────────────────────────
// F10: a stream with no effective audience is not a stream
// ──────────────────────────────────────────────────

// Config.defaults() sets no Audience, so on a default config audienceFor
// returned "" and setjwt then required every token's aud to contain "".
// It failed closed, which is right, but it failed looking like an IdP problem
// rather than a misconfiguration nobody could see.
func TestCreateStream_RefusesAnEmptyEffectiveAudience(t *testing.T) {
	ctx := context.Background()
	p := New() // no Audience configured
	p.store = NewMemoryStore()

	_, err := p.CreateStream(ctx, id.NewAppID(), id.NewEnvironmentID(), CreateStreamRequest{
		Name: "okta", Issuer: "https://org.okta.com",
		JWKSURI: "https://org.okta.com/oauth2/v1/keys",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")

	// A per-request audience is enough on its own.
	created, err := p.CreateStream(ctx, id.NewAppID(), id.NewEnvironmentID(), CreateStreamRequest{
		Name: "okta", Issuer: "https://org.okta.com",
		JWKSURI:  "https://org.okta.com/oauth2/v1/keys",
		Audience: "https://authsome.example/ssf",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://authsome.example/ssf", created.Stream.Audience)
}
