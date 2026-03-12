package scim

import (
	"strconv"

	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// SCIM 2.0 Resource Types (RFC 7643)
// ──────────────────────────────────────────────────

// SCIMUserResource represents a SCIM 2.0 User resource.
type SCIMUserResource struct {
	Schemas    []string       `json:"schemas"`
	ID         string         `json:"id,omitempty"`
	ExternalID string         `json:"externalId,omitempty"`
	UserName   string         `json:"userName"`
	Name       SCIMName       `json:"name"`
	Emails     []SCIMEmail    `json:"emails,omitempty"`
	Active     bool           `json:"active"`
	Meta       *SCIMMeta      `json:"meta,omitempty"`
	Groups     []SCIMGroupRef `json:"groups,omitempty"`
}

// SCIMName represents a SCIM name component.
type SCIMName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	Formatted  string `json:"formatted,omitempty"`
}

// SCIMEmail represents a SCIM email address.
type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary"`
}

// SCIMGroupRef is a reference to a SCIM group.
type SCIMGroupRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// SCIMMeta holds SCIM resource metadata.
type SCIMMeta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
}

// SCIMGroupResource represents a SCIM 2.0 Group resource.
type SCIMGroupResource struct {
	Schemas     []string         `json:"schemas"`
	ID          string           `json:"id,omitempty"`
	ExternalID  string           `json:"externalId,omitempty"`
	DisplayName string           `json:"displayName"`
	Members     []SCIMMemberRef  `json:"members,omitempty"`
	Meta        *SCIMMeta        `json:"meta,omitempty"`
}

// SCIMMemberRef is a member reference within a SCIM Group.
type SCIMMemberRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// SCIMListResponse is a SCIM list response (RFC 7644 Section 3.4.2).
type SCIMListResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	StartIndex   int         `json:"startIndex"`
	ItemsPerPage int         `json:"itemsPerPage"`
	Resources    []any       `json:"Resources"`
}

// SCIMError represents a SCIM error response (RFC 7644 Section 3.12).
type SCIMError struct {
	Schemas  []string `json:"schemas"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
}

// SCIMPatchOp represents a SCIM PATCH request (RFC 7644 Section 3.5.2).
type SCIMPatchOp struct {
	Schemas    []string        `json:"schemas"`
	Operations []SCIMOperation `json:"Operations"`
}

// SCIMOperation is a single SCIM PATCH operation.
type SCIMOperation struct {
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
func (u *SCIMUserResource) PrimaryEmail() string {
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
func UserToSCIM(u *user.User, baseURL string) *SCIMUserResource {
	scimUser := &SCIMUserResource{
		Schemas:  []string{SchemaUser},
		ID:       u.ID.String(),
		UserName: u.Email,
		Name: SCIMName{
			GivenName:  u.FirstName,
			FamilyName: u.LastName,
			Formatted:  u.Name(),
		},
		Active: !u.Banned,
		Emails: []SCIMEmail{
			{
				Value:   u.Email,
				Type:    "work",
				Primary: true,
			},
		},
		Meta: &SCIMMeta{
			ResourceType: "User",
			Created:      u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			LastModified: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			Location:     baseURL + "/Users/" + u.ID.String(),
		},
	}
	return scimUser
}

// TeamToSCIMGroup converts an AuthSome team to a SCIM Group resource.
func TeamToSCIMGroup(t *organization.Team, members []*organization.Member, baseURL string) *SCIMGroupResource {
	scimGroup := &SCIMGroupResource{
		Schemas:     []string{SchemaGroup},
		ID:          t.ID.String(),
		DisplayName: t.Name,
		Meta: &SCIMMeta{
			ResourceType: "Group",
			Created:      t.CreatedAt.Format("2006-01-02T15:04:05Z"),
			LastModified: t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			Location:     baseURL + "/Groups/" + t.ID.String(),
		},
	}

	for _, m := range members {
		scimGroup.Members = append(scimGroup.Members, SCIMMemberRef{
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
