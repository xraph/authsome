package multisession

import (
	"context"
	"net/url"

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

// List List returns sessions for the current user with optional filtering
func (p *Plugin) List(ctx context.Context, req *authsome.ListRequest) error {
	path := "/multi-session/list"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

// SetActive SetActive switches the current session cookie to the provided session id
func (p *Plugin) SetActive(ctx context.Context, req *authsome.SetActiveRequest) (*authsome.SetActiveResponse, error) {
	path := "/multi-session/set-active"
	var result authsome.SetActiveResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete Delete revokes a session by id for the current user
func (p *Plugin) Delete(ctx context.Context, id xid.ID) error {
	path := "/multi-session/delete/" + url.PathEscape(id) + ""
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetCurrent GetCurrent returns details about the currently active session
func (p *Plugin) GetCurrent(ctx context.Context) (*authsome.GetCurrentResponse, error) {
	path := "/multi-session/current"
	var result authsome.GetCurrentResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByID GetByID returns details about a specific session by ID
func (p *Plugin) GetByID(ctx context.Context, id xid.ID) (*authsome.GetByIDResponse, error) {
	path := "/multi-session/" + url.PathEscape(id) + ""
	var result authsome.GetByIDResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeAll RevokeAll revokes all sessions for the current user
func (p *Plugin) RevokeAll(ctx context.Context, req *authsome.RevokeAllRequest) (*authsome.RevokeAllResponse, error) {
	path := "/multi-session/revoke-all"
	var result authsome.RevokeAllResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeOthers RevokeOthers revokes all sessions except the current one
func (p *Plugin) RevokeOthers(ctx context.Context) (*authsome.RevokeOthersResponse, error) {
	path := "/multi-session/revoke-others"
	var result authsome.RevokeOthersResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Refresh Refresh extends the current session's expiry time
func (p *Plugin) Refresh(ctx context.Context) (*authsome.RefreshResponse, error) {
	path := "/multi-session/refresh"
	var result authsome.RefreshResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetStats GetStats returns aggregated session statistics for the current user
func (p *Plugin) GetStats(ctx context.Context) (*authsome.GetStatsResponse, error) {
	path := "/multi-session/stats"
	var result authsome.GetStatsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

