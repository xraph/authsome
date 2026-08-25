package scim

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/user"
)

const scimLifecycleTestAppID = "aapp_01jf0000000000000000000000"

// Both tests below give their SCIMConfig a zero-value AppID. That is a
// workaround for a pre-existing, unrelated bug, not a design choice:
// Service.ValidateToken authenticates a bearer token by calling
// s.store.ListConfigs(ctx, "") — an empty string — and matching
// c.AppID.String() == appID for each candidate config. Every real config
// carries a non-empty AppID (dashboard config creation parses one from a
// required field; see handleCreateConfig), so that comparison can never
// succeed for a real deployment: authenticateSCIM would 401 every genuine
// SCIM token. This looks like it should have called the store's
// FindTokenByHash instead, which ValidateToken's own comment describes but
// the code below it does not use.
//
// That bug is unrelated to grant revocation and out of scope for this task,
// so it is not fixed here — flagged separately instead (see task-11-report
// round 2). Zeroing the config's AppID is the one case ValidateToken's
// comparison actually matches, which is what lets these tests still drive
// the real HTTP handlers (handlePatchUser / handleReplaceUser) end to end
// instead of calling their Go internals directly. The users under test
// still carry a normal, non-zero AppID.

// TestHandlePatchUser_ActiveFalse_RevokesAgentGrants covers Gap A from
// review round 2: SCIM PATCH /Users/:userId with {"path":"active","value":
// false} is the deactivation shape Okta and Entra ID send by default, and it
// went through applyUserPatchReplace + authStore.UpdateUser with no plugin
// emit at all. This drives the real HTTP handler (not the Go method behind
// it) with a real agentauth plugin registered on the same engine, the same
// way production traffic would hit it.
func TestHandlePatchUser_ActiveFalse_RevokesAgentGrants(t *testing.T) {
	p := New()
	agentStore := agentauth.NewMemoryStore()
	ag := agentauth.New(agentauth.WithStore(agentStore))
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p), authsome.WithPlugin(ag))

	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	h := router.Handler()

	appID, err := id.ParseAppID(scimLifecycleTestAppID)
	require.NoError(t, err)

	// AppID intentionally left zero-value on the config only — see
	// scimConfigWorkaroundNote above.
	cfg := &SCIMConfig{
		ID: id.NewSCIMConfigID(), Name: "okta",
		Enabled: true, AutoSuspend: true,
	}
	require.NoError(t, p.service.CreateConfig(context.Background(), cfg))
	token, _, err := p.service.GenerateToken(context.Background(), cfg.ID, "test-token", nil)
	require.NoError(t, err)

	u := &user.User{
		ID: id.NewUserID(), AppID: appID, Email: "patch-user@example.com",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, eng.Store().CreateUser(context.Background(), u))

	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: id.NewAgentID(),
		UserID: u.ID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), g))

	body, err := json.Marshal(PatchOp{
		Schemas:    []string{SchemaPatchOp},
		Operations: []Operation{{Op: "replace", Path: "active", Value: false}},
	})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, p.config.BasePath+"/Users/"+u.ID.String(), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/scim+json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	got, err := agentStore.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.RevokedAt, "SCIM PATCH active:false must revoke the user's agent grants")
}

// TestHandleReplaceUser_ActiveFalse_RevokesAgentGrants covers Gap B: SCIM PUT
// /Users/:userId (handleReplaceUser -> ProvisionUser's existing-user-update
// branch) sets Banned = !Active and persisted it with no emit. PUT and PATCH
// are genuinely different code paths (ProvisionUser vs
// applyUserPatchReplace), so this exercises PUT specifically, again through
// the real HTTP handler.
func TestHandleReplaceUser_ActiveFalse_RevokesAgentGrants(t *testing.T) {
	p := New()
	agentStore := agentauth.NewMemoryStore()
	ag := agentauth.New(agentauth.WithStore(agentStore))
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p), authsome.WithPlugin(ag))

	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	h := router.Handler()

	appID, err := id.ParseAppID(scimLifecycleTestAppID)
	require.NoError(t, err)

	// AppID intentionally left zero-value on the config only — see the
	// workaround note above TestHandlePatchUser_ActiveFalse_RevokesAgentGrants.
	// ProvisionUser's existing-user lookup keys on cfg.AppID
	// (GetUserByAnyEmail(ctx, cfg.AppID, ...)), so the user's own email row
	// must match it — both zero here — or ProvisionUser would silently take
	// the create-new-user branch instead of finding this one.
	cfg := &SCIMConfig{
		ID: id.NewSCIMConfigID(), Name: "entra",
		Enabled: true, AutoCreate: true, AutoSuspend: true,
	}
	require.NoError(t, p.service.CreateConfig(context.Background(), cfg))
	token, _, err := p.service.GenerateToken(context.Background(), cfg.ID, "test-token", nil)
	require.NoError(t, err)

	// ProvisionUser's existing-user branch is reached via
	// GetUserByAnyEmail, so the user needs a primary-email row, not just a
	// bare user record.
	u := &user.User{
		ID: id.NewUserID(), Email: "put-user@example.com",
		FirstName: "Ada", LastName: "Lovelace",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, eng.Store().CreateUserWithPrimaryEmail(context.Background(), u, user.NewPrimaryEmail(u, "test")))

	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: id.NewAgentID(),
		UserID: u.ID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), g))

	payload := UserResource{
		Schemas:  []string{SchemaUser},
		UserName: u.Email,
		Name:     Name{GivenName: "Ada", FamilyName: "Lovelace"},
		Emails:   []Email{{Value: u.Email, Primary: true}},
		Active:   false,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, p.config.BasePath+"/Users/"+u.ID.String(), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/scim+json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	got, err := agentStore.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.RevokedAt, "SCIM PUT active:false must revoke the user's agent grants")
}
