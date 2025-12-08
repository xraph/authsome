package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
)

// MockUserService is a mock implementation of user.ServiceInterface
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Create(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	args := m.Called(ctx, req)
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

func (m *MockUserService) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
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

func (m *MockUserService) FindByAppAndEmail(ctx context.Context, appID xid.ID, email string) (*user.User, error) {
	args := m.Called(ctx, appID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
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

// MockSessionService is a mock implementation of session.ServiceInterface
type MockSessionService struct {
	mock.Mock
}

// RefreshSession implements session.ServiceInterface.
func (m *MockSessionService) RefreshSession(ctx context.Context, refreshToken string) (*session.RefreshResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.RefreshResponse), args.Error(1)
}

// TouchSession implements session.ServiceInterface.
func (m *MockSessionService) TouchSession(ctx context.Context, sess *session.Session) (*session.Session, bool, error) {
	args := m.Called(ctx, sess)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*session.Session), args.Bool(1), args.Error(2)
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

func (m *MockSessionService) ListSessions(ctx context.Context, filter *session.ListSessionsFilter) (*session.ListSessionsResponse, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ListSessionsResponse), args.Error(1)
}

func (m *MockSessionService) Revoke(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockSessionService) RevokeByID(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Helper function to create a test service
func newTestService(userSvc user.ServiceInterface, sessionSvc session.ServiceInterface, cfg ...Config) *Service {
	config := Config{}
	if len(cfg) > 0 {
		config = cfg[0]
	}
	return NewService(userSvc, sessionSvc, config, nil)
}

func TestService_SignUp(t *testing.T) {
	password := "SecurePass123!"
	passwordHash, _ := crypto.HashPassword(password)

	tests := []struct {
		name    string
		req     *SignUpRequest
		config  Config
		setup   func(*MockUserService, *MockSessionService)
		wantErr bool
		errMsg  string
		check   func(*testing.T, *responses.AuthResponse)
	}{
		{
			name: "successful signup without email verification",
			req: &SignUpRequest{
				Email:      "test@example.com",
				Password:   password,
				Name:       "Test User",
				RememberMe: false,
				IPAddress:  "192.168.1.1",
				UserAgent:  "Mozilla/5.0",
			},
			config: Config{RequireEmailVerification: false},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("not found"))
				createdUser := &user.User{
					ID:           xid.New(),
					Email:        "test@example.com",
					Name:         "Test User",
					PasswordHash: passwordHash,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				mu.On("Create", mock.Anything, mock.AnythingOfType("*user.CreateUserRequest")).Return(createdUser, nil)

				createdSession := &session.Session{
					ID:        xid.New(),
					Token:     "test-token-12345",
					UserID:    createdUser.ID,
					ExpiresAt: time.Now().Add(24 * time.Hour),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				ms.On("Create", mock.Anything, mock.AnythingOfType("*session.CreateSessionRequest")).Return(createdSession, nil)
			},
			wantErr: false,
			check: func(t *testing.T, resp *responses.AuthResponse) {
				assert.NotNil(t, resp.User)
				assert.NotNil(t, resp.Session)
				assert.NotEmpty(t, resp.Token)
				assert.Equal(t, "test@example.com", resp.User.Email)
				assert.Equal(t, "Test User", resp.User.Name)
			},
		},
		{
			name: "successful signup with email verification required",
			req: &SignUpRequest{
				Email:    "test@example.com",
				Password: password,
				Name:     "Test User",
			},
			config: Config{RequireEmailVerification: true},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("not found"))
				createdUser := &user.User{
					ID:           xid.New(),
					Email:        "test@example.com",
					Name:         "Test User",
					PasswordHash: passwordHash,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				mu.On("Create", mock.Anything, mock.AnythingOfType("*user.CreateUserRequest")).Return(createdUser, nil)
				// Session should NOT be created when verification is required
			},
			wantErr: false,
			check: func(t *testing.T, resp *responses.AuthResponse) {
				assert.NotNil(t, resp.User)
				assert.Nil(t, resp.Session) // No session when verification required
				assert.Empty(t, resp.Token)
			},
		},
		{
			name: "signup with existing email",
			req: &SignUpRequest{
				Email:    "existing@example.com",
				Password: password,
				Name:     "Test User",
			},
			config: Config{},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				existingUser := &user.User{
					ID:    xid.New(),
					Email: "existing@example.com",
				}
				mu.On("FindByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)
			},
			wantErr: true,
			errMsg:  "email already exists",
		},
		{
			name: "signup with user creation error",
			req: &SignUpRequest{
				Email:    "test@example.com",
				Password: password,
				Name:     "Test User",
			},
			config: Config{},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("not found"))
				mu.On("Create", mock.Anything, mock.AnythingOfType("*user.CreateUserRequest")).Return(nil, errors.New("database error"))
			},
			wantErr: true,
			errMsg:  "database error",
		},
		{
			name: "signup with session creation error",
			req: &SignUpRequest{
				Email:     "test@example.com",
				Password:  password,
				Name:      "Test User",
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			config: Config{RequireEmailVerification: false},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("not found"))
				createdUser := &user.User{
					ID:           xid.New(),
					Email:        "test@example.com",
					Name:         "Test User",
					PasswordHash: passwordHash,
				}
				mu.On("Create", mock.Anything, mock.AnythingOfType("*user.CreateUserRequest")).Return(createdUser, nil)
				ms.On("Create", mock.Anything, mock.AnythingOfType("*session.CreateSessionRequest")).Return(nil, errors.New("session error"))
			},
			wantErr: true,
			errMsg:  "session error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := new(MockUserService)
			mockSessionSvc := new(MockSessionService)
			tt.setup(mockUserSvc, mockSessionSvc)
			svc := newTestService(mockUserSvc, mockSessionSvc, tt.config)

			// Create context with AppID
			ctx := context.Background()
			ctx = contexts.SetAppID(ctx, xid.New())

			resp, err := svc.SignUp(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.check != nil {
					tt.check(t, resp)
				}
			}

			mockUserSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestService_SignIn(t *testing.T) {
	password := "SecurePass123!"
	passwordHash, _ := crypto.HashPassword(password)
	userID := xid.New()

	tests := []struct {
		name    string
		req     *SignInRequest
		setup   func(*MockUserService, *MockSessionService)
		wantErr bool
		errMsg  string
		check   func(*testing.T, *responses.AuthResponse)
	}{
		{
			name: "successful signin",
			req: &SignInRequest{
				Email:      "test@example.com",
				Password:   password,
				RememberMe: false,
				IPAddress:  "192.168.1.1",
				UserAgent:  "Mozilla/5.0",
			},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				existingUser := &user.User{
					ID:           userID,
					Email:        "test@example.com",
					Name:         "Test User",
					PasswordHash: passwordHash,
				}
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)

				createdSession := &session.Session{
					ID:        xid.New(),
					Token:     "test-token-12345",
					UserID:    userID,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				ms.On("Create", mock.Anything, mock.AnythingOfType("*session.CreateSessionRequest")).Return(createdSession, nil)
			},
			wantErr: false,
			check: func(t *testing.T, resp *responses.AuthResponse) {
				assert.NotNil(t, resp.User)
				assert.NotNil(t, resp.Session)
				assert.NotEmpty(t, resp.Token)
				assert.Equal(t, "test@example.com", resp.User.Email)
			},
		},
		{
			name: "signin with remember me",
			req: &SignInRequest{
				Email:      "test@example.com",
				Password:   password,
				RememberMe: true,
				IPAddress:  "192.168.1.1",
				UserAgent:  "Mozilla/5.0",
			},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				existingUser := &user.User{
					ID:           userID,
					Email:        "test@example.com",
					PasswordHash: passwordHash,
				}
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)

				createdSession := &session.Session{
					ID:        xid.New(),
					Token:     "test-token-12345",
					UserID:    userID,
					ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // Long TTL
				}
				ms.On("Create", mock.Anything, mock.AnythingOfType("*session.CreateSessionRequest")).Return(createdSession, nil)
			},
			wantErr: false,
			check: func(t *testing.T, resp *responses.AuthResponse) {
				assert.NotNil(t, resp.User)
				assert.NotNil(t, resp.Session)
				// Verify session has extended TTL
				assert.True(t, resp.Session.ExpiresAt.After(time.Now().Add(6*24*time.Hour)))
			},
		},
		{
			name: "signin with nonexistent email",
			req: &SignInRequest{
				Email:    "nonexistent@example.com",
				Password: password,
			},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				mu.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "invalid credentials",
		},
		{
			name: "signin with wrong password",
			req: &SignInRequest{
				Email:    "test@example.com",
				Password: "WrongPassword123!",
			},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				existingUser := &user.User{
					ID:           userID,
					Email:        "test@example.com",
					PasswordHash: passwordHash,
				}
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)
			},
			wantErr: true,
			errMsg:  "invalid credentials",
		},
		{
			name: "signin with session creation error",
			req: &SignInRequest{
				Email:     "test@example.com",
				Password:  password,
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			setup: func(mu *MockUserService, ms *MockSessionService) {
				existingUser := &user.User{
					ID:           userID,
					Email:        "test@example.com",
					PasswordHash: passwordHash,
				}
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)
				ms.On("Create", mock.Anything, mock.AnythingOfType("*session.CreateSessionRequest")).Return(nil, errors.New("session error"))
			},
			wantErr: true,
			errMsg:  "session error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := new(MockUserService)
			mockSessionSvc := new(MockSessionService)
			tt.setup(mockUserSvc, mockSessionSvc)
			svc := newTestService(mockUserSvc, mockSessionSvc)

			// Create context with AppID
			ctx := context.Background()
			ctx = contexts.SetAppID(ctx, xid.New())

			resp, err := svc.SignIn(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.check != nil {
					tt.check(t, resp)
				}
			}

			mockUserSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestService_CheckCredentials(t *testing.T) {
	password := "SecurePass123!"
	passwordHash, _ := crypto.HashPassword(password)
	userID := xid.New()

	tests := []struct {
		name     string
		email    string
		password string
		setup    func(*MockUserService)
		wantErr  bool
	}{
		{
			name:     "valid credentials",
			email:    "test@example.com",
			password: password,
			setup: func(mu *MockUserService) {
				existingUser := &user.User{
					ID:           userID,
					Email:        "test@example.com",
					PasswordHash: passwordHash,
				}
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)
			},
			wantErr: false,
		},
		{
			name:     "invalid email",
			email:    "nonexistent@example.com",
			password: password,
			setup: func(mu *MockUserService) {
				mu.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name:     "invalid password",
			email:    "test@example.com",
			password: "WrongPassword123!",
			setup: func(mu *MockUserService) {
				existingUser := &user.User{
					ID:           userID,
					Email:        "test@example.com",
					PasswordHash: passwordHash,
				}
				mu.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := new(MockUserService)
			mockSessionSvc := new(MockSessionService)
			tt.setup(mockUserSvc)
			svc := newTestService(mockUserSvc, mockSessionSvc)

			user, err := svc.CheckCredentials(context.Background(), tt.email, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
			}

			mockUserSvc.AssertExpectations(t)
		})
	}
}

func TestService_SignOut(t *testing.T) {
	validToken := "valid-token-12345"

	tests := []struct {
		name    string
		req     *SignOutRequest
		setup   func(*MockSessionService)
		wantErr bool
	}{
		{
			name: "successful signout",
			req:  &SignOutRequest{Token: validToken},
			setup: func(ms *MockSessionService) {
				ms.On("Revoke", mock.Anything, validToken).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "signout with invalid token",
			req:  &SignOutRequest{Token: "invalid-token"},
			setup: func(ms *MockSessionService) {
				ms.On("Revoke", mock.Anything, "invalid-token").Return(errors.New("session not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := new(MockUserService)
			mockSessionSvc := new(MockSessionService)
			tt.setup(mockSessionSvc)
			svc := newTestService(mockUserSvc, mockSessionSvc)

			err := svc.SignOut(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestService_GetSession(t *testing.T) {
	validToken := "valid-token-12345"
	userID := xid.New()

	tests := []struct {
		name    string
		token   string
		setup   func(*MockUserService, *MockSessionService)
		wantErr bool
		errMsg  string
		check   func(*testing.T, *responses.AuthResponse)
	}{
		{
			name:  "valid session",
			token: validToken,
			setup: func(mu *MockUserService, ms *MockSessionService) {
				sess := &session.Session{
					ID:        xid.New(),
					Token:     validToken,
					UserID:    userID,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				ms.On("FindByToken", mock.Anything, validToken).Return(sess, nil)

				existingUser := &user.User{
					ID:    userID,
					Email: "test@example.com",
					Name:  "Test User",
				}
				mu.On("FindByID", mock.Anything, userID).Return(existingUser, nil)
			},
			wantErr: false,
			check: func(t *testing.T, resp *responses.AuthResponse) {
				assert.NotNil(t, resp.User)
				assert.NotNil(t, resp.Session)
				assert.Equal(t, validToken, resp.Token)
			},
		},
		{
			name:  "session not found",
			token: "invalid-token",
			setup: func(mu *MockUserService, ms *MockSessionService) {
				ms.On("FindByToken", mock.Anything, "invalid-token").Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "session not found",
		},
		{
			name:  "expired session",
			token: validToken,
			setup: func(mu *MockUserService, ms *MockSessionService) {
				sess := &session.Session{
					ID:        xid.New(),
					Token:     validToken,
					UserID:    userID,
					ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
				}
				ms.On("FindByToken", mock.Anything, validToken).Return(sess, nil)
			},
			wantErr: true,
			errMsg:  "session expired",
		},
		{
			name:  "user not found",
			token: validToken,
			setup: func(mu *MockUserService, ms *MockSessionService) {
				sess := &session.Session{
					ID:        xid.New(),
					Token:     validToken,
					UserID:    userID,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				ms.On("FindByToken", mock.Anything, validToken).Return(sess, nil)
				mu.On("FindByID", mock.Anything, userID).Return(nil, errors.New("user not found"))
			},
			wantErr: true,
			errMsg:  "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := new(MockUserService)
			mockSessionSvc := new(MockSessionService)
			tt.setup(mockUserSvc, mockSessionSvc)
			svc := newTestService(mockUserSvc, mockSessionSvc)

			resp, err := svc.GetSession(context.Background(), tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.check != nil {
					tt.check(t, resp)
				}
			}

			mockUserSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}
