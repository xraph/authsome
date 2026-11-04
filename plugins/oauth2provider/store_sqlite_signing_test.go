package oauth2provider

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/id"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

func newSigningKeySqlite(t *testing.T) (*SqliteStore, *sqlitedriver.SqliteDB, *grove.DB) {
	t.Helper()
	ctx := context.Background()
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, "file:"+filepath.Join(t.TempDir(), "signing.db")+"?cache=shared"))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlitestore.New(db).Migrate(ctx, SqliteMigrations))
	return NewSqliteStore(db), sdb, db
}

func sqliteKey(gen int64) *SigningKey {
	return &SigningKey{
		ID: id.NewOAuth2SigningKeyID(), Scope: "app", Alg: "RS256", PrivateKey: "v1:sealed",
		PublicJWK: []byte(`{"kty":"RSA","n":"AQAB","e":"AQAB"}`), Generation: gen,
	}
}

// A key stored without a creation time gets one, and a retirement date set
// at insert survives the round trip.
func TestSqliteSigningKeys_DefaultsAndRetiresAt(t *testing.T) {
	ctx := context.Background()
	st, _, _ := newSigningKeySqlite(t)

	before := time.Now().Add(-time.Second)
	fresh := sqliteKey(1)
	require.NoError(t, st.CreateSigningKey(ctx, fresh))
	assert.False(t, fresh.CreatedAt.IsZero(), "CreatedAt was filled in")

	retires := time.Now().Add(48 * time.Hour)
	old := sqliteKey(0)
	old.CreatedAt = time.Now().Add(-time.Hour)
	old.RetiresAt = &retires
	require.NoError(t, st.CreateSigningKey(ctx, old))

	keys, err := st.ListSigningKeys(ctx, "app")
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.Equal(t, fresh.ID.String(), keys[0].ID.String())
	assert.True(t, keys[0].CreatedAt.After(before), "created_at = %v", keys[0].CreatedAt)
	assert.Nil(t, keys[0].RetiresAt)
	require.NotNil(t, keys[1].RetiresAt)
	assert.True(t, retires.Equal(*keys[1].RetiresAt), "retires_at = %v, want %v", *keys[1].RetiresAt, retires)
}

// A failure that isn't a duplicate generation is passed on as itself, so the
// key manager doesn't mistake an outage for a lost race.
func TestSqliteSigningKeys_ErrorsAreNotConflicts(t *testing.T) {
	ctx := context.Background()
	st, _, db := newSigningKeySqlite(t)
	require.NoError(t, db.Close())

	err := st.CreateSigningKey(ctx, sqliteKey(1))
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrSigningKeyConflict))

	_, err = st.ListSigningKeys(ctx, "app")
	require.Error(t, err)
}

// A row whose id isn't a signing key id is reported, not skipped.
func TestSqliteSigningKeys_CorruptRowIsAnError(t *testing.T) {
	ctx := context.Background()
	st, sdb, _ := newSigningKeySqlite(t)
	require.NoError(t, st.CreateSigningKey(ctx, sqliteKey(1)))
	_, err := sdb.NewRaw(`UPDATE authsome_oauth2_signing_keys SET id = ?`, id.NewOAuth2ClientID().String()).Exec(ctx)
	require.NoError(t, err)

	keys, err := st.ListSigningKeys(ctx, "app")
	require.Error(t, err)
	assert.Nil(t, keys)
	assert.Contains(t, err.Error(), "aosk", "the error names the id it could not parse as a signing key id")
}

func TestIsSigningKeyConflict(t *testing.T) {
	assert.False(t, isSigningKeyConflict(nil))
	assert.False(t, isSigningKeyConflict(errors.New("connection refused")))
	for _, msg := range []string{
		"ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)",
		"duplicate key value violates unique constraint",
		"UNIQUE constraint failed: authsome_oauth2_signing_keys.scope, authsome_oauth2_signing_keys.generation",
	} {
		assert.True(t, isSigningKeyConflict(errors.New("insert: "+msg)), msg)
	}
}
