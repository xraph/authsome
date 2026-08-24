// Package setjwt parses and validates RFC 8417 Security Event Tokens. It has
// no authsome dependencies. Errors are the RFC 8935 sentinels so a caller can
// turn a failure straight into a push-endpoint error body.
package setjwt

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MaxJTILength bounds the jti we are willing to store as a dedupe key.
const MaxJTILength = 255

// RFC 8935 error sentinels. ErrCode turns one into its wire code.
var (
	ErrInvalidRequest  = errors.New("setjwt: invalid_request")
	ErrInvalidKey      = errors.New("setjwt: invalid_key")
	ErrInvalidIssuer   = errors.New("setjwt: invalid_issuer")
	ErrInvalidAudience = errors.New("setjwt: invalid_audience")
)

// ErrCode maps an error to its RFC 8935 code, defaulting to invalid_request.
func ErrCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidKey):
		return "invalid_key"
	case errors.Is(err, ErrInvalidIssuer):
		return "invalid_issuer"
	case errors.Is(err, ErrInvalidAudience):
		return "invalid_audience"
	default:
		return "invalid_request"
	}
}

// allowedAlgs is the signature algorithm allow-list. HMAC is absent on
// purpose: accepting a symmetric algorithm alongside a published public key
// is the classic JWT confusion attack.
var allowedAlgs = map[string]struct{}{
	"RS256": {}, "RS384": {}, "RS512": {},
	"ES256": {}, "ES384": {}, "ES512": {},
	"EdDSA": {},
}

// KeyResolver hands back the public key for a kid.
type KeyResolver interface {
	Key(ctx context.Context, kid string) (crypto.PublicKey, error)
}

// Token is a validated SET.
type Token struct {
	Issuer   string
	Audience []string
	JTI      string
	IssuedAt time.Time
	Events   map[string]json.RawMessage
}

// Options configures Validate. Every field is required.
type Options struct {
	Issuer    string
	Audience  string
	Keys      KeyResolver
	Now       func() time.Time
	MaxAge    time.Duration
	ClockSkew time.Duration
	MaxEvents int
}

// Header reads kid, alg and typ without verifying anything. Use it only to
// pick a key and to reject an algorithm before doing real work.
func Header(raw []byte) (kid, alg, typ string, err error) {
	parser := jwt.NewParser()
	tok, _, err := parser.ParseUnverified(string(raw), jwt.MapClaims{})
	if err != nil {
		return "", "", "", fmt.Errorf("%w: parse header: %v", ErrInvalidRequest, err)
	}
	kid, _ = tok.Header["kid"].(string)
	alg, _ = tok.Header["alg"].(string)
	typ, _ = tok.Header["typ"].(string)
	return kid, alg, typ, nil
}

type setClaims struct {
	Issuer   string                     `json:"iss"`
	Audience any                        `json:"aud"`
	JTI      string                     `json:"jti"`
	IssuedAt int64                      `json:"iat"`
	Events   map[string]json.RawMessage `json:"events"`
}

func (setClaims) GetExpirationTime() (*jwt.NumericDate, error) { return nil, nil }
func (setClaims) GetIssuedAt() (*jwt.NumericDate, error)       { return nil, nil }
func (setClaims) GetNotBefore() (*jwt.NumericDate, error)      { return nil, nil }
func (c setClaims) GetIssuer() (string, error)                 { return c.Issuer, nil }
func (setClaims) GetSubject() (string, error)                  { return "", nil }
func (setClaims) GetAudience() (jwt.ClaimStrings, error)       { return nil, nil }

// Validate verifies a SET end to end and returns it. The claim checks run
// after signature verification, so nothing in the token is trusted until the
// key that the caller pinned has vouched for it.
func Validate(ctx context.Context, raw []byte, opts Options) (*Token, error) {
	kid, alg, typ, err := Header(raw)
	if err != nil {
		return nil, err
	}
	if typ != "secevent+jwt" {
		return nil, fmt.Errorf("%w: typ header must be secevent+jwt", ErrInvalidRequest)
	}
	if _, ok := allowedAlgs[alg]; !ok {
		return nil, fmt.Errorf("%w: algorithm %q is not accepted", ErrInvalidKey, alg)
	}

	key, err := opts.Keys.Key(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("%w: no key for kid", ErrInvalidKey)
	}

	var claims setClaims
	parser := jwt.NewParser(
		jwt.WithValidMethods(algList()),
		jwt.WithoutClaimsValidation(),
	)
	if _, err := parser.ParseWithClaims(string(raw), &claims,
		func(*jwt.Token) (any, error) { return key, nil }); err != nil {
		return nil, fmt.Errorf("%w: signature verification failed", ErrInvalidKey)
	}

	if claims.Issuer != opts.Issuer {
		return nil, fmt.Errorf("%w: issuer mismatch", ErrInvalidIssuer)
	}

	audience, err := normalizeAudience(claims.Audience)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAudience, err)
	}
	if !contains(audience, opts.Audience) {
		return nil, fmt.Errorf("%w: audience mismatch", ErrInvalidAudience)
	}

	if claims.JTI == "" {
		return nil, fmt.Errorf("%w: jti is required", ErrInvalidRequest)
	}
	if len(claims.JTI) > MaxJTILength {
		return nil, fmt.Errorf("%w: jti is too long", ErrInvalidRequest)
	}

	if claims.IssuedAt == 0 {
		return nil, fmt.Errorf("%w: iat is required", ErrInvalidRequest)
	}
	now := opts.Now()
	issued := time.Unix(claims.IssuedAt, 0)
	if issued.Before(now.Add(-opts.MaxAge)) {
		return nil, fmt.Errorf("%w: iat is too old", ErrInvalidRequest)
	}
	if issued.After(now.Add(opts.ClockSkew)) {
		return nil, fmt.Errorf("%w: iat is in the future", ErrInvalidRequest)
	}

	if len(claims.Events) == 0 {
		return nil, fmt.Errorf("%w: events must not be empty", ErrInvalidRequest)
	}
	if len(claims.Events) > opts.MaxEvents {
		return nil, fmt.Errorf("%w: too many events in one SET", ErrInvalidRequest)
	}

	return &Token{
		Issuer:   claims.Issuer,
		Audience: audience,
		JTI:      claims.JTI,
		IssuedAt: issued,
		Events:   claims.Events,
	}, nil
}

func algList() []string {
	out := make([]string, 0, len(allowedAlgs))
	for a := range allowedAlgs {
		out = append(out, a)
	}
	return out
}

func normalizeAudience(v any) ([]string, error) {
	switch t := v.(type) {
	case string:
		return []string{t}, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				return nil, errors.New("audience array holds a non-string")
			}
			out = append(out, s)
		}
		return out, nil
	case nil:
		return nil, errors.New("aud is required")
	default:
		return nil, errors.New("aud must be a string or an array of strings")
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
