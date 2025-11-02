// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/xraph/authsome"
	"github.com/xraph/forge"
)

// TestAuthFlow_Complete tests the complete authentication flow
func TestAuthFlow_Complete(t *testing.T) {
	// Setup test database
	db := setupTestDatabase(t)
	defer db.Close()

	// Setup AuthSome with test configuration
	app := forge.New()
	auth := authsome.New(
		authsome.WithDatabase(db),
		authsome.WithForgeApp(app),
		authsome.WithMode(authsome.ModeStandalone),
	)

	err := auth.Initialize(context.Background())
	require.NoError(t, err, "Failed to initialize AuthSome")

	err = auth.Mount(app.Router(), "/auth")
	require.NoError(t, err, "Failed to mount AuthSome")

	// Create test server
	server := httptest.NewServer(app)
	defer server.Close()

	client := server.Client()
	baseURL := server.URL

	// Test data
	email := "integration-test@example.com"
	password := "SecureTestPass123!"
	name := "Integration Test User"

	// Step 1: User Registration
	t.Run("User Registration", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    email,
			"password": password,
			"name":     name,
		}
		body, _ := json.Marshal(payload)

		resp, err := client.Post(baseURL+"/auth/register", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Registration should succeed")

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.NotNil(t, result["user"], "Response should contain user")
		assert.NotNil(t, result["token"], "Response should contain token")
		
		user := result["user"].(map[string]interface{})
		assert.Equal(t, email, user["email"], "User email should match")
		assert.Equal(t, name, user["name"], "User name should match")
	})

	// Step 2: Duplicate Registration (should fail)
	t.Run("Duplicate Registration", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    email,
			"password": password,
			"name":     name,
		}
		body, _ := json.Marshal(payload)

		resp, err := client.Post(baseURL+"/auth/register", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Duplicate registration should fail")
	})

	// Step 3: User Login
	var sessionToken string
	t.Run("User Login", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    email,
			"password": password,
		}
		body, _ := json.Marshal(payload)

		resp, err := client.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Login should succeed")

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.NotNil(t, result["user"], "Response should contain user")
		assert.NotNil(t, result["token"], "Response should contain token")
		
		sessionToken = result["token"].(string)
		assert.NotEmpty(t, sessionToken, "Session token should not be empty")
	})

	// Step 4: Login with wrong password (should fail)
	t.Run("Invalid Login", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    email,
			"password": "WrongPassword123!",
		}
		body, _ := json.Marshal(payload)

		resp, err := client.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Login with wrong password should fail")
	})

	// Step 5: Get Current User (with valid session)
	t.Run("Get Current User", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseURL+"/auth/me", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+sessionToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Get user should succeed")

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, email, result["email"], "User email should match")
		assert.Equal(t, name, result["name"], "User name should match")
	})

	// Step 6: Get Current User (without auth - should fail)
	t.Run("Unauthorized Access", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/auth/me")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Unauthorized access should fail")
	})

	// Step 7: Update User Profile
	t.Run("Update User Profile", func(t *testing.T) {
		newName := "Updated Test User"
		payload := map[string]interface{}{
			"name": newName,
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest("PUT", baseURL+"/auth/me", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+sessionToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Update profile should succeed")

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, newName, result["name"], "User name should be updated")
	})

	// Step 8: User Logout
	t.Run("User Logout", func(t *testing.T) {
		payload := map[string]interface{}{}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", baseURL+"/auth/logout", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+sessionToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Logout should succeed")
	})

	// Step 9: Use session after logout (should fail)
	t.Run("Session After Logout", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseURL+"/auth/me", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+sessionToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Session should be invalid after logout")
	})
}

// TestAuthFlow_RateLimiting tests rate limiting
func TestAuthFlow_RateLimiting(t *testing.T) {
	t.Skip("Implement rate limiting tests")
	// TODO: Test rate limiting on login attempts
	// TODO: Test rate limiting on registration
	// TODO: Test rate limiting on API endpoints
}

// TestAuthFlow_Concurrency tests concurrent authentication requests
func TestAuthFlow_Concurrency(t *testing.T) {
	t.Skip("Implement concurrency tests")
	// TODO: Test concurrent registrations
	// TODO: Test concurrent logins
	// TODO: Test concurrent session operations
}

// setupTestDatabase creates an in-memory SQLite database for testing
func setupTestDatabase(t *testing.T) *bun.DB {
	sqldb, err := sqliteshim.Open(":memory:")
	require.NoError(t, err, "Failed to open in-memory database")

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Run migrations
	ctx := context.Background()
	// TODO: Run actual migrations
	// For now, create basic tables manually
	
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			username TEXT,
			display_username TEXT,
			image TEXT,
			verified BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err, "Failed to create users table")

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			token TEXT UNIQUE NOT NULL,
			user_id TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	require.NoError(t, err, "Failed to create sessions table")

	return db
}

