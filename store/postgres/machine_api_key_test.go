//go:build integration

package postgres_test

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"
	"github.com/xraph/grove/drivers/pgdriver/pgmigrate"
	"github.com/xraph/grove/migrate"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/store"
	pgstore "github.com/xraph/authsome/store/postgres"
)

const machineKeyMigration = "20261009000001"

func machineAccount(t *testing.T, s *pgstore.Store, appID id.AppID, name string) *serviceaccount.ServiceAccount {
	t.Helper()
	sa := &serviceaccount.ServiceAccount{ID: id.NewServiceAccountID(), AppID: appID, EnvID: testEnvID(t, appID), Name: name, Kind: principal.KindWorkload, Active: true, Scopes: []string{"dispatch:send"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.CreateServiceAccount(context.Background(), sa))
	return sa
}

// The CI conformance selector includes this test. All owner resolution comes
// from the public Engine and the PostgreSQL store.
func TestStoreConformance_MachineAPIKeyEngine(t *testing.T) {
	s, db := setupTestDatabase(t, true)
	ctx := context.Background()
	a := createTestApp(t, s, "machine-engine")
	other := createTestApp(t, s, "other-engine")
	u := createTestUser(t, s, a.ID, "human@test.com")
	p := apikeyPlugin.New()
	cfg := authsome.DefaultConfig()
	cfg.AppID = a.ID.String()
	cfg.RateLimit.Enabled = true
	cfg.RateLimit.APIKeyFailureLimit = 2
	eng := secutil.NewTestEngine(t, authsome.WithStore(s), authsome.WithConfig(cfg), authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()), authsome.WithPlugin(p))
	sa, err := eng.CreateServiceAccountInEnvironment(ctx, a.ID, testEnvID(t, a.ID), "runner", "", []string{"dispatch:send"})
	require.NoError(t, err)
	t.Run("account environment validation", func(t *testing.T) {
		before, listErr := eng.ListServiceAccounts(ctx, a.ID, 100)
		require.NoError(t, listErr)
		for _, envID := range []id.EnvironmentID{id.Nil, id.NewEnvironmentID(), testEnvID(t, other.ID)} {
			denied, createErr := eng.CreateServiceAccountInEnvironment(ctx, a.ID, envID, "invalid", "", nil)
			require.Nil(t, denied)
			if envID.IsNil() {
				require.ErrorIs(t, createErr, authsome.ErrServiceAccountEnvironmentRequired)
			} else {
				require.ErrorIs(t, createErr, store.ErrNotFound)
			}
		}
		after, listErr := eng.ListServiceAccounts(ctx, a.ID, 100)
		require.NoError(t, listErr)
		assert.Equal(t, before.Total, after.Total)
		keys, listErr := s.ListAPIKeysByApp(ctx, a.ID)
		require.NoError(t, listErr)
		require.Empty(t, keys)
	})
	key, raw, err := eng.CreateServiceAccountAPIKey(ctx, sa.ID, "runner key", []string{"dispatch:send"}, nil)
	require.NoError(t, err)
	checkMachineKey(t, s, key, sa)

	req := httptest.NewRequestWithContext(ctx, "GET", "/api/data", nil)
	req.RemoteAddr = "203.0.113.1:4000"
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("X-App-ID", a.ID.String())
	req.Header.Set("X-Env-ID", testEnvID(t, other.ID).String())
	result, err := p.Strategy().Authenticate(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result.Session)
	assert.Nil(t, result.User)
	assert.True(t, result.Session.UserID.IsNil())
	assert.Equal(t, principal.Ref{Kind: principal.KindService, ID: sa.ID.String()}, result.Session.Subject())
	assert.Equal(t, a.ID, result.Session.AppID)
	assert.Equal(t, sa.EnvID, result.Session.EnvID)
	got, err := s.GetAPIKey(ctx, key.ID)
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt)
	checkMachineKey(t, s, key, sa)

	t.Run("wrong app header", func(t *testing.T) {
		req.Header.Set("X-App-ID", other.ID.String())
		gotResult, authErr := p.Strategy().Authenticate(ctx, req)
		assert.Nil(t, gotResult)
		require.EqualError(t, authErr, "apikey: key not found")
		req.Header.Set("X-App-ID", a.ID.String())
	})
	t.Run("scope escalation", func(t *testing.T) {
		denied, secret, mintErr := eng.CreateServiceAccountAPIKey(ctx, sa.ID, "too wide", []string{"admin:*"}, nil)
		require.ErrorIs(t, mintErr, authsome.ErrScopeEscalation)
		assert.Nil(t, denied)
		assert.Empty(t, secret)
	})
	t.Run("empty environment", func(t *testing.T) {
		unscoped, createErr := eng.CreateServiceAccount(ctx, a.ID, "app wide", "", nil)
		require.NoError(t, createErr)
		before, listErr := s.ListAPIKeysByApp(ctx, a.ID)
		require.NoError(t, listErr)
		denied, secret, mintErr := eng.CreateServiceAccountAPIKey(ctx, unscoped.ID, "no environment", nil, nil)
		require.ErrorIs(t, mintErr, authsome.ErrServiceAccountEnvironmentRequired)
		assert.Nil(t, denied)
		assert.Empty(t, secret)
		after, listErr := s.ListAPIKeysByApp(ctx, a.ID)
		require.NoError(t, listErr)
		assert.Len(t, after, len(before))
	})
	t.Run("key states", func(t *testing.T) {
		for _, state := range []string{"revoked", "expired"} {
			t.Run(state, func(t *testing.T) {
				req.RemoteAddr = "203.0.113.2:4000"
				key.Revoked = state == "revoked"
				if state == "expired" {
					past := time.Now().Add(-time.Hour)
					key.ExpiresAt = &past
				}
				require.NoError(t, s.UpdateAPIKey(ctx, key))
				denied, authErr := p.Strategy().Authenticate(ctx, req)
				assert.Nil(t, denied)
				require.EqualError(t, authErr, "apikey: key is revoked or expired")
				checkMachineKey(t, s, key, sa)
			})
		}
		key.Revoked = false
		key.ExpiresAt = nil
		require.NoError(t, s.UpdateAPIKey(ctx, key))
	})
	t.Run("account states and failure budget", func(t *testing.T) {
		for i, state := range []string{"inactive", "expired", "unavailable"} {
			t.Run(state, func(t *testing.T) {
				req.RemoteAddr = []string{"203.0.113.3:4000", "203.0.113.4:4000", "203.0.113.5:4000"}[i]
				sa.Active = state != "inactive"
				if state == "expired" {
					past := time.Now().Add(-time.Hour)
					sa.ExpiresAt = &past
				}
				require.NoError(t, s.UpdateServiceAccount(ctx, sa))
				if state == "unavailable" {
					_, renameErr := db.Exec(ctx, "ALTER TABLE authsome_service_accounts RENAME TO machine_accounts_unavailable")
					require.NoError(t, renameErr)
				}
				for range 2 {
					denied, authErr := p.Strategy().Authenticate(ctx, req)
					assert.Nil(t, denied)
					assert.EqualError(t, authErr, "apikey: service account is not active")
				}
				denied, authErr := p.Strategy().Authenticate(ctx, req)
				assert.Nil(t, denied)
				assert.EqualError(t, authErr, "apikey: too many failed attempts from this address")
				if state == "unavailable" {
					_, renameErr := db.Exec(ctx, "ALTER TABLE machine_accounts_unavailable RENAME TO authsome_service_accounts")
					require.NoError(t, renameErr)
				}
				sa.Active = true
				sa.ExpiresAt = nil
				require.NoError(t, s.UpdateServiceAccount(ctx, sa))
			})
		}
	})
	t.Run("human key", func(t *testing.T) {
		human, humanRaw := humanAPIKey(t, a.ID, sa.EnvID, u.ID)
		require.NoError(t, s.CreateAPIKey(ctx, human))
		req.RemoteAddr = "203.0.113.6:4000"
		req.Header.Set("Authorization", "Bearer "+humanRaw)
		humanResult, authErr := p.Strategy().Authenticate(ctx, req)
		require.NoError(t, authErr)
		require.NotNil(t, humanResult.User)
		assert.Equal(t, u.ID, humanResult.User.ID)
		assert.Equal(t, sa.EnvID, humanResult.Session.EnvID)
		keys, listErr := s.ListAPIKeysByUser(ctx, a.ID, u.ID)
		require.NoError(t, listErr)
		require.Len(t, keys, 1)
		assert.Equal(t, human.ID, keys[0].ID)
		keys, listErr = s.ListAPIKeysByUser(ctx, a.ID, id.Nil)
		require.NoError(t, listErr)
		assert.Empty(t, keys)
	})
	t.Run("deleted account", func(t *testing.T) {
		require.NoError(t, eng.DeleteServiceAccount(ctx, sa.ID))
		_, getErr := s.GetAPIKey(ctx, key.ID)
		require.ErrorIs(t, getErr, store.ErrNotFound)
		req.RemoteAddr = "203.0.113.7:4000"
		req.Header.Set("Authorization", "Bearer "+raw)
		denied, authErr := p.Strategy().Authenticate(ctx, req)
		assert.Nil(t, denied)
		require.EqualError(t, authErr, "apikey: key not found")
	})
	t.Run("app cascade", func(t *testing.T) {
		account := machineAccount(t, s, a.ID, "app-delete")
		appKey, _, mintErr := eng.CreateServiceAccountAPIKey(ctx, account.ID, "app-delete", nil, nil)
		require.NoError(t, mintErr)
		require.NoError(t, s.DeleteApp(ctx, a.ID))
		_, getErr := s.GetAPIKey(ctx, appKey.ID)
		require.ErrorIs(t, getErr, store.ErrNotFound)
		_, getErr = s.GetServiceAccount(ctx, account.ID)
		require.ErrorIs(t, getErr, store.ErrNotFound)
		_, getErr = s.GetApp(ctx, other.ID)
		require.NoError(t, getErr)
	})
}

func checkMachineKey(t *testing.T, s *pgstore.Store, key *apikey.APIKey, sa *serviceaccount.ServiceAccount) {
	t.Helper()
	ctx := context.Background()
	lookups := []func() (*apikey.APIKey, error){
		func() (*apikey.APIKey, error) { return s.GetAPIKey(ctx, key.ID) },
		func() (*apikey.APIKey, error) { return s.GetAPIKeyByPrefix(ctx, key.AppID, key.KeyPrefix) },
		func() (*apikey.APIKey, error) { return s.FindByPrefix(ctx, key.KeyPrefix) },
		func() (*apikey.APIKey, error) { return s.GetAPIKeyByPublicKey(ctx, key.AppID, key.PublicKey) },
	}
	for _, lookup := range lookups {
		got, err := lookup()
		require.NoError(t, err)
		assert.Equal(t, sa.ID, got.ServiceAccountID)
		assert.True(t, got.UserID.IsNil())
		assert.Equal(t, sa.AppID, got.AppID)
		assert.Equal(t, sa.EnvID, got.EnvID)
		assert.Equal(t, key.Scopes, got.Scopes)
		assert.Equal(t, key.KeyHash, got.KeyHash)
		assert.Equal(t, key.PublicKey, got.PublicKey)
		assert.Equal(t, key.PublicKeyPrefix, got.PublicKeyPrefix)
		assert.Equal(t, key.KeyPrefix, got.KeyPrefix)
		assert.Equal(t, key.Revoked, got.Revoked)
	}
	keys, err := s.ListAPIKeysByApp(ctx, key.AppID)
	require.NoError(t, err)
	found := false
	for _, got := range keys {
		if got.ID == key.ID {
			found = true
			assert.Equal(t, sa.ID, got.ServiceAccountID)
		}
	}
	require.True(t, found)
}

func humanAPIKey(t *testing.T, appID id.AppID, envID id.EnvironmentID, userID id.UserID) (*apikey.APIKey, string) {
	t.Helper()
	pub, raw, hash, pubPrefix, prefix, err := apikey.GenerateKeyPair()
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &apikey.APIKey{ID: id.NewAPIKeyID(), AppID: appID, EnvID: envID, UserID: userID, Name: "human", KeyHash: hash, KeyPrefix: prefix, PublicKey: pub, PublicKeyPrefix: pubPrefix, Scopes: []string{"dispatch:send"}, CreatedAt: now, UpdatedAt: now}, raw
}

func previousAPIKeySchema(t *testing.T, db *pgdriver.PgDB) {
	t.Helper()
	previous := migrate.NewGroup("authsome")
	for _, migration := range pgstore.Migrations.Migrations() {
		if migration.Version < machineKeyMigration {
			previous.MustRegister(migration)
		}
	}
	_, err := migrate.NewOrchestrator(pgmigrate.New(db), previous).Migrate(context.Background())
	require.NoError(t, err)
}

func insertLegacyKey(t *testing.T, db *pgdriver.PgDB, key *apikey.APIKey) {
	t.Helper()
	_, err := db.Exec(context.Background(), `INSERT INTO authsome_api_keys
 (id,app_id,env_id,user_id,name,key_hash,key_prefix,public_key,public_key_prefix,scopes,expires_at,last_used_at,revoked,created_at,updated_at)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		key.ID.String(), key.AppID.String(), key.EnvID.String(), key.UserID.String(), key.Name, key.KeyHash, key.KeyPrefix, key.PublicKey, key.PublicKeyPrefix, "dispatch:send", key.ExpiresAt, key.LastUsedAt, key.Revoked, key.CreatedAt, key.UpdatedAt)
	require.NoError(t, err)
}

func TestMigration_MachineAPIKeyOwners(t *testing.T) {
	s, db := setupTestDatabase(t, false)
	ctx := context.Background()
	previousAPIKeySchema(t, db)
	a := createTestApp(t, s, "upgrade")
	other := createTestApp(t, s, "upgrade-other")
	u := createTestUser(t, s, a.ID, "upgrade@test.com")
	otherUser := createTestUser(t, s, other.ID, "other@test.com")
	sa := machineAccount(t, s, a.ID, "upgrade-runner")
	human, _ := humanAPIKey(t, a.ID, sa.EnvID, u.ID)
	expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	touched := human.CreatedAt
	human.ExpiresAt = &expiry
	human.LastUsedAt = &touched
	human.Revoked = true
	insertLegacyKey(t, db, human)
	require.NoError(t, s.Migrate(ctx))
	require.NoError(t, s.Migrate(ctx))
	got, err := s.GetAPIKey(ctx, human.ID)
	require.NoError(t, err)
	assertKeyPreserved(t, human, got)

	machine := *human
	machine.ID = id.NewAPIKeyID()
	machine.UserID = id.Nil
	machine.ServiceAccountID = sa.ID
	machine.KeyHash = "machine-hash"
	machine.KeyPrefix = "machine-prefix"
	machine.PublicKey = "machine-public"
	require.NoError(t, s.CreateAPIKey(ctx, &machine))
	machine.Name = "updated"
	require.NoError(t, s.UpdateAPIKey(ctx, &machine))
	require.NoError(t, s.TouchAPIKey(ctx, machine.ID, time.Now()))
	checkMachineKey(t, s, &machine, sa)

	t.Run("database owner and scope constraints", func(t *testing.T) {
		cases := []struct {
			name, query string
			args        []any
		}{
			{"ownerless", "UPDATE authsome_api_keys SET service_account_id=NULL WHERE id=$1", []any{machine.ID.String()}},
			{"mixed", "UPDATE authsome_api_keys SET user_id=$1 WHERE id=$2", []any{u.ID.String(), machine.ID.String()}},
			{"empty machine", "UPDATE authsome_api_keys SET service_account_id='' WHERE id=$1", []any{machine.ID.String()}},
			{"missing machine", "UPDATE authsome_api_keys SET service_account_id=$1 WHERE id=$2", []any{id.NewServiceAccountID().String(), machine.ID.String()}},
			{"wrong app machine", "UPDATE authsome_api_keys SET app_id=$1,env_id=$2 WHERE id=$3", []any{other.ID.String(), testEnvID(t, other.ID).String(), machine.ID.String()}},
			{"wrong environment", "UPDATE authsome_api_keys SET env_id=$1 WHERE id=$2", []any{testEnvID(t, other.ID).String(), machine.ID.String()}},
			{"wrong app human", "UPDATE authsome_api_keys SET user_id=$1 WHERE id=$2", []any{otherUser.ID.String(), human.ID.String()}},
			{"missing human", "UPDATE authsome_api_keys SET user_id=$1 WHERE id=$2", []any{id.NewUserID().String(), human.ID.String()}},
			{"empty human", "UPDATE authsome_api_keys SET user_id='' WHERE id=$1", []any{human.ID.String()}},
			{"move account", "UPDATE authsome_service_accounts SET app_id=$1 WHERE id=$2", []any{other.ID.String(), sa.ID.String()}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) { _, updateErr := db.Exec(ctx, tc.query, tc.args...); require.Error(t, updateErr) })
		}
		invalid := machine
		invalid.ID = id.NewAPIKeyID()
		invalid.KeyHash = "new-invalid"
		invalid.KeyPrefix = "new-invalid"
		invalid.ServiceAccountID = id.Nil
		require.Error(t, s.CreateAPIKey(ctx, &invalid))
		invalid.ServiceAccountID = sa.ID
		invalid.UserID = u.ID
		require.Error(t, s.CreateAPIKey(ctx, &invalid))
	})
	t.Run("old reader cannot get human identity", func(t *testing.T) {
		// This is the former owner's exact Grove field shape. A NULL either
		// fails its scan or yields an empty string, which the strategy refuses.
		type oldOwner struct {
			grove.BaseModel `grove:"table:authsome_api_keys,alias:ak"`
			UserID          string `grove:"user_id,notnull"`
		}
		legacy := new(oldOwner)
		scanErr := db.NewSelect(legacy).Where("id = ?", machine.ID.String()).Scan(ctx)
		if scanErr == nil {
			require.Empty(t, legacy.UserID)
		}
		var owner sql.NullString
		require.NoError(t, db.QueryRow(ctx, "SELECT user_id FROM authsome_api_keys WHERE id=$1", machine.ID.String()).Scan(&owner))
		require.False(t, owner.Valid)
	})
	t.Run("non destructive downgrade", func(t *testing.T) {
		orchestrator := migrate.NewOrchestrator(pgmigrate.New(db), pgstore.Migrations)
		_, rollbackErr := orchestrator.Rollback(ctx)
		require.ErrorContains(t, rollbackErr, "while machine keys exist")
		checkMachineKey(t, s, &machine, sa)
		retained, readErr := s.GetAPIKey(ctx, human.ID)
		require.NoError(t, readErr)
		assertKeyPreserved(t, human, retained)
		require.NoError(t, s.DeleteAPIKey(ctx, machine.ID))
		_, rollbackErr = orchestrator.Rollback(ctx)
		require.NoError(t, rollbackErr)
		var owner string
		require.NoError(t, db.QueryRow(ctx, "SELECT user_id FROM authsome_api_keys WHERE id=$1", human.ID.String()).Scan(&owner))
		require.Equal(t, u.ID.String(), owner)
		_, nullErr := db.Exec(ctx, "UPDATE authsome_api_keys SET user_id=NULL WHERE id=$1", human.ID.String())
		require.Error(t, nullErr)
		_, orphanErr := db.Exec(ctx, "UPDATE authsome_api_keys SET user_id=$1 WHERE id=$2", id.NewUserID().String(), human.ID.String())
		require.Error(t, orphanErr)
		require.NoError(t, s.Migrate(ctx))
		require.NoError(t, s.Migrate(ctx))
		retained, readErr = s.GetAPIKey(ctx, human.ID)
		require.NoError(t, readErr)
		assertKeyPreserved(t, human, retained)
	})
}

func TestMigration_MachineAPIKeyRejectsMalformedLegacyScope(t *testing.T) {
	s, db := setupTestDatabase(t, false)
	ctx := context.Background()
	previousAPIKeySchema(t, db)
	a := createTestApp(t, s, "malformed")
	other := createTestApp(t, s, "malformed-other")
	u := createTestUser(t, s, other.ID, "wrong@test.com")
	key, _ := humanAPIKey(t, a.ID, testEnvID(t, a.ID), u.ID)
	insertLegacyKey(t, db, key)
	require.Error(t, s.Migrate(ctx))
	var owner, hash string
	require.NoError(t, db.QueryRow(ctx, "SELECT user_id,key_hash FROM authsome_api_keys WHERE id=$1", key.ID.String()).Scan(&owner, &hash))
	assert.Equal(t, key.UserID.String(), owner)
	assert.Equal(t, key.KeyHash, hash)
	var count int
	require.NoError(t, db.QueryRow(ctx, "SELECT count(*) FROM information_schema.columns WHERE table_name='authsome_api_keys' AND column_name='service_account_id'").Scan(&count))
	require.Zero(t, count, "failed upgrade must roll back every schema change")
}

func assertKeyPreserved(t *testing.T, want, got *apikey.APIKey) {
	t.Helper()
	normalized := *got
	normalized.CreatedAt = normalized.CreatedAt.UTC()
	normalized.UpdatedAt = normalized.UpdatedAt.UTC()
	if got.ExpiresAt != nil {
		at := got.ExpiresAt.UTC()
		normalized.ExpiresAt = &at
	}
	if got.LastUsedAt != nil {
		at := got.LastUsedAt.UTC()
		normalized.LastUsedAt = &at
	}
	require.Equal(t, want, &normalized)
}

func TestMigration_MachineAPIKeyDowngradeWaitsForIssuer(t *testing.T) {
	s, db := setupTestDatabase(t, true)
	ctx := context.Background()
	a := createTestApp(t, s, "rollback-race")
	sa := machineAccount(t, s, a.ID, "concurrent-issuer")
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }() // Cleanup after commit or test failure.
	keyID := id.NewAPIKeyID()
	_, err = tx.Exec(ctx, `INSERT INTO authsome_api_keys (id,app_id,env_id,service_account_id,name,key_hash,key_prefix) VALUES ($1,$2,$3,$4,'concurrent','concurrent-hash','concurrent-prefix')`, keyID.String(), a.ID.String(), sa.EnvID.String(), sa.ID.String())
	require.NoError(t, err)
	finished := make(chan error, 1)
	go func() {
		_, rollbackErr := migrate.NewOrchestrator(pgmigrate.New(db), pgstore.Migrations).Rollback(ctx)
		finished <- rollbackErr
	}()
	require.Eventually(t, func() bool {
		var blocked int
		queryErr := db.QueryRow(ctx, `SELECT count(*) FROM pg_locks WHERE relation='authsome_api_keys'::regclass AND mode='AccessExclusiveLock' AND NOT granted`).Scan(&blocked)
		return queryErr == nil && blocked > 0
	}, 5*time.Second, 10*time.Millisecond, "rollback must wait for the issuer's transaction")
	require.NoError(t, tx.Commit())
	select {
	case rollbackErr := <-finished:
		require.ErrorContains(t, rollbackErr, "while machine keys exist")
	case <-time.After(5 * time.Second):
		t.Fatal("rollback did not finish after issuer committed")
	}
	got, err := s.GetAPIKey(ctx, keyID)
	require.NoError(t, err)
	require.Equal(t, sa.ID, got.ServiceAccountID)
	require.True(t, got.UserID.IsNil())
	require.NoError(t, s.Migrate(ctx))
}
