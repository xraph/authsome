package bridge

import (
	"context"
	"errors"
	"time"
)

// ErrHeraldNotAvailable is returned when Herald is not configured.
var ErrHeraldNotAvailable = errors.New("authsome: herald not available")

// Herald is the bridge interface for the Herald notification system.
// It provides a unified API for sending notifications across multiple channels.
type Herald interface {
	// Send sends a notification via a specific channel.
	Send(ctx context.Context, req *HeraldSendRequest) error
	// Notify sends a notification across multiple channels using a template.
	Notify(ctx context.Context, req *HeraldNotifyRequest) error
}

// HeraldSendRequest describes a notification to send on a single channel.
type HeraldSendRequest struct {
	AppID    string            `json:"app_id"`
	EnvID    string            `json:"env_id,omitempty"`
	OrgID    string            `json:"org_id,omitempty"`
	UserID   string            `json:"user_id,omitempty"`
	Channel  string            `json:"channel"`
	Template string            `json:"template"`
	Locale   string            `json:"locale,omitempty"`
	To       []string          `json:"to"`
	Data     map[string]any    `json:"data,omitempty"`
	Async    bool              `json:"async,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// HeraldNotifyRequest describes a multi-channel notification using a template.
type HeraldNotifyRequest struct {
	AppID    string            `json:"app_id"`
	EnvID    string            `json:"env_id,omitempty"`
	OrgID    string            `json:"org_id,omitempty"`
	UserID   string            `json:"user_id,omitempty"`
	Template string            `json:"template"`
	Locale   string            `json:"locale,omitempty"`
	To       []string          `json:"to"`
	Data     map[string]any    `json:"data,omitempty"`
	Channels []string          `json:"channels"`
	Async    bool              `json:"async,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ──────────────────────────────────────────────────
// Template management types
// ──────────────────────────────────────────────────

// HeraldTemplate represents a notification template with locale-specific versions.
type HeraldTemplate struct {
	ID        string                   `json:"id"`
	AppID     string                   `json:"app_id"`
	Slug      string                   `json:"slug"`
	Name      string                   `json:"name"`
	Channel   string                   `json:"channel"`
	Category  string                   `json:"category"`
	Variables []HeraldTemplateVariable `json:"variables,omitempty"`
	Versions  []HeraldTemplateVersion  `json:"versions,omitempty"`
	IsSystem  bool                     `json:"is_system"`
	Enabled   bool                     `json:"enabled"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// HeraldTemplateVersion represents a locale-specific content version.
type HeraldTemplateVersion struct {
	ID         string    `json:"id"`
	TemplateID string    `json:"template_id"`
	Locale     string    `json:"locale"`
	Subject    string    `json:"subject,omitempty"`
	HTML       string    `json:"html,omitempty"`
	Text       string    `json:"text,omitempty"`
	Title      string    `json:"title,omitempty"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// HeraldTemplateVariable describes an expected template variable.
type HeraldTemplateVariable struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// HeraldRenderedContent holds fully rendered template output.
type HeraldRenderedContent struct {
	Subject string `json:"subject,omitempty"`
	HTML    string `json:"html,omitempty"`
	Text    string `json:"text,omitempty"`
	Title   string `json:"title,omitempty"`
}

// Template category constants.
const (
	HeraldCategoryAuth          = "auth"
	HeraldCategoryTransactional = "transactional"
	HeraldCategoryMarketing     = "marketing"
	HeraldCategorySystem        = "system"
)

// HeraldTemplateManager provides template CRUD, preview, and test send operations.
// Implementations can be discovered via type assertion on a Herald bridge.
type HeraldTemplateManager interface {
	// Template CRUD
	ListTemplates(ctx context.Context, appID string) ([]*HeraldTemplate, error)
	GetTemplate(ctx context.Context, templateID string) (*HeraldTemplate, error)
	CreateTemplate(ctx context.Context, t *HeraldTemplate) error
	UpdateTemplate(ctx context.Context, t *HeraldTemplate) error
	DeleteTemplate(ctx context.Context, templateID string) error

	// Version management
	CreateVersion(ctx context.Context, v *HeraldTemplateVersion) error
	UpdateVersion(ctx context.Context, v *HeraldTemplateVersion) error
	DeleteVersion(ctx context.Context, versionID string) error

	// Preview renders a template with sample data without sending.
	RenderTemplate(ctx context.Context, templateID string, locale string, data map[string]any) (*HeraldRenderedContent, error)

	// TestSend sends a test notification to a recipient.
	TestSend(ctx context.Context, req *HeraldSendRequest) error

	// SeedDefaultTemplates creates default system templates if they don't exist.
	// Existing templates (including customised ones) are left untouched.
	SeedDefaultTemplates(ctx context.Context, appID string) error

	// ResetDefaultTemplates deletes all system templates and re-seeds defaults.
	// Custom (non-system) templates are preserved.
	ResetDefaultTemplates(ctx context.Context, appID string) error
}
