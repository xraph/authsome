package middleware

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"

	"github.com/xraph/forge"
	forgemw "github.com/xraph/forge/middleware"
	log "github.com/xraph/go-utils/log"
)

// internalLog is where InternalError writes the cause. The engine installs
// its logger at construction; until then causes go nowhere, which is the
// safe default for a value nobody asked to see.
var internalLog atomic.Pointer[log.Logger]

// SetInternalErrorLogger installs the logger InternalError reports causes to.
func SetInternalErrorLogger(l log.Logger) {
	if l == nil {
		return
	}
	internalLog.Store(&l)
}

func internalLogger() log.Logger {
	if l := internalLog.Load(); l != nil {
		return *l
	}
	return log.NewNoopLogger()
}

// InternalHTTPError is the 500 a client sees for a failure that is ours:
// a fixed message, never the cause. Forge masks every 5xx body to its
// status text, so the request id that ties the response to the log line
// travels on the X-Request-ID response header instead. The cause stays on
// the error for errors.Is and errors.As, and in the log line written when
// the error was made, keyed by the same request id.
type InternalHTTPError struct {
	RequestID string
	cause     error
}

// RequestIDHeader carries the request id back to the client on a 500, so
// a support ticket can quote it and the log line can be found.
const RequestIDHeader = "X-Request-ID"

func (e *InternalHTTPError) Error() string   { return "internal error" }
func (e *InternalHTTPError) Unwrap() error   { return e.cause }
func (e *InternalHTTPError) StatusCode() int { return http.StatusInternalServerError }

// ResponseBody is what forge serialises (and then masks, as for any 5xx).
func (e *InternalHTTPError) ResponseBody() any {
	return map[string]any{"error": "internal error", "code": http.StatusInternalServerError}
}

// InternalError logs err with the request id, puts that id on the
// response, and returns the generic 500. Handlers use it wherever they
// used to hand forge the raw error, which put the cause (a store's error
// text, a path, a driver message) in the log without the id that finds it.
func InternalError(ctx forge.Context, err error) error {
	if ctx == nil {
		return InternalErrorCtx(context.Background(), err)
	}
	httpErr := InternalErrorCtx(ctx.Context(), err)
	var ie *InternalHTTPError
	if errors.As(httpErr, &ie) && ie.RequestID != "" && ctx.Response().Header().Get(RequestIDHeader) == "" {
		ctx.Response().Header().Set(RequestIDHeader, ie.RequestID)
	}
	return httpErr
}

// InternalErrorCtx is InternalError for code that holds a context.Context.
func InternalErrorCtx(ctx context.Context, err error) error {
	rid := ""
	if ctx != nil {
		rid = forgemw.GetRequestID(ctx)
	}
	msg := "<nil>"
	if err != nil {
		msg = err.Error()
	}
	internalLogger().Error("authsome: internal error",
		log.String("request_id", rid),
		log.String("error", msg),
	)
	return &InternalHTTPError{RequestID: rid, cause: err}
}
