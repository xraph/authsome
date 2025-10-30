package phone

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
	"time"
)

// setupTestApp initializes in-memory Bun DB, creates necessary tables, and mounts phone routes
func setupTestAppPhone(t *testing.T) (*bun.DB, *http.ServeMux) {
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
	if _, err := db.NewCreateTable().Model((*schema.PhoneVerification)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create phone_verifications: %v", err)
	}

	// Create audit and webhook services for dependencies
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)

	pr := repo.NewPhoneRepository(db)
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
	sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, webhookSvc)
	authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})

	// Mock SMS provider for testing
	mockSMSProvider := &mockSMSProvider{}
	svc := NewService(pr, userSvc, authSvc, auditSvc, mockSMSProvider, Config{DevExposeCode: true, AllowImplicitSignup: false})

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	grp := app.Group("/api/auth")
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/phone/send-code": {Window: time.Minute, Max: 5}}})
	h := NewHandler(svc, rls)
	grp.POST("/phone/send-code", h.SendCode)
	grp.POST("/phone/verify", h.Verify)
	return db, mux
}

// Mock SMS provider for testing
type mockSMSProvider struct{}

func (m *mockSMSProvider) SendSMS(to, message string) error {
	// Mock implementation - just return nil for testing
	return nil
}

func TestPhone_SendReturnsDevCode(t *testing.T) {
	_, mux := setupTestAppPhone(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := map[string]any{"phone": "+15551234567"}
	buf, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/auth/phone/send-code", "application/json", bytes.NewReader(buf))
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
	if _, ok := out["dev_code"].(string); !ok {
		t.Fatalf("expected dev_code in response for dev mode")
	}
	if out["status"] != "sent" {
		t.Fatalf("expected status=sent, got %v", out["status"])
	}
}

func TestPhone_VerifyCreatesSession(t *testing.T) {
	db, mux := setupTestAppPhone(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create audit and webhook services for dependencies
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)

	// Create user in DB (email used for session creation)
	uSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
	if _, err := uSvc.Create(context.Background(), &user.CreateUserRequest{
		Email:    "phone.user@example.com",
		Password: "password123",
		Name:     "Phone User",
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Send code
	sendBody := map[string]any{"phone": "+15551234567"}
	sendBuf, _ := json.Marshal(sendBody)
	resp, err := http.Post(srv.URL+"/api/auth/phone/send-code", "application/json", bytes.NewReader(sendBuf))
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
	codeVal, ok := sendOut["dev_code"].(string)
	if !ok || codeVal == "" {
		t.Fatalf("expected dev_code string")
	}

	// Verify code using email for session
	verifyBody := map[string]any{"phone": "+15551234567", "code": codeVal, "email": "phone.user@example.com", "remember": true}
	verifyBuf, _ := json.Marshal(verifyBody)
	resp2, err := http.Post(srv.URL+"/api/auth/phone/verify", "application/json", bytes.NewReader(verifyBuf))
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
	if userMap, ok := verifyOut["user"].(map[string]any); !ok || userMap["Email"] != "phone.user@example.com" {
		t.Fatalf("expected user email phone.user@example.com, got %v", verifyOut["user"])
	}
}

func TestPhone_VerifyRequiresExistingUserWhenImplicitDisabled(t *testing.T) {
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
	if _, err := db.NewCreateTable().Model((*schema.PhoneVerification)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create phone_verifications: %v", err)
	}

	// Create audit and webhook services for dependencies
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)

	pr := repo.NewPhoneRepository(db)
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
	sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, webhookSvc)
	authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})

	// Mock SMS provider for testing
	mockSMSProvider := &mockSMSProvider{}
	svc := NewService(pr, userSvc, authSvc, auditSvc, mockSMSProvider, Config{DevExposeCode: true, AllowImplicitSignup: false})

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	grp := app.Group("/api/auth")
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/phone/send-code": {Window: time.Minute, Max: 5}}})
	h := NewHandler(svc, rls)
	grp.POST("/phone/send-code", h.SendCode)
	grp.POST("/phone/verify", h.Verify)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Send for non-existent user
	sendBody := map[string]any{"phone": "+15550987654"}
	sendBuf, _ := json.Marshal(sendBody)
	resp, err := http.Post(srv.URL+"/api/auth/phone/send-code", "application/json", bytes.NewReader(sendBuf))
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
	codeVal, ok := sendOut["dev_code"].(string)
	if !ok || codeVal == "" {
		t.Fatalf("expected dev_code string")
	}

	// Verify should fail with user not found when email missing and implicit disabled
	verifyBody := map[string]any{"phone": "+15550987654", "code": codeVal, "email": "", "remember": false}
	verifyBuf, _ := json.Marshal(verifyBody)
	resp2, err := http.Post(srv.URL+"/api/auth/phone/verify", "application/json", bytes.NewReader(verifyBuf))
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
	if msg, ok := verifyOut["error"].(string); !ok || msg != "missing fields" {
		t.Fatalf("expected error 'missing fields', got %v", verifyOut["error"])
	}
}
