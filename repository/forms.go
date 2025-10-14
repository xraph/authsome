package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/forms"
	"github.com/xraph/authsome/schema"
)

// formsRepository implements the forms.Repository interface using Bun ORM
type formsRepository struct {
	db *bun.DB
}

// NewFormsRepository creates a new forms repository
func NewFormsRepository(db *bun.DB) forms.Repository {
	return &formsRepository{
		db: db,
	}
}

// Create creates a new form schema in the database
func (r *formsRepository) Create(ctx context.Context, form *schema.FormSchema) error {
	if form.ID.IsNil() {
		form.ID = xid.New()
	}

	_, err := r.db.NewInsert().
		Model(form).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create form: %w", err)
	}

	return nil
}

// GetByID retrieves a form schema by ID
func (r *formsRepository) GetByID(ctx context.Context, id xid.ID) (*schema.FormSchema, error) {
	form := &schema.FormSchema{}
	err := r.db.NewSelect().
		Model(form).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("form not found")
		}
		return nil, fmt.Errorf("failed to get form: %w", err)
	}

	return form, nil
}

// GetByOrganization retrieves a form schema by organization and type
func (r *formsRepository) GetByOrganization(ctx context.Context, orgID xid.ID, formType string) (*schema.FormSchema, error) {
	form := &schema.FormSchema{}
	err := r.db.NewSelect().
		Model(form).
		Where("organization_id = ? AND type = ? AND is_active = ?", orgID, formType, true).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("form not found")
		}
		return nil, fmt.Errorf("failed to get form by organization: %w", err)
	}

	return form, nil
}

// List retrieves forms for an organization with pagination
func (r *formsRepository) List(ctx context.Context, orgID xid.ID, formType string, page, pageSize int) ([]*schema.FormSchema, int, error) {
	offset := (page - 1) * pageSize

	query := r.db.NewSelect().
		Model((*schema.FormSchema)(nil)).
		Where("organization_id = ?", orgID)

	if formType != "" {
		query = query.Where("type = ?", formType)
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count forms: %w", err)
	}

	// Get forms with pagination
	var forms []*schema.FormSchema
	err = query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(ctx, &forms)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list forms: %w", err)
	}

	return forms, total, nil
}

// Update updates an existing form schema
func (r *formsRepository) Update(ctx context.Context, form *schema.FormSchema) error {
	result, err := r.db.NewUpdate().
		Model(form).
		Where("id = ?", form.ID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update form: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("form not found")
	}

	return nil
}

// Delete deletes a form schema by ID
func (r *formsRepository) Delete(ctx context.Context, id xid.ID) error {
	result, err := r.db.NewDelete().
		Model((*schema.FormSchema)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete form: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("form not found")
	}

	return nil
}

// CreateSubmission creates a new form submission
func (r *formsRepository) CreateSubmission(ctx context.Context, submission *schema.FormSubmission) error {
	if submission.ID.IsNil() {
		submission.ID = xid.New()
	}

	_, err := r.db.NewInsert().
		Model(submission).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create form submission: %w", err)
	}

	return nil
}

// GetSubmissionByID retrieves a form submission by ID
func (r *formsRepository) GetSubmissionByID(ctx context.Context, id xid.ID) (*schema.FormSubmission, error) {
	submission := &schema.FormSubmission{}
	err := r.db.NewSelect().
		Model(submission).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("form submission not found")
		}
		return nil, fmt.Errorf("failed to get form submission: %w", err)
	}

	return submission, nil
}

// ListSubmissions retrieves form submissions for a form with pagination
func (r *formsRepository) ListSubmissions(ctx context.Context, formSchemaID xid.ID, page, pageSize int) ([]*schema.FormSubmission, int, error) {
	offset := (page - 1) * pageSize

	query := r.db.NewSelect().
		Model((*schema.FormSubmission)(nil)).
		Where("form_schema_id = ?", formSchemaID)

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count form submissions: %w", err)
	}

	// Get submissions with pagination
	var submissions []*schema.FormSubmission
	err = query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(ctx, &submissions)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list form submissions: %w", err)
	}

	return submissions, total, nil
}

// GetSubmissionsByUser retrieves form submissions for a user with pagination
func (r *formsRepository) GetSubmissionsByUser(ctx context.Context, userID xid.ID, page, pageSize int) ([]*schema.FormSubmission, int, error) {
	offset := (page - 1) * pageSize

	query := r.db.NewSelect().
		Model((*schema.FormSubmission)(nil)).
		Where("user_id = ?", userID)

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user form submissions: %w", err)
	}

	// Get submissions with pagination
	var submissions []*schema.FormSubmission
	err = query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(ctx, &submissions)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list user form submissions: %w", err)
	}

	return submissions, total, nil
}

// Additional helper methods for advanced queries

// GetActiveFormsByType retrieves all active forms of a specific type for an organization
func (r *formsRepository) GetActiveFormsByType(ctx context.Context, orgID xid.ID, formType string) ([]*schema.FormSchema, error) {
	var forms []*schema.FormSchema
	err := r.db.NewSelect().
		Model(&forms).
		Where("organization_id = ? AND type = ? AND is_active = ?", orgID, formType, true).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get active forms by type: %w", err)
	}

	return forms, nil
}

// GetFormVersions retrieves all versions of a form (by name and organization)
func (r *formsRepository) GetFormVersions(ctx context.Context, orgID xid.ID, name string) ([]*schema.FormSchema, error) {
	var forms []*schema.FormSchema
	err := r.db.NewSelect().
		Model(&forms).
		Where("organization_id = ? AND name = ?", orgID, name).
		Order("version DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get form versions: %w", err)
	}

	return forms, nil
}

// GetSubmissionStats retrieves submission statistics for a form
func (r *formsRepository) GetSubmissionStats(ctx context.Context, formSchemaID xid.ID) (*FormSubmissionStats, error) {
	stats := &FormSubmissionStats{}

	// Get total submissions
	total, err := r.db.NewSelect().
		Model((*schema.FormSubmission)(nil)).
		Where("form_schema_id = ?", formSchemaID).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total submissions: %w", err)
	}
	stats.Total = total

	// Get submissions by status
	var statusCounts []struct {
		Status string `bun:"status"`
		Count  int    `bun:"count"`
	}

	err = r.db.NewSelect().
		Model((*schema.FormSubmission)(nil)).
		Column("status").
		ColumnExpr("COUNT(*) as count").
		Where("form_schema_id = ?", formSchemaID).
		Group("status").
		Scan(ctx, &statusCounts)

	if err != nil {
		return nil, fmt.Errorf("failed to get submission status counts: %w", err)
	}

	stats.ByStatus = make(map[string]int)
	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
	}

	return stats, nil
}

// FormSubmissionStats represents submission statistics for a form
type FormSubmissionStats struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"byStatus"`
}

// Transaction support can be added later if needed
// For now, the repository works with the main database connection