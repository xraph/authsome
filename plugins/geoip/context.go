package geoip

import "context"

type ctxKey struct{}

// WithGeoLocation stores a GeoLocation in the context.
func WithGeoLocation(ctx context.Context, loc *GeoLocation) context.Context {
	return context.WithValue(ctx, ctxKey{}, loc)
}

// GeoLocationFrom extracts the GeoLocation from context. Returns nil if
// not present (e.g. geoip plugin not loaded).
func GeoLocationFrom(ctx context.Context) *GeoLocation {
	loc, _ := ctx.Value(ctxKey{}).(*GeoLocation) //nolint:errcheck // type assertion
	return loc
}
