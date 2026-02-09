package pagination

import (
	"fmt"
	"strings"

	"github.com/xraph/authsome/internal/errs"
)

// Constants for pagination limits.
const (
	DefaultLimit = 10
	MaxLimit     = 10000
	MinLimit     = 1
)

// SortOrder represents the sort direction.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// BaseRequestParams contains common request parameters for sorting, searching, and filtering
// Can be used in both paginated and non-paginated requests.
type BaseRequestParams struct {
	SortBy string    `default:"created_at" example:"created_at"    json:"sortBy" optional:"true" query:"sortBy"`
	Order  SortOrder `default:"desc"       example:"desc"          json:"order"  optional:"true" query:"order"  validate:"oneof=asc desc"`
	Search string    `default:""           example:"john"          json:"search" optional:"true" query:"search"`
	Filter string    `default:""           example:"status:active" json:"filter" optional:"true" query:"filter"`
	Fields string    `default:""           example:"id,name,email" json:"fields" optional:"true" query:"fields"`
}

// PaginationParams represents offset-based pagination request parameters.
type PaginationParams struct {
	BaseRequestParams

	Limit  int `default:"10" example:"10" json:"limit"  optional:"true" query:"limit"  validate:"min=1,max=10000"`
	Offset int `default:"0"  example:"0"  json:"offset" optional:"true" query:"offset" validate:"min=0"`
	Page   int `default:"1"  example:"1"  json:"page"   optional:"true" query:"page"   validate:"min=1"`
}

// CursorParams represents cursor-based pagination parameters.
type CursorParams struct {
	BaseRequestParams

	Limit  int    `default:"10" example:"10"                                       json:"limit"  optional:"true" query:"limit"  validate:"min=1,max=10000"`
	Cursor string `default:""   example:"eyJpZCI6IjEyMyIsInRzIjoxNjQwMDAwMDAwfQ==" json:"cursor" optional:"true" query:"cursor"`
}

// PageResponse represents a paginated response with metadata.
type PageResponse[T any] struct {
	Data       []T         `json:"data"`
	Pagination *PageMeta   `json:"pagination,omitempty"`
	Cursor     *CursorMeta `json:"cursor,omitempty"`
}

// PageMeta contains offset-based pagination metadata.
type PageMeta struct {
	Total       int64 `example:"1000"  json:"total"`
	Limit       int   `example:"10"    json:"limit"`
	Offset      int   `example:"0"     json:"offset"`
	CurrentPage int   `example:"1"     json:"currentPage"`
	TotalPages  int   `example:"100"   json:"totalPages"`
	HasNext     bool  `example:"true"  json:"hasNext"`
	HasPrev     bool  `example:"false" json:"hasPrev"`
}

// Pagination is a simple pagination response struct for use in services.
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

// HasNext returns true if there are more pages.
func (p *Pagination) HasNext() bool {
	return p.Page < p.TotalPages
}

// HasPrev returns true if there are previous pages.
func (p *Pagination) HasPrev() bool {
	return p.Page > 1
}

// ToPageMeta converts Pagination to PageMeta for compatibility.
func (p *Pagination) ToPageMeta() *PageMeta {
	return &PageMeta{
		Total:       int64(p.TotalItems),
		Limit:       p.PageSize,
		Offset:      (p.Page - 1) * p.PageSize,
		CurrentPage: p.Page,
		TotalPages:  p.TotalPages,
		HasNext:     p.HasNext(),
		HasPrev:     p.HasPrev(),
	}
}

// CursorMeta contains cursor-based pagination metadata.
type CursorMeta struct {
	NextCursor string `example:"eyJpZCI6IjEyMyIsInRzIjoxNjQwMDAwMDAwfQ==" json:"nextCursor,omitempty"`
	PrevCursor string `example:"eyJpZCI6IjEwMCIsInRzIjoxNjM5OTAwMDAwfQ==" json:"prevCursor,omitempty"`
	HasNext    bool   `example:"true"                                     json:"hasNext"`
	HasPrev    bool   `example:"false"                                    json:"hasPrev"`
	Count      int    `example:"10"                                       json:"count"`
}

// Validate validates and normalizes base request parameters.
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
		return errs.InvalidInput("order", "must be 'asc' or 'desc'")
	}

	return nil
}

// GetSortBy returns the sort field with fallback.
func (b *BaseRequestParams) GetSortBy() string {
	if b.SortBy == "" {
		return "created_at"
	}

	return b.SortBy
}

// GetOrder returns the sort order with fallback.
func (b *BaseRequestParams) GetOrder() SortOrder {
	if b.Order == "" {
		return SortOrderDesc
	}

	return b.Order
}

// GetOrderClause returns SQL ORDER BY clause.
func (b *BaseRequestParams) GetOrderClause() string {
	return fmt.Sprintf("%s %s", b.GetSortBy(), strings.ToUpper(string(b.GetOrder())))
}

// GetFields returns the parsed list of fields to select
// Returns nil if no fields specified (select all).
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

// HasFields returns true if field selection is specified.
func (b *BaseRequestParams) HasFields() bool {
	return b.Fields != ""
}

// Validate validates and normalizes pagination parameters.
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
		return errs.InvalidInput("offset", "cannot be negative")
	}

	// Validate page
	if p.Page < 1 {
		return errs.InvalidInput("page", "must be at least 1")
	}

	// Calculate offset from page if offset not explicitly set
	if p.Offset == 0 && p.Page > 1 {
		p.Offset = (p.Page - 1) * p.Limit
	}

	return nil
}

// Validate validates cursor pagination parameters.
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

// GetLimit returns the limit with fallback to default.
func (p *PaginationParams) GetLimit() int {
	if p.Limit == 0 {
		return DefaultLimit
	}

	if p.Limit > MaxLimit {
		return MaxLimit
	}

	return p.Limit
}

// GetOffset returns the calculated offset.
func (p *PaginationParams) GetOffset() int {
	if p.Offset > 0 {
		return p.Offset
	}

	if p.Page > 1 {
		return (p.Page - 1) * p.GetLimit()
	}

	return 0
}

// GetPage returns the current page number.
func (p *PaginationParams) GetPage() int {
	if p.Page > 0 {
		return p.Page
	}

	if p.Offset > 0 {
		return (p.Offset / p.GetLimit()) + 1
	}

	return 1
}

// GetSortBy returns the sort field with fallback.
func (p *PaginationParams) GetSortBy() string {
	if p.SortBy == "" {
		return "created_at"
	}

	return p.SortBy
}

// GetOrder returns the sort order with fallback.
func (p *PaginationParams) GetOrder() SortOrder {
	if p.Order == "" {
		return SortOrderDesc
	}

	return p.Order
}

// GetOrderClause returns SQL ORDER BY clause.
func (p *PaginationParams) GetOrderClause() string {
	return fmt.Sprintf("%s %s", p.GetSortBy(), strings.ToUpper(string(p.GetOrder())))
}

// GetLimit returns the limit for cursor pagination.
func (c *CursorParams) GetLimit() int {
	if c.Limit == 0 {
		return DefaultLimit
	}

	if c.Limit > MaxLimit {
		return MaxLimit
	}

	return c.Limit
}

// GetSortBy returns the sort field with fallback.
func (c *CursorParams) GetSortBy() string {
	if c.SortBy == "" {
		return "created_at"
	}

	return c.SortBy
}

// GetOrder returns the sort order with fallback.
func (c *CursorParams) GetOrder() SortOrder {
	if c.Order == "" {
		return SortOrderDesc
	}

	return c.Order
}

// GetOrderClause returns SQL ORDER BY clause.
func (c *CursorParams) GetOrderClause() string {
	return fmt.Sprintf("%s %s", c.GetSortBy(), strings.ToUpper(string(c.GetOrder())))
}

// NewPageResponse creates a new paginated response.
func NewPageResponse[T any](data []T, total int64, params *PaginationParams) *PageResponse[T] {
	limit := params.GetLimit()
	offset := params.GetOffset()
	currentPage := params.GetPage()
	totalPages := max(int((total+int64(limit)-1)/int64(limit)), 1)

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

// NewCursorResponse creates a new cursor-based paginated response.
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

// NewEmptyPageResponse creates an empty paginated response.
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

// NewEmptyCursorResponse creates an empty cursor-based response.
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
