package dashboard

import (
	"strings"
	"testing"
	"time"
)

func TestCSRFProtector_GenerateToken(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	sessionID := "test-session-123"
	token, err := protector.GenerateToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should not be empty
	if token == "" {
		t.Error("Generated token is empty")
	}

	// Token should have two parts separated by "."
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		t.Errorf("Token should have 2 parts, got %d", len(parts))
	}

	// Both parts should be base64-encoded
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		t.Error("Token parts should not be empty")
	}
}

func TestCSRFProtector_ValidateToken(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	sessionID := "test-session-123"

	// Generate a valid token
	token, err := protector.GenerateToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Test: Valid token should pass
	if !protector.ValidateToken(token, sessionID) {
		t.Error("Valid token failed validation")
	}

	// Test: Invalid token should fail
	if protector.ValidateToken("invalid-token", sessionID) {
		t.Error("Invalid token passed validation")
	}

	// Test: Valid token with wrong session should fail
	if protector.ValidateToken(token, "different-session") {
		t.Error("Token validated with wrong session ID")
	}

	// Test: Empty token should fail
	if protector.ValidateToken("", sessionID) {
		t.Error("Empty token passed validation")
	}

	// Test: Empty session ID should fail
	if protector.ValidateToken(token, "") {
		t.Error("Token validated with empty session ID")
	}
}

func TestCSRFProtector_TokenExpiration(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	// Override TTL for testing
	protector.tokenStore.ttl = 100 * time.Millisecond

	sessionID := "test-session-123"
	token, err := protector.GenerateToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should be valid immediately
	if !protector.ValidateToken(token, sessionID) {
		t.Error("Fresh token failed validation")
	}

	// Wait for token to expire
	time.Sleep(150 * time.Millisecond)

	// Token should now be invalid
	if protector.ValidateToken(token, sessionID) {
		t.Error("Expired token passed validation")
	}
}

func TestCSRFProtector_InvalidateToken(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	sessionID := "test-session-123"
	token, err := protector.GenerateToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should be valid
	if !protector.ValidateToken(token, sessionID) {
		t.Error("Token should be valid before invalidation")
	}

	// Invalidate token
	protector.InvalidateToken(token)

	// Token should now be invalid
	if protector.ValidateToken(token, sessionID) {
		t.Error("Invalidated token passed validation")
	}
}

func TestCSRFProtector_RotateSecret(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	sessionID := "test-session-123"

	// Generate token with old secret
	token1, err := protector.GenerateToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Rotate secret
	if err := protector.RotateSecret(); err != nil {
		t.Fatalf("Failed to rotate secret: %v", err)
	}

	// Old token should fail validation (signature won't match)
	// Note: Token is still in store, but HMAC verification will fail
	if protector.ValidateToken(token1, sessionID) {
		t.Error("Token from old secret should fail validation after rotation")
	}

	// Generate new token with new secret
	token2, err := protector.GenerateToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token after rotation: %v", err)
	}

	// New token should be valid
	if !protector.ValidateToken(token2, sessionID) {
		t.Error("New token should be valid after rotation")
	}
}

func TestCSRFProtector_Stats(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	// Initially empty
	stats := protector.Stats()
	if stats["total_tokens"].(int) != 0 {
		t.Error("Expected 0 tokens initially")
	}

	// Generate some tokens
	sessionID := "test-session-123"
	for i := 0; i < 5; i++ {
		_, err := protector.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
	}

	// Check stats
	stats = protector.Stats()
	if stats["total_tokens"].(int) != 5 {
		t.Errorf("Expected 5 tokens, got %d", stats["total_tokens"].(int))
	}
	if stats["valid_tokens"].(int) != 5 {
		t.Errorf("Expected 5 valid tokens, got %d", stats["valid_tokens"].(int))
	}
}

func TestCSRFProtector_CleanupExpiredTokens(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	// Override TTL for testing
	protector.tokenStore.ttl = 50 * time.Millisecond

	sessionID := "test-session-123"

	// Generate tokens
	for i := 0; i < 3; i++ {
		_, err := protector.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
	}

	// Wait for tokens to expire
	time.Sleep(100 * time.Millisecond)

	// Generate fresh tokens
	for i := 0; i < 2; i++ {
		_, err := protector.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
	}

	// Before cleanup: 5 total, 2 valid
	stats := protector.Stats()
	if stats["total_tokens"].(int) != 5 {
		t.Errorf("Expected 5 total tokens, got %d", stats["total_tokens"].(int))
	}
	if stats["valid_tokens"].(int) != 2 {
		t.Errorf("Expected 2 valid tokens, got %d", stats["valid_tokens"].(int))
	}

	// Trigger cleanup
	protector.CleanupExpiredTokens()

	// After cleanup: 2 total, 2 valid
	stats = protector.Stats()
	if stats["total_tokens"].(int) != 2 {
		t.Errorf("Expected 2 total tokens after cleanup, got %d", stats["total_tokens"].(int))
	}
	if stats["valid_tokens"].(int) != 2 {
		t.Errorf("Expected 2 valid tokens after cleanup, got %d", stats["valid_tokens"].(int))
	}
}

func TestCSRFProtector_SessionBinding(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	session1 := "session-1"
	session2 := "session-2"

	// Generate token for session 1
	token, err := protector.GenerateToken(session1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should work with session 1
	if !protector.ValidateToken(token, session1) {
		t.Error("Token should be valid for original session")
	}

	// Token should NOT work with session 2
	if protector.ValidateToken(token, session2) {
		t.Error("Token should not be valid for different session")
	}
}

func TestCSRFProtector_ConcurrentAccess(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	done := make(chan bool)
	errors := make(chan error, 100)

	// Spawn 10 goroutines generating tokens concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			sessionID := "session-" + string(rune(id))
			token, err := protector.GenerateToken(sessionID)
			if err != nil {
				errors <- err
				return
			}

			// Validate token
			if !protector.ValidateToken(token, sessionID) {
				errors <- err
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestCSRFProtector_TokenUniqueness(t *testing.T) {
	protector, err := NewCSRFProtector()
	if err != nil {
		t.Fatalf("Failed to create CSRF protector: %v", err)
	}

	sessionID := "test-session"
	tokens := make(map[string]bool)

	// Generate 100 tokens
	for i := 0; i < 100; i++ {
		token, err := protector.GenerateToken(sessionID)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// Check uniqueness
		if tokens[token] {
			t.Error("Generated duplicate token")
		}
		tokens[token] = true
	}

	// All 100 tokens should be unique
	if len(tokens) != 100 {
		t.Errorf("Expected 100 unique tokens, got %d", len(tokens))
	}
}

func BenchmarkCSRFProtector_Generate(b *testing.B) {
	protector, err := NewCSRFProtector()
	if err != nil {
		b.Fatalf("Failed to create CSRF protector: %v", err)
	}

	sessionID := "test-session-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := protector.GenerateToken(sessionID)
		if err != nil {
			b.Fatalf("Failed to generate token: %v", err)
		}
	}
}

func BenchmarkCSRFProtector_Validate(b *testing.B) {
	protector, err := NewCSRFProtector()
	if err != nil {
		b.Fatalf("Failed to create CSRF protector: %v", err)
	}

	sessionID := "test-session-123"
	token, err := protector.GenerateToken(sessionID)
	if err != nil {
		b.Fatalf("Failed to generate token: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protector.ValidateToken(token, sessionID)
	}
}

func BenchmarkCSRFProtector_GenerateAndValidate(b *testing.B) {
	protector, err := NewCSRFProtector()
	if err != nil {
		b.Fatalf("Failed to create CSRF protector: %v", err)
	}

	sessionID := "test-session-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token, err := protector.GenerateToken(sessionID)
		if err != nil {
			b.Fatalf("Failed to generate token: %v", err)
		}
		protector.ValidateToken(token, sessionID)
	}
}
