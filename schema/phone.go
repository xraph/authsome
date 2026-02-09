package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// PhoneVerification stores SMS verification codes
// Updated for V2 architecture: App → Environment → Organization.
type PhoneVerification struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:phone_verifications,alias:pver"`

	ID                 xid.ID    `bun:"id,pk,type:varchar(20)"                json:"id"`
	Phone              string    `bun:"phone,notnull"                         json:"phone"`
	Code               string    `bun:"code,notnull"                          json:"code"`
	AppID              xid.ID    `bun:"app_id,notnull,type:varchar(20)"       json:"appId"`                        // Platform app (required)
	UserOrganizationID *xid.ID   `bun:"user_organization_id,type:varchar(20)" json:"userOrganizationId,omitempty"` // User-created org (optional)
	ExpiresAt          time.Time `bun:"expires_at,notnull"                    json:"expiresAt"`
	Attempts           int       `bun:"attempts,notnull,default:0"            json:"attempts"`
}
