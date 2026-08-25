package plugin_test

import (
	"testing"

	"github.com/xraph/authsome/plugin"
)

// The three principal methods are on the core interface rather than an
// optional capability interface. Plugins are meant to reason about non-human
// callers, and a type assertion makes that undiscoverable.
func TestEngineInterfaceExposesPrincipalMethods(_ *testing.T) {
	// Method expressions on the interface type: this is a compile-time check
	// with nothing to run. The previous spelling guarded a nil interface with
	// `if e != nil`, which is never true, so the methods were never named at
	// all and the build could have lost them silently.
	_ = plugin.Engine.ResolvePrincipal
	_ = plugin.Engine.PrincipalStore
	_ = plugin.Engine.Can
}
