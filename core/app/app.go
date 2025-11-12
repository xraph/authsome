package app

import (
	"time"

	"github.com/rs/xid"
)

// App represents an app entity (platform-level tenant)
type App struct {
	ID        xid.ID                 `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	Logo      string                 `json:"logo"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// CreateAppRequest represents a create app request
type CreateAppRequest struct {
	Name     string                 `json:"name"`
	Slug     string                 `json:"slug"`
	Logo     string                 `json:"logo"`
	Metadata map[string]interface{} `json:"metadata"`
}

// UpdateAppRequest represents an update app request
type UpdateAppRequest struct {
	Name     *string                `json:"name"`
	Logo     *string                `json:"logo"`
	Metadata map[string]interface{} `json:"metadata"`
}
