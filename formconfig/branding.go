package formconfig

import (
	"time"

	"github.com/xraph/authsome/id"
)

// BrandingConfig defines per-organization styling overrides for auth pages.
type BrandingConfig struct {
	ID              id.BrandingConfigID `json:"id"`
	OrgID           id.OrgID            `json:"org_id"`
	AppID           id.AppID            `json:"app_id"`
	LogoURL         string              `json:"logo_url,omitempty"`
	PrimaryColor    string              `json:"primary_color,omitempty"`
	BackgroundColor string              `json:"background_color,omitempty"`
	AccentColor     string              `json:"accent_color,omitempty"`
	FontFamily      string              `json:"font_family,omitempty"`
	CustomCSS       string              `json:"custom_css,omitempty"`
	CompanyName     string              `json:"company_name,omitempty"`
	Tagline         string              `json:"tagline,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}
