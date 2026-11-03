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
	"github.com/xraph/authsome/id"
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
		"garbage PEM":       {IDTokenSigningKey: []byte("-----BEGIN PRIVATE KEY-----\nbm9wZQ==\n-----END PRIVATE KEY-----\n")},
		"not PEM":           {IDTokenSigningKey: []byte("hello")},
		"RSA 1024":          {IDTokenSigningKey: rsaPEM(t, 1024)},
		"garbage file":      {IDTokenSigningKeyFile: keyFile},
		"missing file":      {IDTokenSigningKeyFile: filepath.Join(t.TempDir(), "absent.pem")},
		"bad previous":      {IDTokenPreviousKeys: [][]byte{[]byte("hello")}},
		"negative TTL":      {IDTokenTTL: -time.Second},
		"negative rotation": {IDTokenKeyRotation: -time.Second},
		"negative grace":    {IDTokenKeyRetireGrace: -time.Second},
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

func publicPEM(t *testing.T) ([]byte, crypto.PublicKey) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), &k.PublicKey
}

// An operator moving from an operator key to managed keys lists the old key
// as a previous key, and it must stay published next to the managed one.
func TestKeyManager_PreviousKeysPublishedOnManagedPath(t *testing.T) {
	ctx := context.Background()
	prevPEM, prevPub := publicPEM(t)
	m := newTestKeyManager(t, Config{IDTokenPreviousKeys: [][]byte{prevPEM}}, NewMemoryStore(), testEncryptor(t))

	s, err := m.Active(ctx, "")
	require.NoError(t, err)
	prevKID := thumbprintOf(t, prevPub)
	assert.Equal(t, []string{s.kid, prevKID}, publishedKIDs(t, m, ""))

	info, err := m.Describe(ctx, "")
	require.NoError(t, err)
	require.Len(t, info, 2)
	assert.Equal(t, KeyInfo{KID: prevKID, Alg: "RS256", Source: "operator", Status: "previous"}, info[1])

	// Previous keys belong to the deployment issuer only.
	app, err := m.Active(ctx, "app")
	require.NoError(t, err)
	assert.Equal(t, []string{app.kid}, publishedKIDs(t, m, "app"))
}

// Switching from managed keys to an operator key retires the managed key on
// boot, and it stays published through its grace.
func TestKeyManager_OperatorSwitchRetiresManagedKey(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	managed, err := newTestKeyManager(t, Config{}, st, testEncryptor(t)).Active(ctx, "")
	require.NoError(t, err)

	eng := secutil.NewTestEngine(t)
	boot := func() *Plugin {
		p := New(Config{IDTokenSigningKey: rsaPEM(t, 2048)})
		p.SetOAuth2Store(st)
		require.NoError(t, p.OnInit(ctx, eng))
		return p
	}
	p := boot()

	op, err := p.keys.Active(ctx, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{op.kid, managed.kid}, publishedKIDs(t, p.keys, ""))

	keys, err := st.ListSigningKeys(ctx, "")
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.NotNil(t, keys[0].RetiresAt)
	retiresAt := *keys[0].RetiresAt
	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), retiresAt, time.Minute)

	info, err := p.keys.Describe(ctx, "")
	require.NoError(t, err)
	require.Len(t, info, 2)
	assert.Equal(t, "operator", info[0].Source)
	assert.Equal(t, KeyInfo{
		KID: managed.kid, Alg: "RS256", Source: "managed", Status: "retiring",
		CreatedAt: keys[0].CreatedAt, RetiresAt: keys[0].RetiresAt,
	}, info[1])

	// A restart leaves the retirement date where it was.
	boot()
	keys, err = st.ListSigningKeys(ctx, "")
	require.NoError(t, err)
	assert.True(t, keys[0].RetiresAt.Equal(retiresAt))

	// Once the grace is over the managed key leaves the JWKS.
	require.NoError(t, st.RetireSigningKey(ctx, keys[0].ID, time.Now().Add(-time.Minute)))
	assert.Equal(t, []string{op.kid}, publishedKIDs(t, p.keys, ""))
}

// racingStore loses every CreateSigningKey to a competitor: it inserts the
// competing row at the caller's (scope, generation) first, then reports the
// conflict the unique index would.
type racingStore struct {
	Store
	competitor *SigningKey
	attempted  []id.OAuth2SigningKeyID
	// failList makes every ListSigningKeys after the conflict fail.
	failList bool
	lost     bool
}

func (s *racingStore) CreateSigningKey(ctx context.Context, k *SigningKey) error {
	s.attempted = append(s.attempted, k.ID)
	s.competitor.Generation = k.Generation
	if err := s.Store.CreateSigningKey(ctx, s.competitor); err != nil {
		return err
	}
	s.lost = true
	return ErrSigningKeyConflict
}

func (s *racingStore) ListSigningKeys(ctx context.Context, scope string) ([]*SigningKey, error) {
	if s.failList && s.lost {
		return nil, errors.New("database is down")
	}
	return s.Store.ListSigningKeys(ctx, scope)
}

func TestKeyManager_ConflictTakesCompetingKey(t *testing.T) {
	ctx := context.Background()
	enc := testEncryptor(t)
	rival := newTestKeyManager(t, Config{}, NewMemoryStore(), enc)
	competitor, err := rival.mint("", 0, time.Now())
	require.NoError(t, err)
	want, err := jwkFromStored(competitor)
	require.NoError(t, err)

	st := &racingStore{Store: NewMemoryStore(), competitor: competitor}
	m := newTestKeyManager(t, Config{}, st, enc)
	s, err := m.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, want.KID, s.kid)

	keys, err := st.ListSigningKeys(ctx, "")
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, competitor.ID, keys[0].ID)
	require.Len(t, st.attempted, 1)
	assert.NotEqual(t, competitor.ID, st.attempted[0], "the losing key was never stored")
}

// When the store can't be read after losing a rotation race, the current
// key keeps signing.
func TestKeyManager_ConflictRelistFailureKeepsSigning(t *testing.T) {
	ctx := context.Background()
	enc := testEncryptor(t)
	inner := NewMemoryStore()
	old := newTestKeyManager(t, Config{}, inner, enc)
	old.now = func() time.Time { return time.Now().Add(-100 * 24 * time.Hour) }
	aged, err := old.Active(ctx, "")
	require.NoError(t, err)

	competitor, err := old.mint("", 0, time.Now())
	require.NoError(t, err)
	st := &racingStore{Store: inner, competitor: competitor, failList: true}
	m := newTestKeyManager(t, Config{}, st, enc)
	s, err := m.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, aged.kid, s.kid)
}

// A key that can't be unsealed must not take the JWKS down with it.
func TestKeyManager_PublishedSurvivesActiveFailure(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	s, err := newTestKeyManager(t, Config{}, st, testEncryptor(t)).Active(ctx, "app")
	require.NoError(t, err)

	otherKey, err := bridge.NewAESGCMEncryptor(bytes.Repeat([]byte{9}, 32))
	require.NoError(t, err)
	m := newTestKeyManager(t, Config{}, st, otherKey)
	_, err = m.Active(ctx, "app")
	require.Error(t, err)
	assert.Equal(t, []string{s.kid}, publishedKIDs(t, m, "app"))
}

// Under an operator key, a managed key that never started retiring (the boot
// retire failed, say) is not published: only retiring keys are.
func TestKeyManager_OperatorPathSkipsUnretiredManagedKey(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	_, err := newTestKeyManager(t, Config{}, st, testEncryptor(t)).Active(ctx, "")
	require.NoError(t, err)

	m := newTestKeyManager(t, Config{IDTokenSigningKey: rsaPEM(t, 2048)}, st, testEncryptor(t))
	op, err := m.Active(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, []string{op.kid}, publishedKIDs(t, m, ""))
}
