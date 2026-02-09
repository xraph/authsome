package rbac

import "github.com/rs/xid"

// Permission represents a named permission; policies can reference or embed permissions.
type Permission struct {
	ID             xid.ID  `json:"id"`
	OrganizationID *xid.ID `json:"organizationId"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
}
