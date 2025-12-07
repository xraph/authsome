// Package service implements the business logic layer for the CMS plugin.
package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/repository"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// slugPattern defines valid slug format: letters, numbers, underscores, and hyphens
// Must start with a letter (case-insensitive)
var slugPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// generateSlug creates an identifier from a name, preserving casing
func generateSlug(name string) string {
	// Trim whitespace but preserve casing
	slug := strings.TrimSpace(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove invalid characters (keep letters, numbers, underscores, hyphens)
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	// Remove multiple consecutive hyphens or underscores
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}

	// Remove leading/trailing hyphens and underscores
	slug = strings.Trim(slug, "-_")

	// Ensure it starts with a letter
	if len(slug) > 0 && slug[0] >= '0' && slug[0] <= '9' {
		slug = "type-" + slug
	}

	// Default if empty
	if slug == "" {
		slug = "content-type"
	}

	return slug
}

// ContentTypeService handles content type business logic
type ContentTypeService struct {
	repo      repository.ContentTypeRepository
	fieldRepo repository.ContentFieldRepository
	maxTypes  int
	logger    forge.Logger
}

// ContentTypeServiceConfig holds configuration for the service
type ContentTypeServiceConfig struct {
	MaxContentTypes int
	Logger          forge.Logger
}

// NewContentTypeService creates a new content type service
func NewContentTypeService(
	repo repository.ContentTypeRepository,
	fieldRepo repository.ContentFieldRepository,
	config ContentTypeServiceConfig,
) *ContentTypeService {
	maxTypes := config.MaxContentTypes
	if maxTypes <= 0 {
		maxTypes = 100 // Default
	}
	return &ContentTypeService{
		repo:      repo,
		fieldRepo: fieldRepo,
		maxTypes:  maxTypes,
		logger:    config.Logger,
	}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new content type
func (s *ContentTypeService) Create(ctx context.Context, req *core.CreateContentTypeRequest) (*core.ContentTypeDTO, error) {
	// Get app/env context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, core.ErrEnvContextMissing()
	}

	// Auto-generate slug from name if not provided
	slug := strings.TrimSpace(req.Name)
	if slug == "" {
		slug = generateSlug(req.Name)
	}

	// Validate slug
	if !isValidSlug(slug) {
		return nil, core.ErrInvalidContentTypeSlug(slug, "must start with a letter and contain only letters, numbers, underscores, and hyphens")
	}

	// Check if slug already exists
	exists, err := s.repo.ExistsWithName(ctx, appID, envID, slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, core.ErrContentTypeExists(slug)
	}

	// Check content type limit
	count, err := s.repo.Count(ctx, appID, envID)
	if err != nil {
		return nil, err
	}
	if s.maxTypes > 0 && count >= s.maxTypes {
		return nil, core.ErrInvalidRequest("content type limit reached")
	}

	// Get user ID from context
	userID, _ := contexts.GetUserID(ctx)

	// Build settings
	settings := schema.DefaultSettings()
	if req.Settings != nil {
		settings.TitleField = req.Settings.TitleField
		settings.DescriptionField = req.Settings.DescriptionField
		settings.EnableRevisions = req.Settings.EnableRevisions
		settings.EnableDrafts = req.Settings.EnableDrafts
		settings.EnableSoftDelete = req.Settings.EnableSoftDelete
		settings.EnableSearch = req.Settings.EnableSearch
		settings.EnableScheduling = req.Settings.EnableScheduling
		settings.DefaultPermissions = req.Settings.DefaultPermissions
		settings.MaxEntries = req.Settings.MaxEntries
	}

	// Create content type
	contentType := &schema.ContentType{
	ID:            xid.New(),
	AppID:         appID,
	EnvironmentID: envID,
	Title:         strings.TrimSpace(req.Title),
	Name:          slug,
	Description:   strings.TrimSpace(req.Description),
	Icon:          req.Icon,
	Settings:      settings,
	CreatedBy:     userID,
	UpdatedBy:     userID,
}

	if err := s.repo.Create(ctx, contentType); err != nil {
		return nil, err
	}

	return s.toDTO(contentType), nil
}

// GetByID retrieves a content type by ID
func (s *ContentTypeService) GetByID(ctx context.Context, id xid.ID) (*core.ContentTypeDTO, error) {
	contentType, err := s.repo.FindWithFields(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toDTO(contentType), nil
}

// GetBySlug retrieves a content type by slug
func (s *ContentTypeService) GetByName(ctx context.Context, name string) (*core.ContentTypeDTO, error) {
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, core.ErrEnvContextMissing()
	}

	contentType, err := s.repo.FindByNameWithFields(ctx, appID, envID, name)
	if err != nil {
		return nil, err
	}
	return s.toDTO(contentType), nil
}

// List lists content types with filtering and pagination
func (s *ContentTypeService) List(ctx context.Context, query *core.ListContentTypesQuery) (*core.ListContentTypesResponse, error) {
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, core.ErrEnvContextMissing()
	}

	contentTypes, total, err := s.repo.List(ctx, appID, envID, query)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	dtos := make([]*core.ContentTypeSummaryDTO, len(contentTypes))
	for i, ct := range contentTypes {
		// Get entry count for each type
		entryCount, _ := s.repo.CountEntries(ctx, ct.ID)
		fieldCount := len(ct.Fields)

		dtos[i] = &core.ContentTypeSummaryDTO{
			ID:          ct.ID.String(),
			Title: ct.Title,
			Name:        ct.Name,
			Description: ct.Description,
			Icon:        ct.Icon,
			EntryCount:  entryCount,
			FieldCount:  fieldCount,
			CreatedAt:   ct.CreatedAt,
			UpdatedAt:   ct.UpdatedAt,
		}
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

	return &core.ListContentTypesResponse{
		ContentTypes: dtos,
		Page:         page,
		PageSize:     pageSize,
		TotalItems:   total,
		TotalPages:   totalPages,
	}, nil
}

// Update updates a content type
func (s *ContentTypeService) Update(ctx context.Context, id xid.ID, req *core.UpdateContentTypeRequest) (*core.ContentTypeDTO, error) {
	contentType, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	userID, _ := contexts.GetUserID(ctx)

	// Update fields
	if req.Title != "" {
		contentType.Title = strings.TrimSpace(req.Title)
	}
	if req.Description != "" {
		contentType.Description = strings.TrimSpace(req.Description)
	}
	if req.Icon != "" {
		contentType.Icon = req.Icon
	}

	// Update settings
	if req.Settings != nil {
		if req.Settings.TitleField != "" {
			contentType.Settings.TitleField = req.Settings.TitleField
		}
		if req.Settings.DescriptionField != "" {
			contentType.Settings.DescriptionField = req.Settings.DescriptionField
		}
		contentType.Settings.EnableRevisions = req.Settings.EnableRevisions
		contentType.Settings.EnableDrafts = req.Settings.EnableDrafts
		contentType.Settings.EnableSoftDelete = req.Settings.EnableSoftDelete
		contentType.Settings.EnableSearch = req.Settings.EnableSearch
		contentType.Settings.EnableScheduling = req.Settings.EnableScheduling
		if len(req.Settings.DefaultPermissions) > 0 {
			contentType.Settings.DefaultPermissions = req.Settings.DefaultPermissions
		}
		if req.Settings.MaxEntries > 0 {
			contentType.Settings.MaxEntries = req.Settings.MaxEntries
		}
	}

	contentType.UpdatedBy = userID
	contentType.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, contentType); err != nil {
		return nil, err
	}

	// Reload with fields
	return s.GetByID(ctx, id)
}

// Delete deletes a content type
func (s *ContentTypeService) Delete(ctx context.Context, id xid.ID) error {
	// Check if content type exists
	contentType, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if there are entries
	entryCount, err := s.repo.CountEntries(ctx, id)
	if err != nil {
		return err
	}
	if entryCount > 0 {
		return core.ErrContentTypeHasEntries(contentType.Name, entryCount)
	}

	// Soft delete
	return s.repo.Delete(ctx, id)
}

// HardDelete permanently deletes a content type and all its data
func (s *ContentTypeService) HardDelete(ctx context.Context, id xid.ID) error {
	// First delete all fields
	if s.fieldRepo != nil {
		if err := s.fieldRepo.DeleteAllForContentType(ctx, id); err != nil {
			return err
		}
	}

	// Then delete the content type
	return s.repo.HardDelete(ctx, id)
}

// =============================================================================
// Helper Methods
// =============================================================================

// toDTO converts a content type to its DTO representation
func (s *ContentTypeService) toDTO(ct *schema.ContentType) *core.ContentTypeDTO {
	if ct == nil {
		return nil
	}

	dto := &core.ContentTypeDTO{
	ID:            ct.ID.String(),
	AppID:         ct.AppID.String(),
	EnvironmentID: ct.EnvironmentID.String(),
	Title:         ct.Title,
	Name:          ct.Name,
	Description:   ct.Description,
	Icon:          ct.Icon,
	Settings: core.ContentTypeSettingsDTO{
			TitleField:         ct.Settings.TitleField,
			DescriptionField:   ct.Settings.DescriptionField,
			EnableRevisions:    ct.Settings.EnableRevisions,
			EnableDrafts:       ct.Settings.EnableDrafts,
			EnableSoftDelete:   ct.Settings.EnableSoftDelete,
			EnableSearch:       ct.Settings.EnableSearch,
			EnableScheduling:   ct.Settings.EnableScheduling,
			DefaultPermissions: ct.Settings.DefaultPermissions,
			MaxEntries:         ct.Settings.MaxEntries,
		},
		CreatedAt: ct.CreatedAt,
		UpdatedAt: ct.UpdatedAt,
	}

	if !ct.CreatedBy.IsNil() {
		dto.CreatedBy = ct.CreatedBy.String()
	}
	if !ct.UpdatedBy.IsNil() {
		dto.UpdatedBy = ct.UpdatedBy.String()
	}

	// Convert fields if loaded
	if len(ct.Fields) > 0 {
		dto.Fields = make([]*core.ContentFieldDTO, len(ct.Fields))
		for i, field := range ct.Fields {
			dto.Fields[i] = fieldToDTO(field)
		}
	}

	return dto
}

// fieldToDTO converts a content field to its DTO representation
func fieldToDTO(field *schema.ContentField) *core.ContentFieldDTO {
	if field == nil {
		return nil
	}

	dto := &core.ContentFieldDTO{
		ID:            field.ID.String(),
		ContentTypeID: field.ContentTypeID.String(),
		Title: field.Title,
		Name:          field.Name,
		Description:   field.Description,
		Type:          field.Type,
		Required:      field.Required,
		Unique:        field.Unique,
		Indexed:       field.Indexed,
		Localized:     field.Localized,
		Order:         field.Order,
		Hidden:        field.Hidden,
		ReadOnly:      field.ReadOnly,
		CreatedAt:     field.CreatedAt,
		UpdatedAt:     field.UpdatedAt,
	}

	// Parse default value
	if defaultVal, err := field.GetDefaultValue(); err == nil && defaultVal != nil {
		dto.DefaultValue = defaultVal
	}

	// Convert options
	dto.Options = core.FieldOptionsDTO{
		MinLength:                  field.Options.MinLength,
		MaxLength:                  field.Options.MaxLength,
		Pattern:                    field.Options.Pattern,
		Min:                        field.Options.Min,
		Max:                        field.Options.Max,
		Step:                       field.Options.Step,
		Integer:                    field.Options.Integer,
		RelatedType:                field.Options.RelatedType,
		RelationType:               field.Options.RelationType,
		OnDelete:                   field.Options.OnDelete,
		InverseField:               field.Options.InverseField,
		AllowHTML:                  field.Options.AllowHTML,
		MaxWords:                   field.Options.MaxWords,
		AllowedMimeTypes:           field.Options.AllowedMimeTypes,
		MaxFileSize:                field.Options.MaxFileSize,
		SourceField:                field.Options.SourceField,
		Schema:                     field.Options.Schema,
		MinDate:                    field.Options.MinDate,
		MaxDate:                    field.Options.MaxDate,
		DateFormat:                 field.Options.DateFormat,
		ComponentRef:               field.Options.ComponentRef,
		MinItems:                   field.Options.MinItems,
		MaxItems:                   field.Options.MaxItems,
		Collapsible:                field.Options.Collapsible,
		DefaultExpanded:            field.Options.DefaultExpanded,
		DiscriminatorField:         field.Options.DiscriminatorField,
		ClearOnDiscriminatorChange: field.Options.ClearOnDiscriminatorChange,
		ClearWhenHidden:            field.Options.ClearWhenHidden,
	}

	// Convert choices
	if len(field.Options.Choices) > 0 {
		dto.Options.Choices = make([]core.ChoiceDTO, len(field.Options.Choices))
		for i, choice := range field.Options.Choices {
			dto.Options.Choices[i] = core.ChoiceDTO{
				Value:    choice.Value,
				Label:    choice.Label,
				Icon:     choice.Icon,
				Color:    choice.Color,
				Disabled: choice.Disabled,
			}
		}
	}

	// Convert nested fields
	if len(field.Options.NestedFields) > 0 {
		dto.Options.NestedFields = schemaToNestedFieldDTOs(field.Options.NestedFields)
	}

	// Convert oneOf schemas
	if len(field.Options.Schemas) > 0 {
		dto.Options.Schemas = make(map[string]core.OneOfSchemaOptionDTO, len(field.Options.Schemas))
		for key, schemaOpt := range field.Options.Schemas {
			dto.Options.Schemas[key] = schemaOneOfOptionToDTO(schemaOpt)
		}
	}

	// Convert conditional visibility
	if field.Options.ShowWhen != nil {
		dto.Options.ShowWhen = schemaConditionToDTO(field.Options.ShowWhen)
	}
	if field.Options.HideWhen != nil {
		dto.Options.HideWhen = schemaConditionToDTO(field.Options.HideWhen)
	}

	return dto
}

// schemaToNestedFieldDTOs converts schema nested fields to DTOs
func schemaToNestedFieldDTOs(fields schema.NestedFieldDefs) []core.NestedFieldDefDTO {
	if fields == nil {
		return nil
	}

	dtos := make([]core.NestedFieldDefDTO, len(fields))
	for i, f := range fields {
		dtos[i] = core.NestedFieldDefDTO{
			Title:       f.Title,
			Name:        f.Name,
			Type:        f.Type,
			Required:    f.Required,
			Description: f.Description,
		}
		if f.Options != nil {
			dtos[i].Options = schemaOptionsToDTO(f.Options)
		}
	}
	return dtos
}

// schemaOptionsToDTO converts schema field options to DTO
func schemaOptionsToDTO(opts *schema.FieldOptions) *core.FieldOptionsDTO {
	if opts == nil {
		return nil
	}

	dto := &core.FieldOptionsDTO{
		MinLength:                  opts.MinLength,
		MaxLength:                  opts.MaxLength,
		Pattern:                    opts.Pattern,
		Min:                        opts.Min,
		Max:                        opts.Max,
		Step:                       opts.Step,
		Integer:                    opts.Integer,
		RelatedType:                opts.RelatedType,
		RelationType:               opts.RelationType,
		OnDelete:                   opts.OnDelete,
		InverseField:               opts.InverseField,
		AllowHTML:                  opts.AllowHTML,
		MaxWords:                   opts.MaxWords,
		AllowedMimeTypes:           opts.AllowedMimeTypes,
		MaxFileSize:                opts.MaxFileSize,
		SourceField:                opts.SourceField,
		Schema:                     opts.Schema,
		MinDate:                    opts.MinDate,
		MaxDate:                    opts.MaxDate,
		DateFormat:                 opts.DateFormat,
		ComponentRef:               opts.ComponentRef,
		MinItems:                   opts.MinItems,
		MaxItems:                   opts.MaxItems,
		Collapsible:                opts.Collapsible,
		DefaultExpanded:            opts.DefaultExpanded,
		DiscriminatorField:         opts.DiscriminatorField,
		ClearOnDiscriminatorChange: opts.ClearOnDiscriminatorChange,
		ClearWhenHidden:            opts.ClearWhenHidden,
	}

	// Convert choices
	if len(opts.Choices) > 0 {
		dto.Choices = make([]core.ChoiceDTO, len(opts.Choices))
		for i, c := range opts.Choices {
			dto.Choices[i] = core.ChoiceDTO{
				Value:    c.Value,
				Label:    c.Label,
				Icon:     c.Icon,
				Color:    c.Color,
				Disabled: c.Disabled,
			}
		}
	}

	// Recursively convert nested fields
	if len(opts.NestedFields) > 0 {
		dto.NestedFields = schemaToNestedFieldDTOs(opts.NestedFields)
	}

	// Convert oneOf schemas
	if len(opts.Schemas) > 0 {
		dto.Schemas = make(map[string]core.OneOfSchemaOptionDTO, len(opts.Schemas))
		for key, schemaOpt := range opts.Schemas {
			dto.Schemas[key] = schemaOneOfOptionToDTO(schemaOpt)
		}
	}

	// Convert conditional visibility
	if opts.ShowWhen != nil {
		dto.ShowWhen = schemaConditionToDTO(opts.ShowWhen)
	}
	if opts.HideWhen != nil {
		dto.HideWhen = schemaConditionToDTO(opts.HideWhen)
	}

	return dto
}

// schemaOneOfOptionToDTO converts a schema OneOfSchemaOption to DTO
func schemaOneOfOptionToDTO(opt schema.OneOfSchemaOption) core.OneOfSchemaOptionDTO {
	dto := core.OneOfSchemaOptionDTO{
		ComponentRef: opt.ComponentRef,
		Label:        opt.Label,
	}
	if len(opt.NestedFields) > 0 {
		dto.NestedFields = schemaToNestedFieldDTOs(opt.NestedFields)
	}
	return dto
}

// schemaConditionToDTO converts a schema FieldCondition to DTO
func schemaConditionToDTO(cond *schema.FieldCondition) *core.FieldConditionDTO {
	if cond == nil {
		return nil
	}
	return &core.FieldConditionDTO{
		Field:    cond.Field,
		Operator: cond.Operator,
		Value:    cond.Value,
	}
}

// isValidSlug validates a slug format
func isValidSlug(slug string) bool {
	if len(slug) < 1 || len(slug) > 63 {
		return false
	}
	return slugPattern.MatchString(slug)
}

// =============================================================================
// Stats Operations
// =============================================================================

// GetStats returns statistics for content types
func (s *ContentTypeService) GetStats(ctx context.Context) (*core.CMSStatsDTO, error) {
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, core.ErrEnvContextMissing()
	}

	totalTypes, err := s.repo.Count(ctx, appID, envID)
	if err != nil {
		return nil, err
	}

	return &core.CMSStatsDTO{
		TotalContentTypes: totalTypes,
		TotalEntries:      0, // Will be populated when entry service is available
		TotalRevisions:    0,
		EntriesByStatus:   make(map[string]int),
		EntriesByType:     make(map[string]int),
	}, nil
}
