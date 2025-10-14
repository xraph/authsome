package magiclink

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
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/audit"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
)

// setupTestApp initializes in-memory Bun DB, creates necessary tables, and mounts magic-link routes
func setupTestAppML(t *testing.T) (*bun.DB, *http.ServeMux) {
	t.Helper()
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	// Core and plugin tables
	if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create sessions: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.MagicLink)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create magic_links: %v", err)
	}

	// Initialize plugin
	p := NewPlugin()
	if err := p.Init(db); err != nil {
		t.Fatalf("plugin init: %v", err)
	}
	if err := p.Migrate(); err != nil {
		t.Fatalf("plugin migrate: %v", err)
	}

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	if err := p.RegisterRoutes(app); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	return db, mux
}

func TestMagicLink_SendReturnsDevURL(t *testing.T) {
	_, mux := setupTestAppML(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := map[string]any{"email": "ml.user@example.com"}
	buf, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/auth/magic-link/send", "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("send request error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := out["dev_url"].(string); !ok {
		t.Fatalf("expected dev_url in response for dev mode")
	}
	if out["status"] != "sent" {
		t.Fatalf("expected status=sent, got %v", out["status"])
	}
}

func TestMagicLink_VerifyCreatesSession(t *testing.T) {
	db, mux := setupTestAppML(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create user in DB
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)
	uSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
	if _, err := uSvc.Create(context.Background(), &user.CreateUserRequest{
		Email:    "ml.verify@example.com",
		Password: "password123",
		Name:     "ML Verify",
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Send magic link
	sendBody := map[string]any{"email": "ml.verify@example.com"}
	sendBuf, _ := json.Marshal(sendBody)
	resp, err := http.Post(srv.URL+"/api/auth/magic-link/send", "application/json", bytes.NewReader(sendBuf))
	if err != nil {
		t.Fatalf("send request error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 send, got %d", resp.StatusCode)
	}
	var sendOut map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&sendOut); err != nil {
		t.Fatalf("decode send: %v", err)
	}
	url, ok := sendOut["dev_url"].(string)
	if !ok || url == "" {
		t.Fatalf("expected dev_url string")
	}

	// Extract token from url query param
	// Expected format: /api/auth/magic-link/verify?token=<token>
	// So split on '=' and take last part
	parts := bytes.Split([]byte(url), []byte("="))
	token := string(parts[len(parts)-1])

	// Verify magic link
	verifyURL := srv.URL + "/api/auth/magic-link/verify?token=" + token
	resp2, err := http.Get(verifyURL)
	if err != nil {
		t.Fatalf("verify request error: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200 verify, got %d", resp2.StatusCode)
	}
	var verifyOut map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&verifyOut); err != nil {
		t.Fatalf("decode verify: %v", err)
	}
	if _, ok := verifyOut["token"].(string); !ok {
		t.Fatalf("expected token in verify response")
	}
	if _, ok := verifyOut["session"].(map[string]any); !ok {
		t.Fatalf("expected session in verify response")
	}
	if userMap, ok := verifyOut["user"].(map[string]any); !ok || userMap["Email"] != "ml.verify@example.com" {
		t.Fatalf("expected user email ml.verify@example.com, got %v", verifyOut["user"])
	}
}

func TestMagicLink_VerifyRequiresExistingUserWhenImplicitDisabled(t *testing.T) {
	// Setup in-memory DB and routes with implicit signup disabled
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create sessions: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.MagicLink)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create magic_links: %v", err)
	}

	mr := repo.NewMagicLinkRepository(db)
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
	sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, webhookSvc)
	authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})
	svc := NewService(mr, userSvc, authSvc, nil, nil, Config{BaseURL: "", DevExposeURL: true, AllowImplicitSignup: false})

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	grp := app.Group("/api/auth")
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/magic-link/send": {Window: time.Minute, Max: 5}}})
	h := NewHandler(svc, rls)
	grp.POST("/magic-link/send", h.Send)
	grp.GET("/magic-link/verify", h.Verify)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Send for non-existent user
	sendBody := map[string]any{"email": "ml.no.user@example.com"}
	sendBuf, _ := json.Marshal(sendBody)
	resp, err := http.Post(srv.URL+"/api/auth/magic-link/send", "application/json", bytes.NewReader(sendBuf))
	if err != nil {
		t.Fatalf("send request error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 send, got %d", resp.StatusCode)
	}
	var sendOut map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&sendOut); err != nil {
		t.Fatalf("decode send: %v", err)
	}
	url, ok := sendOut["dev_url"].(string)
	if !ok || url == "" {
		t.Fatalf("expected dev_url string")
	}
	parts := bytes.Split([]byte(url), []byte("="))
	token := string(parts[len(parts)-1])

	// Verify should fail with user not found
	resp2, err := http.Get(srv.URL + "/api/auth/magic-link/verify?token=" + token)
	if err != nil {
		t.Fatalf("verify request error: %v", err)
	}
	if resp2.StatusCode != 400 {
		t.Fatalf("expected 400 verify, got %d", resp2.StatusCode)
	}
	var verifyOut map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&verifyOut); err != nil {
		t.Fatalf("decode verify: %v", err)
	}
	if msg, ok := verifyOut["error"].(string); !ok || msg != "user not found" {
		t.Fatalf("expected error 'user not found', got %v", verifyOut["error"])
	}
}
