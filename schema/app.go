package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// App represents the app table (formerly Organization - platform-level tenant)
type App struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:apps,alias:a"`

	ID         xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	Name       string                 `json:"name" bun:"name,notnull"`
	Slug       string                 `json:"slug" bun:"slug,notnull,unique"`
	Logo       string                 `json:"logo" bun:"logo"`
	Metadata   map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	IsPlatform bool                   `json:"isPlatform" bun:"is_platform,default:false"` // Identifies the single platform app

	// Relations
	Members []Member `json:"members,omitempty" bun:"rel:has-many,join:id=app_id"`
	Teams   []Team   `json:"teams,omitempty" bun:"rel:has-many,join:id=app_id"`
}
