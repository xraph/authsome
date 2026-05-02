// Package captcha provides a small pluggable captcha verifier interface and a
// Cloudflare Turnstile implementation.
//
// Implementations are stateless and safe for concurrent use. Failures are
// classified into typed sentinels so callers can implement fail-open vs
// fail-closed policy without parsing provider responses.
package captcha

import (
	"context"
	"errors"
	"strings"
)

// Verifier validates a captcha challenge token. Implementations are stateless
// and safe for concurrent use.
type Verifier interface {
	// Verify returns a populated *Result and nil error on success. On failure
	// it returns (nil, err) — typically a sentinel (ErrMissingToken,
	// ErrInvalidToken, ErrDuplicateToken, ErrTransientFailure) or a wrapped
	// *VerifyError carrying provider-specific codes.
	//
	// The remoteIP is the client's IP (can be empty if unknown). The action
	// is an optional binding the caller expects: when non-empty, the verifier
	// compares it against the action echoed back by the provider and returns
	// *VerifyError{Codes: ["action-mismatch"]} if they differ.
	Verify(ctx context.Context, token, remoteIP, action string) (*Result, error)
}

// Result is the structured response from a verifier returned on success.
//
// For the Turnstile verifier, ChallengeTS, Hostname, and Action are populated
// from the provider response and are suitable for audit-log enrichment.
// ErrorCodes is unused on success (failures return (nil, err) instead).
type Result struct {
	Success     bool
	ChallengeTS string   // RFC3339 timestamp of the challenge solve
	Hostname    string   // hostname of the challenge issuer
	Action      string   // optional client-supplied action binding (echoed by provider)
	ErrorCodes  []string // provider-specific error codes (unused on success)
}

// VerifyError is returned when a verifier rejects a token. Codes are the
// provider-specific reasons. Use errors.As to extract.
type VerifyError struct {
	Codes []string
}

// Error implements error.
func (e *VerifyError) Error() string {
	if e == nil || len(e.Codes) == 0 {
		return "captcha: verification failed"
	}
	return "captcha: verification failed: " + strings.Join(e.Codes, ",")
}

// Common sentinel errors that map to recurring failure modes.
var (
	ErrMissingToken     = errors.New("captcha: missing token")
	ErrInvalidToken     = errors.New("captcha: invalid token")
	ErrDuplicateToken   = errors.New("captcha: token already used")
	ErrTransientFailure = errors.New("captcha: transient verifier failure")
)
