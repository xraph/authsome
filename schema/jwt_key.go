package schema

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// JWTKey represents a JWT signing key.
type JWTKey struct {
	AuditableModel
	bun.BaseModel `bun:"table:jwt_keys"`

	// App context
	AppID         xid.ID `bun:"app_id,notnull,type:varchar(20)"       json:"appID"`
	IsPlatformKey bool   `bun:"is_platform_key,notnull,default:false" json:"isPlatformKey"`

	// Key identification
	KeyID     string `bun:"key_id,notnull"    json:"keyID"`           // Kid for JWKS (unique per app)
	Algorithm string `bun:"algorithm,notnull" json:"algorithm"`       // EdDSA, RS256, etc.
	KeyType   string `bun:"key_type,notnull"  json:"keyType"`         // OKP, RSA
	Curve     string `bun:"curve"             json:"curve,omitempty"` // Ed25519, P-256, etc.

	// Key material (encrypted)
	PrivateKey []byte `bun:"private_key,notnull" json:"-"`         // Encrypted private key
	PublicKey  []byte `bun:"public_key,notnull"  json:"publicKey"` // Public key for JWKS

	// Key status
	Active    bool       `bun:"active,notnull,default:true" json:"active"`
	ExpiresAt *time.Time `bun:"expires_at"                  json:"expiresAt,omitempty"`

	// Usage tracking
	UsageCount int64      `bun:"usage_count,notnull,default:0" json:"usageCount"`
	LastUsedAt *time.Time `bun:"last_used_at"                  json:"lastUsedAt,omitempty"`

	// Metadata
	Name        string            `bun:"name"                json:"name,omitempty"`
	Description string            `bun:"description"         json:"description,omitempty"`
	Metadata    map[string]string `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (k *JWTKey) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if k.ID.IsNil() {
			k.ID = xid.New()
		}

		k.CreatedAt = time.Now()
		k.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		k.UpdatedAt = time.Now()
	}

	return nil
}
