package dpop_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/dpop"
)

// TestRequestScope_SameProofTwiceOnOneRequest is the unit-level statement of
// what the request scope buys: a proof validated once under a scoped context
// validates again under the same context, because the second look is the same
// request asking about its own proof rather than a replay.
func TestRequestScope_SameProofTwiceOnOneRequest(t *testing.T) {
	now := time.Now()
	key := testKey(t)
	claims := validClaims()
	claims["iat"] = now.Unix()

	p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
	require.NoError(t, err)

	v := newTestValidator(now)
	ctx := dpop.WithRequestScope(context.Background())
	exp := dpop.Expectation{Method: "POST", URL: testURL}

	require.NoError(t, v.Validate(ctx, p, exp))
	assert.NoError(t, v.Validate(ctx, p, exp),
		"the same proof on the same request is not a replay of itself")
}

// TestRequestScope_DoesNotCrossRequests is the property the scope must not
// cost us. Two scoped contexts are two requests, and the replay cache still
// catches the second.
func TestRequestScope_DoesNotCrossRequests(t *testing.T) {
	now := time.Now()
	key := testKey(t)
	claims := validClaims()
	claims["iat"] = now.Unix()

	p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
	require.NoError(t, err)

	v := newTestValidator(now)
	exp := dpop.Expectation{Method: "POST", URL: testURL}

	require.NoError(t, v.Validate(dpop.WithRequestScope(context.Background()), p, exp))
	assert.ErrorIs(t,
		v.Validate(dpop.WithRequestScope(context.Background()), p, exp),
		dpop.ErrReplayed)
}

// TestRequestScope_FailedValidationIsNotRecorded pins the ordering. The scope
// is written after every check has passed, so a proof that failed on one
// expectation cannot be waved through by a later check on the same request.
// Recording before validating would make the second call here succeed.
func TestRequestScope_FailedValidationIsNotRecorded(t *testing.T) {
	now := time.Now()
	key := testKey(t)
	claims := validClaims()
	claims["iat"] = now.Unix()

	p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", claims))
	require.NoError(t, err)

	v := newTestValidator(now)
	ctx := dpop.WithRequestScope(context.Background())

	// Fails on the method, well before the replay step.
	require.ErrorIs(t,
		v.Validate(ctx, p, dpop.Expectation{Method: "GET", URL: testURL}),
		dpop.ErrMethodMismatch)

	// Re-checking the same failure on the same request still fails.
	assert.ErrorIs(t,
		v.Validate(ctx, p, dpop.Expectation{Method: "GET", URL: testURL}),
		dpop.ErrMethodMismatch)
}

// TestRequestScope_OnlyTheSameProofIsExempt: a second, distinct proof on the
// same request goes through the replay cache normally, so the scope cannot be
// used to smuggle a captured proof alongside a fresh one.
func TestRequestScope_OnlyTheSameProofIsExempt(t *testing.T) {
	now := time.Now()
	key := testKey(t)

	first := validClaims()
	first["iat"] = now.Unix()
	first["jti"] = "scope-jti-a"
	pa, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", first))
	require.NoError(t, err)

	second := validClaims()
	second["iat"] = now.Unix()
	second["jti"] = "scope-jti-b"
	pb, err := dpop.Parse(mintProof(t, key, "dpop+jwt", "ES256", second))
	require.NoError(t, err)

	v := newTestValidator(now)
	exp := dpop.Expectation{Method: "POST", URL: testURL}

	// Burn jti b on an earlier request.
	require.NoError(t, v.Validate(context.Background(), pb, exp))

	ctx := dpop.WithRequestScope(context.Background())
	require.NoError(t, v.Validate(ctx, pa, exp))
	assert.ErrorIs(t, v.Validate(ctx, pb, exp), dpop.ErrReplayed,
		"a scope opened by one proof must not exempt a different one")
}

// TestRequestScope_InstallIsIdempotent: nesting the call must keep one record
// rather than shadowing it with an empty one, or two enforcement points that
// each install would stop sharing what they learned.
func TestRequestScope_InstallIsIdempotent(t *testing.T) {
	outer := dpop.WithRequestScope(context.Background())
	assert.Same(t, outer, dpop.WithRequestScope(outer)) //nolint:testifylint // contexts are pointers here and identity is exactly the claim
}
