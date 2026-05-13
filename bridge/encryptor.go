package bridge

// Encryptor encrypts and decrypts opaque byte payloads. Implementations
// must be deterministic about the format (envelope) so plaintext rows
// from before encryption was deployed can be migrated transparently.
//
// Concrete implementations:
//   - NoopEncryptor: identity. Suitable for tests / explicit-no-encryption
//     dev configs. Operators MUST NOT ship this in production.
//   - AESGCMEncryptor: AES-256-GCM with a versioned envelope. See aesgcm.go.
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// NoopEncryptor is a passthrough Encryptor. It returns the input unchanged
// for both Encrypt and Decrypt. Use only in tests or when an operator has
// explicitly opted out of at-rest encryption (NOT recommended in production).
type NoopEncryptor struct{}

// Encrypt returns plaintext unchanged.
func (NoopEncryptor) Encrypt(plaintext []byte) ([]byte, error) { return plaintext, nil }

// Decrypt returns ciphertext unchanged.
func (NoopEncryptor) Decrypt(ciphertext []byte) ([]byte, error) { return ciphertext, nil }
