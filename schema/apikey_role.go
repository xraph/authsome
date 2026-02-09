package schema

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// APIKeyRole represents the many-to-many relationship between API keys and roles
// This enables API keys to leverage the RBAC system for structured permissions.
type APIKeyRole struct {
	bun.BaseModel `bun:"table:apikey_roles,alias:akr"`

	ID             xid.ID     `bun:"id,pk,type:varchar(20)"`
	APIKeyID       xid.ID     `bun:"api_key_id,notnull,type:varchar(20)"`
	RoleID         xid.ID     `bun:"role_id,notnull,type:varchar(20)"`
	OrganizationID *xid.ID    `bun:"organization_id,type:varchar(20)"` // Optional: org-scoped assignment
	CreatedAt      time.Time  `bun:"created_at,notnull"`
	CreatedBy      *xid.ID    `bun:"created_by,type:varchar(20)"`
	DeletedAt      *time.Time `bun:"deleted_at"`

	// Relations
	APIKey *APIKey `bun:"rel:belongs-to,join:api_key_id=id"`
	Role   *Role   `bun:"rel:belongs-to,join:role_id=id"`
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (ar *APIKeyRole) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if ar.ID.IsNil() {
			ar.ID = xid.New()
		}

		ar.CreatedAt = time.Now()
	}

	return nil
}
