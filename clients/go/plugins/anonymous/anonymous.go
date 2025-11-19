package anonymous

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated anonymous plugin

// Plugin implements the anonymous plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new anonymous plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "anonymous"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// SignIn SignIn creates a guest user and session
func (p *Plugin) SignIn(ctx context.Context) error {
	path := "/anonymous/signin"
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return nil
}

// LinkRequest is the request for Link
type LinkRequest struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Password string `json:"password"`
}

// Link Link upgrades an anonymous session to a real account
func (p *Plugin) Link(ctx context.Context, req *LinkRequest) error {
	path := "/anonymous/link"
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return nil
}

