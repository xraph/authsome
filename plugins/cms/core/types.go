// Package core provides core types and utilities for the CMS plugin.
package core

import (
	"time"

	"github.com/rs/xid"
)

// =============================================================================
// ENUMS
// =============================================================================

// EntryStatus defines the status of a content entry
type EntryStatus string

const (
	// EntryStatusDraft indicates the entry is a draft
	EntryStatusDraft EntryStatus = "draft"
	// EntryStatusPublished indicates the entry is published
	EntryStatusPublished EntryStatus = "published"
	// EntryStatusArchived indicates the entry is archived
	EntryStatusArchived EntryStatus = "archived"
	// EntryStatusScheduled indicates the entry is scheduled for publishing
	EntryStatusScheduled EntryStatus = "scheduled"
)

// String returns the string representation of the status
func (s EntryStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s EntryStatus) IsValid() bool {
	switch s {
	case EntryStatusDraft, EntryStatusPublished, EntryStatusArchived, EntryStatusScheduled:
		return true
	default:
		return false
	}
}

// ParseEntryStatus parses a string into an EntryStatus
func ParseEntryStatus(s string) (EntryStatus, bool) {
	status := EntryStatus(s)
	if status.IsValid() {
		return status, true
	}
	return EntryStatusDraft, false
}

// RelationType defines the type of relation between content types
type RelationType string

const (
	// RelationTypeOneToOne represents a one-to-one relation
	RelationTypeOneToOne RelationType = "oneToOne"
	// RelationTypeOneToMany represents a one-to-many relation
	RelationTypeOneToMany RelationType = "oneToMany"
	// RelationTypeManyToOne represents a many-to-one relation
	RelationTypeManyToOne RelationType = "manyToOne"
	// RelationTypeManyToMany represents a many-to-many relation
	RelationTypeManyToMany RelationType = "manyToMany"
)

// String returns the string representation
func (r RelationType) String() string {
	return string(r)
}

// IsValid checks if the relation type is valid
func (r RelationType) IsValid() bool {
	switch r {
	case RelationTypeOneToOne, RelationTypeOneToMany, RelationTypeManyToOne, RelationTypeManyToMany:
		return true
	default:
		return false
	}
}

// OnDeleteAction defines what happens when a related entry is deleted
type OnDeleteAction string

const (
	// OnDeleteCascade deletes the related entries
	OnDeleteCascade OnDeleteAction = "cascade"
	// OnDeleteSetNull sets the relation to null
	OnDeleteSetNull OnDeleteAction = "setNull"
	// OnDeleteRestrict prevents deletion if related entries exist
	OnDeleteRestrict OnDeleteAction = "restrict"
	// OnDeleteNoAction takes no action
	OnDeleteNoAction OnDeleteAction = "noAction"
)

// IsValid checks if the on-delete action is valid
func (a OnDeleteAction) IsValid() bool {
	switch a {
	case OnDeleteCascade, OnDeleteSetNull, OnDeleteRestrict, OnDeleteNoAction:
		return true
	default:
		return false
	}
}

// =============================================================================
// RELATION DTOs
// =============================================================================

// RelatedEntryDTO represents a related entry
type RelatedEntryDTO struct {
	ID    string                  `json:"id"`
	Order int                     `json:"order"`
	Entry *ContentEntrySummaryDTO `json:"entry,omitempty"`
}

// TypeRelationDTO represents a type relation definition
type TypeRelationDTO struct {
	ID                     string    `json:"id"`
	SourceContentTypeID    string    `json:"sourceContentTypeId"`
	TargetContentTypeID    string    `json:"targetContentTypeId"`
	SourceContentTypeTitle string    `json:"sourceContentTypeTitle,omitempty"`
	SourceContentTypeName  string    `json:"sourceContentTypeName,omitempty"`
	TargetContentTypeTitle string    `json:"targetContentTypeTitle,omitempty"`
	TargetContentTypeName  string    `json:"targetContentTypeName,omitempty"`
	SourceFieldName        string    `json:"sourceFieldName"`
	TargetFieldName        string    `json:"targetFieldName,omitempty"`
	RelationType           string    `json:"relationType"`
	OnDelete               string    `json:"onDelete"`
	CreatedAt              time.Time `json:"createdAt"`
}

// CreateTypeRelationRequest is the request to create a type relation
type CreateTypeRelationRequest struct {
	SourceContentTypeID xid.ID `json:"sourceContentTypeId"`
	TargetContentTypeID xid.ID `json:"targetContentTypeId"`
	SourceFieldName     string `json:"sourceFieldName"`
	TargetFieldName     string `json:"targetFieldName,omitempty"`
	RelationType        string `json:"relationType"`
	OnDelete            string `json:"onDelete,omitempty"`
}

// UpdateTypeRelationRequest is the request to update a type relation
type UpdateTypeRelationRequest struct {
	TargetFieldName *string `json:"targetFieldName,omitempty"`
	OnDelete        *string `json:"onDelete,omitempty"`
}

// SetRelationRequest is the request to set a relation
type SetRelationRequest struct {
	TargetID xid.ID `json:"targetId"`
}

// SetRelationsRequest is the request to set multiple relations
type SetRelationsRequest struct {
	TargetIDs []xid.ID `json:"targetIds"`
}

// ReorderRelationsRequest is the request to reorder relations
type ReorderRelationsRequest struct {
	OrderedTargetIDs []xid.ID `json:"orderedTargetIds"`
}

// =============================================================================
// CONTENT TYPE DTOs
// =============================================================================

// ContentTypeDTO is the API response for a content type
type ContentTypeDTO struct {
	ID            string                  `json:"id"`
	AppID         string                  `json:"appId"`
	EnvironmentID string                  `json:"environmentId"`
	Title         string                  `json:"title"`
	Name          string                  `json:"name"`
	Description   string                  `json:"description,omitempty"`
	Icon          string                  `json:"icon,omitempty"`
	Settings      ContentTypeSettingsDTO  `json:"settings"`
	Fields        []*ContentFieldDTO      `json:"fields,omitempty"`
	EntryCount    int                     `json:"entryCount,omitempty"`
	CreatedBy     string                  `json:"createdBy,omitempty"`
	UpdatedBy     string                  `json:"updatedBy,omitempty"`
	CreatedAt     time.Time               `json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
}

// ContentTypeSettingsDTO represents content type settings
type ContentTypeSettingsDTO struct {
	// Display settings
	TitleField       string `json:"titleField,omitempty"`
	DescriptionField string `json:"descriptionField,omitempty"`
	
	// Features
	EnableRevisions   bool `json:"enableRevisions"`
	EnableDrafts      bool `json:"enableDrafts"`
	EnableSoftDelete  bool `json:"enableSoftDelete"`
	EnableSearch      bool `json:"enableSearch"`
	EnableScheduling  bool `json:"enableScheduling"`
	
	// Permissions
	DefaultPermissions []string `json:"defaultPermissions,omitempty"`
	
	// Limits
	MaxEntries int `json:"maxEntries,omitempty"`
}

// ContentTypeSummaryDTO is a lightweight content type for lists
type ContentTypeSummaryDTO struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	EntryCount  int       `json:"entryCount"`
	FieldCount  int       `json:"fieldCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// =============================================================================
// COMPONENT SCHEMA DTOs
// =============================================================================

// ComponentSchemaDTO is the API response for a component schema
type ComponentSchemaDTO struct {
	ID            string               `json:"id"`
	AppID         string               `json:"appId"`
	EnvironmentID string               `json:"environmentId"`
	Title         string               `json:"title"`
	Name          string               `json:"name"`
	Description   string               `json:"description,omitempty"`
	Icon          string               `json:"icon,omitempty"`
	Fields        []NestedFieldDefDTO  `json:"fields"`
	UsageCount    int                  `json:"usageCount,omitempty"`
	CreatedBy     string               `json:"createdBy,omitempty"`
	UpdatedBy     string               `json:"updatedBy,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
	UpdatedAt     time.Time            `json:"updatedAt"`
}

// ComponentSchemaSummaryDTO is a lightweight component schema for lists
type ComponentSchemaSummaryDTO struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	FieldCount  int       `json:"fieldCount"`
	UsageCount  int       `json:"usageCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// NestedFieldDefDTO represents a field definition within a nested object or component schema
type NestedFieldDefDTO struct {
	Title       string            `json:"title"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Required    bool              `json:"required,omitempty"`
	Description string            `json:"description,omitempty"`
	Options     *FieldOptionsDTO  `json:"options,omitempty"`
}

// CreateComponentSchemaRequest is the request to create a component schema
type CreateComponentSchemaRequest struct {
	Title       string              `json:"title" validate:"required,min=1,max=100"`
	Name        string              `json:"name" validate:"required"`
	Description string              `json:"description,omitempty"`
	Icon        string              `json:"icon,omitempty"`
	Fields      []NestedFieldDefDTO `json:"fields,omitempty"`
}

// UpdateComponentSchemaRequest is the request to update a component schema
type UpdateComponentSchemaRequest struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Icon        string              `json:"icon,omitempty"`
	Fields      []NestedFieldDefDTO `json:"fields,omitempty"`
}

// ListComponentSchemasResponse is the response for listing component schemas
type ListComponentSchemasResponse struct {
	Components []*ComponentSchemaSummaryDTO `json:"components"`
	Page       int                          `json:"page"`
	PageSize   int                          `json:"pageSize"`
	TotalItems int                          `json:"totalItems"`
	TotalPages int                          `json:"totalPages"`
}

// =============================================================================
// CONTENT FIELD DTOs
// =============================================================================

// ContentFieldDTO is the API response for a content field
type ContentFieldDTO struct {
	ID            string          `json:"id"`
	ContentTypeID string          `json:"contentTypeId"`
	Title         string          `json:"title"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	Type          string          `json:"type"`
	Required      bool            `json:"required"`
	Unique        bool            `json:"unique"`
	Indexed       bool            `json:"indexed"`
	Localized     bool            `json:"localized"`
	DefaultValue  any             `json:"defaultValue,omitempty"`
	Options       FieldOptionsDTO `json:"options,omitempty"`
	Order         int             `json:"order"`
	Hidden        bool            `json:"hidden"`
	ReadOnly      bool            `json:"readOnly"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

// FieldOptionsDTO contains type-specific field options
type FieldOptionsDTO struct {
	// Text fields
	MinLength int    `json:"minLength,omitempty"`
	MaxLength int    `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	
	// Number fields
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Step    *float64 `json:"step,omitempty"`
	Integer bool     `json:"integer,omitempty"`
	
	// Select fields
	Choices []ChoiceDTO `json:"choices,omitempty"`
	
	// Relation fields
	RelatedType    string         `json:"relatedType,omitempty"`
	RelationType   string         `json:"relationType,omitempty"`
	OnDelete       string         `json:"onDelete,omitempty"`
	InverseField   string         `json:"inverseField,omitempty"`
	
	// Rich text fields
	AllowHTML bool     `json:"allowHtml,omitempty"`
	MaxWords  int      `json:"maxWords,omitempty"`
	
	// Media fields
	AllowedMimeTypes []string `json:"allowedMimeTypes,omitempty"`
	MaxFileSize      int64    `json:"maxFileSize,omitempty"`
	
	// Slug fields
	SourceField string `json:"sourceField,omitempty"`
	
	// JSON fields
	Schema string `json:"schema,omitempty"`
	
	// Date fields
	MinDate    *time.Time `json:"minDate,omitempty"`
	MaxDate    *time.Time `json:"maxDate,omitempty"`
	DateFormat string     `json:"dateFormat,omitempty"`
	
	// Object/Array fields (nested structures)
	NestedFields    []NestedFieldDefDTO `json:"nestedFields,omitempty"`    // Inline sub-field definitions
	ComponentRef    string              `json:"componentRef,omitempty"`    // Reference to ComponentSchema slug
	MinItems        *int                `json:"minItems,omitempty"`        // For array: minimum items
	MaxItems        *int                `json:"maxItems,omitempty"`        // For array: maximum items
	Collapsible     bool                `json:"collapsible,omitempty"`     // UI: collapsible in form
	DefaultExpanded bool                `json:"defaultExpanded,omitempty"` // UI: expanded by default

	// OneOf fields (discriminated union)
	DiscriminatorField         string                        `json:"discriminatorField,omitempty"`         // Field name to watch for schema selection
	Schemas                    map[string]OneOfSchemaOptionDTO `json:"schemas,omitempty"`                    // Value -> schema mapping
	ClearOnDiscriminatorChange bool                          `json:"clearOnDiscriminatorChange,omitempty"` // Clear data when discriminator changes

	// Conditional visibility
	ShowWhen        *FieldConditionDTO `json:"showWhen,omitempty"`        // Show field when condition is met
	HideWhen        *FieldConditionDTO `json:"hideWhen,omitempty"`        // Hide field when condition is met
	ClearWhenHidden bool               `json:"clearWhenHidden,omitempty"` // Clear value when hidden
}

// OneOfSchemaOptionDTO defines a schema option for oneOf fields
type OneOfSchemaOptionDTO struct {
	ComponentRef string              `json:"componentRef,omitempty"` // Reference to ComponentSchema slug
	NestedFields []NestedFieldDefDTO `json:"nestedFields,omitempty"` // Or inline field definitions
	Label        string              `json:"label,omitempty"`        // Display label for this option
}

// FieldConditionDTO defines a condition for showing/hiding fields
type FieldConditionDTO struct {
	Field    string `json:"field"`           // Field name to watch
	Operator string `json:"operator"`        // eq, ne, in, notIn, exists, notExists
	Value    any    `json:"value,omitempty"` // Value(s) to compare
}

// ChoiceDTO represents a choice option for select fields
type ChoiceDTO struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Icon     string `json:"icon,omitempty"`
	Color    string `json:"color,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// =============================================================================
// CONTENT ENTRY DTOs
// =============================================================================

// ContentEntryDTO is the API response for a content entry
type ContentEntryDTO struct {
	ID            string                 `json:"id"`
	ContentTypeID string                 `json:"contentTypeId"`
	ContentType   *ContentTypeSummaryDTO `json:"contentType,omitempty"`
	AppID         string                 `json:"appId"`
	EnvironmentID string                 `json:"environmentId"`
	Data          map[string]any         `json:"data"`
	Status        string                 `json:"status"`
	Version       int                    `json:"version"`
	PublishedAt   *time.Time             `json:"publishedAt,omitempty"`
	ScheduledAt   *time.Time             `json:"scheduledAt,omitempty"`
	CreatedBy     string                 `json:"createdBy,omitempty"`
	UpdatedBy     string                 `json:"updatedBy,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	// Relations maps field names to related entry IDs
	Relations     map[string][]string    `json:"relations,omitempty"`
}

// ContentEntrySummaryDTO is a lightweight entry for lists
type ContentEntrySummaryDTO struct {
	ID          string     `json:"id"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Version     int        `json:"version"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// =============================================================================
// CONTENT REVISION DTOs
// =============================================================================

// ContentRevisionDTO is the API response for a content revision
type ContentRevisionDTO struct {
	ID           string         `json:"id"`
	EntryID      string         `json:"entryId"`
	Version      int            `json:"version"`
	Data         map[string]any `json:"data"`
	Status       string         `json:"status"`
	ChangeReason string         `json:"changeReason,omitempty"`
	ChangedBy    string         `json:"changedBy,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

// =============================================================================
// REQUEST DTOs
// =============================================================================

// CreateContentTypeRequest is the request to create a content type
type CreateContentTypeRequest struct {
	Title       string                  `json:"title" validate:"required,min=1,max=100"`
	Name        string                  `json:"name" validate:"required"`
	Description string                  `json:"description,omitempty"`
	Icon        string                  `json:"icon,omitempty"`
	Settings    *ContentTypeSettingsDTO `json:"settings,omitempty"`
}

// UpdateContentTypeRequest is the request to update a content type
type UpdateContentTypeRequest struct {
	Title       string                  `json:"title,omitempty"`
	Description string                  `json:"description,omitempty"`
	Icon        string                  `json:"icon,omitempty"`
	Settings    *ContentTypeSettingsDTO `json:"settings,omitempty"`
}

// CreateFieldRequest is the request to create a content field
type CreateFieldRequest struct {
	Title        string          `json:"title" validate:"required,min=1,max=100"`
	Name         string          `json:"name" validate:"required"`
	Description  string          `json:"description,omitempty"`
	Type         string          `json:"type" validate:"required"`
	Required     bool            `json:"required"`
	Unique       bool            `json:"unique"`
	Indexed      bool            `json:"indexed"`
	Localized    bool            `json:"localized"`
	DefaultValue any             `json:"defaultValue,omitempty"`
	Options      *FieldOptionsDTO `json:"options,omitempty"`
	Order        int             `json:"order"`
	Hidden       bool            `json:"hidden"`
	ReadOnly     bool            `json:"readOnly"`
}

// UpdateFieldRequest is the request to update a content field
type UpdateFieldRequest struct {
	Title        string          `json:"title,omitempty"`
	Description  string          `json:"description,omitempty"`
	Required     *bool           `json:"required,omitempty"`
	Unique       *bool           `json:"unique,omitempty"`
	Indexed      *bool           `json:"indexed,omitempty"`
	Localized    *bool           `json:"localized,omitempty"`
	DefaultValue any             `json:"defaultValue,omitempty"`
	Options      *FieldOptionsDTO `json:"options,omitempty"`
	Order        *int            `json:"order,omitempty"`
	Hidden       *bool           `json:"hidden,omitempty"`
	ReadOnly     *bool           `json:"readOnly,omitempty"`
}

// ReorderFieldsRequest is the request to reorder fields
type ReorderFieldsRequest struct {
	FieldOrders []FieldOrderItem `json:"fieldOrders" validate:"required,min=1"`
}

// FieldOrderItem represents a field and its new order
type FieldOrderItem struct {
	FieldID string `json:"fieldId" validate:"required"`
	Order   int    `json:"order"`
}

// CreateEntryRequest is the request to create a content entry
type CreateEntryRequest struct {
	Data        map[string]any `json:"data" validate:"required"`
	Status      string         `json:"status,omitempty"`
	ScheduledAt *time.Time     `json:"scheduledAt,omitempty"`
}

// UpdateEntryRequest is the request to update a content entry
type UpdateEntryRequest struct {
	Data         map[string]any `json:"data,omitempty"`
	Status       string         `json:"status,omitempty"`
	ScheduledAt  *time.Time     `json:"scheduledAt,omitempty"`
	ChangeReason string         `json:"changeReason,omitempty"`
}

// PublishEntryRequest is the request to publish an entry
type PublishEntryRequest struct {
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
}

// RollbackEntryRequest is the request to rollback an entry
type RollbackEntryRequest struct {
	TargetVersion int    `json:"targetVersion" validate:"required,min=1"`
	Reason        string `json:"reason,omitempty"`
}

// =============================================================================
// QUERY DTOs
// =============================================================================

// ListContentTypesQuery defines query parameters for listing content types
type ListContentTypesQuery struct {
	Search    string `json:"search,omitempty"`
	SortBy    string `json:"sortBy,omitempty"`
	SortOrder string `json:"sortOrder,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"pageSize,omitempty"`
}

// ListComponentSchemasQuery defines query parameters for listing component schemas
type ListComponentSchemasQuery struct {
	Search    string `json:"search,omitempty"`
	SortBy    string `json:"sortBy,omitempty"`
	SortOrder string `json:"sortOrder,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"pageSize,omitempty"`
}

// ListEntriesQuery defines query parameters for listing entries
type ListEntriesQuery struct {
	Status    string         `json:"status,omitempty"`
	Search    string         `json:"search,omitempty"`
	Filters   map[string]any `json:"filters,omitempty"`
	SortBy    string         `json:"sortBy,omitempty"`
	SortOrder string         `json:"sortOrder,omitempty"`
	Page      int            `json:"page,omitempty"`
	PageSize  int            `json:"pageSize,omitempty"`
	Select    []string       `json:"select,omitempty"`
	Populate  []string       `json:"populate,omitempty"`
}

// ListRevisionsQuery defines query parameters for listing revisions
type ListRevisionsQuery struct {
	Page     int `json:"page,omitempty"`
	PageSize int `json:"pageSize,omitempty"`
}

// =============================================================================
// RESPONSE DTOs
// =============================================================================

// ListContentTypesResponse is the response for listing content types
type ListContentTypesResponse struct {
	ContentTypes []*ContentTypeSummaryDTO `json:"contentTypes"`
	Page         int                      `json:"page"`
	PageSize     int                      `json:"pageSize"`
	TotalItems   int                      `json:"totalItems"`
	TotalPages   int                      `json:"totalPages"`
}

// ListEntriesResponse is the response for listing entries
type ListEntriesResponse struct {
	Entries    []*ContentEntryDTO `json:"entries"`
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
	TotalItems int                `json:"totalItems"`
	TotalPages int                `json:"totalPages"`
}

// ListRevisionsResponse is the response for listing revisions
type ListRevisionsResponse struct {
	Revisions  []*ContentRevisionDTO `json:"revisions"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"pageSize"`
	TotalItems int                   `json:"totalItems"`
	TotalPages int                   `json:"totalPages"`
}

// =============================================================================
// STATISTICS DTOs
// =============================================================================

// CMSStatsDTO contains overall CMS statistics
type CMSStatsDTO struct {
	TotalContentTypes int                   `json:"totalContentTypes"`
	TotalEntries      int                   `json:"totalEntries"`
	TotalRevisions    int                   `json:"totalRevisions"`
	EntriesByStatus   map[string]int        `json:"entriesByStatus"`
	EntriesByType     map[string]int        `json:"entriesByType"`
	RecentlyUpdated   int                   `json:"recentlyUpdated"`
	ScheduledEntries  int                   `json:"scheduledEntries"`
}

// ContentTypeStatsDTO contains statistics for a specific content type
type ContentTypeStatsDTO struct {
	ContentTypeID   string         `json:"contentTypeId"`
	TotalEntries    int            `json:"totalEntries"`
	DraftEntries    int            `json:"draftEntries"`
	PublishedEntries int           `json:"publishedEntries"`
	ArchivedEntries int            `json:"archivedEntries"`
	EntriesByStatus map[string]int `json:"entriesByStatus"`
	LastEntryAt     *time.Time     `json:"lastEntryAt,omitempty"`
}

// =============================================================================
// REVISION DTOs (for revision service)
// =============================================================================

// RevisionDTO represents a content revision
type RevisionDTO struct {
	ID        string         `json:"id"`
	EntryID   string         `json:"entryId"`
	Version   int            `json:"version"`
	Data      map[string]any `json:"data"`
	ChangedBy string         `json:"changedBy,omitempty"`
	Reason    string         `json:"reason,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// RevisionCompareDTO contains revision comparison results
type RevisionCompareDTO struct {
	From        *RevisionDTO      `json:"from"`
	To          *RevisionDTO      `json:"to"`
	Differences []FieldDifference `json:"differences"`
}

// DiffType represents the type of difference
type DiffType string

const (
	// DiffTypeAdded indicates a field was added
	DiffTypeAdded DiffType = "added"
	// DiffTypeRemoved indicates a field was removed
	DiffTypeRemoved DiffType = "removed"
	// DiffTypeModified indicates a field was modified
	DiffTypeModified DiffType = "modified"
)

// FieldDifference represents a difference in a specific field
type FieldDifference struct {
	Field    string   `json:"field"`
	OldValue any      `json:"oldValue,omitempty"`
	NewValue any      `json:"newValue,omitempty"`
	Type     DiffType `json:"type"`
}

// PaginatedResponse is a generic paginated response wrapper
type PaginatedResponse[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalPages int `json:"totalPages"`
}

