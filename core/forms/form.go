package forms

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Form represents a form configuration
type Form struct {
	ID             xid.ID                 `json:"id"`
	OrganizationID xid.ID                 `json:"organizationId"`
	Type           string                 `json:"type"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Schema         map[string]interface{} `json:"schema"`
	IsActive       bool                   `json:"isActive"`
	Version        int                    `json:"version"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

// FormField represents a form field configuration
type FormField struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Label       string                 `json:"label"`
	Placeholder string                 `json:"placeholder"`
	Required    bool                   `json:"required"`
	Validation  map[string]interface{} `json:"validation"`
	Options     []string               `json:"options,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FormSubmission represents a form submission
type FormSubmission struct {
	ID           xid.ID                 `json:"id"`
	FormSchemaID xid.ID                 `json:"formSchemaId"`
	UserID       *xid.ID                `json:"userId"`
	SessionID    *xid.ID                `json:"sessionId"`
	Data         map[string]interface{} `json:"data"`
	IPAddress    string                 `json:"ipAddress"`
	UserAgent    string                 `json:"userAgent"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// CreateFormRequest represents a request to create a new form
type CreateFormRequest struct {
	OrganizationID xid.ID                 `json:"organizationId"`
	Type           string                 `json:"type"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Schema         map[string]interface{} `json:"schema"`
}

// UpdateFormRequest represents a request to update a form
type UpdateFormRequest struct {
	ID          xid.ID                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
	IsActive    bool                   `json:"isActive"`
}

// SubmitFormRequest represents a request to submit a form
type SubmitFormRequest struct {
	FormSchemaID xid.ID                 `json:"formSchemaId"`
	UserID       *xid.ID                `json:"userId"`
	SessionID    *xid.ID                `json:"sessionId"`
	Data         map[string]interface{} `json:"data"`
	IPAddress    string                 `json:"ipAddress"`
	UserAgent    string                 `json:"userAgent"`
}

// ListFormsRequest represents a request to list forms
type ListFormsRequest struct {
	OrganizationID xid.ID `json:"organizationId"`
	Type           string `json:"type,omitempty"`
	Page           int    `json:"page"`
	PageSize       int    `json:"pageSize"`
}

// ListFormsResponse represents a response with forms list
type ListFormsResponse struct {
	Forms      []*Form `json:"forms"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalPages int     `json:"totalPages"`
}

// Repository defines the interface for form data access
type Repository interface {
	// Form management
	Create(ctx context.Context, form *schema.FormSchema) error
	GetByID(ctx context.Context, id xid.ID) (*schema.FormSchema, error)
	GetByOrganization(ctx context.Context, orgID xid.ID, formType string) (*schema.FormSchema, error)
	List(ctx context.Context, orgID xid.ID, formType string, page, pageSize int) ([]*schema.FormSchema, int, error)
	Update(ctx context.Context, form *schema.FormSchema) error
	Delete(ctx context.Context, id xid.ID) error

	// Form submissions
	CreateSubmission(ctx context.Context, submission *schema.FormSubmission) error
	GetSubmissionByID(ctx context.Context, id xid.ID) (*schema.FormSubmission, error)
	ListSubmissions(ctx context.Context, formSchemaID xid.ID, page, pageSize int) ([]*schema.FormSubmission, int, error)
	GetSubmissionsByUser(ctx context.Context, userID xid.ID, page, pageSize int) ([]*schema.FormSubmission, int, error)
}

// Config represents the forms configuration
type Config struct {
	DefaultFormType string                 `json:"defaultFormType"`
	MaxFieldCount   int                    `json:"maxFieldCount"`
	MaxFileSize     int64                  `json:"maxFileSize"`
	AllowedTypes    []string               `json:"allowedTypes"`
	ValidationRules map[string]interface{} `json:"validationRules"`
}

// DefaultConfig returns the default forms configuration
func DefaultConfig() Config {
	return Config{
		DefaultFormType: "signup",
		MaxFieldCount:   50,
		MaxFileSize:     10 * 1024 * 1024, // 10MB
		AllowedTypes:    []string{"signup", "signin", "profile", "contact", "survey"},
		ValidationRules: map[string]interface{}{
			"email": map[string]interface{}{
				"pattern": `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`,
			},
			"password": map[string]interface{}{
				"minLength": 8,
				"maxLength": 128,
			},
			"phone": map[string]interface{}{
				"pattern": `^\+?[1-9]\d{1,14}$`,
			},
		},
	}
}
