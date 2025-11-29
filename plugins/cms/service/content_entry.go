package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/repository"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// ContentEntryService handles content entry business logic
type ContentEntryService struct {
	repo            repository.ContentEntryRepository
	contentTypeRepo repository.ContentTypeRepository
	revisionRepo    repository.RevisionRepository
	config          ContentEntryServiceConfig
	logger          forge.Logger
}

// ContentEntryServiceConfig holds configuration for the service
type ContentEntryServiceConfig struct {
	EnableRevisions      bool
	MaxRevisionsPerEntry int
	Logger               forge.Logger
}

// NewContentEntryService creates a new content entry service
func NewContentEntryService(
	repo repository.ContentEntryRepository,
	contentTypeRepo repository.ContentTypeRepository,
	revisionRepo repository.RevisionRepository,
	config ContentEntryServiceConfig,
) *ContentEntryService {
	return &ContentEntryService{
		repo:            repo,
		contentTypeRepo: contentTypeRepo,
		revisionRepo:    revisionRepo,
		config:          config,
		logger:          config.Logger,
	}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new content entry
func (s *ContentEntryService) Create(ctx context.Context, contentTypeID xid.ID, req *core.CreateEntryRequest) (*core.ContentEntryDTO, error) {
	// Get app/env context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, core.ErrEnvContextMissing()
	}

	// Get content type with fields
	contentType, err := s.contentTypeRepo.FindWithFields(ctx, contentTypeID)
	if err != nil {
		return nil, err
	}

	// Check entry limit
	if contentType.Settings.MaxEntries > 0 {
		count, err := s.repo.Count(ctx, contentTypeID)
		if err != nil {
			return nil, err
		}
		if count >= contentType.Settings.MaxEntries {
			return nil, core.ErrEntryLimitReached(contentType.Slug, contentType.Settings.MaxEntries)
		}
	}

	// Create validator
	validator := NewEntryValidator(contentType)

	// Apply defaults
	data := validator.ApplyDefaults(req.Data)

	// Sanitize data (remove read-only fields)
	data = validator.SanitizeData(data)

	// Validate data
	validationResult := validator.ValidateCreate(data)
	if !validationResult.Valid {
		return nil, core.ErrEntryValidationFailed(ValidationResultToMap(validationResult))
	}

	// Validate unique constraints
	uniqueResult := validator.ValidateUniqueConstraints(data, nil, func(field string, value interface{}, excludeID *xid.ID) (bool, error) {
		return s.repo.ExistsWithFieldValue(ctx, contentTypeID, field, value, excludeID)
	})
	if !uniqueResult.Valid {
		return nil, core.ErrEntryValidationFailed(ValidationResultToMap(uniqueResult))
	}

	// Get user ID from context
	userID, _ := contexts.GetUserID(ctx)

	// Determine status
	status := "draft"
	if req.Status != "" {
		if _, valid := core.ParseEntryStatus(req.Status); valid {
			status = req.Status
		}
	}

	// Create entry
	entry := &schema.ContentEntry{
		ID:            xid.New(),
		ContentTypeID: contentTypeID,
		AppID:         appID,
		EnvironmentID: envID,
		Data:          data,
		Status:        status,
		Version:       1,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	}

	// Handle scheduling
	if status == "scheduled" && req.ScheduledAt != nil {
		entry.ScheduledAt = req.ScheduledAt
	}

	// Handle immediate publish
	if status == "published" {
		now := time.Now()
		entry.PublishedAt = &now
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return nil, err
	}

	// Create initial revision if enabled
	if s.config.EnableRevisions && contentType.HasRevisions() {
		revision := schema.CreateRevisionFromEntry(entry, "Initial creation", userID)
		if s.revisionRepo != nil {
			_ = s.revisionRepo.Create(ctx, revision)
		}
	}

	return s.toDTO(entry, contentType), nil
}

// GetByID retrieves a content entry by ID
func (s *ContentEntryService) GetByID(ctx context.Context, id xid.ID) (*core.ContentEntryDTO, error) {
	entry, err := s.repo.FindByIDWithType(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toDTO(entry, entry.ContentType), nil
}

// List lists content entries with filtering and pagination
func (s *ContentEntryService) List(ctx context.Context, contentTypeID xid.ID, query *core.ListEntriesQuery) (*core.ListEntriesResponse, error) {
	// Convert to repository query
	repoQuery := &repository.EntryListQuery{
		Status:      query.Status,
		Search:      query.Search,
		SortBy:      query.SortBy,
		SortOrder:   query.SortOrder,
		Page:        query.Page,
		PageSize:    query.PageSize,
		Select:      query.Select,
		IncludeType: true,
	}

	// Convert filters
	if len(query.Filters) > 0 {
		repoQuery.Filters = make(map[string]repository.FilterCondition)
		for field, value := range query.Filters {
			// Simple equality filter for now
			repoQuery.Filters[field] = repository.FilterCondition{
				Operator: "eq",
				Value:    value,
			}
		}
	}

	entries, total, err := s.repo.List(ctx, contentTypeID, repoQuery)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	dtos := make([]*core.ContentEntryDTO, len(entries))
	for i, entry := range entries {
		dtos[i] = s.toDTO(entry, entry.ContentType)
	}

	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &core.ListEntriesResponse{
		Entries:    dtos,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

// Update updates a content entry
func (s *ContentEntryService) Update(ctx context.Context, id xid.ID, req *core.UpdateEntryRequest) (*core.ContentEntryDTO, error) {
	// Get existing entry with content type
	entry, err := s.repo.FindByIDWithType(ctx, id)
	if err != nil {
		return nil, err
	}

	contentType := entry.ContentType

	// Create validator
	validator := NewEntryValidator(contentType)

	// Sanitize incoming data
	newData := validator.SanitizeData(req.Data)

	// Merge with existing data
	mergedData := make(map[string]interface{})
	for k, v := range entry.Data {
		mergedData[k] = v
	}
	for k, v := range newData {
		mergedData[k] = v
	}

	// Validate merged data
	validationResult := validator.ValidateUpdate(mergedData, entry)
	if !validationResult.Valid {
		return nil, core.ErrEntryValidationFailed(ValidationResultToMap(validationResult))
	}

	// Validate unique constraints
	uniqueResult := validator.ValidateUniqueConstraints(mergedData, &entry.ID, func(field string, value interface{}, excludeID *xid.ID) (bool, error) {
		return s.repo.ExistsWithFieldValue(ctx, entry.ContentTypeID, field, value, excludeID)
	})
	if !uniqueResult.Valid {
		return nil, core.ErrEntryValidationFailed(ValidationResultToMap(uniqueResult))
	}

	// Get user ID from context
	userID, _ := contexts.GetUserID(ctx)

	// Create revision before update if enabled
	if s.config.EnableRevisions && contentType.HasRevisions() && s.revisionRepo != nil {
		revision := schema.CreateRevisionFromEntry(entry, req.ChangeReason, userID)
		_ = s.revisionRepo.Create(ctx, revision)

		// Cleanup old revisions
		if s.config.MaxRevisionsPerEntry > 0 {
			_ = s.revisionRepo.DeleteOldRevisions(ctx, entry.ID, s.config.MaxRevisionsPerEntry)
		}
	}

	// Update entry
	entry.Data = mergedData
	entry.UpdatedBy = userID
	entry.Version++
	entry.UpdatedAt = time.Now()

	// Handle status change
	if req.Status != "" && req.Status != entry.Status {
		if _, valid := core.ParseEntryStatus(req.Status); valid {
			entry.Status = req.Status
			if req.Status == "published" {
				now := time.Now()
				entry.PublishedAt = &now
				entry.ScheduledAt = nil
			} else if req.Status == "scheduled" && req.ScheduledAt != nil {
				entry.ScheduledAt = req.ScheduledAt
			}
		}
	}

	if err := s.repo.Update(ctx, entry); err != nil {
		return nil, err
	}

	return s.toDTO(entry, contentType), nil
}

// Delete deletes a content entry
func (s *ContentEntryService) Delete(ctx context.Context, id xid.ID) error {
	// Verify entry exists
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}

// =============================================================================
// Status Operations
// =============================================================================

// Publish publishes a content entry
func (s *ContentEntryService) Publish(ctx context.Context, id xid.ID, req *core.PublishEntryRequest) (*core.ContentEntryDTO, error) {
	entry, err := s.repo.FindByIDWithType(ctx, id)
	if err != nil {
		return nil, err
	}

	if entry.IsPublished() {
		return nil, core.ErrEntryAlreadyPublished()
	}

	userID, _ := contexts.GetUserID(ctx)

	// Create revision if enabled
	if s.config.EnableRevisions && entry.ContentType.HasRevisions() && s.revisionRepo != nil {
		revision := schema.CreateRevisionFromEntry(entry, "Published", userID)
		_ = s.revisionRepo.Create(ctx, revision)
	}

	// Handle scheduled publish
	if req != nil && req.ScheduledAt != nil && req.ScheduledAt.After(time.Now()) {
		entry.Schedule(*req.ScheduledAt)
	} else {
		entry.Publish()
	}

	entry.UpdatedBy = userID
	entry.Version++

	if err := s.repo.Update(ctx, entry); err != nil {
		return nil, err
	}

	return s.toDTO(entry, entry.ContentType), nil
}

// Unpublish unpublishes a content entry
func (s *ContentEntryService) Unpublish(ctx context.Context, id xid.ID) (*core.ContentEntryDTO, error) {
	entry, err := s.repo.FindByIDWithType(ctx, id)
	if err != nil {
		return nil, err
	}

	if !entry.IsPublished() {
		return nil, core.ErrEntryNotPublished()
	}

	userID, _ := contexts.GetUserID(ctx)

	// Create revision if enabled
	if s.config.EnableRevisions && entry.ContentType.HasRevisions() && s.revisionRepo != nil {
		revision := schema.CreateRevisionFromEntry(entry, "Unpublished", userID)
		_ = s.revisionRepo.Create(ctx, revision)
	}

	entry.Unpublish()
	entry.UpdatedBy = userID
	entry.Version++

	if err := s.repo.Update(ctx, entry); err != nil {
		return nil, err
	}

	return s.toDTO(entry, entry.ContentType), nil
}

// Archive archives a content entry
func (s *ContentEntryService) Archive(ctx context.Context, id xid.ID) (*core.ContentEntryDTO, error) {
	entry, err := s.repo.FindByIDWithType(ctx, id)
	if err != nil {
		return nil, err
	}

	userID, _ := contexts.GetUserID(ctx)

	entry.Archive()
	entry.UpdatedBy = userID
	entry.Version++

	if err := s.repo.Update(ctx, entry); err != nil {
		return nil, err
	}

	return s.toDTO(entry, entry.ContentType), nil
}

// =============================================================================
// Bulk Operations
// =============================================================================

// BulkPublish publishes multiple entries
func (s *ContentEntryService) BulkPublish(ctx context.Context, ids []xid.ID) error {
	return s.repo.BulkUpdateStatus(ctx, ids, "published")
}

// BulkUnpublish unpublishes multiple entries
func (s *ContentEntryService) BulkUnpublish(ctx context.Context, ids []xid.ID) error {
	return s.repo.BulkUpdateStatus(ctx, ids, "draft")
}

// BulkDelete deletes multiple entries
func (s *ContentEntryService) BulkDelete(ctx context.Context, ids []xid.ID) error {
	return s.repo.BulkDelete(ctx, ids)
}

// =============================================================================
// Scheduled Publishing
// =============================================================================

// ProcessScheduledEntries processes entries scheduled for publishing
func (s *ContentEntryService) ProcessScheduledEntries(ctx context.Context) (int, error) {
	entries, err := s.repo.FindScheduledForPublish(ctx, time.Now())
	if err != nil {
		return 0, err
	}

	published := 0
	for _, entry := range entries {
		entry.Publish()
		entry.Version++
		if err := s.repo.Update(ctx, entry); err != nil {
			if s.logger != nil {
				s.logger.Error("failed to publish scheduled entry",
					forge.F("entryId", entry.ID.String()),
					forge.F("error", err.Error()))
			}
			continue
		}
		published++
	}

	return published, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// toDTO converts a content entry to its DTO representation
func (s *ContentEntryService) toDTO(entry *schema.ContentEntry, contentType *schema.ContentType) *core.ContentEntryDTO {
	if entry == nil {
		return nil
	}

	dto := &core.ContentEntryDTO{
		ID:            entry.ID.String(),
		ContentTypeID: entry.ContentTypeID.String(),
		AppID:         entry.AppID.String(),
		EnvironmentID: entry.EnvironmentID.String(),
		Data:          entry.Data,
		Status:        entry.Status,
		Version:       entry.Version,
		PublishedAt:   entry.PublishedAt,
		ScheduledAt:   entry.ScheduledAt,
		CreatedAt:     entry.CreatedAt,
		UpdatedAt:     entry.UpdatedAt,
	}

	if !entry.CreatedBy.IsNil() {
		dto.CreatedBy = entry.CreatedBy.String()
	}
	if !entry.UpdatedBy.IsNil() {
		dto.UpdatedBy = entry.UpdatedBy.String()
	}

	// Include content type summary if available
	if contentType != nil {
		dto.ContentType = &core.ContentTypeSummaryDTO{
			ID:          contentType.ID.String(),
			Name:        contentType.Name,
			Slug:        contentType.Slug,
			Description: contentType.Description,
			Icon:        contentType.Icon,
		}
	}

	return dto
}

// =============================================================================
// Revision Operations
// =============================================================================

// Restore restores an entry to a specific revision version
func (s *ContentEntryService) Restore(ctx context.Context, id xid.ID, version int) (*core.ContentEntryDTO, error) {
	// Get the current entry
	entry, err := s.repo.FindByIDWithType(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get the target revision
	if s.revisionRepo == nil {
		return nil, core.ErrRevisionsNotEnabled()
	}

	revision, err := s.revisionRepo.FindByVersion(ctx, id, version)
	if err != nil {
		return nil, err
	}

	userID, _ := contexts.GetUserID(ctx)

	// Create a revision of current state before restoring
	if s.config.EnableRevisions && entry.ContentType.HasRevisions() {
		currentRevision := schema.CreateRevisionFromEntry(entry, fmt.Sprintf("Before restore to version %d", version), userID)
		_ = s.revisionRepo.Create(ctx, currentRevision)
	}

	// Restore the data from revision
	entry.Data = revision.Data
	entry.Version++
	entry.UpdatedBy = userID
	entry.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, entry); err != nil {
		return nil, err
	}

	// Create a revision for the restore action
	if s.config.EnableRevisions && entry.ContentType.HasRevisions() {
		restoreRevision := schema.CreateRevisionFromEntry(entry, fmt.Sprintf("Restored from version %d", version), userID)
		_ = s.revisionRepo.Create(ctx, restoreRevision)
	}

	return s.toDTO(entry, entry.ContentType), nil
}

// =============================================================================
// Stats Operations
// =============================================================================

// GetStats returns statistics for entries
func (s *ContentEntryService) GetStats(ctx context.Context, contentTypeID xid.ID) (*core.ContentTypeStatsDTO, error) {
	total, err := s.repo.Count(ctx, contentTypeID)
	if err != nil {
		return nil, err
	}

	byStatus, err := s.repo.CountByStatus(ctx, contentTypeID)
	if err != nil {
		return nil, err
	}

	return &core.ContentTypeStatsDTO{
		ContentTypeID:    contentTypeID.String(),
		TotalEntries:     total,
		DraftEntries:     byStatus["draft"],
		PublishedEntries: byStatus["published"],
		ArchivedEntries:  byStatus["archived"],
		EntriesByStatus:  byStatus,
	}, nil
}

