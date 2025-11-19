package emailotp

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated emailotp plugin

// Plugin implements the emailotp plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new emailotp plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "emailotp"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// SendRequest is the request for Send
type SendRequest struct {
	Email string `json:"email"`
}

// Send Send handles sending of OTP to email
func (p *Plugin) Send(ctx context.Context, req *SendRequest) error {
	path := "/email-otp/send"
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return nil
}

// VerifyRequest is the request for Verify
type VerifyRequest struct {
	Email string `json:"email"`
	Otp string `json:"otp"`
	Remember bool `json:"remember"`
}

// VerifyResponse is the response for Verify
type VerifyResponse struct {
	Session authsome. `json:"session"`
	Token string `json:"token"`
	User authsome. `json:"user"`
}

// Verify Verify checks the OTP and creates a session on success
func (p *Plugin) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResponse, error) {
	path := "/email-otp/verify"
	var result VerifyResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

