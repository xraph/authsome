package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/store"
)

// ──────────────────────────────────────────────────
// Organization CRUD
// ──────────────────────────────────────────────────

// CreateOrganization persists a new organization.
func (s *Store) CreateOrganization(ctx context.Context, o *organization.Organization) error {
	m := toOrganizationModel(o)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create organization: %w", err)
	}

	return nil
}

// GetOrganization returns an organization by ID.
func (s *Store) GetOrganization(ctx context.Context, orgID id.OrgID) (*organization.Organization, error) {
	var m organizationModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": orgID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get organization: %w", err)
	}

	return fromOrganizationModel(&m)
}

// GetOrganizationBySlug returns an organization by app ID and slug.
func (s *Store) GetOrganizationBySlug(ctx context.Context, appID id.AppID, slug string) (*organization.Organization, error) {
	var m organizationModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"app_id": appID.String(),
			"slug":   slug,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get organization by slug: %w", err)
	}

	return fromOrganizationModel(&m)
}

// UpdateOrganization modifies an existing organization.
func (s *Store) UpdateOrganization(ctx context.Context, o *organization.Organization) error {
	m := toOrganizationModel(o)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update organization: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteOrganization removes an organization.
func (s *Store) DeleteOrganization(ctx context.Context, orgID id.OrgID) error {
	res, err := s.mdb.NewDelete((*organizationModel)(nil)).
		Filter(bson.M{"_id": orgID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete organization: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListOrganizations returns all organizations for an app, ordered by creation date descending.
func (s *Store) ListOrganizations(ctx context.Context, appID id.AppID) ([]*organization.Organization, error) {
	var models []organizationModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"app_id": appID.String()}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list organizations: %w", err)
	}

	return toOrganizationSlice(models)
}

// ListUserOrganizations returns all organizations a user belongs to.
func (s *Store) ListUserOrganizations(ctx context.Context, userID id.UserID) ([]*organization.Organization, error) {
	// First, find all org IDs the user is a member of.
	var members []memberModel

	err := s.mdb.NewFind(&members).
		Filter(bson.M{"user_id": userID.String()}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list user organizations (members): %w", err)
	}

	if len(members) == 0 {
		return []*organization.Organization{}, nil
	}

	orgIDs := make([]string, 0, len(members))
	for i := range members {
		orgIDs = append(orgIDs, members[i].OrgID)
	}

	// Then, fetch the organizations.
	var models []organizationModel

	err = s.mdb.NewFind(&models).
		Filter(bson.M{"_id": bson.M{"$in": orgIDs}}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list user organizations: %w", err)
	}

	return toOrganizationSlice(models)
}

func toOrganizationSlice(models []organizationModel) ([]*organization.Organization, error) {
	result := make([]*organization.Organization, 0, len(models))

	for i := range models {
		o, err := fromOrganizationModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, o)
	}

	return result, nil
}

// ──────────────────────────────────────────────────
// Member CRUD
// ──────────────────────────────────────────────────

// CreateMember persists a new organization member.
func (s *Store) CreateMember(ctx context.Context, mem *organization.Member) error {
	m := toMemberModel(mem)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create member: %w", err)
	}

	return nil
}

// GetMember returns a member by ID.
func (s *Store) GetMember(ctx context.Context, memberID id.MemberID) (*organization.Member, error) {
	var m memberModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": memberID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get member: %w", err)
	}

	return fromMemberModel(&m)
}

// GetMemberByUserAndOrg returns a member by user ID and org ID.
func (s *Store) GetMemberByUserAndOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) (*organization.Member, error) {
	var m memberModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"user_id": userID.String(),
			"org_id":  orgID.String(),
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get member by user and org: %w", err)
	}

	return fromMemberModel(&m)
}

// UpdateMember modifies an existing member.
func (s *Store) UpdateMember(ctx context.Context, mem *organization.Member) error {
	m := toMemberModel(mem)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update member: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteMember removes a member.
func (s *Store) DeleteMember(ctx context.Context, memberID id.MemberID) error {
	res, err := s.mdb.NewDelete((*memberModel)(nil)).
		Filter(bson.M{"_id": memberID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete member: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListMembers returns all members of an organization, ordered by creation date ascending.
func (s *Store) ListMembers(ctx context.Context, orgID id.OrgID) ([]*organization.Member, error) {
	var models []memberModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"org_id": orgID.String()}).
		Sort(bson.D{{Key: "created_at", Value: 1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list members: %w", err)
	}

	result := make([]*organization.Member, 0, len(models))

	for i := range models {
		mem, err := fromMemberModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, mem)
	}

	return result, nil
}

// ──────────────────────────────────────────────────
// Invitation CRUD
// ──────────────────────────────────────────────────

// CreateInvitation persists a new invitation.
func (s *Store) CreateInvitation(ctx context.Context, inv *organization.Invitation) error {
	m := toInvitationModel(inv)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create invitation: %w", err)
	}

	return nil
}

// GetInvitation returns an invitation by ID.
func (s *Store) GetInvitation(ctx context.Context, invID id.InvitationID) (*organization.Invitation, error) {
	var m invitationModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": invID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get invitation: %w", err)
	}

	return fromInvitationModel(&m)
}

// GetInvitationByToken returns an invitation by its token.
func (s *Store) GetInvitationByToken(ctx context.Context, token string) (*organization.Invitation, error) {
	var m invitationModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"token": token}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get invitation by token: %w", err)
	}

	return fromInvitationModel(&m)
}

// UpdateInvitation modifies an existing invitation.
func (s *Store) UpdateInvitation(ctx context.Context, inv *organization.Invitation) error {
	m := toInvitationModel(inv)

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update invitation: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListInvitations returns all invitations for an organization, ordered by creation date descending.
func (s *Store) ListInvitations(ctx context.Context, orgID id.OrgID) ([]*organization.Invitation, error) {
	var models []invitationModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"org_id": orgID.String()}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list invitations: %w", err)
	}

	result := make([]*organization.Invitation, 0, len(models))

	for i := range models {
		inv, err := fromInvitationModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, inv)
	}

	return result, nil
}

// ──────────────────────────────────────────────────
// Team CRUD
// ──────────────────────────────────────────────────

// CreateTeam persists a new team.
func (s *Store) CreateTeam(ctx context.Context, t *organization.Team) error {
	m := toTeamModel(t)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create team: %w", err)
	}

	return nil
}

// GetTeam returns a team by ID.
func (s *Store) GetTeam(ctx context.Context, teamID id.TeamID) (*organization.Team, error) {
	var m teamModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": teamID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get team: %w", err)
	}

	return fromTeamModel(&m)
}

// UpdateTeam modifies an existing team.
func (s *Store) UpdateTeam(ctx context.Context, t *organization.Team) error {
	m := toTeamModel(t)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update team: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteTeam removes a team.
func (s *Store) DeleteTeam(ctx context.Context, teamID id.TeamID) error {
	res, err := s.mdb.NewDelete((*teamModel)(nil)).
		Filter(bson.M{"_id": teamID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete team: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListTeams returns all teams for an organization, ordered by name ascending.
func (s *Store) ListTeams(ctx context.Context, orgID id.OrgID) ([]*organization.Team, error) {
	var models []teamModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"org_id": orgID.String()}).
		Sort(bson.D{{Key: "name", Value: 1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list teams: %w", err)
	}

	result := make([]*organization.Team, 0, len(models))

	for i := range models {
		t, err := fromTeamModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, t)
	}

	return result, nil
}
