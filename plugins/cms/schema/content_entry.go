package schema

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainSchema "github.com/xraph/authsome/schema"
)

// EntryData represents the dynamic data stored in a content entry
type EntryData map[string]any

// Value implements the driver.Valuer interface for database storage
func (d EntryData) Value() (driver.Value, error) {
	if d == nil {
		return "{}", nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for database retrieval
func (d *EntryData) Scan(value interface{}) error {
	if value == nil {
		*d = make(EntryData)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		*d = make(EntryData)
		return nil
	}
	return json.Unmarshal(bytes, d)
}

// Get returns a value from the entry data
func (d EntryData) Get(key string) (any, bool) {
	if d == nil {
		return nil, false
	}
	v, ok := d[key]
	return v, ok
}

// GetString returns a string value from the entry data
func (d EntryData) GetString(key string) string {
	v, ok := d.Get(key)
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetInt returns an integer value from the entry data
func (d EntryData) GetInt(key string) int {
	v, ok := d.Get(key)
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// GetFloat returns a float value from the entry data
func (d EntryData) GetFloat(key string) float64 {
	v, ok := d.Get(key)
	if !ok {
		return 0
	}
	if n, ok := v.(float64); ok {
		return n
	}
	return 0
}

// GetBool returns a boolean value from the entry data
func (d EntryData) GetBool(key string) bool {
	v, ok := d.Get(key)
	if !ok {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// ContentEntry represents a content entry in the database
type ContentEntry struct {
	bun.BaseModel `bun:"table:cms_content_entries,alias:ce"`

	ID            xid.ID     `bun:"id,pk,type:varchar(20)" json:"id"`
	ContentTypeID xid.ID     `bun:"content_type_id,notnull,type:varchar(20)" json:"contentTypeId"`
	AppID         xid.ID     `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	EnvironmentID xid.ID     `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"`
	Data          EntryData  `bun:"data,type:jsonb,notnull" json:"data"`
	Status        string     `bun:"status,notnull,default:'draft'" json:"status"`
	Version       int        `bun:"version,notnull,default:1" json:"version"`
	Locale        string     `bun:"locale,nullzero" json:"locale,omitempty"`
	PublishedAt   *time.Time `bun:"published_at,nullzero" json:"publishedAt"`
	ScheduledAt   *time.Time `bun:"scheduled_at,nullzero" json:"scheduledAt"`
	CreatedBy     xid.ID     `bun:"created_by,type:varchar(20)" json:"createdBy"`
	UpdatedBy     xid.ID     `bun:"updated_by,type:varchar(20)" json:"updatedBy"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"-"`

	// Relations
	App         *mainSchema.App         `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
	Environment *mainSchema.Environment `bun:"rel:belongs-to,join:environment_id=id" json:"environment,omitempty"`
	ContentType *ContentType            `bun:"rel:belongs-to,join:content_type_id=id" json:"contentType,omitempty"`
	Revisions   []*ContentRevision      `bun:"rel:has-many,join:id=entry_id" json:"revisions,omitempty"`

	// Populated relations (not stored in DB, used for query population)
	PopulatedRelations map[string][]*ContentEntry `bun:"-" json:"populatedRelations,omitempty"`
}

// TableName returns the table name for ContentEntry
func (ce *ContentEntry) TableName() string {
	return "cms_content_entries"
}

// BeforeInsert sets default values before insert
func (ce *ContentEntry) BeforeInsert() {
	if ce.ID.IsNil() {
		ce.ID = xid.New()
	}
	if ce.Status == "" {
		ce.Status = "draft"
	}
	if ce.Version == 0 {
		ce.Version = 1
	}
	if ce.Data == nil {
		ce.Data = make(EntryData)
	}
	now := time.Now()
	ce.CreatedAt = now
	ce.UpdatedAt = now
}

// BeforeUpdate updates the UpdatedAt timestamp and increments version
func (ce *ContentEntry) BeforeUpdate() {
	ce.UpdatedAt = time.Now()
	ce.Version++
}

// IsDraft returns true if the entry is a draft
func (ce *ContentEntry) IsDraft() bool {
	return ce.Status == "draft"
}

// IsPublished returns true if the entry is published
func (ce *ContentEntry) IsPublished() bool {
	return ce.Status == "published"
}

// IsArchived returns true if the entry is archived
func (ce *ContentEntry) IsArchived() bool {
	return ce.Status == "archived"
}

// IsScheduled returns true if the entry is scheduled for publishing
func (ce *ContentEntry) IsScheduled() bool {
	return ce.Status == "scheduled"
}

// CanPublish returns true if the entry can be published
func (ce *ContentEntry) CanPublish() bool {
	return ce.Status == "draft" || ce.Status == "scheduled"
}

// CanUnpublish returns true if the entry can be unpublished
func (ce *ContentEntry) CanUnpublish() bool {
	return ce.Status == "published"
}

// CanArchive returns true if the entry can be archived
func (ce *ContentEntry) CanArchive() bool {
	return ce.Status == "published" || ce.Status == "draft"
}

// Publish publishes the entry
func (ce *ContentEntry) Publish() {
	ce.Status = "published"
	now := time.Now()
	ce.PublishedAt = &now
	ce.ScheduledAt = nil
}

// Unpublish unpublishes the entry (returns to draft)
func (ce *ContentEntry) Unpublish() {
	ce.Status = "draft"
	ce.PublishedAt = nil
}

// Archive archives the entry
func (ce *ContentEntry) Archive() {
	ce.Status = "archived"
}

// Schedule schedules the entry for publishing
func (ce *ContentEntry) Schedule(at time.Time) {
	ce.Status = "scheduled"
	ce.ScheduledAt = &at
}

// GetTitle returns the title from entry data based on content type settings
func (ce *ContentEntry) GetTitle(titleField string) string {
	if titleField == "" {
		return ""
	}
	return ce.Data.GetString(titleField)
}

// GetDescription returns the description from entry data based on content type settings
func (ce *ContentEntry) GetDescription(descField string) string {
	if descField == "" {
		return ""
	}
	return ce.Data.GetString(descField)
}

// GetFieldValue returns a specific field value
func (ce *ContentEntry) GetFieldValue(field string) any {
	if ce.Data == nil {
		return nil
	}
	return ce.Data[field]
}

// SetFieldValue sets a specific field value
func (ce *ContentEntry) SetFieldValue(field string, value any) {
	if ce.Data == nil {
		ce.Data = make(EntryData)
	}
	ce.Data[field] = value
}

// DeleteFieldValue removes a field value
func (ce *ContentEntry) DeleteFieldValue(field string) {
	if ce.Data != nil {
		delete(ce.Data, field)
	}
}

// Clone creates a copy of the entry with a new ID
func (ce *ContentEntry) Clone() *ContentEntry {
	clone := &ContentEntry{
		ContentTypeID: ce.ContentTypeID,
		AppID:         ce.AppID,
		EnvironmentID: ce.EnvironmentID,
		Data:          make(EntryData),
		Status:        "draft",
		Version:       1,
		Locale:        ce.Locale,
		CreatedBy:     ce.CreatedBy,
		UpdatedBy:     ce.UpdatedBy,
	}
	// Deep copy data
	for k, v := range ce.Data {
		clone.Data[k] = v
	}
	return clone
}

// ToMap converts the entry to a map for API responses
func (ce *ContentEntry) ToMap() map[string]any {
	m := map[string]any{
		"id":            ce.ID.String(),
		"contentTypeId": ce.ContentTypeID.String(),
		"appId":         ce.AppID.String(),
		"environmentId": ce.EnvironmentID.String(),
		"data":          ce.Data,
		"status":        ce.Status,
		"version":       ce.Version,
		"createdAt":     ce.CreatedAt,
		"updatedAt":     ce.UpdatedAt,
	}
	if ce.PublishedAt != nil {
		m["publishedAt"] = ce.PublishedAt
	}
	if ce.ScheduledAt != nil {
		m["scheduledAt"] = ce.ScheduledAt
	}
	if ce.Locale != "" {
		m["locale"] = ce.Locale
	}
	if !ce.CreatedBy.IsNil() {
		m["createdBy"] = ce.CreatedBy.String()
	}
	if !ce.UpdatedBy.IsNil() {
		m["updatedBy"] = ce.UpdatedBy.String()
	}
	return m
}

