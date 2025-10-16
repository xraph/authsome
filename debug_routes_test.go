package authsome

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/xraph/authsome/plugins/oidcprovider"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

func TestDebugRoutes(t *testing.T) {
	// Setup database
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*schema.OAuthClient)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create oauth_clients: %v", err)
	}

	// Setup plugin
	p := oidcprovider.NewPlugin()
	if err := p.Init(db); err != nil {
		t.Fatalf("plugin init: %v", err)
	}
	if err := p.Migrate(); err != nil {
		t.Fatalf("plugin migrate: %v", err)
	}

	// Setup routes
	mux := http.NewServeMux()
	app := forge.NewApp(mux)
	if err := p.RegisterRoutes(app); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	// Test server
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Test GET endpoints
	getEndpoints := []string{
		"/oauth2/authorize",
		"/oauth2/userinfo",
		"/oauth2/jwks",
	}

	for _, endpoint := range getEndpoints {
		resp, err := http.Get(srv.URL + endpoint)
		if err != nil {
			t.Logf("Error accessing GET %s: %v", endpoint, err)
			continue
		}
		t.Logf("GET %s returned status: %d", endpoint, resp.StatusCode)
		resp.Body.Close()
	}

	// Test POST endpoints
	postEndpoints := []string{
		"/oauth2/register",
		"/oauth2/token",
		"/oauth2/consent",
	}

	for _, endpoint := range postEndpoints {
		resp, err := http.Post(srv.URL+endpoint, "application/json", nil)
		if err != nil {
			t.Logf("Error accessing POST %s: %v", endpoint, err)
			continue
		}
		t.Logf("POST %s returned status: %d", endpoint, resp.StatusCode)
		resp.Body.Close()
	}
}
