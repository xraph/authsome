package forms

import (
	"context"
	"fmt"

	"github.com/rs/xid"
)

// SignupFormService handles signup form management with Smartform integration
type SignupFormService struct {
	service *Service
}

// NewSignupFormService creates a new signup form service
func NewSignupFormService(service *Service) *SignupFormService {
	return &SignupFormService{
		service: service,
	}
}

// GetSignupForm retrieves the signup form for an organization
// Returns custom form if exists, otherwise returns default form
func (s *SignupFormService) GetSignupForm(ctx context.Context, orgID xid.ID) (*Form, error) {
	// Try to get custom form first
	customForm, err := s.service.GetFormByOrganization(ctx, orgID, "signup")
	if err == nil && customForm != nil {
		return customForm, nil
	}

	// Return default form if no custom form exists
	return s.getDefaultForm(ctx, orgID)
}

// SaveFormSchema saves a custom signup form schema for an organization
func (s *SignupFormService) SaveFormSchema(ctx context.Context, req *SaveFormSchemaRequest) (*Form, error) {
	// Validate the form schema
	if err := s.validateSignupFormSchema(req.Schema); err != nil {
		return nil, fmt.Errorf("invalid form schema: %w", err)
	}

	// Check if form already exists
	existingForm, err := s.service.GetFormByOrganization(ctx, req.OrganizationID, "signup")

	if err == nil && existingForm != nil {
		// Update existing form
		return s.service.UpdateForm(ctx, &UpdateFormRequest{
			ID:     existingForm.ID,
			Schema: req.Schema,
		})
	}

	// Create new form
	return s.service.CreateForm(ctx, &CreateFormRequest{
		OrganizationID: req.OrganizationID,
		Name:           "Signup Form",
		Type:           "signup",
		Schema:         req.Schema,
	})
}

// GetCustomForm retrieves a custom signup form for an organization
func (s *SignupFormService) GetCustomForm(ctx context.Context, orgID xid.ID) (*Form, error) {
	return s.service.GetFormByOrganization(ctx, orgID, "signup")
}

// DeleteCustomForm removes a custom signup form for an organization
func (s *SignupFormService) DeleteCustomForm(ctx context.Context, orgID xid.ID) error {
	form, err := s.GetCustomForm(ctx, orgID)
	if err != nil {
		return err
	}
	if form == nil {
		return fmt.Errorf("no custom signup form found for organization")
	}

	return s.service.DeleteForm(ctx, form.ID)
}

// SubmitSignupForm processes a signup form submission
func (s *SignupFormService) SubmitSignupForm(ctx context.Context, req *SubmitSignupFormRequest) (*FormSubmission, error) {
	// Get the form to validate against
	form, err := s.GetSignupForm(ctx, req.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get signup form: %w", err)
	}

	// Validate submission data against form schema
	if err := s.validateSubmissionData(form.Schema, req.Data); err != nil {
		return nil, fmt.Errorf("invalid submission data: %w", err)
	}

	// Submit the form
	return s.service.SubmitForm(ctx, &SubmitFormRequest{
		FormSchemaID: form.ID,
		Data:         req.Data,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
	})
}

// getDefaultForm returns the default signup form
func (s *SignupFormService) getDefaultForm(ctx context.Context, orgID xid.ID) (*Form, error) {
	// Create default signup form schema
	defaultSchema := map[string]interface{}{
		"fields": []map[string]interface{}{
			{
				"name":        "email",
				"type":        "email",
				"label":       "Email Address",
				"required":    true,
				"placeholder": "Enter your email",
				"validation": map[string]interface{}{
					"email": true,
				},
			},
			{
				"name":        "password",
				"type":        "password",
				"label":       "Password",
				"required":    true,
				"placeholder": "Enter your password",
				"validation": map[string]interface{}{
					"minLength": 8,
				},
			},
			{
				"name":        "firstName",
				"type":        "text",
				"label":       "First Name",
				"required":    false,
				"placeholder": "Enter your first name",
			},
			{
				"name":        "lastName",
				"type":        "text",
				"label":       "Last Name",
				"required":    false,
				"placeholder": "Enter your last name",
			},
		},
		"settings": map[string]interface{}{
			"submitText":    "Sign Up",
			"redirectUrl":   "/dashboard",
			"showTerms":     true,
			"termsText":     "I agree to the Terms of Service and Privacy Policy",
			"allowSocial":   true,
			"socialOptions": []string{"google", "github"},
		},
	}

	return &Form{
		ID:             xid.New(),
		OrganizationID: orgID,
		Name:           "Default Signup Form",
		Type:           "signup",
		Schema:         defaultSchema,
		IsActive:       true,
		Version:        1,
	}, nil
}

// validateSignupFormSchema validates a signup form schema
func (s *SignupFormService) validateSignupFormSchema(schema map[string]interface{}) error {
	// Check if fields exist
	fields, ok := schema["fields"]
	if !ok {
		return fmt.Errorf("schema must contain 'fields' array")
	}

	fieldsArray, ok := fields.([]interface{})
	if !ok {
		return fmt.Errorf("'fields' must be an array")
	}

	// Validate required fields exist
	hasEmail := false
	hasPassword := false

	for _, field := range fieldsArray {
		fieldMap, ok := field.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := fieldMap["name"].(string)
		if !ok {
			continue
		}

		switch name {
		case "email":
			hasEmail = true
		case "password":
			hasPassword = true
		}
	}

	if !hasEmail {
		return fmt.Errorf("signup form must contain an 'email' field")
	}

	if !hasPassword {
		return fmt.Errorf("signup form must contain a 'password' field")
	}

	return nil
}

// validateSubmissionData validates form submission data against schema
func (s *SignupFormService) validateSubmissionData(schema map[string]interface{}, data map[string]interface{}) error {
	fields, ok := schema["fields"]
	if !ok {
		return fmt.Errorf("invalid schema: missing fields")
	}

	fieldsArray, ok := fields.([]interface{})
	if !ok {
		return fmt.Errorf("invalid schema: fields must be array")
	}

	// Validate each field
	for _, field := range fieldsArray {
		fieldMap, ok := field.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := fieldMap["name"].(string)
		if !ok {
			continue
		}

		required, _ := fieldMap["required"].(bool)
		value, exists := data[name]

		// Check required fields
		if required && (!exists || value == nil || value == "") {
			return fmt.Errorf("required field '%s' is missing or empty", name)
		}

		// Validate field type if value exists
		if exists && value != nil {
			fieldType, _ := fieldMap["type"].(string)
			if err := s.validateFieldValue(name, fieldType, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateFieldValue validates a single field value
func (s *SignupFormService) validateFieldValue(name, fieldType string, value interface{}) error {
	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' must be a string", name)
	}

	switch fieldType {
	case "email":
		// Basic email validation (in real implementation, use proper email validation)
		if len(strValue) == 0 || !contains(strValue, "@") {
			return fmt.Errorf("field '%s' must be a valid email address", name)
		}
	case "password":
		if len(strValue) < 8 {
			return fmt.Errorf("field '%s' must be at least 8 characters long", name)
		}
	}

	return nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || s[len(s)-len(substr):] == substr || 
		s[:len(substr)] == substr || containsMiddle(s, substr))
}

// containsMiddle checks if substring exists in the middle of string
func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Request/Response types for signup form service

// SaveFormSchemaRequest represents a request to save a form schema
type SaveFormSchemaRequest struct {
	OrganizationID xid.ID                 `json:"organization_id"`
	Schema         map[string]interface{} `json:"schema"`
}

// SubmitSignupFormRequest represents a signup form submission request
type SubmitSignupFormRequest struct {
	OrganizationID xid.ID                 `json:"organization_id"`
	Data           map[string]interface{} `json:"data"`
	IPAddress      string                 `json:"ip_address,omitempty"`
	UserAgent      string                 `json:"user_agent,omitempty"`
}