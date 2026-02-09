package forms

import (
	"context"
	"fmt"
	"slices"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// Service handles form business logic.
type Service struct {
	repo   Repository
	config Config
}

// NewService creates a new forms service.
func NewService(forgeConfig any, repo Repository) (*Service, error) {
	var cfg Config = DefaultConfig()

	return &Service{
		repo:   repo,
		config: cfg,
	}, nil
}

// CreateForm creates a new form configuration.
func (s *Service) CreateForm(ctx context.Context, req *CreateFormRequest) (*Form, error) {
	// Validate form type
	if !s.isValidFormType(req.Type) {
		return nil, fmt.Errorf("invalid form type: %s", req.Type)
	}

	// Validate schema
	if err := s.validateFormSchema(req.Schema); err != nil {
		return nil, fmt.Errorf("invalid form schema: %w", err)
	}

	// Create schema entity
	formSchema := &schema.FormSchema{
		ID:          xid.New(),
		AppID:       req.AppID,
		Type:        req.Type,
		Name:        req.Name,
		Description: req.Description,
		Schema:      req.Schema,
		IsActive:    true,
		Version:     1,
	}

	// Set auditable fields
	formSchema.CreatedBy = req.AppID // Use org ID as creator for now
	formSchema.UpdatedBy = req.AppID

	if err := s.repo.Create(ctx, formSchema); err != nil {
		return nil, fmt.Errorf("failed to create form: %w", err)
	}

	return s.schemaToForm(formSchema), nil
}

// GetForm retrieves a form by ID.
func (s *Service) GetForm(ctx context.Context, id xid.ID) (*Form, error) {
	formSchema, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get form: %w", err)
	}

	return s.schemaToForm(formSchema), nil
}

// GetFormByOrganization retrieves a form by organization and type.
func (s *Service) GetFormByOrganization(ctx context.Context, orgID xid.ID, formType string) (*Form, error) {
	formSchema, err := s.repo.GetByOrganization(ctx, orgID, formType)
	if err != nil {
		return nil, fmt.Errorf("failed to get form by organization: %w", err)
	}

	return s.schemaToForm(formSchema), nil
}

// ListForms lists forms for an organization.
func (s *Service) ListForms(ctx context.Context, req *ListFormsRequest) (*ListFormsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}

	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	formSchemas, total, err := s.repo.List(ctx, req.OrganizationID, req.Type, req.Page, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list forms: %w", err)
	}

	forms := make([]*Form, len(formSchemas))
	for i, fs := range formSchemas {
		forms[i] = s.schemaToForm(fs)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return &ListFormsResponse{
		Forms:      forms,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateForm updates an existing form.
func (s *Service) UpdateForm(ctx context.Context, req *UpdateFormRequest) (*Form, error) {
	// Get existing form
	formSchema, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get form for update: %w", err)
	}

	// Validate schema if provided
	if req.Schema != nil {
		if err := s.validateFormSchema(req.Schema); err != nil {
			return nil, fmt.Errorf("invalid form schema: %w", err)
		}

		formSchema.Schema = req.Schema
		formSchema.Version++
	}

	// Update fields
	if req.Name != "" {
		formSchema.Name = req.Name
	}

	if req.Description != "" {
		formSchema.Description = req.Description
	}

	formSchema.IsActive = req.IsActive

	// Update auditable fields
	formSchema.UpdatedBy = formSchema.AppID // Use org ID as updater for now

	if err := s.repo.Update(ctx, formSchema); err != nil {
		return nil, fmt.Errorf("failed to update form: %w", err)
	}

	return s.schemaToForm(formSchema), nil
}

// DeleteForm deletes a form.
func (s *Service) DeleteForm(ctx context.Context, id xid.ID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete form: %w", err)
	}

	return nil
}

// SubmitForm submits form data.
func (s *Service) SubmitForm(ctx context.Context, req *SubmitFormRequest) (*FormSubmission, error) {
	// Get form schema to validate against
	formSchema, err := s.repo.GetByID(ctx, req.FormSchemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get form schema: %w", err)
	}

	if !formSchema.IsActive {
		return nil, errs.BadRequest("form is not active")
	}

	// Validate submission data against schema
	if err := s.validateSubmissionData(formSchema.Schema, req.Data); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create submission
	submission := &schema.FormSubmission{
		ID:           xid.New(),
		FormSchemaID: req.FormSchemaID,
		UserID:       req.UserID,
		SessionID:    req.SessionID,
		Data:         req.Data,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		Status:       "submitted",
	}

	// Set auditable fields
	createdBy := formSchema.AppID
	if req.UserID != nil {
		createdBy = *req.UserID
	}

	submission.CreatedBy = createdBy
	submission.UpdatedBy = createdBy

	if err := s.repo.CreateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	return s.schemaToSubmission(submission), nil
}

// GetSubmission retrieves a form submission by ID.
func (s *Service) GetSubmission(ctx context.Context, id xid.ID) (*FormSubmission, error) {
	submission, err := s.repo.GetSubmissionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	return s.schemaToSubmission(submission), nil
}

// GetDefaultSignupForm returns the default signup form configuration.
func (s *Service) GetDefaultSignupForm() map[string]any {
	return map[string]any{
		"fields": []map[string]any{
			{
				"id":          "email",
				"type":        "email",
				"label":       "Email Address",
				"placeholder": "Enter your email",
				"required":    true,
				"validation": map[string]any{
					"pattern": s.config.ValidationRules["email"].(map[string]any)["pattern"],
				},
			},
			{
				"id":          "password",
				"type":        "password",
				"label":       "Password",
				"placeholder": "Enter your password",
				"required":    true,
				"validation": map[string]any{
					"minLength": s.config.ValidationRules["password"].(map[string]any)["minLength"],
					"maxLength": s.config.ValidationRules["password"].(map[string]any)["maxLength"],
				},
			},
			{
				"id":          "name",
				"type":        "text",
				"label":       "Full Name",
				"placeholder": "Enter your full name",
				"required":    true,
			},
		},
	}
}

// Helper methods

func (s *Service) isValidFormType(formType string) bool {

	return slices.Contains(s.config.AllowedTypes, formType)
}

func (s *Service) validateFormSchema(schema map[string]any) error {
	fields, ok := schema["fields"].([]any)
	if !ok {
		return errs.InvalidInput("schema", "must contain fields array")
	}

	if len(fields) > s.config.MaxFieldCount {
		return fmt.Errorf("too many fields: %d (max: %d)", len(fields), s.config.MaxFieldCount)
	}

	for i, field := range fields {
		fieldMap, ok := field.(map[string]any)
		if !ok {
			return fmt.Errorf("field %d is not a valid object", i)
		}

		// Validate required field properties
		if _, ok := fieldMap["id"]; !ok {
			return fmt.Errorf("field %d missing id", i)
		}

		if _, ok := fieldMap["type"]; !ok {
			return fmt.Errorf("field %d missing type", i)
		}

		if _, ok := fieldMap["label"]; !ok {
			return fmt.Errorf("field %d missing label", i)
		}
	}

	return nil
}

func (s *Service) validateSubmissionData(schema map[string]any, data map[string]any) error {
	fields, ok := schema["fields"].([]any)
	if !ok {
		return errs.InvalidInput("schema", "missing fields")
	}

	for _, field := range fields {
		fieldMap, ok := field.(map[string]any)
		if !ok {
			continue
		}

		fieldID, _ := fieldMap["id"].(string)
		required, _ := fieldMap["required"].(bool)

		if required {
			if _, exists := data[fieldID]; !exists {
				return fmt.Errorf("required field '%s' is missing", fieldID)
			}
		}

		// Additional validation can be added here based on field type and validation rules
	}

	return nil
}

func (s *Service) schemaToForm(fs *schema.FormSchema) *Form {
	return &Form{
		ID:          fs.ID,
		AppID:       fs.AppID,
		Type:        fs.Type,
		Name:        fs.Name,
		Description: fs.Description,
		Schema:      fs.Schema,
		IsActive:    fs.IsActive,
		Version:     fs.Version,
		CreatedAt:   fs.CreatedAt,
		UpdatedAt:   fs.UpdatedAt,
	}
}

func (s *Service) schemaToSubmission(fs *schema.FormSubmission) *FormSubmission {
	return &FormSubmission{
		ID:           fs.ID,
		FormSchemaID: fs.FormSchemaID,
		UserID:       fs.UserID,
		SessionID:    fs.SessionID,
		Data:         fs.Data,
		IPAddress:    fs.IPAddress,
		UserAgent:    fs.UserAgent,
		Status:       fs.Status,
		CreatedAt:    fs.CreatedAt,
		UpdatedAt:    fs.UpdatedAt,
	}
}
