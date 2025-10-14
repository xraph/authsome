package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

type Passkey struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:passkeys,alias:pk"`

	ID           xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID       string `json:"userID" bun:"user_id,notnull"`
	CredentialID string `json:"credentialID" bun:"credential_id,notnull,unique"`
}
