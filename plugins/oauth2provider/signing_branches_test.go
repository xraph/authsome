package oauth2provider

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
)

func pkcs8PEM(t *testing.T, key any) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func TestNewKeyManager_NilLoggerIsANoop(t *testing.T) {
	m, err := newKeyManager(Config{}, NewMemoryStore(), testEncryptor(t), nil)
	require.NoError(t, err)
	require.NotNil(t, m.logger)
	_, err = m.Active(context.Background(), "")
	require.NoError(t, err)
}

// Each operator key a deployment may not use stops the boot, and the error
// says why.
func TestNewKeyManager_RefusesUnusableOperatorKeys(t *testing.T) {
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	_, edPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	x, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	keyFile := filepath.Join(t.TempDir(), "op.pem")
	require.NoError(t, os.WriteFile(keyFile, rsaPEM(t, 2048), 0o600))

	for _, c := range []struct {
		name string
		cfg  Config
		want string
	}{
		{"key and file", Config{IDTokenSigningKey: rsaPEM(t, 2048), IDTokenSigningKeyFile: keyFile}, "not both"},
		{"EC P-384", Config{IDTokenSigningKey: pkcs8PEM(t, p384)}, "EC key must use P-256, not P-384"},
		{"Ed25519", Config{IDTokenSigningKey: pkcs8PEM(t, edPriv)}, "unsupported key type ed25519.PublicKey"},
		{"X25519", Config{IDTokenSigningKey: pkcs8PEM(t, x)}, "unsupported key type *ecdh.PrivateKey"},
		{"weak previous", Config{IDTokenPreviousKeys: [][]byte{rsaPEM(t, 1024)}}, "previous key 0: RSA key has 1024 bits"},
		{"non-PEM previous", Config{IDTokenPreviousKeys: [][]byte{[]byte("nope")}}, "previous key 0: no PEM block found"},
		{"garbage PEM previous", Config{IDTokenPreviousKeys: [][]byte{
			pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("junk")}),
		}}, `previous key 0: PEM block "PUBLIC KEY" is not a PKCS#8, PKCS#1 or EC private key`},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := newKeyManager(c.cfg, NewMemoryStore(), testEncryptor(t), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.want)
		})
	}
}

// PKCS#1 keys are accepted for both the signing key and a previous key, and a
// previous key given as a private key publishes only its public half.
func TestNewKeyManager_AcceptsPKCS1AndPrivatePrevious(t *testing.T) {
	ctx := context.Background()
	op, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	opPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(op)})

	prev1, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	prev1PEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&prev1.PublicKey)})
	prev2, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	prev2PEM := pkcs8PEM(t, prev2)

	m, err := newKeyManager(Config{IDTokenSigningKey: opPEM, IDTokenPreviousKeys: [][]byte{prev1PEM, prev2PEM}},
		NewMemoryStore(), testEncryptor(t), nil)
	require.NoError(t, err)

	s, err := m.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "RS256", s.alg)
	assert.Equal(t, thumbprintOf(t, &op.PublicKey), s.kid)

	keys, err := m.Published(ctx, "")
	require.NoError(t, err)
	require.Len(t, keys, 3)
	assert.Equal(t, thumbprintOf(t, &prev1.PublicKey), keys[1].KID)
	assert.Equal(t, thumbprintOf(t, &prev2.PublicKey), keys[2].KID)
}

// When a reload fails after the cache has expired, the key already in hand
// keeps signing rather than failing every login.
func TestKeyManager_ReloadFailureKeepsCachedKey(t *testing.T) {
	ctx := context.Background()
	st := &faultyStore{Store: NewMemoryStore()}
	m := newTestKeyManager(t, Config{}, st, testEncryptor(t))
	tl := log.NewTestLogger()
	m.logger = tl
	clock := time.Now()
	m.now = func() time.Time { return clock }

	first, err := m.Active(ctx, "app")
	require.NoError(t, err)

	clock = clock.Add(activeKeyCacheTTL + time.Second)
	st.listKeys = errStoreDown
	again, err := m.Active(ctx, "app")
	require.NoError(t, err)
	assert.Same(t, first, again)
	assert.Len(t, tl.(*log.TestLogger).GetLogsByLevel("WARN"), 1)
}

// A first-key race lost to another replica whose key can't then be read is
// an error, not a key the manager made up.
func TestKeyManager_FirstKeyRaceWithFailedReread(t *testing.T) {
	st := &faultyStore{Store: NewMemoryStore(), createKey: ErrSigningKeyConflict, listKeysQueue: []error{nil, errStoreDown}}
	m := newTestKeyManager(t, Config{}, st, testEncryptor(t))
	_, err := m.Active(context.Background(), "app")
	require.ErrorIs(t, err, errStoreDown)
	assert.Contains(t, err.Error(), "list signing keys")
}

// A rotation race lost while the active key was retired behind the
// manager's back, with the winner unreadable, keeps the cached key signing.
func TestKeyManager_RaceAfterExternalRetireKeepsCachedKey(t *testing.T) {
	ctx := context.Background()
	st := &faultyStore{Store: NewMemoryStore()}
	m := newTestKeyManager(t, Config{}, st, testEncryptor(t))
	clock := time.Now()
	m.now = func() time.Time { return clock }
	first, err := m.Active(ctx, "app")
	require.NoError(t, err)

	keys, err := st.Store.ListSigningKeys(ctx, "app")
	require.NoError(t, err)
	require.NoError(t, st.Store.RetireSigningKey(ctx, keys[0].ID, clock.Add(time.Hour)))
	clock = clock.Add(activeKeyCacheTTL + time.Second)
	st.createKey = ErrSigningKeyConflict
	st.listKeysQueue = []error{nil, errStoreDown}

	again, err := m.Active(ctx, "app")
	require.NoError(t, err)
	assert.Same(t, first, again)
}

// A race whose winner is nowhere to be found leaves no key to sign with.
func TestKeyManager_FirstKeyRaceWithNoWinner(t *testing.T) {
	st := &faultyStore{Store: NewMemoryStore(), createKey: ErrSigningKeyConflict}
	m := newTestKeyManager(t, Config{}, st, testEncryptor(t))
	_, err := m.Active(context.Background(), "app")
	require.ErrorIs(t, err, errNoSigningKey)
}

func TestKeyManager_RotateFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("listing fails", func(t *testing.T) {
		st := &faultyStore{Store: NewMemoryStore(), listKeys: errStoreDown}
		_, err := newTestKeyManager(t, Config{}, st, testEncryptor(t)).Rotate(ctx, "app", 0)
		require.ErrorIs(t, err, errStoreDown)
	})

	t.Run("race reread fails", func(t *testing.T) {
		st := &faultyStore{Store: NewMemoryStore(), createKey: ErrSigningKeyConflict, listKeysQueue: []error{nil, errStoreDown}}
		_, err := newTestKeyManager(t, Config{}, st, testEncryptor(t)).Rotate(ctx, "app", 0)
		require.ErrorIs(t, err, errStoreDown)
	})

	t.Run("race with no winner", func(t *testing.T) {
		st := &faultyStore{Store: NewMemoryStore(), createKey: ErrSigningKeyConflict}
		_, err := newTestKeyManager(t, Config{}, st, testEncryptor(t)).Rotate(ctx, "app", 0)
		require.ErrorIs(t, err, errNoSigningKey)
	})

	t.Run("insert fails and is counted", func(t *testing.T) {
		st := &faultyStore{Store: NewMemoryStore()}
		m := newTestKeyManager(t, Config{}, st, testEncryptor(t))
		var results []string
		m.onRotate = func(_, result string) { results = append(results, result) }
		first, err := m.Active(ctx, "app")
		require.NoError(t, err)

		st.createKey = errStoreDown
		_, err = m.Rotate(ctx, "app", 0)
		require.ErrorIs(t, err, errStoreDown)
		assert.Equal(t, []string{telemetry.RotationError}, results)

		still, err := m.Active(ctx, "app")
		require.NoError(t, err)
		assert.Equal(t, first.kid, still.kid, "the current key keeps signing")
	})
}

// A rotation goes through when retiring the old key fails: the new key
// signs, and the old one just stays published until a later rotation.
func TestKeyManager_RetireFailureDoesNotUndoRotation(t *testing.T) {
	ctx := context.Background()
	st := &faultyStore{Store: NewMemoryStore()}
	m := newTestKeyManager(t, Config{}, st, testEncryptor(t))
	tl := log.NewTestLogger()
	m.logger = tl
	first, err := m.Active(ctx, "app")
	require.NoError(t, err)

	st.retireKey = errStoreDown
	row, err := m.Rotate(ctx, "app", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), row.Generation)

	keys, err := st.Store.ListSigningKeys(ctx, "app")
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Nil(t, keys[1].RetiresAt, "the old key could not be retired")
	assert.NotEmpty(t, tl.(*log.TestLogger).GetLogsByLevel("WARN"))

	// The unretired old key is listed as previous, not active.
	info, err := m.Describe(ctx, "app")
	require.NoError(t, err)
	require.Len(t, info, 2)
	assert.Equal(t, keyStatusActive, info[0].Status)
	assert.Equal(t, keyStatusPrevious, info[1].Status)
	assert.Equal(t, first.kid, info[1].KID)
}

// A key already retiring keeps its date through a later rotation.
func TestKeyManager_RotationKeepsRetiringDate(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	m := newTestKeyManager(t, Config{}, st, testEncryptor(t))
	_, err := m.Active(ctx, "app")
	require.NoError(t, err)
	_, err = m.Rotate(ctx, "app", 0)
	require.NoError(t, err)
	keys, err := st.ListSigningKeys(ctx, "app")
	require.NoError(t, err)
	require.NotNil(t, keys[1].RetiresAt)
	retiring := *keys[1].RetiresAt

	_, err = m.Rotate(ctx, "app", 0)
	require.NoError(t, err)
	keys, err = st.ListSigningKeys(ctx, "app")
	require.NoError(t, err)
	require.Len(t, keys, 3)
	require.NotNil(t, keys[2].RetiresAt)
	assert.True(t, retiring.Equal(*keys[2].RetiresAt), "the oldest key's retirement was not pushed out")
}

// failingEncryptor refuses to seal anything.
type failingEncryptor struct{ bridge.Encryptor }

func (failingEncryptor) Encrypt([]byte) ([]byte, error) { return nil, errors.New("kms unavailable") }

func TestKeyManager_SealFailureStoresNothing(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	m := newTestKeyManager(t, Config{}, st, failingEncryptor{testEncryptor(t)})
	_, err := m.Active(ctx, "app")
	require.ErrorIs(t, err, errNoSigningKey)
	assert.Contains(t, err.Error(), "seal signing key")
	keys, err := st.ListSigningKeys(ctx, "app")
	require.NoError(t, err)
	assert.Empty(t, keys)
}

// A stored key that can't be opened is an error for signing, and is skipped,
// not served, by the JWKS and the dashboard listing.
func TestKeyManager_UnreadableStoredKeys(t *testing.T) {
	ctx := context.Background()
	enc := testEncryptor(t)
	good := newTestKeyManager(t, Config{}, NewMemoryStore(), enc)
	template, err := good.mint("app", 1, time.Now())
	require.NoError(t, err)
	seal := func(plain string) string {
		b, sealErr := enc.Encrypt([]byte(plain))
		require.NoError(t, sealErr)
		return string(b)
	}
	x, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	xDER, err := x509.MarshalPKCS8PrivateKey(x)
	require.NoError(t, err)

	for _, c := range []struct {
		name   string
		mutate func(*SigningKey)
		want   string
		listed bool
	}{
		{"bad public JWK", func(k *SigningKey) { k.PublicJWK = []byte(`{"kty":"oct","k":"c2VjcmV0"}`) }, "signing key", false},
		{"not base64", func(k *SigningKey) { k.PrivateKey = seal("!!! not base64 !!!") }, "decode signing key", true},
		{"not PKCS#8", func(k *SigningKey) { k.PrivateKey = seal(base64.StdEncoding.EncodeToString([]byte("junk"))) }, "parse signing key", true},
		{"not a signer", func(k *SigningKey) { k.PrivateKey = seal(base64.StdEncoding.EncodeToString(xDER)) }, "is a *ecdh.PrivateKey", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			st := NewMemoryStore()
			row := *template
			row.ID = id.NewOAuth2SigningKeyID()
			c.mutate(&row)
			require.NoError(t, st.CreateSigningKey(ctx, &row))
			m := newTestKeyManager(t, Config{}, st, enc)

			_, err := m.Active(ctx, "app")
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.want)

			info, err := m.Describe(ctx, "app")
			require.NoError(t, err)
			pub, err := m.Published(ctx, "app")
			require.NoError(t, err)
			if c.listed {
				// The public half is fine; only signing is impossible.
				assert.Len(t, info, 1)
				assert.Len(t, pub, 1)
			} else {
				assert.Empty(t, info)
				assert.Empty(t, pub)
			}
		})
	}
}

// With an operator key the JWKS still serves it when the managed keys can't
// be listed.
func TestKeyManager_OperatorJWKSSurvivesListFailure(t *testing.T) {
	st := &faultyStore{Store: NewMemoryStore()}
	m := newTestKeyManager(t, Config{IDTokenSigningKey: rsaPEM(t, 2048)}, st, testEncryptor(t))
	tl := log.NewTestLogger()
	m.logger = tl
	s, err := m.Active(context.Background(), "")
	require.NoError(t, err)

	st.listKeys = errStoreDown
	assert.Equal(t, []string{s.kid}, publishedKIDs(t, m, ""))
	assert.NotEmpty(t, tl.(*log.TestLogger).GetLogsByLevel("WARN"))
}

func TestKeyManager_DescribeListFailure(t *testing.T) {
	st := &faultyStore{Store: NewMemoryStore(), listKeys: errStoreDown}
	_, err := newTestKeyManager(t, Config{}, st, testEncryptor(t)).Describe(context.Background(), "app")
	require.ErrorIs(t, err, errStoreDown)
}

// Under an operator key the dashboard lists the operator key and the managed
// keys still retiring, never a managed key that would no longer verify.
func TestKeyManager_DescribeUnderOperatorSkipsUnretiredManagedKey(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	_, err := newTestKeyManager(t, Config{}, st, testEncryptor(t)).Active(ctx, "")
	require.NoError(t, err)

	m := newTestKeyManager(t, Config{IDTokenSigningKey: rsaPEM(t, 2048)}, st, testEncryptor(t))
	info, err := m.Describe(ctx, "")
	require.NoError(t, err)
	require.Len(t, info, 1)
	assert.Equal(t, keySourceOperator, info[0].Source)
}

func TestKeyManager_RetireForOperatorListFailure(t *testing.T) {
	st := &faultyStore{Store: NewMemoryStore(), listKeys: errStoreDown}
	m := newTestKeyManager(t, Config{IDTokenSigningKey: rsaPEM(t, 2048)}, st, testEncryptor(t))
	err := m.retireForOperator(context.Background())
	require.ErrorIs(t, err, errStoreDown)
}

// The boot does not stop when the managed keys can't be retired under a new
// operator key: it warns, and the operator key signs.
func TestInitSigningKeys_RetireFailureOnlyWarns(t *testing.T) {
	ctx := context.Background()
	st := &faultyStore{Store: NewMemoryStore(), listKeys: errStoreDown}
	p := New(Config{IDTokenSigningKey: rsaPEM(t, 2048)})
	p.SetOAuth2Store(st)
	tl := log.NewTestLogger()
	p.logger = tl
	require.NoError(t, p.initSigningKeys(ctx, testEncryptor(t)))
	require.NotNil(t, p.keys)
	warns := tl.(*log.TestLogger).GetLogsByLevel("WARN")
	require.Len(t, warns, 1)
	assert.Contains(t, warns[0].Message, "retire managed id_token keys")

	s, err := p.keys.Active(ctx, "")
	require.NoError(t, err)
	assert.Same(t, p.keys.operator, s)
}

// The plugin reports each key rotation to the engine's metrics under the
// issuer it rotated for.
func TestInitSigningKeys_ReportsRotations(t *testing.T) {
	ctx := context.Background()
	reg, scrape := telemetrytest.Registry(t)
	eng := secutil.NewTestEngine(t, authsome.WithMetricsSink(reg), authsome.WithTokenEncryptor(testEncryptor(t)))
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	require.NoError(t, p.OnInit(ctx, eng))

	_, err := p.keys.Active(ctx, "app")
	require.NoError(t, err)
	assert.Empty(t, telemetrytest.Series(scrape(), "forge_authsome_oauth2_signing_key_"), "a first key is not a rotation")

	row, err := p.keys.Rotate(ctx, "app", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), row.Generation)
	series := telemetrytest.Series(scrape(), "forge_authsome_oauth2_signing_key_")
	require.Len(t, series, 1)
	assert.Contains(t, series[0], `issuer="app"`)
	assert.Contains(t, series[0], `result="ok"`)
}
