package agentauth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/plugins/agentauth"
)

func testRegistry() *agentauth.ScopeRegistry {
	r := agentauth.NewScopeRegistry()
	r.Register("invoices:read", agentauth.Grants("read", "invoice"))
	r.Register("invoices:write", agentauth.Grants("write", "invoice"))
	return r
}

func TestScopeRegistry_Covers(t *testing.T) {
	r := testRegistry()

	tests := []struct {
		name     string
		scopes   []string
		action   string
		resource string
		want     bool
	}{
		{"granted scope covers its permission", []string{"invoices:read"}, "read", "invoice", true},
		{"read does not cover write", []string{"invoices:read"}, "write", "invoice", false},
		{"scope does not cover another resource", []string{"invoices:read"}, "read", "payment", false},
		{"empty grant covers nothing", nil, "read", "invoice", false},
		{"unregistered scope covers nothing", []string{"invoices:delete"}, "delete", "invoice", false},
		{"any one matching scope is enough", []string{"invoices:read", "invoices:write"}, "write", "invoice", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, r.Covers(tt.scopes, tt.action, tt.resource))
		})
	}
}

// A scope with no mapping must be rejected at consent time, so the registry
// has to be able to answer this question before a grant is written.
func TestScopeRegistry_Known(t *testing.T) {
	r := testRegistry()

	assert.True(t, r.Known("invoices:read"))
	assert.False(t, r.Known("invoices:delete"))
}
