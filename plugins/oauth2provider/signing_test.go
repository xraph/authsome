package oauth2provider

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/internal/secutil"
)

// testEncryptor is the AES-GCM encryptor a real deployment would hand the
// plugin, built from a fixed 32 byte key.
func testEncryptor(t *testing.T) bridge.Encryptor {
	t.Helper()
	enc, err := bridge.NewAESGCMEncryptor(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	return enc
}

func newTestKeyManager(t *testing.T, cfg Config, st Store, enc bridge.Encryptor) *keyManager {
	t.Helper()
	m, err := newKeyManager(New(cfg).config, st, enc, log.NewNoopLogger())
	require.NoError(t, err)
	return m
}

func rsaPEM(t *testing.T, bits int) []byte {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(k)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func thumbprintOf(t *testing.T, pub crypto.PublicKey) string {
	t.Helper()
	j, err := jwkutil.Encode(pub, "", "")
	require.NoError(t, err)
	tp, err := jwkutil.Thumbprint(j)
	require.NoError(t, err)
	return tp
}

func publishedKIDs(t *testing.T, m *keyManager, scope string) []string {
	t.Helper()
	keys, err := m.Published(context.Background(), scope)
	require.NoError(t, err)
	kids := make([]string, 0, len(keys))
	for i := range keys {
		kids = append(kids, keys[i].KID)
	}
	return kids
}

func TestKeyManager_OperatorKeyWins(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	m := newTestKeyManager(t, Config{IDTokenSigningKey: rsaPEM(t, 2048)}, st, testEncryptor(t))

	s, err := m.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "RS256", s.alg)
	assert.Equal(t, thumbprintOf(t, s.key.Public()), s.kid)

	keys, err := st.ListSigningKeys(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, keys, "an operator key must not generate a managed one")

	assert.Equal(t, []string{s.kid}, publishedKIDs(t, m, ""))

	_, err = m.Rotate(ctx, "")
	require.ErrorIs(t, err, ErrOperatorKey)

	info, err := m.Describe(ctx, "")
	require.NoError(t, err)
	require.Len(t, info, 1)
	assert.Equal(t, "operator", info[0].Source)
	assert.Equal(t, "active", info[0].Status)
}

func TestKeyManager_ManagedKeyCreatedOnce(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	m := newTestKeyManager(t, Config{}, st, testEncryptor(t))

	first, err := m.Active(ctx, "")
	require.NoError(t, err)
	second, err := m.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, first.kid, second.kid)
	assert.Equal(t, "RS256", first.alg)

	keys, err := st.ListSigningKeys(ctx, "")
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, int64(1), keys[0].Generation)
	assert.True(t, strings.HasPrefix(keys[0].PrivateKey, "v1:"), "the private key is stored sealed")

	// A second replica with a cold cache reads the same key back.
	other := newTestKeyManager(t, Config{}, st, testEncryptor(t))
	third, err := other.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, first.kid, third.kid)
}

func TestKeyManager_RotationRaceHasOneWinner(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	enc := testEncryptor(t)

	// Mint the first key 100 days ago, past the 90 day rotation period.
	old := newTestKeyManager(t, Config{}, st, enc)
	old.now = func() time.Time { return time.Now().Add(-100 * 24 * time.Hour) }
	aged, err := old.Active(ctx, "app")
	require.NoError(t, err)

	// Two replicas sharing one store, four goroutines each.
	replicas := []*keyManager{
		newTestKeyManager(t, Config{}, st, enc),
		newTestKeyManager(t, Config{}, st, enc),
	}
	kids := make([]string, 8)
	var wg sync.WaitGroup
	for i := range kids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, activeErr := replicas[i%2].Active(ctx, "app")
			if activeErr != nil {
				t.Errorf("goroutine %d: %v", i, activeErr)
				return
			}
			kids[i] = s.kid
		}(i)
	}
	wg.Wait()

	keys, err := st.ListSigningKeys(ctx, "app")
	require.NoError(t, err)
	require.Len(t, keys, 2, "exactly one new key")
	winner := keys[0]
	assert.Equal(t, int64(2), winner.Generation)
	assert.Nil(t, winner.RetiresAt)
	require.NotNil(t, keys[1].RetiresAt, "the aged key is retiring")
	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), *keys[1].RetiresAt, time.Minute)

	j, err := jwkFromStored(winner)
	require.NoError(t, err)
	for i, kid := range kids {
		assert.Equal(t, j.KID, kid, "goroutine %d", i)
		assert.NotEqual(t, aged.kid, kid, "goroutine %d", i)
	}
}

func TestKeyManager_RetiredKeyLeavesJWKS(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	m := newTestKeyManager(t, Config{}, st, testEncryptor(t))

	first, err := m.Active(ctx, "app")
	require.NoError(t, err)
	_, err = m.Rotate(ctx, "app")
	require.NoError(t, err)
	current, err := m.Active(ctx, "app")
	require.NoError(t, err)
	require.NotEqual(t, first.kid, current.kid)

	// Retiring in the future: still published.
	assert.ElementsMatch(t, []string{current.kid, first.kid}, publishedKIDs(t, m, "app"))

	info, err := m.Describe(ctx, "app")
	require.NoError(t, err)
	require.Len(t, info, 2)
	assert.Equal(t, current.kid, info[0].KID)
	assert.Equal(t, "active", info[0].Status)
	assert.Equal(t, "retiring", info[1].Status)
	assert.Equal(t, "managed", info[1].Source)

	// Retired in the past: gone.
	keys, err := st.ListSigningKeys(ctx, "app")
	require.NoError(t, err)
	require.Len(t, keys, 2)
	require.NoError(t, st.RetireSigningKey(ctx, keys[1].ID, time.Now().Add(-time.Minute)))
	assert.Equal(t, []string{current.kid}, publishedKIDs(t, m, "app"))
}

// failingCreateStore refuses every new signing key while fail is set.
type failingCreateStore struct {
	Store
	mu   sync.Mutex
	fail bool
}

func (s *failingCreateStore) CreateSigningKey(ctx context.Context, k *SigningKey) error {
	s.mu.Lock()
	fail := s.fail
	s.mu.Unlock()
	if fail {
		return errors.New("database is down")
	}
	return s.Store.CreateSigningKey(ctx, k)
}

func TestKeyManager_RotationFailureKeepsSigning(t *testing.T) {
	ctx := context.Background()
	st := &failingCreateStore{Store: NewMemoryStore()}
	enc := testEncryptor(t)

	old := newTestKeyManager(t, Config{}, st, enc)
	old.now = func() time.Time { return time.Now().Add(-100 * 24 * time.Hour) }
	aged, err := old.Active(ctx, "")
	require.NoError(t, err)

	st.mu.Lock()
	st.fail = true
	st.mu.Unlock()

	m := newTestKeyManager(t, Config{}, st, enc)
	s, err := m.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, aged.kid, s.kid)

	keys, err := st.ListSigningKeys(ctx, "")
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Nil(t, keys[0].RetiresAt, "a failed rotation retires nothing")
}

func TestKeyManager_NoEncryptorNoKey(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	m := newTestKeyManager(t, Config{}, st, bridge.NoopEncryptor{})

	_, err := m.Active(ctx, "")
	require.ErrorIs(t, err, errNoSigningKey)
	_, err = m.Active(ctx, "app")
	require.ErrorIs(t, err, errNoSigningKey)

	keys, err := st.ListSigningKeys(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, keys, "nothing is stored unencrypted")
}

func TestOnInit_RejectsBadOperatorKey(t *testing.T) {
	eng := secutil.NewTestEngine(t)

	keyFile := filepath.Join(t.TempDir(), "garbage.pem")
	require.NoError(t, os.WriteFile(keyFile, []byte("not a key"), 0o600))

	cases := map[string]Config{
		"garbage PEM":  {IDTokenSigningKey: []byte("-----BEGIN PRIVATE KEY-----\nbm9wZQ==\n-----END PRIVATE KEY-----\n")},
		"not PEM":      {IDTokenSigningKey: []byte("hello")},
		"RSA 1024":     {IDTokenSigningKey: rsaPEM(t, 1024)},
		"garbage file": {IDTokenSigningKeyFile: keyFile},
		"missing file": {IDTokenSigningKeyFile: filepath.Join(t.TempDir(), "absent.pem")},
		"bad previous": {IDTokenPreviousKeys: [][]byte{[]byte("hello")}},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			p := New(cfg)
			p.SetOAuth2Store(NewMemoryStore())
			require.Error(t, p.OnInit(context.Background(), eng))
		})
	}

	t.Run("RSA 2048 boots", func(t *testing.T) {
		p := New(Config{IDTokenSigningKey: rsaPEM(t, 2048)})
		p.SetOAuth2Store(NewMemoryStore())
		require.NoError(t, p.OnInit(context.Background(), eng))
		require.NotNil(t, p.keys)
	})
}

func TestKeyManager_ECOperatorKeyIsES256(t *testing.T) {
	ctx := context.Background()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(k)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})

	// A previous key given as a public PEM is published but never signs.
	prev, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(&prev.PublicKey)
	require.NoError(t, err)
	prevPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	m := newTestKeyManager(t, Config{
		IDTokenSigningKey:   keyPEM,
		IDTokenSigningKeyID: "op-1",
		IDTokenPreviousKeys: [][]byte{prevPEM},
	}, NewMemoryStore(), testEncryptor(t))

	s, err := m.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "ES256", s.alg)
	assert.Equal(t, "op-1", s.kid)
	_, isEC := s.key.(*ecdsa.PrivateKey)
	assert.True(t, isEC)

	keys, err := m.Published(ctx, "")
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Equal(t, "op-1", keys[0].KID)
	assert.Equal(t, "ES256", keys[0].ALG)
	assert.Equal(t, thumbprintOf(t, &prev.PublicKey), keys[1].KID)
	assert.Equal(t, "RS256", keys[1].ALG)

	info, err := m.Describe(ctx, "")
	require.NoError(t, err)
	require.Len(t, info, 2)
	assert.Equal(t, "previous", info[1].Status)
	assert.Equal(t, "operator", info[1].Source)
}
