package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// App represents the app table (formerly Organization - platform-level tenant).
type App struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:apps,alias:a"`

	ID         xid.ID         `bun:"id,pk,type:varchar(20)"    json:"id"`
	Name       string         `bun:"name,notnull"              json:"name"`
	Slug       string         `bun:"slug,notnull,unique"       json:"slug"`
	Logo       string         `bun:"logo"                      json:"logo"`
	Metadata   map[string]any `bun:"metadata,type:jsonb"       json:"metadata"`
	IsPlatform bool           `bun:"is_platform,default:false" json:"isPlatform"` // Identifies the single platform app

	// Relations
	Members []Member `bun:"rel:has-many,join:id=app_id" json:"members,omitempty"`
	Teams   []Team   `bun:"rel:has-many,join:id=app_id" json:"teams,omitempty"`
}
