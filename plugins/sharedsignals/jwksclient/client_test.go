package jwksclient

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
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rsaJWKS(t *testing.T, kid string, pub *rsa.PublicKey) string {
	t.Helper()
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	body, err := json.Marshal(map[string]any{
		"keys": []map[string]string{
			{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": kid, "n": n, "e": e},
		},
	})
	require.NoError(t, err)
	return string(body)
}

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return k
}

func testOptions(srv *httptest.Server) Options {
	return Options{
		HTTPClient:         srv.Client(),
		MinRefetchInterval: time.Minute,
		MaxResponseBytes:   256 * 1024,
		MaxKeys:            20,
		Now:                time.Now,
		// httptest.NewServer serves plain http on a loopback address, which
		// the real ValidateURI correctly rejects. Substitute a permissive
		// validator so these tests can exercise fetch/cache/single-flight
		// behaviour without also standing up TLS on a public address.
		ValidateURI: func(string) error { return nil },
	}
}

func TestKey_FetchesAndReturnsKey(t *testing.T) {
	key := newKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	got, err := c.Key(context.Background(), srv.URL, "kid-1")
	require.NoError(t, err)

	pub, ok := got.(*rsa.PublicKey)
	require.True(t, ok)
	assert.Equal(t, key.N, pub.N)
}

func TestKey_CachesBetweenCalls(t *testing.T) {
	key := newKey(t)
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	for i := 0; i < 5; i++ {
		_, err := c.Key(context.Background(), srv.URL, "kid-1")
		require.NoError(t, err)
	}
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits))
}

// An unknown kid must not let a caller drive our fetch rate. The first miss
// may refetch once; every miss inside MinRefetchInterval must not.
func TestKey_UnknownKidRespectsRefetchInterval(t *testing.T) {
	key := newKey(t)
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	for i := 0; i < 10; i++ {
		_, err := c.Key(context.Background(), srv.URL, "unknown-kid")
		require.Error(t, err)
	}
	assert.LessOrEqual(t, atomic.LoadInt64(&hits), int64(1),
		"unknown kid drove more than one fetch")
}

// The fixture is large enough that reading only MaxResponseBytes+1 of it
// truncates mid-array, which on its own would fail JSON decoding regardless
// of whether the size check runs -- a prior version of this test asserted
// only require.Error and stayed green even after the size check was
// deleted, because the truncated JSON was invalid for an unrelated reason.
// Asserting the specific sentinel closes that hole: if the check is removed,
// the error becomes a decode error instead of ErrResponseTooLarge and this
// test fails. Verified by temporarily deleting the
// `if int64(len(body)) > c.opts.MaxResponseBytes` check in fetch(): this
// test failed as expected, and passed again once the check was restored.
func TestKey_RejectsOversizedDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"keys":[`+strings.Repeat(`{"kty":"oct"},`, 100_000)+`{"kty":"oct"}]}`)
	}))
	defer srv.Close()

	opts := testOptions(srv)
	opts.MaxResponseBytes = 1024
	c := New(opts)

	_, err := c.Key(context.Background(), srv.URL, "kid-1")
	require.ErrorIs(t, err, ErrResponseTooLarge)
}

func TestKey_RejectsTooManyKeys(t *testing.T) {
	key := newKey(t)
	n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
	var entries []string
	for i := 0; i < 50; i++ {
		entries = append(entries, fmt.Sprintf(
			`{"kty":"RSA","kid":"k%d","n":%q,"e":%q}`, i, n, e))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"keys":[`+strings.Join(entries, ",")+`]}`)
	}))
	defer srv.Close()

	opts := testOptions(srv)
	opts.MaxKeys = 20
	c := New(opts)

	_, err := c.Key(context.Background(), srv.URL, "k0")
	require.Error(t, err)
}

// countingTransport counts how many times RoundTrip was invoked, so a test
// can prove a rejected URI never reached the network.
type countingTransport struct {
	calls int64
}

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	atomic.AddInt64(&t.calls, 1)
	return nil, fmt.Errorf("countingTransport: unexpected request to %s", req.URL)
}

// The fetch-time gate is the only runtime defence against a jwks_uri that
// validated as public at registration but points somewhere unsafe by the
// time it is actually fetched (e.g. DNS rebinding). Leaving Options.ValidateURI
// at its default must still reject a loopback URI, and must do so before any
// HTTP request is attempted.
func TestKey_DefaultValidateURIGatesFetch(t *testing.T) {
	transport := &countingTransport{}
	c := New(Options{
		HTTPClient:         &http.Client{Transport: transport},
		MinRefetchInterval: time.Minute,
		MaxResponseBytes:   256 * 1024,
		MaxKeys:            20,
		Now:                time.Now,
		// ValidateURI left unset: must fall back to the package-level
		// ValidateURI, which rejects http and loopback addresses.
	})

	_, err := c.Key(context.Background(), "http://127.0.0.1:1/keys", "k1")
	require.Error(t, err)
	assert.Equal(t, int64(0), atomic.LoadInt64(&transport.calls),
		"rejected jwks_uri reached the network")
}

// net.ParseIP returns nil for any hostname, so ValidateURI's literal-IP
// check never even runs for "localhost" or "metadata.google.internal" -- it
// waves both straight through. The real enforcement has to happen after DNS
// resolution, on the address that is actually about to be dialed, which is
// what guardDialAddress (wired into the default HTTP client's transport)
// does. This exercises the DEFAULT client end to end: Options.HTTPClient and
// Options.ValidateURI are both left unset.
func TestKey_DefaultDialerRejectsLoopbackHostname(t *testing.T) {
	c := New(Options{
		MinRefetchInterval: time.Minute,
		MaxResponseBytes:   256 * 1024,
		MaxKeys:            20,
		Now:                time.Now,
	})

	// "localhost" passes ValidateURI's literal-IP pre-filter (it isn't an IP
	// literal at all) but resolves to 127.0.0.1, which the dialer's Control
	// func must refuse before ever attempting to connect.
	_, err := c.Key(context.Background(), "https://localhost:1/keys", "kid-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal address",
		"expected the dial guard to reject the resolved loopback address")
}

// panicOnceTransport blocks in RoundTrip until told to proceed, then panics.
// It lets the test deterministically observe "the owner fetch is registered
// and now blocked inside the network call" before deciding whether a
// concurrent waiter should be allowed to see that in-flight state.
type panicOnceTransport struct {
	reached chan struct{}
	proceed chan struct{}
	once    sync.Once
}

func (t *panicOnceTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	t.once.Do(func() { close(t.reached) })
	<-t.proceed
	panic("boom: simulated defect while handling the IdP response")
}

// Before this fix, a panic inside fetch() skipped the inflight cleanup and
// left `done` unclosed. Any concurrent caller for the same URI, waiting on
// that channel, would then block until its own context deadline -- forever,
// with a background context. This is the inbound hot path, so a single
// response that triggers a panic would otherwise be a lasting denial of
// service on that IdP's key verification.
func TestKey_PanicDuringFetchDoesNotStrandWaiters(t *testing.T) {
	transport := &panicOnceTransport{
		reached: make(chan struct{}),
		proceed: make(chan struct{}),
	}
	c := New(Options{
		HTTPClient:         &http.Client{Transport: transport},
		MinRefetchInterval: time.Minute,
		MaxResponseBytes:   256 * 1024,
		MaxKeys:            20,
		Now:                time.Now,
		ValidateURI:        func(string) error { return nil },
	})

	const jwksURI = "http://internal.example/keys"

	var ownerWG sync.WaitGroup
	ownerWG.Add(1)
	go func() {
		defer ownerWG.Done()
		defer func() { _ = recover() }() // the panic is expected to propagate here
		_, _ = c.Key(context.Background(), jwksURI, "kid-1")
	}()

	// Wait until the owner goroutine is registered as the in-flight fetch
	// and is blocked inside RoundTrip, so the waiter below is guaranteed to
	// observe the in-flight entry rather than racing to become the owner
	// itself.
	select {
	case <-transport.reached:
	case <-time.After(2 * time.Second):
		t.Fatal("owner goroutine never reached the network call")
	}

	waiterDone := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = c.Key(ctx, jwksURI, "kid-1")
		close(waiterDone)
	}()

	// Give the waiter goroutine time to reach the inflight wait before we
	// let the owner panic.
	time.Sleep(50 * time.Millisecond)
	close(transport.proceed)

	select {
	case <-waiterDone:
		// Progress: the waiter was released once the owner's panic was
		// cleaned up, instead of hanging until its own 5s context deadline.
	case <-time.After(3 * time.Second):
		t.Fatal("waiter blocked forever after the owner's fetch panicked")
	}

	ownerWG.Wait()

	// The client must still be usable afterwards: the panic must not have
	// left the inflight map or cache in a state that jams this URI. The
	// panic was negative-cached, so a call still inside MinRefetchInterval
	// should return promptly from cache without touching the network again.
	c.opts.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("should not re-fetch: still inside MinRefetchInterval")
		return nil, nil
	})}
	_, err := c.Key(context.Background(), jwksURI, "kid-1")
	require.Error(t, err)
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// Without the scheme check in checkRedirect, a same-host redirect from https
// to http would be followed, and the key set fetched in plaintext -- exactly
// what ValidateURI's https-only requirement is meant to prevent in the first
// place.
func TestCheckRedirect_RejectsSchemeDowngrade(t *testing.T) {
	orig, err := url.Parse("https://idp.example.com/keys")
	require.NoError(t, err)
	downgraded, err := url.Parse("http://idp.example.com/keys")
	require.NoError(t, err)

	err = checkRedirect(&http.Request{URL: downgraded}, []*http.Request{{URL: orig}})
	require.Error(t, err)
}

func TestCheckRedirect_AllowsSameHostHTTPSRedirect(t *testing.T) {
	orig, err := url.Parse("https://idp.example.com/keys")
	require.NoError(t, err)
	next, err := url.Parse("https://idp.example.com/new-keys")
	require.NoError(t, err)

	err = checkRedirect(&http.Request{URL: next}, []*http.Request{{URL: orig}})
	require.NoError(t, err)
}

func TestCheckRedirect_RejectsCrossHost(t *testing.T) {
	orig, err := url.Parse("https://idp.example.com/keys")
	require.NoError(t, err)
	other, err := url.Parse("https://attacker.example.com/keys")
	require.NoError(t, err)

	err = checkRedirect(&http.Request{URL: other}, []*http.Request{{URL: orig}})
	require.Error(t, err)
}

// The core anti-abuse property is that many concurrent lookups of a URI that
// has never been fetched before still produce exactly one upstream request.
// All the other tests in this file are sequential; this one drives ~50
// goroutines at a fresh URI simultaneously, with an artificial server delay
// to widen the window in which a broken single-flight implementation would
// let more than one request through.
func TestKey_ConcurrentCallsSingleFlight(t *testing.T) {
	key := newKey(t)
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		time.Sleep(50 * time.Millisecond)
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	c := New(testOptions(srv))

	const n = 50
	var wg sync.WaitGroup
	errs := make([]error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := c.Key(context.Background(), srv.URL, "kid-1")
			errs[i] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "goroutine %d", i)
	}
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits),
		"concurrent lookups of a fresh URI produced more than one upstream fetch")
}

func TestValidateURI(t *testing.T) {
	require.NoError(t, ValidateURI("https://idp.example.com/keys"))

	// Plain HTTP would let a network attacker swap the keys.
	require.Error(t, ValidateURI("http://idp.example.com/keys"))
	// The cloud metadata endpoint is the canonical SSRF target.
	require.Error(t, ValidateURI("https://169.254.169.254/latest/meta-data/"))
	require.Error(t, ValidateURI("https://127.0.0.1/keys"))
	require.Error(t, ValidateURI("https://10.0.0.5/keys"))
	require.Error(t, ValidateURI("https://192.168.1.1/keys"))
	require.Error(t, ValidateURI("not-a-url"))
}

// ──────────────────────────────────────────────────
// A failed refresh must not cost us the keys we already have
// ──────────────────────────────────────────────────

// The bug this pins: refresh() used to replace the cache entry with an EMPTY
// key map on any fetch error, and then refuse to refetch for
// MinRefetchInterval. A warm cache holding kid-1 was therefore destroyed by
// one 503 raised while looking up an unknown kid-2, and kid-1 became
// unverifiable for the next five minutes -- long enough to drop a genuine
// compromise event on the floor.
func TestKey_WarmKeySurvivesAFailedRefresh(t *testing.T) {
	key := newKey(t)
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	now := time.Now()
	opts := testOptions(srv)
	opts.MinRefetchInterval = 5 * time.Minute
	opts.Now = func() time.Time { return now }
	c := New(opts)

	// Warm the cache with a good fetch.
	_, err := c.Key(context.Background(), srv.URL, "kid-1")
	require.NoError(t, err)

	// The IdP starts failing, and a token arrives naming a kid we do not
	// hold. Move past MinRefetchInterval so the lookup really does attempt a
	// fetch, and let that fetch fail.
	fail.Store(true)
	now = now.Add(10 * time.Minute)

	_, err = c.Key(context.Background(), srv.URL, "kid-2")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrFetchFailed,
		"a fetch that failed must be reported as a fetch failure, not as an unknown kid")

	// The key we already had is still usable.
	got, err := c.Key(context.Background(), srv.URL, "kid-1")
	require.NoError(t, err, "a failed refresh must never destroy keys we already hold")
	pub, ok := got.(*rsa.PublicKey)
	require.True(t, ok)
	assert.Equal(t, key.N, pub.N)
}

// The two failures a caller has to tell apart: the key set loaded fine and
// carries no such kid (permanent, the token is wrong) versus we could not
// load the key set at all (transient, our side, retry).
func TestKey_DistinguishesFetchFailureFromUnknownKid(t *testing.T) {
	key := newKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	_, err := c.Key(context.Background(), srv.URL, "nope")
	require.ErrorIs(t, err, ErrKeyNotFound)
	assert.NotErrorIs(t, err, ErrFetchFailed,
		"a key set that loaded cleanly must not look like a fetch failure")

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer dead.Close()

	c2 := New(testOptions(dead))
	_, err = c2.Key(context.Background(), dead.URL, "kid-1")
	require.ErrorIs(t, err, ErrFetchFailed)
	assert.NotErrorIs(t, err, ErrKeyNotFound,
		"an unreachable IdP must not look like a token naming a key that does not exist")
}

// A caller arriving inside the negative-cache window, after a failed fetch,
// must still be told the fetch failed rather than being handed
// ErrKeyNotFound -- that is the answer that becomes a 400 and stops the
// transmitter retrying.
func TestKey_NegativeCacheReportsTheFetchFailure(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	for i := 0; i < 5; i++ {
		_, err := c.Key(context.Background(), srv.URL, "kid-1")
		require.ErrorIs(t, err, ErrFetchFailed, "call %d", i)
	}
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits),
		"a broken endpoint must still be negative-cached")
}

// ──────────────────────────────────────────────────
// Keys do not stay trusted forever
// ──────────────────────────────────────────────────

// An IdP that pulls a compromised signing key from its JWKS has to be able to
// make us stop honouring it. Before MaxKeyAge, only an UNKNOWN kid ever
// triggered a refetch, so a kid already in the cache was trusted for the life
// of the process.
func TestKey_AgedEntryRefetchesAndDropsARetiredKey(t *testing.T) {
	retired, replacement := newKey(t), newKey(t)
	var serveReplacement atomic.Bool
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		if serveReplacement.Load() {
			fmt.Fprint(w, rsaJWKS(t, "kid-2", &replacement.PublicKey))
			return
		}
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &retired.PublicKey))
	}))
	defer srv.Close()

	now := time.Now()
	opts := testOptions(srv)
	opts.MaxKeyAge = time.Hour
	opts.MinRefetchInterval = time.Minute
	opts.Now = func() time.Time { return now }
	c := New(opts)

	_, err := c.Key(context.Background(), srv.URL, "kid-1")
	require.NoError(t, err)
	require.Equal(t, int64(1), atomic.LoadInt64(&hits))

	// Still inside MaxKeyAge: served from cache, no fetch.
	now = now.Add(30 * time.Minute)
	_, err = c.Key(context.Background(), srv.URL, "kid-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits),
		"a cached key younger than MaxKeyAge must not drive a fetch")

	// Past MaxKeyAge, with the IdP now publishing a different key: the aged
	// hit forces a refetch and the retired key stops verifying anything.
	serveReplacement.Store(true)
	now = now.Add(31 * time.Minute)
	_, err = c.Key(context.Background(), srv.URL, "kid-1")
	require.ErrorIs(t, err, ErrKeyNotFound,
		"a key the IdP has retired must stop being honoured once the entry ages out")
	assert.Equal(t, int64(2), atomic.LoadInt64(&hits))

	got, err := c.Key(context.Background(), srv.URL, "kid-2")
	require.NoError(t, err)
	pub, ok := got.(*rsa.PublicKey)
	require.True(t, ok)
	assert.Equal(t, replacement.N, pub.N)
}
