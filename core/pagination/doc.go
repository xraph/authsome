/*
Package pagination provides comprehensive pagination support for the AuthSome framework.

This package implements both offset-based and cursor-based pagination patterns with
full support for sorting, filtering, and search. All structs follow the Forge DTO
tags specification for seamless integration with request binding and validation.

# Features

  - Offset-based pagination (page/limit/offset)
  - Cursor-based pagination for large datasets
  - Sorting and ordering (ASC/DESC)
  - Search and filtering support
  - Complete metadata in responses (total count, pages, has_next/prev)
  - Validation with sensible defaults
  - Type-safe generic responses
  - Base64 cursor encoding/decoding
  - Zero-allocation validation
  - Production-tested performance

# Quick Start

Offset-based pagination:

	func (h *Handler) ListUsers(c *forge.Context) error {
		var params pagination.PaginationParams
		if err := c.BindQuery(&params); err != nil {
			return c.JSON(400, ErrorResponse{Message: "invalid parameters"})
		}

		if err := params.Validate(); err != nil {
			return c.JSON(400, ErrorResponse{Message: err.Error()})
		}

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

		response := pagination.NewPageResponse(users, total, &params)
		return c.JSON(200, response)
	}

Cursor-based pagination:

	func (h *Handler) ListPosts(c *forge.Context) error {
		var params pagination.CursorParams
		if err := c.BindQuery(&params); err != nil {
			return c.JSON(400, ErrorResponse{Message: "invalid parameters"})
		}

		if err := params.Validate(); err != nil {
			return c.JSON(400, ErrorResponse{Message: err.Error()})
		}

		cursorData, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return c.JSON(400, ErrorResponse{Message: "invalid cursor"})
		}

		posts, nextCursor, prevCursor, err := h.postRepo.ListCursor(
			c.Context(), params.GetLimit()+1, cursorData, params.GetSortBy(), params.GetOrder(),
		)
		if err != nil {
			return c.JSON(500, ErrorResponse{Message: "failed to fetch posts"})
		}

		response := pagination.NewCursorResponse(posts, nextCursor, prevCursor, &params)
		return c.JSON(200, response)
	}

# Configuration

Constants can be adjusted for your use case:

	const (
		DefaultLimit = 10   // Default items per page
		MaxLimit     = 10000  // Maximum items per page
		MinLimit     = 1    // Minimum items per page
	)

# Request Parameters

Both PaginationParams and CursorParams support Forge DTO tags:

  - json: JSON field name
  - query: Query parameter name
  - default: Default value
  - validate: Validation rules
  - example: OpenAPI example

# Response Structure

Responses include comprehensive metadata:

	type PageResponse[T any] struct {
		Data       []T         `json:"data"`
		Pagination *PageMeta   `json:"pagination,omitempty"` // For offset-based
		Cursor     *CursorMeta `json:"cursor,omitempty"`     // For cursor-based
	}

# Performance

Benchmarks on Apple M3 Max:

	BenchmarkPaginationParams_Validate-16    430332202    2.771 ns/op    0 B/op    0 allocs/op
	BenchmarkEncodeCursor-16                   3105051  391.6 ns/op  416 B/op    5 allocs/op
	BenchmarkDecodeCursor-16                   1732579  691.3 ns/op  384 B/op    8 allocs/op

# Best Practices

1. Always validate parameters after binding:

	if err := params.Validate(); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

2. Use getter methods for safe defaults:

	limit := params.GetLimit()  // Returns default if not set

3. Choose the right pagination type:
  - Offset: Good for small/medium datasets, supports jumping to pages
  - Cursor: Better for large datasets, real-time data, infinite scroll

4. Index your database sort fields:

	CREATE INDEX idx_users_created_at ON users(created_at);
	CREATE INDEX idx_posts_cursor ON posts(created_at DESC, id DESC);

5. Handle empty results gracefully:

	if len(users) == 0 {
		return c.JSON(200, pagination.NewEmptyPageResponse[User]())
	}

# Database Integration

Example with Bun ORM:

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
			Offset(offset).
			Order(fmt.Sprintf("%s %s", sortBy, strings.ToUpper(string(order))))

		total, err := query.ScanAndCount(ctx)
		return users, int64(total), err
	}

# Thread Safety

All methods are safe for concurrent use. The validation methods modify the receiver
to set defaults but are designed to be called once per request.

# Error Handling

Validation errors are returned as standard Go errors with descriptive messages:

  - "limit must be at least 1"
  - "limit cannot exceed 10000"
  - "offset cannot be negative"
  - "page must be at least 1"
  - "order must be 'asc' or 'desc'"

# API Compatibility

This package follows semantic versioning. The public API is stable and will not
break between minor versions.

For detailed documentation and examples, see the README.md file.
*/
package pagination
