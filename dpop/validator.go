package dpop

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Default freshness window.
//
// Lopsided on purpose. A proof dated in the future means somebody minted it
// ahead of time and deserves almost no slack. A proof dated in the past is
// nearly always a client with a drifting clock, and tightening that side buys
// support tickets rather than security.
const (
	DefaultIatLeewayPast   = 60 * time.Second
	DefaultIatLeewayFuture = 30 * time.Second
)

// NonceVerifier checks a server-issued nonce against the key it was minted
// for. NonceSigner implements it.
type NonceVerifier interface {
	Verify(jkt, nonce string) bool
}

// Expectation describes what a proof has to match for this request.
type Expectation struct {
	// Method is the HTTP method the request arrived on.
	Method string
	// URL is the request URL. Query and fragment are ignored during
	// comparison, so callers may pass the full URL.
	URL string
	// AccessToken is the token presented alongside the proof. Empty at the
	// token endpoint, where no token exists yet, and an ath claim is then
	// neither required nor checked.
	AccessToken string
	// ExpectedJKT is the thumbprint the presented token is bound to. Empty at
	// issuance, where the key is learned from the proof instead of checked.
	ExpectedJKT string
	// NonceRequired demands a valid server-issued nonce.
	NonceRequired bool
}

// Config configures a Validator.
type Config struct {
	IatLeewayPast   time.Duration
	IatLeewayFuture time.Duration
	Replay          ReplayCache
	Nonce           NonceVerifier
	// Now defaults to time.Now. Tests inject a fixed clock so iat boundaries
	// can be asserted exactly.
	Now func() time.Time
}

// Validator checks proofs against requests. It holds no HTTP types and no
// store, so its whole behaviour is reachable from a table-driven test.
type Validator struct {
	cfg Config
}

// NewValidator returns a Validator, filling in defaults for anything unset.
func NewValidator(cfg Config) *Validator {
	if cfg.IatLeewayPast <= 0 {
		cfg.IatLeewayPast = DefaultIatLeewayPast
	}
	if cfg.IatLeewayFuture <= 0 {
		cfg.IatLeewayFuture = DefaultIatLeewayFuture
	}
	if cfg.Replay == nil {
		cfg.Replay = NewMemoryReplayCache(0)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Validator{cfg: cfg}
}

// AccessTokenHash returns the ath claim value for a token: the base64url
// SHA-256 of its ASCII bytes.
func AccessTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// Validate checks a parsed proof against the request it arrived on.
//
// The jti replay check runs last, after every other check has passed. Recording
// it earlier would let an attacker burn a client's jti values using proofs that
// were going to be rejected anyway, turning a replay defence into a denial of
// service against the legitimate holder.
func (v *Validator) Validate(ctx context.Context, p *Proof, e Expectation) error {
	if p == nil {
		return ErrMalformedProof
	}

	if p.HTM != e.Method {
		return fmt.Errorf("%w: proof htm %q, request %q", ErrMethodMismatch, p.HTM, e.Method)
	}

	proofURI, err := normalizeHTU(p.HTU)
	if err != nil {
		return fmt.Errorf("%w: proof htu %q: %w", ErrURIMismatch, p.HTU, err)
	}
	requestURI, err := normalizeHTU(e.URL)
	if err != nil {
		return fmt.Errorf("%w: request url %q: %w", ErrURIMismatch, e.URL, err)
	}
	if proofURI != requestURI {
		return fmt.Errorf("%w: proof %q, request %q", ErrURIMismatch, proofURI, requestURI)
	}

	now := v.cfg.Now()
	if p.IssuedAt.After(now.Add(v.cfg.IatLeewayFuture)) {
		return fmt.Errorf("%w: iat is %s ahead", ErrStaleProof, p.IssuedAt.Sub(now))
	}
	if now.Sub(p.IssuedAt) > v.cfg.IatLeewayPast {
		return fmt.Errorf("%w: iat is %s old", ErrStaleProof, now.Sub(p.IssuedAt))
	}

	if e.NonceRequired {
		if p.Nonce == "" {
			return ErrNonceRequired
		}
		if v.cfg.Nonce == nil || !v.cfg.Nonce.Verify(p.JKT, p.Nonce) {
			return ErrNonceMismatch
		}
	}

	// ath binds the proof to one specific token. Without it a proof captured
	// from any request replays against every other endpoint for that client.
	if e.AccessToken != "" {
		want := AccessTokenHash(e.AccessToken)
		if subtle.ConstantTimeCompare([]byte(p.ATH), []byte(want)) != 1 {
			return ErrATHMismatch
		}
	}

	if e.ExpectedJKT != "" {
		if subtle.ConstantTimeCompare([]byte(p.JKT), []byte(e.ExpectedJKT)) != 1 {
			return fmt.Errorf("%w: proof key %s", ErrKeyMismatch, p.JKT)
		}
	}

	replayKey := p.JKT + ":" + p.JTI

	// A request that reaches two enforcement points presents one proof to
	// both. The second consultation would find the jti the first one recorded
	// and call the request a replay of itself, so a proof already accepted
	// under this request's scope skips the cache rather than re-recording it.
	// Only this last step is skipped; every check above has already run again
	// against whatever expectation this caller passed.
	scope := requestScopeFrom(ctx)
	if scope != nil && scope.alreadyAccepted(replayKey) {
		return nil
	}

	ttl := v.cfg.IatLeewayPast + v.cfg.IatLeewayFuture
	seen, err := v.cfg.Replay.Seen(ctx, replayKey, ttl)
	if err != nil {
		return fmt.Errorf("dpop: replay check: %w", err)
	}
	if seen {
		return fmt.Errorf("%w: jti %q", ErrReplayed, p.JTI)
	}

	if scope != nil {
		scope.accept(replayKey)
	}

	return nil
}

// normalizeHTU reduces a URL to scheme, host and path, lowercasing the scheme
// and host. RFC 9449 compares htu without query or fragment, so a client may
// vary its query string without reminting the proof.
func normalizeHTU(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("url %q is not absolute", raw)
	}
	return (&url.URL{
		Scheme: strings.ToLower(u.Scheme),
		Host:   strings.ToLower(u.Host),
		Path:   u.Path,
	}).String(), nil
}
