package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Passkey stores WebAuthn/FIDO2 credentials
// Updated for V2 architecture: App → Environment → Organization
// Now includes full WebAuthn fields for production-ready implementation.
type Passkey struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:passkeys,alias:pk"`

	ID           xid.ID `bun:"id,pk,type:varchar(20)"           json:"id"`
	UserID       xid.ID `bun:"user_id,notnull,type:varchar(20)" json:"userId"`
	CredentialID string `bun:"credential_id,notnull,unique"     json:"credentialId"`

	// WebAuthn cryptographic fields
	PublicKey []byte `bun:"public_key,notnull"   json:"-"`                // COSE encoded public key
	AAGUID    []byte `bun:"aaguid"               json:"aaguid,omitempty"` // Authenticator AAGUID
	SignCount uint32 `bun:"sign_count,default:0" json:"signCount"`        // Counter for replay attack detection

	// Authenticator metadata
	AuthenticatorType string `bun:"authenticator_type" json:"authenticatorType,omitempty"` // "platform" or "cross-platform"

	// User-friendly naming for device management
	Name string `bun:"name" json:"name,omitempty"`

	// Resident key / discoverable credential support
	IsResidentKey bool `bun:"is_resident_key,default:false" json:"isResidentKey"`

	// Usage tracking
	LastUsedAt *time.Time `bun:"last_used_at" json:"lastUsedAt,omitempty"`

	// Multi-tenant scoping (App → Environment → Organization)
	AppID              xid.ID  `bun:"app_id,notnull,type:varchar(20)"       json:"appId"`                        // Platform app (required)
	UserOrganizationID *xid.ID `bun:"user_organization_id,type:varchar(20)" json:"userOrganizationId,omitempty"` // User-created org (optional)
}
