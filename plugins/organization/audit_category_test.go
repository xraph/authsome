package organization_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
)

// Chronicle refuses an audit record with no category, so the org events that
// never set one (org.create, the invitation and team events) were dropped.
// Everything the plugin records must reach the sink with a category.
func TestOrgEvents_ReachTheTrailWithACategory(t *testing.T) {
	s := newTxTestSetup(t)
	ch := secutil.NewBufferedChronicle()
	secutil.AttachChronicle(t, s.eng, ch)
	ctx := context.Background()

	o := seedOrg(t, s.plugin, id.NewUserID())
	team := &organization.Team{ID: id.NewTeamID(), OrgID: o.ID, Name: "Ops", Slug: "ops"}
	assert.NoError(t, s.plugin.CreateTeam(ctx, team))
	assert.NoError(t, s.plugin.DeleteTeam(ctx, team.ID))
	inv := &organization.Invitation{
		ID: id.NewInvitationID(), OrgID: o.ID, Email: "x@example.com", Token: "tok-audit",
		Role: organization.RoleMember, Status: organization.InvitationPending,
	}
	assert.NoError(t, s.plugin.CreateInvitation(ctx, inv))
	assert.NoError(t, s.plugin.DeclineInvitation(ctx, inv.Token))

	for _, action := range []string{hook.ActionOrgCreate, hook.ActionTeamCreate, hook.ActionTeamDelete, hook.ActionInvitationDecline} {
		secutil.AssertAuditEvent(t, ch, action, func(ev *bridge.AuditEvent) {
			assert.Equal(t, hook.CategoryOrg, ev.Category, "action %s", action)
		})
	}
	for _, ev := range ch.Events() {
		assert.NotEmpty(t, ev.Category, "event %s reached the sink without a category", ev.Action)
	}
}
