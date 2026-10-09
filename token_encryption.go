package authsome

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
)

// Environment variables that configure encryption of provider tokens and
// connection secrets at rest. Config.TokenEncryption takes precedence over
// all of them.
//
// #nosec G101 -- not credentials: environment variable names.
const (
	// envTokenEncryptionKeys is a comma-separated list of "kid:hex" pairs;
	// the first is the key that seals new values and every key decrypts.
	envTokenEncryptionKeys = "AUTHSOME_TOKEN_ENCRYPTION_KEYS"
	// envTokenEncryptionKey is the single 64-hex-char (32-byte) key form.
	envTokenEncryptionKey = "AUTHSOME_TOKEN_ENCRYPTION_KEY"
	// envTokenEncryptionStrict, when "true", refuses to decrypt a stored
	// value that carries no envelope.
	envTokenEncryptionStrict = "AUTHSOME_TOKEN_ENCRYPTION_STRICT"
)

// ErrTokenEncryptionKeyRequired is returned by NewEngine when no encryption
// key is configured and no Encryptor was passed. Provider tokens and SSO
// secrets are never written in the clear by accident: an operator who wants
// that in a development setup passes WithTokenEncryptor(bridge.NoopEncryptor{}).
var ErrTokenEncryptionKeyRequired = errors.New("authsome: token encryption key is required: set " +
	envTokenEncryptionKeys + " (kid:hex,kid:hex) or " + envTokenEncryptionKey +
	" (64 hex chars), or pass WithTokenEncryptor")

// TokenEncryptionConfig configures at-rest encryption of provider tokens,
// MFA secrets and SSO connection secrets.
type TokenEncryptionConfig struct {
	// Keys is a comma-separated list of "kid:hex" pairs, hex being 64
	// characters (32 bytes). The first pair seals new values; every pair
	// decrypts, which is how a key is rotated: add the new one in front,
	// rewrite rows, then drop the old one. Falls back to the environment
	// when empty.
	Keys string `json:"keys"`
	// Strict refuses to decrypt a stored value with no envelope. Leave it
	// off until every row written before encryption was deployed has been
	// rewritten; a plaintext read after that is a fault, not a leftover.
	Strict bool `json:"strict"`
}

// resolveTokenEncryptor builds the Encryptor from configuration and the
// environment. allowMissing lets a missing key resolve to the no-op
// encryptor; the engine passes it only under `go test`, so production boots
// fail closed rather than storing secrets in the clear.
func resolveTokenEncryptor(cfg TokenEncryptionConfig, logger log.Logger, allowMissing bool) (bridge.Encryptor, error) {
	var enc bridge.Encryptor
	switch {
	case strings.TrimSpace(cfg.Keys) != "":
		ring, err := parseKeyring(cfg.Keys)
		if err != nil {
			return nil, fmt.Errorf("authsome: token_encryption.keys: %w", err)
		}
		enc = ring
	case strings.TrimSpace(os.Getenv(envTokenEncryptionKeys)) != "":
		ring, err := parseKeyring(os.Getenv(envTokenEncryptionKeys))
		if err != nil {
			return nil, fmt.Errorf("authsome: %s: %w", envTokenEncryptionKeys, err)
		}
		enc = ring
	case strings.TrimSpace(os.Getenv(envTokenEncryptionKey)) != "":
		key, err := hex.DecodeString(strings.TrimSpace(os.Getenv(envTokenEncryptionKey)))
		if err != nil {
			return nil, fmt.Errorf("authsome: %s is not valid hex: %w", envTokenEncryptionKey, err)
		}
		single, err := bridge.NewAESGCMEncryptor(key)
		if err != nil {
			return nil, fmt.Errorf("authsome: %s: %w", envTokenEncryptionKey, err)
		}
		enc = single
	default:
		if !allowMissing {
			return nil, ErrTokenEncryptionKeyRequired
		}
		if logger != nil {
			logger.Warn("authsome: no token encryption key configured; provider tokens are stored in the clear because this is a test process")
		}
		return bridge.NoopEncryptor{}, nil
	}
	if cfg.Strict || strings.EqualFold(strings.TrimSpace(os.Getenv(envTokenEncryptionStrict)), "true") {
		enc = bridge.NewStrict(enc)
	}
	return enc, nil
}

// parseKeyring turns "kid:hex,kid:hex" into a Keyring whose writer is the
// first entry.
func parseKeyring(spec string) (*bridge.Keyring, error) {
	keys := make(map[string][]byte)
	var order []string
	for _, entry := range strings.Split(spec, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		kid, hexKey, ok := strings.Cut(entry, ":")
		if !ok || strings.TrimSpace(kid) == "" {
			return nil, fmt.Errorf("entry %q must be kid:hex", entry)
		}
		kid = strings.TrimSpace(kid)
		key, err := hex.DecodeString(strings.TrimSpace(hexKey))
		if err != nil {
			return nil, fmt.Errorf("key %q is not valid hex: %w", kid, err)
		}
		if _, dup := keys[kid]; dup {
			return nil, fmt.Errorf("key id %q is listed twice", kid)
		}
		keys[kid] = key
		order = append(order, kid)
	}
	if len(order) == 0 {
		return nil, errors.New("no keys listed")
	}
	return bridge.NewKeyring(order[0], keys, order)
}
