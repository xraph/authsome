package authsome

import (
	"encoding/hex"
	"os"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
)

// envTokenEncryptionKey is the environment variable name that operators set
// to a 64-hex-char (32-byte) key. When unset, the engine falls back to a
// NoopEncryptor and logs a warning — production deployments MUST set this.
const envTokenEncryptionKey = "AUTHSOME_TOKEN_ENCRYPTION_KEY"

// resolveTokenEncryptor reads AUTHSOME_TOKEN_ENCRYPTION_KEY and constructs
// an AES-256-GCM Encryptor. If the env var is unset or invalid, it returns
// a NoopEncryptor and logs a warning rather than failing boot — this keeps
// dev environments friction-free while making the lack of encryption noisy
// for operators.
func resolveTokenEncryptor(logger log.Logger) bridge.Encryptor {
	raw := os.Getenv(envTokenEncryptionKey)
	if raw == "" {
		if logger != nil {
			logger.Warn("authsome: " + envTokenEncryptionKey + " is not set; OAuth provider tokens will be stored in plaintext. DO NOT ship this to production.")
		}
		return bridge.NoopEncryptor{}
	}
	key, err := hex.DecodeString(raw)
	if err != nil {
		if logger != nil {
			logger.Warn("authsome: " + envTokenEncryptionKey + " is not valid hex; falling back to plaintext. DO NOT ship this to production.")
		}
		return bridge.NoopEncryptor{}
	}
	enc, err := bridge.NewAESGCMEncryptor(key)
	if err != nil {
		if logger != nil {
			logger.Warn("authsome: " + envTokenEncryptionKey + " has invalid length (" + err.Error() + "); falling back to plaintext. DO NOT ship this to production.")
		}
		return bridge.NoopEncryptor{}
	}
	return enc
}
