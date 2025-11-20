package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Passkey stores WebAuthn/FIDO2 credentials
// Updated for V2 architecture: App → Environment → Organization
// Now includes full WebAuthn fields for production-ready implementation
type Passkey struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:passkeys,alias:pk"`

	ID           xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID       xid.ID `json:"userId" bun:"user_id,notnull,type:varchar(20)"`
	CredentialID string `json:"credentialId" bun:"credential_id,notnull,unique"`

	// WebAuthn cryptographic fields
	PublicKey []byte `json:"-" bun:"public_key,notnull"`           // COSE encoded public key
	AAGUID    []byte `json:"aaguid,omitempty" bun:"aaguid"`        // Authenticator AAGUID
	SignCount uint32 `json:"signCount" bun:"sign_count,default:0"` // Counter for replay attack detection

	// Authenticator metadata
	AuthenticatorType string `json:"authenticatorType,omitempty" bun:"authenticator_type"` // "platform" or "cross-platform"

	// User-friendly naming for device management
	Name string `json:"name,omitempty" bun:"name"`

	// Resident key / discoverable credential support
	IsResidentKey bool `json:"isResidentKey" bun:"is_resident_key,default:false"`

	// Usage tracking
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty" bun:"last_used_at"`

	// Multi-tenant scoping (App → Environment → Organization)
	AppID              xid.ID  `json:"appId" bun:"app_id,notnull,type:varchar(20)"`                              // Platform app (required)
	UserOrganizationID *xid.ID `json:"userOrganizationId,omitempty" bun:"user_organization_id,type:varchar(20)"` // User-created org (optional)
}
