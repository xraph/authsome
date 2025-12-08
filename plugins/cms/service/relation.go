package service

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/repository"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// RelationServiceConfig holds configuration for the relation service
type RelationServiceConfig struct {
	// MaxRelationsPerField limits relations per field (0 = unlimited)
	MaxRelationsPerField int
	// AllowCircularRelations allows circular references (default: false)
	AllowCircularRelations bool
	// Logger for relation operations
	Logger forge.Logger
}

// RelationService handles content relations
type RelationService struct {
	repo            repository.RelationRepository
	entryRepo       repository.ContentEntryRepository
	contentTypeRepo repository.ContentTypeRepository
	config          RelationServiceConfig
	logger          forge.Logger
}

// NewRelationService creates a new relation service
func NewRelationService(
	repo repository.RelationRepository,
	entryRepo repository.ContentEntryRepository,
	contentTypeRepo repository.ContentTypeRepository,
	config RelationServiceConfig,
) *RelationService {
	svc := &RelationService{
		repo:            repo,
		entryRepo:       entryRepo,
		contentTypeRepo: contentTypeRepo,
		config:          config,
	}
	if config.Logger != nil {
		svc.logger = config.Logger.Named("relation")
	}
	return svc
}

// =============================================================================
// Content Relations (entry-to-entry)
// =============================================================================

// SetRelation sets a single relation (for one-to-one or many-to-one)
func (s *RelationService) SetRelation(ctx context.Context, sourceID xid.ID, fieldSlug string, targetID xid.ID) error {
	// Verify target entry exists
	if _, err := s.entryRepo.FindByID(ctx, targetID); err != nil {
		return core.ErrEntryNotFound(targetID.String())
	}

	// Check for circular relations if disabled
	if !s.config.AllowCircularRelations {
		if err := s.checkCircularRelation(ctx, sourceID, targetID, fieldSlug, nil); err != nil {
			return err
		}
	}

	// Delete existing relation for this field (one-to-one behavior)
	if err := s.repo.DeleteAllForField(ctx, sourceID, fieldSlug); err != nil {
		return err
	}

	// Create new relation
	relation := schema.NewRelation(sourceID, targetID, fieldSlug)
	return s.repo.CreateRelation(ctx, relation)
}

// SetRelations sets multiple relations (for one-to-many or many-to-many)
func (s *RelationService) SetRelations(ctx context.Context, sourceID xid.ID, fieldSlug string, targetIDs []xid.ID) error {
	// Check limit
	if s.config.MaxRelationsPerField > 0 && len(targetIDs) > s.config.MaxRelationsPerField {
		return core.ErrInvalidRelation(fmt.Sprintf("exceeds maximum relations per field (%d)", s.config.MaxRelationsPerField))
	}

	// Verify all target entries exist
	for _, targetID := range targetIDs {
		if _, err := s.entryRepo.FindByID(ctx, targetID); err != nil {
			return core.ErrEntryNotFound(targetID.String())
		}
	}

	// Check for circular relations
	if !s.config.AllowCircularRelations {
		for _, targetID := range targetIDs {
			if err := s.checkCircularRelation(ctx, sourceID, targetID, fieldSlug, nil); err != nil {
				return err
			}
		}
	}

	// Delete existing relations for this field
	if err := s.repo.DeleteAllForField(ctx, sourceID, fieldSlug); err != nil {
		return err
	}

	// Create new relations with order
	relations := make([]*schema.ContentRelation, len(targetIDs))
	for i, targetID := range targetIDs {
		relations[i] = schema.NewOrderedRelation(sourceID, targetID, fieldSlug, i)
	}

	return s.repo.BulkCreateRelations(ctx, relations)
}

// AddRelation adds a single relation to existing relations
func (s *RelationService) AddRelation(ctx context.Context, sourceID xid.ID, fieldSlug string, targetID xid.ID) error {
	// Verify target entry exists
	if _, err := s.entryRepo.FindByID(ctx, targetID); err != nil {
		return core.ErrEntryNotFound(targetID.String())
	}

	// Check for circular relations
	if !s.config.AllowCircularRelations {
		if err := s.checkCircularRelation(ctx, sourceID, targetID, fieldSlug, nil); err != nil {
			return err
		}
	}

	// Get existing relations to determine order
	existing, err := s.repo.FindRelations(ctx, sourceID, fieldSlug)
	if err != nil {
		return err
	}

	// Check limit
	if s.config.MaxRelationsPerField > 0 && len(existing) >= s.config.MaxRelationsPerField {
		return core.ErrInvalidRelation(fmt.Sprintf("exceeds maximum relations per field (%d)", s.config.MaxRelationsPerField))
	}

	// Check if relation already exists
	for _, rel := range existing {
		if rel.TargetEntryID == targetID {
			return nil // Already exists, no-op
		}
	}

	// Create new relation at end
	relation := schema.NewOrderedRelation(sourceID, targetID, fieldSlug, len(existing))
	return s.repo.CreateRelation(ctx, relation)
}

// RemoveRelation removes a single relation
func (s *RelationService) RemoveRelation(ctx context.Context, sourceID xid.ID, fieldSlug string, targetID xid.ID) error {
	return s.repo.DeleteRelationByEntries(ctx, sourceID, targetID, fieldSlug)
}

// GetRelations returns all related entries for a field
func (s *RelationService) GetRelations(ctx context.Context, sourceID xid.ID, fieldSlug string) ([]*core.RelatedEntryDTO, error) {
	relations, err := s.repo.FindRelations(ctx, sourceID, fieldSlug)
	if err != nil {
		return nil, err
	}

	result := make([]*core.RelatedEntryDTO, len(relations))
	for i, rel := range relations {
		result[i] = &core.RelatedEntryDTO{
			ID:    rel.TargetEntryID.String(),
			Order: rel.Order,
		}
		if rel.TargetEntry != nil {
			result[i].Entry = s.entryToSummaryDTO(rel.TargetEntry)
		}
	}
	return result, nil
}

// GetRelatedIDs returns just the IDs of related entries
func (s *RelationService) GetRelatedIDs(ctx context.Context, sourceID xid.ID, fieldSlug string) ([]xid.ID, error) {
	relations, err := s.repo.FindRelations(ctx, sourceID, fieldSlug)
	if err != nil {
		return nil, err
	}

	ids := make([]xid.ID, len(relations))
	for i, rel := range relations {
		ids[i] = rel.TargetEntryID
	}
	return ids, nil
}

// GetReverseRelations returns all entries that reference this entry
func (s *RelationService) GetReverseRelations(ctx context.Context, targetID xid.ID, fieldSlug string) ([]*core.RelatedEntryDTO, error) {
	relations, err := s.repo.FindReverseRelations(ctx, targetID, fieldSlug)
	if err != nil {
		return nil, err
	}

	result := make([]*core.RelatedEntryDTO, len(relations))
	for i, rel := range relations {
		result[i] = &core.RelatedEntryDTO{
			ID:    rel.SourceEntryID.String(),
			Order: rel.Order,
		}
		if rel.SourceEntry != nil {
			result[i].Entry = s.entryToSummaryDTO(rel.SourceEntry)
		}
	}
	return result, nil
}

// ReorderRelations reorders the relations for a field
func (s *RelationService) ReorderRelations(ctx context.Context, sourceID xid.ID, fieldSlug string, orderedTargetIDs []xid.ID) error {
	return s.repo.BulkUpdateOrder(ctx, sourceID, fieldSlug, orderedTargetIDs)
}

// ClearRelations removes all relations for a field
func (s *RelationService) ClearRelations(ctx context.Context, sourceID xid.ID, fieldSlug string) error {
	return s.repo.DeleteAllForField(ctx, sourceID, fieldSlug)
}

// DeleteAllEntryRelations removes all relations for an entry (when deleting entry)
func (s *RelationService) DeleteAllEntryRelations(ctx context.Context, entryID xid.ID) error {
	return s.repo.DeleteAllForEntry(ctx, entryID)
}

// =============================================================================
// Content Type Relations (type-to-type definitions)
// =============================================================================

// CreateTypeRelation creates a new type relation definition
func (s *RelationService) CreateTypeRelation(ctx context.Context, req *core.CreateTypeRelationRequest) (*core.TypeRelationDTO, error) {
	// Validate relation type
	relationType := core.RelationType(req.RelationType)
	if !relationType.IsValid() {
		return nil, core.ErrInvalidRelation("invalid relation type: " + req.RelationType)
	}

	// Validate source content type exists
	sourceType, err := s.contentTypeRepo.FindByID(ctx, req.SourceContentTypeID)
	if err != nil {
		return nil, core.ErrContentTypeNotFound(req.SourceContentTypeID.String())
	}

	// Validate target content type exists
	targetType, err := s.contentTypeRepo.FindByID(ctx, req.TargetContentTypeID)
	if err != nil {
		return nil, core.ErrContentTypeNotFound(req.TargetContentTypeID.String())
	}

	// Validate on-delete action
	onDelete := core.OnDeleteAction(req.OnDelete)
	if req.OnDelete != "" && !onDelete.IsValid() {
		return nil, core.ErrInvalidRelation("invalid on-delete action: " + req.OnDelete)
	}
	if req.OnDelete == "" {
		onDelete = core.OnDeleteSetNull
	}

	// Create type relation
	relation := schema.NewContentTypeRelation(
		req.SourceContentTypeID,
		req.TargetContentTypeID,
		req.SourceFieldName,
		req.TargetFieldName,
		req.RelationType,
		string(onDelete),
	)

	if err := s.repo.CreateTypeRelation(ctx, relation); err != nil {
		return nil, err
	}

	return s.typeRelationToDTO(relation, sourceType, targetType), nil
}

// UpdateTypeRelation updates a type relation definition
func (s *RelationService) UpdateTypeRelation(ctx context.Context, id xid.ID, req *core.UpdateTypeRelationRequest) (*core.TypeRelationDTO, error) {
	relation, err := s.repo.FindTypeRelationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.TargetFieldName != nil {
		relation.TargetFieldName = *req.TargetFieldName
	}
	if req.OnDelete != nil {
		onDelete := core.OnDeleteAction(*req.OnDelete)
		if !onDelete.IsValid() {
			return nil, core.ErrInvalidRelation("invalid on-delete action: " + *req.OnDelete)
		}
		relation.OnDelete = *req.OnDelete
	}

	if err := s.repo.UpdateTypeRelation(ctx, relation); err != nil {
		return nil, err
	}

	return s.typeRelationToDTO(relation, relation.SourceContentType, relation.TargetContentType), nil
}

// DeleteTypeRelation deletes a type relation definition
func (s *RelationService) DeleteTypeRelation(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteTypeRelation(ctx, id)
}

// GetTypeRelation gets a type relation by ID
func (s *RelationService) GetTypeRelation(ctx context.Context, id xid.ID) (*core.TypeRelationDTO, error) {
	relation, err := s.repo.FindTypeRelationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.typeRelationToDTO(relation, relation.SourceContentType, relation.TargetContentType), nil
}

// GetTypeRelationByField gets a type relation by content type and field
func (s *RelationService) GetTypeRelationByField(ctx context.Context, contentTypeID xid.ID, fieldSlug string) (*core.TypeRelationDTO, error) {
	relation, err := s.repo.FindTypeRelationByField(ctx, contentTypeID, fieldSlug)
	if err != nil {
		return nil, err
	}
	if relation == nil {
		return nil, nil
	}
	return s.typeRelationToDTO(relation, relation.SourceContentType, relation.TargetContentType), nil
}

// GetTypeRelationsForType gets all type relations for a content type
func (s *RelationService) GetTypeRelationsForType(ctx context.Context, contentTypeID xid.ID) ([]*core.TypeRelationDTO, error) {
	relations, err := s.repo.FindTypeRelationsForType(ctx, contentTypeID)
	if err != nil {
		return nil, err
	}

	result := make([]*core.TypeRelationDTO, len(relations))
	for i, rel := range relations {
		result[i] = s.typeRelationToDTO(rel, rel.SourceContentType, rel.TargetContentType)
	}
	return result, nil
}

// =============================================================================
// Populate Support (for query builder)
// =============================================================================

// PopulateRelations populates relation fields on entries
func (s *RelationService) PopulateRelations(ctx context.Context, entries []*schema.ContentEntry, fieldSlugs []string) error {
	if len(entries) == 0 || len(fieldSlugs) == 0 {
		return nil
	}

	for _, entry := range entries {
		if entry.PopulatedRelations == nil {
			entry.PopulatedRelations = make(map[string][]*schema.ContentEntry)
		}

		for _, fieldSlug := range fieldSlugs {
			relations, err := s.repo.FindRelations(ctx, entry.ID, fieldSlug)
			if err != nil {
				if s.logger != nil {
					s.logger.Warn("failed to populate relation",
						forge.F("entryID", entry.ID.String()),
						forge.F("field", fieldSlug),
						forge.F("error", err.Error()))
				}
				continue
			}

			relatedEntries := make([]*schema.ContentEntry, 0, len(relations))
			for _, rel := range relations {
				if rel.TargetEntry != nil {
					relatedEntries = append(relatedEntries, rel.TargetEntry)
				}
			}
			entry.PopulatedRelations[fieldSlug] = relatedEntries
		}
	}

	return nil
}

// PopulateRelationsMap returns populated relations as a map of field -> entries
func (s *RelationService) PopulateRelationsMap(ctx context.Context, entryID xid.ID, fieldSlugs []string) (map[string][]*core.ContentEntrySummaryDTO, error) {
	result := make(map[string][]*core.ContentEntrySummaryDTO)

	for _, fieldSlug := range fieldSlugs {
		relations, err := s.repo.FindRelations(ctx, entryID, fieldSlug)
		if err != nil {
			return nil, err
		}

		entries := make([]*core.ContentEntrySummaryDTO, 0, len(relations))
		for _, rel := range relations {
			if rel.TargetEntry != nil {
				entries = append(entries, s.entryToSummaryDTO(rel.TargetEntry))
			}
		}
		result[fieldSlug] = entries
	}

	return result, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// checkCircularRelation checks for circular relations
func (s *RelationService) checkCircularRelation(ctx context.Context, sourceID, targetID xid.ID, fieldSlug string, visited map[xid.ID]bool) error {
	if visited == nil {
		visited = make(map[xid.ID]bool)
	}

	// Check if we've already visited target (would create cycle)
	if visited[sourceID] {
		return core.ErrCircularRelation(fmt.Sprintf("%s -> %s creates a cycle", sourceID.String(), targetID.String()))
	}

	// If target equals source, it's a self-reference cycle
	if sourceID == targetID {
		return core.ErrCircularRelation("self-reference detected")
	}

	visited[sourceID] = true

	// Get relations from target to check deeper cycles
	relations, err := s.repo.FindAllRelations(ctx, targetID)
	if err != nil {
		return err
	}

	// Check each relation from target
	for _, rel := range relations {
		if err := s.checkCircularRelation(ctx, targetID, rel.TargetEntryID, rel.FieldName, visited); err != nil {
			return err
		}
	}

	return nil
}

// entryToSummaryDTO converts a schema entry to summary DTO
func (s *RelationService) entryToSummaryDTO(entry *schema.ContentEntry) *core.ContentEntrySummaryDTO {
	if entry == nil {
		return nil
	}
	return &core.ContentEntrySummaryDTO{
		ID:        entry.ID.String(),
		Status:    entry.Status,
		Version:   entry.Version,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
	}
}

// typeRelationToDTO converts a schema type relation to DTO
func (s *RelationService) typeRelationToDTO(rel *schema.ContentTypeRelation, source, target *schema.ContentType) *core.TypeRelationDTO {
	if rel == nil {
		return nil
	}

	dto := &core.TypeRelationDTO{
		ID:                  rel.ID.String(),
		SourceContentTypeID: rel.SourceContentTypeID.String(),
		TargetContentTypeID: rel.TargetContentTypeID.String(),
		SourceFieldName:     rel.SourceFieldName,
		TargetFieldName:     rel.TargetFieldName,
		RelationType:        rel.RelationType,
		OnDelete:            rel.OnDelete,
		CreatedAt:           rel.CreatedAt,
	}

	if source != nil {
		dto.SourceContentTypeName = source.Name
		dto.SourceContentTypeName = source.Name
	}
	if target != nil {
		dto.TargetContentTypeName = target.Name
		dto.TargetContentTypeName = target.Name
	}

	return dto
}
