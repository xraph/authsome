package organization

// ---------------------------------------------------------------------------
// Organization requests
// ---------------------------------------------------------------------------

// CreateOrgRequest binds the body for POST /orgs.
type CreateOrgRequest struct {
	AppID string `json:"app_id,omitempty" description:"Application ID (optional, uses default)"`
	Name  string `json:"name" description:"Organization name"`
	Slug  string `json:"slug" description:"URL-safe slug"`
	Logo  string `json:"logo,omitempty" description:"Organization logo URL"`
}

// ListOrgsRequest is an empty request for GET /orgs (user from context).
type ListOrgsRequest struct {
	Limit  int    `query:"limit" description:"Page size (default 50, max 200)" optional:"true"`
	Cursor string `query:"cursor" description:"Cursor from the previous page's next_cursor" optional:"true"`
}

// GetOrgRequest binds the path for GET /orgs/:orgId.
type GetOrgRequest struct {
	OrgID string `path:"orgId" description:"Organization identifier"`
}

// UpdateOrgRequest binds path + body for PATCH /orgs/:orgId.
type UpdateOrgRequest struct {
	OrgID string  `path:"orgId" description:"Organization identifier"`
	Name  *string `json:"name,omitempty" description:"Organization name"`
	Logo  *string `json:"logo,omitempty" description:"Organization logo URL"`
}

// DeleteOrgRequest binds the path for DELETE /orgs/:orgId.
type DeleteOrgRequest struct {
	OrgID string `path:"orgId" description:"Organization identifier"`
}

// ---------------------------------------------------------------------------
// Member requests
// ---------------------------------------------------------------------------

// ListMembersRequest binds the path for GET /orgs/:orgId/members.
type ListMembersRequest struct {
	OrgID  string `path:"orgId" description:"Organization identifier"`
	Limit  int    `query:"limit" description:"Page size (default 50, max 200)" optional:"true"`
	Cursor string `query:"cursor" description:"Cursor from the previous page's next_cursor" optional:"true"`
}

// AddMemberRequest binds path + body for POST /orgs/:orgId/members.
type AddMemberRequest struct {
	OrgID  string `path:"orgId" description:"Organization identifier"`
	UserID string `json:"user_id" description:"User identifier to add"`
	Role   string `json:"role,omitempty" description:"Member role (owner, admin, member)"`
}

// RemoveMemberRequest binds the path for DELETE /orgs/:orgId/members/:memberId.
type RemoveMemberRequest struct {
	OrgID    string `path:"orgId" description:"Organization identifier"`
	MemberID string `path:"memberId" description:"Member identifier"`
}

// UpdateMemberRequest binds path + body for PATCH /orgs/:orgId/members/:memberId.
type UpdateMemberRequest struct {
	OrgID    string `path:"orgId" description:"Organization identifier"`
	MemberID string `path:"memberId" description:"Member identifier"`
	Role     string `json:"role" description:"New member role (owner, admin, member)"`
}

// ---------------------------------------------------------------------------
// Invitation requests
// ---------------------------------------------------------------------------

// CreateInvitationRequest binds path + body for POST /orgs/:orgId/invitations.
type CreateInvitationRequest struct {
	OrgID string `path:"orgId" description:"Organization identifier"`
	Email string `json:"email" description:"Email address to invite"`
	Role  string `json:"role,omitempty" description:"Member role (owner, admin, member)"`
}

// ListInvitationsRequest binds the path for GET /orgs/:orgId/invitations.
type ListInvitationsRequest struct {
	OrgID  string `path:"orgId" description:"Organization identifier"`
	Limit  int    `query:"limit" description:"Page size (default 50, max 200)" optional:"true"`
	Cursor string `query:"cursor" description:"Cursor from the previous page's next_cursor" optional:"true"`
}

// AcceptInvitationRequest binds the body for POST /orgs/invitations/accept.
type AcceptInvitationRequest struct {
	Token string `json:"token" description:"Invitation token"`
}

// DeclineInvitationRequest binds the body for POST /orgs/invitations/decline.
type DeclineInvitationRequest struct {
	Token string `json:"token" description:"Invitation token"`
}

// ---------------------------------------------------------------------------
// Team requests
// ---------------------------------------------------------------------------

// CreateTeamRequest binds path + body for POST /orgs/:orgId/teams.
type CreateTeamRequest struct {
	OrgID string `path:"orgId" description:"Organization identifier"`
	Name  string `json:"name" description:"Team name"`
	Slug  string `json:"slug" description:"URL-safe team slug"`
}

// ListTeamsRequest binds the path for GET /orgs/:orgId/teams.
type ListTeamsRequest struct {
	OrgID  string `path:"orgId" description:"Organization identifier"`
	Limit  int    `query:"limit" description:"Page size (default 50, max 200)" optional:"true"`
	Cursor string `query:"cursor" description:"Cursor from the previous page's next_cursor" optional:"true"`
}

// GetTeamRequest binds the path for GET /orgs/:orgId/teams/:teamId.
type GetTeamRequest struct {
	OrgID  string `path:"orgId" description:"Organization identifier"`
	TeamID string `path:"teamId" description:"Team identifier"`
}

// UpdateTeamRequest binds path + body for PATCH /orgs/:orgId/teams/:teamId.
type UpdateTeamRequest struct {
	OrgID  string  `path:"orgId" description:"Organization identifier"`
	TeamID string  `path:"teamId" description:"Team identifier"`
	Name   *string `json:"name,omitempty" description:"Team name"`
	Slug   *string `json:"slug,omitempty" description:"URL-safe team slug"`
}

// DeleteTeamRequest binds the path for DELETE /orgs/:orgId/teams/:teamId.
type DeleteTeamRequest struct {
	OrgID  string `path:"orgId" description:"Organization identifier"`
	TeamID string `path:"teamId" description:"Team identifier"`
}

// ---------------------------------------------------------------------------
// Slug check request
// ---------------------------------------------------------------------------

// CheckSlugRequest binds query params for GET /orgs/check-slug.
type CheckSlugRequest struct {
	AppID string `query:"app_id" description:"Application ID"`
	Slug  string `query:"slug" description:"Slug to check"`
}

// ---------------------------------------------------------------------------
// Admin requests
// ---------------------------------------------------------------------------

// AdminListOrgsRequest binds query params for GET /admin/orgs.
type AdminListOrgsRequest struct {
	AppID string `query:"app_id" description:"Application ID"`
}

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// StatusResponse is a generic status response.
type StatusResponse struct {
	Status string `json:"status" description:"Operation status"`
}

// OrgListResponse wraps a list of organizations.
type OrgListResponse struct {
	Organizations any `json:"organizations" description:"List of organizations"`
	// NextCursor is set when more remain; pass it as cursor to continue.
	NextCursor string `json:"next_cursor,omitempty" description:"Cursor for the next page, absent on the last"`
}

// MemberListResponse wraps a list of members.
type MemberListResponse struct {
	Members any `json:"members" description:"List of members"`
	// NextCursor is set when more remain; pass it as cursor to continue.
	NextCursor string `json:"next_cursor,omitempty" description:"Cursor for the next page, absent on the last"`
}

// InvitationListResponse wraps a list of invitations.
type InvitationListResponse struct {
	Invitations any `json:"invitations" description:"List of invitations"`
	// NextCursor is set when more remain; pass it as cursor to continue.
	NextCursor string `json:"next_cursor,omitempty" description:"Cursor for the next page, absent on the last"`
}

// TeamListResponse wraps a list of teams.
type TeamListResponse struct {
	Teams any `json:"teams" description:"List of teams"`
	// NextCursor is set when more remain; pass it as cursor to continue.
	NextCursor string `json:"next_cursor,omitempty" description:"Cursor for the next page, absent on the last"`
}

// SlugAvailableResponse reports whether a slug is available.
type SlugAvailableResponse struct {
	Available bool `json:"available" description:"Whether the slug is available"`
}
