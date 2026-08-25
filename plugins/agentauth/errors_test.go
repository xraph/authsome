package agentauth_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
