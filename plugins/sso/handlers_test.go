package sso

import (
    "bytes"
    "context"
    "database/sql"
    "encoding/base64"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "net/url"
    "testing"

    _ "github.com/mattn/go-sqlite3"
    "github.com/uptrace/bun"
    "github.com/uptrace/bun/dialect/sqlitedialect"
    "github.com/xraph/authsome/schema"
    "github.com/xraph/forge"
)

// setupTestApp initializes in-memory Bun DB, creates necessary tables, and mounts SSO routes
func setupTestAppSSO(t *testing.T) (*bun.DB, *http.ServeMux) {
    t.Helper()
    sqldb, err := sql.Open("sqlite3", ":memory:")
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    db := bun.NewDB(sqldb, sqlitedialect.New())

    ctx := context.Background()
    if _, err := db.NewCreateTable().Model((*schema.SSOProvider)(nil)).IfNotExists().Exec(ctx); err != nil { t.Fatalf("create sso_providers: %v", err) }

    // Initialize plugin
    p := NewPlugin()
    if err := p.Init(db); err != nil { t.Fatalf("plugin init: %v", err) }
    if err := p.Migrate(); err != nil { t.Fatalf("plugin migrate: %v", err) }

    mux := http.NewServeMux()
    app := forge.NewApp(mux)
    if err := p.RegisterRoutes(app); err != nil { t.Fatalf("register routes: %v", err) }
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
    return db, mux
}

func TestSSO_RegisterAndCallbacks(t *testing.T) {
    _, mux := setupTestAppSSO(t)
    srv := httptest.NewServer(mux)
    defer srv.Close()

    // Register OIDC provider
    body := map[string]any{"providerId": "google-oidc", "type": "oidc", "domain": "example.com", "OIDCClientID": "client", "OIDCIssuer": "https://accounts.google.com", "OIDCRedirectURI": "http://localhost/oidc/cb"}
    buf, _ := json.Marshal(body)
    resp, err := http.Post(srv.URL+"/api/auth/sso/provider/register", "application/json", bytes.NewReader(buf))
    if err != nil { t.Fatalf("register provider error: %v", err) }
    if resp.StatusCode != 200 { t.Fatalf("expected 200 register, got %d", resp.StatusCode) }

    // OIDC callback requires code and state (expects 400 due to fake credentials)
    resp2, err := http.Get(srv.URL+"/api/auth/sso/oidc/callback/google-oidc?code=abc123&state=test_state")
    if err != nil { t.Fatalf("oidc callback error: %v", err) }
    if resp2.StatusCode != 400 { t.Fatalf("expected 400 oidc callback (fake credentials), got %d", resp2.StatusCode) }

    // Register SAML provider
    body2 := map[string]any{"providerId": "okta-saml", "type": "saml", "domain": "example.com", "SAMLEntryPoint": "https://example.com/sso", "SAMLIssuer": "urn:okta"}
    buf2, _ := json.Marshal(body2)
    resp3, err := http.Post(srv.URL+"/api/auth/sso/provider/register", "application/json", bytes.NewReader(buf2))
    if err != nil { t.Fatalf("register provider saml error: %v", err) }
    if resp3.StatusCode != 200 { t.Fatalf("expected 200 register saml, got %d", resp3.StatusCode) }

    // SAML metadata
    resp4, err := http.Get(srv.URL+"/api/auth/sso/saml2/sp/metadata")
    if err != nil { t.Fatalf("saml md error: %v", err) }
    if resp4.StatusCode != 200 { t.Fatalf("expected 200 metadata, got %d", resp4.StatusCode) }
    var md map[string]string
    _ = json.NewDecoder(resp4.Body).Decode(&md)
    if md["metadata"] == "" { t.Fatalf("expected metadata string") }

    // Test SAML login initiation
    resp5, err := http.Get(srv.URL+"/api/auth/sso/saml2/login/okta-saml")
    if err != nil { t.Fatalf("saml login error: %v", err) }
    if resp5.StatusCode != 200 { t.Fatalf("expected 200 saml login, got %d", resp5.StatusCode) }
    
    var loginResp map[string]any
    _ = json.NewDecoder(resp5.Body).Decode(&loginResp)
    if loginResp["redirect_url"] == "" { t.Fatalf("expected redirect_url in login response") }
    if loginResp["provider_id"] != "okta-saml" { t.Fatalf("expected provider_id okta-saml") }

    // SAML callback
    // Minimal SAML Response XML with matching Issuer and NameID
    samlXML := "<Response xmlns=\"urn:oasis:names:tc:SAML:2.0:protocol\"><Issuer xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\">urn:okta</Issuer><Assertion xmlns=\"urn:oasis:names:tc:SAML:2.0:assertion\"><Subject><NameID>user@example.com</NameID></Subject></Assertion></Response>"
    samlB64 := base64.StdEncoding.EncodeToString([]byte(samlXML))
    vals := url.Values{}
    vals.Set("SAMLResponse", samlB64)
    form := bytes.NewBufferString(vals.Encode())
    req, _ := http.NewRequest("POST", srv.URL+"/api/auth/sso/saml2/callback/okta-saml", form)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    resp6, err := http.DefaultClient.Do(req)
    if err != nil { t.Fatalf("saml callback error: %v", err) }
    if resp6.StatusCode != 200 { t.Fatalf("expected 200 saml callback, got %d", resp6.StatusCode) }
}