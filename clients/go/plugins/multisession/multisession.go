package multisession

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated multisession plugin

// Plugin implements the multisession plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new multisession plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "multisession"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// List List returns sessions for the current user based on cookie
func (p *Plugin) List(ctx context.Context) (*authsome.ListResponse, error) {
	path := "/list"
	var result authsome.ListResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SetActive SetActive switches the current session cookie to the provided session id
func (p *Plugin) SetActive(ctx context.Context, req *authsome.SetActiveRequest) (*authsome.SetActiveResponse, error) {
	path := "/set-active"
	var result authsome.SetActiveResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete Delete revokes a session by id for the current user
func (p *Plugin) Delete(ctx context.Context) error {
	path := "/delete/{id}"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetCurrent GetCurrent returns details about the currently active session
func (p *Plugin) GetCurrent(ctx context.Context) (*authsome.GetCurrentResponse, error) {
	path := "/current"
	var result authsome.GetCurrentResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByID GetByID returns details about a specific session by ID
func (p *Plugin) GetByID(ctx context.Context) (*authsome.GetByIDResponse, error) {
	path := "/{id}"
	var result authsome.GetByIDResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeAll RevokeAll revokes all sessions for the current user
func (p *Plugin) RevokeAll(ctx context.Context, req *authsome.RevokeAllRequest) error {
	path := "/revoke-all"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// RevokeOthers RevokeOthers revokes all sessions except the current one
func (p *Plugin) RevokeOthers(ctx context.Context) error {
	path := "/revoke-others"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// Refresh Refresh extends the current session's expiry time
func (p *Plugin) Refresh(ctx context.Context) (*authsome.RefreshResponse, error) {
	path := "/refresh"
	var result authsome.RefreshResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetStats GetStats returns aggregated session statistics for the current user
func (p *Plugin) GetStats(ctx context.Context) error {
	path := "/stats"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

