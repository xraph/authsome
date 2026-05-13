package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SignatureHeader is the canonical HTTP header carrying the signature
// envelope on outgoing webhook deliveries.
//
// Format (Stripe-style for operator familiarity):
//
//	X-Authsome-Signature: t=<unix-seconds>,v1=<hex(hmac-sha256)>
//
// The HMAC is computed over `<unix-seconds>.<raw-body-bytes>` so the
// signature also binds the timestamp — an attacker can't replay an old
// signature with a fresh timestamp without knowing the secret.
const SignatureHeader = "X-Authsome-Signature"

// SignatureVersion is the only currently-supported signature scheme.
// New schemes (e.g. ed25519) would land as v2 alongside v1 so receivers
// can roll forward without breaking existing senders.
const SignatureVersion = "v1"

// DefaultSignatureTolerance is the max clock skew allowed between
// sender and receiver before VerifySignature rejects with
// ErrTimestampSkew. Stripe uses 5 minutes; we match for familiarity.
const DefaultSignatureTolerance = 5 * time.Minute

// Errors returned by VerifySignature. Callers should match with
// errors.Is so the signaling stays stable across error wrapping.
var (
	ErrMalformedHeader   = errors.New("webhook: malformed signature header")
	ErrUnknownVersion    = errors.New("webhook: unknown signature version")
	ErrTimestampSkew     = errors.New("webhook: signature timestamp outside tolerance")
	ErrSignatureMismatch = errors.New("webhook: signature mismatch")
)

// SignBody computes the X-Authsome-Signature value for a given
// timestamp, raw request body, and shared secret. Senders compose
// this once per delivery and set the header on the outgoing request;
// receivers reproduce it under their own secret to verify.
//
// The timestamp is included in the header in plaintext AND folded
// into the HMAC input — this is intentional. A receiver that only
// MAC'd the body would accept a replayed (body, signature) pair with
// a fresh timestamp; binding the timestamp closes that gap.
func SignBody(secret []byte, ts time.Time, body []byte) string {
	tsStr := strconv.FormatInt(ts.Unix(), 10)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(tsStr))
	mac.Write([]byte("."))
	mac.Write(body)
	return "t=" + tsStr + "," + SignatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature returns nil when header was produced by SignBody
// using the same secret over the same body within tolerance of now.
// Pass tolerance=0 to use DefaultSignatureTolerance.
//
// Comparison is constant-time; failures are returned as wrapped
// sentinel errors so callers can distinguish "stale timestamp" (worth
// logging at debug, often a clock-drift symptom) from "MAC didn't
// match" (worth alerting — either a misconfigured secret or an
// attacker probing).
func VerifySignature(secret []byte, body []byte, header string, tolerance time.Duration) error {
	return verifySignatureAt(secret, body, header, tolerance, time.Now())
}

func verifySignatureAt(secret, body []byte, header string, tolerance time.Duration, now time.Time) error {
	if tolerance <= 0 {
		tolerance = DefaultSignatureTolerance
	}
	tsRaw, sigHex, err := parseSignatureHeader(header)
	if err != nil {
		return err
	}
	tsUnix, err := strconv.ParseInt(tsRaw, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: timestamp not an integer", ErrMalformedHeader)
	}
	delta := now.Sub(time.Unix(tsUnix, 0))
	if delta < 0 {
		delta = -delta
	}
	if delta > tolerance {
		return fmt.Errorf("%w: drift=%s tolerance=%s", ErrTimestampSkew, delta, tolerance)
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(tsRaw))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := mac.Sum(nil)

	got, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("%w: signature not hex", ErrMalformedHeader)
	}
	if !hmac.Equal(expected, got) {
		return ErrSignatureMismatch
	}
	return nil
}

// parseSignatureHeader splits "t=<ts>,v1=<hex>" into its parts. Order
// of the kv pairs is not enforced — Stripe places t first but the spec
// shouldn't break if a future version interleaves new keys.
//
// Unknown versions return ErrUnknownVersion rather than mismatch, so
// receivers can distinguish "this server speaks a newer protocol" from
// "this server's secret is wrong."
func parseSignatureHeader(header string) (ts, sig string, err error) {
	parts := strings.Split(header, ",")
	hasV1 := false
	for _, part := range parts {
		eq := strings.IndexByte(part, '=')
		if eq < 0 {
			return "", "", fmt.Errorf("%w: missing '=' in part %q", ErrMalformedHeader, part)
		}
		k := strings.TrimSpace(part[:eq])
		v := strings.TrimSpace(part[eq+1:])
		switch k {
		case "t":
			ts = v
		case SignatureVersion:
			sig = v
			hasV1 = true
		}
	}
	if ts == "" {
		return "", "", fmt.Errorf("%w: missing t=", ErrMalformedHeader)
	}
	if sig == "" {
		if !hasV1 {
			// Header parsed but no v1 — receiver speaks an older or
			// unknown version of the spec.
			return "", "", fmt.Errorf("%w: no recognized signature version (expected %s=)", ErrUnknownVersion, SignatureVersion)
		}
		return "", "", fmt.Errorf("%w: empty signature value", ErrMalformedHeader)
	}
	return ts, sig, nil
}
