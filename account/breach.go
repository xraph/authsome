package account

import (
	"context"
	"crypto/sha1" //nolint:gosec // SHA-1 required by HIBP k-Anonymity API
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrPasswordBreached is returned when a password has been found in a data breach.
var ErrPasswordBreached = fmt.Errorf("%w: password found in data breach", ErrWeakPassword)

// BreachChecker checks passwords against the HaveIBeenPwned Passwords API
// using the k-Anonymity model (only a 5-character SHA-1 prefix is sent).
type BreachChecker struct {
	client  *http.Client
	baseURL string
}

// NewBreachChecker creates a new breach checker with sensible defaults.
func NewBreachChecker() *BreachChecker {
	return &BreachChecker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: "https://api.pwnedpasswords.com/range/",
	}
}

// IsBreached checks if a password appears in known data breaches.
// Returns true if the password hash appears in the HIBP database.
// On network errors, returns false (fail-open to avoid blocking auth).
func (bc *BreachChecker) IsBreached(password string) (bool, error) {
	// SHA-1 hash the password
	h := sha1.New() //nolint:gosec // SHA-1 required by HIBP k-Anonymity API protocol
	h.Write([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))

	prefix := hash[:5]
	suffix := hash[5:]

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, bc.baseURL+prefix, http.NoBody)
	if err != nil {
		return false, nil //nolint:nilerr // fail-open: don't block auth on request build errors
	}
	resp, err := bc.client.Do(req)
	if err != nil {
		// Fail open on network errors — don't block authentication.
		return false, nil //nolint:nilerr // fail-open: don't block auth on network errors
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB limit
	if err != nil {
		return false, nil //nolint:nilerr // fail-open: don't block auth on read errors
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
