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

// StartRecoveryRequest is the request for StartRecovery
type StartRecoveryRequest struct {
	DeviceId string `json:"deviceId"`
	Email string `json:"email"`
	PreferredMethod authsome.RecoveryMethod `json:"preferredMethod"`
	UserId string `json:"userId"`
}

// StartRecoveryResponse is the response for StartRecovery
type StartRecoveryResponse struct {
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
	Code string `json:"code"`
}

// StartRecovery StartRecovery handles POST /recovery/start
func (p *Plugin) StartRecovery(ctx context.Context, req *StartRecoveryRequest) (*StartRecoveryResponse, error) {
	path := "/recovery/start"
	var result StartRecoveryResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// ContinueRecoveryRequest is the request for ContinueRecovery
type ContinueRecoveryRequest struct {
	Method authsome.RecoveryMethod `json:"method"`
	SessionId authsome.xid.ID `json:"sessionId"`
}

// ContinueRecoveryResponse is the response for ContinueRecovery
type ContinueRecoveryResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// ContinueRecovery ContinueRecovery handles POST /recovery/continue
func (p *Plugin) ContinueRecovery(ctx context.Context, req *ContinueRecoveryRequest) (*ContinueRecoveryResponse, error) {
	path := "/recovery/continue"
	var result ContinueRecoveryResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// CompleteRecoveryRequest is the request for CompleteRecovery
type CompleteRecoveryRequest struct {
	SessionId authsome.xid.ID `json:"sessionId"`
}

// CompleteRecoveryResponse is the response for CompleteRecovery
type CompleteRecoveryResponse struct {
	Error string `json:"error"`
	Message string `json:"message"`
	Code string `json:"code"`
	Details authsome. `json:"details"`
}

// CompleteRecovery CompleteRecovery handles POST /recovery/complete
func (p *Plugin) CompleteRecovery(ctx context.Context, req *CompleteRecoveryRequest) (*CompleteRecoveryResponse, error) {
	path := "/recovery/complete"
	var result CompleteRecoveryResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// CancelRecoveryRequest is the request for CancelRecovery
type CancelRecoveryRequest struct {
	Reason string `json:"reason"`
	SessionId authsome.xid.ID `json:"sessionId"`
}

// CancelRecoveryResponse is the response for CancelRecovery
type CancelRecoveryResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

// CancelRecovery CancelRecovery handles POST /recovery/cancel
func (p *Plugin) CancelRecovery(ctx context.Context, req *CancelRecoveryRequest) (*CancelRecoveryResponse, error) {
	path := "/recovery/cancel"
	var result CancelRecoveryResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// GenerateRecoveryCodesRequest is the request for GenerateRecoveryCodes
type GenerateRecoveryCodesRequest struct {
	Format string `json:"format"`
	Count int `json:"count"`
}

// GenerateRecoveryCodesResponse is the response for GenerateRecoveryCodes
type GenerateRecoveryCodesResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// GenerateRecoveryCodes GenerateRecoveryCodes handles POST /recovery-codes/generate
func (p *Plugin) GenerateRecoveryCodes(ctx context.Context, req *GenerateRecoveryCodesRequest) (*GenerateRecoveryCodesResponse, error) {
	path := "/recovery-codes/generate"
	var result GenerateRecoveryCodesResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// VerifyRecoveryCodeRequest is the request for VerifyRecoveryCode
type VerifyRecoveryCodeRequest struct {
	SessionId authsome.xid.ID `json:"sessionId"`
	Code string `json:"code"`
}

// VerifyRecoveryCodeResponse is the response for VerifyRecoveryCode
type VerifyRecoveryCodeResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// VerifyRecoveryCode VerifyRecoveryCode handles POST /recovery-codes/verify
func (p *Plugin) VerifyRecoveryCode(ctx context.Context, req *VerifyRecoveryCodeRequest) (*VerifyRecoveryCodeResponse, error) {
	path := "/recovery-codes/verify"
	var result VerifyRecoveryCodeResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// SetupSecurityQuestionsRequest is the request for SetupSecurityQuestions
type SetupSecurityQuestionsRequest struct {
	Questions authsome.[]SetupSecurityQuestionRequest `json:"questions"`
}

// SetupSecurityQuestionsResponse is the response for SetupSecurityQuestions
type SetupSecurityQuestionsResponse struct {
	Message string `json:"message"`
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
}

// SetupSecurityQuestions SetupSecurityQuestions handles POST /security-questions/setup
func (p *Plugin) SetupSecurityQuestions(ctx context.Context, req *SetupSecurityQuestionsRequest) (*SetupSecurityQuestionsResponse, error) {
	path := "/security-questions/setup"
	var result SetupSecurityQuestionsResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// GetSecurityQuestionsRequest is the request for GetSecurityQuestions
type GetSecurityQuestionsRequest struct {
	SessionId authsome.xid.ID `json:"sessionId"`
}

// GetSecurityQuestionsResponse is the response for GetSecurityQuestions
type GetSecurityQuestionsResponse struct {
	Message string `json:"message"`
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
}

// GetSecurityQuestions GetSecurityQuestions handles POST /security-questions/get
func (p *Plugin) GetSecurityQuestions(ctx context.Context, req *GetSecurityQuestionsRequest) (*GetSecurityQuestionsResponse, error) {
	path := "/security-questions/get"
	var result GetSecurityQuestionsResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// VerifySecurityAnswersRequest is the request for VerifySecurityAnswers
type VerifySecurityAnswersRequest struct {
	Answers authsome. `json:"answers"`
	SessionId authsome.xid.ID `json:"sessionId"`
}

// VerifySecurityAnswersResponse is the response for VerifySecurityAnswers
type VerifySecurityAnswersResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// VerifySecurityAnswers VerifySecurityAnswers handles POST /security-questions/verify
func (p *Plugin) VerifySecurityAnswers(ctx context.Context, req *VerifySecurityAnswersRequest) (*VerifySecurityAnswersResponse, error) {
	path := "/security-questions/verify"
	var result VerifySecurityAnswersResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// AddTrustedContactRequest is the request for AddTrustedContact
type AddTrustedContactRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Phone string `json:"phone"`
	Relationship string `json:"relationship"`
}

// AddTrustedContactResponse is the response for AddTrustedContact
type AddTrustedContactResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// AddTrustedContact AddTrustedContact handles POST /trusted-contacts/add
func (p *Plugin) AddTrustedContact(ctx context.Context, req *AddTrustedContactRequest) (*AddTrustedContactResponse, error) {
	path := "/trusted-contacts/add"
	var result AddTrustedContactResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// ListTrustedContactsResponse is the response for ListTrustedContacts
type ListTrustedContactsResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// ListTrustedContacts ListTrustedContacts handles GET /trusted-contacts
func (p *Plugin) ListTrustedContacts(ctx context.Context) (*ListTrustedContactsResponse, error) {
	path := "/trusted-contacts"
	var result ListTrustedContactsResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// VerifyTrustedContactRequest is the request for VerifyTrustedContact
type VerifyTrustedContactRequest struct {
	Token string `json:"token"`
}

// VerifyTrustedContactResponse is the response for VerifyTrustedContact
type VerifyTrustedContactResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// VerifyTrustedContact VerifyTrustedContact handles POST /trusted-contacts/verify
func (p *Plugin) VerifyTrustedContact(ctx context.Context, req *VerifyTrustedContactRequest) (*VerifyTrustedContactResponse, error) {
	path := "/trusted-contacts/verify"
	var result VerifyTrustedContactResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// RequestTrustedContactVerificationRequest is the request for RequestTrustedContactVerification
type RequestTrustedContactVerificationRequest struct {
	SessionId authsome.xid.ID `json:"sessionId"`
	ContactId authsome.xid.ID `json:"contactId"`
}

// RequestTrustedContactVerificationResponse is the response for RequestTrustedContactVerification
type RequestTrustedContactVerificationResponse struct {
	Error string `json:"error"`
	Message string `json:"message"`
	Code string `json:"code"`
	Details authsome. `json:"details"`
}

// RequestTrustedContactVerification RequestTrustedContactVerification handles POST /trusted-contacts/request-verification
func (p *Plugin) RequestTrustedContactVerification(ctx context.Context, req *RequestTrustedContactVerificationRequest) (*RequestTrustedContactVerificationResponse, error) {
	path := "/trusted-contacts/request-verification"
	var result RequestTrustedContactVerificationResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// RemoveTrustedContactResponse is the response for RemoveTrustedContact
type RemoveTrustedContactResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

// RemoveTrustedContact RemoveTrustedContact handles DELETE /trusted-contacts/:id
func (p *Plugin) RemoveTrustedContact(ctx context.Context) (*RemoveTrustedContactResponse, error) {
	path := "/trusted-contacts/:id"
	var result RemoveTrustedContactResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// SendVerificationCodeRequest is the request for SendVerificationCode
type SendVerificationCodeRequest struct {
	Method authsome.RecoveryMethod `json:"method"`
	SessionId authsome.xid.ID `json:"sessionId"`
	Target string `json:"target"`
}

// SendVerificationCodeResponse is the response for SendVerificationCode
type SendVerificationCodeResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// SendVerificationCode SendVerificationCode handles POST /verification/send
func (p *Plugin) SendVerificationCode(ctx context.Context, req *SendVerificationCodeRequest) (*SendVerificationCodeResponse, error) {
	path := "/verification/send"
	var result SendVerificationCodeResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// VerifyCodeRequest is the request for VerifyCode
type VerifyCodeRequest struct {
	Code string `json:"code"`
	SessionId authsome.xid.ID `json:"sessionId"`
}

// VerifyCodeResponse is the response for VerifyCode
type VerifyCodeResponse struct {
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
	Code string `json:"code"`
}

// VerifyCode VerifyCode handles POST /verification/verify
func (p *Plugin) VerifyCode(ctx context.Context, req *VerifyCodeRequest) (*VerifyCodeResponse, error) {
	path := "/verification/verify"
	var result VerifyCodeResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// ScheduleVideoSessionRequest is the request for ScheduleVideoSession
type ScheduleVideoSessionRequest struct {
	ScheduledAt authsome.time.Time `json:"scheduledAt"`
	SessionId authsome.xid.ID `json:"sessionId"`
	TimeZone string `json:"timeZone"`
}

// ScheduleVideoSessionResponse is the response for ScheduleVideoSession
type ScheduleVideoSessionResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// ScheduleVideoSession ScheduleVideoSession handles POST /video/schedule
func (p *Plugin) ScheduleVideoSession(ctx context.Context, req *ScheduleVideoSessionRequest) (*ScheduleVideoSessionResponse, error) {
	path := "/video/schedule"
	var result ScheduleVideoSessionResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// StartVideoSessionRequest is the request for StartVideoSession
type StartVideoSessionRequest struct {
	VideoSessionId authsome.xid.ID `json:"videoSessionId"`
}

// StartVideoSessionResponse is the response for StartVideoSession
type StartVideoSessionResponse struct {
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
	Code string `json:"code"`
}

// StartVideoSession StartVideoSession handles POST /video/start
func (p *Plugin) StartVideoSession(ctx context.Context, req *StartVideoSessionRequest) (*StartVideoSessionResponse, error) {
	path := "/video/start"
	var result StartVideoSessionResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// CompleteVideoSessionRequest is the request for CompleteVideoSession
type CompleteVideoSessionRequest struct {
	LivenessPassed bool `json:"livenessPassed"`
	LivenessScore float64 `json:"livenessScore"`
	Notes string `json:"notes"`
	VerificationResult string `json:"verificationResult"`
	VideoSessionId authsome.xid.ID `json:"videoSessionId"`
}

// CompleteVideoSessionResponse is the response for CompleteVideoSession
type CompleteVideoSessionResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// CompleteVideoSession CompleteVideoSession handles POST /video/complete (admin)
func (p *Plugin) CompleteVideoSession(ctx context.Context, req *CompleteVideoSessionRequest) (*CompleteVideoSessionResponse, error) {
	path := "/video/complete"
	var result CompleteVideoSessionResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// UploadDocumentRequest is the request for UploadDocument
type UploadDocumentRequest struct {
	BackImage string `json:"backImage"`
	DocumentType string `json:"documentType"`
	FrontImage string `json:"frontImage"`
	Selfie string `json:"selfie"`
	SessionId authsome.xid.ID `json:"sessionId"`
}

// UploadDocumentResponse is the response for UploadDocument
type UploadDocumentResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// UploadDocument UploadDocument handles POST /documents/upload
func (p *Plugin) UploadDocument(ctx context.Context, req *UploadDocumentRequest) (*UploadDocumentResponse, error) {
	path := "/documents/upload"
	var result UploadDocumentResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// GetDocumentVerificationResponse is the response for GetDocumentVerification
type GetDocumentVerificationResponse struct {
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
	Code string `json:"code"`
}

// GetDocumentVerification GetDocumentVerification handles GET /documents/:id
func (p *Plugin) GetDocumentVerification(ctx context.Context) (*GetDocumentVerificationResponse, error) {
	path := "/documents/:id"
	var result GetDocumentVerificationResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// ReviewDocumentRequest is the request for ReviewDocument
type ReviewDocumentRequest struct {
	Approved bool `json:"approved"`
	DocumentId authsome.xid.ID `json:"documentId"`
	Notes string `json:"notes"`
	RejectionReason string `json:"rejectionReason"`
}

// ReviewDocumentResponse is the response for ReviewDocument
type ReviewDocumentResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

// ReviewDocument ReviewDocument handles POST /documents/:id/review (admin)
func (p *Plugin) ReviewDocument(ctx context.Context, req *ReviewDocumentRequest) (*ReviewDocumentResponse, error) {
	path := "/documents/:id/review"
	var result ReviewDocumentResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// ListRecoverySessions ListRecoverySessions handles GET /admin/sessions (admin)
func (p *Plugin) ListRecoverySessions(ctx context.Context) error {
	path := "/sessions"
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return nil
}

// ApproveRecoveryRequest is the request for ApproveRecovery
type ApproveRecoveryRequest struct {
	Notes string `json:"notes"`
	SessionId authsome.xid.ID `json:"sessionId"`
}

// ApproveRecoveryResponse is the response for ApproveRecovery
type ApproveRecoveryResponse struct {
	Message string `json:"message"`
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
}

// ApproveRecovery ApproveRecovery handles POST /admin/sessions/:id/approve (admin)
func (p *Plugin) ApproveRecovery(ctx context.Context, req *ApproveRecoveryRequest) (*ApproveRecoveryResponse, error) {
	path := "/sessions/:id/approve"
	var result ApproveRecoveryResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// RejectRecoveryRequest is the request for RejectRecovery
type RejectRecoveryRequest struct {
	Notes string `json:"notes"`
	Reason string `json:"reason"`
	SessionId authsome.xid.ID `json:"sessionId"`
}

// RejectRecoveryResponse is the response for RejectRecovery
type RejectRecoveryResponse struct {
	Code string `json:"code"`
	Details authsome. `json:"details"`
	Error string `json:"error"`
	Message string `json:"message"`
}

// RejectRecovery RejectRecovery handles POST /admin/sessions/:id/reject (admin)
func (p *Plugin) RejectRecovery(ctx context.Context, req *RejectRecoveryRequest) (*RejectRecoveryResponse, error) {
	path := "/sessions/:id/reject"
	var result RejectRecoveryResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// GetRecoveryStats GetRecoveryStats handles GET /admin/stats (admin)
func (p *Plugin) GetRecoveryStats(ctx context.Context) error {
	path := "/stats"
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return nil
}

// GetRecoveryConfig GetRecoveryConfig handles GET /admin/config (admin)
func (p *Plugin) GetRecoveryConfig(ctx context.Context) error {
	path := "/config"
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return nil
}

// UpdateRecoveryConfigRequest is the request for UpdateRecoveryConfig
type UpdateRecoveryConfigRequest struct {
	EnabledMethods authsome.[]RecoveryMethod `json:"enabledMethods"`
	MinimumStepsRequired int `json:"minimumStepsRequired"`
	RequireAdminReview bool `json:"requireAdminReview"`
	RequireMultipleSteps bool `json:"requireMultipleSteps"`
	RiskScoreThreshold float64 `json:"riskScoreThreshold"`
}

// UpdateRecoveryConfigResponse is the response for UpdateRecoveryConfig
type UpdateRecoveryConfigResponse struct {
	Message string `json:"message"`
	Success bool `json:"success"`
}

// UpdateRecoveryConfig UpdateRecoveryConfig handles PUT /admin/config (admin)
func (p *Plugin) UpdateRecoveryConfig(ctx context.Context, req *UpdateRecoveryConfigRequest) (*UpdateRecoveryConfigResponse, error) {
	path := "/config"
	var result UpdateRecoveryConfigResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// HealthCheck HealthCheck handles GET /health
func (p *Plugin) HealthCheck(ctx context.Context) error {
	path := "/health"
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return nil
}

