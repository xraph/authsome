// Cookie-forwarding refresh, for services running in client mode.
//
// This file is hand-written. client.go is regenerated from the OpenAPI spec on
// every `make -C sdkgen generate` and anything added there is silently deleted,
// so companion files like this one are where hand-maintained calls belong (same
// rule as client_scoped.go).

package authclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RefreshedSession is the result of a cookie-forwarded refresh: the rotated
// tokens plus the raw Set-Cookie headers the identity server issued alongside
// them.
//
// SetCookies matters as much as the tokens. The identity server resolves the
// session cookie's name, domain, path, Secure, HttpOnly, SameSite and the
// __Host- prefix opt-in from its own dynamic settings cascade, and a
// client-mode service has no access to that cascade — it holds no settings
// manager and no engine. Replaying these headers verbatim to the browser keeps
// the identity server the single source of truth for cookie attributes instead
// of making every downstream service guess at them.
type RefreshedSession struct {
	SessionToken string
	RefreshToken string
	ExpiresAt    time.Time

	// SetCookies holds the response's Set-Cookie header values, in order.
	SetCookies []string
}

// RefreshTokensWithCookies rotates the session carried by cookieHeader and
// returns the new tokens together with the identity server's Set-Cookie
// headers.
//
// POST /v1/refresh is cookie-first: when the request carries a valid session
// cookie, the server rotates using that session's *current* server-side
// refresh token rather than one supplied in the body (api/auth_handlers.go,
// handleRefresh). That is what makes this call possible from a client-mode
// service, which sees the user's access token but never their refresh token —
// forwarding the browser's own Cookie header is enough, and no request body is
// needed.
//
// Two deliberate differences from the generated RefreshTokens:
//
//   - The call carries NO service credentials. /v1/refresh is registered with
//     auth "none", and sending this service's API key alongside a user cookie
//     would only muddy which principal the server is acting for. The cookie is
//     the whole credential.
//   - Response headers are returned rather than discarded, for the reason on
//     RefreshedSession.SetCookies above.
//
// cookieHeader is the raw Cookie header from the inbound browser request. An
// empty cookieHeader is refused before any HTTP call: without it the server
// would fall through to its body-token branch and answer 400, which is a
// round-trip spent to learn something already known here.
func (c *Client) RefreshTokensWithCookies(ctx context.Context, cookieHeader string) (*RefreshedSession, error) {
	if strings.TrimSpace(cookieHeader) == "" {
		return nil, fmt.Errorf("refresh: no cookies to forward")
	}

	// Empty body. The cookie-first branch never reads it, and sending a stale
	// refresh_token here would be actively harmful: it trips replay detection,
	// which cascade-revokes the whole token family and logs the user out —
	// precisely the failure this middleware exists to prevent.
	body, err := json.Marshal(struct{}{})
	if err != nil {
		return nil, fmt.Errorf("refresh: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/refresh", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("refresh: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", cookieHeader)

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10)) //nolint:errcheck // best-effort: the status code is the signal
		var env struct {
			Error   string `json:"error"`
			Message string `json:"message"`
			Details string `json:"details"`
		}
		msg := ""
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &env) //nolint:errcheck // best-effort: a non-JSON error body just leaves msg empty, and RawBody still carries it
			msg = firstNonEmptyClientErrorMessage(env.Error, env.Message, env.Details)
		}
		return nil, &ClientError{
			StatusCode: resp.StatusCode,
			Message:    msg,
			RawBody:    raw,
			Headers: map[string]string{
				"Content-Type": resp.Header.Get("Content-Type"),
				"X-Request-ID": resp.Header.Get("X-Request-ID"),
			},
		}
	}

	var decoded struct {
		SessionToken string `json:"session_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresAt    string `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("refresh: decode response: %w", err)
	}
	if decoded.SessionToken == "" {
		return nil, fmt.Errorf("refresh: server returned no session token")
	}

	out := &RefreshedSession{
		SessionToken: decoded.SessionToken,
		RefreshToken: decoded.RefreshToken,
		SetCookies:   resp.Header.Values("Set-Cookie"),
	}
	// A server too old to send expires_at, or one that changes the format,
	// leaves the zero time rather than failing the refresh: the rotated token
	// is still good, and the caller treats a zero expiry as "unknown".
	if decoded.ExpiresAt != "" {
		if parsed, perr := time.Parse(time.RFC3339, decoded.ExpiresAt); perr == nil {
			out.ExpiresAt = parsed
		}
	}
	return out, nil
}
