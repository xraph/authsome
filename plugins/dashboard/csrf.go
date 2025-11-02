package dashboard

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	csrfTokenLength   = 32
	csrfSecretLength  = 32
	csrfTokenLifetime = 1 * time.Hour
)

// CSRFProtector provides production-grade CSRF protection
type CSRFProtector struct {
	secret     []byte
	mu         sync.RWMutex
	tokenStore *csrfTokenStore
}

// NewCSRFProtector creates a new CSRF protector with a random secret
func NewCSRFProtector() (*CSRFProtector, error) {
	secret := make([]byte, csrfSecretLength)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate CSRF secret: %w", err)
	}
	
	return &CSRFProtector{
		secret:     secret,
		tokenStore: newCSRFTokenStore(csrfTokenLifetime),
	}, nil
}

// csrfTokenStore stores issued tokens with expiration
type csrfTokenStore struct {
	mu      sync.RWMutex
	tokens  map[string]*csrfTokenEntry
	ttl     time.Duration
}

type csrfTokenEntry struct {
	sessionID string
	issuedAt  time.Time
	expiresAt time.Time
}

func newCSRFTokenStore(ttl time.Duration) *csrfTokenStore {
	s := &csrfTokenStore{
		tokens: make(map[string]*csrfTokenEntry),
		ttl:    ttl,
	}
	
	// Background cleanup
	go func() {
		ticker := time.NewTicker(ttl / 2)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanup()
		}
	}()
	
	return s
}

func (s *csrfTokenStore) add(token, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	s.tokens[token] = &csrfTokenEntry{
		sessionID: sessionID,
		issuedAt:  now,
		expiresAt: now.Add(s.ttl),
	}
}

func (s *csrfTokenStore) get(token string) (*csrfTokenEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	entry, exists := s.tokens[token]
	if !exists {
		return nil, false
	}
	
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}
	
	return entry, true
}

func (s *csrfTokenStore) remove(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

func (s *csrfTokenStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	for token, entry := range s.tokens {
		if now.After(entry.expiresAt) {
			delete(s.tokens, token)
		}
	}
}

// GenerateToken generates a new CSRF token for a session
// Format: base64(randomBytes) + "." + base64(hmac(randomBytes + sessionID))
func (c *CSRFProtector) GenerateToken(sessionID string) (string, error) {
	// Generate random bytes
	randomBytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	// Encode random bytes
	tokenData := base64.URLEncoding.EncodeToString(randomBytes)
	
	// Create HMAC signature
	mac := hmac.New(sha256.New, c.secret)
	mac.Write(randomBytes)
	mac.Write([]byte(sessionID))
	signature := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	
	// Combine token data and signature
	token := tokenData + "." + signature
	
	// Store token
	c.tokenStore.add(token, sessionID)
	
	return token, nil
}

// ValidateToken validates a CSRF token against a session
func (c *CSRFProtector) ValidateToken(token, sessionID string) bool {
	if token == "" || sessionID == "" {
		return false
	}
	
	// Split token into data and signature
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	
	tokenData := parts[0]
	providedSignature := parts[1]
	
	// Decode token data
	randomBytes, err := base64.URLEncoding.DecodeString(tokenData)
	if err != nil {
		return false
	}
	
	// Verify HMAC signature
	mac := hmac.New(sha256.New, c.secret)
	mac.Write(randomBytes)
	mac.Write([]byte(sessionID))
	expectedSignature := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	
	// Constant-time comparison
	if subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(providedSignature)) != 1 {
		return false
	}
	
	// Check if token exists in store and matches session
	entry, exists := c.tokenStore.get(token)
	if !exists {
		return false
	}
	
	if entry.sessionID != sessionID {
		return false
	}
	
	return true
}

// InvalidateToken removes a token from the store
func (c *CSRFProtector) InvalidateToken(token string) {
	c.tokenStore.remove(token)
}

// RotateSecret generates a new CSRF secret
// Should be called periodically for enhanced security
func (c *CSRFProtector) RotateSecret() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	newSecret := make([]byte, csrfSecretLength)
	if _, err := rand.Read(newSecret); err != nil {
		return fmt.Errorf("failed to generate new CSRF secret: %w", err)
	}
	
	c.secret = newSecret
	
	// Note: This will invalidate all existing tokens
	// In a production system, you might want to keep the old secret
	// for a grace period to allow existing tokens to remain valid
	
	return nil
}

// Stats returns statistics about the CSRF token store
func (c *CSRFProtector) Stats() map[string]interface{} {
	c.tokenStore.mu.RLock()
	defer c.tokenStore.mu.RUnlock()
	
	totalTokens := len(c.tokenStore.tokens)
	validTokens := 0
	now := time.Now()
	
	for _, entry := range c.tokenStore.tokens {
		if now.Before(entry.expiresAt) {
			validTokens++
		}
	}
	
	return map[string]interface{}{
		"total_tokens": totalTokens,
		"valid_tokens": validTokens,
		"expired_tokens": totalTokens - validTokens,
	}
}

// CleanupExpiredTokens manually triggers cleanup of expired tokens
func (c *CSRFProtector) CleanupExpiredTokens() {
	c.tokenStore.cleanup()
}

