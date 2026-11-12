package organization

import (
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugin"
)

// SetPermCheckerForTest replaces the plugin's optional permission checker.
// Test-only seam so authz tests can inject a stub without booting warden.
func (p *Plugin) SetPermCheckerForTest(pc plugin.PermissionChecker) {
	p.permChecker = pc
}

// RoleRankForTest exposes the rank a role gets under the plugin's config.
func (p *Plugin) RoleRankForTest(r organization.MemberRole) int { return p.roleRank(r) }

// MaskEmail and InviterName expose the invitation preview's two formatting
// helpers so they can be tested directly. Both decide what a page shows to
// someone who may not own the address, so their edges are worth pinning.
var (
	MaskEmail   = maskEmail
	InviterName = inviterName
)
