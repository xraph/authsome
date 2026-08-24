package dpop

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
	"time"
)

// DefaultNonceTTL is how long an issued nonce stays acceptable.
const DefaultNonceTTL = 5 * time.Minute

// minNonceSecretBytes keeps a misconfigured secret loud rather than weak.
const minNonceSecretBytes = 16

// ErrNonceSecretMissing is returned when the signing secret is absent or too
// short.
//
// Callers must treat this as fatal when nonces are switched on. Engine
// .NonceSecret returns nil with no HMAC JWT key and no environment override,
// and falling back to a per-process random secret would mint nonces that no
// other replica can verify: a security feature that presents as an
// intermittent outage the moment you scale past one instance.
var ErrNonceSecretMissing = errors.New("dpop: nonce signing secret not configured")

// NonceSigner mints and verifies server nonces without storing them.
//
// The construction follows dashboard/nonce.go, with one deliberate difference:
// a DPoP nonce is NOT single use. Consuming it on first sight would break the
// client on its second request, because one nonce covers every request made
// during its lifetime. Do not reach for nonceSigner.Consume here.
type NonceSigner struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

var _ NonceVerifier = (*NonceSigner)(nil)

// NewNonceSigner returns a signer, or ErrNonceSecretMissing if the secret is
// unusable.
func NewNonceSigner(secret []byte, ttl time.Duration) (*NonceSigner, error) {
	if len(secret) < minNonceSecretBytes {
		return nil, ErrNonceSecretMissing
	}
	if ttl <= 0 {
		ttl = DefaultNonceTTL
	}
	cp := make([]byte, len(secret))
	copy(cp, secret)
	return &NonceSigner{secret: cp, ttl: ttl, now: time.Now}, nil
}

// Issue returns a nonce bound to jkt at the current time.
func (s *NonceSigner) Issue(jkt string) string {
	if jkt == "" {
		return ""
	}
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(s.now().Unix()))
	mac := s.sign(tsBytes, jkt)
	return base64.RawURLEncoding.EncodeToString(tsBytes) + "." +
		base64.RawURLEncoding.EncodeToString(mac)
}

// Verify reports whether nonce was issued by this signer for jkt and is still
// inside its TTL. It does not consume the nonce, so it may return true many
// times for the same value.
func (s *NonceSigner) Verify(jkt, nonce string) bool {
	tsBytes, macBytes, ok := splitNonce(nonce)
	if !ok || jkt == "" {
		return false
	}

	issued := time.Unix(int64(binary.BigEndian.Uint64(tsBytes)), 0)
	now := s.now()
	if issued.After(now.Add(DefaultIatLeewayFuture)) {
		return false
	}
	if now.Sub(issued) > s.ttl {
		return false
	}

	return hmac.Equal(macBytes, s.sign(tsBytes, jkt))
}

// NeedsRefresh reports whether a nonce is past half its TTL, so responses can
// carry a replacement before the client's current one expires. An unparseable
// nonce always needs replacing.
func (s *NonceSigner) NeedsRefresh(nonce string) bool {
	tsBytes, _, ok := splitNonce(nonce)
	if !ok {
		return true
	}
	issued := time.Unix(int64(binary.BigEndian.Uint64(tsBytes)), 0)
	return s.now().Sub(issued) > s.ttl/2
}

func splitNonce(nonce string) (tsBytes, macBytes []byte, ok bool) {
	dot := strings.IndexByte(nonce, '.')
	if dot <= 0 || dot == len(nonce)-1 {
		return nil, nil, false
	}
	tsBytes, err := base64.RawURLEncoding.DecodeString(nonce[:dot])
	if err != nil || len(tsBytes) != 8 {
		return nil, nil, false
	}
	macBytes, err = base64.RawURLEncoding.DecodeString(nonce[dot+1:])
	if err != nil || len(macBytes) != sha256.Size {
		return nil, nil, false
	}
	return tsBytes, macBytes, true
}

// sign length-prefixes jkt so a future caller passing structured data cannot
// construct a collision by moving a separator.
func (s *NonceSigner) sign(tsBytes []byte, jkt string) []byte {
	h := hmac.New(sha256.New, s.secret)
	h.Write(tsBytes)
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(jkt)))
	h.Write(lenBuf[:])
	h.Write([]byte(jkt))
	return h.Sum(nil)
}
