package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, session *Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockRepository) FindByToken(ctx context.Context, token string) (*Session, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockRepository) Revoke(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRepository) FindByUserID(ctx context.Context, userID xid.ID) ([]*Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Session), args.Error(1)
}

func (m *MockRepository) RevokeAllForUser(ctx context.Context, userID xid.ID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRepository) FindByID(ctx context.Context, id xid.ID) (*Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockRepository) ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*Session, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Session), args.Error(1)
}

func (m *MockRepository) RevokeByID(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListAll(ctx context.Context, limit, offset int) ([]*Session, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Session), args.Error(1)
}

// Helper function to create a test service
func newTestService(repo Repository, cfg ...Config) *Service {
	config := Config{}
	if len(cfg) > 0 {
		config = cfg[0]
	}
	return NewService(repo, config, nil)
}

func TestService_Create(t *testing.T) {
	userID := xid.New()

	tests := []struct {
		name    string
		req     *CreateSessionRequest
		config  Config
		setup   func(*MockRepository)
		wantErr bool
		check   func(*testing.T, *Session)
	}{
		{
			name: "successful session creation with default TTL",
			req: &CreateSessionRequest{
				UserID:    userID,
				Remember:  false,
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			config: Config{}, // Uses defaults
			setup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*session.Session")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, s *Session) {
				assert.NotEmpty(t, s.ID)
				assert.NotEmpty(t, s.Token)
				assert.Equal(t, userID, s.UserID)
				assert.Equal(t, "192.168.1.1", s.IPAddress)
				assert.Equal(t, "Mozilla/5.0", s.UserAgent)
				assert.True(t, time.Now().Add(23*time.Hour).Before(s.ExpiresAt))
				assert.True(t, time.Now().Add(25*time.Hour).After(s.ExpiresAt))
			},
		},
		{
			name: "successful session creation with remember me",
			req: &CreateSessionRequest{
				UserID:    userID,
				Remember:  true,
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			config: Config{}, // Uses defaults
			setup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*session.Session")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, s *Session) {
				assert.NotEmpty(t, s.ID)
				assert.NotEmpty(t, s.Token)
				assert.Equal(t, userID, s.UserID)
				// Should use RememberTTL (7 days default)
				assert.True(t, time.Now().Add(6*24*time.Hour).Before(s.ExpiresAt))
				assert.True(t, time.Now().Add(8*24*time.Hour).After(s.ExpiresAt))
			},
		},
		{
			name: "successful session creation with custom TTL",
			req: &CreateSessionRequest{
				UserID:    userID,
				Remember:  false,
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			config: Config{
				DefaultTTL:  2 * time.Hour,
				RememberTTL: 48 * time.Hour,
			},
			setup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*session.Session")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, s *Session) {
				// Should use custom DefaultTTL (2 hours)
				assert.True(t, time.Now().Add(1*time.Hour+50*time.Minute).Before(s.ExpiresAt))
				assert.True(t, time.Now().Add(2*time.Hour+10*time.Minute).After(s.ExpiresAt))
			},
		},
		{
			name: "successful session creation with custom remember TTL",
			req: &CreateSessionRequest{
				UserID:    userID,
				Remember:  true,
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			config: Config{
				DefaultTTL:  2 * time.Hour,
				RememberTTL: 48 * time.Hour,
			},
			setup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*session.Session")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, s *Session) {
				// Should use custom RememberTTL (48 hours)
				assert.True(t, time.Now().Add(47*time.Hour).Before(s.ExpiresAt))
				assert.True(t, time.Now().Add(49*time.Hour).After(s.ExpiresAt))
			},
		},
		{
			name: "repository create error",
			req: &CreateSessionRequest{
				UserID:    userID,
				Remember:  false,
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
			config: Config{},
			setup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*session.Session")).Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo, tt.config)

			session, err := svc.Create(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
				if tt.check != nil {
					tt.check(t, session)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_FindByToken(t *testing.T) {
	validToken := "valid-token-12345"
	expectedSession := &Session{
		ID:        xid.New(),
		Token:     validToken,
		UserID:    xid.New(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name    string
		token   string
		setup   func(*MockRepository)
		want    *Session
		wantErr bool
	}{
		{
			name:  "session found",
			token: validToken,
			setup: func(m *MockRepository) {
				m.On("FindByToken", mock.Anything, validToken).Return(expectedSession, nil)
			},
			want:    expectedSession,
			wantErr: false,
		},
		{
			name:  "session not found",
			token: "invalid-token",
			setup: func(m *MockRepository) {
				m.On("FindByToken", mock.Anything, "invalid-token").Return(nil, errors.New("not found"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:  "empty token",
			token: "",
			setup: func(m *MockRepository) {
				m.On("FindByToken", mock.Anything, "").Return(nil, errors.New("not found"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo)

			session, err := svc.FindByToken(context.Background(), tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, session)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_Revoke(t *testing.T) {
	validToken := "valid-token-12345"
	session := &Session{
		ID:        xid.New(),
		Token:     validToken,
		UserID:    xid.New(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name    string
		token   string
		setup   func(*MockRepository)
		wantErr bool
	}{
		{
			name:  "successful revocation",
			token: validToken,
			setup: func(m *MockRepository) {
				m.On("FindByToken", mock.Anything, validToken).Return(session, nil)
				m.On("Revoke", mock.Anything, validToken).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "session not found before revocation",
			token: "invalid-token",
			setup: func(m *MockRepository) {
				m.On("FindByToken", mock.Anything, "invalid-token").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name:  "repository revoke error",
			token: validToken,
			setup: func(m *MockRepository) {
				m.On("FindByToken", mock.Anything, validToken).Return(session, nil)
				m.On("Revoke", mock.Anything, validToken).Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo)

			err := svc.Revoke(context.Background(), tt.token)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_ConfigDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		wantDefaultTTL time.Duration
		wantRememberTTL time.Duration
	}{
		{
			name:            "zero values use defaults",
			config:          Config{},
			wantDefaultTTL:  24 * time.Hour,
			wantRememberTTL: 7 * 24 * time.Hour,
		},
		{
			name: "custom values are preserved",
			config: Config{
				DefaultTTL:  2 * time.Hour,
				RememberTTL: 48 * time.Hour,
			},
			wantDefaultTTL:  2 * time.Hour,
			wantRememberTTL: 48 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := newTestService(mockRepo, tt.config)

			assert.Equal(t, tt.wantDefaultTTL, svc.config.DefaultTTL)
			assert.Equal(t, tt.wantRememberTTL, svc.config.RememberTTL)
		})
	}
}

func TestService_TokenGeneration(t *testing.T) {
	userID := xid.New()
	mockRepo := new(MockRepository)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*session.Session")).Return(nil)
	svc := newTestService(mockRepo)

	// Create multiple sessions and verify tokens are unique
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		session, err := svc.Create(context.Background(), &CreateSessionRequest{
			UserID:    userID,
			Remember:  false,
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, session.Token)
		
		// Check for uniqueness
		assert.False(t, tokens[session.Token], "Token should be unique but found duplicate: %s", session.Token)
		tokens[session.Token] = true
		
		// Token should be of reasonable length (base64 encoded 32 bytes = ~44 chars)
		assert.GreaterOrEqual(t, len(session.Token), 40)
	}
}

