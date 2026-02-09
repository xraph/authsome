package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

func main() {

	// Setup database
	sqldb, err := sql.Open("sqlite3", "file:integration_test.db?mode=memory&cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer sqldb.Close()

	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()

	// Create tables
	createTables(db, ctx)

	// Create test organization
	orgID := createTestOrganization(db, ctx, "acme-corp")

	// Start HTTP server with multi-tenancy
	app := forge.NewApp(forge.AppConfig{
		Name:        "test-multitenancy-integration",
		Version:     "1.0.0",
		Environment: "development",
		HTTPAddress: ":3003",
	})

	auth := authsome.New(
		authsome.WithDatabase(db),
		authsome.WithForgeApp(app),
		authsome.WithBasePath("/api/auth"),
	)

	// Note: Multi-tenancy plugin would be registered here if available
	// mtPlugin := multitenancy.NewPlugin()
	// if err := auth.RegisterPlugin(mtPlugin); err != nil {
	// 	log.Fatalf("Failed to register plugin: %v", err)
	// }

	// Initialize
	if err := auth.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Mount routes
	if err := auth.Mount(app.Router(), "/api/auth"); err != nil {
		log.Fatalf("Failed to mount: %v", err)
	}

	// Start server in background
	go func() {
		if err := app.Run(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Run integration tests

	// Test 1: Signup with org context
	testSignupWithOrg(orgID)

	// Test 2: Signup without org context
	testSignupWithoutOrg()

	// Test 3: Verify decorator logging

	// TODO: Implement graceful shutdown for forge.App
	// Server will be terminated when the program exits
}

func testSignupWithOrg(orgID string) {

	payload := map[string]interface{}{
		"email":    "user@acme.com",
		"password": "securepass123",
		"name":     "Acme User",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "http://localhost:18080/api/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-ID", orgID)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {

		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {

		var result map[string]interface{}
		json.Unmarshal(respBody, &result)
		if _, ok := result["user"].(map[string]interface{}); ok {
			// User data available
		}
	} else {
		if resp.StatusCode == 400 {

		}
	}
}

func testSignupWithoutOrg() {

	payload := map[string]interface{}{
		"email":    "noorg@example.com",
		"password": "securepass123",
		"name":     "No Org User",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "http://localhost:18080/api/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Organization-ID header

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {

		return
	}
	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		// Error handling
	} else {
		// Success handling
	}
}

func createTables(db *bun.DB, ctx context.Context) {
	tables := []interface{}{
		(*schema.User)(nil),
		(*schema.Session)(nil),
		(*schema.Organization)(nil),
		(*schema.Member)(nil),
		(*schema.Team)(nil),
		(*schema.TeamMember)(nil),
		(*schema.Invitation)(nil),
		(*schema.AuditEvent)(nil),
	}

	db.RegisterModel((*schema.TeamMember)(nil))
	db.RegisterModel((*schema.OrganizationTeamMember)(nil))
	db.RegisterModel((*schema.RolePermission)(nil))
	db.RegisterModel((*schema.APIKeyRole)(nil))

	for _, model := range tables {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		if err != nil {
			log.Printf("Warning: Failed to create table for %T: %v", model, err)
		}
	}
}

func createTestOrganization(db *bun.DB, ctx context.Context, slug string) string {
	orgID := xid.New()

	org := &schema.Organization{
		ID:   orgID,
		Name: "Acme Corporation",
		Slug: slug,
	}

	_, err := db.NewInsert().Model(org).Exec(ctx)
	if err != nil {
		log.Printf("Warning: Failed to create test org: %v", err)
	}

	return orgID.String()
}
