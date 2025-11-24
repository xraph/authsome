package compliance

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated compliance plugin

// Plugin implements the compliance plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new compliance plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "compliance"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// CreateProfile CreateProfile creates a new compliance profile
POST /auth/compliance/profiles
func (p *Plugin) CreateProfile(ctx context.Context, req *authsome.CreateProfileRequest) error {
	path := "/profiles"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// CreateProfileFromTemplate CreateProfileFromTemplate creates a profile from a template
POST /auth/compliance/profiles/from-template
func (p *Plugin) CreateProfileFromTemplate(ctx context.Context, req *authsome.CreateProfileFromTemplateRequest) error {
	path := "/profiles/from-template"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetProfile GetProfile retrieves a compliance profile
GET /auth/compliance/profiles/:id
func (p *Plugin) GetProfile(ctx context.Context) error {
	path := "/profiles/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetAppProfile GetAppProfile retrieves the compliance profile for an app
GET /auth/compliance/apps/:appId/profile
func (p *Plugin) GetAppProfile(ctx context.Context) error {
	path := "/apps/:appId/profile"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateProfile UpdateProfile updates a compliance profile
PUT /auth/compliance/profiles/:id
func (p *Plugin) UpdateProfile(ctx context.Context, req *authsome.UpdateProfileRequest) error {
	path := "/profiles/:id"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// DeleteProfile DeleteProfile deletes a compliance profile
DELETE /auth/compliance/profiles/:id
func (p *Plugin) DeleteProfile(ctx context.Context) error {
	path := "/profiles/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// GetComplianceStatus GetComplianceStatus gets overall compliance status for an app
GET /auth/compliance/apps/:appId/status
func (p *Plugin) GetComplianceStatus(ctx context.Context) error {
	path := "/apps/:appId/status"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetDashboard GetDashboard gets compliance dashboard data
GET /auth/compliance/apps/:appId/dashboard
func (p *Plugin) GetDashboard(ctx context.Context) error {
	path := "/apps/:appId/dashboard"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// RunCheck RunCheck executes a compliance check
POST /auth/compliance/profiles/:profileId/checks
func (p *Plugin) RunCheck(ctx context.Context, req *authsome.RunCheckRequest) error {
	path := "/profiles/:profileId/checks"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListChecks ListChecks lists compliance checks
GET /auth/compliance/profiles/:profileId/checks
func (p *Plugin) ListChecks(ctx context.Context) error {
	path := "/profiles/:profileId/checks"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetCheck GetCheck retrieves a compliance check
GET /auth/compliance/checks/:id
func (p *Plugin) GetCheck(ctx context.Context) error {
	path := "/checks/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListViolations ListViolations lists compliance violations
GET /auth/compliance/apps/:appId/violations
func (p *Plugin) ListViolations(ctx context.Context) error {
	path := "/apps/:appId/violations"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetViolation GetViolation retrieves a compliance violation
GET /auth/compliance/violations/:id
func (p *Plugin) GetViolation(ctx context.Context) error {
	path := "/violations/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ResolveViolation ResolveViolation resolves a compliance violation
PUT /auth/compliance/violations/:id/resolve
func (p *Plugin) ResolveViolation(ctx context.Context) error {
	path := "/violations/:id/resolve"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// GenerateReport GenerateReport generates a compliance report
POST /auth/compliance/apps/:appId/reports
func (p *Plugin) GenerateReport(ctx context.Context, req *authsome.GenerateReportRequest) error {
	path := "/apps/:appId/reports"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListReports ListReports lists compliance reports
GET /auth/compliance/apps/:appId/reports
func (p *Plugin) ListReports(ctx context.Context) error {
	path := "/apps/:appId/reports"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetReport GetReport retrieves a compliance report
GET /auth/compliance/reports/:id
func (p *Plugin) GetReport(ctx context.Context) error {
	path := "/reports/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// DownloadReport DownloadReport downloads a compliance report file
GET /auth/compliance/reports/:id/download
func (p *Plugin) DownloadReport(ctx context.Context) error {
	path := "/reports/:id/download"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// CreateEvidence CreateEvidence creates compliance evidence
POST /auth/compliance/apps/:appId/evidence
func (p *Plugin) CreateEvidence(ctx context.Context, req *authsome.CreateEvidenceRequest) error {
	path := "/apps/:appId/evidence"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListEvidence ListEvidence lists compliance evidence
GET /auth/compliance/apps/:appId/evidence
func (p *Plugin) ListEvidence(ctx context.Context) error {
	path := "/apps/:appId/evidence"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetEvidence GetEvidence retrieves compliance evidence
GET /auth/compliance/evidence/:id
func (p *Plugin) GetEvidence(ctx context.Context) error {
	path := "/evidence/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// DeleteEvidence DeleteEvidence deletes compliance evidence
DELETE /auth/compliance/evidence/:id
func (p *Plugin) DeleteEvidence(ctx context.Context) error {
	path := "/evidence/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// CreatePolicy CreatePolicy creates a compliance policy
POST /auth/compliance/apps/:appId/policies
func (p *Plugin) CreatePolicy(ctx context.Context, req *authsome.CreatePolicyRequest) error {
	path := "/apps/:appId/policies"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListPolicies ListPolicies lists compliance policies
GET /auth/compliance/apps/:appId/policies
func (p *Plugin) ListPolicies(ctx context.Context) error {
	path := "/apps/:appId/policies"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetPolicy GetPolicy retrieves a compliance policy
GET /auth/compliance/policies/:id
func (p *Plugin) GetPolicy(ctx context.Context) error {
	path := "/policies/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdatePolicy UpdatePolicy updates a compliance policy
PUT /auth/compliance/policies/:id
func (p *Plugin) UpdatePolicy(ctx context.Context, req *authsome.UpdatePolicyRequest) error {
	path := "/policies/:id"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// DeletePolicy DeletePolicy deletes a compliance policy
DELETE /auth/compliance/policies/:id
func (p *Plugin) DeletePolicy(ctx context.Context) error {
	path := "/policies/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// CreateTraining CreateTraining creates a training record
POST /auth/compliance/apps/:appId/training
func (p *Plugin) CreateTraining(ctx context.Context, req *authsome.CreateTrainingRequest) error {
	path := "/apps/:appId/training"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListTraining ListTraining lists training records
GET /auth/compliance/apps/:appId/training
func (p *Plugin) ListTraining(ctx context.Context) error {
	path := "/apps/:appId/training"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetUserTraining GetUserTraining gets training status for a user
GET /auth/compliance/users/:userId/training
func (p *Plugin) GetUserTraining(ctx context.Context) error {
	path := "/users/:userId/training"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// CompleteTraining CompleteTraining marks training as completed
PUT /auth/compliance/training/:id/complete
func (p *Plugin) CompleteTraining(ctx context.Context, req *authsome.CompleteTrainingRequest) error {
	path := "/training/:id/complete"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// ListTemplates ListTemplates lists available compliance templates
GET /auth/compliance/templates
func (p *Plugin) ListTemplates(ctx context.Context) error {
	path := "/templates"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetTemplate GetTemplate retrieves a compliance template
GET /auth/compliance/templates/:standard
func (p *Plugin) GetTemplate(ctx context.Context) error {
	path := "/templates/:standard"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

