package multisession

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/schema"
)

// setup TestDB initializes an in-memory database with necessary tables
func setupTestDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	// Create necessary tables
	models := []interface{}{
		(*schema.User)(nil),
		(*schema.Session)(nil),
		(*schema.Device)(nil),
	}
	for _, model := range models {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	return db
}

// mockSessionService implements session.ServiceInterface for testing
type mockSessionService struct {
	db *bun.DB
}

func (m *mockSessionService) Create(ctx context.Context, req *session.CreateSessionRequest) (*session.Session, error) {
	panic("not implemented")
}

func (m *mockSessionService) FindByToken(ctx context.Context, token string) (*session.Session, error) {
	var s schema.Session
	err := m.db.NewSelect().
		Model(&s).
		Where("token = ?", token).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return session.FromSchemaSession(&s), nil
}

func (m *mockSessionService) FindByID(ctx context.Context, id xid.ID) (*session.Session, error) {
	var s schema.Session
	err := m.db.NewSelect().
		Model(&s).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return session.FromSchemaSession(&s), nil
}

func (m *mockSessionService) ListSessions(ctx context.Context, filter *session.ListSessionsFilter) (*session.ListSessionsResponse, error) {
	var sessions []*schema.Session
	query := m.db.NewSelect().Model(&sessions)

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &session.ListSessionsResponse{
		Data: session.FromSchemaSessions(sessions),
	}, nil
}

func (m *mockSessionService) Revoke(ctx context.Context, token string) error {
	_, err := m.db.NewDelete().
		Model((*schema.Session)(nil)).
		Where("token = ?", token).
		Exec(ctx)
	return err
}

func (m *mockSessionService) RevokeByID(ctx context.Context, id xid.ID) error {
	_, err := m.db.NewDelete().
		Model((*schema.Session)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (m *mockSessionService) RefreshSession(ctx context.Context, refreshToken string) (*session.RefreshResponse, error) {
	return nil, nil
}

func (m *mockSessionService) TouchSession(ctx context.Context, sess *session.Session) (*session.Session, bool, error) {
	return sess, false, nil
}

// TestService_ListSessions tests the List method
func TestService_ListSessions(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create user
	systemID := xid.New()
	appID := xid.New()
	user := &schema.User{
		ID:           xid.New(),
		AppID:        &appID,
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "$2a$10$example.hash",
	}
	user.CreatedBy = systemID
	user.UpdatedBy = systemID
	if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create two sessions for the user
	sess1 := &schema.Session{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    user.ID,
		Token:     "token-1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "ua-1",
	}
	sess1.CreatedBy = systemID
	sess1.UpdatedBy = systemID

	sess2 := &schema.Session{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    user.ID,
		Token:     "token-2",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "ua-2",
	}
	sess2.CreatedBy = systemID
	sess2.UpdatedBy = systemID

	if _, err := db.NewInsert().Model(sess1).Exec(ctx); err != nil {
		t.Fatalf("create session1: %v", err)
	}
	if _, err := db.NewInsert().Model(sess2).Exec(ctx); err != nil {
		t.Fatalf("create session2: %v", err)
	}

	// Setup service - use minimal dependencies by only initializing sessionSvc
	sessionSvc := &mockSessionService{db: db}

	// Create service with nil for dependencies we won't use in this test
	svc := &Service{
		sessionSvc: sessionSvc,
		// auth, sessions, devices not needed for List test
	}

	// Test List with default params
	req := &ListSessionsRequest{
		Limit:  100,
		Offset: 0,
	}
	result, err := svc.List(ctx, user.ID, req)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(result.Data))
	}
}

// TestService_FindSession tests the Find method
func TestService_FindSession(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create user
	systemID := xid.New()
	appID := xid.New()
	user := &schema.User{
		ID:           xid.New(),
		AppID:        &appID,
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "$2a$10$example.hash",
	}
	user.CreatedBy = systemID
	user.UpdatedBy = systemID
	if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create session
	sess := &schema.Session{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    user.ID,
		Token:     "token-1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "ua-1",
	}
	sess.CreatedBy = systemID
	sess.UpdatedBy = systemID
	if _, err := db.NewInsert().Model(sess).Exec(ctx); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Setup service
	sessionSvc := &mockSessionService{db: db}
	svc := &Service{
		sessionSvc: sessionSvc,
	}

	// Test Find - correct user
	t.Run("Find with correct user", func(t *testing.T) {
		found, err := svc.Find(ctx, user.ID, sess.ID)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}
		if found.ID != sess.ID {
			t.Errorf("expected session %s, got %s", sess.ID, found.ID)
		}
	})

	// Test Find - wrong user
	t.Run("Find with wrong user", func(t *testing.T) {
		otherUserID := xid.New()
		_, err := svc.Find(ctx, otherUserID, sess.ID)
		if err == nil {
			t.Error("expected error when finding session for wrong user")
		}
		if err.Error() != "unauthorized" {
			t.Errorf("expected 'unauthorized' error, got %v", err)
		}
	})
}

// TestService_DeleteSession tests the Delete method
func TestService_DeleteSession(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create user
	systemID := xid.New()
	appID := xid.New()
	user := &schema.User{
		ID:           xid.New(),
		AppID:        &appID,
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "$2a$10$example.hash",
	}
	user.CreatedBy = systemID
	user.UpdatedBy = systemID
	if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create two sessions
	sess1 := &schema.Session{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    user.ID,
		Token:     "token-1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "ua-1",
	}
	sess1.CreatedBy = systemID
	sess1.UpdatedBy = systemID

	sess2 := &schema.Session{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    user.ID,
		Token:     "token-2",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "ua-2",
	}
	sess2.CreatedBy = systemID
	sess2.UpdatedBy = systemID

	if _, err := db.NewInsert().Model(sess1).Exec(ctx); err != nil {
		t.Fatalf("create session1: %v", err)
	}
	if _, err := db.NewInsert().Model(sess2).Exec(ctx); err != nil {
		t.Fatalf("create session2: %v", err)
	}

	// Setup service
	sessionSvc := &mockSessionService{db: db}
	svc := &Service{
		sessionSvc: sessionSvc,
	}

	// Test Delete - correct user
	t.Run("Delete with correct user", func(t *testing.T) {
		err := svc.Delete(ctx, user.ID, sess1.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deletion
		_, err = svc.Find(ctx, user.ID, sess1.ID)
		if err == nil {
			t.Error("expected error when finding deleted session")
		}
	})

	// Test Delete - wrong user
	t.Run("Delete with wrong user", func(t *testing.T) {
		otherUserID := xid.New()
		err := svc.Delete(ctx, otherUserID, sess2.ID)
		if err == nil {
			t.Error("expected error when deleting session for wrong user")
		}
		if err.Error() != "unauthorized" {
			t.Errorf("expected 'unauthorized' error, got %v", err)
		}
	})
}
