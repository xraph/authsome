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
	dev "github.com/xraph/authsome/core/device"
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

func (m *mockAuditRepo) Create(ctx context.Context, event *audit.Event) error { return nil }
func (m *mockAuditRepo) List(ctx context.Context, limit, offset int) ([]*audit.Event, error) {
	return []*audit.Event{}, nil
}
func (m *mockAuditRepo) Search(ctx context.Context, params audit.ListParams) ([]*audit.Event, error) {
	return []*audit.Event{}, nil
}
func (m *mockAuditRepo) Count(ctx context.Context) (int, error) { return 0, nil }
func (m *mockAuditRepo) SearchCount(ctx context.Context, params audit.ListParams) (int, error) {
	return 0, nil
}

// Mock webhook repository for testing
type mockWebhookRepo struct{}

func (m *mockWebhookRepo) Create(ctx context.Context, webhook *webhook.Webhook) error { return nil }
func (m *mockWebhookRepo) FindByID(ctx context.Context, id xid.ID) (*webhook.Webhook, error) {
	return nil, nil
}
func (m *mockWebhookRepo) FindByOrgID(ctx context.Context, orgID string, enabled *bool, offset, limit int) ([]*webhook.Webhook, int64, error) {
	return []*webhook.Webhook{}, 0, nil
}
func (m *mockWebhookRepo) FindByOrgAndEvent(ctx context.Context, orgID, eventType string) ([]*webhook.Webhook, error) {
	return []*webhook.Webhook{}, nil
}
func (m *mockWebhookRepo) Update(ctx context.Context, webhook *webhook.Webhook) error { return nil }
func (m *mockWebhookRepo) Delete(ctx context.Context, id xid.ID) error                { return nil }
func (m *mockWebhookRepo) UpdateFailureCount(ctx context.Context, id xid.ID, count int) error {
	return nil
}
func (m *mockWebhookRepo) UpdateLastDelivery(ctx context.Context, id xid.ID, timestamp time.Time) error {
	return nil
}
func (m *mockWebhookRepo) CreateEvent(ctx context.Context, event *webhook.Event) error { return nil }
func (m *mockWebhookRepo) FindEventByID(ctx context.Context, id xid.ID) (*webhook.Event, error) {
	return nil, nil
}
func (m *mockWebhookRepo) ListEvents(ctx context.Context, orgID string, offset, limit int) ([]*webhook.Event, int64, error) {
	return []*webhook.Event{}, 0, nil
}
func (m *mockWebhookRepo) CreateDelivery(ctx context.Context, delivery *webhook.Delivery) error {
	return nil
}
func (m *mockWebhookRepo) FindDeliveryByID(ctx context.Context, id xid.ID) (*webhook.Delivery, error) {
	return nil, nil
}
func (m *mockWebhookRepo) FindDeliveriesByWebhook(ctx context.Context, webhookID xid.ID, status string, offset, limit int) ([]*webhook.Delivery, int64, error) {
	return []*webhook.Delivery{}, 0, nil
}
func (m *mockWebhookRepo) UpdateDelivery(ctx context.Context, delivery *webhook.Delivery) error {
	return nil
}
func (m *mockWebhookRepo) FindPendingDeliveries(ctx context.Context, limit int) ([]*webhook.Delivery, error) {
	return []*webhook.Delivery{}, nil
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
	byID    map[xid.ID]*user.User
	byEmail map[string]*user.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{byID: map[xid.ID]*user.User{}, byEmail: map[string]*user.User{}}
}
func (m *memUserRepo) Create(_ context.Context, u *user.User) error {
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	return nil
}
func (m *memUserRepo) FindByID(_ context.Context, id xid.ID) (*user.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, nil
}
func (m *memUserRepo) FindByEmail(_ context.Context, email string) (*user.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, nil
}
func (m *memUserRepo) Update(_ context.Context, u *user.User) error {
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
func (m *memUserRepo) List(_ context.Context, limit, offset int) ([]*user.User, error) {
	out := []*user.User{}
	i := 0
	for _, u := range m.byID {
		if i >= offset && len(out) < limit {
			out = append(out, u)
		}
		i++
	}
	return out, nil
}
func (m *memUserRepo) Count(_ context.Context) (int, error) { return len(m.byID), nil }

// FindByUsername implements the interface; simple scan over users
func (m *memUserRepo) FindByUsername(_ context.Context, username string) (*user.User, error) {
	for _, u := range m.byID {
		if u.Username == username || u.DisplayUsername == username {
			return u, nil
		}
	}
	return nil, nil
}

// In-memory session repo
type memSessionRepo struct {
	byToken map[string]*session.Session
	byID    map[xid.ID]*session.Session
}

func newMemSessionRepo() *memSessionRepo {
	return &memSessionRepo{
		byToken: make(map[string]*session.Session),
		byID:    make(map[xid.ID]*session.Session),
	}
}
func (m *memSessionRepo) Create(_ context.Context, s *session.Session) error {
	m.byToken[s.Token] = s
	m.byID[s.ID] = s
	return nil
}
func (m *memSessionRepo) FindByToken(_ context.Context, token string) (*session.Session, error) {
	s, ok := m.byToken[token]
	if !ok {
		return nil, nil
	}
	return s, nil
}
func (m *memSessionRepo) Revoke(_ context.Context, token string) error {
	if s, ok := m.byToken[token]; ok {
		delete(m.byID, s.ID)
	}
	delete(m.byToken, token)
	return nil
}
func (m *memSessionRepo) FindByID(_ context.Context, id xid.ID) (*session.Session, error) {
	s, ok := m.byID[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}
func (m *memSessionRepo) ListByUser(_ context.Context, userID xid.ID, limit, offset int) ([]*session.Session, error) {
	var sessions []*session.Session
	count := 0
	for _, s := range m.byID {
		if s.UserID == userID {
			if count >= offset && len(sessions) < limit {
				sessions = append(sessions, s)
			}
			count++
		}
	}
	return sessions, nil
}
func (m *memSessionRepo) RevokeByID(_ context.Context, id xid.ID) error {
	if s, ok := m.byID[id]; ok {
		delete(m.byToken, s.Token)
		delete(m.byID, id)
	}
	return nil
}

func TestAuthRoutes_SignUpSignInSessionSignOut(t *testing.T) {
	// Create mock services for dependencies
	auditSvc := audit.NewService(&mockAuditRepo{})
	webhookSvc := webhook.NewService(webhook.Config{}, &mockWebhookRepo{}, auditSvc)

	// Services
	uSvc := user.NewService(newMemUserRepo(), user.Config{PasswordRequirements: validator.DefaultPasswordRequirements()}, webhookSvc)
	sSvc := session.NewService(newMemSessionRepo(), session.Config{}, webhookSvc)
	aSvc := auth.NewService(uSvc, sSvc, auth.Config{})
	// rate limit service for tests
	rlStorage := store.NewMemoryStorage()
	rlSvc := ratelimit.NewService(rlStorage, ratelimit.Config{})
	// device service is optional for tests; pass nil to keep scope minimal
	var dSvc *dev.Service
	// security and audit services optional for tests; pass nil
	h := handlers.NewAuthHandler(aSvc, rlSvc, dSvc, nil, nil, nil)

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	Register(app, "/api/auth", h)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// SignUp
	signupBody := map[string]any{"email": "alice@example.com", "password": "password123", "name": "Alice"}
	buf, _ := json.Marshal(signupBody)
	resp, err := http.Post(srv.URL+"/api/auth/signup", "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("signup request error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
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
	req3.AddCookie(&http.Cookie{Name: "session_token", Value: signinRes.Token})
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
