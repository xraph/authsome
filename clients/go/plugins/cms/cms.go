package cms

import (
	"context"
	"net/url"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated cms plugin

// Plugin implements the cms plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new cms plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "cms"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// ListEntries ListEntries lists entries for a content type
GET /cms/:type
func (p *Plugin) ListEntries(ctx context.Context, typeSlug string) error {
	path := "/cms/:typeSlug"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// CreateEntry CreateEntry creates a new content entry
POST /cms/:type
func (p *Plugin) CreateEntry(ctx context.Context, typeSlug string) error {
	path := "/cms/:typeSlug"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetEntry GetEntry retrieves a content entry by ID
GET /cms/:type/:id
func (p *Plugin) GetEntry(ctx context.Context, typeSlug string, entryId xid.ID) error {
	path := "/cms/:typeSlug/:entryId"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateEntry UpdateEntry updates a content entry
PUT /cms/:type/:id
func (p *Plugin) UpdateEntry(ctx context.Context, typeSlug string, entryId xid.ID) error {
	path := "/cms/:typeSlug/:entryId"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// DeleteEntry DeleteEntry deletes a content entry
DELETE /cms/:type/:id
func (p *Plugin) DeleteEntry(ctx context.Context, typeSlug string, entryId xid.ID) error {
	path := "/cms/:typeSlug/:entryId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// PublishEntry PublishEntry publishes a content entry
POST /cms/:type/:id/publish
func (p *Plugin) PublishEntry(ctx context.Context, typeSlug string, entryId xid.ID) error {
	path := "/cms/:typeSlug/:entryId/publish"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// UnpublishEntry UnpublishEntry unpublishes a content entry
POST /cms/:type/:id/unpublish
func (p *Plugin) UnpublishEntry(ctx context.Context, typeSlug string, entryId xid.ID) error {
	path := "/cms/:typeSlug/:entryId/unpublish"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// ArchiveEntry ArchiveEntry archives a content entry
POST /cms/:type/:id/archive
func (p *Plugin) ArchiveEntry(ctx context.Context, typeSlug string, entryId xid.ID) error {
	path := "/cms/:typeSlug/:entryId/archive"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// QueryEntries QueryEntries performs an advanced query on entries
POST /cms/:type/query
func (p *Plugin) QueryEntries(ctx context.Context, typeSlug string) error {
	path := "/cms/:typeSlug/query"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// BulkPublish BulkPublish publishes multiple entries
POST /cms/:type/bulk/publish
func (p *Plugin) BulkPublish(ctx context.Context, typeSlug string) error {
	path := "/cms/:typeSlug/bulk/publish"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// BulkUnpublish BulkUnpublish unpublishes multiple entries
POST /cms/:type/bulk/unpublish
func (p *Plugin) BulkUnpublish(ctx context.Context, typeSlug string) error {
	path := "/cms/:typeSlug/bulk/unpublish"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// BulkDelete BulkDelete deletes multiple entries
POST /cms/:type/bulk/delete
func (p *Plugin) BulkDelete(ctx context.Context, typeSlug string) error {
	path := "/cms/:typeSlug/bulk/delete"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetEntryStats GetEntryStats returns statistics for entries
GET /cms/:type/stats
func (p *Plugin) GetEntryStats(ctx context.Context, typeSlug string) error {
	path := "/cms/:typeSlug/stats"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListContentTypes ListContentTypes lists all content types
GET /cms/types
func (p *Plugin) ListContentTypes(ctx context.Context) error {
	path := "/cms/types"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// CreateContentType CreateContentType creates a new content type
POST /cms/types
func (p *Plugin) CreateContentType(ctx context.Context) error {
	path := "/cms/types"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetContentType GetContentType retrieves a content type by slug
GET /cms/types/:slug
func (p *Plugin) GetContentType(ctx context.Context, slug string) error {
	path := "/cms/types/:slug"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateContentType UpdateContentType updates a content type
PUT /cms/types/:slug
func (p *Plugin) UpdateContentType(ctx context.Context, slug string) error {
	path := "/cms/types/:slug"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// DeleteContentType DeleteContentType deletes a content type
DELETE /cms/types/:slug
func (p *Plugin) DeleteContentType(ctx context.Context, slug string) error {
	path := "/cms/types/:slug"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// ListFields ListFields lists all fields for a content type
GET /cms/types/:slug/fields
func (p *Plugin) ListFields(ctx context.Context, slug string) error {
	path := "/cms/types/:slug/fields"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// AddField AddField adds a new field to a content type
POST /cms/types/:slug/fields
func (p *Plugin) AddField(ctx context.Context, slug string) error {
	path := "/cms/types/:slug/fields"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetField GetField retrieves a field by slug
GET /cms/types/:slug/fields/:fieldSlug
func (p *Plugin) GetField(ctx context.Context, slug string, fieldSlug string) error {
	path := "/cms/types/:slug/fields/:fieldSlug"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateField UpdateField updates a field
PUT /cms/types/:slug/fields/:fieldSlug
func (p *Plugin) UpdateField(ctx context.Context, slug string, fieldSlug string) error {
	path := "/cms/types/:slug/fields/:fieldSlug"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// DeleteField DeleteField deletes a field
DELETE /cms/types/:slug/fields/:fieldSlug
func (p *Plugin) DeleteField(ctx context.Context, slug string, fieldSlug string) error {
	path := "/cms/types/:slug/fields/:fieldSlug"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// ReorderFields ReorderFields reorders fields in a content type
POST /cms/types/:slug/fields/reorder
func (p *Plugin) ReorderFields(ctx context.Context, slug string) error {
	path := "/cms/types/:slug/fields/reorder"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetFieldTypes GetFieldTypes returns all available field types
GET /cms/field-types
func (p *Plugin) GetFieldTypes(ctx context.Context) error {
	path := "/cms/field-types"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListRevisions ListRevisions lists revisions for an entry
GET /cms/:type/:id/revisions
func (p *Plugin) ListRevisions(ctx context.Context, typeSlug string, entryId xid.ID) error {
	path := "/cms/:typeSlug/:entryId/revisions"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetRevision GetRevision retrieves a specific revision
GET /cms/:type/:id/revisions/:version
func (p *Plugin) GetRevision(ctx context.Context, typeSlug string, entryId xid.ID, version int) error {
	path := "/cms/:typeSlug/:entryId/revisions/:version"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// RestoreRevision RestoreRevision restores an entry to a specific revision
POST /cms/:type/:id/revisions/:version/restore
func (p *Plugin) RestoreRevision(ctx context.Context, typeSlug string, entryId xid.ID, version int) error {
	path := "/cms/:typeSlug/:entryId/revisions/:version/restore"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// CompareRevisions CompareRevisions compares two revisions
GET /cms/:type/:id/revisions/compare?from=:v1&to=:v2
func (p *Plugin) CompareRevisions(ctx context.Context, typeSlug string, entryId xid.ID) error {
	path := "/cms/:typeSlug/:entryId/revisions/compare"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

