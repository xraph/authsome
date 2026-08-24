package setjwt

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer = "https://org.okta.com"
	testAud    = "https://authsome.example/ssf"
	testKID    = "kid-1"
)

var testNow = time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

type staticKeys struct {
	key   crypto.PublicKey
	err   error
	calls *int // when set, incremented on every Key call so tests can prove it was never invoked
}

func (s staticKeys) Key(_ context.Context, _ string) (crypto.PublicKey, error) {
	if s.calls != nil {
		*s.calls++
	}
	return s.key, s.err
}

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return k
}

type claimOverride func(jwt.MapClaims)

func signSET(t *testing.T, key *rsa.PrivateKey, typ string, method jwt.SigningMethod,
	secret any, overrides ...claimOverride) []byte {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": testIssuer,
		"aud": testAud,
		"jti": "jti-1",
		"iat": testNow.Unix(),
		"events": map[string]any{
			"https://schemas.openid.net/secevent/caep/event-type/session-revoked": map[string]any{
				"subject": map[string]any{"format": "opaque", "id": "u1"},
			},
		},
	}
	for _, o := range overrides {
		o(claims)
	}
	tok := jwt.NewWithClaims(method, claims)
	tok.Header["typ"] = typ
	tok.Header["kid"] = testKID
	if secret == nil {
		secret = key
	}
	s, err := tok.SignedString(secret)
	require.NoError(t, err)
	return []byte(s)
}

func baseOpts(key *rsa.PrivateKey) Options {
	return Options{
		Issuer:    testIssuer,
		Audience:  testAud,
		Keys:      staticKeys{key: key.Public()},
		Now:       func() time.Time { return testNow },
		MaxAge:    24 * time.Hour,
		ClockSkew: 5 * time.Minute,
		MaxEvents: 10,
	}
}

func TestValidate_Accepts(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil)

	tok, err := Validate(context.Background(), raw, baseOpts(key))
	require.NoError(t, err)
	assert.Equal(t, testIssuer, tok.Issuer)
	assert.Equal(t, "jti-1", tok.JTI)
	assert.Len(t, tok.Events, 1)
}

func TestHeader_ReadsKidAndAlg(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil)

	kid, alg, typ, err := Header(raw)
	require.NoError(t, err)
	assert.Equal(t, testKID, kid)
	assert.Equal(t, "RS256", alg)
	assert.Equal(t, "secevent+jwt", typ)
}

// alg:none is the oldest JWT attack there is. It must be rejected by the
// allow-list before the key resolver is ever consulted: a real KeyResolver
// does network fetches and cache lookups keyed on an attacker-controlled kid
// header, and a disallowed algorithm should never trigger that work.
func TestValidate_RejectsAlgNone(t *testing.T) {
	key := newTestKey(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"iss": testIssuer, "aud": testAud, "jti": "j", "iat": testNow.Unix(),
		"events": map[string]any{"x": map[string]any{}},
	})
	tok.Header["typ"] = "secevent+jwt"
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	calls := 0
	opts := baseOpts(key)
	opts.Keys = staticKeys{key: key.Public(), calls: &calls}

	_, err = Validate(context.Background(), []byte(s), opts)
	require.ErrorIs(t, err, ErrInvalidKey)
	assert.Equal(t, 0, calls, "alg allow-list must reject alg:none before any key resolution")
}

// Algorithm confusion: sign with HMAC using the issuer's PUBLIC key as the
// shared secret. If we accepted HS*, this forgery would verify. Like
// alg:none, this must be rejected before the key resolver runs.
func TestValidate_RejectsHMACConfusion(t *testing.T) {
	key := newTestKey(t)
	pubDER, err := json.Marshal(testKID) // any attacker-known bytes work
	require.NoError(t, err)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodHS256, pubDER)

	calls := 0
	opts := baseOpts(key)
	opts.Keys = staticKeys{key: key.Public(), calls: &calls}

	_, err = Validate(context.Background(), raw, opts)
	require.ErrorIs(t, err, ErrInvalidKey)
	assert.Equal(t, 0, calls, "alg allow-list must reject HMAC before any key resolution")
}

func TestValidate_RejectsWrongTyp(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "JWT", jwt.SigningMethodRS256, nil)

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsWrongIssuer(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["iss"] = "https://evil.example" })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidIssuer)
}

func TestValidate_RejectsWrongAudience(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["aud"] = "https://someone-else.example" })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidAudience)
}

func TestValidate_AcceptsAudienceArray(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["aud"] = []string{"https://other", testAud} })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.NoError(t, err)
}

func TestValidate_RejectsStaleIAT(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["iat"] = testNow.Add(-25 * time.Hour).Unix() })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsFutureIAT(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["iat"] = testNow.Add(10 * time.Minute).Unix() })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsMissingJTI(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { delete(c, "jti") })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsOverlongJTI(t *testing.T) {
	key := newTestKey(t)
	long := make([]byte, MaxJTILength+1)
	for i := range long {
		long[i] = 'a'
	}
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["jti"] = string(long) })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsEmptyEvents(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["events"] = map[string]any{} })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsTooManyEvents(t *testing.T) {
	key := newTestKey(t)
	many := map[string]any{}
	for i := 0; i < 25; i++ {
		many["https://example.com/e"+string(rune('a'+i))] = map[string]any{}
	}
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["events"] = many })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsUnknownKid(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil)

	opts := baseOpts(key)
	opts.Keys = staticKeys{err: assertKeyMissError{}}

	_, err := Validate(context.Background(), raw, opts)
	require.ErrorIs(t, err, ErrInvalidKey)
}

// Signed by a key that is not the one the resolver hands back.
func TestValidate_RejectsWrongSigningKey(t *testing.T) {
	signer := newTestKey(t)
	other := newTestKey(t)
	raw := signSET(t, signer, "secevent+jwt", jwt.SigningMethodRS256, nil)

	opts := baseOpts(signer)
	opts.Keys = staticKeys{key: other.Public()}

	_, err := Validate(context.Background(), raw, opts)
	require.ErrorIs(t, err, ErrInvalidKey)
}

// events must be a JSON object keyed by event type URI. A malformed body
// (wrong JSON shape) is a caller mistake, not a key problem, so it must map
// to invalid_request rather than invalid_key.
func TestValidate_RejectsEventsAsArray(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["events"] = []any{"not-an-object"} })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Equal(t, "invalid_request", ErrCode(err))
}

// iat must be a JSON number. A string in its place is malformed input, not a
// key problem, so it must map to invalid_request rather than invalid_key.
func TestValidate_RejectsIATAsString(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["iat"] = "not-a-number" })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
	assert.Equal(t, "invalid_request", ErrCode(err))
}

type assertKeyMissError struct{}

func (assertKeyMissError) Error() string { return "no such kid" }

func TestErrCode(t *testing.T) {
	assert.Equal(t, "invalid_request", ErrCode(ErrInvalidRequest))
	assert.Equal(t, "invalid_key", ErrCode(ErrInvalidKey))
	assert.Equal(t, "invalid_issuer", ErrCode(ErrInvalidIssuer))
	assert.Equal(t, "invalid_audience", ErrCode(ErrInvalidAudience))
	assert.Equal(t, "invalid_request", ErrCode(assertKeyMissError{}))
}
