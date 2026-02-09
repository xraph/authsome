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
	ID        xid.ID     `bun:"id,pk,type:varchar(20)"                                json:"id"`
	CreatedAt time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	CreatedBy xid.ID     `bun:"created_by,nullzero"                                   json:"createdBy"`
	UpdatedAt time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	UpdatedBy xid.ID     `bun:"updated_by,nullzero"                                   json:"updatedBy"`
	DeletedAt *time.Time `bun:"deleted_at,nullzero"                                   json:"deletedAt,omitempty"`
	Version   int        `bun:"version,default:1"                                     json:"version"`
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
