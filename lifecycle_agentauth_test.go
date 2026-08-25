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
	"github.com/xraph/authsome/plugins/agentauth"
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
