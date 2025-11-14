package pagination_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/xraph/authsome/core/pagination"
)

// Test model
type TestUser struct {
	bun.BaseModel `bun:"table:test_users"`

	ID        string    `bun:"id,pk"`
	Name      string    `bun:"name"`
	Email     string    `bun:"email"`
	Status    string    `bun:"status"`
	Role      string    `bun:"role"`
	CreatedAt time.Time `bun:"created_at"`
}

func setupTestDB(t *testing.T) *bun.DB {
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create table
	_, err = db.NewCreateTable().Model((*TestUser)(nil)).IfNotExists().Exec(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Insert test data
	users := []TestUser{
		{ID: "1", Name: "Alice", Email: "alice@example.com", Status: "active", Role: "admin", CreatedAt: time.Now().Add(-3 * time.Hour)},
		{ID: "2", Name: "Bob", Email: "bob@example.com", Status: "active", Role: "user", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{ID: "3", Name: "Charlie", Email: "charlie@example.com", Status: "inactive", Role: "user", CreatedAt: time.Now().Add(-1 * time.Hour)},
		{ID: "4", Name: "David", Email: "david@example.com", Status: "active", Role: "moderator", CreatedAt: time.Now()},
	}

	_, err = db.NewInsert().Model(&users).Exec(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestApplyToQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			SortBy: "name",
			Order:  pagination.SortOrderAsc,
		},
		Limit: 2,
		Page:  1,
	}

	if err := params.Validate(); err != nil {
		t.Fatal(err)
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)

	qb := pagination.NewQueryBuilder(params)
	query = qb.ApplyToQuery(query)

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	// Check order (should be ascending by name)
	if len(users) > 0 && users[0].Name != "Alice" {
		t.Errorf("Expected first user to be Alice, got %s", users[0].Name)
	}
}

func TestApplyLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		Limit: 2,
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)

	qb := pagination.NewQueryBuilder(params)
	query = qb.ApplyLimit(query)

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestApplyOffset(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			SortBy: "id",
			Order:  pagination.SortOrderAsc,
		},
		Offset: 1,
		Limit:  2,
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)

	qb := pagination.NewQueryBuilder(params)
	query = qb.ApplyToQuery(query)

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	// First user should be ID "2" (offset 1)
	if len(users) > 0 && users[0].ID != "2" {
		t.Errorf("Expected first user ID to be 2, got %s", users[0].ID)
	}
}

func TestApplyOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name          string
		sortBy        string
		order         pagination.SortOrder
		expectedFirst string
	}{
		{
			name:          "ascending by name",
			sortBy:        "name",
			order:         pagination.SortOrderAsc,
			expectedFirst: "Alice",
		},
		{
			name:          "descending by name",
			sortBy:        "name",
			order:         pagination.SortOrderDesc,
			expectedFirst: "David",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &pagination.PaginationParams{
				BaseRequestParams: pagination.BaseRequestParams{
					SortBy: tt.sortBy,
					Order:  tt.order,
				},
			}

			var users []TestUser
			query := db.NewSelect().Model(&users)

			qb := pagination.NewQueryBuilder(params)
			query = qb.ApplyOrder(query)

			err := query.Scan(context.Background())
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if len(users) == 0 {
				t.Fatal("No users returned")
			}

			if users[0].Name != tt.expectedFirst {
				t.Errorf("Expected first user to be %s, got %s", tt.expectedFirst, users[0].Name)
			}
		})
	}
}

func TestApplySearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			Search: "alice",
		},
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)

	qb := pagination.NewQueryBuilder(params)
	query = qb.ApplySearch(query, "name", "email")

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	if len(users) > 0 && users[0].Name != "Alice" {
		t.Errorf("Expected Alice, got %s", users[0].Name)
	}
}

func TestApplyFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			Filter: "status:active,role:admin",
		},
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)

	qb := pagination.NewQueryBuilder(params)
	filters := pagination.ParseFilters(params.Filter)
	query = qb.ApplyFilters(query, filters)

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	if len(users) > 0 && users[0].Name != "Alice" {
		t.Errorf("Expected Alice, got %s", users[0].Name)
	}
}

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "single filter",
			input: "status:active",
			expected: map[string]string{
				"status": "active",
			},
		},
		{
			name:  "multiple filters",
			input: "status:active,role:admin",
			expected: map[string]string{
				"status": "active",
				"role":   "admin",
			},
		},
		{
			name:  "with spaces",
			input: "status: active , role: admin",
			expected: map[string]string{
				"status": "active",
				"role":   "admin",
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pagination.ParseFilters(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d filters, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Expected key %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}

func TestApplyHelperFunction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			SortBy: "name",
			Order:  pagination.SortOrderAsc,
		},
		Limit: 2,
		Page:  1,
	}

	if err := params.Validate(); err != nil {
		t.Fatal(err)
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)
	query = pagination.Apply(query, params)

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestApplyWithSearchHelperFunction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			Search: "bob",
		},
		Limit: 10,
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)
	query = pagination.ApplyWithSearch(query, params, "name", "email")

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	if len(users) > 0 && users[0].Name != "Bob" {
		t.Errorf("Expected Bob, got %s", users[0].Name)
	}
}

func TestApplyFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			Fields: "id,name",
		},
		Limit: 10,
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)

	qb := pagination.NewQueryBuilder(params)
	query = qb.ApplyFields(query)

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Should have data
	if len(users) == 0 {
		t.Fatal("No users returned")
	}

	// Check that only requested fields are populated
	// Note: id and name should be populated, email should be empty
	for _, user := range users {
		if user.ID == "" {
			t.Error("ID should be populated")
		}
		if user.Name == "" {
			t.Error("Name should be populated")
		}
		if user.Email != "" {
			t.Error("Email should not be populated (not in fields list)")
		}
	}
}

func TestGetFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   string
		expected []string
	}{
		{
			name:     "single field",
			fields:   "id",
			expected: []string{"id"},
		},
		{
			name:     "multiple fields",
			fields:   "id,name,email",
			expected: []string{"id", "name", "email"},
		},
		{
			name:     "fields with spaces",
			fields:   "id, name, email",
			expected: []string{"id", "name", "email"},
		},
		{
			name:     "empty string",
			fields:   "",
			expected: nil,
		},
		{
			name:     "only spaces",
			fields:   "  ,  ,  ",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &pagination.BaseRequestParams{
				Fields: tt.fields,
			}

			result := params.GetFields()

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d fields, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Field %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestApplyAllHelperFunction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			Search: "active",
			Filter: "role:user",
			SortBy: "name",
			Order:  pagination.SortOrderAsc,
		},
		Limit: 10,
	}

	var users []TestUser
	query := db.NewSelect().Model(&users)
	query = pagination.ApplyAll(query, params, "status")

	err := query.Scan(context.Background())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Should find Bob and Charlie (both role:user)
	// Search "active" matches both "active" and "inactive" (substring match)
	// Filter role:user excludes Alice (admin) and David (moderator)
	if len(users) != 2 {
		t.Errorf("Expected 2 users (Bob and Charlie), got %d", len(users))
		for _, u := range users {
			t.Logf("User: %s, Status: %s, Role: %s", u.Name, u.Status, u.Role)
		}
	}

	// Verify we got Bob and Charlie
	if len(users) == 2 {
		if users[0].Name != "Bob" && users[1].Name != "Bob" {
			t.Error("Expected to find Bob")
		}
		if users[0].Name != "Charlie" && users[1].Name != "Charlie" {
			t.Error("Expected to find Charlie")
		}
	}
}
