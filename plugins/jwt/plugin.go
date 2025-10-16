package jwt

import (
	"github.com/xraph/authsome/core/jwt"
)

// Plugin implements the JWT authentication plugin
type Plugin struct {
	service *jwt.Service
	handler *Handler
}

// NewPlugin creates a new JWT plugin instance
func NewPlugin(service *jwt.Service) *Plugin {
	return &Plugin{
		service: service,
		handler: NewHandler(service),
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "jwt"
}

// GetHandler returns the JWT handler
func (p *Plugin) GetHandler() *Handler {
	return p.handler
}
