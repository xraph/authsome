package schema

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// JWTKey represents a JWT signing key
type JWTKey struct {
	bun.BaseModel `bun:"table:jwt_keys"`

	// Primary key
	ID xid.ID `bun:"id,pk" json:"id"`

	// App context
	AppID         xid.ID `bun:"app_id,notnull" json:"app_id"`
	IsPlatformKey bool   `bun:"is_platform_key,notnull,default:false" json:"is_platform_key"`

	// Key identification
	KeyID     string `bun:"key_id,notnull" json:"key_id"`       // Kid for JWKS (unique per app)
	Algorithm string `bun:"algorithm,notnull" json:"algorithm"` // EdDSA, RS256, etc.
	KeyType   string `bun:"key_type,notnull" json:"key_type"`   // OKP, RSA
	Curve     string `bun:"curve" json:"curve,omitempty"`       // Ed25519, P-256, etc.

	// Key material (encrypted)
	PrivateKey []byte `bun:"private_key,notnull" json:"-"`         // Encrypted private key
	PublicKey  []byte `bun:"public_key,notnull" json:"public_key"` // Public key for JWKS

	// Key status
	Active    bool       `bun:"active,notnull,default:true" json:"active"`
	ExpiresAt *time.Time `bun:"expires_at" json:"expires_at,omitempty"`

	// Usage tracking
	UsageCount int64      `bun:"usage_count,notnull,default:0" json:"usage_count"`
	LastUsedAt *time.Time `bun:"last_used_at" json:"last_used_at,omitempty"`

	// Audit fields
	CreatedAt time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete" json:"deleted_at,omitempty"`

	// Metadata
	Name        string            `bun:"name" json:"name,omitempty"`
	Description string            `bun:"description" json:"description,omitempty"`
	Metadata    map[string]string `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
}

// BeforeAppendModel implements bun.BeforeAppendModelHook
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
