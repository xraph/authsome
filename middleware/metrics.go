package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
)

// idSegment matches a TypeID path segment (prefix, underscore, 26 base32
// characters) so a route label is the route, not one label per user.
var idSegment = regexp.MustCompile(`^[a-z]+_[0-9a-hjkmnp-tv-z]{26}$`)

// RouteLabel returns the request path with id segments replaced by :id.
func RouteLabel(path string) string {
	out := make([]byte, 0, len(path))
	start := 0
	for i := 0; i <= len(path); i++ {
		if i < len(path) && path[i] != '/' {
			continue
		}
		seg := path[start:i]
		if idSegment.MatchString(seg) {
			seg = ":id"
		}
		out = append(out, seg...)
		if i < len(path) {
			out = append(out, '/')
		}
		start = i + 1
	}
	return string(out)
}

// statusWriter remembers the status a handler wrote. Forge's own writer
// does not expose it, and forge middleware never sees a handler's error
// (the router answers it first), so the only place the outcome is visible
// is on the wire.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the writer underneath.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Metrics records one event per request: action http.request, the method
// and route as the resource, the status code as the outcome, and the
// handler's latency. It runs at the net/http level, converted into a forge
// middleware, so it can see the status the handler wrote.
func Metrics(c bridge.MetricsCollector) forge.Middleware {
	if c == nil {
		return func(next forge.Handler) forge.Handler { return next }
	}
	pure := forge.PureMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			status := sw.status
			if status == 0 {
				status = http.StatusOK
			}
			tenant := ""
			if appID, ok := AppIDFrom(r.Context()); ok {
				tenant = appID.String()
			}
			c.RecordEvent("http.request", r.Method+" "+RouteLabel(r.URL.Path), strconv.Itoa(status), tenant, time.Since(start))
		})
	})
	return pure.ToMiddleware()
}
