package consent

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated consent plugin

// Plugin implements the consent plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new consent plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "consent"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// CreateConsent CreateConsent handles POST /consent/records
func (p *Plugin) CreateConsent(ctx context.Context, req *authsome.CreateConsentRequest) error {
	path := "/records"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetConsent GetConsent handles GET /consent/records/:id
func (p *Plugin) GetConsent(ctx context.Context) error {
	path := "/records/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateConsent UpdateConsent handles PATCH /consent/records/:id
func (p *Plugin) UpdateConsent(ctx context.Context, req *authsome.UpdateConsentRequest) error {
	path := "/records/:id"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// RevokeConsent RevokeConsent handles POST /consent/records/:id/revoke
func (p *Plugin) RevokeConsent(ctx context.Context, req *authsome.RevokeConsentRequest) error {
	path := "/revoke/:id"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// CreateConsentPolicy CreateConsentPolicy handles POST /consent/policies
func (p *Plugin) CreateConsentPolicy(ctx context.Context, req *authsome.CreateConsentPolicyRequest) error {
	path := "/policies"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetConsentPolicy GetConsentPolicy handles GET /consent/policies/:id
func (p *Plugin) GetConsentPolicy(ctx context.Context) error {
	path := "/policies/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// RecordCookieConsent RecordCookieConsent handles POST /consent/cookies
func (p *Plugin) RecordCookieConsent(ctx context.Context, req *authsome.RecordCookieConsentRequest) error {
	path := "/cookies"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetCookieConsent GetCookieConsent handles GET /consent/cookies
func (p *Plugin) GetCookieConsent(ctx context.Context) error {
	path := "/cookies"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// RequestDataExport RequestDataExport handles POST /consent/data-exports
func (p *Plugin) RequestDataExport(ctx context.Context, req *authsome.RequestDataExportRequest) error {
	path := "/export"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetDataExport GetDataExport handles GET /consent/data-exports/:id
func (p *Plugin) GetDataExport(ctx context.Context) error {
	path := "/export/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// DownloadDataExport DownloadDataExport handles GET /consent/data-exports/:id/download
func (p *Plugin) DownloadDataExport(ctx context.Context) error {
	path := "/export/:id/download"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// RequestDataDeletion RequestDataDeletion handles POST /consent/data-deletions
func (p *Plugin) RequestDataDeletion(ctx context.Context, req *authsome.RequestDataDeletionRequest) error {
	path := "/deletion"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetDataDeletion GetDataDeletion handles GET /consent/data-deletions/:id
func (p *Plugin) GetDataDeletion(ctx context.Context) error {
	path := "/deletion/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ApproveDeletionRequest ApproveDeletionRequest handles POST /consent/data-deletions/:id/approve (Admin only)
func (p *Plugin) ApproveDeletionRequest(ctx context.Context) error {
	path := "/deletion/:id/approve"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetPrivacySettings GetPrivacySettings handles GET /consent/privacy-settings
func (p *Plugin) GetPrivacySettings(ctx context.Context) error {
	path := "/settings"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdatePrivacySettings UpdatePrivacySettings handles PATCH /consent/privacy-settings (Admin only)
func (p *Plugin) UpdatePrivacySettings(ctx context.Context, req *authsome.UpdatePrivacySettingsRequest) error {
	path := "/settings"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// GetConsentAuditLogs GetConsentAuditLogs handles GET /consent/audit-logs
func (p *Plugin) GetConsentAuditLogs(ctx context.Context) error {
	path := "/audit"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GenerateConsentReport GenerateConsentReport handles GET /consent/reports
func (p *Plugin) GenerateConsentReport(ctx context.Context) error {
	path := "/reports"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

