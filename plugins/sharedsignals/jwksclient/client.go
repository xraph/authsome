// Package jwksclient fetches and caches JSON Web Key Sets for inbound token
// verification. Every limit in here exists because the fetch is triggered by
// traffic we do not control.
package jwksclient

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"sync"
	"syscall"
	"time"
)

// ErrKeyNotFound is returned when the key set loaded cleanly and simply does
// not carry the wanted kid. It is a statement about the IdP's published keys,
// so a caller may safely treat it as a permanent rejection of the token.
var ErrKeyNotFound = errors.New("jwksclient: no key for kid")

// ErrFetchFailed is returned when we could not load the key set at all: the
// endpoint was unreachable, answered non-200, sent something we could not
// parse, or was refused by the URI/dial guards. It says nothing about the
// token, only about our own ability to check it right now, so a caller must
// answer with something that invites a retry rather than a permanent
// rejection. Every error surfaced from a failed fetch wraps this.
var ErrFetchFailed = errors.New("jwksclient: key set fetch failed")

// ErrResponseTooLarge is returned when a JWKS response exceeds MaxResponseBytes.
var ErrResponseTooLarge = errors.New("jwksclient: key set exceeds the size limit")

// Options configures a Client.
type Options struct {
	HTTPClient         *http.Client
	MinRefetchInterval time.Duration
	MaxResponseBytes   int64
	MaxKeys            int
	// MaxKeyAge bounds how long a cached key set is served without going
	// back to the IdP. Without it, a kid that is present in the cache is
	// trusted forever and an IdP that pulls a compromised signing key from
	// its JWKS never causes us to stop honouring it, because only an
	// UNKNOWN kid triggers a refetch. Checking the age inside Key is
	// simpler than a background goroutine and needs no lifecycle of its
	// own: the next inbound token pays for the refresh.
	MaxKeyAge time.Duration
	Now       func() time.Time
	// ValidateURI gates every fetch. Defaults to the package-level
	// ValidateURI. Tests substitute a permissive validator so they can serve
	// a key set from an httptest loopback server; production must not.
	ValidateURI func(rawURL string) error
}

func (o *Options) defaults() {
	if o.HTTPClient == nil {
		o.HTTPClient = &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 5 * time.Second,
					// Control runs after DNS resolution, on the resolved IP,
					// immediately before the socket opens. That is the only
					// point that catches both a hostname that resolves
					// straight to an internal address (ValidateURI's literal
					// check never sees a bare hostname) and DNS rebinding
					// (an address that was public when ValidateURI ran and
					// is not by the time we actually connect).
					Control: guardDialAddress,
				}).DialContext,
			},
			CheckRedirect: checkRedirect,
		}
	}
	if o.MinRefetchInterval == 0 {
		o.MinRefetchInterval = 5 * time.Minute
	}
	if o.MaxResponseBytes == 0 {
		o.MaxResponseBytes = 256 * 1024
	}
	if o.MaxKeys == 0 {
		o.MaxKeys = 20
	}
	if o.MaxKeyAge == 0 {
		// An hour is long enough that ordinary traffic almost never pays for
		// a fetch, and short enough that a key an IdP retired stops being
		// honoured on the same day somebody noticed.
		o.MaxKeyAge = time.Hour
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.ValidateURI == nil {
		o.ValidateURI = ValidateURI
	}
}

// guardDialAddress is the net.Dialer Control func for the default HTTP
// client's transport. It runs after DNS resolution and immediately before
// the socket opens, on the address actually being connected to, and refuses
// loopback, private, link-local and unspecified addresses. This is the real
// enforcement point for "do not let an inbound request make us dial an
// internal address" -- ValidateURI is only a cheap pre-filter that cannot
// see through a hostname.
func guardDialAddress(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("jwksclient: parse dial address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("jwksclient: dial address %q did not resolve to an IP", host)
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("jwksclient: refusing to dial internal address %s", ip)
	}
	return nil
}

// checkRedirect is the default HTTP client's redirect policy: same host,
// https only, and at most a couple of hops. Without the scheme check a
// same-host redirect from https to http would be followed and the key set
// fetched in plaintext, defeating ValidateURI's https-only requirement.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > 0 {
		if req.URL.Scheme != "https" {
			return errors.New("jwksclient: redirect to non-https refused")
		}
		if req.URL.Host != via[0].URL.Host {
			return errors.New("jwksclient: cross-host redirect refused")
		}
	}
	if len(via) >= 3 {
		return errors.New("jwksclient: too many redirects")
	}
	return nil
}

// entry is one URI's cached key set. It carries two clocks on purpose.
// lastAttempt stamps every fetch, successful or not, and is what the
// negative-cache interval reads, so a broken endpoint cannot be hammered.
// lastSuccess stamps only a fetch that actually produced a key set, and is
// what MaxKeyAge reads. Keeping them apart is what stops a run of failures
// from making a stale key look freshly confirmed.
type entry struct {
	keys        map[string]crypto.PublicKey
	lastAttempt time.Time
	lastSuccess time.Time
	// lastErr is the failure from the most recent attempt, already wrapped
	// with ErrFetchFailed, or nil when that attempt succeeded. It lets a
	// caller arriving inside the negative-cache window learn that the fetch
	// FAILED rather than being told the kid does not exist.
	lastErr error
}

// Client caches one key set per JWKS URI.
type Client struct {
	opts  Options
	mu    sync.Mutex
	cache map[string]*entry
	// inflight collapses concurrent fetches of the same URI into one request.
	inflight map[string]chan struct{}
}

// New builds a Client, filling in the hardened defaults for any zero option.
func New(opts Options) *Client {
	opts.defaults()
	return &Client{
		opts:     opts,
		cache:    make(map[string]*entry),
		inflight: make(map[string]chan struct{}),
	}
}

// Key returns the public key for kid from the set at jwksURI. A cache miss,
// or a hit older than MaxKeyAge, triggers at most one fetch per
// MinRefetchInterval per URI, so neither an unknown kid nor an aged entry can
// be used to drive outbound traffic.
//
// A key that is still cached is returned even when the refresh that was
// supposed to confirm it failed. That is deliberate and it is the one place
// this file trades a little freshness for availability: a successful refresh
// replaces the entry wholesale, so a retired kid does disappear, while a
// transient 503 on the IdP's side must not blind the receiver to a
// compromise event signed with a key we already hold and already trust.
func (c *Client) Key(ctx context.Context, jwksURI, kid string) (crypto.PublicKey, error) {
	if key, ok := c.freshKey(jwksURI, kid); ok {
		return key, nil
	}
	refreshErr := c.refresh(ctx, jwksURI)
	if key, ok := c.cachedKey(jwksURI, kid); ok {
		return key, nil
	}
	if refreshErr != nil {
		return nil, refreshErr
	}
	return nil, ErrKeyNotFound
}

// freshKey returns a cached key only when the set it came from was
// successfully loaded within MaxKeyAge.
func (c *Client) freshKey(jwksURI, kid string) (crypto.PublicKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[jwksURI]
	if !ok {
		return nil, false
	}
	if e.lastSuccess.IsZero() ||
		c.opts.Now().Sub(e.lastSuccess) >= c.opts.MaxKeyAge {
		return nil, false
	}
	key, ok := e.keys[kid]
	return key, ok
}

func (c *Client) cachedKey(jwksURI, kid string) (crypto.PublicKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[jwksURI]
	if !ok {
		return nil, false
	}
	key, ok := e.keys[kid]
	return key, ok
}

// lastError reports how the most recent fetch of this URI ended, so a
// goroutine that waited on somebody else's in-flight fetch learns the same
// thing the owner did.
func (c *Client) lastError(jwksURI string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.cache[jwksURI]; ok {
		return e.lastErr
	}
	return nil
}

// refresh fetches the key set, unless another goroutine is already doing so
// or the last fetch is more recent than MinRefetchInterval.
func (c *Client) refresh(ctx context.Context, jwksURI string) (err error) {
	c.mu.Lock()
	if e, ok := c.cache[jwksURI]; ok &&
		c.opts.Now().Sub(e.lastAttempt) < c.opts.MinRefetchInterval {
		lastErr := e.lastErr
		c.mu.Unlock()
		// Inside the negative-cache window we cannot go and look, so answer
		// with whatever the last look actually found. Reporting a fetch
		// failure as "no such kid" is what turned one transient 503 into
		// five minutes of 400s telling the transmitter to stop retrying.
		if lastErr != nil {
			return lastErr
		}
		return ErrKeyNotFound
	}
	if wait, ok := c.inflight[jwksURI]; ok {
		c.mu.Unlock()
		select {
		case <-wait:
			return c.lastError(jwksURI)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	done := make(chan struct{})
	c.inflight[jwksURI] = done
	c.mu.Unlock()

	var keys map[string]crypto.PublicKey
	// This is the inbound-triggered hot path: fetch() parses whatever an
	// external IdP sent back, so a panic in there (malformed response, a bug
	// in a decoder) must not be allowed to skip cleanup. Without this defer,
	// a panic would leave the inflight entry registered and `done` unclosed,
	// stranding every concurrent waiter on this URI until its own context
	// deadline -- a lasting denial of service on that IdP's key verification
	// triggered by a single bad response.
	defer func() {
		r := recover()
		now := c.opts.Now()
		c.mu.Lock()
		delete(c.inflight, jwksURI)
		switch {
		case r == nil && err == nil:
			c.cache[jwksURI] = &entry{
				keys: keys, lastAttempt: now, lastSuccess: now,
			}
		default:
			// Negative caching: a failed fetch (including a panic) still
			// stamps lastAttempt so a broken endpoint cannot be hammered on
			// every inbound request.
			//
			// What it must NOT do is throw away keys that are already here.
			// A warm cache holding the kid every real token is signed with
			// used to be wiped by a single 503 raised while looking up some
			// other, unknown kid, and stayed wiped for the whole
			// negative-cache interval -- five minutes in which a genuine
			// compromise event could not be verified at all.
			failure := err
			if r != nil {
				failure = fmt.Errorf("panic while loading the key set: %v", r)
			}
			wrapped := fmt.Errorf("%w: %w", ErrFetchFailed, failure)
			if existing, ok := c.cache[jwksURI]; ok {
				existing.lastAttempt = now
				existing.lastErr = wrapped
			} else {
				c.cache[jwksURI] = &entry{
					keys:        map[string]crypto.PublicKey{},
					lastAttempt: now,
					lastErr:     wrapped,
				}
			}
			// Hand the caller the sentinel too, so a receiver can tell "we
			// could not check this token" from "this token names a key the
			// IdP does not publish" and answer 5xx rather than a 400 that
			// stops the transmitter retrying.
			err = wrapped
		}
		c.mu.Unlock()
		close(done)
		if r != nil {
			panic(r)
		}
	}()

	keys, err = c.fetch(ctx, jwksURI)
	return err
}

type jwk struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	CRV string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (c *Client) fetch(ctx context.Context, jwksURI string) (map[string]crypto.PublicKey, error) {
	if err := c.opts.ValidateURI(jwksURI); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return nil, fmt.Errorf("jwksclient: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.opts.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jwksclient: fetch: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body close

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwksclient: fetch returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.opts.MaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("jwksclient: read body: %w", err)
	}
	if int64(len(body)) > c.opts.MaxResponseBytes {
		return nil, ErrResponseTooLarge
	}

	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("jwksclient: decode key set: %w", err)
	}
	if len(doc.Keys) > c.opts.MaxKeys {
		return nil, errors.New("jwksclient: key set holds too many keys")
	}

	out := make(map[string]crypto.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		pub, err := k.publicKey()
		if err != nil {
			continue // skip a key we cannot use, keep the rest
		}
		out[k.KID] = pub
	}
	return out, nil
}

func (k jwk) publicKey() (crypto.PublicKey, error) {
	switch k.KTY {
	case "RSA":
		n, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, err
		}
		e, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, err
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(n),
			E: int(new(big.Int).SetBytes(e).Int64()),
		}, nil
	case "EC":
		var curve elliptic.Curve
		switch k.CRV {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("jwksclient: unsupported curve %q", k.CRV)
		}
		x, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, err
		}
		y, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(x),
			Y:     new(big.Int).SetBytes(y),
		}, nil
	default:
		return nil, fmt.Errorf("jwksclient: unsupported key type %q", k.KTY)
	}
}

// ValidateURI is a cheap pre-filter, run when a JWKS URI is accepted and
// stored on a stream. It refuses plain HTTP and a host that is a literal
// private, loopback or link-local IP address (the obvious cases: someone
// pastes "https://169.254.169.254/..." directly). It does NOT resolve
// hostnames, so it cannot catch "https://localhost/keys" or
// "https://metadata.google.internal/keys" -- a bare hostname always fails
// net.ParseIP and skips the whole check. It also cannot catch DNS rebinding,
// where a hostname resolves to a public address now and an internal one
// later. Neither gap matters for what actually reaches the network: the
// default HTTP client's dialer (see guardDialAddress) inspects the resolved
// address immediately before every connection, after DNS resolution, and is
// what actually enforces the address policy for real traffic.
func ValidateURI(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("jwksclient: parse jwks_uri: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("jwksclient: jwks_uri must use https")
	}
	if u.Host == "" {
		return errors.New("jwksclient: jwks_uri has no host")
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return errors.New("jwksclient: jwks_uri points at a non-public address")
		}
	}
	return nil
}
