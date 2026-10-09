package hook

import "context"

// RequestInfo is what the auth middleware knows about the HTTP request an
// action ran under. Emit copies it onto every event that leaves the fields
// empty, so call sites never have to thread it by hand.
type RequestInfo struct {
	IP        string
	UserAgent string
	RequestID string
	SessionID string
}

type requestInfoKey struct{}

// WithRequestInfo returns a context carrying the request's correlation data.
func WithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey{}, info)
}

// RequestInfoFrom returns the request info set by WithRequestInfo.
func RequestInfoFrom(ctx context.Context) (RequestInfo, bool) {
	info, ok := ctx.Value(requestInfoKey{}).(RequestInfo)
	return info, ok
}
