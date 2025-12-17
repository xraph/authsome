// Package schema defines the database schema for the CMS plugin.
package schema

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainSchema "github.com/xraph/authsome/schema"
)

// ContentTypeSettings holds the configuration for a content type
type ContentTypeSettings struct {
	// Display settings
	TitleField       string `json:"titleField,omitempty"`
	DescriptionField string `json:"descriptionField,omitempty"`
	PreviewField     string `json:"previewField,omitempty"`

	// Features
	EnableRevisions  bool `json:"enableRevisions"`
	EnableDrafts     bool `json:"enableDrafts"`
	EnableSoftDelete bool `json:"enableSoftDelete"`
	EnableSearch     bool `json:"enableSearch"`
	EnableScheduling bool `json:"enableScheduling"`

	// Permissions
	DefaultPermissions []string `json:"defaultPermissions,omitempty"`

	// Limits
	MaxEntries int `json:"maxEntries,omitempty"`
}

// Value implements the driver.Valuer interface for database storage
func (s ContentTypeSettings) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for database retrieval
func (s *ContentTypeSettings) Scan(value interface{}) error {
	if value == nil {
		*s = ContentTypeSettings{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// ContentType represents a content type definition in the database
type ContentType struct {
	bun.BaseModel `bun:"table:cms_content_types,alias:ct"`

	ID            xid.ID              `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID         xid.ID              `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	EnvironmentID xid.ID              `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"`
	Title         string              `bun:"title,notnull" json:"title"`
	Name          string              `bun:"name,notnull" json:"name"`
	Description   string              `bun:"description,nullzero" json:"description"`
	Icon          string              `bun:"icon,nullzero" json:"icon"`
	Settings      ContentTypeSettings `bun:"settings,type:jsonb,notnull" json:"settings"`
	CreatedBy     xid.ID              `bun:"created_by,type:varchar(20)" json:"createdBy"`
	UpdatedBy     xid.ID              `bun:"updated_by,type:varchar(20)" json:"updatedBy"`
	CreatedAt     time.Time           `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time           `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	DeletedAt     *time.Time          `bun:"deleted_at,soft_delete,nullzero" json:"-"`

	// Relations
	App         *mainSchema.App         `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
	Environment *mainSchema.Environment `bun:"rel:belongs-to,join:environment_id=id" json:"environment,omitempty"`
	Fields      []*ContentField         `bun:"rel:has-many,join:id=content_type_id" json:"fields,omitempty"`
	Entries     []*ContentEntry         `bun:"rel:has-many,join:id=content_type_id" json:"entries,omitempty"`
}

// TableName returns the table name for ContentType
func (ct *ContentType) TableName() string {
	return "cms_content_types"
}

// DefaultSettings returns default content type settings
func DefaultSettings() ContentTypeSettings {
	return ContentTypeSettings{
		EnableRevisions:  true,
		EnableDrafts:     true,
		EnableSoftDelete: true,
		EnableSearch:     false,
		EnableScheduling: false,
		MaxEntries:       0, // Unlimited
	}
}

// BeforeInsert sets default values before insert
func (ct *ContentType) BeforeInsert() {
	if ct.ID.IsNil() {
		ct.ID = xid.New()
	}
	now := time.Now()
	ct.CreatedAt = now
	ct.UpdatedAt = now
}

// BeforeUpdate updates the UpdatedAt timestamp
func (ct *ContentType) BeforeUpdate() {
	ct.UpdatedAt = time.Now()
}

// GetTitleField returns the field name to use as title
func (ct *ContentType) GetTitleField() string {
	if ct.Settings.TitleField != "" {
		return ct.Settings.TitleField
	}
	// Default to common field names
	for _, name := range []string{"title", "name", "label"} {
		for _, f := range ct.Fields {
			if f.Name == name {
				return name
			}
		}
	}
	return ""
}

// GetDescriptionField returns the field name to use as description
func (ct *ContentType) GetDescriptionField() string {
	if ct.Settings.DescriptionField != "" {
		return ct.Settings.DescriptionField
	}
	// Default to common field names
	for _, name := range []string{"description", "summary", "excerpt", "content"} {
		for _, f := range ct.Fields {
			if f.Name == name {
				return name
			}
		}
	}
	return ""
}

// GetPreviewField returns the field name to use as preview
func (ct *ContentType) GetPreviewField() string {
	return ct.Settings.PreviewField
}

// HasRevisions returns true if revisions are enabled
func (ct *ContentType) HasRevisions() bool {
	return ct.Settings.EnableRevisions
}

// HasDrafts returns true if drafts are enabled
func (ct *ContentType) HasDrafts() bool {
	return ct.Settings.EnableDrafts
}

// HasScheduling returns true if scheduling is enabled
func (ct *ContentType) HasScheduling() bool {
	return ct.Settings.EnableScheduling
}

// GetFieldByName returns a field by its name
func (ct *ContentType) GetFieldByName(name string) *ContentField {
	for _, f := range ct.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// GetRequiredFields returns all required fields
func (ct *ContentType) GetRequiredFields() []*ContentField {
	var required []*ContentField
	for _, f := range ct.Fields {
		if f.Required {
			required = append(required, f)
		}
	}
	return required
}

// GetUniqueFields returns all unique fields
func (ct *ContentType) GetUniqueFields() []*ContentField {
	var unique []*ContentField
	for _, f := range ct.Fields {
		if f.Unique {
			unique = append(unique, f)
		}
	}
	return unique
}

// GetIndexedFields returns all indexed fields
func (ct *ContentType) GetIndexedFields() []*ContentField {
	var indexed []*ContentField
	for _, f := range ct.Fields {
		if f.Indexed {
			indexed = append(indexed, f)
		}
	}
	return indexed
}

// GetSearchableFields returns all searchable fields
func (ct *ContentType) GetSearchableFields() []*ContentField {
	var searchable []*ContentField
	for _, f := range ct.Fields {
		if f.IsSearchable() {
			searchable = append(searchable, f)
		}
	}
	return searchable
}

// GetRelationFields returns all relation fields
func (ct *ContentType) GetRelationFields() []*ContentField {
	var relations []*ContentField
	for _, f := range ct.Fields {
		if f.Type == "relation" {
			relations = append(relations, f)
		}
	}
	return relations
}
