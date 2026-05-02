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

// Verify implements Verifier. Returns nil on success; ErrMissingToken if the
// token is empty; *VerifyError with provider codes on rejection;
// ErrTransientFailure on network/parse errors so callers can decide policy.
func (v *TurnstileVerifier) Verify(ctx context.Context, token, remoteIP, action string) error {
	if strings.TrimSpace(token) == "" {
		return ErrMissingToken
	}

	form := url.Values{}
	form.Set("secret", v.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	if action != "" {
		// Turnstile validates the action server-side via its policy; we still
		// pass it for symmetry with the API where supported.
		form.Set("action", action)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTransientFailure, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		// Preserve context errors so callers can detect cancellation/timeout.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%w: %v", ctxErr, err)
		}
		return fmt.Errorf("%w: %v", ErrTransientFailure, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: unexpected status %d", ErrTransientFailure, resp.StatusCode)
	}

	var out turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("%w: %v", ErrTransientFailure, err)
	}

	if out.Success {
		return nil
	}

	return classifyTurnstileCodes(out.ErrorCodes)
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
