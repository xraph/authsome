package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateSecureToken generates a cryptographically secure random token.
func GenerateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
