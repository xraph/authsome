package repository

import (
    "context"

    "github.com/uptrace/bun"
    "github.com/rs/xid"

    core "github.com/xraph/authsome/core/organization"
    "github.com/xraph/authsome/schema"
)

// OrganizationRepository is a Bun-backed implementation of core organization repository
type OrganizationRepository struct {
    db *bun.DB
}

func NewOrganizationRepository(db *bun.DB) *OrganizationRepository {
    return &OrganizationRepository{db: db}
}

// ===== Organization =====

func (r *OrganizationRepository) toOrgSchema(o *core.Organization) *schema.Organization {
    return &schema.Organization{
        ID:        o.ID,
        Name:      o.Name,
        Slug:      o.Slug,
        Logo:      o.Logo,
        Metadata:  o.Metadata,
    }
}

func (r *OrganizationRepository) fromOrgSchema(so *schema.Organization) *core.Organization {
    if so == nil { return nil }
    return &core.Organization{
        ID:        so.ID,
        Name:      so.Name,
        Slug:      so.Slug,
        Logo:      so.Logo,
        Metadata:  so.Metadata,
        CreatedAt: so.CreatedAt,
        UpdatedAt: so.UpdatedAt.Time,
    }
}

func (r *OrganizationRepository) CreateOrganization(ctx context.Context, org *core.Organization) error {
    so := r.toOrgSchema(org)
    _, err := r.db.NewInsert().Model(so).Exec(ctx)
    return err
}

func (r *OrganizationRepository) FindOrganizationByID(ctx context.Context, id xid.ID) (*core.Organization, error) {
    so := new(schema.Organization)
    err := r.db.NewSelect().Model(so).Where("id = ?", id).Scan(ctx)
    if err != nil { return nil, err }
    return r.fromOrgSchema(so), nil
}

func (r *OrganizationRepository) FindOrganizationBySlug(ctx context.Context, slug string) (*core.Organization, error) {
    so := new(schema.Organization)
    err := r.db.NewSelect().Model(so).Where("slug = ?", slug).Scan(ctx)
    if err != nil { return nil, err }
    return r.fromOrgSchema(so), nil
}

func (r *OrganizationRepository) UpdateOrganization(ctx context.Context, org *core.Organization) error {
    so := r.toOrgSchema(org)
    _, err := r.db.NewUpdate().Model(so).WherePK().Exec(ctx)
    return err
}

func (r *OrganizationRepository) DeleteOrganization(ctx context.Context, id xid.ID) error {
    _, err := r.db.NewDelete().Model((*schema.Organization)(nil)).Where("id = ?", id).Exec(ctx)
    return err
}

func (r *OrganizationRepository) ListOrganizations(ctx context.Context, limit, offset int) ([]*core.Organization, error) {
    var sos []schema.Organization
    err := r.db.NewSelect().Model(&sos).OrderExpr("created_at DESC").Limit(limit).Offset(offset).Scan(ctx)
    if err != nil { return nil, err }
    res := make([]*core.Organization, 0, len(sos))
    for i := range sos {
        res = append(res, r.fromOrgSchema(&sos[i]))
    }
    return res, nil
}

// CountOrganizations returns total number of organizations
func (r *OrganizationRepository) CountOrganizations(ctx context.Context) (int, error) {
    q := r.db.NewSelect().Model((*schema.Organization)(nil))
    return q.Count(ctx)
}

// ===== Member =====

func (r *OrganizationRepository) toMemberSchema(m *core.Member) *schema.Member {
    return &schema.Member{
        ID:             m.ID,
        OrganizationID: m.OrganizationID,
        UserID:         m.UserID,
        Role:           m.Role,
    }
}

func (r *OrganizationRepository) fromMemberSchema(sm *schema.Member) *core.Member {
    if sm == nil { return nil }
    return &core.Member{
        ID:             sm.ID,
        OrganizationID: sm.OrganizationID,
        UserID:         sm.UserID,
        Role:           sm.Role,
        CreatedAt:      sm.CreatedAt,
        UpdatedAt:      sm.UpdatedAt.Time,
    }
}

func (r *OrganizationRepository) CreateMember(ctx context.Context, member *core.Member) error {
    sm := r.toMemberSchema(member)
    _, err := r.db.NewInsert().Model(sm).Exec(ctx)
    return err
}

func (r *OrganizationRepository) FindMemberByID(ctx context.Context, id xid.ID) (*core.Member, error) {
    sm := new(schema.Member)
    err := r.db.NewSelect().Model(sm).Where("id = ?", id).Scan(ctx)
    if err != nil { return nil, err }
    return r.fromMemberSchema(sm), nil
}

func (r *OrganizationRepository) FindMember(ctx context.Context, orgID, userID xid.ID) (*core.Member, error) {
    sm := new(schema.Member)
    err := r.db.NewSelect().Model(sm).Where("organization_id = ?", orgID).Where("user_id = ?", userID).Scan(ctx)
    if err != nil { return nil, err }
    return r.fromMemberSchema(sm), nil
}

func (r *OrganizationRepository) ListMembers(ctx context.Context, orgID xid.ID, limit, offset int) ([]*core.Member, error) {
    var sms []schema.Member
    q := r.db.NewSelect().Model(&sms).Where("organization_id = ?", orgID).OrderExpr("created_at DESC")
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Scan(ctx); err != nil { return nil, err }
    res := make([]*core.Member, 0, len(sms))
    for i := range sms {
        res = append(res, r.fromMemberSchema(&sms[i]))
    }
    return res, nil
}

// CountMembers returns total number of members in an organization
func (r *OrganizationRepository) CountMembers(ctx context.Context, orgID xid.ID) (int, error) {
    q := r.db.NewSelect().Model((*schema.Member)(nil)).Where("organization_id = ?", orgID)
    return q.Count(ctx)
}

func (r *OrganizationRepository) UpdateMember(ctx context.Context, member *core.Member) error {
    sm := r.toMemberSchema(member)
    _, err := r.db.NewUpdate().Model(sm).WherePK().Exec(ctx)
    return err
}

func (r *OrganizationRepository) DeleteMember(ctx context.Context, id xid.ID) error {
    _, err := r.db.NewDelete().Model((*schema.Member)(nil)).Where("id = ?", id).Exec(ctx)
    return err
}

// ===== Team =====

func (r *OrganizationRepository) toTeamSchema(t *core.Team) *schema.Team {
    return &schema.Team{
        ID:             t.ID,
        OrganizationID: t.OrganizationID,
        Name:           t.Name,
        Description:    t.Description,
    }
}

func (r *OrganizationRepository) fromTeamSchema(st *schema.Team) *core.Team {
    if st == nil { return nil }
    return &core.Team{
        ID:             st.ID,
        OrganizationID: st.OrganizationID,
        Name:           st.Name,
        Description:    st.Description,
        CreatedAt:      st.CreatedAt,
        UpdatedAt:      st.UpdatedAt.Time,
    }
}

func (r *OrganizationRepository) CreateTeam(ctx context.Context, team *core.Team) error {
    st := r.toTeamSchema(team)
    _, err := r.db.NewInsert().Model(st).Exec(ctx)
    return err
}

func (r *OrganizationRepository) FindTeamByID(ctx context.Context, id xid.ID) (*core.Team, error) {
    st := new(schema.Team)
    err := r.db.NewSelect().Model(st).Where("id = ?", id).Scan(ctx)
    if err != nil { return nil, err }
    return r.fromTeamSchema(st), nil
}

func (r *OrganizationRepository) ListTeams(ctx context.Context, orgID xid.ID, limit, offset int) ([]*core.Team, error) {
    var sts []schema.Team
    q := r.db.NewSelect().Model(&sts).Where("organization_id = ?", orgID).OrderExpr("created_at DESC")
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Scan(ctx); err != nil { return nil, err }
    res := make([]*core.Team, 0, len(sts))
    for i := range sts {
        res = append(res, r.fromTeamSchema(&sts[i]))
    }
    return res, nil
}

// CountTeams returns total number of teams in an organization
func (r *OrganizationRepository) CountTeams(ctx context.Context, orgID xid.ID) (int, error) {
    q := r.db.NewSelect().Model((*schema.Team)(nil)).Where("organization_id = ?", orgID)
    return q.Count(ctx)
}

func (r *OrganizationRepository) UpdateTeam(ctx context.Context, team *core.Team) error {
    st := r.toTeamSchema(team)
    _, err := r.db.NewUpdate().Model(st).WherePK().Exec(ctx)
    return err
}

func (r *OrganizationRepository) DeleteTeam(ctx context.Context, id xid.ID) error {
    _, err := r.db.NewDelete().Model((*schema.Team)(nil)).Where("id = ?", id).Exec(ctx)
    return err
}

// ===== Team Member =====

func (r *OrganizationRepository) toTeamMemberSchema(tm *core.TeamMember) *schema.TeamMember {
    return &schema.TeamMember{
        ID:       tm.ID,
        TeamID:   tm.TeamID,
        MemberID: tm.MemberID,
    }
}

func (r *OrganizationRepository) fromTeamMemberSchema(stm *schema.TeamMember) *core.TeamMember {
    if stm == nil { return nil }
    return &core.TeamMember{
        ID:       stm.ID,
        TeamID:   stm.TeamID,
        MemberID: stm.MemberID,
        CreatedAt: stm.CreatedAt,
    }
}

func (r *OrganizationRepository) AddTeamMember(ctx context.Context, tm *core.TeamMember) error {
    stm := r.toTeamMemberSchema(tm)
    _, err := r.db.NewInsert().Model(stm).Exec(ctx)
    return err
}

func (r *OrganizationRepository) RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error {
    _, err := r.db.NewDelete().Model((*schema.TeamMember)(nil)).Where("team_id = ?", teamID).Where("member_id = ?", memberID).Exec(ctx)
    return err
}

func (r *OrganizationRepository) ListTeamMembers(ctx context.Context, teamID xid.ID, limit, offset int) ([]*core.TeamMember, error) {
    var stms []schema.TeamMember
    q := r.db.NewSelect().Model(&stms).Where("team_id = ?", teamID).OrderExpr("created_at DESC")
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Scan(ctx); err != nil { return nil, err }
    res := make([]*core.TeamMember, 0, len(stms))
    for i := range stms {
        res = append(res, r.fromTeamMemberSchema(&stms[i]))
    }
    return res, nil
}

// CountTeamMembers returns total number of members in a team
func (r *OrganizationRepository) CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error) {
    q := r.db.NewSelect().Model((*schema.TeamMember)(nil)).Where("team_id = ?", teamID)
    return q.Count(ctx)
}

// ===== Invitation =====

func (r *OrganizationRepository) toInvitationSchema(inv *core.Invitation) *schema.Invitation {
    return &schema.Invitation{
        ID:             inv.ID,
        OrganizationID: inv.OrganizationID,
        Email:          inv.Email,
        Role:           inv.Role,
        InviterID:      inv.InviterID,
        Token:          inv.Token,
        ExpiresAt:      inv.ExpiresAt,
        AcceptedAt:     inv.AcceptedAt,
        Status:         inv.Status,
    }
}

func (r *OrganizationRepository) fromInvitationSchema(si *schema.Invitation) *core.Invitation {
    if si == nil { return nil }
    return &core.Invitation{
        ID:             si.ID,
        OrganizationID: si.OrganizationID,
        Email:          si.Email,
        Role:           si.Role,
        InviterID:      si.InviterID,
        Token:          si.Token,
        ExpiresAt:      si.ExpiresAt,
        AcceptedAt:     si.AcceptedAt,
        Status:         si.Status,
        CreatedAt:      si.CreatedAt,
    }
}

func (r *OrganizationRepository) CreateInvitation(ctx context.Context, inv *core.Invitation) error {
    si := r.toInvitationSchema(inv)
    _, err := r.db.NewInsert().Model(si).Exec(ctx)
    return err
}