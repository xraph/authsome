package passkey

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
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// setupTestApp initializes in-memory Bun DB, creates necessary tables, and mounts passkey routes
func setupTestAppPK(t *testing.T) (*bun.DB, *http.ServeMux) {
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
	if _, err := db.NewCreateTable().Model((*schema.Passkey)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create passkeys: %v", err)
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

func TestPasskey_RegisterFlowPersistsPasskey(t *testing.T) {
	db, mux := setupTestAppPK(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create audit and webhook services for dependencies
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)

	// Create a user
	uSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
	u, err := uSvc.Create(context.Background(), &user.CreateUserRequest{
		Email:    "pk.user@example.com",
		Password: "password123",
		Name:     "PK User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Begin registration
	beginBody := map[string]any{"user_id": u.ID.String()}
	beginBuf, _ := json.Marshal(beginBody)
	resp, err := http.Post(srv.URL+"/api/auth/passkey/register/begin", "application/json", bytes.NewReader(beginBuf))
	if err != nil {
		t.Fatalf("begin request error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var beginOut map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&beginOut); err != nil {
		t.Fatalf("decode begin: %v", err)
	}
	if _, ok := beginOut["challenge"]; !ok {
		t.Fatalf("expected challenge in begin response")
	}
	if beginOut["userId"] != u.ID.String() {
		t.Fatalf("expected userId %s, got %v", u.ID.String(), beginOut["userId"])
	}

	// Finish registration
	credID := "cred-123"
	finishBody := map[string]any{"user_id": u.ID.String(), "credential_id": credID}
	finishBuf, _ := json.Marshal(finishBody)
	resp2, err := http.Post(srv.URL+"/api/auth/passkey/register/finish", "application/json", bytes.NewReader(finishBuf))
	if err != nil {
		t.Fatalf("finish request error: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200 finish, got %d", resp2.StatusCode)
	}

	// List passkeys should include the new credential
	resp3, err := http.Get(srv.URL + "/api/auth/passkey/list?user_id=" + u.ID.String())
	if err != nil {
		t.Fatalf("list request error: %v", err)
	}
	if resp3.StatusCode != 200 {
		t.Fatalf("expected 200 list, got %d", resp3.StatusCode)
	}
	var listOut []map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&listOut); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listOut) != 1 {
		t.Fatalf("expected 1 passkey, got %d", len(listOut))
	}
	if listOut[0]["credentialID"] != credID {
		t.Fatalf("expected credentialID=%s, got %v", credID, listOut[0]["credentialID"])
	}
}

func TestPasskey_LoginReturnsSession(t *testing.T) {
	db, mux := setupTestAppPK(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create audit and webhook services for dependencies
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)

	// Create a user
	uSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
	u, err := uSvc.Create(context.Background(), &user.CreateUserRequest{
		Email:    "pk.login@example.com",
		Password: "password123",
		Name:     "PK Login",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Begin login
	beginBody := map[string]any{"user_id": u.ID.String()}
	beginBuf, _ := json.Marshal(beginBody)
	resp, err := http.Post(srv.URL+"/api/auth/passkey/login/begin", "application/json", bytes.NewReader(beginBuf))
	if err != nil {
		t.Fatalf("begin login request error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Finish login
	finishBody := map[string]any{"user_id": u.ID.String(), "remember": true}
	finishBuf, _ := json.Marshal(finishBody)
	resp2, err := http.Post(srv.URL+"/api/auth/passkey/login/finish", "application/json", bytes.NewReader(finishBuf))
	if err != nil {
		t.Fatalf("finish login request error: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200 finish, got %d", resp2.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&out); err != nil {
		t.Fatalf("decode finish: %v", err)
	}
	if _, ok := out["token"].(string); !ok {
		t.Fatalf("expected token in finish response")
	}
	if _, ok := out["session"].(map[string]any); !ok {
		t.Fatalf("expected session in finish response")
	}
	if userMap, ok := out["user"].(map[string]any); !ok || userMap["Email"] != "pk.login@example.com" {
		t.Fatalf("expected user email pk.login@example.com, got %v", out["user"])
	}
}

func TestPasskey_DeleteRemovesCredential(t *testing.T) {
	db, mux := setupTestAppPK(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create audit and webhook services for dependencies
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)

	// Create a user and register one passkey
	uSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
	_ = session.NewService(repo.NewSessionRepository(db), session.Config{}, webhookSvc)
	u, err := uSvc.Create(context.Background(), &user.CreateUserRequest{
		Email:    "pk.delete@example.com",
		Password: "password123",
		Name:     "PK Delete",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	credID := "cred-del-1"
	finishBody := map[string]any{"user_id": u.ID.String(), "credential_id": credID}
	finishBuf, _ := json.Marshal(finishBody)
	resp, err := http.Post(srv.URL+"/api/auth/passkey/register/finish", "application/json", bytes.NewReader(finishBuf))
	if err != nil {
		t.Fatalf("finish request error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 finish, got %d", resp.StatusCode)
	}

	// List to get ID
	resp2, err := http.Get(srv.URL + "/api/auth/passkey/list?user_id=" + u.ID.String())
	if err != nil {
		t.Fatalf("list request error: %v", err)
	}
	var listOut []map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&listOut); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listOut) != 1 {
		t.Fatalf("expected 1 passkey, got %d", len(listOut))
	}
	idStr, ok := listOut[0]["id"].(string)
	if !ok || idStr == "" {
		t.Fatalf("expected id string in list output")
	}

	// Delete via POST route and verify list is empty
	req, _ := http.NewRequest("POST", srv.URL+"/api/auth/passkey/delete/"+idStr, nil)
	resp3, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete request error: %v", err)
	}
	if resp3.StatusCode != 200 {
		t.Fatalf("expected 200 delete, got %d", resp3.StatusCode)
	}

	resp4, err := http.Get(srv.URL + "/api/auth/passkey/list?user_id=" + u.ID.String())
	if err != nil {
		t.Fatalf("list request error: %v", err)
	}
	var listOut2 []map[string]any
	if err := json.NewDecoder(resp4.Body).Decode(&listOut2); err != nil {
		t.Fatalf("decode list2: %v", err)
	}
	if len(listOut2) != 0 {
		t.Fatalf("expected 0 passkeys after delete, got %d", len(listOut2))
	}
}
