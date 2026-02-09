package notification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// TEMPLATE INITIALIZATION & RESET
// =============================================================================

// InitializeDefaultTemplates creates default templates for an app.
func (s *Service) InitializeDefaultTemplates(ctx context.Context, appID xid.ID) error {
	// Get all default template metadata
	defaultTemplates := GetDefaultTemplateMetadata()

	// Create each default template if it doesn't exist
	for _, metadata := range defaultTemplates {
		// Check if template already exists
		exists, err := s.TemplateExists(ctx, appID, metadata.Key)
		if err != nil {
			return fmt.Errorf("failed to check template existence: %w", err)
		}

		if exists {
			// Skip if template already exists
			continue
		}

		// Use HTML body if available, otherwise use text body
		body := metadata.DefaultBody
		if metadata.DefaultBodyHTML != "" {
			body = metadata.DefaultBodyHTML
		}

		// Calculate hash of default content
		defaultHash := calculateTemplateHash(metadata.DefaultSubject, body)

		// Create the template
		template := &schema.NotificationTemplate{
			ID:          xid.New(),
			AppID:       appID,
			TemplateKey: metadata.Key,
			Name:        metadata.Name,
			Type:        string(metadata.Type),
			Language:    "en", // Default language
			Subject:     metadata.DefaultSubject,
			Body:        body,
			Variables:   metadata.Variables,
			Metadata: map[string]any{
				"description": metadata.Description,
			},
			Active:      true,
			IsDefault:   true,
			IsModified:  false,
			DefaultHash: defaultHash,
		}

		if err := s.repo.CreateTemplate(ctx, template); err != nil {
			return fmt.Errorf("failed to create template %s: %w", metadata.Key, err)
		}
	}

	return nil
}

// ResetTemplate resets a template to its default values.
func (s *Service) ResetTemplate(ctx context.Context, templateID xid.ID) error {
	// Find the template
	template, err := s.repo.FindTemplateByID(ctx, templateID)
	if err != nil {
		return err
	}

	// Get the default template metadata
	defaultMeta, err := GetDefaultTemplate(template.TemplateKey)
	if err != nil {
		return fmt.Errorf("cannot reset template: %w", err)
	}

	// Use HTML body if available, otherwise use text body
	body := defaultMeta.DefaultBody
	if defaultMeta.DefaultBodyHTML != "" {
		body = defaultMeta.DefaultBodyHTML
	}

	// Calculate hash of default content
	defaultHash := calculateTemplateHash(defaultMeta.DefaultSubject, body)

	// Update the template with default values
	updateReq := &UpdateTemplateRequest{
		Name:      &defaultMeta.Name,
		Subject:   &defaultMeta.DefaultSubject,
		Body:      &body,
		Variables: defaultMeta.Variables,
		Metadata: map[string]any{
			"description": defaultMeta.Description,
		},
		Active: boolPtr(true),
	}

	if err := s.repo.UpdateTemplate(ctx, templateID, updateReq); err != nil {
		return fmt.Errorf("failed to reset template: %w", err)
	}

	// Update metadata flags
	if err := s.repo.UpdateTemplateMetadata(ctx, templateID, true, false, defaultHash); err != nil {
		return fmt.Errorf("failed to update template metadata: %w", err)
	}

	return nil
}

// ResetAllTemplates resets all templates for an app to defaults.
func (s *Service) ResetAllTemplates(ctx context.Context, appID xid.ID) error {
	// First, ensure all default templates exist
	// InitializeDefaultTemplates is idempotent - it only creates missing templates
	if err := s.InitializeDefaultTemplates(ctx, appID); err != nil {
		return fmt.Errorf("failed to initialize default templates: %w", err)
	}

	// Get all templates for the app
	filter := &ListTemplatesFilter{
		AppID: appID,
	}
	filter.Limit = 1000 // Get all templates

	response, err := s.repo.ListTemplates(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	// Reset each template that is a default template
	for _, template := range response.Data {
		if template.IsDefault {
			_ = s.ResetTemplate(ctx, template.ID)
		}
	}

	return nil
}

// TemplateExists checks if a template exists.
func (s *Service) TemplateExists(ctx context.Context, appID xid.ID, templateKey string) (bool, error) {
	// Try to find template by key
	template, err := s.repo.FindTemplateByKey(ctx, appID, templateKey, "", "")
	if err != nil {
		// Check if it's a not found error
		if IsTemplateNotFoundError(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to check template existence: %w", err)
	}

	// Check if template is nil (some repos return nil, nil for not found)
	if template == nil {
		return false, nil
	}

	return true, nil
}

// CompareWithDefault checks if template differs from default.
func (s *Service) CompareWithDefault(ctx context.Context, templateID xid.ID) (bool, error) {
	// Find the template
	template, err := s.repo.FindTemplateByID(ctx, templateID)
	if err != nil {
		return false, err
	}

	// If not a default template, can't compare
	if !template.IsDefault {
		return false, errs.New(errs.CodeBadRequest, "template is not a default template", http.StatusBadRequest)
	}

	// Get the default template metadata
	defaultMeta, err := GetDefaultTemplate(template.TemplateKey)
	if err != nil {
		return false, fmt.Errorf("cannot find default template: %w", err)
	}

	// Use HTML body if available, otherwise use text body
	body := defaultMeta.DefaultBody
	if defaultMeta.DefaultBodyHTML != "" {
		body = defaultMeta.DefaultBodyHTML
	}

	// Calculate hash of current default content
	currentDefaultHash := calculateTemplateHash(defaultMeta.DefaultSubject, body)

	// Compare with stored hash
	isDifferent := template.DefaultHash != currentDefaultHash

	return isDifferent, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// calculateTemplateHash calculates a SHA256 hash of template content.
func calculateTemplateHash(subject, body string) string {
	content := subject + "|" + body
	hash := sha256.Sum256([]byte(content))

	return hex.EncodeToString(hash[:])
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// IsTemplateNotFoundError checks if an error is a template not found error.
func IsTemplateNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error is our template not found error
	return err.Error() == ErrTemplateNotFound.Error() || errs.Is(err, ErrTemplateNotFound)
}
