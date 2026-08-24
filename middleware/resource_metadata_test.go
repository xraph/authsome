package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
)

const metaURL = "https://auth.example.com/.well-known/oauth-protected-resource"

// gate is a stand-in for the kind of middleware that produces the real 401s
// this package returns today (middleware/rbac.go, middleware/auth.go): it
// short-circuits the chain by returning an error without ever calling next,
// and never writes to ctx.Response() itself. That is the shape
// ResourceMetadataChallenge is built to detect, and it is a different shape
// from a route handler returning an error — a route handler's error is
// converted to a written response by forge before it is ever handed back to
// enclosing middleware, so wiring a fake "handler" via router.GET in this
// test would not exercise the code path this middleware actually runs on.
func gate(err error) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			if err != nil {
				return err
			}
			return next(ctx)
		}
	}
}

// newResourceMetadataRouter wires ResourceMetadataChallenge in front of gate
// on a real forge.Router, so the request travels through forge's own error
// handler the way it does in production. That handler is what turns a
// returned error into a written response, and it runs at the top of the
// chain, after both middlewares have returned.
func newResourceMetadataRouter(metadataURL string, gateErr error) http.Handler {
	r := forge.NewRouter()
	r.Use(middleware.ResourceMetadataChallenge(metadataURL))
	r.Use(gate(gateErr))
	r.GET("/x", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})
	return r
}

func TestResourceMetadataChallenge_SetsHeaderOn401(t *testing.T) {
	router := newResourceMetadataRouter(metaURL, forge.Unauthorized("authentication required"))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), `resource_metadata="`+metaURL+`"`)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "Bearer")
}

func TestResourceMetadataChallenge_LeavesSuccessAlone(t *testing.T) {
	router := newResourceMetadataRouter(metaURL, nil)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("WWW-Authenticate"))
}

// An empty URL means the operator has not configured discovery, so the
// middleware must be inert rather than emitting a broken header.
func TestResourceMetadataChallenge_InertWhenUnset(t *testing.T) {
	router := newResourceMetadataRouter("", forge.Unauthorized("authentication required"))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Header().Get("WWW-Authenticate"))
}

// A handler that already set its own challenge, like the RFC 7592 routes,
// keeps it. The header is not overwritten or appended to. RFC 7592's
// authenticateRegistration sets the header directly on ctx before returning
// its error (plugins/oauth2provider/register.go), which is why the gate
// here does the same instead of calling forge.Unauthorized.
func TestResourceMetadataChallenge_DoesNotOverwrite(t *testing.T) {
	r := forge.NewRouter()
	r.Use(middleware.ResourceMetadataChallenge(metaURL))
	r.Use(func(_ forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			ctx.Response().Header().Set("WWW-Authenticate", `Bearer realm="registration"`)
			return forge.Unauthorized("a valid registration access token is required")
		}
	})
	r.GET("/x", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `Bearer realm="registration"`, rec.Header().Get("WWW-Authenticate"))
}
