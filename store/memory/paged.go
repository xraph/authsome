package memory

import (
	"context"
	"sort"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/page"
	"github.com/xraph/authsome/session"
)

// pageOf sorts items newest first by id, drops those at or past the
// cursor, and cuts the page. See store.Paged.
func pageOf[T any](items []T, opts page.Opts, idOf func(T) string) page.Page[T] {
	opts = opts.Normalize()
	sort.Slice(items, func(i, j int) bool { return idOf(items[i]) > idOf(items[j]) })
	if opts.Cursor != "" {
		kept := items[:0]
		for _, it := range items {
			if idOf(it) < opts.Cursor {
				kept = append(kept, it)
			}
		}
		items = kept
	}
	if len(items) > opts.Limit+1 {
		items = items[:opts.Limit+1]
	}
	return page.Cut(items, opts.Limit, idOf)
}

func (s *Store) ListUserSessionsPage(_ context.Context, userID id.UserID, opts page.Opts) (page.Page[*session.Session], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []*session.Session
	for _, x := range s.sessions {
		if x.UserID.String() == userID.String() {
			items = append(items, x)
		}
	}
	return pageOf(items, opts, func(x *session.Session) string { return x.ID.String() }), nil
}

func (s *Store) ListOrganizationsPage(_ context.Context, appID id.AppID, opts page.Opts) (page.Page[*organization.Organization], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []*organization.Organization
	for _, x := range s.orgs {
		if x.AppID.String() == appID.String() {
			items = append(items, x)
		}
	}
	return pageOf(items, opts, func(x *organization.Organization) string { return x.ID.String() }), nil
}

func (s *Store) ListUserOrganizationsPage(_ context.Context, userID id.UserID, opts page.Opts) (page.Page[*organization.Organization], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orgIDs := make(map[string]bool)
	for _, m := range s.members {
		if m.UserID.String() == userID.String() {
			orgIDs[m.OrgID.String()] = true
		}
	}
	var items []*organization.Organization
	for _, x := range s.orgs {
		if orgIDs[x.ID.String()] {
			items = append(items, x)
		}
	}
	return pageOf(items, opts, func(x *organization.Organization) string { return x.ID.String() }), nil
}

func (s *Store) ListMembersPage(_ context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Member], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []*organization.Member
	for _, x := range s.members {
		if x.OrgID.String() == orgID.String() {
			items = append(items, x)
		}
	}
	return pageOf(items, opts, func(x *organization.Member) string { return x.ID.String() }), nil
}

func (s *Store) ListInvitationsPage(_ context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Invitation], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []*organization.Invitation
	for _, x := range s.invitations {
		if x.OrgID.String() == orgID.String() {
			items = append(items, x)
		}
	}
	return pageOf(items, opts, func(x *organization.Invitation) string { return x.ID.String() }), nil
}

func (s *Store) ListTeamsPage(_ context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Team], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []*organization.Team
	for _, x := range s.teams {
		if x.OrgID.String() == orgID.String() {
			items = append(items, x)
		}
	}
	return pageOf(items, opts, func(x *organization.Team) string { return x.ID.String() }), nil
}
