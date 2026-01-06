package pages

import (
	"github.com/xraph/authsome/plugins/dashboard/v2/services"
)

func (p *PagesManager) SetServices(services *services.Services) {
	p.services = services
}
