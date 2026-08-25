package agentauth

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.AfterUserUpdate    = (*Plugin)(nil)
	_ plugin.AfterUserDelete    = (*Plugin)(nil)
	_ plugin.BeforeMemberRemove = (*Plugin)(nil)
	_ plugin.AfterOrgDelete     = (*Plugin)(nil)
)

// OnAfterUserUpdate revokes every grant a user issued once they are banned.
// Sessions die separately through DeleteUserSessions; the grant has to go too,
// or the next refresh mints a replacement session. OnAfterUserUpdate fires on
// every user update, ordinary profile edits included, so it must check the
// ban flag rather than acting unconditionally.
func (p *Plugin) OnAfterUserUpdate(ctx context.Context, u *user.User) error {
	if u == nil || !u.Banned {
		return nil
	}
	return p.revokeUserGrantsBulk(ctx, u.ID)
}

// OnAfterUserDelete revokes every grant a deleted user issued.
func (p *Plugin) OnAfterUserDelete(ctx context.Context, userID id.UserID) error {
	return p.revokeUserGrantsBulk(ctx, userID)
}

// OnBeforeMemberRemove revokes the leaving member's grants in that org only.
// Their grants in the user's other orgs are untouched: leaving one
// organization must not disarm agents acting for the user elsewhere.
func (p *Plugin) OnBeforeMemberRemove(ctx context.Context, m *organization.Member) error {
	if m == nil {
		return nil
	}
	if err := p.store.RevokeGrantsByUserOrg(ctx, m.UserID, m.OrgID); err != nil {
		return fmt.Errorf("agentauth: revoke org grants: %w", err)
	}
	// A bulk sweep revokes by query, not by id, so there is no single grant id
	// to hand cache.invalidate. clear drops every cached entry instead; it is
	// coarser than necessary here (it also drops unrelated grants' cache
	// entries), but correctness matters more than cache hit rate on an
	// offboarding path that runs rarely.
	p.cache.clear()
	return nil
}

// OnAfterOrgDelete revokes every grant scoped to the deleted organization,
// regardless of which user issued it. organization.Plugin.DeleteOrganization
// cascades member deletion without emitting BeforeMemberRemove (it deletes
// members as part of its own atomic cascade, not one at a time through
// RemoveMember), so OnBeforeMemberRemove never sees an org's members leave
// when the org itself is deleted. Without this hook, deleting an org would
// leave every grant scoped to it live until each one separately expired —
// the same offboarding hole member removal closes, just at the org level.
func (p *Plugin) OnAfterOrgDelete(ctx context.Context, orgID id.OrgID) error {
	if err := p.store.RevokeGrantsByOrg(ctx, orgID); err != nil {
		return fmt.Errorf("agentauth: revoke org grants: %w", err)
	}
	p.cache.clear()
	return nil
}

// revokeUserGrantsBulk revokes all of a user's grants, across every org, and
// clears the grant cache so none of them keep authorizing off a stale read.
func (p *Plugin) revokeUserGrantsBulk(ctx context.Context, userID id.UserID) error {
	if err := p.store.RevokeGrantsByUser(ctx, userID); err != nil {
		return fmt.Errorf("agentauth: revoke user grants: %w", err)
	}
	p.cache.clear()
	return nil
}
