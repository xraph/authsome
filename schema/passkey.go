package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Passkey stores WebAuthn/FIDO2 credentials
// Updated for V2 architecture: App → Environment → Organization
type Passkey struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:passkeys,alias:pk"`

	ID                 xid.ID  `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID             xid.ID  `json:"userID" bun:"user_id,notnull,type:varchar(20)"`
	CredentialID       string  `json:"credentialID" bun:"credential_id,notnull,unique"`
	AppID              xid.ID  `json:"appId" bun:"app_id,notnull,type:varchar(20)"`                              // Platform app (required)
	UserOrganizationID *xid.ID `json:"userOrganizationId,omitempty" bun:"user_organization_id,type:varchar(20)"` // User-created org (optional)
}
