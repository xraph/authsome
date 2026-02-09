package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/xraph/authsome/internal/errs"
)

// EncryptionKey is the encryption key for provider credentials
// In production, this should be loaded from environment variables or a key management service.
var EncryptionKey = []byte("authsome-notification-key-32b") // Must be 32 bytes for AES-256

// EncryptConfig encrypts a provider configuration map.
func EncryptConfig(config map[string]any) (map[string]any, error) {
	encrypted := make(map[string]any)

	for key, value := range config {
		// Only encrypt sensitive fields
		if isSensitiveField(key) {
			if strValue, ok := value.(string); ok {
				encryptedValue, err := encryptString(strValue)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt field %s: %w", key, err)
				}

				encrypted[key] = encryptedValue
			} else {
				encrypted[key] = value
			}
		} else {
			encrypted[key] = value
		}
	}

	return encrypted, nil
}

// DecryptConfig decrypts a provider configuration map.
func DecryptConfig(config map[string]any) (map[string]any, error) {
	decrypted := make(map[string]any)

	for key, value := range config {
		// Only decrypt sensitive fields
		if isSensitiveField(key) {
			if strValue, ok := value.(string); ok {
				decryptedValue, err := decryptString(strValue)
				if err != nil {
					return nil, fmt.Errorf("failed to decrypt field %s: %w", key, err)
				}

				decrypted[key] = decryptedValue
			} else {
				decrypted[key] = value
			}
		} else {
			decrypted[key] = value
		}
	}

	return decrypted, nil
}

// isSensitiveField checks if a field should be encrypted.
func isSensitiveField(fieldName string) bool {
	sensitiveFields := map[string]bool{
		"password":      true,
		"api_key":       true,
		"apikey":        true,
		"apiKey":        true,
		"secret":        true,
		"auth_token":    true,
		"authToken":     true,
		"access_key":    true,
		"accessKey":     true,
		"secret_key":    true,
		"secretKey":     true,
		"private_key":   true,
		"privateKey":    true,
		"account_sid":   true,
		"accountSid":    true,
		"client_secret": true,
		"clientSecret":  true,
	}

	return sensitiveFields[fieldName]
}

// encryptString encrypts a string using AES-256-GCM.
func encryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Create cipher block
	block, err := aes.NewCipher(EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptString decrypts a string using AES-256-GCM.
func decryptString(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create cipher block
	block, err := aes.NewCipher(EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errs.BadRequest("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// SetEncryptionKey sets a custom encryption key
// This should be called at application startup with a key from environment.
func SetEncryptionKey(key []byte) error {
	if len(key) != 32 {
		return errs.InvalidInput("key", "encryption key must be exactly 32 bytes for AES-256")
	}

	EncryptionKey = key

	return nil
}

// GenerateEncryptionKey generates a new random encryption key.
func GenerateEncryptionKey() ([]byte, error) {
	key := make([]byte, 32) // 32 bytes = 256 bits for AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	return key, nil
}
