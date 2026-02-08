package pages

import (
	"github.com/xraph/authsome/plugins/dashboard/services"
)

func (p *PagesManager) SetServices(services *services.Services) {
	p.services = services
}

func (p *PagesManager) SetBaseUIPath(baseUIPath string) {
	p.baseUIPath = baseUIPath
}

// SetExtensionRegistry sets the extension registry for accessing plugin extensions
func (p *PagesManager) SetExtensionRegistry(registry ExtensionRegistry) {
	p.extensionRegistry = registry
}
