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
        // Try to find a matching route (longest base first)
        for _, rt := range a.routes {
            if r.Method != rt.method {
                continue
            }
            params, ok := matchPath(rt.base+rt.pattern, r.URL.Path)
            if !ok {
                continue
            }
            ctx := &Context{w: w, r: r, params: params}
            if err := rt.handler(ctx); err != nil {
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

// Param returns a path parameter captured by {name}
func (c *Context) Param(name string) string {
    if c.params == nil {
        return ""
    }
    return c.params[name]
}

// Group groups routes under a base path
type Group struct {
    app  *App
    base string
}

func (a *App) Group(basePath string) *Group { return &Group{app: a, base: basePath} }

// Router interface for route registration
type Router interface {
	Group(basePath string) *Group
}

func (g *Group) GET(path string, h func(*Context) error) { g.handle(http.MethodGet, path, h) }
func (g *Group) POST(path string, h func(*Context) error) { g.handle(http.MethodPost, path, h) }
func (g *Group) PUT(path string, h func(*Context) error) { g.handle(http.MethodPut, path, h) }
func (g *Group) DELETE(path string, h func(*Context) error) { g.handle(http.MethodDelete, path, h) }

// Group creates a sub-group under this group's base path
func (g *Group) Group(basePath string) *Group {
    return &Group{app: g.app, base: g.base + basePath}
}

func (g *Group) handle(method, path string, h func(*Context) error) {
    // store route; dispatching happens via catch-all handler
    g.app.routes = append(g.app.routes, route{method: method, base: g.base, pattern: path, handler: h})
}

// matchPath matches a pattern with optional {param} segments to an actual path
func matchPath(pattern, path string) (map[string]string, bool) {
    // Ensure both start with '/'
    if !strings.HasPrefix(pattern, "/") || !strings.HasPrefix(path, "/") {
        return nil, false
    }
    pSegs := splitPath(pattern)
    aSegs := splitPath(path)
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