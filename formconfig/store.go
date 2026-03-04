package formconfig

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines persistence operations for form configurations.
type Store interface {
	CreateFormConfig(ctx context.Context, fc *FormConfig) error
	GetFormConfig(ctx context.Context, appID id.AppID, formType string) (*FormConfig, error)
	UpdateFormConfig(ctx context.Context, fc *FormConfig) error
	DeleteFormConfig(ctx context.Context, appID id.AppID, formType string) error
	ListFormConfigs(ctx context.Context, appID id.AppID) ([]*FormConfig, error)
}

// BrandingStore defines persistence operations for branding configurations.
type BrandingStore interface {
	GetBranding(ctx context.Context, orgID id.OrgID) (*BrandingConfig, error)
	SaveBranding(ctx context.Context, b *BrandingConfig) error
	DeleteBranding(ctx context.Context, orgID id.OrgID) error
}
