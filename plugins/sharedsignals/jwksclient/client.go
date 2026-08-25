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

// ErrKeyNotFound is returned when no key in the set carries the wanted kid.
var ErrKeyNotFound = errors.New("jwksclient: no key for kid")

// ErrResponseTooLarge is returned when a JWKS response exceeds MaxResponseBytes.
var ErrResponseTooLarge = errors.New("jwksclient: key set exceeds the size limit")

// Options configures a Client.
type Options struct {
	HTTPClient         *http.Client
	MinRefetchInterval time.Duration
	MaxResponseBytes   int64
	MaxKeys            int
	Now                func() time.Time
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

type entry struct {
	keys        map[string]crypto.PublicKey
	lastFetched time.Time
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

// Key returns the public key for kid from the set at jwksURI. A cache miss
// triggers at most one fetch per MinRefetchInterval per URI, so an unknown
// kid cannot be used to drive outbound traffic.
func (c *Client) Key(ctx context.Context, jwksURI, kid string) (crypto.PublicKey, error) {
	if key, ok := c.cachedKey(jwksURI, kid); ok {
		return key, nil
	}
	if err := c.refresh(ctx, jwksURI); err != nil {
		return nil, err
	}
	if key, ok := c.cachedKey(jwksURI, kid); ok {
		return key, nil
	}
	return nil, ErrKeyNotFound
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

// refresh fetches the key set, unless another goroutine is already doing so
// or the last fetch is more recent than MinRefetchInterval.
func (c *Client) refresh(ctx context.Context, jwksURI string) (err error) {
	c.mu.Lock()
	if e, ok := c.cache[jwksURI]; ok &&
		c.opts.Now().Sub(e.lastFetched) < c.opts.MinRefetchInterval {
		c.mu.Unlock()
		return ErrKeyNotFound
	}
	if wait, ok := c.inflight[jwksURI]; ok {
		c.mu.Unlock()
		select {
		case <-wait:
			return nil
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
		c.mu.Lock()
		delete(c.inflight, jwksURI)
		if r == nil && err == nil {
			c.cache[jwksURI] = &entry{keys: keys, lastFetched: c.opts.Now()}
		} else {
			// Negative caching: a failed fetch (including a panic) still
			// stamps the clock so a broken endpoint cannot be hammered on
			// every inbound request.
			c.cache[jwksURI] = &entry{keys: map[string]crypto.PublicKey{}, lastFetched: c.opts.Now()}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("jwksclient: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.opts.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jwksclient: fetch: %w", err)
	}
	defer resp.Body.Close()

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
		// Assembling the point and parsing it, rather than assigning X and Y,
		// is what Go 1.26 deprecated those fields in favour of. It is also
		// stricter in a way that matters here: ParseUncompressedPublicKey
		// rejects a point that is not on the curve or is the point at
		// infinity, and a struct literal accepts both. These coordinates come
		// off the wire from someone else's JWKS endpoint.
		byteLen := (curve.Params().BitSize + 7) / 8
		if len(x) > byteLen || len(y) > byteLen {
			return nil, fmt.Errorf("jwksclient: coordinate longer than curve %s", k.CRV)
		}

		// RFC 7518 section 6.2.1.2 fixes the octet length per curve, but a
		// leading zero byte is easy to drop, so left-pad rather than trust it.
		point := make([]byte, 1+2*byteLen)
		point[0] = 4
		copy(point[1+byteLen-len(x):1+byteLen], x)
		copy(point[1+2*byteLen-len(y):], y)

		pub, parseErr := ecdsa.ParseUncompressedPublicKey(curve, point)
		if parseErr != nil {
			return nil, fmt.Errorf("jwksclient: invalid %s public key: %w", k.CRV, parseErr)
		}

		return pub, nil
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
