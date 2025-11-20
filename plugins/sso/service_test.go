package sso

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// MOCKS
// =============================================================================

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) FindByAppAndEmail(ctx context.Context, appID xid.ID, email string) (*user.User, error) {
	args := m.Called(ctx, appID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) FindByID(ctx context.Context, id xid.ID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) Create(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) Update(ctx context.Context, u *user.User, req *user.UpdateUserRequest) (*user.User, error) {
	args := m.Called(ctx, u, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) Delete(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) ListUsers(ctx context.Context, filter *user.ListUsersFilter) (*pagination.PageResponse[*user.User], error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.PageResponse[*user.User]), args.Error(1)
}

func (m *MockUserService) CountUsers(ctx context.Context, filter *user.CountUsersFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

type MockSessionService struct {
	mock.Mock
}

func (m *MockSessionService) Create(ctx context.Context, req *session.CreateSessionRequest) (*session.Session, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.Session), args.Error(1)
}

func (m *MockSessionService) FindByToken(ctx context.Context, token string) (*session.Session, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.Session), args.Error(1)
}

func (m *MockSessionService) FindByID(ctx context.Context, id xid.ID) (*session.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.Session), args.Error(1)
}

func (m *MockSessionService) Delete(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionService) DeleteByToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockSessionService) ListSessions(ctx context.Context, filter *session.ListSessionsFilter) (*session.ListSessionsResponse, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ListSessionsResponse), args.Error(1)
}

func (m *MockSessionService) DeleteAllForUser(ctx context.Context, userID xid.ID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionService) Revoke(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockSessionService) RevokeByID(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// =============================================================================
// TESTS
// =============================================================================

func TestProvisionUser_ExistingUser_NoUpdate(t *testing.T) {
	// Setup
	mockUserSvc := new(MockUserService)
	mockSessionSvc := new(MockSessionService)
	
	config := Config{
		AutoProvision:    true,
		UpdateAttributes: false,
	}
	
	svc := &Service{
		userSvc:    mockUserSvc,
		sessionSvc: mockSessionSvc,
		config:     config,
	}
	
	// Existing user
	existingUser := &user.User{
		ID:    xid.New(),
		Email: "test@example.com",
		Name:  "Original Name",
	}
	
	// Mock: user exists
	mockUserSvc.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)
	
	// Test
	ctx := context.Background()
	attributes := map[string][]string{
		"name": {"New Name"},
	}
	provider := &schema.SSOProvider{}
	
	usr, err := svc.ProvisionUser(ctx, "test@example.com", attributes, provider)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, usr)
	assert.Equal(t, existingUser.ID, usr.ID)
	assert.Equal(t, "Original Name", usr.Name) // Not updated
	mockUserSvc.AssertExpectations(t)
}

func TestProvisionUser_ExistingUser_WithUpdate(t *testing.T) {
	// Setup
	mockUserSvc := new(MockUserService)
	mockSessionSvc := new(MockSessionService)
	
	config := Config{
		AutoProvision:    true,
		UpdateAttributes: true,
		AttributeMapping: map[string]string{
			"name": "name",
		},
	}
	
	svc := &Service{
		userSvc:    mockUserSvc,
		sessionSvc: mockSessionSvc,
		config:     config,
	}
	
	// Existing user
	existingUser := &user.User{
		ID:    xid.New(),
		Email: "test@example.com",
		Name:  "Original Name",
	}
	
	// Mock: user exists
	mockUserSvc.On("FindByAppAndEmail", mock.Anything, mock.Anything, "test@example.com").Return(existingUser, nil)
	mockUserSvc.On("Update", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.Name == "New Name"
	}), mock.Anything).Return(existingUser, nil)
	
	// Test
	ctx := contexts.SetAppID(context.Background(), xid.New())
	attributes := map[string][]string{
		"name": {"New Name"},
	}
	provider := &schema.SSOProvider{}
	
	usr, err := svc.ProvisionUser(ctx, "test@example.com", attributes, provider)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, usr)
	assert.Equal(t, "New Name", usr.Name) // Updated
	mockUserSvc.AssertExpectations(t)
}

func TestProvisionUser_NewUser_JITEnabled(t *testing.T) {
	// Setup
	mockUserSvc := new(MockUserService)
	mockSessionSvc := new(MockSessionService)
	
	config := Config{
		AutoProvision: true,
	}
	
	svc := &Service{
		userSvc:    mockUserSvc,
		sessionSvc: mockSessionSvc,
		config:     config,
	}
	
	// Mock: user not found, will be created
	mockUserSvc.On("FindByAppAndEmail", mock.Anything, mock.Anything, "newuser@example.com").Return(nil, nil)
	mockUserSvc.On("Create", mock.Anything, mock.MatchedBy(func(req *user.CreateUserRequest) bool {
		return req.Email == "newuser@example.com"
	})).Return(&user.User{
		ID:    xid.New(),
		Email: "newuser@example.com",
		AppID: xid.New(),
	}, nil)
	
	// Test
	ctx := contexts.SetAppID(context.Background(), xid.New())
	ctx = contexts.SetEnvironmentID(ctx, xid.New())
	attributes := map[string][]string{
		"name": {"New User"},
	}
	provider := &schema.SSOProvider{}
	
	usr, err := svc.ProvisionUser(ctx, "newuser@example.com", attributes, provider)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, usr)
	assert.Equal(t, "newuser@example.com", usr.Email)
	mockUserSvc.AssertExpectations(t)
}

func TestProvisionUser_NewUser_JITDisabled(t *testing.T) {
	// Setup
	mockUserSvc := new(MockUserService)
	mockSessionSvc := new(MockSessionService)
	
	config := Config{
		AutoProvision: false, // JIT disabled
	}
	
	svc := &Service{
		userSvc:    mockUserSvc,
		sessionSvc: mockSessionSvc,
		config:     config,
	}
	
	// Mock: user not found
	mockUserSvc.On("FindByAppAndEmail", mock.Anything, mock.Anything, "newuser@example.com").Return(nil, nil)
	
	// Test
	ctx := contexts.SetAppID(context.Background(), xid.New())
	attributes := map[string][]string{}
	provider := &schema.SSOProvider{}
	
	usr, err := svc.ProvisionUser(ctx, "newuser@example.com", attributes, provider)
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, usr)
	mockUserSvc.AssertExpectations(t)
}

func TestCreateSSOSession(t *testing.T) {
	// Setup
	mockUserSvc := new(MockUserService)
	mockSessionSvc := new(MockSessionService)
	
	svc := &Service{
		userSvc:    mockUserSvc,
		sessionSvc: mockSessionSvc,
		config:     Config{},
	}
	
	userID := xid.New()
	sessionID := xid.New()
	
	// Mock: session creation
	mockSessionSvc.On("Create", mock.Anything, mock.MatchedBy(func(req *session.CreateSessionRequest) bool {
		return req.UserID == userID
	})).Return(&session.Session{
		ID:     sessionID,
		UserID: userID,
		Token:  "test-token-123",
	}, nil)
	
	// Test
	ctx := contexts.SetAppID(context.Background(), xid.New())
	provider := &schema.SSOProvider{
		ProviderID: "test-provider",
	}
	
	sess, token, err := svc.CreateSSOSession(ctx, userID, provider)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, sessionID, sess.ID)
	assert.NotEmpty(t, token)
	mockSessionSvc.AssertExpectations(t)
}

func TestBuildCreateUserRequest(t *testing.T) {
	svc := &Service{
		config: Config{},
	}
	
	appID := xid.New()
	
	ctx := contexts.SetAppID(context.Background(), appID)
	
	attributes := map[string][]string{
		"name":       {"John Doe"},
		"givenName":  {"John"},
		"familyName": {"Doe"},
	}
	
	provider := &schema.SSOProvider{}
	
	req := svc.buildCreateUserRequest(ctx, "john@example.com", attributes, provider)
	
	assert.NotNil(t, req)
	assert.Equal(t, "john@example.com", req.Email)
	assert.Equal(t, "John Doe", req.Name)
	assert.Equal(t, appID, req.AppID)
	assert.NotEmpty(t, req.Password) // Should have generated password
}

func TestApplyAttributeMapping(t *testing.T) {
	svc := &Service{
		config: Config{
			AttributeMapping: map[string]string{
				"name": "displayName",
			},
		},
	}
	
	usr := &user.User{
		Email: "test@example.com",
		Name:  "Original",
	}
	
	attributes := map[string][]string{
		"displayName": {"New Name"},
		"otherAttr":   {"value"},
	}
	
	provider := &schema.SSOProvider{}
	
	svc.applyAttributeMapping(usr, attributes, provider)
	
	assert.Equal(t, "New Name", usr.Name)
}

func TestInitiateOIDCLogin(t *testing.T) {
	svc := NewService(nil, Config{}, nil, nil)
	
	provider := &schema.SSOProvider{
		Type:            "oidc",
		OIDCIssuer:      "https://idp.example.com",
		OIDCClientID:    "client-123",
		OIDCRedirectURI: "https://app.example.com/callback",
	}
	
	ctx := context.Background()
	state := "random-state"
	nonce := "random-nonce"
	redirectURI := "https://app.example.com/callback"
	
	authURL, pkce, err := svc.InitiateOIDCLogin(ctx, provider, redirectURI, state, nonce)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "https://idp.example.com")
	assert.Contains(t, authURL, "client_id=client-123")
	assert.Contains(t, authURL, "state="+state)
	assert.Contains(t, authURL, "nonce="+nonce)
	assert.Contains(t, authURL, "code_challenge")
	assert.NotNil(t, pkce)
	assert.NotEmpty(t, pkce.CodeChallenge)
	assert.NotEmpty(t, pkce.CodeVerifier)
	assert.Equal(t, "S256", pkce.Method)
}

func TestStateStore(t *testing.T) {
	store := NewStateStore()
	
	state := &OIDCState{
		State:        "test-state",
		Nonce:        "test-nonce",
		CodeVerifier: "test-verifier",
		ProviderID:   "test-provider",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	
	ctx := context.Background()
	
	// Store state
	err := store.Store(ctx, state)
	assert.NoError(t, err)
	
	// Retrieve state
	retrieved, err := store.Get(ctx, "test-state")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-nonce", retrieved.Nonce)
	assert.Equal(t, "test-verifier", retrieved.CodeVerifier)
	
	// Delete state
	err = store.Delete(ctx, "test-state")
	assert.NoError(t, err)
	
	// Verify deletion
	retrieved, err = store.Get(ctx, "test-state")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestStateStore_Expiration(t *testing.T) {
	store := NewStateStore()
	
	state := &OIDCState{
		State:      "expired-state",
		Nonce:      "test-nonce",
		CreatedAt:  time.Now().Add(-20 * time.Minute),
		ExpiresAt:  time.Now().Add(-10 * time.Minute), // Already expired
	}
	
	ctx := context.Background()
	
	// Store expired state
	err := store.Store(ctx, state)
	assert.NoError(t, err)
	
	// Try to retrieve expired state
	retrieved, err := store.Get(ctx, "expired-state")
	assert.NoError(t, err)
	assert.Nil(t, retrieved) // Should not return expired state
}

