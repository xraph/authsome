package emailotp

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "database/sql"
    "time"

    "github.com/xraph/authsome/core/auth"
    "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/core/webhook"
    "github.com/xraph/authsome/core/audit"
    rl "github.com/xraph/authsome/core/ratelimit"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/authsome/schema"
    "github.com/xraph/authsome/storage"
    "github.com/xraph/forge"
    "github.com/uptrace/bun"
    "github.com/uptrace/bun/dialect/sqlitedialect"
    _ "github.com/mattn/go-sqlite3"
)

// setupTestApp initializes in-memory Bun DB, creates necessary tables, and mounts email-otp routes
func setupTestApp(t *testing.T) (*bun.DB, *http.ServeMux) {
    t.Helper()
    sqldb, err := sql.Open("sqlite3", ":memory:")
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    db := bun.NewDB(sqldb, sqlitedialect.New())

    ctx := context.Background()
    // Core tables required by plugin/service interactions
    if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil { t.Fatalf("create users: %v", err) }
    if _, err := db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx); err != nil { t.Fatalf("create sessions: %v", err) }

    // Initialize plugin and run migration for email_otps
    p := NewPlugin()
    if err := p.Init(db); err != nil { t.Fatalf("plugin init: %v", err) }
    if err := p.Migrate(); err != nil { t.Fatalf("plugin migrate: %v", err) }

    mux := http.NewServeMux()
    app := forge.NewApp(mux)
    if err := p.RegisterRoutes(app); err != nil { t.Fatalf("register routes: %v", err) }
    // Basic health to ease local debugging if needed
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
    return db, mux
}

func TestEmailOTP_SendReturnsDevOTP(t *testing.T) {
    _, mux := setupTestApp(t)
    srv := httptest.NewServer(mux)
    defer srv.Close()

    body := map[string]any{"email": "test.user@example.com"}
    buf, _ := json.Marshal(body)
    resp, err := http.Post(srv.URL+"/api/auth/email-otp/send", "application/json", bytes.NewReader(buf))
    if err != nil { t.Fatalf("send request error: %v", err) }
    if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }

    var out map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { t.Fatalf("decode: %v", err) }
    if _, ok := out["dev_otp"]; !ok { t.Fatalf("expected dev_otp in response for dev mode") }
    if out["status"] != "sent" { t.Fatalf("expected status=sent, got %v", out["status"]) }
}

func TestEmailOTP_VerifyCreatesSession(t *testing.T) {
    db, mux := setupTestApp(t)
    srv := httptest.NewServer(mux)
    defer srv.Close()

    // Create user in DB
    auditSvc := audit.NewService(repo.NewAuditRepository(db))
    webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)
    uSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
    if _, err := uSvc.Create(context.Background(), &user.CreateUserRequest{
        Email:    "verify.user@example.com",
        Password: "password123",
        Name:     "Verify User",
    }); err != nil { t.Fatalf("create user: %v", err) }

    // Send OTP
    sendBody := map[string]any{"email": "verify.user@example.com"}
    sendBuf, _ := json.Marshal(sendBody)
    resp, err := http.Post(srv.URL+"/api/auth/email-otp/send", "application/json", bytes.NewReader(sendBuf))
    if err != nil { t.Fatalf("send request error: %v", err) }
    if resp.StatusCode != 200 { t.Fatalf("expected 200 send, got %d", resp.StatusCode) }
    var sendOut map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&sendOut); err != nil { t.Fatalf("decode send: %v", err) }
    otpVal, ok := sendOut["dev_otp"].(string)
    if !ok || otpVal == "" { t.Fatalf("expected dev_otp string") }

    // Verify OTP
    verifyBody := map[string]any{"email": "verify.user@example.com", "otp": otpVal, "remember": true}
    verifyBuf, _ := json.Marshal(verifyBody)
    resp2, err := http.Post(srv.URL+"/api/auth/email-otp/verify", "application/json", bytes.NewReader(verifyBuf))
    if err != nil { t.Fatalf("verify request error: %v", err) }
    if resp2.StatusCode != 200 { t.Fatalf("expected 200 verify, got %d", resp2.StatusCode) }

    var verifyOut map[string]any
    if err := json.NewDecoder(resp2.Body).Decode(&verifyOut); err != nil { t.Fatalf("decode verify: %v", err) }
    if _, ok := verifyOut["token"].(string); !ok { t.Fatalf("expected token in verify response") }
    if _, ok := verifyOut["session"].(map[string]any); !ok { t.Fatalf("expected session in verify response") }
    if userMap, ok := verifyOut["user"].(map[string]any); !ok || userMap["Email"] != "verify.user@example.com" {
        t.Fatalf("expected user email verify.user@example.com, got %v", verifyOut["user"])
    }
}

func TestEmailOTP_VerifyCreatesUserIfMissing(t *testing.T) {
    _, mux := setupTestApp(t)
    srv := httptest.NewServer(mux)
    defer srv.Close()

    // Send OTP for a user that does not exist yet
    sendBody := map[string]any{"email": "new.user@example.com"}
    sendBuf, _ := json.Marshal(sendBody)
    resp, err := http.Post(srv.URL+"/api/auth/email-otp/send", "application/json", bytes.NewReader(sendBuf))
    if err != nil { t.Fatalf("send request error: %v", err) }
    if resp.StatusCode != 200 { t.Fatalf("expected 200 send, got %d", resp.StatusCode) }
    var sendOut map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&sendOut); err != nil { t.Fatalf("decode send: %v", err) }
    otpVal, ok := sendOut["dev_otp"].(string)
    if !ok || otpVal == "" { t.Fatalf("expected dev_otp string") }

    // Verify should implicitly create the user and return session
    verifyBody := map[string]any{"email": "new.user@example.com", "otp": otpVal, "remember": false}
    verifyBuf, _ := json.Marshal(verifyBody)
    resp2, err := http.Post(srv.URL+"/api/auth/email-otp/verify", "application/json", bytes.NewReader(verifyBuf))
    if err != nil { t.Fatalf("verify request error: %v", err) }
    if resp2.StatusCode != 200 { t.Fatalf("expected 200 verify, got %d", resp2.StatusCode) }
    var verifyOut map[string]any
    if err := json.NewDecoder(resp2.Body).Decode(&verifyOut); err != nil { t.Fatalf("decode verify: %v", err) }
    if _, ok := verifyOut["token"].(string); !ok { t.Fatalf("expected token in verify response") }
    if userMap, ok := verifyOut["user"].(map[string]any); !ok || userMap["Email"] != "new.user@example.com" {
        t.Fatalf("expected user email new.user@example.com, got %v", verifyOut["user"])
    }
}

func TestEmailOTP_VerifyRequiresExistingUserWhenImplicitDisabled(t *testing.T) {
    // Setup in-memory DB and routes with implicit signup disabled
    sqldb, err := sql.Open("sqlite3", ":memory:")
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    db := bun.NewDB(sqldb, sqlitedialect.New())

    ctx := context.Background()
    if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil { t.Fatalf("create users: %v", err) }
    if _, err := db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx); err != nil { t.Fatalf("create sessions: %v", err) }
    if _, err := db.NewCreateTable().Model((*schema.EmailOTP)(nil)).IfNotExists().Exec(ctx); err != nil { t.Fatalf("create email_otps: %v", err) }

    eotpr := repo.NewEmailOTPRepository(db)
    auditSvc := audit.NewService(repo.NewAuditRepository(db))
    webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)
    userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
    sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, webhookSvc)
    authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})
    // Construct service with AllowImplicitSignup=false
    svc := NewService(eotpr, userSvc, authSvc, auditSvc, nil, Config{DevExposeOTP: true, AllowImplicitSignup: false})

    mux := http.NewServeMux()
    app := forge.NewApp(mux)
    grp := app.Group("/api/auth")
    rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/email-otp/send": {Window: time.Minute, Max: 1000}}})
    h := NewHandler(svc, rls)
    grp.POST("/email-otp/send", h.Send)
    grp.POST("/email-otp/verify", h.Verify)
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

    srv := httptest.NewServer(mux)
    defer srv.Close()

    // Send OTP for non-existent user
    sendBody := map[string]any{"email": "no.user@example.com"}
    sendBuf, _ := json.Marshal(sendBody)
    resp, err := http.Post(srv.URL+"/api/auth/email-otp/send", "application/json", bytes.NewReader(sendBuf))
    if err != nil { t.Fatalf("send request error: %v", err) }
    if resp.StatusCode != 200 { t.Fatalf("expected 200 send, got %d", resp.StatusCode) }
    var sendOut map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&sendOut); err != nil { t.Fatalf("decode send: %v", err) }
    otpVal, ok := sendOut["dev_otp"].(string)
    if !ok || otpVal == "" { t.Fatalf("expected dev_otp string") }

    // Verify should fail with user not found
    verifyBody := map[string]any{"email": "no.user@example.com", "otp": otpVal, "remember": false}
    verifyBuf, _ := json.Marshal(verifyBody)
    resp2, err := http.Post(srv.URL+"/api/auth/email-otp/verify", "application/json", bytes.NewReader(verifyBuf))
    if err != nil { t.Fatalf("verify request error: %v", err) }
    if resp2.StatusCode != 400 { t.Fatalf("expected 400 verify, got %d", resp2.StatusCode) }
    var verifyOut map[string]any
    if err := json.NewDecoder(resp2.Body).Decode(&verifyOut); err != nil { t.Fatalf("decode verify: %v", err) }
    if msg, ok := verifyOut["error"].(string); !ok || msg != "user not found" {
        t.Fatalf("expected error 'user not found', got %v", verifyOut["error"])
    }
}

func TestEmailOTP_SendRespectsRateLimit(t *testing.T) {
    _, mux := setupTestApp(t)
    srv := httptest.NewServer(mux)
    defer srv.Close()

    email := "ratelimit.user@example.com"
    body := map[string]any{"email": email}

    // Hit the send endpoint 6 times; default rule allows 5/min
    for i := 1; i <= 6; i++ {
        buf, _ := json.Marshal(body)
        resp, err := http.Post(srv.URL+"/api/auth/email-otp/send", "application/json", bytes.NewReader(buf))
        if err != nil { t.Fatalf("send request error #%d: %v", i, err) }
        if i <= 5 {
            if resp.StatusCode != 200 { t.Fatalf("expected 200 on attempt %d, got %d", i, resp.StatusCode) }
        } else {
            if resp.StatusCode != 429 { t.Fatalf("expected 429 on attempt %d, got %d", i, resp.StatusCode) }
        }
    }
}