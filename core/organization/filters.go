package organization

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// ListOrganizationsFilter defines filters for listing organizations.
type ListOrganizationsFilter struct {
	pagination.PaginationParams

	AppID         xid.ID
	EnvironmentID xid.ID
}

// Validate validates the filter parameters.
func (f *ListOrganizationsFilter) Validate() error {
	return f.PaginationParams.Validate()
}

// GetLimit returns the limit for pagination.
func (f *ListOrganizationsFilter) GetLimit() int {
	return f.PaginationParams.GetLimit()
}

// GetOffset returns the offset for pagination.
func (f *ListOrganizationsFilter) GetOffset() int {
	return f.PaginationParams.GetOffset()
}

// ListMembersFilter defines filters for listing organization members.
type ListMembersFilter struct {
	pagination.PaginationParams

	OrganizationID xid.ID
	Role           *string // Filter by role (owner, admin, member)
	Status         *string // Filter by status (active, suspended, pending)
}

// Validate validates the filter parameters.
func (f *ListMembersFilter) Validate() error {
	if err := f.PaginationParams.Validate(); err != nil {
		return err
	}

	if f.Role != nil && !IsValidRole(*f.Role) {
		return InvalidRole(*f.Role)
	}

	if f.Status != nil && !IsValidStatus(*f.Status) {
		return InvalidStatus(*f.Status)
	}

	return nil
}

// ListTeamsFilter defines filters for listing teams.
type ListTeamsFilter struct {
	pagination.PaginationParams

	OrganizationID xid.ID
}

// Validate validates the filter parameters.
func (f *ListTeamsFilter) Validate() error {
	return f.PaginationParams.Validate()
}

// ListTeamMembersFilter defines filters for listing team members.
type ListTeamMembersFilter struct {
	pagination.PaginationParams

	TeamID xid.ID
}

// Validate validates the filter parameters.
func (f *ListTeamMembersFilter) Validate() error {
	return f.PaginationParams.Validate()
}

// ListInvitationsFilter defines filters for listing invitations.
type ListInvitationsFilter struct {
	pagination.PaginationParams

	OrganizationID xid.ID
	Status         *string // Filter by status (pending, accepted, expired, etc.)
}

// Validate validates the filter parameters.
func (f *ListInvitationsFilter) Validate() error {
	if err := f.PaginationParams.Validate(); err != nil {
		return err
	}

	if f.Status != nil && !IsValidInvitationStatus(*f.Status) {
		return InvalidStatus(*f.Status)
	}

	return nil
}
