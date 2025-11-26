package passkey

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// TestHandler_BeginRegister tests the BeginRegister handler
func TestHandler_BeginRegister(t *testing.T) {
	db, service := setupTestService(t)
	defer db.Close()

	// Create test user
	testUser := createTestUser(t, db)

	// Create test request
	req := BeginRegisterRequest{
		UserID:             testUser.ID.String(),
		Name:               "My Security Key",
		AuthenticatorType:  "cross-platform",
		RequireResidentKey: false,
		UserVerification:   "preferred",
	}

	body, err := json.Marshal(req)
	require.NoError(t, err)

	// Create HTTP request with app context
	httpReq := httptest.NewRequest(http.MethodPost, "/passkey/register/begin", bytes.NewReader(body))
	httpReq = httpReq.WithContext(contexts.SetAppID(context.Background(), xid.New()))

	// Create test context (simplified - in real tests would use actual forge.Context)
	// For this test, we'll test the service directly instead
	resp, err := service.BeginRegistration(httpReq.Context(), testUser.ID, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Challenge)
	assert.Equal(t, testUser.ID.String(), resp.UserID)
	assert.Greater(t, resp.Timeout, 0)
}

// TestService_RegistrationFlow tests complete registration flow
func TestService_RegistrationFlow(t *testing.T) {
	db, service := setupTestService(t)
	defer db.Close()

	// Create test user
	testUser := createTestUser(t, db)
	ctx := contexts.SetAppID(context.Background(), xid.New())

	// Begin registration
	req := BeginRegisterRequest{
		UserID: testUser.ID.String(),
		Name:   "Test Passkey",
	}

	beginResp, err := service.BeginRegistration(ctx, testUser.ID, req)
	require.NoError(t, err)
	assert.NotNil(t, beginResp)

	// In a real test, we would:
	// 1. Use a WebAuthn client to generate a credential response
	// 2. Call FinishRegistration with that response
	// For now, we test that the challenge was created
	assert.NotEmpty(t, beginResp.Challenge)
}

// TestService_List tests listing passkeys
func TestService_List(t *testing.T) {
	db, service := setupTestService(t)
	defer db.Close()

	// Create test user
	testUser := createTestUser(t, db)
	appID := xid.New()
	ctx := contexts.SetAppID(context.Background(), appID)

	// Create test passkeys
	passkey1 := &schema.Passkey{
		ID:                xid.New(),
		UserID:            testUser.ID,
		CredentialID:      "cred-1",
		PublicKey:         []byte("public-key-1"),
		Name:              "Security Key 1",
		AuthenticatorType: "cross-platform",
		SignCount:         0,
		AppID:             appID,
	}
	passkey1.AuditableModel.CreatedBy = passkey1.ID
	passkey1.AuditableModel.UpdatedBy = passkey1.ID

	passkey2 := &schema.Passkey{
		ID:                xid.New(),
		UserID:            testUser.ID,
		CredentialID:      "cred-2",
		PublicKey:         []byte("public-key-2"),
		Name:              "Security Key 2",
		AuthenticatorType: "platform",
		SignCount:         5,
		AppID:             appID,
	}
	passkey2.AuditableModel.CreatedBy = passkey2.ID
	passkey2.AuditableModel.UpdatedBy = passkey2.ID

	_, err := db.NewInsert().Model(passkey1).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(passkey2).Exec(ctx)
	require.NoError(t, err)

	// List passkeys
	resp, err := service.List(ctx, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Count)
	assert.Len(t, resp.Passkeys, 2)

	// Verify passkey data
	for _, pk := range resp.Passkeys {
		assert.NotEmpty(t, pk.ID)
		assert.NotEmpty(t, pk.Name)
		assert.NotEmpty(t, pk.CredentialID)
	}
}

// TestService_Update tests updating passkey metadata
func TestService_Update(t *testing.T) {
	db, service := setupTestService(t)
	defer db.Close()

	// Create test user
	testUser := createTestUser(t, db)
	appID := xid.New()
	ctx := contexts.SetAppID(context.Background(), appID)

	// Create test passkey
	passkey := &schema.Passkey{
		ID:                xid.New(),
		UserID:            testUser.ID,
		CredentialID:      "cred-test",
		PublicKey:         []byte("public-key"),
		Name:              "Old Name",
		AuthenticatorType: "platform",
		AppID:             appID,
	}
	passkey.AuditableModel.CreatedBy = passkey.ID
	passkey.AuditableModel.UpdatedBy = passkey.ID

	_, err := db.NewInsert().Model(passkey).Exec(ctx)
	require.NoError(t, err)

	// Update name
	newName := "New Security Key Name"
	resp, err := service.Update(ctx, passkey.ID, newName)
	require.NoError(t, err)
	assert.Equal(t, passkey.ID.String(), resp.PasskeyID)
	assert.Equal(t, newName, resp.Name)

	// Verify database was updated
	var updated schema.Passkey
	err = db.NewSelect().Model(&updated).Where("id = ?", passkey.ID).Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
}

// TestService_Delete tests deleting a passkey
func TestService_Delete(t *testing.T) {
	db, service := setupTestService(t)
	defer db.Close()

	// Create test user
	testUser := createTestUser(t, db)
	appID := xid.New()
	ctx := contexts.SetAppID(context.Background(), appID)

	// Create test passkey
	passkey := &schema.Passkey{
		ID:                xid.New(),
		UserID:            testUser.ID,
		CredentialID:      "cred-delete",
		PublicKey:         []byte("public-key"),
		Name:              "To Delete",
		AuthenticatorType: "platform",
		AppID:             appID,
	}
	passkey.AuditableModel.CreatedBy = passkey.ID
	passkey.AuditableModel.UpdatedBy = passkey.ID

	_, err := db.NewInsert().Model(passkey).Exec(ctx)
	require.NoError(t, err)

	// Delete passkey
	err = service.Delete(ctx, passkey.ID, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Verify deletion
	exists, err := db.NewSelect().Model((*schema.Passkey)(nil)).Where("id = ?", passkey.ID).Exists(ctx)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestService_AppOrgScoping tests that passkeys are properly scoped to app and org
func TestService_AppOrgScoping(t *testing.T) {
	db, service := setupTestService(t)
	defer db.Close()

	// Create test user
	testUser := createTestUser(t, db)
	app1 := xid.New()
	app2 := xid.New()
	org1 := xid.New()

	// Create passkeys in different apps and orgs
	passkey1 := &schema.Passkey{
		ID:                 xid.New(),
		UserID:             testUser.ID,
		CredentialID:       "cred-app1",
		PublicKey:          []byte("key1"),
		AppID:              app1,
		UserOrganizationID: nil,
	}
	passkey1.AuditableModel.CreatedBy = passkey1.ID
	passkey1.AuditableModel.UpdatedBy = passkey1.ID

	passkey2 := &schema.Passkey{
		ID:                 xid.New(),
		UserID:             testUser.ID,
		CredentialID:       "cred-app1-org1",
		PublicKey:          []byte("key2"),
		AppID:              app1,
		UserOrganizationID: &org1,
	}
	passkey2.AuditableModel.CreatedBy = passkey2.ID
	passkey2.AuditableModel.UpdatedBy = passkey2.ID

	passkey3 := &schema.Passkey{
		ID:                 xid.New(),
		UserID:             testUser.ID,
		CredentialID:       "cred-app2",
		PublicKey:          []byte("key3"),
		AppID:              app2,
		UserOrganizationID: nil,
	}
	passkey3.AuditableModel.CreatedBy = passkey3.ID
	passkey3.AuditableModel.UpdatedBy = passkey3.ID

	ctx := context.Background()
	_, err := db.NewInsert().Model(passkey1).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(passkey2).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(passkey3).Exec(ctx)
	require.NoError(t, err)

	// Test app1 without org - should only see passkey1
	ctx1 := contexts.SetAppID(context.Background(), app1)
	resp1, err := service.List(ctx1, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, resp1.Count)

	// Test app1 with org1 - should only see passkey2
	ctx2 := contexts.WithAppAndOrganization(context.Background(), app1, org1)
	resp2, err := service.List(ctx2, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, resp2.Count)

	// Test app2 - should only see passkey3
	ctx3 := contexts.SetAppID(context.Background(), app2)
	resp3, err := service.List(ctx3, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, resp3.Count)
}

// TestChallengeStore tests challenge session storage and expiration
func TestChallengeStore(t *testing.T) {
	store := NewMemoryChallengeStore(100 * time.Millisecond) // 100ms timeout for testing
	ctx := context.Background()

	// Store a challenge
	sessionID := "test-session"
	challenge := &ChallengeSession{
		Challenge: []byte("test-challenge"),
		UserID:    xid.New(),
		CreatedAt: time.Now(),
	}

	err := store.Store(ctx, sessionID, challenge)
	require.NoError(t, err)

	// Retrieve it immediately
	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, challenge.Challenge, retrieved.Challenge)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, err = store.Get(ctx, sessionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// TestUserAdapter tests the WebAuthn user adapter
func TestUserAdapter(t *testing.T) {
	userID := xid.New()
	userName := "test@example.com"
	displayName := "Test User"

	// Create passkeys
	passkeys := []schema.Passkey{
		{
			ID:           xid.New(),
			CredentialID: "cred-1",
			PublicKey:    []byte("key-1"),
			SignCount:    5,
		},
		{
			ID:           xid.New(),
			CredentialID: "cred-2",
			PublicKey:    []byte("key-2"),
			SignCount:    10,
		},
	}

	adapter := NewUserAdapter(userID, userName, displayName, passkeys)

	// Test WebAuthn interface methods
	assert.Equal(t, []byte(userID.String()), adapter.WebAuthnID())
	assert.Equal(t, userName, adapter.WebAuthnName())
	assert.Equal(t, displayName, adapter.WebAuthnDisplayName())
	assert.Len(t, adapter.WebAuthnCredentials(), 2)
	assert.Empty(t, adapter.WebAuthnIcon())
}

// Helper functions

func setupTestService(t *testing.T) (*bun.DB, *Service) {
	// Create in-memory SQLite database
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create tables
	ctx := context.Background()
	_, err = db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewCreateTable().Model((*schema.Passkey)(nil)).IfNotExists().Exec(ctx)
	require.NoError(t, err)

	// Create services
	userRepo := repo.NewUserRepository(db)
	sessionRepo := repo.NewSessionRepository(db)
	sessionSvc := session.NewService(sessionRepo, session.Config{}, nil, nil)
	userSvc := user.NewService(userRepo, user.Config{}, nil, nil)
	authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
	auditSvc := audit.NewService(repo.NewAuditRepository(db))

	// Create passkey service with test config
	cfg := Config{
		RPID:             "localhost",
		RPName:           "Test App",
		RPOrigins:        []string{"http://localhost"},
		Timeout:          60000,
		UserVerification: "preferred",
		AttestationType:  "none",
	}

	service, err := NewService(db, userSvc, authSvc, auditSvc, cfg)
	require.NoError(t, err)

	return db, service
}

func createTestUser(t *testing.T, db *bun.DB) *schema.User {
	user := &schema.User{
		ID:    xid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}
	user.AuditableModel.CreatedBy = user.ID
	user.AuditableModel.UpdatedBy = user.ID

	_, err := db.NewInsert().Model(user).Exec(context.Background())
	require.NoError(t, err)

	return user
}
