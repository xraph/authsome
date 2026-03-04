//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	pgstore "github.com/xraph/authsome/store/postgres"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// ──────────────────────────────────────────────────
// Setup helper
// ──────────────────────────────────────────────────

func setupTestStore(t *testing.T) *pgstore.Store {
	t.Helper()
	ctx := context.Background()

	container, err := pgmodule.Run(ctx, "postgres:16-alpine",
		pgmodule.WithDatabase("authsome_test"),
		pgmodule.WithUsername("test"),
		pgmodule.WithPassword("test"),
		pgmodule.BasicWaitStrategies(),
		pgmodule.WithSQLDriver("pgx"),
	)
	require.NoError(t, err, "start postgres container")

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx), "terminate container")
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "get connection string")

	pgdb := pgdriver.New()
	require.NoError(t, pgdb.Open(ctx, connStr), "open grove pg connection")

	db, err := grove.Open(pgdb)
	require.NoError(t, err, "open grove db")

	t.Cleanup(func() {
		require.NoError(t, db.Close(), "close db")
	})

	s := pgstore.New(db)
	require.NoError(t, s.Migrate(ctx), "run migrations")

	return s
}

// createTestApp is a convenience that creates + returns a persisted App.
func createTestApp(t *testing.T, s *pgstore.Store, slug string) *app.App {
	t.Helper()
	a := &app.App{
		ID:        id.NewAppID(),
		Name:      "Test App " + slug,
		Slug:      slug,
		Metadata:  app.Metadata{"env": "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateApp(context.Background(), a))
	return a
}

// createTestUser creates + returns a persisted User scoped to the given app.
func createTestUser(t *testing.T, s *pgstore.Store, appID id.AppID, email string) *user.User {
	t.Helper()
	u := &user.User{
		ID:           id.NewUserID(),
		AppID:        appID,
		Email:        email,
		FirstName:    "Test User",
		Username:     email[:len(email)-len("@test.com")],
		PasswordHash: "$2a$10$testhashedpassword",
		Metadata:     user.Metadata{"role": "user"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, s.CreateUser(context.Background(), u))
	return u
}

// ──────────────────────────────────────────────────
// Lifecycle tests
// ──────────────────────────────────────────────────

func TestPing(t *testing.T) {
	s := setupTestStore(t)
	require.NoError(t, s.Ping(context.Background()))
}

func TestMigrate_Idempotent(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Migrate was already called in setupTestStore; running again should be safe.
	require.NoError(t, s.Migrate(ctx))
	require.NoError(t, s.Migrate(ctx))
}

// ──────────────────────────────────────────────────
// App store tests
// ──────────────────────────────────────────────────

func TestApp_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	a := &app.App{
		ID:         id.NewAppID(),
		Name:       "My App",
		Slug:       "my-app",
		Logo:       "https://example.com/logo.png",
		IsPlatform: true,
		Metadata:   app.Metadata{"plan": "pro"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Create
	require.NoError(t, s.CreateApp(ctx, a))

	// Get by ID
	got, err := s.GetApp(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.Name, got.Name)
	assert.Equal(t, a.Slug, got.Slug)
	assert.Equal(t, a.Logo, got.Logo)
	assert.True(t, got.IsPlatform)
	assert.Equal(t, "pro", got.Metadata["plan"])

	// Get by slug
	got, err = s.GetAppBySlug(ctx, "my-app")
	require.NoError(t, err)
	assert.Equal(t, a.ID, got.ID)

	// Update
	a.Name = "My Updated App"
	a.Logo = "https://example.com/new-logo.png"
	require.NoError(t, s.UpdateApp(ctx, a))

	got, err = s.GetApp(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "My Updated App", got.Name)
	assert.Equal(t, "https://example.com/new-logo.png", got.Logo)

	// List
	a2 := createTestApp(t, s, "second-app")
	apps, err := s.ListApps(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(apps), 2)
	_ = a2

	// Delete
	require.NoError(t, s.DeleteApp(ctx, a.ID))
	_, err = s.GetApp(ctx, a.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestApp_UniqueSlugs(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	createTestApp(t, s, "unique-slug")

	// Duplicate slug should fail
	dup := &app.App{
		ID:        id.NewAppID(),
		Name:      "Dup",
		Slug:      "unique-slug",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.CreateApp(ctx, dup)
	require.Error(t, err)
}

func TestApp_GetNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.GetApp(context.Background(), id.NewAppID())
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// ──────────────────────────────────────────────────
// User store tests
// ──────────────────────────────────────────────────

func TestUser_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "user-test")

	u := &user.User{
		ID:              id.NewUserID(),
		AppID:           a.ID,
		Email:           "alice@test.com",
		EmailVerified:   true,
		FirstName:       "Alice",
		Username:        "alice",
		DisplayUsername: "Alice",
		Phone:           "+1234567890",
		PasswordHash:    "$2a$10$hash",
		Metadata:        user.Metadata{"tier": "premium"},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Create
	require.NoError(t, s.CreateUser(ctx, u))

	// Get by ID
	got, err := s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alice", got.FirstName)
	assert.Equal(t, "alice@test.com", got.Email)
	assert.True(t, got.EmailVerified)
	assert.Equal(t, "+1234567890", got.Phone)
	assert.Equal(t, "premium", got.Metadata["tier"])

	// Get by email
	got, err = s.GetUserByEmail(ctx, a.ID, "alice@test.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	// Get by username
	got, err = s.GetUserByUsername(ctx, a.ID, "alice")
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	// Update
	u.FirstName = "Alice Updated"
	u.Banned = true
	u.BanReason = "testing"
	banExpires := time.Now().Add(24 * time.Hour)
	u.BanExpires = &banExpires
	require.NoError(t, s.UpdateUser(ctx, u))

	got, err = s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", got.FirstName)
	assert.True(t, got.Banned)
	assert.Equal(t, "testing", got.BanReason)
	require.NotNil(t, got.BanExpires)

	// List
	createTestUser(t, s, a.ID, "bob@test.com")
	list, err := s.ListUsers(ctx, &user.UserQuery{AppID: a.ID, Limit: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Users), 2)

	// Soft delete
	require.NoError(t, s.DeleteUser(ctx, u.ID))
	_, err = s.GetUser(ctx, u.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "soft-deleted user should not be found")

	// Soft-deleted user should not appear in email lookups
	_, err = s.GetUserByEmail(ctx, a.ID, "alice@test.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestUser_UniqueEmailPerApp(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "email-unique")

	createTestUser(t, s, a.ID, "dup@test.com")

	// Same email, same app -> should fail
	dup := &user.User{
		ID:        id.NewUserID(),
		AppID:     a.ID,
		Email:     "dup@test.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.CreateUser(ctx, dup)
	require.Error(t, err)
}

func TestUser_SameEmailDifferentApp(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a1 := createTestApp(t, s, "app-one")
	a2 := createTestApp(t, s, "app-two")

	createTestUser(t, s, a1.ID, "shared@test.com")

	// Same email, different app -> should succeed
	u2 := &user.User{
		ID:        id.NewUserID(),
		AppID:     a2.ID,
		Email:     "shared@test.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateUser(ctx, u2))
}

func TestUser_ListWithLimit(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "pagination-test")

	// Create 5 users with unique emails
	emails := []string{"page-a@test.com", "page-b@test.com", "page-c@test.com", "page-d@test.com", "page-e@test.com"}
	for _, email := range emails {
		createTestUser(t, s, a.ID, email)
	}

	// Get all
	all, err := s.ListUsers(ctx, &user.UserQuery{AppID: a.ID, Limit: 50})
	require.NoError(t, err)
	assert.Len(t, all.Users, 5, "should have 5 users total")
	assert.Empty(t, all.NextCursor, "should not need cursor for full page")

	// Limit to 3 — should get 3 and indicate more are available
	limited, err := s.ListUsers(ctx, &user.UserQuery{AppID: a.ID, Limit: 3})
	require.NoError(t, err)
	assert.Len(t, limited.Users, 3)
	assert.NotEmpty(t, limited.NextCursor, "should have cursor when more results exist")

	// Default limit (no explicit limit)
	defaulted, err := s.ListUsers(ctx, &user.UserQuery{AppID: a.ID})
	require.NoError(t, err)
	assert.Len(t, defaulted.Users, 5, "default limit should be enough for 5 users")
}

func TestUser_ListByEmail(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "email-filter")

	createTestUser(t, s, a.ID, "findme@test.com")
	createTestUser(t, s, a.ID, "other@test.com")

	list, err := s.ListUsers(ctx, &user.UserQuery{AppID: a.ID, Email: "findme"})
	require.NoError(t, err)
	assert.Len(t, list.Users, 1)
	assert.Equal(t, "findme@test.com", list.Users[0].Email)
}

func TestUser_GetNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.GetUser(context.Background(), id.NewUserID())
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// ──────────────────────────────────────────────────
// Session store tests
// ──────────────────────────────────────────────────

func TestSession_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "session-test")
	u := createTestUser(t, s, a.ID, "session-user@test.com")

	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 a.ID,
		UserID:                u.ID,
		Token:                 "tok_" + id.NewSessionID().String(),
		RefreshToken:          "rtk_" + id.NewSessionID().String(),
		IPAddress:             "127.0.0.1",
		UserAgent:             "Go-Test/1.0",
		ExpiresAt:             time.Now().Add(1 * time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Create
	require.NoError(t, s.CreateSession(ctx, sess))

	// Get by ID
	got, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.Token, got.Token)
	assert.Equal(t, sess.RefreshToken, got.RefreshToken)
	assert.Equal(t, sess.IPAddress, got.IPAddress)
	assert.Equal(t, sess.UserAgent, got.UserAgent)

	// Get by token
	got, err = s.GetSessionByToken(ctx, sess.Token)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, got.ID)

	// Get by refresh token
	got, err = s.GetSessionByRefreshToken(ctx, sess.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, got.ID)

	// Update
	sess.IPAddress = "192.168.1.1"
	require.NoError(t, s.UpdateSession(ctx, sess))
	got, err = s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.1", got.IPAddress)

	// List user sessions
	sess2 := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 a.ID,
		UserID:                u.ID,
		Token:                 "tok2_" + id.NewSessionID().String(),
		RefreshToken:          "rtk2_" + id.NewSessionID().String(),
		ExpiresAt:             time.Now().Add(1 * time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	require.NoError(t, s.CreateSession(ctx, sess2))

	sessions, err := s.ListUserSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Delete single session
	require.NoError(t, s.DeleteSession(ctx, sess.ID))
	_, err = s.GetSession(ctx, sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)

	// Delete all user sessions
	require.NoError(t, s.DeleteUserSessions(ctx, u.ID))
	sessions, err = s.ListUserSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestSession_UniqueTokens(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "sess-unique")
	u := createTestUser(t, s, a.ID, "sess-uniq@test.com")

	sharedToken := "unique_token_" + id.NewSessionID().String()

	s1 := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 a.ID,
		UserID:                u.ID,
		Token:                 sharedToken,
		RefreshToken:          "rt1_" + id.NewSessionID().String(),
		ExpiresAt:             time.Now().Add(time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	require.NoError(t, s.CreateSession(ctx, s1))

	// Duplicate token should fail
	s2 := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 a.ID,
		UserID:                u.ID,
		Token:                 sharedToken,
		RefreshToken:          "rt2_" + id.NewSessionID().String(),
		ExpiresAt:             time.Now().Add(time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	err := s.CreateSession(ctx, s2)
	require.Error(t, err)
}

func TestSession_GetNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.GetSession(context.Background(), id.NewSessionID())
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// ──────────────────────────────────────────────────
// Account store tests (verification + password reset)
// ──────────────────────────────────────────────────

func TestVerification_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "verify-test")
	u := createTestUser(t, s, a.ID, "verify@test.com")

	v := &account.Verification{
		ID:        id.NewVerificationID(),
		AppID:     a.ID,
		UserID:    u.ID,
		Token:     "vrf_" + id.NewVerificationID().String(),
		Type:      account.VerificationEmail,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Consumed:  false,
		CreatedAt: time.Now(),
	}

	// Create
	require.NoError(t, s.CreateVerification(ctx, v))

	// Get
	got, err := s.GetVerification(ctx, v.Token)
	require.NoError(t, err)
	assert.Equal(t, v.ID, got.ID)
	assert.Equal(t, account.VerificationEmail, got.Type)
	assert.False(t, got.Consumed)

	// Consume
	require.NoError(t, s.ConsumeVerification(ctx, v.Token))
	got, err = s.GetVerification(ctx, v.Token)
	require.NoError(t, err)
	assert.True(t, got.Consumed)

	// Consuming again should be a no-op (already consumed)
	require.NoError(t, s.ConsumeVerification(ctx, v.Token))
}

func TestVerification_GetNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.GetVerification(context.Background(), "nonexistent-token")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestPasswordReset_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "pwreset-test")
	u := createTestUser(t, s, a.ID, "pwreset@test.com")

	pr := &account.PasswordReset{
		ID:        id.NewPasswordResetID(),
		AppID:     a.ID,
		UserID:    u.ID,
		Token:     "pwr_" + id.NewPasswordResetID().String(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Consumed:  false,
		CreatedAt: time.Now(),
	}

	// Create
	require.NoError(t, s.CreatePasswordReset(ctx, pr))

	// Get
	got, err := s.GetPasswordReset(ctx, pr.Token)
	require.NoError(t, err)
	assert.Equal(t, pr.ID, got.ID)
	assert.False(t, got.Consumed)

	// Consume
	require.NoError(t, s.ConsumePasswordReset(ctx, pr.Token))
	got, err = s.GetPasswordReset(ctx, pr.Token)
	require.NoError(t, err)
	assert.True(t, got.Consumed)
}

func TestPasswordReset_UniqueToken(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "pr-unique")
	u := createTestUser(t, s, a.ID, "pr-uniq@test.com")

	sharedToken := "unique_pwr_" + id.NewPasswordResetID().String()

	pr1 := &account.PasswordReset{
		ID:        id.NewPasswordResetID(),
		AppID:     a.ID,
		UserID:    u.ID,
		Token:     sharedToken,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, s.CreatePasswordReset(ctx, pr1))

	pr2 := &account.PasswordReset{
		ID:        id.NewPasswordResetID(),
		AppID:     a.ID,
		UserID:    u.ID,
		Token:     sharedToken,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	err := s.CreatePasswordReset(ctx, pr2)
	require.Error(t, err)
}

// ──────────────────────────────────────────────────
// Organization store tests
// ──────────────────────────────────────────────────

func TestOrganization_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "org-test")
	u := createTestUser(t, s, a.ID, "org-owner@test.com")

	org := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     a.ID,
		Name:      "Acme Corp",
		Slug:      "acme-corp",
		Logo:      "https://acme.com/logo.png",
		Metadata:  organization.Metadata{"industry": "tech"},
		CreatedBy: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create
	require.NoError(t, s.CreateOrganization(ctx, org))

	// Get by ID
	got, err := s.GetOrganization(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", got.Name)
	assert.Equal(t, "acme-corp", got.Slug)
	assert.Equal(t, "tech", got.Metadata["industry"])
	assert.Equal(t, u.ID, got.CreatedBy)

	// Get by slug
	got, err = s.GetOrganizationBySlug(ctx, a.ID, "acme-corp")
	require.NoError(t, err)
	assert.Equal(t, org.ID, got.ID)

	// Update
	org.Name = "Acme Corp Updated"
	require.NoError(t, s.UpdateOrganization(ctx, org))
	got, err = s.GetOrganization(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, "Acme Corp Updated", got.Name)

	// List by app
	org2 := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     a.ID,
		Name:      "Beta Inc",
		Slug:      "beta-inc",
		CreatedBy: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(ctx, org2))

	orgs, err := s.ListOrganizations(ctx, a.ID)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)

	// Delete
	require.NoError(t, s.DeleteOrganization(ctx, org.ID))
	_, err = s.GetOrganization(ctx, org.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestOrganization_UniqueSlugPerApp(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "org-slug-uniq")
	u := createTestUser(t, s, a.ID, "slug-owner@test.com")

	org1 := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Org One", Slug: "same-slug",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(ctx, org1))

	org2 := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Org Two", Slug: "same-slug",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	err := s.CreateOrganization(ctx, org2)
	require.Error(t, err)
}

func TestOrganization_GetNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.GetOrganization(context.Background(), id.NewOrgID())
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestOrganization_ListUserOrganizations(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "user-orgs")
	u := createTestUser(t, s, a.ID, "multi-org@test.com")

	org1 := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Org A", Slug: "org-a",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	org2 := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Org B", Slug: "org-b",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(ctx, org1))
	require.NoError(t, s.CreateOrganization(ctx, org2))

	mem1 := &organization.Member{
		ID: id.NewMemberID(), OrgID: org1.ID, UserID: u.ID,
		Role: organization.RoleOwner, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mem2 := &organization.Member{
		ID: id.NewMemberID(), OrgID: org2.ID, UserID: u.ID,
		Role: organization.RoleMember, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateMember(ctx, mem1))
	require.NoError(t, s.CreateMember(ctx, mem2))

	orgs, err := s.ListUserOrganizations(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)
}

// ──────────────────────────────────────────────────
// Member store tests
// ──────────────────────────────────────────────────

func TestMember_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "member-test")
	u := createTestUser(t, s, a.ID, "member@test.com")
	u2 := createTestUser(t, s, a.ID, "member2@test.com")

	org := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Members Org", Slug: "members-org",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(ctx, org))

	mem := &organization.Member{
		ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID,
		Role: organization.RoleOwner, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	require.NoError(t, s.CreateMember(ctx, mem))

	got, err := s.GetMember(ctx, mem.ID)
	require.NoError(t, err)
	assert.Equal(t, organization.RoleOwner, got.Role)

	got, err = s.GetMemberByUserAndOrg(ctx, u.ID, org.ID)
	require.NoError(t, err)
	assert.Equal(t, mem.ID, got.ID)

	mem.Role = organization.RoleAdmin
	require.NoError(t, s.UpdateMember(ctx, mem))
	got, err = s.GetMember(ctx, mem.ID)
	require.NoError(t, err)
	assert.Equal(t, organization.RoleAdmin, got.Role)

	mem2 := &organization.Member{
		ID: id.NewMemberID(), OrgID: org.ID, UserID: u2.ID,
		Role: organization.RoleMember, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateMember(ctx, mem2))

	members, err := s.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2)

	require.NoError(t, s.DeleteMember(ctx, mem.ID))
	_, err = s.GetMember(ctx, mem.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestMember_UniqueUserOrgConstraint(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "mem-unique")
	u := createTestUser(t, s, a.ID, "mem-uniq@test.com")

	org := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Uniq Org", Slug: "uniq-org",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(ctx, org))

	mem1 := &organization.Member{
		ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID,
		Role: organization.RoleOwner, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateMember(ctx, mem1))

	mem2 := &organization.Member{
		ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID,
		Role: organization.RoleMember, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	err := s.CreateMember(ctx, mem2)
	require.Error(t, err)
}

func TestMember_GetNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.GetMember(context.Background(), id.NewMemberID())
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// ──────────────────────────────────────────────────
// Invitation, Team, Device, Webhook, Notification, APIKey tests
// (Remaining tests follow the same pattern as above)
// ──────────────────────────────────────────────────

func TestInvitation_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "inv-test")
	u := createTestUser(t, s, a.ID, "inviter@test.com")

	org := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Inv Org", Slug: "inv-org",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(ctx, org))

	inv := &organization.Invitation{
		ID: id.NewInvitationID(), OrgID: org.ID, Email: "invitee@test.com",
		Role: organization.RoleMember, InviterID: u.ID, Status: organization.InvitationPending,
		Token: "inv_" + id.NewInvitationID().String(), ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}

	require.NoError(t, s.CreateInvitation(ctx, inv))

	got, err := s.GetInvitation(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, "invitee@test.com", got.Email)

	got, err = s.GetInvitationByToken(ctx, inv.Token)
	require.NoError(t, err)
	assert.Equal(t, inv.ID, got.ID)

	inv.Status = organization.InvitationAccepted
	require.NoError(t, s.UpdateInvitation(ctx, inv))
	got, err = s.GetInvitation(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, organization.InvitationAccepted, got.Status)

	invitations, err := s.ListInvitations(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, invitations, 1)
}

func TestTeam_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "team-test")
	u := createTestUser(t, s, a.ID, "team-owner@test.com")

	org := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Team Org", Slug: "team-org",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(ctx, org))

	team := &organization.Team{
		ID: id.NewTeamID(), OrgID: org.ID, Name: "Engineering", Slug: "engineering",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	require.NoError(t, s.CreateTeam(ctx, team))

	got, err := s.GetTeam(ctx, team.ID)
	require.NoError(t, err)
	assert.Equal(t, "Engineering", got.Name)

	team.Name = "Platform Engineering"
	require.NoError(t, s.UpdateTeam(ctx, team))

	teams, err := s.ListTeams(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 1)

	require.NoError(t, s.DeleteTeam(ctx, team.ID))
	_, err = s.GetTeam(ctx, team.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestDevice_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "device-test")
	u := createTestUser(t, s, a.ID, "device-user@test.com")

	dev := &device.Device{
		ID: id.NewDeviceID(), UserID: u.ID, AppID: a.ID,
		Name: "MacBook Pro", Type: "desktop", Browser: "Chrome", OS: "macOS",
		IPAddress: "10.0.0.1", Fingerprint: "fp_abc123", Trusted: true,
		LastSeenAt: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	require.NoError(t, s.CreateDevice(ctx, dev))

	got, err := s.GetDevice(ctx, dev.ID)
	require.NoError(t, err)
	assert.Equal(t, "MacBook Pro", got.Name)
	assert.True(t, got.Trusted)

	got, err = s.GetDeviceByFingerprint(ctx, u.ID, "fp_abc123")
	require.NoError(t, err)
	assert.Equal(t, dev.ID, got.ID)

	devices, err := s.ListUserDevices(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, devices, 1)

	require.NoError(t, s.DeleteDevice(ctx, dev.ID))
	_, err = s.GetDevice(ctx, dev.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestWebhook_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "webhook-test")

	wh := &webhook.Webhook{
		ID: id.NewWebhookID(), AppID: a.ID,
		URL: "https://example.com/webhook", Events: []string{"user.created"},
		Secret: "wh_secret_123", Active: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	require.NoError(t, s.CreateWebhook(ctx, wh))

	got, err := s.GetWebhook(ctx, wh.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/webhook", got.URL)

	webhooks, err := s.ListWebhooks(ctx, a.ID)
	require.NoError(t, err)
	assert.Len(t, webhooks, 1)

	require.NoError(t, s.DeleteWebhook(ctx, wh.ID))
	_, err = s.GetWebhook(ctx, wh.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestNotification_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "notif-test")
	u := createTestUser(t, s, a.ID, "notif-user@test.com")

	n := &notification.Notification{
		ID: id.NewNotificationID(), AppID: a.ID, UserID: u.ID,
		Type: "welcome", Channel: notification.ChannelEmail,
		Subject: "Welcome!", Body: "Welcome to AuthSome",
		CreatedAt: time.Now(),
	}

	require.NoError(t, s.CreateNotification(ctx, n))

	got, err := s.GetNotification(ctx, n.ID)
	require.NoError(t, err)
	assert.False(t, got.Sent)

	require.NoError(t, s.MarkSent(ctx, n.ID))
	got, err = s.GetNotification(ctx, n.ID)
	require.NoError(t, err)
	assert.True(t, got.Sent)
	require.NotNil(t, got.SentAt)

	notifications, err := s.ListUserNotifications(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
}

func TestAPIKey_CRUD(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "apikey-crud-app")
	u := createTestUser(t, s, a.ID, "apikey-user@test.com")

	now := time.Now().Truncate(time.Millisecond)
	expires := now.Add(24 * time.Hour)

	key := &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: a.ID, UserID: u.ID,
		Name: "Test Key", KeyHash: "sha256_testhash_abc123", KeyPrefix: "ak_test1",
		Scopes: []string{"read", "write"}, ExpiresAt: &expires,
		CreatedAt: now, UpdatedAt: now,
	}

	require.NoError(t, s.CreateAPIKey(ctx, key))

	got, err := s.GetAPIKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, key.Name, got.Name)
	assert.Equal(t, key.Scopes, got.Scopes)

	got2, err := s.GetAPIKeyByPrefix(ctx, a.ID, "ak_test1")
	require.NoError(t, err)
	assert.Equal(t, key.ID, got2.ID)

	got.Revoked = true
	require.NoError(t, s.UpdateAPIKey(ctx, got))
	got3, err := s.GetAPIKey(ctx, key.ID)
	require.NoError(t, err)
	assert.True(t, got3.Revoked)

	appKeys, err := s.ListAPIKeysByApp(ctx, a.ID)
	require.NoError(t, err)
	assert.Len(t, appKeys, 1)

	userKeys, err := s.ListAPIKeysByUser(ctx, a.ID, u.ID)
	require.NoError(t, err)
	assert.Len(t, userKeys, 1)

	require.NoError(t, s.DeleteAPIKey(ctx, key.ID))
	_, err = s.GetAPIKey(ctx, key.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestAPIKey_GetNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.GetAPIKey(context.Background(), id.NewAPIKeyID())
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestOrganization_DeleteCascadesMembers(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	a := createTestApp(t, s, "cascade-test")
	u := createTestUser(t, s, a.ID, "cascade@test.com")

	org := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, Name: "Cascade Org", Slug: "cascade-org",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(ctx, org))

	mem := &organization.Member{
		ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID,
		Role: organization.RoleOwner, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateMember(ctx, mem))

	team := &organization.Team{
		ID: id.NewTeamID(), OrgID: org.ID, Name: "Cascade Team", Slug: "cascade-team",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateTeam(ctx, team))

	require.NoError(t, s.DeleteOrganization(ctx, org.ID))

	_, err := s.GetMember(ctx, mem.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)

	_, err = s.GetTeam(ctx, team.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}
