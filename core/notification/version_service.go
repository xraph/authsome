package notification

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// VersionService handles template versioning operations.
type VersionService struct {
	repo Repository
}

// NewVersionService creates a new version service.
func NewVersionService(repo Repository) *VersionService {
	return &VersionService{repo: repo}
}

// CreateVersion creates a new version snapshot of a template.
func (s *VersionService) CreateVersion(ctx context.Context, template *schema.NotificationTemplate, changedBy *xid.ID, changes string) (*schema.NotificationTemplateVersion, error) {
	version := &schema.NotificationTemplateVersion{
		ID:         xid.New(),
		TemplateID: template.ID,
		Version:    template.Version,
		Subject:    template.Subject,
		Body:       template.Body,
		Variables:  template.Variables,
		Changes:    changes,
		ChangedBy:  changedBy,
		Metadata:   template.Metadata,
	}

	if err := s.repo.CreateTemplateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create template version: %w", err)
	}

	return version, nil
}

// GetVersion retrieves a specific version by ID.
func (s *VersionService) GetVersion(ctx context.Context, id xid.ID) (*schema.NotificationTemplateVersion, error) {
	version, err := s.repo.FindTemplateVersionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find version: %w", err)
	}

	if version == nil {
		return nil, VersionNotFound()
	}

	return version, nil
}

// ListVersions lists all versions for a template.
func (s *VersionService) ListVersions(ctx context.Context, templateID xid.ID) ([]*schema.NotificationTemplateVersion, error) {
	versions, err := s.repo.ListTemplateVersions(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	return versions, nil
}

// GetLatestVersion gets the most recent version for a template.
func (s *VersionService) GetLatestVersion(ctx context.Context, templateID xid.ID) (*schema.NotificationTemplateVersion, error) {
	version, err := s.repo.GetLatestTemplateVersion(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	if version == nil {
		return nil, VersionNotFound()
	}

	return version, nil
}

// RestoreVersion restores a template to a previous version.
func (s *VersionService) RestoreVersion(ctx context.Context, templateID xid.ID, versionID xid.ID, restoredBy *xid.ID) error {
	// Get the version to restore
	version, err := s.repo.FindTemplateVersionByID(ctx, versionID)
	if err != nil {
		return fmt.Errorf("failed to find version: %w", err)
	}

	if version == nil {
		return VersionNotFound()
	}

	// Get the current template
	template, err := s.repo.FindTemplateByID(ctx, templateID)
	if err != nil {
		return fmt.Errorf("failed to find template: %w", err)
	}

	if template == nil {
		return TemplateNotFound()
	}

	// Create a version of current state before restoring
	currentVersion := &schema.NotificationTemplateVersion{
		ID:         xid.New(),
		TemplateID: template.ID,
		Version:    template.Version,
		Subject:    template.Subject,
		Body:       template.Body,
		Variables:  template.Variables,
		Changes:    fmt.Sprintf("Pre-restore snapshot before restoring to version %d", version.Version),
		ChangedBy:  restoredBy,
		Metadata:   template.Metadata,
	}
	if err := s.repo.CreateTemplateVersion(ctx, currentVersion); err != nil {
		return fmt.Errorf("failed to create pre-restore snapshot: %w", err)
	}

	// Update template with restored content
	subject := version.Subject
	body := version.Body
	updateReq := &UpdateTemplateRequest{
		Subject:   &subject,
		Body:      &body,
		Variables: version.Variables,
		Metadata:  version.Metadata,
	}

	if err := s.repo.UpdateTemplate(ctx, templateID, updateReq); err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	// Increment version number for the template
	template.Version++

	// Create a new version entry marking this as a restore
	restoredVersion := &schema.NotificationTemplateVersion{
		ID:         xid.New(),
		TemplateID: template.ID,
		Version:    template.Version,
		Subject:    version.Subject,
		Body:       version.Body,
		Variables:  version.Variables,
		Changes:    fmt.Sprintf("Restored from version %d", version.Version),
		ChangedBy:  restoredBy,
		Metadata:   version.Metadata,
	}

	if err := s.repo.CreateTemplateVersion(ctx, restoredVersion); err != nil {
		return fmt.Errorf("failed to create restored version: %w", err)
	}

	return nil
}

// CompareVersions compares two versions and returns the differences.
func (s *VersionService) CompareVersions(ctx context.Context, version1ID, version2ID xid.ID) (*VersionComparison, error) {
	v1, err := s.repo.FindTemplateVersionByID(ctx, version1ID)
	if err != nil || v1 == nil {
		return nil, fmt.Errorf("failed to find version 1: %w", err)
	}

	v2, err := s.repo.FindTemplateVersionByID(ctx, version2ID)
	if err != nil || v2 == nil {
		return nil, fmt.Errorf("failed to find version 2: %w", err)
	}

	comparison := &VersionComparison{
		Version1:       v1,
		Version2:       v2,
		SubjectChanged: v1.Subject != v2.Subject,
		BodyChanged:    v1.Body != v2.Body,
	}

	return comparison, nil
}

// VersionComparison represents a comparison between two versions.
type VersionComparison struct {
	Version1       *schema.NotificationTemplateVersion `json:"version1"`
	Version2       *schema.NotificationTemplateVersion `json:"version2"`
	SubjectChanged bool                                `json:"subjectChanged"`
	BodyChanged    bool                                `json:"bodyChanged"`
}
