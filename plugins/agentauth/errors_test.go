package agentauth_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
)

// An expired or revoked grant is 401, not 403. The distinction is what tells
// an agent to re-authorize instead of giving up.
func TestAuthorizeHTTP_InactiveGrantIs401(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(-time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	err := p.AuthorizeHTTP(context.Background(), sess, "read", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, agentauth.StatusOf(err))
	assert.Contains(t, agentauth.HeaderOf(err, "WWW-Authenticate"), `error="invalid_token"`)
}

// A missing scope is 403 and must name the scope the agent needs, so a
// developer can fix their authorization request without guesswork.
func TestAuthorizeHTTP_MissingScopeIs403WithScopeNamed(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|write|invoice": true,
	}})

	err := p.AuthorizeHTTP(context.Background(), sess, "write", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, agentauth.StatusOf(err))
	header := agentauth.HeaderOf(err, "WWW-Authenticate")
	assert.Contains(t, header, `error="insufficient_scope"`)
	assert.Contains(t, header, `scope="invoices:write"`, "the response must name the scope that would satisfy the route")
}

// A user-gate failure must NOT name a scope. Reporting it the same way as a
// scope failure would let an agent enumerate its owner's permissions.
func TestAuthorizeHTTP_UserGateFailureIsOpaque(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{}})

	err := p.AuthorizeHTTP(context.Background(), sess, "read", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, agentauth.StatusOf(err))
	assert.NotContains(t, agentauth.HeaderOf(err, "WWW-Authenticate"), "insufficient_scope",
		"a user-gate failure must not be reported as a scope problem")
}

func TestAuthorizeHTTP_NoPermissionCheckerIs403(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))

	err := p.AuthorizeHTTP(context.Background(), sess, "read", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, agentauth.StatusOf(err))
}

func TestAuthorizeHTTP_AllowedReturnsNil(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	require.NoError(t, p.AuthorizeHTTP(context.Background(), sess, "read", "invoice"))
}

// The security property this whole file exists to protect: a user-gate
// refusal and the "no permission checker" refusal must be byte-for-byte the
// same shape. If an agent could tell these two apart, it could distinguish
// "my owner lacks this permission" from "RBAC is unavailable" and use that
// signal to probe its owner's permission set failure by failure.
func TestAuthorizeHTTP_UserGateFailureMatchesNoCheckerFailure(t *testing.T) {
	pWithChecker, _, sessWithChecker, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	pWithChecker.SetPermissionChecker(&stubChecker{allow: map[string]bool{}})
	userGateErr := pWithChecker.AuthorizeHTTP(context.Background(), sessWithChecker, "read", "invoice")

	pNoChecker, _, sessNoChecker, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	noCheckerErr := pNoChecker.AuthorizeHTTP(context.Background(), sessNoChecker, "read", "invoice")

	require.Error(t, userGateErr)
	require.Error(t, noCheckerErr)
	assert.Equal(t, agentauth.StatusOf(noCheckerErr), agentauth.StatusOf(userGateErr))
	assert.Equal(t, userGateErr.Error(), noCheckerErr.Error(),
		"the two refusals must render identical messages so an agent cannot tell them apart")
}

// erroringChecker always fails the RBAC call itself with a transport-shaped
// error, standing in for a Warden outage. It is deliberately not stubChecker:
// stubChecker's HasPermission never returns a non-nil error, so nothing
// using it can exercise Authorize's default branch in errors.go (the one
// catching a wrapped, non-sentinel error) — which is exactly the coverage
// hole I2 reported: `return opaqueDenial()` there could be changed to
// `return nil` and the suite stayed green, because no test's checker was
// ever capable of returning an error at all.
type erroringChecker struct{}

func (erroringChecker) HasPermission(context.Context, id.UserID, string, string) (bool, error) {
	return false, errors.New("warden: connection refused")
}

// TestAuthorizeHTTP_PermissionCheckTransportFailureIsOpaque is I2's fix: a
// Warden transport failure (middleware.go:127's
// `fmt.Errorf("agentauth: permission check: %w", err)`, which is not one of
// Authorize's named sentinels) must render exactly like any other opaque
// denial — same status, same message, no scope leaked into the header.
func TestAuthorizeHTTP_PermissionCheckTransportFailureIsOpaque(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(erroringChecker{})

	err := p.AuthorizeHTTP(context.Background(), sess, "read", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, agentauth.StatusOf(err))
	assert.Equal(t, "insufficient permissions", err.Error(),
		"a Warden transport failure must render identically to any other opaque denial, not leak its own text")
	assert.Empty(t, agentauth.HeaderOf(err, "WWW-Authenticate"))
}

// TestAuthorizeHTTP_PermissionCheckTransportFailureIsLogged is I3's fix: the
// client-facing response for a store/RBAC transport failure is, by design,
// indistinguishable from a plain denial (see the test above) — so the only
// place the actual cause can go is a server-side log. Guard-verify: removing
// the p.logger.Warn call in errors.go's default branch (or reverting
// SetLogger to a no-op) makes this fail while every other test in the
// package stays green, which is precisely what I3 reported as the bug.
func TestAuthorizeHTTP_PermissionCheckTransportFailureIsLogged(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	testLogger := log.NewTestLogger()
	p.SetLogger(testLogger)
	p.SetPermissionChecker(erroringChecker{})

	_ = p.AuthorizeHTTP(context.Background(), sess, "read", "invoice")

	tl, ok := testLogger.(*log.TestLogger)
	require.True(t, ok)
	assert.NotEmpty(t, tl.GetLogsByLevel("WARN"),
		"a store or RBAC transport failure must be logged server-side even though the client only sees an opaque 403")
}

// TestAuthorizeHTTP_ScopeForIsDeterministicUnderDuplicateMapping is M1's fix:
// two scopes conferring the same Permission must always name the same one in
// the insufficient_scope response, not whichever one a Go map's iteration
// order happens to visit first on a given request.
func TestAuthorizeHTTP_ScopeForIsDeterministicUnderDuplicateMapping(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	// A second scope conferring the exact same (write, invoice) permission as
	// invoices:write, alphabetically after it, so "the smallest name wins"
	// is distinguishable from "whichever scope was registered first".
	agentauth.WithScope("invoices:write:zzz-duplicate", agentauth.Grants("write", "invoice"))(p)
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|write|invoice": true,
	}})

	var got string
	for i := 0; i < 20; i++ {
		err := p.AuthorizeHTTP(context.Background(), sess, "write", "invoice")
		require.Error(t, err)
		header := agentauth.HeaderOf(err, "WWW-Authenticate")
		if got == "" {
			got = header
		}
		assert.Equal(t, got, header, "the scope named in the response must not vary across requests")
	}
	assert.Contains(t, got, `scope="invoices:write"`,
		"the lexicographically smaller of the two conferring scopes must be the one named")
}
