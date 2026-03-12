package account

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params holds the parameters for Argon2id hashing.
type Argon2Params struct {
	Memory      uint32 // Memory in KiB (default: 64*1024 = 64 MiB)
	Iterations  uint32 // Time cost (default: 3)
	Parallelism uint8  // Parallelism factor (default: 2)
	SaltLength  uint32 // Salt length in bytes (default: 16)
	KeyLength   uint32 // Derived key length in bytes (default: 32)
}

// DefaultArgon2Params returns OWASP-recommended Argon2id parameters.
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Memory:      64 * 1024, // 64 MiB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// argon2Prefix identifies Argon2id hashes (PHC string format).
const argon2Prefix = "$argon2id$"

// HashPasswordArgon2 hashes a password using Argon2id and returns a PHC-format string.
func HashPasswordArgon2(password string, params Argon2Params) (string, error) {
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("account: generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)

	// Encode in PHC string format: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// CheckPasswordArgon2 verifies a password against an Argon2id PHC-format hash.
func CheckPasswordArgon2(encoded, password string) error {
	params, salt, hash, err := decodeArgon2Hash(encoded)
	if err != nil {
		return err
	}

	otherHash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)

	if subtle.ConstantTimeCompare(hash, otherHash) != 1 {
		return ErrInvalidCredentials
	}
	return nil
}

// IsArgon2Hash returns true if the hash string is an Argon2id hash.
func IsArgon2Hash(hash string) bool {
	return strings.HasPrefix(hash, argon2Prefix)
}

// decodeArgon2Hash parses a PHC-format Argon2id hash string.
func decodeArgon2Hash(encoded string) (params Argon2Params, salt []byte, hash []byte, err error) {
	// Expected format: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return Argon2Params{}, nil, nil, fmt.Errorf("account: invalid argon2 hash format")
	}

	var version int
	if _, scanErr := fmt.Sscanf(parts[2], "v=%d", &version); scanErr != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("account: invalid argon2 version: %w", scanErr)
	}
	if version != argon2.Version {
		return Argon2Params{}, nil, nil, fmt.Errorf("account: unsupported argon2 version %d", version)
	}

	if _, scanErr := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); scanErr != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("account: invalid argon2 params: %w", scanErr)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4]) //nolint:gosec // G115: length validated by argon2 output
	if err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("account: decode argon2 salt: %w", err)
	}
	params.SaltLength = uint32(len(salt)) //nolint:gosec // G115: length validated by argon2 output

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("account: decode argon2 hash: %w", err)
	}
	params.KeyLength = uint32(len(hash)) //nolint:gosec // G115: length validated by argon2 output

	return params, salt, hash, nil
}
