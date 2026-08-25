//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// agentIssueEngine is a minimal plugin.Engine wired to a real backend, with
// one purpose: proving IssueAgentSession's fixes hold against postgres's
// actual constraints, not just against the memory store used by
// plugins/agentauth's own unit tests, which enforces neither the unique
// token index nor the env_id foreign key.
//
// It embeds a nil plugin.Engine so only the methods OnInit and
// IssueAgentSession actually call need implementing — anything else this
// test provoked would panic loudly rather than silently pass through, the
// same reasoning stubSessionStore documents in engine_session_roles_test.go.
type agentIssueEngine struct {
	plugin.Engine
	store    store.Store
	registry *plugin.Registry
}

func (e *agentIssueEngine) Store() store.Store          { return e.store }
func (e *agentIssueEngine) Plugins() *plugin.Registry   { return e.registry }
func (e *agentIssueEngine) Hooks() *hook.Bus            { return hook.NewBus(log.NewNoopLogger()) }
func (e *agentIssueEngine) Logger() log.Logger          { return log.NewNoopLogger() }
func (e *agentIssueEngine) BasePath() string            { return "/v1" }
func (e *agentIssueEngine) Chronicle() bridge.Chronicle { return nil }
func (e *agentIssueEngine) GetUser(ctx context.Context, userID id.UserID) (*user.User, error) {
	return e.store.GetUser(ctx, userID)
}

// TestIssueAgentSession_AgainstPostgres_TwoSessionsInsertWithDistinctTokensAndValidEnv
// reproduces the reviewer's finding as a permanent regression test, run
// against a real postgres instance via testcontainers rather than the memory
// store.
//
// Before the C1/C2 fixes: IssueAgentSession hand-built a session with
// Token="" and RefreshToken="" and no EnvID. The first insert succeeded
// (empty strings and a zero env_id are valid Go zero values, and the FK
// violation happens exactly the same way regardless of which insert first
// hits it); the second call failed with "authsome: conflict" on
// idx_authsome_sessions_token / idx_authsome_sessions_refresh_token, and
// depending on which app the test used, the very first insert could equally
// fail outright on the authsome_sessions env_id NOT NULL + foreign-key
// constraint added in migrations.go. This test exercises the exact
// production code path (agentauth.Plugin.IssueAgentSession) against the exact
// production constraints.
func TestIssueAgentSession_AgainstPostgres_TwoSessionsInsertWithDistinctTokensAndValidEnv(t *testing.T) {
	s := setupTestStore(t)
	a := createTestApp(t, s, "agentauth-issue")
	owner := createTestUser(t, s, a.ID, "owner@test.com")

	reg := plugin.NewRegistry(log.NewNoopLogger())
	eng := &agentIssueEngine{store: s, registry: reg}

	grantStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(grantStore))
	require.NoError(t, p.OnInit(context.Background(), eng))

	grant := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: a.ID, AgentID: id.NewAgentID(),
		UserID: owner.ID, Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, grantStore.CreateAgentGrant(context.Background(), grant))

	sess1, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})
	require.NoError(t, err, "first agent session must insert cleanly against postgres")

	sess2, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})
	require.NoError(t, err, "a second agent session from the same grant must not collide on the unique token index")

	assert.NotEmpty(t, sess1.Token)
	assert.NotEmpty(t, sess1.RefreshToken)
	assert.NotEmpty(t, sess2.Token)
	assert.NotEmpty(t, sess2.RefreshToken)
	assert.NotEqual(t, sess1.Token, sess2.Token)
	assert.NotEqual(t, sess1.RefreshToken, sess2.RefreshToken)

	assert.False(t, sess1.EnvID.IsNil(), "EnvID must be set for the postgres FK to be satisfiable")
	assert.Equal(t, testEnvID(t, a.ID).String(), sess1.EnvID.String())

	// Round-trip through the actual backend, not just the returned struct:
	// GetSession would fail on a session that didn't really persist.
	got, err := s.GetSession(context.Background(), sess1.ID)
	require.NoError(t, err)
	assert.Equal(t, sess1.Token, got.Token)
}
