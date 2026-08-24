package principal

import "context"

// Context keys are unexported struct types so no other package can collide
// with them, which is the same convention middleware/context.go uses.
type principalCtxKey struct{}
type actorsCtxKey struct{}

// NewContext returns ctx carrying p as the resolved caller.
func NewContext(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalCtxKey{}, p)
}

// FromContext returns the resolved caller.
//
// This lives here rather than only in middleware so a plugin can read the
// caller without importing middleware, which would pull in forge and the
// whole HTTP surface.
func FromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalCtxKey{}).(*Principal)
	return p, ok
}

// NewActorsContext returns ctx carrying the actor chain.
func NewActorsContext(ctx context.Context, c Chain) context.Context {
	return context.WithValue(ctx, actorsCtxKey{}, c)
}

// ActorsFromContext returns the actor chain, if the caller is acting for
// somebody else.
func ActorsFromContext(ctx context.Context) (Chain, bool) {
	c, ok := ctx.Value(actorsCtxKey{}).(Chain)
	return c, ok
}
