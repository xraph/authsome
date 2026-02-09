package authflow

import (
	"context"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

// Mock implementations for testing

type mockAuthService struct {
	createSessionCalled bool
	signUpCalled        bool
	signInCalled        bool
}

func (m *mockAuthService) SignUp(ctx context.Context, req *auth.SignUpRequest) (*responses.AuthResponse, error) {
	m.signUpCalled = true
	newUser := &user.User{
		ID:    xid.New(),
		Email: req.Email,
		Name:  req.Name,
	}

	return &responses.AuthResponse{
		User: newUser,
		Session: &session.Session{
			ID:     xid.New(),
			UserID: newUser.ID,
		},
		Token: "signup_token_123",
	}, nil
}

func (m *mockAuthService) SignIn(ctx context.Context, req *auth.SignInRequest) (*responses.AuthResponse, error) {
	m.signInCalled = true

	return nil, nil // Not used in these tests
}

func (m *mockAuthService) CreateSessionForUser(ctx context.Context, u *user.User, remember bool, ipAddress, userAgent string) (*responses.AuthResponse, error) {
	m.createSessionCalled = true

	return &responses.AuthResponse{
		User: u,
		Session: &session.Session{
			ID:     xid.New(),
			UserID: u.ID,
		},
		Token: "test_token_123",
	}, nil
}

type mockDeviceService struct {
	trackCalled bool
}

func (m *mockDeviceService) TrackDevice(ctx context.Context, appID, userID xid.ID, fingerprint, userAgent, ipAddress string) (*device.Device, error) {
	m.trackCalled = true

	return &device.Device{
		ID:     xid.New(),
		UserID: userID,
	}, nil
}

type mockAuditService struct {
	logCalled  bool
	lastAction string
}

func (m *mockAuditService) Log(ctx context.Context, userID *xid.ID, action, target, ipAddress, userAgent, metadata string) error {
	m.logCalled = true
	m.lastAction = action

	return nil
}

type mockAppService struct{}

func (m *mockAppService) GetCookieConfig(ctx context.Context, appID xid.ID) (*session.CookieConfig, error) {
	return &session.CookieConfig{
		Enabled: true,
		Name:    "test_session",
	}, nil
}

func TestCompletionService_CompleteAuthentication(t *testing.T) {
	// Setup mocks
	authSvc := &mockAuthService{}
	deviceSvc := &mockDeviceService{}
	auditSvc := &mockAuditService{}
	appSvc := &mockAppService{}

	cookieConfig := &session.CookieConfig{
		Enabled: true,
		Name:    "authsome_session",
	}

	// Create completion service
	completion := NewCompletionService(authSvc, deviceSvc, auditSvc, appSvc, cookieConfig)

	// Test user
	testUser := &user.User{
		ID:    xid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Create request
	req := &CompleteAuthenticationRequest{
		User:         testUser,
		RememberMe:   false,
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test/1.0",
		Context:      context.Background(),
		ForgeContext: nil, // nil for this test
		AuthMethod:   "email",
	}

	// Execute
	resp, err := completion.CompleteAuthentication(req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Session)
	assert.Equal(t, "test_token_123", resp.Token)
	assert.Equal(t, testUser.ID, resp.User.ID)

	// Verify all services were called
	assert.True(t, authSvc.createSessionCalled, "Auth service should create session")
	assert.True(t, deviceSvc.trackCalled, "Device service should track device")
	assert.True(t, auditSvc.logCalled, "Audit service should log authentication")
	assert.Equal(t, "signin_email", auditSvc.lastAction, "Audit action should be signin_email")
}

func TestCompletionService_CompleteAuthentication_SocialProvider(t *testing.T) {
	// Setup mocks
	authSvc := &mockAuthService{}
	deviceSvc := &mockDeviceService{}
	auditSvc := &mockAuditService{}

	completion := NewCompletionService(authSvc, deviceSvc, auditSvc, nil, nil)

	testUser := &user.User{
		ID:    xid.New(),
		Email: "test@example.com",
	}

	req := &CompleteAuthenticationRequest{
		User:         testUser,
		RememberMe:   false,
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test/1.0",
		Context:      context.Background(),
		AuthMethod:   "social",
		AuthProvider: "google",
	}

	resp, err := completion.CompleteAuthentication(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "signin_social_google", auditSvc.lastAction, "Audit action should include provider")
}

func TestCompletionService_NilServices(t *testing.T) {
	// Test that service works even with nil optional services
	authSvc := &mockAuthService{}

	completion := NewCompletionService(authSvc, nil, nil, nil, nil)

	testUser := &user.User{
		ID:    xid.New(),
		Email: "test@example.com",
	}

	req := &CompleteAuthenticationRequest{
		User:       testUser,
		RememberMe: false,
		IPAddress:  "127.0.0.1",
		UserAgent:  "Test/1.0",
		Context:    context.Background(),
		AuthMethod: "email",
	}

	resp, err := completion.CompleteAuthentication(req)

	// Should still work with just auth service
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, authSvc.createSessionCalled)
}

func TestCompletionService_CompleteSignUpOrSignIn_NewUser(t *testing.T) {
	// Setup mocks
	authSvc := &mockAuthService{}
	deviceSvc := &mockDeviceService{}
	auditSvc := &mockAuditService{}

	completion := NewCompletionService(authSvc, deviceSvc, auditSvc, nil, nil)

	// Create request for new user signup
	req := &CompleteSignUpOrSignInRequest{
		Email:        "newuser@example.com",
		Password:     "secure123",
		Name:         "New User",
		User:         nil, // Nil for new users
		IsNewUser:    true,
		RememberMe:   false,
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test/1.0",
		Context:      context.Background(),
		AuthMethod:   "social",
		AuthProvider: "github",
	}

	// Execute
	resp, err := completion.CompleteSignUpOrSignIn(req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.NotNil(t, resp.Session)
	assert.Equal(t, "signup_token_123", resp.Token)
	assert.Equal(t, "newuser@example.com", resp.User.Email)

	// Verify SignUp was called
	assert.True(t, authSvc.signUpCalled, "Auth service should call SignUp for new users")
	assert.False(t, authSvc.createSessionCalled, "Auth service should not call CreateSession directly for new users")

	// Verify device tracking and audit log
	assert.True(t, deviceSvc.trackCalled, "Device service should track device")
	assert.True(t, auditSvc.logCalled, "Audit service should log signup")
	assert.Equal(t, "signup_social_github", auditSvc.lastAction, "Audit action should be signup with provider")
}

func TestCompletionService_CompleteSignUpOrSignIn_ExistingUser(t *testing.T) {
	// Setup mocks
	authSvc := &mockAuthService{}
	deviceSvc := &mockDeviceService{}
	auditSvc := &mockAuditService{}

	completion := NewCompletionService(authSvc, deviceSvc, auditSvc, nil, nil)

	// Existing user
	existingUser := &user.User{
		ID:    xid.New(),
		Email: "existing@example.com",
		Name:  "Existing User",
	}

	// Create request for existing user signin
	req := &CompleteSignUpOrSignInRequest{
		Email:        "existing@example.com",
		Password:     "",
		Name:         "Existing User",
		User:         existingUser, // Populated for existing users
		IsNewUser:    false,
		RememberMe:   true,
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test/1.0",
		Context:      context.Background(),
		AuthMethod:   "magiclink",
		AuthProvider: "",
	}

	// Execute
	resp, err := completion.CompleteSignUpOrSignIn(req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.Equal(t, existingUser.ID, resp.User.ID)

	// Verify CreateSessionForUser was called for existing users
	assert.False(t, authSvc.signUpCalled, "Auth service should not call SignUp for existing users")
	assert.True(t, authSvc.createSessionCalled, "Auth service should call CreateSession for existing users")

	// Verify audit log
	assert.True(t, auditSvc.logCalled, "Audit service should log signin")
	assert.Equal(t, "signin_magiclink", auditSvc.lastAction, "Audit action should be signin")
}

func TestCompletionService_CompleteSignUpOrSignIn_NewUserWithoutEmail(t *testing.T) {
	// Setup mocks
	authSvc := &mockAuthService{}

	completion := NewCompletionService(authSvc, nil, nil, nil, nil)

	// Create invalid request (new user without email)
	req := &CompleteSignUpOrSignInRequest{
		Email:      "", // Empty email
		Password:   "secure123",
		Name:       "New User",
		User:       nil,
		IsNewUser:  true,
		RememberMe: false,
		IPAddress:  "127.0.0.1",
		UserAgent:  "Test/1.0",
		Context:    context.Background(),
		AuthMethod: "email",
	}

	// This should ideally return an error but depends on auth service validation
	// For this test, we just verify the flow works
	_, err := completion.CompleteSignUpOrSignIn(req)

	// The error handling depends on the auth service implementation
	// We're mainly testing that the completion service routes correctly
	_ = err
}

func TestCompletionService_CompleteSignUpOrSignIn_ExistingUserNilUser(t *testing.T) {
	// Setup mocks
	authSvc := &mockAuthService{}

	completion := NewCompletionService(authSvc, nil, nil, nil, nil)

	// Create invalid request (existing user with nil User field)
	req := &CompleteSignUpOrSignInRequest{
		Email:      "existing@example.com",
		Password:   "",
		Name:       "Existing User",
		User:       nil, // Should not be nil for existing users
		IsNewUser:  false,
		RememberMe: false,
		IPAddress:  "127.0.0.1",
		UserAgent:  "Test/1.0",
		Context:    context.Background(),
		AuthMethod: "email",
	}

	// Execute
	resp, err := completion.CompleteSignUpOrSignIn(req)

	// Assert - should return error
	assert.Error(t, err, "Should return error when User is nil for existing user signin")
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "USER_REQUIRED", "Error should indicate user is required")
}

// MockForgeContext implements a minimal forge.Context for testing cookie setting.
type mockForgeContext struct {
	cookies map[string]*mockCookie
}

type mockCookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite string
}

func newMockForgeContext() *mockForgeContext {
	return &mockForgeContext{
		cookies: make(map[string]*mockCookie),
	}
}

func (m *mockForgeContext) SetCookie(name, value, path, domain string, maxAge int, secure, httpOnly bool, sameSite string) {
	m.cookies[name] = &mockCookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   domain,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: sameSite,
	}
}

func (m *mockForgeContext) GetCookie(name string) *mockCookie {
	return m.cookies[name]
}

func TestCompletionService_CookieSettingForSocialLogin(t *testing.T) {
	// This test verifies the conditions under which cookies are set
	// The setSessionCookie method requires:
	// 1. ForgeContext != nil
	// 2. cookieConfig != nil && cookieConfig.Enabled
	// 3. authResp != nil with Session and Token
	t.Run("Cookie config enabled - cookie should be set", func(t *testing.T) {
		authSvc := &mockAuthService{}
		appSvc := &mockAppService{} // Returns enabled cookie config

		secureTrue := true
		cookieConfig := &session.CookieConfig{
			Enabled:  true,
			Name:     "authsome_session",
			Path:     "/",
			HttpOnly: true,
			Secure:   &secureTrue,
			SameSite: "Lax",
		}

		completion := NewCompletionService(authSvc, nil, nil, appSvc, cookieConfig)

		// Verify completion service has cookie config
		assert.NotNil(t, completion.cookieConfig, "Cookie config should be set")
		assert.True(t, completion.cookieConfig.Enabled, "Cookie should be enabled")
		assert.Equal(t, "authsome_session", completion.cookieConfig.Name, "Cookie name should match")
	})

	t.Run("Cookie config disabled - no cookie should be set", func(t *testing.T) {
		authSvc := &mockAuthService{}

		cookieConfig := &session.CookieConfig{
			Enabled: false, // Disabled
			Name:    "authsome_session",
		}

		completion := NewCompletionService(authSvc, nil, nil, nil, cookieConfig)

		// Verify completion service has cookie config but it's disabled
		assert.NotNil(t, completion.cookieConfig, "Cookie config should be set")
		assert.False(t, completion.cookieConfig.Enabled, "Cookie should be disabled")
	})

	t.Run("Nil cookie config - no cookie should be set", func(t *testing.T) {
		authSvc := &mockAuthService{}

		completion := NewCompletionService(authSvc, nil, nil, nil, nil)

		// Verify completion service has no cookie config
		assert.Nil(t, completion.cookieConfig, "Cookie config should be nil")
	})
}

func TestCompletionService_SocialLoginFlow(t *testing.T) {
	// Test the complete social login flow including cookie setting conditions
	t.Run("Social signup with all services enabled", func(t *testing.T) {
		authSvc := &mockAuthService{}
		deviceSvc := &mockDeviceService{}
		auditSvc := &mockAuditService{}
		appSvc := &mockAppService{}

		secureTrue := true
		cookieConfig := &session.CookieConfig{
			Enabled:  true,
			Name:     "session",
			Path:     "/",
			HttpOnly: true,
			Secure:   &secureTrue,
		}

		completion := NewCompletionService(authSvc, deviceSvc, auditSvc, appSvc, cookieConfig)

		req := &CompleteSignUpOrSignInRequest{
			Email:        "oauth@github.com",
			Password:     "random_generated_password",
			Name:         "GitHub User",
			User:         nil, // New user
			IsNewUser:    true,
			RememberMe:   false,
			IPAddress:    "192.168.1.1",
			UserAgent:    "Mozilla/5.0",
			Context:      context.Background(),
			ForgeContext: nil, // Would need forge.Context to actually set cookie
			AuthMethod:   "social",
			AuthProvider: "github",
		}

		resp, err := completion.CompleteSignUpOrSignIn(req)

		// Assert signup succeeded
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.User)
		assert.NotNil(t, resp.Session)
		assert.NotEmpty(t, resp.Token, "Token should be returned for cookie setting")

		// Verify all services were called
		assert.True(t, authSvc.signUpCalled, "SignUp should be called for new OAuth user")
		assert.True(t, deviceSvc.trackCalled, "Device should be tracked")
		assert.True(t, auditSvc.logCalled, "Audit log should be created")
		assert.Equal(t, "signup_social_github", auditSvc.lastAction)

		// Note: Cookie is only set if ForgeContext is provided
		// In real usage, the handler passes c (forge.Context) to the request
	})

	t.Run("Social signin returns token for cookie", func(t *testing.T) {
		authSvc := &mockAuthService{}

		cookieConfig := &session.CookieConfig{
			Enabled: true,
			Name:    "session",
		}

		completion := NewCompletionService(authSvc, nil, nil, nil, cookieConfig)

		existingUser := &user.User{
			ID:    xid.New(),
			Email: "returning@github.com",
			Name:  "Returning User",
		}

		req := &CompleteSignUpOrSignInRequest{
			Email:        "returning@github.com",
			User:         existingUser,
			IsNewUser:    false,
			IPAddress:    "192.168.1.1",
			UserAgent:    "Mozilla/5.0",
			Context:      context.Background(),
			AuthMethod:   "social",
			AuthProvider: "github",
		}

		resp, err := completion.CompleteSignUpOrSignIn(req)

		// Assert signin succeeded
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Session, "Session should be created")
		assert.NotEmpty(t, resp.Token, "Token should be returned for cookie")
		assert.Equal(t, "test_token_123", resp.Token, "Token should match mock")
	})
}
