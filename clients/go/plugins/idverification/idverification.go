package idverification

import (
	"context"
	"net/url"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated idverification plugin

// Plugin implements the idverification plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new idverification plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "idverification"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// CreateVerificationSession CreateVerificationSession creates a new verification session
POST /verification/sessions
func (p *Plugin) CreateVerificationSession(ctx context.Context, req *authsome.CreateVerificationSessionRequest) (*authsome.CreateVerificationSessionResponse, error) {
	path := "/verification/sessions"
	var result authsome.CreateVerificationSessionResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetVerificationSession GetVerificationSession retrieves a verification session
GET /verification/sessions/:id
func (p *Plugin) GetVerificationSession(ctx context.Context, id xid.ID) (*authsome.GetVerificationSessionResponse, error) {
	path := "/verification/sessions/:id"
	var result authsome.GetVerificationSessionResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetVerification GetVerification retrieves a verification by ID
GET /verification/:id
func (p *Plugin) GetVerification(ctx context.Context, id xid.ID) (*authsome.GetVerificationResponse, error) {
	path := "/verification/:id"
	var result authsome.GetVerificationResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetUserVerifications GetUserVerifications retrieves all verifications for the current user
GET /verification/me
func (p *Plugin) GetUserVerifications(ctx context.Context) (*authsome.GetUserVerificationsResponse, error) {
	path := "/verification/me"
	var result authsome.GetUserVerificationsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetUserVerificationStatus GetUserVerificationStatus retrieves the verification status for the current user
GET /verification/me/status
func (p *Plugin) GetUserVerificationStatus(ctx context.Context) (*authsome.GetUserVerificationStatusResponse, error) {
	path := "/verification/me/status"
	var result authsome.GetUserVerificationStatusResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RequestReverification RequestReverification requests re-verification for the current user
POST /verification/me/reverify
func (p *Plugin) RequestReverification(ctx context.Context, req *authsome.RequestReverificationRequest) (*authsome.RequestReverificationResponse, error) {
	path := "/verification/me/reverify"
	var result authsome.RequestReverificationResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// HandleWebhook HandleWebhook handles provider webhook callbacks
POST /verification/webhook/:provider
func (p *Plugin) HandleWebhook(ctx context.Context, provider string) (*authsome.HandleWebhookResponse, error) {
	path := "/verification/webhook/:provider"
	var result authsome.HandleWebhookResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AdminBlockUser AdminBlockUser blocks a user from verification (admin only)
POST /verification/admin/users/:userId/block
func (p *Plugin) AdminBlockUser(ctx context.Context, req *authsome.AdminBlockUserRequest, userId xid.ID) (*authsome.AdminBlockUserResponse, error) {
	path := "/verification/users/:userId/block"
	var result authsome.AdminBlockUserResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AdminUnblockUser AdminUnblockUser unblocks a user (admin only)
POST /verification/admin/users/:userId/unblock
func (p *Plugin) AdminUnblockUser(ctx context.Context, userId xid.ID) (*authsome.AdminUnblockUserResponse, error) {
	path := "/verification/users/:userId/unblock"
	var result authsome.AdminUnblockUserResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AdminGetUserVerificationStatus AdminGetUserVerificationStatus retrieves verification status for any user (admin only)
GET /verification/admin/users/:userId/status
func (p *Plugin) AdminGetUserVerificationStatus(ctx context.Context, userId xid.ID) (*authsome.AdminGetUserVerificationStatusResponse, error) {
	path := "/verification/users/:userId/status"
	var result authsome.AdminGetUserVerificationStatusResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AdminGetUserVerifications AdminGetUserVerifications retrieves all verifications for any user (admin only)
GET /verification/admin/users/:userId/verifications
func (p *Plugin) AdminGetUserVerifications(ctx context.Context, userId xid.ID) (*authsome.AdminGetUserVerificationsResponse, error) {
	path := "/verification/users/:userId/verifications"
	var result authsome.AdminGetUserVerificationsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

