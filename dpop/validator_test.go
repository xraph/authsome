package dpop_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/dpop"
)

const testURL = "https://auth.example.com/v1/oauth/token"

// fixedClock lets iat boundary cases be exact rather than approximate.
func fixedClock(at time.Time) func() time.Time { return func() time.Time { return at } }

func newTestValidator(now time.Time) *dpop.Validator {
	return dpop.NewValidator(dpop.Config{
		Replay: dpop.NewMemoryReplayCache(256),
		Now:    fixedClock(now),
	})
}

func TestValidate_HappyPath(t *testing.T) {
	now := time.Now()
	key := testKey(t)
	claims := validClaims()
	claims["iat"] = now.Unix()

	p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
	require.NoError(t, err)

	v := newTestValidator(now)
	err = v.Validate(context.Background(), p, dpop.Expectation{Method: "POST", URL: testURL})
	assert.NoError(t, err)
}

func TestValidate_MethodAndURI(t *testing.T) {
	now := time.Now()
	key := testKey(t)
	claims := validClaims()
	claims["iat"] = now.Unix()
	p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
	require.NoError(t, err)

	v := newTestValidator(now)
	ctx := context.Background()

	assert.ErrorIs(t,
		v.Validate(ctx, p, dpop.Expectation{Method: "GET", URL: testURL}),
		dpop.ErrMethodMismatch)
	assert.ErrorIs(t,
		v.Validate(ctx, p, dpop.Expectation{Method: "POST", URL: "https://auth.example.com/v1/oauth/revoke"}),
		dpop.ErrURIMismatch)
}

// TestValidate_URIIgnoresQueryAndFragment: RFC 9449 compares htu without the
// query or fragment, so a proof stays valid across differing query strings.
func TestValidate_URIIgnoresQueryAndFragment(t *testing.T) {
	now := time.Now()
	key := testKey(t)
	claims := validClaims()
	claims["iat"] = now.Unix()
	p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
	require.NoError(t, err)

	v := newTestValidator(now)
	err = v.Validate(context.Background(), p, dpop.Expectation{
		Method: "POST",
		URL:    testURL + "?scope=openid#frag",
	})
	assert.NoError(t, err)
}

// TestValidate_IatBoundaries pins the window exactly. Off-by-one here is
// invisible in casual testing and produces intermittent failures in the field.
func TestValidate_IatBoundaries(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	cases := []struct {
		name    string
		offset  time.Duration
		wantErr bool
	}{
		{"59s in the past", -59 * time.Second, false},
		{"exactly 60s in the past", -60 * time.Second, false},
		{"61s in the past", -61 * time.Second, true},
		{"29s ahead", 29 * time.Second, false},
		{"exactly 30s ahead", 30 * time.Second, false},
		{"31s ahead", 31 * time.Second, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := testKey(t)
			claims := validClaims()
			claims["iat"] = now.Add(tc.offset).Unix()
			p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
			require.NoError(t, err)

			v := newTestValidator(now)
			err = v.Validate(context.Background(), p, dpop.Expectation{Method: "POST", URL: testURL})
			if tc.wantErr {
				assert.ErrorIs(t, err, dpop.ErrStaleProof)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_ReplayedJTI(t *testing.T) {
	now := time.Now()
	key := testKey(t)
	claims := validClaims()
	claims["iat"] = now.Unix()
	raw := mintProof(t, key, "dpop+jwt", "ES256", claims)

	v := newTestValidator(now)
	ctx := context.Background()
	e := dpop.Expectation{Method: "POST", URL: testURL}

	first, err := dpop.Parse(raw)
	require.NoError(t, err)
	require.NoError(t, v.Validate(ctx, first, e))

	second, err := dpop.Parse(raw)
	require.NoError(t, err)
	assert.ErrorIs(t, v.Validate(ctx, second, e), dpop.ErrReplayed)
}

// TestValidate_FailedProofDoesNotBurnJTI: the jti check runs last so a proof
// rejected earlier never enters the cache. Otherwise an attacker can burn a
// client's jti values with cheap malformed proofs.
func TestValidate_FailedProofDoesNotBurnJTI(t *testing.T) {
	now := time.Now()
	key := testKey(t)
	claims := validClaims()
	claims["iat"] = now.Unix()
	raw := mintProof(t, key, "dpop+jwt", "ES256", claims)

	v := newTestValidator(now)
	ctx := context.Background()

	bad, err := dpop.Parse(raw)
	require.NoError(t, err)
	require.Error(t, v.Validate(ctx, bad, dpop.Expectation{Method: "GET", URL: testURL}))

	good, err := dpop.Parse(raw)
	require.NoError(t, err)
	assert.NoError(t, v.Validate(ctx, good, dpop.Expectation{Method: "POST", URL: testURL}))
}

func TestValidate_ATH(t *testing.T) {
	now := time.Now()
	const token = "at-abc123"
	key := testKey(t)

	t.Run("matching ath passes", func(t *testing.T) {
		claims := validClaims()
		claims["iat"] = now.Unix()
		claims["ath"] = dpop.AccessTokenHash(token)
		p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
		require.NoError(t, err)

		v := newTestValidator(now)
		assert.NoError(t, v.Validate(context.Background(), p, dpop.Expectation{
			Method: "POST", URL: testURL, AccessToken: token,
		}))
	})

	t.Run("missing ath is rejected when a token is presented", func(t *testing.T) {
		claims := validClaims()
		claims["iat"] = now.Unix()
		p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
		require.NoError(t, err)

		v := newTestValidator(now)
		assert.ErrorIs(t, v.Validate(context.Background(), p, dpop.Expectation{
			Method: "POST", URL: testURL, AccessToken: token,
		}), dpop.ErrATHMismatch)
	})

	t.Run("wrong ath is rejected", func(t *testing.T) {
		claims := validClaims()
		claims["iat"] = now.Unix()
		claims["ath"] = dpop.AccessTokenHash("some-other-token")
		p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
		require.NoError(t, err)

		v := newTestValidator(now)
		assert.ErrorIs(t, v.Validate(context.Background(), p, dpop.Expectation{
			Method: "POST", URL: testURL, AccessToken: token,
		}), dpop.ErrATHMismatch)
	})
}

func TestValidate_KeyMismatch(t *testing.T) {
	now := time.Now()
	claims := validClaims()
	claims["iat"] = now.Unix()
	p, err := dpop.Parse(mintProof(t, testKey(t), "dpop+jwt", "ES256", claims))
	require.NoError(t, err)

	v := newTestValidator(now)
	assert.ErrorIs(t, v.Validate(context.Background(), p, dpop.Expectation{
		Method: "POST", URL: testURL, ExpectedJKT: "some-other-thumbprint",
	}), dpop.ErrKeyMismatch)
}
