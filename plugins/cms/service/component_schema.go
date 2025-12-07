// Package service implements the business logic layer for the CMS plugin.
package service

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/repository"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// ComponentSchemaService handles component schema business logic
type ComponentSchemaService struct {
	repo          repository.ComponentSchemaRepository
	maxComponents int
	logger        forge.Logger
}

// ComponentSchemaServiceConfig holds configuration for the service
type ComponentSchemaServiceConfig struct {
	MaxComponentSchemas int
	Logger              forge.Logger
}

// NewComponentSchemaService creates a new component schema service
func NewComponentSchemaService(
	repo repository.ComponentSchemaRepository,
	config ComponentSchemaServiceConfig,
) *ComponentSchemaService {
	maxComponents := config.MaxComponentSchemas
	if maxComponents <= 0 {
		maxComponents = 100 // Default
	}
	return &ComponentSchemaService{
		repo:          repo,
		maxComponents: maxComponents,
		logger:        config.Logger,
	}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new component schema
func (s *ComponentSchemaService) Create(ctx context.Context, req *core.CreateComponentSchemaRequest) (*core.ComponentSchemaDTO, error) {
	// Get app/env context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, core.ErrEnvContextMissing()
	}

	// Validate slug format
	slug := req.Name
	if slug == "" {
		slug = generateSlug(req.Name)
	}
	if !slugPattern.MatchString(slug) {
		return nil, core.ErrInvalidComponentSchemaSlug(slug, "must start with a letter and contain only letters, numbers, underscores, and hyphens")
	}

	// Check for duplicates
	exists, err := s.repo.ExistsWithName(ctx, appID, envID, slug)
	if err != nil {
		return nil, core.ErrDatabaseError("failed to check slug existence", err)
	}
	if exists {
		return nil, core.ErrComponentSchemaExists(slug)
	}

	// Validate nested fields
	if err := s.validateNestedFields(req.Fields, nil); err != nil {
		return nil, err
	}

	// Get user ID
	userID, _ := contexts.GetUserID(ctx)

	// Create component schema
	component := &schema.ComponentSchema{
		ID:            xid.New(),
		AppID:         appID,
		EnvironmentID: envID,
		Title:         req.Title,
		Name:          slug,
		Description:   req.Description,
		Icon:          req.Icon,
		Fields:        s.dtoToSchemaFields(req.Fields),
		CreatedBy:     userID,
		UpdatedBy:     userID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, component); err != nil {
		return nil, core.ErrDatabaseError("failed to create component schema", err)
	}

	return s.toDTO(component, 0), nil
}

// GetByID retrieves a component schema by ID
func (s *ComponentSchemaService) GetByID(ctx context.Context, id xid.ID) (*core.ComponentSchemaDTO, error) {
	component, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get app/env context for usage count
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)

	usageCount := 0
	if !appID.IsNil() && !envID.IsNil() {
		usageCount, _ = s.repo.CountUsages(ctx, appID, envID, component.Name)
	}

	return s.toDTO(component, usageCount), nil
}

// GetBySlug retrieves a component schema by slug
func (s *ComponentSchemaService) GetByName(ctx context.Context, name string) (*core.ComponentSchemaDTO, error) {
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, core.ErrEnvContextMissing()
	}

	component, err := s.repo.FindByName(ctx, appID, envID, name)
	if err != nil {
		return nil, err
	}

	usageCount, _ := s.repo.CountUsages(ctx, appID, envID, name)

	return s.toDTO(component, usageCount), nil
}

// List lists component schemas with pagination
func (s *ComponentSchemaService) List(ctx context.Context, query *core.ListComponentSchemasQuery) (*core.ListComponentSchemasResponse, error) {
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, core.ErrEnvContextMissing()
	}

	components, total, err := s.repo.List(ctx, appID, envID, query)
	if err != nil {
		return nil, core.ErrDatabaseError("failed to list component schemas", err)
	}

	// Calculate pagination
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPages := (total + pageSize - 1) / pageSize

	// Convert to DTOs
	dtos := make([]*core.ComponentSchemaSummaryDTO, len(components))
	for i, component := range components {
		usageCount, _ := s.repo.CountUsages(ctx, appID, envID, component.Name)
		dtos[i] = s.toSummaryDTO(component, usageCount)
	}

	return &core.ListComponentSchemasResponse{
		Components: dtos,
		Page:       query.Page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

// Update updates a component schema
func (s *ComponentSchemaService) Update(ctx context.Context, id xid.ID, req *core.UpdateComponentSchemaRequest) (*core.ComponentSchemaDTO, error) {
	// Get existing component
	component, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate nested fields if provided
	if req.Fields != nil {
		if err := s.validateNestedFields(req.Fields, nil); err != nil {
			return nil, err
		}
		component.Fields = s.dtoToSchemaFields(req.Fields)
	}

	// Update fields
	if req.Title != "" {
		component.Title = req.Title
	}
	if req.Description != "" {
		component.Description = req.Description
	}
	if req.Icon != "" {
		component.Icon = req.Icon
	}

	// Update metadata
	userID, _ := contexts.GetUserID(ctx)
	component.UpdatedBy = userID
	component.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, component); err != nil {
		return nil, core.ErrDatabaseError("failed to update component schema", err)
	}

	// Get usage count
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	usageCount, _ := s.repo.CountUsages(ctx, appID, envID, component.Name)

	return s.toDTO(component, usageCount), nil
}

// Delete deletes a component schema
func (s *ComponentSchemaService) Delete(ctx context.Context, id xid.ID) error {
	// Get existing component
	component, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if in use
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	usageCount, err := s.repo.CountUsages(ctx, appID, envID, component.Name)
	if err != nil {
		return core.ErrDatabaseError("failed to check component usage", err)
	}
	if usageCount > 0 {
		return core.ErrComponentSchemaInUse(component.Name, usageCount)
	}

	return s.repo.Delete(ctx, id)
}

// =============================================================================
// Validation
// =============================================================================

// validateNestedFields validates nested field definitions
func (s *ComponentSchemaService) validateNestedFields(fields []core.NestedFieldDefDTO, seenSlugs map[string]bool) error {
	if seenSlugs == nil {
		seenSlugs = make(map[string]bool)
	}

	for _, field := range fields {
		// Validate required fields
		if field.Name == "" {
			return core.ErrFieldRequired("name")
		}
		if field.Name == "" {
			return core.ErrFieldRequired("slug")
		}
		if field.Type == "" {
			return core.ErrFieldRequired("type")
		}

		// Validate slug format
		if !slugPattern.MatchString(field.Name) {
			return core.ErrInvalidFieldSlug(field.Name, "must start with a letter and contain only letters, numbers, underscores, and hyphens")
		}

		// Check for duplicate slugs
		if seenSlugs[field.Name] {
			return core.ErrFieldExists(field.Name)
		}
		seenSlugs[field.Name] = true

		// Validate field type
		fieldType := core.FieldType(field.Type)
		if !fieldType.IsValid() {
			return core.ErrInvalidFieldType(field.Type)
		}

		// Recursively validate nested fields
		if field.Options != nil && len(field.Options.NestedFields) > 0 {
			if !fieldType.IsNested() {
				return core.ErrInvalidRequest("nestedFields can only be defined for object or array types")
			}
			// Create a new map for nested scope
			nestedSlugs := make(map[string]bool)
			if err := s.validateNestedFields(field.Options.NestedFields, nestedSlugs); err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateComponentRef validates that a component reference is valid and not circular
func (s *ComponentSchemaService) ValidateComponentRef(ctx context.Context, componentSlug string, visited []string) error {
	// Check for circular reference
	for _, v := range visited {
		if v == componentSlug {
			return core.ErrCircularComponentRef(buildRefPath(visited, componentSlug))
		}
	}

	// Get app/env context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return core.ErrAppContextMissing()
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return core.ErrEnvContextMissing()
	}

	// Check if component exists
	component, err := s.repo.FindByName(ctx, appID, envID, componentSlug)
	if err != nil {
		return err
	}

	// Check nested component refs recursively
	visited = append(visited, componentSlug)
	for _, field := range component.Fields {
		if field.Options != nil && field.Options.ComponentRef != "" {
			if err := s.ValidateComponentRef(ctx, field.Options.ComponentRef, visited); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildRefPath builds a string representation of the reference path
func buildRefPath(visited []string, current string) string {
	path := ""
	for _, v := range visited {
		if path != "" {
			path += " -> "
		}
		path += v
	}
	if path != "" {
		path += " -> "
	}
	path += current
	return path
}

// =============================================================================
// Conversion helpers
// =============================================================================

// toDTO converts a schema component to a DTO
func (s *ComponentSchemaService) toDTO(component *schema.ComponentSchema, usageCount int) *core.ComponentSchemaDTO {
	return &core.ComponentSchemaDTO{
	ID:            component.ID.String(),
	AppID:         component.AppID.String(),
	EnvironmentID: component.EnvironmentID.String(),
	Title:         component.Title,
	Name:          component.Name,
	Description:   component.Description,
	Icon:          component.Icon,
	Fields:        s.schemaToFieldDTOs(component.Fields),
	UsageCount:    usageCount,
	CreatedBy:     component.CreatedBy.String(),
	UpdatedBy:     component.UpdatedBy.String(),
	CreatedAt:     component.CreatedAt,
	UpdatedAt:     component.UpdatedAt,
}
}

// toSummaryDTO converts a schema component to a summary DTO
func (s *ComponentSchemaService) toSummaryDTO(component *schema.ComponentSchema, usageCount int) *core.ComponentSchemaSummaryDTO {
	return &core.ComponentSchemaSummaryDTO{
		ID:          component.ID.String(),
		Title:       component.Title,
		Name:        component.Name,
		Description: component.Description,
		Icon:        component.Icon,
		FieldCount:  len(component.Fields),
		UsageCount:  usageCount,
		CreatedAt:   component.CreatedAt,
		UpdatedAt:   component.UpdatedAt,
	}
}

// schemaToFieldDTOs converts schema nested fields to DTOs
func (s *ComponentSchemaService) schemaToFieldDTOs(fields schema.NestedFieldDefs) []core.NestedFieldDefDTO {
	if fields == nil {
		return []core.NestedFieldDefDTO{}
	}

	dtos := make([]core.NestedFieldDefDTO, len(fields))
	for i, field := range fields {
		dtos[i] = core.NestedFieldDefDTO{
			Title:       field.Title,
			Name:        field.Name,
			Type:        field.Type,
			Required:    field.Required,
			Description: field.Description,
		}
		if field.Options != nil {
			dtos[i].Options = s.schemaOptionsToDTO(field.Options)
		}
	}
	return dtos
}

// schemaOptionsToDTO converts schema field options to DTO
func (s *ComponentSchemaService) schemaOptionsToDTO(opts *schema.FieldOptions) *core.FieldOptionsDTO {
	if opts == nil {
		return nil
	}

	dto := &core.FieldOptionsDTO{
		MinLength:        opts.MinLength,
		MaxLength:        opts.MaxLength,
		Pattern:          opts.Pattern,
		Min:              opts.Min,
		Max:              opts.Max,
		Step:             opts.Step,
		Integer:          opts.Integer,
		RelatedType:      opts.RelatedType,
		RelationType:     opts.RelationType,
		OnDelete:         opts.OnDelete,
		InverseField:     opts.InverseField,
		AllowHTML:        opts.AllowHTML,
		MaxWords:         opts.MaxWords,
		AllowedMimeTypes: opts.AllowedMimeTypes,
		MaxFileSize:      opts.MaxFileSize,
		SourceField:      opts.SourceField,
		Schema:           opts.Schema,
		MinDate:          opts.MinDate,
		MaxDate:          opts.MaxDate,
		DateFormat:       opts.DateFormat,
		ComponentRef:     opts.ComponentRef,
		MinItems:         opts.MinItems,
		MaxItems:         opts.MaxItems,
		Collapsible:      opts.Collapsible,
		DefaultExpanded:  opts.DefaultExpanded,
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
		dto.NestedFields = s.schemaToFieldDTOs(opts.NestedFields)
	}

	return dto
}

// dtoToSchemaFields converts DTO nested fields to schema
func (s *ComponentSchemaService) dtoToSchemaFields(fields []core.NestedFieldDefDTO) schema.NestedFieldDefs {
	if fields == nil {
		return schema.NestedFieldDefs{}
	}

	schemaFields := make(schema.NestedFieldDefs, len(fields))
	for i, field := range fields {
		schemaFields[i] = schema.NestedFieldDef{
			Title:       field.Title,
			Name:        field.Name,
			Type:        field.Type,
			Required:    field.Required,
			Description: field.Description,
		}
		if field.Options != nil {
			schemaFields[i].Options = s.dtoToSchemaOptions(field.Options)
		}
	}
	return schemaFields
}

// dtoToSchemaOptions converts DTO field options to schema
func (s *ComponentSchemaService) dtoToSchemaOptions(opts *core.FieldOptionsDTO) *schema.FieldOptions {
	if opts == nil {
		return nil
	}

	schemaOpts := &schema.FieldOptions{
		MinLength:        opts.MinLength,
		MaxLength:        opts.MaxLength,
		Pattern:          opts.Pattern,
		Min:              opts.Min,
		Max:              opts.Max,
		Step:             opts.Step,
		Integer:          opts.Integer,
		RelatedType:      opts.RelatedType,
		RelationType:     opts.RelationType,
		OnDelete:         opts.OnDelete,
		InverseField:     opts.InverseField,
		AllowHTML:        opts.AllowHTML,
		MaxWords:         opts.MaxWords,
		AllowedMimeTypes: opts.AllowedMimeTypes,
		MaxFileSize:      opts.MaxFileSize,
		SourceField:      opts.SourceField,
		Schema:           opts.Schema,
		MinDate:          opts.MinDate,
		MaxDate:          opts.MaxDate,
		DateFormat:       opts.DateFormat,
		ComponentRef:     opts.ComponentRef,
		MinItems:         opts.MinItems,
		MaxItems:         opts.MaxItems,
		Collapsible:      opts.Collapsible,
		DefaultExpanded:  opts.DefaultExpanded,
	}

	// Convert choices
	if len(opts.Choices) > 0 {
		schemaOpts.Choices = make([]schema.Choice, len(opts.Choices))
		for i, c := range opts.Choices {
			schemaOpts.Choices[i] = schema.Choice{
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
		schemaOpts.NestedFields = s.dtoToSchemaFields(opts.NestedFields)
	}

	return schemaOpts
}

// =============================================================================
// Resolution helpers
// =============================================================================

// ResolveComponentSchema resolves a component reference and returns the nested fields
func (s *ComponentSchemaService) ResolveComponentSchema(ctx context.Context, componentName string) ([]core.NestedFieldDefDTO, error) {
	dto, err := s.GetByName(ctx, componentName)
	if err != nil {
		return nil, err
	}
	return dto.Fields, nil
}

// GetEffectiveFields returns the effective nested fields for a field, resolving component refs
func (s *ComponentSchemaService) GetEffectiveFields(ctx context.Context, field *core.ContentFieldDTO) ([]core.NestedFieldDefDTO, error) {
	if field.Options.ComponentRef != "" {
		return s.ResolveComponentSchema(ctx, field.Options.ComponentRef)
	}
	return field.Options.NestedFields, nil
}

