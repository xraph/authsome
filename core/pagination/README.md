# Pagination Package

A comprehensive, production-ready pagination system for the AuthSome framework with support for both offset-based and cursor-based pagination patterns.

## Features

- ✅ **Offset-based pagination** (page/limit/offset)
- ✅ **Cursor-based pagination** (for large datasets)
- ✅ **Sorting & ordering** (ASC/DESC)
- ✅ **Search & filtering** support
- ✅ **Full metadata** (total count, pages, has_next/prev)
- ✅ **Forge DTO tags** compliance
- ✅ **Validation** with sensible defaults
- ✅ **Helper methods** for common operations
- ✅ **Type-safe generics** for responses
- ✅ **Base64 cursor encoding/decoding**

## Installation

The package is part of the AuthSome core:

```go
import "github.com/xraph/authsome/core/pagination"
```

## Quick Start

### Offset-Based Pagination

```go
// In your handler
func (h *Handler) ListUsers(c *forge.Context) error {
    var params pagination.PaginationParams
    if err := c.BindQuery(&params); err != nil {
        return c.JSON(400, ErrorResponse{Message: "invalid parameters"})
    }

    // Validate parameters
    if err := params.Validate(); err != nil {
        return c.JSON(400, ErrorResponse{Message: err.Error()})
    }

    // Fetch data from repository
    users, total, err := h.userRepo.List(
        c.Context(),
        params.GetLimit(),
        params.GetOffset(),
        params.GetSortBy(),
        params.GetOrder(),
    )
    if err != nil {
        return c.JSON(500, ErrorResponse{Message: "failed to fetch users"})
    }

    // Create response with metadata
    response := pagination.NewPageResponse(users, total, &params)
    return c.JSON(200, response)
}
```

### Cursor-Based Pagination

```go
func (h *Handler) ListPosts(c *forge.Context) error {
    var params pagination.CursorParams
    if err := c.BindQuery(&params); err != nil {
        return c.JSON(400, ErrorResponse{Message: "invalid parameters"})
    }

    if err := params.Validate(); err != nil {
        return c.JSON(400, ErrorResponse{Message: err.Error()})
    }

    // Decode cursor
    var cursorData *pagination.CursorData
    if params.Cursor != "" {
        var err error
        cursorData, err = pagination.DecodeCursor(params.Cursor)
        if err != nil {
            return c.JSON(400, ErrorResponse{Message: "invalid cursor"})
        }
    }

    // Fetch data
    posts, nextCursor, prevCursor, err := h.postRepo.ListCursor(
        c.Context(),
        params.GetLimit()+1, // Fetch one extra to check if there's more
        cursorData,
        params.GetSortBy(),
        params.GetOrder(),
    )
    if err != nil {
        return c.JSON(500, ErrorResponse{Message: "failed to fetch posts"})
    }

    response := pagination.NewCursorResponse(posts, nextCursor, prevCursor, &params)
    return c.JSON(200, response)
}
```

## API Reference

### Types

#### PaginationParams

Request parameters for offset-based pagination:

| Field    | Type      | JSON Tag | Query Param | Default      | Validation      | Description                    |
|----------|-----------|----------|-------------|--------------|-----------------|--------------------------------|
| Limit    | int       | limit    | limit       | 10           | min=1, max=10000  | Number of items per page       |
| Offset   | int       | offset   | offset      | 0            | min=0           | Number of items to skip        |
| Page     | int       | page     | page        | 1            | min=1           | Current page number            |
| SortBy   | string    | sortBy   | sort_by     | created_at   | -               | Field to sort by               |
| Order    | SortOrder | order    | order       | desc         | oneof=asc,desc  | Sort order (asc/desc)          |
| Search   | string    | search   | search      | ""           | -               | Search query                   |
| Filter   | string    | filter   | filter      | ""           | -               | Filter expression              |

#### CursorParams

Request parameters for cursor-based pagination:

| Field    | Type      | JSON Tag | Query Param | Default      | Validation      | Description                    |
|----------|-----------|----------|-------------|--------------|-----------------|--------------------------------|
| Limit    | int       | limit    | limit       | 10           | min=1, max=10000  | Number of items per page       |
| Cursor   | string    | cursor   | cursor      | ""           | -               | Base64-encoded cursor          |
| SortBy   | string    | sortBy   | sort_by     | created_at   | -               | Field to sort by               |
| Order    | SortOrder | order    | order       | desc         | oneof=asc,desc  | Sort order (asc/desc)          |
| Search   | string    | search   | search      | ""           | -               | Search query                   |
| Filter   | string    | filter   | filter      | ""           | -               | Filter expression              |

#### PageResponse[T]

Generic response structure:

```go
type PageResponse[T any] struct {
    Data       []T         `json:"data"`
    Pagination *PageMeta   `json:"pagination,omitempty"`
    Cursor     *CursorMeta `json:"cursor,omitempty"`
}
```

#### PageMeta

Metadata for offset-based pagination:

```go
type PageMeta struct {
    Total       int64 `json:"total"`        // Total number of items
    Limit       int   `json:"limit"`        // Items per page
    Offset      int   `json:"offset"`       // Current offset
    CurrentPage int   `json:"currentPage"`  // Current page number
    TotalPages  int   `json:"totalPages"`   // Total number of pages
    HasNext     bool  `json:"hasNext"`      // Whether there's a next page
    HasPrev     bool  `json:"hasPrev"`      // Whether there's a previous page
}
```

#### CursorMeta

Metadata for cursor-based pagination:

```go
type CursorMeta struct {
    NextCursor string `json:"nextCursor,omitempty"` // Cursor for next page
    PrevCursor string `json:"prevCursor,omitempty"` // Cursor for previous page
    HasNext    bool   `json:"hasNext"`              // Whether there's a next page
    HasPrev    bool   `json:"hasPrev"`              // Whether there's a previous page
    Count      int    `json:"count"`                // Number of items in current page
}
```

### Methods

#### PaginationParams Methods

```go
func (p *PaginationParams) Validate() error
func (p *PaginationParams) GetLimit() int
func (p *PaginationParams) GetOffset() int
func (p *PaginationParams) GetPage() int
func (p *PaginationParams) GetSortBy() string
func (p *PaginationParams) GetOrder() SortOrder
func (p *PaginationParams) GetOrderClause() string  // Returns "field_name ASC/DESC"
```

#### CursorParams Methods

```go
func (c *CursorParams) Validate() error
func (c *CursorParams) GetLimit() int
func (c *CursorParams) GetSortBy() string
func (c *CursorParams) GetOrder() SortOrder
func (c *CursorParams) GetOrderClause() string
```

#### Response Constructors

```go
// Create paginated response with metadata
func NewPageResponse[T any](data []T, total int64, params *PaginationParams) *PageResponse[T]

// Create cursor-based response with metadata
func NewCursorResponse[T any](data []T, nextCursor, prevCursor string, params *CursorParams) *PageResponse[T]

// Create empty paginated response
func NewEmptyPageResponse[T any]() *PageResponse[T]

// Create empty cursor-based response
func NewEmptyCursorResponse[T any]() *PageResponse[T]
```

#### Cursor Utilities

```go
// Encode cursor data to base64 string
func EncodeCursor(id string, timestamp time.Time, value string) (string, error)

// Decode base64 cursor string
func DecodeCursor(cursor string) (*CursorData, error)

// Simple string encoding/decoding
func SimpleCursorEncode(value string) string
func SimpleCursorDecode(cursor string) (string, error)
```

## Usage Examples

### Basic Listing with Pagination

```go
// GET /api/users?limit=20&page=2&sort_by=name&order=asc

type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) ListUsers(c *forge.Context) error {
    var params pagination.PaginationParams
    if err := c.BindQuery(&params); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid parameters"})
    }

    if err := params.Validate(); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }

    users, total, err := h.service.ListUsers(
        c.Context(),
        params.GetLimit(),
        params.GetOffset(),
        params.GetSortBy(),
        params.GetOrder(),
    )
    if err != nil {
        return c.JSON(500, map[string]string{"error": "failed to fetch users"})
    }

    response := pagination.NewPageResponse(users, total, &params)
    return c.JSON(200, response)
}
```

**Response:**
```json
{
  "data": [
    {"id": "21", "name": "Alice", "email": "alice@example.com", "created_at": "2024-01-15T10:00:00Z"},
    {"id": "22", "name": "Bob", "email": "bob@example.com", "created_at": "2024-01-16T10:00:00Z"}
  ],
  "pagination": {
    "total": 100,
    "limit": 20,
    "offset": 20,
    "currentPage": 2,
    "totalPages": 5,
    "hasNext": true,
    "hasPrev": true
  }
}
```

### Cursor-Based Pagination (Large Datasets)

```go
// GET /api/posts?limit=10&cursor=eyJpZCI6IjEyMyIsInRzIjoxNjQwMDAwMDAwfQ==

func (h *Handler) ListPosts(c *forge.Context) error {
    var params pagination.CursorParams
    if err := c.BindQuery(&params); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid parameters"})
    }

    if err := params.Validate(); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }

    // Decode cursor
    var afterID string
    var afterTime time.Time
    if params.Cursor != "" {
        cursorData, err := pagination.DecodeCursor(params.Cursor)
        if err != nil {
            return c.JSON(400, map[string]string{"error": "invalid cursor"})
        }
        afterID = cursorData.ID
        afterTime = cursorData.Timestamp
    }

    // Fetch one extra item to determine if there's a next page
    limit := params.GetLimit() + 1
    posts, err := h.service.ListPostsCursor(
        c.Context(),
        afterID,
        afterTime,
        limit,
        params.GetSortBy(),
        params.GetOrder(),
    )
    if err != nil {
        return c.JSON(500, map[string]string{"error": "failed to fetch posts"})
    }

    // Check if there's a next page
    hasNext := len(posts) > params.GetLimit()
    if hasNext {
        posts = posts[:params.GetLimit()]
    }

    // Generate next cursor
    var nextCursor string
    if hasNext && len(posts) > 0 {
        lastPost := posts[len(posts)-1]
        nextCursor, _ = pagination.EncodeCursor(
            lastPost.ID,
            lastPost.CreatedAt,
            "",
        )
    }

    response := pagination.NewCursorResponse(posts, nextCursor, "", &params)
    return c.JSON(200, response)
}
```

**Response:**
```json
{
  "data": [
    {"id": "post_1", "title": "First Post", "created_at": "2024-01-15T10:00:00Z"},
    {"id": "post_2", "title": "Second Post", "created_at": "2024-01-16T10:00:00Z"}
  ],
  "cursor": {
    "nextCursor": "eyJpZCI6InBvc3RfMiIsInRzIjoxNzA1NDAwMDAwfQ==",
    "prevCursor": "",
    "hasNext": true,
    "hasPrev": false,
    "count": 2
  }
}
```

### Repository Integration (Bun ORM)

```go
type userRepository struct {
    db *bun.DB
}

func (r *userRepository) List(
    ctx context.Context,
    limit, offset int,
    sortBy string,
    order pagination.SortOrder,
) ([]*User, int64, error) {
    var users []*User
    
    query := r.db.NewSelect().
        Model(&users).
        Limit(limit).
        Offset(offset)

    // Apply sorting
    orderClause := fmt.Sprintf("%s %s", sortBy, strings.ToUpper(string(order)))
    query = query.Order(orderClause)

    // Get total count
    total, err := query.ScanAndCount(ctx)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to fetch users: %w", err)
    }

    return users, int64(total), nil
}

// Cursor-based query
func (r *userRepository) ListCursor(
    ctx context.Context,
    afterID string,
    afterTime time.Time,
    limit int,
    sortBy string,
    order pagination.SortOrder,
) ([]*User, error) {
    var users []*User
    
    query := r.db.NewSelect().
        Model(&users).
        Limit(limit)

    // Apply cursor condition
    if afterID != "" {
        if order == pagination.SortOrderDesc {
            query = query.Where("created_at < ? OR (created_at = ? AND id < ?)",
                afterTime, afterTime, afterID)
        } else {
            query = query.Where("created_at > ? OR (created_at = ? AND id > ?)",
                afterTime, afterTime, afterID)
        }
    }

    // Apply sorting
    orderClause := fmt.Sprintf("%s %s", sortBy, strings.ToUpper(string(order)))
    query = query.Order(orderClause)

    err := query.Scan(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch users: %w", err)
    }

    return users, nil
}
```

### With Search and Filtering

```go
func (h *Handler) SearchUsers(c *forge.Context) error {
    var params pagination.PaginationParams
    if err := c.BindQuery(&params); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid parameters"})
    }

    if err := params.Validate(); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }

    // Parse filter (e.g., "status:active,role:admin")
    filters := parseFilters(params.Filter)

    users, total, err := h.service.SearchUsers(
        c.Context(),
        params.Search,
        filters,
        params.GetLimit(),
        params.GetOffset(),
        params.GetSortBy(),
        params.GetOrder(),
    )
    if err != nil {
        return c.JSON(500, map[string]string{"error": "failed to search users"})
    }

    response := pagination.NewPageResponse(users, total, &params)
    return c.JSON(200, response)
}

func parseFilters(filterStr string) map[string]string {
    filters := make(map[string]string)
    if filterStr == "" {
        return filters
    }

    pairs := strings.Split(filterStr, ",")
    for _, pair := range pairs {
        kv := strings.SplitN(pair, ":", 2)
        if len(kv) == 2 {
            filters[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
        }
    }
    return filters
}
```

## Route Registration

```go
// Register paginated endpoints
func Register(router forge.Router, h *Handler) {
    router.GET("/users", h.ListUsers,
        forge.WithName("users.list"),
        forge.WithSummary("List users with pagination"),
        forge.WithDescription("Returns a paginated list of users"),
        forge.WithRequestSchema(pagination.PaginationParams{}),
        forge.WithResponseSchema(200, "Success", pagination.PageResponse[User]{}),
        forge.WithResponseSchema(400, "Invalid parameters", ErrorResponse{}),
        forge.WithTags("Users"),
    )

    router.GET("/posts", h.ListPosts,
        forge.WithName("posts.list"),
        forge.WithSummary("List posts with cursor pagination"),
        forge.WithDescription("Returns a cursor-paginated list of posts"),
        forge.WithRequestSchema(pagination.CursorParams{}),
        forge.WithResponseSchema(200, "Success", pagination.PageResponse[Post]{}),
        forge.WithResponseSchema(400, "Invalid parameters", ErrorResponse{}),
        forge.WithTags("Posts"),
    )
}
```

## Configuration

Constants can be adjusted in the package:

```go
const (
    DefaultLimit = 10   // Default number of items per page
    MaxLimit     = 10000  // Maximum allowed items per page
    MinLimit     = 1    // Minimum allowed items per page
)
```

## Best Practices

### 1. Always Validate

```go
if err := params.Validate(); err != nil {
    return c.JSON(400, map[string]string{"error": err.Error()})
}
```

### 2. Use Getters for Safe Defaults

```go
// Instead of params.Limit directly
limit := params.GetLimit()  // Returns default if not set or invalid
```

### 3. Choose the Right Pagination Type

- **Offset-based**: Good for small to medium datasets, supports jumping to specific pages
- **Cursor-based**: Better for large datasets, real-time data, infinite scroll

### 4. Index Your Sort Fields

Ensure database indexes exist on fields used in `SortBy`:

```sql
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_name ON users(name);
```

### 5. Cursor Query Optimization

For cursor pagination, use composite indexes:

```sql
CREATE INDEX idx_posts_cursor ON posts(created_at DESC, id DESC);
```

### 6. Handle Empty Results

```go
if len(users) == 0 {
    return c.JSON(200, pagination.NewEmptyPageResponse[User]())
}
```

## Performance Considerations

1. **Offset Performance**: For large offsets, consider cursor-based pagination
2. **Count Queries**: Cache total counts for frequently accessed lists
3. **Index Usage**: Ensure proper indexes on sort fields
4. **Fetch N+1**: Fetch one extra item to determine `has_next` without separate query

## Testing

Run tests:

```bash
go test ./core/pagination/... -v
```

Run benchmarks:

```bash
go test ./core/pagination/... -bench=. -benchmem
```

## License

Part of the AuthSome project.

