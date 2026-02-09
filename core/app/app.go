package app

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// App represents an app entity DTO (Data Transfer Object)
// This is separate from schema.App to maintain proper separation of concerns.
type App struct {
	ID         xid.ID         `json:"id"`
	Name       string         `json:"name"`
	Slug       string         `json:"slug"`
	Logo       string         `json:"logo,omitempty"`
	IsPlatform bool           `json:"isPlatform"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the App DTO to a schema.App model.
func (a *App) ToSchema() *schema.App {
	return &schema.App{
		ID:         a.ID,
		Name:       a.Name,
		Slug:       a.Slug,
		Logo:       a.Logo,
		IsPlatform: a.IsPlatform,
		Metadata:   a.Metadata,
		AuditableModel: schema.AuditableModel{
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
			DeletedAt: a.DeletedAt,
		},
	}
}

// FromSchemaApp converts a schema.App model to App DTO.
func FromSchemaApp(sa *schema.App) *App {
	if sa == nil {
		return nil
	}

	return &App{
		ID:         sa.ID,
		Name:       sa.Name,
		Slug:       sa.Slug,
		Logo:       sa.Logo,
		IsPlatform: sa.IsPlatform,
		Metadata:   sa.Metadata,
		CreatedAt:  sa.CreatedAt,
		UpdatedAt:  sa.UpdatedAt,
		DeletedAt:  sa.DeletedAt,
	}
}

// FromSchemaApps converts a slice of schema.App to App DTOs.
func FromSchemaApps(apps []*schema.App) []*App {
	result := make([]*App, len(apps))
	for i, a := range apps {
		result[i] = FromSchemaApp(a)
	}

	return result
}

// CreateAppRequest represents a create app request.
type CreateAppRequest struct {
	Name     string         `json:"name"`
	Slug     string         `json:"slug"`
	Logo     string         `json:"logo"`
	Metadata map[string]any `json:"metadata"`
}

// UpdateAppRequest represents an update app request.
type UpdateAppRequest struct {
	Name     *string        `json:"name"`
	Logo     *string        `json:"logo"`
	Metadata map[string]any `json:"metadata"`
}
