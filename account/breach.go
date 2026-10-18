package account

import (
	"context"
	// #nosec G505 -- SHA-1 imported solely for the HIBP k-Anonymity protocol; see account/breach.go.
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrPasswordBreached is returned when a password has been found in a data breach.
var ErrPasswordBreached = fmt.Errorf("%w: password found in data breach", ErrWeakPassword)

// ErrBreachCheckUnavailable is what IsBreachedContext returns when the
// lookup could not run: the request failed, HIBP answered something other
// than 200, or the body could not be read. The password is not reported as
// breached in that case. The transport's own error is not wrapped, because
// it quotes the request URL and the URL carries the hash prefix.
var ErrBreachCheckUnavailable = errors.New("account: breach check could not run")

// BreachChecker checks passwords against the HaveIBeenPwned Passwords API
// using the k-Anonymity model (only a 5-character SHA-1 prefix is sent).
type BreachChecker struct {
	client  *http.Client
	baseURL string
}

// BreachCheckTimeout bounds one HIBP range lookup.
const BreachCheckTimeout = 5 * time.Second

// NewBreachChecker creates a new breach checker with sensible defaults.
func NewBreachChecker() *BreachChecker {
	return NewBreachCheckerWithClient(nil)
}

// NewBreachCheckerWithClient is NewBreachChecker sending its lookups
// through client. A nil client is a plain one with BreachCheckTimeout.
func NewBreachCheckerWithClient(client *http.Client) *BreachChecker {
	if client == nil {
		client = &http.Client{Timeout: BreachCheckTimeout}
	}
	return &BreachChecker{
		client:  client,
		baseURL: "https://api.pwnedpasswords.com/range/",
	}
}

// IsBreached checks if a password appears in known data breaches.
// Returns true if the password hash appears in the HIBP database.
// On network errors, returns false (fail-open to avoid blocking auth).
func (bc *BreachChecker) IsBreached(password string) (bool, error) {
	breached, _ := bc.IsBreachedContext(context.Background(), password) //nolint:errcheck // IsBreached fails open and has always returned a nil error
	return breached, nil
}

// IsBreachedContext is IsBreached with the lookup made under ctx, so it
// carries ctx's values (its trace span among them) and stops when ctx does.
// It fails open too, reporting false, but says so: when the lookup could
// not run the error is ErrBreachCheckUnavailable, so a caller can count a
// control that did not run.
func (bc *BreachChecker) IsBreachedContext(ctx context.Context, password string) (bool, error) {
	// SHA-1 hash the password
	// #nosec G401 -- SHA-1 is mandated by the HIBP k-Anonymity range API; this is a breach lookup, not password storage.
	h := sha1.New()
	h.Write([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))

	prefix := hash[:5]
	suffix := hash[5:]

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bc.baseURL+prefix, http.NoBody)
	if err != nil {
		return false, ErrBreachCheckUnavailable
	}
	resp, err := bc.client.Do(req)
	if err != nil {
		// Fail open on network errors: don't block authentication.
		return false, ErrBreachCheckUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, ErrBreachCheckUnavailable
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB limit
	if err != nil {
		return false, ErrBreachCheckUnavailable
	}

	// Response format: SUFFIX:COUNT\r\n
	for _, line := range strings.Split(string(body), "\r\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == suffix {
			return true, nil
		}
	}

	return false, nil
}
