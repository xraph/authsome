package oauth2provider

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
)

// errNoSigningKey means no key can sign an id_token for the scope: there is
// no operator key and a managed one cannot be loaded or generated.
var errNoSigningKey = errors.New("oauth2: no id_token signing key available")

// ErrOperatorKey is returned when a rotation is asked of an issuer whose key
// comes from Config. Rotating it means changing the configuration.
var ErrOperatorKey = errors.New("oauth2: the signing key for this issuer is managed by configuration")

const (
	// activeKeyCacheTTL is how long a scope's active key is trusted before
	// the store is read again.
	activeKeyCacheTTL = 5 * time.Minute

	// managedKeyBits is the size of every key authsome generates.
	managedKeyBits = 2048

	keySourceOperator = "operator"
	keySourceManaged  = "managed"

	keyStatusActive   = "active"
	keyStatusPrevious = "previous"
	keyStatusRetiring = "retiring"
)

// signer is the key that signs id_tokens for one scope.
type signer struct {
	kid, alg string
	key      crypto.Signer
}

// KeyInfo describes one signing key for the admin dashboard. It never carries
// key material.
type KeyInfo struct {
	KID string
	Alg string
	// Source is "operator" or "managed".
	Source string
	// Status is "active", "previous" or "retiring".
	Status    string
	CreatedAt time.Time
	RetiresAt *time.Time
}

// keyManager resolves, generates and rotates id_token signing keys. The
// deployment issuer (scope "") uses the operator key when one is configured;
// every other scope, and the deployment issuer without one, uses managed keys
// kept in the store.
type keyManager struct {
	store    Store
	enc      bridge.Encryptor
	logger   log.Logger
	rotation time.Duration
	grace    time.Duration
	now      func() time.Time

	// operator signs for scope "" when set; operatorPrev are published
	// alongside it and never sign.
	operator     *signer
	operatorJWK  jwkutil.JWK
	operatorPrev []jwkutil.JWK

	mu     sync.Mutex
	scopes map[string]*scopeKeys

	noEncryptorOnce sync.Once
}

// scopeKeys caches one scope's active key. Its mutex is held across the
// store round trip and any key generation, so concurrent callers in this
// process wait for one generation instead of each running their own. The
// (scope, generation) unique index settles races between replicas.
type scopeKeys struct {
	mu       sync.Mutex
	active   *signer
	row      *SigningKey
	loadedAt time.Time
	// retryAfter holds off another rotation attempt after one failed, so a
	// database outage costs one key generation per cache period, not one
	// per token.
	retryAfter time.Time
}

// newKeyManager parses and validates the operator keys in cfg. A key that
// fails to parse, or is weaker than allowed, is an error: a bad key should
// stop the boot.
func newKeyManager(cfg Config, st Store, enc bridge.Encryptor, logger log.Logger) (*keyManager, error) {
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	m := &keyManager{
		store:    st,
		enc:      enc,
		logger:   logger,
		rotation: cfg.IDTokenKeyRotation,
		grace:    cfg.IDTokenKeyRetireGrace,
		now:      time.Now,
		scopes:   make(map[string]*scopeKeys),
	}

	keyPEM := cfg.IDTokenSigningKey
	if cfg.IDTokenSigningKeyFile != "" {
		if len(keyPEM) > 0 {
			return nil, errors.New("oauth2: set IDTokenSigningKey or IDTokenSigningKeyFile, not both")
		}
		raw, err := os.ReadFile(cfg.IDTokenSigningKeyFile)
		if err != nil {
			return nil, fmt.Errorf("oauth2: read id_token signing key: %w", err)
		}
		keyPEM = raw
	}
	if len(keyPEM) > 0 {
		priv, err := parsePrivateKeyPEM(keyPEM)
		if err != nil {
			return nil, fmt.Errorf("oauth2: id_token signing key: %w", err)
		}
		alg, err := operatorAlg(priv.Public())
		if err != nil {
			return nil, fmt.Errorf("oauth2: id_token signing key: %w", err)
		}
		j, err := publicJWK(priv.Public(), cfg.IDTokenSigningKeyID, alg)
		if err != nil {
			return nil, fmt.Errorf("oauth2: id_token signing key: %w", err)
		}
		m.operator = &signer{kid: j.KID, alg: alg, key: priv}
		m.operatorJWK = *j
	}

	for i, raw := range cfg.IDTokenPreviousKeys {
		pub, err := parsePublicKeyPEM(raw)
		if err != nil {
			return nil, fmt.Errorf("oauth2: id_token previous key %d: %w", i, err)
		}
		alg, err := operatorAlg(pub)
		if err != nil {
			return nil, fmt.Errorf("oauth2: id_token previous key %d: %w", i, err)
		}
		j, err := publicJWK(pub, "", alg)
		if err != nil {
			return nil, fmt.Errorf("oauth2: id_token previous key %d: %w", i, err)
		}
		m.operatorPrev = append(m.operatorPrev, *j)
	}

	return m, nil
}

// usesOperatorKey reports whether scope is served by the operator key.
// Operator keys only ever apply to the deployment issuer.
func (m *keyManager) usesOperatorKey(scope string) bool {
	return scope == "" && m.operator != nil
}

// canStoreKeys reports whether a managed key can be sealed at rest. A no-op
// encryptor would put the private key in the database in the clear, so the
// manager refuses rather than do that.
func (m *keyManager) canStoreKeys() bool {
	switch m.enc.(type) {
	case nil, bridge.NoopEncryptor, *bridge.NoopEncryptor:
		m.noEncryptorOnce.Do(func() {
			m.logger.Error("oauth2: no token encryptor is configured, so no id_token signing key " +
				"can be stored; set a token encryption key or an operator IDTokenSigningKey")
		})
		return false
	}
	return true
}

func (m *keyManager) scopeState(scope string) *scopeKeys {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.scopes[scope]
	if !ok {
		st = &scopeKeys{}
		m.scopes[scope] = st
	}
	return st
}

// Active returns the key that signs id_tokens for scope, generating the
// first managed key or rotating an aged one as needed. A failed rotation is
// logged and the current key keeps signing.
func (m *keyManager) Active(ctx context.Context, scope string) (*signer, error) {
	if m.usesOperatorKey(scope) {
		return m.operator, nil
	}
	if !m.canStoreKeys() {
		return nil, errNoSigningKey
	}

	st := m.scopeState(scope)
	st.mu.Lock()
	defer st.mu.Unlock()

	now := m.now()
	if st.active != nil && now.Sub(st.loadedAt) < activeKeyCacheTTL &&
		(!m.due(st.row, now) || now.Before(st.retryAfter)) {
		return st.active, nil
	}

	keys, err := m.store.ListSigningKeys(ctx, scope)
	if err != nil {
		if st.active != nil {
			m.logger.Warn("oauth2: reload id_token signing keys; keeping the cached key",
				log.String("scope", scope), log.Error(err))
			return st.active, nil
		}
		return nil, fmt.Errorf("oauth2: list signing keys: %w", err)
	}

	row := activeRow(keys)
	if row == nil || m.due(row, now) {
		created, createErr := m.create(ctx, scope, keys, now)
		switch {
		case createErr == nil:
			row = created
		case errors.Is(createErr, ErrSigningKeyConflict):
			// Another replica inserted this generation first. Its key wins.
			keys, err = m.store.ListSigningKeys(ctx, scope)
			if err != nil {
				return nil, fmt.Errorf("oauth2: list signing keys: %w", err)
			}
			row = activeRow(keys)
		case row != nil:
			m.logger.Error("oauth2: rotate id_token signing key; signing with the current key",
				log.String("scope", scope), log.Error(createErr))
			st.retryAfter = now.Add(activeKeyCacheTTL)
		default:
			return nil, fmt.Errorf("%w: %w", errNoSigningKey, createErr)
		}
	}
	if row == nil {
		return nil, errNoSigningKey
	}

	s, err := m.open(row)
	if err != nil {
		return nil, err
	}
	st.active, st.row, st.loadedAt = s, row, now
	return s, nil
}

// Rotate generates the next managed key for scope straight away, whatever
// the age of the current one. It is what the dashboard calls.
func (m *keyManager) Rotate(ctx context.Context, scope string) (*SigningKey, error) {
	if m.usesOperatorKey(scope) {
		return nil, ErrOperatorKey
	}
	if !m.canStoreKeys() {
		return nil, errNoSigningKey
	}

	st := m.scopeState(scope)
	st.mu.Lock()
	defer st.mu.Unlock()
	// Whatever happens next, the cached key is no longer the newest.
	st.active, st.row = nil, nil

	keys, err := m.store.ListSigningKeys(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("oauth2: list signing keys: %w", err)
	}
	row, err := m.create(ctx, scope, keys, m.now())
	if errors.Is(err, ErrSigningKeyConflict) {
		// A scheduled rotation landed at the same moment. Its key is just
		// as fresh, so hand that one back.
		keys, err = m.store.ListSigningKeys(ctx, scope)
		if err != nil {
			return nil, fmt.Errorf("oauth2: list signing keys: %w", err)
		}
		if row = activeRow(keys); row == nil {
			return nil, errNoSigningKey
		}
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// Published returns the public keys a relying party needs to verify
// id_tokens for scope: the active key, every managed key still inside its
// retirement grace, and the operator's previous keys. Retired keys and
// symmetric keys are never included.
func (m *keyManager) Published(ctx context.Context, scope string) ([]jwkutil.JWK, error) {
	if m.usesOperatorKey(scope) {
		out := make([]jwkutil.JWK, 0, 1+len(m.operatorPrev))
		out = append(out, m.operatorJWK)
		return append(out, m.operatorPrev...), nil
	}

	// Make sure the scope has a key, and that an aged one has rotated,
	// before saying which keys exist.
	if _, err := m.Active(ctx, scope); err != nil && !errors.Is(err, errNoSigningKey) {
		return nil, err
	}

	keys, err := m.store.ListSigningKeys(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("oauth2: list signing keys: %w", err)
	}
	now := m.now()
	out := make([]jwkutil.JWK, 0, len(keys))
	for _, k := range keys {
		if k.RetiresAt != nil && !k.RetiresAt.After(now) {
			continue
		}
		j, err := jwkFromStored(k)
		if err != nil {
			m.logger.Warn("oauth2: skip unreadable signing key in JWKS",
				log.String("key_id", k.ID.String()), log.Error(err))
			continue
		}
		out = append(out, *j)
	}
	return out, nil
}

// Describe lists the keys for scope as the dashboard shows them, active key
// first. It reads only and never generates a key.
func (m *keyManager) Describe(ctx context.Context, scope string) ([]KeyInfo, error) {
	if m.usesOperatorKey(scope) {
		out := make([]KeyInfo, 0, 1+len(m.operatorPrev))
		out = append(out, KeyInfo{
			KID: m.operator.kid, Alg: m.operator.alg,
			Source: keySourceOperator, Status: keyStatusActive,
		})
		for _, j := range m.operatorPrev {
			out = append(out, KeyInfo{
				KID: j.KID, Alg: j.ALG,
				Source: keySourceOperator, Status: keyStatusPrevious,
			})
		}
		return out, nil
	}

	keys, err := m.store.ListSigningKeys(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("oauth2: list signing keys: %w", err)
	}
	active := activeRow(keys)
	now := m.now()
	out := make([]KeyInfo, 0, len(keys))
	for _, k := range keys {
		var status string
		switch {
		case k == active:
			status = keyStatusActive
		case k.RetiresAt == nil:
			status = keyStatusPrevious
		case k.RetiresAt.After(now):
			status = keyStatusRetiring
		default:
			continue
		}
		j, err := jwkFromStored(k)
		if err != nil {
			return nil, err
		}
		out = append(out, KeyInfo{
			KID: j.KID, Alg: k.Alg, Source: keySourceManaged, Status: status,
			CreatedAt: k.CreatedAt, RetiresAt: k.RetiresAt,
		})
	}
	return out, nil
}

// due reports whether k is old enough to rotate.
func (m *keyManager) due(k *SigningKey, now time.Time) bool {
	return k != nil && m.rotation > 0 && now.Sub(k.CreatedAt) >= m.rotation
}

// create generates and inserts the next managed key for scope, then starts
// the retirement grace of every key that was unretired. keys is the scope's
// current listing, newest generation first.
func (m *keyManager) create(ctx context.Context, scope string, keys []*SigningKey, now time.Time) (*SigningKey, error) {
	// The next generation is the active key's plus one. Keys are listed
	// newest generation first, so keys[0] is the active key unless a newer
	// one was retired, in which case going past it is the only way through
	// the unique index.
	gen := int64(1)
	if len(keys) > 0 {
		gen = keys[0].Generation + 1
	}

	priv, err := rsa.GenerateKey(rand.Reader, managedKeyBits)
	if err != nil {
		return nil, fmt.Errorf("oauth2: generate signing key: %w", err)
	}
	j, err := publicJWK(&priv.PublicKey, "", "RS256")
	if err != nil {
		return nil, err
	}
	pubJSON, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("oauth2: encode signing key: %w", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("oauth2: encode signing key: %w", err)
	}
	sealed, err := m.enc.Encrypt([]byte(base64.StdEncoding.EncodeToString(der)))
	if err != nil {
		return nil, fmt.Errorf("oauth2: seal signing key: %w", err)
	}

	row := &SigningKey{
		ID:         id.NewOAuth2SigningKeyID(),
		Scope:      scope,
		Alg:        "RS256",
		PrivateKey: string(sealed),
		PublicJWK:  pubJSON,
		Generation: gen,
		CreatedAt:  now.UTC(),
	}
	if err := m.store.CreateSigningKey(ctx, row); err != nil {
		return nil, err
	}

	retireAt := now.Add(m.grace).UTC()
	for _, k := range keys {
		if k.RetiresAt != nil {
			continue
		}
		if err := m.store.RetireSigningKey(ctx, k.ID, retireAt); err != nil {
			// The new key already signs. The old one just stays published
			// until a later rotation retires it.
			m.logger.Warn("oauth2: retire previous id_token signing key",
				log.String("key_id", k.ID.String()), log.Error(err))
		}
	}
	return row, nil
}

// open decrypts a stored key into a signer.
func (m *keyManager) open(k *SigningKey) (*signer, error) {
	j, err := jwkFromStored(k)
	if err != nil {
		return nil, err
	}
	plain, err := m.enc.Decrypt([]byte(k.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("oauth2: unseal signing key %s: %w", k.ID, err)
	}
	der, err := base64.StdEncoding.DecodeString(string(plain))
	if err != nil {
		return nil, fmt.Errorf("oauth2: decode signing key %s: %w", k.ID, err)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("oauth2: parse signing key %s: %w", k.ID, err)
	}
	priv, ok := parsed.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("oauth2: signing key %s is a %T", k.ID, parsed)
	}
	return &signer{kid: j.KID, alg: k.Alg, key: priv}, nil
}

// activeRow returns the newest key that has not been retired, or nil.
func activeRow(keys []*SigningKey) *SigningKey {
	for _, k := range keys {
		if k.RetiresAt == nil {
			return k
		}
	}
	return nil
}

// jwkFromStored parses a stored public JWK. ParseJSON refuses symmetric and
// private key material, so nothing it returns is unsafe to publish.
func jwkFromStored(k *SigningKey) (*jwkutil.JWK, error) {
	_, j, err := jwkutil.ParseJSON(k.PublicJWK)
	if err != nil {
		return nil, fmt.Errorf("oauth2: signing key %s: %w", k.ID, err)
	}
	return j, nil
}

// publicJWK encodes pub with kid, defaulting the kid to the RFC 7638
// thumbprint.
func publicJWK(pub crypto.PublicKey, kid, alg string) (*jwkutil.JWK, error) {
	j, err := jwkutil.Encode(pub, kid, alg)
	if err != nil {
		return nil, err
	}
	if j.KID == "" {
		tp, err := jwkutil.Thumbprint(j)
		if err != nil {
			return nil, err
		}
		j.KID = tp
	}
	return j, nil
}

// operatorAlg checks that pub is a key an operator may configure and returns
// the alg it signs with: RSA of at least 2048 bits is RS256, EC P-256 is
// ES256.
func operatorAlg(pub crypto.PublicKey) (string, error) {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		if k.N.BitLen() < managedKeyBits {
			return "", fmt.Errorf("RSA key has %d bits, at least %d are required", k.N.BitLen(), managedKeyBits)
		}
		return "RS256", nil
	case *ecdsa.PublicKey:
		if k.Curve != elliptic.P256() {
			return "", fmt.Errorf("EC key must use P-256, not %s", k.Curve.Params().Name)
		}
		return "ES256", nil
	default:
		return "", fmt.Errorf("unsupported key type %T", pub)
	}
}

// parsePrivateKeyPEM reads a PKCS#8, PKCS#1 or SEC 1 private key.
func parsePrivateKeyPEM(raw []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		s, ok := k.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("unsupported key type %T", k)
		}
		return s, nil
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	if k, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	return nil, fmt.Errorf("PEM block %q is not a PKCS#8, PKCS#1 or EC private key", block.Type)
}

// parsePublicKeyPEM reads the public half of a PEM key that may be public
// (PKIX or PKCS#1) or private.
func parsePublicKeyPEM(raw []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	if k, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		return k, nil
	}
	if k, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return k, nil
	}
	priv, err := parsePrivateKeyPEM(raw)
	if err != nil {
		return nil, err
	}
	return priv.Public(), nil
}
