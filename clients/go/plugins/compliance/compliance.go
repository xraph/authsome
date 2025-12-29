package compliance

import (
	"context"
	"net/url"

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
func (p *Plugin) CreateProfile(ctx context.Context, req *authsome.CreateProfileRequest) (*authsome.CreateProfileResponse, error) {
	path := "/profiles"
	var result authsome.CreateProfileResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateProfileFromTemplate CreateProfileFromTemplate creates a profile from a template
POST /auth/compliance/profiles/from-template
func (p *Plugin) CreateProfileFromTemplate(ctx context.Context, req *authsome.CreateProfileFromTemplateRequest) (*authsome.CreateProfileFromTemplateResponse, error) {
	path := "/profiles/from-template"
	var result authsome.CreateProfileFromTemplateResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProfile GetProfile retrieves a compliance profile
GET /auth/compliance/profiles/:id
func (p *Plugin) GetProfile(ctx context.Context, id xid.ID) (*authsome.GetProfileResponse, error) {
	path := "/profiles/:id"
	var result authsome.GetProfileResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAppProfile GetAppProfile retrieves the compliance profile for an app
GET /auth/compliance/apps/:appId/profile
func (p *Plugin) GetAppProfile(ctx context.Context, appId xid.ID) (*authsome.GetAppProfileResponse, error) {
	path := "/apps/:appId/profile"
	var result authsome.GetAppProfileResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateProfile UpdateProfile updates a compliance profile
PUT /auth/compliance/profiles/:id
func (p *Plugin) UpdateProfile(ctx context.Context, req *authsome.UpdateProfileRequest, id xid.ID) (*authsome.UpdateProfileResponse, error) {
	path := "/profiles/:id"
	var result authsome.UpdateProfileResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteProfile DeleteProfile deletes a compliance profile
DELETE /auth/compliance/profiles/:id
func (p *Plugin) DeleteProfile(ctx context.Context, id xid.ID) (*authsome.DeleteProfileResponse, error) {
	path := "/profiles/:id"
	var result authsome.DeleteProfileResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetComplianceStatus GetComplianceStatus gets overall compliance status for an app
GET /auth/compliance/apps/:appId/status
func (p *Plugin) GetComplianceStatus(ctx context.Context, appId xid.ID) (*authsome.GetComplianceStatusResponse, error) {
	path := "/apps/:appId/status"
	var result authsome.GetComplianceStatusResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDashboard GetDashboard gets compliance dashboard data
GET /auth/compliance/apps/:appId/dashboard
func (p *Plugin) GetDashboard(ctx context.Context, appId xid.ID) (*authsome.GetDashboardResponse, error) {
	path := "/apps/:appId/dashboard"
	var result authsome.GetDashboardResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RunCheck RunCheck executes a compliance check
POST /auth/compliance/profiles/:profileId/checks
func (p *Plugin) RunCheck(ctx context.Context, req *authsome.RunCheckRequest, profileId xid.ID) (*authsome.RunCheckResponse, error) {
	path := "/profiles/:profileId/checks"
	var result authsome.RunCheckResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListChecks ListChecks lists compliance checks
GET /auth/compliance/profiles/:profileId/checks
func (p *Plugin) ListChecks(ctx context.Context, profileId xid.ID) (*authsome.ListChecksResponse, error) {
	path := "/profiles/:profileId/checks"
	var result authsome.ListChecksResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCheck GetCheck retrieves a compliance check
GET /auth/compliance/checks/:id
func (p *Plugin) GetCheck(ctx context.Context, id xid.ID) (*authsome.GetCheckResponse, error) {
	path := "/checks/:id"
	var result authsome.GetCheckResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListViolations ListViolations lists compliance violations
GET /auth/compliance/apps/:appId/violations
func (p *Plugin) ListViolations(ctx context.Context, appId xid.ID) (*authsome.ListViolationsResponse, error) {
	path := "/apps/:appId/violations"
	var result authsome.ListViolationsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetViolation GetViolation retrieves a compliance violation
GET /auth/compliance/violations/:id
func (p *Plugin) GetViolation(ctx context.Context, id xid.ID) (*authsome.GetViolationResponse, error) {
	path := "/violations/:id"
	var result authsome.GetViolationResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ResolveViolation ResolveViolation resolves a compliance violation
PUT /auth/compliance/violations/:id/resolve
func (p *Plugin) ResolveViolation(ctx context.Context, req *authsome.ResolveViolationRequest, id xid.ID) (*authsome.ResolveViolationResponse, error) {
	path := "/violations/:id/resolve"
	var result authsome.ResolveViolationResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateReport GenerateReport generates a compliance report
POST /auth/compliance/apps/:appId/reports
func (p *Plugin) GenerateReport(ctx context.Context, req *authsome.GenerateReportRequest, appId xid.ID) (*authsome.GenerateReportResponse, error) {
	path := "/apps/:appId/reports"
	var result authsome.GenerateReportResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListReports ListReports lists compliance reports
GET /auth/compliance/apps/:appId/reports
func (p *Plugin) ListReports(ctx context.Context, appId xid.ID) (*authsome.ListReportsResponse, error) {
	path := "/apps/:appId/reports"
	var result authsome.ListReportsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetReport GetReport retrieves a compliance report
GET /auth/compliance/reports/:id
func (p *Plugin) GetReport(ctx context.Context, id xid.ID) (*authsome.GetReportResponse, error) {
	path := "/reports/:id"
	var result authsome.GetReportResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DownloadReport DownloadReport downloads a compliance report file
GET /auth/compliance/reports/:id/download
func (p *Plugin) DownloadReport(ctx context.Context, id xid.ID) (*authsome.DownloadReportResponse, error) {
	path := "/reports/:id/download"
	var result authsome.DownloadReportResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateEvidence CreateEvidence creates compliance evidence
POST /auth/compliance/apps/:appId/evidence
func (p *Plugin) CreateEvidence(ctx context.Context, req *authsome.CreateEvidenceRequest, appId xid.ID) (*authsome.CreateEvidenceResponse, error) {
	path := "/apps/:appId/evidence"
	var result authsome.CreateEvidenceResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListEvidence ListEvidence lists compliance evidence
GET /auth/compliance/apps/:appId/evidence
func (p *Plugin) ListEvidence(ctx context.Context, appId xid.ID) (*authsome.ListEvidenceResponse, error) {
	path := "/apps/:appId/evidence"
	var result authsome.ListEvidenceResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetEvidence GetEvidence retrieves compliance evidence
GET /auth/compliance/evidence/:id
func (p *Plugin) GetEvidence(ctx context.Context, id xid.ID) (*authsome.GetEvidenceResponse, error) {
	path := "/evidence/:id"
	var result authsome.GetEvidenceResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteEvidence DeleteEvidence deletes compliance evidence
DELETE /auth/compliance/evidence/:id
func (p *Plugin) DeleteEvidence(ctx context.Context, id xid.ID) (*authsome.DeleteEvidenceResponse, error) {
	path := "/evidence/:id"
	var result authsome.DeleteEvidenceResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePolicy CreatePolicy creates a compliance policy
POST /auth/compliance/apps/:appId/policies
func (p *Plugin) CreatePolicy(ctx context.Context, req *authsome.CreatePolicyRequest, appId xid.ID) (*authsome.CreatePolicyResponse, error) {
	path := "/apps/:appId/policies"
	var result authsome.CreatePolicyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListPolicies ListPolicies lists compliance policies
GET /auth/compliance/apps/:appId/policies
func (p *Plugin) ListPolicies(ctx context.Context, appId xid.ID) (*authsome.ListPoliciesResponse, error) {
	path := "/apps/:appId/policies"
	var result authsome.ListPoliciesResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPolicy GetPolicy retrieves a compliance policy
GET /auth/compliance/policies/:id
func (p *Plugin) GetPolicy(ctx context.Context, id xid.ID) (*authsome.GetPolicyResponse, error) {
	path := "/policies/:id"
	var result authsome.GetPolicyResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePolicy UpdatePolicy updates a compliance policy
PUT /auth/compliance/policies/:id
func (p *Plugin) UpdatePolicy(ctx context.Context, req *authsome.UpdatePolicyRequest, id xid.ID) (*authsome.UpdatePolicyResponse, error) {
	path := "/policies/:id"
	var result authsome.UpdatePolicyResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePolicy DeletePolicy deletes a compliance policy
DELETE /auth/compliance/policies/:id
func (p *Plugin) DeletePolicy(ctx context.Context, id xid.ID) (*authsome.DeletePolicyResponse, error) {
	path := "/policies/:id"
	var result authsome.DeletePolicyResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTraining CreateTraining creates a training record
POST /auth/compliance/apps/:appId/training
func (p *Plugin) CreateTraining(ctx context.Context, req *authsome.CreateTrainingRequest, appId xid.ID) (*authsome.CreateTrainingResponse, error) {
	path := "/apps/:appId/training"
	var result authsome.CreateTrainingResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTraining ListTraining lists training records
GET /auth/compliance/apps/:appId/training
func (p *Plugin) ListTraining(ctx context.Context, appId xid.ID) (*authsome.ListTrainingResponse, error) {
	path := "/apps/:appId/training"
	var result authsome.ListTrainingResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetUserTraining GetUserTraining gets training status for a user
GET /auth/compliance/users/:userId/training
func (p *Plugin) GetUserTraining(ctx context.Context, userId xid.ID) (*authsome.GetUserTrainingResponse, error) {
	path := "/users/:userId/training"
	var result authsome.GetUserTrainingResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CompleteTraining CompleteTraining marks training as completed
PUT /auth/compliance/training/:id/complete
func (p *Plugin) CompleteTraining(ctx context.Context, req *authsome.CompleteTrainingRequest, id xid.ID) (*authsome.CompleteTrainingResponse, error) {
	path := "/training/:id/complete"
	var result authsome.CompleteTrainingResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTemplates ListTemplates lists available compliance templates
GET /auth/compliance/templates
func (p *Plugin) ListTemplates(ctx context.Context) (*authsome.ListTemplatesResponse, error) {
	path := "/templates"
	var result authsome.ListTemplatesResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTemplate GetTemplate retrieves a compliance template
GET /auth/compliance/templates/:standard
func (p *Plugin) GetTemplate(ctx context.Context, standard string) (*authsome.GetTemplateResponse, error) {
	path := "/templates/:standard"
	var result authsome.GetTemplateResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

