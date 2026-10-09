package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/page"
	"github.com/xraph/authsome/session"
)

// Paged lists fetch limit+1 documents newest first (ids are time-ordered),
// with the cursor as an exclusive upper bound on _id, and let
// store.CutPage decide whether a next page exists. See store.Paged.

// pageFilter adds the cursor bound to a filter, merging with any bound
// already on _id (a user's organizations are filtered by an $in on it).
func pageFilter(base bson.M, cursor string) bson.M {
	if cursor == "" {
		return base
	}
	if existing, ok := base["_id"].(bson.M); ok {
		existing["$lt"] = cursor
		return base
	}
	base["_id"] = bson.M{"$lt": cursor}
	return base
}

var newestFirst = bson.D{{Key: "_id", Value: -1}}

func (s *Store) ListUserSessionsPage(ctx context.Context, userID id.UserID, opts page.Opts) (page.Page[*session.Session], error) {
	opts = opts.Normalize()
	var models []sessionModel
	err := s.mdb.NewFind(&models).
		Filter(pageFilter(bson.M{"user_id": userID.String()}, opts.Cursor)).
		Sort(newestFirst).Limit(int64(opts.Limit + 1)).Scan(ctx)
	if err != nil {
		return page.Page[*session.Session]{}, fmt.Errorf("authsome/mongo: list user sessions page: %w", err)
	}
	items := make([]*session.Session, 0, len(models))
	for i := range models {
		sess, convErr := fromSessionModel(&models[i])
		if convErr != nil {
			return page.Page[*session.Session]{}, convErr
		}
		items = append(items, sess)
	}
	return page.Cut(items, opts.Limit, func(x *session.Session) string { return x.ID.String() }), nil
}

func (s *Store) organizationsPage(ctx context.Context, filter bson.M, opts page.Opts) (page.Page[*organization.Organization], error) {
	var models []organizationModel
	err := s.mdb.NewFind(&models).
		Filter(pageFilter(filter, opts.Cursor)).
		Sort(newestFirst).Limit(int64(opts.Limit + 1)).Scan(ctx)
	if err != nil {
		return page.Page[*organization.Organization]{}, fmt.Errorf("authsome/mongo: list organizations page: %w", err)
	}
	items := make([]*organization.Organization, 0, len(models))
	for i := range models {
		o, convErr := fromOrganizationModel(&models[i])
		if convErr != nil {
			return page.Page[*organization.Organization]{}, convErr
		}
		items = append(items, o)
	}
	return page.Cut(items, opts.Limit, func(x *organization.Organization) string { return x.ID.String() }), nil
}

func (s *Store) ListOrganizationsPage(ctx context.Context, appID id.AppID, opts page.Opts) (page.Page[*organization.Organization], error) {
	return s.organizationsPage(ctx, bson.M{"app_id": appID.String()}, opts.Normalize())
}

func (s *Store) ListUserOrganizationsPage(ctx context.Context, userID id.UserID, opts page.Opts) (page.Page[*organization.Organization], error) {
	opts = opts.Normalize()
	var members []memberModel
	if err := s.mdb.NewFind(&members).Filter(bson.M{"user_id": userID.String()}).Scan(ctx); err != nil {
		return page.Page[*organization.Organization]{}, fmt.Errorf("authsome/mongo: list user organizations page (members): %w", err)
	}
	if len(members) == 0 {
		return page.Page[*organization.Organization]{Items: []*organization.Organization{}}, nil
	}
	orgIDs := make([]string, 0, len(members))
	for i := range members {
		orgIDs = append(orgIDs, members[i].OrgID)
	}
	return s.organizationsPage(ctx, bson.M{"_id": bson.M{"$in": orgIDs}}, opts)
}

func (s *Store) ListMembersPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Member], error) {
	opts = opts.Normalize()
	var models []memberModel
	err := s.mdb.NewFind(&models).
		Filter(pageFilter(bson.M{"org_id": orgID.String()}, opts.Cursor)).
		Sort(newestFirst).Limit(int64(opts.Limit + 1)).Scan(ctx)
	if err != nil {
		return page.Page[*organization.Member]{}, fmt.Errorf("authsome/mongo: list members page: %w", err)
	}
	items := make([]*organization.Member, 0, len(models))
	for i := range models {
		m, convErr := fromMemberModel(&models[i])
		if convErr != nil {
			return page.Page[*organization.Member]{}, convErr
		}
		items = append(items, m)
	}
	return page.Cut(items, opts.Limit, func(x *organization.Member) string { return x.ID.String() }), nil
}

func (s *Store) ListInvitationsPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Invitation], error) {
	opts = opts.Normalize()
	var models []invitationModel
	err := s.mdb.NewFind(&models).
		Filter(pageFilter(bson.M{"org_id": orgID.String()}, opts.Cursor)).
		Sort(newestFirst).Limit(int64(opts.Limit + 1)).Scan(ctx)
	if err != nil {
		return page.Page[*organization.Invitation]{}, fmt.Errorf("authsome/mongo: list invitations page: %w", err)
	}
	items := make([]*organization.Invitation, 0, len(models))
	for i := range models {
		inv, convErr := fromInvitationModel(&models[i])
		if convErr != nil {
			return page.Page[*organization.Invitation]{}, convErr
		}
		items = append(items, inv)
	}
	return page.Cut(items, opts.Limit, func(x *organization.Invitation) string { return x.ID.String() }), nil
}

func (s *Store) ListTeamsPage(ctx context.Context, orgID id.OrgID, opts page.Opts) (page.Page[*organization.Team], error) {
	opts = opts.Normalize()
	var models []teamModel
	err := s.mdb.NewFind(&models).
		Filter(pageFilter(bson.M{"org_id": orgID.String()}, opts.Cursor)).
		Sort(newestFirst).Limit(int64(opts.Limit + 1)).Scan(ctx)
	if err != nil {
		return page.Page[*organization.Team]{}, fmt.Errorf("authsome/mongo: list teams page: %w", err)
	}
	items := make([]*organization.Team, 0, len(models))
	for i := range models {
		tm, convErr := fromTeamModel(&models[i])
		if convErr != nil {
			return page.Page[*organization.Team]{}, convErr
		}
		items = append(items, tm)
	}
	return page.Cut(items, opts.Limit, func(x *organization.Team) string { return x.ID.String() }), nil
}
