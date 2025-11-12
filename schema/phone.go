package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// PhoneVerification stores SMS verification codes
// Updated for V2 architecture: App → Environment → Organization
type PhoneVerification struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:phone_verifications,alias:pver"`

	ID                 xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	Phone              string    `json:"phone" bun:"phone,notnull"`
	Code               string    `json:"code" bun:"code,notnull"`
	AppID              xid.ID    `json:"appId" bun:"app_id,notnull,type:varchar(20)"`                              // Platform app (required)
	UserOrganizationID *xid.ID   `json:"userOrganizationId,omitempty" bun:"user_organization_id,type:varchar(20)"` // User-created org (optional)
	ExpiresAt          time.Time `json:"expiresAt" bun:"expires_at,notnull"`
	Attempts           int       `json:"attempts" bun:"attempts,notnull,default:0"`
}
