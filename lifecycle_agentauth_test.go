package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugins/agentauth"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/user"
)

// TestAdminBanUser_RevokesAgentGrants pins the wiring, not just the handler:
// AdminBanUser is a real production ban path, distinct from UpdateMe, and it
// bypassed every plugin hook until this fix. OnAfterUserUpdate being correct
// in isolation (as covered in plugins/agentauth/lifecycle_test.go) says
// nothing about whether it ever gets called from here.
func TestAdminBanUser_RevokesAgentGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(agentauth.New(agentauth.WithStore(store))))

	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	u := &user.User{
		ID:        id.NewUserID(),
		AppID:     appID,
		Email:     "banned-user@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, eng.Store().CreateUser(context.Background(), u))

	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: id.NewAgentID(),
		UserID: u.ID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), g))

	require.NoError(t, eng.AdminBanUser(context.Background(), id.NewUserID(), u.ID, "policy violation", nil))

	got, err := store.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.RevokedAt, "banning a user through AdminBanUser must revoke their agent grants")
}

// TestDeleteOrganization_RevokesOnlyThatOrgsAgentGrants pins the wiring for
// review round 2's Item 2: organization.Plugin.DeleteOrganization cascades
// member deletion and then calls EmitAfterOrgDelete (plugins/organization/
// service.go) without ever going through RemoveMember, so
// OnBeforeMemberRemove never sees an org's members leave when the org itself
// is deleted — agentauth.Plugin.OnAfterOrgDelete is the only thing that can
// catch this path. Also proves the sweep is scoped to the deleted org only:
// a grant the same user holds in a surviving org must not be touched.
func TestDeleteOrganization_RevokesOnlyThatOrgsAgentGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	orgPlugin := orgplugin.New()
	_ = secutil.NewTestEngine(t,
		authsome.WithPlugin(orgPlugin),
		authsome.WithPlugin(agentauth.New(agentauth.WithStore(store))),
	)

	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	owner := id.NewUserID()

	deletedOrg := &organization.Organization{
		ID: id.NewOrgID(), AppID: appID, Name: "Acme", Slug: "acme-delete-test",
		CreatedBy: owner, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, orgPlugin.CreateOrganization(context.Background(), deletedOrg))

	survivingOrg := &organization.Organization{
		ID: id.NewOrgID(), AppID: appID, Name: "Acme 2", Slug: "acme-survives-test",
		CreatedBy: owner, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, orgPlugin.CreateOrganization(context.Background(), survivingOrg))

	gone := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: id.NewAgentID(),
		UserID: owner, OrgID: deletedOrg.ID, Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), gone))

	kept := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: id.NewAgentID(),
		UserID: owner, OrgID: survivingOrg.ID, Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), kept))

	require.NoError(t, orgPlugin.DeleteOrganization(context.Background(), deletedOrg.ID))

	gotGone, err := store.GetAgentGrant(context.Background(), gone.ID)
	require.NoError(t, err)
	assert.NotNil(t, gotGone.RevokedAt, "deleting an organization through DeleteOrganization must revoke agent grants scoped to it")

	gotKept, err := store.GetAgentGrant(context.Background(), kept.ID)
	require.NoError(t, err)
	assert.Nil(t, gotKept.RevokedAt, "a grant scoped to a surviving org must not be touched by deleting a different org")
}
