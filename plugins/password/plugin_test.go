package password_test

import (
	"context"
	log "github.com/xraph/go-utils/log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/password"
)

func TestPlugin_Name(t *testing.T) {
	p := password.New()
	assert.Equal(t, "password", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	p := password.New()

	// Plugin should implement base Plugin interface
	var _ plugin.Plugin = p

	// Plugin should implement BeforeSignUp
	var _ plugin.BeforeSignUp = p

	// Plugin should implement RouteProvider
	var _ plugin.RouteProvider = p
}

func TestPlugin_BeforeSignUp_NoRestrictions(t *testing.T) {
	p := password.New()
	ctx := context.Background()

	err := p.OnBeforeSignUp(ctx, &account.SignUpRequest{
		AppID:    id.NewAppID(),
		Email:    "anyone@anydomain.com",
		Password: "SecureP@ss1",
		FirstName: "Anyone",
	})

	assert.NoError(t, err)
}

func TestPlugin_BeforeSignUp_DomainRestriction_Allowed(t *testing.T) {
	p := password.New(password.Config{
		AllowedDomains: []string{"company.com", "example.com"},
	})
	ctx := context.Background()

	err := p.OnBeforeSignUp(ctx, &account.SignUpRequest{
		AppID:    id.NewAppID(),
		Email:    "user@company.com",
		Password: "SecureP@ss1",
	})

	assert.NoError(t, err)
}

func TestPlugin_BeforeSignUp_DomainRestriction_Blocked(t *testing.T) {
	p := password.New(password.Config{
		AllowedDomains: []string{"company.com"},
	})
	ctx := context.Background()

	err := p.OnBeforeSignUp(ctx, &account.SignUpRequest{
		AppID:    id.NewAppID(),
		Email:    "hacker@evil.com",
		Password: "SecureP@ss1",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestPlugin_RegisterRoutes(t *testing.T) {
	p := password.New()

	// RegisterRoutes should be a no-op
	err := p.RegisterRoutes(nil)
	assert.NoError(t, err)
}

func TestPlugin_Strategy(t *testing.T) {
	p := password.New()
	s := p.Strategy()

	assert.Equal(t, "password", s.Name())
}

func TestPlugin_RegisterInRegistry(t *testing.T) {
	reg := plugin.NewRegistry(log.NewNoopLogger())
	p := password.New()

	reg.Register(p)

	assert.Len(t, reg.Plugins(), 1)
	assert.Equal(t, "password", reg.Plugins()[0].Name())

	// Should be discoverable as a RouteProvider
	assert.Len(t, reg.RouteProviders(), 1)
}

func TestPlugin_FullIntegration_BeforeSignUp(t *testing.T) {
	reg := plugin.NewRegistry(log.NewNoopLogger())
	p := password.New(password.Config{
		AllowedDomains: []string{"allowed.com"},
	})
	reg.Register(p)

	ctx := context.Background()

	// Allowed domain should pass
	err := reg.EmitBeforeSignUp(ctx, &account.SignUpRequest{
		Email: "user@allowed.com",
	})
	require.NoError(t, err)

	// Blocked domain should fail
	err = reg.EmitBeforeSignUp(ctx, &account.SignUpRequest{
		Email: "user@blocked.com",
	})
	assert.Error(t, err)
}
