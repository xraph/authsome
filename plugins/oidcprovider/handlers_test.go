package oidcprovider

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// setupTestApp initializes in-memory Bun DB, creates necessary tables, and mounts OIDC Provider routes
func setupTestAppOIDCP(t *testing.T) (*bun.DB, *http.ServeMux, string) {
	t.Helper()
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	// Create all required tables
	if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create sessions: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.OAuthClient)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create oauth_clients: %v", err)
	}

	p := NewPlugin()
	if err := p.Init(db); err != nil {
		t.Fatalf("plugin init: %v", err)
	}
	if err := p.Migrate(); err != nil {
		t.Fatalf("plugin migrate: %v", err)
	}

	// Create a test user and session
	userID := xid.New()
	testUser := &schema.User{
		ID:            userID,
		Email:         "test@example.com",
		EmailVerified: true,
	}
	testUser.CreatedBy = userID // Self-created for test
	testUser.UpdatedBy = userID
	if _, err := db.NewInsert().Model(testUser).Exec(ctx); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	sessionID := xid.New()
	testSession := &schema.Session{
		ID:        sessionID,
		UserID:    testUser.ID,
		Token:     "test-session-token",
		ExpiresAt: time.Now().Add(time.Hour), // 1 hour from now
	}
	testSession.CreatedBy = userID
	testSession.UpdatedBy = userID
	if _, err := db.NewInsert().Model(testSession).Exec(ctx); err != nil {
		t.Fatalf("create test session: %v", err)
	}

	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	if err := p.RegisterRoutes(app); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	return db, mux, testSession.Token
}

func TestOIDCProvider_Flow(t *testing.T) {
	_, mux, sessionToken := setupTestAppOIDCP(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Register client
	reg := map[string]string{"name": "test-app", "redirect_uri": "http://localhost/cb"}
	regBuf, _ := json.Marshal(reg)
	resp, err := http.Post(srv.URL+"/oauth2/register", "application/json", bytes.NewReader(regBuf))
	if err != nil {
		t.Fatalf("register client error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 register, got %d", resp.StatusCode)
	}
	var out map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&out)
	cid := out["client_id"]
	if cid == "" {
		t.Fatalf("expected client_id")
	}

	// Authorize (with session cookie) - expect redirect
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}
	req, err := http.NewRequest("GET", srv.URL+"/oauth2/authorize?client_id="+cid+"&redirect_uri=http://localhost/cb&response_type=code&state=xyz", nil)
	if err != nil {
		t.Fatalf("create authorize request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("authorize error: %v", err)
	}
	if resp2.StatusCode != 302 {
		t.Fatalf("expected 302 authorize redirect, got %d", resp2.StatusCode)
	}

	// Extract code from redirect URL
	location := resp2.Header.Get("Location")
	if location == "" {
		t.Fatalf("expected Location header")
	}
	redirectURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	code := redirectURL.Query().Get("code")
	if code == "" {
		t.Fatalf("expected code in redirect URL")
	}

	// Token
	tokReq := map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"client_id":    cid,
		"redirect_uri": "http://localhost/cb",
	}
	tokBuf, _ := json.Marshal(tokReq)
	resp3, err := http.Post(srv.URL+"/oauth2/token", "application/json", bytes.NewReader(tokBuf))
	if err != nil {
		t.Fatalf("token error: %v", err)
	}
	if resp3.StatusCode != 200 {
		body := make([]byte, 1024)
		n, _ := resp3.Body.Read(body)
		t.Fatalf("expected 200 token, got %d: %s", resp3.StatusCode, string(body[:n]))
	}

	// Extract access token from response
	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp3.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("failed to decode token response: %v", err)
	}
	accessToken, ok := tokenResp["access_token"].(string)
	if !ok {
		t.Fatalf("access_token not found in response")
	}

	// UserInfo
	req4, _ := http.NewRequest("GET", srv.URL+"/oauth2/userinfo", nil)
	req4.Header.Set("Authorization", "Bearer "+accessToken)
	resp4, err := http.DefaultClient.Do(req4)
	if err != nil {
		t.Fatalf("userinfo error: %v", err)
	}
	if resp4.StatusCode != 200 {
		t.Fatalf("expected 200 userinfo, got %d", resp4.StatusCode)
	}

	// JWKS
	resp5, err := http.Get(srv.URL + "/oauth2/jwks")
	if err != nil {
		t.Fatalf("jwks error: %v", err)
	}
	if resp5.StatusCode != 200 {
		t.Fatalf("expected 200 jwks, got %d", resp5.StatusCode)
	}
}
