package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	dev "github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/authsome/internal/validator"
	"github.com/xraph/authsome/schema"
	store "github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
)

// Mock audit repository for testing
type mockAuditRepo struct{}

func (m *mockAuditRepo) Create(ctx context.Context, event *schema.AuditEvent) error { return nil }
func (m *mockAuditRepo) Get(ctx context.Context, id xid.ID) (*schema.AuditEvent, error) {
	return nil, nil
}
func (m *mockAuditRepo) List(ctx context.Context, filter *audit.ListEventsFilter) (*pagination.PageResponse[*schema.AuditEvent], error) {
	return &pagination.PageResponse[*schema.AuditEvent]{
		Data: []*schema.AuditEvent{},
		Pagination: &pagination.PageMeta{
			Total:       0,
			Limit:       50,
			Offset:      0,
			CurrentPage: 1,
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
	}, nil
}

// Mock webhook repository for testing
type mockWebhookRepo struct{}

func (m *mockWebhookRepo) CreateWebhook(ctx context.Context, wh *schema.Webhook) error { return nil }
func (m *mockWebhookRepo) FindWebhookByID(ctx context.Context, id xid.ID) (*schema.Webhook, error) {
	return nil, nil
}
func (m *mockWebhookRepo) ListWebhooks(ctx context.Context, filter *webhook.ListWebhooksFilter) (*pagination.PageResponse[*schema.Webhook], error) {
	return &pagination.PageResponse[*schema.Webhook]{
		Data: []*schema.Webhook{},
		Pagination: &pagination.PageMeta{
			Total:       0,
			Limit:       50,
			Offset:      0,
			CurrentPage: 1,
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
	}, nil
}
func (m *mockWebhookRepo) FindWebhooksByAppAndEvent(ctx context.Context, appID xid.ID, envID xid.ID, eventType string) ([]*schema.Webhook, error) {
	return []*schema.Webhook{}, nil
}
func (m *mockWebhookRepo) UpdateWebhook(ctx context.Context, wh *schema.Webhook) error { return nil }
func (m *mockWebhookRepo) DeleteWebhook(ctx context.Context, id xid.ID) error          { return nil }
func (m *mockWebhookRepo) UpdateFailureCount(ctx context.Context, id xid.ID, count int) error {
	return nil
}
func (m *mockWebhookRepo) UpdateLastDelivery(ctx context.Context, id xid.ID, timestamp time.Time) error {
	return nil
}
func (m *mockWebhookRepo) CreateEvent(ctx context.Context, event *schema.Event) error { return nil }
func (m *mockWebhookRepo) FindEventByID(ctx context.Context, id xid.ID) (*schema.Event, error) {
	return nil, nil
}
func (m *mockWebhookRepo) ListEvents(ctx context.Context, filter *webhook.ListEventsFilter) (*pagination.PageResponse[*schema.Event], error) {
	return &pagination.PageResponse[*schema.Event]{
		Data: []*schema.Event{},
		Pagination: &pagination.PageMeta{
			Total:       0,
			Limit:       50,
			Offset:      0,
			CurrentPage: 1,
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
	}, nil
}
func (m *mockWebhookRepo) CreateDelivery(ctx context.Context, delivery *schema.Delivery) error {
	return nil
}
func (m *mockWebhookRepo) FindDeliveryByID(ctx context.Context, id xid.ID) (*schema.Delivery, error) {
	return nil, nil
}
func (m *mockWebhookRepo) ListDeliveries(ctx context.Context, filter *webhook.ListDeliveriesFilter) (*pagination.PageResponse[*schema.Delivery], error) {
	return &pagination.PageResponse[*schema.Delivery]{
		Data: []*schema.Delivery{},
		Pagination: &pagination.PageMeta{
			Total:       0,
			Limit:       50,
			Offset:      0,
			CurrentPage: 1,
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
	}, nil
}
func (m *mockWebhookRepo) UpdateDelivery(ctx context.Context, delivery *schema.Delivery) error {
	return nil
}
func (m *mockWebhookRepo) FindPendingDeliveries(ctx context.Context, limit int) ([]*schema.Delivery, error) {
	return []*schema.Delivery{}, nil
}

// Envelope type for paginated policies list responses
type PaginatedPolicies struct {
	Data       []schema.Policy
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// In-memory user repo
type memUserRepo struct {
	byID    map[xid.ID]*schema.User
	byEmail map[string]*schema.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{byID: map[xid.ID]*schema.User{}, byEmail: map[string]*schema.User{}}
}
func (m *memUserRepo) Create(_ context.Context, u *schema.User) error {
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	return nil
}
func (m *memUserRepo) FindByID(_ context.Context, id xid.ID) (*schema.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, nil
}
func (m *memUserRepo) FindByEmail(_ context.Context, email string) (*schema.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, nil
}
func (m *memUserRepo) FindByAppAndEmail(_ context.Context, appID xid.ID, email string) (*schema.User, error) {
	// For testing, ignore appID filtering
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, nil
}
func (m *memUserRepo) FindByUsername(_ context.Context, username string) (*schema.User, error) {
	for _, u := range m.byID {
		if u.Username == username || u.DisplayUsername == username {
			return u, nil
		}
	}
	return nil, nil
}
func (m *memUserRepo) Update(_ context.Context, u *schema.User) error {
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	return nil
}
func (m *memUserRepo) Delete(_ context.Context, id xid.ID) error {
	if u, ok := m.byID[id]; ok {
		delete(m.byID, id)
		delete(m.byEmail, u.Email)
	}
	return nil
}
func (m *memUserRepo) ListUsers(_ context.Context, filter *user.ListUsersFilter) (*pagination.PageResponse[*schema.User], error) {
	out := []*schema.User{}
	i := 0
	offset := filter.Offset
	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}
	for _, u := range m.byID {
		if i >= offset && len(out) < limit {
			out = append(out, u)
		}
		i++
	}
	return &pagination.PageResponse[*schema.User]{
		Data: out,
		Pagination: &pagination.PageMeta{
			Total:       int64(len(m.byID)),
			Limit:       limit,
			Offset:      offset,
			CurrentPage: (offset / limit) + 1,
			TotalPages:  (len(m.byID) + limit - 1) / limit,
			HasNext:     offset+limit < len(m.byID),
			HasPrev:     offset > 0,
		},
	}, nil
}
func (m *memUserRepo) CountUsers(_ context.Context, filter *user.CountUsersFilter) (int, error) {
	return len(m.byID), nil
}

// In-memory session repo
type memSessionRepo struct {
	byToken map[string]*schema.Session
	byID    map[xid.ID]*schema.Session
}

func newMemSessionRepo() *memSessionRepo {
	return &memSessionRepo{
		byToken: make(map[string]*schema.Session),
		byID:    make(map[xid.ID]*schema.Session),
	}
}
func (m *memSessionRepo) CreateSession(_ context.Context, s *schema.Session) error {
	m.byToken[s.Token] = s
	m.byID[s.ID] = s
	return nil
}
func (m *memSessionRepo) FindSessionByID(_ context.Context, id xid.ID) (*schema.Session, error) {
	s, ok := m.byID[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}
func (m *memSessionRepo) FindSessionByToken(_ context.Context, token string) (*schema.Session, error) {
	s, ok := m.byToken[token]
	if !ok {
		return nil, nil
	}
	return s, nil
}
func (m *memSessionRepo) ListSessions(_ context.Context, filter *session.ListSessionsFilter) (*pagination.PageResponse[*schema.Session], error) {
	var sessions []*schema.Session
	count := 0
	offset := filter.Offset
	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}
	for _, s := range m.byID {
		// Apply userID filter if present
		if filter.UserID != nil && !filter.UserID.IsNil() && s.UserID != *filter.UserID {
			continue
		}
		if count >= offset && len(sessions) < limit {
			sessions = append(sessions, s)
		}
		count++
	}
	total := count
	return &pagination.PageResponse[*schema.Session]{
		Data: sessions,
		Pagination: &pagination.PageMeta{
			Total:       int64(total),
			Limit:       limit,
			Offset:      offset,
			CurrentPage: (offset / limit) + 1,
			TotalPages:  (total + limit - 1) / limit,
			HasNext:     offset+limit < total,
			HasPrev:     offset > 0,
		},
	}, nil
}
func (m *memSessionRepo) RevokeSession(_ context.Context, token string) error {
	if s, ok := m.byToken[token]; ok {
		delete(m.byID, s.ID)
	}
	delete(m.byToken, token)
	return nil
}
func (m *memSessionRepo) RevokeSessionByID(_ context.Context, id xid.ID) error {
	if s, ok := m.byID[id]; ok {
		delete(m.byToken, s.Token)
		delete(m.byID, id)
	}
	return nil
}
func (m *memSessionRepo) CountSessions(_ context.Context, appID xid.ID, userID *xid.ID) (int, error) {
	count := 0
	for _, s := range m.byID {
		if userID != nil && !userID.IsNil() && s.UserID != *userID {
			continue
		}
		count++
	}
	return count, nil
}
func (m *memSessionRepo) CleanupExpiredSessions(_ context.Context) (int, error) {
	// For testing purposes, return 0 as we don't clean up expired sessions
	return 0, nil
}
func (m *memSessionRepo) FindSessionByRefreshToken(_ context.Context, refreshToken string) (*schema.Session, error) {
	for _, s := range m.byID {
		if s.RefreshToken != nil && *s.RefreshToken == refreshToken {
			return s, nil
		}
	}
	return nil, nil
}
func (m *memSessionRepo) UpdateSessionExpiry(_ context.Context, id xid.ID, expiresAt time.Time) error {
	if s, ok := m.byID[id]; ok {
		s.ExpiresAt = expiresAt
		s.UpdatedAt = time.Now()
		return nil
	}
	return nil
}
func (m *memSessionRepo) RefreshSessionTokens(_ context.Context, id xid.ID, newAccessToken string, accessTokenExpiresAt time.Time, newRefreshToken string, refreshTokenExpiresAt time.Time) error {
	s, ok := m.byID[id]
	if !ok {
		return nil
	}
	// Update the token maps
	delete(m.byToken, s.Token)
	s.Token = newAccessToken
	s.ExpiresAt = accessTokenExpiresAt
	s.RefreshToken = &newRefreshToken
	s.RefreshTokenExpiresAt = &refreshTokenExpiresAt
	now := time.Now()
	s.LastRefreshedAt = &now
	s.UpdatedAt = now
	m.byToken[newAccessToken] = s
	return nil
}

func TestAuthRoutes_SignUpSignInSessionSignOut(t *testing.T) {
	// Create mock services for dependencies
	auditSvc := audit.NewService(&mockAuditRepo{})
	webhookSvc := webhook.NewService(webhook.Config{}, &mockWebhookRepo{}, auditSvc)

	// Services
	uSvc := user.NewService(newMemUserRepo(), user.Config{PasswordRequirements: validator.DefaultPasswordRequirements()}, webhookSvc, nil)
	sSvc := session.NewService(newMemSessionRepo(), session.Config{}, webhookSvc, nil)
	aSvc := auth.NewService(uSvc, sSvc, auth.Config{}, nil)
	// rate limit service for tests
	rlStorage := store.NewMemoryStorage()
	rlSvc := ratelimit.NewService(rlStorage, ratelimit.Config{})
	// device service is optional for tests; pass nil to keep scope minimal
	var dSvc *dev.Service
	// security and audit services optional for tests; pass nil
	h := handlers.NewAuthHandler(aSvc, rlSvc, dSvc, nil, auditSvc, nil, nil, nil)

	// Create a test AppID
	testAppID := xid.New()

	// Create Forge app with middleware that injects AppID into context
	app := forge.NewApp(forge.AppConfig{
		Name:        "routes-test",
		Version:     "1.0.0",
		Environment: "test",
	})

	// Wrap the router with middleware that sets AppID in context
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = contexts.SetAppID(ctx, testAppID)
		r = r.WithContext(ctx)
		app.Router().ServeHTTP(w, r)
	})

	Register(app.Router(), "/api/auth", h, nil)

	srv := httptest.NewServer(wrappedHandler)

	// SignUp
	signupBody := map[string]any{"email": "alice@example.com", "password": "password123", "name": "Alice"}
	buf, _ := json.Marshal(signupBody)
	resp, err := http.Post(srv.URL+"/api/auth/signup", "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("signup request error: %v", err)
	}
	if resp.StatusCode != 200 {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		t.Fatalf("expected 200, got %d, error: %v", resp.StatusCode, errResp)
	}
	var signupRes auth.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&signupRes); err != nil {
		t.Fatalf("decode signup: %v", err)
	}
	if signupRes.Token == "" {
		t.Fatalf("expected token after signup")
	}

	// SignIn
	signinBody := map[string]any{"email": "alice@example.com", "password": "password123"}
	buf2, _ := json.Marshal(signinBody)
	resp2, err := http.Post(srv.URL+"/api/auth/signin", "application/json", bytes.NewReader(buf2))
	if err != nil {
		t.Fatalf("signin request error: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200 signin, got %d", resp2.StatusCode)
	}
	var signinRes auth.AuthResponse
	if err := json.NewDecoder(resp2.Body).Decode(&signinRes); err != nil {
		t.Fatalf("decode signin: %v", err)
	}
	if signinRes.Token == "" {
		t.Fatalf("expected token after signin")
	}

	// GetSession via cookie
	req3, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/session", nil)
	req3.AddCookie(&http.Cookie{Name: "authsome_session", Value: signinRes.Token})
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("get session error: %v", err)
	}
	if resp3.StatusCode != 200 {
		t.Fatalf("expected 200 session, got %d", resp3.StatusCode)
	}

	// SignOut
	signoutBody := map[string]any{"token": signinRes.Token}
	buf4, _ := json.Marshal(signoutBody)
	resp4, err := http.Post(srv.URL+"/api/auth/signout", "application/json", bytes.NewReader(buf4))
	if err != nil {
		t.Fatalf("signout request error: %v", err)
	}
	if resp4.StatusCode != 200 {
		t.Fatalf("expected 200 signout, got %d", resp4.StatusCode)
	}
}
