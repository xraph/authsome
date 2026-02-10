package service

import (
	"context"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/repository"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// ContentFieldService handles content field business logic.
type ContentFieldService struct {
	repo                   repository.ContentFieldRepository
	contentTypeRepo        repository.ContentTypeRepository
	componentSchemaRepo    repository.ComponentSchemaRepository
	componentSchemaService *ComponentSchemaService
	maxFields              int
	logger                 forge.Logger
}

// ContentFieldServiceConfig holds configuration for the service.
type ContentFieldServiceConfig struct {
	MaxFieldsPerType int
	Logger           forge.Logger
}

// NewContentFieldService creates a new content field service.
func NewContentFieldService(
	repo repository.ContentFieldRepository,
	contentTypeRepo repository.ContentTypeRepository,
	config ContentFieldServiceConfig,
) *ContentFieldService {
	maxFields := config.MaxFieldsPerType
	if maxFields <= 0 {
		maxFields = 50 // Default
	}

	return &ContentFieldService{
		repo:            repo,
		contentTypeRepo: contentTypeRepo,
		maxFields:       maxFields,
		logger:          config.Logger,
	}
}

// SetComponentSchemaService sets the component schema service for resolving component refs.
func (s *ContentFieldService) SetComponentSchemaService(svc *ComponentSchemaService) {
	s.componentSchemaService = svc
}

// SetComponentSchemaRepository sets the component schema repository.
func (s *ContentFieldService) SetComponentSchemaRepository(repo repository.ComponentSchemaRepository) {
	s.componentSchemaRepo = repo
}

// =============================================================================
// Helper Functions
// =============================================================================

// generateFieldSlug creates a slug from a field name.
func generateFieldSlug(name string) string {
	// Trim whitespace but preserve casing
	slug := strings.TrimSpace(name)

	// slug spaces with underscores (or can be converted to camelCase)
	slug = strings.ReplaceAll(slug, " ", "_")

	// result invalid characters (keep letters, numbers, underscores, hyphens)
	var result strings.Builder

	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}

	slug = result.String()

	// Remove multiple consecutive underscores or hyphens
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}

	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// slug leading/trailing underscores and hyphens
	slug = strings.Trim(slug, "_-")

	// Ensure it starts with a letter
	if len(slug) > 0 && slug[0] >= '0' && slug[0] <= '9' {
		slug = "field_" + slug
	}

	// Default if empty
	if slug == "" {
		slug = "field"
	}

	return slug
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new content field.
func (s *ContentFieldService) Create(ctx context.Context, contentTypeID xid.ID, req *core.CreateFieldRequest) (*core.ContentFieldDTO, error) {
	// Verify content type exists
	_, err := s.contentTypeRepo.FindByID(ctx, contentTypeID)
	if err != nil {
		return nil, err
	}

	// Auto-generate slug from name if not provided
	slug := strings.TrimSpace(req.Name)
	if slug == "" {
		slug = generateFieldSlug(req.Name)
	}

	// Validate slug
	if !isValidSlug(slug) {
		return nil, core.ErrInvalidFieldSlug(slug, "must start with a letter and contain only letters, numbers, underscores, and hyphens")
	}

	// Check if slug already exists
	exists, err := s.repo.ExistsWithName(ctx, contentTypeID, slug)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, core.ErrFieldExists(slug)
	}

	// Check field limit
	count, err := s.repo.Count(ctx, contentTypeID)
	if err != nil {
		return nil, err
	}

	if s.maxFields > 0 && count >= s.maxFields {
		return nil, core.ErrInvalidRequest("field limit reached for content type")
	}

	// Validate field type
	fieldType, valid := core.ParseFieldType(req.Type)
	if !valid {
		return nil, core.ErrInvalidFieldType(req.Type)
	}

	// Validate options for field type
	if err := s.validateFieldOptions(fieldType, req.Options); err != nil {
		return nil, err
	}

	// Build field options
	options := s.buildFieldOptions(req.Options)

	// Create field
	field := &schema.ContentField{
		ID:            xid.New(),
		ContentTypeID: contentTypeID,
		Title:         strings.TrimSpace(req.Title),
		Name:          slug,
		Description:   strings.TrimSpace(req.Description),
		Type:          string(fieldType),
		Required:      req.Required,
		Unique:        req.Unique,
		Indexed:       req.Indexed,
		Localized:     req.Localized,
		Options:       options,
		Order:         req.Order,
		Hidden:        req.Hidden,
		ReadOnly:      req.ReadOnly,
	}

	// Set default value if provided
	if req.DefaultValue != nil {
		if err := field.SetDefaultValue(req.DefaultValue); err != nil {
			return nil, core.ErrInvalidRequest("invalid default value: " + err.Error())
		}
	}

	if err := s.repo.Create(ctx, field); err != nil {
		return nil, err
	}

	return fieldToDTO(field), nil
}

// GetByID retrieves a content field by ID.
func (s *ContentFieldService) GetByID(ctx context.Context, id xid.ID) (*core.ContentFieldDTO, error) {
	field, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return fieldToDTO(field), nil
}

// GetByName retrieves a content field by slug.
func (s *ContentFieldService) GetByName(ctx context.Context, contentTypeID xid.ID, name string) (*core.ContentFieldDTO, error) {
	field, err := s.repo.FindByName(ctx, contentTypeID, name)
	if err != nil {
		return nil, err
	}

	return fieldToDTO(field), nil
}

// List lists fields for a content type.
func (s *ContentFieldService) List(ctx context.Context, contentTypeID xid.ID) ([]*core.ContentFieldDTO, error) {
	fields, err := s.repo.ListByContentType(ctx, contentTypeID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*core.ContentFieldDTO, len(fields))
	for i, field := range fields {
		dtos[i] = fieldToDTO(field)
	}

	return dtos, nil
}

// Update updates a content field.
func (s *ContentFieldService) Update(ctx context.Context, id xid.ID, req *core.UpdateFieldRequest) (*core.ContentFieldDTO, error) {
	field, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update basic fields
	if req.Title != "" {
		field.Title = strings.TrimSpace(req.Title)
	}

	if req.Description != "" {
		field.Description = strings.TrimSpace(req.Description)
	}

	if req.Required != nil {
		field.Required = *req.Required
	}

	if req.Unique != nil {
		field.Unique = *req.Unique
	}

	if req.Indexed != nil {
		field.Indexed = *req.Indexed
	}

	if req.Localized != nil {
		field.Localized = *req.Localized
	}

	if req.Order != nil {
		field.Order = *req.Order
	}

	if req.Hidden != nil {
		field.Hidden = *req.Hidden
	}

	if req.ReadOnly != nil {
		field.ReadOnly = *req.ReadOnly
	}

	// Update options if provided
	if req.Options != nil {
		fieldType := core.FieldType(field.Type)
		if err := s.validateFieldOptions(fieldType, req.Options); err != nil {
			return nil, err
		}

		field.Options = s.buildFieldOptions(req.Options)
	}

	// Update default value if provided
	if req.DefaultValue != nil {
		if err := field.SetDefaultValue(req.DefaultValue); err != nil {
			return nil, core.ErrInvalidRequest("invalid default value: " + err.Error())
		}
	}

	field.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, field); err != nil {
		return nil, err
	}

	return fieldToDTO(field), nil
}

// Delete deletes a content field.
func (s *ContentFieldService) Delete(ctx context.Context, id xid.ID) error {
	// Verify field exists
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}

// UpdateByName updates a content field by its name within a content type.
func (s *ContentFieldService) UpdateByName(ctx context.Context, contentTypeID xid.ID, name string, req *core.UpdateFieldRequest) (*core.ContentFieldDTO, error) {
	// Find field by name
	field, err := s.repo.FindByName(ctx, contentTypeID, name)
	if err != nil {
		return nil, err
	}

	return s.Update(ctx, field.ID, req)
}

// DeleteByName deletes a content field by its name within a content type.
func (s *ContentFieldService) DeleteByName(ctx context.Context, contentTypeID xid.ID, name string) error {
	// Find field by name
	field, err := s.repo.FindByName(ctx, contentTypeID, name)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, field.ID)
}

// =============================================================================
// Ordering Operations
// =============================================================================

// Reorder reorders fields in a content type.
func (s *ContentFieldService) Reorder(ctx context.Context, contentTypeID xid.ID, req *core.ReorderFieldsRequest) error {
	// Verify content type exists
	_, err := s.contentTypeRepo.FindByID(ctx, contentTypeID)
	if err != nil {
		return err
	}

	orders := make([]repository.FieldOrder, len(req.FieldOrders))
	for i, fo := range req.FieldOrders {
		fieldID, err := xid.FromString(fo.FieldID)
		if err != nil {
			return core.ErrInvalidRequest("invalid field ID: " + fo.FieldID)
		}

		orders[i] = repository.FieldOrder{
			FieldID: fieldID,
			Order:   fo.Order,
		}
	}

	return s.repo.ReorderFields(ctx, contentTypeID, orders)
}

// =============================================================================
// Validation Helpers
// =============================================================================

// validateFieldOptions validates options for a specific field type.
func (s *ContentFieldService) validateFieldOptions(fieldType core.FieldType, options *core.FieldOptionsDTO) error {
	if options == nil {
		// Check if field type requires options
		if fieldType.RequiresOptions() {
			return core.ErrInvalidRequest("field type '" + string(fieldType) + "' requires options")
		}

		return nil
	}

	// Validate based on field type
	switch fieldType {
	case core.FieldTypeSelect, core.FieldTypeMultiSelect, core.FieldTypeEnumeration:
		if len(options.Choices) == 0 {
			return core.ErrInvalidRequest("select/enumeration fields require choices")
		}
		// Validate choices have unique values
		values := make(map[string]bool)

		for _, choice := range options.Choices {
			if choice.Value == "" {
				return core.ErrInvalidRequest("choice value cannot be empty")
			}

			if values[choice.Value] {
				return core.ErrInvalidRequest("duplicate choice value: " + choice.Value)
			}

			values[choice.Value] = true
		}

	case core.FieldTypeRelation:
		if options.RelatedType == "" {
			return core.ErrInvalidRequest("relation fields require relatedType")
		}

		if options.RelationType == "" {
			return core.ErrInvalidRequest("relation fields require relationType")
		}

		relType := core.RelationType(options.RelationType)
		if !relType.IsValid() {
			return core.ErrInvalidRequest("invalid relation type: " + options.RelationType)
		}

	case core.FieldTypeNumber, core.FieldTypeInteger, core.FieldTypeFloat, core.FieldTypeDecimal:
		if options.Min != nil && options.Max != nil && *options.Min > *options.Max {
			return core.ErrInvalidRequest("min cannot be greater than max")
		}

	case core.FieldTypeText, core.FieldTypeTextarea:
		if options.MinLength > 0 && options.MaxLength > 0 && options.MinLength > options.MaxLength {
			return core.ErrInvalidRequest("minLength cannot be greater than maxLength")
		}

	case core.FieldTypeSlug:
		if options.SourceField == "" {
			return core.ErrInvalidRequest("slug fields require sourceField")
		}

	case core.FieldTypeObject, core.FieldTypeArray:
		// Object/array fields require either nestedFields or componentRef
		if len(options.NestedFields) == 0 && options.ComponentRef == "" {
			return core.ErrInvalidRequest("object/array fields require either nestedFields or componentRef")
		}
		// Validate minItems/maxItems for array fields
		if fieldType == core.FieldTypeArray {
			if options.MinItems != nil && options.MaxItems != nil && *options.MinItems > *options.MaxItems {
				return core.ErrInvalidRequest("minItems cannot be greater than maxItems")
			}
		}

	case core.FieldTypeOneOf:
		// OneOf fields require discriminatorField and at least one schema
		if options.DiscriminatorField == "" {
			return core.ErrInvalidRequest("oneOf fields require discriminatorField")
		}

		if len(options.Schemas) == 0 {
			return core.ErrInvalidRequest("oneOf fields require at least one schema in schemas map")
		}
		// Validate each schema option - allow empty schemas for "no config needed" cases
		// (e.g., a "custom" option where config comes from a separate JSON field)
		// Schema options can have componentRef, nestedFields, or be empty (just a label)
	}

	// Validate conditional visibility options (can apply to any field type)
	if err := s.validateConditionalVisibility(options); err != nil {
		return err
	}

	return nil
}

// validateConditionalVisibility validates showWhen/hideWhen conditions.
func (s *ContentFieldService) validateConditionalVisibility(options *core.FieldOptionsDTO) error {
	if options == nil {
		return nil
	}

	// Validate showWhen condition
	if options.ShowWhen != nil {
		if err := s.validateFieldCondition(options.ShowWhen); err != nil {
			return core.ErrInvalidRequest("invalid showWhen: " + err.Error())
		}
	}

	// Validate hideWhen condition
	if options.HideWhen != nil {
		if err := s.validateFieldCondition(options.HideWhen); err != nil {
			return core.ErrInvalidRequest("invalid hideWhen: " + err.Error())
		}
	}

	// Cannot have both showWhen and hideWhen on the same field
	if options.ShowWhen != nil && options.HideWhen != nil {
		return core.ErrInvalidRequest("field cannot have both showWhen and hideWhen conditions")
	}

	return nil
}

// validateFieldCondition validates a field condition.
func (s *ContentFieldService) validateFieldCondition(condition *core.FieldConditionDTO) error {
	if condition.Field == "" {
		return core.ErrInvalidRequest("condition field is required")
	}

	// Validate operator
	validOperators := map[string]bool{
		"eq":        true,
		"ne":        true,
		"in":        true,
		"notIn":     true,
		"exists":    true,
		"notExists": true,
	}
	if !validOperators[condition.Operator] {
		return core.ErrInvalidRequest("invalid operator: " + condition.Operator + ". Valid operators: eq, ne, in, notIn, exists, notExists")
	}

	// exists/notExists don't need a value
	if condition.Operator == "exists" || condition.Operator == "notExists" {
		return nil
	}

	// Other operators require a value
	if condition.Value == nil {
		return core.ErrInvalidRequest("operator '" + condition.Operator + "' requires a value")
	}

	return nil
}

// buildFieldOptions builds schema.FieldOptions from DTO.
func (s *ContentFieldService) buildFieldOptions(dto *core.FieldOptionsDTO) schema.FieldOptions {
	if dto == nil {
		return schema.FieldOptions{}
	}

	options := schema.FieldOptions{
		MinLength:                  dto.MinLength,
		MaxLength:                  dto.MaxLength,
		Pattern:                    dto.Pattern,
		Min:                        dto.Min,
		Max:                        dto.Max,
		Step:                       dto.Step,
		Integer:                    dto.Integer,
		RelatedType:                dto.RelatedType,
		RelationType:               dto.RelationType,
		OnDelete:                   dto.OnDelete,
		InverseField:               dto.InverseField,
		AllowHTML:                  dto.AllowHTML,
		MaxWords:                   dto.MaxWords,
		AllowedMimeTypes:           dto.AllowedMimeTypes,
		MaxFileSize:                dto.MaxFileSize,
		SourceField:                dto.SourceField,
		Schema:                     dto.Schema,
		MinDate:                    dto.MinDate,
		MaxDate:                    dto.MaxDate,
		DateFormat:                 dto.DateFormat,
		ComponentRef:               dto.ComponentRef,
		MinItems:                   dto.MinItems,
		MaxItems:                   dto.MaxItems,
		Collapsible:                dto.Collapsible,
		DefaultExpanded:            dto.DefaultExpanded,
		DiscriminatorField:         dto.DiscriminatorField,
		ClearOnDiscriminatorChange: dto.ClearOnDiscriminatorChange,
		ClearWhenHidden:            dto.ClearWhenHidden,
	}

	// Convert choices
	if len(dto.Choices) > 0 {
		options.Choices = make([]schema.Choice, len(dto.Choices))
		for i, choice := range dto.Choices {
			options.Choices[i] = schema.Choice{
				Value:    choice.Value,
				Label:    choice.Label,
				Icon:     choice.Icon,
				Color:    choice.Color,
				Disabled: choice.Disabled,
			}
		}
	}

	// Convert nested fields
	if len(dto.NestedFields) > 0 {
		options.NestedFields = s.dtoToSchemaNestedFields(dto.NestedFields)
	}

	// Convert oneOf schemas
	if len(dto.Schemas) > 0 {
		options.Schemas = make(map[string]schema.OneOfSchemaOption, len(dto.Schemas))
		for key, schemaOpt := range dto.Schemas {
			options.Schemas[key] = schema.OneOfSchemaOption{
				ComponentRef: schemaOpt.ComponentRef,
				NestedFields: s.dtoToSchemaNestedFields(schemaOpt.NestedFields),
				Label:        schemaOpt.Label,
			}
		}
	}

	// Convert conditional visibility
	if dto.ShowWhen != nil {
		options.ShowWhen = &schema.FieldCondition{
			Field:    dto.ShowWhen.Field,
			Operator: dto.ShowWhen.Operator,
			Value:    dto.ShowWhen.Value,
		}
	}

	if dto.HideWhen != nil {
		options.HideWhen = &schema.FieldCondition{
			Field:    dto.HideWhen.Field,
			Operator: dto.HideWhen.Operator,
			Value:    dto.HideWhen.Value,
		}
	}

	return options
}

// dtoToSchemaNestedFields converts DTO nested fields to schema nested fields.
func (s *ContentFieldService) dtoToSchemaNestedFields(fields []core.NestedFieldDefDTO) schema.NestedFieldDefs {
	result := make(schema.NestedFieldDefs, len(fields))
	for i, f := range fields {
		result[i] = schema.NestedFieldDef{
			Title:       f.Title,
			Name:        f.Name,
			Type:        f.Type,
			Required:    f.Required,
			Description: f.Description,
		}
		if f.Options != nil {
			result[i].Options = s.buildFieldOptionsPtr(f.Options)
		}
	}

	return result
}

// buildFieldOptionsPtr builds a pointer to schema.FieldOptions from DTO.
func (s *ContentFieldService) buildFieldOptionsPtr(dto *core.FieldOptionsDTO) *schema.FieldOptions {
	if dto == nil {
		return nil
	}

	opts := s.buildFieldOptions(dto)

	return &opts
}

// ResolveNestedFields resolves nested fields for a field, handling component refs.
func (s *ContentFieldService) ResolveNestedFields(ctx context.Context, field *core.ContentFieldDTO) ([]core.NestedFieldDefDTO, error) {
	if field.Options.ComponentRef != "" && s.componentSchemaService != nil {
		return s.componentSchemaService.ResolveComponentSchema(ctx, field.Options.ComponentRef)
	}

	return field.Options.NestedFields, nil
}

// GetFieldWithResolvedNested returns a field DTO with resolved nested fields.
func (s *ContentFieldService) GetFieldWithResolvedNested(ctx context.Context, id xid.ID) (*core.ContentFieldDTO, error) {
	field, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	dto := fieldToDTO(field)

	// If this is a nested field type with a component ref, resolve it
	fieldType := core.FieldType(field.Type)
	if fieldType.IsNested() && field.Options.ComponentRef != "" && s.componentSchemaService != nil {
		resolvedFields, err := s.componentSchemaService.ResolveComponentSchema(ctx, field.Options.ComponentRef)
		if err != nil {
			return nil, err
		}

		dto.Options.NestedFields = resolvedFields
	}

	return dto, nil
}
