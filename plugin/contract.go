// Package plugin: contract.go — optional interface a plugin implements
// when it ships a forge-dashboard contract surface (manifest + handlers).
//
// Plugins that adopt this interface participate in authsome's
// RegisterContractContributor lifecycle: the extension iterates every
// plugin after the auth contributor itself registers, invoking
// RegisterContract on each implementer in the order they were registered
// with the plugin registry. Order matters when a plugin extends another
// plugin's pages — register host-owning plugins first.
//
// Plugins remain free to ship only a templ-based DashboardContributor,
// only a contract one, or both during the migration. The interface is
// purely additive — non-implementers are skipped.
package plugin

import (
	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
)

// ContractContributor is implemented by plugins that ship a contract
// manifest + handler bundle. The signature mirrors authsome's own
// contract.Register, with the addition of the typed Engine handle so
// plugins don't have to capture it during OnInit just to wire a
// contract. The Engine parameter is a plugin.Engine interface; plugins
// that need the concrete *authsome.Engine for app-resolution helpers
// can type-assert internally.
type ContractContributor interface {
	// RegisterContract is called once during engine bootstrap, after the
	// auth contributor's own Register has succeeded. Implementations
	// should:
	//   1. Load and validate their embedded manifest YAML.
	//   2. Register the manifest with reg (which applies any `extends`
	//      blocks against previously-registered contributors).
	//   3. Register every intent handler with d under the manifest's
	//      contributor name.
	//
	// Returning a non-nil error aborts the plugin loop — keep error
	// messages prefixed with the plugin name so the failing plugin is
	// identifiable in startup logs.
	RegisterContract(d *dispatcher.Dispatcher, reg contract.Registry, wreg contract.WardenRegistry, engine Engine) error
}
