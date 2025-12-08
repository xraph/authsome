package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// ContentRevision stores historical versions of content entries
type ContentRevision struct {
	bun.BaseModel `bun:"table:cms_content_revisions,alias:cr"`

	ID           xid.ID    `bun:"id,pk,type:varchar(20)" json:"id"`
	EntryID      xid.ID    `bun:"entry_id,notnull,type:varchar(20)" json:"entryId"`
	Version      int       `bun:"version,notnull" json:"version"`
	Data         EntryData `bun:"data,type:jsonb,notnull" json:"data"`
	Status       string    `bun:"status,notnull" json:"status"`
	ChangeReason string    `bun:"change_reason,nullzero" json:"changeReason"`
	ChangedBy    xid.ID    `bun:"changed_by,type:varchar(20)" json:"changedBy"`
	CreatedAt    time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Entry *ContentEntry `bun:"rel:belongs-to,join:entry_id=id" json:"entry,omitempty"`
}

// TableName returns the table name for ContentRevision
func (cr *ContentRevision) TableName() string {
	return "cms_content_revisions"
}

// BeforeInsert sets default values before insert
func (cr *ContentRevision) BeforeInsert() {
	if cr.ID.IsNil() {
		cr.ID = xid.New()
	}
	if cr.Data == nil {
		cr.Data = make(EntryData)
	}
	cr.CreatedAt = time.Now()
}

// CreateFromEntry creates a new revision from an entry
func CreateRevisionFromEntry(entry *ContentEntry, changeReason string, changedBy xid.ID) *ContentRevision {
	rev := &ContentRevision{
		ID:           xid.New(),
		EntryID:      entry.ID,
		Version:      entry.Version,
		Data:         make(EntryData),
		Status:       entry.Status,
		ChangeReason: changeReason,
		ChangedBy:    changedBy,
		CreatedAt:    time.Now(),
	}
	// Deep copy data
	for k, v := range entry.Data {
		rev.Data[k] = v
	}
	return rev
}

// RestoreToEntry restores this revision to the given entry
func (cr *ContentRevision) RestoreToEntry(entry *ContentEntry) {
	entry.Data = make(EntryData)
	for k, v := range cr.Data {
		entry.Data[k] = v
	}
	entry.Status = cr.Status
}

// ToMap converts the revision to a map for API responses
func (cr *ContentRevision) ToMap() map[string]any {
	m := map[string]any{
		"id":        cr.ID.String(),
		"entryId":   cr.EntryID.String(),
		"version":   cr.Version,
		"data":      cr.Data,
		"status":    cr.Status,
		"createdAt": cr.CreatedAt,
	}
	if cr.ChangeReason != "" {
		m["changeReason"] = cr.ChangeReason
	}
	if !cr.ChangedBy.IsNil() {
		m["changedBy"] = cr.ChangedBy.String()
	}
	return m
}

// CompareData compares this revision's data with another and returns the differences
func (cr *ContentRevision) CompareData(other *ContentRevision) map[string]FieldDiff {
	diffs := make(map[string]FieldDiff)

	// Check fields in current revision
	for key, value := range cr.Data {
		otherValue, exists := other.Data[key]
		if !exists {
			diffs[key] = FieldDiff{
				Field:    key,
				OldValue: nil,
				NewValue: value,
				Type:     DiffTypeAdded,
			}
		} else if !deepEqual(value, otherValue) {
			diffs[key] = FieldDiff{
				Field:    key,
				OldValue: otherValue,
				NewValue: value,
				Type:     DiffTypeModified,
			}
		}
	}

	// Check for removed fields
	for key, value := range other.Data {
		if _, exists := cr.Data[key]; !exists {
			diffs[key] = FieldDiff{
				Field:    key,
				OldValue: value,
				NewValue: nil,
				Type:     DiffTypeRemoved,
			}
		}
	}

	return diffs
}

// DiffType represents the type of change in a field
type DiffType string

const (
	DiffTypeAdded    DiffType = "added"
	DiffTypeRemoved  DiffType = "removed"
	DiffTypeModified DiffType = "modified"
)

// FieldDiff represents a difference in a field between revisions
type FieldDiff struct {
	Field    string   `json:"field"`
	OldValue any      `json:"oldValue,omitempty"`
	NewValue any      `json:"newValue,omitempty"`
	Type     DiffType `json:"type"`
}

// deepEqual compares two values for equality
func deepEqual(a, b any) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle maps
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		if len(aMap) != len(bMap) {
			return false
		}
		for k, v := range aMap {
			if bv, ok := bMap[k]; !ok || !deepEqual(v, bv) {
				return false
			}
		}
		return true
	}

	// Handle slices
	aSlice, aIsSlice := a.([]any)
	bSlice, bIsSlice := b.([]any)
	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return false
		}
		for i, v := range aSlice {
			if !deepEqual(v, bSlice[i]) {
				return false
			}
		}
		return true
	}

	// Default comparison
	return a == b
}
