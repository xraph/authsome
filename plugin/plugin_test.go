package plugin_test

import (
	"testing"

	"github.com/xraph/authsome/plugin"
)

// The three principal methods are on the core interface rather than an
// optional capability interface. Plugins are meant to reason about non-human
// callers, and a type assertion makes that undiscoverable.
func TestEngineInterfaceExposesPrincipalMethods(t *testing.T) {
	var e plugin.Engine
	if e != nil {
		_ = e.ResolvePrincipal
		_ = e.PrincipalStore
		_ = e.Can
	}
}
