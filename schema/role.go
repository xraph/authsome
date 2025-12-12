package schema

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Role table
type Role struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:roles,alias:r"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)"`
	AppID          *xid.ID `bun:"app_id,type:varchar(20)"`          // App-scoped roles
	EnvironmentID  *xid.ID `bun:"environment_id,type:varchar(20)"`  // Environment-scoped roles
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)"` // Org-scoped roles (NULL = app-level template)
	Name           string  `bun:"name,notnull"`                     // Slug/identifier (e.g., "workspace_owner")
	DisplayName    string  `bun:"display_name,notnull"`             // Human-readable name (e.g., "Workspace Owner")
	Description    string  `bun:"description"`
	IsTemplate     bool    `bun:"is_template,notnull,default:false"`   // Marks roles as templates for cloning
	IsOwnerRole    bool    `bun:"is_owner_role,notnull,default:false"` // Marks the default owner role for new orgs
	TemplateID     *xid.ID `bun:"template_id,type:varchar(20)"`        // Tracks which template this role was cloned from

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Template     *Role         `bun:"rel:belongs-to,join:template_id=id"`
	Permissions  []Permission  `bun:"m2m:role_permissions,join:Role=Permission"`
}

// BeforeAppendModel is called before inserting or updating a role
// This ensures critical validation happens at the database layer
func (r *Role) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	// Only validate on INSERT operations
	switch query.(type) {
	case *bun.InsertQuery:
		// Validate environment_id is set and not nil
		if r.EnvironmentID == nil || r.EnvironmentID.IsNil() {
			return fmt.Errorf("ROLE VALIDATION FAILED: environment_id is REQUIRED but was nil for role '%s' (app_id: %v, org_id: %v). "+
				"This role cannot be created without a valid environment. "+
				"Please ensure your code provides an environment_id when registering or creating roles. "+
				"See MIGRATION_GUIDE_FOR_EXTERNAL_PROJECTS.md for help",
				r.Name,
				r.AppID,
				r.OrganizationID)
		}

		// Validate app_id is set and not nil
		if r.AppID == nil || r.AppID.IsNil() {
			return fmt.Errorf("ROLE VALIDATION FAILED: app_id is REQUIRED but was nil for role '%s'. "+
				"This role cannot be created without a valid app_id",
				r.Name)
		}

		// Ensure display_name is set
		if r.DisplayName == "" {
			return fmt.Errorf("ROLE VALIDATION FAILED: display_name is REQUIRED but was empty for role '%s'. "+
				"Please provide a display_name or let the system auto-generate one",
				r.Name)
		}
	}

	return nil
}
