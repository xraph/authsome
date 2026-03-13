// Package dashboard provides shared dashboard utilities.
package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// nonceStore tracks form submission nonces to prevent duplicate submissions on refresh.
var nonceStore = struct {
	sync.Mutex
	nonces map[string]time.Time
}{nonces: make(map[string]time.Time)}

// GenerateNonce creates a random nonce and stores it for later validation.
func GenerateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	nonce := hex.EncodeToString(b)
	nonceStore.Lock()
	// Purge expired nonces (older than 10 minutes).
	cutoff := time.Now().Add(-10 * time.Minute)
	for k, t := range nonceStore.nonces {
		if t.Before(cutoff) {
			delete(nonceStore.nonces, k)
		}
	}
	nonceStore.nonces[nonce] = time.Now()
	nonceStore.Unlock()
	return nonce
}

// ConsumeNonce returns true if the nonce is valid and consumes it.
// A consumed nonce cannot be reused, preventing duplicate form submissions.
func ConsumeNonce(nonce string) bool {
	if nonce == "" {
		return false
	}
	nonceStore.Lock()
	defer nonceStore.Unlock()
	if _, ok := nonceStore.nonces[nonce]; ok {
		delete(nonceStore.nonces, nonce)
		return true
	}
	return false
}
