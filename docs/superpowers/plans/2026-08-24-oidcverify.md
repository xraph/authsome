# oidcverify implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A standalone package that takes a JWT signed by an external issuer plus the trust configuration for that issuer, and returns verified claims or a specific error.

**Architecture:** Four small files behind one `Verifier` type. Discovery resolves a JWKS URI under three constraints, a key cache fetches and holds public keys with a TTL and a refresh cooldown, and the verify pipeline checks the algorithm against an allowlist before the signature and the claims after it. No HTTP handlers, no store, no plugin. Everything is exercised with locally generated keys and `httptest`, so the whole suite runs offline.

**Tech Stack:** Go, `github.com/golang-jwt/jwt/v5` v5.3.1 for parsing and signature verification, `github.com/go-jose/go-jose/v4` v4.1.4 for JWK decoding (already in the module graph as an indirect dependency; this promotes it to direct), `stretchr/testify`.

**Spec:** [docs/superpowers/specs/2026-08-24-workload-identity-federation-design.md](../specs/2026-08-24-workload-identity-federation-design.md), the "Token verification" section. This plan covers phase 1 of that spec's phasing. Phase 0 is done.

## Global Constraints

- This package must not import `plugin`, `store`, `session`, or anything under `plugins/`. It takes values and returns values. The plugin that consumes it is a later plan.
- Steps 1 and 2 of the spec's seven-step pipeline (resolve the trust config by exact `iss` match, scoped to app and environment) are **not** in this package. They belong to the plugin, which owns the store. This package receives an already-resolved `Issuer` and must never fetch from a URL that did not come from one.
- Every outbound request uses the `Verifier`'s own `*http.Client` with an explicit timeout and a response size cap. Never `http.DefaultClient`, which is what `plugins/sso/oidc.go:178` does and what this package exists partly to avoid repeating.
- `exp` is mandatory. A token without one is refused, never defaulted.
- `alg: none` is refused unconditionally, and any algorithm outside the issuer's `AllowedAlgorithms` is refused before the signature is checked.
- Do not add a package-level singleton or global cache. The `Verifier` holds its own state so a test can build one per case.
- Run `go test ./oidcverify/... -race` before every commit. The repo-wide suite is currently green; keep it that way.
- Another session may be clearing the shared Go build cache. If you see `link: cannot open file .../go-build/...`, it is not your code. Set `GOCACHE` to a private directory and retry.

## File structure

| File | Responsibility |
|---|---|
| `oidcverify/doc.go` | Package doc: what this verifies and what it deliberately does not |
| `oidcverify/errors.go` | Sentinel errors callers match on |
| `oidcverify/claims.go` | `Issuer` config, `Claims` result, unverified header inspection |
| `oidcverify/discovery.go` | Discovery document fetch and its three constraints |
| `oidcverify/keys.go` | JWKS fetch, decode and cache |
| `oidcverify/verify.go` | The `Verifier` and the pipeline |
| `oidcverify/claims_test.go` | Header inspection and algorithm gating |
| `oidcverify/discovery_test.go` | Discovery constraints, driven by `httptest` |
| `oidcverify/keys_test.go` | Cache TTL, negative caching, refresh cooldown |
| `oidcverify/verify_test.go` | The adversarial table over crafted tokens |

---

### Task 1: Issuer config, claims and unverified header inspection

Everything downstream needs to read `iss`, `kid` and `alg` before it can decide anything, and it has to do that without trusting the token. This task builds that and the types around it.

**Files:**
- Create: `oidcverify/doc.go`, `oidcverify/errors.go`, `oidcverify/claims.go`
- Test: `oidcverify/claims_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Issuer struct` with fields `URL`, `JWKSURI`, `Audience`, `AllowedAlgorithms []string`, `MaxTokenAge time.Duration`.
  - `type Claims struct` with `Issuer`, `Subject`, `Audience []string`, `ID`, `IssuedAt`, `ExpiresAt time.Time`, and `All map[string]any`.
  - `func (c *Claims) String(name string) (string, bool)` reading a string claim out of `All`.
  - `func inspect(raw string) (alg, kid, iss string, err error)`, unexported, used by the pipeline.
  - Sentinel errors listed in Step 3.

- [ ] **Step 1: Write the failing test**

Create `oidcverify/claims_test.go`:

```go
package oidcverify

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signHS256 builds a token with an arbitrary header and claim set. Used only
// where the test cares about parsing rather than about the signature.
func signHS256(t *testing.T, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if kid != "" {
		tok.Header["kid"] = kid
	}
	s, err := tok.SignedString([]byte("test-key-at-least-32-bytes-long!!!!"))
	require.NoError(t, err)
	return s
}

func TestInspectReadsHeaderAndIssuer(t *testing.T) {
	raw := signHS256(t, "key-1", jwt.MapClaims{
		"iss": "https://token.actions.githubusercontent.com",
		"sub": "repo:acme/api:ref:refs/heads/main",
	})

	alg, kid, iss, err := inspect(raw)
	require.NoError(t, err)
	assert.Equal(t, "HS256", alg)
	assert.Equal(t, "key-1", kid)
	assert.Equal(t, "https://token.actions.githubusercontent.com", iss)
}

func TestInspectRejectsGarbage(t *testing.T) {
	_, _, _, err := inspect("not-a-jwt")
	require.Error(t, err)
}

// A token with no kid is legal. Some issuers publish a single key and omit it.
func TestInspectAllowsMissingKid(t *testing.T) {
	raw := signHS256(t, "", jwt.MapClaims{"iss": "https://example.com"})
	alg, kid, iss, err := inspect(raw)
	require.NoError(t, err)
	assert.Equal(t, "HS256", alg)
	assert.Empty(t, kid)
	assert.Equal(t, "https://example.com", iss)
}

func TestClaimsStringReadsFromAll(t *testing.T) {
	c := &Claims{All: map[string]any{"repository": "acme/api", "run_id": float64(17)}}

	got, ok := c.String("repository")
	require.True(t, ok)
	assert.Equal(t, "acme/api", got)

	_, ok = c.String("run_id")
	assert.False(t, ok, "a non-string claim is not a string claim")

	_, ok = c.String("absent")
	assert.False(t, ok)
}

func TestIssuerNormalisesTrailingSlash(t *testing.T) {
	a := Issuer{URL: "https://accounts.google.com/"}
	b := Issuer{URL: "https://accounts.google.com"}
	assert.Equal(t, b.normalisedURL(), a.normalisedURL())
}

func TestClaimsExpiryRoundTrip(t *testing.T) {
	exp := time.Now().Add(time.Minute).Truncate(time.Second)
	c := &Claims{ExpiresAt: exp}
	assert.Equal(t, exp, c.ExpiresAt)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./oidcverify/ -run "TestInspect|TestClaims|TestIssuer" -v`

Expected: the package does not exist yet, so this fails with `no Go files in .../oidcverify` or a build error naming `inspect`, `Claims` and `Issuer`.

- [ ] **Step 3: Write the package doc and errors**

Create `oidcverify/doc.go`:

```go
// Package oidcverify verifies a JWT signed by an external OIDC issuer.
//
// It answers one question: given a token and the trust configuration for the
// issuer that claims to have signed it, are the claims genuine? It does not
// decide which issuers are trusted, does not read a store, and does not know
// what a workload is. The caller resolves trust and passes it in.
//
// That boundary is deliberate. Key resolution performs an outbound HTTPS
// request, so a verifier that accepted an issuer straight from a token would
// let anyone holding a token make the server fetch a URL of their choosing.
// Callers must resolve the issuer against their own registered configuration
// first and pass the result here.
package oidcverify
```

Create `oidcverify/errors.go`:

```go
package oidcverify

import "errors"

var (
	// ErrMalformedToken means the token could not be parsed at all.
	ErrMalformedToken = errors.New("oidcverify: malformed token")
	// ErrAlgorithmNotAllowed means the token's alg is absent from the
	// issuer's allowlist, or is "none".
	ErrAlgorithmNotAllowed = errors.New("oidcverify: algorithm not allowed")
	// ErrIssuerMismatch means the token's iss is not the configured issuer.
	ErrIssuerMismatch = errors.New("oidcverify: issuer mismatch")
	// ErrAudienceMismatch means aud does not contain the configured audience.
	ErrAudienceMismatch = errors.New("oidcverify: audience mismatch")
	// ErrMissingExpiry means the token carries no exp claim. Refused rather
	// than defaulted: a token that never expires is not a short-lived one.
	ErrMissingExpiry = errors.New("oidcverify: token has no expiry")
	// ErrTokenExpired means exp is in the past.
	ErrTokenExpired = errors.New("oidcverify: token expired")
	// ErrTokenTooOld means the token is older than the issuer's MaxTokenAge,
	// enforced separately from exp because some platforms issue long-lived
	// OIDC tokens and the exchange window should not be whatever they chose.
	ErrTokenTooOld = errors.New("oidcverify: token older than the configured maximum age")
	// ErrUnknownKey means no key in the issuer's JWKS matches the token's kid.
	ErrUnknownKey = errors.New("oidcverify: no matching key")
	// ErrBadSignature means the signature did not verify against the key.
	ErrBadSignature = errors.New("oidcverify: signature verification failed")
	// ErrDiscoveryFailed means the discovery document could not be fetched or
	// did not satisfy the constraints in discovery.go.
	ErrDiscoveryFailed = errors.New("oidcverify: discovery failed")
	// ErrJWKSFailed means the key set could not be fetched or decoded.
	ErrJWKSFailed = errors.New("oidcverify: jwks fetch failed")
	// ErrInvalidIssuerConfig means the Issuer passed in is unusable.
	ErrInvalidIssuerConfig = errors.New("oidcverify: invalid issuer configuration")
)
```

- [ ] **Step 4: Write the types and header inspection**

Create `oidcverify/claims.go`:

```go
package oidcverify

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer is the trust configuration for one external OIDC issuer. The caller
// resolves this from its own store before calling Verify.
type Issuer struct {
	// URL is the exact issuer identifier, matched against the iss claim.
	URL string

	// JWKSURI names the key set directly. Empty means discover it from URL.
	JWKSURI string

	// Audience must appear in the token's aud. Empty is a configuration
	// error: an unpinned audience lets a token minted for somebody else's
	// service be replayed at yours.
	Audience string

	// AllowedAlgorithms is the signing algorithms this issuer may use, for
	// example []string{"RS256"}. Empty is a configuration error.
	AllowedAlgorithms []string

	// MaxTokenAge bounds how old a token may be regardless of its exp. Zero
	// means only exp applies.
	MaxTokenAge time.Duration
}

// normalisedURL is the issuer identifier with any trailing slash removed, so
// "https://x/" and "https://x" are the same issuer.
func (i Issuer) normalisedURL() string {
	return strings.TrimRight(i.URL, "/")
}

func (i Issuer) validate() error {
	switch {
	case i.normalisedURL() == "":
		return fmt.Errorf("%w: issuer url is empty", ErrInvalidIssuerConfig)
	case i.Audience == "":
		return fmt.Errorf("%w: audience is empty", ErrInvalidIssuerConfig)
	case len(i.AllowedAlgorithms) == 0:
		return fmt.Errorf("%w: no allowed algorithms", ErrInvalidIssuerConfig)
	}
	for _, a := range i.AllowedAlgorithms {
		if strings.EqualFold(a, "none") {
			return fmt.Errorf("%w: alg none is never allowed", ErrInvalidIssuerConfig)
		}
	}
	return nil
}

// allows reports whether alg is on the issuer's allowlist.
func (i Issuer) allows(alg string) bool {
	if strings.EqualFold(alg, "none") {
		return false
	}
	for _, a := range i.AllowedAlgorithms {
		if a == alg {
			return true
		}
	}
	return false
}

// Claims is a verified token's payload.
type Claims struct {
	Issuer    string
	Subject   string
	Audience  []string
	ID        string // the jti claim, used for replay prevention
	IssuedAt  time.Time
	ExpiresAt time.Time

	// All is every claim in the token, which is what claim-matching rules and
	// attribution read. The typed fields above are conveniences over it.
	All map[string]any
}

// String returns a string-valued claim. Returns false when the claim is
// absent or is not a string, so a caller never silently matches against a
// number or an object.
func (c *Claims) String(name string) (string, bool) {
	if c == nil || c.All == nil {
		return "", false
	}
	v, ok := c.All[name].(string)
	return v, ok
}

// inspect reads alg, kid and iss without verifying anything. Nothing it
// returns may be trusted; it exists so the pipeline can pick an allowlist and
// a key before it has a reason to believe the token.
func inspect(raw string) (alg, kid, iss string, err error) {
	var claims jwt.MapClaims
	parser := jwt.NewParser()
	tok, _, err := parser.ParseUnverified(raw, &claims)
	if err != nil {
		return "", "", "", fmt.Errorf("%w: %w", ErrMalformedToken, err)
	}

	alg, _ = tok.Header["alg"].(string)
	kid, _ = tok.Header["kid"].(string)
	iss, _ = claims["iss"].(string)
	return alg, kid, iss, nil
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./oidcverify/ -run "TestInspect|TestClaims|TestIssuer" -v`

Expected: PASS, six tests.

- [ ] **Step 6: Tidy the module**

`go-jose` is currently an indirect dependency and is not yet imported, so this
step only confirms the build is clean before it becomes direct in Task 3.

Run: `go mod tidy && go build ./... && go test ./oidcverify/... -race`

Expected: build succeeds, tests pass.

- [ ] **Step 7: Commit**

```bash
git add oidcverify/doc.go oidcverify/errors.go oidcverify/claims.go oidcverify/claims_test.go go.mod go.sum
git commit -m "feat(oidcverify): issuer config, claims and unverified header inspection"
```

---

### Task 2: Discovery with its three constraints

An issuer that does not name a `jwks_uri` directly gets one from its discovery document. That document is fetched over the network, so what it is allowed to say has to be bounded.

**Files:**
- Create: `oidcverify/discovery.go`
- Test: `oidcverify/discovery_test.go`

**Interfaces:**
- Consumes: `Issuer`, `ErrDiscoveryFailed` from Task 1.
- Produces: `func (v *Verifier) resolveJWKSURI(ctx context.Context, iss Issuer) (string, error)` and the `httpDoer` interface plus `fetchJSON` helper that Task 3 also uses.

- [ ] **Step 1: Write the failing test**

Create `oidcverify/discovery_test.go`:

```go
package oidcverify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each test below builds its own server because httptest assigns the URL
// only after the listener starts, and the discovery document has to name that
// URL. A shared helper cannot express that ordering.

// An explicit JWKSURI short-circuits discovery entirely, so no request is made.
func TestResolveJWKSURI_ExplicitSkipsDiscovery(t *testing.T) {
	v := New(WithInsecureAllowHTTP())
	got, err := v.resolveJWKSURI(context.Background(), Issuer{
		URL:               "https://example.com",
		JWKSURI:           "https://example.com/keys",
		Audience:          "aud",
		AllowedAlgorithms: []string{"RS256"},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/keys", got)
}

func TestResolveJWKSURI_DiscoversFromServer(t *testing.T) {
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issuer":"` + srv.URL + `","jwks_uri":"` + srv.URL + `/keys"}`))
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	v := New(WithInsecureAllowHTTP())
	got, err := v.resolveJWKSURI(context.Background(), Issuer{
		URL:               srv.URL,
		Audience:          "aud",
		AllowedAlgorithms: []string{"RS256"},
	})
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/keys", got)
}

// A discovery document whose own issuer field disagrees with the registered
// issuer is refused. Without this a hijacked well-known path can hand key
// resolution to somebody else.
func TestResolveJWKSURI_RefusesIssuerMismatch(t *testing.T) {
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"issuer":"https://evil.example","jwks_uri":"` + srv.URL + `/keys"}`))
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	v := New(WithInsecureAllowHTTP())
	_, err := v.resolveJWKSURI(context.Background(), Issuer{
		URL:               srv.URL,
		Audience:          "aud",
		AllowedAlgorithms: []string{"RS256"},
	})
	require.ErrorIs(t, err, ErrDiscoveryFailed)
}

// The jwks_uri must live on the issuer's own host.
func TestResolveJWKSURI_RefusesForeignJWKSHost(t *testing.T) {
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"issuer":"` + srv.URL + `","jwks_uri":"https://evil.example/keys"}`))
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	v := New(WithInsecureAllowHTTP())
	_, err := v.resolveJWKSURI(context.Background(), Issuer{
		URL:               srv.URL,
		Audience:          "aud",
		AllowedAlgorithms: []string{"RS256"},
	})
	require.ErrorIs(t, err, ErrDiscoveryFailed)
}

// Plain HTTP is refused unless a test opts in.
func TestResolveJWKSURI_RefusesPlainHTTP(t *testing.T) {
	v := New()
	_, err := v.resolveJWKSURI(context.Background(), Issuer{
		URL:               "http://insecure.example",
		Audience:          "aud",
		AllowedAlgorithms: []string{"RS256"},
	})
	require.ErrorIs(t, err, ErrDiscoveryFailed)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./oidcverify/ -run TestResolveJWKSURI -v`

Expected: build failure naming `New`, `WithInsecureAllowHTTP` and `resolveJWKSURI`.

- [ ] **Step 3: Write discovery**

Create `oidcverify/discovery.go`:

```go
package oidcverify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// maxResponseBytes caps what we will read from an issuer. A key set is a few
// kilobytes; anything approaching this is either broken or hostile.
const maxResponseBytes = 1 << 20 // 1 MiB

type discoveryDoc struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

// resolveJWKSURI returns the key set URI for iss, either the one configured
// directly or the one its discovery document names.
//
// Three constraints apply to a discovered URI, and each of them closes a way
// for a document we do not control to redirect key resolution: the transport
// must be HTTPS, the document's own issuer field must equal the registered
// issuer, and the key set must live on the issuer's host.
func (v *Verifier) resolveJWKSURI(ctx context.Context, iss Issuer) (string, error) {
	base := iss.normalisedURL()
	if err := v.checkScheme(base); err != nil {
		return "", err
	}

	if iss.JWKSURI != "" {
		if err := v.checkScheme(iss.JWKSURI); err != nil {
			return "", err
		}
		return iss.JWKSURI, nil
	}

	var doc discoveryDoc
	if err := v.fetchJSON(ctx, base+"/.well-known/openid-configuration", &doc); err != nil {
		return "", fmt.Errorf("%w: %w", ErrDiscoveryFailed, err)
	}

	if strings.TrimRight(doc.Issuer, "/") != base {
		return "", fmt.Errorf("%w: document declares issuer %q, expected %q",
			ErrDiscoveryFailed, doc.Issuer, base)
	}
	if doc.JWKSURI == "" {
		return "", fmt.Errorf("%w: document names no jwks_uri", ErrDiscoveryFailed)
	}
	if err := v.checkScheme(doc.JWKSURI); err != nil {
		return "", err
	}

	issURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("%w: unparseable issuer url: %w", ErrDiscoveryFailed, err)
	}
	jwksURL, err := url.Parse(doc.JWKSURI)
	if err != nil {
		return "", fmt.Errorf("%w: unparseable jwks_uri: %w", ErrDiscoveryFailed, err)
	}
	if jwksURL.Host != issURL.Host {
		return "", fmt.Errorf("%w: jwks_uri host %q is not the issuer host %q",
			ErrDiscoveryFailed, jwksURL.Host, issURL.Host)
	}

	return doc.JWKSURI, nil
}

// checkScheme refuses anything that is not HTTPS, unless the verifier was
// built with WithInsecureAllowHTTP, which exists for tests against httptest.
func (v *Verifier) checkScheme(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: unparseable url %q: %w", ErrDiscoveryFailed, raw, err)
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && v.allowHTTP {
		return nil
	}
	return fmt.Errorf("%w: %q is not https", ErrDiscoveryFailed, raw)
}

// fetchJSON performs a bounded GET and decodes the body into out.
func (v *Verifier) fetchJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("get %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // response body close

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: status %d", endpoint, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("read %s: %w", endpoint, err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s: %w", endpoint, err)
	}
	return nil
}
```

- [ ] **Step 4: Add the Verifier shell so this compiles**

Create `oidcverify/verify.go` with only the constructor for now. Task 4 fills in `Verify`.

```go
package oidcverify

import (
	"net/http"
	"sync"
	"time"
)

// Verifier verifies tokens from external OIDC issuers. It holds a key cache,
// so build one and keep it rather than one per request.
type Verifier struct {
	client    *http.Client
	allowHTTP bool
	now       func() time.Time

	keyTTL         time.Duration
	refreshCooldown time.Duration
	leeway         time.Duration

	mu    sync.Mutex
	cache map[string]*keyEntry
}

// Option configures a Verifier.
type Option func(*Verifier)

// WithHTTPClient replaces the default client. The default has a 10 second
// timeout; never pass http.DefaultClient, which has none.
func WithHTTPClient(c *http.Client) Option { return func(v *Verifier) { v.client = c } }

// WithKeyTTL sets how long a fetched key set is reused. Default 15 minutes.
func WithKeyTTL(d time.Duration) Option { return func(v *Verifier) { v.keyTTL = d } }

// WithRefreshCooldown bounds how often an unknown kid may trigger a refetch.
// Default 1 minute. Without it a flood of tokens carrying random kids becomes
// a request amplifier aimed at the issuer.
func WithRefreshCooldown(d time.Duration) Option {
	return func(v *Verifier) { v.refreshCooldown = d }
}

// WithLeeway sets the clock-skew allowance for exp, nbf and iat. Default 60s.
func WithLeeway(d time.Duration) Option { return func(v *Verifier) { v.leeway = d } }

// WithTimeFunc replaces the clock. Tests use it to age a token.
func WithTimeFunc(f func() time.Time) Option { return func(v *Verifier) { v.now = f } }

// WithInsecureAllowHTTP permits plain HTTP issuers. For tests against
// httptest only. Never enable this in a deployment.
func WithInsecureAllowHTTP() Option { return func(v *Verifier) { v.allowHTTP = true } }

// New builds a Verifier.
func New(opts ...Option) *Verifier {
	v := &Verifier{
		client:          &http.Client{Timeout: 10 * time.Second},
		now:             time.Now,
		keyTTL:          15 * time.Minute,
		refreshCooldown: time.Minute,
		leeway:          60 * time.Second,
		cache:           make(map[string]*keyEntry),
	}
	for _, o := range opts {
		o(v)
	}
	return v
}
```

- [ ] **Step 5: Run the test to verify it passes**

`keyEntry` is defined in `keys.go`, which Task 3 writes, so add a one-line
forward declaration at the bottom of `verify.go` to keep this task compiling
on its own. Task 3 deletes it in its Step 4.

```go
// keyEntry is replaced by the real type in keys.go. It exists so this task
// compiles and tests independently of the one after it.
type keyEntry struct{}
```

Run: `go test ./oidcverify/ -run TestResolveJWKSURI -v`

Expected: PASS, five tests.

- [ ] **Step 6: Commit**

```bash
git add oidcverify/discovery.go oidcverify/verify.go oidcverify/discovery_test.go
git commit -m "feat(oidcverify): discovery with https, issuer and host constraints"
```

---

### Task 3: The key cache

Keys come from the network, so they are cached. The cache has to hold a positive result, remember a negative one, and refuse to be used as an amplifier.

**Files:**
- Create: `oidcverify/keys.go`
- Modify: `oidcverify/verify.go` (delete the `keyEntry` placeholder from Task 2)
- Test: `oidcverify/keys_test.go`

**Interfaces:**
- Consumes: `resolveJWKSURI`, `fetchJSON` from Task 2.
- Produces: `func (v *Verifier) publicKey(ctx context.Context, iss Issuer, kid string) (crypto.PublicKey, error)` and the real `keyEntry` type.

- [ ] **Step 1: Write the failing test**

Create `oidcverify/keys_test.go`:

```go
package oidcverify

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jwksServer serves a JWKS containing one RSA key under the given kid, and
// counts how many times the key set was fetched.
func jwksServer(t *testing.T, kid string, pub *rsa.PublicKey, hits *int64) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": kid,
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	})
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func testIssuer(jwksURI string) Issuer {
	return Issuer{
		URL:               "https://issuer.example",
		JWKSURI:           jwksURI,
		Audience:          "aud",
		AllowedAlgorithms: []string{"RS256"},
	}
}

func TestPublicKey_FetchesAndCaches(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	var hits int64
	srv := jwksServer(t, "k1", &key.PublicKey, &hits)

	v := New(WithInsecureAllowHTTP())
	iss := testIssuer(srv.URL + "/keys")

	for i := 0; i < 3; i++ {
		got, err := v.publicKey(context.Background(), iss, "k1")
		require.NoError(t, err)
		require.NotNil(t, got)
	}
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits), "three lookups, one fetch")
}

func TestPublicKey_RefetchesAfterTTL(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	var hits int64
	srv := jwksServer(t, "k1", &key.PublicKey, &hits)

	now := time.Now()
	v := New(
		WithInsecureAllowHTTP(),
		WithKeyTTL(time.Minute),
		WithTimeFunc(func() time.Time { return now }),
	)
	iss := testIssuer(srv.URL + "/keys")

	_, err = v.publicKey(context.Background(), iss, "k1")
	require.NoError(t, err)
	now = now.Add(2 * time.Minute)
	_, err = v.publicKey(context.Background(), iss, "k1")
	require.NoError(t, err)

	assert.Equal(t, int64(2), atomic.LoadInt64(&hits), "expired entry is refetched")
}

// An unknown kid triggers at most one refresh per cooldown window, so a flood
// of tokens carrying random kids cannot be turned into an amplifier.
func TestPublicKey_UnknownKidRespectsCooldown(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	var hits int64
	srv := jwksServer(t, "k1", &key.PublicKey, &hits)

	now := time.Now()
	v := New(
		WithInsecureAllowHTTP(),
		WithRefreshCooldown(time.Minute),
		WithTimeFunc(func() time.Time { return now }),
	)
	iss := testIssuer(srv.URL + "/keys")

	for i := 0; i < 5; i++ {
		_, err := v.publicKey(context.Background(), iss, "does-not-exist")
		require.ErrorIs(t, err, ErrUnknownKey)
	}
	assert.LessOrEqual(t, atomic.LoadInt64(&hits), int64(2),
		"five unknown-kid lookups must not mean five fetches")

	now = now.Add(2 * time.Minute)
	_, err = v.publicKey(context.Background(), iss, "does-not-exist")
	require.ErrorIs(t, err, ErrUnknownKey)
	assert.Greater(t, atomic.LoadInt64(&hits), int64(1), "cooldown expiry allows another try")
}

// A token with no kid resolves when the issuer publishes exactly one key.
func TestPublicKey_EmptyKidUsesSoleKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	var hits int64
	srv := jwksServer(t, "k1", &key.PublicKey, &hits)

	v := New(WithInsecureAllowHTTP())
	got, err := v.publicKey(context.Background(), testIssuer(srv.URL+"/keys"), "")
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestPublicKey_FetchFailureIsReported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	v := New(WithInsecureAllowHTTP())
	_, err := v.publicKey(context.Background(), testIssuer(srv.URL+"/keys"), "k1")
	require.ErrorIs(t, err, ErrJWKSFailed)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./oidcverify/ -run TestPublicKey -v`

Expected: build failure naming `publicKey`.

- [ ] **Step 3: Write the cache**

Create `oidcverify/keys.go`:

```go
package oidcverify

import (
	"context"
	"crypto"
	"fmt"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

// keyEntry is one issuer's cached key set.
type keyEntry struct {
	keys        jose.JSONWebKeySet
	fetchedAt   time.Time
	lastAttempt time.Time
}

// publicKey returns the verification key for kid from iss's key set.
//
// An empty kid resolves only when the issuer publishes exactly one key, which
// is what issuers that omit kid rely on. Anything ambiguous is refused rather
// than guessed: picking a key by position would let an issuer rotation change
// which key a token verifies against.
func (v *Verifier) publicKey(ctx context.Context, iss Issuer, kid string) (crypto.PublicKey, error) {
	uri, err := v.resolveJWKSURI(ctx, iss)
	if err != nil {
		return nil, err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	entry := v.cache[uri]
	now := v.now()

	fresh := entry != nil && now.Sub(entry.fetchedAt) < v.keyTTL
	if !fresh {
		if err := v.refreshLocked(ctx, uri, now); err != nil {
			return nil, err
		}
		entry = v.cache[uri]
	}

	if key, ok := selectKey(entry.keys, kid); ok {
		return key, nil
	}

	// The kid is not in what we hold. It may be a rotation we have not seen,
	// so refetch, but only once per cooldown window.
	if now.Sub(entry.lastAttempt) >= v.refreshCooldown {
		if err := v.refreshLocked(ctx, uri, now); err != nil {
			return nil, err
		}
		if key, ok := selectKey(v.cache[uri].keys, kid); ok {
			return key, nil
		}
	}

	return nil, fmt.Errorf("%w: kid %q", ErrUnknownKey, kid)
}

// refreshLocked fetches the key set. The caller holds v.mu.
func (v *Verifier) refreshLocked(ctx context.Context, uri string, now time.Time) error {
	prev := v.cache[uri]
	var set jose.JSONWebKeySet
	if err := v.fetchJSON(ctx, uri, &set); err != nil {
		// Record the attempt so a failing issuer is not retried per request.
		if prev != nil {
			prev.lastAttempt = now
		} else {
			v.cache[uri] = &keyEntry{lastAttempt: now, fetchedAt: now.Add(-v.keyTTL)}
		}
		return fmt.Errorf("%w: %w", ErrJWKSFailed, err)
	}

	v.cache[uri] = &keyEntry{keys: set, fetchedAt: now, lastAttempt: now}
	return nil
}

// selectKey picks the key matching kid, or the only key when kid is empty.
func selectKey(set jose.JSONWebKeySet, kid string) (crypto.PublicKey, bool) {
	if kid != "" {
		for _, k := range set.Keys {
			if k.KeyID == kid {
				return k.Key, true
			}
		}
		return nil, false
	}
	if len(set.Keys) == 1 {
		return set.Keys[0].Key, true
	}
	return nil, false
}
```

- [ ] **Step 4: Delete the placeholder keyEntry**

In `oidcverify/verify.go`, remove:

```go
// keyEntry is filled in by keys.go in the next task.
type keyEntry struct{}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go mod tidy && go test ./oidcverify/ -run TestPublicKey -v`

Expected: PASS, five tests. `go mod tidy` promotes `go-jose` from indirect to direct.

- [ ] **Step 6: Commit**

```bash
git add oidcverify/keys.go oidcverify/verify.go oidcverify/keys_test.go go.mod go.sum
git commit -m "feat(oidcverify): jwks cache with ttl, negative caching and a refresh cooldown"
```

---

### Task 4: The verify pipeline

Everything above is machinery. This is the function callers use, and the table that proves it refuses what it should.

**Files:**
- Modify: `oidcverify/verify.go`
- Test: `oidcverify/verify_test.go`

**Interfaces:**
- Consumes: `inspect`, `Issuer.allows`, `Issuer.validate` from Task 1; `publicKey` from Task 3.
- Produces: `func (v *Verifier) Verify(ctx context.Context, iss Issuer, rawToken string) (*Claims, error)`. This is the whole public surface the plugin plan consumes.

- [ ] **Step 1: Write the failing test**

Create `oidcverify/verify_test.go`:

```go
package oidcverify

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	verifier *Verifier
	issuer   Issuer
	key      *rsa.PrivateKey
	now      time.Time
}

// newFixture stands up a signing key, a JWKS server and a verifier whose
// clock the test controls.
func newFixture(t *testing.T, opts ...Option) *fixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	var hits int64
	srv := jwksServer(t, "k1", &key.PublicKey, &hits)

	f := &fixture{key: key, now: time.Now()}
	base := []Option{
		WithInsecureAllowHTTP(),
		WithTimeFunc(func() time.Time { return f.now }),
	}
	f.verifier = New(append(base, opts...)...)
	f.issuer = Issuer{
		URL:               "https://issuer.example",
		JWKSURI:           srv.URL + "/keys",
		Audience:          "https://authsome.test/app_123",
		AllowedAlgorithms: []string{"RS256"},
	}
	return f
}

// sign builds a token from the given claims, signed with the fixture key.
func (f *fixture) sign(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, kid string) string {
	t.Helper()
	tok := jwt.NewWithClaims(method, claims)
	tok.Header["kid"] = kid

	var (
		s   string
		err error
	)
	if method == jwt.SigningMethodHS256 {
		s, err = tok.SignedString([]byte("attacker-chosen-symmetric-key-32b!!"))
	} else {
		s, err = tok.SignedString(f.key)
	}
	require.NoError(t, err)
	return s
}

// goodClaims is a token that must verify, so each negative case below can
// change exactly one thing.
func (f *fixture) goodClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss":        f.issuer.URL,
		"aud":        f.issuer.Audience,
		"sub":        "repo:acme/api:ref:refs/heads/main",
		"jti":        "token-1",
		"iat":        f.now.Add(-time.Minute).Unix(),
		"exp":        f.now.Add(10 * time.Minute).Unix(),
		"repository": "acme/api",
		"ref":        "refs/heads/main",
	}
}

func TestVerify_AcceptsAGoodToken(t *testing.T) {
	f := newFixture(t)
	raw := f.sign(t, jwt.SigningMethodRS256, f.goodClaims(), "k1")

	claims, err := f.verifier.Verify(context.Background(), f.issuer, raw)
	require.NoError(t, err)
	assert.Equal(t, f.issuer.URL, claims.Issuer)
	assert.Equal(t, "repo:acme/api:ref:refs/heads/main", claims.Subject)
	assert.Equal(t, "token-1", claims.ID)
	assert.Contains(t, claims.Audience, f.issuer.Audience)

	repo, ok := claims.String("repository")
	require.True(t, ok)
	assert.Equal(t, "acme/api", repo)
}

func TestVerify_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		method jwt.SigningMethod
		kid    string
		mutate func(c jwt.MapClaims)
		want   error
	}{
		{
			name:   "wrong audience",
			method: jwt.SigningMethodRS256,
			kid:    "k1",
			mutate: func(c jwt.MapClaims) { c["aud"] = "https://someone-else.test" },
			want:   ErrAudienceMismatch,
		},
		{
			name:   "wrong issuer",
			method: jwt.SigningMethodRS256,
			kid:    "k1",
			mutate: func(c jwt.MapClaims) { c["iss"] = "https://evil.example" },
			want:   ErrIssuerMismatch,
		},
		{
			name:   "expired",
			method: jwt.SigningMethodRS256,
			kid:    "k1",
			mutate: func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-2 * time.Hour).Unix() },
			want:   ErrTokenExpired,
		},
		{
			name:   "no expiry",
			method: jwt.SigningMethodRS256,
			kid:    "k1",
			mutate: func(c jwt.MapClaims) { delete(c, "exp") },
			want:   ErrMissingExpiry,
		},
		{
			name:   "unknown kid",
			method: jwt.SigningMethodRS256,
			kid:    "rotated-away",
			mutate: func(jwt.MapClaims) {},
			want:   ErrUnknownKey,
		},
		{
			name:   "algorithm not on the allowlist",
			method: jwt.SigningMethodHS256,
			kid:    "k1",
			mutate: func(jwt.MapClaims) {},
			want:   ErrAlgorithmNotAllowed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			claims := f.goodClaims()
			tc.mutate(claims)
			raw := f.sign(t, tc.method, claims, tc.kid)

			_, err := f.verifier.Verify(context.Background(), f.issuer, raw)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

// A token whose payload was edited after signing must not verify. This is the
// case the whole package exists for.
func TestVerify_RejectsTamperedPayload(t *testing.T) {
	f := newFixture(t)
	raw := f.sign(t, jwt.SigningMethodRS256, f.goodClaims(), "k1")

	// Flip a character in the payload segment.
	b := []byte(raw)
	dot := 0
	for i, c := range b {
		if c == '.' {
			dot = i
			break
		}
	}
	if b[dot+5] == 'a' {
		b[dot+5] = 'b'
	} else {
		b[dot+5] = 'a'
	}

	_, err := f.verifier.Verify(context.Background(), f.issuer, string(b))
	require.Error(t, err)
	assert.NotErrorIs(t, err, nil)
}

func TestVerify_RejectsGarbage(t *testing.T) {
	f := newFixture(t)
	_, err := f.verifier.Verify(context.Background(), f.issuer, "not-a-token")
	require.ErrorIs(t, err, ErrMalformedToken)
}

// MaxTokenAge is enforced separately from exp, because a platform can issue a
// token whose exp is hours away.
func TestVerify_RejectsTokenOlderThanMaxAge(t *testing.T) {
	f := newFixture(t)
	f.issuer.MaxTokenAge = 5 * time.Minute

	claims := f.goodClaims()
	claims["iat"] = f.now.Add(-time.Hour).Unix()
	claims["exp"] = f.now.Add(time.Hour).Unix()
	raw := f.sign(t, jwt.SigningMethodRS256, claims, "k1")

	_, err := f.verifier.Verify(context.Background(), f.issuer, raw)
	require.ErrorIs(t, err, ErrTokenTooOld)
}

func TestVerify_RejectsInvalidIssuerConfig(t *testing.T) {
	f := newFixture(t)
	raw := f.sign(t, jwt.SigningMethodRS256, f.goodClaims(), "k1")

	bad := f.issuer
	bad.Audience = ""
	_, err := f.verifier.Verify(context.Background(), bad, raw)
	require.ErrorIs(t, err, ErrInvalidIssuerConfig)
}

// The key set is fetched once across many verifications.
func TestVerify_ReusesCachedKeys(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	var hits int64
	srv := jwksServer(t, "k1", &key.PublicKey, &hits)

	now := time.Now()
	v := New(WithInsecureAllowHTTP(), WithTimeFunc(func() time.Time { return now }))
	iss := Issuer{
		URL:               "https://issuer.example",
		JWKSURI:           srv.URL + "/keys",
		Audience:          "aud",
		AllowedAlgorithms: []string{"RS256"},
	}

	for i := 0; i < 4; i++ {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss": iss.URL, "aud": iss.Audience, "sub": "s",
			"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
		})
		tok.Header["kid"] = "k1"
		raw, signErr := tok.SignedString(key)
		require.NoError(t, signErr)

		_, verifyErr := v.Verify(context.Background(), iss, raw)
		require.NoError(t, verifyErr)
	}

	assert.Equal(t, int64(1), atomic.LoadInt64(&hits))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./oidcverify/ -run TestVerify -v`

Expected: build failure naming `Verify`.

- [ ] **Step 3: Write the pipeline**

Append to `oidcverify/verify.go`:

```go
// Verify checks rawToken against iss and returns its claims.
//
// The order matters. Configuration is validated first, then the algorithm is
// checked against the allowlist, then the key is resolved, then the signature,
// and only then the claims. Nothing read from the token is trusted before the
// signature check except the algorithm and key id, which are used solely to
// decide what to verify with.
func (v *Verifier) Verify(ctx context.Context, iss Issuer, rawToken string) (*Claims, error) {
	if err := iss.validate(); err != nil {
		return nil, err
	}

	alg, kid, _, err := inspect(rawToken)
	if err != nil {
		return nil, err
	}
	if !iss.allows(alg) {
		return nil, fmt.Errorf("%w: %q", ErrAlgorithmNotAllowed, alg)
	}

	key, err := v.publicKey(ctx, iss, kid)
	if err != nil {
		return nil, err
	}

	claims := jwt.MapClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods(iss.AllowedAlgorithms),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(v.leeway),
		jwt.WithTimeFunc(v.now),
	)
	if _, err := parser.ParseWithClaims(rawToken, &claims, func(*jwt.Token) (any, error) {
		return key, nil
	}); err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, fmt.Errorf("%w: %w", ErrTokenExpired, err)
		case errors.Is(err, jwt.ErrTokenRequiredClaimMissing):
			return nil, fmt.Errorf("%w: %w", ErrMissingExpiry, err)
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, fmt.Errorf("%w: %w", ErrBadSignature, err)
		default:
			return nil, fmt.Errorf("%w: %w", ErrMalformedToken, err)
		}
	}

	out, err := toClaims(claims)
	if err != nil {
		return nil, err
	}

	// iss and aud are checked here rather than through parser options so the
	// failures carry this package's errors, which callers switch on.
	if strings.TrimRight(out.Issuer, "/") != iss.normalisedURL() {
		return nil, fmt.Errorf("%w: token says %q, expected %q",
			ErrIssuerMismatch, out.Issuer, iss.normalisedURL())
	}
	if !containsString(out.Audience, iss.Audience) {
		return nil, fmt.Errorf("%w: token audience %v does not contain %q",
			ErrAudienceMismatch, out.Audience, iss.Audience)
	}
	if out.ExpiresAt.IsZero() {
		return nil, ErrMissingExpiry
	}
	if iss.MaxTokenAge > 0 && !out.IssuedAt.IsZero() {
		if age := v.now().Sub(out.IssuedAt); age > iss.MaxTokenAge+v.leeway {
			return nil, fmt.Errorf("%w: issued %s ago, maximum is %s",
				ErrTokenTooOld, age, iss.MaxTokenAge)
		}
	}

	return out, nil
}

// toClaims converts the raw map into the typed result.
func toClaims(m jwt.MapClaims) (*Claims, error) {
	out := &Claims{All: make(map[string]any, len(m))}
	for k, val := range m {
		out.All[k] = val
	}

	out.Issuer, _ = m["iss"].(string)
	out.Subject, _ = m["sub"].(string)
	out.ID, _ = m["jti"].(string)

	if exp, err := m.GetExpirationTime(); err == nil && exp != nil {
		out.ExpiresAt = exp.Time
	}
	if iat, err := m.GetIssuedAt(); err == nil && iat != nil {
		out.IssuedAt = iat.Time
	}
	aud, err := m.GetAudience()
	if err != nil {
		return nil, fmt.Errorf("%w: unreadable aud claim: %w", ErrMalformedToken, err)
	}
	out.Audience = aud

	return out, nil
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
```

Extend the imports at the top of `oidcverify/verify.go`:

```go
import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./oidcverify/ -run TestVerify -v`

Expected: PASS. If the `algorithm not on the allowlist` case fails with `ErrUnknownKey` instead, the allowlist check has been placed after key resolution; move it before.

- [ ] **Step 5: Run the whole package with the race detector**

Run: `go test ./oidcverify/... -race -count=1`

Expected: PASS, every test.

- [ ] **Step 6: Confirm the repo still builds and is green**

Run: `go build ./... && go test ./... -race 2>&1 | grep -E "^FAIL" | head`

Expected: no `FAIL` lines. If `TestSignup_DuplicateRunsDummyHash` in `api` fails, that is a known pre-existing flake in a timing assertion and is not related to this work; re-run that package to confirm.

- [ ] **Step 7: Commit**

```bash
git add oidcverify/verify.go oidcverify/verify_test.go
git commit -m "feat(oidcverify): verify pipeline with algorithm, signature and claim gates"
```

---

## Notes for the executor

**What this package deliberately does not do.** It never decides which issuers are trusted. Steps 1 and 2 of the spec's pipeline, resolving a trust config by exact `iss` match scoped to app and environment, belong to the plugin because the plugin owns the store. If you find yourself wanting to pass a raw issuer string from a token into `Verify`, stop: that is the SSRF the ordering exists to prevent.

**Replay is not here either.** The spec prevents replay with a unique index on `(issuer_id, jti)` in the exchange record. This package surfaces `jti` as `Claims.ID` and does nothing else with it. There is no replay cache to build.

**`WithInsecureAllowHTTP` is test-only.** It exists because `httptest` serves plain HTTP. If it ever appears in a code path a deployment can reach, that is a bug.

**On the `go-jose` dependency.** It is already in the module graph as an indirect dependency, and Task 3 promotes it to direct. It is used for exactly one thing: turning a JWKS document into `crypto.PublicKey` values. The alternative was hand-decoding RSA `n`/`e` and EC `crv`/`x`/`y`, which `api/jwks_handler.go` already does in the encoding direction. Decoding is where curve confusion and unchecked key sizes live, so a reviewed library is worth the dependency.

**Next plan.** Phase 2 onward, the plugin core, consumes exactly one symbol from here: `func (v *Verifier) Verify(ctx context.Context, iss Issuer, rawToken string) (*Claims, error)`. Keep that signature stable.
