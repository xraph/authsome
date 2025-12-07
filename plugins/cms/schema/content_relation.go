package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// ContentRelation represents a many-to-many relation between content entries
// Used for storing bidirectional relations and many-to-many relationships
type ContentRelation struct {
	bun.BaseModel `bun:"table:cms_content_relations,alias:crel"`

	ID              xid.ID    `bun:"id,pk,type:varchar(20)" json:"id"`
	SourceEntryID   xid.ID    `bun:"source_entry_id,notnull,type:varchar(20)" json:"sourceEntryId"`
	TargetEntryID   xid.ID    `bun:"target_entry_id,notnull,type:varchar(20)" json:"targetEntryId"`
	FieldName       string    `bun:"field_name,notnull" json:"fieldName"`
	Order           int       `bun:"\"order\",notnull,default:0" json:"order"`
	Metadata        EntryData `bun:"metadata,type:jsonb,nullzero" json:"metadata,omitempty"`
	CreatedAt       time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	SourceEntry *ContentEntry `bun:"rel:belongs-to,join:source_entry_id=id" json:"sourceEntry,omitempty"`
	TargetEntry *ContentEntry `bun:"rel:belongs-to,join:target_entry_id=id" json:"targetEntry,omitempty"`
}

// TableName returns the table name for ContentRelation
func (cr *ContentRelation) TableName() string {
	return "cms_content_relations"
}

// BeforeInsert sets default values before insert
func (cr *ContentRelation) BeforeInsert() {
	if cr.ID.IsNil() {
		cr.ID = xid.New()
	}
	cr.CreatedAt = time.Now()
}

// NewRelation creates a new relation between two entries
func NewRelation(sourceID, targetID xid.ID, fieldName string) *ContentRelation {
	return &ContentRelation{
		ID:            xid.New(),
		SourceEntryID: sourceID,
		TargetEntryID: targetID,
		FieldName:     fieldName,
		Order:         0,
		CreatedAt:     time.Now(),
	}
}

// NewOrderedRelation creates a new relation with ordering
func NewOrderedRelation(sourceID, targetID xid.ID, fieldName string, order int) *ContentRelation {
	return &ContentRelation{
		ID:            xid.New(),
		SourceEntryID: sourceID,
		TargetEntryID: targetID,
		FieldName:     fieldName,
		Order:         order,
		CreatedAt:     time.Now(),
	}
}

// ContentTypeRelation represents a relation definition between content types
// This is metadata about how two content types are related
type ContentTypeRelation struct {
	bun.BaseModel `bun:"table:cms_content_type_relations,alias:ctr"`

	ID                  xid.ID    `bun:"id,pk,type:varchar(20)" json:"id"`
	SourceContentTypeID xid.ID    `bun:"source_content_type_id,notnull,type:varchar(20)" json:"sourceContentTypeId"`
	TargetContentTypeID xid.ID    `bun:"target_content_type_id,notnull,type:varchar(20)" json:"targetContentTypeId"`
	SourceFieldName     string    `bun:"source_field_name,notnull" json:"sourceFieldName"`
	TargetFieldName     string    `bun:"target_field_name,nullzero" json:"targetFieldName,omitempty"`
	RelationType        string    `bun:"relation_type,notnull" json:"relationType"`
	OnDelete            string    `bun:"on_delete,notnull,default:'setNull'" json:"onDelete"`
	CreatedAt           time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	SourceContentType *ContentType `bun:"rel:belongs-to,join:source_content_type_id=id" json:"sourceContentType,omitempty"`
	TargetContentType *ContentType `bun:"rel:belongs-to,join:target_content_type_id=id" json:"targetContentType,omitempty"`
}

// TableName returns the table name for ContentTypeRelation
func (ctr *ContentTypeRelation) TableName() string {
	return "cms_content_type_relations"
}

// BeforeInsert sets default values before insert
func (ctr *ContentTypeRelation) BeforeInsert() {
	if ctr.ID.IsNil() {
		ctr.ID = xid.New()
	}
	if ctr.OnDelete == "" {
		ctr.OnDelete = "setNull"
	}
	ctr.CreatedAt = time.Now()
}

// IsOneToOne returns true if this is a one-to-one relation
func (ctr *ContentTypeRelation) IsOneToOne() bool {
	return ctr.RelationType == "oneToOne"
}

// IsOneToMany returns true if this is a one-to-many relation
func (ctr *ContentTypeRelation) IsOneToMany() bool {
	return ctr.RelationType == "oneToMany"
}

// IsManyToOne returns true if this is a many-to-one relation
func (ctr *ContentTypeRelation) IsManyToOne() bool {
	return ctr.RelationType == "manyToOne"
}

// IsManyToMany returns true if this is a many-to-many relation
func (ctr *ContentTypeRelation) IsManyToMany() bool {
	return ctr.RelationType == "manyToMany"
}

// IsBidirectional returns true if this relation has an inverse field
func (ctr *ContentTypeRelation) IsBidirectional() bool {
	return ctr.TargetFieldName != ""
}

// RequiresJoinTable returns true if this relation needs a join table
func (ctr *ContentTypeRelation) RequiresJoinTable() bool {
	return ctr.RelationType == "manyToMany"
}

// GetOnDeleteAction returns the on-delete action
func (ctr *ContentTypeRelation) GetOnDeleteAction() string {
	if ctr.OnDelete == "" {
		return "setNull"
	}
	return ctr.OnDelete
}

// NewContentTypeRelation creates a new content type relation definition
func NewContentTypeRelation(
	sourceTypeID, targetTypeID xid.ID,
	sourceField, targetField string,
	relationType string,
	onDelete string,
) *ContentTypeRelation {
	if onDelete == "" {
		onDelete = "setNull"
	}
	return &ContentTypeRelation{
		ID:                  xid.New(),
		SourceContentTypeID: sourceTypeID,
		TargetContentTypeID: targetTypeID,
		SourceFieldName:     sourceField,
		TargetFieldName:     targetField,
		RelationType:        relationType,
		OnDelete:            onDelete,
		CreatedAt:           time.Now(),
	}
}

