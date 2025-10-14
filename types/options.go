package types

// Pagination options
type PaginationOptions struct {
    Page     int
    PageSize int
    OrderBy  string
    OrderDir string // "asc" or "desc"
}

// DefaultPagination returns default pagination options
func DefaultPagination() PaginationOptions {
    return PaginationOptions{
        Page:     1,
        PageSize: 20,
        OrderBy:  "created_at",
        OrderDir: "desc",
    }
}

// PaginatedResult represents a paginated result
type PaginatedResult struct {
    Data       interface{}
    Total      int
    Page       int
    PageSize   int
    TotalPages int
}