package store

import (
	"context"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/page"
	"github.com/xraph/authsome/session"
)

// Paged is the bounded twin of the unbounded list methods, for the request
// path: every method returns at most opts.Limit rows, newest first, and a
// cursor for the rest. The unbounded methods stay for internal callers that
// need every row (cascades, cap enforcement, adoption). The domain store
// interfaces (session.Store, organization.Store) carry the same methods;
// this lists them in one place.
type Paged interface {
	ListUserSessionsPage(ctx context.Context, userID id.UserID, opts page.Opts) (page.Page[*session.Session], error)
	ListOrganizationsPage(ctx context.Context, appID id.AppID, opts page.Opts) (page.Page[*organization.Organization], error)
	ListUserOrganizationsPage(ctx context.Context, userID id.UserID, opts page.Opts) (page.Page[*organization.Organization], error)
	ListMembersPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Member], error)
	ListInvitationsPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Invitation], error)
	ListTeamsPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Team], error)
}
