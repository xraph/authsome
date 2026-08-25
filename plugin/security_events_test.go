package plugin_test

import (
	"testing"

	"github.com/xraph/authsome/plugin"
)

// Plugins write security events directly rather than through the hook bus.
// The bus bridge at engine.go:526 never sets AppID, and securityevent.Query
// filters on it, so events recorded that way are written but unqueryable.
func TestEngineInterfaceExposesSecurityEvents(_ *testing.T) {
	// See TestEngineInterfaceExposesPrincipalMethods: a method expression
	// checks this at compile time without a receiver to dereference.
	_ = plugin.Engine.SecurityEvents
}
