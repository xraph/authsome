package consent

import (
	"context"
	"net/url"

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
func (p *Plugin) CreateConsent(ctx context.Context, req *authsome.CreateConsentRequest) (*authsome.CreateConsentResponse, error) {
	path := "/records"
	var result authsome.CreateConsentResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConsent GetConsent handles GET /consent/records/:id
func (p *Plugin) GetConsent(ctx context.Context, id xid.ID) (*authsome.GetConsentResponse, error) {
	path := "/records/:id"
	var result authsome.GetConsentResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateConsent UpdateConsent handles PATCH /consent/records/:id
func (p *Plugin) UpdateConsent(ctx context.Context, req *authsome.UpdateConsentRequest, id xid.ID) (*authsome.UpdateConsentResponse, error) {
	path := "/records/:id"
	var result authsome.UpdateConsentResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeConsent RevokeConsent handles POST /consent/records/:id/revoke
func (p *Plugin) RevokeConsent(ctx context.Context, req *authsome.RevokeConsentRequest, id xid.ID) (*authsome.RevokeConsentResponse, error) {
	path := "/revoke/:id"
	var result authsome.RevokeConsentResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateConsentPolicy CreateConsentPolicy handles POST /consent/policies
func (p *Plugin) CreateConsentPolicy(ctx context.Context, req *authsome.CreateConsentPolicyRequest) (*authsome.CreateConsentPolicyResponse, error) {
	path := "/policies"
	var result authsome.CreateConsentPolicyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConsentPolicy GetConsentPolicy handles GET /consent/policies/:id
func (p *Plugin) GetConsentPolicy(ctx context.Context, id xid.ID) (*authsome.GetConsentPolicyResponse, error) {
	path := "/policies/:id"
	var result authsome.GetConsentPolicyResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RecordCookieConsent RecordCookieConsent handles POST /consent/cookies
func (p *Plugin) RecordCookieConsent(ctx context.Context, req *authsome.RecordCookieConsentRequest) (*authsome.RecordCookieConsentResponse, error) {
	path := "/cookies"
	var result authsome.RecordCookieConsentResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCookieConsent GetCookieConsent handles GET /consent/cookies
func (p *Plugin) GetCookieConsent(ctx context.Context) (*authsome.GetCookieConsentResponse, error) {
	path := "/cookies"
	var result authsome.GetCookieConsentResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RequestDataExport RequestDataExport handles POST /consent/data-exports
func (p *Plugin) RequestDataExport(ctx context.Context, req *authsome.RequestDataExportRequest) (*authsome.RequestDataExportResponse, error) {
	path := "/export"
	var result authsome.RequestDataExportResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDataExport GetDataExport handles GET /consent/data-exports/:id
func (p *Plugin) GetDataExport(ctx context.Context, id xid.ID) (*authsome.GetDataExportResponse, error) {
	path := "/export/:id"
	var result authsome.GetDataExportResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DownloadDataExport DownloadDataExport handles GET /consent/data-exports/:id/download
func (p *Plugin) DownloadDataExport(ctx context.Context, id xid.ID) (*authsome.DownloadDataExportResponse, error) {
	path := "/export/:id/download"
	var result authsome.DownloadDataExportResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RequestDataDeletion RequestDataDeletion handles POST /consent/data-deletions
func (p *Plugin) RequestDataDeletion(ctx context.Context, req *authsome.RequestDataDeletionRequest) (*authsome.RequestDataDeletionResponse, error) {
	path := "/deletion"
	var result authsome.RequestDataDeletionResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDataDeletion GetDataDeletion handles GET /consent/data-deletions/:id
func (p *Plugin) GetDataDeletion(ctx context.Context, id xid.ID) (*authsome.GetDataDeletionResponse, error) {
	path := "/deletion/:id"
	var result authsome.GetDataDeletionResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ApproveDeletionRequest ApproveDeletionRequest handles POST /consent/data-deletions/:id/approve (Admin only)
func (p *Plugin) ApproveDeletionRequest(ctx context.Context, id xid.ID) (*authsome.ApproveDeletionRequestResponse, error) {
	path := "/deletion/:id/approve"
	var result authsome.ApproveDeletionRequestResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPrivacySettings GetPrivacySettings handles GET /consent/privacy-settings
func (p *Plugin) GetPrivacySettings(ctx context.Context) (*authsome.GetPrivacySettingsResponse, error) {
	path := "/settings"
	var result authsome.GetPrivacySettingsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePrivacySettings UpdatePrivacySettings handles PATCH /consent/privacy-settings (Admin only)
func (p *Plugin) UpdatePrivacySettings(ctx context.Context, req *authsome.UpdatePrivacySettingsRequest) (*authsome.UpdatePrivacySettingsResponse, error) {
	path := "/settings"
	var result authsome.UpdatePrivacySettingsResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConsentAuditLogs GetConsentAuditLogs handles GET /consent/audit-logs
func (p *Plugin) GetConsentAuditLogs(ctx context.Context) (*authsome.GetConsentAuditLogsResponse, error) {
	path := "/audit"
	var result authsome.GetConsentAuditLogsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateConsentReport GenerateConsentReport handles GET /consent/reports
func (p *Plugin) GenerateConsentReport(ctx context.Context) (*authsome.GenerateConsentReportResponse, error) {
	path := "/reports"
	var result authsome.GenerateConsentReportResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

