package organization_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// txTestSetup builds a fresh engine + org plugin and seeds an organization
// with one member, one team, and one invitation so the cascade has work
// to do. Returns everything tests need.
type txTestSetup struct {
	eng    *authsome.Engine
	plugin *orgplugin.Plugin
	org    *organization.Organization
	team   *organization.Team
	inv    *organization.Invitation
	owner  id.UserID
}

func newTxTestSetup(t *testing.T) *txTestSetup {
	t.Helper()

	p := orgplugin.New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))

	owner := id.NewUserID()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	o := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      "Acme",
		Slug:      "acme-tx",
		CreatedBy: owner,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, p.CreateOrganization(context.Background(), o))

	team := &organization.Team{
		ID:        id.NewTeamID(),
		OrgID:     o.ID,
		Name:      "engineering",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, eng.Store().CreateTeam(context.Background(), team))

	inv := &organization.Invitation{
		ID:        id.NewInvitationID(),
		OrgID:     o.ID,
		Email:     "guest@example.com",
		Token:     "tok-tx-test",
		Role:      organization.RoleMember,
		CreatedAt: time.Now(),
	}
	require.NoError(t, eng.Store().CreateInvitation(context.Background(), inv))

	return &txTestSetup{eng: eng, plugin: p, org: o, team: team, inv: inv, owner: owner}
}

func TestDeleteOrganization_TransactionalRollback(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()

	// Inject a fault on DeleteTeam — the cascade hits members first, then teams.
	// When DeleteTeam fails, the already-deleted member must be restored.
	faultErr := errors.New("synthetic delete-team failure")
	secutil.InjectStoreFault(t, s.eng, "DeleteTeam", faultErr)

	var hookCalls int32
	secutil.OnAfterOrgDelete(t, s.eng, func(_ context.Context, _ id.OrgID) error {
		atomic.AddInt32(&hookCalls, 1)
		return nil
	})

	err := s.plugin.DeleteOrganization(ctx, s.org.ID)
	require.Error(t, err, "expected DeleteOrganization to fail when DeleteTeam is faulted")
	assert.ErrorIs(t, err, faultErr, "expected wrapped fault to be returned")

	// Org must still exist.
	got, err := s.plugin.GetOrganization(ctx, s.org.ID)
	require.NoError(t, err, "org should still exist after rollback")
	require.NotNil(t, got)

	// Owner member must still exist (was deleted before DeleteTeam failed; rollback restores).
	members, err := s.eng.Store().ListMembers(ctx, s.org.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, members, "rollback should restore members deleted before the fault")

	// Team must still exist (the fault prevented its deletion).
	teams, err := s.eng.Store().ListTeams(ctx, s.org.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, teams, "team should not have been deleted")

	// Invitation should still exist (cascade aborted before reaching it).
	invs, err := s.eng.Store().ListInvitations(ctx, s.org.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, invs, "invitation should not have been deleted")

	assert.Equal(t, int32(0), atomic.LoadInt32(&hookCalls),
		"AfterOrgDelete must NOT fire when the cascade fails")
}

func TestDeleteOrganization_HookOnlyOnCommit(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()

	// Fault the very last step — DeleteOrganization (the org row).
	faultErr := errors.New("synthetic delete-org failure")
	secutil.InjectStoreFault(t, s.eng, "DeleteOrganization", faultErr)

	var hookCalls int32
	secutil.OnAfterOrgDelete(t, s.eng, func(_ context.Context, _ id.OrgID) error {
		atomic.AddInt32(&hookCalls, 1)
		return nil
	})

	err := s.plugin.DeleteOrganization(ctx, s.org.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, faultErr)

	assert.Equal(t, int32(0), atomic.LoadInt32(&hookCalls),
		"AfterOrgDelete must NOT fire when the org-row delete fails")

	// Org must still exist (whole tx rolled back on memory store).
	got, err := s.plugin.GetOrganization(ctx, s.org.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestDeleteOrganization_HookFiresOnSuccess(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()

	var hookCalls int32
	var seenOrg id.OrgID
	secutil.OnAfterOrgDelete(t, s.eng, func(_ context.Context, orgID id.OrgID) error {
		atomic.AddInt32(&hookCalls, 1)
		seenOrg = orgID
		return nil
	})

	require.NoError(t, s.plugin.DeleteOrganization(ctx, s.org.ID))

	assert.Equal(t, int32(1), atomic.LoadInt32(&hookCalls),
		"AfterOrgDelete must fire exactly once on commit")
	assert.Equal(t, s.org.ID, seenOrg, "hook should receive the deleted orgID")

	// Org must be gone.
	_, err := s.plugin.GetOrganization(ctx, s.org.ID)
	require.Error(t, err)
}
