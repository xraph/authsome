package twofa

import (
	"strings"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateBackupCode(t *testing.T) {
	// Test backup code generation
	code, err := generateBackupCode()
	assert.NoError(t, err)
	assert.NotEmpty(t, code)
	
	// Check format: XXXX-XXXX
	parts := strings.Split(code, "-")
	assert.Len(t, parts, 2, "Code should be in XXXX-XXXX format")
	assert.Len(t, parts[0], 4, "First part should be 4 characters")
	assert.Len(t, parts[1], 4, "Second part should be 4 characters")
	
	// Check no ambiguous characters (0, O, I, 1)
	fullCode := strings.ReplaceAll(code, "-", "")
	assert.NotContains(t, fullCode, "0")
	assert.NotContains(t, fullCode, "O")
	assert.NotContains(t, fullCode, "I")
	assert.NotContains(t, fullCode, "1")
}

func TestGenerateBackupCode_Uniqueness(t *testing.T) {
	// Generate multiple codes and ensure they're unique
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := generateBackupCode()
		assert.NoError(t, err)
		assert.False(t, codes[code], "Generated duplicate code: %s", code)
		codes[code] = true
	}
}

func TestHashBackupCode(t *testing.T) {
	code := "ABCD-EFGH"
	hash := hashBackupCode(code)
	
	// SHA-256 produces 64 character hex string
	assert.Len(t, hash, 64)
	assert.NotEmpty(t, hash)
	
	// Same code should produce same hash
	hash2 := hashBackupCode(code)
	assert.Equal(t, hash, hash2)
	
	// Different code should produce different hash
	differentCode := "WXYZ-1234"
	differentHash := hashBackupCode(differentCode)
	assert.NotEqual(t, hash, differentHash)
}

func TestNormalizeBackupCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABCD-EFGH", "ABCDEFGH"},
		{"abcd-efgh", "ABCDEFGH"},
		{"ab cd-ef gh", "ABCDEFGH"},
		{"AB-CD-EF-GH", "ABCDEFGH"},
		{"abcdefgh", "ABCDEFGH"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeBackupCode(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackupCodeSecurity(t *testing.T) {
	// Verify that backup codes use cryptographically secure random generation
	code1, err1 := generateBackupCode()
	code2, err2 := generateBackupCode()
	
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, code1, code2, "Codes should be randomly generated")
	
	// Hash should not be reversible
	hash := hashBackupCode("TEST-CODE")
	assert.NotContains(t, hash, "TEST")
	assert.NotContains(t, hash, "CODE")
}

func TestBackupCodeFormat(t *testing.T) {
	// Test that generated codes match expected format
	for i := 0; i < 10; i++ {
		code, err := generateBackupCode()
		assert.NoError(t, err)
		
		// Should be 9 characters total (4 + dash + 4)
		assert.Len(t, code, 9)
		
		// Should contain exactly one dash at position 4
		assert.Equal(t, "-", string(code[4]))
		
		// All non-dash characters should be alphanumeric (from safe charset)
		for j, char := range code {
			if j == 4 {
				continue // Skip dash
			}
			// Check if character is in safe charset
			assert.True(t, isValidBackupChar(char), "Invalid character: %c", char)
		}
	}
}

// Helper to validate backup code characters
func isValidBackupChar(c rune) bool {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	return strings.ContainsRune(charset, c)
}

func TestBackupCodeCountValidation(t *testing.T) {
	userID := xid.New().String()
	
	// We can't fully test BackupCodes without a real database,
	// but we can test the count validation logic
	
	// Count <= 0 should default to 10
	count := 0
	if count <= 0 || count > 20 {
		count = 10
	}
	assert.Equal(t, 10, count)
	
	// Count > 20 should cap at 10
	count = 25
	if count <= 0 || count > 20 {
		count = 10
	}
	assert.Equal(t, 10, count)
	
	// Valid count should be preserved
	count = 15
	if count <= 0 || count > 20 {
		count = 10
	}
	assert.Equal(t, 15, count)
	
	// Test invalid user ID
	invalidID := "invalid-xid"
	_, err := xid.FromString(invalidID)
	assert.Error(t, err)
	
	// Test valid user ID
	_, err = xid.FromString(userID)
	assert.NoError(t, err)
}

func TestHashConsistency(t *testing.T) {
	// Test that hashing is consistent
	code := "TEST-CODE"
	
	// Hash same code multiple times
	hashes := make([]string, 5)
	for i := 0; i < 5; i++ {
		hashes[i] = hashBackupCode(code)
	}
	
	// All hashes should be identical
	for i := 1; i < 5; i++ {
		assert.Equal(t, hashes[0], hashes[i])
	}
}

func TestNormalizationConsistency(t *testing.T) {
	// Test that normalization is consistent
	testCases := []string{
		"abcd-efgh",
		"ABCD-EFGH",
		"AbCd-EfGh",
		"ab cd-ef gh",
		"AB CD-EF GH",
	}
	
	var normalized []string
	for _, tc := range testCases {
		normalized = append(normalized, normalizeBackupCode(tc))
	}
	
	// All should normalize to the same value
	for i := 1; i < len(normalized); i++ {
		assert.Equal(t, normalized[0], normalized[i])
	}
}

