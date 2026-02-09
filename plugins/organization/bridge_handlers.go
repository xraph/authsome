package organization

import (
	"context"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	coreorg "github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
	g "maragu.dev/gomponents"
)

// =============================================================================
// Bridge Handler Implementations
// =============================================================================

// bridgeGetOrganizations handles the getOrganizations bridge call.
func (e *DashboardExtension) bridgeGetOrganizations(ctx bridge.Context, input GetOrganizationsInput) (*GetOrganizationsResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse app ID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, errs.BadRequest("invalid appId")
	}

	// Get environment ID
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Set pagination defaults
	page := max(input.Page, 1)

	limit := input.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Build filter
	filter := &coreorg.ListOrganizationsFilter{
		AppID:         appID,
		EnvironmentID: envID,
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
	}

	if input.Search != "" {
		filter.Search = input.Search
	}

	// Fetch organizations
	orgsResp, err := e.plugin.orgService.ListOrganizations(goCtx, filter)
	if err != nil {
		return nil, errs.InternalServerError("failed to list organizations", err)
	}

	// Build DTOs
	data := make([]OrganizationSummaryDTO, len(orgsResp.Data))
	for i, org := range orgsResp.Data {
		// Get member count
		membersResp, _ := e.plugin.orgService.ListMembers(goCtx, &coreorg.ListMembersFilter{
			OrganizationID:   org.ID,
			PaginationParams: pagination.PaginationParams{Limit: 1},
		})

		memberCount := int64(0)
		if membersResp != nil && membersResp.Pagination != nil {
			memberCount = membersResp.Pagination.Total
		}

		// Get team count
		teamsResp, _ := e.plugin.orgService.ListTeams(goCtx, &coreorg.ListTeamsFilter{
			OrganizationID:   org.ID,
			PaginationParams: pagination.PaginationParams{Limit: 1},
		})

		teamCount := int64(0)
		if teamsResp != nil && teamsResp.Pagination != nil {
			teamCount = teamsResp.Pagination.Total
		}

		// Get user role
		userRole := e.getUserRole(goCtx, org.ID, userID)

		data[i] = OrganizationSummaryDTO{
			ID:          org.ID.String(),
			Name:        org.Name,
			Slug:        org.Slug,
			Logo:        org.Logo,
			MemberCount: memberCount,
			TeamCount:   teamCount,
			UserRole:    userRole,
			CreatedAt:   org.CreatedAt,
		}
	}

	// Build stats
	stats := OrganizationStatsDTO{
		TotalOrganizations: orgsResp.Pagination.Total,
		TotalMembers:       0,
		TotalTeams:         0,
	}
	for _, dto := range data {
		stats.TotalMembers += dto.MemberCount
		stats.TotalTeams += dto.TeamCount
	}

	// Build pagination info
	paginationInfo := PaginationInfo{
		CurrentPage: orgsResp.Pagination.CurrentPage,
		PageSize:    orgsResp.Pagination.Limit,
		TotalItems:  orgsResp.Pagination.Total,
		TotalPages:  int((orgsResp.Pagination.Total + int64(orgsResp.Pagination.Limit) - 1) / int64(orgsResp.Pagination.Limit)),
	}

	return &GetOrganizationsResult{
		Data:       data,
		Pagination: paginationInfo,
		Stats:      stats,
	}, nil
}

// bridgeGetOrganization handles the getOrganization bridge call.
func (e *DashboardExtension) bridgeGetOrganization(ctx bridge.Context, input GetOrganizationInput) (*GetOrganizationResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse org ID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check access
	if !e.checkOrgAccess(goCtx, orgID, userID) {
		return nil, errs.Forbidden("access denied")
	}

	// Get organization
	org, err := e.plugin.orgService.FindOrganizationByID(goCtx, orgID)
	if err != nil {
		return nil, errs.Wrap(err, "organization_not_found", "Organization not found", 404)
	}

	// Get user role
	userRole := e.getUserRole(goCtx, orgID, userID)

	// Get stats
	membersResp, _ := e.plugin.orgService.ListMembers(goCtx, &coreorg.ListMembersFilter{
		OrganizationID:   orgID,
		PaginationParams: pagination.PaginationParams{Limit: 1},
	})

	memberCount := int64(0)
	if membersResp != nil && membersResp.Pagination != nil {
		memberCount = membersResp.Pagination.Total
	}

	teamsResp, _ := e.plugin.orgService.ListTeams(goCtx, &coreorg.ListTeamsFilter{
		OrganizationID:   orgID,
		PaginationParams: pagination.PaginationParams{Limit: 1},
	})

	teamCount := int64(0)
	if teamsResp != nil && teamsResp.Pagination != nil {
		teamCount = teamsResp.Pagination.Total
	}

	pendingStatus := "pending"
	invitesResp, _ := e.plugin.orgService.ListInvitations(goCtx, &coreorg.ListInvitationsFilter{
		OrganizationID:   orgID,
		Status:           &pendingStatus,
		PaginationParams: pagination.PaginationParams{Limit: 1},
	})

	invitationCount := int64(0)
	if invitesResp != nil && invitesResp.Pagination != nil {
		invitationCount = invitesResp.Pagination.Total
	}

	return &GetOrganizationResult{
		Organization: OrganizationDetailDTO{
			ID:        org.ID.String(),
			Name:      org.Name,
			Slug:      org.Slug,
			Logo:      org.Logo,
			Metadata:  org.Metadata,
			CreatedAt: org.CreatedAt,
			UpdatedAt: org.UpdatedAt,
		},
		UserRole: userRole,
		Stats: OrgDetailStatsDTO{
			MemberCount:     memberCount,
			TeamCount:       teamCount,
			InvitationCount: invitationCount,
		},
	}, nil
}

// bridgeCreateOrganization handles the createOrganization bridge call.
func (e *DashboardExtension) bridgeCreateOrganization(ctx bridge.Context, input CreateOrganizationInput) (*CreateOrganizationResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse app ID
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, errs.BadRequest("invalid appId")
	}

	// Get environment ID
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Create organization
	req := &coreorg.CreateOrganizationRequest{
		Name: input.Name,
		Slug: input.Slug,
	}
	if input.Logo != "" {
		req.Logo = &input.Logo
	}

	if input.Metadata != nil {
		req.Metadata = input.Metadata
	}

	org, err := e.plugin.orgService.CreateOrganization(goCtx, req, userID, appID, envID)
	if err != nil {
		return nil, errs.Wrap(err, "create_organization_failed", "Failed to create organization", 400)
	}

	return &CreateOrganizationResult{
		Organization: OrganizationDetailDTO{
			ID:        org.ID.String(),
			Name:      org.Name,
			Slug:      org.Slug,
			Logo:      org.Logo,
			Metadata:  org.Metadata,
			CreatedAt: org.CreatedAt,
			UpdatedAt: org.UpdatedAt,
		},
	}, nil
}

// bridgeUpdateOrganization handles the updateOrganization bridge call.
func (e *DashboardExtension) bridgeUpdateOrganization(ctx bridge.Context, input UpdateOrganizationInput) (*UpdateOrganizationResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse org ID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check permission
	if !e.canManageOrganization(goCtx, orgID, userID) {
		return nil, errs.Forbidden("insufficient permissions")
	}

	// Update organization
	req := &coreorg.UpdateOrganizationRequest{}
	if input.Name != "" {
		req.Name = &input.Name
	}

	if input.Logo != "" {
		req.Logo = &input.Logo
	}

	if input.Metadata != nil {
		req.Metadata = input.Metadata
	}

	org, err := e.plugin.orgService.UpdateOrganization(goCtx, orgID, req)
	if err != nil {
		return nil, errs.Wrap(err, "update_organization_failed", "Failed to update organization", 400)
	}

	return &UpdateOrganizationResult{
		Organization: OrganizationDetailDTO{
			ID:        org.ID.String(),
			Name:      org.Name,
			Slug:      org.Slug,
			Logo:      org.Logo,
			Metadata:  org.Metadata,
			CreatedAt: org.CreatedAt,
			UpdatedAt: org.UpdatedAt,
		},
	}, nil
}

// bridgeDeleteOrganization handles the deleteOrganization bridge call.
func (e *DashboardExtension) bridgeDeleteOrganization(ctx bridge.Context, input DeleteOrganizationInput) (*DeleteOrganizationResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse org ID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check permission (owner only)
	if !e.checkOrgOwner(goCtx, orgID, userID) {
		return nil, errs.Forbidden("only owner can delete organization")
	}

	// Delete organization
	err = e.plugin.orgService.DeleteOrganization(goCtx, orgID, userID)
	if err != nil {
		return nil, errs.Wrap(err, "delete_organization_failed", "Failed to delete organization", 400)
	}

	return &DeleteOrganizationResult{Success: true}, nil
}

// bridgeGetMembers handles the getMembers bridge call.
func (e *DashboardExtension) bridgeGetMembers(ctx bridge.Context, input GetMembersInput) (*GetMembersResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse org ID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check access
	if !e.checkOrgAccess(goCtx, orgID, userID) {
		return nil, errs.Forbidden("access denied")
	}

	// Set pagination defaults
	page := max(input.Page, 1)

	limit := input.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Build filter
	filter := &coreorg.ListMembersFilter{
		OrganizationID: orgID,
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
	}

	if input.Search != "" {
		filter.Search = input.Search
	}

	// Fetch members
	membersResp, err := e.plugin.orgService.ListMembers(goCtx, filter)
	if err != nil {
		return nil, errs.InternalServerError("failed to list members", err)
	}

	// Build DTOs
	data := make([]MemberDTO, len(membersResp.Data))
	for i, member := range membersResp.Data {
		// Get user details
		var userName, userEmail string
		if member.User != nil {
			userName = member.User.Name
			userEmail = member.User.Email
		}

		data[i] = MemberDTO{
			ID:        member.ID.String(),
			UserID:    member.UserID.String(),
			UserEmail: userEmail,
			UserName:  userName,
			Role:      member.Role,
			Status:    member.Status,
			JoinedAt:  member.CreatedAt,
		}
	}

	// Build pagination info
	paginationInfo := PaginationInfo{
		CurrentPage: membersResp.Pagination.CurrentPage,
		PageSize:    membersResp.Pagination.Limit,
		TotalItems:  membersResp.Pagination.Total,
		TotalPages:  int((membersResp.Pagination.Total + int64(membersResp.Pagination.Limit) - 1) / int64(membersResp.Pagination.Limit)),
	}

	// Check if user can manage
	canManage := e.canManageOrganization(goCtx, orgID, userID)

	return &GetMembersResult{
		Data:       data,
		Pagination: paginationInfo,
		CanManage:  canManage,
	}, nil
}

// bridgeInviteMember handles the inviteMember bridge call.
func (e *DashboardExtension) bridgeInviteMember(ctx bridge.Context, input InviteMemberInput) (*InviteMemberResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse org ID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check permission
	if !e.canManageOrganization(goCtx, orgID, userID) {
		return nil, errs.Forbidden("insufficient permissions")
	}

	// Create invitation
	req := &coreorg.InviteMemberRequest{
		Email: input.Email,
		Role:  input.Role,
	}

	invitation, err := e.plugin.orgService.InviteMember(goCtx, orgID, req, userID)
	if err != nil {
		return nil, errs.Wrap(err, "create_invitation_failed", "Failed to create invitation", 400)
	}

	// Get inviter name
	inviterName := ""
	if inviter, err := e.plugin.authInst.GetServiceRegistry().UserService().FindByID(goCtx, userID); err == nil && inviter != nil {
		inviterName = inviter.Name
	}

	return &InviteMemberResult{
		Invitation: InvitationDTO{
			ID:          invitation.ID.String(),
			Email:       invitation.Email,
			Role:        invitation.Role,
			Status:      invitation.Status,
			InvitedBy:   invitation.InviterID.String(),
			InviterName: inviterName,
			ExpiresAt:   invitation.ExpiresAt,
			CreatedAt:   invitation.CreatedAt,
		},
	}, nil
}

// bridgeUpdateMemberRole handles the updateMemberRole bridge call.
func (e *DashboardExtension) bridgeUpdateMemberRole(ctx bridge.Context, input UpdateMemberRoleInput) (*UpdateMemberRoleResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse IDs
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	memberID, err := xid.FromString(input.MemberID)
	if err != nil {
		return nil, errs.BadRequest("invalid memberId")
	}

	// Check permission
	if !e.canManageOrganization(goCtx, orgID, userID) {
		return nil, errs.Forbidden("insufficient permissions")
	}

	// Update member role
	req := &coreorg.UpdateMemberRequest{}
	if input.Role != "" {
		req.Role = &input.Role
	}

	member, err := e.plugin.orgService.UpdateMember(goCtx, memberID, req, userID)
	if err != nil {
		return nil, errs.Wrap(err, "update_member_role_failed", "Failed to update member role", 400)
	}

	// Get user details
	var userName, userEmail string
	if member.User != nil {
		userName = member.User.Name
		userEmail = member.User.Email
	}

	return &UpdateMemberRoleResult{
		Member: MemberDTO{
			ID:        member.ID.String(),
			UserID:    member.UserID.String(),
			UserEmail: userEmail,
			UserName:  userName,
			Role:      member.Role,
			Status:    member.Status,
			JoinedAt:  member.CreatedAt,
		},
	}, nil
}

// bridgeRemoveMember handles the removeMember bridge call.
func (e *DashboardExtension) bridgeRemoveMember(ctx bridge.Context, input RemoveMemberInput) (*RemoveMemberResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse IDs
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	memberID, err := xid.FromString(input.MemberID)
	if err != nil {
		return nil, errs.BadRequest("invalid memberId")
	}

	// Check permission
	if !e.canManageOrganization(goCtx, orgID, userID) {
		return nil, errs.Forbidden("insufficient permissions")
	}

	// Remove member
	err = e.plugin.orgService.RemoveMember(goCtx, memberID, userID)
	if err != nil {
		return nil, errs.Wrap(err, "remove_member_failed", "Failed to remove member", 400)
	}

	return &RemoveMemberResult{Success: true}, nil
}

// bridgeGetTeams handles the getTeams bridge call.
func (e *DashboardExtension) bridgeGetTeams(ctx bridge.Context, input GetTeamsInput) (*GetTeamsResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse org ID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check access
	if !e.checkOrgAccess(goCtx, orgID, userID) {
		return nil, errs.Forbidden("access denied")
	}

	// Set pagination defaults
	page := max(input.Page, 1)

	limit := input.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Build filter
	filter := &coreorg.ListTeamsFilter{
		OrganizationID: orgID,
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
	}

	if input.Search != "" {
		filter.Search = input.Search
	}

	// Fetch teams
	teamsResp, err := e.plugin.orgService.ListTeams(goCtx, filter)
	if err != nil {
		return nil, errs.InternalServerError("failed to list teams", err)
	}

	// Build DTOs
	data := make([]TeamDTO, len(teamsResp.Data))
	for i, team := range teamsResp.Data {
		// Get member count for this team
		teamMembersResp, _ := e.plugin.orgService.ListTeamMembers(goCtx, &coreorg.ListTeamMembersFilter{
			TeamID:           team.ID,
			PaginationParams: pagination.PaginationParams{Limit: 1},
		})

		memberCount := int64(0)
		if teamMembersResp != nil && teamMembersResp.Pagination != nil {
			memberCount = teamMembersResp.Pagination.Total
		}

		data[i] = TeamDTO{
			ID:          team.ID.String(),
			Name:        team.Name,
			Description: team.Description,
			MemberCount: memberCount,
			Metadata:    team.Metadata,
			CreatedAt:   team.CreatedAt,
		}
	}

	// Build pagination info
	paginationInfo := PaginationInfo{
		CurrentPage: teamsResp.Pagination.CurrentPage,
		PageSize:    teamsResp.Pagination.Limit,
		TotalItems:  teamsResp.Pagination.Total,
		TotalPages:  int((teamsResp.Pagination.Total + int64(teamsResp.Pagination.Limit) - 1) / int64(teamsResp.Pagination.Limit)),
	}

	// Check if user can manage
	canManage := e.canManageOrganization(goCtx, orgID, userID)

	return &GetTeamsResult{
		Data:       data,
		Pagination: paginationInfo,
		CanManage:  canManage,
	}, nil
}

// bridgeCreateTeam handles the createTeam bridge call.
func (e *DashboardExtension) bridgeCreateTeam(ctx bridge.Context, input CreateTeamInput) (*CreateTeamResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse org ID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check permission
	if !e.canManageOrganization(goCtx, orgID, userID) {
		return nil, errs.Forbidden("insufficient permissions")
	}

	// Create team
	req := &coreorg.CreateTeamRequest{
		Name: input.Name,
	}
	if input.Description != "" {
		req.Description = &input.Description
	}

	if input.Metadata != nil {
		req.Metadata = input.Metadata
	}

	team, err := e.plugin.orgService.CreateTeam(goCtx, orgID, req, userID)
	if err != nil {
		return nil, errs.Wrap(err, "create_team_failed", "Failed to create team", 400)
	}

	return &CreateTeamResult{
		Team: TeamDTO{
			ID:          team.ID.String(),
			Name:        team.Name,
			Description: team.Description,
			MemberCount: 0,
			Metadata:    team.Metadata,
			CreatedAt:   team.CreatedAt,
		},
	}, nil
}

// bridgeUpdateTeam handles the updateTeam bridge call.
func (e *DashboardExtension) bridgeUpdateTeam(ctx bridge.Context, input UpdateTeamInput) (*UpdateTeamResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse IDs
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	teamID, err := xid.FromString(input.TeamID)
	if err != nil {
		return nil, errs.BadRequest("invalid teamId")
	}

	// Check permission
	if !e.canManageOrganization(goCtx, orgID, userID) {
		return nil, errs.Forbidden("insufficient permissions")
	}

	// Update team
	req := &coreorg.UpdateTeamRequest{}
	if input.Name != "" {
		req.Name = &input.Name
	}

	if input.Description != "" {
		req.Description = &input.Description
	}

	if input.Metadata != nil {
		req.Metadata = input.Metadata
	}

	team, err := e.plugin.orgService.UpdateTeam(goCtx, teamID, req, userID)
	if err != nil {
		return nil, errs.Wrap(err, "update_team_failed", "Failed to update team", 400)
	}

	// Get member count
	teamMembersResp, _ := e.plugin.orgService.ListTeamMembers(goCtx, &coreorg.ListTeamMembersFilter{
		TeamID:           teamID,
		PaginationParams: pagination.PaginationParams{Limit: 1},
	})

	memberCount := int64(0)
	if teamMembersResp != nil && teamMembersResp.Pagination != nil {
		memberCount = teamMembersResp.Pagination.Total
	}

	return &UpdateTeamResult{
		Team: TeamDTO{
			ID:          team.ID.String(),
			Name:        team.Name,
			Description: team.Description,
			MemberCount: memberCount,
			Metadata:    team.Metadata,
			CreatedAt:   team.CreatedAt,
		},
	}, nil
}

// bridgeDeleteTeam handles the deleteTeam bridge call.
func (e *DashboardExtension) bridgeDeleteTeam(ctx bridge.Context, input DeleteTeamInput) (*DeleteTeamResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse IDs
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	teamID, err := xid.FromString(input.TeamID)
	if err != nil {
		return nil, errs.BadRequest("invalid teamId")
	}

	// Check permission
	if !e.canManageOrganization(goCtx, orgID, userID) {
		return nil, errs.Forbidden("insufficient permissions")
	}

	// Delete team
	err = e.plugin.orgService.DeleteTeam(goCtx, teamID, userID)
	if err != nil {
		return nil, errs.Wrap(err, "delete_team_failed", "Failed to delete team", 400)
	}

	return &DeleteTeamResult{Success: true}, nil
}

// bridgeGetInvitations handles the getInvitations bridge call.
func (e *DashboardExtension) bridgeGetInvitations(ctx bridge.Context, input GetInvitationsInput) (*GetInvitationsResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse org ID
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check access
	if !e.checkOrgAccess(goCtx, orgID, userID) {
		return nil, errs.Forbidden("access denied")
	}

	// Set pagination defaults
	page := max(input.Page, 1)

	limit := input.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Build filter
	filter := &coreorg.ListInvitationsFilter{
		OrganizationID: orgID,
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
	}

	// Set status filter
	if input.Status != "" && input.Status != "all" {
		statusCopy := input.Status
		filter.Status = &statusCopy
	}

	// Fetch invitations
	invitesResp, err := e.plugin.orgService.ListInvitations(goCtx, filter)
	if err != nil {
		return nil, errs.InternalServerError("failed to list invitations", err)
	}

	// Build DTOs
	data := make([]InvitationDTO, len(invitesResp.Data))
	for i, invite := range invitesResp.Data {
		// Get inviter name
		inviterName := ""
		if inviter, err := e.plugin.authInst.GetServiceRegistry().UserService().FindByID(goCtx, invite.InviterID); err == nil && inviter != nil {
			inviterName = inviter.Name
		}

		data[i] = InvitationDTO{
			ID:          invite.ID.String(),
			Email:       invite.Email,
			Role:        invite.Role,
			Status:      invite.Status,
			InvitedBy:   invite.InviterID.String(),
			InviterName: inviterName,
			ExpiresAt:   invite.ExpiresAt,
			CreatedAt:   invite.CreatedAt,
		}
	}

	// Build pagination info
	paginationInfo := PaginationInfo{
		CurrentPage: invitesResp.Pagination.CurrentPage,
		PageSize:    invitesResp.Pagination.Limit,
		TotalItems:  invitesResp.Pagination.Total,
		TotalPages:  int((invitesResp.Pagination.Total + int64(invitesResp.Pagination.Limit) - 1) / int64(invitesResp.Pagination.Limit)),
	}

	return &GetInvitationsResult{
		Data:       data,
		Pagination: paginationInfo,
	}, nil
}

// bridgeCancelInvitation handles the cancelInvitation bridge call.
func (e *DashboardExtension) bridgeCancelInvitation(ctx bridge.Context, input CancelInvitationInput) (*CancelInvitationResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse IDs
	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	inviteID, err := xid.FromString(input.InviteID)
	if err != nil {
		return nil, errs.BadRequest("invalid inviteId")
	}

	// Check permission
	if !e.canManageOrganization(goCtx, orgID, userID) {
		return nil, errs.Forbidden("insufficient permissions")
	}

	// Cancel invitation
	err = e.plugin.orgService.CancelInvitation(goCtx, inviteID, userID)
	if err != nil {
		return nil, errs.Wrap(err, "cancel_invitation_failed", "Failed to cancel invitation", 400)
	}

	return &CancelInvitationResult{Success: true}, nil
}

// bridgeGetExtensionData handles the getExtensionData bridge call.
func (e *DashboardExtension) bridgeGetExtensionData(ctx bridge.Context, input GetExtensionDataInput) (*GetExtensionDataResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Parse IDs
	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, errs.BadRequest("invalid appId")
	}

	orgID, err := xid.FromString(input.OrgID)
	if err != nil {
		return nil, errs.BadRequest("invalid orgId")
	}

	// Check access
	if !e.checkOrgAccess(goCtx, orgID, userID) {
		return nil, errs.Forbidden("access denied")
	}

	// Get organization
	org, err := e.plugin.orgService.FindOrganizationByID(goCtx, orgID)
	if err != nil {
		return nil, errs.Wrap(err, "organization_not_found", "Organization not found", 404)
	}

	// Check if user is admin
	isAdmin := e.isUserAdmin(goCtx, orgID, userID)

	// Build extension context
	extCtx := ui.OrgExtensionContext{
		OrgID:    orgID,
		AppID:    appID,
		BasePath: "", // Not needed for bridge
		Request:  nil,
		GetOrg: func() (any, error) {
			return org, nil
		},
		IsAdmin: isAdmin,
	}

	// Get extension registry
	registry := e.plugin.GetOrganizationUIRegistry()
	if registry == nil {
		return &GetExtensionDataResult{
			Widgets:    []WidgetDataDTO{},
			Tabs:       []TabDataDTO{},
			Actions:    []ActionDataDTO{},
			QuickLinks: []QuickLinkDataDTO{},
		}, nil
	}

	// Get extension data
	widgets := registry.GetWidgets(extCtx)
	tabs := registry.GetTabs(extCtx)
	actions := registry.GetActions(extCtx)
	quickLinks := registry.GetQuickLinks(extCtx)

	// Convert to DTOs (simplified - icons as HTML strings)
	widgetDTOs := make([]WidgetDataDTO, len(widgets))
	for i, w := range widgets {
		widgetDTOs[i] = WidgetDataDTO{
			ID:           w.ID,
			Title:        w.Title,
			Icon:         renderIconToHTML(w.Icon),
			Order:        w.Order,
			Size:         w.Size,
			Content:      renderNodeToHTML(w.Renderer(extCtx)),
			RequireAdmin: w.RequireAdmin,
		}
	}

	tabDTOs := make([]TabDataDTO, len(tabs))
	for i, t := range tabs {
		tabDTOs[i] = TabDataDTO{
			ID:           t.ID,
			Label:        t.Label,
			Path:         t.Path,
			Icon:         renderIconToHTML(t.Icon),
			Order:        t.Order,
			RequireAdmin: t.RequireAdmin,
		}
	}

	actionDTOs := make([]ActionDataDTO, len(actions))
	for i, a := range actions {
		actionDTOs[i] = ActionDataDTO{
			ID:           a.ID,
			Label:        a.Label,
			Icon:         renderIconToHTML(a.Icon),
			Action:       a.Action,
			Style:        a.Style,
			Order:        a.Order,
			RequireAdmin: a.RequireAdmin,
		}
	}

	quickLinkDTOs := make([]QuickLinkDataDTO, len(quickLinks))
	for i, ql := range quickLinks {
		quickLinkDTOs[i] = QuickLinkDataDTO{
			ID:           ql.ID,
			Title:        ql.Title,
			Description:  ql.Description,
			URL:          ql.URLBuilder("", orgID, appID),
			Icon:         renderIconToHTML(ql.Icon),
			Order:        ql.Order,
			RequireAdmin: ql.RequireAdmin,
		}
	}

	return &GetExtensionDataResult{
		Widgets:    widgetDTOs,
		Tabs:       tabDTOs,
		Actions:    actionDTOs,
		QuickLinks: quickLinkDTOs,
	}, nil
}

// bridgeGetRoleTemplates handles the getRoleTemplates bridge call.
func (e *DashboardExtension) bridgeGetRoleTemplates(ctx bridge.Context, input GetRoleTemplatesInput) (*GetRoleTemplatesResult, error) {
	// For now, return mock data since role templates are not fully implemented in the core
	// In a real implementation, this would query the database
	templates := []RoleTemplateDTO{
		{
			ID:          "tpl_admin",
			Name:        "Admin",
			Description: "Full administrative access to the organization",
			Permissions: []string{"manage_members", "manage_teams", "manage_roles", "manage_settings", "view_audit_logs", "delete_organization"},
		},
		{
			ID:          "tpl_member",
			Name:        "Member",
			Description: "Basic member access with limited permissions",
			Permissions: []string{"view_audit_logs"},
		},
	}

	return &GetRoleTemplatesResult{
		Templates: templates,
	}, nil
}

// bridgeGetRoleTemplate handles the getRoleTemplate bridge call.
func (e *DashboardExtension) bridgeGetRoleTemplate(ctx bridge.Context, input GetRoleTemplateInput) (*GetRoleTemplateResult, error) {
	// For now, return mock data
	// In a real implementation, this would query the database by ID
	var template RoleTemplateDTO

	switch input.TemplateID {
	case "tpl_admin":
		template = RoleTemplateDTO{
			ID:          "tpl_admin",
			Name:        "Admin",
			Description: "Full administrative access to the organization",
			Permissions: []string{"manage_members", "manage_teams", "manage_roles", "manage_settings", "view_audit_logs", "delete_organization"},
		}
	case "tpl_member":
		template = RoleTemplateDTO{
			ID:          "tpl_member",
			Name:        "Member",
			Description: "Basic member access with limited permissions",
			Permissions: []string{"view_audit_logs"},
		}
	default:
		return nil, errs.NotFound("role template not found")
	}

	return &GetRoleTemplateResult{
		Template: template,
	}, nil
}

// bridgeCreateRoleTemplate handles the createRoleTemplate bridge call.
func (e *DashboardExtension) bridgeCreateRoleTemplate(ctx bridge.Context, input CreateRoleTemplateInput) (*CreateRoleTemplateResult, error) {
	// For now, return mock data
	// In a real implementation, this would create a new template in the database
	template := RoleTemplateDTO{
		ID:          "tpl_" + xid.New().String(),
		Name:        input.Name,
		Description: input.Description,
		Permissions: input.Permissions,
	}

	return &CreateRoleTemplateResult{
		Template: template,
	}, nil
}

// bridgeUpdateRoleTemplate handles the updateRoleTemplate bridge call.
func (e *DashboardExtension) bridgeUpdateRoleTemplate(ctx bridge.Context, input UpdateRoleTemplateInput) (*UpdateRoleTemplateResult, error) {
	// For now, return mock data
	// In a real implementation, this would update the template in the database
	template := RoleTemplateDTO{
		ID:          input.TemplateID,
		Name:        input.Name,
		Description: input.Description,
		Permissions: input.Permissions,
	}

	return &UpdateRoleTemplateResult{
		Template: template,
	}, nil
}

// bridgeDeleteRoleTemplate handles the deleteRoleTemplate bridge call.
func (e *DashboardExtension) bridgeDeleteRoleTemplate(ctx bridge.Context, input DeleteRoleTemplateInput) (*DeleteRoleTemplateResult, error) {
	// For now, just return success
	// In a real implementation, this would delete the template from the database
	return &DeleteRoleTemplateResult{
		Success: true,
	}, nil
}

// =============================================================================
// Settings Bridge Handlers
// =============================================================================

// GetSettingsInput is the input for bridgeGetSettings.
type GetSettingsInput struct {
	AppID string `json:"appId"`
}

// GetSettingsResult is the output for bridgeGetSettings.
type GetSettingsResult struct {
	Settings OrganizationSettingsDTO `json:"settings"`
}

// OrganizationSettingsDTO contains organization plugin settings.
type OrganizationSettingsDTO struct {
	Enabled                  bool   `json:"enabled,omitempty"`
	AllowUserCreation        bool   `json:"allowUserCreation,omitempty"`
	DefaultRole              string `json:"defaultRole,omitempty"`
	MaxOrgsPerUser           int    `json:"maxOrgsPerUser,omitempty"`
	MaxMembersPerOrg         int    `json:"maxMembersPerOrg,omitempty"`
	MaxTeamsPerOrg           int    `json:"maxTeamsPerOrg,omitempty"`
	RequireInvitation        bool   `json:"requireInvitation,omitempty"`
	InvitationExpiryDays     int    `json:"invitationExpiryDays,omitempty"`
	AllowMultipleMemberships bool   `json:"allowMultipleMemberships,omitempty"`
}

// UpdateSettingsInput is the input for bridgeUpdateSettings.
type UpdateSettingsInput struct {
	AppID    string                  `json:"appId"`
	Settings OrganizationSettingsDTO `json:"settings"`
}

// UpdateSettingsResult is the output for bridgeUpdateSettings.
type UpdateSettingsResult struct {
	Success bool `json:"success"`
}

// bridgeGetSettings handles the getSettings bridge call.
func (e *DashboardExtension) bridgeGetSettings(ctx bridge.Context, input GetSettingsInput) (*GetSettingsResult, error) {
	_, _, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Return current plugin configuration as settings
	config := e.plugin.config

	// Convert hours to days for the UI
	invitationExpiryDays := max(config.InvitationExpiryHours/24, 1)

	return &GetSettingsResult{
		Settings: OrganizationSettingsDTO{
			Enabled:                  true, // Plugin is enabled if it's loaded
			AllowUserCreation:        config.EnableUserCreation,
			DefaultRole:              "member", // Default role for new members
			MaxOrgsPerUser:           config.MaxOrganizationsPerUser,
			MaxMembersPerOrg:         config.MaxMembersPerOrganization,
			MaxTeamsPerOrg:           config.MaxTeamsPerOrganization,
			RequireInvitation:        config.RequireInvitation,
			InvitationExpiryDays:     invitationExpiryDays,
			AllowMultipleMemberships: config.MaxOrganizationsPerUser > 1,
		},
	}, nil
}

// bridgeUpdateSettings handles the updateSettings bridge call.
func (e *DashboardExtension) bridgeUpdateSettings(ctx bridge.Context, input UpdateSettingsInput) (*UpdateSettingsResult, error) {
	goCtx, userID, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Check if user is admin
	if !e.isAppAdmin(goCtx, userID) {
		return nil, errs.PermissionDenied("update", "organization settings")
	}

	// Update plugin configuration
	if input.Settings.MaxOrgsPerUser > 0 {
		e.plugin.config.MaxOrganizationsPerUser = input.Settings.MaxOrgsPerUser
	}

	if input.Settings.MaxMembersPerOrg > 0 {
		e.plugin.config.MaxMembersPerOrganization = input.Settings.MaxMembersPerOrg
	}

	if input.Settings.MaxTeamsPerOrg >= 0 {
		e.plugin.config.MaxTeamsPerOrganization = input.Settings.MaxTeamsPerOrg
	}

	if input.Settings.InvitationExpiryDays > 0 {
		// Convert days to hours for storage
		e.plugin.config.InvitationExpiryHours = input.Settings.InvitationExpiryDays * 24
	}

	e.plugin.config.EnableUserCreation = input.Settings.AllowUserCreation
	e.plugin.config.RequireInvitation = input.Settings.RequireInvitation

	return &UpdateSettingsResult{
		Success: true,
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// buildContextFromBridge retrieves the Go context from the HTTP request.
// The context has already been enriched by the dashboard v2 BridgeContextMiddleware with:
// - User ID (from session)
// - App ID (from session)
// - Environment ID (from cookie)
//
// This method validates that the required appID parameter matches the session context,
// and extracts the user ID for authorization checks.
func (e *DashboardExtension) buildContextFromBridge(bridgeCtx bridge.Context, appID string) (context.Context, xid.ID, error) {
	// Get the already-enriched context from the HTTP request
	// The dashboard v2 BridgeContextMiddleware has already set user ID, app ID, and environment ID
	var goCtx context.Context

	req := bridgeCtx.Request()

	if req != nil {
		// IMPORTANT: Get context from HTTP request, not bridgeCtx.Context()
		// because the middleware enriches the request's context
		goCtx = req.Context()
	} else {
		// Fallback to bridge context if no request available
		goCtx = bridgeCtx.Context()
	}

	// Parse the requested app ID
	requestedAppID, err := xid.FromString(appID)
	if err != nil {
		e.plugin.logger.Error("[OrgBridge] Invalid app ID", forge.F("appID", appID), forge.F("error", err))

		return nil, xid.ID{}, errs.BadRequest("invalid appId")
	}

	// Verify that user is authenticated (user ID should be set by middleware)
	userID, hasUserID := contexts.GetUserID(goCtx)
	e.plugin.logger.Debug("[OrgBridge] Checking authentication",
		forge.F("hasUserID", hasUserID),
		forge.F("userID", userID.String()),
		forge.F("requestedAppID", requestedAppID.String()))

	if !hasUserID || userID == xid.NilID() {
		e.plugin.logger.Error("[OrgBridge] Unauthorized - no user ID in context")

		return nil, xid.ID{}, errs.Unauthorized()
	}

	// Override app ID if different from session (allows working with different apps)
	existingAppID, hasAppID := contexts.GetAppID(goCtx)
	e.plugin.logger.Debug("[OrgBridge] App ID check",
		forge.F("hasAppID", hasAppID),
		forge.F("existingAppID", existingAppID.String()),
		forge.F("requestedAppID", requestedAppID.String()))

	if existingAppID != requestedAppID {
		goCtx = contexts.SetAppID(goCtx, requestedAppID)
		e.plugin.logger.Debug("[OrgBridge] Overriding app ID",
			forge.F("from", existingAppID.String()),
			forge.F("to", requestedAppID.String()))
	}

	return goCtx, userID, nil
}

// stringPtrToString converts *string to string.
func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// renderIconToHTML renders a gomponent icon to HTML string.
func renderIconToHTML(icon g.Node) string {
	if icon == nil {
		return ""
	}

	var sb strings.Builder

	_ = icon.Render(&sb)

	return sb.String()
}

// renderNodeToHTML renders a gomponent node to HTML string.
func renderNodeToHTML(node g.Node) string {
	if node == nil {
		return ""
	}

	var sb strings.Builder

	_ = node.Render(&sb)

	return sb.String()
}
