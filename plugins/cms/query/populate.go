package query

import (
	"context"
	"strings"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// PopulateConfig configures relation population.
type PopulateConfig struct {
	// Fields to populate (comma-separated or array)
	Fields []string
	// MaxDepth limits recursive population (default: 1)
	MaxDepth int
	// SelectFields limits which fields to return from related entries
	SelectFields map[string][]string
}

// DefaultPopulateConfig returns the default populate configuration.
func DefaultPopulateConfig() *PopulateConfig {
	return &PopulateConfig{
		Fields:       []string{},
		MaxDepth:     1,
		SelectFields: make(map[string][]string),
	}
}

// ParsePopulate parses a populate string into config
// Format: "field1,field2" or "field1.subfield,field2".
func ParsePopulate(populate string) *PopulateConfig {
	if populate == "" {
		return nil
	}

	config := DefaultPopulateConfig()

	fields := strings.SplitSeq(populate, ",")
	for f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			config.Fields = append(config.Fields, f)
		}
	}

	return config
}

// Populator handles relation population for queries.
type Populator struct {
	db *bun.DB
}

// NewPopulator creates a new populator.
func NewPopulator(db *bun.DB) *Populator {
	return &Populator{db: db}
}

// PopulateEntries populates relations for a slice of entries.
func (p *Populator) PopulateEntries(ctx context.Context, entries []*schema.ContentEntry, config *PopulateConfig) error {
	if config == nil || len(config.Fields) == 0 || len(entries) == 0 {
		return nil
	}

	// Collect all entry IDs
	entryIDs := make([]xid.ID, len(entries))
	for i, e := range entries {
		entryIDs[i] = e.ID
	}

	// For each field to populate
	for _, fieldSlug := range config.Fields {
		// Handle nested populate (field.subfield)
		parts := strings.SplitN(fieldSlug, ".", 2)
		baseField := parts[0]

		// Get all relations for this field across all entries
		var relations []*schema.ContentRelation

		err := p.db.NewSelect().
			Model(&relations).
			Where("source_entry_id IN (?)", bun.In(entryIDs)).
			Where("field_slug = ?", baseField).
			Order("\"order\" ASC").
			Relation("TargetEntry").
			Scan(ctx)
		if err != nil {
			return core.ErrInternalError("failed to populate relations for field: "+baseField, err)
		}

		// Group relations by source entry
		relationsBySource := make(map[xid.ID][]*schema.ContentEntry)

		for _, rel := range relations {
			if rel.TargetEntry != nil {
				relationsBySource[rel.SourceEntryID] = append(relationsBySource[rel.SourceEntryID], rel.TargetEntry)
			}
		}

		// Assign populated relations to entries
		for _, entry := range entries {
			if entry.PopulatedRelations == nil {
				entry.PopulatedRelations = make(map[string][]*schema.ContentEntry)
			}

			if relatedEntries, ok := relationsBySource[entry.ID]; ok {
				entry.PopulatedRelations[baseField] = relatedEntries
			} else {
				entry.PopulatedRelations[baseField] = []*schema.ContentEntry{}
			}
		}

		// Handle nested population if specified and depth allows
		if len(parts) > 1 && config.MaxDepth > 1 {
			// Collect related entries for nested population
			var nestedEntries []*schema.ContentEntry

			for _, entry := range entries {
				if related, ok := entry.PopulatedRelations[baseField]; ok {
					nestedEntries = append(nestedEntries, related...)
				}
			}

			// Recursively populate nested entries
			if len(nestedEntries) > 0 {
				nestedConfig := &PopulateConfig{
					Fields:       []string{parts[1]},
					MaxDepth:     config.MaxDepth - 1,
					SelectFields: config.SelectFields,
				}
				if err := p.PopulateEntries(ctx, nestedEntries, nestedConfig); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// PopulateEntry populates relations for a single entry.
func (p *Populator) PopulateEntry(ctx context.Context, entry *schema.ContentEntry, config *PopulateConfig) error {
	if entry == nil {
		return nil
	}

	return p.PopulateEntries(ctx, []*schema.ContentEntry{entry}, config)
}

// GetRelatedEntryIDs returns the IDs of related entries for a field.
func (p *Populator) GetRelatedEntryIDs(ctx context.Context, entryID xid.ID, fieldSlug string) ([]xid.ID, error) {
	var relations []*schema.ContentRelation

	err := p.db.NewSelect().
		Model(&relations).
		Where("source_entry_id = ?", entryID).
		Where("field_slug = ?", fieldSlug).
		Order("\"order\" ASC").
		Scan(ctx)
	if err != nil {
		return nil, core.ErrInternalError("failed to get related entry IDs", err)
	}

	ids := make([]xid.ID, len(relations))
	for i, rel := range relations {
		ids[i] = rel.TargetEntryID
	}

	return ids, nil
}

// GetRelatedEntries returns the related entries for a field.
func (p *Populator) GetRelatedEntries(ctx context.Context, entryID xid.ID, fieldSlug string) ([]*schema.ContentEntry, error) {
	var relations []*schema.ContentRelation

	err := p.db.NewSelect().
		Model(&relations).
		Where("source_entry_id = ?", entryID).
		Where("field_slug = ?", fieldSlug).
		Order("\"order\" ASC").
		Relation("TargetEntry").
		Scan(ctx)
	if err != nil {
		return nil, core.ErrInternalError("failed to get related entries", err)
	}

	entries := make([]*schema.ContentEntry, 0, len(relations))
	for _, rel := range relations {
		if rel.TargetEntry != nil {
			entries = append(entries, rel.TargetEntry)
		}
	}

	return entries, nil
}

// GetReverseRelatedEntries returns entries that reference this entry.
func (p *Populator) GetReverseRelatedEntries(ctx context.Context, entryID xid.ID, fieldSlug string) ([]*schema.ContentEntry, error) {
	var relations []*schema.ContentRelation

	err := p.db.NewSelect().
		Model(&relations).
		Where("target_entry_id = ?", entryID).
		Where("field_slug = ?", fieldSlug).
		Order("\"order\" ASC").
		Relation("SourceEntry").
		Scan(ctx)
	if err != nil {
		return nil, core.ErrInternalError("failed to get reverse related entries", err)
	}

	entries := make([]*schema.ContentEntry, 0, len(relations))
	for _, rel := range relations {
		if rel.SourceEntry != nil {
			entries = append(entries, rel.SourceEntry)
		}
	}

	return entries, nil
}

// PopulateEntryDTO populates relation fields in an entry DTO.
func PopulateEntryDTO(entry *schema.ContentEntry, dto *core.ContentEntryDTO) {
	if entry == nil || dto == nil || entry.PopulatedRelations == nil {
		return
	}

	// Initialize relations map in DTO if needed
	if dto.Relations == nil {
		dto.Relations = make(map[string][]string)
	}

	// Add populated relation IDs to the DTO
	for fieldSlug, relatedEntries := range entry.PopulatedRelations {
		ids := make([]string, len(relatedEntries))
		for i, related := range relatedEntries {
			ids[i] = related.ID.String()
		}

		dto.Relations[fieldSlug] = ids
	}
}
