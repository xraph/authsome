package pages

import (
	"github.com/xraph/authsome/plugins/oidcprovider/bridge"
	"github.com/xraph/forge"
)

// PagesManager manages all dashboard pages for the OIDC provider plugin
type PagesManager struct {
	bridgeManager *bridge.BridgeManager
	logger        forge.Logger
	baseUIPath    string
}

// NewPagesManager creates a new pages manager
func NewPagesManager(bridgeManager *bridge.BridgeManager, logger forge.Logger) *PagesManager {
	return &PagesManager{
		bridgeManager: bridgeManager,
		logger:        logger,
		baseUIPath:    "/ui", // Default base path
	}
}

// SetBaseUIPath sets the base UI path (called by dashboard plugin)
func (p *PagesManager) SetBaseUIPath(path string) {
	p.baseUIPath = path
}
