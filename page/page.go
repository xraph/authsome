// Package page holds the request and answer shape of a bounded list. It
// is a leaf package so the store, the domain packages and the API can all
// name a page without importing one another.
package page

// DefaultLimit and MaxLimit bound a paged list: a caller that asks for
// nothing gets fifty, and nobody gets more than two hundred in one call,
// so no request can pull a whole tenant's rows at once.
const (
	DefaultLimit = 50
	MaxLimit     = 200
)

// Opts asks for one page. Cursor is the NextCursor of the previous page,
// or empty for the first. Pages run newest first: ids are time-ordered,
// and the cursor is the id the previous page ended on.
type Opts struct {
	Limit  int
	Cursor string
}

// Normalize applies the default and the ceiling.
func (o Opts) Normalize() Opts {
	switch {
	case o.Limit <= 0:
		o.Limit = DefaultLimit
	case o.Limit > MaxLimit:
		o.Limit = MaxLimit
	}
	return o
}

// Page is one page of a list and, when more remain, the cursor for the next.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// Cut turns a fetch of limit+1 rows into a page: when the extra row is
// there, the page is cut to limit and the cursor is the last item's id.
func Cut[T any](items []T, limit int, idOf func(T) string) Page[T] {
	if items == nil {
		items = []T{}
	}
	if len(items) <= limit {
		return Page[T]{Items: items}
	}
	items = items[:limit]
	return Page[T]{Items: items, NextCursor: idOf(items[len(items)-1])}
}
