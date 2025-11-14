package impersonation

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/schema"
)

// mockImpersonationRepository is a mock implementation of the impersonation repository
type mockImpersonationRepository struct {
	sessions    map[string]*schema.ImpersonationSession
	auditEvents map[string]*schema.ImpersonationAuditEvent
}

func newMockImpersonationRepository() *mockImpersonationRepository {
	return &mockImpersonationRepository{
		sessions:    make(map[string]*schema.ImpersonationSession),
		auditEvents: make(map[string]*schema.ImpersonationAuditEvent),
	}
}

func (r *mockImpersonationRepository) Create(ctx context.Context, session *schema.ImpersonationSession) error {
	r.sessions[session.ID.String()] = session
	return nil
}

func (r *mockImpersonationRepository) Get(ctx context.Context, id xid.ID, orgID xid.ID) (*schema.ImpersonationSession, error) {
	session, ok := r.sessions[id.String()]
	if !ok || session.AppID != orgID {
		return nil, impersonation.ErrImpersonationNotFound
	}
	return session, nil
}

func (r *mockImpersonationRepository) GetBySessionID(ctx context.Context, sessionID xid.ID) (*schema.ImpersonationSession, error) {
	for _, session := range r.sessions {
		if session.NewSessionID != nil && *session.NewSessionID == sessionID && session.Active {
			return session, nil
		}
	}
	return nil, impersonation.ErrImpersonationNotFound
}

func (r *mockImpersonationRepository) Update(ctx context.Context, session *schema.ImpersonationSession) error {
	r.sessions[session.ID.String()] = session
	return nil
}

func (r *mockImpersonationRepository) List(ctx context.Context, req *impersonation.ListRequest) ([]*schema.ImpersonationSession, error) {
	var sessions []*schema.ImpersonationSession
	for _, session := range r.sessions {
		if session.AppID != req.AppID {
			continue
		}
		if req.ActiveOnly && (!session.Active || session.IsExpired()) {
			continue
		}
		if req.ImpersonatorID != nil && session.ImpersonatorID != *req.ImpersonatorID {
			continue
		}
		if req.TargetUserID != nil && session.TargetUserID != *req.TargetUserID {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (r *mockImpersonationRepository) Count(ctx context.Context, req *impersonation.ListRequest) (int, error) {
	sessions, _ := r.List(ctx, req)
	return len(sessions), nil
}

func (r *mockImpersonationRepository) GetActive(ctx context.Context, impersonatorID xid.ID, orgID xid.ID) (*schema.ImpersonationSession, error) {
	for _, session := range r.sessions {
		if session.ImpersonatorID == impersonatorID &&
			session.AppID == orgID &&
			session.Active &&
			!session.IsExpired() {
			return session, nil
		}
	}
	return nil, impersonation.ErrImpersonationNotFound
}

func (r *mockImpersonationRepository) ExpireOldSessions(ctx context.Context) (int, error) {
	count := 0
	now := time.Now().UTC()
	for _, session := range r.sessions {
		if session.Active && now.After(session.ExpiresAt) {
			session.Active = false
			session.EndedAt = &now
			session.EndReason = "timeout"
			count++
		}
	}
	return count, nil
}

func (r *mockImpersonationRepository) CreateAuditEvent(ctx context.Context, event *schema.ImpersonationAuditEvent) error {
	r.auditEvents[event.ID.String()] = event
	return nil
}

func (r *mockImpersonationRepository) ListAuditEvents(ctx context.Context, req *impersonation.AuditListRequest) ([]*schema.ImpersonationAuditEvent, error) {
	var events []*schema.ImpersonationAuditEvent
	for _, event := range r.auditEvents {
		if event.AppID != req.AppID {
			continue
		}
		if req.ImpersonationID != nil && event.ImpersonationID != *req.ImpersonationID {
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *mockImpersonationRepository) CountAuditEvents(ctx context.Context, req *impersonation.AuditListRequest) (int, error) {
	events, _ := r.ListAuditEvents(ctx, req)
	return len(events), nil
}

// mockUserService is a simple mock for testing
type mockUserService struct {
	users map[string]*schema.User
}

func newMockUserService() *mockUserService {
	return &mockUserService{
		users: make(map[string]*schema.User),
	}
}

func (s *mockUserService) FindByID(ctx context.Context, id xid.ID) (*schema.User, error) {
	user, ok := s.users[id.String()]
	if !ok {
		return nil, impersonation.ErrUserNotFound
	}
	return user, nil
}

func (s *mockUserService) Create(ctx context.Context, req interface{}) (*schema.User, error) {
	return nil, nil
}

func (s *mockUserService) FindByEmail(ctx context.Context, email string) (*schema.User, error) {
	return nil, nil
}

func (s *mockUserService) FindByUsername(ctx context.Context, username string) (*schema.User, error) {
	return nil, nil
}

func (s *mockUserService) Update(ctx context.Context, u *schema.User, req interface{}) (*schema.User, error) {
	return nil, nil
}

func (s *mockUserService) Delete(ctx context.Context, id xid.ID) error {
	return nil
}

func (s *mockUserService) List(ctx context.Context, opts interface{}) ([]*schema.User, int, error) {
	return nil, 0, nil
}

// mockSessionService is a simple mock for testing
type mockSessionService struct {
	sessions map[string]*schema.Session
}

func newMockSessionService() *mockSessionService {
	return &mockSessionService{
		sessions: make(map[string]*schema.Session),
	}
}

func (s *mockSessionService) Create(ctx context.Context, req interface{}) (*schema.Session, error) {
	session := &schema.Session{
		ID:        xid.New(),
		Token:     "token_" + xid.New().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	s.sessions[session.ID.String()] = session
	return session, nil
}

func (s *mockSessionService) FindByToken(ctx context.Context, token string) (*schema.Session, error) {
	for _, session := range s.sessions {
		if session.Token == token {
			return session, nil
		}
	}
	return nil, impersonation.ErrSessionNotFound
}

func (s *mockSessionService) Revoke(ctx context.Context, token string) error {
	for id, session := range s.sessions {
		if session.Token == token {
			delete(s.sessions, id)
			return nil
		}
	}
	return nil
}

// Test helpers
func setupTestService(t *testing.T) (*impersonation.Service, *mockImpersonationRepository, *mockUserService, *mockSessionService) {
	repo := newMockImpersonationRepository()
	userSvc := newMockUserService()
	sessionSvc := newMockSessionService()

	config := impersonation.DefaultConfig()
	config.RequirePermission = false // Disable RBAC for basic tests

	service := impersonation.NewService(
		repo,
		userSvc,
		sessionSvc,
		nil, // audit service (optional)
		nil, // rbac service (optional)
		config,
	)

	return service, repo, userSvc, sessionSvc
}

func createTestUsers(userSvc *mockUserService) (admin, target *schema.User) {
	admin = &schema.User{
		ID:    xid.New(),
		Email: "admin@example.com",
		Name:  "Admin User",
	}
	target = &schema.User{
		ID:    xid.New(),
		Email: "target@example.com",
		Name:  "Target User",
	}
	userSvc.users[admin.ID.String()] = admin
	userSvc.users[target.ID.String()] = target
	return admin, target
}

// Tests

func TestService_Start_Success(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	req := &impersonation.StartRequest{
		AppID:           orgID,
		ImpersonatorID:  admin.ID,
		TargetUserID:    target.ID,
		Reason:          "Testing impersonation feature for debugging customer issue",
		TicketNumber:    "TICKET-12345",
		DurationMinutes: 30,
		IPAddress:       "192.168.1.1",
		UserAgent:       "Test Agent",
	}

	resp, err := service.Start(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.ImpersonationID)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotEmpty(t, resp.SessionToken)
	assert.False(t, resp.ExpiresAt.IsZero())
	assert.Contains(t, resp.Message, target.Email)
}

func TestService_Start_CannotImpersonateSelf(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	admin, _ := createTestUsers(userSvc)
	orgID := xid.New()

	req := &impersonation.StartRequest{
		AppID:          orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   admin.ID, // Same as impersonator
		Reason:         "Testing self impersonation",
	}

	_, err := service.Start(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, impersonation.ErrCannotImpersonateSelf, err)
}

func TestService_Start_ReasonTooShort(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	req := &impersonation.StartRequest{
		AppID:          orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Reason:         "Short", // Less than 10 characters
	}

	_, err := service.Start(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, impersonation.ErrInvalidReason, err)
}

func TestService_Start_AlreadyImpersonating(t *testing.T) {
	service, repo, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create existing active impersonation
	existingSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		AppID:          orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		CreatedAt:      time.Now(),
	}
	repo.sessions[existingSession.ID.String()] = existingSession

	// Try to start another impersonation
	req := &impersonation.StartRequest{
		AppID:          orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Reason:         "Testing multiple impersonations",
	}

	_, err := service.Start(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, impersonation.ErrAlreadyImpersonating, err)
}

func TestService_Start_UserNotFound(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	orgID := xid.New()

	req := &impersonation.StartRequest{
		AppID:          orgID,
		ImpersonatorID: xid.New(), // Non-existent user
		TargetUserID:   xid.New(),
		Reason:         "Testing with non-existent user",
	}

	_, err := service.Start(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get impersonator")
}

func TestService_End_Success(t *testing.T) {
	service, repo, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create active impersonation
	sessionID := xid.New()
	sessionToken := "token_test"
	impSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		AppID:          orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		NewSessionID:   &sessionID,
		SessionToken:   sessionToken,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		Reason:         "Test impersonation",
		CreatedAt:      time.Now(),
	}
	repo.sessions[impSession.ID.String()] = impSession

	req := &impersonation.EndRequest{
		ImpersonationID: impSession.ID,
		AppID:           orgID,
		ImpersonatorID:  admin.ID,
		Reason:          "manual",
	}

	resp, err := service.End(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, impSession.ID, resp.ImpersonationID)
	assert.False(t, resp.EndedAt.IsZero())

	// Verify session is inactive
	session, _ := repo.Get(context.Background(), impSession.ID, orgID)
	assert.False(t, session.Active)
	assert.NotNil(t, session.EndedAt)
}

func TestService_End_NotFound(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	admin, _ := createTestUsers(userSvc)
	orgID := xid.New()

	req := &impersonation.EndRequest{
		ImpersonationID: xid.New(), // Non-existent
		AppID:           orgID,
		ImpersonatorID:  admin.ID,
	}

	_, err := service.End(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, impersonation.ErrImpersonationNotFound, err)
}

func TestService_End_WrongImpersonator(t *testing.T) {
	service, repo, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	otherUser := &schema.User{
		ID:    xid.New(),
		Email: "other@example.com",
		Name:  "Other User",
	}
	userSvc.users[otherUser.ID.String()] = otherUser
	orgID := xid.New()

	// Create active impersonation
	impSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		AppID:          orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		CreatedAt:      time.Now(),
	}
	repo.sessions[impSession.ID.String()] = impSession

	// Try to end with different user
	req := &impersonation.EndRequest{
		ImpersonationID: impSession.ID,
		AppID:           orgID,
		ImpersonatorID:  otherUser.ID, // Different from actual impersonator
	}

	_, err := service.End(context.Background(), req)

	require.Error(t, err)
	assert.Equal(t, impersonation.ErrPermissionDenied, err)
}

func TestService_List_Success(t *testing.T) {
	service, repo, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create multiple impersonation sessions
	for i := 0; i < 5; i++ {
		session := &schema.ImpersonationSession{
			ID:             xid.New(),
			AppID:          orgID,
			ImpersonatorID: admin.ID,
			TargetUserID:   target.ID,
			Active:         i < 3, // First 3 active, last 2 inactive
			ExpiresAt:      time.Now().Add(1 * time.Hour),
			Reason:         "Test session",
			CreatedAt:      time.Now(),
		}
		repo.sessions[session.ID.String()] = session
	}

	// List all sessions
	req := &impersonation.ListRequest{
		AppID: orgID,
		Limit: 10,
	}

	resp, err := service.List(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Sessions, 5)
	assert.Equal(t, 5, resp.Total)

	// List only active sessions
	req.ActiveOnly = true
	resp, err = service.List(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Sessions, 3)
	assert.Equal(t, 3, resp.Total)
}

func TestService_Verify_Active(t *testing.T) {
	service, repo, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	sessionID := xid.New()
	impSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		AppID:          orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		NewSessionID:   &sessionID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		CreatedAt:      time.Now(),
	}
	repo.sessions[impSession.ID.String()] = impSession

	req := &impersonation.VerifyRequest{
		SessionID: sessionID,
	}

	resp, err := service.Verify(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.IsImpersonating)
	assert.NotNil(t, resp.ImpersonationID)
	assert.Equal(t, impSession.ID, *resp.ImpersonationID)
	assert.Equal(t, admin.ID, *resp.ImpersonatorID)
	assert.Equal(t, target.ID, *resp.TargetUserID)
}

func TestService_Verify_NotImpersonating(t *testing.T) {
	service, _, _, _ := setupTestService(t)

	req := &impersonation.VerifyRequest{
		SessionID: xid.New(), // Non-existent session
	}

	resp, err := service.Verify(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.IsImpersonating)
	assert.Nil(t, resp.ImpersonationID)
}

func TestService_ExpireSessions(t *testing.T) {
	service, repo, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create expired sessions
	for i := 0; i < 3; i++ {
		session := &schema.ImpersonationSession{
			ID:             xid.New(),
			AppID:          orgID,
			ImpersonatorID: admin.ID,
			TargetUserID:   target.ID,
			Active:         true,
			ExpiresAt:      time.Now().Add(-1 * time.Hour), // Expired
			CreatedAt:      time.Now(),
		}
		repo.sessions[session.ID.String()] = session
	}

	// Create active non-expired session
	activeSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		AppID:          orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour), // Not expired
		CreatedAt:      time.Now(),
	}
	repo.sessions[activeSession.ID.String()] = activeSession

	count, err := service.ExpireSessions(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Verify active session is still active
	session, _ := repo.Get(context.Background(), activeSession.ID, orgID)
	assert.True(t, session.Active)
}

func TestService_CustomDuration(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	customDuration := 120 // 2 hours

	req := &impersonation.StartRequest{
		AppID:           orgID,
		ImpersonatorID:  admin.ID,
		TargetUserID:    target.ID,
		Reason:          "Testing custom duration impersonation",
		DurationMinutes: customDuration,
	}

	resp, err := service.Start(context.Background(), req)

	require.NoError(t, err)

	// Verify expiration is approximately correct (within 1 minute)
	expectedExpiry := time.Now().Add(time.Duration(customDuration) * time.Minute)
	assert.WithinDuration(t, expectedExpiry, resp.ExpiresAt, 1*time.Minute)
}

func TestService_InvalidDuration(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	tests := []struct {
		name     string
		duration int
		wantErr  error
	}{
		{
			name:     "duration too short",
			duration: 0,
			wantErr:  impersonation.ErrInvalidDuration,
		},
		{
			name:     "duration too long",
			duration: 1000, // Exceeds max of 480
			wantErr:  impersonation.ErrInvalidDuration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &impersonation.StartRequest{
				AppID:           orgID,
				ImpersonatorID:  admin.ID,
				TargetUserID:    target.ID,
				Reason:          "Testing invalid duration",
				DurationMinutes: tt.duration,
			}

			_, err := service.Start(context.Background(), req)

			require.Error(t, err)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
