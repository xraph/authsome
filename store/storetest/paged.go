package storetest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/page"
	"github.com/xraph/authsome/store"
)

func seedOrg(t *testing.T, s store.Store, tn tenant, creator id.UserID, slug string) *organization.Organization {
	t.Helper()
	o := &organization.Organization{
		ID: id.NewOrgID(), AppID: tn.AppID, EnvID: tn.EnvID, Name: "Org " + slug, Slug: slug + "-" + suffix(id.NewOrgID().String()),
		CreatedBy: creator, CreatedAt: now(), UpdatedAt: now(),
	}
	require.NoError(t, s.CreateOrganization(context.Background(), o))
	return o
}

func seedMember(t *testing.T, s store.Store, orgID id.OrgID, userID id.UserID) *organization.Member {
	t.Helper()
	m := &organization.Member{ID: id.NewMemberID(), OrgID: orgID, UserID: userID, Role: organization.RoleMember, CreatedAt: now(), UpdatedAt: now()}
	require.NoError(t, s.CreateMember(context.Background(), m))
	return m
}

func seedTeam(t *testing.T, s store.Store, orgID id.OrgID, slug string) *organization.Team {
	t.Helper()
	tm := &organization.Team{ID: id.NewTeamID(), OrgID: orgID, Name: "Team " + slug, Slug: slug + "-" + suffix(id.NewTeamID().String()), CreatedAt: now(), UpdatedAt: now()}
	require.NoError(t, s.CreateTeam(context.Background(), tm))
	return tm
}

// checkPaging walks a list of three seeded ids (in creation order) with a
// page size of two: the first page holds the two newest with a cursor, the
// second holds the oldest and no cursor, and a foreign id never appears.
func checkPaging(t *testing.T, seeded []string, foreign string, list func(opts page.Opts) ([]string, string)) {
	t.Helper()
	first, cursor := list(page.Opts{Limit: 2})
	require.Len(t, first, 2, "a page holds at most limit items")
	assert.Equal(t, []string{seeded[2], seeded[1]}, first, "newest first")
	require.NotEmpty(t, cursor, "more remain, so there is a cursor")

	second, cursor2 := list(page.Opts{Limit: 2, Cursor: cursor})
	assert.Equal(t, []string{seeded[0]}, second, "the cursor continues where the page ended")
	assert.Empty(t, cursor2, "the last page carries no cursor")

	all, _ := list(page.Opts{})
	assert.NotContains(t, all, foreign, "another scope's rows never appear")
	assert.Len(t, all, 3)
}

func testListUserSessionsPage(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u, other := seedUser(t, s, tn, "page-sess@test.com"), seedUser(t, s, tn, "page-sess-other@test.com")
	sfx := suffix(u.ID.String())
	ids := make([]string, 0, 3)
	for _, n := range []string{"a", "b", "c"} {
		ids = append(ids, seedSession(t, s, tn, u.ID, n+"-tok-"+sfx, n+"-rtok-"+sfx).ID.String())
	}
	foreign := seedSession(t, s, tn, other.ID, "o-tok-"+sfx, "o-rtok-"+sfx).ID.String()
	checkPaging(t, ids, foreign, func(opts page.Opts) ([]string, string) {
		pg, err := s.ListUserSessionsPage(ctx, u.ID, opts)
		require.NoError(t, err)
		out := make([]string, 0, len(pg.Items))
		for _, x := range pg.Items {
			out = append(out, x.ID.String())
		}
		return out, pg.NextCursor
	})
}

func testListOrganizationsPage(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn, other := seedTenant(t, s), seedTenant(t, s)
	u := seedUser(t, s, tn, "page-org@test.com")
	ids := make([]string, 0, 3)
	for _, n := range []string{"a", "b", "c"} {
		ids = append(ids, seedOrg(t, s, tn, u.ID, n).ID.String())
	}
	foreign := seedOrg(t, s, other, u.ID, "o").ID.String()
	checkPaging(t, ids, foreign, func(opts page.Opts) ([]string, string) {
		pg, err := s.ListOrganizationsPage(ctx, tn.AppID, opts)
		require.NoError(t, err)
		out := make([]string, 0, len(pg.Items))
		for _, x := range pg.Items {
			out = append(out, x.ID.String())
		}
		return out, pg.NextCursor
	})
}

func testListUserOrganizationsPage(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u, other := seedUser(t, s, tn, "page-uorg@test.com"), seedUser(t, s, tn, "page-uorg-other@test.com")
	ids := make([]string, 0, 3)
	for _, n := range []string{"a", "b", "c"} {
		o := seedOrg(t, s, tn, u.ID, n)
		seedMember(t, s, o.ID, u.ID)
		ids = append(ids, o.ID.String())
	}
	foreignOrg := seedOrg(t, s, tn, other.ID, "o")
	seedMember(t, s, foreignOrg.ID, other.ID)
	checkPaging(t, ids, foreignOrg.ID.String(), func(opts page.Opts) ([]string, string) {
		pg, err := s.ListUserOrganizationsPage(ctx, u.ID, opts)
		require.NoError(t, err)
		out := make([]string, 0, len(pg.Items))
		for _, x := range pg.Items {
			out = append(out, x.ID.String())
		}
		return out, pg.NextCursor
	})
}

func testListMembersPage(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	owner := seedUser(t, s, tn, "page-mem-owner@test.com")
	org, other := seedOrg(t, s, tn, owner.ID, "m"), seedOrg(t, s, tn, owner.ID, "mo")
	ids := make([]string, 0, 3)
	for _, n := range []string{"a", "b", "c"} {
		ids = append(ids, seedMember(t, s, org.ID, seedUser(t, s, tn, "page-mem-"+n+"@test.com").ID).ID.String())
	}
	foreign := seedMember(t, s, other.ID, owner.ID).ID.String()
	checkPaging(t, ids, foreign, func(opts page.Opts) ([]string, string) {
		pg, err := s.ListMembersPage(ctx, org.ID, opts)
		require.NoError(t, err)
		out := make([]string, 0, len(pg.Items))
		for _, x := range pg.Items {
			out = append(out, x.ID.String())
		}
		return out, pg.NextCursor
	})
}

func testListInvitationsPage(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	owner := seedUser(t, s, tn, "page-inv-owner@test.com")
	org, other := seedOrg(t, s, tn, owner.ID, "i"), seedOrg(t, s, tn, owner.ID, "io")
	sfx := suffix(org.ID.String())
	ids := make([]string, 0, 3)
	for _, n := range []string{"a", "b", "c"} {
		ids = append(ids, seedInvitation(t, s, org.ID, owner.ID, n+"-inv-"+sfx).ID.String())
	}
	foreign := seedInvitation(t, s, other.ID, owner.ID, "o-inv-"+sfx).ID.String()
	checkPaging(t, ids, foreign, func(opts page.Opts) ([]string, string) {
		pg, err := s.ListInvitationsPage(ctx, org.ID, opts)
		require.NoError(t, err)
		out := make([]string, 0, len(pg.Items))
		for _, x := range pg.Items {
			out = append(out, x.ID.String())
		}
		return out, pg.NextCursor
	})
}

func testListTeamsPage(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	owner := seedUser(t, s, tn, "page-team-owner@test.com")
	org, other := seedOrg(t, s, tn, owner.ID, "t"), seedOrg(t, s, tn, owner.ID, "to")
	ids := make([]string, 0, 3)
	for _, n := range []string{"a", "b", "c"} {
		ids = append(ids, seedTeam(t, s, org.ID, n).ID.String())
	}
	foreign := seedTeam(t, s, other.ID, "o").ID.String()
	checkPaging(t, ids, foreign, func(opts page.Opts) ([]string, string) {
		pg, err := s.ListTeamsPage(ctx, org.ID, opts)
		require.NoError(t, err)
		out := make([]string, 0, len(pg.Items))
		for _, x := range pg.Items {
			out = append(out, x.ID.String())
		}
		return out, pg.NextCursor
	})
}
