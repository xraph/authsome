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

// ListResponse is the response for List
type ListResponse struct {
	Sessions authsome. `json:"sessions"`
}

// List List returns sessions for the current user based on cookie
func (p *Plugin) List(ctx context.Context) (*ListResponse, error) {
	path := "/list"
	var result ListResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// SetActiveRequest is the request for SetActive
type SetActiveRequest struct {
	Id string `json:"id"`
}

// SetActiveResponse is the response for SetActive
type SetActiveResponse struct {
	Token string `json:"token"`
	Session authsome. `json:"session"`
}

// SetActive SetActive switches the current session cookie to the provided session id
func (p *Plugin) SetActive(ctx context.Context, req *SetActiveRequest) (*SetActiveResponse, error) {
	path := "/set-active"
	var result SetActiveResponse
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return &result, nil
}

// Delete Delete revokes a session by id for the current user
func (p *Plugin) Delete(ctx context.Context) error {
	path := "/delete/{id}"
	// Note: This requires exposing client.request or using a different approach
	// For now, this is a placeholder
	_ = path
	return nil
}

