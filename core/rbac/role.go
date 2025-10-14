package rbac

import "github.com/rs/xid"

// Role represents a named role, optionally scoped to an organization
type Role struct {
	ID             xid.ID  `json:"id"`
	OrganizationID *xid.ID `json:"organizationId"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
}
