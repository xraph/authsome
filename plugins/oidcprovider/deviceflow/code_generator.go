package deviceflow

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// CodeGenerator generates device codes and user codes
type CodeGenerator struct {
	userCodeLength int
	userCodeFormat string
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(userCodeLength int, userCodeFormat string) *CodeGenerator {
	if userCodeLength <= 0 {
		userCodeLength = 8
	}
	if userCodeFormat == "" {
		userCodeFormat = "XXXX-XXXX"
	}
	return &CodeGenerator{
		userCodeLength: userCodeLength,
		userCodeFormat: userCodeFormat,
	}
}

// GenerateDeviceCode generates a long, secure device code (32+ bytes, URL-safe base64)
func (g *CodeGenerator) GenerateDeviceCode() (string, error) {
	// Generate 32 random bytes (256 bits)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate device code: %w", err)
	}

	// URL-safe base64 encoding without padding
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateUserCode generates a short, human-typable code
// Uses base20 charset (BCDFGHJKLMNPQRSTVWXZ) to avoid ambiguous characters
func (g *CodeGenerator) GenerateUserCode() (string, error) {
	// Base20 charset without ambiguous characters (no 0, O, I, 1, etc.)
	const charset = "BCDFGHJKLMNPQRSTVWXZ"

	// Count how many X's we need
	xCount := strings.Count(g.userCodeFormat, "X")
	if xCount == 0 {
		xCount = g.userCodeLength
	}

	// Generate random characters
	chars := make([]byte, xCount)
	randBytes := make([]byte, xCount)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("failed to generate user code: %w", err)
	}

	for i := 0; i < xCount; i++ {
		chars[i] = charset[int(randBytes[i])%len(charset)]
	}

	// Format the code according to the format string
	if strings.Contains(g.userCodeFormat, "X") {
		result := g.userCodeFormat
		for i := 0; i < xCount; i++ {
			result = strings.Replace(result, "X", string(chars[i]), 1)
		}
		return result, nil
	}

	// No format specified, just return the raw code
	return string(chars), nil
}

// ValidateUserCodeFormat validates the user code format string
func ValidateUserCodeFormat(format string) error {
	if format == "" {
		return fmt.Errorf("user code format cannot be empty")
	}

	xCount := strings.Count(format, "X")
	if xCount < 6 {
		return fmt.Errorf("user code format must contain at least 6 'X' placeholders")
	}
	if xCount > 12 {
		return fmt.Errorf("user code format must contain at most 12 'X' placeholders")
	}

	// Check for only allowed characters: X, -, space
	for _, ch := range format {
		if ch != 'X' && ch != '-' && ch != ' ' {
			return fmt.Errorf("user code format can only contain 'X', '-', and space characters")
		}
	}

	return nil
}
