package postgres

import (
	"context"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/page"
	"github.com/xraph/authsome/session"
)

// Paged lists fetch limit+1 rows newest first (ids are time-ordered), with
// the cursor as an exclusive upper bound on the id, and let store.CutPage
// decide whether a next page exists. See store.Paged.

func (s *Store) ListUserSessionsPage(ctx context.Context, userID id.UserID, opts page.Opts) (page.Page[*session.Session], error) {
	opts = opts.Normalize()
	var models []SessionModel
	q := s.pg.NewSelect(&models).Where("user_id = ?", userID.String())
	if opts.Cursor != "" {
		q = q.Where("id < ?", opts.Cursor)
	}
	if err := q.OrderExpr("id DESC").Limit(opts.Limit + 1).Scan(ctx); err != nil {
		return page.Page[*session.Session]{}, pgError(err)
	}
	items := make([]*session.Session, 0, len(models))
	for i := range models {
		sess, err := toSession(&models[i])
		if err != nil {
			return page.Page[*session.Session]{}, err
		}
		items = append(items, sess)
	}
	return page.Cut(items, opts.Limit, func(x *session.Session) string { return x.ID.String() }), nil
}

func (s *Store) ListOrganizationsPage(ctx context.Context, appID id.AppID, opts page.Opts) (page.Page[*organization.Organization], error) {
	opts = opts.Normalize()
	var models []OrganizationModel
	q := s.pg.NewSelect(&models).Where("app_id = ?", appID.String())
	if opts.Cursor != "" {
		q = q.Where("id < ?", opts.Cursor)
	}
	if err := q.OrderExpr("id DESC").Limit(opts.Limit + 1).Scan(ctx); err != nil {
		return page.Page[*organization.Organization]{}, pgError(err)
	}
	items, err := toOrganizations(models)
	if err != nil {
		return page.Page[*organization.Organization]{}, err
	}
	return page.Cut(items, opts.Limit, func(x *organization.Organization) string { return x.ID.String() }), nil
}

func (s *Store) ListUserOrganizationsPage(ctx context.Context, userID id.UserID, opts page.Opts) (page.Page[*organization.Organization], error) {
	opts = opts.Normalize()
	var models []OrganizationModel
	q := s.pg.NewSelect(&models).
		Join("JOIN", "authsome_members AS mem", "mem.org_id = o.id").
		Where("mem.user_id = ?", userID.String())
	if opts.Cursor != "" {
		q = q.Where("o.id < ?", opts.Cursor)
	}
	if err := q.OrderExpr("o.id DESC").Limit(opts.Limit + 1).Scan(ctx); err != nil {
		return page.Page[*organization.Organization]{}, pgError(err)
	}
	items, err := toOrganizations(models)
	if err != nil {
		return page.Page[*organization.Organization]{}, err
	}
	return page.Cut(items, opts.Limit, func(x *organization.Organization) string { return x.ID.String() }), nil
}

func (s *Store) ListMembersPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Member], error) {
	opts = opts.Normalize()
	var models []MemberModel
	q := s.pg.NewSelect(&models).Where("org_id = ?", orgID.String())
	if opts.Cursor != "" {
		q = q.Where("id < ?", opts.Cursor)
	}
	if err := q.OrderExpr("id DESC").Limit(opts.Limit + 1).Scan(ctx); err != nil {
		return page.Page[*organization.Member]{}, pgError(err)
	}
	items := make([]*organization.Member, 0, len(models))
	for i := range models {
		m, err := toMember(&models[i])
		if err != nil {
			return page.Page[*organization.Member]{}, err
		}
		items = append(items, m)
	}
	return page.Cut(items, opts.Limit, func(x *organization.Member) string { return x.ID.String() }), nil
}

func (s *Store) ListInvitationsPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Invitation], error) {
	opts = opts.Normalize()
	var models []InvitationModel
	q := s.pg.NewSelect(&models).Where("org_id = ?", orgID.String())
	if opts.Cursor != "" {
		q = q.Where("id < ?", opts.Cursor)
	}
	if err := q.OrderExpr("id DESC").Limit(opts.Limit + 1).Scan(ctx); err != nil {
		return page.Page[*organization.Invitation]{}, pgError(err)
	}
	items := make([]*organization.Invitation, 0, len(models))
	for i := range models {
		inv, err := toInvitation(&models[i])
		if err != nil {
			return page.Page[*organization.Invitation]{}, err
		}
		items = append(items, inv)
	}
	return page.Cut(items, opts.Limit, func(x *organization.Invitation) string { return x.ID.String() }), nil
}

func (s *Store) ListTeamsPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Team], error) {
	opts = opts.Normalize()
	var models []TeamModel
	q := s.pg.NewSelect(&models).Where("org_id = ?", orgID.String())
	if opts.Cursor != "" {
		q = q.Where("id < ?", opts.Cursor)
	}
	if err := q.OrderExpr("id DESC").Limit(opts.Limit + 1).Scan(ctx); err != nil {
		return page.Page[*organization.Team]{}, pgError(err)
	}
	items := make([]*organization.Team, 0, len(models))
	for i := range models {
		tm, err := toTeam(&models[i])
		if err != nil {
			return page.Page[*organization.Team]{}, err
		}
		items = append(items, tm)
	}
	return page.Cut(items, opts.Limit, func(x *organization.Team) string { return x.ID.String() }), nil
}
