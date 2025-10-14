package multisession

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
    "github.com/xraph/forge"
    "github.com/xraph/authsome/schema"
)

// setupTestApp initializes in-memory Bun DB, creates necessary tables, and mounts multisession routes
func setupTestAppMS(t *testing.T) (*bun.DB, *http.ServeMux, *Plugin) {
    t.Helper()
    sqldb, err := sql.Open("sqlite3", ":memory:")
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    db := bun.NewDB(sqldb, sqlitedialect.New())

    ctx := context.Background()
    if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil { t.Fatalf("create users: %v", err) }
    if _, err := db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx); err != nil { t.Fatalf("create sessions: %v", err) }

    // Initialize plugin
    p := NewPlugin()
    if err := p.Init(db); err != nil { t.Fatalf("plugin init: %v", err) }
    if err := p.Migrate(); err != nil { t.Fatalf("plugin migrate: %v", err) }

    mux := http.NewServeMux()
    app := forge.NewApp(mux)
    if err := p.RegisterRoutes(app); err != nil { t.Fatalf("register routes: %v", err) }
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
    return db, mux, p
}

func TestMultiSession_ListSetActiveDelete(t *testing.T) {
    db, mux, _ := setupTestAppMS(t)
    srv := httptest.NewServer(mux)
    defer srv.Close()

    // Create user and sessions directly in database
    ctx := context.Background()
    
    // Create user
    systemID := xid.New() // System user ID for audit fields
    user := &schema.User{
        ID:           xid.New(),
        Email:        "ms.user@example.com",
        Name:         "MS User",
        PasswordHash: "$2a$10$example.hash", // dummy hash
    }
    user.CreatedBy = systemID
    user.UpdatedBy = systemID
    if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
        t.Fatalf("create user: %v", err)
    }
    
    // Create two sessions
    sess1 := &schema.Session{
        ID:        xid.New(),
        UserID:    user.ID,
        Token:     "test-token-1",
        ExpiresAt: time.Now().Add(24 * time.Hour),
        IPAddress: "127.0.0.1",
        UserAgent: "ua-1",
    }
    sess1.CreatedBy = systemID
    sess1.UpdatedBy = systemID
    sess2 := &schema.Session{
        ID:        xid.New(),
        UserID:    user.ID,
        Token:     "test-token-2",
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

    // List sessions for user using cookie from first session
    reqList, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/multi-session/list", nil)
    reqList.AddCookie(&http.Cookie{Name: "session_token", Value: sess1.Token})
    respList, err := http.DefaultClient.Do(reqList)
    if err != nil { t.Fatalf("list request error: %v", err) }
    if respList.StatusCode != 200 { t.Fatalf("expected 200, got %d", respList.StatusCode) }
    var listOut map[string]any
    if err := json.NewDecoder(respList.Body).Decode(&listOut); err != nil { t.Fatalf("decode list: %v", err) }
    sessionsVal, ok := listOut["sessions"].([]any)
    if !ok || len(sessionsVal) != 2 { t.Fatalf("expected 2 sessions, got %v", len(sessionsVal)) }

    // Set active to second session
    body := map[string]any{"id": sess2.ID.String()}
    buf, _ := json.Marshal(body)
    reqSet, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/multi-session/set-active", bytes.NewReader(buf))
    reqSet.Header.Set("Content-Type", "application/json")
    reqSet.AddCookie(&http.Cookie{Name: "session_token", Value: sess1.Token})
    respSet, err := http.DefaultClient.Do(reqSet)
    if err != nil { t.Fatalf("set-active request error: %v", err) }
    if respSet.StatusCode != 200 { t.Fatalf("expected 200 set-active, got %d", respSet.StatusCode) }
    // Expect Set-Cookie header to include new token
    sc := respSet.Header.Get("Set-Cookie")
    if sc == "" || !containsCookieWithToken(sc, sess2.Token) { t.Fatalf("expected Set-Cookie with new token, got %q", sc) }
    var setOut map[string]any
    if err := json.NewDecoder(respSet.Body).Decode(&setOut); err != nil { t.Fatalf("decode set-active: %v", err) }
    if tok, ok := setOut["token"].(string); !ok || tok != sess2.Token { t.Fatalf("expected token %s, got %v", sess2.Token, tok) }

    // Delete first session via path param
    reqDel, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/multi-session/delete/"+sess1.ID.String(), nil)
    reqDel.AddCookie(&http.Cookie{Name: "session_token", Value: sess2.Token})
    respDel, err := http.DefaultClient.Do(reqDel)
    if err != nil { t.Fatalf("delete request error: %v", err) }
    if respDel.StatusCode != 200 { t.Fatalf("expected 200 delete, got %d", respDel.StatusCode) }

    // List again should show 1 session
    reqList2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/multi-session/list", nil)
    reqList2.AddCookie(&http.Cookie{Name: "session_token", Value: sess2.Token})
    respList2, err := http.DefaultClient.Do(reqList2)
    if err != nil { t.Fatalf("list2 request error: %v", err) }
    var listOut2 map[string]any
    if err := json.NewDecoder(respList2.Body).Decode(&listOut2); err != nil { t.Fatalf("decode list2: %v", err) }
    sessionsVal2, ok := listOut2["sessions"].([]any)
    if !ok || len(sessionsVal2) != 1 { t.Fatalf("expected 1 session after delete, got %v", len(sessionsVal2)) }
}

// containsCookieWithToken checks if Set-Cookie header contains session_token=<token>
func containsCookieWithToken(sc, token string) bool {
    want := "session_token=" + token
    return sc != "" && (sc == want || (len(sc) > len(want) && sc[:len(want)] == want))
}