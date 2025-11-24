package backupauth

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated backupauth plugin

// Plugin implements the backupauth plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new backupauth plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "backupauth"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// StartRecovery StartRecovery handles POST /recovery/start
func (p *Plugin) StartRecovery(ctx context.Context, req *authsome.StartRecoveryRequest) (*authsome.StartRecoveryResponse, error) {
	path := "/recovery/start"
	var result authsome.StartRecoveryResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ContinueRecovery ContinueRecovery handles POST /recovery/continue
func (p *Plugin) ContinueRecovery(ctx context.Context, req *authsome.ContinueRecoveryRequest) (*authsome.ContinueRecoveryResponse, error) {
	path := "/recovery/continue"
	var result authsome.ContinueRecoveryResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CompleteRecovery CompleteRecovery handles POST /recovery/complete
func (p *Plugin) CompleteRecovery(ctx context.Context, req *authsome.CompleteRecoveryRequest) (*authsome.CompleteRecoveryResponse, error) {
	path := "/recovery/complete"
	var result authsome.CompleteRecoveryResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CancelRecovery CancelRecovery handles POST /recovery/cancel
func (p *Plugin) CancelRecovery(ctx context.Context, req *authsome.CancelRecoveryRequest) (*authsome.CancelRecoveryResponse, error) {
	path := "/recovery/cancel"
	var result authsome.CancelRecoveryResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateRecoveryCodes GenerateRecoveryCodes handles POST /recovery-codes/generate
func (p *Plugin) GenerateRecoveryCodes(ctx context.Context, req *authsome.GenerateRecoveryCodesRequest) (*authsome.GenerateRecoveryCodesResponse, error) {
	path := "/recovery-codes/generate"
	var result authsome.GenerateRecoveryCodesResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// VerifyRecoveryCode VerifyRecoveryCode handles POST /recovery-codes/verify
func (p *Plugin) VerifyRecoveryCode(ctx context.Context, req *authsome.VerifyRecoveryCodeRequest) (*authsome.VerifyRecoveryCodeResponse, error) {
	path := "/recovery-codes/verify"
	var result authsome.VerifyRecoveryCodeResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SetupSecurityQuestions SetupSecurityQuestions handles POST /security-questions/setup
func (p *Plugin) SetupSecurityQuestions(ctx context.Context, req *authsome.SetupSecurityQuestionsRequest) (*authsome.SetupSecurityQuestionsResponse, error) {
	path := "/security-questions/setup"
	var result authsome.SetupSecurityQuestionsResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSecurityQuestions GetSecurityQuestions handles POST /security-questions/get
func (p *Plugin) GetSecurityQuestions(ctx context.Context, req *authsome.GetSecurityQuestionsRequest) (*authsome.GetSecurityQuestionsResponse, error) {
	path := "/security-questions/get"
	var result authsome.GetSecurityQuestionsResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// VerifySecurityAnswers VerifySecurityAnswers handles POST /security-questions/verify
func (p *Plugin) VerifySecurityAnswers(ctx context.Context, req *authsome.VerifySecurityAnswersRequest) (*authsome.VerifySecurityAnswersResponse, error) {
	path := "/security-questions/verify"
	var result authsome.VerifySecurityAnswersResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AddTrustedContact AddTrustedContact handles POST /trusted-contacts/add
func (p *Plugin) AddTrustedContact(ctx context.Context, req *authsome.AddTrustedContactRequest) (*authsome.AddTrustedContactResponse, error) {
	path := "/trusted-contacts/add"
	var result authsome.AddTrustedContactResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTrustedContacts ListTrustedContacts handles GET /trusted-contacts
func (p *Plugin) ListTrustedContacts(ctx context.Context) (*authsome.ListTrustedContactsResponse, error) {
	path := "/trusted-contacts"
	var result authsome.ListTrustedContactsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// VerifyTrustedContact VerifyTrustedContact handles POST /trusted-contacts/verify
func (p *Plugin) VerifyTrustedContact(ctx context.Context, req *authsome.VerifyTrustedContactRequest) (*authsome.VerifyTrustedContactResponse, error) {
	path := "/trusted-contacts/verify"
	var result authsome.VerifyTrustedContactResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RequestTrustedContactVerification RequestTrustedContactVerification handles POST /trusted-contacts/request-verification
func (p *Plugin) RequestTrustedContactVerification(ctx context.Context, req *authsome.RequestTrustedContactVerificationRequest) (*authsome.RequestTrustedContactVerificationResponse, error) {
	path := "/trusted-contacts/request-verification"
	var result authsome.RequestTrustedContactVerificationResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RemoveTrustedContact RemoveTrustedContact handles DELETE /trusted-contacts/:id
func (p *Plugin) RemoveTrustedContact(ctx context.Context) (*authsome.RemoveTrustedContactResponse, error) {
	path := "/trusted-contacts/:id"
	var result authsome.RemoveTrustedContactResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SendVerificationCode SendVerificationCode handles POST /verification/send
func (p *Plugin) SendVerificationCode(ctx context.Context, req *authsome.SendVerificationCodeRequest) (*authsome.SendVerificationCodeResponse, error) {
	path := "/verification/send"
	var result authsome.SendVerificationCodeResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// VerifyCode VerifyCode handles POST /verification/verify
func (p *Plugin) VerifyCode(ctx context.Context, req *authsome.VerifyCodeRequest) (*authsome.VerifyCodeResponse, error) {
	path := "/verification/verify"
	var result authsome.VerifyCodeResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ScheduleVideoSession ScheduleVideoSession handles POST /video/schedule
func (p *Plugin) ScheduleVideoSession(ctx context.Context, req *authsome.ScheduleVideoSessionRequest) (*authsome.ScheduleVideoSessionResponse, error) {
	path := "/video/schedule"
	var result authsome.ScheduleVideoSessionResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// StartVideoSession StartVideoSession handles POST /video/start
func (p *Plugin) StartVideoSession(ctx context.Context, req *authsome.StartVideoSessionRequest) (*authsome.StartVideoSessionResponse, error) {
	path := "/video/start"
	var result authsome.StartVideoSessionResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CompleteVideoSession CompleteVideoSession handles POST /video/complete (admin)
func (p *Plugin) CompleteVideoSession(ctx context.Context, req *authsome.CompleteVideoSessionRequest) (*authsome.CompleteVideoSessionResponse, error) {
	path := "/video/complete"
	var result authsome.CompleteVideoSessionResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UploadDocument UploadDocument handles POST /documents/upload
func (p *Plugin) UploadDocument(ctx context.Context, req *authsome.UploadDocumentRequest) (*authsome.UploadDocumentResponse, error) {
	path := "/documents/upload"
	var result authsome.UploadDocumentResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDocumentVerification GetDocumentVerification handles GET /documents/:id
func (p *Plugin) GetDocumentVerification(ctx context.Context) (*authsome.GetDocumentVerificationResponse, error) {
	path := "/documents/:id"
	var result authsome.GetDocumentVerificationResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ReviewDocument ReviewDocument handles POST /documents/:id/review (admin)
func (p *Plugin) ReviewDocument(ctx context.Context, req *authsome.ReviewDocumentRequest) (*authsome.ReviewDocumentResponse, error) {
	path := "/documents/:id/review"
	var result authsome.ReviewDocumentResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRecoverySessions ListRecoverySessions handles GET /admin/sessions (admin)
func (p *Plugin) ListRecoverySessions(ctx context.Context) error {
	path := "/sessions"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ApproveRecovery ApproveRecovery handles POST /admin/sessions/:id/approve (admin)
func (p *Plugin) ApproveRecovery(ctx context.Context, req *authsome.ApproveRecoveryRequest) (*authsome.ApproveRecoveryResponse, error) {
	path := "/sessions/:id/approve"
	var result authsome.ApproveRecoveryResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RejectRecovery RejectRecovery handles POST /admin/sessions/:id/reject (admin)
func (p *Plugin) RejectRecovery(ctx context.Context, req *authsome.RejectRecoveryRequest) (*authsome.RejectRecoveryResponse, error) {
	path := "/sessions/:id/reject"
	var result authsome.RejectRecoveryResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRecoveryStats GetRecoveryStats handles GET /admin/stats (admin)
func (p *Plugin) GetRecoveryStats(ctx context.Context) error {
	path := "/stats"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetRecoveryConfig GetRecoveryConfig handles GET /admin/config (admin)
func (p *Plugin) GetRecoveryConfig(ctx context.Context) error {
	path := "/config"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateRecoveryConfig UpdateRecoveryConfig handles PUT /admin/config (admin)
func (p *Plugin) UpdateRecoveryConfig(ctx context.Context, req *authsome.UpdateRecoveryConfigRequest) (*authsome.UpdateRecoveryConfigResponse, error) {
	path := "/config"
	var result authsome.UpdateRecoveryConfigResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// HealthCheck HealthCheck handles GET /health
func (p *Plugin) HealthCheck(ctx context.Context) error {
	path := "/health"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

