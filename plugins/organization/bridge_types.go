package organization

import (
	"time"
)

// =============================================================================
// Bridge Function Input/Output Types
// =============================================================================

// Common input fields.
type BridgeAppInput struct {
	AppID string `json:"appId"`
}

// GetOrganizationsInput is the input for bridgeGetOrganizations.
type GetOrganizationsInput struct {
	AppID  string `json:"appId"`
	Search string `json:"search,omitempty"`
	Page   int    `json:"page,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// GetOrganizationsResult is the output for bridgeGetOrganizations.
type GetOrganizationsResult struct {
	Data       []OrganizationSummaryDTO `json:"data"`
	Pagination PaginationInfo           `json:"pagination"`
	Stats      OrganizationStatsDTO     `json:"stats"`
}

// GetOrganizationInput is the input for bridgeGetOrganization.
type GetOrganizationInput struct {
	AppID string `json:"appId"`
	OrgID string `json:"orgId"`
}

// GetOrganizationResult is the output for bridgeGetOrganization.
type GetOrganizationResult struct {
	Organization OrganizationDetailDTO `json:"organization"`
	UserRole     string                `json:"userRole"`
	Stats        OrgDetailStatsDTO     `json:"stats"`
}

// CreateOrganizationInput is the input for bridgeCreateOrganization.
type CreateOrganizationInput struct {
	AppID    string         `json:"appId"`
	Name     string         `json:"name"`
	Slug     string         `json:"slug,omitempty"`
	Logo     string         `json:"logo,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CreateOrganizationResult is the output for bridgeCreateOrganization.
type CreateOrganizationResult struct {
	Organization OrganizationDetailDTO `json:"organization"`
}

// UpdateOrganizationInput is the input for bridgeUpdateOrganization.
type UpdateOrganizationInput struct {
	AppID    string         `json:"appId"`
	OrgID    string         `json:"orgId"`
	Name     string         `json:"name,omitempty"`
	Logo     string         `json:"logo,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// UpdateOrganizationResult is the output for bridgeUpdateOrganization.
type UpdateOrganizationResult struct {
	Organization OrganizationDetailDTO `json:"organization"`
}

// DeleteOrganizationInput is the input for bridgeDeleteOrganization.
type DeleteOrganizationInput struct {
	AppID string `json:"appId"`
	OrgID string `json:"orgId"`
}

// DeleteOrganizationResult is the output for bridgeDeleteOrganization.
type DeleteOrganizationResult struct {
	Success bool `json:"success"`
}

// GetMembersInput is the input for bridgeGetMembers.
type GetMembersInput struct {
	AppID  string `json:"appId"`
	OrgID  string `json:"orgId"`
	Search string `json:"search,omitempty"`
	Page   int    `json:"page,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// GetMembersResult is the output for bridgeGetMembers.
type GetMembersResult struct {
	Data       []MemberDTO    `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
	CanManage  bool           `json:"canManage"`
}

// InviteMemberInput is the input for bridgeInviteMember.
type InviteMemberInput struct {
	AppID string `json:"appId"`
	OrgID string `json:"orgId"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// InviteMemberResult is the output for bridgeInviteMember.
type InviteMemberResult struct {
	Invitation InvitationDTO `json:"invitation"`
}

// UpdateMemberRoleInput is the input for bridgeUpdateMemberRole.
type UpdateMemberRoleInput struct {
	AppID    string `json:"appId"`
	OrgID    string `json:"orgId"`
	MemberID string `json:"memberId"`
	Role     string `json:"role"`
}

// UpdateMemberRoleResult is the output for bridgeUpdateMemberRole.
type UpdateMemberRoleResult struct {
	Member MemberDTO `json:"member"`
}

// RemoveMemberInput is the input for bridgeRemoveMember.
type RemoveMemberInput struct {
	AppID    string `json:"appId"`
	OrgID    string `json:"orgId"`
	MemberID string `json:"memberId"`
}

// RemoveMemberResult is the output for bridgeRemoveMember.
type RemoveMemberResult struct {
	Success bool `json:"success"`
}

// GetTeamsInput is the input for bridgeGetTeams.
type GetTeamsInput struct {
	AppID  string `json:"appId"`
	OrgID  string `json:"orgId"`
	Search string `json:"search,omitempty"`
	Page   int    `json:"page,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// GetTeamsResult is the output for bridgeGetTeams.
type GetTeamsResult struct {
	Data       []TeamDTO      `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
	CanManage  bool           `json:"canManage"`
}

// CreateTeamInput is the input for bridgeCreateTeam.
type CreateTeamInput struct {
	AppID       string         `json:"appId"`
	OrgID       string         `json:"orgId"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// CreateTeamResult is the output for bridgeCreateTeam.
type CreateTeamResult struct {
	Team TeamDTO `json:"team"`
}

// UpdateTeamInput is the input for bridgeUpdateTeam.
type UpdateTeamInput struct {
	AppID       string         `json:"appId"`
	OrgID       string         `json:"orgId"`
	TeamID      string         `json:"teamId"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// UpdateTeamResult is the output for bridgeUpdateTeam.
type UpdateTeamResult struct {
	Team TeamDTO `json:"team"`
}

// DeleteTeamInput is the input for bridgeDeleteTeam.
type DeleteTeamInput struct {
	AppID  string `json:"appId"`
	OrgID  string `json:"orgId"`
	TeamID string `json:"teamId"`
}

// DeleteTeamResult is the output for bridgeDeleteTeam.
type DeleteTeamResult struct {
	Success bool `json:"success"`
}

// GetInvitationsInput is the input for bridgeGetInvitations.
type GetInvitationsInput struct {
	AppID  string `json:"appId"`
	OrgID  string `json:"orgId"`
	Status string `json:"status,omitempty"` // pending, accepted, declined, expired, all
	Page   int    `json:"page,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// GetInvitationsResult is the output for bridgeGetInvitations.
type GetInvitationsResult struct {
	Data       []InvitationDTO `json:"data"`
	Pagination PaginationInfo  `json:"pagination"`
}

// CancelInvitationInput is the input for bridgeCancelInvitation.
type CancelInvitationInput struct {
	AppID    string `json:"appId"`
	OrgID    string `json:"orgId"`
	InviteID string `json:"inviteId"`
}

// CancelInvitationResult is the output for bridgeCancelInvitation.
type CancelInvitationResult struct {
	Success bool `json:"success"`
}

// GetExtensionDataInput is the input for bridgeGetExtensionData.
type GetExtensionDataInput struct {
	AppID string `json:"appId"`
	OrgID string `json:"orgId"`
}

// GetExtensionDataResult is the output for bridgeGetExtensionData.
type GetExtensionDataResult struct {
	Widgets    []WidgetDataDTO    `json:"widgets"`
	Tabs       []TabDataDTO       `json:"tabs"`
	Actions    []ActionDataDTO    `json:"actions"`
	QuickLinks []QuickLinkDataDTO `json:"quickLinks"`
}

// =============================================================================
// DTO Types
// =============================================================================

// OrganizationSummaryDTO is a summary DTO for organization list.
type OrganizationSummaryDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Logo        string    `json:"logo"`
	MemberCount int64     `json:"memberCount"`
	TeamCount   int64     `json:"teamCount"`
	UserRole    string    `json:"userRole"`
	CreatedAt   time.Time `json:"createdAt"`
}

// OrganizationDetailDTO is a detailed DTO for organization.
type OrganizationDetailDTO struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	Logo      string         `json:"logo"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// OrganizationStatsDTO contains stats for organizations list.
type OrganizationStatsDTO struct {
	TotalOrganizations int64 `json:"totalOrganizations"`
	TotalMembers       int64 `json:"totalMembers"`
	TotalTeams         int64 `json:"totalTeams"`
}

// OrgDetailStatsDTO contains stats for a single organization.
type OrgDetailStatsDTO struct {
	MemberCount     int64 `json:"memberCount"`
	TeamCount       int64 `json:"teamCount"`
	InvitationCount int64 `json:"invitationCount"`
}

// MemberDTO is a DTO for organization member.
type MemberDTO struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	UserEmail string    `json:"userEmail"`
	UserName  string    `json:"userName"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joinedAt"`
}

// TeamDTO is a DTO for organization team.
type TeamDTO struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	MemberCount int64          `json:"memberCount"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"createdAt"`
}

// InvitationDTO is a DTO for organization invitation.
type InvitationDTO struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	InvitedBy   string    `json:"invitedBy"`
	InviterName string    `json:"inviterName"`
	ExpiresAt   time.Time `json:"expiresAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

// PaginationInfo contains pagination metadata.
type PaginationInfo struct {
	CurrentPage int   `json:"currentPage"`
	PageSize    int   `json:"pageSize"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int   `json:"totalPages"`
}

// WidgetDataDTO contains widget data for extension system.
type WidgetDataDTO struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Icon         string `json:"icon"` // Icon name or HTML
	Order        int    `json:"order"`
	Size         int    `json:"size"`
	Content      string `json:"content"` // HTML content
	RequireAdmin bool   `json:"requireAdmin"`
}

// TabDataDTO contains tab data for extension system.
type TabDataDTO struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Path         string `json:"path"`
	Icon         string `json:"icon"` // Icon name or HTML
	Order        int    `json:"order"`
	RequireAdmin bool   `json:"requireAdmin"`
}

// ActionDataDTO contains action data for extension system.
type ActionDataDTO struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Icon         string `json:"icon"`   // Icon name or HTML
	Action       string `json:"action"` // JavaScript action
	Style        string `json:"style"`  // primary, secondary, danger
	Order        int    `json:"order"`
	RequireAdmin bool   `json:"requireAdmin"`
}

// QuickLinkDataDTO contains quick link data for extension system.
type QuickLinkDataDTO struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	URL          string `json:"url"`
	Icon         string `json:"icon"` // Icon name or HTML
	Order        int    `json:"order"`
	RequireAdmin bool   `json:"requireAdmin"`
}

// GetRoleTemplatesInput is the input for bridgeGetRoleTemplates.
type GetRoleTemplatesInput struct {
	AppID string `json:"appId"`
}

// GetRoleTemplatesResult is the output for bridgeGetRoleTemplates.
type GetRoleTemplatesResult struct {
	Templates []RoleTemplateDTO `json:"templates"`
}

// GetRoleTemplateInput is the input for bridgeGetRoleTemplate.
type GetRoleTemplateInput struct {
	AppID      string `json:"appId"`
	TemplateID string `json:"templateId"`
}

// GetRoleTemplateResult is the output for bridgeGetRoleTemplate.
type GetRoleTemplateResult struct {
	Template RoleTemplateDTO `json:"template"`
}

// CreateRoleTemplateInput is the input for bridgeCreateRoleTemplate.
type CreateRoleTemplateInput struct {
	AppID       string   `json:"appId"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// CreateRoleTemplateResult is the output for bridgeCreateRoleTemplate.
type CreateRoleTemplateResult struct {
	Template RoleTemplateDTO `json:"template"`
}

// UpdateRoleTemplateInput is the input for bridgeUpdateRoleTemplate.
type UpdateRoleTemplateInput struct {
	AppID       string   `json:"appId"`
	TemplateID  string   `json:"templateId"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// UpdateRoleTemplateResult is the output for bridgeUpdateRoleTemplate.
type UpdateRoleTemplateResult struct {
	Template RoleTemplateDTO `json:"template"`
}

// DeleteRoleTemplateInput is the input for bridgeDeleteRoleTemplate.
type DeleteRoleTemplateInput struct {
	AppID      string `json:"appId"`
	TemplateID string `json:"templateId"`
}

// DeleteRoleTemplateResult is the output for bridgeDeleteRoleTemplate.
type DeleteRoleTemplateResult struct {
	Success bool `json:"success"`
}

// RoleTemplateDTO is a DTO for role template.
type RoleTemplateDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
