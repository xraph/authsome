package hook_test

import (
	"context"
	"testing"

	"github.com/xraph/authsome/hook"
)

// An audit sink such as Chronicle refuses a record with no category, and
// most call sites never set one, so session.create, device.register,
// org.create, rbac.role.assign and org.invitation.accept were dropped from
// the trail. The bus now fills the category when a call site leaves it out.

func TestEmitFillsMissingCategory(t *testing.T) {
	cases := []struct {
		action, resource, want string
	}{
		{hook.ActionSessionCreate, hook.ResourceSession, hook.CategoryAuth},
		{"device.register", hook.ResourceDevice, hook.CategoryAuth},
		{hook.ActionOrgCreate, hook.ResourceOrganization, hook.CategoryOrg},
		{hook.ActionRoleAssign, hook.ResourceRole, hook.CategoryAccess},
		{hook.ActionInvitationAccept, hook.ResourceInvitation, hook.CategoryOrg},
		{hook.ActionTeamDelete, hook.ResourceTeam, hook.CategoryOrg},
		{hook.ActionDelegationGrant, "delegation", hook.CategoryAccess},
		{hook.ActionWebhookCreate, hook.ResourceWebhook, hook.CategoryAdmin},
		{hook.ActionEnvironmentClone, hook.ResourceEnvironment, hook.CategoryAdmin},
		{hook.ActionUserUpdate, hook.ResourceUser, hook.CategoryAccount},
		{hook.ActionAPIKeyCreate, hook.ResourceAPIKey, "apikey"},
		{hook.ActionDPoPProofReplayed, hook.ResourceSession, hook.CategoryAuth},
		{"consent.grant", "consent", "consent"},
		{"custom", hook.ResourceMember, hook.CategoryOrg},
		{"custom", "custom", hook.CategorySystem},
		{"", "", hook.CategorySystem},
	}
	for _, tc := range cases {
		t.Run(tc.action+"/"+tc.resource, func(t *testing.T) {
			bus := newBus()
			var got *hook.Event
			bus.On("capture", func(_ context.Context, ev *hook.Event) error { got = ev; return nil })

			bus.Emit(context.Background(), &hook.Event{Action: tc.action, Resource: tc.resource, Tenant: "app_1"})

			if got == nil {
				t.Fatal("handler did not run")
			}
			if got.Category != tc.want {
				t.Errorf("Category = %q, want %q", got.Category, tc.want)
			}
		})
	}
}

func TestEmitKeepsCallSiteCategory(t *testing.T) {
	bus := newBus()
	var got *hook.Event
	bus.On("capture", func(_ context.Context, ev *hook.Event) error { got = ev; return nil })

	bus.Emit(context.Background(), &hook.Event{
		Action: hook.ActionSessionRevoke, Resource: hook.ResourceSession, Tenant: "app_1", Category: hook.CategoryAccess,
	})

	if got == nil || got.Category != hook.CategoryAccess {
		t.Fatalf("a category the call site set must survive, got %+v", got)
	}
}
