package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// turnstileEndpoint is Cloudflare's siteverify endpoint.
const turnstileEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// defaultTurnstileTimeout is the default HTTP timeout for the Turnstile
// verifier. Captcha verification sits on the auth path; we'd rather fail fast
// than hang the request.
const defaultTurnstileTimeout = 5 * time.Second

// TurnstileVerifier verifies Cloudflare Turnstile tokens.
//
// It is stateless and safe for concurrent use. Construct via
// NewTurnstileVerifier; the zero value is not usable.
type TurnstileVerifier struct {
	secret     string
	httpClient *http.Client
	endpoint   string // override for tests; defaults to Cloudflare's URL
}

// NewTurnstileVerifier returns a verifier configured with the given secret.
// httpClient may be nil — defaults to a client with a 5s timeout.
func NewTurnstileVerifier(secret string, httpClient *http.Client) *TurnstileVerifier {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTurnstileTimeout}
	}
	return &TurnstileVerifier{
		secret:     secret,
		httpClient: httpClient,
		endpoint:   turnstileEndpoint,
	}
}

// turnstileResponse is the JSON shape returned by Cloudflare's siteverify.
type turnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
	ErrorCodes  []string `json:"error-codes"`
}

// Verify implements Verifier.
//
// On success: returns (*Result, nil) with ChallengeTS, Hostname, and Action
// populated from the provider response. If the caller supplied a non-empty
// action and it does not match the response's action, returns
// (nil, *VerifyError{Codes: ["action-mismatch"]}).
//
// On failure: returns (nil, err) — ErrMissingToken if token is empty;
// ErrTransientFailure on network/parse/non-2xx; otherwise the classified
// sentinel for known Turnstile error codes or a *VerifyError preserving the
// provider codes.
//
// Note: Cloudflare's siteverify does NOT accept "action" as a request form
// field; the action is bound at widget render time and echoed back in the
// response. We therefore do not send it on the wire — we only compare the
// echoed value against what the caller expects.
func (v *TurnstileVerifier) Verify(ctx context.Context, token, remoteIP, action string) (*Result, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrMissingToken
	}

	form := url.Values{}
	form.Set("secret", v.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransientFailure, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		// Preserve context errors so callers can detect cancellation/timeout.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%w: %v", ctxErr, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrTransientFailure, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: unexpected status %d", ErrTransientFailure, resp.StatusCode)
	}

	var out turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransientFailure, err)
	}

	if !out.Success {
		return nil, classifyTurnstileCodes(out.ErrorCodes)
	}

	if action != "" && out.Action != action {
		return nil, &VerifyError{Codes: []string{"action-mismatch"}}
	}

	return &Result{
		Success:     true,
		ChallengeTS: out.ChallengeTS,
		Hostname:    out.Hostname,
		Action:      out.Action,
	}, nil
}

// classifyTurnstileCodes maps Turnstile error-codes to typed sentinels.
//
// Translation:
//   - missing-input-secret, invalid-input-secret -> *VerifyError (server config bug)
//   - missing-input-response                      -> ErrMissingToken
//   - invalid-input-response                      -> ErrInvalidToken
//   - bad-request                                 -> *VerifyError
//   - timeout-or-duplicate                        -> ErrDuplicateToken
//   - any other code (or empty)                   -> *VerifyError with codes preserved
func classifyTurnstileCodes(codes []string) error {
	for _, c := range codes {
		switch c {
		case "missing-input-response":
			return ErrMissingToken
		case "invalid-input-response":
			return ErrInvalidToken
		case "timeout-or-duplicate":
			return ErrDuplicateToken
		}
	}
	// Preserve original codes (including empty list) in a *VerifyError so
	// callers can inspect provider-specific reasons.
	return &VerifyError{Codes: append([]string(nil), codes...)}
}
