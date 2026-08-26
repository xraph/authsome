package oauth2test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

func newAuthCode(f Fixture, clientID string) *oauth2provider.AuthorizationCode {
	return &oauth2provider.AuthorizationCode{
		ID:                  id.NewAuthCodeID(),
		Code:                unique("code"),
		ClientID:            clientID,
		UserID:              f.UserID,
		AppID:               f.AppID,
		RedirectURI:         "https://example.test/cb",
		Scopes:              []string{"openid"},
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
		ExpiresAt:           now().Add(10 * time.Minute),
		CreatedAt:           now(),
	}
}

// seedCode creates a client and an unconsumed auth code against it.
func seedCode(t *testing.T, f Fixture) *oauth2provider.AuthorizationCode {
	t.Helper()
	ctx := context.Background()
	c := newClient(f.AppID)
	require.NoError(t, f.Store.CreateClient(ctx, c))
	ac := newAuthCode(f, c.ClientID)
	require.NoError(t, f.Store.CreateAuthCode(ctx, ac))
	return ac
}

func testAuthCodeRoundTrip(t *testing.T, f Fixture) {
	ctx := context.Background()
	ac := seedCode(t, f)

	got, err := f.Store.GetAuthCode(ctx, ac.Code)
	require.NoError(t, err)
	assert.Equal(t, ac.ID, got.ID)
	assert.Equal(t, ac.ClientID, got.ClientID)
	assert.Equal(t, ac.UserID, got.UserID)
	assert.Equal(t, ac.AppID, got.AppID)
	assert.Equal(t, ac.RedirectURI, got.RedirectURI)
	assert.Equal(t, ac.Scopes, got.Scopes)
	// PKCE is the whole point of the code exchange; losing either field
	// silently downgrades the flow.
	assert.Equal(t, ac.CodeChallenge, got.CodeChallenge)
	assert.Equal(t, ac.CodeChallengeMethod, got.CodeChallengeMethod)
	assert.False(t, got.Consumed, "a fresh code must not read back as consumed")
	assert.WithinDuration(t, ac.ExpiresAt, got.ExpiresAt, time.Second,
		"expiry must survive the round trip; a shifted expiry is either a dead code or an immortal one")
}

func testConsumeAuthCodeOnce(t *testing.T, f Fixture) {
	ctx := context.Background()
	ac := seedCode(t, f)

	ok, err := f.Store.ConsumeAuthCode(ctx, ac.Code)
	require.NoError(t, err)
	assert.True(t, ok, "the first consume of a fresh code must report that it won")

	got, err := f.Store.GetAuthCode(ctx, ac.Code)
	require.NoError(t, err)
	assert.True(t, got.Consumed, "consume must persist the consumed flag")
}

func testConsumeAuthCodeReplay(t *testing.T, f Fixture) {
	ctx := context.Background()
	ac := seedCode(t, f)

	ok, err := f.Store.ConsumeAuthCode(ctx, ac.Code)
	require.NoError(t, err)
	require.True(t, ok)

	// A replayed code must lose. Returning true here would let an attacker
	// who captured the code exchange it a second time.
	ok, err = f.Store.ConsumeAuthCode(ctx, ac.Code)
	require.NoError(t, err)
	assert.False(t, ok, "replaying a consumed code must report false")
}

// testConsumeAuthCodeIsAtomic runs the test-and-set concurrently. Exactly one
// caller may win, whatever the backend's locking story. A read-then-write
// implementation passes the sequential replay case and fails this one.
func testConsumeAuthCodeIsAtomic(t *testing.T, f Fixture) {
	ctx := context.Background()
	ac := seedCode(t, f)

	const racers = 8
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		wins    int
		errs    []error
		startCh = make(chan struct{})
	)
	wg.Add(racers)
	for range racers {
		go func() {
			defer wg.Done()
			<-startCh
			ok, err := f.Store.ConsumeAuthCode(ctx, ac.Code)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			if ok {
				wins++
			}
		}()
	}
	close(startCh)
	wg.Wait()

	assert.Empty(t, errs, "concurrent consume must not error")
	assert.Equal(t, 1, wins, "exactly one concurrent consumer may win the code, got %d", wins)
}

// testConsumeAuthCodeUnknown pins the safety-critical half of the contract for
// a code that was never issued: the caller must not be told it won.
//
// The error half diverges by backend and is deliberately not asserted here.
// The memory store returns ErrCodeNotFound; the SQL and mongo stores return a
// nil error because a conditional UPDATE that matches nothing is not an error
// to them. Both are safe, since every caller keys off the bool.
func testConsumeAuthCodeUnknown(t *testing.T, f Fixture) {
	ok, err := f.Store.ConsumeAuthCode(context.Background(), unique("never-issued"))
	assert.False(t, ok, "consuming a code that was never issued must not report a win")
	// Logged rather than asserted, so the divergence stays visible in test
	// output without pinning a contract the backends do not agree on.
	t.Logf("unknown-code error for this backend: %v", err)
}
