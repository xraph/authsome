package scim

import (
	"strconv"

	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// SCIM 2.0 Resource Types (RFC 7643)
// ──────────────────────────────────────────────────

// UserResource represents a SCIM 2.0 User resource.
type UserResource struct {
	Schemas    []string   `json:"schemas"`
	ID         string     `json:"id,omitempty"`
	ExternalID string     `json:"externalId,omitempty"`
	UserName   string     `json:"userName"`
	Name       Name       `json:"name"`
	Emails     []Email    `json:"emails,omitempty"`
	Active     bool       `json:"active"`
	Meta       *Meta      `json:"meta,omitempty"`
	Groups     []GroupRef `json:"groups,omitempty"`
}

// Name represents a SCIM name component.
type Name struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	Formatted  string `json:"formatted,omitempty"`
}

// Email represents a SCIM email address.
type Email struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary"`
}

// GroupRef is a reference to a SCIM group.
type GroupRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// Meta holds SCIM resource metadata.
type Meta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
}

// GroupResource represents a SCIM 2.0 Group resource.
type GroupResource struct {
	Schemas     []string    `json:"schemas"`
	ID          string      `json:"id,omitempty"`
	ExternalID  string      `json:"externalId,omitempty"`
	DisplayName string      `json:"displayName"`
	Members     []MemberRef `json:"members,omitempty"`
	Meta        *Meta       `json:"meta,omitempty"`
}

// MemberRef is a member reference within a SCIM Group.
type MemberRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// ListResponse is a SCIM list response (RFC 7644 Section 3.4.2).
type ListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    []any    `json:"Resources"`
}

// SCIMError represents a SCIM error response (RFC 7644 Section 3.12).
type SCIMError struct { //nolint:revive // Error conflicts with builtin
	Schemas  []string `json:"schemas"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
}

// PatchOp represents a SCIM PATCH request (RFC 7644 Section 3.5.2).
type PatchOp struct {
	Schemas    []string    `json:"schemas"`
	Operations []Operation `json:"Operations"`
}

// Operation is a single SCIM PATCH operation.
type Operation struct {
	Op    string `json:"op"`
	Path  string `json:"path,omitempty"`
	Value any    `json:"value,omitempty"`
}

// SCIM schema URIs.
const (
	SchemaUser          = "urn:ietf:params:scim:schemas:core:2.0:User"
	SchemaGroup         = "urn:ietf:params:scim:schemas:core:2.0:Group"
	SchemaListResponse  = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	SchemaError         = "urn:ietf:params:scim:api:messages:2.0:Error"
	SchemaPatchOp       = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	SchemaServiceConfig = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	SchemaResourceType  = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
	SchemaSchema        = "urn:ietf:params:scim:schemas:core:2.0:Schema"
)

// PrimaryEmail returns the primary email from the emails list,
// falling back to UserName.
func (u *UserResource) PrimaryEmail() string {
	for _, e := range u.Emails {
		if e.Primary {
			return e.Value
		}
	}
	if len(u.Emails) > 0 {
		return u.Emails[0].Value
	}
	return u.UserName
}

// ──────────────────────────────────────────────────
// Mappers: AuthSome -> SCIM
// ──────────────────────────────────────────────────

// UserToSCIM converts an AuthSome user to a SCIM User resource.
func UserToSCIM(u *user.User, baseURL string) *UserResource {
	scimUser := &UserResource{
		Schemas:  []string{SchemaUser},
		ID:       u.ID.String(),
		UserName: u.Email,
		Name: Name{
			GivenName:  u.FirstName,
			FamilyName: u.LastName,
			Formatted:  u.Name(),
		},
		Active: !u.Banned,
		Emails: []Email{
			{
				Value:   u.Email,
				Type:    "work",
				Primary: true,
			},
		},
		Meta: &Meta{
			ResourceType: "User",
			Created:      u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			LastModified: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			Location:     baseURL + "/Users/" + u.ID.String(),
		},
	}
	return scimUser
}

// TeamToSCIMGroup converts an AuthSome team to a SCIM Group resource.
func TeamToSCIMGroup(t *organization.Team, members []*organization.Member, baseURL string) *GroupResource {
	scimGroup := &GroupResource{
		Schemas:     []string{SchemaGroup},
		ID:          t.ID.String(),
		DisplayName: t.Name,
		Meta: &Meta{
			ResourceType: "Group",
			Created:      t.CreatedAt.Format("2006-01-02T15:04:05Z"),
			LastModified: t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			Location:     baseURL + "/Groups/" + t.ID.String(),
		},
	}

	for _, m := range members {
		scimGroup.Members = append(scimGroup.Members, MemberRef{
			Value: m.UserID.String(),
			Ref:   baseURL + "/Users/" + m.UserID.String(),
		})
	}

	return scimGroup
}

// NewSCIMError creates a SCIM error response.
func NewSCIMError(status int, detail string) *SCIMError {
	return &SCIMError{
		Schemas: []string{SchemaError},
		Detail:  detail,
		Status:  strconv.Itoa(status),
	}
}
