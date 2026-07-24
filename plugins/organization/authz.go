package organization

import (
	"errors"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/store"
)

// orgRoleRank ranks membership roles so they can be compared (higher is more
// privileged): owner > admin > member.
func orgRoleRank(r organization.MemberRole) int {
	switch r {
	case organization.RoleOwner:
		return 3
	case organization.RoleAdmin:
		return 2
	case organization.RoleMember:
		return 1
	default:
		return 0
	}
}

// requireOrgRole verifies the authenticated caller is a member of orgID holding
// at least minRole, returning their membership record. Failure modes:
//   - no authenticated identity on the request → 401
//   - caller is not a member of the org        → 404 (so org existence is not
//     disclosed to outsiders)
//   - caller's role is below minRole           → 403
func (p *Plugin) requireOrgRole(ctx forge.Context, orgID id.OrgID, minRole organization.MemberRole) (*organization.Member, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	member, err := p.store.GetMemberByUserAndOrg(ctx.Context(), userID, orgID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, forge.NotFound("organization not found")
		}
		return nil, p.mapError(err)
	}
	if orgRoleRank(member.Role) < orgRoleRank(minRole) {
		return nil, forge.Forbidden("insufficient organization role")
	}
	return member, nil
}

// assertMemberInOrg verifies the member identified by memberID belongs to
// orgID, returning 404 otherwise. This stops a caller authorized in one org
// from mutating a member row that lives in another org via a crafted path.
func (p *Plugin) assertMemberInOrg(ctx forge.Context, memberID id.MemberID, orgID id.OrgID) error {
	m, err := p.store.GetMember(ctx.Context(), memberID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return forge.NotFound("member not found")
		}
		return p.mapError(err)
	}
	if m.OrgID.String() != orgID.String() {
		return forge.NotFound("member not found")
	}
	return nil
}

// assertTeamInOrg verifies the team identified by teamID belongs to orgID,
// returning 404 otherwise.
func (p *Plugin) assertTeamInOrg(ctx forge.Context, teamID id.TeamID, orgID id.OrgID) error {
	tm, err := p.store.GetTeam(ctx.Context(), teamID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return forge.NotFound("team not found")
		}
		return p.mapError(err)
	}
	if tm.OrgID.String() != orgID.String() {
		return forge.NotFound("team not found")
	}
	return nil
}
