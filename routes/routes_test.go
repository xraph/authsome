package routes

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	dev "github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/ratelimit"
	rbac "github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/authsome/internal/validator"
	repo "github.com/xraph/authsome/repository"
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

func TestOrganizationPolicyRoutes_CRUD(t *testing.T) {
	// In-memory SQLite Bun DB for policy repository
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())
	// Create policies table
	if _, err := db.NewCreateTable().Model((*schema.Policy)(nil)).IfNotExists().Exec(context.Background()); err != nil {
		t.Fatalf("create table policies: %v", err)
	}
	polRepo := repo.NewPolicyRepository(db)

	// Construct OrganizationHandler with only policyRepo; disable RBAC and rate limit
	orgH := handlers.NewOrganizationHandler(nil, nil, nil, nil, nil, nil, polRepo, false)

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	RegisterOrganization(app, "/api/orgs", orgH)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create
	createBody := map[string]any{"expression": "role:owner:create,read,update,delete on policy:*"}
	buf, _ := json.Marshal(createBody)
	resp, err := http.Post(srv.URL+"/api/orgs/orgs/policies", "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("create policy request error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201 create, got %d", resp.StatusCode)
	}

	// List
	resp2, err := http.Get(srv.URL + "/api/orgs/orgs/policies")
	if err != nil {
		t.Fatalf("list policies request error: %v", err)
	}
	if resp2.StatusCode != 200 {
		var e map[string]string
		_ = json.NewDecoder(resp2.Body).Decode(&e)
		t.Fatalf("expected 200 list, got %d (error=%v)", resp2.StatusCode, e)
	}
	var env PaginatedPolicies
	if err := json.NewDecoder(resp2.Body).Decode(&env); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(env.Data) == 0 {
		t.Fatalf("expected at least 1 policy after create")
	}

	// Update
	updBody := map[string]any{"id": env.Data[0].ID.String(), "expression": "role:admin:read on policy:*"}
	buf2, _ := json.Marshal(updBody)
	resp3, err := http.Post(srv.URL+"/api/orgs/orgs/policies/update", "application/json", bytes.NewReader(buf2))
	if err != nil {
		t.Fatalf("update policy request error: %v", err)
	}
	if resp3.StatusCode != 200 {
		var e map[string]string
		_ = json.NewDecoder(resp3.Body).Decode(&e)
		t.Fatalf("expected 200 update, got %d (error=%v)", resp3.StatusCode, e)
	}

	// Delete
	delBody := map[string]any{"id": env.Data[0].ID.String()}
	buf3, _ := json.Marshal(delBody)
	resp4, err := http.Post(srv.URL+"/api/orgs/orgs/policies/delete", "application/json", bytes.NewReader(buf3))
	if err != nil {
		t.Fatalf("delete policy request error: %v", err)
	}
	if resp4.StatusCode != 200 {
		var e map[string]string
		_ = json.NewDecoder(resp4.Body).Decode(&e)
		t.Fatalf("expected 200 delete, got %d (error=%v)", resp4.StatusCode, e)
	}

	// List again should be empty
	resp5, err := http.Get(srv.URL + "/api/orgs/orgs/policies")
	if err != nil {
		t.Fatalf("list after delete request error: %v", err)
	}
	if resp5.StatusCode != 200 {
		t.Fatalf("expected 200 list after delete, got %d", resp5.StatusCode)
	}
	var env2 PaginatedPolicies
	if err := json.NewDecoder(resp5.Body).Decode(&env2); err != nil {
		t.Fatalf("decode list2: %v", err)
	}
	if len(env2.Data) != 0 {
		t.Fatalf("expected 0 policies after delete, got %d", len(env2.Data))
	}
}

func TestOrganizationPolicyRoutes_NegativeCases(t *testing.T) {
	// Setup in-memory SQLite Bun DB and policies table
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())
	if _, err := db.NewCreateTable().Model((*schema.Policy)(nil)).IfNotExists().Exec(context.Background()); err != nil {
		t.Fatalf("create table policies: %v", err)
	}
	polRepo := repo.NewPolicyRepository(db)

	// Handler with RBAC disabled
	orgH := handlers.NewOrganizationHandler(nil, nil, nil, nil, nil, nil, polRepo, false)

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	RegisterOrganization(app, "/api/orgs", orgH)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create with empty expression -> 400
	badCreate := map[string]any{"expression": ""}
	buf0, _ := json.Marshal(badCreate)
	resp0, err := http.Post(srv.URL+"/api/orgs/orgs/policies", "application/json", bytes.NewReader(buf0))
	if err != nil {
		t.Fatalf("bad create request error: %v", err)
	}
	if resp0.StatusCode != 400 {
		t.Fatalf("expected 400 create invalid, got %d", resp0.StatusCode)
	}

	// Create a valid policy to get an ID
	goodCreate := map[string]any{"expression": "role:owner:create,read,update,delete on policy:*"}
	buf1, _ := json.Marshal(goodCreate)
	resp1, err := http.Post(srv.URL+"/api/orgs/orgs/policies", "application/json", bytes.NewReader(buf1))
	if err != nil {
		t.Fatalf("create policy request error: %v", err)
	}
	if resp1.StatusCode != 201 {
		t.Fatalf("expected 201 create, got %d", resp1.StatusCode)
	}

	// List to get ID
	respList, err := http.Get(srv.URL + "/api/orgs/orgs/policies")
	if err != nil {
		t.Fatalf("list policies request error: %v", err)
	}
	if respList.StatusCode != 200 {
		t.Fatalf("expected 200 list, got %d", respList.StatusCode)
	}
	var envN PaginatedPolicies
	if err := json.NewDecoder(respList.Body).Decode(&envN); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(envN.Data) == 0 {
		t.Fatalf("expected at least 1 policy after create")
	}
	firstID := envN.Data[0].ID.String()

	// Update missing id -> 400
	badUpdateNoID := map[string]any{"expression": "role:admin:read on policy:*"}
	buf2, _ := json.Marshal(badUpdateNoID)
	resp2, err := http.Post(srv.URL+"/api/orgs/orgs/policies/update", "application/json", bytes.NewReader(buf2))
	if err != nil {
		t.Fatalf("update no id request error: %v", err)
	}
	if resp2.StatusCode != 400 {
		t.Fatalf("expected 400 update missing id, got %d", resp2.StatusCode)
	}

	// Update invalid expression (parser should reject) -> 400
	badUpdateExpr := map[string]any{"id": firstID, "expression": "this is not a valid policy"}
	buf3, _ := json.Marshal(badUpdateExpr)
	resp3, err := http.Post(srv.URL+"/api/orgs/orgs/policies/update", "application/json", bytes.NewReader(buf3))
	if err != nil {
		t.Fatalf("update invalid expr request error: %v", err)
	}
	if resp3.StatusCode != 400 {
		t.Fatalf("expected 400 update invalid expression, got %d", resp3.StatusCode)
	}

	// Delete missing id -> 400
	badDelete := map[string]any{}
	buf4, _ := json.Marshal(badDelete)
	resp4, err := http.Post(srv.URL+"/api/orgs/orgs/policies/delete", "application/json", bytes.NewReader(buf4))
	if err != nil {
		t.Fatalf("delete missing id request error: %v", err)
	}
	if resp4.StatusCode != 400 {
		t.Fatalf("expected 400 delete missing id, got %d", resp4.StatusCode)
	}
}

func TestOrganizationPolicyRoutes_RBACEnforcement(t *testing.T) {
	// In-memory SQLite Bun DB and required tables
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()
	// Create tables: policies, roles, user_roles
	if _, err := db.NewCreateTable().Model((*schema.Policy)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create table policies: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.Role)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create table roles: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.UserRole)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create table user_roles: %v", err)
	}

	polRepo := repo.NewPolicyRepository(db)
	roleRepo := repo.NewRoleRepository(db)
	userRoleRepo := repo.NewUserRoleRepository(db)

	// Seed policies for owner (full) and admin (read-only)
	if err := polRepo.Create(ctx, "role:owner:create,read,update,delete on policy:*"); err != nil {
		t.Fatalf("seed owner policy: %v", err)
	}
	if err := polRepo.Create(ctx, "role:admin:read on policy:*"); err != nil {
		t.Fatalf("seed admin policy: %v", err)
	}

	// RBAC service load policies
	rbacSvc := rbac.NewService()
	if err := rbacSvc.LoadPolicies(ctx, polRepo); err != nil {
		t.Fatalf("load policies: %v", err)
	}

	// Create roles
	ownerRole := &schema.Role{Name: "owner", Description: "Owner role"}
	if err := roleRepo.Create(ctx, ownerRole); err != nil {
		t.Fatalf("create owner role: %v", err)
	}
	adminRole := &schema.Role{Name: "admin", Description: "Admin role"}
	if err := roleRepo.Create(ctx, adminRole); err != nil {
		t.Fatalf("create admin role: %v", err)
	}

	// Session service with two users
	memSess := newMemSessionRepo()
	auditSvc := audit.NewService(&mockAuditRepo{})
	webhookSvc := webhook.NewService(webhook.Config{}, &mockWebhookRepo{}, auditSvc)
	sSvc := session.NewService(memSess, session.Config{}, webhookSvc)
	ownerUserID := xid.New()
	adminUserID := xid.New()
	ownerSess, _ := sSvc.Create(ctx, &session.CreateSessionRequest{UserID: ownerUserID})
	adminSess, _ := sSvc.Create(ctx, &session.CreateSessionRequest{UserID: adminUserID})

	// Assign roles to users
	orgID := xid.New()
	if err := userRoleRepo.Assign(ctx, ownerUserID, ownerRole.ID, orgID); err != nil {
		t.Fatalf("assign owner role: %v", err)
	}
	if err := userRoleRepo.Assign(ctx, adminUserID, adminRole.ID, orgID); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}

	// Verify role retrieval for owner before making requests
	ownerRoles, err := userRoleRepo.ListRolesForUser(ctx, ownerUserID, nil)
	if err != nil {
		t.Fatalf("list roles for owner error: %v", err)
	}
	if len(ownerRoles) == 0 {
		t.Fatalf("owner should have at least one role; got 0")
	}
	// Verify role retrieval for admin
	adminRoles, err := userRoleRepo.ListRolesForUser(ctx, adminUserID, nil)
	if err != nil {
		t.Fatalf("list roles for admin error: %v", err)
	}
	if len(adminRoles) == 0 {
		t.Fatalf("admin should have at least one role; got 0")
	}

	// Construct handler with RBAC enforcement enabled
	orgH := handlers.NewOrganizationHandler(nil, nil, sSvc, rbacSvc, userRoleRepo, roleRepo, polRepo, true)

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	RegisterOrganization(app, "/api/orgs", orgH)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Owner: Create policy allowed
	createBody := map[string]any{"expression": "role:owner:create,read on policy:*"}
	bufCreate, _ := json.Marshal(createBody)
	reqCreate, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/orgs/orgs/policies", bytes.NewReader(bufCreate))
	reqCreate.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSess.Token})
	respCreate, err := http.DefaultClient.Do(reqCreate)
	if err != nil {
		t.Fatalf("owner create policy request error: %v", err)
	}
	if respCreate.StatusCode != 201 {
		var e map[string]string
		_ = json.NewDecoder(respCreate.Body).Decode(&e)
		t.Fatalf("expected 201 owner create, got %d (error=%v)", respCreate.StatusCode, e)
	}

	// Owner: List policies allowed
	reqList, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/orgs/orgs/policies", nil)
	reqList.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSess.Token})
	respList, err := http.DefaultClient.Do(reqList)
	if err != nil {
		t.Fatalf("owner list policies request error: %v", err)
	}
	if respList.StatusCode != 200 {
		t.Fatalf("expected 200 owner list, got %d", respList.StatusCode)
	}
	var envR PaginatedPolicies
	if err := json.NewDecoder(respList.Body).Decode(&envR); err != nil {
		t.Fatalf("decode owner list: %v", err)
	}
	if len(envR.Data) == 0 {
		t.Fatalf("expected policies present for owner")
	}

	// Owner: Update policy allowed
	updBody := map[string]any{"id": envR.Data[0].ID.String(), "expression": "role:admin:read on policy:*"}
	bufUpd, _ := json.Marshal(updBody)
	reqUpd, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/orgs/orgs/policies/update", bytes.NewReader(bufUpd))
	reqUpd.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSess.Token})
	respUpd, err := http.DefaultClient.Do(reqUpd)
	if err != nil {
		t.Fatalf("owner update policy request error: %v", err)
	}
	if respUpd.StatusCode != 200 {
		var e map[string]string
		_ = json.NewDecoder(respUpd.Body).Decode(&e)
		t.Fatalf("expected 200 owner update, got %d (error=%v)", respUpd.StatusCode, e)
	}

	// Owner: Delete policy allowed
	delBody := map[string]any{"id": envR.Data[0].ID.String()}
	bufDel, _ := json.Marshal(delBody)
	reqDel, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/orgs/orgs/policies/delete", bytes.NewReader(bufDel))
	reqDel.AddCookie(&http.Cookie{Name: "session_token", Value: ownerSess.Token})
	respDel, err := http.DefaultClient.Do(reqDel)
	if err != nil {
		t.Fatalf("owner delete policy request error: %v", err)
	}
	if respDel.StatusCode != 200 {
		var e map[string]string
		_ = json.NewDecoder(respDel.Body).Decode(&e)
		t.Fatalf("expected 200 owner delete, got %d (error=%v)", respDel.StatusCode, e)
	}

	// Admin: Create policy forbidden
	reqCreate2, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/orgs/orgs/policies", bytes.NewReader(bufCreate))
	reqCreate2.AddCookie(&http.Cookie{Name: "session_token", Value: adminSess.Token})
	respCreate2, err := http.DefaultClient.Do(reqCreate2)
	if err != nil {
		t.Fatalf("admin create policy request error: %v", err)
	}
	if respCreate2.StatusCode != 403 {
		t.Fatalf("expected 403 admin create, got %d", respCreate2.StatusCode)
	}

	// Admin: List policies allowed
	reqList2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/orgs/orgs/policies", nil)
	reqList2.AddCookie(&http.Cookie{Name: "session_token", Value: adminSess.Token})
	respList2, err := http.DefaultClient.Do(reqList2)
	if err != nil {
		t.Fatalf("admin list policies request error: %v", err)
	}
	if respList2.StatusCode != 200 {
		t.Fatalf("expected 200 admin list, got %d", respList2.StatusCode)
	}

	// Admin: Update policy forbidden
	reqUpd2, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/orgs/orgs/policies/update", bytes.NewReader(bufUpd))
	reqUpd2.AddCookie(&http.Cookie{Name: "session_token", Value: adminSess.Token})
	respUpd2, err := http.DefaultClient.Do(reqUpd2)
	if err != nil {
		t.Fatalf("admin update policy request error: %v", err)
	}
	if respUpd2.StatusCode != 403 {
		t.Fatalf("expected 403 admin update, got %d", respUpd2.StatusCode)
	}

	// Admin: Delete policy forbidden
	reqDel2, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/orgs/orgs/policies/delete", bytes.NewReader(bufDel))
	reqDel2.AddCookie(&http.Cookie{Name: "session_token", Value: adminSess.Token})
	respDel2, err := http.DefaultClient.Do(reqDel2)
	if err != nil {
		t.Fatalf("admin delete policy request error: %v", err)
	}
	if respDel2.StatusCode != 403 {
		t.Fatalf("expected 403 admin delete, got %d", respDel2.StatusCode)
	}
}
