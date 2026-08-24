package plugin_test

import (
	"testing"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
)

// The two capability interfaces exist so plugins can reach engine methods
// without importing the concrete type. If the engine ever loses either
// method, this fails at compile time rather than at runtime in a receiver.
func TestEngineSatisfiesCapabilities(t *testing.T) {
	var _ plugin.SessionRevoker = (*authsome.Engine)(nil)
	var _ plugin.DispatcherProvider = (*authsome.Engine)(nil)
}
