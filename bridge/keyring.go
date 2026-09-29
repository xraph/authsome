package bridge

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// envelopeV2Prefix tags ciphertexts produced by a Keyring. The on-disk
// format names the key that sealed the value so a rotation can decrypt
// rows written under an earlier key:
//
//	"v2:" + kid + ":" + base64(nonce[12] || ciphertext || tag[16])
//
// A kid never contains ':' or ',', which keeps the envelope and the
// operator's key list unambiguous.
const envelopeV2Prefix = "v2:"

var (
	// ErrUnknownKeyID is returned when a v2 envelope names a key the ring
	// does not hold: the operator retired a key that still has rows.
	ErrUnknownKeyID = errors.New("bridge: keyring: envelope names an unknown key id")
	// ErrPlaintextRefused is returned by a strict encryptor asked to decrypt
	// a value that carries no envelope at all.
	ErrPlaintextRefused = errors.New("bridge: strict encryptor refuses a value with no envelope")
	// ErrBadKeyID is returned when a key id is empty or contains a
	// separator the envelope reserves.
	ErrBadKeyID = errors.New("bridge: keyring: key id must be non-empty and free of ':' and ','")
)

// Keyring is an Encryptor over several AES-256-GCM keys. New values are
// sealed under the writer key and tagged with its id; values sealed under
// any key in the ring decrypt, so a rotation is: add the new key as writer,
// keep the old one until every row has been rewritten, then drop it.
//
// Decrypt also accepts the single-key "v1:" envelope, trying each key in
// the ring until one authenticates, so a deployment that moves from
// AUTHSOME_TOKEN_ENCRYPTION_KEY to a ring keeps reading its rows. Values
// with no envelope pass through unchanged, the migration path for rows
// written before at-rest encryption was deployed; wrap the ring in
// NewStrict once none remain.
type Keyring struct {
	writer string
	keys   map[string]cipher.AEAD
	order  []string
}

// NewKeyring builds a ring from key ids to 32-byte keys. writer names the
// key that seals new values and must be in keys. order fixes the sequence
// in which v1 envelopes are tried; callers pass the ids in the order the
// operator listed them.
func NewKeyring(writer string, keys map[string][]byte, order []string) (*Keyring, error) {
	if len(keys) == 0 {
		return nil, errors.New("bridge: keyring: at least one key is required")
	}
	if _, ok := keys[writer]; !ok {
		return nil, fmt.Errorf("bridge: keyring: writer %q is not in the ring", writer)
	}
	ring := &Keyring{writer: writer, keys: make(map[string]cipher.AEAD, len(keys))}
	seen := make(map[string]bool, len(keys))
	for _, kid := range order {
		if _, ok := keys[kid]; ok && !seen[kid] {
			ring.order = append(ring.order, kid)
			seen[kid] = true
		}
	}
	for kid, key := range keys {
		if kid == "" || strings.ContainsAny(kid, ":,") {
			return nil, ErrBadKeyID
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("bridge: keyring: key %q: %w", kid, ErrInvalidKeyLength)
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("bridge: keyring: key %q: %w", kid, err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("bridge: keyring: key %q: %w", kid, err)
		}
		ring.keys[kid] = gcm
		if !seen[kid] {
			ring.order = append(ring.order, kid)
			seen[kid] = true
		}
	}
	return ring, nil
}

// Writer reports the id of the key that seals new values.
func (r *Keyring) Writer() string { return r.writer }

// Encrypt seals plaintext under the writer key.
func (r *Keyring) Encrypt(plaintext []byte) ([]byte, error) {
	gcm := r.keys[r.writer]
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("bridge: nonce read: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	buf := make([]byte, 0, len(nonce)+len(sealed))
	buf = append(buf, nonce...)
	buf = append(buf, sealed...)
	encoded := base64.StdEncoding.EncodeToString(buf)
	out := make([]byte, 0, len(envelopeV2Prefix)+len(r.writer)+1+len(encoded))
	out = append(out, envelopeV2Prefix...)
	out = append(out, r.writer...)
	out = append(out, ':')
	out = append(out, encoded...)
	return out, nil
}

// Decrypt opens a v2 envelope under the key it names, a v1 envelope under
// whichever ring key authenticates it, and passes anything else through as
// legacy plaintext.
func (r *Keyring) Decrypt(ciphertext []byte) ([]byte, error) {
	switch {
	case hasPrefix(ciphertext, envelopeV2Prefix):
		rest := ciphertext[len(envelopeV2Prefix):]
		sep := bytes.IndexByte(rest, ':')
		if sep <= 0 {
			return nil, errors.New("bridge: keyring: malformed v2 envelope")
		}
		kid := string(rest[:sep])
		gcm, ok := r.keys[kid]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownKeyID, kid)
		}
		return openEnvelope(gcm, rest[sep+1:])
	case hasPrefix(ciphertext, envelopeV1Prefix):
		body := ciphertext[len(envelopeV1Prefix):]
		var last error
		for _, kid := range r.order {
			pt, err := openEnvelope(r.keys[kid], body)
			if err == nil {
				return pt, nil
			}
			last = err
		}
		return nil, fmt.Errorf("bridge: keyring: no key opens this v1 envelope: %w", last)
	default:
		notePlaintextRead(ciphertext)
		return ciphertext, nil
	}
}

// openEnvelope decodes base64(nonce || ciphertext || tag) and opens it.
func openEnvelope(gcm cipher.AEAD, body []byte) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return nil, fmt.Errorf("bridge: aesgcm: invalid base64: %w", err)
	}
	ns := gcm.NonceSize()
	if len(raw) < ns+gcm.Overhead() {
		return nil, errors.New("bridge: aesgcm: ciphertext too short")
	}
	pt, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return nil, fmt.Errorf("bridge: aesgcm: open: %w", err)
	}
	return pt, nil
}

// StrictEncryptor wraps an Encryptor and refuses to decrypt a value that
// carries no envelope. Turn it on once every legacy plaintext row has been
// rewritten: from then on a value that reads back without an envelope is a
// sign of tampering or of a write that bypassed encryption, not a leftover.
type StrictEncryptor struct {
	Inner Encryptor
}

// NewStrict wraps enc so that unenveloped values are refused on Decrypt.
func NewStrict(enc Encryptor) Encryptor { return StrictEncryptor{Inner: enc} }

// Encrypt delegates to the wrapped Encryptor.
func (s StrictEncryptor) Encrypt(plaintext []byte) ([]byte, error) { return s.Inner.Encrypt(plaintext) }

// Decrypt refuses values without a recognised envelope and delegates the
// rest.
func (s StrictEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return ciphertext, nil
	}
	if !hasPrefix(ciphertext, envelopeV1Prefix) && !hasPrefix(ciphertext, envelopeV2Prefix) {
		return nil, ErrPlaintextRefused
	}
	return s.Inner.Decrypt(ciphertext)
}
