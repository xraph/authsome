package schema

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

type IDModel struct {
	ID xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`
}

type AuditableModel struct {
	ID        xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	CreatedAt time.Time  `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`
	CreatedBy xid.ID     `json:"createdBy" bun:"created_by,nullzero"`
	UpdatedAt time.Time  `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	UpdatedBy xid.ID     `json:"updatedBy" bun:"updated_by,notnull"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" bun:"deleted_at,nullzero"`
	Version   int        `json:"version" bun:"version,default:1"`
}

func (u *AuditableModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now()
		u.UpdatedAt = time.Now()
		if u.ID.IsNil() {
			u.ID = xid.New()
		}
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now()
	case *bun.DeleteQuery:
		now := time.Now()
		u.DeletedAt = &now
	}
	return nil
}
