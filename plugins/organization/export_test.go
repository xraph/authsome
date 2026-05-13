package organization

import (
	"github.com/xraph/authsome/plugin"
)

// SetPermCheckerForTest replaces the plugin's optional permission checker.
// Test-only seam so authz tests can inject a stub without booting warden.
func (p *Plugin) SetPermCheckerForTest(pc plugin.PermissionChecker) {
	p.permChecker = pc
}
