package pagination_test

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/core/pagination"
)

// User represents a sample user model
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Example demonstrates basic offset-based pagination
func Example_offsetPagination() {
	// Simulate request parameters
	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			SortBy: "created_at",
			Order:  pagination.SortOrderDesc,
		},
		Limit: 10,
		Page:  1,
	}

	// Validate parameters
	if err := params.Validate(); err != nil {
		fmt.Printf("Validation error: %v\n", err)
		return
	}

	// Simulate fetching users from database
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
		{ID: "2", Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
	}
	total := int64(25)

	// Create paginated response
	response := pagination.NewPageResponse(users, total, params)

	fmt.Printf("Current page: %d\n", response.Pagination.CurrentPage)
	fmt.Printf("Total pages: %d\n", response.Pagination.TotalPages)
	fmt.Printf("Has next: %v\n", response.Pagination.HasNext)
	fmt.Printf("Total items: %d\n", response.Pagination.Total)
	// Output:
	// Current page: 1
	// Total pages: 3
	// Has next: true
	// Total items: 25
}

// Example demonstrates cursor-based pagination
func Example_cursorPagination() {
	params := &pagination.CursorParams{
		BaseRequestParams: pagination.BaseRequestParams{
			Order: pagination.SortOrderDesc,
		},
		Limit: 10,
	}

	if err := params.Validate(); err != nil {
		fmt.Printf("Validation error: %v\n", err)
		return
	}

	// Simulate fetching posts
	posts := []User{
		{ID: "1", Name: "Post 1", CreatedAt: time.Now()},
		{ID: "2", Name: "Post 2", CreatedAt: time.Now()},
	}

	// Generate next cursor
	nextCursor, _ := pagination.EncodeCursor("2", time.Now(), "")

	// Create response
	response := pagination.NewCursorResponse(posts, nextCursor, "", params)

	fmt.Printf("Item count: %d\n", response.Cursor.Count)
	fmt.Printf("Has next: %v\n", response.Cursor.HasNext)
	fmt.Printf("Has cursor: %v\n", response.Cursor.NextCursor != "")
	// Output:
	// Item count: 2
	// Has next: true
	// Has cursor: true
}

// Example demonstrates parameter validation with defaults
func Example_parameterValidation() {
	// Parameters with defaults
	params := &pagination.PaginationParams{}

	if err := params.Validate(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Limit: %d\n", params.GetLimit())
	fmt.Printf("Page: %d\n", params.GetPage())
	fmt.Printf("Order: %s\n", params.GetOrder())
	// Output:
	// Limit: 10
	// Page: 1
	// Order: desc
}

// Example demonstrates field selection
func Example_fieldSelection() {
	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			Fields: "id, name, email",
		},
		Limit: 10,
	}

	fields := params.GetFields()
	fmt.Printf("Selected fields: %v\n", fields)
	fmt.Printf("Has fields: %v\n", params.HasFields())
	// Output:
	// Selected fields: [id name email]
	// Has fields: true
}

// Example demonstrates SQL ORDER BY clause generation
func Example_sqlOrderClause() {
	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			SortBy: "name",
			Order:  pagination.SortOrderAsc,
		},
	}

	orderClause := params.GetOrderClause()
	fmt.Printf("ORDER BY %s\n", orderClause)
	// Output:
	// ORDER BY name ASC
}

// Example demonstrates cursor encoding and decoding
func Example_cursorEncoding() {
	// Encode cursor
	id := "user_123"
	timestamp := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	value := "alice"

	encoded, err := pagination.EncodeCursor(id, timestamp, value)
	if err != nil {
		fmt.Printf("Encoding error: %v\n", err)
		return
	}

	// Decode cursor
	decoded, err := pagination.DecodeCursor(encoded)
	if err != nil {
		fmt.Printf("Decoding error: %v\n", err)
		return
	}

	fmt.Printf("ID: %s\n", decoded.ID)
	fmt.Printf("Value: %s\n", decoded.Value)
	// Output:
	// ID: user_123
	// Value: alice
}

// Example demonstrates empty response handling
func Example_emptyResponse() {
	response := pagination.NewEmptyPageResponse[User]()

	fmt.Printf("Data count: %d\n", len(response.Data))
	fmt.Printf("Total: %d\n", response.Pagination.Total)
	fmt.Printf("Has next: %v\n", response.Pagination.HasNext)
	// Output:
	// Data count: 0
	// Total: 0
	// Has next: false
}

// ExamplePaginationParams_GetOffset demonstrates offset calculation
func ExamplePaginationParams_GetOffset() {
	// Using page number
	params := &pagination.PaginationParams{
		Limit: 10,
		Page:  3,
	}

	offset := params.GetOffset()
	fmt.Printf("Page 3 offset: %d\n", offset)
	// Output:
	// Page 3 offset: 20
}

// ExamplePaginationParams_GetPage demonstrates page calculation
func ExamplePaginationParams_GetPage() {
	// Using offset
	params := &pagination.PaginationParams{
		Limit:  10,
		Offset: 50,
	}

	page := params.GetPage()
	fmt.Printf("Offset 50 page: %d\n", page)
	// Output:
	// Offset 50 page: 6
}

// Example_handlerIntegration demonstrates integration with a handler
func Example_handlerIntegration() {
	// This example shows typical handler usage

	// 1. Bind query parameters
	params := &pagination.PaginationParams{
		BaseRequestParams: pagination.BaseRequestParams{
			SortBy: "name",
			Order:  pagination.SortOrderAsc,
		},
		Limit: 20,
		Page:  2,
	}

	// 2. Validate
	if err := params.Validate(); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}

	// 3. Query database (simulated)
	users, total := queryUsers(context.Background(), params)

	// 4. Create response
	response := pagination.NewPageResponse(users, total, params)

	// 5. Return response
	fmt.Printf("Showing page %d of %d\n", response.Pagination.CurrentPage, response.Pagination.TotalPages)
	fmt.Printf("Items: %d-%d of %d\n",
		response.Pagination.Offset+1,
		response.Pagination.Offset+len(response.Data),
		response.Pagination.Total,
	)
	// Output:
	// Showing page 2 of 3
	// Items: 21-40 of 50
}

// queryUsers simulates a database query
func queryUsers(ctx context.Context, params *pagination.PaginationParams) ([]User, int64) {
	// In real code, this would query your database
	// using params.GetLimit(), params.GetOffset(), etc.

	users := make([]User, 20) // Simulate 20 results
	for i := range users {
		users[i] = User{
			ID:        fmt.Sprintf("user_%d", params.GetOffset()+i+1),
			Name:      fmt.Sprintf("User %d", params.GetOffset()+i+1),
			Email:     fmt.Sprintf("user%d@example.com", params.GetOffset()+i+1),
			CreatedAt: time.Now(),
		}
	}

	return users, 50 // Simulate 50 total users
}
