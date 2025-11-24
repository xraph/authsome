package pagination

import (
	"fmt"
	"strings"
)

// Constants for pagination limits
const (
	DefaultLimit = 10
	MaxLimit     = 100
	MinLimit     = 1
)

// SortOrder represents the sort direction
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// BaseRequestParams contains common request parameters for sorting, searching, and filtering
// Can be used in both paginated and non-paginated requests
type BaseRequestParams struct {
	SortBy string    `json:"sortBy" query:"sortBy" default:"created_at" example:"created_at" optional:"true"`
	Order  SortOrder `json:"order" query:"order" default:"desc" validate:"oneof=asc desc" example:"desc" optional:"true"`
	Search string    `json:"search" query:"search" default:"" example:"john" optional:"true"`
	Filter string    `json:"filter" query:"filter" default:"" example:"status:active" optional:"true"`
	Fields string    `json:"fields" query:"fields" default:"" example:"id,name,email" optional:"true"`
}

// PaginationParams represents offset-based pagination request parameters
type PaginationParams struct {
	BaseRequestParams
	Limit  int `json:"limit" query:"limit" default:"10" validate:"min=1,max=100" example:"10" optional:"true"`
	Offset int `json:"offset" query:"offset" default:"0" validate:"min=0" example:"0" optional:"true"`
	Page   int `json:"page" query:"page" default:"1" validate:"min=1" example:"1" optional:"true"`
}

// CursorParams represents cursor-based pagination parameters
type CursorParams struct {
	BaseRequestParams
	Limit  int    `json:"limit" query:"limit" default:"10" validate:"min=1,max=100" example:"10" optional:"true"`
	Cursor string `json:"cursor" query:"cursor" default:"" example:"eyJpZCI6IjEyMyIsInRzIjoxNjQwMDAwMDAwfQ==" optional:"true"`
}

// PageResponse represents a paginated response with metadata
type PageResponse[T any] struct {
	Data       []T         `json:"data"`
	Pagination *PageMeta   `json:"pagination,omitempty"`
	Cursor     *CursorMeta `json:"cursor,omitempty"`
}

// PageMeta contains offset-based pagination metadata
type PageMeta struct {
	Total       int64 `json:"total" example:"1000"`
	Limit       int   `json:"limit" example:"10"`
	Offset      int   `json:"offset" example:"0"`
	CurrentPage int   `json:"currentPage" example:"1"`
	TotalPages  int   `json:"totalPages" example:"100"`
	HasNext     bool  `json:"hasNext" example:"true"`
	HasPrev     bool  `json:"hasPrev" example:"false"`
}

// CursorMeta contains cursor-based pagination metadata
type CursorMeta struct {
	NextCursor string `json:"nextCursor,omitempty" example:"eyJpZCI6IjEyMyIsInRzIjoxNjQwMDAwMDAwfQ=="`
	PrevCursor string `json:"prevCursor,omitempty" example:"eyJpZCI6IjEwMCIsInRzIjoxNjM5OTAwMDAwfQ=="`
	HasNext    bool   `json:"hasNext" example:"true"`
	HasPrev    bool   `json:"hasPrev" example:"false"`
	Count      int    `json:"count" example:"10"`
}

// Validate validates and normalizes base request parameters
func (b *BaseRequestParams) Validate() error {
	// Set defaults
	if b.Order == "" {
		b.Order = SortOrderDesc
	}
	if b.SortBy == "" {
		b.SortBy = "created_at"
	}

	// Validate order
	if b.Order != SortOrderAsc && b.Order != SortOrderDesc {
		return fmt.Errorf("order must be 'asc' or 'desc'")
	}

	return nil
}

// GetSortBy returns the sort field with fallback
func (b *BaseRequestParams) GetSortBy() string {
	if b.SortBy == "" {
		return "created_at"
	}
	return b.SortBy
}

// GetOrder returns the sort order with fallback
func (b *BaseRequestParams) GetOrder() SortOrder {
	if b.Order == "" {
		return SortOrderDesc
	}
	return b.Order
}

// GetOrderClause returns SQL ORDER BY clause
func (b *BaseRequestParams) GetOrderClause() string {
	return fmt.Sprintf("%s %s", b.GetSortBy(), strings.ToUpper(string(b.GetOrder())))
}

// GetFields returns the parsed list of fields to select
// Returns nil if no fields specified (select all)
func (b *BaseRequestParams) GetFields() []string {
	if b.Fields == "" {
		return nil
	}

	fields := strings.Split(b.Fields, ",")
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			result = append(result, field)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// HasFields returns true if field selection is specified
func (b *BaseRequestParams) HasFields() bool {
	return b.Fields != ""
}

// Validate validates and normalizes pagination parameters
func (p *PaginationParams) Validate() error {
	// Set defaults first
	if p.Limit == 0 {
		p.Limit = DefaultLimit
	}
	if p.Page == 0 {
		p.Page = 1
	}

	// Validate base params (sorts, order, search, filter)
	if err := p.BaseRequestParams.Validate(); err != nil {
		return err
	}

	// Validate pagination-specific fields
	if p.Limit < MinLimit {
		return fmt.Errorf("limit must be at least %d", MinLimit)
	}
	if p.Limit > MaxLimit {
		return fmt.Errorf("limit cannot exceed %d", MaxLimit)
	}

	// Validate offset
	if p.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	// Validate page
	if p.Page < 1 {
		return fmt.Errorf("page must be at least 1")
	}

	// Calculate offset from page if offset not explicitly set
	if p.Offset == 0 && p.Page > 1 {
		p.Offset = (p.Page - 1) * p.Limit
	}

	return nil
}

// Validate validates cursor pagination parameters
func (c *CursorParams) Validate() error {
	// Set defaults first
	if c.Limit == 0 {
		c.Limit = DefaultLimit
	}

	// Validate base params (sorts, order, search, filter)
	if err := c.BaseRequestParams.Validate(); err != nil {
		return err
	}

	// Validate cursor-specific fields
	if c.Limit < MinLimit {
		return fmt.Errorf("limit must be at least %d", MinLimit)
	}
	if c.Limit > MaxLimit {
		return fmt.Errorf("limit cannot exceed %d", MaxLimit)
	}

	return nil
}

// GetLimit returns the limit with fallback to default
func (p *PaginationParams) GetLimit() int {
	if p.Limit == 0 {
		return DefaultLimit
	}
	if p.Limit > MaxLimit {
		return MaxLimit
	}
	return p.Limit
}

// GetOffset returns the calculated offset
func (p *PaginationParams) GetOffset() int {
	if p.Offset > 0 {
		return p.Offset
	}
	if p.Page > 1 {
		return (p.Page - 1) * p.GetLimit()
	}
	return 0
}

// GetPage returns the current page number
func (p *PaginationParams) GetPage() int {
	if p.Page > 0 {
		return p.Page
	}
	if p.Offset > 0 {
		return (p.Offset / p.GetLimit()) + 1
	}
	return 1
}

// GetSortBy returns the sort field with fallback
func (p *PaginationParams) GetSortBy() string {
	if p.SortBy == "" {
		return "created_at"
	}
	return p.SortBy
}

// GetOrder returns the sort order with fallback
func (p *PaginationParams) GetOrder() SortOrder {
	if p.Order == "" {
		return SortOrderDesc
	}
	return p.Order
}

// GetOrderClause returns SQL ORDER BY clause
func (p *PaginationParams) GetOrderClause() string {
	return fmt.Sprintf("%s %s", p.GetSortBy(), strings.ToUpper(string(p.GetOrder())))
}

// GetLimit returns the limit for cursor pagination
func (c *CursorParams) GetLimit() int {
	if c.Limit == 0 {
		return DefaultLimit
	}
	if c.Limit > MaxLimit {
		return MaxLimit
	}
	return c.Limit
}

// GetSortBy returns the sort field with fallback
func (c *CursorParams) GetSortBy() string {
	if c.SortBy == "" {
		return "created_at"
	}
	return c.SortBy
}

// GetOrder returns the sort order with fallback
func (c *CursorParams) GetOrder() SortOrder {
	if c.Order == "" {
		return SortOrderDesc
	}
	return c.Order
}

// GetOrderClause returns SQL ORDER BY clause
func (c *CursorParams) GetOrderClause() string {
	return fmt.Sprintf("%s %s", c.GetSortBy(), strings.ToUpper(string(c.GetOrder())))
}

// NewPageResponse creates a new paginated response
func NewPageResponse[T any](data []T, total int64, params *PaginationParams) *PageResponse[T] {
	limit := params.GetLimit()
	offset := params.GetOffset()
	currentPage := params.GetPage()
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	if totalPages < 1 {
		totalPages = 1
	}

	return &PageResponse[T]{
		Data: data,
		Pagination: &PageMeta{
			Total:       total,
			Limit:       limit,
			Offset:      offset,
			CurrentPage: currentPage,
			TotalPages:  totalPages,
			HasNext:     currentPage < totalPages,
			HasPrev:     currentPage > 1,
		},
	}
}

// NewCursorResponse creates a new cursor-based paginated response
func NewCursorResponse[T any](data []T, nextCursor, prevCursor string, params *CursorParams) *PageResponse[T] {
	return &PageResponse[T]{
		Data: data,
		Cursor: &CursorMeta{
			NextCursor: nextCursor,
			PrevCursor: prevCursor,
			HasNext:    nextCursor != "",
			HasPrev:    prevCursor != "",
			Count:      len(data),
		},
	}
}

// NewEmptyPageResponse creates an empty paginated response
func NewEmptyPageResponse[T any]() *PageResponse[T] {
	return &PageResponse[T]{
		Data: []T{},
		Pagination: &PageMeta{
			Total:       0,
			Limit:       DefaultLimit,
			Offset:      0,
			CurrentPage: 1,
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
	}
}

// NewEmptyCursorResponse creates an empty cursor-based response
func NewEmptyCursorResponse[T any]() *PageResponse[T] {
	return &PageResponse[T]{
		Data: []T{},
		Cursor: &CursorMeta{
			NextCursor: "",
			PrevCursor: "",
			HasNext:    false,
			HasPrev:    false,
			Count:      0,
		},
	}
}
