package forge

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// App is a minimal HTTP app wrapper providing Group routing
type App struct {
	mux    *http.ServeMux
	routes []route
}

type route struct {
	method  string
	base    string
	pattern string // may include {param}
	handler func(*Context) error
}

func NewApp(mux *http.ServeMux) *App {
	a := &App{mux: mux}
	// catch-all dispatcher; we rely on internal route matching
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to find a matching route
		// Routes are checked in registration order, so more specific routes should be registered first
		// Within the dashboard plugin, "/assets/*" is registered before "/*" which is correct
		var matchedRoute *route
		var matchedParams map[string]string

		for i := range a.routes {
			rt := &a.routes[i]
			if r.Method != rt.method {
				continue
			}
			params, ok := matchPath(rt.base+rt.pattern, r.URL.Path)
			if !ok {
				continue
			}
			// Take the first match (allows plugin to control priority by registration order)
			matchedRoute = rt
			matchedParams = params
			break
		}

		if matchedRoute != nil {
			ctx := &Context{w: w, r: r, params: matchedParams}
			if err := matchedRoute.handler(ctx); err != nil {
				if !errors.Is(err, http.ErrAbortHandler) {
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				}
			}
			return
		}

		// No route matched
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})
	return a
}

// Context mimics a typical framework context
type Context struct {
	w      http.ResponseWriter
	r      *http.Request
	params map[string]string
	values map[string]interface{} // For storing request-scoped values
}

func (c *Context) Request() *http.Request { return c.r }

// Header returns the writable response headers
func (c *Context) Header() http.Header { return c.w.Header() }

func (c *Context) JSON(status int, v interface{}) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(status)
	return json.NewEncoder(c.w).Encode(v)
}

func (c *Context) HTML(status int, html string) error {
	c.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.w.WriteHeader(status)
	_, err := c.w.Write([]byte(html))
	return err
}

func (c *Context) Cookie(name string) (string, error) {
	ck, err := c.r.Cookie(name)
	if err != nil {
		return "", err
	}
	return ck.Value, nil
}

func (c *Context) SetHeader(key, value string) {
	c.w.Header().Set(key, value)
}

func (c *Context) Response() http.ResponseWriter {
	return c.w
}

func (c *Context) String(status int, s string) error {
	c.w.WriteHeader(status)
	_, err := c.w.Write([]byte(s))
	return err
}

func (c *Context) Redirect(status int, url string) error {
	http.Redirect(c.w, c.r, url, status)
	return nil
}

func (c *Context) Query(key string) string {
	return c.r.URL.Query().Get(key)
}

// Param returns a path parameter captured by {name}
func (c *Context) Param(name string) string {
	if c.params == nil {
		return ""
	}
	return c.params[name]
}

// Get retrieves a value from the context by key
func (c *Context) Get(key string) interface{} {
	if c.values == nil {
		return nil
	}
	return c.values[key]
}

// Set stores a value in the context by key
func (c *Context) Set(key string, value interface{}) {
	if c.values == nil {
		c.values = make(map[string]interface{})
	}
	c.values[key] = value
}

// SetRequest allows replacing the underlying HTTP request
func (c *Context) SetRequest(r *http.Request) {
	c.r = r
}

// Group groups routes under a base path
type Group struct {
	app  *App
	base string
}

func (a *App) Group(basePath string) *Group {
	// Ensure base path starts with /
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	return &Group{app: a, base: basePath}
}

// Router interface for route registration
type Router interface {
	Group(basePath string) *Group
}

func (g *Group) GET(path string, h func(*Context) error)    { g.handle(http.MethodGet, path, h) }
func (g *Group) POST(path string, h func(*Context) error)   { g.handle(http.MethodPost, path, h) }
func (g *Group) PUT(path string, h func(*Context) error)    { g.handle(http.MethodPut, path, h) }
func (g *Group) DELETE(path string, h func(*Context) error) { g.handle(http.MethodDelete, path, h) }

// Group creates a sub-group under this group's base path
func (g *Group) Group(basePath string) *Group {
	// Ensure proper path separator
	base := g.base
	if !strings.HasSuffix(base, "/") && !strings.HasPrefix(basePath, "/") {
		base += "/"
	}
	return &Group{app: g.app, base: base + basePath}
}

func (g *Group) handle(method, path string, h func(*Context) error) {
	// store route; dispatching happens via catch-all handler
	g.app.routes = append(g.app.routes, route{method: method, base: g.base, pattern: path, handler: h})
}

// matchPath matches a pattern with optional {param} segments or wildcards to an actual path
func matchPath(pattern, path string) (map[string]string, bool) {
	// Ensure both start with '/'
	if !strings.HasPrefix(pattern, "/") || !strings.HasPrefix(path, "/") {
		return nil, false
	}
	pSegs := splitPath(pattern)
	aSegs := splitPath(path)

	// Check for wildcard pattern (ends with *)
	if len(pSegs) > 0 && pSegs[len(pSegs)-1] == "*" {
		// Wildcard must match at least the prefix segments
		if len(aSegs) < len(pSegs)-1 {
			return nil, false
		}
		// Match the prefix segments (all except the *)
		params := make(map[string]string)
		for i := 0; i < len(pSegs)-1; i++ {
			ps := pSegs[i]
			as := aSegs[i]
			if isParam(ps) {
				name := strings.Trim(ps, "{}")
				if name == "" || as == "" {
					return nil, false
				}
				params[name] = as
				continue
			}
			if ps != as {
				return nil, false
			}
		}
		// Wildcard matches the rest
		return params, true
	}

	// Exact length matching for non-wildcard patterns
	if len(pSegs) != len(aSegs) {
		return nil, false
	}
	params := make(map[string]string)
	for i := range pSegs {
		ps := pSegs[i]
		as := aSegs[i]
		if isParam(ps) {
			name := strings.Trim(ps, "{}")
			if name == "" || as == "" {
				return nil, false
			}
			params[name] = as
			continue
		}
		if ps != as {
			return nil, false
		}
	}
	return params, true
}

func splitPath(p string) []string {
	s := strings.TrimSuffix(p, "/")
	s = strings.TrimPrefix(s, "/")
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "/")
}

func isParam(seg string) bool { return strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") }
