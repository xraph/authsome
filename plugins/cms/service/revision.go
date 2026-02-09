package service

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/repository"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// RevisionService handles revision operations.
type RevisionService struct {
	repo   repository.RevisionRepository
	logger forge.Logger
}

// NewRevisionService creates a new revision service.
func NewRevisionService(repo repository.RevisionRepository, logger forge.Logger) *RevisionService {
	return &RevisionService{
		repo:   repo,
		logger: logger,
	}
}

// List returns revisions for an entry.
func (s *RevisionService) List(ctx context.Context, entryID xid.ID, query *core.ListRevisionsQuery) (*core.PaginatedResponse[*core.RevisionDTO], error) {
	if query == nil {
		query = &core.ListRevisionsQuery{
			Page:     1,
			PageSize: 20,
		}
	}

	if query.Page < 1 {
		query.Page = 1
	}

	if query.PageSize < 1 {
		query.PageSize = 20
	}

	revisions, total, err := s.repo.List(ctx, entryID, query.Page, query.PageSize)
	if err != nil {
		return nil, core.ErrInternalError("failed to list revisions", err)
	}

	dtos := make([]*core.RevisionDTO, len(revisions))
	for i, rev := range revisions {
		dtos[i] = s.toDTO(rev)
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize
	if totalPages == 0 && total > 0 {
		totalPages = 1
	}

	return &core.PaginatedResponse[*core.RevisionDTO]{
		Items:      dtos,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByVersion returns a specific revision by version.
func (s *RevisionService) GetByVersion(ctx context.Context, entryID xid.ID, version int) (*core.RevisionDTO, error) {
	revision, err := s.repo.FindByVersion(ctx, entryID, version)
	if err != nil {
		return nil, err
	}

	return s.toDTO(revision), nil
}

// Compare compares two revisions and returns the differences.
func (s *RevisionService) Compare(ctx context.Context, entryID xid.ID, fromVersion, toVersion int) (*core.RevisionCompareDTO, error) {
	fromRev, err := s.repo.FindByVersion(ctx, entryID, fromVersion)
	if err != nil {
		return nil, err
	}

	toRev, err := s.repo.FindByVersion(ctx, entryID, toVersion)
	if err != nil {
		return nil, err
	}

	// Calculate differences using schema's built-in compare
	diffs := toRev.CompareData(fromRev)

	// Convert to core.FieldDifference
	var differences []core.FieldDifference
	for _, diff := range diffs {
		differences = append(differences, core.FieldDifference{
			Field:    diff.Field,
			OldValue: diff.OldValue,
			NewValue: diff.NewValue,
			Type:     core.DiffType(diff.Type),
		})
	}

	return &core.RevisionCompareDTO{
		From:        s.toDTO(fromRev),
		To:          s.toDTO(toRev),
		Differences: differences,
	}, nil
}

// Create creates a new revision.
func (s *RevisionService) Create(ctx context.Context, entryID xid.ID, data map[string]any, changedBy, reason string) (*core.RevisionDTO, error) {
	// Get latest version
	latestVersion, err := s.repo.GetLatestVersion(ctx, entryID)
	if err != nil {
		return nil, core.ErrInternalError("failed to get latest revision", err)
	}

	version := latestVersion + 1

	revision := &schema.ContentRevision{
		ID:           xid.New(),
		EntryID:      entryID,
		Version:      version,
		Data:         data,
		ChangeReason: reason,
	}

	// Parse changedBy if it's a valid xid
	if changedByID, err := xid.FromString(changedBy); err == nil {
		revision.ChangedBy = changedByID
	}

	if err := s.repo.Create(ctx, revision); err != nil {
		return nil, core.ErrInternalError("failed to create revision", err)
	}

	return s.toDTO(revision), nil
}

// CleanupOld removes old revisions exceeding the max count.
func (s *RevisionService) CleanupOld(ctx context.Context, entryID xid.ID, maxRevisions int) error {
	if maxRevisions < 1 {
		maxRevisions = 50 // Default
	}

	return s.repo.DeleteOldRevisions(ctx, entryID, maxRevisions)
}

// toDTO converts a revision model to DTO.
func (s *RevisionService) toDTO(rev *schema.ContentRevision) *core.RevisionDTO {
	dto := &core.RevisionDTO{
		ID:        rev.ID.String(),
		EntryID:   rev.EntryID.String(),
		Version:   rev.Version,
		Data:      rev.Data,
		Reason:    rev.ChangeReason,
		CreatedAt: rev.CreatedAt,
	}
	if !rev.ChangedBy.IsNil() {
		dto.ChangedBy = rev.ChangedBy.String()
	}

	return dto
}
