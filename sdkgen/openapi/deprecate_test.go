package openapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/sdkgen/openapi"
)

// The tenant routes are tagged "Billing" and the operator ones "Billing
// Plans", "Billing Usage" and so on, so the longer "Billing " rule has to win
// over "Billing" whatever order the map gives them in.
func TestDeprecationFor_LongestPrefixWins(t *testing.T) {
	rules := map[string]string{
		"Billing":  "tenant",
		"Billing ": "operator",
	}
	assert.Equal(t, "tenant", openapi.DeprecationFor([]string{"Billing"}, rules))
	assert.Equal(t, "operator", openapi.DeprecationFor([]string{"Billing Plans"}, rules))
	assert.Equal(t, "operator", openapi.DeprecationFor([]string{"Auth", "Billing Usage"}, rules))
	assert.Empty(t, openapi.DeprecationFor([]string{"Auth"}, rules))
	assert.Empty(t, openapi.DeprecationFor([]string{"Billing"}, nil))
}

// Rules come from one flag, split on semicolons; a note can still contain
// one when it's escaped as \;.
func TestParseDeprecationRules(t *testing.T) {
	assert.Equal(t, map[string]string{
		"Billing":  "Use the plugin; it's newer.",
		"Billing ": "Operator route.",
	}, openapi.ParseDeprecationRules(`Billing=Use the plugin\; it's newer.;Billing =Operator route.`))
	assert.Empty(t, openapi.ParseDeprecationRules(""))
	assert.Empty(t, openapi.ParseDeprecationRules("no-equals; =no key"))
}
