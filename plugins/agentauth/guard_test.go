package agentauth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/session"
)

// I1 — before this file, Guard shipped with zero tests and zero callers.
// It is the only place a WWW-Authenticate header actually reaches a client
// (errors.go's own doc comment says so: forge has no header support on any
// error type it recognizes), and it is the only enforcement point a host app
// will actually wire onto a route. The reviewer proved the gap by mutation:
// changing Guard's body to `return next(ctx)` unconditionally left the whole
// package suite green. These tests drive Guard through a real forge router
// and a real net/http round trip via httptest, exactly the machinery
// routes_test.go already uses, rather than calling the returned middleware
// function directly — a direct call cannot prove the header lands on an
// actual ResponseRecorder, only that the returned error carries one.

// guardTestRouter wires p.Guard(action, resource) in front of a handler that
// records whether it ran.
func guardTestRouter(t *testing.T, p *agentauth.Plugin, action string) (forge.Router, *bool) {
	t.Helper()
	called := false
	mux := forge.NewRouter()
	err := mux.GET("/protected", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	}, forge.WithMiddleware(p.Guard(action, "invoice")))
	require.NoError(t, err)
	return mux, &called
}

func doGuardedRequest(ctx context.Context, t *testing.T, mux forge.Router) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// TestGuard_NonAgentSessionReachesNext is the "human passes through
// untouched" half of Guard's doc comment, driven end to end through a real
// router rather than asserted against AuthorizeHTTP's return value alone.
func TestGuard_NonAgentSessionReachesNext(t *testing.T) {
	p, _, _, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	mux, called := guardTestRouter(t, p, "read")

	sess := &session.Session{ID: id.NewSessionID(), PrincipalKind: session.PrincipalKindUser}
	ctx := middleware.WithSession(context.Background(), sess)
	w := doGuardedRequest(ctx, t, mux)

	assert.True(t, *called, "a human session must reach the guarded handler")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGuard_DenialNeverReachesNext is the negative half: a denial must stop
// the request before the handler runs at all, not merely report the right
// status after the handler already ran. `return next(ctx)` unconditionally
// is exactly the mutation the reviewer used to prove this had no coverage;
// this test fails immediately under that mutation on the *called assertion.
func TestGuard_DenialNeverReachesNext(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|write|invoice": true,
	}})
	mux, called := guardTestRouter(t, p, "write") // grant only confers read

	ctx := middleware.WithSession(context.Background(), sess)
	w := doGuardedRequest(ctx, t, mux)

	assert.False(t, *called, "a denied request must never reach the guarded handler")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestGuard_InactiveGrantSetsWWWAuthenticateHeader is the header-delivery
// property Guard exists for. forge's own error-to-response conversion has no
// header support at all, so if Guard did not set the header explicitly on
// the response, it would be silently dropped no matter what AuthorizeHTTP's
// returned error carries. This is the only test in the package that reads a
// header back off a real ResponseRecorder rather than off the Go error.
func TestGuard_InactiveGrantSetsWWWAuthenticateHeader(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(-time.Hour)) // already expired
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})
	mux, called := guardTestRouter(t, p, "read")

	ctx := middleware.WithSession(context.Background(), sess)
	w := doGuardedRequest(ctx, t, mux)

	assert.False(t, *called)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Header().Get("WWW-Authenticate"), `error="invalid_token"`,
		"the header must actually land on the HTTP response, not just be recoverable from the Go error")
}

// TestGuard_ForbiddenBodyMatchesOpaqueDenial proves Guard renders exactly the
// body AuthorizeHTTP built for an opaque denial, rather than reinterpreting
// or rewriting it on the way out.
func TestGuard_ForbiddenBodyMatchesOpaqueDenial(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{}}) // owner denied
	mux, _ := guardTestRouter(t, p, "read")

	ctx := middleware.WithSession(context.Background(), sess)
	w := doGuardedRequest(ctx, t, mux)

	require.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), "response body must be valid JSON")
	assert.Equal(t, "insufficient permissions", body["error"])
	assert.Equal(t, float64(http.StatusForbidden), body["code"])
}

// M2 — Guard used to discard SessionFrom's ok and pass a nil session straight
// into AuthorizeHTTP, which (via Authorize's own nil-session handling) denies
// with ErrGrantInactive: a 401 claiming a grant was "revoked or expired".
// For a request that never carried a session at all, that description is
// simply false and sends an agent developer hunting a revocation that never
// happened. This proves the fix: an unauthenticated request gets the same
// "authentication required" response middleware/rbac.go gives for the same
// condition, not a fabricated revocation story.
func TestGuard_UnauthenticatedRequestGetsAuthRequiredNotFabricatedRevocation(t *testing.T) {
	p, _, _, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	mux, called := guardTestRouter(t, p, "read")

	w := doGuardedRequest(context.Background(), t, mux) // no session in context at all

	assert.False(t, *called)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Header().Get("WWW-Authenticate"), "revoked or expired",
		"a request that never carried a session must not be told a grant was revoked")
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "authentication required", body["error"])
}
