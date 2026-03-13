package apikey_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	memoryStore "github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/user"
)

// mockEngine provides the interfaces that apikey.Plugin.OnInit expects.
type mockEngine struct {
	logger log.Logger
	store  apikey.Store
}

func (m *mockEngine) Logger() log.Logger        { return m.logger }
func (m *mockEngine) APIKeyStore() apikey.Store { return m.store }
func (m *mockEngine) ResolveUser(userID string) (*user.User, error) {
	uid, err := id.ParseUserID(userID)
	if err != nil {
		return nil, err
	}
	return &user.User{ID: uid, Email: "test@example.com", FirstName: "Test User"}, nil
}

func newTestPlugin(cfg ...apikeyPlugin.Config) (*apikeyPlugin.Plugin, *memoryStore.Store) {
	s := memoryStore.New()
	p := apikeyPlugin.New(cfg...)
	p.SetStore(s)
	return p, s
}

func TestPlugin_Name(t *testing.T) {
	p, _ := newTestPlugin()
	assert.Equal(t, "apikey", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) { //nolint:revive // test function signature
	p, _ := newTestPlugin()

	var _ plugin.Plugin = p
	var _ plugin.RouteProvider = p
}

func TestPlugin_Strategy(t *testing.T) {
	p, _ := newTestPlugin()
	s := p.Strategy()

	assert.Equal(t, "apikey", s.Name())
}

func TestPlugin_RegisterInRegistry(t *testing.T) {
	reg := plugin.NewRegistry(log.NewNoopLogger())
	p, _ := newTestPlugin()

	reg.Register(p)

	assert.Len(t, reg.Plugins(), 1)
	assert.Equal(t, "apikey", reg.Plugins()[0].Name())
	assert.Len(t, reg.RouteProviders(), 1)
}

func TestPlugin_CreateKey(t *testing.T) {
	p, _ := newTestPlugin()

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	appID := id.NewAppID()
	userID := id.NewUserID()

	body, _ := json.Marshal(map[string]any{
		"app_id":  appID.String(),
		"user_id": userID.String(),
		"name":    "Test Key",
		"scopes":  []string{"read", "write"},
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/auth/keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp["id"])
	assert.NotEmpty(t, resp["key"])
	assert.NotEmpty(t, resp["key_prefix"])
	assert.NotEmpty(t, resp["public_key"])
	assert.NotEmpty(t, resp["public_key_prefix"])
	assert.Equal(t, "Test Key", resp["name"])

	// The secret key should be recognized as a secret key (sk_* or ask_)
	secretKey := resp["key"].(string)
	assert.True(t, apikey.IsSecretKey(secretKey), "key should be a secret key, got: %s", secretKey)
	assert.True(t, len(secretKey) > 12, "secret key should be at least 12 chars")

	// The public key should be recognized as a public key (pk_*)
	publicKey := resp["public_key"].(string)
	assert.True(t, apikey.IsPublicKey(publicKey), "public_key should be a public key, got: %s", publicKey)
	assert.True(t, len(publicKey) > 12, "public key should be at least 12 chars")

	// Prefixes should match
	publicKeyPrefix := resp["public_key_prefix"].(string)
	assert.True(t, len(publicKeyPrefix) > 0, "public_key_prefix should not be empty")
}

func TestPlugin_CreateKey_ValidationErrors(t *testing.T) {
	p, _ := newTestPlugin()

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	tests := []struct {
		name string
		body map[string]any
	}{
		{"missing app_id", map[string]any{"user_id": id.NewUserID().String(), "name": "key"}},
		{"missing user_id", map[string]any{"app_id": id.NewAppID().String(), "name": "key"}},
		{"missing name", map[string]any{"app_id": id.NewAppID().String(), "user_id": id.NewUserID().String()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/auth/keys", bytes.NewReader(body))
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestPlugin_CreateKey_MaxKeysLimit(t *testing.T) {
	p, store := newTestPlugin(apikeyPlugin.Config{MaxKeysPerUser: 2})

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	appID := id.NewAppID()
	userID := id.NewUserID()

	// Pre-create 2 keys directly in the store
	for i := 0; i < 2; i++ {
		raw, hash, prefix, err := apikey.GenerateKey()
		require.NoError(t, err)
		_ = raw
		now := time.Now()
		err = store.CreateAPIKey(context.Background(), &apikey.APIKey{
			ID:        id.NewAPIKeyID(),
			AppID:     appID,
			UserID:    userID,
			Name:      "existing",
			KeyHash:   hash,
			KeyPrefix: prefix,
			CreatedAt: now,
			UpdatedAt: now,
		})
		require.NoError(t, err)
	}

	// Third key should be rejected
	body, _ := json.Marshal(map[string]any{
		"app_id":  appID.String(),
		"user_id": userID.String(),
		"name":    "One Too Many",
	})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/auth/keys", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPlugin_ListKeys(t *testing.T) {
	p, store := newTestPlugin()

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	appID := id.NewAppID()
	userID := id.NewUserID()
	now := time.Now()

	// Create a key in the store
	_, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)
	err = store.CreateAPIKey(context.Background(), &apikey.APIKey{
		ID:        id.NewAPIKeyID(),
		AppID:     appID,
		UserID:    userID,
		Name:      "Test Key",
		KeyHash:   hash,
		KeyPrefix: prefix,
		Scopes:    []string{"read"},
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)

	// List by app
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/auth/keys?app_id="+appID.String(), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total"])

	keys := resp["keys"].([]any)
	require.Len(t, keys, 1)

	firstKey := keys[0].(map[string]any)
	assert.Equal(t, "Test Key", firstKey["name"])
	assert.Equal(t, prefix, firstKey["key_prefix"])
	// raw key should NOT be present in list
	assert.Nil(t, firstKey["key"])
}

func TestPlugin_ListKeys_ByUser(t *testing.T) {
	p, store := newTestPlugin()

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	appID := id.NewAppID()
	user1 := id.NewUserID()
	user2 := id.NewUserID()
	now := time.Now()

	// Create keys for two different users
	for _, uid := range []id.UserID{user1, user2} {
		_, hash, prefix, err := apikey.GenerateKey()
		require.NoError(t, err)
		err = store.CreateAPIKey(context.Background(), &apikey.APIKey{
			ID: id.NewAPIKeyID(), AppID: appID, UserID: uid,
			Name: "Key", KeyHash: hash, KeyPrefix: prefix,
			CreatedAt: now, UpdatedAt: now,
		})
		require.NoError(t, err)
	}

	// List by user1 only
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/auth/keys?app_id="+appID.String()+"&user_id="+user1.String(), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total"])
}

func TestPlugin_RevokeKey(t *testing.T) {
	p, store := newTestPlugin()

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	appID := id.NewAppID()
	userID := id.NewUserID()
	keyID := id.NewAPIKeyID()
	now := time.Now()

	_, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)

	err = store.CreateAPIKey(context.Background(), &apikey.APIKey{
		ID: keyID, AppID: appID, UserID: userID,
		Name: "Revoke Me", KeyHash: hash, KeyPrefix: prefix,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/auth/keys/"+keyID.String(), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify the key is revoked
	key, err := store.GetAPIKey(context.Background(), keyID)
	require.NoError(t, err)
	assert.True(t, key.Revoked)
}

func TestPlugin_RevokeKey_NotFound(t *testing.T) {
	p, _ := newTestPlugin()

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	fakeID := id.NewAPIKeyID()
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/auth/keys/"+fakeID.String(), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPlugin_Strategy_NotApplicable(t *testing.T) {
	p, _ := newTestPlugin()
	s := p.Strategy()

	// Request without API key header should return not applicable
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/some-endpoint", nil)
	_, err := s.Authenticate(context.Background(), req)

	assert.Error(t, err)
	var target strategy.NotApplicableError
	assert.True(t, errors.As(err, &target), "should return NotApplicableError")
}

func TestPlugin_Strategy_Authenticate(t *testing.T) {
	p, store := newTestPlugin()
	require.NoError(t, p.OnInit(context.Background(), &mockEngine{logger: log.NewNoopLogger(), store: store}))
	s := p.Strategy()

	appID := id.NewAppID()
	userID := id.NewUserID()

	raw, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)

	now := time.Now()
	err = store.CreateAPIKey(context.Background(), &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: userID,
		Name: "Auth Key", KeyHash: hash, KeyPrefix: prefix,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	// Authenticate with Bearer token
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("X-App-ID", appID.String())

	result, err := s.Authenticate(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.User, "result should contain resolved user")
	assert.NotNil(t, result.Session, "result should contain synthetic session")
	assert.Equal(t, userID, result.User.ID)
	assert.Equal(t, appID, result.Session.AppID)
	assert.Equal(t, userID, result.Session.UserID)
}

func TestPlugin_Strategy_Authenticate_XAPIKey(t *testing.T) {
	p, store := newTestPlugin()
	require.NoError(t, p.OnInit(context.Background(), &mockEngine{logger: log.NewNoopLogger(), store: store}))
	s := p.Strategy()

	appID := id.NewAppID()
	userID := id.NewUserID()

	raw, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)

	now := time.Now()
	err = store.CreateAPIKey(context.Background(), &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: userID,
		Name: "X-API-Key", KeyHash: hash, KeyPrefix: prefix,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	// Authenticate with X-API-Key header
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
	req.Header.Set("X-API-Key", raw)
	req.Header.Set("X-App-ID", appID.String())

	result, err := s.Authenticate(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.User, "result should contain resolved user")
	assert.NotNil(t, result.Session, "result should contain synthetic session")
}

func TestPlugin_Strategy_Authenticate_InvalidKey(t *testing.T) {
	p, store := newTestPlugin()
	s := p.Strategy()

	appID := id.NewAppID()
	userID := id.NewUserID()

	_, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)

	now := time.Now()
	err = store.CreateAPIKey(context.Background(), &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: userID,
		Name: "Key", KeyHash: hash, KeyPrefix: prefix,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	// Try with wrong key but same prefix
	fakeKey := "ask_" + prefix[4:] + "ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+fakeKey)
	req.Header.Set("X-App-ID", appID.String())

	_, err = s.Authenticate(context.Background(), req)
	assert.Error(t, err)
}

func TestPlugin_Strategy_Authenticate_RevokedKey(t *testing.T) {
	p, store := newTestPlugin()
	s := p.Strategy()

	appID := id.NewAppID()
	userID := id.NewUserID()

	raw, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)

	now := time.Now()
	key := &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: userID,
		Name: "Revoked", KeyHash: hash, KeyPrefix: prefix,
		Revoked:   true,
		CreatedAt: now, UpdatedAt: now,
	}
	err = store.CreateAPIKey(context.Background(), key)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("X-App-ID", appID.String())

	_, err = s.Authenticate(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestPlugin_Strategy_Authenticate_ExpiredKey(t *testing.T) {
	p, store := newTestPlugin()
	s := p.Strategy()

	appID := id.NewAppID()
	userID := id.NewUserID()

	raw, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)

	expired := time.Now().Add(-1 * time.Hour)
	now := time.Now()
	key := &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: userID,
		Name: "Expired", KeyHash: hash, KeyPrefix: prefix,
		ExpiresAt: &expired,
		CreatedAt: now, UpdatedAt: now,
	}
	err = store.CreateAPIKey(context.Background(), key)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("X-App-ID", appID.String())

	_, err = s.Authenticate(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked or expired")
}

func TestPlugin_Strategy_Authenticate_RejectsPublicKey(t *testing.T) {
	p, store := newTestPlugin()
	require.NoError(t, p.OnInit(context.Background(), &mockEngine{logger: log.NewNoopLogger(), store: store}))
	s := p.Strategy()

	appID := id.NewAppID()
	userID := id.NewUserID()

	publicKey, secretKey, secretHash, publicPrefix, secretPrefix, err := apikey.GenerateKeyPair()
	require.NoError(t, err)
	_ = secretKey

	now := time.Now()
	err = store.CreateAPIKey(context.Background(), &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: userID,
		Name: "KeyPair Test", KeyHash: secretHash, KeyPrefix: secretPrefix,
		PublicKey: publicKey, PublicKeyPrefix: publicPrefix,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	// Attempting to authenticate with a public key should fail
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+publicKey)
	req.Header.Set("X-App-ID", appID.String())

	_, err = s.Authenticate(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "public key")
}

func TestAPIKey_GenerateKeyPair(t *testing.T) {
	publicKey, secretKey, secretHash, publicPrefix, secretPrefix, err := apikey.GenerateKeyPair()
	require.NoError(t, err)

	assert.True(t, apikey.IsPublicKey(publicKey), "public key should start with pk_")
	assert.True(t, apikey.IsSecretKey(secretKey), "secret key should start with sk_")
	assert.NotEmpty(t, secretHash)
	assert.NotEmpty(t, publicPrefix)
	assert.NotEmpty(t, secretPrefix)

	// Secret key should verify against its hash
	assert.True(t, apikey.VerifyKey(secretKey, secretHash))
	assert.False(t, apikey.VerifyKey(publicKey, secretHash), "public key should not match secret hash")

	// Detect environment types
	envType, ok := apikey.DetectEnvironmentType(publicKey)
	assert.True(t, ok)
	assert.NotEmpty(t, envType)

	envType, ok = apikey.DetectEnvironmentType(secretKey)
	assert.True(t, ok)
	assert.NotEmpty(t, envType)
}

func TestAPIKey_GenerateKey(t *testing.T) {
	raw, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)

	assert.True(t, len(raw) > 12)
	assert.Equal(t, "ask_", raw[:4])
	assert.NotEmpty(t, hash)
	assert.Equal(t, raw[:12], prefix)

	// Verify works
	assert.True(t, apikey.VerifyKey(raw, hash))
	assert.False(t, apikey.VerifyKey("wrong_key", hash))
}

func TestAPIKey_IsExpired(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	past := time.Now().Add(-1 * time.Hour)

	k1 := &apikey.APIKey{ExpiresAt: nil}
	assert.False(t, k1.IsExpired(), "nil ExpiresAt should not be expired")

	k2 := &apikey.APIKey{ExpiresAt: &future}
	assert.False(t, k2.IsExpired(), "future ExpiresAt should not be expired")

	k3 := &apikey.APIKey{ExpiresAt: &past}
	assert.True(t, k3.IsExpired(), "past ExpiresAt should be expired")
}

func TestAPIKey_IsValid(t *testing.T) {
	k1 := &apikey.APIKey{Revoked: false}
	assert.True(t, k1.IsValid())

	k2 := &apikey.APIKey{Revoked: true}
	assert.False(t, k2.IsValid())

	past := time.Now().Add(-1 * time.Hour)
	k3 := &apikey.APIKey{Revoked: false, ExpiresAt: &past}
	assert.False(t, k3.IsValid())
}

func TestPlugin_CreateKey_DefaultExpiry(t *testing.T) {
	p, _ := newTestPlugin(apikeyPlugin.Config{DefaultExpiry: 30 * 24 * time.Hour})

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	body, _ := json.Marshal(map[string]any{
		"app_id":  id.NewAppID().String(),
		"user_id": id.NewUserID().String(),
		"name":    "Expiring Key",
	})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/auth/keys", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["expires_at"], "should have an expires_at when DefaultExpiry is set")
}
